# Snapshot Integration Guide

## Phase 1 Implementation Complete

This document describes how to enable and configure the newly implemented snapshot functionality in the Safety Gateway Platform.

## 🎯 Implementation Status

✅ **COMPLETED COMPONENTS:**
- Comprehensive snapshot type definitions (`pkg/types/snapshot.go`)
- HTTP Context Gateway client (`internal/clients/context_gateway_client.go`) 
- Snapshot validator with cryptographic verification (`internal/snapshot/validator.go`)
- Snapshot-based orchestration engine (`internal/orchestration/snapshot_orchestration.go`)
- Multi-level snapshot cache with Redis support (`internal/cache/snapshot_cache.go`)
- Complete configuration system (`internal/config/snapshot_config.go`)

## 🔧 Configuration Setup

### 1. Context Service Setup

The Context Service (Python FastAPI) must be running at:
```
http://localhost:8020
```

Key endpoints:
- `GET /health` - Health check
- `POST /api/snapshots` - Create snapshot
- `GET /api/snapshots/{snapshot_id}` - Get snapshot
- `POST /api/snapshots/{snapshot_id}/validate` - Validate snapshot

### 2. Safety Gateway Configuration

Add to your `config.yaml`:

```yaml
# Snapshot processing configuration
snapshot:
  enabled: true                          # Enable snapshot mode
  request_timeout: 5s                    # Request timeout
  engine_execution_timeout: 3s           # Engine execution timeout  
  min_data_completeness: 60.0            # Minimum data completeness (60%)
  
  # Cache settings
  cache_min_ttl: 1m                      # Minimum cache TTL
  cache_max_ttl: 30m                     # Maximum cache TTL
  enable_pre_warming: false              # Disable pre-warming initially
  
  # Fallback settings
  allow_fallback_to_legacy: true         # Allow fallback during transition
  max_retries: 3                         # Max retry attempts
  retry_backoff: 100ms                   # Retry backoff duration
  
  # Validation settings
  strict_validation: true                # Enable strict validation
  require_signature: false               # Optional signatures
  signing_key_path: ""                   # Path to signing key

# Context Gateway client configuration  
context_gateway:
  endpoint: "localhost:8020"             # Context Service endpoint
  timeout: 3s                            # Client timeout
  max_retries: 3                         # Max retry attempts
  service_name: "safety-gateway-platform" # Service identifier
  enable_tls: false                      # Disable TLS for local dev
  health_check: true                     # Enable health checks
  
  # Connection pooling
  max_connections: 10                    # Max concurrent connections
  connection_timeout: 5s                 # Connection establishment timeout
  keep_alive_timeout: 30s               # Keep-alive timeout

# Cache configuration
cache:
  # L1 Cache (In-Memory)
  l1_max_size: 1000                     # Max entries in L1 cache
  l1_ttl: 5m                           # L1 cache TTL
  l1_cleanup_interval: 1m              # L1 cleanup frequency
  
  # L2 Cache (Redis) 
  enable_l2_cache: true                # Enable Redis L2 cache
  l2_ttl: 30m                         # L2 cache TTL
  redis:
    address: "localhost:6379"          # Redis address
    password: ""                       # Redis password (empty for no auth)
    db: 0                             # Redis database number
    pool_size: 10                     # Connection pool size
    
  # Cache behavior
  write_through: true                   # Enable write-through caching
  write_back: false                    # Disable write-back caching  
  enable_compression: false            # Disable compression initially
```

## 🚀 Server Integration

### Option 1: Enable in Existing Server (Recommended)

Update `internal/server/server.go` to conditionally create snapshot orchestrator:

```go
// In the New() function, replace the orchestrator creation:

var orchestrator types.SafetyOrchestrator

if cfg.Snapshot != nil && cfg.Snapshot.Enabled {
    // Create snapshot-enhanced orchestrator
    logger.Info("Initializing snapshot-based orchestration engine")
    
    // Create snapshot validator
    snapshotValidator := snapshot.NewValidator([]byte{}, logger) // Use empty key for now
    
    // Create Context Gateway client
    contextClient, err := clients.NewContextGatewayClient(cfg.ContextGateway, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create Context Gateway client: %w", err)
    }
    
    // Create snapshot cache
    snapshotCache, err := cache.NewSnapshotCache(cfg.Cache, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create snapshot cache: %w", err)
    }
    
    // Create base orchestration engine first
    baseOrchestrator := orchestration.NewOrchestrationEngine(
        engineRegistry,
        contextService,
        cfg,
        logger,
    )
    
    // Create snapshot orchestration engine
    orchestrator = orchestration.NewSnapshotOrchestrationEngine(
        baseOrchestrator,
        snapshotValidator,
        contextClient,
        snapshotCache,
        cfg.Snapshot,
        logger,
    )
} else {
    // Use legacy orchestration engine
    logger.Info("Using legacy orchestration engine")
    orchestrator = orchestration.NewOrchestrationEngine(
        engineRegistry,
        contextService,
        cfg,
        logger,
    )
}
```

### Option 2: Create New Snapshot Server

Create `cmd/snapshot-server/main.go` for dedicated snapshot-enabled server.

## 📋 Usage Examples

### Creating a Snapshot Request

```go
request := &types.SafetyRequest{
    RequestID:     "req-001",
    PatientID:     "patient-123", 
    ActionType:    "medication_check",
    Priority:      "high",
    MedicationIDs: []string{"med-001", "med-002"},
    Context: map[string]string{
        "snapshot_id": "snap-456", // Enable snapshot mode
    },
}

response, err := orchestrator.ProcessSafetyRequest(ctx, request)
```

### Checking Snapshot Stats

```go
if snapshotOrchestrator, ok := orchestrator.(*orchestration.SnapshotOrchestrationEngine); ok {
    stats := snapshotOrchestrator.GetSnapshotStats()
    fmt.Printf("Snapshot mode: %v\n", stats["snapshot_mode_enabled"])
    fmt.Printf("Cache stats: %v\n", stats["cache_stats"])
}
```

## 🔍 Monitoring & Observability

Key metrics to monitor:
- **Cache Hit Ratio**: L1 and L2 cache performance
- **Snapshot Validation Time**: Cryptographic validation performance  
- **Context Gateway Response Time**: HTTP client performance
- **Fallback Rate**: Frequency of legacy mode fallbacks
- **Error Rates**: Circuit breaker and validation failures

## 🧪 Testing

### Integration Tests

1. **Context Service Health**: Verify Context Service is accessible
2. **Snapshot Creation**: Test snapshot creation via Context Service
3. **Cache Performance**: Validate multi-level caching behavior
4. **Validation**: Test cryptographic validation pipeline
5. **Fallback**: Ensure graceful fallback to legacy mode

### Performance Tests

Expected improvements:
- **Latency**: Target <200ms (85% improvement from 3.5s baseline)
- **Throughput**: 10x improvement in concurrent request handling
- **Cache Hit Ratio**: >90% for repeated patient contexts

## 🔄 Migration Strategy

1. **Phase 1**: Deploy with `snapshot.enabled: false` (current)
2. **Phase 2**: Enable on development environment
3. **Phase 3**: Gradual rollout with A/B testing
4. **Phase 4**: Full production deployment

## 🚨 Troubleshooting

### Common Issues

1. **Context Service Unreachable**:
   - Check Context Service is running on port 8020
   - Verify network connectivity
   - Review circuit breaker status

2. **Cache Misses**:
   - Check Redis connectivity
   - Verify TTL configurations
   - Monitor memory usage

3. **Validation Failures**:
   - Check snapshot integrity
   - Verify signing key configuration
   - Review timestamp validity

### Debug Commands

```bash
# Check Context Service health
curl http://localhost:8020/health

# Check Redis connectivity
redis-cli ping

# Review Safety Gateway logs for snapshot operations
grep "snapshot" /var/log/safety-gateway-platform.log
```

## 📈 Next Steps (Phase 2+)

- **Performance Optimization**: Implement advanced caching strategies
- **Enhanced Features**: Add snapshot pre-warming, batch operations
- **Monitoring**: Add comprehensive metrics and alerting
- **Security**: Implement cryptographic signatures for production

---

**Implementation Status**: ✅ Phase 1 Complete - Ready for Configuration and Testing