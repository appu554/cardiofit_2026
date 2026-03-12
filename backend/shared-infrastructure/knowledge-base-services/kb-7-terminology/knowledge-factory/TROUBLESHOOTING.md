# Knowledge Factory Troubleshooting Guide

Common issues and solutions for the KB-7 Knowledge Factory pipeline.

## Table of Contents
- [Reasoning Stage OOM Errors](#reasoning-stage-oom-errors)
- [Validation Failures](#validation-failures)
- [Transformation Errors](#transformation-errors)
- [S3 Upload Issues](#s3-upload-issues)
- [Docker Build Problems](#docker-build-problems)
- [Performance Issues](#performance-issues)

---

## Reasoning Stage OOM Errors

### Symptom
```
Stage 4: OWL Reasoning
ERROR: OutOfMemoryError: Java heap space
```

### Root Cause
ELK reasoner requires 14-16GB RAM for 8M+ triple ontology. Standard GitHub runners have only 7GB.

### Solutions

#### Solution 1: Use GitHub Larger Runners (RECOMMENDED)
```yaml
# .github/workflows/kb-factory.yml
reasoning:
  runs-on: ubuntu-latest-16-core  # 16GB RAM
```

**Cost**: ~$4.80/month for monthly runs
**Reliability**: High (99%+ success rate)

#### Solution 2: Reduce JVM Heap
```yaml
# .github/workflows/kb-factory.yml
env:
  ROBOT_JAVA_ARGS: "-Xmx12G -XX:+UseG1GC"  # Reduce from 14G
```

**Trade-off**: May still OOM with large ontologies
**Use case**: Temporary cost optimization

#### Solution 3: Migrate to AWS CodeBuild
```yaml
# aws/codebuild-reasoning.yml
environment:
  compute-type: BUILD_GENERAL1_LARGE  # 15GB RAM
  image: aws/codebuild/standard:5.0
```

**Cost**: ~$0.10/min (similar to GitHub)
**Benefit**: No timeout limits, custom instance sizing

#### Solution 4: Split Reasoning into Parallel Jobs
```yaml
# .github/workflows/kb-factory.yml
reasoning-snomed:
  runs-on: ubuntu-latest
  # Reason SNOMED only (5GB heap OK)

reasoning-rxnorm:
  runs-on: ubuntu-latest
  # Reason RxNorm only

reasoning-loinc:
  runs-on: ubuntu-latest
  # Reason LOINC only

merge-reasoned:
  needs: [reasoning-snomed, reasoning-rxnorm, reasoning-loinc]
  # Merge results with ROBOT
```

**Complexity**: Higher (requires custom merge logic)
**Cost**: Free (standard runners)

---

## Validation Failures

### Symptom
```
Stage 5: Quality Validation
FAILED: concept-count.sparql
  Expected: >500000
  Actual:   342156
```

### Diagnosis

#### 1. Download validation results
```bash
gh run download <run-id> --name validation-results
cd validation-results
cat concept-count.txt
```

#### 2. Check specific query failures
```bash
# Check each validation file
for f in *.txt; do
  echo "=== $f ==="
  cat "$f"
done
```

### Common Failures & Solutions

#### Low Concept Count
**Symptom**: `concept-count.sparql` returns <500,000

**Causes**:
1. Incomplete source download (Lambda timeout)
2. Transformation failure (Stage 2)
3. Source file corruption

**Solutions**:
```bash
# Check Lambda logs
aws logs tail /aws/lambda/snomed-downloader --follow

# Verify S3 checksums
aws s3 cp s3://cardiofit-kb-sources/checksums.sha256 -
sha256sum -c checksums.sha256

# Retry transformation manually
docker run --rm \
  -v ./sources:/input \
  -v ./output:/output \
  kb7-snomed-toolkit:latest
```

#### Orphaned Concepts
**Symptom**: `orphaned-concepts.sparql` returns >10 concepts

**Causes**:
1. Hierarchy transformation failed
2. Parent relationships missing in source
3. SNOMED RF2 structure incorrect

**Solutions**:
```bash
# Check SNOMED hierarchy file
unzip -l sources/snomed.zip | grep "Relationship"

# Verify relationship file exists
ls -lh sources/extracted/snomed/Full/Terminology/sct2_Relationship_*

# Retry SNOMED transformation with debug logging
docker run --rm \
  -v ./sources:/input \
  -v ./output:/output \
  -e DEBUG=true \
  kb7-snomed-toolkit:latest
```

#### SNOMED Root Missing
**Symptom**: `snomed-roots.sparql` returns 0 (expected: 1)

**Causes**:
1. SNOMED-OWL-Toolkit version mismatch
2. RF2 snapshot incomplete
3. Root concept ID changed

**Solutions**:
```bash
# Verify toolkit version
docker run --rm kb7-snomed-toolkit:latest \
  java -jar /app/snomed-owl-toolkit.jar --version

# Expected: v4.0.6

# Check SNOMED root concept exists
grep "138875005" output/snomed-ontology.owl
```

---

## Transformation Errors

### SNOMED-OWL-Toolkit Errors

#### Symptom
```
ERROR: RF2 snapshot archive not found
```

**Solution**:
```bash
# Verify RF2 archive naming
ls -lh sources/extracted/snomed/

# Expected: SnomedCT_InternationalRF2_PRODUCTION_YYYYMMDD.zip

# If missing, check Lambda download logs
aws logs tail /aws/lambda/snomed-downloader
```

#### Symptom
```
ERROR: Invalid RF2 file structure
```

**Solution**:
```bash
# Extract and verify RF2 structure
unzip -l sources/snomed.zip

# Expected directories:
#   Full/Terminology/
#   Snapshot/Terminology/
#   Delta/Terminology/

# If corrupted, re-download from NCTS
```

### RxNorm Transformation Errors

#### Symptom
```
ERROR: RXNCONSO.RRF not found
```

**Solution**:
```bash
# Verify RRF files
ls -lh sources/extracted/rxnorm/

# Expected files:
#   RXNCONSO.RRF
#   RXNREL.RRF
#   RXNSAT.RRF

# If missing, check UMLS download
aws logs tail /aws/lambda/rxnorm-downloader
```

#### Symptom
```
Python error: UnicodeDecodeError
```

**Solution**:
```python
# RRF files use UTF-8 encoding
# Update transform-rxnorm.py if needed:
with open(rrf_file, 'r', encoding='utf-8', errors='replace') as f:
    # errors='replace' handles encoding issues
```

---

## S3 Upload Issues

### Symptom
```
Stage 7: Upload & Notify
ERROR: Access Denied (S3)
```

### Solutions

#### 1. Verify AWS Credentials
```bash
# Check GitHub Secrets
gh secret list

# Expected:
#   AWS_ACCESS_KEY_ID
#   AWS_SECRET_ACCESS_KEY

# Test credentials locally
aws s3 ls s3://cardiofit-kb-artifacts/
```

#### 2. Check S3 Bucket Permissions
```json
// S3 bucket policy should allow:
{
  "Effect": "Allow",
  "Action": [
    "s3:PutObject",
    "s3:GetObject",
    "s3:ListBucket"
  ],
  "Resource": [
    "arn:aws:s3:::cardiofit-kb-artifacts",
    "arn:aws:s3:::cardiofit-kb-artifacts/*"
  ]
}
```

#### 3. Verify Bucket Exists
```bash
aws s3 mb s3://cardiofit-kb-artifacts --region us-east-1
```

---

## Docker Build Problems

### Symptom
```
ERROR: failed to solve with frontend dockerfile.v0
```

### Solutions

#### 1. Clean Docker Cache
```bash
docker builder prune -a
docker system prune -a  # WARNING: Removes all unused images
```

#### 2. Rebuild with No Cache
```bash
docker build --no-cache \
  -f docker/Dockerfile.snomed-toolkit \
  -t kb7-snomed-toolkit:latest .
```

#### 3. Check Dockerfile Syntax
```bash
# Validate Dockerfile
docker build --check -f docker/Dockerfile.snomed-toolkit .
```

### Symptom
```
ERROR: Download failed - SNOMED-OWL-Toolkit v4.0.6
```

**Solution**:
```dockerfile
# Update Dockerfile with alternative download URL
RUN curl -L -o snomed-owl-toolkit.jar \
    https://github.com/IHTSDO/snomed-owl-toolkit/releases/download/v4.0.6/snomed-owl-toolkit-4.0.6-executable.jar \
    || curl -L -o snomed-owl-toolkit.jar \
       https://backup-mirror.com/snomed-owl-toolkit-4.0.6.jar
```

---

## Performance Issues

### Symptom: Slow Reasoning Stage (>45 minutes)

#### Solution 1: Optimize JVM Settings
```yaml
env:
  ROBOT_JAVA_ARGS: "-Xmx14G -XX:+UseG1GC -XX:ParallelGCThreads=8"
```

#### Solution 2: Use Parallel GC
```yaml
env:
  ROBOT_JAVA_ARGS: "-Xmx14G -XX:+UseParallelGC"
```

#### Solution 3: Increase Runner Size
```yaml
reasoning:
  runs-on: ubuntu-latest-32-core  # 32GB RAM, more CPU cores
```

### Symptom: Slow S3 Upload (>10 minutes)

#### Solution: Use Multipart Upload
```bash
# scripts/upload-to-s3.sh
aws s3 cp kb7-kernel.ttl \
  s3://$S3_BUCKET/$VERSION/kb7-kernel.ttl \
  --storage-class STANDARD \
  --metadata "version=$VERSION" \
  --expected-size 2684354560  # Enable multipart for large files
```

---

## Emergency Procedures

### Complete Pipeline Failure

1. **Immediate**: Use previous month's kernel
```bash
aws s3 cp s3://cardiofit-kb-artifacts/20241101/kb7-kernel.ttl \
  s3://cardiofit-kb-artifacts/latest/kb7-kernel.ttl
```

2. **Diagnose**: Check all stage logs
```bash
gh run view <run-id> --log > pipeline-failure.log
```

3. **Escalate**: Notify team
```bash
# Post to Slack
curl -X POST $SLACK_WEBHOOK_URL \
  -d '{"text":"🚨 KB-7 Pipeline FAILED - Manual review required"}'
```

### Data Corruption Detected

1. **Stop**: Abort current deployment
```bash
gh run cancel <run-id>
```

2. **Rollback**: Use validated kernel
```bash
cd ../scripts
./rollback-kernel.sh 20241101
```

3. **Investigate**: Check source file integrity
```bash
aws s3 cp s3://cardiofit-kb-sources/checksums.sha256 -
sha256sum -c checksums.sha256
```

---

## Getting Help

### Debug Checklist
- [ ] Check GitHub Actions logs (`gh run view <run-id> --log`)
- [ ] Review S3 source files (`aws s3 ls s3://cardiofit-kb-sources/`)
- [ ] Verify Docker container versions
- [ ] Test locally with sample data (`./test-local-pipeline.sh`)
- [ ] Check AWS Lambda logs (download stage)

### Support Channels
- **Slack**: `#kb7-automation`
- **Email**: kb7-team@cardiofit.ai
- **GitHub Issues**: Tag with `knowledge-factory` label
- **On-call**: Check rotation schedule in PagerDuty

### Escalation Path
1. **Level 1**: Team lead (Slack)
2. **Level 2**: DevOps engineer (AWS issues)
3. **Level 3**: Senior architect (design decisions)
