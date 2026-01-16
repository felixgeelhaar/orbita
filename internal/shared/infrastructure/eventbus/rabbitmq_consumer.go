package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// DefaultConsumerQueueName is the default queue name for consuming events.
	DefaultConsumerQueueName = "orbita.consumer"
)

// RabbitMQConsumer consumes events from RabbitMQ.
type RabbitMQConsumer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queue     string
	exchange  string
	registry  *ConsumerRegistry
	logger    *slog.Logger
	mu        sync.Mutex
	running   bool
	closeChan chan struct{}
}

// RabbitMQConsumerConfig configures the RabbitMQ consumer.
type RabbitMQConsumerConfig struct {
	URL       string
	QueueName string
	Exchange  string
	Logger    *slog.Logger
}

// NewRabbitMQConsumer creates a new RabbitMQ consumer.
func NewRabbitMQConsumer(cfg RabbitMQConsumerConfig, registry *ConsumerRegistry) (*RabbitMQConsumer, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.QueueName == "" {
		cfg.QueueName = DefaultConsumerQueueName
	}
	if cfg.Exchange == "" {
		cfg.Exchange = ExchangeName
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the exchange (should already exist from publisher)
	err = ch.ExchangeDeclare(
		cfg.Exchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare the queue
	_, err = ch.QueueDeclare(
		cfg.QueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	cfg.Logger.Info("RabbitMQ consumer connected",
		"queue", cfg.QueueName,
		"exchange", cfg.Exchange,
	)

	return &RabbitMQConsumer{
		conn:      conn,
		channel:   ch,
		queue:     cfg.QueueName,
		exchange:  cfg.Exchange,
		registry:  registry,
		logger:    cfg.Logger,
		closeChan: make(chan struct{}),
	}, nil
}

// RegisterConsumer registers an event consumer and binds its event types to the queue.
func (c *RabbitMQConsumer) RegisterConsumer(consumer EventConsumer) {
	c.registry.Register(consumer)

	// Bind the queue to the exchange for each event type
	for _, eventType := range consumer.EventTypes() {
		if err := c.bindQueue(eventType); err != nil {
			c.logger.Error("failed to bind queue for event type",
				"event_type", eventType,
				"error", err,
			)
		}
	}
}

func (c *RabbitMQConsumer) bindQueue(routingKey string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.channel.QueueBind(
		c.queue,
		routingKey,
		c.exchange,
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	c.logger.Debug("bound queue to routing key",
		"queue", c.queue,
		"routing_key", routingKey,
	)

	return nil
}

// Start begins consuming messages from the queue.
func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	c.mu.Unlock()

	// Set prefetch count to process one message at a time
	if err := c.channel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := c.channel.Consume(
		c.queue,
		"",    // consumer tag (auto-generated)
		false, // auto-ack (we'll manually ack)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.logger.Info("started consuming events",
		"queue", c.queue,
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer context cancelled, stopping")
			return ctx.Err()

		case <-c.closeChan:
			c.logger.Info("consumer close requested, stopping")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				c.logger.Warn("message channel closed")
				return fmt.Errorf("message channel closed unexpectedly")
			}

			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.Error("failed to process message",
					"routing_key", msg.RoutingKey,
					"error", err,
				)
				// Nack and requeue for retry
				if nackErr := msg.Nack(false, true); nackErr != nil {
					c.logger.Error("failed to nack message", "error", nackErr)
				}
			} else {
				// Ack successful processing
				if ackErr := msg.Ack(false); ackErr != nil {
					c.logger.Error("failed to ack message", "error", ackErr)
				}
			}
		}
	}
}

func (c *RabbitMQConsumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	event := &ConsumedEvent{}
	if err := json.Unmarshal(msg.Body, event); err != nil {
		// Can't unmarshal - this is a bad message, don't retry
		c.logger.Error("failed to unmarshal event",
			"routing_key", msg.RoutingKey,
			"error", err,
		)
		return nil // Return nil to ack and discard the bad message
	}

	// Override routing key from AMQP metadata if not in payload
	if event.RoutingKey == "" {
		event.RoutingKey = msg.RoutingKey
	}

	start := time.Now()
	err := c.registry.Dispatch(ctx, event)
	duration := time.Since(start)

	if err != nil {
		c.logger.Error("event dispatch failed",
			"routing_key", event.RoutingKey,
			"event_id", event.EventID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return err
	}

	c.logger.Debug("event processed successfully",
		"routing_key", event.RoutingKey,
		"event_id", event.EventID,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// Close closes the consumer connection.
func (c *RabbitMQConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.closeChan)
	c.running = false

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			c.logger.Warn("error closing channel", "error", err)
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}

	c.logger.Info("RabbitMQ consumer closed")
	return nil
}
