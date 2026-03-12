# KB-7 GCP Deployment - Installation Guide

## Step 1: Install Google Cloud SDK

### macOS Installation (Recommended)

```bash
# Using Homebrew (recommended)
brew install --cask google-cloud-sdk

# Add gcloud to PATH (if not automatically added)
echo 'source "$(brew --prefix)/share/google-cloud-sdk/path.bash.inc"' >> ~/.zshrc
echo 'source "$(brew --prefix)/share/google-cloud-sdk/completion.bash.inc"' >> ~/.zshrc
source ~/.zshrc

# Verify installation
gcloud version
```

**Expected Output**:
```
Google Cloud SDK 457.0.0
bq 2.0.101
core 2024.01.12
gcloud 2024.01.12
gsutil 5.27
```

### Alternative: Manual Installation

If Homebrew is not available:

```bash
# Download installer
curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-darwin-arm64.tar.gz

# Extract (adjust filename if x86_64)
tar -xzf google-cloud-cli-darwin-arm64.tar.gz

# Run installer
./google-cloud-sdk/install.sh

# Initialize
./google-cloud-sdk/bin/gcloud init
```

---

## Step 2: Install Terraform

```bash
# Using Homebrew
brew tap hashicorp/tap
brew install hashicorp/tap/terraform

# Verify installation
terraform --version
```

**Expected Output**: `Terraform v1.7.0` or higher

---

## Step 3: Install Optional Tools

```bash
# jq (JSON parsing for logs)
brew install jq

# curl (should already be installed)
which curl
```

---

## Step 4: Verify Installation

Run the preflight check:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/gcp/scripts
./preflight-check.sh
```

**Success Criteria**: All tool checks should pass (✅)

---

## Step 5: Authenticate with GCP

```bash
# Login to GCP (opens browser)
gcloud auth login

# Set application default credentials
gcloud auth application-default login

# Verify authentication
gcloud auth list
```

**Expected Output**:
```
        Credentialed Accounts
ACTIVE  ACCOUNT
*       your-email@domain.com
```

---

## Step 6: Create/Select GCP Project

### Option A: Create New Project

```bash
# Set project ID (must be globally unique)
export PROJECT_ID="cardiofit-kb7-production"

# Create project
gcloud projects create $PROJECT_ID \
  --name="KB-7 Knowledge Factory" \
  --set-as-default

# Verify project created
gcloud config get-value project
```

### Option B: Use Existing Project

```bash
# List your projects
gcloud projects list

# Set active project
gcloud config set project YOUR_EXISTING_PROJECT_ID
```

---

## Step 7: Enable Billing

```bash
# List billing accounts
gcloud billing accounts list

# Link billing account to project
export BILLING_ACCOUNT="XXXXXX-XXXXXX-XXXXXX"  # From list above
gcloud billing projects link $PROJECT_ID \
  --billing-account=$BILLING_ACCOUNT

# Verify billing enabled
gcloud billing projects describe $PROJECT_ID
```

**Expected Output**: `billingEnabled: true`

---

## Step 8: Enable Required GCP APIs

This takes 2-3 minutes:

```bash
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
  eventarc.googleapis.com \
  --project=$PROJECT_ID

# Wait for APIs to be fully enabled
echo "Waiting 60 seconds for APIs to be fully enabled..."
sleep 60

# Verify APIs enabled
gcloud services list --enabled | grep -E "cloudfunctions|storage|workflows"
```

---

## Step 9: Gather API Credentials

You'll need these external API keys (may take 24-48 hours to receive):

### NHS TRUD API Key
- **URL**: https://isd.digital.nhs.uk/trud/
- **Registration**: Create account → Subscribe to SNOMED CT UK Edition
- **Time**: 24-48 hours for approval
- **Format**: `TRUD-API-KEY-xxxxxxxxxxxxxxxx`

### UMLS API Key
- **URL**: https://uts.nlm.nih.gov/uts/signup-login
- **Registration**: Create UTS account → Request API access
- **Time**: Instant for basic, 24 hours for full access
- **Format**: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`

### LOINC Credentials
- **URL**: https://loinc.org/downloads/
- **Registration**: Create account → Accept license
- **Time**: Instant
- **Format**: Username + Password

### GitHub Personal Access Token
- **URL**: https://github.com/settings/tokens
- **Scopes**: `repo`, `workflow`
- **Format**: `ghp_xxxxxxxxxxxxxxxxxxxx`

---

## Step 10: Configure Terraform Variables

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/gcp/terraform

# Copy example configuration
cp terraform.tfvars.example terraform.tfvars

# Edit with your values
nano terraform.tfvars
```

**Required Values**:
```hcl
project_id  = "cardiofit-kb7-production"    # Your GCP project ID
region      = "us-central1"                 # Or your preferred region

# API Credentials (from Step 9)
nhs_trud_api_key = "your-nhs-trud-api-key-here"
umls_api_key     = "your-umls-api-key-here"
loinc_username   = "your-loinc-username"
loinc_password   = "your-loinc-password"
github_pat       = "ghp_your-github-token"

# Notification (optional)
alert_email = "kb7-alerts@yourdomain.com"
```

---

## Step 11: Run Final Preflight Check

```bash
cd ../scripts
./preflight-check.sh
```

**Success Criteria**: All checks pass (0 failures)

---

## Step 12: Deploy Infrastructure

```bash
cd ../terraform

# Initialize Terraform
terraform init

# Preview deployment
terraform plan -out=tfplan

# Review the plan, then apply
terraform apply tfplan
```

**Deployment Time**: 10-12 minutes

**Expected Resources**: 27 resources created

---

## Troubleshooting

### Issue: "gcloud: command not found"

**Solution**: Path not set correctly
```bash
# Add to shell profile
echo 'source "$(brew --prefix)/share/google-cloud-sdk/path.bash.inc"' >> ~/.zshrc
source ~/.zshrc
```

### Issue: "Project ID already exists"

**Solution**: Choose different project ID (must be globally unique)
```bash
export PROJECT_ID="cardiofit-kb7-prod-$(date +%s)"
gcloud projects create $PROJECT_ID
```

### Issue: "Billing not enabled"

**Solution**: Must link billing account
```bash
gcloud billing accounts list
gcloud billing projects link $PROJECT_ID --billing-account=BILLING_ACCOUNT_ID
```

### Issue: "API not enabled"

**Solution**: Re-run API enablement
```bash
gcloud services enable cloudfunctions.googleapis.com --project=$PROJECT_ID
```

---

## Next Steps After Deployment

1. **Verify Functions**: `cd ../scripts && ./test-functions.sh`
2. **Test Workflow**: Trigger manual execution
3. **Monitor**: Access Cloud Console monitoring dashboard
4. **Schedule**: Workflow will run automatically on 1st of each month

---

## Quick Reference

```bash
# Check authentication
gcloud auth list

# Check active project
gcloud config get-value project

# Check billing
gcloud billing projects describe $(gcloud config get-value project)

# List deployed functions
gcloud functions list --gen2

# View function logs
gcloud functions logs read FUNCTION_NAME --gen2
```

---

**Total Setup Time**: 45-60 minutes (excluding API credential wait times)

**Ready to Deploy**: Complete Steps 1-11, then run terraform apply
