# KB-7 GCP Deployment - Quick Start Guide

**Estimated Time**: 45-60 minutes
**Cost**: ~$21.50/month
**Prerequisites**: GCP account with billing enabled

---

## ✅ Pre-Flight Checklist

Before starting deployment, gather these credentials:

- [ ] **GCP Project ID** (or create new one)
- [ ] **Billing Account** (for GCP charges)
- [ ] **NHS TRUD API Key** - https://isd.digital.nhs.uk/trud/
- [ ] **UMLS API Key** - https://uts.nlm.nih.gov/uts/signup-login
- [ ] **LOINC Credentials** - https://loinc.org/downloads/
- [ ] **GitHub Personal Access Token** (repo scope) - https://github.com/settings/tokens
- [ ] **Slack Webhook URL** (optional) - https://api.slack.com/messaging/webhooks

---

## 🚀 Deployment Steps

### Step 1: Setup GCP Project (5 minutes)

```bash
# Login to GCP
gcloud auth login
gcloud auth application-default login

# Create new project (or use existing)
export PROJECT_ID="cardiofit-kb7-production"
gcloud projects create $PROJECT_ID --name="KB-7 Knowledge Factory"

# Set as active project
gcloud config set project $PROJECT_ID

# Link billing account (replace with your billing account ID)
# Find your billing account: gcloud billing accounts list
export BILLING_ACCOUNT="XXXXXX-XXXXXX-XXXXXX"
gcloud billing projects link $PROJECT_ID --billing-account=$BILLING_ACCOUNT

# Verify billing is enabled
gcloud billing projects describe $PROJECT_ID
```

**Expected Output**: Billing account linked, billingEnabled: true

---

### Step 2: Enable Required APIs (3 minutes)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/gcp

# Enable all required GCP services
gcloud services enable \
  cloudfunctions.googleapis.com \
  cloudbuild.googleapis.com \
  cloudscheduler.googleapis.com \
  workflows.googleapis.com \
  storage.googleapis.com \
  secretmanager.googleapis.com \
  monitoring.googleapis.com \
  logging.googleapis.com \
  compute.googleapis.com \
  artifactregistry.googleapis.com \
  run.googleapis.com \
  --project=$PROJECT_ID

# Wait for API enablement (can take 1-2 minutes)
echo "Waiting for APIs to be fully enabled..."
sleep 60
```

**Expected Output**: Services enabled successfully

---

### Step 3: Configure Terraform Variables (5 minutes)

```bash
cd terraform

# Copy example configuration
cp terraform.tfvars.example terraform.tfvars

# Edit with your values
nano terraform.tfvars  # or use your preferred editor
```

**terraform.tfvars** - Update these values:

```hcl
# GCP Configuration
project_id  = "cardiofit-kb7-production"  # Your GCP project ID
region      = "us-central1"               # Or your preferred region
environment = "production"                # production, staging, or development

# API Credentials (from prerequisites)
nhs_trud_api_key    = "your-nhs-trud-api-key-here"
umls_api_key        = "your-umls-api-key-here"
loinc_username      = "your-loinc-username-here"
loinc_password      = "your-loinc-password-here"
github_pat          = "ghp_your-github-personal-access-token"

# Notification Configuration (optional)
slack_webhook_url   = "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
alert_email         = "kb7-alerts@cardiofit.ai"

# Resource Configuration (can keep defaults)
snomed_function_memory = "10Gi"   # 10GB for SNOMED downloads
rxnorm_function_memory = "3Gi"    # 3GB for RxNorm downloads
loinc_function_memory  = "2Gi"    # 2GB for LOINC downloads

# Schedule (monthly on 1st at 2 AM UTC)
schedule_cron = "0 2 1 * *"
```

**Save the file** (Ctrl+X, Y, Enter in nano)

---

### Step 4: Initialize Terraform (2 minutes)

```bash
# Still in gcp/terraform directory

# Initialize Terraform
terraform init

# Validate configuration
terraform validate

# Preview changes (no deployment yet)
terraform plan
```

**Expected Output**:
- "Terraform has been successfully initialized!"
- Plan shows ~25-30 resources to create
- No errors in validation

---

### Step 5: Deploy Infrastructure (10 minutes)

```bash
# Deploy all resources
terraform apply

# Review the plan, then type: yes
```

**What's Being Created**:
- 2 Cloud Storage buckets (sources + artifacts)
- 4 Cloud Functions (SNOMED, RxNorm, LOINC, GitHub dispatcher)
- 1 Cloud Workflows workflow
- 4 Secret Manager secrets
- 1 Cloud Scheduler job
- 4 Service accounts with IAM bindings
- 3 Cloud Monitoring alert policies
- 3 Log-based metrics

**Deployment Progress**:
```
Creating Cloud Storage buckets...       [1 min]
Creating Secret Manager secrets...      [1 min]
Creating IAM service accounts...        [1 min]
Creating Cloud Functions...             [5-7 min] ← Longest step
Creating Cloud Workflows...             [1 min]
Creating Cloud Scheduler...             [1 min]
Setting up monitoring...                [1 min]
```

**Expected Output**: "Apply complete! Resources: 27 added, 0 changed, 0 destroyed."

---

### Step 6: Verify Deployment (5 minutes)

```bash
# Go to scripts directory
cd ../scripts

# Make scripts executable
chmod +x *.sh

# Run verification tests
./test-functions.sh
```

**Expected Test Results**:
```
Testing SNOMED Downloader...        ✅ PASS
Testing RxNorm Downloader...        ✅ PASS
Testing LOINC Downloader...         ✅ PASS
Testing GitHub Dispatcher...        ✅ PASS

All 4 functions operational!
```

---

### Step 7: Test End-to-End Workflow (15 minutes)

```bash
# Trigger the complete workflow manually
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"trigger":"manual-test","timestamp":"'$(date -u +%Y%m%d)'"}'

# Get execution ID from output
export EXECUTION_ID="<execution-id-from-output>"

# Monitor execution progress
gcloud workflows executions describe $EXECUTION_ID \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1

# Watch logs in real-time
gcloud workflows executions list \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1 \
  --limit=1 \
  --format="table(name,state,startTime,endTime)"
```

**Expected Workflow Steps**:
1. **Parallel Downloads** (5-10 min):
   - SNOMED: Download 1.2GB from NHS TRUD
   - RxNorm: Download 450MB from NIH UMLS
   - LOINC: Download 180MB from LOINC.org
2. **GitHub Dispatch** (30 sec):
   - Trigger GitHub Actions workflow
   - Pass S3 keys and checksums

**Success Indicators**:
- Workflow state: "SUCCEEDED"
- All 4 functions complete successfully
- Files uploaded to Cloud Storage bucket

---

### Step 8: Verify Cloud Storage (2 minutes)

```bash
# List downloaded files
gsutil ls -lh gs://cardiofit-kb-sources-production/

# Should see:
# gs://cardiofit-kb-sources-production/snomed-ct/YYYYMMDD/...
# gs://cardiofit-kb-sources-production/rxnorm/YYYYMMDD/...
# gs://cardiofit-kb-sources-production/loinc/YYYYMMDD/...
```

---

### Step 9: Setup Monitoring (5 minutes)

```bash
# Open Cloud Console Monitoring
echo "Opening Cloud Monitoring..."
open "https://console.cloud.google.com/monitoring/dashboards?project=$PROJECT_ID"

# Import custom dashboard (from terraform outputs)
terraform output monitoring_dashboard_url

# Check alert policies
gcloud alpha monitoring policies list --project=$PROJECT_ID
```

**Monitoring Features Enabled**:
- ✅ Function duration alerts (>50 minutes)
- ✅ Workflow failure alerts
- ✅ Error rate monitoring
- ✅ Log-based metrics (download success/failure)
- ✅ Slack notifications (if configured)

---

### Step 10: Enable Monthly Automation (1 minute)

```bash
# Verify Cloud Scheduler is enabled
gcloud scheduler jobs describe kb7-monthly-terminology-update-production \
  --location=us-central1

# Schedule is already active! Next run: 1st of next month at 2 AM UTC
```

---

## ✅ Deployment Complete!

Your GCP Knowledge Factory is now operational. Here's what happens automatically:

**Monthly (1st of month, 2 AM UTC)**:
1. Cloud Scheduler triggers Cloud Workflows
2. Workflows execute 4 Cloud Functions in parallel
3. Functions download latest terminologies (SNOMED, RxNorm, LOINC)
4. Files uploaded to Cloud Storage with SHA256 verification
5. GitHub Actions triggered for transformation pipeline
6. Slack notification on success/failure

---

## 🔧 Post-Deployment Configuration

### Connect to GitHub Actions

Your GitHub Actions workflow needs to read from Cloud Storage:

```bash
# Create service account for GitHub Actions
gcloud iam service-accounts create kb7-github-actions \
  --display-name="KB-7 GitHub Actions Service Account" \
  --project=$PROJECT_ID

# Grant Storage Object Viewer access
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:kb7-github-actions@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectViewer"

# Create key for GitHub Secrets
gcloud iam service-accounts keys create ~/kb7-github-sa-key.json \
  --iam-account=kb7-github-actions@$PROJECT_ID.iam.gserviceaccount.com

# Add to GitHub repository secrets as GCP_SA_KEY
cat ~/kb7-github-sa-key.json
# Copy output and add as repository secret in GitHub
```

---

## 📊 Cost Monitoring

```bash
# View current month charges
gcloud billing accounts list
gcloud billing projects describe $PROJECT_ID

# Setup budget alerts
gcloud billing budgets create \
  --billing-account=$BILLING_ACCOUNT \
  --display-name="KB-7 Monthly Budget" \
  --budget-amount=50 \
  --threshold-rule=percent=50 \
  --threshold-rule=percent=90 \
  --threshold-rule=percent=100
```

---

## 🐛 Troubleshooting

### Issue: Terraform Apply Fails

**Error**: "Error creating function: generic::invalid_argument: region is not available"

**Solution**:
```bash
# Change region in terraform.tfvars to an available region
region = "us-east1"  # or "europe-west1", "asia-northeast1"

# Re-run terraform apply
terraform apply
```

---

### Issue: Function Test Fails

**Error**: "Permission denied" or "403 Forbidden"

**Solution**:
```bash
# Grant yourself permissions to invoke functions
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="user:$(gcloud config get-value account)" \
  --role="roles/cloudfunctions.invoker"

# Re-run tests
./test-functions.sh
```

---

### Issue: Workflow Execution Fails

**Error**: "Failed to download from NHS TRUD API"

**Solution**:
```bash
# Verify NHS TRUD API key is correct
gcloud secrets versions access latest --secret=kb7-nhs-trud-api-key

# Update if incorrect
echo -n "your-correct-api-key" | gcloud secrets versions add kb7-nhs-trud-api-key --data-file=-

# Test SNOMED function again
gcloud functions call kb7-snomed-downloader-production \
  --gen2 \
  --region=us-central1 \
  --data='{"test":true}'
```

---

### Issue: High Costs

**Solution**:
```bash
# Check Cloud Storage usage
gsutil du -sh gs://cardiofit-kb-sources-production
gsutil du -sh gs://cardiofit-kb-artifacts-production

# Delete old versions if needed
gsutil -m rm -r gs://cardiofit-kb-sources-production/snomed-ct/old-version/

# Verify lifecycle policies are active
gsutil lifecycle get gs://cardiofit-kb-sources-production
```

---

## 🎯 Next Steps

1. **Monitor First Run**: Check logs on 1st of next month
2. **Integrate with GraphDB**: Update GraphDB deployment scripts to read from GCS
3. **Setup Alerts**: Configure Slack/email notifications
4. **Document Credentials**: Store API keys in secure password manager
5. **Train Team**: Share access and operational procedures

---

## 📞 Support

- **Documentation**: See [gcp/README.md](README.md) for detailed guide
- **Terraform Docs**: See [terraform/README.md](terraform/README.md)
- **GCP Console**: https://console.cloud.google.com/
- **Cost Explorer**: https://console.cloud.google.com/billing/

---

## 🔒 Security Checklist

- [ ] API keys stored in Secret Manager (not environment variables)
- [ ] Service accounts use least-privilege IAM
- [ ] Functions not publicly accessible (internal only)
- [ ] Audit logging enabled
- [ ] Budget alerts configured
- [ ] GitHub SA key secured

---

**Deployment Status**: ✅ COMPLETE

Your Knowledge Factory is now running on Google Cloud Platform!
