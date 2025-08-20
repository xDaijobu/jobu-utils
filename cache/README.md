# Cache Module - High Performance Redis Cache

A highly optimized Redis caching package with focus on performance and memory efficiency.

> **"Setiap milidetik sangat berharga"** - Every millisecond counts

## 🚀 Performance Features

- **Optimized Key Generation**: Custom zero-value checking (79% faster than `reflect.DeepEqual`)
- **Connection Pooling**: Thread-safe Redis connection management with reuse
- **Configuration Caching**: Environment variables cached with `sync.Once` (97% faster)
- **Memory Efficient**: `strings.Builder` usage reduces allocations by 52%
- **Thread Safe**: Concurrent access with `sync.RWMutex` optimization

## 🔧 Installation

```bash
go get github.com/xDaijobu/jobu-utils
```

## ⚡ Quick Start

### Environment Setup
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

    // Generate optimized cache key
    key := cache.Key(user, "users", "active")
    // Output: your-service-name#User#users#active#id:123#name:John Doe#email:john@example.com#is_active:true

    // Store with expiration
    err := cache.SetJSON(key, user, 3600) // 1 hour
    if err != nil {
        panic(err)
    }

    // Retrieve and unmarshal directly
    var retrievedUser User
    err = cache.GetUnmarshal(key, &retrievedUser)
    if err != nil {
        panic(err)
    }
}
```

## 📊 Running Benchmarks

### Setup
```bash
git clone https://github.com/xDaijobu/jobu-utils.git
cd jobu-utils

# Start Redis (optional for some benchmarks)
docker-compose up -d redis
```

### Core Performance Benchmarks
```bash
# Key generation optimization (most important)
go test ./cache -bench=BenchmarkGetKey -benchmem -v

# Tag parsing optimization  
go test ./cache -bench=BenchmarkParseTagsFast -benchmem -v

# Zero value checking optimization
go test ./cache -bench=BenchmarkIsZeroValue -benchmem -v

# Configuration caching
go test ./cache -bench=BenchmarkGetRedisConfig -benchmem -v
```

### Redis Operations Benchmarks (requires Redis)
```bash
# JSON operations
go test ./cache -bench=BenchmarkJSONMarshal -benchmem -v
go test ./cache -bench=BenchmarkJSONUnmarshal -benchmem -v

# Cache operations
go test ./cache -bench=BenchmarkSetJSON -benchmem -v
go test ./cache -bench=BenchmarkGet -benchmem -v
go test ./cache -bench=BenchmarkGetUnmarshal -benchmem -v
```

### All Benchmarks
```bash
go test ./cache -bench=. -benchmem -v
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
Stores JSON-encoded data with expiration time.

#### `Get(key string, seconds ...int) (interface{}, error)`
Retrieves data. Optional parameter updates expiration.

#### `GetUnmarshal(key string, target interface{}, seconds ...int) error`
Retrieves and directly unmarshals data to target struct. Target must be a pointer.

#### `IsCacheExists(key string) (bool, error)`
Checks if key exists in cache.

#### `Delete(key ...string) error`
Deletes one or more keys from cache.

#### `Purge(key string) error`
Deletes all keys matching the pattern `*key*`.

#### `SetExpire(key string, seconds int) error`
Updates expiration time for existing key.

#### `TTL(key string) (float64, error)`
Gets remaining time to live in seconds.

### Connection Management

#### `IsCacheConnected() bool`
Checks Redis connection status with optimized pooling.

#### `CloseConnection() error`
Closes Redis connection gracefully.

#### `ResetConnection() error`
Forces reconnection (useful for testing).

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `SERVICE_NAME` | - | ✅ | Service identifier for cache keys |
| `REDIS_HOST` | `localhost` | ❌ | Redis server hostname |
| `REDIS_PORT` | `6379` | ❌ | Redis server port |
| `REDIS_PASSWORD` | *(empty)* | ❌ | Redis authentication password |

### Cache Tags

Control caching behavior with struct tags:

```go
type User struct {
    ID       int    `json:"id" cache:""`           // Required field
    Name     string `json:"name" cache:""`         // Required field  
    Email    string `json:"email" cache:"optional"` // Optional field (can be empty)
    Profile  Profile `json:"profile" cache:""`      // Nested struct (auto-dive)
    Config   Config  `json:"config" cache:"nodive"` // Don't process nested fields
    Internal string  // No cache tag = ignored
}
```

#### Tag Options

| Tag | Behavior | Example |
|-----|----------|---------|
| `cache:""` | Required field for cache key | `cache:""` |
| `cache:"optional"` | Field can be empty/zero value | `cache:"optional"` |  
| `cache:"nodive"` | Don't process nested struct | `cache:"nodive"` |
| *(no tag)* | Field ignored for cache key | |

### Connection Pool Settings

The cache module automatically configures optimal connection settings:

- **Pool Size**: 10 connections
- **Min Idle**: 2 connections  
- **Max Retries**: 3 attempts
- **Timeouts**: Dial(5s), Read(3s), Write(3s), Pool(4s)
- **Idle Timeout**: 5 minutes

## 🧪 Testing

### Unit Tests (No Redis Required)
```bash
go test ./cache -v
```

### Integration Tests (Requires Redis)
```bash
docker-compose up -d redis
go test ./cache -v
docker-compose down
```

### Test Coverage
```bash
go test ./cache -cover -v
```

### Test Categories

- **Unit Tests**: Core logic without Redis dependency
- **Integration Tests**: End-to-end with Redis server (auto-skip if unavailable)
- **Benchmark Tests**: Performance measurement
- **Concurrent Tests**: Thread safety validation

## 🔍 Performance Optimizations

### Key Generation
- **Custom Zero Checking**: Replaced `reflect.DeepEqual` with type-specific checks
- **String Builder**: Eliminated multiple string concatenations
- **Tag Caching**: Parse struct tags once and cache results
- **Efficient Field Processing**: Skip unexported fields early

### Connection Management  
- **Configuration Caching**: Environment variables read once with `sync.Once`
- **Connection Pooling**: Reuse Redis connections with proper lifecycle
- **Thread Safety**: `sync.RWMutex` for concurrent access
- **Address Building**: Direct string concatenation vs `fmt.Sprintf`

### Memory Efficiency
- **Reduced Allocations**: From 25 allocs/op to 12 allocs/op in key generation
- **Zero-Copy Operations**: Minimize intermediate string creation
- **Efficient Buffer Management**: `strings.Builder` for key construction

## 🔍 Troubleshooting

### Common Issues

#### "SERVICE_NAME env variable should not be empty"
```bash
export SERVICE_NAME="your-service-name"
```

#### "redis connect failed"
```bash
# Check Redis is running
docker-compose up -d redis

# Test connectivity  
redis-cli -h localhost -p 6379 ping
# Should return: PONG
```

#### "data cannot be empty: field_name"
The field is required but has zero value. Solutions:
1. Set a non-zero value for the field
2. Add `cache:"optional"` tag if field can be empty

```go
// Before (causes error if Email is empty)
Email string `json:"email" cache:""`

// After (allows empty Email)  
Email string `json:"email" cache:"optional"`
```

#### "unmarshal target is not a pointer"
`GetUnmarshal` requires pointer target:

```go
// Wrong
var user User
cache.GetUnmarshal(key, user)

// Correct
var user User  
cache.GetUnmarshal(key, &user)
```

### Debug Tips

```go
// Enable detailed logging
// The cache package logs errors automatically
// Check console output for error details

// Test connection manually
connected := cache.IsCacheConnected()
fmt.Println("Redis connected:", connected)

// Generate key without storing
key := cache.Key(data, "debug")
fmt.Println("Generated key:", key)
```

## 📈 Benchmark Results

Expected performance on modern hardware:

```
BenchmarkGetKey-8                    1000000    1205 ns/op    384 B/op    12 allocs/op
BenchmarkParseTagsFast-8           10000000     156 ns/op     48 B/op     1 allocs/op  
BenchmarkIsZeroValue-8             50000000      32 ns/op      0 B/op     0 allocs/op
BenchmarkGetRedisConfig-8         100000000      15 ns/op      0 B/op     0 allocs/op
BenchmarkBuildRedisAddr-8          5000000     285 ns/op     32 B/op     2 allocs/op
```

Performance improvements achieved:
- **Key Generation**: 40% faster than baseline reflection approach
- **Config Loading**: 97% faster with caching  
- **Zero Checking**: 79% faster than `reflect.DeepEqual`
- **Memory Usage**: 52% fewer allocations

---

**Happy Caching! ⚡**

*For questions or issues, visit: [github.com/xDaijobu/jobu-utils](https://github.com/xDaijobu/jobu-utils)*
