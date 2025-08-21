# RabbitMQ Module

Simple RabbitMQ client with auto-reconnection for Go applications.

## 🚀 Quick Start

### 1. Setup Environment

Create a `.env` file in your project root:

```env
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
```

### 2. Basic Usage

```go
package main

import (
    "log"
    "sync"
    
    "github.com/xDaijobu/jobu-utils/rabbitmq"
)

func main() {
    var mutex sync.Mutex
    
    // Connect to RabbitMQ
    channel, err := rabbitmq.Start(&mutex)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    
    log.Println("Connected to RabbitMQ!")
    
    // Clean up when done
    defer rabbitmq.InitConnectionManager().Close()
}
```

## 📖 Features

- **Auto-reconnection** - Automatically reconnects when connection is lost
- **Environment variables** - Loads config from `.env` file
- **Thread-safe** - Safe for concurrent use
- **Connection pooling** - Reuses connections efficiently

## 🔧 Configuration

Configuration is done through environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RABBITMQ_HOST` | `localhost` | RabbitMQ server hostname |
| `RABBITMQ_PORT` | `5672` | RabbitMQ server port |
| `RABBITMQ_USER` | `guest` | Username for authentication |
| `RABBITMQ_PASSWORD` | `guest` | Password for authentication |

## 📝 Examples

### Publisher Example

```go
package main

import (
    "log"
    "sync"
    
    "github.com/xDaijobu/jobu-utils/rabbitmq"
    "github.com/xDaijobu/jobu-utils/rabbitmq/publisher"
)

func main() {
    var mutex sync.Mutex
    
    // Connect to RabbitMQ
    channel, err := rabbitmq.Start(&mutex)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    
    // Configure exchange
    config := &rabbitmq.RouteConfig{
        ExchangeName: "my-exchange",
        ExchangeType: "direct",
        RoutingKey:   "my.routing.key",
        QueueName:    "my-queue",
    }
    
    // Create publisher
    pub, err := publisher.NewExchange(channel, config)
    if err != nil {
        log.Fatal("Failed to create publisher:", err)
    }
    
    // Publish a message
    err = pub.Publish([]byte("Hello World!"))
    if err != nil {
        log.Fatal("Failed to publish message:", err)
    }
    
    log.Println("Message published successfully!")
    
    // Clean up
    rabbitmq.InitConnectionManager().Close()
}
```

### Consumer Example

```go
package main

import (
    "log"
    "sync"
    
    "github.com/xDaijobu/jobu-utils/rabbitmq"
    "github.com/xDaijobu/jobu-utils/rabbitmq/consumer"
)

func main() {
    var mutex sync.Mutex
    
    // Connect to RabbitMQ
    channel, err := rabbitmq.Start(&mutex)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    
    // Configure exchange and queue
    config := &rabbitmq.RouteConfig{
        ExchangeName: "my-exchange",
        ExchangeType: "direct",
        RoutingKey:   "my.routing.key",
        QueueName:    "my-queue",
    }
    
    // Create consumer
    cons, err := consumer.NewExchange(channel, config)
    if err != nil {
        log.Fatal("Failed to create consumer:", err)
    }
    
    // Start consuming messages
    messages, err := cons.Consume()
    if err != nil {
        log.Fatal("Failed to start consuming:", err)
    }
    
    log.Println("Waiting for messages...")
    
    // Process messages
    for msg := range messages {
        log.Printf("Received message: %s", msg.Body)
        msg.Ack(false) // Acknowledge the message
    }
    
    // Clean up
    rabbitmq.InitConnectionManager().Close()
}
```

## 🏗️ Installation

```bash
go get github.com/xDaijobu/jobu-utils
```

## 📚 License

MIT
