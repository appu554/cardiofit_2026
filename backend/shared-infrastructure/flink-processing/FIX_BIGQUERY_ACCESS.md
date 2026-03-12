# Fix BigQuery Access Issue

## Current Status

✅ **Credentials created**: `mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com`
❌ **BigQuery access**: "Access Denied" error

**Error**: `User does not have permission to query table physionet-data:mimiciv_hosp.patients`

---

## Issue #1: Missing BigQuery Roles

Your service account needs **TWO roles** to query BigQuery:

### Solution: Add Roles in GCP Console

**Direct link**: https://console.cloud.google.com/iam-admin/iam?project=sincere-hybrid-477206-h2

**Steps**:
1. Click on your service account email: `mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com`
2. Click **"EDIT PRINCIPAL"** (pencil icon)
3. Click **"+ ADD ANOTHER ROLE"**
4. Add these two roles:
   - **BigQuery Data Viewer** (lets you read tables)
   - **BigQuery Job User** (lets you run queries)
5. Click **"SAVE"**

### Alternative: Command Line

```bash
gcloud projects add-iam-policy-binding sincere-hybrid-477206-h2 \
    --member="serviceAccount:mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
    --role="roles/bigquery.dataViewer"

gcloud projects add-iam-policy-binding sincere-hybrid-477206-h2 \
    --member="serviceAccount:mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
    --role="roles/bigquery.jobUser"
```

---

## Issue #2: PhysioNet MIMIC-IV Access

You received an email that says:
```
"You have requested access to MIMIC-IV v3.1 in GCP BigQuery"
```

This means your **request was received**, but you need to:

### Check Access Status

1. Go to: https://physionet.org/settings/cloud/
2. Look for **"MIMIC-IV v3.1"** in the list
3. Check status:
   - ✅ **"Approved"**: You have access!
   - ⏳ **"Pending"**: Wait for approval (1-2 business days)
   - ❌ **"Not requested"**: Need to request access

### Request Access (if needed)

1. Go to: https://physionet.org/content/mimiciv/3.1/
2. Click **"Request Access"** button
3. Complete requirements:
   - ✅ CITI training certificate (if not done: https://physionet.org/about/citi-course/)
   - ✅ Sign Data Use Agreement
   - ✅ Link GCP Project: `sincere-hybrid-477206-h2`
4. Submit request
5. Wait for approval email (usually 1-2 business days)

---

## Testing Access

After fixing both issues, test again:

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

query = "SELECT COUNT(*) as count FROM `physionet-data.mimiciv_hosp.patients`"

try:
    result = client.query(query).to_dataframe()
    print(f"✅ SUCCESS! MIMIC-IV patients: {result['count'].iloc[0]:,}")
except Exception as e:
    print(f"❌ Still blocked: {e}")
EOF
```

**Expected output when fixed**:
```
✅ SUCCESS! MIMIC-IV patients: 315,460
```

---

## Quick Checklist

Before running training pipeline, verify:

- [ ] Service account has **BigQuery Data Viewer** role
- [ ] Service account has **BigQuery Job User** role
- [ ] PhysioNet MIMIC-IV access is **Approved** (not just requested)
- [ ] GCP project `sincere-hybrid-477206-h2` is linked to PhysioNet account
- [ ] Test query returns patient count (315,460)

---

## Common Issues

### "Permission denied" even after adding roles

**Wait 5-10 minutes** for IAM changes to propagate

### "Table does not exist" error

**Check**: PhysioNet access status at https://physionet.org/settings/cloud/

### "Project not linked" error

**Fix**: Link GCP project to PhysioNet account:
1. Go to https://physionet.org/settings/cloud/
2. Add project: `sincere-hybrid-477206-h2`
3. Wait for approval

---

## Once Fixed

Run the training pipeline:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
./train-mimic-models.sh
```

This will extract 33,000+ real ICU patients and train production models!
