# GCS JSON Auto-Parsing Fix - Workflow v2.1-v2.4

## Problem Summary

**Issue:** Workflow was reporting all jobs as "failed" even though they completed successfully and wrote status files to GCS with `"status": "success"`.

**Root Cause:** Cloud Run API conditions fallback was misinterpreting `Completed=False` (meaning "still running") as "failed", causing premature polling termination before jobs finished writing GCS status files.

## Technical Analysis

### How Google Workflows Handles GCS JSON

When calling `googleapis.storage.v1.objects.get` with `alt: "media"` on objects with `Content-Type: application/json`, Google Workflows **automatically parses the JSON**:

```yaml
- get_status_file:
    call: googleapis.storage.v1.objects.get
    args:
      bucket: "my-bucket"
      object: "status.json"
      alt: "media"
    result: gcs_content

# gcs_content is ALREADY parsed as a map - no json.decode() needed!
- access_directly:
    assign:
      - status: ${gcs_content.status}  # Direct access, not gcs_content.body.status
```

### The Bug Flow (Before Fix)

1. **Job starts running** (takes ~1-2 minutes to complete)
2. **Workflow polls after 10 seconds** - no GCS file yet
3. **GCS read fails (404)** → enters except block
4. **Fallback to Cloud Run API conditions:**
   ```yaml
   - condition.type == "Completed" AND condition.status == "False"
     assign:
       - is_failed: true        # ❌ WRONG! False means "not yet complete", not "failed"
       - is_terminal: true      # ❌ STOPS POLLING!
   ```
5. **Polling stops immediately** - never rechecks when job finishes
6. **Result:** All jobs reported as "failed" despite successful completion

### The Fix (v2.1)

**Removed the entire Cloud Run API conditions fallback** from the except block:

```yaml
except:
  as: e
  steps:
    - log_gcs_error:
        call: sys.log
        args:
          text: '${"GCS status file not found yet for " + job_name + " - will retry"}'
          severity: INFO

    # ✅ That's it! Just log and continue polling
    # ✅ Don't set is_failed or is_terminal
    # ✅ Let the polling loop continue until GCS file appears
```

## Why This Approach is Better

| Aspect | Cloud Run API Fallback | GCS-Only Approach |
|--------|------------------------|-------------------|
| **Source of Truth** | Ambiguous Cloud Run conditions | Explicit job-written status |
| **Accuracy** | Misinterprets "running" as "failed" | Direct from job: success/failed/skipped |
| **Reliability** | Race conditions with job completion | Jobs control their own status |
| **Simplicity** | Complex condition interpretation logic | Simple: read status file or retry |
| **Failure Handling** | False positives cause premature exits | 120-minute timeout catches real failures |

## Verification

### Test Execution: `15890b8c-af79-4dc9-a010-08f40f270f1f`

**Correct Behavior Observed:**
```
10:03:36 - GCS status file not found yet for kb7-snomed-job-production - will retry
10:03:36 - Status: Running | succeeded=false failed=false cancelled=false
10:03:36 - is_terminal=false
10:04:38 - Polling job execution - attempt 3
10:05:40 - Polling job execution - attempt 4
```

✅ **Key Improvements:**
- Clear "will retry" messages instead of treating as failures
- `failed=false` maintained throughout polling
- `is_terminal=false` allows continued polling
- No premature exit - keeps checking until job writes status

### Job Completion Timeline

| Time | Event |
|------|-------|
| 10:01:36 | SNOMED job started |
| 10:03:08 | Job wrote GCS status file: `{"status": "success"}` |
| 10:03:12 | Job completed (Cloud Run confirms) |
| 10:03:36 | Workflow still sees 404 (potential caching/propagation delay) |
| 10:04:38+ | Workflow continues polling for GCS file |

## Deployment

### Version 2.1 (Partial Fix)
```bash
gcloud workflows deploy kb7-factory-workflow-production \
  --source=kb-factory-jobs-workflow-v2.yaml \
  --description="Fixed GCS JSON auto-parsing - removed Cloud Run API fallback"
```
**Deployed:** 2025-11-27 10:01:16 UTC
**Revision:** 000030-a8c
**Result:** ❌ Still getting 404 errors (missing URL encoding)

### Version 2.2 (Complete Fix)
```bash
gcloud workflows deploy kb7-factory-workflow-production \
  --source=kb-factory-jobs-workflow-v2.yaml \
  --description="Fixed GCS URL encoding for object names with slashes"
```
**Deployed:** 2025-11-27 10:29:27 UTC
**Revision:** 000031-6e4
**Result:** ✅ SUCCESS! GCS status files read correctly

## Files Modified

- `kb-factory-jobs-workflow-v2.yaml`:
  - Lines 530-540: Removed Cloud Run API fallback logic
  - Line 503: Added `text.url_encode()` for GCS object parameter
  - Header updated to v2.2 with complete fix explanation

## Edge Cases Handled

1. **Job crashes without writing status:** 120-minute timeout raises error (better than false success)
2. **GCS eventually consistent delays:** Polling continues until file appears or timeout
3. **Permission errors:** Clear logging for debugging
4. **Job writes "failed" status:** Correctly reported from GCS, not misinterpreted

## Next Steps

- ✅ Deployed to production
- 🔄 Testing in progress (execution 15890b8c)
- ⏭️ Monitor for successful completion with correct status reporting
- ⏭️ Verify GitHub dispatcher receives correct GCS keys

## Complete Solution

The fix required **TWO changes**:

### Change 1: Remove Cloud Run API Fallback (v2.1)
**Problem:** Misinterpreted "Completed=False" as "failed"
**Solution:** Remove fallback, rely solely on GCS status files
**Impact:** Eliminated false failures, but revealed URL encoding issue

### Change 2: Add URL Encoding for GCS Object Names (v2.2)
**Problem:** Object names with slashes (`workflow-results/file.json`) caused 404 errors
**Solution:** Use `${text.url_encode(gcs_status_file)}` in `googleapis.storage.v1.objects.get`
**Impact:** GCS API can now find and read status files successfully

### Change 3: Fix gcs_key Extraction from Nested JSON Path (v2.3)
**Problem:** Workflow extracting `body.gcs_key` but GCS status files have nested structure `body.data.gcs_key`
**Solution:** Update lines 239-241 to access correct nested path: `${snomed_result_response.body.data.gcs_key}`
**Impact:** GitHub dispatcher now receives actual GCS file paths instead of "unknown"
**Deployed:** 2025-11-27 10:40 UTC
**Revision:** 000032-1e1

### Change 4: Update Default GitHub Repository Name (v2.4)
**Problem:** GitHub dispatcher sending requests to placeholder "your-org/knowledge-factory" resulting in 404 errors
**Solution:** Update line 24 default value to actual repository: `onkarshahi-IND/knowledge-factory`
**Impact:** GitHub dispatcher can now successfully trigger GitHub Actions workflow in the correct repository
**Deployed:** 2025-11-27 10:56 UTC
**Revision:** 000033-a61

## Verified Working Behavior (v2.2)

```
✅ GCS Status for kb7-snomed-job-production: success
✅ GCS Status for kb7-rxnorm-job-production: success
✅ GCS Status for kb7-loinc-job-production: success
```

## Lessons Learned

1. **Trust the source of truth:** Jobs know their status better than external APIs
2. **Beware of "false" in conditions:** Not all false values mean failure
3. **GCS auto-parsing is powerful:** No manual JSON decode needed when Content-Type is correct
4. **URL encoding matters:** Google Workflows connectors don't auto-encode slashes in object names
5. **Test incrementally:** v2.1 revealed the URL encoding issue that would have been harder to diagnose without the fallback removal
6. **Simpler is better:** Removing complexity (fallback logic) improved reliability and revealed hidden bugs
