package publisher

import (
	"fmt"
	"jobu-utils/rabbitmq"
	"log"
	"sync"

	"github.com/streadway/amqp"
)

// Route type Schema - now embeds the shared RouteConfig
type Route struct {
	rabbitmq.RouteConfig
}

// Publish type Schema
type Publish struct {
	Headers amqp.Table
	Body    string
}

var m sync.Mutex

// Publish sends message to message broker
func (route *Route) Publish(publish *Publish) error {
	// Validate input
	if route == nil {
		return fmt.Errorf("route cannot be nil")
	}
	if publish == nil {
		return fmt.Errorf("publish cannot be nil")
	}
	if route.ExchangeName == "" {
		return fmt.Errorf("exchange name cannot be empty")
	}
	if route.QueueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}

	channel, err := rabbitmq.Start(&m)
	if err != nil {
		log.Printf("ERROR: Failed to connect to RabbitMQ: %v", err)
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Use shared setup function - this replaces the 3 separate calls
	if err := rabbitmq.SetupRouting(channel, &route.RouteConfig); err != nil {
		return fmt.Errorf("failed to setup routing: %w", err)
	}

	// Publish message
	if err := route.publishMessage(channel, publish); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Successfully published message to exchange: %s, routing key: %s", route.ExchangeName, route.RoutingKey)
	return nil
}

// publishMessage publishes the message to the exchange
func (route *Route) publishMessage(channel *amqp.Channel, publish *Publish) error {
	err := channel.Publish(
		route.ExchangeName, // exchange name
		route.RoutingKey,   // routing key
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         []byte(publish.Body),
			Headers:      publish.Headers,
		},
	)
	if err != nil {
		log.Printf("ERROR: Failed to publish message to exchange '%s': %v", route.ExchangeName, err)
		return err
	}
	return nil
}
