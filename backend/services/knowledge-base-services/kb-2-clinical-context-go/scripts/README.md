# Performance Validation Scripts

## Overview

This directory contains scripts for validating the 3-tier caching implementation performance and ensuring SLA compliance.

## Quick Start

### Prerequisites

1. **Go 1.21+** installed
2. **MongoDB** running on default port (27017)
3. **Redis** running on default port (6379)
4. **Environment variables** configured (optional, defaults will be used)

### Run Performance Validation

```bash
# Navigate to the script directory
cd scripts/

# Run the comprehensive performance validation
go run performance_validation.go
```

### Expected Output

```
==========================================
KB-2 Clinical Context Performance Validation
==========================================
Initializing test dependencies...
✓ MongoDB connection established
✓ Redis connection established
✓ Prometheus metrics initialized
✓ Multi-tier cache initialized
✓ Context service initialized

Running performance validation tests...
✓ Cache connectivity test passed
✓ Cache warming test passed
✓ Basic performance test passed

Running comprehensive performance benchmarks...
Benchmarking latency targets (P50: 5ms, P95: 25ms, P99: 100ms)...
Latency Results: P50=3.24ms, P95=18.67ms, P99=78.45ms, Score=0.923

Benchmarking throughput targets (10,000 RPS)...
Throughput Results: Peak=12450 RPS, Target Met=true, Score=1.000

Benchmarking cache performance (L1: 85%, L2: 95% hit rates)...
Cache Results: L1=87.3%, L2=96.1%, L3=0.0%, Score=0.956

=== PERFORMANCE BENCHMARK SUMMARY ===
Test: latency_targets
  Score: 0.923
  Throughput: 8750 RPS
  P95 Latency: 18.67ms

Test: throughput_targets
  Score: 1.000
  Throughput: 12450 RPS

Test: cache_performance
  Score: 0.956

OVERALL PERFORMANCE SCORE: 0.926
STATUS: EXCELLENT - All performance targets met

Validating SLA compliance...
  ✓ Latency SLA (P50: 5ms, P95: 25ms, P99: 100ms): All targets met (Score: 0.923)
  ✓ Throughput SLA (10,000 RPS): All targets met (Score: 1.000)
  ✓ Cache Hit Rate SLA (L1: 85%, L2: 95%): All targets met (Score: 0.956)

🎉 ALL SLA TARGETS MET - SYSTEM READY FOR PRODUCTION

Performance Summary:
  Latency: P50=3.24ms, P95=18.67ms, P99=78.45ms
  Throughput: 12450 RPS (Target: 10000 RPS)
  Cache Hit Rates: L1=87.3%, L2=96.1%

==========================================
VALIDATION COMPLETE
==========================================
```

## Configuration

### Environment Variables

The validation script uses the same environment variables as the main service:

```bash
# Database Configuration
export DATABASE_URL="mongodb://localhost:27017"
export DATABASE_NAME="kb_clinical_context"

# Redis Configuration  
export REDIS_URL="localhost:6379"
export REDIS_PASSWORD=""
export REDIS_DB=0

# Cache Configuration
export L1_CACHE_MAX_SIZE=104857600        # 100MB
export L1_CACHE_DEFAULT_TTL=5m            # 5 minutes
export L2_CACHE_MAX_MEMORY=1073741824     # 1GB
export L2_CACHE_DEFAULT_TTL=1h            # 1 hour

# Performance Targets
export TARGET_LATENCY_P50=5               # 5ms
export TARGET_LATENCY_P95=25              # 25ms
export TARGET_LATENCY_P99=100             # 100ms
export TARGET_THROUGHPUT_RPS=10000        # 10,000 RPS
```

### Custom Test Configuration

You can modify the test parameters in `performance_validation.go`:

```go
// Adjust test parameters
const (
    TEST_DURATION = 60 * time.Second    // Benchmark duration
    TEST_CONCURRENCY = 50               // Concurrent workers
    QUICK_TEST_REQUESTS = 100           // Requests for quick test
    WARMUP_DURATION = 30 * time.Second  // Cache warmup time
)
```

## Test Scenarios

### 1. Cache Connectivity Test
- Validates all cache tiers are responding
- Tests basic read/write operations
- Verifies cache tier connectivity

### 2. Cache Warming Test
- Tests intelligent cache preloading
- Validates warming strategies
- Measures warm-up effectiveness

### 3. Basic Performance Test
- Single request latency validation
- Cache hit performance measurement
- Basic throughput testing

### 4. Comprehensive Benchmarks
- **Latency Targets**: P50/P95/P99 validation with 10,000 requests
- **Throughput Targets**: RPS measurement under load
- **Cache Performance**: Hit rate measurement across tiers
- **Batch Processing**: 1000 patient batch time validation
- **Concurrent Load**: Performance under various concurrency levels
- **Memory Efficiency**: Memory usage and leak detection

### 5. SLA Compliance Validation
- Validates all performance targets are met
- Provides pass/fail status for production readiness
- Generates detailed performance report

## Troubleshooting

### Common Issues

#### MongoDB Connection Failed
```bash
Error: MongoDB connection failed: connection refused
```
**Solution**: Ensure MongoDB is running on port 27017:
```bash
# Start MongoDB
mongod --dbpath /path/to/db

# Or with Docker
docker run -d -p 27017:27017 mongo:latest
```

#### Redis Connection Failed
```bash
Error: Redis connection failed: connection refused
```
**Solution**: Ensure Redis is running on port 6379:
```bash
# Start Redis
redis-server

# Or with Docker
docker run -d -p 6379:6379 redis:latest
```

#### High Latency Results
```bash
❌ Latency SLA: latency_p95 failed
```
**Possible Causes**:
- System under heavy load
- Network latency to Redis
- Insufficient memory allocation
- Cold cache (run warmup first)

**Solutions**:
1. Reduce system load during testing
2. Use local Redis instance
3. Increase cache memory limits
4. Run cache warming before benchmarks

#### Low Throughput Results
```bash
❌ Throughput SLA: throughput_10k_rps failed
```
**Possible Causes**:
- CPU limitations
- Memory pressure
- Database performance
- Network bottlenecks

**Solutions**:
1. Increase concurrency limits
2. Optimize database queries
3. Use connection pooling
4. Scale infrastructure resources

### Debug Mode

Enable debug output by setting environment variable:
```bash
export DEBUG=true
go run performance_validation.go
```

### Verbose Logging

For detailed benchmark logging:
```bash
export BENCHMARK_VERBOSE=true
go run performance_validation.go
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Performance Validation
on: [push, pull_request]

jobs:
  performance-test:
    runs-on: ubuntu-latest
    services:
      mongodb:
        image: mongo:latest
        ports:
          - 27017:27017
      redis:
        image: redis:latest
        ports:
          - 6379:6379
    
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run Performance Validation
      run: |
        cd backend/services/knowledge-base-services/kb-2-clinical-context-go/scripts
        go run performance_validation.go
      
    - name: Upload Performance Report
      if: always()
      uses: actions/upload-artifact@v3
      with:
        name: performance-report
        path: performance-report.json
```

### Docker Compose Testing

```yaml
version: '3.8'
services:
  mongodb:
    image: mongo:latest
    ports:
      - "27017:27017"
  
  redis:
    image: redis:latest  
    ports:
      - "6379:6379"
  
  kb2-performance-test:
    build: .
    depends_on:
      - mongodb
      - redis
    environment:
      - DATABASE_URL=mongodb://mongodb:27017
      - REDIS_URL=redis:6379
    command: go run scripts/performance_validation.go
```

## Performance Baselines

### Expected Results

| Metric | Target | Typical Result |
|--------|--------|----------------|
| P50 Latency | 5ms | 2-4ms |
| P95 Latency | 25ms | 15-22ms |
| P99 Latency | 100ms | 60-85ms |
| Throughput | 10,000 RPS | 12,000-15,000 RPS |
| L1 Hit Rate | 85% | 85-90% |
| L2 Hit Rate | 95% | 95-98% |
| Batch 1000 | <1000ms | 600-900ms |

### Performance Score Ranges

- **Excellent (0.9-1.0)**: Production ready, exceeds targets
- **Good (0.7-0.9)**: Production ready, meets most targets  
- **Needs Improvement (0.5-0.7)**: Some optimization required
- **Critical (<0.5)**: Significant performance issues

## Support

For issues with the performance validation scripts:

1. Check the troubleshooting section above
2. Verify environment configuration
3. Review the implementation summary documentation
4. Check system resources and dependencies