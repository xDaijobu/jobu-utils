package publisher

import (
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xDaijobu/jobu-utils/rabbitmq"
)

// Publisher represents a RabbitMQ publisher
type Publisher struct {
	config  *rabbitmq.RouteConfig
	channel *amqp.Channel
	mutex   sync.RWMutex
}

// Message represents a message to be published
type Message struct {
	Body        []byte
	ContentType string
	Priority    uint8
	Headers     amqp.Table
}

// NewPublisher creates a new publisher instance
func NewPublisher(config *rabbitmq.RouteConfig) (*Publisher, error) {
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

	return &Publisher{
		config:  config,
		channel: channel,
	}, nil
}

// Publish sends a message to the configured exchange
func (p *Publisher) Publish(msg *Message) error {
	return p.PublishWithRoutingKey(msg, p.config.RoutingKey)
}

// PublishWithRoutingKey sends a message with a custom routing key
func (p *Publisher) PublishWithRoutingKey(msg *Message, routingKey string) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Set default content type if not provided
	contentType := msg.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	// Create AMQP publishing
	publishing := amqp.Publishing{
		ContentType:  contentType,
		Body:         msg.Body,
		Priority:     msg.Priority,
		Headers:      msg.Headers,
		Timestamp:    time.Now(),
		DeliveryMode: amqp.Persistent, // Make messages persistent
	}

	// Publish the message
	err := p.channel.Publish(
		p.config.ExchangeName, // exchange
		routingKey,            // routing key
		false,                 // mandatory
		false,                 // immediate
		publishing,
	)

	if err != nil {
		log.Printf("ERROR: Failed to publish message: %v", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Message published to exchange '%s' with routing key '%s'",
		p.config.ExchangeName, routingKey)
	return nil
}

// PublishJSON is a convenience method for publishing JSON messages
func (p *Publisher) PublishJSON(data []byte) error {
	msg := &Message{
		Body:        data,
		ContentType: "application/json",
	}
	return p.Publish(msg)
}

// PublishText is a convenience method for publishing text messages
func (p *Publisher) PublishText(text string) error {
	msg := &Message{
		Body:        []byte(text),
		ContentType: "text/plain",
	}
	return p.Publish(msg)
}

// PublishWithPriority publishes a message with a specific priority
func (p *Publisher) PublishWithPriority(data []byte, priority uint8) error {
	msg := &Message{
		Body:        data,
		ContentType: "application/json",
		Priority:    priority,
	}
	return p.Publish(msg)
}

// Close gracefully closes the publisher
func (p *Publisher) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.channel != nil {
		return p.channel.Close()
	}
	return nil
}
