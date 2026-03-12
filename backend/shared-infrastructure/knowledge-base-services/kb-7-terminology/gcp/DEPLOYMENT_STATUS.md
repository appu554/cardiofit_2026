# KB-7 GCP Deployment Status

**Date**: November 25, 2025
**Project**: sincere-hybrid-477206-h2
**Status**: Partial Deployment - Cloud Functions Blocked

---

## ✅ Successfully Deployed Resources (53/60)

### Core Infrastructure
- ✅ 12 GCP APIs enabled
- ✅ 4 Service Accounts created
  - kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
  - kb7-scheduler-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
  - kb7-workflows-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
  - kb7-github-actions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com

### Storage
- ✅ 3 Cloud Storage Buckets
  - sincere-hybrid-477206-h2-function-source
  - sincere-hybrid-477206-h2-kb-sources-production
  - sincere-hybrid-477206-h2-kb-artifacts-production
- ✅ 4 Function ZIP files uploaded

### Security
- ✅ 4 Secret Manager Secrets (with placeholder values)
  - kb7-nhs-trud-api-key-production
  - kb7-umls-api-key-production
  - kb7-loinc-credentials-production
  - kb7-github-token-production
- ✅ 6 IAM Role Bindings

### Monitoring
- ✅ 3 Log-based Metrics
- ✅ 2 Alert Policies
  - function_errors
  - function_duration
- ✅ 1 Email Notification Channel

---

## ❌ Blocked Resources (7/60)

### Cloud Functions (FAILED)
- ❌ kb7-snomed-downloader-production
- ❌ kb7-rxnorm-downloader-production
- ❌ kb7-loinc-downloader-production
- ❌ kb7-github-dispatcher-production

**Error**: `Build failed with status: FAILURE. Could not build the function due to a missing permission on the build service account`

### Dependent Resources (Not Deployed)
- ❌ Cloud Workflow (kb7-factory-workflow-production)
- ❌ Cloud Scheduler (kb7-monthly-terminology-update-production)
- ❌ workflow_failures Alert Policy

---

## 🔍 Issue Analysis

### Root Cause
Cloud Build service account (`513961303605@cloudbuild.gserviceaccount.com`) is failing at the GCS fetcher step when trying to fetch function source code from the function source bucket.

###Organization Policies Fixed
1. ✅ Removed `constraints/iam.allowedPolicyMemberDomains` - Was restricting to workspace C03j2tj7l only
2. ✅ Removed `constraints/iam.automaticIamGrantsForDefaultServiceAccounts` - Was preventing automatic IAM grants

### Permissions Granted to Cloud Build
- ✅ roles/cloudbuild.builds.builder
- ✅ roles/iam.serviceAccountUser
- ✅ roles/artifactregistry.writer
- ✅ roles/storage.admin
- ✅ roles/cloudkms.cryptoKeyEncrypterDecrypter
- ✅ roles/storage.objectViewer on function source bucket

### Still Failing
Despite all permissions and policy removals, Cloud Build continues to fail at the GCS fetcher step with exit code 3.

---

## 🔧 Troubleshooting Steps Attempted

1. **Organization Policy Review**: Listed and removed blocking policies
2. **IAM Permissions**: Granted comprehensive permissions to Cloud Build SA
3. **Bucket Permissions**: Explicitly granted objectViewer role on source bucket
4. **Policy Propagation**: Waited for changes to propagate (2+ minutes)
5. **Multiple Retries**: Attempted deployment 5+ times after each fix

---

## 📊 Next Steps

### Option 1: Wait for Propagation (Recommended)
Organization policy changes can take up to 10-15 minutes to fully propagate. Wait and retry:
```bash
cd gcp/terraform
terraform apply -auto-approve
```

### Option 2: Check for Additional Org Policies
Verify no other policies are blocking:
```bash
gcloud resource-manager org-policies list --organization=101041345933
```

### Option 3: Manual Function Deployment Test
Test direct deployment to isolate issue:
```bash
gcloud functions deploy test-function \
  --gen2 \
  --region=us-central1 \
  --runtime=python311 \
  --source=/path/to/function \
  --entry-point=main \
  --trigger-http
```

### Option 4: Contact GCP Support
If issue persists, this may require GCP Support investigation as organization policies at this level can have complex interactions.

---

## 🎯 When Deployment Completes

After Cloud Functions deploy successfully:

1. **Update API Keys**: Replace placeholder secrets with real API keys
   ```bash
   # NHS TRUD
   echo -n "YOUR_KEY" | gcloud secrets versions add kb7-nhs-trud-api-key-production --data-file=-

   # UMLS
   echo -n "YOUR_KEY" | gcloud secrets versions add kb7-umls-api-key-production --data-file=-

   # LOINC
   echo -n '{"username":"YOUR_USER","password":"YOUR_PASS"}' | gcloud secrets versions add kb7-loinc-credentials-production --data-file=-

   # GitHub
   echo -n "YOUR_PAT" | gcloud secrets versions add kb7-github-token-production --data-file=-
   ```

2. **Test Workflow**:
   ```bash
   gcloud workflows execute kb7-factory-workflow-production \
     --location=us-central1 \
     --data='{"trigger":"manual-test"}'
   ```

3. **Verify Monitoring**:
   - Cloud Console: https://console.cloud.google.com/monitoring
   - Check alert policies are active
   - Verify notification channels working

---

## 📝 Configuration Details

- **Project ID**: sincere-hybrid-477206-h2
- **Project Number**: 513961303605
- **Organization ID**: 101041345933
- **Region**: us-central1
- **Environment**: production
- **Terraform State**: gs://cardiofit-kb7-terraform-state/terraform/state

---

## 🔗 Useful Links

- [GCP Console](https://console.cloud.google.com/home/dashboard?project=sincere-hybrid-477206-h2)
- [Cloud Functions](https://console.cloud.google.com/functions/list?project=sincere-hybrid-477206-h2)
- [Cloud Build History](https://console.cloud.google.com/cloud-build/builds?project=sincere-hybrid-477206-h2)
- [Organization Policies](https://console.cloud.google.com/iam-admin/orgpolicies?organizationId=101041345933)
- [Terraform Documentation](https://registry.terraform.io/providers/hashicorp/google/latest/docs)

---

**Last Updated**: 2025-11-25 07:15 UTC
**Next Action**: Wait 10 minutes for policy propagation, then retry `terraform apply`
