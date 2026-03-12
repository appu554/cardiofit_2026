# KB-7 Knowledge Factory - GCP Quick Start Guide

Get the KB-7 Knowledge Factory running on GCP in under 1 hour.

## Prerequisites Checklist

- [ ] Google Cloud account with billing enabled
- [ ] gcloud CLI installed (`gcloud --version`)
- [ ] Terraform installed (`terraform --version` >= 1.5.0)
- [ ] API keys obtained (see below)

### API Keys Required

1. **NHS TRUD API Key** (for SNOMED CT)
   - Register: https://isd.digital.nhs.uk/trud/users/guest/filters/0/home
   - Navigate to: SNOMED CT UK Edition
   - Copy API key from account settings

2. **UMLS API Key** (for RxNorm)
   - Register: https://uts.nlm.nih.gov/uts/
   - Sign UMLS License Agreement
   - Generate API key in profile settings

3. **LOINC Credentials** (for LOINC)
   - Register: https://loinc.org/downloads/
   - Agree to LOINC license
   - Note username and password

4. **GitHub Personal Access Token**
   - Create: https://github.com/settings/tokens
   - Scopes: `repo` (full repository access)
   - Copy token immediately (shown once)

## Step-by-Step Deployment

### Step 1: Create GCP Project (5 minutes)

```bash
# Set your project ID (must be globally unique)
export PROJECT_ID="cardiofit-kb7-production"

# Create project
gcloud projects create $PROJECT_ID --name="KB-7 Knowledge Factory"

# Set as active project
gcloud config set project $PROJECT_ID

# Link billing account (replace with your billing account ID)
gcloud billing projects link $PROJECT_ID --billing-account=YOUR_BILLING_ACCOUNT_ID

# Verify
gcloud projects describe $PROJECT_ID
```

**Don't have a billing account?**
1. Visit: https://console.cloud.google.com/billing
2. Create billing account
3. Link credit card
4. Note the billing account ID

### Step 2: Configure Authentication (2 minutes)

```bash
# Authenticate with GCP
gcloud auth login

# Set application default credentials (for Terraform)
gcloud auth application-default login

# Verify authentication
gcloud auth list
```

### Step 3: Configure Terraform (5 minutes)

```bash
cd gcp/terraform

# Copy example configuration
cp terraform.tfvars.example terraform.tfvars

# Edit with your values
nano terraform.tfvars  # or vim, code, etc.
```

**Minimal required configuration**:
```hcl
project_id           = "cardiofit-kb7-production"  # Your project ID
region              = "us-central1"                # Or your preferred region
environment         = "production"

# API Keys (from Step 1)
nhs_trud_api_key    = "your-nhs-api-key-here"
umls_api_key        = "your-umls-api-key-here"
loinc_username      = "your-loinc-username"
loinc_password      = "your-loinc-password"
github_token        = "ghp_your-github-token"
github_repository   = "your-org/your-repo"

# Notifications (optional)
notification_email  = "your-email@domain.com"
slack_webhook_url   = ""  # Leave empty to disable Slack
```

**Save and close** the file.

### Step 4: Deploy Infrastructure (10 minutes)

```bash
cd ../scripts

# Make deployment script executable (if needed)
chmod +x deploy-infrastructure.sh

# Run deployment
./deploy-infrastructure.sh
```

**What happens**:
1. Validates prerequisites
2. Packages Cloud Functions
3. Initializes Terraform
4. Shows deployment plan
5. Asks for confirmation
6. Deploys all infrastructure

**You'll see**:
```
========================================
KB-7 Knowledge Factory - GCP Deployment
========================================

Step 1: Validating prerequisites...
✓ All prerequisites satisfied

Step 2: Verifying GCP authentication...
✓ Authenticated with GCP
  Current project: cardiofit-kb7-production

...

Deployment Complete!
```

### Step 5: Test Functions (10 minutes)

```bash
# Test individual functions
./test-functions.sh
```

**Expected output**:
```
Testing snomed_downloader...
  ✓ Function executed successfully (HTTP 200)

Testing rxnorm_downloader...
  ✓ Function executed successfully (HTTP 200)

Testing loinc_downloader...
  ✓ Function executed successfully (HTTP 200)

✓ All tests passed!
```

### Step 6: Test Complete Workflow (60 minutes)

```bash
# Execute workflow manually
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"trigger":"manual-test"}'
```

**Sample output**:
```
Waiting for execution [abc123-def456-ghi789] to complete...
...done.
state: SUCCEEDED
```

**Monitor progress**:
```bash
# View workflow executions
gcloud workflows executions list kb7-factory-workflow-production \
  --location=us-central1 \
  --limit=5

# View function logs
gcloud logging read \
  'resource.type="cloud_function" AND resource.labels.function_name=~"kb7-.*"' \
  --limit=20 \
  --format=json
```

### Step 7: Verify Results (5 minutes)

```bash
# Check source files
gsutil ls gs://cardiofit-kb7-production-kb-sources-production/

# Expected output:
# gs://cardiofit-kb7-production-kb-sources-production/snomed-ct/
# gs://cardiofit-kb7-production-kb-sources-production/rxnorm/
# gs://cardiofit-kb7-production-kb-sources-production/loinc/

# Check specific downloads
gsutil ls -lh gs://cardiofit-kb7-production-kb-sources-production/snomed-ct/*/

# Expected: Large zip file (8-10GB)
```

## Congratulations!

Your KB-7 Knowledge Factory is now running on GCP!

### What's Next?

#### Enable Scheduled Execution
The Cloud Scheduler is already configured but you may want to verify:
```bash
gcloud scheduler jobs describe kb7-monthly-terminology-update-production \
  --location=us-central1
```

**Schedule**: 1st of each month at 2 AM UTC

#### View Monitoring Dashboard
1. Visit: https://console.cloud.google.com/monitoring
2. Navigate to: Dashboards → KB-7 Knowledge Factory
3. View: Execution metrics, error rates, storage usage

#### Set Up Alerts
Already configured! Check:
```bash
gcloud alpha monitoring policies list --filter="displayName~'KB-7'"
```

**Alerts configured**:
- Function duration > 50 minutes
- Function execution errors
- Workflow failures

#### Review Costs
1. Visit: https://console.cloud.google.com/billing
2. Navigate to: Cost table
3. Filter by: KB-7 resources

**Expected monthly cost**: ~$21.50

## Troubleshooting

### Deployment Failed at Terraform Apply

**Check**:
1. Project billing is enabled
2. All required APIs are enabled
3. terraform.tfvars is correctly formatted

**Fix**:
```bash
# Enable APIs manually
gcloud services enable \
  cloudfunctions.googleapis.com \
  workflows.googleapis.com \
  storage.googleapis.com \
  secretmanager.googleapis.com

# Retry deployment
./deploy-infrastructure.sh
```

### Function Test Failed

**Check function logs**:
```bash
gcloud logging read \
  'resource.labels.function_name="kb7-snomed-downloader-production" AND severity>=ERROR' \
  --limit=10
```

**Common issues**:
- Invalid API key → Update secret in Secret Manager
- Timeout → Increase function timeout in functions.tf
- Permission denied → Check IAM bindings in iam.tf

### Workflow Execution Failed

**View execution details**:
```bash
# Get execution ID from error message
EXECUTION_ID="your-execution-id"

gcloud workflows executions describe $EXECUTION_ID \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1
```

**Common issues**:
- Function error → Check function logs (see above)
- Timeout → Increase workflow timeout in workflows.tf
- GitHub dispatch failed → Verify GitHub token has correct permissions

### No Files in Storage Bucket

**Verify function execution**:
```bash
# Check function invocation count
gcloud functions describe kb7-snomed-downloader-production \
  --gen2 \
  --region=us-central1 \
  --format="value(serviceConfig.availableMemory)"

# Check recent invocations
gcloud logging read \
  'resource.labels.function_name="kb7-snomed-downloader-production"' \
  --limit=5 \
  --format=json | jq '.[] | {timestamp, message: .jsonPayload.message}'
```

**Common issues**:
- Function not invoked → Check workflow execution
- API authentication failed → Verify API keys in Secret Manager
- Storage permission denied → Check IAM bindings on bucket

## Quick Reference

### Essential Commands

**Deploy infrastructure**:
```bash
cd gcp/scripts && ./deploy-infrastructure.sh
```

**Test functions**:
```bash
cd gcp/scripts && ./test-functions.sh
```

**Execute workflow**:
```bash
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"trigger":"manual-test"}'
```

**View logs**:
```bash
gcloud logging read \
  'resource.type="cloud_function" AND resource.labels.function_name=~"kb7-.*"' \
  --limit=50
```

**Check costs**:
```bash
gcloud billing accounts list
gcloud billing projects describe $PROJECT_ID
```

**Teardown (CAREFUL!)**:
```bash
cd gcp/scripts && ./teardown-infrastructure.sh
```

### Important URLs

- **GCP Console**: https://console.cloud.google.com
- **Cloud Functions**: https://console.cloud.google.com/functions?project=$PROJECT_ID
- **Cloud Workflows**: https://console.cloud.google.com/workflows?project=$PROJECT_ID
- **Cloud Storage**: https://console.cloud.google.com/storage/browser?project=$PROJECT_ID
- **Cloud Monitoring**: https://console.cloud.google.com/monitoring?project=$PROJECT_ID
- **Cloud Scheduler**: https://console.cloud.google.com/cloudscheduler?project=$PROJECT_ID

### Configuration Files

| File | Purpose |
|------|---------|
| `terraform/terraform.tfvars` | Your configuration (secrets) |
| `terraform/*.tf` | Infrastructure definitions |
| `functions/*/main.py` | Function source code |
| `workflows/kb-factory-workflow.yaml` | Orchestration logic |

## Need Help?

### Documentation
- **Full README**: `gcp/README.md`
- **Implementation Guide**: `GCP_IMPLEMENTATION_GUIDE.md`
- **Deployment Summary**: `gcp/DEPLOYMENT_SUMMARY.md`

### Support Channels
- **Email**: kb7-team@cardiofit.ai
- **Slack**: #kb7-automation
- **GitHub Issues**: Tag with `gcp-implementation`

### GCP Documentation
- [Cloud Functions](https://cloud.google.com/functions/docs)
- [Cloud Workflows](https://cloud.google.com/workflows/docs)
- [Terraform GCP Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)

---

**Happy Deploying!** 🚀
