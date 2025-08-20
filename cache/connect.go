package cache

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

var (
	redisClient    *redis.Client
	clientMutex    sync.RWMutex
	connectionOnce sync.Once
	redisConfig    *redisConnectionConfig
	configOnce     sync.Once
)

type redisConnectionConfig struct {
	addr     string
	password string
	db       int
	poolSize int
}

// getRedisConfig builds and caches Redis configuration
func getRedisConfig() *redisConnectionConfig {
	configOnce.Do(func() {
		redisConfig = &redisConnectionConfig{
			addr:     buildRedisAddr(),
			password: os.Getenv("REDIS_PASSWORD"),
			db:       0,  // default DB
			poolSize: 10, // default pool size
		}
	})
	return redisConfig
}

// buildRedisAddr builds Redis address string efficiently
func buildRedisAddr() string {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	// Set default port if not specified
	if port == "" {
		port = "6379"
	}

	// If no host specified, use localhost
	if host == "" {
		return "localhost:" + port
	}

	return host + ":" + port
}

// ConnectCache initializes Redis connection with optimized settings
func ConnectCache() error {
	var err error
	connectionOnce.Do(func() {
		config := getRedisConfig()

		redisClient = redis.NewClient(&redis.Options{
			Addr:         config.addr,
			Password:     config.password,
			DB:           config.db,
			PoolSize:     config.poolSize,
			MinIdleConns: 2,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolTimeout:  4 * time.Second,
			IdleTimeout:  5 * time.Minute,
		})

		// Test connection
		_, err = redisClient.Ping().Result()
	})

	return err
}

// IsCacheConnected checks Redis connection status efficiently
func IsCacheConnected() bool {
	clientMutex.RLock()
	client := redisClient
	clientMutex.RUnlock()

	if client == nil {
		if err := ConnectCache(); err != nil {
			return false
		}
		clientMutex.RLock()
		client = redisClient
		clientMutex.RUnlock()
	}

	// Quick ping check
	result := client.Ping()
	return result.Err() == nil && result.Val() == "PONG"
}

// getRedisClient returns Redis client with lazy initialization
func getRedisClient() (*redis.Client, error) {
	clientMutex.RLock()
	client := redisClient
	clientMutex.RUnlock()

	if client == nil {
		clientMutex.Lock()
		defer clientMutex.Unlock()

		// Double-check after acquiring write lock
		if redisClient == nil {
			if err := ConnectCache(); err != nil {
				return nil, fmt.Errorf("failed to connect to Redis: %w", err)
			}
		}
		client = redisClient
	}

	return client, nil
}

// GetRedisClient returns Redis client (public version with error handling)
func GetRedisClient() (*redis.Client, error) {
	return getRedisClient()
}

// CloseConnection closes Redis connection gracefully
func CloseConnection() error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if redisClient != nil {
		err := redisClient.Close()
		redisClient = nil
		return err
	}
	return nil
}

// ResetConnection forces a new connection (useful for testing)
func ResetConnection() error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if redisClient != nil {
		_ = redisClient.Close() // Ignore close error as we're resetting anyway
		redisClient = nil
	}

	// Reset the once to allow reconnection
	connectionOnce = sync.Once{}

	return ConnectCache()
}
