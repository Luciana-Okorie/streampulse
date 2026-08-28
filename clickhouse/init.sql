CREATE DATABASE IF NOT EXISTS streampulse;

CREATE TABLE IF NOT EXISTS streampulse.events
(
    event_id       UUID DEFAULT generateUUIDv4(),
    event_type     LowCardinality(String),
    user_id        String,
    source         LowCardinality(String),
    event_time     DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3),
    metadata       String, -- raw JSON, parsed on read with JSONExtract*
    is_error       UInt8 MATERIALIZED (event_type LIKE '%.error' OR event_type LIKE '%.failed')
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(event_time)
ORDER BY (event_type, event_time)
TTL toDateTime(event_time) + INTERVAL 90 DAY;

-- Pre-aggregated per-minute rollups for fast "last N minutes/hours" charts,
-- so the dashboard never has to scan raw events for the common queries.
CREATE MATERIALIZED VIEW IF NOT EXISTS streampulse.events_per_minute_mv
ENGINE = SummingMergeTree
PARTITION BY toYYYYMMDD(minute)
ORDER BY (event_type, minute)
AS
SELECT
    toStartOfMinute(event_time) AS minute,
    event_type,
    count()                      AS event_count,
    sum(is_error)                AS error_count
FROM streampulse.events
GROUP BY minute, event_type;
