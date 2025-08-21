package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"jobu-utils/rabbitmq"
)

// Consumer represents a RabbitMQ consumer
type Consumer struct {
	config  *rabbitmq.RouteConfig
	channel *amqp.Channel
	mutex   sync.RWMutex
}

// MessageHandler is a function type for handling incoming messages
type MessageHandler func(msg amqp.Delivery) error

// NewConsumer creates a new consumer instance
func NewConsumer(config *rabbitmq.RouteConfig) (*Consumer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	var mutex sync.Mutex
	channel, err := rabbitmq.Start(&mutex)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Setup routing (exchange, queue, binding)
	if err := rabbitmq.SetupRouting(channel, config); err != nil {
		return nil, fmt.Errorf("failed to setup routing: %w", err)
	}

	return &Consumer{
		config:  config,
		channel: channel,
	}, nil
}

// Consume starts consuming messages with the provided handler
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Set QoS to control message prefetch
	if err := c.channel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	msgs, err := c.channel.Consume(
		c.config.QueueName, // queue
		"",                 // consumer tag
		false,              // auto-ack (we'll manually ack)
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Consumer started on queue: %s", c.config.QueueName)

	// Process messages
	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer stopped by context")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				log.Println("Message channel closed")
				return fmt.Errorf("message channel closed")
			}
			c.handleMessage(msg, handler)
		}
	}
}

// handleMessage processes a single message
func (c *Consumer) handleMessage(msg amqp.Delivery, handler MessageHandler) {
	start := time.Now()

	log.Printf("Processing message: %s", string(msg.Body))

	err := handler(msg)
	if err != nil {
		log.Printf("ERROR: Handler failed for message: %v", err)
		// Reject and requeue the message
		if nackErr := msg.Nack(false, true); nackErr != nil {
			log.Printf("ERROR: Failed to nack message: %v", nackErr)
		}
	} else {
		// Acknowledge successful processing
		if ackErr := msg.Ack(false); ackErr != nil {
			log.Printf("ERROR: Failed to ack message: %v", ackErr)
		}
		log.Printf("Message processed successfully in %v", time.Since(start))
	}
}

// Close gracefully closes the consumer
func (c *Consumer) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.channel != nil {
		return c.channel.Close()
	}
	return nil
}
