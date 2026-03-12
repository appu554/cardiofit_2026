# KB-7 Knowledge Factory - API Response Structure Issue

**Date**: 2025-11-26
**Discovery**: Post TypeError Fix Investigation
**Status**: 🔴 **ROOT CAUSE IDENTIFIED**

---

## Executive Summary

After successfully fixing the TypeError issue at line 498, end-to-end testing revealed the workflow still cannot complete successfully. Investigation shows the workflow is trying to access job execution fields at the root level (`execution_info.taskCount`), but the Cloud Run v2 API returns these fields nested within `spec` and `status` objects.

**TypeError Fix**: ✅ **WORKING** (no crashes, proper JSON serialization)
**New Issue**: ❌ **API Response Structure Mismatch**

---

## Root Cause Analysis

### Workflow Expectations vs Actual API Response

**What the Workflow Expects** ([kb-factory-jobs-workflow-v2.yaml:457](../gcp/workflows/kb-factory-jobs-workflow-v2.yaml#L457)):
```yaml
text: ${"Execution taskCount=" + string(default(execution_info.taskCount, 0)) + " runningCount=" + string(default(execution_info.runningCount, 0)) + " succeededCount=" + string(default(execution_info.succeededCount, 0)) + " failedCount=" + string(default(execution_info.failedCount, 0))}
```

The workflow expects fields at root level:
- `execution_info.taskCount`
- `execution_info.runningCount`  
- `execution_info.succeededCount`
- `execution_info.failedCount`

**What the API Actually Returns**:
```json
{
  "apiVersion": "run.googleapis.com/v1",
  "kind": "Execution",
  "metadata": { ... },
  "spec": {
    "parallelism": 1,
    "taskCount": 1,
    "template": { ... }
  },
  "status": {
    "completionTime": "2025-11-26T10:31:29.567069Z",
    "succeededCount": 1,
    "observedGeneration": 1,
    "startTime": "2025-11-26T10:30:20.730563Z",
    "conditions": [ ... ]
  }
}
```

The API returns nested fields:
- `execution_info.spec.taskCount`
- `execution_info.status.succeededCount`
- `execution_info.status.runningCount` (when present)
- `execution_info.status.failedCount` (when present)

### Why This Causes KeyError

When the workflow tries to access `execution_info.taskCount`, Cloud Workflows looks for a field called `taskCount` at the root level of the execution_info object. Since it doesn't exist there (it's nested in `spec.taskCount`), Cloud Workflows throws:

```
KeyError: key not found: taskCount
```

This error is caught by the `try/except` block at line 492, which logs it (with our fixed JSON serialization) and retries. But the field will NEVER exist at the root level, so the workflow loops indefinitely until timeout.

---

## Evidence

### Job Execution API Response
```bash
$ gcloud run jobs executions describe kb7-snomed-job-production-t9mml --region=us-central1 --format=json
```

**Key Fields**:
```json
{
  "spec": {
    "taskCount": 1  ← Field is HERE, not at root
  },
  "status": {
    "succeededCount": 1  ← Field is HERE, not at root
  }
}
```

### Workflow Execution Timeline
- **10:30:15**: Workflow started (execution ID `2ca73032-f73f-499b-b177-520eb540eae1`)
- **10:31:29**: SNOMED job completed successfully (1m8s duration)
- **10:31:25**: RxNorm job completed successfully
- **10:32:01**: LOINC job completed successfully
- **10:31:57**: Workflow polling attempt 1 → `KeyError: taskCount`
- **10:33:00**: Workflow polling attempt 2 → `KeyError: taskCount`
- **10:34:00**: Workflow polling attempt 3 → `KeyError: taskCount`
- ...(continues)...
- **10:44:17**: Workflow cancelled manually after 14 minutes

---

## Required Fix

### Update Field Access Patterns

The workflow needs to access nested fields instead of root-level fields:

**Lines to Update**:

**1. Line 457 - Log execution info**:
```yaml
# BEFORE (BROKEN):
text: ${"Execution taskCount=" + string(default(execution_info.taskCount, 0)) + " runningCount=" + string(default(execution_info.runningCount, 0)) + " succeededCount=" + string(default(execution_info.succeededCount, 0)) + " failedCount=" + string(default(execution_info.failedCount, 0))}

# AFTER (FIXED):
text: ${"Execution taskCount=" + string(default(map.get(execution_info, "spec", {}).get("taskCount", 0), 0)) + " runningCount=" + string(default(map.get(execution_info, "status", {}).get("runningCount", 0), 0)) + " succeededCount=" + string(default(map.get(execution_info, "status", {}).get("succeededCount", 0), 0)) + " failedCount=" + string(default(map.get(execution_info, "status", {}).get("failedCount", 0), 0))}
```

**2. Line 462 - Check success condition**:
```yaml
# BEFORE (BROKEN):
- condition: ${default(execution_info.succeededCount, 0) > 0 AND default(execution_info.runningCount, 0) == 0}

# AFTER (FIXED):
- condition: ${default(map.get(map.get(execution_info, "status", {}), "succeededCount", 0), 0) > 0 AND default(map.get(map.get(execution_info, "status", {}), "runningCount", 0), 0) == 0}
```

**3. Line 472 - Check failure condition**:
```yaml
# BEFORE (BROKEN):
- condition: ${default(execution_info.failedCount, 0) > 0 AND default(execution_info.runningCount, 0) == 0 AND default(execution_info.succeededCount, 0) == 0}

# AFTER (FIXED):
- condition: ${default(map.get(map.get(execution_info, "status", {}), "failedCount", 0), 0) > 0 AND default(map.get(map.get(execution_info, "status", {}), "runningCount", 0), 0) == 0 AND default(map.get(map.get(execution_info, "status", {}), "succeededCount", 0), 0) == 0}
```

**4. Line 482 - Check cancelled condition**:
```yaml
# BEFORE (BROKEN):
- condition: ${default(execution_info.cancelledCount, 0) > 0}

# AFTER (FIXED):
- condition: ${default(map.get(map.get(execution_info, "status", {}), "cancelledCount", 0), 0) > 0}
```

---

## Alternative: Simpler Field Extraction

Instead of using complex `map.get()` chains, we can extract fields into variables first:

```yaml
- get_execution_safe:
    try:
      steps:
        - get_execution:
            call: googleapis.run.v2.projects.locations.jobs.executions.get
            args:
              name: ${execution_name}
            result: execution_info
        
        - extract_status_fields:
            assign:
              - task_count: ${default(map.get(map.get(execution_info, "spec", {}), "taskCount", 0), 0)}
              - running_count: ${default(map.get(map.get(execution_info, "status", {}), "runningCount", 0), 0)}
              - succeeded_count: ${default(map.get(map.get(execution_info, "status", {}), "succeededCount", 0), 0)}
              - failed_count: ${default(map.get(map.get(execution_info, "status", {}), "failedCount", 0), 0)}
              - cancelled_count: ${default(map.get(map.get(execution_info, "status", {}), "cancelledCount", 0), 0)}
        
        - log_execution_info:
            call: sys.log
            args:
              text: ${"Execution taskCount=" + string(task_count) + " runningCount=" + string(running_count) + " succeededCount=" + string(succeeded_count) + " failedCount=" + string(failed_count)}
              severity: INFO
        
        - check_status:
            switch:
              - condition: ${succeeded_count > 0 AND running_count == 0}
                steps:
                  - assign_success:
                      assign:
                        - final_status:
                            status: "succeeded"
                            execution: ${execution_info}
                        - is_terminal: true
                next: check_terminal
```

---

## Impact Assessment

### Before Fix
- ✅ Jobs execute successfully
- ✅ Result files written to GCS
- ❌ Workflow cannot detect job completion
- ❌ Workflow stuck in infinite polling loop
- ❌ Timeout after 120 minutes (max_attempts)

### After Fix
- ✅ Jobs execute successfully
- ✅ Result files written to GCS
- ✅ Workflow can read job execution status
- ✅ Workflow detects completion and proceeds
- ✅ GitHub Dispatcher receives real GCS keys
- ✅ Complete pipeline operational

---

## Testing Plan

1. **Apply Fix**: Update workflow with correct field access patterns
2. **Deploy**: Deploy as new revision
3. **Test**: Run end-to-end workflow execution
4. **Verify**:
   - Jobs complete successfully ✅ (already verified)
   - Workflow reads execution status correctly (NEW)
   - Workflow detects success and exits polling (NEW)
   - GitHub Dispatcher executes with real GCS keys (NEW)

---

## Files to Modify

1. **[gcp/workflows/kb-factory-jobs-workflow-v2.yaml](../gcp/workflows/kb-factory-jobs-workflow-v2.yaml)**
   - Lines 457, 462, 472, 482 - Update field access patterns

---

## Summary

🔍 **Root Cause**: API response structure mismatch - fields are nested in `spec` and `status` objects, not at root level

💡 **Solution**: Use `map.get()` to safely access nested fields or extract to variables first

📊 **Impact**: Blocks complete pipeline operation - workflow cannot detect job completion

⚡ **Priority**: 🔴 CRITICAL - Must fix before production deployment

**Status**: Ready to implement fix

---

**Created**: 2025-11-26
**Author**: Claude Code
**Session**: API Response Structure Investigation
**Previous Fix**: TypeError at line 498 (COMPLETE)
**Current Fix**: API field access patterns (PENDING)
