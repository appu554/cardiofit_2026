# 📋 KB-7 GCP Deployment Checklist

**Track your deployment progress with this simple checklist**

---

## ✅ Prerequisites (Before Deployment)

### 1. Tools Installation
- [ ] Google Cloud SDK (gcloud) installed
  ```bash
  brew install --cask google-cloud-sdk
  gcloud version
  ```
- [ ] Terraform installed
  ```bash
  brew install hashicorp/tap/terraform
  terraform --version
  ```
- [ ] jq installed (optional, for JSON parsing)
  ```bash
  brew install jq
  ```

### 2. GCP Authentication
- [ ] Logged into GCP
  ```bash
  gcloud auth login
  ```
- [ ] Application default credentials set
  ```bash
  gcloud auth application-default login
  ```

### 3. GCP Project Setup
- [ ] GCP project created or selected
  ```bash
  gcloud config set project YOUR_PROJECT_ID
  ```
- [ ] Billing enabled on project
  ```bash
  gcloud billing projects describe $(gcloud config get-value project)
  ```

### 4. API Credentials Obtained
- [ ] **NHS TRUD API Key** - https://isd.digital.nhs.uk/trud/
  - Registration status: _______
  - API key received: _______

- [ ] **UMLS API Key** - https://uts.nlm.nih.gov/
  - UTS account created: _______
  - API key received: _______

- [ ] **LOINC Credentials** - https://loinc.org/downloads/
  - Account created: _______
  - Username: _______
  - Password: _______

- [ ] **GitHub PAT** - https://github.com/settings/tokens
  - Scopes: `repo`, `workflow`
  - Token generated: _______

---

## 🚀 Deployment Steps

### 5. Terraform Configuration
- [ ] Copied terraform.tfvars.example
  ```bash
  cd gcp/terraform
  cp terraform.tfvars.example terraform.tfvars
  ```
- [ ] Updated terraform.tfvars with:
  - [ ] `project_id`
  - [ ] `region`
  - [ ] `nhs_trud_api_key`
  - [ ] `umls_api_key`
  - [ ] `loinc_username`
  - [ ] `loinc_password`
  - [ ] `github_pat`
  - [ ] `slack_webhook_url` (optional)
  - [ ] `alert_email`

### 6. Pre-Flight Check
- [ ] Ran preflight check script
  ```bash
  cd gcp/scripts
  ./preflight-check.sh
  ```
- [ ] All checks passed: _______
- [ ] Warnings addressed: _______

### 7. Terraform Deployment
- [ ] Terraform initialized
  ```bash
  cd gcp/terraform
  terraform init
  ```
- [ ] Terraform plan reviewed
  ```bash
  terraform plan -out=tfplan
  ```
  - Resources to create: _______
  - Estimated monthly cost: _______

- [ ] Terraform applied
  ```bash
  terraform apply tfplan
  ```
  - Deployment completed: _______
  - Time taken: _______

### 8. Verification
- [ ] Cloud Storage buckets created
  ```bash
  gsutil ls | grep cardiofit-kb
  ```
- [ ] Cloud Functions deployed
  ```bash
  gcloud functions list --gen2 | grep kb7
  ```
- [ ] Secret Manager secrets created
  ```bash
  gcloud secrets list | grep kb7
  ```
- [ ] Cloud Workflows deployed
  ```bash
  gcloud workflows list | grep kb7
  ```
- [ ] Cloud Scheduler job created
  ```bash
  gcloud scheduler jobs list | grep kb7
  ```

### 9. Function Testing
- [ ] Ran test script
  ```bash
  cd gcp/scripts
  ./test-functions.sh
  ```
- [ ] Test results:
  - [ ] SNOMED downloader: _______
  - [ ] RxNorm downloader: _______
  - [ ] LOINC downloader: _______
  - [ ] GitHub dispatcher: _______

### 10. End-to-End Workflow Test
- [ ] Triggered manual workflow execution
  ```bash
  gcloud workflows execute kb7-factory-workflow-production \
    --location=us-central1 \
    --data='{"trigger":"manual-test"}'
  ```
- [ ] Workflow execution status: _______
- [ ] Execution time: _______
- [ ] Files downloaded to Cloud Storage: _______

---

## 📊 Post-Deployment

### 11. Monitoring Setup
- [ ] Accessed Cloud Console Monitoring
  - URL: https://console.cloud.google.com/monitoring
- [ ] Reviewed alert policies
  ```bash
  gcloud alpha monitoring policies list
  ```
- [ ] Configured Slack notifications (if enabled)
- [ ] Set up billing alerts
  ```bash
  gcloud billing budgets create \
    --billing-account=BILLING_ACCOUNT_ID \
    --display-name="KB-7 Monthly Budget" \
    --budget-amount=50
  ```

### 12. GitHub Actions Integration
- [ ] Created GitHub service account
  ```bash
  gcloud iam service-accounts create kb7-github-actions
  ```
- [ ] Generated service account key
  ```bash
  gcloud iam service-accounts keys create ~/kb7-github-sa-key.json \
    --iam-account=kb7-github-actions@PROJECT_ID.iam.gserviceaccount.com
  ```
- [ ] Added `GCP_SA_KEY` to GitHub repository secrets
- [ ] Updated GitHub Actions workflow to use GCP credentials

### 13. Documentation
- [ ] Documented project ID and region
- [ ] Stored API keys in secure password manager
- [ ] Updated team runbook with GCP-specific procedures
- [ ] Scheduled team training session

---

## 🎯 Production Readiness

### 14. Cost Optimization
- [ ] Reviewed billing dashboard
- [ ] Verified lifecycle policies active on buckets
- [ ] Confirmed function concurrency limits
- [ ] Set up cost anomaly alerts

### 15. Security Review
- [ ] Verified least-privilege IAM assignments
- [ ] Confirmed secrets stored in Secret Manager
- [ ] Enabled audit logging
- [ ] Reviewed service account permissions

### 16. Operational Procedures
- [ ] Documented rollback procedure
- [ ] Created incident response plan
- [ ] Set up on-call rotation (if applicable)
- [ ] Scheduled monthly review meetings

---

## 📅 Ongoing Operations

### Monthly Tasks
- [ ] Review Cloud Scheduler execution logs
- [ ] Verify terminology downloads successful
- [ ] Check for API key expiration warnings
- [ ] Review billing costs vs. budget

### Quarterly Tasks
- [ ] Rotate API keys (NHS TRUD: 90 days)
- [ ] Review and update alert thresholds
- [ ] Audit IAM permissions
- [ ] Update documentation

---

## 🐛 Troubleshooting References

If issues arise, check these resources:

1. **Pre-flight check failed**: Run `./scripts/preflight-check.sh` for diagnostic
2. **Terraform errors**: See `terraform/README.md` for common issues
3. **Function failures**: Check `gcloud functions logs read FUNCTION_NAME`
4. **Workflow failures**: Check `gcloud workflows executions describe EXECUTION_ID`
5. **Cost overruns**: Review `gcloud billing accounts list` and lifecycle policies

---

## 📞 Support Contacts

- **GCP Support**: https://console.cloud.google.com/support
- **Terraform Docs**: https://registry.terraform.io/providers/hashicorp/google/latest/docs
- **KB-7 Architecture**: See [GCP_IMPLEMENTATION_GUIDE.md](../GCP_IMPLEMENTATION_GUIDE.md)
- **Quick Start**: See [QUICKSTART_DEPLOYMENT.md](QUICKSTART_DEPLOYMENT.md)

---

## ✅ Deployment Status

**Overall Progress**: _____ / 16 sections complete

**Deployment Date**: _________________
**Deployed By**: _________________
**Production Ready**: [ ] Yes [ ] No
**Next Review Date**: _________________

---

**Notes**:
_Use this space to document any deployment-specific notes, issues encountered, or customizations made_

---

_Last Updated: November 24, 2025_
