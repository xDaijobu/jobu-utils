package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xDaijobu/jobu-utils/rabbitmq"
	"github.com/xDaijobu/jobu-utils/rabbitmq/consumer"
	"github.com/xDaijobu/jobu-utils/rabbitmq/publisher"
)

// UserEvent represents a user event message
type UserEvent struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	log.Println("🚀 Starting RabbitMQ Example...")

	// Set default environment variables if not set
	setDefaultEnvVars()

	// Configure routing for user events
	config := &rabbitmq.RouteConfig{
		ExchangeName: "user-events",
		ExchangeType: "direct",
		RoutingKey:   "user.created",
		QueueName:    "user-notifications",
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start consumer in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		runConsumer(ctx, config)
	}()

	// Start publisher in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		runPublisher(ctx, config)
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("🛑 Shutting down gracefully...")
	cancel()

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("✅ Shutdown complete")
}

// runConsumer starts the message consumer
func runConsumer(ctx context.Context, config *rabbitmq.RouteConfig) {
	log.Println("📥 Starting Consumer...")

	for {
		select {
		case <-ctx.Done():
			log.Println("📥 Consumer stopped by context")
			return
		default:
			// Create consumer with retry logic
			cons, err := consumer.NewConsumer(config)
			if err != nil {
				log.Printf("❌ Failed to create consumer: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			// Message handler function
			messageHandler := func(delivery amqp.Delivery) error {
				log.Printf("📩 Received message from exchange: %s, routing key: %s",
					delivery.Exchange, delivery.RoutingKey)

				// Parse the user event
				var event UserEvent
				if err := json.Unmarshal(delivery.Body, &event); err != nil {
					log.Printf("❌ Failed to parse message: %v", err)
					// Reject message and don't requeue on parse error
					if rejectErr := delivery.Reject(false); rejectErr != nil {
						log.Printf("❌ Failed to reject message: %v", rejectErr)
					}
					return nil // Don't return error to avoid channel issues
				}

				// Process the event
				log.Printf("✅ Processing user event: User %s (%s) - Action: %s at %s",
					event.UserID, event.Email, event.Action, event.Timestamp.Format(time.RFC3339))

				// Simulate processing time
				time.Sleep(100 * time.Millisecond)

				// Acknowledge the message - this is where the error was happening
				if ackErr := delivery.Ack(false); ackErr != nil {
					log.Printf("❌ Failed to acknowledge message: %v", ackErr)
					// Don't return error, just log it to avoid breaking the channel
					return nil
				}

				log.Println("✅ Message processed and acknowledged")
				return nil
			}

			// Start consuming messages with error handling
			log.Println("🔄 Starting to consume messages...")
			if err := cons.Consume(ctx, messageHandler); err != nil {
				if ctx.Err() != nil {
					// Context was cancelled, exit gracefully
					log.Println("📥 Consumer stopped")
					return
				}
				log.Printf("❌ Consumer error: %v", err)
				log.Println("🔄 Restarting consumer in 2 seconds...")
				time.Sleep(2 * time.Second)
				continue
			}
		}
	}
}

// runPublisher starts the message publisher
func runPublisher(ctx context.Context, config *rabbitmq.RouteConfig) {
	log.Println("📤 Starting Publisher...")

	// Wait a bit before starting to publish to ensure consumer is ready
	time.Sleep(2 * time.Second)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	userID := 1
	var pub *publisher.Publisher

	for {
		select {
		case <-ctx.Done():
			log.Println("📤 Publisher stopped")
			return
		case <-ticker.C:
			// Create or recreate publisher if needed
			if pub == nil {
				var err error
				pub, err = publisher.NewPublisher(config)
				if err != nil {
					log.Printf("❌ Failed to create publisher: %v", err)
					continue
				}
				log.Println("📤 Publisher (re)connected successfully")
			}

			// Create a user event
			event := UserEvent{
				UserID:    fmt.Sprintf("user_%d", userID),
				Email:     fmt.Sprintf("user%d@example.com", userID),
				Action:    "created",
				Timestamp: time.Now(),
			}

			// Marshal to JSON
			eventData, err := json.Marshal(event)
			if err != nil {
				log.Printf("❌ Failed to marshal event: %v", err)
				continue
			}

			// Create message
			message := &publisher.Message{
				Body:        eventData,
				ContentType: "application/json",
				Headers: amqp.Table{
					"source":    "user-service",
					"version":   "1.0",
					"timestamp": time.Now().Unix(),
				},
			}

			// Publish message with retry logic
			log.Printf("📤 Publishing user event for user: %s", event.UserID)
			if err := pub.Publish(message); err != nil {
				log.Printf("❌ Failed to publish message: %v", err)
				// If publish fails, reset publisher to force reconnection
				pub = nil
				log.Println("🔄 Will reconnect publisher on next attempt")
			} else {
				log.Printf("✅ Successfully published message for user: %s", event.UserID)
			}

			userID++
		}
	}
}

// setDefaultEnvVars sets default environment variables if they're not already set
func setDefaultEnvVars() {
	envVars := map[string]string{
		"RABBITMQ_HOST":     "localhost",
		"RABBITMQ_PORT":     "5672",
		"RABBITMQ_USER":     "admin",
		"RABBITMQ_PASSWORD": "password",
	}

	for key, defaultValue := range envVars {
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, defaultValue); err != nil {
				log.Printf("❌ Failed to set environment variable %s: %v", key, err)
			} else {
				log.Printf("🔧 Set %s to default value: %s", key, defaultValue)
			}
		}
	}
}
