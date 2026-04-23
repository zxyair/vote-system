# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a distributed voting system built with Go, featuring both HTTP and gRPC APIs, with support for multiple storage backends (memory and Redis). The system includes real-time updates via Server-Sent Events (SSE), observability with Prometheus metrics, and a simple web frontend.

## Architecture

### Core Components

1. **Protocol Buffers** (`api/proto/voting/v1/voting.proto`): Defines the gRPC service API and message types for polls, votes, and statistics.

2. **gRPC Server** (`cmd/grpcserver/main.go`): Main backend server that implements the VotingService. Handles all business logic and data storage. Supports both memory and Redis storage backends.

3. **HTTP Server** (`cmd/httpserver/main.go`): Frontend server that serves the web interface and exposes HTTP endpoints. Acts as a proxy to the gRPC server for API calls.

4. **Service Layer** (`internal/service/`): Contains the business logic implementation with validation, idempotency handling, and poll management.

5. **Storage Layer** (`internal/store/`): 
   - `memory/`: In-memory storage with mutex-based concurrency control
   - `redis/`: Redis-backed storage with Lua scripts for atomic operations

6. **HTTP Handlers** (`internal/http/handler/`): Handle HTTP requests, gRPC client communication, SSE streaming, and serve static web assets.

7. **Observability** (`internal/obs/`): Prometheus metrics collection for HTTP requests, gRPC calls, and business operations.

### Key Features

- **Poll Management**: Create, close, delete polls with expiration times
- **Voting System**: Vote and undo vote with idempotency keys
- **Real-time Updates**: Server-sent events for live vote counts
- **Public/Private Polls**: Support for both public and private polls
- **User-centric Views**: Search polls, view user's votes and created polls
- **Idempotency**: Protection against duplicate operations
- **Pagination**: Cursor-based pagination for large datasets

## Development Commands

### Building and Running

```bash
# Generate protobuf files
buf generate

# Run gRPC server (default :9090)
# Set REDIS_ADDR to use Redis instead of memory storage
go run cmd/grpcserver/main.go

# Run HTTP server (default :8080)
# Connects to gRPC server at :9090 by default
go run cmd/httpserver/main.go

# Run both servers in parallel
go run cmd/grpcserver/main.go &
go run cmd/httpserver/main.go &
```

### Testing and Benchmarking

```bash
# Run correctness tests with k6
./scripts/run_correctness.sh

# Run benchmark tests
./scripts/run_bench.sh

# Run individual test files
go test ./...
```

### Observability

```bash
# Setup observability stack (Prometheus + Grafana)
./scripts/observability_up.sh

# Apply observability configurations
./scripts/observability_apply.sh

# Port forwarding for services
./scripts/portforward_prometheus.sh
./scripts/portforward_grafana.sh
```

## Configuration

### Environment Variables

- `HTTP_ADDR`: HTTP server address (default: `:8080`)
- `GRPC_ADDR`: gRPC server address (default: `:9090`)
- `REDIS_ADDR`: Redis server address (optional, if not set uses memory store)
- `REDIS_PASSWORD`: Redis password (optional)
- `METRICS_ADDR`: Metrics server address (default: `:2112`)

### Storage Backends

The system automatically detects the storage backend:
- If `REDIS_ADDR` is set and connects successfully, uses Redis
- Otherwise falls back to in-memory storage

## Code Structure Guidelines

### Protocol Buffer Generation

Always regenerate protobuf files after making changes to `.proto` files:
```bash
buf generate
```

### Error Handling

- Use error types from `internal/service/errors.go`
- HTTP handlers automatically convert gRPC errors to appropriate HTTP status codes
- Business operations are tracked with metrics in `obs.VoteOp()`

### Idempotency

All operations support idempotency keys to prevent duplicate actions:
- Keys are stored for 5 minutes
- Memory store tracks per-user idempotency keys
- Redis store uses Lua scripts for atomic operations

### Concurrency

- Memory store uses `sync.RWMutex` for thread safety
- All map operations are properly guarded
- Vote operations use atomic patterns to prevent race conditions

### Frontend Integration

The web frontend (`web/`) is a simple single-page application that:
- Makes HTTP requests to the HTTP server
- Uses SSE for real-time vote updates
- Communicates with `/api` endpoints for all operations

## Development Workflow

1. Protocol changes require:
   - Update `.proto` file
   - Run `buf generate`
   - Update service implementation if needed

2. New features should:
   - Add to protobuf definition
   - Implement in gRPC server
   - Add HTTP handler if needed
   - Update web frontend

3. Testing should cover:
   - Correctness tests via `test_correctness.js`
   - Performance benchmarks via `bench_vote.js`
   - Unit tests for business logic