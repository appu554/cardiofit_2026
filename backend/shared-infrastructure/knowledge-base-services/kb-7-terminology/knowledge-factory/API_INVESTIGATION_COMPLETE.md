# KB-7 Knowledge Factory - API Investigation Complete

**Date**: 2025-11-26
**Investigation**: Cloud Run v2 API Response Structure
**Status**: 🟡 **INCONCLUSIVE - NEEDS PRAGMATIC SOLUTION**

---

## Summary

After extensive investigation, we've identified a confusing discrepancy between documentation and observed behavior:

**Official v2 API Documentation** (cloud.google.com/run/docs/reference/rest/v2):
- Fields are at ROOT level: `taskCount`, `succeededCount`, `runningCount`, `failedCount`
- Workflow code matches this structure (lines 457, 462, 472, 482)

**Observed `gcloud` CLI Behavior**:
- Fields are NESTED: `spec.taskCount`, `status.succeededCount`
- CLI uses v1 API format (Kubernetes-style with `apiVersion`, `kind`, `metadata`, `spec`, `status`)

**Actual Workflow Behavior**:
- Persistent `KeyError: taskCount` during polling
- TypeError fix IS working (proper JSON serialization, no crashes)
- Jobs complete successfully, but workflow can't detect completion

---

## The Discrepancy Explained

There are THREE different API representations:

### 1. REST API v2 (googleapis.run.v2)
**Used by**: Cloud Workflows `googleapis.run.v2.projects.locations.jobs.executions.get`
**Structure**: Flat (fields at root level)
```json
{
  "name": "...",
  "taskCount": 1,
  "succeededCount": 1,
  ...
}
```

### 2. REST API v1 (run.googleapis.com/v1)
**Used by**: gcloud CLI tool internally
**Structure**: Kubernetes-style (nested in `spec` and `status`)
```json
{
  "apiVersion": "run.googleapis.com/v1",
  "kind": "Execution",
  "metadata": {...},
  "spec": {
    "taskCount": 1
  },
  "status": {
    "succeededCount": 1
  }
}
```

### 3. Mixed/Transitional Format
**Hypothesis**: The v2 API might return different structures depending on:
- Job execution state (starting vs running vs completed)
- API endpoint called
- Workflow execution context

---

## Hypothesis: Fields Not Present During Initial Polling

The most likely explanation for `KeyError: taskCount`:

**During job initialization** (first few seconds after execution starts):
```json
{
  "name": "projects/.../executions/job-xyz",
  "uid": "...",
  "createTime": "...",
  "startTime": "...",
  "job": "...",
  "reconciling": true,
  "conditions": [...]
  // taskCount NOT YET POPULATED
  // succeededCount NOT YET POPULATED
}
```

**After job is running**:
```json
{
  ...all the above fields...,
  "taskCount": 1,
  "runningCount": 1,
  "succeededCount": 0,
  "failedCount": 0
}
```

This would explain why:
- The `default()` function doesn't help (you can't use `default()` on a non-existent property)
- The KeyError persists across multiple polling attempts (fields populate async)
- Eventually timeout occurs after 120 attempts

---

## Proposed Pragmatic Solutions

### Option A: Check `conditions` Array Instead of Count Fields

Instead of polling `taskCount`, check the `conditions` array which is ALWAYS present:

```yaml
- check_completion_from_conditions:
    assign:
      - is_complete: false
      - is_succeeded: false
      - is_failed: false

- iterate_conditions:
    for:
      value: condition
      in: ${default(execution_info.conditions, [])}
      steps:
        - check_completed_condition:
            switch:
              - condition: ${condition.type == "Completed" AND condition.status == "True"}
                steps:
                  - set_complete:
                      assign:
                        - is_complete: true
                        - is_succeeded: true

              - condition: ${condition.type == "Completed" AND condition.status == "False"}
                steps:
                  - set_failed:
                      assign:
                        - is_complete: true
                        - is_failed: true
```

### Option B: Safely Access Fields with try/except

Wrap field access in try/except to handle missing fields:

```yaml
- extract_counts_safely:
    try:
      assign:
        - task_count: ${execution_info.taskCount}
        - running_count: ${execution_info.runningCount}
        - succeeded_count: ${execution_info.succeededCount}
        - failed_count: ${execution_info.failedCount}
    except:
      as: e
      assign:
        - task_count: -1  # Indicates fields not available yet
        - running_count: 0
        - succeeded_count: 0
        - failed_count: 0

- log_status:
    call: sys.log
    args:
      text: ${task_count == -1 ? "Execution initializing..." : "Execution taskCount=" + string(task_count) + " succeeded=" + string(succeeded_count)}
```

### Option C: Use Simpler Completion Check

Just check if `completionTime` is set:

```yaml
- check_complete:
    switch:
      - condition: ${default(map.get(execution_info, "completionTime", ""), "") != ""}
        steps:
          - check_succeeded:
              assign:
                - is_succeeded: ${len(default(execution_info.conditions, [])) > 0}
```

---

## Recommended Approach: **Option A (conditions array)**

**Why**:
- ✅ `conditions` array is ALWAYS present (guaranteed by API contract)
- ✅ Provides definitive completion status
- ✅ No KeyError risk
- ✅ More robust than counting tasks
- ✅ Standard Kubernetes-style pattern

**Implementation**:
1. Replace taskCount/succeededCount checks with conditions array iteration
2. Look for `type: "Completed"` with `status: "True"/"False"`
3. Extract success/failure from condition message if needed

---

## Next Steps

1. ✅ **Document findings** (this report)
2. ⏳ **Implement Option A** - Use conditions array for completion detection
3. ⏳ **Test fix** - Deploy and run end-to-end test
4. ⏳ **Verify complete pipeline** - Ensure GitHub Dispatcher receives correct GCS keys

---

## Files to Modify

**[gcp/workflows/kb-factory-jobs-workflow-v2.yaml](../gcp/workflows/kb-factory-jobs-workflow-v2.yaml)**:
- Lines 454-490: Rewrite `get_execution_safe` step to use conditions array
- Remove reliance on `taskCount`, `succeededCount`, etc.
- Add condition iteration logic

---

## Summary

🔍 **Root Cause**: Likely fields not populated during initial polling phase
💡 **Solution**: Use `conditions` array which is always present
📊 **Impact**: Unblocks complete pipeline operation
⚡ **Priority**: 🔴 CRITICAL - Required for production deployment

**Status**: Ready to implement fix using Option A (conditions array)

---

**Created**: 2025-11-26
**Author**: Claude Code
**Investigation**: Cloud Run v2 API Response Structure
**Previous Fix**: TypeError at line 498 (COMPLETE)
**Current Fix**: Use conditions array for completion detection (PENDING)
