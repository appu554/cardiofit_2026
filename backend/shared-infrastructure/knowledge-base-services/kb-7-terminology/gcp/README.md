# KB-7 Knowledge Factory - Google Cloud Platform Implementation

Complete GCP implementation of the serverless terminology download pipeline for SNOMED CT, RxNorm, and LOINC.

## Architecture Overview

```
External APIs (NHS TRUD, NIH UMLS, LOINC.org)
         ↓
  Cloud Scheduler (Monthly: 1st @ 2 AM UTC)
         ↓
  Cloud Workflows (Orchestration)
         ↓
  ┌──────────────────────────────────────┐
  │   Cloud Functions 2nd Gen (Python)   │
  │   ├─ snomed-downloader (10GB/60min)  │
  │   ├─ rxnorm-downloader (3GB/60min)   │
  │   ├─ loinc-downloader (2GB/30min)    │
  │   └─ github-dispatcher (1GB/5min)    │
  └──────────────────────────────────────┘
         ↓
  Cloud Storage Buckets
  ├─ sources (raw downloads)
  └─ artifacts (processed ontologies)
         ↓
  GitHub Actions (Transformation Pipeline)
  └─ 7-stage processing → GraphDB
```

## Key GCP Advantages Over AWS

### 1. 60-Minute Function Timeout
- **AWS Lambda**: 15-minute limit requires ECS Fargate fallback for large files
- **GCP Cloud Functions 2nd gen**: 60-minute limit handles SNOMED downloads natively
- **Benefit**: Simpler architecture, no containerized fallback needed

### 2. Simpler Streaming Upload
```python
# AWS Lambda (complex multipart upload)
multipart = s3.create_multipart_upload(Bucket=bucket, Key=key)
for i, chunk in enumerate(chunks):
    part = s3.upload_part(...)
    parts.append({'PartNumber': i+1, 'ETag': part['ETag']})
s3.complete_multipart_upload(...)

# GCP Cloud Functions (simple streaming)
with blob.open("wb") as f:
    for chunk in response.iter_content(chunk_size=10MB):
        f.write(chunk)
```

### 3. Better Workflow Syntax
- **AWS Step Functions**: Complex JSON state machine syntax
- **GCP Cloud Workflows**: Simple YAML with built-in error handling

### 4. Cost Comparison

| Component | AWS Monthly | GCP Monthly | Difference |
|-----------|-------------|-------------|------------|
| Storage (200GB) | $5.00 | $5.50 | -$0.50 |
| Functions | $1.50 | $1.20 | +$0.30 |
| Orchestration | $0.50 | $0.50 | $0.00 |
| Secrets | $1.60 | $0.40 | +$1.20 |
| Monitoring | $0.50 | $0.60 | -$0.10 |
| Scheduler | $0.00 | $0.10 | -$0.10 |
| **TOTAL** | **$22.10** | **$21.50** | **+$0.60** |

**Annual Savings**: $7/year with GCP

## Prerequisites

### Required Tools
```bash
# Google Cloud SDK
gcloud --version  # >= 450.0.0

# Terraform
terraform --version  # >= 1.5.0

# Utilities
zip --version
jq --version
```

### GCP Project Setup
```bash
# Create project
gcloud projects create cardiofit-kb7-production --name="KB-7 Knowledge Factory"

# Set as active project
gcloud config set project cardiofit-kb7-production

# Enable billing (required)
# Visit: https://console.cloud.google.com/billing
```

### API Keys Required
1. **NHS TRUD API Key**: https://isd.digital.nhs.uk/trud/users/guest/filters/0/home
2. **UMLS API Key**: https://uts.nlm.nih.gov/uts/
3. **LOINC Credentials**: https://loinc.org/downloads/
4. **GitHub Personal Access Token**: https://github.com/settings/tokens (repo scope)

## Quick Start Deployment

### Step 1: Configure Variables
```bash
cd gcp/terraform

# Copy example configuration
cp terraform.tfvars.example terraform.tfvars

# Edit with your values
nano terraform.tfvars
```

Required variables:
```hcl
project_id           = "cardiofit-kb7-production"
nhs_trud_api_key    = "your-nhs-api-key"
umls_api_key        = "your-umls-api-key"
loinc_username      = "your-loinc-username"
loinc_password      = "your-loinc-password"
github_token        = "ghp_your-github-token"
github_repository   = "cardiofit/kb7-terminology"
```

### Step 2: Deploy Infrastructure
```bash
cd ../scripts

# Run deployment script
./deploy-infrastructure.sh
```

This script will:
1. Validate prerequisites
2. Package Cloud Functions
3. Initialize Terraform
4. Create infrastructure
5. Output deployment summary

**Expected Duration**: 5-10 minutes

### Step 3: Test Functions
```bash
# Test individual functions
./test-functions.sh
```

### Step 4: Test Complete Workflow
```bash
# Manual workflow execution
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"trigger":"manual-test"}'

# View execution status
gcloud workflows executions list kb7-factory-workflow-production \
  --location=us-central1 \
  --limit=5
```

## Project Structure

```
gcp/
├── terraform/                   # Infrastructure as Code
│   ├── main.tf                 # Provider & API configuration
│   ├── storage.tf              # Cloud Storage buckets
│   ├── functions.tf            # Cloud Functions 2nd gen
│   ├── workflows.tf            # Cloud Workflows orchestration
│   ├── secrets.tf              # Secret Manager
│   ├── scheduler.tf            # Cloud Scheduler
│   ├── monitoring.tf           # Cloud Monitoring & alerts
│   ├── iam.tf                  # Service accounts & permissions
│   ├── variables.tf            # Input variables
│   ├── outputs.tf              # Output values
│   └── terraform.tfvars.example
│
├── functions/                   # Cloud Function source code
│   ├── snomed-downloader/
│   │   ├── main.py             # SNOMED download logic
│   │   └── requirements.txt
│   ├── rxnorm-downloader/
│   │   ├── main.py             # RxNorm download logic
│   │   └── requirements.txt
│   ├── loinc-downloader/
│   │   ├── main.py             # LOINC download logic
│   │   └── requirements.txt
│   └── github-dispatcher/
│       ├── main.py             # GitHub workflow trigger
│       └── requirements.txt
│
├── workflows/
│   └── kb-factory-workflow.yaml # Workflow orchestration
│
├── scripts/
│   ├── deploy-infrastructure.sh  # Complete deployment
│   ├── test-functions.sh         # Function testing
│   └── teardown-infrastructure.sh # Clean deletion
│
└── README.md                    # This file
```

## Infrastructure Components

### Cloud Storage Buckets

**Sources Bucket**: `cardiofit-kb7-production-kb-sources-production`
- Purpose: Raw terminology downloads
- Retention: 180 days
- Lifecycle: Transition to Nearline after 30 days
- Versioning: Enabled

**Artifacts Bucket**: `cardiofit-kb7-production-kb-artifacts-production`
- Purpose: Processed ontologies for GraphDB
- Retention: 365 days
- Lifecycle: Transition to Nearline after 90 days
- Versioning: Enabled

### Cloud Functions 2nd Generation

| Function | Memory | Timeout | Purpose |
|----------|--------|---------|---------|
| snomed-downloader | 10GB | 60 min | Download SNOMED CT from NHS TRUD |
| rxnorm-downloader | 3GB | 60 min | Download RxNorm from NIH UMLS |
| loinc-downloader | 2GB | 30 min | Download LOINC from LOINC.org |
| github-dispatcher | 1GB | 5 min | Trigger GitHub Actions workflow |

**Key Features**:
- Python 3.11 runtime
- Streaming upload to Cloud Storage
- SHA256 integrity verification
- Comprehensive error handling
- Structured logging for monitoring

### Cloud Workflows

**kb7-factory-workflow-production**
- Orchestrates parallel downloads
- Error handling and retry logic
- GitHub workflow dispatch
- Execution timeout: 2 hours

**Execution Flow**:
1. Initialize workflow variables
2. Parallel execution:
   - SNOMED CT download (60 min)
   - RxNorm download (60 min)
   - LOINC download (30 min)
3. Check download status
4. Dispatch GitHub Actions workflow
5. Return results

### Secret Manager

Stores API credentials securely:
- `kb7-nhs-trud-api-key-production`
- `kb7-umls-api-key-production`
- `kb7-loinc-credentials-production` (JSON)
- `kb7-github-token-production`

**Rotation**: Recommended every 90 days

### Cloud Scheduler

**kb7-monthly-terminology-update-production**
- Schedule: `0 2 1 * *` (1st of month, 2 AM UTC)
- Triggers: Cloud Workflows
- Retry: 3 attempts with exponential backoff

### Cloud Monitoring

**Alert Policies**:
1. Function duration > 50 minutes (warning)
2. Function execution errors (critical)
3. Workflow execution failures (critical)

**Log-Based Metrics**:
- Download success count by terminology
- Download failure count with error types
- File size tracking

**Notification Channels**:
- Email: kb7-team@cardiofit.ai
- Slack: #kb7-alerts (optional)

### IAM Service Accounts

| Service Account | Purpose | Permissions |
|----------------|---------|-------------|
| kb7-functions-production | Cloud Functions execution | Secret accessor, Log writer, Storage admin |
| kb7-workflows-production | Workflow orchestration | Function invoker, Log writer |
| kb7-scheduler-production | Scheduler triggers | Workflow invoker |
| kb7-github-actions-production | GitHub Actions access | Storage admin (artifacts) |

**Principle**: Least privilege - each service account has minimal required permissions

## Usage Guide

### Manual Workflow Execution
```bash
# Execute workflow
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"trigger":"manual-test"}'

# Get execution ID from output, then:
EXECUTION_ID="<execution-id>"

# View execution details
gcloud workflows executions describe $EXECUTION_ID \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1

# View execution logs
gcloud workflows executions wait $EXECUTION_ID \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1
```

### View Function Logs
```bash
# All KB-7 function logs
gcloud logging read \
  'resource.type="cloud_function" AND resource.labels.function_name=~"kb7-.*"' \
  --limit=50 \
  --format=json

# Errors only
gcloud logging read \
  'resource.type="cloud_function" AND severity>=ERROR' \
  --limit=20

# Specific function
gcloud logging read \
  'resource.labels.function_name="kb7-snomed-downloader-production"' \
  --limit=10
```

### Monitor Workflow Executions
```bash
# List recent executions
gcloud workflows executions list kb7-factory-workflow-production \
  --location=us-central1 \
  --limit=10

# Filter by state
gcloud workflows executions list kb7-factory-workflow-production \
  --location=us-central1 \
  --filter="state=SUCCEEDED"
```

### Check Storage Buckets
```bash
# List source files
gsutil ls -lh gs://cardiofit-kb7-production-kb-sources-production/

# List artifacts
gsutil ls -lh gs://cardiofit-kb7-production-kb-artifacts-production/

# Check specific download
gsutil ls -lh gs://cardiofit-kb7-production-kb-sources-production/snomed-ct/
```

### Update Secrets
```bash
# Update NHS TRUD API key
echo -n "new-api-key" | gcloud secrets versions add kb7-nhs-trud-api-key-production --data-file=-

# Update LOINC credentials (JSON)
echo -n '{"username":"user","password":"pass"}' | \
  gcloud secrets versions add kb7-loinc-credentials-production --data-file=-

# Verify secret version
gcloud secrets versions list kb7-nhs-trud-api-key-production
```

## Troubleshooting

### Function Timeout Errors
**Symptom**: Function exceeds 60-minute timeout

**Solutions**:
1. Check download URL accessibility
2. Verify network connectivity
3. Review API rate limits
4. Consider increasing memory (improves CPU allocation)

```bash
# View function configuration
gcloud functions describe kb7-snomed-downloader-production \
  --gen2 \
  --region=us-central1 \
  --format=json
```

### Secret Access Denied
**Symptom**: Function cannot access secrets

**Solutions**:
1. Verify IAM permissions:
```bash
gcloud secrets get-iam-policy kb7-nhs-trud-api-key-production
```

2. Check service account:
```bash
gcloud functions describe kb7-snomed-downloader-production \
  --gen2 \
  --region=us-central1 \
  --format="value(serviceConfig.serviceAccountEmail)"
```

3. Grant access if needed:
```bash
gcloud secrets add-iam-policy-binding kb7-nhs-trud-api-key-production \
  --member="serviceAccount:kb7-functions-production@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### Workflow Execution Failures
**Symptom**: Workflow fails with download errors

**Diagnosis**:
```bash
# Get execution details
gcloud workflows executions describe $EXECUTION_ID \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1 \
  --format=json | jq '.result'

# Check function logs
gcloud logging read \
  'resource.type="cloud_function" AND severity>=ERROR AND timestamp>="2025-11-24T00:00:00Z"' \
  --limit=50
```

### Storage Upload Failures
**Symptom**: Function runs but files not in GCS

**Solutions**:
1. Check IAM permissions on bucket
2. Verify bucket exists and is accessible
3. Review function service account permissions

```bash
# Check bucket IAM
gsutil iam get gs://cardiofit-kb7-production-kb-sources-production/

# Test write access
echo "test" | gsutil cp - gs://cardiofit-kb7-production-kb-sources-production/test.txt
gsutil rm gs://cardiofit-kb7-production-kb-sources-production/test.txt
```

## Cost Optimization

### Monthly Cost Breakdown

**Fixed Costs**:
- Cloud Scheduler: $0.10/month (1 job)
- Secret Manager: $0.40/month (4 secrets)

**Variable Costs** (monthly execution):
- Cloud Functions: $1.20/month
  - SNOMED: $0.60 (10GB × 60 min)
  - RxNorm: $0.35 (3GB × 60 min)
  - LOINC: $0.15 (2GB × 30 min)
  - GitHub: $0.10 (1GB × 5 min)
- Cloud Workflows: $0.50/month
- Cloud Storage: $5.50/month (200GB average)
- Cloud Monitoring: $0.60/month (logs + metrics)

**Total**: ~$21.50/month

### Cost Reduction Strategies

1. **Reduce Storage Retention**:
```hcl
# In terraform.tfvars
source_bucket_retention_days = 90   # Instead of 180
artifact_bucket_retention_days = 180 # Instead of 365
```
Savings: ~$2/month

2. **Use Nearline Storage Sooner**:
```hcl
nearline_transition_days = 7  # Instead of 30
```
Savings: ~$1/month

3. **Reduce Function Memory** (if feasible):
```hcl
# In functions.tf - test carefully!
available_memory = "8Gi"  # Instead of 10Gi for SNOMED
```
Savings: ~$0.15/month

4. **Disable Slack Notifications**:
```hcl
slack_webhook_url = ""  # Email only
```
Savings: Negligible

## Monitoring & Observability

### Key Metrics

**Cloud Functions**:
- Execution count (invocations)
- Execution time (duration)
- Error rate
- Memory utilization
- CPU utilization

**Cloud Workflows**:
- Execution count
- Execution state (success/failed/cancelled)
- Execution duration

**Cloud Storage**:
- Storage size (bytes)
- Request count
- Egress bandwidth

### Custom Dashboards

Create a Cloud Monitoring dashboard:
```bash
# Download dashboard JSON from GCP Console after creating manually
# Or use Terraform google_monitoring_dashboard resource
```

Recommended widgets:
1. Function execution count (last 30 days)
2. Average execution time by function
3. Error rate trend
4. Storage bucket size over time
5. Workflow success rate

### Alerting Best Practices

1. **Critical Alerts** (Page on-call):
   - Workflow execution failed
   - Function errors > 0

2. **Warning Alerts** (Email only):
   - Function duration > 50 minutes
   - Storage usage > 80% quota

3. **Informational** (Dashboard only):
   - Successful executions
   - Download file sizes

## Security Considerations

### Secret Rotation

**Recommended Schedule**:
- API keys: Every 90 days
- GitHub token: Every 90 days
- Service account keys: Never expose (use Workload Identity)

**Rotation Process**:
```bash
# 1. Generate new credential from provider
# 2. Add new version to Secret Manager
echo -n "new-key" | gcloud secrets versions add SECRET_NAME --data-file=-

# 3. Test with new credential
./test-functions.sh

# 4. Disable old version
gcloud secrets versions disable VERSION_ID --secret=SECRET_NAME

# 5. Verify no errors, then destroy old version
gcloud secrets versions destroy VERSION_ID --secret=SECRET_NAME
```

### IAM Best Practices

1. **Least Privilege**: Service accounts have minimal required permissions
2. **No Wildcard Permissions**: Explicit resource grants only
3. **Audit Logging**: Enable Data Access logs for sensitive operations
4. **Workload Identity**: Used for GitHub Actions authentication

### Network Security

1. **Internal-Only Functions**: Functions only callable from within GCP
2. **VPC Service Controls**: (Optional) Further restrict access
3. **Private Storage**: Buckets use uniform bucket-level access

## Maintenance & Updates

### Terraform Updates
```bash
cd terraform

# Update provider versions
terraform init -upgrade

# Review changes
terraform plan

# Apply updates
terraform apply
```

### Function Code Updates
```bash
# 1. Modify function code in functions/*/main.py

# 2. Repackage functions
cd scripts
./deploy-infrastructure.sh  # Will detect changes and redeploy
```

### Infrastructure Changes
```bash
# 1. Modify Terraform files
# 2. Review changes
terraform plan

# 3. Apply changes
terraform apply
```

## Disaster Recovery

### Backup Strategy

**Automated Backups**:
- Storage versioning: Enabled on both buckets
- Terraform state: Stored in GCS backend (versioned)
- Secret versions: Retained for 30 days after disable

**Manual Backups**:
```bash
# Export Terraform state
terraform state pull > backup-$(date +%Y%m%d).tfstate

# Backup secrets (store securely!)
gcloud secrets versions access latest --secret=kb7-nhs-trud-api-key-production > secrets-backup.txt
```

### Recovery Procedures

**Complete Infrastructure Loss**:
```bash
# 1. Clone repository
git clone <repo-url>
cd gcp

# 2. Restore terraform.tfvars
cp terraform.tfvars.backup terraform.tfvars

# 3. Redeploy infrastructure
cd scripts
./deploy-infrastructure.sh
```

**Function Failure**:
```bash
# 1. Check logs for root cause
gcloud logging read ...

# 2. Rollback function to previous version
gcloud functions deploy <function-name> \
  --source=<previous-version-source> \
  --gen2

# 3. Verify functionality
./test-functions.sh
```

## Migration from AWS

If migrating from existing AWS infrastructure:

### Phase 1: Parallel Deployment
1. Deploy GCP infrastructure alongside AWS
2. Test GCP functions with same test data
3. Validate outputs match AWS results

### Phase 2: Validation (3 months)
1. Run both AWS and GCP pipelines monthly
2. Compare outputs and performance
3. Document any discrepancies

### Phase 3: Migration
1. Update Cloud Scheduler to trigger GCP (disable AWS EventBridge)
2. Monitor first production run closely
3. Keep AWS infrastructure for 1 month as backup

### Phase 4: Decommissioning
1. Verify GCP stability for 3 months
2. Archive AWS CloudFormation templates
3. Delete AWS infrastructure

**Timeline**: 4-5 months for safe migration

## Support & Contact

- **Issues**: Create GitHub issue with `gcp-implementation` label
- **Slack**: #kb7-automation
- **Email**: kb7-team@cardiofit.ai
- **Documentation**: See GCP_IMPLEMENTATION_GUIDE.md for detailed specifications

## License

Proprietary - CardioFit Platform
