# GitHub Actions Pipeline Troubleshooting Guide

## Stage 2: Extract - Common Failures

### How to View the Error

1. **Go to GitHub Actions:**
   - Navigate to: `https://github.com/onkarshahi-IND/knowledge-factory/actions`
   - Click on the failed workflow run (should be at the top)
   - Click on "Stage 2: Extract Source Files" job
   - Expand the failed step to see the error message

2. **What to Look For:**
   - Error messages about file permissions
   - "File not found" errors
   - "No space left on device" errors
   - Unzip command failures
   - GCS authentication errors

### Common Stage 2 Issues

#### Issue 1: GCS Authentication Failed
**Error Message:**
```
ERROR: (gcloud.storage.cp) You do not have permission to access...
```

**Cause:** GitHub Actions doesn't have GCS credentials configured

**Fix:**
You need to set up GCP authentication in GitHub Actions. Add this to your workflow before the download step:

```yaml
- name: Authenticate to Google Cloud
  uses: google-github-actions/auth@v1
  with:
    credentials_json: ${{ secrets.GCP_SA_KEY }}
```

**Required Secret:**
- Go to GitHub repo → Settings → Secrets and variables → Actions
- Add secret named `GCP_SA_KEY` with service account JSON key

**To create service account key:**
```bash
gcloud iam service-accounts keys create ~/gcp-key.json \
  --iam-account=kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
```

---

#### Issue 2: Files Not Downloaded from GCS
**Error Message:**
```
unzip: cannot find or open *.zip
```

**Cause:** Stage 1 (Download) didn't actually download files, or paths are wrong

**Fix:**
Check that Stage 1 outputs are being passed correctly:

```yaml
download:
  outputs:
    snomed_file: ${{ steps.download.outputs.snomed_file }}  # Must match
    rxnorm_file: ${{ steps.download.outputs.rxnorm_file }}
    loinc_file: ${{ steps.download.outputs.loinc_file }}

extract:
  needs: download
  steps:
    - name: Extract files
      run: |
        echo "SNOMED file: ${{ needs.download.outputs.snomed_file }}"  # Debug output
        unzip "${{ needs.download.outputs.snomed_file }}"
```

---

#### Issue 3: Incorrect File Paths
**Error Message:**
```
unzip: cannot find or open snomed-ct/20251101/SnomedCT_...
```

**Cause:** The GCS keys passed are full paths, but files downloaded to current directory

**Current dispatcher payload:**
```json
{
  "snomed_gcs_key": "snomed-ct/20251101/SnomedCT_International...",
  "rxnorm_gcs_key": "rxnorm/10062025/RxNorm_full_10062025.zip",
  "loinc_gcs_key": "loinc/2.81/loinc-complete-2.81.zip"
}
```

**Fix Option A - Update GitHub Actions to use basename:**
```yaml
- name: Download SNOMED
  run: |
    FILENAME=$(basename "${{ github.event.client_payload.snomed_gcs_key }}")
    gcloud storage cp "gs://bucket/${{ github.event.client_payload.snomed_gcs_key }}" .
    echo "snomed_file=$FILENAME" >> $GITHUB_OUTPUT
```

**Fix Option B - Update dispatcher to send just filenames:**
Modify the dispatcher to extract filenames before sending to GitHub.

---

#### Issue 4: Unzip Command Not Installed
**Error Message:**
```
unzip: command not found
```

**Fix:**
Add unzip installation step:
```yaml
- name: Install dependencies
  run: |
    sudo apt-get update
    sudo apt-get install -y unzip
```

---

#### Issue 5: Disk Space Issues
**Error Message:**
```
No space left on device
```

**Fix:**
SNOMED/RxNorm files are large. Free up space or increase runner size:
```yaml
extract:
  runs-on: ubuntu-latest  # Has 14GB disk
  # OR
  runs-on: ubuntu-latest-4-cores  # More resources
```

Clear space before extraction:
```yaml
- name: Free up disk space
  run: |
    sudo rm -rf /usr/share/dotnet
    sudo rm -rf /opt/ghc
    df -h
```

---

## Quick Diagnostic Commands

### Check What the Dispatcher Actually Sent
```bash
# Check recent workflow runs
curl -s https://api.github.com/repos/onkarshahi-IND/knowledge-factory/actions/runs?per_page=1 \
  | jq '.workflow_runs[0] | {id, status, conclusion, created_at}'

# Get run details
RUN_ID="<paste_run_id_here>"
curl -s https://api.github.com/repos/onkarshahi-IND/knowledge-factory/actions/runs/$RUN_ID \
  | jq '{event, head_branch, conclusion}'
```

### Check GCS Files Exist
```bash
# Verify files are in GCS
gsutil ls gs://sincere-hybrid-477206-h2-kb-sources-production/snomed-ct/20251101/
gsutil ls gs://sincere-hybrid-477206-h2-kb-sources-production/rxnorm/10062025/
gsutil ls gs://sincere-hybrid-477206-h2-kb-sources-production/loinc/2.81/
```

### Check Dispatcher Logs Again
```bash
gcloud logging read \
  "resource.type=cloud_run_job AND resource.labels.job_name=kb7-github-dispatcher-job-production" \
  --limit=30 --format=json --freshness=10m \
  | jq -r '.[] | select(.textPayload != null) | .textPayload' \
  | grep -A5 "client_payload"
```

---

## Most Likely Issue: GCS Authentication

**The most common Stage 2 failure is missing GCP authentication in GitHub Actions.**

### Quick Fix Steps:

1. **Create Service Account Key:**
```bash
cd ~/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

gcloud iam service-accounts keys create gcp-github-key.json \
  --iam-account=kb7-functions-production@sincere-hybrid-477206-h2.iam.gserviceaccount.com
```

2. **Copy key contents:**
```bash
cat gcp-github-key.json
```

3. **Add to GitHub Secrets:**
   - Go to: `https://github.com/onkarshahi-IND/knowledge-factory/settings/secrets/actions`
   - Click "New repository secret"
   - Name: `GCP_SA_KEY`
   - Value: [paste entire JSON]
   - Click "Add secret"

4. **Update Workflow File:**
Add this authentication step at the beginning of the download job:

```yaml
download:
  name: "Stage 1: Download Source Files"
  runs-on: ubuntu-latest
  steps:
    - name: Checkout repository
      uses: actions/checkout@v4

    # ADD THIS AUTHENTICATION STEP
    - name: Authenticate to Google Cloud
      uses: google-github-actions/auth@v1
      with:
        credentials_json: ${{ secrets.GCP_SA_KEY }}

    - name: Set up Cloud SDK
      uses: google-github-actions/setup-gcloud@v1

    - name: Download files
      # ... rest of download steps
```

5. **Commit and Push:**
```bash
git add .github/workflows/kb-factory.yml
git commit -m "Add GCP authentication for GitHub Actions"
git push origin main
```

6. **Trigger Again:**
Run the GCP workflow again to trigger GitHub Actions with authentication in place.

---

## How to Share Error Details

To help debug further, please share:

1. **Exact error message** from GitHub Actions logs
2. **Which step failed** (Download SNOMED? Extract? Both?)
3. **Run ID** from GitHub Actions URL

Example:
```
Run ID: 1234567890
Stage: Stage 2: Extract Source Files
Step: Extract SNOMED CT
Error: "unzip: cannot find or open /home/runner/work/..."
```

---

## Testing Locally

You can test the extraction locally before pushing:

```bash
cd knowledge-factory

# Download a test file
gsutil cp gs://sincere-hybrid-477206-h2-kb-sources-production/snomed-ct/20251101/SnomedCT_InternationalRF2_PRODUCTION_20251101T120000Z.zip /tmp/

# Try extracting
cd /tmp
unzip SnomedCT_InternationalRF2_PRODUCTION_20251101T120000Z.zip

# Check size
du -sh *
```

If this works, the issue is with GitHub Actions configuration, not the files themselves.
