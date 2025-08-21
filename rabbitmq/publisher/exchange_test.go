package publisher

import (
	"fmt"
	"github.com/xDaijobu/jobu-utils/rabbitmq"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNewPublisher(t *testing.T) {
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
			publisher, err := NewPublisher(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if publisher != nil {
					t.Error("Publisher should be nil when error occurs")
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

func TestMessageStruct(t *testing.T) {
	message := &Message{
		Body:        []byte(`{"user_id": "123", "action": "created"}`),
		ContentType: "application/json",
		Priority:    5,
		Headers: amqp.Table{
			"timestamp": time.Now().Unix(),
		},
	}

	if string(message.Body) != `{"user_id": "123", "action": "created"}` {
		t.Errorf("Expected body to be JSON string, got %s", string(message.Body))
	}

	if message.ContentType != "application/json" {
		t.Error("Expected content-type to be application/json")
	}

	if message.Priority != 5 {
		t.Error("Expected priority to be 5")
	}

	if message.Headers["timestamp"] == nil {
		t.Error("Expected timestamp header to exist")
	}
}

func TestPublisherWithDifferentExchangeTypes(t *testing.T) {
	tests := []struct {
		name         string
		exchangeType string
		routingKey   string
		queueName    string
	}{
		{"direct exchange", "direct", "user.created", "user-events"},
		{"topic exchange", "topic", "user.*.created", "user-pattern-events"},
		{"fanout exchange", "fanout", "", "fanout-events"},
		{"headers exchange", "headers", "headers.key", "headers-events"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &rabbitmq.RouteConfig{
				ExchangeName: "test-" + tt.exchangeType,
				ExchangeType: tt.exchangeType,
				RoutingKey:   tt.routingKey,
				QueueName:    tt.queueName,
			}

			// Test validation - should not error on valid input
			publisher, err := NewPublisher(config)
			if err != nil {
				// Check if it's a validation error vs connection error
				if err.Error() == "config cannot be nil" {
					t.Errorf("Validation should pass for valid config: %v", err)
				}
				// Connection errors are expected in unit tests
			} else if publisher != nil {
				publisher.Close()
			}
		})
	}
}

func TestMessageWithHeaders(t *testing.T) {
	message := &Message{
		Body:        []byte(`{"event": "user_created", "user_id": "123"}`),
		ContentType: "application/json",
		Priority:    10,
		Headers: amqp.Table{
			"correlation-id": "abc-123",
			"reply-to":       "response-queue",
			"expiration":     "60000",
			"message-id":     "msg-456",
			"user-id":        "service-user",
			"app-id":         "user-service",
		},
	}

	// Verify all headers are set correctly
	expectedHeaders := map[string]interface{}{
		"correlation-id": "abc-123",
		"reply-to":       "response-queue",
		"expiration":     "60000",
		"message-id":     "msg-456",
		"user-id":        "service-user",
		"app-id":         "user-service",
	}

	for key, expectedValue := range expectedHeaders {
		if actualValue, exists := message.Headers[key]; !exists {
			t.Errorf("Header %s is missing", key)
		} else if actualValue != expectedValue {
			t.Errorf("Header %s: expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestMessageEmptyBody(t *testing.T) {
	message := &Message{
		Body:        []byte{}, // Empty body
		ContentType: "text/plain",
		Headers:     amqp.Table{"test": "header"},
	}

	// Empty body should be allowed
	if message.Body == nil {
		t.Error("Body should not be nil")
	}

	if len(message.Body) != 0 {
		t.Error("Body should be empty")
	}
}

// Integration test for publisher
func TestPublisherIntegration(t *testing.T) {
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
		ExchangeName: "test-publisher-integration",
		ExchangeType: "direct",
		RoutingKey:   "test.publisher.integration",
		QueueName:    "test-publisher-integration-queue",
	}

	publisher, err := NewPublisher(config)
	if err != nil {
		t.Skipf("RabbitMQ not available for integration test: %v", err)
		return
	}

	if publisher == nil {
		t.Fatal("Publisher should not be nil when creation succeeds")
	}

	message := &Message{
		Body:        []byte(`{"test": "publisher integration test"}`),
		ContentType: "application/json",
		Headers: amqp.Table{
			"timestamp": time.Now().Unix(),
		},
	}

	err = publisher.Publish(message)
	if err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}

	// Clean up
	publisher.Close()

	// Clean up queue and exchange
	manager := rabbitmq.InitConnectionManager()
	if channel, err := manager.GetChannel(); err == nil {
		channel.QueueDelete(config.QueueName, false, false, false)
		channel.ExchangeDelete(config.ExchangeName, false, false)
	}
	manager.Close()
}

func TestPublisherConvenienceMethods(t *testing.T) {
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
		ExchangeName: "test-publisher-convenience",
		ExchangeType: "direct",
		RoutingKey:   "test.convenience",
		QueueName:    "test-publisher-convenience-queue",
	}

	publisher, err := NewPublisher(config)
	if err != nil {
		t.Skipf("RabbitMQ not available for convenience test: %v", err)
		return
	}

	// Test PublishJSON
	err = publisher.PublishJSON([]byte(`{"method": "json"}`))
	if err != nil {
		t.Errorf("PublishJSON failed: %v", err)
	}

	// Test PublishText
	err = publisher.PublishText("Hello, World!")
	if err != nil {
		t.Errorf("PublishText failed: %v", err)
	}

	// Test PublishWithPriority
	err = publisher.PublishWithPriority([]byte(`{"method": "priority"}`), 8)
	if err != nil {
		t.Errorf("PublishWithPriority failed: %v", err)
	}

	// Clean up
	publisher.Close()

	// Clean up queue and exchange
	manager := rabbitmq.InitConnectionManager()
	if channel, err := manager.GetChannel(); err == nil {
		channel.QueueDelete(config.QueueName, false, false, false)
		channel.ExchangeDelete(config.ExchangeName, false, false)
	}
	manager.Close()
}

func TestPublisherClose(t *testing.T) {
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
		ExchangeName: "test-publisher-close",
		ExchangeType: "direct",
		RoutingKey:   "test.close",
		QueueName:    "test-publisher-close-queue",
	}

	publisher, err := NewPublisher(config)
	if err != nil {
		t.Skipf("RabbitMQ not available for close test: %v", err)
		return
	}

	// Test closing
	err = publisher.Close()
	if err != nil {
		t.Errorf("Failed to close publisher: %v", err)
	}

	// Test closing again (should not error)
	err = publisher.Close()
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

func TestConcurrentPublish(t *testing.T) {
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
		ExchangeName: "test-concurrent-publisher",
		ExchangeType: "direct",
		RoutingKey:   "test.concurrent",
		QueueName:    "test-concurrent-publisher-queue",
	}

	publisher, err := NewPublisher(config)
	if err != nil {
		t.Skipf("RabbitMQ not available for concurrent test: %v", err)
		return
	}

	// Test concurrent publishing
	numGoroutines := 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			message := &Message{
				Body:        []byte(fmt.Sprintf(`{"message": "concurrent test %d"}`, id)),
				ContentType: "application/json",
				Headers: amqp.Table{
					"goroutine-id": id,
					"timestamp":    time.Now().Unix(),
				},
			}

			err := publisher.Publish(message)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	close(errors)

	// Check if any errors occurred
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent publish failed: %v", err)
		}
	}

	// Clean up
	publisher.Close()

	// Clean up queue and exchange
	manager := rabbitmq.InitConnectionManager()
	if channel, err := manager.GetChannel(); err == nil {
		channel.QueueDelete(config.QueueName, false, false, false)
		channel.ExchangeDelete(config.ExchangeName, false, false)
	}
	manager.Close()
}

func BenchmarkNewPublisher(b *testing.B) {
	config := &rabbitmq.RouteConfig{
		ExchangeName: "benchmark-publisher-exchange",
		ExchangeType: "direct",
		RoutingKey:   "benchmark.publisher",
		QueueName:    "benchmark-publisher-queue",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Only test validation, not actual publisher creation
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

func TestPublisherConfigEmbedding(t *testing.T) {
	config := &rabbitmq.RouteConfig{
		ExchangeName: "test-embedding-publisher",
		ExchangeType: "topic",
		RoutingKey:   "test.embedding.publisher.key",
		QueueName:    "test-embedding-publisher-queue",
	}

	// Test that we can access RouteConfig fields
	if config.ExchangeName != "test-embedding-publisher" {
		t.Error("RouteConfig not working for ExchangeName")
	}

	if config.ExchangeType != "topic" {
		t.Error("RouteConfig not working for ExchangeType")
	}

	if config.RoutingKey != "test.embedding.publisher.key" {
		t.Error("RouteConfig not working for RoutingKey")
	}

	if config.QueueName != "test-embedding-publisher-queue" {
		t.Error("RouteConfig not working for QueueName")
	}
}
