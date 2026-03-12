# KB-7 Cloud Run Jobs - Status Investigation

**Date**: November 25, 2025
**Time**: 17:22 UTC
**Status**: ⚠️ All Jobs Failing - Container Execution Issue

---

## 🚨 Critical Finding

ALL jobs triggered by the workflow execution (dca91583-67a7-4c5b-9ff2-684037321538) at 15:09:20 UTC are failing or stuck:

### Workflow Execution Summary

**Workflow Started**: 2025-11-25 15:09:20 UTC
**Workflow Status**: SUCCEEDED (workflow logic completed)
**Jobs Status**: ALL FAILED (container executions failed)

### Individual Job Status

#### 1. SNOMED Job (`kb7-snomed-job-production-s5sn5`)
- **Status**: Shows "RUNNING" but stuck
- **Started**: 15:09:20 UTC
- **Container Started**: 15:10:34 UTC
- **Duration**: 2+ hours (should timeout at 60 minutes)
- **Issue**:
  - No application logs (stdout/stderr)
  - Has "Retry" condition with "WaitingForOperation"
  - Past 60-minute timeout but not terminated
  - Likely container crashed/failed silently

#### 2. RxNorm Job (`kb7-rxnorm-job-production-czwrv`)
- **Status**: Shows "RUNNING" but stuck
- **Started**: 15:09:21 UTC
- **Duration**: 2+ hours (should timeout at 60 minutes)
- **Issue**: Same pattern as SNOMED - no logs, likely stuck

#### 3. LOINC Job (`kb7-loinc-job-production-k59gc`)
- **Status**: FAILED
- **Started**: 15:09:20 UTC
- **Container Started**: 15:10:39 UTC
- **Failed**: 15:40:44 UTC (after 30 minutes)
- **Error**: "The configured timeout was reached"
- **Issue**: Timeout too short (1800s vs 3600s needed)

---

## 📊 Detailed Timeline

### Workflow Execution
```
15:09:20 - Workflow execution starts
15:09:20 - SNOMED job invoked
15:09:20 - LOINC job invoked
15:09:21 - RxNorm job invoked
15:09:26 - SNOMED container starts
15:09:27 - RxNorm container starts
15:10:39 - LOINC container starts
15:19:49 - SNOMED enters "Retry/WaitingForOperation" state
15:40:44 - LOINC fails with timeout
17:22:00 - Current time: SNOMED and RxNorm still showing "RUNNING"
```

---

## 🔍 Root Cause Analysis

### Primary Issue: Container Execution Failure
All three jobs show the same pattern:
1. ✅ Container images import successfully
2. ✅ Containers provision and start
3. ❌ **No application logs appear** (Python code not executing)
4. ❌ Containers crash or hang silently

### Evidence
- **Zero stdout/stderr logs** from any job container
- SNOMED/RxNorm show "running" but are stuck with "WaitingForOperation" status
- LOINC failed after 30 minutes (its configured timeout)
- SNOMED/RxNorm past 60-minute timeout but not terminated (infrastructure polling issue)

### Possible Causes
1. **Container Entrypoint Issue**: Python main.py may not be executing
2. **Import/Dependency Error**: Silent failure loading Python modules
3. **Environment Variable Issue**: Missing or incorrect env vars causing immediate crash
4. **Service Account Permissions**: Container can't access Secret Manager or GCS
5. **Container Health Check**: Cloud Run may think container is "starting" forever

---

## 🔧 Diagnostic Steps Completed

1. ✅ Verified all 4 API keys are configured in Secret Manager
2. ✅ Verified workflow service account has `roles/run.invoker` permission
3. ✅ Verified job service account is `kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com`
4. ✅ Verified container images exist and are accessible
5. ✅ Verified environment variables are set correctly
6. ✅ Checked application logs - NONE found (confirms container execution issue)

---

## 📋 Next Steps to Resolve

### Option 1: Check Container Entrypoint (Most Likely)
```bash
# Verify the container image entrypoint
gcloud artifacts docker images describe \
  us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy/kb7-snomed-downloader:latest

# Check if main.py exists and is executable
docker pull us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy/kb7-snomed-downloader:latest
docker run --rm --entrypoint=/bin/sh us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy/kb7-snomed-downloader:latest -c "ls -la /app"
```

### Option 2: Test Container Locally
```bash
# Run container locally to see actual error
docker run --rm \
  -e PROJECT_ID=sincere-hybrid-477206-h2 \
  -e SOURCE_BUCKET=sincere-hybrid-477206-h2-kb-sources-production \
  -e ENVIRONMENT=production \
  -e SECRET_NAME=kb7-nhs-trud-api-key-production \
  us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy/kb7-snomed-downloader:latest
```

### Option 3: Check Service Account Permissions
```bash
# Verify Secret Manager access
gcloud secrets get-iam-policy kb7-nhs-trud-api-key-production | grep kb7-functions-production

# Verify GCS bucket access
gcloud storage buckets get-iam-policy gs://sincere-hybrid-477206-h2-kb-sources-production | grep kb7-functions-production
```

### Option 4: Fix LOINC Timeout Immediately
```bash
# This is a confirmed issue regardless of container execution
gcloud run jobs update kb7-loinc-job-production \
  --region=us-central1 \
  --task-timeout=3600s
```

### Option 5: Check Function Code Main Entrypoint
```bash
# Verify main.py in each function has proper Flask app structure
# Cloud Run Jobs need a specific entrypoint format (not Flask request handlers)
```

---

## ⚠️ Immediate Actions Required

1. **Kill Stuck Jobs**:
   ```bash
   gcloud run jobs executions cancel kb7-snomed-job-production-s5sn5 --region=us-central1
   gcloud run jobs executions cancel kb7-rxnorm-job-production-czwrv --region=us-central1
   ```

2. **Fix LOINC Timeout**:
   ```bash
   gcloud run jobs update kb7-loinc-job-production --region=us-central1 --task-timeout=3600s
   ```

3. **Investigate Container Entrypoint**: Check if Cloud Run Jobs code is different from Cloud Functions/Services code

---

## 🎓 Key Learning

**Cloud Functions vs Cloud Run Jobs**: The container images may have been built for Cloud Functions Gen2 (which expects HTTP handlers) but Cloud Run Jobs need a **standalone executable** that runs to completion, not an HTTP server.

**Required Code Pattern for Jobs**:
```python
# WRONG for Jobs (Flask/Functions pattern):
@functions_framework.http
def download_snomed(request):
    # HTTP handler

# CORRECT for Jobs:
if __name__ == "__main__":
    # Direct execution
    download_snomed()
```

---

**Last Updated**: 2025-11-25 17:22 UTC
**Status**: Investigation in progress - Container execution failure identified
