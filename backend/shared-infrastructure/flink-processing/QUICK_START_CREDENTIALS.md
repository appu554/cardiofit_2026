# Quick Start: Get Your MIMIC-IV Credentials (5 Minutes)

**Your Project**: `sincere-hybrid-477206-h2` ✅

Follow these exact steps to get your credentials file.

---

## ⚡ Fast Track (Copy & Paste Ready)

### 1️⃣ Open GCP Console

**Direct Link**: https://console.cloud.google.com/iam-admin/serviceaccounts?project=sincere-hybrid-477206-h2

Make sure you see **"sincere-hybrid-477206-h2"** in the top navigation bar.

---

### 2️⃣ Create Service Account (1 minute)

Click **"+ CREATE SERVICE ACCOUNT"** (blue button at top)

**Fill in form**:
```
Service account name: mimic-iv-reader
Service account ID:   mimic-iv-reader (auto-fills)
Description:          Read-only access to MIMIC-IV BigQuery
```

Click **"CREATE AND CONTINUE"** → Step 2 opens

**Add roles** (IMPORTANT - add BOTH):
```
Role 1: BigQuery Data Viewer
Role 2: BigQuery Job User  (click "+ ADD ANOTHER ROLE")
```

Click **"CONTINUE"** → Click **"DONE"**

---

### 3️⃣ Download JSON Key (1 minute)

In the service accounts list:

1. Find `mimic-iv-reader@sincere-hybrid-477206-h2.iam.gserviceaccount.com`
2. Click the **three dots (⋮)** on the right
3. Click **"Manage keys"**
4. Click **"ADD KEY"** → **"Create new key"**
5. Select **"JSON"** (default)
6. Click **"CREATE"**

**📥 A file downloads automatically**: `sincere-hybrid-477206-h2-XXXXX.json`

---

### 4️⃣ Move File to Secure Location (1 minute)

**Mac/Linux**:
```bash
# Create directory
mkdir -p ~/.gcp

# Move file (replace XXXXX with actual random suffix from downloaded file)
mv ~/Downloads/sincere-hybrid-477206-h2-*.json ~/.gcp/mimic-iv-credentials.json

# Set secure permissions
chmod 600 ~/.gcp/mimic-iv-credentials.json

# Verify
ls -la ~/.gcp/mimic-iv-credentials.json
```

**Windows PowerShell**:
```powershell
New-Item -ItemType Directory -Force -Path $HOME\.gcp
Move-Item $HOME\Downloads\sincere-hybrid-477206-h2-*.json $HOME\.gcp\mimic-iv-credentials.json
Get-Item $HOME\.gcp\mimic-iv-credentials.json
```

---

### 5️⃣ Set Environment Variables (1 minute)

**Mac/Linux (for current session)**:
```bash
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.gcp/mimic-iv-credentials.json"
export GCP_PROJECT_ID="sincere-hybrid-477206-h2"

# Verify it worked
echo $GOOGLE_APPLICATION_CREDENTIALS
```

**To make permanent (Mac/Linux)**:
```bash
# Add to your shell config
echo 'export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.gcp/mimic-iv-credentials.json"' >> ~/.zshrc
echo 'export GCP_PROJECT_ID="sincere-hybrid-477206-h2"' >> ~/.zshrc

# Reload
source ~/.zshrc
```

**Windows PowerShell (permanent)**:
```powershell
[System.Environment]::SetEnvironmentVariable("GOOGLE_APPLICATION_CREDENTIALS", "$HOME\.gcp\mimic-iv-credentials.json", "User")
[System.Environment]::SetEnvironmentVariable("GCP_PROJECT_ID", "sincere-hybrid-477206-h2", "User")

# Restart terminal, then verify:
echo $env:GOOGLE_APPLICATION_CREDENTIALS
```

---

### 6️⃣ Test Connection (1 minute)

```bash
cd backend/shared-infrastructure/flink-processing

# Install dependencies (if not done yet)
pip3 install google-cloud-bigquery google-auth pandas

# Test configuration
python3 scripts/mimic_iv_config.py
```

**✅ Success looks like**:
```
MIMIC-IV Configuration
======================================================================
GCP Project: sincere-hybrid-477206-h2
Credentials: /Users/you/.gcp/mimic-iv-credentials.json
MIMIC Dataset: physionet-data.mimiciv_hosp
Tables: 9 configured

✅ Configuration validated successfully
```

**❌ If you see errors**, see Troubleshooting section below.

---

## 🚀 Ready? Start Training!

Once you see ✅ above, run:

```bash
./train-mimic-models.sh
```

This will:
1. Extract 33,000+ real ICU patients from MIMIC-IV BigQuery
2. Build 70-dimensional clinical feature vectors
3. Train 4 XGBoost models on real data
4. Export to ONNX format
5. Replace mock models with real clinical models

**Total time**: ~60-90 minutes (mostly BigQuery queries)

---

## 🆘 Troubleshooting

### ❌ Error: "Credentials file not found"

**Check**:
```bash
ls -la ~/.gcp/mimic-iv-credentials.json
```

**If missing**: Redo Step 4 (move file from Downloads)

---

### ❌ Error: "Permission denied" or "Access Denied"

**Cause**: Missing BigQuery roles

**Fix**: Add both roles to service account:
1. Go to: https://console.cloud.google.com/iam-admin/serviceaccounts?project=sincere-hybrid-477206-h2
2. Click on `mimic-iv-reader` email
3. Go to **"PERMISSIONS"** tab
4. Verify these roles exist:
   - ✅ BigQuery Data Viewer
   - ✅ BigQuery Job User

**If missing**: Click **"GRANT ACCESS"** → Add project → Add both roles

---

### ❌ Error: "Dataset not found: physionet-data.mimiciv_hosp"

**Cause**: You don't have PhysioNet credentialed access to MIMIC-IV yet

**Fix**:
1. Go to: https://physionet.org/content/mimiciv/3.1/
2. Sign in with your PhysioNet account
3. Click **"Request Access"**
4. Complete CITI training if needed: https://physionet.org/about/citi-course/
5. Sign Data Use Agreement
6. Wait for approval (usually 1-2 business days)

**Check access status**: Visit https://physionet.org/settings/cloud/ and look for MIMIC-IV v3.1 approval

---

### ❌ Error: "API has not been enabled"

**Fix**: Enable BigQuery API
```bash
# Install gcloud if needed: brew install google-cloud-sdk

gcloud services enable bigquery.googleapis.com --project=sincere-hybrid-477206-h2
```

Or via Console:
1. https://console.cloud.google.com/apis/library/bigquery.googleapis.com?project=sincere-hybrid-477206-h2
2. Click **"ENABLE"**

---

### ❌ Python packages missing

**Install all dependencies**:
```bash
cd backend/shared-infrastructure/flink-processing
pip3 install -r requirements_mimic.txt
```

---

## 📞 Need Help?

**Check these files**:
- Full guide: `GCP_CREDENTIALS_SETUP.md` (detailed explanations)
- Training guide: `MIMIC_IV_TRAINING_GUIDE.md` (complete pipeline)
- Config file: `scripts/mimic_iv_config.py` (verify settings)

**Common issues**:
1. **No PhysioNet access**: Most common! Request access at physionet.org
2. **Missing roles**: Need both BigQuery Data Viewer AND BigQuery Job User
3. **Wrong project**: Make sure you see `sincere-hybrid-477206-h2` in GCP Console
4. **File permissions**: Run `chmod 600 ~/.gcp/mimic-iv-credentials.json`

---

## ✅ Verification Checklist

Before running `./train-mimic-models.sh`:

- [ ] File exists: `~/.gcp/mimic-iv-credentials.json`
- [ ] Environment variable set: `echo $GOOGLE_APPLICATION_CREDENTIALS` shows path
- [ ] Service account created: `mimic-iv-reader@sincere-hybrid-477206-h2.iam.gserviceaccount.com`
- [ ] Roles assigned: BigQuery Data Viewer + BigQuery Job User
- [ ] PhysioNet access: Approved for MIMIC-IV v3.1
- [ ] Test passed: `python3 scripts/mimic_iv_config.py` shows ✅

---

**All green? You're ready to train real clinical models! 🚀**

```bash
./train-mimic-models.sh
```
