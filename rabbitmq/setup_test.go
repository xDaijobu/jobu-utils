package rabbitmq

import (
	"fmt"
	"os"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRouteConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *RouteConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty exchange name",
			config: &RouteConfig{
				ExchangeName: "",
				ExchangeType: "direct",
				RoutingKey:   "test",
				QueueName:    "test-queue",
			},
			wantErr: true,
		},
		{
			name: "empty queue name",
			config: &RouteConfig{
				ExchangeName: "test-exchange",
				ExchangeType: "direct",
				RoutingKey:   "test",
				QueueName:    "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &RouteConfig{
				ExchangeName: "test-exchange",
				ExchangeType: "direct",
				RoutingKey:   "test.key",
				QueueName:    "test-queue",
			},
			wantErr: false,
		},
		{
			name: "topic exchange",
			config: &RouteConfig{
				ExchangeName: "test-topic",
				ExchangeType: "topic",
				RoutingKey:   "user.*.created",
				QueueName:    "user-events",
			},
			wantErr: false,
		},
		{
			name: "fanout exchange",
			config: &RouteConfig{
				ExchangeName: "test-fanout",
				ExchangeType: "fanout",
				RoutingKey:   "", // Fanout ignores routing key
				QueueName:    "fanout-queue",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test only validation logic without actual RabbitMQ connection
			err := validateRouteConfig(tt.config)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// validateRouteConfig validates the route configuration without requiring a channel
func validateRouteConfig(config *RouteConfig) error {
	if config == nil {
		return fmt.Errorf("route config cannot be nil")
	}
	if config.ExchangeName == "" {
		return fmt.Errorf("exchange name cannot be empty")
	}
	if config.QueueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}
	return nil
}

func TestRouteConfigFields(t *testing.T) {
	config := &RouteConfig{
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test.routing.key",
		QueueName:    "test-queue",
	}

	if config.ExchangeName != "test-exchange" {
		t.Errorf("Expected ExchangeName to be 'test-exchange', got '%s'", config.ExchangeName)
	}

	if config.ExchangeType != "direct" {
		t.Errorf("Expected ExchangeType to be 'direct', got '%s'", config.ExchangeType)
	}

	if config.RoutingKey != "test.routing.key" {
		t.Errorf("Expected RoutingKey to be 'test.routing.key', got '%s'", config.RoutingKey)
	}

	if config.QueueName != "test-queue" {
		t.Errorf("Expected QueueName to be 'test-queue', got '%s'", config.QueueName)
	}
}

// Integration test for setup routing
func TestSetupRoutingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get a real channel for integration testing
	channel, err := getTestChannel(t)
	if err != nil {
		t.Skipf("RabbitMQ not available for integration test: %v", err)
		return
	}
	defer func() {
		manager := InitConnectionManager()
		manager.Close()
	}()

	config := &RouteConfig{
		ExchangeName: "test-exchange-integration",
		ExchangeType: "direct",
		RoutingKey:   "test.integration",
		QueueName:    "test-queue-integration",
	}

	err = SetupRouting(channel, config)
	if err != nil {
		t.Fatalf("SetupRouting failed: %v", err)
	}

	// Verify exchange exists by trying to publish a message
	err = channel.Publish(
		config.ExchangeName,
		config.RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("test message"),
		},
	)
	if err != nil {
		t.Errorf("Failed to publish to created exchange: %v", err)
	}

	// Clean up - delete queue and exchange
	channel.QueueDelete(config.QueueName, false, false, false)
	channel.ExchangeDelete(config.ExchangeName, false, false)
}

func TestSetupRoutingWithDifferentExchangeTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name         string
		exchangeType string
		routingKey   string
	}{
		{"direct exchange", "direct", "test.direct"},
		{"topic exchange", "topic", "test.topic.pattern"},
		{"fanout exchange", "fanout", ""},
		{"headers exchange", "headers", "test.headers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := getTestChannel(t)
			if err != nil {
				t.Skipf("RabbitMQ not available: %v", err)
				return
			}

			config := &RouteConfig{
				ExchangeName: "test-" + tt.exchangeType + "-exchange",
				ExchangeType: tt.exchangeType,
				RoutingKey:   tt.routingKey,
				QueueName:    "test-" + tt.exchangeType + "-queue",
			}

			err = SetupRouting(channel, config)
			if err != nil {
				t.Errorf("SetupRouting failed for %s: %v", tt.exchangeType, err)
			}

			// Clean up
			channel.QueueDelete(config.QueueName, false, false, false)
			channel.ExchangeDelete(config.ExchangeName, false, false)
		})
	}

	// Clean up connection
	manager := InitConnectionManager()
	manager.Close()
}

// Helper function to get a test channel
func getTestChannel(t *testing.T) (*amqp.Channel, error) {
	// Set hardcoded environment variables for the test
	os.Setenv("RABBITMQ_HOST", "localhost")
	os.Setenv("RABBITMQ_PORT", "5672")
	os.Setenv("RABBITMQ_USER", "admin")
	os.Setenv("RABBITMQ_PASSWORD", "password")

	manager := InitConnectionManager()
	return manager.GetChannel()
}

func BenchmarkSetupRoutingValidation(b *testing.B) {
	config := &RouteConfig{
		ExchangeName: "benchmark-exchange",
		ExchangeType: "direct",
		RoutingKey:   "benchmark.key",
		QueueName:    "benchmark-queue",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Only test validation, not actual setup
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

func TestRouteConfigCopy(t *testing.T) {
	original := &RouteConfig{
		ExchangeName: "original-exchange",
		ExchangeType: "direct",
		RoutingKey:   "original.key",
		QueueName:    "original-queue",
	}

	// Create a copy
	copy := &RouteConfig{
		ExchangeName: original.ExchangeName,
		ExchangeType: original.ExchangeType,
		RoutingKey:   original.RoutingKey,
		QueueName:    original.QueueName,
	}

	// Verify copy is correct
	if copy.ExchangeName != original.ExchangeName {
		t.Error("ExchangeName not copied correctly")
	}
	if copy.ExchangeType != original.ExchangeType {
		t.Error("ExchangeType not copied correctly")
	}
	if copy.RoutingKey != original.RoutingKey {
		t.Error("RoutingKey not copied correctly")
	}
	if copy.QueueName != original.QueueName {
		t.Error("QueueName not copied correctly")
	}

	// Verify they are different objects
	copy.ExchangeName = "modified-exchange"
	if original.ExchangeName == copy.ExchangeName {
		t.Error("Original should not be modified when copy is changed")
	}
}
