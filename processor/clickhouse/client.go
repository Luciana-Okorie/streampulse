package clickhouse

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Client struct {
	conn driver.Conn
}

type EventRecord struct {
	EventType string
	UserID    string
	Source    string
	EventTime time.Time
	Metadata  map[string]interface{}
}

func NewClient(addr, database string) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: database},
	})
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// InsertEvent writes a single event. Callers should batch these (see
// InsertBatch) once throughput exceeds a few hundred events/sec — ClickHouse
// is optimized for large infrequent inserts, not one-row-at-a-time writes.
func (c *Client) InsertEvent(ctx context.Context, e EventRecord) error {
	metadataJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		return err
	}
	return c.conn.Exec(ctx,
		`INSERT INTO events (event_type, user_id, source, event_time, metadata) VALUES (?, ?, ?, ?, ?)`,
		e.EventType, e.UserID, e.Source, e.EventTime, string(metadataJSON),
	)
}

// InsertBatch is the production path: the aggregator buffers events and
// flushes them here every N milliseconds or M rows, whichever comes first.
func (c *Client) InsertBatch(ctx context.Context, events []EventRecord) error {
	batch, err := c.conn.PrepareBatch(ctx, "INSERT INTO events (event_type, user_id, source, event_time, metadata)")
	if err != nil {
		return err
	}
	for _, e := range events {
		metadataJSON, err := json.Marshal(e.Metadata)
		if err != nil {
			return err
		}
		if err := batch.Append(e.EventType, e.UserID, e.Source, e.EventTime, string(metadataJSON)); err != nil {
			return err
		}
	}
	return batch.Send()
}

type EventTypeCount struct {
	EventType string `json:"event_type"`
	Count     uint64 `json:"count"`
}

// TopEventTypes powers the "Event Types" bar chart.
func (c *Client) TopEventTypes(ctx context.Context, since time.Time, limit int) ([]EventTypeCount, error) {
	rows, err := c.conn.Query(ctx,
		`SELECT event_type, count() AS cnt FROM events WHERE event_time >= ? GROUP BY event_type ORDER BY cnt DESC LIMIT ?`,
		since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []EventTypeCount
	for rows.Next() {
		var r EventTypeCount
		if err := rows.Scan(&r.EventType, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

type TimeBucket struct {
	Minute time.Time `json:"minute"`
	Count  uint64    `json:"count"`
	Errors uint64    `json:"errors"`
}

// EventsOverTime powers the "Events over time" sparkline, reading from the
// pre-aggregated per-minute materialized view instead of scanning raw rows.
func (c *Client) EventsOverTime(ctx context.Context, since time.Time) ([]TimeBucket, error) {
	rows, err := c.conn.Query(ctx,
		`SELECT minute, sum(event_count) AS cnt, sum(error_count) AS errs
		 FROM events_per_minute_mv WHERE minute >= ? GROUP BY minute ORDER BY minute`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []TimeBucket
	for rows.Next() {
		var b TimeBucket
		if err := rows.Scan(&b.Minute, &b.Count, &b.Errors); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func (c *Client) Close() error {
	return c.conn.Close()
}
