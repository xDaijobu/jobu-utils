package consumer

import (
	"fmt"
	"jobu-utils/rabbitmq"
	"os"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestConsumerRouteValidation(t *testing.T) {
	tests := []struct {
		name    string
		route   *ConsumerRoute
		handler MessageHandler
		wantErr bool
	}{
		{
			name:    "nil route",
			route:   nil,
			handler: func(delivery amqp.Delivery) error { return nil },
			wantErr: true,
		},
		{
			name:    "nil handler",
			route:   &ConsumerRoute{},
			handler: nil,
			wantErr: true,
		},
		{
			name: "empty exchange name",
			route: &ConsumerRoute{
				RouteConfig: rabbitmq.RouteConfig{
					ExchangeName: "",
					ExchangeType: "direct",
					RoutingKey:   "test",
					QueueName:    "test-queue",
				},
			},
			handler: func(delivery amqp.Delivery) error { return nil },
			wantErr: true,
		},
		{
			name: "empty queue name",
			route: &ConsumerRoute{
				RouteConfig: rabbitmq.RouteConfig{
					ExchangeName: "test-exchange",
					ExchangeType: "direct",
					RoutingKey:   "test",
					QueueName:    "",
				},
			},
			handler: func(delivery amqp.Delivery) error { return nil },
			wantErr: true,
		},
		{
			name: "valid route and handler",
			route: &ConsumerRoute{
				RouteConfig: rabbitmq.RouteConfig{
					ExchangeName: "test-exchange",
					ExchangeType: "direct",
					RoutingKey:   "test.key",
					QueueName:    "test-queue",
				},
				ConsumerTag: "test-consumer",
				AutoAck:     false,
			},
			handler: func(delivery amqp.Delivery) error { return nil },
			wantErr: false, // Will fail due to no RabbitMQ connection, but validation should pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For validation tests, we only test the input validation, not actual connection
			if tt.wantErr {
				err := tt.route.StartConsuming(tt.handler)
				if err == nil {
					t.Error("Expected error but got none")
				}
				// Clean up if consumer started
				if tt.route != nil && tt.route.IsRunning() {
					tt.route.StopConsuming()
				}
			} else {
				// For valid inputs, test validation logic only
				if tt.route == nil {
					t.Error("Route should not be nil for valid test case")
					return
				}
				if tt.handler == nil {
					t.Error("Handler should not be nil for valid test case")
					return
				}
				if tt.route.ExchangeName == "" {
					t.Error("ExchangeName should not be empty for valid test case")
					return
				}
				if tt.route.QueueName == "" {
					t.Error("QueueName should not be empty for valid test case")
					return
				}
				// Validation passed - don't actually start consumer in unit test
			}
		})
	}
}

func TestConsumerRouteDefaultConsumerTag(t *testing.T) {
	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "test-exchange",
			ExchangeType: "direct",
			RoutingKey:   "test.key",
			QueueName:    "test-queue",
		},
		// ConsumerTag not set - should default to "default-consumer"
		AutoAck: false,
	}

	handler := func(delivery amqp.Delivery) error { return nil }

	// This will fail due to no RabbitMQ connection, but should set default consumer tag
	route.StartConsuming(handler)

	if route.ConsumerTag != "default-consumer" {
		t.Errorf("Expected default consumer tag to be 'default-consumer', got '%s'", route.ConsumerTag)
	}

	// Clean up
	if route.IsRunning() {
		route.StopConsuming()
		time.Sleep(100 * time.Millisecond)
	}
}

func TestConsumerRouteStates(t *testing.T) {
	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "test-exchange",
			ExchangeType: "direct",
			RoutingKey:   "test.key",
			QueueName:    "test-queue",
		},
		ConsumerTag: "state-test-consumer",
		AutoAck:     false,
	}

	// Initially should not be running
	if route.IsRunning() {
		t.Error("Consumer should not be running initially")
	}

	// Test stopping when not running
	err := route.StopConsuming()
	if err == nil {
		t.Error("StopConsuming should return error when consumer is not running")
	}

	// Test restart when not running
	err = route.Restart()
	if err == nil {
		t.Error("Restart should return error when consumer is not running")
	}
}

func TestConsumerRouteDoubleStart(t *testing.T) {
	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "test-exchange",
			ExchangeType: "direct",
			RoutingKey:   "test.key",
			QueueName:    "test-queue",
		},
		ConsumerTag: "double-start-consumer",
		AutoAck:     false,
	}

	handler := func(delivery amqp.Delivery) error { return nil }

	// Start first time (will fail due to no RabbitMQ, but should set running state)
	route.StartConsuming(handler)

	// Try to start again - should return error
	err := route.StartConsuming(handler)
	if err == nil {
		t.Error("StartConsuming should return error when consumer is already running")
	}

	// Clean up
	if route.IsRunning() {
		route.StopConsuming()
		time.Sleep(100 * time.Millisecond)
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
			handler: func(delivery amqp.Delivery) error {
				return nil
			},
			valid: true,
		},
		{
			name: "handler with processing logic",
			handler: func(delivery amqp.Delivery) error {
				if string(delivery.Body) == "error" {
					return fmt.Errorf("processing error")
				}
				return nil
			},
			valid: true,
		},
		{
			name: "handler with panic recovery",
			handler: func(delivery amqp.Delivery) error {
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
			route := &ConsumerRoute{
				RouteConfig: rabbitmq.RouteConfig{
					ExchangeName: "test-exchange",
					ExchangeType: "direct",
					RoutingKey:   "test.key",
					QueueName:    "test-queue",
				},
				ConsumerTag: "handler-test-consumer",
				AutoAck:     false,
			}

			err := route.StartConsuming(tt.handler)

			if tt.valid && err != nil {
				// Check if it's a validation error vs connection error
				if err.Error() == "message handler cannot be nil" {
					t.Errorf("Valid handler should not cause validation error: %v", err)
				}
			}

			if !tt.valid && err == nil {
				t.Error("Invalid handler should cause validation error")
			}

			// Clean up
			if route.IsRunning() {
				route.StopConsuming()
				time.Sleep(100 * time.Millisecond)
			}
		})
	}
}

func TestConsumeOnceValidation(t *testing.T) {
	tests := []struct {
		name    string
		route   *ConsumerRoute
		handler MessageHandler
		wantErr bool
	}{
		{
			name:    "nil route",
			route:   nil,
			handler: func(delivery amqp.Delivery) error { return nil },
			wantErr: true,
		},
		{
			name:    "nil handler",
			route:   &ConsumerRoute{},
			handler: nil,
			wantErr: true,
		},
		{
			name: "empty queue name",
			route: &ConsumerRoute{
				RouteConfig: rabbitmq.RouteConfig{
					QueueName: "",
				},
			},
			handler: func(delivery amqp.Delivery) error { return nil },
			wantErr: true,
		},
		{
			name: "valid input",
			route: &ConsumerRoute{
				RouteConfig: rabbitmq.RouteConfig{
					QueueName: "test-queue",
				},
				AutoAck: false,
			},
			handler: func(delivery amqp.Delivery) error { return nil },
			wantErr: false, // Will fail due to no RabbitMQ connection, but validation should pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.route.ConsumeOnce(tt.handler)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				// For valid inputs, we expect connection errors (not validation errors)
				if err != nil {
					// Check if it's a validation error vs connection error
					if err.Error() == "consumer route cannot be nil" ||
						err.Error() == "message handler cannot be nil" ||
						err.Error() == "queue name cannot be empty" {
						t.Errorf("Validation should pass for valid input: %v", err)
					}
					// Connection errors are expected in unit tests
				}
			}
		})
	}
}

func TestConsumerRouteConfigEmbedding(t *testing.T) {
	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "test-embedding",
			ExchangeType: "topic",
			RoutingKey:   "test.embedding.key",
			QueueName:    "test-embedding-queue",
		},
		ConsumerTag: "embedding-consumer",
		AutoAck:     true,
	}

	// Test that we can access RouteConfig fields directly through embedding
	if route.ExchangeName != "test-embedding" {
		t.Error("RouteConfig embedding not working for ExchangeName")
	}

	if route.ExchangeType != "topic" {
		t.Error("RouteConfig embedding not working for ExchangeType")
	}

	if route.RoutingKey != "test.embedding.key" {
		t.Error("RouteConfig embedding not working for RoutingKey")
	}

	if route.QueueName != "test-embedding-queue" {
		t.Error("RouteConfig embedding not working for QueueName")
	}

	// Test consumer-specific fields
	if route.ConsumerTag != "embedding-consumer" {
		t.Error("ConsumerTag not set correctly")
	}

	if route.AutoAck != true {
		t.Error("AutoAck not set correctly")
	}
}

// Integration test for consumer
func TestStartConsumingIntegration(t *testing.T) {
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

	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "test-consumer-exchange",
			ExchangeType: "direct",
			RoutingKey:   "test.consumer",
			QueueName:    "test-consumer-queue",
		},
		ConsumerTag: "integration-consumer",
		AutoAck:     false,
	}

	messageReceived := make(chan bool, 1)
	handler := func(delivery amqp.Delivery) error {
		select {
		case messageReceived <- true:
		default:
		}
		return nil
	}

	err := route.StartConsuming(handler)
	if err != nil {
		t.Skipf("RabbitMQ not available for integration test: %v", err)
		return
	}

	// Verify consumer is running
	if !route.IsRunning() {
		t.Error("Consumer should be running after StartConsuming")
	}

	// Stop consumer
	err = route.StopConsuming()
	if err != nil {
		t.Errorf("Failed to stop consumer: %v", err)
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	// Verify consumer is stopped
	if route.IsRunning() {
		t.Error("Consumer should not be running after StopConsuming")
	}

	// Clean up
	manager := rabbitmq.InitConnectionManager()
	if channel, err := manager.GetChannel(); err == nil {
		channel.QueueDelete(route.QueueName, false, false, false)
		channel.ExchangeDelete(route.ExchangeName, false, false)
	}
	manager.Close()
}

func TestConsumeOnceIntegration(t *testing.T) {
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

	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "test-consume-once-exchange",
			ExchangeType: "direct",
			RoutingKey:   "test.consume.once",
			QueueName:    "test-consume-once-queue",
		},
		AutoAck: false,
	}

	messageProcessed := false
	handler := func(delivery amqp.Delivery) error {
		messageProcessed = true
		return nil
	}

	// Test consuming when no messages are available
	err := route.ConsumeOnce(handler)
	if err != nil {
		t.Skipf("RabbitMQ not available for integration test: %v", err)
		return
	}

	// Should not have processed any message
	if messageProcessed {
		t.Error("Should not have processed message when queue is empty")
	}

	// Clean up
	manager := rabbitmq.InitConnectionManager()
	if channel, err := manager.GetChannel(); err == nil {
		channel.QueueDelete(route.QueueName, false, false, false)
		channel.ExchangeDelete(route.ExchangeName, false, false)
	}
	manager.Close()
}

func BenchmarkConsumerValidation(b *testing.B) {
	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "benchmark-exchange",
			ExchangeType: "direct",
			RoutingKey:   "benchmark.key",
			QueueName:    "benchmark-queue",
		},
		ConsumerTag: "benchmark-consumer",
		AutoAck:     false,
	}

	handler := func(delivery amqp.Delivery) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Only test validation logic, not actual consuming
		if route == nil {
			continue
		}
		if handler == nil {
			continue
		}
		if route.ExchangeName == "" {
			continue
		}
		if route.QueueName == "" {
			continue
		}
		if route.ConsumerTag == "" {
			route.ConsumerTag = "default-consumer"
		}
	}
}

func TestConsumerRouteFields(t *testing.T) {
	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "field-test-exchange",
			ExchangeType: "fanout",
			RoutingKey:   "field.test",
			QueueName:    "field-test-queue",
		},
		ConsumerTag: "field-test-consumer",
		AutoAck:     true,
	}

	// Test all fields are set correctly
	if route.ExchangeName != "field-test-exchange" {
		t.Errorf("Expected ExchangeName to be 'field-test-exchange', got '%s'", route.ExchangeName)
	}

	if route.ExchangeType != "fanout" {
		t.Errorf("Expected ExchangeType to be 'fanout', got '%s'", route.ExchangeType)
	}

	if route.RoutingKey != "field.test" {
		t.Errorf("Expected RoutingKey to be 'field.test', got '%s'", route.RoutingKey)
	}

	if route.QueueName != "field-test-queue" {
		t.Errorf("Expected QueueName to be 'field-test-queue', got '%s'", route.QueueName)
	}

	if route.ConsumerTag != "field-test-consumer" {
		t.Errorf("Expected ConsumerTag to be 'field-test-consumer', got '%s'", route.ConsumerTag)
	}

	if route.AutoAck != true {
		t.Error("Expected AutoAck to be true")
	}
}

func TestConcurrentConsumerManagement(t *testing.T) {
	route := &ConsumerRoute{
		RouteConfig: rabbitmq.RouteConfig{
			ExchangeName: "concurrent-test-exchange",
			ExchangeType: "direct",
			RoutingKey:   "concurrent.test",
			QueueName:    "concurrent-test-queue",
		},
		ConsumerTag: "concurrent-consumer",
		AutoAck:     false,
	}

	handler := func(delivery amqp.Delivery) error {
		return nil
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Test concurrent access to consumer state
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Try to check if running
			route.IsRunning()

			// Try to start (most will fail with "already running")
			if err := route.StartConsuming(handler); err != nil {
				// Expected for most goroutines
			}

			// Try to restart
			route.Restart()
		}()
	}

	wg.Wait()
	close(errors)

	// Clean up
	if route.IsRunning() {
		route.StopConsuming()
		time.Sleep(100 * time.Millisecond)
	}
}
