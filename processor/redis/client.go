package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr string) *Client {
	return &Client{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

// IncrEventCounter bumps the per-second and per-type counters. Keys are
// bucketed by unix second so "events per second" is a cheap read of the
// current bucket, and old buckets expire on their own via TTL.
func (c *Client) IncrEventCounter(ctx context.Context, eventType string) error {
	second := time.Now().Unix()
	pipe := c.rdb.Pipeline()

	secondKey := "stream:events:second:" + strconv.FormatInt(second, 10)
	pipe.Incr(ctx, secondKey)
	pipe.Expire(ctx, secondKey, 10*time.Second)

	pipe.Incr(ctx, "stream:event_counts:"+eventType)
	pipe.Incr(ctx, "stream:event_counts:total")

	_, err := pipe.Exec(ctx)
	return err
}

// TrackActiveUser marks a user active in a rolling 5-minute window using a
// sorted set scored by last-seen timestamp, so stale users age out.
func (c *Client) TrackActiveUser(ctx context.Context, userID string) error {
	now := float64(time.Now().Unix())
	if err := c.rdb.ZAdd(ctx, "stream:active_users", redis.Z{Score: now, Member: userID}).Err(); err != nil {
		return err
	}
	cutoff := now - 5*60
	return c.rdb.ZRemRangeByScore(ctx, "stream:active_users", "-inf", strconv.FormatFloat(cutoff, 'f', 0, 64)).Err()
}

func (c *Client) IncrErrors(ctx context.Context) error {
	return c.rdb.Incr(ctx, "stream:errors:total").Err()
}

// Snapshot returns the current values needed for a dashboard broadcast tick.
type Snapshot struct {
	EventsPerSecond int64
	ActiveUsers     int64
	TotalEvents     int64
	TotalErrors     int64
	EventCounts     map[string]int64
}

func (c *Client) Snapshot(ctx context.Context, eventTypes []string) (Snapshot, error) {
	second := time.Now().Unix() - 1 // last full second
	eventsPerSec, _ := c.rdb.Get(ctx, "stream:events:second:"+strconv.FormatInt(second, 10)).Int64()
	activeUsers, _ := c.rdb.ZCard(ctx, "stream:active_users").Result()
	totalEvents, _ := c.rdb.Get(ctx, "stream:event_counts:total").Int64()
	totalErrors, _ := c.rdb.Get(ctx, "stream:errors:total").Int64()

	counts := make(map[string]int64, len(eventTypes))
	for _, et := range eventTypes {
		v, _ := c.rdb.Get(ctx, "stream:event_counts:"+et).Int64()
		counts[et] = v
	}

	return Snapshot{
		EventsPerSecond: eventsPerSec,
		ActiveUsers:     activeUsers,
		TotalEvents:     totalEvents,
		TotalErrors:     totalErrors,
		EventCounts:     counts,
	}, nil
}

