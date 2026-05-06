# Baseline drift investigation

**Audience:** clinical informatics partner, on-call backend engineer.
**Source contract:** Layer 2 doc §2.2 (running-baseline recompute) +
Part 6 Failure 4 (baseline contamination).

## Symptom: a resident's baseline looks wrong

Common shapes:

- A vital baseline shifts dramatically overnight without a clinical
  event.
- A delta-flag fires on every reading (the baseline is too narrow).
- A delta-flag never fires even on obviously abnormal readings (the
  baseline is too wide).
- The clinician asks "why is Mrs M's BP baseline 145?"

## Investigation steps

### 1. Snapshot the affected baseline

```sql
SELECT * FROM baselines
WHERE resident_ref = $resident_id
  AND observation_type = $vital_type_key
ORDER BY computed_at DESC
LIMIT 5;
```

Capture: `value`, `confidence_tier`, `n_obs_used`, `window_start`,
`window_end`, `computed_at`.

### 2. Inspect the BaselineConfig that drove the recompute

```sql
SELECT * FROM baseline_configs WHERE observation_type = $vital_type_key;
```

Check: `window_days`, `min_obs_for_high_confidence`,
`exclude_during_active_concerns`, `morning_only`. If
`exclude_during_active_concerns` is empty for a vital that should drop
acute periods (e.g. systolic BP excluding infection_acute), this is
the Failure 4 contamination shape.

### 3. List the observations that contributed

```sql
SELECT recorded_at, value, source
FROM observations
WHERE resident_ref = $resident_id
  AND vital_type_key = $vital_type_key
  AND recorded_at >= $window_start
  AND recorded_at <  $window_end
ORDER BY recorded_at ASC;
```

Cross-reference against any open active_concerns for that resident in
the same window:

```sql
SELECT kind, opened_at, closed_at
FROM active_concerns
WHERE resident_ref = $resident_id
  AND opened_at < $window_end
  AND (closed_at IS NULL OR closed_at > $window_start);
```

If an active concern listed in `exclude_during_active_concerns`
overlaps the window AND observations from inside that overlap appear
in the contributing list, the exclusion isn't being applied — escalate
as a kb-20 BaselineStore.RecomputeAndUpsertTx bug.

### 4. Check for source contamination

A common source of drift: a misclassified observation (e.g. a finger
prick BG recorded as a fasting BG). Spot it via `source` field on
the observations rows. If multiple observations from the same source
look out of pattern, request a source-system sync from that vendor.

### 5. Force a recompute

If the BaselineConfig changed but the existing baseline didn't refresh:

```sql
SELECT recompute_baseline_for($resident_id, $vital_type_key);
```

(See `kb-20-patient-profile/internal/storage/baseline_store.go` —
`RecomputeAndUpsertTx`.)

Or hit the kb-20 admin endpoint (V1):

```
POST /v2/admin/baselines/recompute
{"resident_ref": "...", "observation_type": "..."}
```

## Common drift patterns

### Pattern: Hospital admission inflated BP baseline

**Cause:** post-fall admission produced a 7-day stretch of high BP
readings; the baseline window includes them; `exclude_during_active_concerns`
doesn't list `hospital_admission`.

**Resolution:** Either (a) close the relevant active_concern with the
correct kind so it ends up on the exclusion list, or (b) add
`hospital_admission_recent` to the systolic-BP BaselineConfig's
exclusion list. Document the change in the BaselineConfig.notes field.

### Pattern: Long-tail of post-discharge readings narrows baseline

**Cause:** post-discharge readings are clustered tightly because the
clinical team is monitoring intensively; the baseline standard
deviation collapses; every subsequent reading flags as a delta.

**Resolution:** confirm the cluster is real and clinically appropriate.
If yes, the narrow baseline is doing its job — calibrate clinician
expectations rather than tweaking the baseline.

### Pattern: Frequency drift after device change

**Cause:** A device upgrade changes the sampling frequency (e.g. CGM
went from 5-minute to 1-minute sampling). The baseline N-counts shift.

**Resolution:** review the BaselineConfig's `min_obs_for_high_confidence`
in light of the new frequency; confirm it remains appropriate.

## Escalation

- Suspected BaselineStore bug → file a critical kb-20 ticket;
  reproduce against the integration test pack.
- Disagreement on whether to exclude a concern type → escalate to
  the clinical informatics lead with the case detail.
- Repeated source contamination from one vendor → file a vendor
  ticket; document in `claudedocs/vendor-issues/`.

## Audit trail

A baseline recompute writes one `evidence_trace_nodes` row
(state_machine=`Monitoring`,
state_change_type=`baseline_recomputed`). The node's `reasoning_summary`
captures the input observation IDs, the BaselineConfig snapshot, the
n-obs and confidence tier outcomes. This is the regulator-audit
artefact for "why does the baseline say what it says?".

## See also

- Layer 2 doc §2.2.
- Layer 2 doc Part 6 Failure 4.
- [evidencetrace-audit-query.md](evidencetrace-audit-query.md)
