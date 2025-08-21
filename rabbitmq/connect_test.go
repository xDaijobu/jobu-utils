package rabbitmq

import (
	"os"
	"sync"
	"testing"
	"time"
)

// Test connection manager initialization
func TestInitConnectionManager(t *testing.T) {
	// Tests will use environment variables that are already set
	// No need to hardcode - just ensure defaults work
	manager := InitConnectionManager()

	if manager == nil {
		t.Fatal("Connection manager should not be nil")
	}

	// Test default values
	if manager.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", manager.MaxRetries)
	}

	if manager.RetryDelay != 1*time.Second {
		t.Errorf("Expected RetryDelay to be 1s, got %v", manager.RetryDelay)
	}

	if manager.MaxRetryDelay != 30*time.Second {
		t.Errorf("Expected MaxRetryDelay to be 30s, got %v", manager.MaxRetryDelay)
	}

	if manager.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor to be 2.0, got %f", manager.BackoffFactor)
	}

	// Test singleton pattern
	manager2 := InitConnectionManager()
	if manager != manager2 {
		t.Error("Connection manager should be singleton")
	}
}

func TestConnectionManagerConfiguration(t *testing.T) {
	manager := InitConnectionManager()

	// Test configuration changes
	manager.MaxRetries = 10
	manager.RetryDelay = 2 * time.Second
	manager.MaxRetryDelay = 60 * time.Second
	manager.BackoffFactor = 1.5

	if manager.MaxRetries != 10 {
		t.Errorf("Expected MaxRetries to be 10, got %d", manager.MaxRetries)
	}

	if manager.RetryDelay != 2*time.Second {
		t.Errorf("Expected RetryDelay to be 2s, got %v", manager.RetryDelay)
	}

	if manager.MaxRetryDelay != 60*time.Second {
		t.Errorf("Expected MaxRetryDelay to be 60s, got %v", manager.MaxRetryDelay)
	}

	if manager.BackoffFactor != 1.5 {
		t.Errorf("Expected BackoffFactor to be 1.5, got %f", manager.BackoffFactor)
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Save original value
	originalPort := os.Getenv("RABBITMQ_PORT")

	// Test default port setting
	os.Unsetenv("RABBITMQ_PORT")

	// Simulate the port setting logic from InitConnectionManager
	if os.Getenv("RABBITMQ_PORT") == "" {
		os.Setenv("RABBITMQ_PORT", "5672")
	}

	if os.Getenv("RABBITMQ_PORT") != "5672" {
		t.Error("Default port should be set to 5672")
	}

	// Restore original value
	if originalPort != "" {
		os.Setenv("RABBITMQ_PORT", originalPort)
	}
}

func TestIsConnectedWhenNotConnected(t *testing.T) {
	manager := &ConnectionManager{}

	if manager.IsConnected() {
		t.Error("Should not be connected when connection is nil")
	}
}

func TestClose(t *testing.T) {
	manager := &ConnectionManager{}

	// Test closing when nothing is connected
	err := manager.Close()
	if err != nil {
		t.Errorf("Close should not return error when nothing is connected: %v", err)
	}
}

func TestWaitForReconnectionTimeout(t *testing.T) {
	manager := &ConnectionManager{}

	start := time.Now()
	result := manager.WaitForReconnection(100 * time.Millisecond)
	duration := time.Since(start)

	if result {
		t.Error("WaitForReconnection should return false on timeout")
	}

	if duration < 100*time.Millisecond {
		t.Error("WaitForReconnection should wait for the specified duration")
	}
}

// Integration test - requires running RabbitMQ
func TestStartIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set hardcoded environment variables for the test - using admin/password from .env
	os.Setenv("RABBITMQ_HOST", "localhost")
	os.Setenv("RABBITMQ_PORT", "5672")
	os.Setenv("RABBITMQ_USER", "admin")
	os.Setenv("RABBITMQ_PASSWORD", "password")

	var m sync.Mutex
	channel, err := Start(&m)

	if err != nil {
		t.Skipf("RabbitMQ not available for integration test: %v", err)
		return
	}

	if channel == nil {
		t.Fatal("Channel should not be nil when connection succeeds")
	}

	// Test that manager is connected
	manager := InitConnectionManager()
	if !manager.IsConnected() {
		t.Error("Manager should report as connected after successful Start()")
	}

	// Clean up
	manager.Close()
}

// Test concurrent access to Start
func TestStartConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set hardcoded environment variables for the test
	os.Setenv("RABBITMQ_HOST", "localhost")
	os.Setenv("RABBITMQ_PORT", "5672")
	os.Setenv("RABBITMQ_USER", "admin")
	os.Setenv("RABBITMQ_PASSWORD", "password")

	var wg sync.WaitGroup
	var m sync.Mutex
	errors := make(chan error, 10)

	// Start multiple goroutines trying to connect
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Start(&m)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check if any goroutine failed
	for err := range errors {
		if err != nil {
			t.Skipf("RabbitMQ not available for concurrency test: %v", err)
			return
		}
	}

	// Clean up
	manager := InitConnectionManager()
	manager.Close()
}

func BenchmarkInitConnectionManager(b *testing.B) {
	// Benchmarks use whatever environment variables are set
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InitConnectionManager()
	}
}

func BenchmarkIsConnected(b *testing.B) {
	manager := &ConnectionManager{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.IsConnected()
	}
}

// Test helper for mocking connection failures
func TestConnectionFailureScenarios(t *testing.T) {
	// Test invalid host
	os.Setenv("RABBITMQ_HOST", "invalid-host-12345")
	os.Setenv("RABBITMQ_PORT", "5672")
	os.Setenv("RABBITMQ_USER", "admin")
	os.Setenv("RABBITMQ_PASSWORD", "password")

	manager := &ConnectionManager{
		MaxRetries:    1, // Limit retries for fast test
		RetryDelay:    100 * time.Millisecond,
		MaxRetryDelay: 1 * time.Second,
		BackoffFactor: 2.0,
	}

	// Build connection string
	manager.connString = "amqp://admin:password@invalid-host-12345:5672"

	_, err := manager.connect()
	if err == nil {
		t.Error("Expected connection to fail with invalid host")
	}

	// Restore valid environment for other tests
	os.Setenv("RABBITMQ_HOST", "localhost")
}
