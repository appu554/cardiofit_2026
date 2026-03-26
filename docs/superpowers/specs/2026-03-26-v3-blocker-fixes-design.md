# V3 Foundation Blocker Fixes — Design Spec

**Date**: 2026-03-26
**Status**: Approved
**Goal**: Unblock end-to-end V3 clinical flow by fixing three integration blockers between KB-22, the API Gateway, and KB-20/KB-23/KB-25/KB-26.

**Context**: V4 North Star implementation (17-task plan at `docs/superpowers/plans/2026-03-26-v4-north-star-implementation.md`) is blocked until the V3 inter-KB communication works end-to-end. These three blockers prevent the core flow: Patient App → Gateway → KB-22 session → KB-20 stratum → HPI questions → KB-23 Decision Card → V-MCU.

---

## Blocker 1: Stratum Naming Mismatch

### Problem

KB-22's session initialization validates the patient's stratum against the HPI node's `strata_supported` list using **exact string equality**.

**Location**: `kb-22-hpi-engine/internal/services/session_service.go:148-160`

```go
for _, supported := range node.StrataSupported {
    if supported == sessionCtx.StratumLabel {
        stratumSupported = true
        break
    }
}
```

KB-20 returns specific strata (`"DM_HTN"`, `"DM_HTN_CKD"`, etc.) from `kb-20-patient-profile/internal/models/stratum.go:44-51`:

```go
const (
    StratumDMHTN      = "DM_HTN"
    StratumDMHTNCKD   = "DM_HTN_CKD"
    StratumDMHTNCKDHF = "DM_HTN_CKD_HF"
    StratumDMOnly     = "DM_ONLY"
    StratumHTNOnly    = "HTN_ONLY"
)
```

But some HPI node YAMLs use `_base` suffix as a catch-all. Example from `kb-22-hpi-engine/nodes/p01_chest_pain.yaml:17-19`:

```yaml
strata_supported:
  - DM_HTN_base
```

**Result**: KB-20 returns `"DM_HTN"`, P01 supports `["DM_HTN_base"]`, equality check fails, session marked **ABANDONED**.

Other nodes like P02 Dyspnea list specific strata and work correctly:

```yaml
strata_supported:
  - DM_ONLY
  - DM_HTN
  - DM_HTN_CKD
  - DM_HTN_CKD_HF
```

### Design Decision

**Hierarchical matching in KB-22** (not renaming in YAML, not adding ancestors to KB-20 response).

Rationale:
- V4 adds new substrata (`DM_HTN_CKD_A3`, `DM_HTN_CKD_HF_REDUCED`). A rename approach would require updating every `_base` node's YAML each time a new substratum is added.
- The hierarchy belongs in KB-22 where the matching happens. KB-20 should remain authoritative for "what is this patient's stratum" without knowledge of KB-22's node definitions.
- Zero changes to KB-20, zero changes to YAML node definitions.

### Solution

**New file**: `kb-22-hpi-engine/internal/services/stratum_hierarchy.go`

Hierarchy map defining parent-child relationships (nested, not flat):

```
DM_HTN_base (catch-all for any cardiometabolic patient)
├── DM_HTN
│   └── DM_HTN_CKD
│       └── DM_HTN_CKD_HF
├── DM_ONLY
└── HTN_ONLY
```

**Design note**: The hierarchy is nested — `DM_HTN_CKD` is a child of `DM_HTN`, which is a child of `DM_HTN_base`. This means:
- A node declaring `DM_HTN_base` accepts ALL strata (catch-all)
- A node declaring `DM_HTN` accepts `DM_HTN`, `DM_HTN_CKD`, and `DM_HTN_CKD_HF` (but not `DM_ONLY` or `HTN_ONLY`)
- A node declaring `DM_HTN_CKD` accepts `DM_HTN_CKD` and `DM_HTN_CKD_HF` (but not `DM_HTN` alone)

Go map representation: `map[string]string` where key = child, value = parent:
```go
var stratumParent = map[string]string{
    "DM_HTN":        "DM_HTN_base",
    "DM_HTN_CKD":    "DM_HTN",
    "DM_HTN_CKD_HF": "DM_HTN_CKD",
    "DM_ONLY":       "DM_HTN_base",
    "HTN_ONLY":      "DM_HTN_base",
}
```

Function signature:

```go
// StratumMatches returns true if the patient's stratum is accepted by
// the node's strata_supported list, accounting for hierarchy.
// A node declaring "DM_HTN_base" accepts any stratum that is a descendant
// of DM_HTN_base (i.e., DM_HTN, DM_HTN_CKD, DM_HTN_CKD_HF, etc.)
func StratumMatches(patientStratum string, nodeStrata []string) bool
```

Implementation:
1. Direct match: if `patientStratum` appears in `nodeStrata`, return true.
2. Ancestor walk: look up `patientStratum` in the hierarchy map, get its parent. Check if parent is in `nodeStrata`. Repeat up the chain (max depth 3).
3. No match: return false.

**One-line change** in `session_service.go`: replace the `for/if` equality loop with `StratumMatches(sessionCtx.StratumLabel, node.StrataSupported)`.

**Test file**: `kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go`

Test cases:
- `StratumMatches("DM_HTN", ["DM_HTN_base"])` → true (grandchild via DM_HTN → DM_HTN_base)
- `StratumMatches("DM_HTN_CKD", ["DM_HTN_base"])` → true (walks DM_HTN_CKD → DM_HTN → DM_HTN_base)
- `StratumMatches("DM_HTN_CKD_HF", ["DM_HTN_base"])` → true (walks 3 levels up)
- `StratumMatches("DM_HTN", ["DM_HTN"])` → true (direct match)
- `StratumMatches("DM_HTN_CKD", ["DM_HTN"])` → true (DM_HTN_CKD is a child of DM_HTN in nested hierarchy)
- `StratumMatches("DM_HTN", ["DM_HTN_CKD"])` → false (parent cannot match child's stratum)
- `StratumMatches("DM_ONLY", ["DM_HTN"])` → false (DM_ONLY is under DM_HTN_base, not DM_HTN)
- `StratumMatches("NONE", ["DM_HTN_base"])` → false (NONE not in hierarchy)
- `StratumMatches("DM_HTN", [])` → false (empty strata list)

### V4 Extensibility

When V4 adds `DM_HTN_CKD_A3`:
1. Add constant in KB-20's `stratum.go`
2. Add one map entry in KB-22's `stratum_hierarchy.go`: `"DM_HTN_CKD_A3": "DM_HTN_CKD"` (child of DM_HTN_CKD, which is child of DM_HTN_base)
3. No YAML changes, no other service changes.

---

## Blocker 2: Gateway Route Translation Errors

### Problem

The API Gateway's Patient App and Doctor Dashboard endpoint handlers forward requests to **paths that don't exist** on the downstream KB services. Additionally, the catch-all proxy's `strip_prefix` logic strips too much from the URL.

**Gateway**: `backend/services/api-gateway/` (Python/FastAPI, port 8000)

### Mismatch Inventory

#### Patient App Handlers (`app/api/endpoints/patient_app.py`)

| Line | Product URL | Gateway forwards to | KB actual endpoint | Issue |
|------|-------------|--------------------|--------------------|-------|
| 56 | `GET /patient/{id}/health-score` | KB-26 `/patients/{id}/health-score` | KB-26 `GET /api/v1/kb26/mri/{id}` | Wrong path + response extract needed |
| 62 | `GET /patient/{id}/actions/today` | KB-23 `/patients/{id}/actions/today` | KB-23 `GET /api/v1/patients/{id}/active-cards` | Wrong path |
| 68 | `GET /patient/{id}/health-drive` | KB-25 `/patients/{id}/health-drive` | KB-25 `POST /api/v1/kb25/recommend-lifestyle` | Wrong path + wrong method |
| 74 | `GET /patient/{id}/progress` | KB-20 `/patients/{id}/progress` | KB-20 `GET /api/v1/patient/{id}/protocols` | Wrong path |
| 80 | `GET /patient/{id}/cause-effect` | KB-26 `/patients/{id}/cause-effect` | No matching KB-26 endpoint | Missing endpoint |
| 86 | `GET /patient/{id}/timeline` | KB-20 `/patients/{id}/timeline` | No matching KB-20 endpoint | Missing endpoint |
| 92 | `GET /patient/{id}/insights` | KB-26 `/patients/{id}/insights` | No matching KB-26 endpoint | Missing endpoint |
| 98 | `POST /patient/{id}/checkin` | KB-22 `/patients/{id}/checkin` | KB-22 `POST /api/v1/sessions` | Wrong path + body transform needed |

#### Doctor Dashboard Handlers (`app/api/endpoints/doctor_dashboard.py`)

| Line | Product URL | Gateway forwards to | KB actual endpoint | Issue |
|------|-------------|--------------------|--------------------|-------|
| 32 | `GET /doctor/patients/{id}/summary` | KB-20 `/patients/{id}/summary` | KB-20 `GET /api/v1/patient/{id}/profile` | Wrong path |
| 38 | `GET /doctor/patients/{id}/mri` | KB-26 `/patients/{id}/mri` | KB-26 `GET /api/v1/kb26/mri/{id}` | Wrong path |
| 44 | `GET /doctor/patients/{id}/cards` | KB-23 `/patients/{id}/cards` | KB-23 `GET /api/v1/patients/{id}/active-cards` | Wrong path |
| 50 | `POST /doctor/cards/{id}/action` | KB-23 `/cards/{id}/action` | KB-23 `POST /api/v1/cards/{id}/mcu-gate-resume` | Wrong path |
| 68 | `GET /doctor/patients/{id}/channel-b-inputs` | KB-20 `/patients/{id}/channel-b-inputs` | KB-20 `GET /api/v1/patient/{id}/channel-b-inputs` | Missing `/api/v1/` prefix + `/patients/` (plural) vs `/patient/` (singular) |
| 74 | `GET /doctor/patients/{id}/channel-c-inputs` | KB-20 `/patients/{id}/channel-c-inputs` | KB-20 `GET /api/v1/patient/{id}/channel-c-inputs` | Missing `/api/v1/` prefix + `/patients/` (plural) vs `/patient/` (singular) |

#### Catch-all Proxy (`app/api/proxy.py`)

The `SERVICE_ROUTES` for Vaidshala services use `strip_prefix=True` with prefix `/api/v1/kb22`. This strips `/api/v1/kb22` from `/api/v1/kb22/sessions` → sends `/sessions` to KB-22. But KB-22's route is `/api/v1/sessions`. **Missing `/api/v1/` prefix after stripping.**

### Solution

#### 2a. Fix Specific Handler Paths

Update all `_forward()` calls to use correct internal KB service paths:

**Patient App corrections:**

```python
# health-score → KB-26 MRI
_forward(request, KB26_URL, f"/api/v1/kb26/mri/{patient_id}")

# actions/today → KB-23 active cards
_forward(request, KB23_URL, f"/api/v1/patients/{patient_id}/active-cards")

# health-drive → KB-25 recommend-lifestyle (POST with patient context)
_forward(request, KB25_URL, f"/api/v1/kb25/recommend-lifestyle", method="POST")

# progress → KB-20 protocols
_forward(request, KB20_URL, f"/api/v1/patient/{patient_id}/protocols")

# checkin → KB-22 sessions (body transform required)
_forward(request, KB22_URL, "/api/v1/sessions", method="POST",
         body_transform=checkin_to_session(patient_id, body))
```

**Doctor Dashboard corrections:**

```python
# summary → KB-20 profile
_forward(request, KB20_URL, f"/api/v1/patient/{patient_id}/profile")

# mri → KB-26 MRI
_forward(request, KB26_URL, f"/api/v1/kb26/mri/{patient_id}")

# cards → KB-23 active-cards
_forward(request, KB23_URL, f"/api/v1/patients/{patient_id}/active-cards")

# card action → KB-23 mcu-gate-resume
_forward(request, KB23_URL, f"/api/v1/cards/{card_id}/mcu-gate-resume", method="POST")

# channel-b/c → KB-20 (add /api/v1/ prefix)
_forward(request, KB20_URL, f"/api/v1/patient/{patient_id}/channel-b-inputs")
_forward(request, KB20_URL, f"/api/v1/patient/{patient_id}/channel-c-inputs")
```

**Not-yet-implemented endpoints** (`timeline`, `cause-effect`, `insights`): Return 501 with `{"error": "endpoint not yet implemented", "planned_service": "KB-20|KB-26"}`.

#### 2b. Body Transformers

**New file**: `app/api/transforms.py`

Two transformers:

```python
def checkin_to_session(patient_id: str, body: dict) -> dict:
    """Transform Patient App checkin → KB-22 CreateSessionRequest."""
    symptom = body.get("symptom", "")
    node_map = {
        "chest_pain": "P01_CHEST_PAIN",
        "breathlessness": "P02_DYSPNEA",
        "palpitations": "P03_PALPITATIONS",
        # extend as nodes are added
    }
    return {
        "patient_id": patient_id,
        "node_id": node_map.get(symptom, f"P01_CHEST_PAIN"),  # default to P01
    }


def extract_health_score(kb26_mri_response: dict) -> dict:
    """Extract simplified health score from KB-26 MRI response."""
    data = kb26_mri_response.get("data", kb26_mri_response)
    return {
        "score": data.get("mri_score", data.get("composite_score")),
        "trend": data.get("trend"),
        "components": data.get("decomposition", {}),
    }
```

#### 2c. Fix Catch-all Proxy

KB services have **inconsistent route group prefixes**:
- KB-20, KB-22, KB-23: routes under `/api/v1` (no service name in group)
- KB-25: routes under `/api/v1/kb25` (service name in group)
- KB-26: routes under `/api/v1/kb26` (service name in group)

The catch-all proxy strips the gateway prefix (e.g., `/api/v1/kb22`) and must prepend the correct **internal prefix** for each service. A blanket "prepend `/api/v1/`" would break KB-25 and KB-26.

**Solution**: Add an `internal_prefix` field to `SERVICE_ROUTES` for each Vaidshala service:

```python
SERVICE_ROUTES = {
    # ...existing legacy routes unchanged...
    "kb20_patient_profile": {
        "prefix": "/api/v1/kb20", "target": "http://localhost:8131",
        "strip_prefix": True, "internal_prefix": "/api/v1",
    },
    "kb22_hpi_engine": {
        "prefix": "/api/v1/kb22", "target": "http://localhost:8132",
        "strip_prefix": True, "internal_prefix": "/api/v1",
    },
    "kb23_decision_cards": {
        "prefix": "/api/v1/kb23", "target": "http://localhost:8134",
        "strip_prefix": True, "internal_prefix": "/api/v1",
    },
    "kb25_lifestyle_graph": {
        "prefix": "/api/v1/kb25", "target": "http://localhost:8136",
        "strip_prefix": True, "internal_prefix": "/api/v1/kb25",
    },
    "kb26_metabolic_twin": {
        "prefix": "/api/v1/kb26", "target": "http://localhost:8137",
        "strip_prefix": True, "internal_prefix": "/api/v1/kb26",
    },
}
```

Proxy logic:
```python
# Before: path = full_path.removeprefix(route["prefix"])
# After:
remainder = full_path.removeprefix(route["prefix"])
internal_prefix = route.get("internal_prefix", "")
path = internal_prefix + remainder
```

Examples:
- `/api/v1/kb22/sessions` → strip `/api/v1/kb22` → `/sessions` → prepend `/api/v1` → `/api/v1/sessions` ✓
- `/api/v1/kb25/causal-chain/sbp` → strip `/api/v1/kb25` → `/causal-chain/sbp` → prepend `/api/v1/kb25` → `/api/v1/kb25/causal-chain/sbp` ✓
- `/api/v1/kb26/mri/123` → strip `/api/v1/kb26` → `/mri/123` → prepend `/api/v1/kb26` → `/api/v1/kb26/mri/123` ✓

Legacy services (`strip_prefix=False`) are unchanged — they don't use `internal_prefix`.

#### 2d. Update _forward() Signature

**File**: `app/api/endpoints/patient_app.py` (line 119) — this is where `_forward()` is defined. The catch-all proxy in `proxy.py` has its own separate `forward_request()` function.

Add optional `body_transform` parameter to `_forward()`:

```python
async def _forward(
    request: Request,
    service_url: str,
    path: str,
    method: str = None,
    body_transform: Callable = None,
    response_transform: Callable = None,
) -> Any:
```

When `body_transform` is provided, parse the request body as JSON, pass through the transformer, and send the transformed body. When `response_transform` is provided, parse the downstream response and return the transformed version.

---

## Blocker 3: Patient ID Dual-Format Resolution

### Problem

The API Gateway extracts `patient_id` from URL path parameters. Client apps (Flutter Patient App, React Doctor Dashboard) use **FHIR UUIDs** (e.g., `550e8400-e29b-41d4-a716-446655440000`).

The gateway forwards this UUID to all KB services. Only KB-20 has a `resolveFHIRPatientID()` middleware that converts FHIR UUID → ABHA ID. Other services (KB-22, KB-23, KB-25, KB-26) store and query data by **ABHA ID** (because they received ABHA IDs from inter-service calls that went through KB-20).

**Result**: `GET /patients/{fhir_uuid}/active-cards` → KB-23 queries by FHIR UUID → finds nothing (cards stored with ABHA ID).

**KB-20's existing middleware** (`kb-20-patient-profile/internal/api/routes.go:13-48`):

```go
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (s *Server) resolveFHIRPatientID() gin.HandlerFunc {
    // If :id is a UUID, resolve to ABHA patient_id via DB lookup
}
```

### Solution

**New file**: `app/api/patient_resolver.py`

Gateway-level patient ID resolution with caching:

```python
import re
from typing import Optional

UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.IGNORECASE
)

async def resolve_patient_id(patient_id: str) -> str:
    """Resolve FHIR UUID to ABHA patient ID via KB-20.

    If patient_id is not a UUID (already ABHA format), returns it unchanged.
    Results are cached in Redis with 1-hour TTL.
    """
```

Implementation:
1. **Format detection**: Check if `patient_id` matches UUID regex. If not, return as-is (already ABHA).
2. **Cache check**: Look up `patient_resolve:{uuid}` in Redis.
3. **KB-20 call**: `GET {KB20_URL}/api/v1/patient/{uuid}/profile` — KB-20's middleware resolves the UUID and returns the profile with `patient_id` (ABHA).
4. **Cache set**: Store `uuid → abha_id` mapping with 1-hour TTL.
5. **Return**: ABHA ID for forwarding.

**Integration**: Called in each Patient App and Doctor Dashboard handler before building the target path:

```python
@router.get("/patient/{patient_id}/health-score")
async def patient_health_score(patient_id: str, request: Request):
    resolved_id = await resolve_patient_id(patient_id)
    return await _forward(request, settings.KB26_SERVICE_URL,
                          f"/api/v1/kb26/mri/{resolved_id}")
```

**Failure mode**: If KB-20 is unreachable, return HTTP 502. Patient ID resolution is a required step — no fallback to raw UUID.

**Redis graceful degradation**: The gateway currently treats Redis as optional (see `_get_cache()` in `patient_app.py:21-26`). The resolver follows the same pattern — if Redis is unavailable, skip the cache layer and call KB-20 on every request. This adds ~10ms per request but maintains functionality. The resolver must NOT fail if Redis is down.

**Cache key format**: `patient_resolve:{uuid}` → `{abha_id}` (Redis string, TTL 3600s)

---

## Execution Order

| Step | Blocker | Action | Verification |
|------|---------|--------|-------------|
| 1 | B1 | Add `stratum_hierarchy.go` + test to KB-22 | `go test ./internal/services/ -run TestStratumMatches` |
| 2 | B1 | Update `session_service.go` to use `StratumMatches()` | `POST /api/v1/sessions` with P01 node succeeds for DM_HTN patient |
| 3 | B3 | Add `patient_resolver.py` to gateway | Unit test: UUID → ABHA resolution |
| 4 | B2 | Fix Patient App handler paths + add transforms | `POST /api/v1/patient/{uuid}/checkin` → KB-22 session created |
| 5 | B2 | Fix Doctor Dashboard handler paths | `GET /api/v1/doctor/patients/{uuid}/cards` → returns active cards |
| 6 | B2 | Fix catch-all proxy prefix stripping | `GET /api/v1/kb22/nodes` → returns node list |
| 7 | All | E2E flow: checkin → HPI session → answers → Decision Card | Full chain from gateway to KB-23 output |

---

## Files Changed

### New Files
- `kb-22-hpi-engine/internal/services/stratum_hierarchy.go` — hierarchy map + `StratumMatches()`
- `kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go` — hierarchy matching tests
- `backend/services/api-gateway/app/api/transforms.py` — body/response transformers
- `backend/services/api-gateway/app/api/patient_resolver.py` — FHIR UUID → ABHA resolver
- `backend/services/api-gateway/tests/test_transforms.py` — transformer unit tests
- `backend/services/api-gateway/tests/test_patient_resolver.py` — resolver unit tests

### Modified Files
- `kb-22-hpi-engine/internal/services/session_service.go` — 1 line: replace equality check with `StratumMatches()`
- `backend/services/api-gateway/app/api/endpoints/patient_app.py` — fix all `_forward()` target paths + add body/response transforms
- `backend/services/api-gateway/app/api/endpoints/doctor_dashboard.py` — fix all `_forward()` target paths
- `backend/services/api-gateway/app/api/proxy.py` — fix catch-all prefix stripping + add transform support to `_forward()`

### Not Changed
- KB-20 (no changes needed — stratum engine and FHIR middleware already correct)
- KB-23, KB-25, KB-26 (no changes — gateway resolves IDs before forwarding)
- HPI node YAML files (hierarchy matching handles `_base` suffix)

---

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| KB-20 latency for patient ID resolution adds ~10ms per gateway request | Redis cache with 1-hour TTL; resolution only needed on first request per patient per hour |
| `checkin_to_session` node_map incomplete | Default to P01_CHEST_PAIN; log unmapped symptoms for iteration |
| Catch-all proxy prefix fix may break legacy services | Fix scoped to `strip_prefix=True` routes only (Vaidshala KB services); legacy services use `strip_prefix=False` |
| Stratum hierarchy map must be kept in sync with KB-20 constants | Both are Go constants; add a compile-time test that validates hierarchy map covers all KB-20 stratum constants |

---

## Success Criteria

1. `POST /api/v1/patient/{fhir_uuid}/checkin {"symptom": "chest_pain"}` through the gateway at port 8000 → KB-22 creates a session → returns first HPI question
2. P01 Chest Pain node (which declares `DM_HTN_base`) accepts patients with stratum `DM_HTN`, `DM_HTN_CKD`, or `DM_HTN_CKD_HF`
3. `GET /api/v1/doctor/patients/{fhir_uuid}/cards` → returns active decision cards for the patient
4. All gateway routes for Patient App and Doctor Dashboard return actual KB service data (not 404s)
5. Catch-all proxy correctly routes `GET /api/v1/kb22/nodes` → KB-22 node list
