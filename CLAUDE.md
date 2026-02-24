```
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
```

## Project Overview

Logos is a semantic logging and analysis platform consisting of multiple Go services:

1. **Log SDK**: High-performance client library for generating structured logs
2. **Config Server**: Web service for managing logging configurations
3. **Log Processor**: Service for processing and enriching log data
4. **Log Analyzer**: Service for analyzing and querying log data
5. **Frontend**: Web interface for log visualization

## Command Reference

### Log SDK Development

```bash
cd log-sdk/log-sdk

# Build and test
go build ./...
go test -v ./...

# Run specific test
go test -v -run TestNew ./pkg/logger

# Run benchmarks
go test -bench="BenchmarkComparison" -benchmem ./pkg/logger
```

### Config Server Development

```bash
cd config-server

# Build and run
go run cmd/main.go

# Run tests
cd config-server
go test -v ./...
```

### Log Processor Development

```bash
cd log-processor

# Build and run
go run cmd/job/main.go

# Run tests
cd log-processor
go test -v ./...
```

### Log Analyzer Development

```bash
cd log-analyzer

# Build and run
go run cmd/main.go

# Run tests
cd log-analyzer
go test -v ./...
```

### Examples

```bash
# Run HTTP service example
cd examples/http
go run main.go

# Run SDK usage example
cd examples/sdk
go run main.go
```

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Logos Platform                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ ┌──────────┐    ┌──────────┐    ┌─────────────┐           │
│ │  Log SDK │───▶│ Kafka    │───▶│ Log Processor│           │
│ │  Client  │    │ Brokers  │    │             │           │
│ └──────────┘    └──────────┘    └─────────────┘           │
│                                                              │
│ ┌──────────┐                ┌──────────────┐                │
│ │ Config   │                │ Log Analyzer │                │
│ │ Server   │                │              │                │
│ └──────────┘                └──────────────┘                │
│                                                              │
│ ┌──────────┐                ┌──────────────┐                │
│ │ Frontend │                │ ETCD         │                │
│ │ (React)  │                │ Config       │                │
│ └──────────┘                └──────────────┘                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Key Files and Directories

| File/Directory | Purpose |
|----------------|---------|
| log-sdk/ | Log SDK client library with dual API styles (traditional/chain-based) |
| config-server/ | Configuration management service |
| log-processor/ | Log data processing and enrichment |
| log-analyzer/ | Log analysis and query service |
| frontend/ | React-based UI for log visualization |
| examples/ | Usage examples (HTTP service, SDK basics) |
| openspec/ | OpenSpec artifacts (changes, specs, designs) |

## Log SDK Architecture

For detailed information about the Log SDK, see [log-sdk/log-sdk/CLAUDE.md](log-sdk/log-sdk/CLAUDE.md).

## Common Workflows

### Adding a New Log SDK Feature

1. Create or update OpenSpec artifacts in `openspec/changes/`
2. Implement changes in `log-sdk/log-sdk/pkg/`
3. Write tests in corresponding `*_test.go` files
4. Run tests and benchmarks
5. Update documentation in `README.md` and examples in `examples/`

### Deploying Services

```bash
# Build all services
cd log-sdk/log-sdk
make all

# Build specific service
cd log-sdk/log-sdk
make log-processor

# Run in development mode
cd log-sdk/log-sdk
make dev
```
