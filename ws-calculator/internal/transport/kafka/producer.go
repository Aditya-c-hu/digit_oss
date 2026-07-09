package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	brokers []string
	mu      sync.Mutex
	writers map[string]*kafka.Writer
	logger  *slog.Logger
}

func NewProducer(brokers []string, l *slog.Logger) *Producer {
	return &Producer{brokers: brokers, writers: map[string]*kafka.Writer{}, logger: l}
}

func (p *Producer) writer(topic string) *kafka.Writer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		BatchTimeout: 50 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}
	p.writers[topic] = w
	return w
}

func (p *Producer) Push(ctx context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.writer(topic).WriteMessages(ctx, kafka.Message{Value: body})
}

func (p *Producer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, w := range p.writers {
		_ = w.Close()
	}
}
