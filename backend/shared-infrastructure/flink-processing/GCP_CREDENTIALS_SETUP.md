# GCP Credentials Setup Guide for MIMIC-IV Access

**Your GCP Project**: `sincere-hybrid-477206-h2` ✅

This guide walks you through creating service account credentials to access MIMIC-IV in BigQuery.

---

## 🎯 Overview

You need a **Service Account JSON key** that allows your Python scripts to query MIMIC-IV data in BigQuery.

**What we'll create**:
- Service Account: `mimic-iv-reader@sincere-hybrid-477206-h2.iam.gserviceaccount.com`
- Role: BigQuery Data Viewer (read-only access)
- Credentials file: `~/.gcp/mimic-iv-credentials.json`

---

## 📋 Step-by-Step Instructions

### Step 1: Open GCP Console

1. Go to: https://console.cloud.google.com
2. **Select your project**: `sincere-hybrid-477206-h2`
3. Look for the project name in the top navigation bar

---

### Step 2: Enable BigQuery API

Before creating credentials, ensure BigQuery API is enabled:

1. In GCP Console, go to **APIs & Services** → **Library**
2. Search for "BigQuery API"
3. Click **Enable** (if not already enabled)
4. Wait for confirmation (usually instant)

✅ **Check**: You should see "API enabled" status

---

### Step 3: Create Service Account

1. In GCP Console, navigate to:
   - **Menu (☰)** → **IAM & Admin** → **Service Accounts**
   - Or direct link: https://console.cloud.google.com/iam-admin/serviceaccounts?project=sincere-hybrid-477206-h2

2. Click **+ CREATE SERVICE ACCOUNT** (top of page)

3. **Service account details**:
   - **Service account name**: `mimic-iv-reader`
   - **Service account ID**: `mimic-iv-reader` (auto-populated)
   - **Description**: `Read-only access to MIMIC-IV BigQuery data`
   - Click **CREATE AND CONTINUE**

4. **Grant this service account access to project**:
   - **Role**: Search for and select **BigQuery Data Viewer**
   - Click **+ ADD ANOTHER ROLE** and add **BigQuery Job User**
   - Click **CONTINUE**

   > **Why these roles?**
   > - **BigQuery Data Viewer**: Read MIMIC-IV tables
   > - **BigQuery Job User**: Run queries (required for query execution)

5. **Grant users access** (optional):
   - Skip this step (click **DONE**)

✅ **Check**: You should see `mimic-iv-reader` in the service accounts list

---

### Step 4: Create and Download JSON Key

1. In the **Service Accounts** list, find `mimic-iv-reader`
2. Click on the **three dots (⋮)** on the right
3. Select **Manage keys**
4. Click **ADD KEY** → **Create new key**
5. **Key type**: Select **JSON** (default)
6. Click **CREATE**

**Important**: The JSON file will download automatically to your Downloads folder!
- File name: `sincere-hybrid-477206-h2-XXXXX.json` (with random suffix)

---

### Step 5: Move Credentials to Secure Location

**On Mac/Linux**:
```bash
# Create directory for GCP credentials
mkdir -p ~/.gcp

# Move downloaded file (replace XXXXX with actual filename)
mv ~/Downloads/sincere-hybrid-477206-h2-*.json ~/.gcp/mimic-iv-credentials.json

# Set proper permissions (read-only for you)
chmod 600 ~/.gcp/mimic-iv-credentials.json

# Verify file exists
ls -la ~/.gcp/mimic-iv-credentials.json
```

**On Windows**:
```powershell
# Create directory
New-Item -ItemType Directory -Force -Path $HOME\.gcp

# Move file (adjust path if needed)
Move-Item $HOME\Downloads\sincere-hybrid-477206-h2-*.json $HOME\.gcp\mimic-iv-credentials.json

# Verify
Get-Item $HOME\.gcp\mimic-iv-credentials.json
```

✅ **Check**: File should be at `~/.gcp/mimic-iv-credentials.json`

---

### Step 6: Set Environment Variables

**On Mac/Linux (Add to `~/.bashrc` or `~/.zshrc`)**:
```bash
# Open your shell config file
nano ~/.zshrc  # or ~/.bashrc for bash

# Add these lines at the end:
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.gcp/mimic-iv-credentials.json"
export GCP_PROJECT_ID="sincere-hybrid-477206-h2"

# Save and exit (Ctrl+X, then Y, then Enter)

# Reload shell config
source ~/.zshrc  # or source ~/.bashrc
```

**On Windows (PowerShell)**:
```powershell
# Set environment variables (user-level, permanent)
[System.Environment]::SetEnvironmentVariable(
    "GOOGLE_APPLICATION_CREDENTIALS",
    "$HOME\.gcp\mimic-iv-credentials.json",
    "User"
)

[System.Environment]::SetEnvironmentVariable(
    "GCP_PROJECT_ID",
    "sincere-hybrid-477206-h2",
    "User"
)

# Restart terminal to apply
```

**For this session only** (temporary):
```bash
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.gcp/mimic-iv-credentials.json"
export GCP_PROJECT_ID="sincere-hybrid-477206-h2"
```

✅ **Check**: Run `echo $GOOGLE_APPLICATION_CREDENTIALS` (should show path)

---

### Step 7: Verify MIMIC-IV Access

**Important**: You must have **PhysioNet credentialed access** to MIMIC-IV in BigQuery!

**Check your PhysioNet access**:
1. Log in to: https://physionet.org/
2. Go to: https://physionet.org/content/mimiciv/3.1/
3. Check access status:
   - ✅ **Green checkmark**: You have access
   - ⚠️ **Pending**: Wait for approval
   - ❌ **No access**: Need to request credentialed access

**Request access if needed**:
1. Complete CITI training: https://physionet.org/about/citi-course/
2. Upload certificate to PhysioNet
3. Sign Data Use Agreement for MIMIC-IV
4. Wait for approval (usually 1-2 business days)

---

### Step 8: Test Connection

**Run validation script**:
```bash
cd backend/shared-infrastructure/flink-processing

python3 scripts/mimic_iv_config.py
```

**Expected output** ✅:
```
MIMIC-IV Configuration
======================================================================
GCP Project: sincere-hybrid-477206-h2
Credentials: /Users/you/.gcp/mimic-iv-credentials.json
MIMIC Dataset: physionet-data.mimiciv_hosp
Tables: 9 configured

✅ Configuration validated successfully
```

**If you get errors**:
- ❌ **Credentials file not found**: Check file path in Step 5
- ❌ **Permission denied**: Verify service account roles in Step 3
- ❌ **Table not found**: Need PhysioNet credentialed access (Step 7)

---

## 🧪 Quick BigQuery Test

**Test direct BigQuery access**:
```bash
# Install Google Cloud SDK (if not installed)
# Mac: brew install google-cloud-sdk
# Linux: https://cloud.google.com/sdk/docs/install

# Authenticate with service account
gcloud auth activate-service-account \
    --key-file=$GOOGLE_APPLICATION_CREDENTIALS \
    --project=sincere-hybrid-477206-h2

# Test query
bq query --use_legacy_sql=false \
    'SELECT COUNT(*) as patient_count FROM `physionet-data.mimiciv_hosp.patients`'
```

**Expected output**:
```
+---------------+
| patient_count |
+---------------+
|        315460 |
+---------------+
```

---

## 📁 Your Credentials File Structure

The JSON file should look like this (with your actual values):
```json
{
  "type": "service_account",
  "project_id": "sincere-hybrid-477206-h2",
  "private_key_id": "abc123...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...",
  "client_email": "mimic-iv-reader@sincere-hybrid-477206-h2.iam.gserviceaccount.com",
  "client_id": "123456789...",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/..."
}
```

**⚠️ SECURITY WARNING**:
- ❌ NEVER commit this file to Git
- ❌ NEVER share this file publicly
- ❌ NEVER upload to websites/forums
- ✅ Keep in `~/.gcp/` directory only
- ✅ Set permissions to 600 (read-only for you)

---

## 🎯 Next Steps After Credentials Setup

Once credentials are working:

```bash
cd backend/shared-infrastructure/flink-processing

# Option 1: Automated pipeline (recommended for first time)
./train-mimic-models.sh

# Option 2: Manual step-by-step
python3 scripts/extract_mimic_cohorts.py
python3 scripts/extract_mimic_features.py
python3 scripts/train_mimic_models.py
```

---

## 🆘 Troubleshooting

### Error: "Permission denied" when querying BigQuery

**Solution**: Add both roles to service account:
```
1. BigQuery Data Viewer  (read tables)
2. BigQuery Job User     (run queries)
```

### Error: "Dataset not found: physionet-data.mimiciv_hosp"

**Cause**: No PhysioNet credentialed access to MIMIC-IV

**Solution**:
1. Go to https://physionet.org/content/mimiciv/3.1/
2. Click "Request Access"
3. Complete CITI training if needed
4. Wait for approval (1-2 business days)

### Error: "API has not been enabled"

**Solution**: Enable BigQuery API
```bash
gcloud services enable bigquery.googleapis.com --project=sincere-hybrid-477206-h2
```

### Error: "Credentials file not found"

**Solution**: Check environment variable
```bash
echo $GOOGLE_APPLICATION_CREDENTIALS
# Should show: /Users/you/.gcp/mimic-iv-credentials.json

# If empty, set it:
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.gcp/mimic-iv-credentials.json"
```

---

## ✅ Checklist

Before proceeding with model training, verify:

- [ ] GCP project selected: `sincere-hybrid-477206-h2`
- [ ] BigQuery API enabled
- [ ] Service account created: `mimic-iv-reader`
- [ ] Roles assigned: BigQuery Data Viewer + BigQuery Job User
- [ ] JSON key downloaded
- [ ] Credentials moved to `~/.gcp/mimic-iv-credentials.json`
- [ ] File permissions set to 600
- [ ] Environment variable set: `GOOGLE_APPLICATION_CREDENTIALS`
- [ ] PhysioNet credentialed access approved for MIMIC-IV
- [ ] Configuration validated: `python3 scripts/mimic_iv_config.py` ✅

---

**Once all checkboxes are ✅, you're ready to start training real clinical models!**

Run:
```bash
./train-mimic-models.sh
```

This will extract 33,000+ real ICU patients from MIMIC-IV and train production-ready clinical prediction models.
