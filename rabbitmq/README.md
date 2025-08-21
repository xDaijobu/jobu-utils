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
    "sync"
    
    "jobu-utils/rabbitmq"
    "github.com/streadway/amqp"
)

func main() {
    var mutex sync.Mutex
    channel, err := rabbitmq.Start(&mutex)
    if err != nil {
        log.Fatal(err)
    }
    defer rabbitmq.InitConnectionManager().Close()
    
    // Publish a message
    err = channel.Publish(
        "my-exchange",   // exchange
        "my.key",        // routing key
        false,           // mandatory
        false,           // immediate
        amqp.Publishing{
            ContentType: "text/plain",
            Body:        []byte("Hello, RabbitMQ!"),
        },
    )
    
    if err != nil {
        log.Printf("Failed to publish: %v", err)
    } else {
        log.Println("Message published successfully!")
    }
}
```

## 📝 Example: Consumer

```go
package main

import (
    "log"
    "sync"
    
    "jobu-utils/rabbitmq"
)

func main() {
    var mutex sync.Mutex
    channel, err := rabbitmq.Start(&mutex)
    if err != nil {
        log.Fatal(err)
    }
    defer rabbitmq.InitConnectionManager().Close()
    
    // Consume messages
    msgs, err := channel.Consume(
        "my-queue", // queue
        "",         // consumer
        true,       // auto-ack
        false,      // exclusive
        false,      // no-local
        false,      // no-wait
        nil,        // args
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Process messages
    for msg := range msgs {
        log.Printf("Received: %s", msg.Body)
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

### Functions

- `Start(mutex *sync.Mutex) (*amqp.Channel, error)` - Connect and get channel
- `InitConnectionManager() *ConnectionManager` - Get connection manager instance
- `SetupRouting(channel *amqp.Channel, config *RouteConfig) error` - Setup exchanges and queues

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
```

---

Made with ❤️ for simple RabbitMQ integration
