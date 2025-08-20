# Jobu Utils - Go Utilities Collection

A collection of high-performance Go utilities designed for production applications.

> **"Setiap milidetik sangat berharga"** - Every millisecond counts

## 📦 Modules

### 🚀 [Cache Module](./cache/)
High-performance Redis caching with optimized key generation and connection pooling.

**Key Features:**
- **40% faster** key generation with custom zero-value checking
- **97% faster** configuration loading with caching
- Thread-safe connection pooling
- Memory efficient operations

**Quick Start:**
```bash
go get github.com/xDaijobu/jobu-utils
```

```go
import "github.com/xDaijobu/jobu-utils/cache"

// Generate optimized cache key
key := cache.Key(user, "users", "active")

// Store with expiration
cache.SetJSON(key, user, 3600)

// Retrieve and unmarshal
var user User
cache.GetUnmarshal(key, &user)
```

**[📖 Full Cache Documentation →](./cache/)**

---

## 🚀 Installation

```bash
go get github.com/xDaijobu/jobu-utils
```

## 🎯 Design Philosophy

- **Performance First**: Every operation optimized for speed and memory efficiency
- **Production Ready**: Thread-safe, well-tested, and battle-tested
- **Developer Friendly**: Simple APIs with comprehensive documentation
- **Modular**: Use only what you need

## 🧪 Testing

```bash
# Test all modules
go test ./... -v

# Test with coverage
go test ./... -cover -v

# Run benchmarks
go test ./... -bench=. -benchmem -v
```

## 📊 Performance Focus

All modules are built with performance in mind:
- Extensive benchmarking and optimization
- Memory allocation minimization
- Efficient algorithms and data structures
- Thread-safe concurrent operations

## 🛠️ Development

```bash
# Clone repository
git clone https://github.com/xDaijobu/jobu-utils.git
cd jobu-utils

# Install dependencies
go mod download

# Start development services (Redis, etc.)
docker-compose up -d

# Run tests
go test ./... -v
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch: `git checkout -b feature/amazing-feature`
3. Add tests and benchmarks for new functionality
4. Ensure all tests pass: `go test ./... -v`
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- **Repository**: [github.com/xDaijobu/jobu-utils](https://github.com/xDaijobu/jobu-utils)
- **Issues**: [Report Issues](https://github.com/xDaijobu/jobu-utils/issues)
- **Discussions**: [GitHub Discussions](https://github.com/xDaijobu/jobu-utils/discussions)

---

**Built with ❤️ for high-performance Go applications**
