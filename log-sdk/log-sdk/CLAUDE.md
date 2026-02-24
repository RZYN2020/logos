```
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
```

## Command Reference

### Build and Test

```bash
# Build the entire SDK
cd log-sdk/log-sdk
go build ./...

# Run all tests
cd log-sdk/log-sdk
go test -v ./...

# Run a single test file
cd log-sdk/log-sdk/pkg/logger
go test -v logger_test.go logger.go

# Run a specific test case
cd log-sdk/log-sdk/pkg/logger
go test -v -run TestNew

# Run benchmarks
cd log-sdk/log-sdk/pkg/logger
go test -bench="BenchmarkComparison" -benchmem

# Run integration tests (requires ETCD)
cd log-sdk/log-sdk/pkg/strategy
go test -run Integration -v
```

### Lint and Format

```bash
# Run gofmt
cd log-sdk/log-sdk
gofmt -w ./...

# Run golint
cd log-sdk/log-sdk
golint ./...

# Run staticcheck
cd log-sdk/log-sdk
staticcheck ./...
```

## High-Level Architecture

The Log SDK is a high-performance, semantic logging library for Go applications. It features dual API styles (traditional and chain-based) with built-in filtering and dynamic policy configuration.

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                        Log SDK                              │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Logger     │  │    Hook      │  │   Strategy   │      │
│  │    API       │──│   System     │──│    Engine    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                 │                 │               │
│         └─────────────────┼─────────────────┘               │
│                           ▼                                 │
│                  ┌─────────────────┐                        │
│                  │  Async Producer │──▶ Kafka               │
│                  └─────────────────┘                        │
│                           │                                 │
│                  ┌────────┴────────┐                        │
│                  ▼                 ▼                        │
│           ┌──────────┐      ┌──────────┐                   │
│           │  Console │      │  Buffer  │                   │
│           │(Fallback)│      │(Backpressure)                │
│           └──────────┘      └──────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

### Package Structure

```
log-sdk/log-sdk/
├── cmd/                # Command-line tools
├── pkg/
│   ├── logger/         # Core Logger API and implementation
│   │   ├── logger.go           # Main Logger interface
│   │   ├── logger_test.go      # Unit tests
│   │   └── performance_test.go # Benchmarks
│   ├── async/          # Async message production and buffering
│   │   ├── producer.go         # Kafka producer
│   │   └── buffer.go           # Ring buffer
│   ├── strategy/       # Dynamic policy engine
│   │   ├── engine.go           # Strategy evaluation
│   │   └── etcd.go             # ETCD configuration management
│   ├── encoder/        # Log entry encoding
│   │   └── json.go             # JSON encoder
│   └── guard/          # Web framework integration
│       └── guard.go            # Gin middleware
└── examples/           # Usage examples
    ├── sdk/             # Basic SDK usage
    └── http/            # HTTP service with Gin middleware
```

### Key Files and Their Purposes

| File | Purpose |
|------|---------|
| logger.go | Core Logger interface with dual API styles (traditional/chain) and object pool |
| async/producer.go | Kafka producer with async sending, batching, and fallback to console |
| async/buffer.go | Lock-free ring buffer for message queuing and backpressure |
| strategy/engine.go | Strategy evaluation engine with ETCD integration |
| strategy/etcd.go | ETCD client with retry logic and strategy management |
| encoder/json.go | JSON encoder with field validation and pretty-print support |
| guard/guard.go | Gin web framework middleware for request logging |

### Important Concepts

1. **Object Pool**: Reduces GC pressure by reusing LogEntry objects
2. **Lock-Free Buffer**: Multi-producer single-consumer (MPSC) ring buffer for high throughput
3. **Hook System**: Filter logs before dispatch (level, line number, regex)
4. **Strategy Engine**: Dynamic policy loading from ETCD with hot-reload
5. **Fallback Mechanism**: Kafka failures automatically log to console

### Common Development Tasks

```bash
# Create a new logger configuration
cd log-sdk/log-sdk/pkg/logger
# Modify logger.go's Config struct

# Add a new hook type
cd log-sdk/log-sdk/pkg/logger
# Implement Hook interface and add to logger.go

# Add a new encoder format
cd log-sdk/log-sdk/pkg/encoder
# Create new file implementing Encoder interface

# Test with real Kafka and ETCD
cd log-sdk/log-sdk/examples/http
# Start Kafka and ETCD locally, then run the example
go run main.go
```
