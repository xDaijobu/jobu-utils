# RabbitMQ Utils

Simple RabbitMQ client library with auto-reconnection for Go applications.

## 🚀 Quick Start

### Installation

```bash
go get github.com/xDaijobu/jobu-utils/rabbitmq
```

### Environment Setup

Set these environment variables (or create a `.env` file):

```bash
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
```

## 📤 Publisher Example

```go
package main

import (
    "log"
    
    "github.com/xDaijobu/jobu-utils/rabbitmq"
    "github.com/xDaijobu/jobu-utils/rabbitmq/publisher"
    amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    // Configure routing
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
    
    // Publish a message
    message := &publisher.Message{
        Body:        []byte(`{"user_id": "123", "event": "created"}`),
        ContentType: "application/json",
        Headers: amqp.Table{
            "source": "user-service",
        },
    }
    
    err = pub.Publish(message)
    if err != nil {
        log.Fatal("Failed to publish:", err)
    }
    
    log.Println("Message published successfully!")
}
```

### Publisher Convenience Methods

```go
// Publish JSON
err = pub.PublishJSON([]byte(`{"event": "user_created"}`))

// Publish text
err = pub.PublishText("Hello World!")

// Publish with priority
err = pub.PublishWithPriority([]byte(`{"urgent": true}`), 9)
```

## 📥 Consumer Example

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/xDaijobu/jobu-utils/rabbitmq"
    "github.com/xDaijobu/jobu-utils/rabbitmq/consumer"
    amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    // Configure routing
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
    
    // Define message handler
    handler := func(msg amqp.Delivery) error {
        log.Printf("Received message: %s", string(msg.Body))
        
        // Process message here
        // Return error to reject and requeue message
        return nil
    }
    
    // Start consuming
    ctx := context.Background()
    err = cons.Consume(ctx, handler)
    if err != nil {
        log.Fatal("Failed to consume:", err)
    }
}
```

## 🔧 Advanced Usage

### Direct Connection

```go
package main

import (
    "log"
    "sync"
    
    "github.com/xDaijobu/jobu-utils/rabbitmq"
)

func main() {
    var mutex sync.Mutex
    
    // Get channel directly
    channel, err := rabbitmq.Start(&mutex)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    
    // Setup routing manually
    config := &rabbitmq.RouteConfig{
        ExchangeName: "my-exchange",
        ExchangeType: "topic",
        RoutingKey:   "events.*",
        QueueName:    "my-queue",
    }
    
    err = rabbitmq.SetupRouting(channel, config)
    if err != nil {
        log.Fatal("Failed to setup routing:", err)
    }
    
    // Use channel for custom operations
    // ...
    
    // Clean up
    defer rabbitmq.InitConnectionManager().Close()
}
```

### Exchange Types

- **direct**: Messages routed by exact routing key match
- **topic**: Messages routed by routing key patterns (wildcards: `*` and `#`)
- **fanout**: Messages broadcasted to all bound queues
- **headers**: Messages routed by header attributes

## 🛠️ Configuration

The library uses these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RABBITMQ_HOST` | `localhost` | RabbitMQ server host |
| `RABBITMQ_PORT` | `5672` | RabbitMQ server port |
| `RABBITMQ_USER` | `guest` | RabbitMQ username |
| `RABBITMQ_PASSWORD` | `guest` | RabbitMQ password |

## 🔄 Auto-Reconnection

The library automatically handles connection failures with:

- **Exponential backoff**: Increasing delays between retry attempts
- **Maximum retries**: Configurable retry limit (default: 5)
- **Thread-safe**: Safe for concurrent access

## 🧪 Testing

Run unit tests:
```bash
go test ./rabbitmq/...
```

Run integration tests (requires RabbitMQ):
```bash
go test ./rabbitmq/... -v
```

## 📦 Dependencies

- `github.com/rabbitmq/amqp091-go` - RabbitMQ client library

## 🔗 Usage in Other Projects

```bash
go get github.com/xDaijobu/jobu-utils/rabbitmq
```

```go
import "github.com/xDaijobu/jobu-utils/rabbitmq"
import "github.com/xDaijobu/jobu-utils/rabbitmq/publisher"
import "github.com/xDaijobu/jobu-utils/rabbitmq/consumer"
```
