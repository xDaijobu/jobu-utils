# Jobu Utils - High Performance Cache Package

A highly optimized Redis caching package for Go applications with focus on performance and ease of use.

> **"Setiap milidetik sangat berharga"** - Every millisecond counts

## 🚀 Features

- **Optimized Key Generation**: Custom zero-value checking and efficient string building
- **Connection Pooling**: Thread-safe Redis connection management
- **Configuration Caching**: Environment variables cached to eliminate repeated system calls
- **Memory Efficient**: `strings.Builder` usage and reduced allocations
- **Thread Safe**: Concurrent access with `sync.RWMutex`

## 🔧 Installation

```bash
go get github.com/xDaijobu/jobu-utils
```

## ⚡ Quick Start

### Environment Setup
```bash
export SERVICE_NAME="your-service-name"
export REDIS_HOST="localhost"        # Optional
export REDIS_PORT="6379"            # Optional
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
    user := User{ID: 123, Name: "John Doe", IsActive: true}

    // Generate cache key
    key := cache.Key(user, "users", "active")
    
    // Store data
    cache.SetJSON(key, user, 3600) // 1 hour expiration
    
    // Retrieve data
    var retrievedUser User
    cache.GetUnmarshal(key, &retrievedUser)
}
```

## 📊 Running Benchmarks

```bash
# Clone and setup
git clone https://github.com/xDaijobu/jobu-utils.git
cd jobu-utils

# Start Redis (optional)
docker-compose up -d redis

# Run benchmarks
go test ./cache/... -bench=. -benchmem -v
```

### Benchmark Categories
```bash
# Key generation performance
go test ./cache/... -bench=BenchmarkGetKey -benchmem -v

# Connection management
go test ./cache/... -bench=BenchmarkGetRedisConfig -benchmem -v

# Cache operations (requires Redis)
go test ./cache/... -bench=BenchmarkSetJSON -benchmem -v
```

## 📖 API Reference

### Key Generation
- `Key(data interface{}, prefixes ...string) string` - Generate key using SERVICE_NAME
- `ExternalKey(serviceName string, data interface{}, prefixes ...string) string` - Generate key with custom service

### Cache Operations
- `SetJSON(key string, value interface{}, seconds int) error` - Store JSON data
- `Get(key string, seconds ...int) (interface{}, error)` - Retrieve data
- `GetUnmarshal(key string, target interface{}, seconds ...int) error` - Retrieve and unmarshal
- `IsCacheExists(key string) (bool, error)` - Check if key exists
- `Delete(key ...string) error` - Delete keys
- `SetExpire(key string, seconds int) error` - Update expiration
- `TTL(key string) (float64, error)` - Get time to live

## ⚙️ Configuration

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | *required* | Service identifier |
| `REDIS_HOST` | `localhost` | Redis hostname |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | *(empty)* | Redis password |

### Cache Tags
```go
type User struct {
    ID       int    `json:"id" cache:""`           // Required
    Email    string `json:"email" cache:"optional"` // Optional
    Profile  Profile `json:"profile" cache:""`      // Auto-dive
    Config   Config  `json:"config" cache:"nodive"` // No dive
}
```

## 🧪 Testing

```bash
# Run all tests
go test ./cache/... -v

# With coverage
go test ./cache/... -cover -v

# With Redis integration
docker-compose up -d redis
go test ./cache/... -v
```

## 🔍 Troubleshooting

**Common Issues:**
1. **"SERVICE_NAME env variable should not be empty"**
   ```bash
   export SERVICE_NAME="your-service-name"
   ```

2. **"redis connect failed"**
   ```bash
   docker-compose up -d redis
   redis-cli -h localhost -p 6379 ping
   ```

3. **"data cannot be empty"**
   - Add `cache:"optional"` tag for optional fields

## 🔗 Links

- **Repository**: [github.com/xDaijobu/jobu-utils](https://github.com/xDaijobu/jobu-utils)
- **Issues**: [Issues](https://github.com/xDaijobu/jobu-utils/issues)

---

**Happy Caching! 🚀**

*Performance-first Redis caching for Go applications*
