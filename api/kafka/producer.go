package kafka

import (
	"context"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer publishes events onto the Kafka events topic, keyed by user_id
// so all events for a given user land on the same partition (ordering).
type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers, topic string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(strings.Split(brokers, ",")...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{}, // key-based partitioning
			BatchTimeout: 50 * time.Millisecond,
			RequiredAcks: kafkago.RequireOne,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	return p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
