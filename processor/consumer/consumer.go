package consumer

import (
	"context"
	"log"
	"strings"

	kafkago "github.com/segmentio/kafka-go"
)

// HandlerFunc processes a single raw Kafka message payload.
type HandlerFunc func(ctx context.Context, payload []byte) error

type Consumer struct {
	reader  *kafkago.Reader
	handler HandlerFunc
}

func New(brokers, topic, groupID string, handler HandlerFunc) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     strings.Split(brokers, ","),
		Topic:       topic,
		GroupID:     groupID, // consumer group -> horizontal scaling of processors
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafkago.LastOffset,
	})
	return &Consumer{reader: reader, handler: handler}
}

// Run reads messages until ctx is cancelled. Failed messages are logged and
// skipped rather than blocking the whole partition (dead-letter handling
// would be the next step for production hardening).
func (c *Consumer) Run(ctx context.Context) {
	defer c.reader.Close()
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // context cancelled, normal shutdown
			}
			log.Printf("kafka fetch error: %v", err)
			continue
		}

		if err := c.handler(ctx, msg.Value); err != nil {
			log.Printf("event handling error (offset %d): %v", msg.Offset, err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("kafka commit error: %v", err)
		}
	}
}
