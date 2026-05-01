# V5 Bbox Provenance — RunPod Smoke Checklist

End-to-end verification of the V5 Bbox Provenance subsystem on RunPod against
the live KB-0 GCP PostgreSQL instance. This checklist is gated on GCP
credentials being available on the pod; run it only after the credentials are
configured.

## Prerequisites
- RunPod pod running `ghcr.io/appu554/cardiofit-pipeline1:runpod-latest` (or rebuilt with V5 branch)
- GCP credentials available: `GOOGLE_APPLICATION_CREDENTIALS` or ADC
- KB-0 GCP PostgreSQL accessible from pod
- Pod has GPU (4090 or equivalent) for MonkeyOCR L1 extraction

## Step 1: Pull latest V5 branch on pod

```bash
cd /workspace/cardiofit_2026
git fetch origin
git checkout feature/v5-bbox-provenance
git pull --ff-only
```

## Step 2: Enable V5 Bbox Provenance flag

```bash
export V5_BBOX_PROVENANCE=1
```

The flag is read by `signal_merger`, `run_pipeline_targeted.py`, and
`push_to_kb0_gcp.py`. With the flag off, V5 code paths are bypassed and the
pipeline runs in V4 mode.

## Step 3: Run smoke pipeline (acs-hcp-summary, ~5 min on 4090)

```bash
cd /workspace/cardiofit_2026/backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
source /workspace/venv/bin/activate
python3 data/run_pipeline_targeted.py \
  --pipeline 1 \
  --guideline heart_foundation_au_2025 \
  --source acs-hcp-summary \
  --l1 monkeyocr \
  --target-kb all
```

Output written to `data/output/v4/job_monkeyocr_<timestamp>/`.

## Step 4: Verify bbox coverage metric

```bash
python3 data/v5_metrics.py data/output/v4/job_monkeyocr_*/
# Expected output: bbox_coverage_pct >= 99.0%
```

If coverage is below 99% the run is a regression — do not push to KB-0. Open
an issue with the output of `v5_metrics.py --verbose`.

## Step 5: Apply migration and push to KB-0

```bash
python3 data/push_to_kb0_gcp.py data/output/v4/job_monkeyocr_*/
# Migration 009 applied automatically on connect
# Verify: provenance_v5 column populated in l2_merged_spans
```

The pusher idempotently applies migration 009 (`provenance_v5 jsonb`) before
upserting rows. Re-running is safe.

## Step 6: Verify in KB-0 GCP (psql or Cloud Console)

```sql
SELECT COUNT(*) FROM l2_merged_spans WHERE provenance_v5 IS NOT NULL;
-- Expected: > 0
SELECT jsonb_array_length(provenance_v5) FROM l2_merged_spans WHERE provenance_v5 IS NOT NULL LIMIT 5;
-- Expected: >= 1 per row
```

## Step 7: Run V5 unit tests on pod

```bash
cd /workspace/cardiofit_2026/backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
python -m pytest tests/v5/ -q
# Expected: 89 passed (86 V5 core + 3 Task 14 checklist tests)
```

## Success Criteria

- [ ] bbox_coverage_pct >= 99.0% on acs-hcp-summary
- [ ] provenance_v5 IS NOT NULL for > 0 rows in KB-0
- [ ] All 89 V5 unit tests pass on pod
- [ ] No errors in `push_to_kb0_gcp.py` output
- [ ] Migration 009 applied without manual intervention

## Rollback

If any criterion fails:

```bash
unset V5_BBOX_PROVENANCE
# Re-run pipeline in V4 mode and re-push; provenance_v5 column remains
# but is not written. Migration 009 is non-breaking and does not need
# to be reverted.
```
