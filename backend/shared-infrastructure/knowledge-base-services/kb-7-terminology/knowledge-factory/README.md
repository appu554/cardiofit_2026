# KB-7 Knowledge Factory Pipeline

Automated terminology transformation pipeline for SNOMED-CT, RxNorm, and LOINC using GitHub Actions, ROBOT, and SNOMED-OWL-Toolkit.

## Overview

The Knowledge Factory is a serverless pipeline that:
1. Downloads terminology sources from authoritative APIs (NCTS, UMLS, LOINC.org) via GCP Cloud Run Jobs
2. Transforms them to RDF/OWL ontologies
3. Merges and applies OWL reasoning
4. Validates quality gates
5. Packages and uploads to GCS for GraphDB deployment

**Execution Frequency**: Monthly (1st of month, 2 AM UTC)
**End-to-End Duration**: 45-60 minutes
**Output**: ~2.5GB Turtle file with 8M+ triples

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│          GCP Cloud Run Jobs (Source Downloads)               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│  │  SNOMED  │  │  RxNorm  │  │  LOINC   │  → GCS (sources)  │
│  └──────────┘  └──────────┘  └──────────┘                   │
└─────────────────────────────────────────────────────────────┘
                          ↓ (repository_dispatch)
┌─────────────────────────────────────────────────────────────┐
│         GitHub Actions (7-Stage Pipeline)                    │
├─────────────────────────────────────────────────────────────┤
│  Stage 1: Download   → Pull from GCS                         │
│  Stage 2: Transform  → SNOMED-OWL-Toolkit, Python converters │
│  Stage 3: Merge      → ROBOT merge                           │
│  Stage 4: Reasoning  → ROBOT + ELK (16GB RAM runner)         │
│  Stage 5: Validation → 5 SPARQL quality gates                │
│  Stage 6: Package    → Convert to Turtle + manifest          │
│  Stage 7: Upload     → GCS artifacts + notifications         │
└─────────────────────────────────────────────────────────────┘
                          ↓
        GCS (sincere-hybrid-477206-h2-kb-artifacts-production)
                          ↓
              GraphDB Deployment (manual review)
```

## Pipeline Stages

### Stage 1: Download (5-8 minutes)
- Authenticates to Google Cloud using service account
- Pulls RF2, RRF, and CSV files from GCS
- Extracts archives

**Artifacts**: `source-files/` (7-day retention)

### Stage 2: Transform (15-20 minutes)
- **SNOMED-CT**: RF2 → OWL via SNOMED-OWL-Toolkit v4.0.6
- **RxNorm**: RRF → RDF via custom Python converter
- **LOINC**: CSV → RDF via ROBOT templates

**Artifacts**: `transformed-ontologies/` (7-day retention)

### Stage 3: Merge (8-12 minutes)
- Combines 3 ontologies using ROBOT merge
- Preserves namespace URIs
- No import closure collapse

**Artifacts**: `kb7-merged.owl` (7-day retention)

### Stage 4: Reasoning (20-30 minutes) ⚠️ MEMORY INTENSIVE
- Applies ELK reasoner via ROBOT
- Infers subsumption relationships
- **Requires**: GitHub Larger Runner (16GB RAM)
- **Cost**: ~$4.80/month for monthly runs

**Artifacts**: `kb7-inferred.owl` (7-day retention)

**Alternative Options**:
- Option B: Migrate to AWS CodeBuild with custom instance sizing
- Option C: Split reasoning into 3 parallel jobs (SNOMED, RxNorm, LOINC)

### Stage 5: Validation (5-8 minutes)
Runs 5 SPARQL quality gates:
1. **Concept Count**: >500,000 concepts
2. **Orphaned Concepts**: <10 orphans
3. **SNOMED Roots**: Exactly 1 root (138875005)
4. **RxNorm Drugs**: >100,000 concepts
5. **LOINC Codes**: >90,000 codes

**Failure Handling**: Any validation failure aborts pipeline

**Artifacts**: `validation-results/` (30-day retention)

### Stage 6: Package (10-15 minutes)
- Converts OWL → Turtle (GraphDB-friendly)
- Generates metadata manifest JSON
- Creates versioned snapshot (YYYYMMDD)

**Outputs**:
- `kb7-kernel.ttl` (~2.5GB)
- `kb7-manifest.json` (metadata)
- `kb7-YYYYMMDD.tar.gz` (rollback archive)

**Artifacts**: `packaged-kernel/` (90-day retention)

### Stage 7: Upload (3-5 minutes)
- Uploads to GCS `sincere-hybrid-477206-h2-kb-artifacts-production/YYYYMMDD/`
- Updates `latest/` pointer
- Sends Slack/email notifications

## Local Development & Testing

### Prerequisites
```bash
# Docker and Docker Compose
docker --version  # >= 20.10

# Google Cloud SDK (for GCS integration)
gcloud --version  # >= 400.0

# GitHub CLI (for workflow testing)
gh --version      # >= 2.0
```

### Build Docker Containers
```bash
cd knowledge-factory

# Build SNOMED-OWL-Toolkit container
docker build -f docker/Dockerfile.snomed-toolkit \
  -t kb7-snomed-toolkit:latest .

# Build ROBOT container
docker build -f docker/Dockerfile.robot \
  -t kb7-robot:latest .

# Build Python converters container
docker build -f docker/Dockerfile.converters \
  -t kb7-converters:latest .
```

### Test Individual Stages Locally

#### Stage 2: Transform SNOMED-CT
```bash
# Requires: SNOMED RF2 snapshot in ./test-data/snomed/
docker run --rm \
  -v $(pwd)/test-data/snomed:/input \
  -v $(pwd)/output:/output \
  kb7-snomed-toolkit:latest
```

#### Stage 2: Transform RxNorm
```bash
# Requires: RxNorm RRF files in ./test-data/rxnorm/
docker run --rm \
  -v $(pwd)/test-data/rxnorm:/input \
  -v $(pwd)/output:/output \
  kb7-converters:latest python /app/scripts/transform-rxnorm.py
```

#### Stage 3: Merge Ontologies
```bash
# Requires: snomed-ontology.owl, rxnorm-ontology.ttl, loinc-ontology.ttl in ./output/
docker run --rm \
  -v $(pwd)/output:/workspace \
  kb7-robot:latest /app/scripts/merge-ontologies.sh
```

#### Stage 4: Reasoning (requires 16GB RAM)
```bash
# WARNING: Requires 16GB+ system RAM
docker run --rm \
  -v $(pwd)/output:/workspace \
  -e ROBOT_JAVA_ARGS="-Xmx14G -XX:+UseG1GC" \
  kb7-robot:latest /app/scripts/run-reasoning.sh
```

#### Stage 5: Validation
```bash
# Requires: kb7-inferred.owl in ./output/
docker run --rm \
  -v $(pwd)/output:/workspace \
  -v $(pwd)/validation:/queries \
  kb7-robot:latest robot verify \
    --input /workspace/kb7-inferred.owl \
    --queries /queries/*.sparql
```

#### Stage 6: Package
```bash
docker run --rm \
  -v $(pwd)/output:/workspace \
  kb7-robot:latest /app/scripts/package-kernel.sh
```

### Test Full Pipeline End-to-End
```bash
# Run all stages sequentially (requires 16GB RAM)
cd knowledge-factory
./test-local-pipeline.sh
```

This script will:
1. Download sample data (small subset for testing)
2. Run all 6 transformation/processing stages
3. Generate validation report
4. Output: `output/kb7-kernel.ttl` and `output/kb7-manifest.json`

**Expected Duration**: ~30-45 minutes (with sample data)

## GitHub Actions Setup

### Required Secrets
Configure in GitHub repository settings:

```
GCS_SERVICE_ACCOUNT_KEY  # GCP service account JSON for GCS access
GRAPHDB_URL              # GraphDB endpoint (e.g., http://host.docker.internal:7200)
GRAPHDB_CREDENTIALS      # GraphDB authentication JSON
GCS_BUCKET              # GCS bucket name for sources
PROJECT_ID              # GCP project ID
SLACK_WEBHOOK_URL       # Slack notifications (optional)
```

### Trigger Pipeline Manually
```bash
# Using GitHub CLI
gh workflow run kb-factory.yml

# Using repository_dispatch (simulates Lambda trigger)
gh api repos/:owner/:repo/dispatches \
  -f event_type=terminology-update \
  -f client_payload[version]="$(date +%Y%m%d)"
```

### Monitor Workflow
```bash
# List recent runs
gh run list --workflow=kb-factory.yml

# View specific run
gh run view <run-id> --log

# Download artifacts
gh run download <run-id>
```

## Cost Analysis

### GitHub Larger Runners (Reasoning Stage)
- **Instance**: ubuntu-latest-16-core (16GB RAM)
- **Duration**: ~30 minutes/month
- **Rate**: $0.16/minute
- **Cost**: ~$4.80/month

### GCP Infrastructure (Cloud Run Jobs + GCS)
- **GCS Storage**: $5/month (200GB at $0.02/GB)
- **Cloud Run Jobs**: $2/month (download jobs)
- **Cloud Workflows**: $0.50/month
- **Data Transfer**: $1/month
- **Total GCP**: ~$8.50/month

### Total Monthly Cost: ~$13-15/month

### Cost Optimization Options
1. **Use Standard Runners**: Free but may OOM (unreliable)
2. **GCP Cloud Run Jobs for Reasoning**: Custom instance sizing, pay-per-use
3. **Parallel Reasoning**: Split into 3 jobs (more complex)

## Troubleshooting

### Reasoning Stage OOM Errors
**Symptom**: Stage 4 fails with "OutOfMemoryError"

**Solutions**:
1. Increase runner size to `ubuntu-latest-32-core` (32GB RAM)
2. Reduce JVM heap: `-Xmx12G` instead of `-Xmx14G`
3. Migrate to AWS CodeBuild with 32GB instance

### Validation Failures
**Symptom**: Stage 5 fails with quality gate violations

**Diagnosis**:
```bash
# Download validation results
gh run download <run-id> --name validation-results

# Check specific query results
cat validation-results/concept-count.txt
```

**Common Issues**:
- **Low concept count**: Transformation incomplete, check Stage 2 logs
- **Orphaned concepts**: Hierarchy transformation failed, check RF2 files
- **SNOMED root missing**: SNOMED-OWL-Toolkit version mismatch

### GCS Upload Failures
**Symptom**: Stage 7 fails with GCS access errors

**Solutions**:
1. Verify GCS service account key in GitHub Secrets
2. Check GCS bucket IAM permissions (Storage Object Creator role required)
3. Verify bucket name: `sincere-hybrid-477206-h2-kb-artifacts-production`

### Docker Build Failures
**Symptom**: Container build fails during local testing

**Solutions**:
```bash
# Clean Docker cache
docker builder prune -a

# Rebuild with no cache
docker build --no-cache -f docker/Dockerfile.snomed-toolkit -t kb7-snomed-toolkit:latest .
```

## Monitoring & Alerts

### Slack Notifications
- ✅ **Success**: Kernel version, concept count, workflow link
- ❌ **Failure**: Failed stage, workflow link, @channel mention

### Grafana Dashboard (Optional)
- Pipeline execution time trend
- Concept count growth over time
- Validation failure rate
- S3 storage usage

### GCP Cloud Monitoring (Cloud Run Jobs)
- Job execution duration >10 minutes (timeout risk)
- GCS bucket storage >80% capacity
- API credential expiration warnings

## Deployment Workflow

### Monthly Automated Run
1. **Cloud Scheduler** triggers Cloud Workflows (1st of month, 2 AM UTC)
2. **Cloud Workflows** orchestrate parallel Cloud Run Job downloads
3. **GitHub Dispatcher Job** dispatches GitHub Actions workflow
4. **GitHub Actions** runs 7-stage pipeline
5. **Slack notification** on completion
6. **Manual review** of `kb7-manifest.json` before GraphDB deployment

### Manual Kernel Deployment
```bash
# After pipeline success, review manifest
gsutil cat gs://sincere-hybrid-477206-h2-kb-artifacts-production/latest/kb7-manifest.json | jq

# Deploy to GraphDB test repository
cd ../scripts
./deploy-kernel.sh YYYYMMDD

# Validate in test environment
./validate-graphdb-kernel.sh

# Promote to production
./promote-kernel-to-production.sh YYYYMMDD
```

## Rollback Procedure

### Scenario: New kernel fails validation
```bash
# Identify last successful kernel version
gsutil ls gs://sincere-hybrid-477206-h2-kb-artifacts-production/ | grep -E '/'

# Rollback to previous version
cd ../scripts
./rollback-kernel.sh YYYYMMDD-previous
```

## Performance Metrics

| Stage | Target Duration | Actual (Avg) | Bottleneck |
|-------|----------------|--------------|------------|
| Download | 5-8 min | 6 min | GCS transfer |
| Transform | 15-20 min | 18 min | SNOMED-OWL-Toolkit |
| Merge | 8-12 min | 10 min | ROBOT I/O |
| Reasoning | 20-30 min | 25 min | ELK reasoner |
| Validation | 5-8 min | 6 min | SPARQL execution |
| Package | 10-15 min | 12 min | OWL → Turtle |
| Upload | 3-5 min | 4 min | GCS upload |
| **Total** | **45-60 min** | **51 min** | Reasoning stage |

## Success Criteria

- ✅ Pipeline completes successfully (exit code 0)
- ✅ All 5 quality gates pass
- ✅ Kernel size >2GB (Turtle format)
- ✅ Concept count >500,000
- ✅ Triple count >8,000,000
- ✅ GCS upload verified (checksums match)
- ✅ Slack notification received

## Support & Contact

- **Issues**: Create GitHub issue with `knowledge-factory` label
- **Slack**: `#kb7-automation` channel
- **Email**: kb7-team@cardiofit.ai
- **Documentation**: See KB7_ARCHITECTURE_TRANSFORMATION_PLAN.md section 1.3

## License

Proprietary - CardioFit Platform
