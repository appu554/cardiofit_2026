# KB-7 Kernel Deployment & Monitoring Guide

**Last Updated**: November 24, 2025
**Phase**: 1.3.4 - Deployment Automation & Monitoring
**Status**: Production Ready

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Deployment Workflow](#deployment-workflow)
4. [Rollback Procedures](#rollback-procedures)
5. [Monitoring Setup](#monitoring-setup)
6. [Troubleshooting](#troubleshooting)
7. [Operational Runbooks](#operational-runbooks)

---

## Overview

The KB-7 deployment system automates the process of deploying new kernel versions from S3 to GraphDB production, with comprehensive validation, monitoring, and rollback capabilities.

### Architecture

```
AWS S3 (Kernels)
    ↓
Download & Validate
    ↓
GraphDB Test Repository
    ↓
5 SPARQL Quality Gates
    ↓
Swap to Production Repository
    ↓
Update PostgreSQL Metadata
    ↓
Clear Redis Cache
    ↓
Notify Team (Slack)
```

### Key Components

| Component | Purpose | Location |
|-----------|---------|----------|
| **deploy-kernel.sh** | Deploy new kernel version | `scripts/deploy-kernel.sh` |
| **rollback-kernel.sh** | Rollback to previous version | `scripts/rollback-kernel.sh` |
| **health-check.sh** | System health validation | `scripts/health-check.sh` |
| **notify-slack.sh** | Slack notifications | `scripts/notify-slack.sh` |
| **Grafana Dashboard** | Visual monitoring | `monitoring/grafana/kb7-dashboard.json` |
| **Prometheus Metrics** | Metrics collection | `monitoring/prometheus/kb7-metrics.yml` |

---

## Prerequisites

### Required Tools

```bash
# Verify all dependencies are installed
curl --version        # >= 7.68
jq --version          # >= 1.6
psql --version        # >= 13
redis-cli --version   # >= 6.0
aws --version         # >= 2.0
```

### Environment Variables

Create `.env.deployment`:

```bash
# GraphDB Configuration
GRAPHDB_ENDPOINT=http://localhost:7200
GRAPHDB_TEST_REPO=kb7-test
GRAPHDB_PROD_REPO=kb7-terminology

# AWS Configuration
S3_BUCKET=cardiofit-kb-artifacts
AWS_REGION=us-east-1
AWS_PROFILE=kb7-deployer

# PostgreSQL Configuration
PG_URL=postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology

# Redis Configuration
REDIS_URL=redis://localhost:6380/0

# Slack Configuration (optional)
SLACK_WEBHOOK=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Deployment Settings
ENVIRONMENT=production
LOG_DIR=/var/log/kb7
```

Load environment:
```bash
source .env.deployment
```

### Permissions

```bash
# Create log directory
sudo mkdir -p /var/log/kb7
sudo chown $USER:$USER /var/log/kb7

# Make scripts executable
chmod +x scripts/deploy-kernel.sh
chmod +x scripts/rollback-kernel.sh
chmod +x scripts/health-check.sh
chmod +x scripts/notify-slack.sh
```

---

## Deployment Workflow

### Step-by-Step Deployment

#### 1. Pre-Deployment Health Check

```bash
# Verify system health before deployment
./scripts/health-check.sh --verbose

# Expected output:
# === GraphDB Health Check ===
# ✓ GraphDB endpoint accessible
# ✓ Repository 'kb7-terminology' exists
# ✓ Triple count: 8,245,731
# ✓ Concept count: 523,451
# ✓ Query latency: 42ms
#
# === PostgreSQL Health Check ===
# ✓ PostgreSQL connection successful
# ✓ Metadata table 'kb7_snapshots' exists
# ✓ Active snapshot: 20250101
#
# === Redis Health Check ===
# ✓ Redis connection successful
# ✓ Redis read/write operations working
#
# Overall Health: PASS
```

If any checks fail, investigate before proceeding.

#### 2. Review Kernel Manifest

```bash
# Download and inspect manifest from S3
aws s3 cp s3://cardiofit-kb-artifacts/20250124/kb7-manifest.json - | jq .

# Example output:
{
  "version": "20250124",
  "build_date": "2025-01-24T02:45:33Z",
  "concept_count": 523451,
  "triple_count": 8245731,
  "terminologies": {
    "SNOMED": {
      "version": "20241231",
      "concept_count": 352874
    },
    "RxNorm": {
      "version": "20250104",
      "concept_count": 123456
    },
    "LOINC": {
      "version": "2.77",
      "concept_count": 97121
    }
  },
  "validation_results": {
    "concept_count": "PASS",
    "orphaned_concepts": "PASS",
    "snomed_roots": "PASS",
    "rxnorm_drugs": "PASS",
    "loinc_codes": "PASS"
  }
}
```

#### 3. Dry Run Deployment

```bash
# Test deployment without making changes
./scripts/deploy-kernel.sh 20250124 --dry-run

# Expected output:
# ==========================================
# KB-7 Kernel Deployment Started
# Version: 20250124
# Dry Run: true
# ==========================================
# Step 1/6: Downloading kernel from S3...
# DRY RUN: Skipping S3 download
# Step 2/6: Loading kernel to test repository...
# DRY RUN: Skipping GraphDB load
# Step 3/6: Running validation queries...
# DRY RUN: Skipping validations
# Step 4/6: Swapping test repository to production...
# DRY RUN: Skipping repository swap
# Step 5/6: Updating metadata registry...
# DRY RUN: Skipping metadata update
# Step 6/6: Clearing Redis cache...
# DRY RUN: Skipping cache clear
# ==========================================
# KB-7 Kernel Deployment Complete
# Duration: 3s
# ==========================================
```

#### 4. Production Deployment

```bash
# Deploy kernel to production
./scripts/deploy-kernel.sh 20250124

# Monitor progress in real-time
tail -f /var/log/kb7/deploy-kernel-$(date +%Y%m%d)*.log
```

**Expected Timeline**:
- Step 1 (Download): 30-60 seconds
- Step 2 (Load to test): 120-180 seconds
- Step 3 (Validation): 60-90 seconds
- Step 4 (Swap repositories): 30-45 seconds
- Step 5 (Metadata update): 5-10 seconds
- Step 6 (Cache clear): 2-5 seconds

**Total Duration**: 4-6 minutes for ~2.5GB kernel

#### 5. Post-Deployment Validation

```bash
# Run health check to verify deployment
./scripts/health-check.sh

# Check active version in PostgreSQL
psql $PG_URL -c "
  SELECT version, status, concept_count, activated_at
  FROM kb7_snapshots
  WHERE status = 'active';
"

# Verify concept count in GraphDB
curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/kb7-terminology" \
  -H 'Accept: application/sparql-results+json' \
  --data-urlencode "query=SELECT (COUNT(DISTINCT ?c) AS ?count) WHERE { ?c a owl:Class }" | \
  jq -r '.results.bindings[0].count.value'

# Test API endpoint
curl -s http://localhost:8092/v1/concepts/SNOMED/387517004 | jq .
```

---

## Rollback Procedures

### When to Rollback

Rollback if:
- Validation failures detected post-deployment
- Query latency increased significantly (>2x)
- Concept count dropped unexpectedly
- Downstream services report errors
- Cache hit ratio dropped below 80%

### Rollback to Previous Version

#### Option 1: Rollback to Last Active Version

```bash
# Automatic rollback to previous active kernel
./scripts/rollback-kernel.sh previous

# Confirm rollback prompt:
# WARNING: You are about to rollback the KB-7 kernel.
# This will:
#   1. Replace the current production kernel in GraphDB
#   2. Update the metadata registry in PostgreSQL
#   3. Clear all cached data in Redis
#   4. Trigger CDC events to downstream systems
#
# Are you sure you want to continue? (yes/no): yes
```

#### Option 2: Rollback to Specific Version

```bash
# List available versions
psql $PG_URL -c "
  SELECT version, status, concept_count, activated_at, deprecated_at
  FROM kb7_snapshots
  ORDER BY activated_at DESC
  LIMIT 10;
"

# Rollback to specific version
./scripts/rollback-kernel.sh 20241201
```

### Rollback Validation

```bash
# Verify rollback completed successfully
./scripts/health-check.sh

# Check rollback event in PostgreSQL
psql $PG_URL -c "
  SELECT event_type, event_data, created_at
  FROM kb7_snapshot_events
  WHERE event_type = 'rollback'
  ORDER BY created_at DESC
  LIMIT 5;
"

# Monitor Slack notifications for confirmation
```

**Expected Rollback Duration**: 3-4 minutes

---

## Monitoring Setup

### Grafana Dashboard Installation

#### Import Dashboard

```bash
# Navigate to Grafana UI
open http://localhost:3000

# Import dashboard:
# 1. Click "+" → "Import"
# 2. Upload: monitoring/grafana/kb7-dashboard.json
# 3. Select Prometheus datasource
# 4. Click "Import"
```

#### Dashboard Panels

| Panel | Description | Alert Threshold |
|-------|-------------|-----------------|
| **Triple Count Over Time** | Total RDF triples in repository | - |
| **Concept Count by Terminology** | SNOMED, RxNorm, LOINC breakdown | - |
| **Query Latency** | p50, p95, p99 percentiles | p95 > 100ms |
| **SPARQL Endpoint Health** | GraphDB availability | < 1 (DOWN) |
| **Cache Hit Ratio** | Redis cache effectiveness | < 90% |
| **Kernel Deployment Timeline** | Recent deployments/rollbacks | - |
| **Repository Size** | GraphDB disk usage | - |
| **Query Error Rate** | Failed queries per second | > 0.1/s |
| **Active Snapshot Version** | Current kernel version | - |
| **Deployment Success Rate** | 30-day success percentage | < 95% |

### Prometheus Metrics Configuration

#### Install Prometheus

```bash
# Download Prometheus configuration
cp monitoring/prometheus/kb7-metrics.yml /etc/prometheus/prometheus.yml

# Restart Prometheus
sudo systemctl restart prometheus

# Verify scrape targets
open http://localhost:9090/targets
```

#### Key Metrics Exported

```yaml
# GraphDB Metrics
kb7_graphdb_triple_count{repository="kb7-terminology"}
kb7_graphdb_repository_size_bytes{repository="kb7-terminology"}
kb7_concept_count{terminology="SNOMED|RxNorm|LOINC"}
kb7_orphaned_concept_count

# Query Performance Metrics
kb7_query_latency_seconds_bucket{le="0.01,0.05,0.1,0.5,1"}
kb7_query_total
kb7_query_errors_total

# Cache Metrics
kb7_cache_hits_total
kb7_cache_misses_total

# Deployment Metrics
kb7_deployment_timestamp{version="YYYYMMDD",event_type="activated|rollback"}
kb7_deployment_total{period="30d"}
kb7_deployment_success_total{period="30d"}
kb7_snapshot_activated_timestamp_seconds
kb7_last_deployment_timestamp_seconds
```

### Alert Rules

#### Critical Alerts (Page On-Call)

| Alert | Condition | Action |
|-------|-----------|--------|
| **GraphDBDown** | GraphDB unavailable for 2 min | Restart GraphDB service |
| **TripleCountDrop** | >5% decrease in 1 hour | Investigate deployment, consider rollback |
| **DeploymentFailed** | Deployment failure detected | Check logs, notify team |
| **VeryHighQueryLatency** | p99 > 500ms for 5 min | Check GraphDB resources, restart if needed |

#### Warning Alerts (Slack Notification)

| Alert | Condition | Action |
|-------|-----------|--------|
| **HighQueryLatency** | p95 > 100ms for 10 min | Investigate query optimization |
| **LowCacheHitRatio** | <90% for 15 min | Check cache invalidation logic |
| **StaleActiveSnapshot** | >31 days since deployment | Schedule kernel update |
| **LowDeploymentSuccessRate** | <95% in 30 days | Review deployment process |

### Slack Notifications

#### Test Notifications

```bash
# Test simple notification
./scripts/notify-slack.sh simple success "Test deployment notification"

# Test deployment notification
./scripts/notify-slack.sh deployment success 20250124 523451 180

# Test validation notification
./scripts/notify-slack.sh validation success "Concept count: PASS\nOrphans: PASS\nSNOMED roots: PASS\nRxNorm: PASS\nLOINC: PASS"

# Test retry notification
./scripts/notify-slack.sh retry 2 3 "GraphDB connection timeout"
```

#### Example Notifications

**Success**:
```
✅ KB-7 Kernel Deployment: KB-7 Kernel v20250124 deployed (concept count: 523,451, duration: 180s)
```

**Failure**:
```
❌ KB-7 Kernel Deployment: Validation failed - orphaned concepts > 10
```

**Warning**:
```
⚠️ KB-7 Kernel Deployment: Deployment retry attempt 2/3
```

---

## Troubleshooting

### Common Issues

#### Issue 1: S3 Download Timeout

**Symptoms**:
```
[ERROR] Failed to download kernel from S3
```

**Solution**:
```bash
# Increase timeout
export AWS_CLI_READ_TIMEOUT=600

# Verify AWS credentials
aws s3 ls s3://cardiofit-kb-artifacts/

# Manual download test
aws s3 cp s3://cardiofit-kb-artifacts/20250124/kb7-kernel.ttl /tmp/test.ttl
```

#### Issue 2: GraphDB Load Failure

**Symptoms**:
```
[ERROR] Failed to load kernel to test repository
```

**Solution**:
```bash
# Check GraphDB disk space
df -h /var/lib/graphdb

# Verify GraphDB heap memory
curl -s http://localhost:7200/rest/monitor/infrastructure | jq .heapMemoryUsage

# Restart GraphDB if needed
docker-compose restart graphdb

# Retry deployment
./scripts/deploy-kernel.sh 20250124
```

#### Issue 3: Validation Failure

**Symptoms**:
```
[ERROR] Concept count validation failed: 450000 (expected: >500,000)
```

**Solution**:
```bash
# Download manifest to check source data
aws s3 cp s3://cardiofit-kb-artifacts/20250124/kb7-manifest.json - | jq .

# Check SPARQL query directly
curl -X POST "$GRAPHDB_ENDPOINT/repositories/kb7-test" \
  -H 'Accept: application/sparql-results+json' \
  --data-urlencode "query=SELECT (COUNT(DISTINCT ?c) AS ?count) WHERE { ?c a owl:Class }"

# If kernel is corrupted, do not proceed
# Report issue to Knowledge Factory team
./scripts/notify-slack.sh simple failure "Kernel v20250124 failed validation - concept count too low"
```

#### Issue 4: PostgreSQL Connection Error

**Symptoms**:
```
[ERROR] Failed to update metadata registry
psql: could not connect to server
```

**Solution**:
```bash
# Test PostgreSQL connectivity
psql $PG_URL -c "SELECT 1"

# Check PostgreSQL service
docker-compose ps postgres-terminology

# Restart PostgreSQL if needed
docker-compose restart postgres-terminology

# Verify kb7_snapshots table
psql $PG_URL -c "\d kb7_snapshots"
```

#### Issue 5: Redis Cache Clear Failure

**Symptoms**:
```
[WARN] Failed to clear Redis cache (non-fatal)
```

**Solution**:
```bash
# Test Redis connectivity
redis-cli -u $REDIS_URL PING

# Manual cache clear
redis-cli -u $REDIS_URL FLUSHDB

# Restart Redis if needed
docker-compose restart redis-terminology
```

### Log Analysis

```bash
# View recent deployment logs
ls -lt /var/log/kb7/deploy-kernel-*.log | head -5

# Search for errors
grep -i "error" /var/log/kb7/deploy-kernel-latest.log

# View full deployment log
less /var/log/kb7/deploy-kernel-latest.log
```

---

## Operational Runbooks

### Runbook 1: Monthly Kernel Deployment

**Trigger**: Knowledge Factory pipeline completes successfully (1st of month)

**Duration**: 30-45 minutes

**Steps**:

1. **Review Manifest** (5 min)
   ```bash
   aws s3 cp s3://cardiofit-kb-artifacts/$(date +%Y%m01)/kb7-manifest.json - | jq .
   ```

2. **Pre-Deployment Health Check** (5 min)
   ```bash
   ./scripts/health-check.sh --verbose
   ```

3. **Dry Run** (5 min)
   ```bash
   ./scripts/deploy-kernel.sh $(date +%Y%m01) --dry-run
   ```

4. **Production Deployment** (10 min)
   ```bash
   ./scripts/deploy-kernel.sh $(date +%Y%m01)
   ```

5. **Post-Deployment Validation** (10 min)
   ```bash
   ./scripts/health-check.sh
   # Monitor Grafana dashboard for 5 minutes
   # Verify Slack notification received
   ```

6. **Documentation** (5 min)
   - Update deployment log
   - Record concept count changes
   - Note any issues encountered

### Runbook 2: Emergency Rollback

**Trigger**: Production issues detected after deployment

**Duration**: 10-15 minutes

**Steps**:

1. **Assess Impact** (2 min)
   ```bash
   # Check query error rate
   curl -s http://localhost:9090/api/v1/query?query=rate(kb7_query_errors_total[5m]) | jq .

   # Check cache hit ratio
   curl -s http://localhost:9090/api/v1/query?query=kb7_cache_hit_ratio | jq .
   ```

2. **Notify Team** (1 min)
   ```bash
   ./scripts/notify-slack.sh simple warning "Initiating emergency rollback due to [ISSUE]"
   ```

3. **Execute Rollback** (5 min)
   ```bash
   ./scripts/rollback-kernel.sh previous
   ```

4. **Verify Rollback** (3 min)
   ```bash
   ./scripts/health-check.sh
   ```

5. **Monitor Recovery** (5 min)
   - Watch Grafana dashboard
   - Check error rate returns to baseline
   - Verify cache hit ratio improves

6. **Post-Mortem** (async)
   - Document root cause
   - File issue for Knowledge Factory team
   - Update runbook if needed

### Runbook 3: Daily Health Check

**Trigger**: Daily cron job (9 AM UTC)

**Duration**: 5 minutes

**Steps**:

1. **Run Health Check**
   ```bash
   ./scripts/health-check.sh --verbose > /var/log/kb7/daily-health-$(date +%Y%m%d).log 2>&1
   ```

2. **Review Results**
   - If PASS: No action required
   - If FAIL: Page on-call engineer

3. **Report to Slack**
   ```bash
   if [ $? -eq 0 ]; then
     ./scripts/notify-slack.sh simple success "Daily health check passed"
   else
     ./scripts/notify-slack.sh simple failure "Daily health check failed - review logs"
   fi
   ```

---

## Production Readiness Checklist

### Infrastructure

- [ ] GraphDB running with 8GB heap memory
- [ ] PostgreSQL metadata registry created
- [ ] Redis cache accessible
- [ ] S3 bucket access configured
- [ ] Log directory created with proper permissions

### Configuration

- [ ] Environment variables set in `.env.deployment`
- [ ] Slack webhook configured
- [ ] AWS credentials configured
- [ ] All scripts executable

### Monitoring

- [ ] Grafana dashboard imported
- [ ] Prometheus scraping KB-7 metrics
- [ ] Alert rules configured
- [ ] On-call rotation established

### Testing

- [ ] Health check script passes
- [ ] Dry run deployment successful
- [ ] Slack notifications working
- [ ] Rollback procedure tested

### Documentation

- [ ] Team trained on deployment procedures
- [ ] Runbooks reviewed and approved
- [ ] Escalation paths defined
- [ ] Contact information up to date

---

## Maintenance

### Log Rotation

```bash
# Add to /etc/logrotate.d/kb7
/var/log/kb7/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 kb7user kb7group
    sharedscripts
    postrotate
        systemctl reload rsyslog > /dev/null 2>&1 || true
    endscript
}
```

### Cleanup Old Kernels

```bash
# Keep last 12 months of kernels in S3
aws s3 ls s3://cardiofit-kb-artifacts/ | \
  awk '{print $2}' | \
  sort | \
  head -n -12 | \
  xargs -I {} aws s3 rm --recursive s3://cardiofit-kb-artifacts/{}
```

---

## Support Contacts

**KB-7 Team**:
- Slack: `#kb7-automation`
- Email: kb7-team@cardiofit.ai
- On-call: PagerDuty rotation

**Escalation**:
- L1 Support: KB-7 team (response: 30 min)
- L2 Support: Platform Engineering (response: 1 hour)
- L3 Support: Architecture team (response: 4 hours)

---

**Document Version**: 1.0
**Last Review**: November 24, 2025
**Next Review**: February 24, 2025
