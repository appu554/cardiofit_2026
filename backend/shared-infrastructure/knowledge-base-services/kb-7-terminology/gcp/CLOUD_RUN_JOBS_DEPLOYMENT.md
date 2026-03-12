# KB-7 Knowledge Factory - Cloud Run Jobs Deployment

**Date**: November 25, 2025
**Status**: ✅ Jobs Deployed - Testing in Progress
**Approach**: Cloud Run Jobs (Batch Execution)

---

## 🎯 Why Cloud Run Jobs?

After encountering authentication and logging issues with Cloud Run Services, we pivoted to Cloud Run Jobs for these reasons:

### Advantages of Jobs
- ✅ **No HTTP Authentication**: Jobs don't require OIDC tokens or IAM invoker permissions
- ✅ **Better Logging**: Direct execution logs are immediately visible
- ✅ **Designed for Batch Work**: Monthly terminology downloads are batch operations, not web services
- ✅ **Simpler Invocation**: Can be executed directly or from Cloud Workflows/Scheduler
- ✅ **No Endpoint Management**: No need to manage HTTP endpoints and request/response formats

### When Services Failed
- 401 Unauthorized errors despite IAM permissions
- SNOMED/RxNorm services produced no logs during workflow execution
- LOINC service worked but required complex IAM permission propagation
- HTTP authentication added unnecessary complexity for batch operations

---

## 📦 Deployed Jobs

| Job Name | Region | Memory | CPU | Timeout | Status |
|----------|--------|--------|-----|---------|--------|
| **kb7-snomed-job-production** | us-central1 | 10Gi | 4 | 3600s | ✅ Deployed & Testing |
| **kb7-rxnorm-job-production** | us-central1 | 3Gi | 2 | 3600s | ✅ Deployed |
| **kb7-loinc-job-production** | us-central1 | 2Gi | 1 | 1800s | ✅ Deployed |

### Job Configuration Details

#### SNOMED Job
```bash
Image: us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy/kb7-snomed-downloader:latest
Service Account: kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
Environment Variables:
  - PROJECT_ID=sincere-hybrid-477206-h2
  - SOURCE_BUCKET=sincere-hybrid-477206-h2-kb-sources-production
  - ENVIRONMENT=production
  - SECRET_NAME=kb7-nhs-trud-api-key-production
Max Retries: 0 (fail fast)
Task Timeout: 3600s (60 minutes)
```

#### RxNorm Job
```bash
Image: us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy/kb7-rxnorm-downloader:latest
Service Account: kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
Environment Variables:
  - PROJECT_ID=sincere-hybrid-477206-h2
  - SOURCE_BUCKET=sincere-hybrid-477206-h2-kb-sources-production
  - ENVIRONMENT=production
  - SECRET_NAME=kb7-umls-api-key-production
Max Retries: 0
Task Timeout: 3600s (60 minutes)
```

#### LOINC Job
```bash
Image: us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy/kb7-loinc-downloader:latest
Service Account: kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
Environment Variables:
  - PROJECT_ID=sincere-hybrid-477206-h2
  - SOURCE_BUCKET=sincere-hybrid-477206-h2-kb-sources-production
  - ENVIRONMENT=production
  - SECRET_NAME=kb7-loinc-credentials-production
Max Retries: 0
Task Timeout: 1800s (30 minutes)
```

---

## 🚀 Manual Job Execution

### Execute Individual Jobs

**SNOMED CT Download:**
```bash
gcloud run jobs execute kb7-snomed-job-production \
  --region=us-central1 \
  --wait
```

**RxNorm Download:**
```bash
gcloud run jobs execute kb7-rxnorm-job-production \
  --region=us-central1 \
  --wait
```

**LOINC Download:**
```bash
gcloud run jobs execute kb7-loinc-job-production \
  --region=us-central1 \
  --wait
```

### Check Job Execution Status

**List recent executions:**
```bash
gcloud run jobs executions list \
  --job=kb7-snomed-job-production \
  --region=us-central1 \
  --limit=5
```

**Describe specific execution:**
```bash
gcloud run jobs executions describe EXECUTION_NAME \
  --region=us-central1
```

### View Job Logs

**Real-time logs during execution:**
```bash
gcloud logging read "resource.type=cloud_run_job AND resource.labels.job_name=kb7-snomed-job-production" \
  --limit=50 \
  --format="table(timestamp,severity,textPayload)" \
  --freshness=10m
```

**All logs for a specific execution:**
```bash
gcloud logging read "resource.type=cloud_run_job AND resource.labels.job_name=kb7-snomed-job-production AND labels.\"run.googleapis.com/execution_name\"=EXECUTION_NAME" \
  --format="table(timestamp,severity,textPayload)"
```

---

## 🔄 Cloud Workflows Integration (TODO)

The existing workflow ([kb-factory-cloudrun-workflow.yaml](workflows/kb-factory-cloudrun-workflow.yaml)) needs to be updated to invoke Cloud Run Jobs instead of Services.

### Current Workflow (Services)
```yaml
- call_snomed_downloader:
    call: http.post
    args:
      url: ${snomed_url}
      auth:
        type: OIDC
```

### Updated Workflow (Jobs) - TO BE IMPLEMENTED
```yaml
- execute_snomed_job:
    call: googleapis.run.v1.namespaces.jobs.run
    args:
      name: projects/sincere-hybrid-477206-h2/locations/us-central1/jobs/kb7-snomed-job-production
      body: {}
    result: snomed_result
```

**Implementation Steps:**
1. Update workflow YAML to use job execution API
2. Remove OIDC authentication (not needed for jobs)
3. Update result parsing (job executions return different response format)
4. Redeploy workflow
5. Test workflow execution
6. Update scheduler to trigger updated workflow

---

## 📊 Testing Results

### SNOMED Job - First Execution
**Status**: In Progress
**Started**: 2025-11-25 13:40 UTC
**Expected Duration**: Up to 60 minutes

**Observations:**
- ✅ Job provisioning successful
- ✅ Execution started without authentication errors
- ⏳ Awaiting download completion and log verification

### Next Testing Steps
1. ✅ Verify SNOMED job completes successfully
2. ⏳ Execute RxNorm job and verify
3. ⏳ Execute LOINC job and verify
4. ⏳ Verify files appear in GCS bucket
5. ⏳ Update workflow to invoke jobs
6. ⏳ Test end-to-end workflow execution

---

## 🔧 Troubleshooting

### Job Fails to Start
**Check job configuration:**
```bash
gcloud run jobs describe kb7-snomed-job-production \
  --region=us-central1 \
  --format=yaml
```

**Verify image exists:**
```bash
gcloud artifacts docker images list \
  us-central1-docker.pkg.dev/sincere-hybrid-477206-h2/cloud-run-source-deploy \
  --filter="package:kb7-snomed-downloader"
```

### No Logs Appearing
**Issue**: Logs take 30-60 seconds to appear
**Solution**: Wait 1-2 minutes after job starts before checking logs

**Alternative log query:**
```bash
gcloud logging read "resource.type=cloud_run_job" \
  --limit=100 \
  --format="table(timestamp,resource.labels.job_name,severity,textPayload)" \
  --freshness=1h
```

### Job Times Out
**Issue**: Download takes longer than timeout
**Solution**: Increase task timeout
```bash
gcloud run jobs update kb7-snomed-job-production \
  --region=us-central1 \
  --task-timeout=7200s
```

### Secret Access Denied
**Issue**: Service account can't read secrets
**Solution**: Verify IAM permissions
```bash
gcloud secrets get-iam-policy kb7-nhs-trud-api-key-production
```

Should show:
```yaml
- members:
  - serviceAccount:kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
  role: roles/secretmanager.secretAccessor
```

---

## 📁 Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│  Cloud Scheduler (Monthly Trigger)                     │
│  Schedule: 0 2 1 * *                                   │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Cloud Workflow (TO BE UPDATED)                        │
│  ┌───────────────────────────────────────────────────┐ │
│  │ Execute Cloud Run Jobs (Parallel)                 │ │
│  │                                                    │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────┐│ │
│  │  │ SNOMED Job   │  │ RxNorm Job   │  │ LOINC   ││ │
│  │  │ - Download   │  │ - Download   │  │ Job     ││ │
│  │  │ - Upload GCS │  │ - Upload GCS │  │         ││ │
│  │  │ - Return     │  │ - Return     │  │         ││ │
│  │  │   metadata   │  │   metadata   │  │         ││ │
│  │  └──────────────┘  └──────────────┘  └─────────┘│ │
│  └───────────────────────────────────────────────────┘ │
│                                                         │
│  ┌───────────────────────────────────────────────────┐ │
│  │ GitHub Dispatcher (Future)                        │ │
│  │ - Trigger workflow with download metadata         │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## ✅ Advantages Over Cloud Functions

| Aspect | Cloud Functions (Failed) | Cloud Run Jobs (Current) |
|--------|-------------------------|-------------------------|
| **Deployment** | Build failed - permission issues | ✅ Deployed successfully |
| **Authentication** | Complex OIDC + IAM | ✅ Simple - no HTTP auth needed |
| **Logging** | Difficult to access | ✅ Direct access via gcloud |
| **Execution** | HTTP triggers only | ✅ Direct execution or workflow |
| **Debugging** | Build logs in Cloud Build | ✅ Execution logs immediate |
| **Cost** | Function invocations | Job executions (similar) |
| **Timeout** | 60 minutes max | ✅ 60 minutes (configurable) |

---

## 📝 Next Steps

1. **Complete Current Test** - Verify SNOMED job execution completes successfully ⏳
2. **Test All Jobs** - Execute RxNorm and LOINC jobs individually
3. **Verify Downloads** - Check GCS bucket for downloaded files
4. **Update Workflow** - Modify workflow to invoke jobs instead of services
5. **Test Workflow** - Execute updated workflow end-to-end
6. **Update Scheduler** - Ensure scheduler triggers updated workflow
7. **Production Readiness** - Update deployment guides and documentation

---

## 🔗 Resources

- **Cloud Run Jobs Documentation**: https://cloud.google.com/run/docs/create-jobs
- **Job Execution API**: https://cloud.google.com/run/docs/reference/rest/v1/namespaces.jobs/run
- **Cloud Workflows Jobs Integration**: https://cloud.google.com/workflows/docs/reference/googleapis/run/v1/namespaces.jobs/run
- **GCS Bucket**: gs://sincere-hybrid-477206-h2-kb-sources-production/
- **Console**: https://console.cloud.google.com/run/jobs?project=sincere-hybrid-477206-h2

---

**Last Updated**: 2025-11-25 13:45 UTC
**Status**: Jobs deployed, first execution in progress
