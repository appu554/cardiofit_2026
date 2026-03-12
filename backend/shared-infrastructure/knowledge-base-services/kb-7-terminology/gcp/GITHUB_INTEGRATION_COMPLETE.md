# GitHub Integration - Complete Implementation

## 📋 Summary

Successfully implemented **Option 2**: Updated the workflow to properly pass download results to the GitHub dispatcher using GCS-based result coordination.

## 🎯 What Was Changed

### 1. **Downloader Jobs** (All Three: SNOMED, RxNorm, LOINC)

Each downloader now writes its result JSON to a known GCS location for workflow coordination:

- **SNOMED**: `gs://[bucket]/workflow-results/snomed-latest.json`
- **RxNorm**: `gs://[bucket]/workflow-results/rxnorm-latest.json`
- **LOINC**: `gs://[bucket]/workflow-results/loinc-latest.json`

**Result JSON Format**:
```json
{
  "status": "success" | "skipped",
  "message": "Download complete" | "File already exists",
  "gcs_uri": "gs://bucket/path/to/file.zip",
  "gcs_key": "path/to/file.zip",
  "version": "version_string",
  "terminology": "snomed" | "rxnorm" | "loinc",
  "timestamp": "2025-11-26T05:30:00.000000"
}
```

### 2. **Workflow** (kb-factory-jobs-workflow-v2.yaml)

**New Capabilities**:
- **Phase 3**: Read result JSON files from GCS using HTTP connector
- **Phase 4**: Pass GCS keys and config to GitHub dispatcher as environment variable overrides
- **Dynamic Parameter Support**: Accept `github_repo` input parameter for flexibility

**Key Workflow Changes**:

```yaml
# Phase 3: Read Download Results from GCS
- read_download_results:
    - read_snomed_result:
        call: http.get
        args:
          url: "https://storage.googleapis.com/.../workflow-results%2Fsnomed-latest.json?alt=media"
          auth:
            type: OAuth2

# Phase 4: GitHub Dispatcher with Overrides
- execute_github_dispatcher:
    - run_github_job:
        call: googleapis.run.v2.projects.locations.jobs.run
        args:
          name: ${github_job}
          overrides:
            containerOverrides:
              - env:
                  - name: "PROJECT_ID"
                    value: ${project_id}
                  - name: "SNOMED_KEY"
                    value: ${snomed_gcs_key}
                  - name: "RXNORM_KEY"
                    value: ${rxnorm_gcs_key}
                  - name: "LOINC_KEY"
                    value: ${loinc_gcs_key}
```

## 🚀 Deployment Steps

### Step 1: Rebuild and Deploy Downloaders

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/gcp

# Run the rebuild script
./rebuild-and-deploy-jobs.sh
```

This will:
1. Build new container images with GCS result-writing code
2. Push to Artifact Registry
3. Update Cloud Run Jobs with latest images

### Step 2: Deploy Updated Workflow

```bash
# Deploy the new workflow version
gcloud workflows deploy kb7-factory-workflow-production \
  --location=us-central1 \
  --source=workflows/kb-factory-jobs-workflow-v2.yaml
```

### Step 3: Test End-to-End

```bash
# Execute the workflow manually
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"trigger":"manual","github_repo":"your-org/knowledge-factory"}'
```

Monitor execution:
```bash
# Get execution ID from previous command output
EXECUTION_ID="<execution-id-from-output>"

# Watch execution progress
gcloud workflows executions describe $EXECUTION_ID \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1
```

## 📊 Architecture Flow

```
┌──────────────────────────────────────────────────────────────┐
│ PHASE 1: Parallel Terminology Downloads                     │
├──────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ SNOMED Job  │  │ RxNorm Job  │  │ LOINC Job   │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         │                │                 │                 │
│         ▼                ▼                 ▼                 │
│  ┌──────────────────────────────────────────────┐           │
│  │ Cloud Storage: workflow-results/             │           │
│  │  - snomed-latest.json (gcs_key, version)     │           │
│  │  - rxnorm-latest.json (gcs_key, version)     │           │
│  │  - loinc-latest.json (gcs_key, version)      │           │
│  └──────────────────────────────────────────────┘           │
└──────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────┐
│ PHASE 2: Read Results from GCS                              │
├──────────────────────────────────────────────────────────────┤
│  Workflow uses HTTP connector to fetch JSON files           │
│  Extracts: snomed_gcs_key, rxnorm_gcs_key, loinc_gcs_key   │
└──────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────┐
│ PHASE 3: GitHub Dispatcher with Environment Overrides       │
├──────────────────────────────────────────────────────────────┤
│  googleapis.run.v2.projects.locations.jobs.run               │
│    overrides:                                                │
│      containerOverrides:                                     │
│        - env:                                                │
│            PROJECT_ID: sincere-hybrid-477206-h2             │
│            ENVIRONMENT: production                           │
│            GITHUB_REPO: your-org/knowledge-factory          │
│            SECRET_NAME: kb7-github-token-production         │
│            SNOMED_KEY: snomed-ct/20251101/SnomedCT_...zip   │
│            RXNORM_KEY: rxnorm/10062025/RxNorm_...zip        │
│            LOINC_KEY: loinc/2.81/loinc-complete-2.81.zip    │
└──────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────┐
│ GitHub Actions Workflow Triggered                           │
├──────────────────────────────────────────────────────────────┤
│  Repository dispatch event: "terminology-update"             │
│  Payload includes download metadata and GCS paths            │
└──────────────────────────────────────────────────────────────┘
```

## ✅ Benefits of This Approach

1. **Decoupled Communication**: Downloaders and workflow communicate via GCS
2. **No Job Modifications**: GitHub dispatcher doesn't need static env vars
3. **Dynamic Data Passing**: Actual download results passed to GitHub
4. **Robust Error Handling**: Workflow handles missing result files gracefully
5. **Auditability**: Result files provide permanent record of each execution

## 🔍 Verification

After deployment, verify the integration:

### 1. Check Result Files
```bash
gsutil ls -lh gs://sincere-hybrid-477206-h2-kb-sources-production/workflow-results/
```

Expected output:
```
  <size>  <timestamp>  workflow-results/snomed-latest.json
  <size>  <timestamp>  workflow-results/rxnorm-latest.json
  <size>  <timestamp>  workflow-results/loinc-latest.json
```

### 2. View Result Content
```bash
gsutil cat gs://sincere-hybrid-477206-h2-kb-sources-production/workflow-results/snomed-latest.json
```

Expected:
```json
{
  "status": "skipped",
  "gcs_key": "snomed-ct/20251101/SnomedCT_InternationalRF2_PRODUCTION_20251101T120000Z.zip",
  "version": "2025-11-01T12:00:00",
  ...
}
```

### 3. Check Workflow Execution Logs
```bash
gcloud logging read \
  "resource.type=workflows.googleapis.com AND severity>=INFO" \
  --limit=50 \
  --format="table(timestamp,severity,textPayload)"
```

Look for:
- "Reading download results from GCS"
- "Extracted GCS keys - SNOMED: ... RxNorm: ... LOINC: ..."
- "Starting GitHub dispatcher job with download results"

### 4. Verify GitHub Dispatcher Received Data
```bash
gcloud logging read \
  "resource.type=cloud_run_job AND resource.labels.job_name=kb7-github-dispatcher-job-production" \
  --limit=20 \
  --format="table(timestamp,textPayload)"
```

Expected logs:
```
Download results received:
  SNOMED: snomed-ct/20251101/...
  RxNorm: rxnorm/10062025/...
  LOINC: loinc/2.81/...
```

## 📝 Configuration Notes

### GitHub Repository Setting

The workflow accepts a `github_repo` parameter. Set this when executing:

```bash
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"github_repo":"your-actual-org/your-actual-repo"}'
```

Or update the default in the workflow YAML (line 17):
```yaml
- github_repo: ${default(map.get(input, "github_repo"), "your-org/your-repo")}
```

## 🎯 Next Steps

1. **Deploy**: Run the rebuild script and deploy updated workflow
2. **Test**: Execute workflow end-to-end
3. **Configure GitHub**: Set up GitHub repository dispatch webhook handler
4. **Monitor**: Watch logs to verify GitHub integration works
5. **Schedule**: Set up Cloud Scheduler for automated runs

## 📚 Files Modified

1. `functions/snomed-downloader/main.py` - Added GCS result writing
2. `functions/rxnorm-downloader/main.py` - Added GCS result writing
3. `functions/loinc-downloader/main.py` - Added GCS result writing
4. `workflows/kb-factory-jobs-workflow-v2.yaml` - New workflow with GCS coordination

## 🔗 Related Documentation

- UMLS Release API: https://documentation.uts.nlm.nih.gov/automating-downloads.html#release-api
- Cloud Workflows HTTP Connector: https://cloud.google.com/workflows/docs/reference/stdlib/http
- Cloud Run Jobs API: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.jobs/run
- GitHub Repository Dispatch: https://docs.github.com/en/rest/repos/repos#create-a-repository-dispatch-event
