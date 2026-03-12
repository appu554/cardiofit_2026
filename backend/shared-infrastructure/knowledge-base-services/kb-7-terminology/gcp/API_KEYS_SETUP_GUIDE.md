# API Keys Setup Guide - KB-7 Terminology Services

Step-by-step guide to obtain and configure API credentials for all terminology services.

---

## 📋 Overview

You need 4 sets of credentials:
1. **NHS TRUD API Key** - For SNOMED CT downloads
2. **UMLS API Key** - For RxNorm downloads
3. **LOINC Credentials** - Username and password for LOINC downloads
4. **GitHub Personal Access Token** - For triggering GitHub workflows

---

## 1️⃣ NHS TRUD API Key (SNOMED CT)

### Registration Steps

1. **Visit NHS TRUD Portal**:
   - Go to: https://isd.digital.nhs.uk/trud/users/guest/filters/0/home
   - Click **"Register"** in top right

2. **Create Account**:
   - Fill in registration form:
     - Full Name
     - Email Address
     - Organization
     - Country
     - Purpose: Select "Research" or "Development"
   - Agree to terms and conditions
   - Click **"Submit"**

3. **Email Verification**:
   - Check your email for verification link
   - Click link to verify account
   - Log in to TRUD

4. **Request SNOMED CT Access**:
   - Search for "SNOMED CT UK"
   - Click on "SNOMED CT UK Edition"
   - Click **"Subscribe"** button
   - Accept the license agreement

5. **Get API Key**:
   - Once subscribed, go to **"My Account"** → **"API Keys"**
   - Click **"Generate New API Key"**
   - Copy the API key (starts with `nhstrud_`)
   - **IMPORTANT**: Save this key immediately - you won't be able to see it again

### Update in GCP

```bash
# Replace YOUR_KEY with the actual key
echo -n "nhstrud_YOUR_API_KEY_HERE" | gcloud secrets versions add kb7-nhs-trud-api-key-production --data-file=-
```

**Verification**:
```bash
# Verify the secret was updated
gcloud secrets versions list kb7-nhs-trud-api-key-production
```

---

## 2️⃣ UMLS API Key (RxNorm)

### Registration Steps

1. **Visit UMLS Account Creation**:
   - Go to: https://uts.nlm.nih.gov/uts/signup-login
   - Click **"Request a License"**

2. **Create Account**:
   - Fill in registration form:
     - Username (will be your UMLS username)
     - Password
     - Email Address
     - Full Name
     - Organization/Institution
     - Position/Title
     - Country
   - Click **"Continue"**

3. **Accept License**:
   - Read the UMLS License Agreement
   - Check "I have read and accept the license"
   - Click **"Submit"**

4. **Email Verification**:
   - Check email for verification link
   - Click link to activate account
   - Log in to UMLS

5. **Get API Key**:
   - Log in to: https://uts.nlm.nih.gov/uts/
   - Click your username in top right → **"My Profile"**
   - Scroll to **"API Key Management"** section
   - Click **"Generate new API Key"**
   - Copy the API key (long alphanumeric string)
   - **IMPORTANT**: Save this key - you can regenerate but old keys will be invalidated

### Update in GCP

```bash
# Replace YOUR_KEY with the actual UMLS API key
echo -n "YOUR_UMLS_API_KEY_HERE" | gcloud secrets versions add kb7-umls-api-key-production --data-file=-
```

**Verification**:
```bash
# Verify the secret was updated
gcloud secrets versions list kb7-umls-api-key-production
```

---

## 3️⃣ LOINC Credentials

### Registration Steps

1. **Visit LOINC Website**:
   - Go to: https://loinc.org/
   - Click **"Downloads"** in top menu
   - Or directly: https://loinc.org/downloads/

2. **Create Account**:
   - Click **"Create Account"** or **"Sign Up"**
   - Fill in registration form:
     - Email Address
     - Password
     - First Name
     - Last Name
     - Organization
     - Country
     - Intended Use (select "Research" or "Development")
   - Accept LOINC License Agreement
   - Click **"Create Account"**

3. **Email Verification**:
   - Check email for verification link
   - Click link to verify account

4. **Accept License**:
   - Log in to: https://loinc.org/
   - Navigate to Downloads section
   - Accept the LOINC License if prompted
   - You may need to fill out a brief survey about intended use

5. **Your Credentials**:
   - **Username**: Your registered email address
   - **Password**: The password you created
   - These will be used to download LOINC files

### Update in GCP

```bash
# Replace with your actual LOINC credentials
# IMPORTANT: This is a JSON string with username and password
echo -n '{"username":"your.email@example.com","password":"your_password_here"}' | gcloud secrets versions add kb7-loinc-credentials-production --data-file=-
```

**Example**:
```bash
# If your email is john@example.com and password is SecurePass123
echo -n '{"username":"john@example.com","password":"SecurePass123"}' | gcloud secrets versions add kb7-loinc-credentials-production --data-file=-
```

**Verification**:
```bash
# Verify the secret was updated
gcloud secrets versions list kb7-loinc-credentials-production
```

---

## 4️⃣ GitHub Personal Access Token

### Creation Steps

1. **Go to GitHub Settings**:
   - Log in to GitHub: https://github.com/
   - Click your profile picture (top right) → **"Settings"**
   - Scroll down to **"Developer settings"** (left sidebar, at bottom)
   - Click **"Personal access tokens"** → **"Tokens (classic)"**
   - Or directly: https://github.com/settings/tokens

2. **Generate New Token**:
   - Click **"Generate new token"** → **"Generate new token (classic)"**
   - You may need to confirm your password

3. **Configure Token**:
   - **Note**: Enter description like "KB-7 Terminology Factory"
   - **Expiration**: Select "No expiration" or "Custom" (90 days recommended)
   - **Scopes**: Check these permissions:
     - ✅ `repo` (Full control of private repositories)
       - This includes: repo:status, repo_deployment, public_repo, repo:invite
     - ✅ `workflow` (Update GitHub Action workflows)
   - Scroll to bottom and click **"Generate token"**

4. **Copy Token**:
   - **CRITICAL**: Copy the token immediately (starts with `ghp_`)
   - GitHub will NEVER show this token again
   - If you lose it, you'll need to generate a new one

### Update in GCP

```bash
# Replace with your actual GitHub token
echo -n "ghp_YOUR_GITHUB_TOKEN_HERE" | gcloud secrets versions add kb7-github-token-production --data-file=-
```

**Verification**:
```bash
# Verify the secret was updated
gcloud secrets versions list kb7-github-token-production
```

---

## 📦 All-in-One Update Script

Once you have all credentials, you can update them all at once:

```bash
#!/bin/bash
# save this as update-secrets.sh

# NHS TRUD API Key
echo -n "nhstrud_YOUR_KEY" | gcloud secrets versions add kb7-nhs-trud-api-key-production --data-file=-

# UMLS API Key
echo -n "YOUR_UMLS_KEY" | gcloud secrets versions add kb7-umls-api-key-production --data-file=-

# LOINC Credentials (JSON format)
echo -n '{"username":"your@email.com","password":"your_pass"}' | gcloud secrets versions add kb7-loinc-credentials-production --data-file=-

# GitHub Token
echo -n "ghp_YOUR_TOKEN" | gcloud secrets versions add kb7-github-token-production --data-file=-

echo "✅ All secrets updated successfully!"
```

**Usage**:
```bash
chmod +x update-secrets.sh
./update-secrets.sh
```

---

## ✅ Verification Steps

### 1. Check All Secrets Were Updated

```bash
# List all secrets with their versions
gcloud secrets list --filter="name:kb7" --format="table(name,createTime)"

# Check each secret has at least 2 versions (original placeholder + your update)
gcloud secrets versions list kb7-nhs-trud-api-key-production
gcloud secrets versions list kb7-umls-api-key-production
gcloud secrets versions list kb7-loinc-credentials-production
gcloud secrets versions list kb7-github-token-production
```

Expected output: Each should show version 2 (or higher) as ENABLED

### 2. Test Workflow with Real Credentials

```bash
# Trigger workflow manually
gcloud workflows execute kb7-factory-workflow-production \
  --location=us-central1 \
  --data='{"trigger":"manual-test"}'

# Get execution ID from output, then wait 2-3 minutes and check status
EXECUTION_ID="<paste-execution-id-here>"
gcloud workflows executions describe $EXECUTION_ID \
  --workflow=kb7-factory-workflow-production \
  --location=us-central1
```

**Success indicators**:
- `state: SUCCEEDED`
- `result: {"status":"success",...}`
- No error messages in workflow logs

### 3. Check Service Logs

```bash
# View recent Cloud Run logs
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name:kb7" \
  --limit=50 \
  --format="table(timestamp,severity,textPayload)" \
  --freshness=10m
```

**Look for**:
- "Download complete" messages
- "Uploaded to GCS" messages
- No "authentication failed" or "invalid credentials" errors

### 4. Verify Downloaded Files

```bash
# Check if files were downloaded to GCS
gsutil ls gs://sincere-hybrid-477206-h2-kb-sources-production/

# List recent files
gsutil ls -lh gs://sincere-hybrid-477206-h2-kb-sources-production/** | head -20
```

Expected: You should see SNOMED, RxNorm, and LOINC files with recent timestamps

---

## 🔐 Security Best Practices

### 1. API Key Storage
- ✅ **DO**: Store keys in Secret Manager (encrypted at rest)
- ✅ **DO**: Use service accounts with minimal permissions
- ✅ **DO**: Rotate keys every 90 days
- ❌ **DON'T**: Commit keys to Git
- ❌ **DON'T**: Share keys via email or chat
- ❌ **DON'T**: Store keys in environment variables on local machine

### 2. GitHub Token Security
- Use tokens with minimum required scopes
- Set expiration dates (90 days recommended)
- Rotate tokens before expiration
- Revoke tokens if compromised: https://github.com/settings/tokens

### 3. Monitor Secret Access

```bash
# View who accessed secrets
gcloud logging read "resource.type=secretmanager.googleapis.com/Secret AND protoPayload.methodName=AccessSecretVersion" \
  --limit=20 \
  --format="table(timestamp,protoPayload.authenticationInfo.principalEmail,resource.labels.secret_id)"
```

---

## 🚨 Troubleshooting

### Issue: "Secret version already exists"
**Solution**: Secret Manager keeps version history. Your new version should be version 2.
```bash
# Force add new version
echo -n "new_key" | gcloud secrets versions add secret-name --data-file=-
```

### Issue: "Permission denied" when updating secrets
**Solution**: Ensure you have `secretmanager.versions.add` permission:
```bash
# Grant yourself permission
gcloud projects add-iam-policy-binding sincere-hybrid-477206-h2 \
  --member="user:onkarshahi@vaidshala.com" \
  --role="roles/secretmanager.admin"
```

### Issue: Services still failing after updating secrets
**Solutions**:
1. Wait 1-2 minutes for secret propagation
2. Restart Cloud Run services:
   ```bash
   # Force new revision deployment
   gcloud run services update kb7-snomed-downloader-production \
     --region=us-central1 \
     --update-env-vars=FORCE_UPDATE=$(date +%s)
   ```
3. Check service account has `secretAccessor` role:
   ```bash
   gcloud secrets get-iam-policy kb7-nhs-trud-api-key-production
   ```

### Issue: "Invalid API key" errors
**Solutions**:
- **NHS TRUD**: Ensure key starts with `nhstrud_`
- **UMLS**: Verify key is correct length (typically 36+ chars)
- **LOINC**: Check JSON format is correct (no extra spaces, valid JSON)
- **GitHub**: Ensure token starts with `ghp_` and has `workflow` scope

---

## 📞 Support Resources

### NHS TRUD Support
- Email: [email protected]
- Help: https://isd.digital.nhs.uk/trud/support

### UMLS Support
- Email: [email protected]
- Help: https://www.nlm.nih.gov/research/umls/knowledge_sources/metathesaurus/release/support.html

### LOINC Support
- Email: [email protected]
- Help: https://loinc.org/contact/

### GitHub Support
- Help: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens
- Community: https://github.community/

---

## ✅ Completion Checklist

- [ ] NHS TRUD account created and API key obtained
- [ ] UMLS account created and API key obtained
- [ ] LOINC account created with username/password
- [ ] GitHub Personal Access Token generated
- [ ] All 4 secrets updated in GCP Secret Manager
- [ ] Secrets verified (version 2 exists for each)
- [ ] Workflow test execution completed successfully
- [ ] Downloaded files appear in GCS bucket
- [ ] No authentication errors in logs

---

**Once all credentials are updated, your KB-7 Knowledge Factory is ready for production!** 🎉

The system will automatically download and process terminology updates on the 1st of every month at 2 AM UTC.
