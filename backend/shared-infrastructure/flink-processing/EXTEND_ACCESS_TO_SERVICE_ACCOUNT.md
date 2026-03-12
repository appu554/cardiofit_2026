# Extend PhysioNet Access to Service Account

**Current Status**: You have personal PhysioNet access (`onkarshahi@vaidshala.com`) but your service account needs access too.

**Goal**: Enable your service account `mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com` to access MIMIC-IV BigQuery.

---

## Why This is Needed

```
Current Access:
✅ onkarshahi@vaidshala.com → Can access MIMIC-IV (personal account)
❌ mimic-iv-reade@...iam.gserviceaccount.com → Cannot access (service account)

After Linking Project:
✅ onkarshahi@vaidshala.com → Can access
✅ mimic-iv-reade@...iam.gserviceaccount.com → Can access ✨
✅ ANY service account in sincere-hybrid-477206-h2 → Can access ✨
```

---

## Step-by-Step Instructions

### Step 1: Link Your GCP Project (2 minutes)

1. **Log into PhysioNet** with your approved account:
   - Go to: https://physionet.org/login/
   - Use: `onkarshahi@vaidshala.com`

2. **Navigate to Cloud Settings**:
   - Direct link: https://physionet.org/settings/cloud/
   - Or: Click username (top right) → Settings → Cloud tab

3. **Add Your GCP Project**:
   - Look for section: **"Google Cloud Platform"**
   - Click: **"Add GCP Project"** or **"Link Project"**
   - Enter project ID: `sincere-hybrid-477206-h2`
   - Click: **"Submit"** or **"Add"**

4. **Verify Project Added**:
   - You should now see in the list:
     ```
     Google Cloud Platform Projects:
     ✅ sincere-hybrid-477206-h2
     ```

---

### Step 2: Request BigQuery Access for Project (1 minute)

1. **Go to MIMIC-IV Dataset Page**:
   - https://physionet.org/content/mimiciv/3.1/

2. **Look for BigQuery Access Section**:
   - Scroll down to find: **"Access the data using Google BigQuery"**
   - Or look for: **Cloud** icon/button

3. **Request Access**:
   - Click: **"Request access in Google BigQuery"** or similar button
   - Select your project: `sincere-hybrid-477206-h2`
   - Submit request

**Note**: Since you already have personal credentialed access, this should be **fast-tracked**!

---

### Step 3: Wait for Approval (Usually hours, not days)

**Timeline**:
- If you already had MIMIC-IV credentialed access: **Few hours** ⚡
- If first-time requesting: **1-2 business days**

**You'll receive an email**:
```
Subject: "Access approved for MIMIC-IV v3.1 on Google Cloud"
Or: "BigQuery access approved for project sincere-hybrid-477206-h2"

The email will confirm that your GCP project can now access:
- Dataset: physionet-data.mimiciv_hosp
- Dataset: physionet-data.mimiciv_icu
- Dataset: physionet-data.mimiciv_derived
```

---

## After Approval: Test Connection

Once you receive the approval email, run this test:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

python3 << 'EOF'
from google.cloud import bigquery
from google.oauth2 import service_account

credentials = service_account.Credentials.from_service_account_file(
    "/Users/apoorvabk/.gcp/mimic-iv-credentials.json",
    scopes=["https://www.googleapis.com/auth/bigquery"],
)

client = bigquery.Client(
    credentials=credentials,
    project="sincere-hybrid-477206-h2",
)

query = "SELECT COUNT(*) as patient_count FROM `physionet-data.mimiciv_hosp.patients`"
result = client.query(query).to_dataframe()
count = result['patient_count'].iloc[0]

print(f"✅ SUCCESS! MIMIC-IV Patients: {count:,}")
print("🚀 Ready to run: ./train-mimic-models.sh")
EOF
```

**Expected output** (after approval):
```
✅ SUCCESS! MIMIC-IV Patients: 315,460
🚀 Ready to run: ./train-mimic-models.sh
```

---

## Troubleshooting

### Still Getting "Access Denied" After Approval?

**Wait 10-15 minutes** after receiving approval email for permissions to propagate.

### Can't Find "Add GCP Project" Button?

- Make sure you're logged in as `onkarshahi@vaidshala.com`
- Try this direct link: https://physionet.org/settings/cloud/
- Look for: "Google Cloud Platform" section (not AWS or Azure)

### Email Never Arrived?

**Check approval status**:
1. Go to: https://physionet.org/content/mimiciv/3.1/
2. Look for your project under "Cloud Access"
3. Status should show: "Approved" or "Active"

If status is "Pending", wait another day. If stuck, contact PhysioNet support.

---

## Once Approved: Run Training Pipeline

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Full automated pipeline (90 minutes)
./train-mimic-models.sh

# Or manual step-by-step:
python3 scripts/extract_mimic_cohorts.py      # Step 1: Extract patients
python3 scripts/extract_mimic_features.py      # Step 2: Build features
python3 scripts/train_mimic_models.py          # Step 3: Train models
```

---

## Summary

**What You Have**:
- ✅ Personal PhysioNet access: `onkarshahi@vaidshala.com`
- ✅ GCP project: `sincere-hybrid-477206-h2`
- ✅ Service account: `mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com`
- ✅ BigQuery roles: Job User + Data Viewer

**What You Need**:
- ⏳ Link GCP project to PhysioNet account
- ⏳ Request BigQuery access for the project
- ⏳ Wait for approval email (hours if already credentialed)

**Then You Can**:
- 🚀 Train real clinical models on 300K+ ICU patients
- 🚀 Replace mock models with production-ready MIMIC-IV models
- 🚀 Get real risk stratification (not ~94% for everyone!)

---

**Next Step**: Go to https://physionet.org/settings/cloud/ and add project `sincere-hybrid-477206-h2`
