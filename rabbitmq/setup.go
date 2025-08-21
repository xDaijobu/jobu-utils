package rabbitmq

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RouteConfig contains the basic routing configuration shared between publisher and consumer
type RouteConfig struct {
	ExchangeName string
	ExchangeType string // direct, topic, fanout, headers
	RoutingKey   string
	QueueName    string
}

// SetupRouting performs all the necessary setup: declare exchange, declare queue, and bind them
// This consolidates the common setup operations used by both publisher and consumer
func SetupRouting(channel *amqp.Channel, config *RouteConfig) error {
	// Validate configuration
	if config == nil {
		return fmt.Errorf("route config cannot be nil")
	}
	if config.ExchangeName == "" {
		return fmt.Errorf("exchange name cannot be empty")
	}
	if config.QueueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}

	// Declare exchange
	if err := declareExchange(channel, config); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	if err := declareQueue(channel, config); err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	if err := bindQueue(channel, config); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Printf("Successfully set up routing: exchange='%s', queue='%s', routing_key='%s'",
		config.ExchangeName, config.QueueName, config.RoutingKey)

	return nil
}

// declareExchange declares the exchange
func declareExchange(channel *amqp.Channel, config *RouteConfig) error {
	err := channel.ExchangeDeclare(
		config.ExchangeName, // name
		config.ExchangeType, // type
		true,                // durable
		false,               // auto-delete
		false,               // internal
		false,               // no-wait
		nil,                 // argument
	)
	if err != nil {
		log.Printf("ERROR: Failed to declare exchange '%s': %v", config.ExchangeName, err)
		return err
	}
	return nil
}

// declareQueue declares the queue with optimized settings
func declareQueue(channel *amqp.Channel, config *RouteConfig) error {
	args := amqp.Table{
		"x-queue-mode":   "lazy",
		"x-max-priority": 255,
	}

	_, err := channel.QueueDeclare(
		config.QueueName, // queue name
		true,             // durable
		false,            // delete when used
		false,            // exclusive
		false,            // no-wait
		args,             // arguments
	)
	if err != nil {
		log.Printf("ERROR: Failed to declare queue '%s': %v", config.QueueName, err)
		return err
	}
	return nil
}

// bindQueue binds the queue to the exchange
func bindQueue(channel *amqp.Channel, config *RouteConfig) error {
	err := channel.QueueBind(
		config.QueueName,    // queue name
		config.RoutingKey,   // routing key
		config.ExchangeName, // exchange
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		log.Printf("ERROR: Failed to bind queue '%s' to exchange '%s': %v",
			config.QueueName, config.ExchangeName, err)
		return err
	}
	return nil
}
