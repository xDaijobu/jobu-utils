package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xDaijobu/jobu-utils/rabbitmq"
	"github.com/xDaijobu/jobu-utils/rabbitmq/consumer"
	"github.com/xDaijobu/jobu-utils/rabbitmq/publisher"
)

type Message struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

func main() {
	log.Println("🚀 Starting Simple RabbitMQ Example...")

	// Set environment variables
	os.Setenv("RABBITMQ_HOST", "localhost")
	os.Setenv("RABBITMQ_PORT", "5672")
	os.Setenv("RABBITMQ_USER", "admin")
	os.Setenv("RABBITMQ_PASSWORD", "password")

	// Configure routing
	config := &rabbitmq.RouteConfig{
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test.message",
		QueueName:    "test-queue",
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("🛑 Shutting down...")
		cancel()
	}()

	// Start consumer
	go startConsumer(ctx, config)

	// Start publisher
	go startPublisher(ctx, config)

	// Wait for shutdown
	<-ctx.Done()
	log.Println("✅ Done")
}

func startConsumer(ctx context.Context, config *rabbitmq.RouteConfig) {
	log.Println("📥 Starting Consumer...")

	cons, err := consumer.NewConsumer(config)
	if err != nil {
		log.Printf("❌ Consumer error: %v", err)
		return
	}

	handler := func(delivery amqp.Delivery) error {
		var msg Message
		if err := json.Unmarshal(delivery.Body, &msg); err != nil {
			log.Printf("❌ Parse error: %v", err)
			return delivery.Reject(false)
		}

		log.Printf("📩 Received: ID=%d, Content=%s, Time=%s", msg.ID, msg.Content, msg.Time)

		// Don't manually ack - the consumer library handles this automatically
		return nil
	}

	cons.Consume(ctx, handler)
}

func startPublisher(ctx context.Context, config *rabbitmq.RouteConfig) {
	log.Println("📤 Starting Publisher...")

	// Wait a bit for consumer to start
	time.Sleep(2 * time.Second)

	pub, err := publisher.NewPublisher(config)
	if err != nil {
		log.Printf("❌ Publisher error: %v", err)
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	id := 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg := Message{
				ID:      id,
				Content: "Hello World!",
				Time:    time.Now().Format("15:04:05"),
			}

			data, _ := json.Marshal(msg)
			message := &publisher.Message{
				Body:        data,
				ContentType: "application/json",
			}

			if err := pub.Publish(message); err != nil {
				log.Printf("❌ Publish error: %v", err)
			} else {
				log.Printf("📤 Sent: ID=%d, Content=%s", msg.ID, msg.Content)
			}

			id++
		}
	}
}
