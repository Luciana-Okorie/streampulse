package aggregator

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	ch "streampulse/processor/clickhouse"
	rds "streampulse/processor/redis"
	"streampulse/processor/websocket"
)

// knownEventTypes drives the per-type counters shown on the dashboard.
// In a fuller implementation this would be discovered dynamically instead
// of hardcoded.
var knownEventTypes = []string{
	"user.login", "user.logout", "order.created",
	"payment.success", "payment.failed", "api.request", "api.error",
}

type rawEvent struct {
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id"`
	Source    string                 `json:"source"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type Aggregator struct {
	redis      *rds.Client
	clickhouse *ch.Client
	hub        *websocket.Hub

	// batching buffer for ClickHouse inserts
	buffer []ch.EventRecord
}

func New(redis *rds.Client, clickhouse *ch.Client, hub *websocket.Hub) *Aggregator {
	return &Aggregator{redis: redis, clickhouse: clickhouse, hub: hub}
}

// HandleEvent is called by the Kafka consumer for every message. It updates
// live Redis counters immediately (cheap) and buffers the event for a
// batched ClickHouse insert (expensive, so not done per-message).
func (a *Aggregator) HandleEvent(ctx context.Context, payload []byte) error {
	var evt rawEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return err
	}

	if err := a.redis.IncrEventCounter(ctx, evt.EventType); err != nil {
		log.Printf("redis counter error: %v", err)
	}
	if err := a.redis.TrackActiveUser(ctx, evt.UserID); err != nil {
		log.Printf("redis active-user error: %v", err)
	}
	if strings.HasSuffix(evt.EventType, ".error") || strings.HasSuffix(evt.EventType, ".failed") {
		if err := a.redis.IncrErrors(ctx); err != nil {
			log.Printf("redis error-counter error: %v", err)
		}
	}

	eventTime, err := time.Parse(time.RFC3339, evt.Timestamp)
	if err != nil {
		eventTime = time.Now()
	}

	a.buffer = append(a.buffer, ch.EventRecord{
		EventType: evt.EventType,
		UserID:    evt.UserID,
		Source:    evt.Source,
		EventTime: eventTime,
		Metadata:  evt.Metadata,
	})

	// Flush every 500 events so ClickHouse gets efficient batch inserts
	// instead of a row per message.
	if len(a.buffer) >= 500 {
		a.flush(ctx)
	}

	return nil
}

func (a *Aggregator) flush(ctx context.Context) {
	if len(a.buffer) == 0 {
		return
	}
	if err := a.clickhouse.InsertBatch(ctx, a.buffer); err != nil {
		log.Printf("clickhouse batch insert error: %v", err)
	}
	a.buffer = a.buffer[:0]
}

// DashboardTick is the JSON payload pushed to every connected dashboard.
type DashboardTick struct {
	EventsPerSecond int64            `json:"events_per_second"`
	ActiveUsers     int64            `json:"active_users"`
	TotalEvents     int64            `json:"total_events"`
	TotalErrors     int64            `json:"total_errors"`
	ErrorRate       float64          `json:"error_rate"`
	EventCounts     map[string]int64 `json:"event_counts"`
	Timestamp       string           `json:"timestamp"`
}

// RunBroadcastLoop periodically reads the current Redis snapshot and pushes
// it to every connected dashboard over the websocket hub, and time-flushes
// the ClickHouse buffer even under light load.
func (a *Aggregator) RunBroadcastLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.flush(ctx)
			return
		case <-ticker.C:
			a.flush(ctx) // time-based flush in addition to the size-based one

			snap, err := a.redis.Snapshot(ctx, knownEventTypes)
			if err != nil {
				log.Printf("snapshot error: %v", err)
				continue
			}

			var errorRate float64
			if snap.TotalEvents > 0 {
				errorRate = float64(snap.TotalErrors) / float64(snap.TotalEvents) * 100
			}

			tick := DashboardTick{
				EventsPerSecond: snap.EventsPerSecond,
				ActiveUsers:     snap.ActiveUsers,
				TotalEvents:     snap.TotalEvents,
				TotalErrors:     snap.TotalErrors,
				ErrorRate:       errorRate,
				EventCounts:     snap.EventCounts,
				Timestamp:       time.Now().UTC().Format(time.RFC3339),
			}

			payload, err := json.Marshal(tick)
			if err != nil {
				log.Printf("tick marshal error: %v", err)
				continue
			}
			a.hub.Broadcast(payload)
		}
	}
}
