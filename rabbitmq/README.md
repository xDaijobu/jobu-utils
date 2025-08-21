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

## 📋 Features

- **Auto-reconnection**: Automatic reconnection with exponential backoff
- **Publisher**: Send messages to RabbitMQ exchanges
- **Consumer**: Consume messages from RabbitMQ queues
- **Routing Configuration**: Flexible exchange, queue, and binding setup
- **Environment Configuration**: Easy setup via environment variables

## 📤 Publisher Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/xDaijobu/jobu-utils/rabbitmq"
    "github.com/xDaijobu/jobu-utils/rabbitmq/publisher"
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
    
    // Publish message
    message := &publisher.Message{
        Body:        []byte(`{"user_id": "123", "email": "user@example.com"}`),
        ContentType: "application/json",
    }
    
    err = pub.Publish(context.Background(), message)
    if err != nil {
        log.Printf("Failed to publish message: %v", err)
    }
}
```

## 📥 Consumer Usage

```go
package main

import (
    "context"
    "log"
    
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
    
    // Message handler
    handler := func(delivery amqp.Delivery) error {
        log.Printf("Received message: %s", string(delivery.Body))
        
        // Process message here
        // ...
        
        // Acknowledge message
        return delivery.Ack(false)
    }
    
    // Start consuming
    err = cons.Consume(context.Background(), handler)
    if err != nil {
        log.Printf("Failed to consume: %v", err)
    }
}
```

## 🔧 Configuration

### RouteConfig Structure

```go
type RouteConfig struct {
    ExchangeName string // Name of the exchange
    ExchangeType string // Type: "direct", "fanout", "topic", "headers"
    RoutingKey   string // Routing key for binding
    QueueName    string // Name of the queue
    Durable      bool   // Whether exchange/queue should survive server restart
    AutoDelete   bool   // Whether to auto-delete when no longer used
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RABBITMQ_HOST` | RabbitMQ server host | `localhost` |
| `RABBITMQ_PORT` | RabbitMQ server port | `5672` |
| `RABBITMQ_USER` | RabbitMQ username | `guest` |
| `RABBITMQ_PASSWORD` | RabbitMQ password | `guest` |

## 🐳 Docker Setup

For local development, you can use Docker to run RabbitMQ:

```bash
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3-management
```

Access the management UI at: http://localhost:15672

## 📝 Example

See `example/main.go` for a complete working example with both publisher and consumer.

## 🔄 Auto-Reconnection

The library automatically handles connection failures and reconnects with exponential backoff:

- **Max Retries**: 5 attempts
- **Initial Delay**: 1 second
- **Max Delay**: 30 seconds
- **Backoff Factor**: 2.0

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License.
