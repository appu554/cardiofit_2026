# KB-7 Knowledge Factory: Google Cloud Platform Implementation

**Objective**: Adapt the Serverless Knowledge Factory from AWS to Google Cloud Platform (GCP)

**Status**: Architecture Design & Implementation Guide
**Date**: November 24, 2025

---

## Executive Summary

This guide provides a complete GCP implementation strategy for the KB-7 Knowledge Factory, replacing AWS services with GCP equivalents while maintaining the same functionality, cost profile, and architectural patterns.

**Key Changes**:
- ☁️ S3 → Cloud Storage
- ⚡ Lambda → Cloud Functions (2nd gen) or Cloud Run
- 🔄 Step Functions → Cloud Workflows
- 🔐 Secrets Manager → Secret Manager
- 📊 CloudWatch → Cloud Monitoring + Cloud Logging
- ⏰ CloudWatch Events → Cloud Scheduler

**Cost Impact**: Similar to AWS ($12-24/month)
**Migration Effort**: 2-3 days (infrastructure templates + function code)
**Compatibility**: 100% feature parity with AWS implementation

---

## Service Mapping: AWS → GCP

| AWS Service | GCP Equivalent | Purpose | Notes |
|-------------|----------------|---------|-------|
| **S3** | **Cloud Storage** | Object storage | Similar pricing, lifecycle policies |
| **Lambda** | **Cloud Functions 2nd gen** | Serverless compute | 60-min timeout (vs 15-min Lambda) |
| **Lambda** (alternative) | **Cloud Run** | Containerized serverless | Better for large files (>10GB RAM) |
| **Step Functions** | **Cloud Workflows** | Orchestration | YAML-based, simpler than Step Functions |
| **Secrets Manager** | **Secret Manager** | Credential storage | Automatic rotation supported |
| **CloudWatch Events** | **Cloud Scheduler** | Cron triggers | More flexible cron expressions |
| **CloudWatch Logs** | **Cloud Logging** | Log aggregation | Better query interface |
| **CloudWatch Metrics** | **Cloud Monitoring** | Metrics & alerts | Integrated with Stackdriver |
| **IAM Roles** | **IAM Service Accounts** | Access control | Similar least-privilege model |
| **CloudFormation** | **Deployment Manager** or **Terraform** | Infrastructure as Code | Terraform recommended for multi-cloud |

---

## Architecture Diagram (GCP)

```
External Sources (NCTS, NIH, LOINC.org)
         ↓
  Cloud Scheduler (Monthly Cron)
         ↓
  Cloud Workflows (Orchestration)
         ↓
  ┌──────────────────────────────────────┐
  │   Cloud Functions (2nd gen)          │
  │   ├─ snomed-downloader (60-min)      │
  │   ├─ rxnorm-downloader (60-min)      │
  │   ├─ loinc-downloader (60-min)       │
  │   └─ github-dispatcher               │
  └──────────────────────────────────────┘
         ↓
  Cloud Storage Buckets
  ├─ gs://cardiofit-kb-sources
  └─ gs://cardiofit-kb-artifacts
         ↓
  GitHub Actions (Transformation Pipeline)
  ├─ SNOMED-OWL-Toolkit
  ├─ ROBOT (ELK Reasoner)
  └─ RxNorm/LOINC Converters
         ↓
  GraphDB (Self-hosted or Cloud Run)
         ↓
  Cloud SQL (PostgreSQL) - Metadata Registry
```

---

## Detailed Service Specifications

### 1. Cloud Storage Buckets

**Equivalent to**: AWS S3

**Configuration**:
```yaml
# gcp/terraform/storage.tf
resource "google_storage_bucket" "kb_sources" {
  name          = "cardiofit-kb-sources-${var.environment}"
  location      = "US"  # Multi-region for availability
  storage_class = "STANDARD"

  lifecycle_rule {
    condition {
      age = 180  # 180 days retention
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"  # Equivalent to S3 Standard-IA
    }
  }

  versioning {
    enabled = true
  }

  encryption {
    default_kms_key_name = google_kms_crypto_key.bucket_key.id
  }
}

resource "google_storage_bucket" "kb_artifacts" {
  name          = "cardiofit-kb-artifacts-${var.environment}"
  location      = "US"
  storage_class = "STANDARD"

  lifecycle_rule {
    condition {
      age = 365  # 1-year retention
    }
    action {
      type = "Delete"
    }
  }
}
```

**Cost**: $5-6/month for 200GB (same as AWS)

---

### 2. Cloud Functions (2nd Generation)

**Equivalent to**: AWS Lambda

**Why 2nd Gen?**
- 60-minute timeout (vs 15-minute in AWS Lambda) - **No ECS Fargate fallback needed!**
- Up to 16GB RAM
- Concurrency control
- Cloud Run infrastructure (containerized)

**Configuration**:
```yaml
# gcp/terraform/functions.tf
resource "google_cloudfunctions2_function" "snomed_downloader" {
  name     = "kb7-snomed-downloader-${var.environment}"
  location = "us-central1"

  build_config {
    runtime     = "python311"
    entry_point = "download_snomed"
    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = "snomed-downloader.zip"
      }
    }
  }

  service_config {
    max_instance_count    = 1
    available_memory      = "10Gi"  # 10GB for SNOMED
    timeout_seconds       = 3600    # 60 minutes (no timeout risk!)
    environment_variables = {
      SOURCE_BUCKET   = google_storage_bucket.kb_sources.name
      SECRET_PROJECT  = var.project_id
    }
    service_account_email = google_service_account.kb_functions.email
  }
}

resource "google_cloudfunctions2_function" "rxnorm_downloader" {
  name     = "kb7-rxnorm-downloader-${var.environment}"
  location = "us-central1"

  service_config {
    available_memory = "3Gi"
    timeout_seconds  = 3600  # 60 minutes
  }
}

resource "google_cloudfunctions2_function" "loinc_downloader" {
  name     = "kb7-loinc-downloader-${var.environment}"
  location = "us-central1"

  service_config {
    available_memory = "2Gi"
    timeout_seconds  = 1800  # 30 minutes
  }
}

resource "google_cloudfunctions2_function" "github_dispatcher" {
  name     = "kb7-github-dispatcher-${var.environment}"
  location = "us-central1"

  service_config {
    available_memory = "1Gi"
    timeout_seconds  = 300  # 5 minutes
  }
}
```

**Key Advantage**: 60-minute timeout eliminates the Lambda timeout risk for SNOMED downloads!

**Cost**: $1.80/month (cheaper than Lambda due to per-100ms billing)

---

### 3. Cloud Workflows

**Equivalent to**: AWS Step Functions

**Configuration**:
```yaml
# gcp/workflows/kb-factory-workflow.yaml
main:
  params: [input]
  steps:
    - init:
        assign:
          - project_id: ${sys.get_env("GOOGLE_CLOUD_PROJECT")}
          - timestamp: ${text.split(time.format(sys.now()), "T")[0]}

    - parallel_downloads:
        parallel:
          branches:
            - snomed_branch:
                steps:
                  - call_snomed_downloader:
                      call: googleapis.cloudfunctions.v2.projects.locations.functions.call
                      args:
                        name: ${"projects/" + project_id + "/locations/us-central1/functions/kb7-snomed-downloader-production"}
                        body:
                          timestamp: ${timestamp}
                      result: snomed_result

            - rxnorm_branch:
                steps:
                  - call_rxnorm_downloader:
                      call: googleapis.cloudfunctions.v2.projects.locations.functions.call
                      args:
                        name: ${"projects/" + project_id + "/locations/us-central1/functions/kb7-rxnorm-downloader-production"}
                        body:
                          timestamp: ${timestamp}
                      result: rxnorm_result

            - loinc_branch:
                steps:
                  - call_loinc_downloader:
                      call: googleapis.cloudfunctions.v2.projects.locations.functions.call
                      args:
                        name: ${"projects/" + project_id + "/locations/us-central1/functions/kb7-loinc-downloader-production"}
                        body:
                          timestamp: ${timestamp}
                      result: loinc_result

    - dispatch_github:
        call: googleapis.cloudfunctions.v2.projects.locations.functions.call
        args:
          name: ${"projects/" + project_id + "/locations/us-central1/functions/kb7-github-dispatcher-production"}
          body:
            snomed_key: ${snomed_result.body.s3_key}
            rxnorm_key: ${rxnorm_result.body.s3_key}
            loinc_key: ${loinc_result.body.s3_key}
        result: dispatch_result

    - return_result:
        return:
          status: "success"
          downloads: ${parallel_downloads}
          github_dispatch: ${dispatch_result}
```

**Advantages over Step Functions**:
- YAML-based (simpler than JSON state machine)
- Built-in retry logic (no manual configuration)
- Better error handling
- Native GCP service integration

**Cost**: $0.50/month (same as Step Functions)

---

### 4. Secret Manager

**Equivalent to**: AWS Secrets Manager

**Configuration**:
```yaml
# gcp/terraform/secrets.tf
resource "google_secret_manager_secret" "nhs_trud_api_key" {
  secret_id = "kb7-nhs-trud-api-key"

  replication {
    automatic = true
  }

  labels = {
    service     = "kb7-knowledge-factory"
    rotation    = "90days"
    environment = var.environment
  }
}

resource "google_secret_manager_secret_version" "nhs_trud_api_key" {
  secret      = google_secret_manager_secret.nhs_trud_api_key.id
  secret_data = var.nhs_trud_api_key  # Provided via terraform.tfvars
}

# Rotation reminder (Cloud Scheduler + Cloud Function)
resource "google_cloud_scheduler_job" "secret_rotation_reminder" {
  name      = "kb7-secret-rotation-reminder"
  schedule  = "0 9 1 * *"  # 1st of month, 9 AM
  time_zone = "America/New_York"

  http_target {
    uri         = google_cloudfunctions2_function.rotation_reminder.service_config[0].uri
    http_method = "POST"

    oidc_token {
      service_account_email = google_service_account.scheduler.email
    }
  }
}
```

**IAM Binding** (Least Privilege):
```yaml
resource "google_secret_manager_secret_iam_member" "function_access" {
  secret_id = google_secret_manager_secret.nhs_trud_api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.kb_functions.email}"
}
```

**Cost**: $0.40/month (4 secrets × $0.10/secret)

---

### 5. Cloud Scheduler

**Equivalent to**: CloudWatch Events

**Configuration**:
```yaml
# gcp/terraform/scheduler.tf
resource "google_cloud_scheduler_job" "monthly_terminology_update" {
  name      = "kb7-monthly-terminology-update"
  schedule  = "0 2 1 * *"  # 1st of month, 2 AM UTC
  time_zone = "UTC"

  http_target {
    uri         = google_workflows_workflow.kb_factory.service_account_uri
    http_method = "POST"

    body = base64encode(jsonencode({
      trigger    = "scheduled"
      timestamp  = "auto"
    }))

    headers = {
      "Content-Type" = "application/json"
    }

    oidc_token {
      service_account_email = google_service_account.scheduler.email
    }
  }

  retry_config {
    retry_count = 3
    min_backoff_duration = "300s"  # 5 minutes
    max_backoff_duration = "3600s" # 1 hour
  }
}
```

**Cost**: $0.10/month (1 job)

---

### 6. Cloud Monitoring & Logging

**Equivalent to**: CloudWatch

**Monitoring Configuration**:
```yaml
# gcp/terraform/monitoring.tf
resource "google_monitoring_alert_policy" "function_duration" {
  display_name = "KB-7 Function Duration Alert"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Function Duration > 50 minutes"

    condition_threshold {
      filter          = "resource.type=\"cloud_function\" AND resource.labels.function_name=monitoring.regex().full_match(\"kb7-.*-downloader.*\") AND metric.type=\"cloudfunctions.googleapis.com/function/execution_times\""
      duration        = "60s"
      comparison      = "COMPARISON_GT"
      threshold_value = 3000000  # 50 minutes in milliseconds

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.slack.name]

  alert_strategy {
    auto_close = "3600s"  # Auto-close after 1 hour if resolved
  }
}

resource "google_monitoring_notification_channel" "slack" {
  display_name = "KB-7 Slack Channel"
  type         = "slack"

  labels = {
    channel_name = "#kb7-alerts"
  }

  sensitive_labels {
    auth_token = var.slack_webhook_token
  }
}
```

**Log-Based Metrics**:
```yaml
resource "google_logging_metric" "download_success" {
  name   = "kb7_download_success_count"
  filter = "resource.type=\"cloud_function\" AND jsonPayload.message=\"Download complete\""

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"

    labels {
      key         = "terminology"
      value_type  = "STRING"
      description = "Terminology type (SNOMED/RxNorm/LOINC)"
    }
  }

  label_extractors = {
    "terminology" = "EXTRACT(jsonPayload.terminology)"
  }
}
```

**Cost**: $0.80/month (logs + metrics)

---

## Python Function Code (GCP Adaptation)

### SNOMED Downloader (Cloud Function)

```python
# gcp/functions/snomed-downloader/main.py
import os
import hashlib
from google.cloud import storage
from google.cloud import secretmanager
import requests
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def download_snomed(request):
    """
    Cloud Function to download SNOMED CT from NHS TRUD API
    Streams directly to Cloud Storage (no local disk needed)
    """

    # Get configuration from environment
    project_id = os.environ['GOOGLE_CLOUD_PROJECT']
    source_bucket = os.environ['SOURCE_BUCKET']

    # Retrieve API key from Secret Manager
    secret_client = secretmanager.SecretManagerServiceClient()
    secret_name = f"projects/{project_id}/secrets/kb7-nhs-trud-api-key/versions/latest"
    api_key = secret_client.access_secret_version(name=secret_name).payload.data.decode('UTF-8')

    # NHS TRUD API endpoint
    trud_url = "https://isd.digital.nhs.uk/trud/api/v1/keys/{}/items/SNOMED/releases".format(api_key)

    # Get latest release metadata
    logger.info("Fetching SNOMED release metadata from NHS TRUD")
    response = requests.get(trud_url, timeout=30)
    response.raise_for_status()

    releases = response.json()['releases']
    latest_release = sorted(releases, key=lambda x: x['releaseDate'], reverse=True)[0]

    download_url = latest_release['archiveFileUrl']
    version = latest_release['releaseDate'].replace('-', '')
    file_size = latest_release['archiveFileSize']

    logger.info(f"Downloading SNOMED version {version} ({file_size} bytes)")

    # Stream download to Cloud Storage (chunked upload)
    storage_client = storage.Client()
    bucket = storage_client.bucket(source_bucket)
    blob_name = f"snomed-ct/{version}/SnomedCT_international_{version}.zip"
    blob = bucket.blob(blob_name)

    # Stream download with hash calculation
    hasher = hashlib.sha256()

    with requests.get(download_url, stream=True, timeout=3600) as r:
        r.raise_for_status()

        # Upload in chunks (no memory overflow!)
        with blob.open("wb") as f:
            for chunk in r.iter_content(chunk_size=10 * 1024 * 1024):  # 10MB chunks
                if chunk:
                    f.write(chunk)
                    hasher.update(chunk)

    sha256_hash = hasher.hexdigest()

    # Set metadata
    blob.metadata = {
        'version': version,
        'source': 'NHS TRUD API',
        'sha256': sha256_hash,
        'file_size': str(file_size)
    }
    blob.patch()

    logger.info(f"Successfully uploaded SNOMED to gs://{source_bucket}/{blob_name}")

    return {
        'status': 'success',
        'gcs_uri': f"gs://{source_bucket}/{blob_name}",
        'version': version,
        'sha256': sha256_hash,
        'file_size': file_size
    }
```

**Key Differences from AWS Lambda**:
- ✅ `google.cloud.storage` instead of `boto3.s3`
- ✅ `blob.open("wb")` for streaming (simpler than multipart upload!)
- ✅ 60-minute timeout (no ECS fallback needed)
- ✅ Automatic retry by Cloud Functions (no manual configuration)

---

## Cost Comparison: AWS vs GCP

| Service | AWS Monthly Cost | GCP Monthly Cost | Savings |
|---------|------------------|------------------|---------|
| **Object Storage** | $5.00 | $5.50 | -$0.50 |
| **Serverless Functions** | $1.50 | $1.20 | +$0.30 |
| **Orchestration** | $0.50 | $0.50 | $0.00 |
| **Secrets** | $1.60 | $0.40 | +$1.20 |
| **Logging** | $0.50 | $0.60 | -$0.10 |
| **Scheduler** | $0.00 | $0.10 | -$0.10 |
| **Data Transfer** | $1.00 | $1.20 | -$0.20 |
| **GitHub Actions** | $12.00 | $12.00 | $0.00 |
| **TOTAL** | **$22.10/month** | **$21.50/month** | **+$0.60/month** |

**Annual**: AWS $265 vs GCP $258 (**$7/year savings**)

**Verdict**: Near-identical costs with slight GCP advantage

---

## Migration Strategy

### Phase 1: Infrastructure Setup (Day 1)

1. **Create GCP Project**
   ```bash
   gcloud projects create cardiofit-kb7-production --name="KB-7 Knowledge Factory"
   gcloud config set project cardiofit-kb7-production
   ```

2. **Enable Required APIs**
   ```bash
   gcloud services enable \
     cloudfunctions.googleapis.com \
     workflows.googleapis.com \
     storage.googleapis.com \
     secretmanager.googleapis.com \
     cloudscheduler.googleapis.com \
     monitoring.googleapis.com \
     logging.googleapis.com
   ```

3. **Deploy Infrastructure with Terraform**
   ```bash
   cd gcp/terraform
   terraform init
   terraform plan -out=tfplan
   terraform apply tfplan
   ```

### Phase 2: Function Deployment (Day 2)

1. **Package Functions**
   ```bash
   cd gcp/functions/snomed-downloader
   zip -r function.zip main.py requirements.txt
   ```

2. **Deploy to Cloud Functions**
   ```bash
   gcloud functions deploy kb7-snomed-downloader \
     --gen2 \
     --runtime=python311 \
     --region=us-central1 \
     --source=. \
     --entry-point=download_snomed \
     --memory=10Gi \
     --timeout=3600s \
     --service-account=kb7-functions@cardiofit-kb7-production.iam.gserviceaccount.com
   ```

3. **Test Functions**
   ```bash
   gcloud functions call kb7-snomed-downloader \
     --gen2 \
     --region=us-central1 \
     --data='{"test": true}'
   ```

### Phase 3: Workflow Deployment (Day 3)

1. **Deploy Workflow**
   ```bash
   gcloud workflows deploy kb7-factory-workflow \
     --source=gcp/workflows/kb-factory-workflow.yaml \
     --location=us-central1 \
     --service-account=kb7-workflows@cardiofit-kb7-production.iam.gserviceaccount.com
   ```

2. **Test Workflow**
   ```bash
   gcloud workflows execute kb7-factory-workflow \
     --location=us-central1 \
     --data='{"trigger":"manual-test"}'
   ```

3. **Enable Scheduler**
   ```bash
   # Scheduler created by Terraform, verify it's enabled
   gcloud scheduler jobs describe kb7-monthly-terminology-update \
     --location=us-central1
   ```

---

## GCP-Specific Advantages

### 1. **60-Minute Function Timeout**
- **AWS Lambda**: 15-minute limit requires ECS Fargate fallback
- **GCP Cloud Functions 2nd gen**: 60-minute limit handles SNOMED downloads natively
- **Result**: Simpler architecture, no containerized fallback needed

### 2. **Simpler Streaming Upload**
```python
# AWS (complex multipart upload)
multipart = s3.create_multipart_upload(Bucket=bucket, Key=key)
parts = []
for i, chunk in enumerate(chunks):
    part = s3.upload_part(Bucket=bucket, Key=key, PartNumber=i+1, UploadId=upload_id, Body=chunk)
    parts.append({'PartNumber': i+1, 'ETag': part['ETag']})
s3.complete_multipart_upload(Bucket=bucket, Key=key, UploadId=upload_id, MultipartUpload={'Parts': parts})

# GCP (simple streaming)
with blob.open("wb") as f:
    for chunk in response.iter_content(chunk_size=10MB):
        f.write(chunk)
```

### 3. **Better Workflow YAML Syntax**
- **AWS Step Functions**: Complex JSON state machine
- **GCP Cloud Workflows**: Simple YAML with built-in retry/error handling

### 4. **Free Tier Benefits**
- Cloud Functions: 2M invocations/month free (vs 1M in Lambda)
- Cloud Storage: 5GB free (vs 5GB in S3)
- Cloud Monitoring: 150MB logs/month free

---

## Hybrid AWS + GCP Strategy

### **Recommended: Multi-Cloud for Redundancy**

```yaml
Primary: AWS (existing implementation)
Backup: GCP (this implementation)

Strategy:
- Deploy both AWS and GCP infrastructures
- Use Cloud Scheduler to trigger AWS Step Functions (HTTP POST)
- If AWS fails: Manual failover to GCP workflow
- GitHub Actions remains cloud-agnostic (works with both)
```

**Benefits**:
- ✅ Cloud provider redundancy
- ✅ No vendor lock-in
- ✅ Test both implementations in parallel
- ✅ Total cost: $43/month (both clouds)

---

## Next Steps

### **Option A: GCP-Only Implementation**
1. Create GCP project
2. Deploy Terraform infrastructure (gcp/terraform/)
3. Deploy Cloud Functions (gcp/functions/)
4. Test end-to-end workflow
5. Enable Cloud Scheduler

**Timeline**: 3 days
**Cost**: $21.50/month

### **Option B: Hybrid AWS + GCP**
1. Keep existing AWS implementation
2. Deploy GCP as backup
3. Test both pipelines monthly
4. Document failover procedures

**Timeline**: 1 day (GCP deployment only)
**Cost**: $43/month (both clouds)

### **Option C: Migrate AWS → GCP**
1. Deploy GCP infrastructure
2. Run both pipelines in parallel for 3 months
3. Validate GCP reliability
4. Decommission AWS

**Timeline**: 3 months validation period
**Cost**: $43/month during migration, then $21.50/month

---

## Files to Create (GCP Implementation)

```
gcp/
├── terraform/
│   ├── main.tf                      # Provider configuration
│   ├── storage.tf                   # Cloud Storage buckets
│   ├── functions.tf                 # Cloud Functions 2nd gen
│   ├── workflows.tf                 # Cloud Workflows
│   ├── secrets.tf                   # Secret Manager
│   ├── scheduler.tf                 # Cloud Scheduler
│   ├── monitoring.tf                # Cloud Monitoring alerts
│   ├── iam.tf                       # Service accounts + IAM
│   └── variables.tf                 # Input variables
│
├── functions/
│   ├── snomed-downloader/
│   │   ├── main.py
│   │   └── requirements.txt
│   ├── rxnorm-downloader/
│   │   ├── main.py
│   │   └── requirements.txt
│   ├── loinc-downloader/
│   │   ├── main.py
│   │   └── requirements.txt
│   └── github-dispatcher/
│       ├── main.py
│       └── requirements.txt
│
├── workflows/
│   └── kb-factory-workflow.yaml
│
├── scripts/
│   ├── deploy-infrastructure.sh
│   ├── test-functions.sh
│   └── teardown-infrastructure.sh
│
└── README.md
```

---

## Conclusion

**GCP Offers**:
- ✅ Near-identical cost to AWS ($21.50 vs $22.10/month)
- ✅ 60-minute function timeout (no ECS Fargate needed)
- ✅ Simpler streaming upload (blob.open() vs multipart)
- ✅ Better workflow YAML syntax
- ✅ Slightly better free tier

**Recommendation**:
1. **If starting fresh**: Use GCP (simpler implementation)
2. **If AWS already deployed**: Keep AWS (migration not worth effort)
3. **For redundancy**: Deploy both (hybrid strategy)

**Contact**: kb7-architecture@cardiofit.ai
