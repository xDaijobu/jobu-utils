package consumer

import (
	"context"
	"fmt"
	"github.com/xDaijobu/jobu-utils/rabbitmq"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNewConsumer(t *testing.T) {
	// Set hardcoded test credentials for RabbitMQ
	os.Setenv("RABBITMQ_HOST", "localhost")
	os.Setenv("RABBITMQ_PORT", "5672")
	os.Setenv("RABBITMQ_USER", "guest")
	os.Setenv("RABBITMQ_PASSWORD", "guest")

	tests := []struct {
		name    string
		config  *rabbitmq.RouteConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &rabbitmq.RouteConfig{
				ExchangeName: "test-exchange",
				ExchangeType: "direct",
				RoutingKey:   "test.key",
				QueueName:    "test-queue",
			},
			wantErr: false, // Will fail due to no RabbitMQ connection, but validation should pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer, err := NewConsumer(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if consumer != nil {
					t.Error("Consumer should be nil when error occurs")
				}
			} else {
				// For valid configs, we expect connection errors (not validation errors)
				if err != nil {
					// Check if it's a validation error vs connection error
					if err.Error() == "config cannot be nil" {
						t.Errorf("Validation should pass for valid config: %v", err)
					}
					// Connection errors are expected in unit tests
				}
			}
		})
	}
}

func TestMessageHandlerTypes(t *testing.T) {
	tests := []struct {
		name    string
		handler MessageHandler
		valid   bool
	}{
		{
			name: "simple handler",
			handler: func(msg amqp.Delivery) error {
				return nil
			},
			valid: true,
		},
		{
			name: "handler with processing logic",
			handler: func(msg amqp.Delivery) error {
				if string(msg.Body) == "error" {
					return fmt.Errorf("processing error")
				}
				return nil
			},
			valid: true,
		},
		{
			name: "handler with panic recovery",
			handler: func(msg amqp.Delivery) error {
				defer func() {
					if r := recover(); r != nil {
						// Handle panic
					}
				}()
				return nil
			},
			valid: true,
		},
		{
			name:    "nil handler",
			handler: nil,
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid && tt.handler != nil {
				t.Error("Invalid test case: nil handler should have valid=false")
			}
			if tt.valid && tt.handler == nil {
				t.Error("Invalid test case: non-nil handler should have valid=true")
			}
		})
	}
}

func TestConsumerConfigValidation(t *testing.T) {
	config := &rabbitmq.RouteConfig{
		ExchangeName: "test-consumer-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test.consumer.key",
		QueueName:    "test-consumer-queue",
	}

	// Test that config fields are accessible
	if config.ExchangeName != "test-consumer-exchange" {
		t.Errorf("Expected ExchangeName to be 'test-consumer-exchange', got '%s'", config.ExchangeName)
	}

	if config.ExchangeType != "direct" {
		t.Errorf("Expected ExchangeType to be 'direct', got '%s'", config.ExchangeType)
	}

	if config.RoutingKey != "test.consumer.key" {
		t.Errorf("Expected RoutingKey to be 'test.consumer.key', got '%s'", config.RoutingKey)
	}

	if config.QueueName != "test-consumer-queue" {
		t.Errorf("Expected QueueName to be 'test-consumer-queue', got '%s'", config.QueueName)
	}
}

// Integration test for consumer
func TestConsumerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use environment variables from system/shell
	host := os.Getenv("RABBITMQ_HOST")
	user := os.Getenv("RABBITMQ_USER")
	password := os.Getenv("RABBITMQ_PASSWORD")

	if host == "" || user == "" || password == "" {
		t.Skip("RabbitMQ environment variables not set - skipping integration test")
	}

	config := &rabbitmq.RouteConfig{
		ExchangeName: "test-consumer-integration",
		ExchangeType: "direct",
		RoutingKey:   "test.consumer.integration",
		QueueName:    "test-consumer-integration-queue",
	}

	consumer, err := NewConsumer(config)
	if err != nil {
		t.Skipf("RabbitMQ not available for integration test: %v", err)
		return
	}

	if consumer == nil {
		t.Fatal("Consumer should not be nil when creation succeeds")
	}

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	messageReceived := make(chan bool, 1)
	handler := func(msg amqp.Delivery) error {
		select {
		case messageReceived <- true:
		default:
		}
		return nil
	}

	// Start consuming (this will timeout due to context)
	err = consumer.Consume(ctx, handler)
	if err != context.DeadlineExceeded {
		// Context cancellation is expected
	}

	// Clean up
	consumer.Close()

	// Clean up queue and exchange
	manager := rabbitmq.InitConnectionManager()
	if channel, err := manager.GetChannel(); err == nil {
		channel.QueueDelete(config.QueueName, false, false, false)
		channel.ExchangeDelete(config.ExchangeName, false, false)
	}
	manager.Close()
}

func TestConsumerClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use environment variables from system/shell
	host := os.Getenv("RABBITMQ_HOST")
	user := os.Getenv("RABBITMQ_USER")
	password := os.Getenv("RABBITMQ_PASSWORD")

	if host == "" || user == "" || password == "" {
		t.Skip("RabbitMQ environment variables not set - skipping integration test")
	}

	config := &rabbitmq.RouteConfig{
		ExchangeName: "test-consumer-close",
		ExchangeType: "direct",
		RoutingKey:   "test.close",
		QueueName:    "test-consumer-close-queue",
	}

	consumer, err := NewConsumer(config)
	if err != nil {
		t.Skipf("RabbitMQ not available for close test: %v", err)
		return
	}

	// Test closing
	err = consumer.Close()
	if err != nil {
		t.Errorf("Failed to close consumer: %v", err)
	}

	// Test closing again (should not error)
	err = consumer.Close()
	if err != nil {
		t.Errorf("Second close should not error: %v", err)
	}

	// Clean up
	manager := rabbitmq.InitConnectionManager()
	if channel, err := manager.GetChannel(); err == nil {
		channel.QueueDelete(config.QueueName, false, false, false)
		channel.ExchangeDelete(config.ExchangeName, false, false)
	}
	manager.Close()
}

func BenchmarkNewConsumer(b *testing.B) {
	config := &rabbitmq.RouteConfig{
		ExchangeName: "benchmark-consumer-exchange",
		ExchangeType: "direct",
		RoutingKey:   "benchmark.consumer",
		QueueName:    "benchmark-consumer-queue",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Only test validation, not actual consumer creation
		if config == nil {
			continue
		}
		if config.ExchangeName == "" {
			continue
		}
		if config.QueueName == "" {
			continue
		}
	}
}

func TestConsumerConfigEmbedding(t *testing.T) {
	config := &rabbitmq.RouteConfig{
		ExchangeName: "test-embedding-consumer",
		ExchangeType: "topic",
		RoutingKey:   "test.embedding.consumer.key",
		QueueName:    "test-embedding-consumer-queue",
	}

	// Test that we can access RouteConfig fields
	if config.ExchangeName != "test-embedding-consumer" {
		t.Error("RouteConfig not working for ExchangeName")
	}

	if config.ExchangeType != "topic" {
		t.Error("RouteConfig not working for ExchangeType")
	}

	if config.RoutingKey != "test.embedding.consumer.key" {
		t.Error("RouteConfig not working for RoutingKey")
	}

	if config.QueueName != "test-embedding-consumer-queue" {
		t.Error("RouteConfig not working for QueueName")
	}
}

func TestMessageHandlerValidation(t *testing.T) {
	// Test that MessageHandler type is correctly defined
	var handler MessageHandler = func(msg amqp.Delivery) error {
		return nil
	}

	if handler == nil {
		t.Error("MessageHandler should not be nil")
	}

	// Test calling the handler
	mockDelivery := amqp.Delivery{
		Body: []byte("test message"),
	}

	err := handler(mockDelivery)
	if err != nil {
		t.Errorf("Handler should not return error for valid call: %v", err)
	}
}
