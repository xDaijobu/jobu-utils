# RabbitMQ Module

Simple RabbitMQ client with auto-reconnection for Go applications.

## 🚀 Quick Start

### 1. Setup Environment

Create a `.env` file in your project root:

```env
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=admin
RABBITMQ_PASSWORD=password
```

### 2. Basic Usage

```go
package main

import (
    "log"
    "sync"
    
    "jobu-utils/rabbitmq"
)

func main() {
    var mutex sync.Mutex
    
    // Connect to RabbitMQ
    channel, err := rabbitmq.Start(&mutex)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    
    // Use the channel for publishing/consuming
    log.Println("Connected to RabbitMQ!")
    
    // Clean up when done
    defer rabbitmq.InitConnectionManager().Close()
}
```

## 📖 Features

- ✅ **Auto-reconnection** - Automatically reconnects when connection is lost
- ✅ **Environment variables** - Loads config from `.env` file
- ✅ **Thread-safe** - Safe for concurrent use
- ✅ **Connection pooling** - Reuses connections efficiently
- ✅ **Error handling** - Comprehensive error logging and retry logic

## 🔧 Configuration

All configuration is done through environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RABBITMQ_HOST` | `localhost` | RabbitMQ server hostname |
| `RABBITMQ_PORT` | `5672` | RabbitMQ server port |
| `RABBITMQ_USER` | - | Username for authentication |
| `RABBITMQ_PASSWORD` | - | Password for authentication |

## 🏗️ Advanced Usage

### Connection Manager

```go
// Get the global connection manager
manager := rabbitmq.InitConnectionManager()

// Configure retry settings
manager.MaxRetries = 10
manager.RetryDelay = 2 * time.Second
manager.MaxRetryDelay = 60 * time.Second

// Check connection status
if manager.IsConnected() {
    log.Println("RabbitMQ is connected")
}

// Get a channel
channel, err := manager.GetChannel()
if err != nil {
    log.Printf("Failed to get channel: %v", err)
}
```

### Setup Routing (Exchanges & Queues)

```go
import "jobu-utils/rabbitmq"

// Configure exchange and queue
config := &rabbitmq.RouteConfig{
    ExchangeName: "my-exchange",
    ExchangeType: "direct",        // direct, topic, fanout, headers
    RoutingKey:   "my.routing.key",
    QueueName:    "my-queue",
}

// Setup the routing
err := rabbitmq.SetupRouting(channel, config)
if err != nil {
    log.Printf("Failed to setup routing: %v", err)
}
```

## 🐳 Docker Setup

Run RabbitMQ with Docker:

```bash
docker run -d \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=admin \
  -e RABBITMQ_DEFAULT_PASS=password \
  rabbitmq:3-management-alpine
```

Access management UI at: http://localhost:15672

## 🧪 Testing

Run unit tests:
```bash
go test -v -short
```

Run integration tests (requires running RabbitMQ):
```bash
go test -v
```

## 🔍 Connection Monitoring

The module automatically monitors connections and will:

- **Detect connection loss** and log errors
- **Attempt auto-reconnection** with exponential backoff
- **Restore functionality** once reconnected
- **Thread-safe operations** during reconnection

## 📝 Example: Publisher

```go
package main

import (
    "log"
    
    "jobu-utils/rabbitmq"
    "jobu-utils/rabbitmq/publisher"
)

func main() {
    // Configure exchange and queue
    config := &rabbitmq.RouteConfig{
        ExchangeName: "user-events",
        ExchangeType: "direct",
        RoutingKey:   "user.created",
        QueueName:    "user-notifications",
    }
    
    // Create publisher
    pub, err := publisher.NewPublisher(config)
    if err != nil {
        log.Fatal("Failed to create publisher:", err)
    }
    defer pub.Close()
    
    // Publish JSON message
    jsonData := []byte(`{"user_id": 123, "name": "John Doe", "email": "john@example.com"}`)
    err = pub.PublishJSON(jsonData)
    if err != nil {
        log.Printf("Failed to publish: %v", err)
    } else {
        log.Println("Message published successfully!")
    }
    
    // Publish text message
    err = pub.PublishText("Hello, RabbitMQ!")
    if err != nil {
        log.Printf("Failed to publish text: %v", err)
    }
    
    // Publish with priority
    err = pub.PublishWithPriority(jsonData, 5)
    if err != nil {
        log.Printf("Failed to publish with priority: %v", err)
    }
}
```

## 📝 Example: Consumer

```go
package main

import (
    "context"
    "log"
    "time"
    
    "jobu-utils/rabbitmq"
    "jobu-utils/rabbitmq/consumer"
    "github.com/streadway/amqp"
)

func main() {
    // Configure exchange and queue
    config := &rabbitmq.RouteConfig{
        ExchangeName: "user-events",
        ExchangeType: "direct",
        RoutingKey:   "user.created",
        QueueName:    "user-notifications",
    }
    
    // Create consumer
    cons, err := consumer.NewConsumer(config)
    if err != nil {
        log.Fatal("Failed to create consumer:", err)
    }
    defer cons.Close()
    
    // Create message handler
    handler := func(msg amqp.Delivery) error {
        log.Printf("Received message: %s", string(msg.Body))
        
        // Process your message here
        // Return error if processing fails (message will be requeued)
        // Return nil if processing succeeds (message will be acked)
        
        time.Sleep(100 * time.Millisecond) // Simulate processing
        return nil
    }
    
    // Start consuming with context for graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    log.Println("Starting consumer...")
    err = cons.Consume(ctx, handler)
    if err != nil {
        log.Printf("Consumer error: %v", err)
    }
}
```

## ⚡ Tips

1. **Always use the same mutex** when calling `Start()` from multiple goroutines
2. **Check connection status** with `manager.IsConnected()` before critical operations
3. **Handle errors gracefully** - the module will auto-reconnect on failures
4. **Use environment variables** for different environments (dev, staging, prod)
5. **Monitor logs** for connection status and error details

## 🐛 Troubleshooting

### Connection Issues

```
ERROR: Failed to connect to RabbitMQ: username or password not allowed
```
**Solution:** Check your `.env` file has correct `RABBITMQ_USER` and `RABBITMQ_PASSWORD`

```
ERROR: Failed to connect to RabbitMQ: dial tcp connect: connection refused
```
**Solution:** Make sure RabbitMQ is running on the specified host and port

### Environment Variables Not Loading

```
Warning: Error loading .env file: no such file or directory
```
**Solution:** Make sure `.env` file exists in your project root directory

## 📚 API Reference

### Core Functions

- `Start(mutex *sync.Mutex) (*amqp.Channel, error)` - Connect and get channel
- `InitConnectionManager() *ConnectionManager` - Get connection manager instance
- `SetupRouting(channel *amqp.Channel, config *RouteConfig) error` - Setup exchanges and queues

### Publisher API

```go
// Create new publisher
func NewPublisher(config *RouteConfig) (*Publisher, error)

// Publishing methods
func (p *Publisher) Publish(msg *Message) error
func (p *Publisher) PublishWithRoutingKey(msg *Message, routingKey string) error
func (p *Publisher) PublishJSON(data []byte) error
func (p *Publisher) PublishText(text string) error
func (p *Publisher) PublishWithPriority(data []byte, priority uint8) error
func (p *Publisher) Close() error
```

### Consumer API

```go
// Create new consumer
func NewConsumer(config *RouteConfig) (*Consumer, error)

// Consuming methods
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error
func (c *Consumer) Close() error

// Message handler function type
type MessageHandler func(msg amqp.Delivery) error
```

### Types

```go
type ConnectionManager struct {
    MaxRetries    int           // Maximum retry attempts (default: 5)
    RetryDelay    time.Duration // Initial retry delay (default: 1s)
    MaxRetryDelay time.Duration // Maximum retry delay (default: 30s)
    BackoffFactor float64       // Backoff multiplier (default: 2.0)
}

type RouteConfig struct {
    ExchangeName string // Exchange name
    ExchangeType string // Exchange type: direct, topic, fanout, headers
    RoutingKey   string // Routing key pattern
    QueueName    string // Queue name
}

type Message struct {
    Body        []byte     // Message payload
    ContentType string     // MIME content type
    Priority    uint8      // Message priority (0-255)
    Headers     amqp.Table // Custom headers
}
```

---

Made with ❤️ for simple RabbitMQ integration
