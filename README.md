# StreamPulse

Real-time event analytics and monitoring platform. Ingest events over HTTP,
stream them through Kafka, aggregate live stats in Redis, persist history in
ClickHouse, and push updates to a Next.js dashboard over WebSockets.

## Architecture

```
Next.js Dashboard  <──WebSocket──  Processor (Go)  ──consumes──  Kafka
                                        │  │
                                        │  └─writes─→ ClickHouse (history)
                                        └─writes/reads─→ Redis (live state)

Client ──POST /events──→ API (Go) ──produces──→ Kafka
```

- **API (Go, `/api`)** — validates and publishes events to the `events` Kafka
  topic, keyed by `user_id` so a user's events land on the same partition.
- **Kafka** — durable, ordered event backbone. Single-node KRaft broker for
  local dev; six partitions on `events` so the processor can scale
  horizontally via consumer groups.
- **Processor (Go, `/processor`)** — consumes `events`, updates live Redis
  counters per message, batches events into ClickHouse (every 500 events or
  1s, whichever comes first), and broadcasts a dashboard snapshot to every
  connected WebSocket client once per second.
- **Redis** — live/temporary state only: per-second event counters, a
  5-minute rolling active-user set, running totals, error counts. Nothing
  here is meant to survive a flush.
- **ClickHouse** — historical analytics. Raw events land in `events`
  (`MergeTree`, partitioned by day, 90-day TTL); a materialized view
  (`events_per_minute_mv`) pre-aggregates per-minute rollups so "events over
  the last hour" charts don't scan raw rows.
- **Frontend (Next.js, `/frontend`)** — dashboard with a live "pulse" trace,
  headline stats, an event-type breakdown, and a scrolling event feed, all
  driven by a single WebSocket subscription.
- **OpenTelemetry Collector** — receives OTLP traces/metrics from the API and
  processor; currently exports to logs, ready to point at a real backend
  (Jaeger, Tempo, Prometheus) later.

## Event flow

1. Client sends `POST /events` to the API.
2. API validates the payload and publishes it to Kafka.
3. Processor's consumer group reads the event, updates Redis counters, and
   buffers it for ClickHouse.
4. Every second, the processor reads a fresh Redis snapshot and broadcasts it
   as JSON to all connected dashboards over the WebSocket hub.
5. Every 500 buffered events (or 1s, whichever is first), the processor
   flushes a batch insert to ClickHouse for historical querying.

## Kafka topic design

Single `events` topic, 6 partitions, replication factor 1 (local dev — bump
replication in a real cluster). Partitioning by `user_id` keeps a given
user's events in order without needing a topic per event type.

## Redis strategy

Keys are short-lived by design:
- `stream:events:second:<unix_ts>` — per-second counter, 10s TTL
- `stream:active_users` — sorted set scored by last-seen unix time, trimmed
  to a 5-minute window on every write
- `stream:event_counts:<type>` / `:total` — running counters
- `stream:errors:total` — running error counter

## WebSocket strategy

One hub per processor instance holds all client connections in memory and
fans out a single JSON tick per second. Clients reconnect with exponential
backoff (1s → 15s cap) on drop, so a processor restart is transparent to an
open dashboard tab.

## Ports (local dev)

Chosen to avoid collisions with other projects already running locally:

| Service              | Host port |
|-----------------------|-----------|
| Frontend               | 3003 |
| API (HTTP)             | 4002 |
| Processor (WebSocket)  | 4003 |
| Kafka (external)       | 9093 |
| Redis                  | 6400 |
| ClickHouse (HTTP)      | 8124 |
| ClickHouse (native)    | 9004 |
| OTel Collector (gRPC)  | 4317 |
| OTel Collector (HTTP)  | 4320 |

## Running it

```bash
docker compose up --build
```

Then send a test event:

```bash
curl -X POST http://localhost:4002/events \
  -H "Content-Type: application/json" \
  -d '{"event_type":"order.created","user_id":"user_123","source":"web","timestamp":"2026-08-27T12:00:00Z","metadata":{"order_id":"order_789","amount":45000}}'
```

Open `http://localhost:3003` to watch the dashboard update live.

## Testing

- **Unit tests**: `go test ./...` in both `api/` and `processor/` — event
  validation, error-rate math, JSON decoding.
- **Integration tests**: (next step) spin up the full compose stack and
  assert an event posted to the API shows up in a ClickHouse query.
- **Load testing**: `k6 run loadtest/k6-script.js` drives a constant
  1,000 events/sec against the API for 60s with p95 latency and error-rate
  thresholds.

## Scalability considerations

- The API is stateless — scale it horizontally behind a load balancer.
- The processor scales via Kafka consumer groups: more partitions + more
  processor replicas = more parallel consumption, as long as Redis/ClickHouse
  can keep up.
- ClickHouse writes are batched specifically because row-at-a-time inserts
  are its worst case; the 500-event/1s flush is a starting point to tune
  against real throughput.
- The WebSocket hub is per-process and in-memory, so multiple processor
  replicas each hold their own set of dashboard clients — a real multi-node
  deployment would need a pub/sub layer (e.g. Redis Pub/Sub) so a tick
  computed on one node reaches clients connected to another.

## Known gaps / next steps

- Multi-tenant auth, API keys, and per-tenant rate limits (stretch goal) are
  not yet implemented.
- The live feed currently shows the dominant event type per tick rather than
  every individual event — wiring the processor to also stream raw
  per-event lines (rate-limited) is the natural next step.
- Structured logging and OpenTelemetry spans are wired for the collector but
  not yet instrumented through every function — tracing context needs to be
  threaded from the API's `POST /events` handler through to the ClickHouse
  insert.
