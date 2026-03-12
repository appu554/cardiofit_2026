# KB-7 Knowledge Factory - API Credentials Configuration

**Date**: 2025-11-26
**Status**: Ready for Configuration
**Prerequisite**: All integration fixes completed

---

## Overview

The KB-7 Knowledge Factory download jobs require valid API credentials to access the terminology sources from authoritative providers. This guide shows how to configure these credentials in GCP Secret Manager.

## Required API Credentials

### 1. SNOMED-CT (via UMLS Terminology Services)

**Provider**: National Library of Medicine (NLM) UMLS Terminology Services
**Signup**: https://uts.nlm.nih.gov/uts/signup-login
**API Documentation**: https://documentation.uts.nlm.nih.gov/rest/api-versioning.html

**Credentials Needed**:
- UMLS API Key (from UTS profile)
- UMLS Username
- UMLS Password

### 2. RxNorm (via UMLS Terminology Services)

**Provider**: National Library of Medicine (NLM) UMLS Terminology Services
**Same Account**: Uses the same UMLS credentials as SNOMED-CT
**Download URL**: https://download.nlm.nih.gov/umls/kss/rxnorm/

**Credentials Needed**:
- UMLS API Key (same as SNOMED-CT)
- UMLS Username (same as SNOMED-CT)
- UMLS Password (same as SNOMED-CT)

### 3. LOINC (via Regenstrief Institute)

**Provider**: Regenstrief Institute
**Signup**: https://loinc.org/downloads/
**License Required**: Accept LOINC Terms of Use

**Credentials Needed**:
- LOINC Username
- LOINC Password

---

## GCP Secret Manager Configuration

### Step 1: Create SNOMED-CT Secret

```bash
# Navigate to GCP directory
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/gcp

# Create secret with your UMLS credentials
# IMPORTANT: Replace YOUR_UMLS_* with your actual credentials

gcloud secrets create kb7-ncts-api-key-production \
  --project=sincere-hybrid-477206-h2 \
  --replication-policy="automatic" \
  --data-file=- <<EOF
{
  "api_key": "YOUR_UMLS_API_KEY",
  "username": "YOUR_UMLS_USERNAME",
  "password": "YOUR_UMLS_PASSWORD"
}
EOF

# Grant access to Cloud Run service account
gcloud secrets add-iam-policy-binding kb7-ncts-api-key-production \
  --project=sincere-hybrid-477206-h2 \
  --member="serviceAccount:kb7-cloud-run-sa@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Verify creation
gcloud secrets describe kb7-ncts-api-key-production --project=sincere-hybrid-477206-h2
```

### Step 2: Create RxNorm Secret

```bash
# RxNorm uses the same UMLS credentials, so reference the same secret structure
# Create separate secret for isolation and auditing

gcloud secrets create kb7-umls-api-key-production \
  --project=sincere-hybrid-477206-h2 \
  --replication-policy="automatic" \
  --data-file=- <<EOF
{
  "api_key": "YOUR_UMLS_API_KEY",
  "username": "YOUR_UMLS_USERNAME",
  "password": "YOUR_UMLS_PASSWORD"
}
EOF

# Grant access to Cloud Run service account
gcloud secrets add-iam-policy-binding kb7-umls-api-key-production \
  --project=sincere-hybrid-477206-h2 \
  --member="serviceAccount:kb7-cloud-run-sa@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Verify creation
gcloud secrets describe kb7-umls-api-key-production --project=sincere-hybrid-477206-h2
```

### Step 3: Create LOINC Secret

```bash
# Create LOINC credentials secret
# IMPORTANT: Replace YOUR_LOINC_* with your actual credentials

gcloud secrets create kb7-loinc-credentials-production \
  --project=sincere-hybrid-477206-h2 \
  --replication-policy="automatic" \
  --data-file=- <<EOF
{
  "username": "YOUR_LOINC_USERNAME",
  "password": "YOUR_LOINC_PASSWORD"
}
EOF

# Grant access to Cloud Run service account
gcloud secrets add-iam-policy-binding kb7-loinc-credentials-production \
  --project=sincere-hybrid-477206-h2 \
  --member="serviceAccount:kb7-cloud-run-sa@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Verify creation
gcloud secrets describe kb7-loinc-credentials-production --project=sincere-hybrid-477206-h2
```

---

## Verification

### Check All Secrets Are Created

```bash
# List all KB-7 secrets
gcloud secrets list \
  --project=sincere-hybrid-477206-h2 \
  --filter="name~kb7"

# Expected output should show:
# - kb7-ncts-api-key-production
# - kb7-umls-api-key-production
# - kb7-loinc-credentials-production
# - kb7-github-token-production (already exists)
```

### Verify IAM Permissions

```bash
# Check SNOMED secret access
gcloud secrets get-iam-policy kb7-ncts-api-key-production \
  --project=sincere-hybrid-477206-h2

# Check RxNorm secret access
gcloud secrets get-iam-policy kb7-umls-api-key-production \
  --project=sincere-hybrid-477206-h2

# Check LOINC secret access
gcloud secrets get-iam-policy kb7-loinc-credentials-production \
  --project=sincere-hybrid-477206-h2

# Each should show:
# - serviceAccount:kb7-cloud-run-sa@*.iam.gserviceaccount.com
# - role: roles/secretmanager.secretAccessor
```

### Test Secret Access (without revealing values)

```bash
# Test that Cloud Run can access secrets
# This command retrieves the latest version without showing the value

gcloud secrets versions describe latest \
  --secret=kb7-ncts-api-key-production \
  --project=sincere-hybrid-477206-h2

gcloud secrets versions describe latest \
  --secret=kb7-umls-api-key-production \
  --project=sincere-hybrid-477206-h2

gcloud secrets versions describe latest \
  --secret=kb7-loinc-credentials-production \
  --project=sincere-hybrid-477206-h2
```

---

## Update Cloud Run Jobs to Use Secrets

The Cloud Run Jobs are already configured to read these secrets. Verify the configuration:

```bash
# Check SNOMED job configuration
gcloud run jobs describe kb7-snomed-job-production \
  --region=us-central1 \
  --format="yaml(spec.template.spec.containers[0].env)" | grep SECRET

# Check RxNorm job configuration
gcloud run jobs describe kb7-rxnorm-job-production \
  --region=us-central1 \
  --format="yaml(spec.template.spec.containers[0].env)" | grep SECRET

# Check LOINC job configuration
gcloud run jobs describe kb7-loinc-job-production \
  --region=us-central1 \
  --format="yaml(spec.template.spec.containers[0].env)" | grep SECRET
```

Expected output should show environment variables like:
```yaml
- name: SECRET_NAME
  value: kb7-ncts-api-key-production  # or kb7-umls-api-key-production, kb7-loinc-credentials-production
```

---

## Test Individual Download Jobs

Once secrets are configured, test each download job individually:

### Test SNOMED-CT Download

```bash
echo "=== Testing SNOMED-CT Download ==="
gcloud run jobs execute kb7-snomed-job-production \
  --region=us-central1 \
  --wait

# Check logs
gcloud logging read "resource.type=cloud_run_job AND resource.labels.job_name=kb7-snomed-job-production" \
  --limit=50 \
  --format="table(timestamp,textPayload)" \
  --project=sincere-hybrid-477206-h2
```

**Success Indicators**:
- Exit code 0
- Log message: "Successfully uploaded SNOMED-CT to GCS"
- File exists in GCS: `gs://sincere-hybrid-477206-h2-kb-sources-production/snomed-ct/YYYYMMDD/`

### Test RxNorm Download

```bash
echo "=== Testing RxNorm Download ==="
gcloud run jobs execute kb7-rxnorm-job-production \
  --region=us-central1 \
  --wait

# Check logs
gcloud logging read "resource.type=cloud_run_job AND resource.labels.job_name=kb7-rxnorm-job-production" \
  --limit=50 \
  --format="table(timestamp,textPayload)" \
  --project=sincere-hybrid-477206-h2
```

**Success Indicators**:
- Exit code 0
- Log message: "Successfully uploaded RxNorm to GCS"
- File exists in GCS: `gs://sincere-hybrid-477206-h2-kb-sources-production/rxnorm/YYYYMMDD/`

### Test LOINC Download

```bash
echo "=== Testing LOINC Download ==="
gcloud run jobs execute kb7-loinc-job-production \
  --region=us-central1 \
  --wait

# Check logs
gcloud logging read "resource.type=cloud_run_job AND resource.labels.job_name=kb7-loinc-job-production" \
  --limit=50 \
  --format="table(timestamp,textPayload)" \
  --project=sincere-hybrid-477206-h2
```

**Success Indicators**:
- Exit code 0
- Log message: "Successfully uploaded LOINC to GCS"
- File exists in GCS: `gs://sincere-hybrid-477206-h2-kb-sources-production/loinc/YYYYMMDD/`

---

## Verify Files in GCS

After successful downloads, verify files are in the correct GCS buckets:

```bash
# Check SNOMED-CT files
gsutil ls gs://sincere-hybrid-477206-h2-kb-sources-production/snomed-ct/

# Check RxNorm files
gsutil ls gs://sincere-hybrid-477206-h2-kb-sources-production/rxnorm/

# Check LOINC files
gsutil ls gs://sincere-hybrid-477206-h2-kb-sources-production/loinc/

# Check file sizes (should be substantial - hundreds of MB)
gsutil du -sh gs://sincere-hybrid-477206-h2-kb-sources-production/snomed-ct/
gsutil du -sh gs://sincere-hybrid-477206-h2-kb-sources-production/rxnorm/
gsutil du -sh gs://sincere-hybrid-477206-h2-kb-sources-production/loinc/
```

**Expected File Sizes**:
- SNOMED-CT: ~500 MB - 1 GB (RF2 snapshot)
- RxNorm: ~200-400 MB (RRF files)
- LOINC: ~150-300 MB (CSV files)

---

## Troubleshooting

### Issue: Secret Creation Fails with "Permission Denied"

**Solution**: Ensure you have the correct IAM permissions:
```bash
gcloud projects add-iam-policy-binding sincere-hybrid-477206-h2 \
  --member="user:$(gcloud config get-value account)" \
  --role="roles/secretmanager.admin"
```

### Issue: Cloud Run Job Can't Access Secret

**Symptom**: Job logs show "Permission denied accessing secret"

**Solution**: Verify the service account has secretAccessor role:
```bash
gcloud secrets add-iam-policy-binding <SECRET_NAME> \
  --project=sincere-hybrid-477206-h2 \
  --member="serviceAccount:kb7-cloud-run-sa@sincere-hybrid-477206-h2.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### Issue: Download Job Fails with "Authentication Failed"

**Symptom**: Job logs show 401 or 403 errors

**Solution**: Verify credentials are correct:
1. Test credentials manually at the provider websites
2. Check for special characters in passwords (may need escaping in JSON)
3. Verify UMLS license is active and not expired
4. For LOINC, ensure Terms of Use are accepted

### Issue: Download Job Times Out

**Symptom**: Job execution exceeds 1-hour timeout

**Solution**: Large downloads may need increased timeout:
```bash
gcloud run jobs update <JOB_NAME> \
  --region=us-central1 \
  --timeout=3600s \
  --max-retries=2
```

---

## Next Steps After API Configuration

Once all three secrets are configured and tested:

1. **Execute Full Workflow**:
   ```bash
   gcloud workflows run kb7-factory-workflow-production \
     --location=us-central1 \
     --data='{"trigger":"production","github_repo":"onkarshahi-IND/knowledge-factory"}'
   ```

2. **Monitor Workflow Execution**:
   ```bash
   # Get execution ID from previous command, then:
   gcloud workflows executions describe <EXECUTION_ID> \
     --workflow=kb7-factory-workflow-production \
     --location=us-central1
   ```

3. **Monitor GitHub Actions**:
   - Navigate to: https://github.com/onkarshahi-IND/knowledge-factory/actions
   - Look for workflow run triggered by repository dispatch
   - Verify all 7 stages complete successfully

4. **Deploy RDF Kernel to GraphDB**:
   ```bash
   # After pipeline success, review manifest
   gsutil cat gs://sincere-hybrid-477206-h2-kb-artifacts-production/latest/kb7-manifest.json | jq

   # Deploy to GraphDB
   cd ../scripts
   ./deploy-kernel.sh YYYYMMDD
   ```

---

## Security Best Practices

- **Rotate Credentials**: Update secrets annually or when compromised
- **Audit Access**: Review secret access logs monthly
- **Least Privilege**: Only grant secretAccessor to necessary service accounts
- **Version Control**: Keep old secret versions for 30 days for rollback
- **Monitoring**: Set up Cloud Monitoring alerts for secret access failures

---

## Summary

| Secret Name | Purpose | Provider | Status |
|-------------|---------|----------|--------|
| `kb7-ncts-api-key-production` | SNOMED-CT downloads | UMLS NLM | ⏳ Pending |
| `kb7-umls-api-key-production` | RxNorm downloads | UMLS NLM | ⏳ Pending |
| `kb7-loinc-credentials-production` | LOINC downloads | Regenstrief | ⏳ Pending |

**Action Required**: Configure the three API credential secrets above using your own credentials from the respective providers.

---

**Created**: 2025-11-26
**Documentation**: Complete
**Ready for**: API Credential Configuration
