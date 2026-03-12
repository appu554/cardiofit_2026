# Link Your GCP Project to PhysioNet for MIMIC-IV Access

## Understanding the Setup

MIMIC-IV data is hosted by PhysioNet in **their BigQuery project**: `physionet-data`

Your GCP project (`sincere-hybrid-477206-h2`) needs **explicit permission** from PhysioNet to query their data.

---

## Step-by-Step: Link Your Project

### 1. Log into PhysioNet

Go to: https://physionet.org/login/

### 2. Navigate to Cloud Settings

Direct link: https://physionet.org/settings/cloud/

Or:
- Click your username (top right)
- Click **"Settings"**
- Click **"Cloud"** tab

### 3. Link Your GCP Project

You'll see a section: **"Google Cloud Platform"**

**Add your project**:
```
Project ID: sincere-hybrid-477206-h2
```

Click **"Add GCP Project"** or **"Link Project"**

### 4. Request MIMIC-IV Access

After linking the project, you need to request access to specific datasets.

**Go to MIMIC-IV dataset page**: https://physionet.org/content/mimiciv/3.1/

Look for:
- **"Files"** tab → Shows if you can access data
- **"Request Access"** button (if you don't have access yet)

**Click "Request Access"** and:
1. Sign the Data Use Agreement (DUA)
2. Confirm you've completed CITI training (required)
3. Select your GCP project: `sincere-hybrid-477206-h2`
4. Submit request

### 5. Wait for Approval

**Timeline**: Usually 1-2 business days

**You'll receive an email** when approved:
```
Subject: "Access approved for MIMIC-IV v3.1 on Google Cloud"
```

---

## Checking CITI Training Status

PhysioNet requires **CITI Data or Specimens Only Research** training.

### Do you have it?

Check: https://physionet.org/settings/training/

**If you need to complete it**:
1. Go to: https://physionet.org/about/citi-course/
2. Follow instructions to complete CITI training (takes ~3 hours)
3. Upload certificate to PhysioNet
4. Wait for verification (usually 1 business day)

---

## Verification: Is Your Project Linked?

### Check on PhysioNet

**Go to**: https://physionet.org/settings/cloud/

You should see:
```
Google Cloud Platform Projects:
✅ sincere-hybrid-477206-h2 (Active)
```

### Check MIMIC-IV Access

**Go to**: https://physionet.org/content/mimiciv/3.1/

Look for one of these:
- ✅ **Green checkmark** + "You have access to these files"
- ⏳ **Orange clock** + "Access request pending"
- ❌ **Red X** + "Request access" button

---

## Testing After Approval

Once PhysioNet approves your access (you'll get an email), test:

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

print("🔍 Testing access to physionet-data (MIMIC-IV)...")
print()

query = """
    SELECT COUNT(*) as patient_count
    FROM `physionet-data.mimiciv_hosp.patients`
"""

try:
    result = client.query(query).to_dataframe()
    count = result['patient_count'].iloc[0]

    print("✅ SUCCESS! PhysioNet access is working!")
    print(f"   📊 MIMIC-IV Patients: {count:,}")
    print()
    print("🚀 Ready to start training pipeline!")
    print("   Run: ./train-mimic-models.sh")

except Exception as e:
    error_msg = str(e)

    if "Access Denied" in error_msg or "permission" in error_msg:
        print("❌ Access still denied")
        print()
        print("Next steps:")
        print("  1. Check PhysioNet cloud settings:")
        print("     https://physionet.org/settings/cloud/")
        print()
        print("  2. Verify project is linked: sincere-hybrid-477206-h2")
        print()
        print("  3. Check MIMIC-IV access status:")
        print("     https://physionet.org/content/mimiciv/3.1/")
        print()
        print("  4. If pending, wait for approval email (1-2 business days)")

    elif "does not exist" in error_msg:
        print("❌ Dataset not found")
        print()
        print("Issue: GCP project not linked to PhysioNet")
        print("Fix: Add project at https://physionet.org/settings/cloud/")

    else:
        print(f"❌ Unexpected error: {error_msg}")
EOF
```

---

## Common Issues

### Issue: "Project ID not recognized"

**Cause**: GCP project not linked to PhysioNet account

**Fix**:
1. Go to https://physionet.org/settings/cloud/
2. Add project: `sincere-hybrid-477206-h2`
3. Verify it appears in the list

### Issue: "Access request pending"

**Status**: Normal! Wait for approval

**Timeline**:
- Weekdays: 1-2 business days
- Weekends: Wait until Monday

**Check email** for approval notification

### Issue: "CITI training required"

**Cause**: PhysioNet requires CITI training certificate

**Fix**:
1. Complete training: https://physionet.org/about/citi-course/
2. Upload certificate to PhysioNet
3. Wait for verification
4. Then request MIMIC-IV access

### Issue: "You must sign the data use agreement"

**Fix**:
1. Go to https://physionet.org/content/mimiciv/3.1/
2. Click "Request Access"
3. Read and sign the DUA
4. Submit request

---

## What the Email Looks Like When Approved

You'll receive an email like:

```
Subject: Access approved for MIMIC-IV v3.1 on Google Cloud

Dear [Your Name],

Your request for access to MIMIC-IV v3.1 on Google Cloud Platform
has been approved.

You can now query the dataset using:
- Project: physionet-data
- Dataset: mimiciv_hosp, mimiciv_icu

Your linked GCP project: sincere-hybrid-477206-h2

Best regards,
PhysioNet Team
```

---

## After Approval: Service Account Permissions

Even after PhysioNet approval, your **service account** needs BigQuery roles in **your project**:

### Add these roles to: `mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com`

**In GCP Console**: https://console.cloud.google.com/iam-admin/iam?project=sincere-hybrid-477206-h2

**Roles needed**:
- ✅ BigQuery Job User (to run queries)
- ✅ BigQuery Data Viewer (to read results)

**Or via command line**:
```bash
gcloud projects add-iam-policy-binding sincere-hybrid-477206-h2 \
    --member="serviceAccount:mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
    --role="roles/bigquery.jobUser"

gcloud projects add-iam-policy-binding sincere-hybrid-477206-h2 \
    --member="serviceAccount:mimic-iv-reade@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
    --role="roles/bigquery.dataViewer"
```

---

## Complete Checklist

Before training pipeline will work:

- [ ] PhysioNet account created
- [ ] CITI training completed and uploaded
- [ ] GCP project linked: `sincere-hybrid-477206-h2` at physionet.org/settings/cloud/
- [ ] MIMIC-IV access requested at physionet.org/content/mimiciv/3.1/
- [ ] Data Use Agreement signed
- [ ] Approval email received (wait 1-2 business days)
- [ ] Service account has BigQuery Job User role
- [ ] Service account has BigQuery Data Viewer role
- [ ] Test query returns 315,460 patients

---

## Quick Test Command

After completing all steps above, verify:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# This should return: ✅ SUCCESS! MIMIC-IV Patients: 315,460
python3 scripts/mimic_iv_config.py
```

**If you see 315,460 patients** → You're ready to train!

```bash
./train-mimic-models.sh
```

---

## Need Help?

**Check these resources**:
- PhysioNet FAQ: https://physionet.org/about/faq/
- MIMIC-IV docs: https://mimic.mit.edu/docs/iv/
- PhysioNet support: https://physionet.org/about/contact/

**Most common delay**: Waiting for PhysioNet approval (1-2 business days after submitting request)
