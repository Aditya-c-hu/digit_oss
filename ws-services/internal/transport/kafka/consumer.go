package kafka

import (
	"context"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"
)

// Handler processes a single message body. Errors are logged and the offset
// is still committed; the Java service does the same on auto-commit groups.
type Handler func(ctx context.Context, value []byte) error

// ConsumerGroup runs one goroutine per topic subscription, mirroring the
// per-listener thread model of @KafkaListener in spring-kafka.
type ConsumerGroup struct {
	brokers []string
	groupID string
	logger  *slog.Logger
	wg      sync.WaitGroup
	cancels []context.CancelFunc
	readers []*kafka.Reader
}

func NewConsumerGroup(brokers []string, groupID string, logger *slog.Logger) *ConsumerGroup {
	return &ConsumerGroup{brokers: brokers, groupID: groupID, logger: logger}
}

// Subscribe wires up a topic to a handler and runs it on its own goroutine.
// Each call adds another concurrent worker; this is the Go equivalent of the
// Java multiple consumer threads (concurrency = N).
func (c *ConsumerGroup) Subscribe(topic string, h Handler) {
	if topic == "" {
		c.logger.Warn("kafka subscribe skipped: empty topic", "group", c.groupID)
		return
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		GroupID:  c.groupID,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10 * 1024 * 1024,
	})
	ctx, cancel := context.WithCancel(context.Background())
	c.readers = append(c.readers, r)
	c.cancels = append(c.cancels, cancel)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.logger.Info("kafka consumer started", "topic", topic, "group", c.groupID)
		for {
			m, err := r.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("kafka read error", "topic", topic, "err", err)
				continue
			}
			if err := h(ctx, m.Value); err != nil {
				c.logger.Error("kafka handler error", "topic", topic, "err", err)
			}
		}
	}()
}

func (c *ConsumerGroup) Stop() {
	for _, cancel := range c.cancels {
		cancel()
	}
	for _, r := range c.readers {
		_ = r.Close()
	}
	c.wg.Wait()
}
