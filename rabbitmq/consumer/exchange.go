package consumer

import (
	"fmt"
	"jobu-utils/rabbitmq"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// ConsumerRoute defines the routing configuration for consuming messages
// Now embeds the shared RouteConfig and adds consumer-specific fields
type ConsumerRoute struct {
	rabbitmq.RouteConfig
	ConsumerTag string
	AutoAck     bool

	// Auto-reconnect fields
	isRunning   bool
	stopChan    chan bool
	restartChan chan bool
	mutex       sync.RWMutex
}

// MessageHandler defines the function signature for handling consumed messages
type MessageHandler func(delivery amqp.Delivery) error

var m sync.Mutex

// StartConsuming starts consuming messages from the queue with auto-reconnect
func (route *ConsumerRoute) StartConsuming(handler MessageHandler) error {
	// Validate input
	if route == nil {
		return fmt.Errorf("consumer route cannot be nil")
	}
	if handler == nil {
		return fmt.Errorf("message handler cannot be nil")
	}
	if route.ExchangeName == "" {
		return fmt.Errorf("exchange name cannot be empty")
	}
	if route.QueueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}
	if route.ConsumerTag == "" {
		route.ConsumerTag = "default-consumer"
	}

	route.mutex.Lock()
	if route.isRunning {
		route.mutex.Unlock()
		return fmt.Errorf("consumer is already running")
	}

	route.isRunning = true
	route.stopChan = make(chan bool, 1)
	route.restartChan = make(chan bool, 1)
	route.mutex.Unlock()

	// Start the consumption loop with auto-reconnect
	go route.consumeWithReconnect(handler)

	log.Printf("Consumer started with auto-reconnect. Queue: %s, Consumer: %s", route.QueueName, route.ConsumerTag)
	return nil
}

// consumeWithReconnect manages the consumption loop with automatic reconnection
func (route *ConsumerRoute) consumeWithReconnect(handler MessageHandler) {
	defer func() {
		route.mutex.Lock()
		route.isRunning = false
		route.mutex.Unlock()
		log.Printf("Consumer stopped: %s", route.ConsumerTag)
	}()

	for {
		// Check if we should stop
		select {
		case <-route.stopChan:
			return
		default:
		}

		// Start consuming
		if err := route.startSingleConsumption(handler); err != nil {
			log.Printf("ERROR: Consumer failed: %v", err)
		}

		// Wait for restart signal or stop signal
		select {
		case <-route.stopChan:
			return
		case <-route.restartChan:
			log.Printf("Restarting consumer: %s", route.ConsumerTag)
			time.Sleep(1 * time.Second) // Brief pause before restart
			continue
		case <-time.After(5 * time.Second):
			// Auto-restart after 5 seconds if no explicit restart signal
			log.Printf("Auto-restarting consumer after timeout: %s", route.ConsumerTag)
			continue
		}
	}
}

// startSingleConsumption starts a single consumption session
func (route *ConsumerRoute) startSingleConsumption(handler MessageHandler) error {
	channel, err := rabbitmq.Start(&m)
	if err != nil {
		log.Printf("ERROR: Failed to connect to RabbitMQ: %v", err)
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Use shared setup function
	if err := rabbitmq.SetupRouting(channel, &route.RouteConfig); err != nil {
		return fmt.Errorf("failed to setup routing: %w", err)
	}

	// Start consuming
	msgs, err := channel.Consume(
		route.QueueName,   // queue
		route.ConsumerTag, // consumer
		route.AutoAck,     // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		log.Printf("ERROR: Failed to register consumer: %v", err)
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Consumer started. Waiting for messages on queue: %s", route.QueueName)

	// Process messages with connection monitoring
	route.processMessagesWithMonitoring(msgs, handler, channel)
	return nil
}

// processMessagesWithMonitoring processes messages and monitors for connection issues
func (route *ConsumerRoute) processMessagesWithMonitoring(msgs <-chan amqp.Delivery, handler MessageHandler, channel *amqp.Channel) {
	// Monitor channel close
	closeChan := channel.NotifyClose(make(chan *amqp.Error))

	for {
		select {
		case <-route.stopChan:
			return

		case delivery, ok := <-msgs:
			if !ok {
				log.Printf("Message channel closed for consumer: %s", route.ConsumerTag)
				// Signal restart
				select {
				case route.restartChan <- true:
				default:
				}
				return
			}

			log.Printf("Received message: %s", string(delivery.Body))

			// Handle the message
			if err := handler(delivery); err != nil {
				log.Printf("ERROR: Failed to handle message: %v", err)

				// If not auto-ack, reject the message
				if !route.AutoAck {
					if nackErr := delivery.Nack(false, true); nackErr != nil {
						log.Printf("ERROR: Failed to nack message: %v", nackErr)
					}
				}
				continue
			}

			// If not auto-ack, acknowledge the message
			if !route.AutoAck {
				if ackErr := delivery.Ack(false); ackErr != nil {
					log.Printf("ERROR: Failed to ack message: %v", ackErr)
				}
			}

			log.Printf("Message processed successfully")

		case closeErr := <-closeChan:
			if closeErr != nil {
				log.Printf("Channel closed for consumer %s: %v", route.ConsumerTag, closeErr)
			}
			// Signal restart
			select {
			case route.restartChan <- true:
			default:
			}
			return
		}
	}
}

// StopConsuming stops the consumer gracefully
func (route *ConsumerRoute) StopConsuming() error {
	route.mutex.Lock()
	defer route.mutex.Unlock()

	if !route.isRunning {
		return fmt.Errorf("consumer is not running")
	}

	// Cancel the consumer first (if connected)
	if channel, err := rabbitmq.Start(&m); err == nil {
		if cancelErr := channel.Cancel(route.ConsumerTag, false); cancelErr != nil {
			log.Printf("WARNING: Failed to cancel consumer: %v", cancelErr)
		}
	}

	// Signal stop
	select {
	case route.stopChan <- true:
	default:
	}

	log.Printf("Consumer stop signal sent: %s", route.ConsumerTag)
	return nil
}

// IsRunning returns true if the consumer is currently running
func (route *ConsumerRoute) IsRunning() bool {
	route.mutex.RLock()
	defer route.mutex.RUnlock()
	return route.isRunning
}

// Restart triggers a manual restart of the consumer
func (route *ConsumerRoute) Restart() error {
	route.mutex.RLock()
	isRunning := route.isRunning
	route.mutex.RUnlock()

	if !isRunning {
		return fmt.Errorf("consumer is not running")
	}

	select {
	case route.restartChan <- true:
		log.Printf("Consumer restart triggered: %s", route.ConsumerTag)
		return nil
	default:
		return fmt.Errorf("restart already pending")
	}
}

// ConsumeOnce consumes a single message from the queue (useful for testing or one-off consumption)
func (route *ConsumerRoute) ConsumeOnce(handler MessageHandler) error {
	// Validate input
	if route == nil {
		return fmt.Errorf("consumer route cannot be nil")
	}
	if handler == nil {
		return fmt.Errorf("message handler cannot be nil")
	}
	if route.QueueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}

	// Try with auto-reconnect
	var delivery amqp.Delivery
	var ok bool
	var err error

	maxAttempts := 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		channel, connErr := rabbitmq.Start(&m)
		if connErr != nil {
			log.Printf("ERROR: Failed to connect to RabbitMQ (attempt %d): %v", attempt+1, connErr)
			if attempt < maxAttempts-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", maxAttempts, connErr)
		}

		// Get a single message
		delivery, ok, err = channel.Get(route.QueueName, route.AutoAck)
		if err != nil {
			log.Printf("ERROR: Failed to get message from queue '%s' (attempt %d): %v", route.QueueName, attempt+1, err)
			if attempt < maxAttempts-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return fmt.Errorf("failed to get message after %d attempts: %w", maxAttempts, err)
		}

		break // Success
	}

	if !ok {
		log.Printf("No messages available in queue: %s", route.QueueName)
		return nil
	}

	log.Printf("Received single message: %s", string(delivery.Body))

	// Handle the message
	if err := handler(delivery); err != nil {
		log.Printf("ERROR: Failed to handle message: %v", err)

		// If not auto-ack, reject the message
		if !route.AutoAck {
			if nackErr := delivery.Nack(false, true); nackErr != nil {
				log.Printf("ERROR: Failed to nack message: %v", nackErr)
			}
		}
		return fmt.Errorf("failed to handle message: %w", err)
	}

	// If not auto-ack, acknowledge the message
	if !route.AutoAck {
		if ackErr := delivery.Ack(false); ackErr != nil {
			log.Printf("ERROR: Failed to ack message: %v", ackErr)
			return fmt.Errorf("failed to ack message: %w", err)
		}
	}

	log.Printf("Single message processed successfully")
	return nil
}
