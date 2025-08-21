package rabbitmq

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ConnectionManager manages RabbitMQ connection with auto-reconnect functionality
type ConnectionManager struct {
	conn           *amqp.Connection
	channel        *amqp.Channel
	connString     string
	mutex          sync.RWMutex
	isReconnecting bool
	reconnectChan  chan bool

	// Reconnect configuration
	MaxRetries    int
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
	BackoffFactor float64
}

var (
	connManager *ConnectionManager
	once        sync.Once
)

// InitConnectionManager initializes the global connection manager with default settings
func InitConnectionManager() *ConnectionManager {
	once.Do(func() {
		// Load .env file - try current directory first, then parent directory
		if err := godotenv.Load(); err != nil {
			// Try loading from parent directory (for tests run from subdirectories)
			if err2 := godotenv.Load("../.env"); err2 != nil {
				log.Printf("Warning: Error loading .env file: %v", err)
			}
		}

		connManager = &ConnectionManager{
			MaxRetries:    5,
			RetryDelay:    1 * time.Second,
			MaxRetryDelay: 30 * time.Second,
			BackoffFactor: 2.0,
			reconnectChan: make(chan bool, 1),
		}

		// Build connection string
		if os.Getenv("RABBITMQ_PORT") == "" {
			os.Setenv("RABBITMQ_PORT", "5672")
		}

		connManager.connString = fmt.Sprintf("amqp://%s:%s@%s:%s",
			os.Getenv("RABBITMQ_USER"),
			os.Getenv("RABBITMQ_PASSWORD"),
			os.Getenv("RABBITMQ_HOST"),
			os.Getenv("RABBITMQ_PORT"),
		)

		// Debug log to see what connection string is being used
		log.Printf("RabbitMQ connection string: amqp://%s:***@%s:%s",
			os.Getenv("RABBITMQ_USER"),
			os.Getenv("RABBITMQ_HOST"),
			os.Getenv("RABBITMQ_PORT"))
	})
	return connManager
}

// Start establishes connection with auto-reconnect capability
func Start(m *sync.Mutex) (*amqp.Channel, error) {
	manager := InitConnectionManager()
	return manager.GetChannel()
}

// GetChannel returns a working channel, reconnecting if necessary
func (cm *ConnectionManager) GetChannel() (*amqp.Channel, error) {
	cm.mutex.RLock()
	if cm.conn != nil && !cm.conn.IsClosed() && cm.channel != nil {
		cm.mutex.RUnlock()
		return cm.channel, nil
	}
	cm.mutex.RUnlock()

	// Need to establish/re-establish connection
	return cm.connect()
}

// connect establishes connection with retry logic
func (cm *ConnectionManager) connect() (*amqp.Channel, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Double-check after acquiring lock
	if cm.conn != nil && !cm.conn.IsClosed() && cm.channel != nil {
		return cm.channel, nil
	}

	log.Printf("Establishing connection to RabbitMQ...")

	var err error
	retryDelay := cm.RetryDelay

	for attempt := 0; attempt <= cm.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Reconnection attempt %d/%d after %v", attempt, cm.MaxRetries, retryDelay)
			time.Sleep(retryDelay)

			// Exponential backoff
			retryDelay = time.Duration(float64(retryDelay) * cm.BackoffFactor)
			if retryDelay > cm.MaxRetryDelay {
				retryDelay = cm.MaxRetryDelay
			}
		}

		// Create connection
		cm.conn, err = amqp.Dial(cm.connString)
		if err != nil {
			log.Printf("ERROR: Failed to connect to RabbitMQ (attempt %d): %v", attempt+1, err)
			cm.conn = nil
			continue
		}

		// Create channel
		cm.channel, err = cm.conn.Channel()
		if err != nil {
			log.Printf("ERROR: Failed to create channel (attempt %d): %v", attempt+1, err)
			cm.conn.Close()
			cm.conn = nil
			cm.channel = nil
			continue
		}

		// Set up connection monitoring
		cm.setupConnectionMonitoring()

		log.Printf("Successfully connected to RabbitMQ (attempt %d)", attempt+1)
		return cm.channel, nil
	}

	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", cm.MaxRetries+1, err)
}

// setupConnectionMonitoring sets up goroutines to monitor connection and channel health
func (cm *ConnectionManager) setupConnectionMonitoring() {
	// Monitor connection
	go func() {
		errChan := cm.conn.NotifyClose(make(chan *amqp.Error))
		for connErr := range errChan {
			if connErr != nil {
				log.Printf("ERROR: RabbitMQ connection lost: %v", connErr)
				cm.handleConnectionLoss()
			}
		}
	}()

	// Monitor channel
	go func() {
		errChan := cm.channel.NotifyClose(make(chan *amqp.Error))
		for chanErr := range errChan {
			if chanErr != nil {
				log.Printf("ERROR: RabbitMQ channel lost: %v", chanErr)
				cm.handleConnectionLoss()
			}
		}
	}()
}

// handleConnectionLoss handles connection/channel loss and triggers reconnection
func (cm *ConnectionManager) handleConnectionLoss() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.isReconnecting {
		return // Already reconnecting
	}

	cm.isReconnecting = true
	cm.conn = nil
	cm.channel = nil

	// Signal connection loss
	select {
	case cm.reconnectChan <- true:
	default:
	}

	go cm.autoReconnect()
}

// autoReconnect attempts to reconnect in the background
func (cm *ConnectionManager) autoReconnect() {
	defer func() {
		cm.mutex.Lock()
		cm.isReconnecting = false
		cm.mutex.Unlock()
	}()

	log.Printf("Starting auto-reconnection process...")

	retryDelay := cm.RetryDelay
	for attempt := 0; attempt <= cm.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Auto-reconnection attempt %d/%d after %v", attempt, cm.MaxRetries, retryDelay)
			time.Sleep(retryDelay)

			// Exponential backoff
			retryDelay = time.Duration(float64(retryDelay) * cm.BackoffFactor)
			if retryDelay > cm.MaxRetryDelay {
				retryDelay = cm.MaxRetryDelay
			}
		}

		// Try to establish connection
		conn, err := amqp.Dial(cm.connString)
		if err != nil {
			log.Printf("ERROR: Auto-reconnection failed (attempt %d): %v", attempt+1, err)
			continue
		}

		// Try to create channel
		channel, err := conn.Channel()
		if err != nil {
			log.Printf("ERROR: Failed to create channel during auto-reconnection (attempt %d): %v", attempt+1, err)
			conn.Close()
			continue
		}

		// Success - update connection manager
		cm.mutex.Lock()
		if cm.conn != nil && !cm.conn.IsClosed() {
			cm.conn.Close() // Close old connection if exists
		}
		cm.conn = conn
		cm.channel = channel
		cm.mutex.Unlock()

		// Set up monitoring for new connection
		cm.setupConnectionMonitoring()

		log.Printf("Auto-reconnection successful (attempt %d)", attempt+1)
		return
	}

	log.Printf("ERROR: Auto-reconnection failed after %d attempts", cm.MaxRetries+1)
}

// IsConnected returns true if connection and channel are healthy
func (cm *ConnectionManager) IsConnected() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.conn != nil && !cm.conn.IsClosed() && cm.channel != nil
}

// Close gracefully closes the connection
func (cm *ConnectionManager) Close() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var err error
	if cm.channel != nil {
		if closeErr := cm.channel.Close(); closeErr != nil {
			err = closeErr
		}
		cm.channel = nil
	}

	if cm.conn != nil && !cm.conn.IsClosed() {
		if closeErr := cm.conn.Close(); closeErr != nil {
			err = closeErr
		}
		cm.conn = nil
	}

	log.Printf("RabbitMQ connection closed")
	return err
}

// WaitForReconnection blocks until reconnection is complete (useful for testing)
func (cm *ConnectionManager) WaitForReconnection(timeout time.Duration) bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ticker.C:
			if cm.IsConnected() && !cm.isReconnecting {
				return true
			}
		case <-timeoutTimer.C:
			return false
		}
	}
}
