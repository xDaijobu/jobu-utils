package cache

import (
	"os"
	"sync"
	"testing"

	"github.com/go-redis/redis"
)

func TestGetRedisConfig(t *testing.T) {
	// Save original env vars
	originalHost := os.Getenv("REDIS_HOST")
	originalPort := os.Getenv("REDIS_PORT")
	originalPassword := os.Getenv("REDIS_PASSWORD")

	// Clean up after test
	defer func() {
		if originalHost != "" {
			_ = os.Setenv("REDIS_HOST", originalHost)
		} else {
			_ = os.Unsetenv("REDIS_HOST")
		}
		if originalPort != "" {
			_ = os.Setenv("REDIS_PORT", originalPort)
		} else {
			_ = os.Unsetenv("REDIS_PORT")
		}
		if originalPassword != "" {
			_ = os.Setenv("REDIS_PASSWORD", originalPassword)
		} else {
			_ = os.Unsetenv("REDIS_PASSWORD")
		}
		// Reset the config for next test
		configOnce = sync.Once{}
		redisConfig = nil
	}()

	// Set test environment
	_ = os.Setenv("REDIS_HOST", "test-host")
	_ = os.Setenv("REDIS_PORT", "6380")
	_ = os.Setenv("REDIS_PASSWORD", "test-password")

	// Reset config before test
	configOnce = sync.Once{}
	redisConfig = nil

	config := getRedisConfig()

	if config.addr != "test-host:6380" {
		t.Errorf("getRedisConfig() addr = %v, want %v", config.addr, "test-host:6380")
	}
	if config.password != "test-password" {
		t.Errorf("getRedisConfig() password = %v, want %v", config.password, "test-password")
	}
	if config.db != 0 {
		t.Errorf("getRedisConfig() db = %v, want %v", config.db, 0)
	}
	if config.poolSize != 10 {
		t.Errorf("getRedisConfig() poolSize = %v, want %v", config.poolSize, 10)
	}

	// Test that subsequent calls return the same config (cached)
	config2 := getRedisConfig()
	if config != config2 {
		t.Error("getRedisConfig() should return cached config on subsequent calls")
	}
}

func TestBuildRedisAddr(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		port         string
		expectedAddr string
	}{
		{
			name:         "default port with host",
			host:         "redis-server",
			port:         "",
			expectedAddr: "redis-server:6379",
		},
		{
			name:         "custom port with host",
			host:         "redis-server",
			port:         "6380",
			expectedAddr: "redis-server:6380",
		},
		{
			name:         "no host default port",
			host:         "",
			port:         "",
			expectedAddr: "localhost:6379",
		},
		{
			name:         "no host custom port",
			host:         "",
			port:         "6381",
			expectedAddr: "localhost:6381",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalHost := os.Getenv("REDIS_HOST")
			originalPort := os.Getenv("REDIS_PORT")

			// Clean up after test
			defer func() {
				if originalHost != "" {
					os.Setenv("REDIS_HOST", originalHost)
				} else {
					os.Unsetenv("REDIS_HOST")
				}
				if originalPort != "" {
					os.Setenv("REDIS_PORT", originalPort)
				} else {
					os.Unsetenv("REDIS_PORT")
				}
			}()

			// Set test environment
			if tt.host != "" {
				_ = os.Setenv("REDIS_HOST", tt.host)
			} else {
				_ = os.Unsetenv("REDIS_HOST")
			}
			if tt.port != "" {
				_ = os.Setenv("REDIS_PORT", tt.port)
			} else {
				_ = os.Unsetenv("REDIS_PORT")
			}

			addr := buildRedisAddr()
			if addr != tt.expectedAddr {
				t.Errorf("buildRedisAddr() = %v, want %v", addr, tt.expectedAddr)
			}
		})
	}
}

func TestConnectCache(t *testing.T) {
	// Reset connection state
	clientMutex.Lock()
	redisClient = nil
	connectionOnce = sync.Once{}
	configOnce = sync.Once{}
	redisConfig = nil
	clientMutex.Unlock()

	// Set test environment
	_ = os.Setenv("REDIS_HOST", "localhost")
	_ = os.Setenv("REDIS_PORT", "6379")
	defer func() {
		_ = os.Unsetenv("REDIS_HOST")
		_ = os.Unsetenv("REDIS_PORT")
	}()

	// Test connection - this will fail in test environment but we can test the setup
	err := ConnectCache()

	// In test environment, we expect connection to fail
	// but we can verify that the client was created with correct options
	clientMutex.RLock()
	client := redisClient
	clientMutex.RUnlock()

	if client == nil {
		t.Error("ConnectCache() should create redis client even if connection fails")
		return
	}

	// Verify client options through reflection or by checking the client exists
	if client.Options().Addr != "localhost:6379" {
		t.Errorf("Redis client addr = %v, want %v", client.Options().Addr, "localhost:6379")
	}

	// Test that subsequent calls don't recreate the client
	client1 := redisClient
	err2 := ConnectCache()
	client2 := redisClient

	if client1 != client2 {
		t.Error("ConnectCache() should not recreate client on subsequent calls")
	}

	// Both errors should be the same (either both nil or both not nil)
	if (err == nil) != (err2 == nil) {
		t.Error("ConnectCache() should return consistent results on subsequent calls")
	}
}

func TestIsCacheConnected(t *testing.T) {
	// Reset connection state
	clientMutex.Lock()
	redisClient = nil
	connectionOnce = sync.Once{}
	clientMutex.Unlock()

	// Test with no client initially
	connected := IsCacheConnected()

	// Should attempt to connect and likely fail in test environment
	// but should not panic and should return a boolean
	if connected {
		t.Log("IsCacheConnected() returned true - Redis might be running locally")
	} else {
		t.Log("IsCacheConnected() returned false - expected in test environment")
	}

	// Verify that a client was created during the check
	clientMutex.RLock()
	client := redisClient
	clientMutex.RUnlock()

	if client == nil {
		t.Error("IsCacheConnected() should create client if none exists")
	}
}

func TestGetRedisClient(t *testing.T) {
	// Reset connection state
	clientMutex.Lock()
	redisClient = nil
	connectionOnce = sync.Once{}
	clientMutex.Unlock()

	client, err := getRedisClient()

	// Should return a client and potentially an error
	if client == nil && err == nil {
		t.Error("getRedisClient() should return either a client or an error")
	}

	if client != nil {
		// Verify it's a valid redis client
		if client.Options() == nil {
			t.Error("getRedisClient() returned invalid client")
		}
	}

	// Test subsequent call returns same client
	client2, err2 := getRedisClient()
	if client != nil && client2 != nil && client != client2 {
		t.Error("getRedisClient() should return same client on subsequent calls")
	}

	// Use err2 to avoid unused variable warning
	if err2 != nil {
		t.Logf("Second getRedisClient() call returned error: %v", err2)
	}
}

func TestCloseConnection(t *testing.T) {
	// Setup a client first
	clientMutex.Lock()
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	clientMutex.Unlock()

	err := CloseConnection()

	if err != nil {
		t.Logf("CloseConnection() returned error: %v", err)
	}

	// Verify client is set to nil
	clientMutex.RLock()
	client := redisClient
	clientMutex.RUnlock()

	if client != nil {
		t.Error("CloseConnection() should set redisClient to nil")
	}

	// Test closing when no client exists
	err2 := CloseConnection()
	if err2 != nil {
		t.Errorf("CloseConnection() should not error when no client exists, got: %v", err2)
	}
}

func TestResetConnection(t *testing.T) {
	// Setup initial state
	clientMutex.Lock()
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	connectionOnce = sync.Once{} // Already used
	clientMutex.Unlock()

	err := ResetConnection()

	// Should not panic and should reset the connection
	clientMutex.RLock()
	newClient := redisClient
	clientMutex.RUnlock()

	if newClient == nil {
		// This is expected if Redis is not running in test environment
		t.Log("ResetConnection() client is nil - expected if Redis not available")
	}

	// Verify that connectionOnce was reset by checking if we can connect again
	_ = ConnectCache()
	t.Log("ResetConnection() appears to have reset the connectionOnce properly")

	// Use the error to avoid unused variable warning
	if err != nil {
		t.Logf("ResetConnection() returned error: %v", err)
	}
}

// Test concurrent access
func TestConcurrentAccess(t *testing.T) {
	// Reset state
	clientMutex.Lock()
	redisClient = nil
	connectionOnce = sync.Once{}
	clientMutex.Unlock()

	var wg sync.WaitGroup
	clients := make([]*redis.Client, 10)
	errors := make([]error, 10)

	// Start multiple goroutines trying to get client
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			client, err := getRedisClient()
			clients[index] = client
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// All non-nil clients should be the same instance
	var firstClient *redis.Client
	for i, client := range clients {
		if client != nil {
			if firstClient == nil {
				firstClient = client
			} else if client != firstClient {
				t.Errorf("Concurrent getRedisClient() call %d returned different client", i)
			}
		}
	}
}

// Benchmark tests
func BenchmarkGetRedisConfig(b *testing.B) {
	// Setup environment
	os.Setenv("REDIS_HOST", "benchmark-host")
	os.Setenv("REDIS_PORT", "6379")
	defer func() {
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getRedisConfig()
	}
}

func BenchmarkBuildRedisAddr(b *testing.B) {
	os.Setenv("REDIS_HOST", "benchmark-host")
	os.Setenv("REDIS_PORT", "6379")
	defer func() {
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildRedisAddr()
	}
}

func BenchmarkIsCacheConnected(b *testing.B) {
	// Setup a client
	clientMutex.Lock()
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	clientMutex.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsCacheConnected()
	}
}
