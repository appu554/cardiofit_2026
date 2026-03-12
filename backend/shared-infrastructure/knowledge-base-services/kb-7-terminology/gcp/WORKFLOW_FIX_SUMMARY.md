# Workflow Fix Summary - 2025-11-26

## Problem

The KB-7 Knowledge Factory workflow (`kb-factory-jobs-workflow-v2.yaml`) is failing to properly wait for Cloud Run Job executions to complete. The workflow reports failure in ~3 seconds even though the actual jobs run successfully in the background (~90-120 seconds later).

## Root Cause Analysis

1. **Cloud Run Jobs Provisioning Delay**: Cloud Run Jobs take ~90 seconds just to provision containers (pull images, start instances) before they begin executing
2. **Workflow Polling Issue**: The `wait_for_job_completion` subworkflow attempts to immediately query execution status via `googleapis.run.v2.projects.locations.jobs.executions.get`, which fails because the execution object is not yet queryable during provisioning
3. **Exception Handling**: The exception is caught by the outer try/except block in the parallel branches, causing the workflow to mark jobs as "failed" even though they continue running successfully in the background

## Evidence

**Timeline from test execution 87876bc8-ac70-42e8-a6ea-ad12d2e4192a**:
- Workflow started: 05:44:37
- Jobs containers provisioning started: 05:44:38
- Workflow marked jobs as "failed": 05:44:39 (2 seconds later!)
- Jobs actually started running: 05:46:03 (86 seconds to provision)
- Jobs completed successfully: 05:46:12

**Actual job execution kb7-snomed-job-production-lhhcn**:
- ResourcesAvailable: 05:40:50
- Started: 05:42:16 (1m26s provisioning time)
- Completed: 05:42:25 (9s execution time)
- Message: "Execution completed successfully in 1m30.69s"

## Attempted Fixes

1. **Enhanced Status Checking**: Added `default()` functions to handle missing fields, improved terminal state detection - FAILED (still exits immediately)
2. **Graceful Provisioning Handling**: Changed exception handling to treat API errors as expected during provisioning - FAILED (still exits immediately)
3. **Diagnostic Logging**: Added detailed logging to identify failure point - revealed that logs aren't appearing, indicating exception before wait function runs

## Recommended Solution

Given the complexity of properly handling Cloud Run Jobs' asynchronous execution model in Cloud Workflows, and the predictable timing (~90s provision + ~30s execute = ~2 minutes total), implement a **simple sleep-based approach**:

1. Call `googleapis.run.v2.projects.locations.jobs.run` to start the jobs
2. Sleep for 180 seconds (3 minutes) to allow for provisioning and execution
3. Check final execution status
4. Proceed to read GCS result files (which jobs write regardless of workflow status)

This approach:
- Avoids complex polling logic that's failing
- Accounts for worst-case provisioning times
- Works reliably since GCS result files are the source of truth
- Is simple and maintainable

## Alternative Approaches Considered

1. **Eventarc-based**: Use Cloud Run Jobs completion events to trigger next workflow step - too complex, requires architecture changes
2. **Cloud Tasks**: Queue tasks to check status - unnecessary complexity
3. **Retry Logic in Wait Function**: Multiple retries with exponential backoff - already attempted, still fails

## Implementation Plan

Update `kb-factory-jobs-workflow-v2.yaml` to:
1. Remove `wait_for_job_completion` subworkflow entirely
2. After calling `jobs.run`, add fixed sleep of 180 seconds
3. Optionally check execution status for logging, but don't fail on errors
4. Rely on GCS result files as source of truth for download status

## Files Involved

- [kb-factory-jobs-workflow-v2.yaml](workflows/kb-factory-jobs-workflow-v2.yaml:409-492) - Wait subworkflow to be simplified
- All three downloader jobs already write GCS result files correctly
- GitHub dispatcher already configured to receive environment variables from workflow

## Success Criteria

- Workflow completes successfully after ~3 minutes
- All three downloader jobs execute and complete
- GCS result files are created with correct content
- GitHub dispatcher receives correct GCS keys via environment variables
- No false "failed" status from premature workflow termination
