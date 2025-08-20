# Jobu Utils - High Performance Cache Package

A highly optimized Redis caching package for Go applications with focus on performance and ease of use.

> **"Setiap milidetik sangat berharga"** - Every millisecond counts

## 🚀 Performance Features

- **Optimized Key Generation**: Custom zero-value checking and efficient string building
- **Connection Pooling**: Thread-safe Redis connection management with connection reuse
- **Configuration Caching**: Environment variables cached to eliminate repeated system calls
- **Memory Efficient**: `strings.Builder` usage and reduced allocations
- **Thread Safe**: Concurrent access with `sync.RWMutex` for optimal performance

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Running Benchmarks](#running-benchmarks)
- [API Reference](#api-reference)
- [Configuration](#configuration)
- [Testing](#testing)
- [Performance Metrics](#performance-metrics)
- [Docker Support](#docker-support)

## 🔧 Installation

```bash
go get github.com/xDaijobu/jobu-utils
```

## 🚀 Quick Start

### Environment Setup

Set the required environment variables:

```bash
export SERVICE_NAME="your-service-name"
export REDIS_HOST="localhost"        # Optional, defaults to localhost
export REDIS_PORT="6379"            # Optional, defaults to 6379
export REDIS_PASSWORD="your-password" # Optional
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/xDaijobu/jobu-utils/cache"
)

type User struct {
    ID       int    `json:"id" cache:""`
    Name     string `json:"name" cache:""`
    Email    string `json:"email" cache:"optional"`
    IsActive bool   `json:"is_active" cache:""`
}

func main() {
    user := User{
        ID:       123,
        Name:     "John Doe",
        Email:    "john@example.com",
        IsActive: true,
    }

    // Generate cache key
    key := cache.Key(user, "users", "active")
    fmt.Println("Cache Key:", key)
    // Output: your-service-name#User#users#active#id:123#name:John Doe#email:john@example.com#is_active:true

    // Store data in cache
    err := cache.SetJSON(key, user, 3600) // 1 hour expiration
    if err != nil {
        panic(err)
    }

    // Retrieve data from cache
    data, err := cache.Get(key)
    if err != nil {
        panic(err)
    }

    // Unmarshal directly to struct
    var retrievedUser User
    err = cache.GetUnmarshal(key, &retrievedUser)
    if err != nil {
        panic(err)
    }
}
```

## 📊 Running Benchmarks

### Prerequisites

1. **Go 1.25+** installed
2. **Redis server** running (optional for benchmarks)

### Quick Benchmark Setup

```bash
# Clone the repository
git clone https://github.com/xDaijobu/jobu-utils.git
cd jobu-utils

# Run Redis with Docker
docker-compose up -d redis

# Run all benchmarks
go test ./cache/... -bench=. -benchmem -v
```

### Running All Benchmarks

```bash
# Run all benchmarks
go test ./cache/... -bench=. -benchmem -v

# Run benchmarks with multiple iterations
go test ./cache/... -bench=. -benchmem -count=5

# Run benchmarks and save results
go test ./cache/... -bench=. -benchmem > benchmark_results.txt
```

### Specific Benchmark Categories

#### Key Generation Benchmarks (Core Optimization)
```bash
# Benchmark key generation performance
go test ./cache/... -bench=BenchmarkGetKey -benchmem -v

# Benchmark tag parsing optimization
go test ./cache/... -bench=BenchmarkParseTagsFast -benchmem -v

# Benchmark zero value checking
go test ./cache/... -bench=BenchmarkIsZeroValue -benchmem -v
```

#### Connection Management Benchmarks
```bash
# Benchmark Redis configuration caching
go test ./cache/... -bench=BenchmarkGetRedisConfig -benchmem -v

# Benchmark address building
go test ./cache/... -bench=BenchmarkBuildRedisAddr -benchmem -v

# Benchmark connection checking
go test ./cache/... -bench=BenchmarkIsCacheConnected -benchmem -v
```

#### Cache Operations Benchmarks (With Redis)
```bash
# Ensure Redis is running first
docker-compose up -d redis

# Benchmark Redis operations
go test ./cache/... -bench=BenchmarkSetJSON -benchmem -v
go test ./cache/... -bench=BenchmarkGet -benchmem -v
go test ./cache/... -bench=BenchmarkGetUnmarshal -benchmem -v
go test ./cache/... -bench=BenchmarkIsCacheExists -benchmem -v
```

### Expected Benchmark Output

```
BenchmarkGetKey-8                    1000000    1205 ns/op    384 B/op    12 allocs/op
BenchmarkParseTagsFast-8           10000000     156 ns/op     48 B/op     1 allocs/op
BenchmarkIsZeroValue-8             50000000      32 ns/op      0 B/op     0 allocs/op
BenchmarkGetRedisConfig-8         100000000      15 ns/op      0 B/op     0 allocs/op
BenchmarkBuildRedisAddr-8          5000000     285 ns/op     32 B/op     2 allocs/op
BenchmarkJSONMarshal-8             2000000     785 ns/op    128 B/op     2 allocs/op
BenchmarkJSONUnmarshal-8           1000000    1120 ns/op    256 B/op     8 allocs/op
```

## 📖 API Reference

### Key Generation

#### `Key(data interface{}, prefixes ...string) string`
Generates cache key using SERVICE_NAME environment variable.

```go
key := cache.Key(user, "users", "active")
```

#### `ExternalKey(serviceName string, data interface{}, prefixes ...string) string`
Generates cache key with custom service name.

```go
key := cache.ExternalKey("external-service", user, "cache")
```

### Cache Operations

#### `SetJSON(key string, value interface{}, seconds int) error`
Stores JSON-encoded data with expiration.

#### `Get(key string, seconds ...int) (interface{}, error)`
Retrieves data and optionally updates expiration.

#### `GetUnmarshal(key string, target interface{}, seconds ...int) error`
Retrieves and unmarshals data directly to struct.

#### `IsCacheExists(key string) (bool, error)`
Checks if key exists in cache.

#### `Delete(key ...string) error`
Deletes one or more keys.

#### `Purge(key string) error`
Deletes all keys matching pattern.

#### `SetExpire(key string, seconds int) error`
Updates expiration time for existing key.

#### `TTL(key string) (float64, error)`
Gets remaining time to live in seconds.

### Connection Management

#### `IsCacheConnected() bool`
Checks Redis connection status.

#### `CloseConnection() error`
Closes Redis connection gracefully.

#### `ResetConnection() error`
Forces reconnection (useful for testing).

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | *required* | Service identifier for cache keys |
| `REDIS_HOST` | `localhost` | Redis server hostname |
| `REDIS_PORT` | `6379` | Redis server port |
| `REDIS_PASSWORD` | *(empty)* | Redis authentication password |

### Cache Tags

Use struct tags to control caching behavior:

```go
type User struct {
    ID       int    `json:"id" cache:""`           // Required field
    Name     string `json:"name" cache:""`         // Required field
    Email    string `json:"email" cache:"optional"` // Optional field
    Profile  Profile `json:"profile" cache:""`      // Nested struct (auto-dive)
    Config   Config  `json:"config" cache:"nodive"` // Don't dive into struct
}
```

### Tag Options

- **Empty tag** `cache:""`: Field is required for cache key
- **Optional** `cache:"optional"`: Field can be empty/zero value
- **No dive** `cache:"nodive"`: Don't process nested struct fields
- **No tag**: Field is ignored for cache key generation

## 🧪 Testing

### Run All Tests
```bash
go test ./cache/... -v
```

### Run Tests with Coverage
```bash
go test ./cache/... -cover -v
```

### Integration Tests with Redis
```bash
# Start Redis using Docker Compose
docker-compose up -d redis

# Run tests (integration tests auto-skip if Redis unavailable)
go test ./cache/... -v

# Stop Redis
docker-compose down
```

### Test Categories

- **Unit Tests**: Core logic testing without Redis dependency
- **Integration Tests**: End-to-end testing with Redis server
- **Benchmark Tests**: Performance measurement and optimization validation
- **Concurrent Tests**: Thread safety and race condition testing

## 📈 Performance Metrics

### Key Optimizations Implemented

1. **Custom Zero Value Checking**: Replaced `reflect.DeepEqual` with type-specific checks
2. **String Builder Usage**: Eliminated multiple string concatenations
3. **Configuration Caching**: `sync.Once` pattern for environment variables
4. **Connection Pooling**: Reuse Redis connections with proper lifecycle management
5. **Tag Parsing Optimization**: Cached parsing results and efficient string operations
6. **Thread-Safe Design**: `sync.RWMutex` for concurrent access optimization

## 🐳 Docker Support

The repository includes Docker Compose configuration for easy development and testing:

### Start Redis for Development
```bash
docker-compose up -d redis
```

### Build and Run Application
```bash
# Build the application
docker-compose build app

# Run with Redis
docker-compose up
```

### Docker Compose Services
- **redis**: Redis server for caching
- **app**: Your Go application (configure in docker-compose.yml)

## 🔍 Troubleshooting

### Common Issues

1. **"SERVICE_NAME env variable should not be empty"**
   ```bash
   export SERVICE_NAME="your-service-name"
   ```

2. **"redis connect failed"**
   ```bash
   # Check Redis server
   docker-compose ps redis
   
   # Check connectivity
   redis-cli -h localhost -p 6379 ping
   ```

3. **"data cannot be empty"**
   - Add `cache:"optional"` tag for optional fields
   - Ensure required fields have non-zero values

4. **Benchmark issues**
   ```bash
   # Clean test cache
   go clean -testcache
   
   # Run with verbose output
   go test ./cache/... -bench=. -benchmem -v
   ```

### Debug Mode

Enable verbose logging:
```go
// The package logs errors automatically to console
// Check application logs for detailed error messages
```

## 🤝 Contributing

1. Fork the repository: [github.com/xDaijobu/jobu-utils](https://github.com/xDaijobu/jobu-utils)
2. Create feature branch: `git checkout -b feature/amazing-feature`
3. Add tests for new functionality
4. Run benchmarks: `go test ./cache/... -bench=. -benchmem`
5. Commit changes: `git commit -m 'Add amazing feature'`
6. Push to branch: `git push origin feature/amazing-feature`
7. Open Pull Request

### Performance Guidelines

- Always include benchmarks for new features
- Maintain or improve existing performance metrics
- Use `strings.Builder` for string concatenation
- Minimize reflection usage where possible
- Add comprehensive test coverage

### Development Setup

```bash
# Clone repository
git clone https://github.com/xDaijobu/jobu-utils.git
cd jobu-utils

# Install dependencies
go mod download

# Start Redis for testing
docker-compose up -d redis

# Run tests
go test ./cache/... -v

# Run benchmarks
go test ./cache/... -bench=. -benchmem
```

## 📄 Repository Structure

```
jobu-utils/
├── cache/                  # Cache package
│   ├── cache.go           # Core cache operations
│   ├── connect.go         # Redis connection management
│   ├── key.go             # Optimized key generation
│   ├── cache_test.go      # Cache operation tests
│   ├── connect_test.go    # Connection tests
│   └── key_test.go        # Key generation tests
├── docker-compose.yml     # Docker services
├── Dockerfile            # Application container
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
└── README.md            # This file
```

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built with performance in mind: **"Setiap milidetik sangat berharga"**
- Optimized for high-throughput applications
- Designed for production environments
- Community-driven development

## 🔗 Links

- **Repository**: [github.com/xDaijobu/jobu-utils](https://github.com/xDaijobu/jobu-utils)
- **Issues**: [github.com/xDaijobu/jobu-utils/issues](https://github.com/xDaijobu/jobu-utils/issues)
- **Releases**: [github.com/xDaijobu/jobu-utils/releases](https://github.com/xDaijobu/jobu-utils/releases)

---

**Happy Caching! 🚀**

*Performance-first Redis caching for Go applications*
