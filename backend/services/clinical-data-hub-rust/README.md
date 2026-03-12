# Clinical Data Hub Rust Service

Ultra-high performance clinical data intelligence hub with multi-layer caching, parallel data aggregation, and real-time stream processing.

## 🏗️ Architecture

### Core Performance Features

- **Multi-Layer Caching**: L1 (Memory) + L2 (Redis) + L3 (Persistent) with sub-millisecond access
- **Parallel Data Aggregation**: Concurrent data fetching with intelligent batching
- **Stream Processing**: Real-time clinical data updates via Kafka
- **Zero-Copy Operations**: Memory-efficient data transformations
- **Compression**: LZ4/Zstd compression for optimal storage efficiency
- **Memory Safety**: Rust's ownership model ensures memory safety without garbage collection

### Performance Targets

- **L1 Cache Access**: < 1ms (in-memory)
- **L2 Cache Access**: 1-5ms (Redis cluster)
- **L3 Cache Access**: 5-50ms (persistent storage)
- **Data Aggregation**: Sub-100ms for multi-source queries
- **Stream Processing**: Real-time with microsecond latency
- **Memory Usage**: Efficient with mimalloc allocator

## 🚀 Quick Start

### Prerequisites

- Rust 1.75+ with stable toolchain
- Redis 6.0+ cluster for L2 caching
- PostgreSQL 13+ for L3 persistent storage
- Kafka 2.8+ for stream processing
- Protocol Buffers compiler (`protoc`)

### Installation

1. **Clone and enter directory**:
   ```bash
   cd backend/services/clinical-data-hub-rust
   ```

2. **Install Rust dependencies**:
   ```bash
   cargo fetch
   ```

3. **Build the service**:
   ```bash
   cargo build --release
   ```

### Running the Service

**Using default configuration**:
```bash
cargo run --release
```

**With custom configuration**:
```bash
cargo run --release -- \
  --grpc-port 8018 \
  --http-port 8118 \
  --redis-addrs "localhost:6379,localhost:6380" \
  --postgres-url "postgresql://user:pass@localhost:5432/clinical_data_hub" \
  --kafka-brokers "localhost:9092" \
  --environment production \
  --l1-cache-size-mb 1024
```

**Using environment variables**:
```bash
export GRPC_PORT=8018
export HTTP_PORT=8118
export REDIS_ADDRS=localhost:6379,localhost:6380
export POSTGRES_URL=postgresql://localhost:5432/clinical_data_hub
export KAFKA_BROKERS=localhost:9092
export ENVIRONMENT=production
export L1_CACHE_SIZE_MB=1024

cargo run --release
```

## 📡 API Endpoints

### gRPC Service (Port 8018)

The Clinical Data Hub implements ultra-high performance gRPC methods:

- `GetCachedData` - Retrieve data from multi-layer cache with freshness controls
- `SetCachedData` - Store data across cache layers with compression
- `InvalidateCache` - Invalidate cache entries with pattern matching
- `WarmCache` - Predictive cache warming for performance optimization
- `AggregateData` - Parallel data aggregation from multiple sources
- `BatchAggregate` - Batch aggregation with concurrency controls
- `StreamDataUpdates` - Real-time data streaming
- `ProcessDataStream` - Process incoming data streams
- `GetPerformanceMetrics` - Detailed performance analytics
- `OptimizeCache` - Dynamic cache optimization
- `GetServiceHealth` - Service and dependency health
- `RunDiagnostics` - Comprehensive system diagnostics

### HTTP Endpoints (Port 8118)

- `GET /health` - Service health check
- `GET /ready` - Readiness probe for Kubernetes
- `GET /metrics` - Prometheus-compatible performance metrics

## ⚡ Performance Optimization

### Multi-Layer Cache Architecture

```
L1 Cache (Memory)    →  < 1ms    →  512 MB
L2 Cache (Redis)     →  1-5ms    →  8 GB  
L3 Cache (Persistent) →  5-50ms   →  100 GB
```

### Data Compression

- **LZ4**: Ultra-fast compression for real-time data
- **Zstd**: High-ratio compression for archival data  
- **MessagePack**: Structured data serialization
- **Adaptive**: Automatic compression type selection

### Memory Management

- **mimalloc**: High-performance memory allocator
- **Zero-Copy**: Minimize data copying between layers
- **Memory Pools**: Pre-allocated buffers for hot paths
- **NUMA Awareness**: Optimize for multi-socket systems

## 🔧 Configuration

### Command Line Arguments

| Argument | Default | Description |
|----------|---------|-------------|
| `--grpc-port` | `8018` | gRPC server port |
| `--http-port` | `8118` | HTTP metrics port |
| `--redis-addrs` | `localhost:6379` | Redis cluster addresses |
| `--postgres-url` | `postgresql://...` | PostgreSQL connection |
| `--kafka-brokers` | `localhost:9092` | Kafka broker list |
| `--environment` | `development` | Runtime environment |
| `--l1-cache-size-mb` | `512` | L1 cache size in MB |
| `--enable-profiling` | `false` | Enable performance profiling |

### Environment Variables

All arguments can be set via environment variables:

- `GRPC_PORT` - gRPC server port
- `HTTP_PORT` - HTTP server port  
- `REDIS_ADDRS` - Comma-separated Redis addresses
- `POSTGRES_URL` - PostgreSQL connection string
- `KAFKA_BROKERS` - Comma-separated Kafka brokers
- `ENVIRONMENT` - Runtime environment
- `L1_CACHE_SIZE_MB` - L1 cache size
- `LOG_LEVEL` - Logging level (trace, debug, info, warn, error)

## 📊 Monitoring & Metrics

### Performance Metrics

- **Latency**: P50, P95, P99 response times per cache layer
- **Throughput**: Operations per second across all layers
- **Hit Ratios**: Cache hit rates for L1, L2, L3
- **Memory Usage**: Detailed memory allocation tracking
- **Compression**: Compression ratios and time costs
- **Stream Processing**: Event processing rates and lag

### Health Checks

- **Liveness**: `GET /health` - Service is running
- **Readiness**: `GET /ready` - Service ready for traffic
- **Dependencies**: Redis cluster, PostgreSQL, Kafka connectivity

### Observability

- **Tracing**: Structured logging with correlation IDs
- **Metrics**: Prometheus-compatible metrics endpoint
- **Profiling**: Optional runtime performance profiling
- **Diagnostics**: Comprehensive system diagnostics API

## 🧪 Testing & Benchmarking

### Unit Tests
```bash
cargo test
```

### Integration Tests
```bash
cargo test --test integration
```

### Benchmarks
```bash
cargo bench
```

### Load Testing
```bash
# Cache performance benchmark
cargo bench cache_performance

# Data aggregation benchmark  
cargo bench data_aggregation

# Memory usage benchmark
cargo bench --bench memory_usage
```

## 🏭 Production Deployment

### Docker Deployment

**Build Docker image**:
```bash
docker build -t clinical-data-hub-rust:latest .
```

**Run with Docker Compose**:
```yaml
version: '3.8'
services:
  clinical-data-hub:
    image: clinical-data-hub-rust:latest
    ports:
      - "8018:8018"
      - "8118:8118"
    environment:
      - ENVIRONMENT=production
      - REDIS_ADDRS=redis-cluster:6379
      - POSTGRES_URL=postgresql://postgres:5432/clinical_data_hub
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      - redis-cluster
      - postgres
      - kafka
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clinical-data-hub-rust
spec:
  replicas: 3
  selector:
    matchLabels:
      app: clinical-data-hub-rust
  template:
    metadata:
      labels:
        app: clinical-data-hub-rust
    spec:
      containers:
      - name: clinical-data-hub
        image: clinical-data-hub-rust:latest
        ports:
        - containerPort: 8018
        - containerPort: 8118
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
```

## 🔄 Integration with Go Context Gateway

### Service Communication

The Rust Clinical Data Hub works in conjunction with the Go Context Gateway:

```
Client Request → Go Context Gateway → Rust Clinical Data Hub
                      ↓                        ↓
               Recipe Management          Multi-Layer Cache
               Snapshot Creation         Data Aggregation
               Audit Logging            Stream Processing
```

### Performance Benefits

- **Go Service**: Recipe orchestration and snapshot management (8017)
- **Rust Service**: Ultra-fast caching and data aggregation (8018)
- **Combined**: Best of both languages for optimal performance

## 🛠️ Development

### Project Structure

```
clinical-data-hub-rust/
├── src/
│   ├── main.rs              # Service entry point
│   ├── models/              # Data models and types
│   │   ├── mod.rs           # Core clinical data types
│   │   └── cache.rs         # Cache-specific models
│   ├── cache/               # Multi-layer cache implementation
│   │   ├── mod.rs           # Cache trait and utilities
│   │   ├── l1_memory.rs     # L1 in-memory cache
│   │   ├── l2_redis.rs      # L2 Redis cluster cache
│   │   ├── l3_persistent.rs # L3 persistent storage
│   │   └── manager.rs       # Cache manager coordination
│   ├── services/            # Business logic services
│   └── proto/               # Generated Protocol Buffer code
├── proto/                   # Protocol Buffer definitions
├── benches/                 # Performance benchmarks
├── examples/                # Usage examples
├── docker/                  # Docker configuration
├── Cargo.toml              # Rust dependencies
├── build.rs                # Build configuration
└── README.md               # This file
```

### Adding New Features

1. **Add gRPC method**: Update `proto/clinical_data_hub.proto`
2. **Regenerate code**: `cargo build` (automatic via build.rs)
3. **Implement service**: Add logic in `src/services/`
4. **Add tests**: Create unit and integration tests
5. **Benchmark**: Add performance benchmarks
6. **Update documentation**: Update this README

## 🔒 Security

### Clinical Data Protection

- **Memory Safety**: Rust's ownership model prevents memory vulnerabilities
- **Data Encryption**: Encryption at rest and in transit
- **Secure Allocator**: mimalloc with security hardening
- **Audit Trail**: All operations logged for compliance
- **Access Control**: Service-level authentication
- **Data Integrity**: Checksums and validation throughout

### Security Best Practices

- No unsafe Rust code in critical paths
- Secure credential management
- Regular dependency updates
- Minimal attack surface
- Comprehensive error handling

## ⚡ Performance Characteristics

### Latency Targets

- **Cache Hit (L1)**: < 100 microseconds
- **Cache Hit (L2)**: < 5 milliseconds  
- **Cache Hit (L3)**: < 50 milliseconds
- **Data Aggregation**: < 100 milliseconds
- **Stream Processing**: < 1 millisecond

### Throughput Targets

- **Cache Operations**: > 1M ops/second (L1)
- **Cache Operations**: > 100K ops/second (L2)
- **Data Aggregation**: > 10K requests/second
- **Stream Processing**: > 1M events/second

### Resource Usage

- **Memory**: Configurable L1 cache size + overhead
- **CPU**: Multi-threaded with work-stealing scheduler
- **Network**: Efficient connection pooling
- **Storage**: Compressed data with optimal layouts

---

**Clinical Data Hub Rust Service** - Ultra-High Performance Clinical Data Intelligence
*Part of the Clinical Synthesis Hub CardioFit Platform*