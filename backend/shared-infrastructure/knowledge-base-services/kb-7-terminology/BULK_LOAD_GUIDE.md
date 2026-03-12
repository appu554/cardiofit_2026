# KB7 Terminology Bulk Load Guide

**Version**: 1.0
**Date**: September 22, 2025
**Status**: Production Ready

## 🎯 Overview

The KB7 Terminology Bulk Load System provides comprehensive capabilities for migrating clinical terminology data from PostgreSQL to Elasticsearch with full data integrity validation, multiple migration strategies, and production-grade reliability features.

## 🏗️ Architecture

### System Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   PostgreSQL    │────│  Bulk Loader     │────│  Elasticsearch  │
│   (Source)      │    │  (Migration      │    │  (Target)       │
│                 │    │   Engine)        │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │ Data Integrity   │
                       │ Validator        │
                       └──────────────────┘
```

### Core Features

- **Multi-Strategy Migration**: Incremental, parallel, blue-green, shadow write
- **Fault Tolerance**: Circuit breakers, retry logic, checkpoint/resume
- **Data Validation**: Comprehensive integrity checks and performance validation
- **Monitoring**: Real-time progress tracking and performance metrics
- **Clinical Optimization**: Specialized for medical terminology data

## 🚀 Quick Start

### Prerequisites

1. **Go 1.21+** installed
2. **PostgreSQL** with clinical terminology data
3. **Elasticsearch 8.x** cluster running
4. **Required tools**: `jq`, `curl` for monitoring scripts

### Basic Migration

```bash
# 1. Build the bulk loader
go build -o bulkload ./cmd/bulkload

# 2. Run a basic parallel migration
./scripts/execute-bulk-load.sh parallel development

# 3. Monitor progress (in another terminal)
./scripts/monitor-bulk-load.sh --dashboard
```

## 📋 Migration Strategies

### 1. Incremental Strategy
**Best for**: Small datasets, development environments, conservative approach

```bash
./scripts/execute-bulk-load.sh incremental development \
  --batch-size 500 \
  --workers 1 \
  --validate
```

**Characteristics**:
- ✅ Single worker, sequential processing
- ✅ Minimal resource usage
- ✅ Easy to debug and monitor
- ❌ Slower for large datasets

### 2. Parallel Strategy (Recommended)
**Best for**: Production environments, large datasets, optimal performance

```bash
./scripts/execute-bulk-load.sh parallel production \
  --batch-size 1000 \
  --workers 4 \
  --validate
```

**Characteristics**:
- ✅ High throughput with multiple workers
- ✅ Optimized for performance
- ✅ Configurable parallelism
- ❌ Higher resource usage

### 3. Blue-Green Strategy
**Best for**: Zero-downtime requirements, production systems

```bash
./scripts/execute-bulk-load.sh blue-green production \
  --batch-size 1000 \
  --workers 3
```

**Characteristics**:
- ✅ Zero downtime migration
- ✅ Atomic switchover
- ✅ Easy rollback capability
- ❌ Requires 2x storage space temporarily

### 4. Shadow Strategy
**Best for**: Gradual migration, risk-averse environments

```bash
./scripts/execute-bulk-load.sh shadow production \
  --batch-size 500 \
  --workers 2
```

**Characteristics**:
- ✅ Gradual traffic shifting
- ✅ Continuous validation
- ✅ Risk mitigation
- ❌ Longer migration timeline

## ⚙️ Configuration

### Environment Variables

```bash
# Required for production
export POSTGRES_URL="postgres://user:pass@host:5432/kb7_terminology"
export ELASTICSEARCH_URL="http://elasticsearch:9200"
export ELASTICSEARCH_INDEX="clinical_terms"

# Optional performance tuning
export BATCH_SIZE=1000
export NUM_WORKERS=4
export FLUSH_INTERVAL=5s
export MAX_RETRIES=3
```

### Configuration Files

Create environment-specific config files in `config/` directory:

**config/production.json**:
```json
{
  "postgres_url": "postgres://user:pass@prod-db:5432/kb7_terminology",
  "elasticsearch_url": "https://prod-es:9200",
  "elasticsearch_index": "clinical_terms",
  "batch_size": 2000,
  "num_workers": 6,
  "flush_interval": "3s",
  "max_retries": 5,
  "validate_data": true
}
```

## 🔧 Command Line Reference

### Basic Usage

```bash
./bulkload [OPTIONS]
```

### Key Options

| Option | Description | Default |
|--------|-------------|---------|
| `--postgres` | PostgreSQL connection string | Required |
| `--elasticsearch` | Elasticsearch URL | Required |
| `--index` | Target index name | `clinical_terms` |
| `--strategy` | Migration strategy | `parallel` |
| `--batch` | Batch size for bulk operations | `1000` |
| `--workers` | Number of parallel workers | `4` |
| `--systems` | Comma-separated list of systems | All systems |
| `--resume` | Resume from specific record ID | `0` |
| `--validate` | Perform data validation | `true` |
| `--dry-run` | Perform dry run without migration | `false` |

### Advanced Options

| Option | Description | Use Case |
|--------|-------------|----------|
| `--config` | Configuration file path | Complex environments |
| `--output` | Migration report file | Audit trail |
| `--checkpoint` | Checkpoint file for resume | Recovery scenarios |
| `--log-level` | Logging verbosity | Debugging |
| `--progress` | Progress reporting interval | Monitoring |

## 📊 Monitoring and Observability

### Real-time Monitoring

```bash
# Basic monitoring
./scripts/monitor-bulk-load.sh

# Comprehensive dashboard
./scripts/monitor-bulk-load.sh --dashboard --alerts

# Export metrics
./scripts/monitor-bulk-load.sh --export metrics.jsonl
```

### Key Metrics

- **Throughput**: Records processed per second
- **Progress**: Processed vs. total records
- **Error Rate**: Failed records percentage
- **Index Health**: Elasticsearch cluster status
- **Resource Usage**: Memory and CPU utilization

### Log Analysis

```bash
# Find latest log
ls -la logs/bulk-load-*.log | tail -1

# Monitor errors in real-time
tail -f logs/bulk-load-*.log | grep ERROR

# Extract performance metrics
grep "Records/Second" logs/bulk-load-*.log
```

## 🛡️ Data Integrity and Validation

### Validation Components

1. **Record Count Validation**: Ensures all records migrated
2. **Checksum Validation**: Verifies data integrity
3. **Sample Comparison**: Deep validation of random samples
4. **Performance Validation**: Confirms search functionality
5. **Schema Validation**: Ensures proper field mapping

### Validation Reports

```bash
# Enable comprehensive validation
./bulkload --validate --output validation-report.json

# View validation results
jq '.validation_results' validation-report.json
```

### Sample Validation Output

```json
{
  "validation_results": {
    "record_count_match": true,
    "checksum_validation": true,
    "sample_comparison": {
      "samples_tested": 100,
      "matches": 100,
      "success_rate": 100.0
    },
    "performance_validation": {
      "search_latency_ms": 45,
      "aggregation_latency_ms": 120,
      "meets_requirements": true
    }
  }
}
```

## 🚨 Error Handling and Recovery

### Common Issues and Solutions

#### 1. Connection Failures

**Problem**: Cannot connect to PostgreSQL/Elasticsearch
```
ERROR Failed to connect to PostgreSQL: connection refused
```

**Solution**:
```bash
# Test connectivity
curl -s $ELASTICSEARCH_URL/_cluster/health
psql $POSTGRES_URL -c "SELECT 1"

# Check firewall/network configuration
```

#### 2. Memory Issues

**Problem**: Out of memory during large batch processing
```
ERROR Failed to process batch: out of memory
```

**Solution**:
```bash
# Reduce batch size and workers
./bulkload --batch 500 --workers 2

# Monitor memory usage
free -h
```

#### 3. Index Mapping Conflicts

**Problem**: Elasticsearch mapping conflicts
```
ERROR Failed to index document: mapping conflict
```

**Solution**:
```bash
# Delete and recreate index
curl -X DELETE $ELASTICSEARCH_URL/clinical_terms
./bulkload --strategy parallel
```

### Resume from Failure

```bash
# Migration stopped at record ID 150000
./bulkload --resume 150000 --strategy parallel

# Or use checkpoint file
./bulkload --checkpoint checkpoints/migration_20231201_153045.json
```

## 🔒 Security and Compliance

### Authentication

```bash
# PostgreSQL with SSL
export POSTGRES_URL="postgres://user:pass@host:5432/db?sslmode=require"

# Elasticsearch with authentication
export ELASTICSEARCH_URL="https://user:pass@elasticsearch:9200"
```

### Data Privacy

- All connections use SSL/TLS encryption
- Sensitive data is not logged
- Audit trails are maintained
- Access controls are enforced

### HIPAA Compliance

- Data encryption in transit and at rest
- Audit logging for all operations
- Access control and authentication
- Data retention policies

## 📈 Performance Tuning

### PostgreSQL Optimization

```sql
-- Increase shared_buffers for read performance
ALTER SYSTEM SET shared_buffers = '256MB';

-- Optimize for read-heavy workload
ALTER SYSTEM SET effective_cache_size = '1GB';

-- Reload configuration
SELECT pg_reload_conf();
```

### Elasticsearch Optimization

```bash
# Optimize for bulk indexing
curl -X PUT "$ELASTICSEARCH_URL/_cluster/settings" -H "Content-Type: application/json" -d '{
  "persistent": {
    "indices.store.throttle.max_bytes_per_sec": "100mb",
    "cluster.routing.allocation.node_concurrent_recoveries": 2
  }
}'

# Increase refresh interval during migration
curl -X PUT "$ELASTICSEARCH_URL/clinical_terms/_settings" -H "Content-Type: application/json" -d '{
  "refresh_interval": "30s"
}'
```

### Performance Baselines

| Environment | Records/sec | Batch Size | Workers |
|-------------|-------------|------------|---------|
| Development | 100-500 | 500 | 2 |
| Staging | 500-1500 | 1000 | 4 |
| Production | 1500-3000 | 2000 | 6-8 |

## 🧪 Testing

### Integration Testing

```bash
# Run comprehensive test suite
go run test-bulk-load-integration.go

# Test specific strategy
go run test-bulk-load-integration.go --strategy incremental

# Performance testing
go run test-bulk-load-integration.go --performance
```

### Load Testing

```bash
# Create test data
./scripts/generate-test-data.sh --records 100000

# Test with high load
./bulkload --batch 5000 --workers 8 --dry-run
```

## 📚 Best Practices

### Pre-Migration

1. **✅ Backup Data**: Always backup before migration
2. **✅ Test Environment**: Validate on staging first
3. **✅ Resource Planning**: Ensure adequate disk space and memory
4. **✅ Connectivity**: Verify network connectivity and permissions
5. **✅ Index Templates**: Set up proper Elasticsearch mappings

### During Migration

1. **✅ Monitor Progress**: Use real-time monitoring
2. **✅ Watch Resources**: Monitor CPU, memory, and disk usage
3. **✅ Error Tracking**: Watch for error patterns
4. **✅ Performance**: Adjust batch size if needed
5. **✅ Alerts**: Set up alerts for critical issues

### Post-Migration

1. **✅ Validation**: Run comprehensive data validation
2. **✅ Performance Testing**: Verify search performance
3. **✅ Cleanup**: Remove temporary files and indices
4. **✅ Documentation**: Update system documentation
5. **✅ Monitoring**: Set up ongoing monitoring

## 🔄 Maintenance

### Regular Operations

```bash
# Health check
curl -s $ELASTICSEARCH_URL/_cluster/health | jq

# Index statistics
curl -s $ELASTICSEARCH_URL/clinical_terms/_stats | jq '.indices[].total'

# Clean old logs (keep last 10)
find logs/ -name "bulk-load-*.log" | sort -r | tail -n +11 | xargs rm
```

### Index Optimization

```bash
# Force merge for read optimization
curl -X POST "$ELASTICSEARCH_URL/clinical_terms/_forcemerge?max_num_segments=1"

# Update index settings post-migration
curl -X PUT "$ELASTICSEARCH_URL/clinical_terms/_settings" -H "Content-Type: application/json" -d '{
  "refresh_interval": "1s",
  "number_of_replicas": 1
}'
```

## 📞 Troubleshooting

### Diagnostic Commands

```bash
# Check system resources
free -h
df -h
top -p $(pgrep bulkload)

# Elasticsearch diagnostics
curl -s $ELASTICSEARCH_URL/_cat/indices?v
curl -s $ELASTICSEARCH_URL/_cat/nodes?v

# PostgreSQL diagnostics
psql $POSTGRES_URL -c "SELECT count(*) FROM clinical_terms"
```

### Emergency Procedures

#### Stop Migration
```bash
# Graceful stop (Ctrl+C or SIGTERM)
pkill -TERM bulkload

# Force stop (if needed)
pkill -KILL bulkload
```

#### Rollback
```bash
# Delete target index
curl -X DELETE $ELASTICSEARCH_URL/clinical_terms

# Restore from backup (if available)
./scripts/restore-from-backup.sh backup-20231201.json
```

## 📋 Checklist

### Pre-Migration Checklist

- [ ] PostgreSQL connection tested
- [ ] Elasticsearch cluster healthy
- [ ] Backup created
- [ ] Sufficient disk space available
- [ ] Test migration completed
- [ ] Monitoring tools ready
- [ ] Team notified

### Post-Migration Checklist

- [ ] Data validation passed
- [ ] Search functionality tested
- [ ] Performance benchmarks met
- [ ] Monitoring configured
- [ ] Documentation updated
- [ ] Cleanup completed
- [ ] Success communicated

## 🆘 Support

### Getting Help

1. **📖 Documentation**: This guide and inline help
2. **🐛 Issues**: Check common issues section
3. **📊 Logs**: Review migration logs for errors
4. **🔍 Monitoring**: Use monitoring dashboard for insights

### Contact Information

- **Technical Issues**: Check logs and error messages
- **Performance Issues**: Review monitoring metrics
- **Configuration Help**: See configuration examples
- **Emergency**: Follow emergency procedures

---

**Note**: This guide represents the production-ready bulk load system for KB7 Terminology Service. Always test in a non-production environment before running production migrations.