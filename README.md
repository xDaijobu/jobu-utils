# Jobu Utils

A collection of high-performance Go utilities for common development tasks including caching and message queuing.

## Modules

### 🗄️ [Cache](./cache/README.md)
High-performance Redis caching utilities with connection pooling and error handling.

### 🐰 [RabbitMQ](./rabbitmq/README.md)
Production-ready RabbitMQ client with auto-reconnect, publisher/consumer patterns, and robust error handling.

## Installation

```bash
go get github.com/xDaijobu/jobu-utils
```

## Quick Start

### Cache Example
```go
import "jobu-utils/cache"

// Set a value
err := cache.Set("user:123", "john_doe", 3600)

// Get a value
value, err := cache.Get("user:123")
```

### RabbitMQ Publisher Example
```go
import (
    "jobu-utils/rabbitmq"
    "jobu-utils/rabbitmq/publisher"
)

route := &publisher.Route{
    RouteConfig: rabbitmq.RouteConfig{
        ExchangeName: "user-events",
        ExchangeType: "direct",
        RoutingKey:   "user.created",
        QueueName:    "user-processor",
    },
}

err := route.Publish(&publisher.Publish{
    Body: `{"user_id": "123", "action": "created"}`,
})
```

### RabbitMQ Consumer Example
```go
import (
    "jobu-utils/rabbitmq"
    "jobu-utils/rabbitmq/consumer"
)

route := &consumer.ConsumerRoute{
    RouteConfig: rabbitmq.RouteConfig{
        ExchangeName: "user-events",
        ExchangeType: "direct",
        RoutingKey:   "user.created",
        QueueName:    "user-processor",
    },
    ConsumerTag: "user-service",
    AutoAck:     false,
}

handler := func(delivery amqp.Delivery) error {
    log.Printf("Processing: %s", string(delivery.Body))
    // Your business logic here
    return nil
}

err := route.StartConsuming(handler)
```

## Features

- ✅ **Production Ready**: Battle-tested utilities with proper error handling
- ✅ **Auto-Reconnect**: Automatic reconnection for network failures
- ✅ **Thread-Safe**: Concurrent operations handled safely
- ✅ **Configurable**: Flexible configuration options
- ✅ **Logging**: Comprehensive logging for debugging and monitoring
- ✅ **High Performance**: Optimized for production workloads

## Environment Variables

### Cache (Redis)
```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

### RabbitMQ
```bash
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
```

## Docker Compose

For development and testing:

```bash
docker-compose up -d
```

This will start Redis and RabbitMQ instances for local development.

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific module tests
go test ./cache/...
go test ./rabbitmq/...
```

## Benchmarks

```bash
# Run benchmarks
go test -bench=. ./...

# Run cache benchmarks
go test -bench=. ./cache/

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./cache/
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

If you have any questions or issues, please open an issue on GitHub.
