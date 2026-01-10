package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// ExchangeName is the name of the topic exchange for domain events
	ExchangeName = "orbita.domain.events"
)

// RabbitMQPublisher publishes events to RabbitMQ.
type RabbitMQPublisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	logger   *slog.Logger
	mu       sync.Mutex
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher.
func NewRabbitMQPublisher(url string, logger *slog.Logger) (*RabbitMQPublisher, error) {
	if logger == nil {
		logger = slog.Default()
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close() // Best-effort cleanup
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the topic exchange
	err = ch.ExchangeDeclare(
		ExchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		_ = ch.Close()   // Best-effort cleanup
		_ = conn.Close() // Best-effort cleanup
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	logger.Info("RabbitMQ publisher connected",
		"exchange", ExchangeName,
	)

	return &RabbitMQPublisher{
		conn:     conn,
		channel:  ch,
		exchange: ExchangeName,
		logger:   logger,
	}, nil
}

// Publish sends a message to the exchange with the given routing key.
func (p *RabbitMQPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.channel.PublishWithContext(ctx,
		p.exchange,  // exchange
		routingKey,  // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         payload,
		},
	)

	if err != nil {
		p.logger.Error("failed to publish message",
			"routing_key", routingKey,
			"error", err,
		)
		return err
	}

	p.logger.Debug("message published",
		"routing_key", routingKey,
		"size", len(payload),
	)

	return nil
}

// Close closes the publisher connection.
func (p *RabbitMQPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			p.logger.Warn("error closing channel", "error", err)
		}
	}

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			return err
		}
	}

	p.logger.Info("RabbitMQ publisher closed")
	return nil
}

// NoopPublisher is a no-op publisher for testing/development.
type NoopPublisher struct {
	logger *slog.Logger
}

// NewNoopPublisher creates a publisher that does nothing.
func NewNoopPublisher(logger *slog.Logger) *NoopPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &NoopPublisher{logger: logger}
}

// Publish logs the message but doesn't actually publish.
func (p *NoopPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	p.logger.Debug("noop publish",
		"routing_key", routingKey,
		"size", len(payload),
	)
	return nil
}

// Close is a no-op.
func (p *NoopPublisher) Close() error {
	return nil
}
