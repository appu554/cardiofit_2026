# V3 Foundation Blocker Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unblock end-to-end V3 clinical flow by fixing stratum naming mismatch (B1), patient ID dual-format (B3), and gateway route translation errors (B2).

**Architecture:** KB-22 gets a hierarchical stratum matching function (Go). The API Gateway (Python/FastAPI) gets a patient ID resolver backed by KB-20 + Redis cache, body/response transformers, corrected handler paths, and a per-service `internal_prefix` for the catch-all proxy.

**Tech Stack:** Go 1.21+ (KB-22), Python 3.11+ / FastAPI / httpx (Gateway), Redis (caching)

**Spec:** `docs/superpowers/specs/2026-03-26-v3-blocker-fixes-design.md`

---

## Execution Order

B1 (stratum) → B3 (patient resolver) → B2 (gateway routes) → E2E verification

This order ensures each blocker's fix is available before the next task that depends on it.

---

## File Map

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-22-hpi-engine/internal/services/stratum_hierarchy.go` | Hierarchy map + `StratumMatches()` |
| Create | `kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go` | Hierarchy matching unit tests |
| Modify | `kb-22-hpi-engine/internal/services/session_service.go:148-155` | Replace equality loop with `StratumMatches()` |
| Create | `backend/services/api-gateway/app/api/patient_resolver.py` | FHIR UUID → ABHA resolver with Redis cache |
| Create | `backend/services/api-gateway/tests/test_patient_resolver.py` | Resolver unit tests |
| Create | `backend/services/api-gateway/app/api/transforms.py` | Body/response transformers |
| Create | `backend/services/api-gateway/tests/test_transforms.py` | Transformer unit tests |
| Modify | `backend/services/api-gateway/app/api/endpoints/patient_app.py:53-98,119-165` | Fix handler paths + upgrade `_forward()` |
| Modify | `backend/services/api-gateway/app/api/endpoints/doctor_dashboard.py:29-74` | Fix handler paths |
| Modify | `backend/services/api-gateway/app/api/proxy.py:119-148,151-178` | Add `internal_prefix` to SERVICE_ROUTES + fix `forward_request()` |

---

### Task 1: Stratum Hierarchy — Write Failing Tests

**Files:**
- Create: `kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go`

- [ ] **Step 1: Write the test file**

```go
package services

import "testing"

func TestStratumMatches(t *testing.T) {
	tests := []struct {
		name           string
		patientStratum string
		nodeStrata     []string
		want           bool
	}{
		// Direct match
		{"direct match", "DM_HTN", []string{"DM_HTN"}, true},
		// Ancestor walk: DM_HTN → parent DM_HTN_base
		{"child matches base", "DM_HTN", []string{"DM_HTN_base"}, true},
		// 2-level walk: DM_HTN_CKD → DM_HTN → DM_HTN_base
		{"grandchild matches base", "DM_HTN_CKD", []string{"DM_HTN_base"}, true},
		// 3-level walk: DM_HTN_CKD_HF → DM_HTN_CKD → DM_HTN → DM_HTN_base
		{"great-grandchild matches base", "DM_HTN_CKD_HF", []string{"DM_HTN_base"}, true},
		// Nested: DM_HTN_CKD is child of DM_HTN
		{"child matches parent", "DM_HTN_CKD", []string{"DM_HTN"}, true},
		// DM_HTN_CKD_HF walks to DM_HTN_CKD
		{"grandchild matches mid-level", "DM_HTN_CKD_HF", []string{"DM_HTN_CKD"}, true},
		// Parent cannot match child
		{"parent does not match child", "DM_HTN", []string{"DM_HTN_CKD"}, false},
		// DM_ONLY → DM_HTN_base (sibling, not under DM_HTN)
		{"DM_ONLY matches base", "DM_ONLY", []string{"DM_HTN_base"}, true},
		{"DM_ONLY does not match DM_HTN", "DM_ONLY", []string{"DM_HTN"}, false},
		// HTN_ONLY → DM_HTN_base
		{"HTN_ONLY matches base", "HTN_ONLY", []string{"DM_HTN_base"}, true},
		{"HTN_ONLY does not match DM_HTN", "HTN_ONLY", []string{"DM_HTN"}, false},
		// Unknown stratum
		{"unknown stratum", "NONE", []string{"DM_HTN_base"}, false},
		// Empty strata list
		{"empty strata list", "DM_HTN", []string{}, false},
		// Multiple strata in list — match any
		{"multi-strata direct", "DM_ONLY", []string{"DM_HTN", "DM_ONLY"}, true},
		{"multi-strata ancestor", "DM_HTN_CKD_HF", []string{"DM_ONLY", "DM_HTN"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StratumMatches(tt.patientStratum, tt.nodeStrata)
			if got != tt.want {
				t.Errorf("StratumMatches(%q, %v) = %v, want %v",
					tt.patientStratum, tt.nodeStrata, got, tt.want)
			}
		})
	}
}

// TestHierarchyCoversAllKnownStrata validates that the stratumParent map
// covers every stratum constant known in the system. If KB-20 adds a new
// stratum, this test will fail until the hierarchy is updated.
// Mirrors KB-20 constants from kb-20-patient-profile/internal/models/stratum.go.
func TestHierarchyCoversAllKnownStrata(t *testing.T) {
	// These must match KB-20's exported stratum constants.
	// Update this list when KB-20 adds new strata.
	kb20Strata := []string{
		"DM_HTN",
		"DM_HTN_CKD",
		"DM_HTN_CKD_HF",
		"DM_ONLY",
		"HTN_ONLY",
	}
	for _, s := range kb20Strata {
		if _, ok := stratumParent[s]; !ok {
			t.Errorf("KB-20 stratum %q is missing from stratumParent hierarchy map — add it to stratum_hierarchy.go", s)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./internal/services/ -run TestStratumMatches -v`
Expected: FAIL — `StratumMatches` undefined

---

### Task 2: Stratum Hierarchy — Implement

**Files:**
- Create: `kb-22-hpi-engine/internal/services/stratum_hierarchy.go`

- [ ] **Step 3: Write the hierarchy implementation**

```go
package services

// stratumParent maps each stratum to its direct parent in the hierarchy.
//
// Hierarchy (nested):
//
//	DM_HTN_base
//	├── DM_HTN
//	│   └── DM_HTN_CKD
//	│       └── DM_HTN_CKD_HF
//	├── DM_ONLY
//	└── HTN_ONLY
var stratumParent = map[string]string{
	"DM_HTN":        "DM_HTN_base",
	"DM_HTN_CKD":    "DM_HTN",
	"DM_HTN_CKD_HF": "DM_HTN_CKD",
	"DM_ONLY":       "DM_HTN_base",
	"HTN_ONLY":      "DM_HTN_base",
}

const maxAncestorDepth = 3

// StratumMatches returns true if patientStratum is accepted by any stratum
// in nodeStrata, accounting for the hierarchy. A node declaring "DM_HTN_base"
// accepts any descendant (DM_HTN, DM_HTN_CKD, DM_HTN_CKD_HF, DM_ONLY, HTN_ONLY).
// A node declaring "DM_HTN" accepts DM_HTN, DM_HTN_CKD, and DM_HTN_CKD_HF
// but NOT DM_ONLY or HTN_ONLY.
func StratumMatches(patientStratum string, nodeStrata []string) bool {
	for _, supported := range nodeStrata {
		if supported == patientStratum {
			return true
		}
	}
	// Walk up the ancestor chain
	current := patientStratum
	for depth := 0; depth < maxAncestorDepth; depth++ {
		parent, ok := stratumParent[current]
		if !ok {
			return false
		}
		for _, supported := range nodeStrata {
			if supported == parent {
				return true
			}
		}
		current = parent
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./internal/services/ -run TestStratumMatches -v`
Expected: All 17 tests PASS (16 matching + 1 hierarchy coverage)

- [ ] **Step 5: Commit B1 hierarchy**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/stratum_hierarchy.go backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go
git commit -m "feat(kb-22): add hierarchical stratum matching for _base catch-all nodes"
```

---

### Task 3: Integrate StratumMatches into Session Service

**Files:**
- Modify: `kb-22-hpi-engine/internal/services/session_service.go:148-155`

- [ ] **Step 6: Replace the equality loop with StratumMatches**

In `session_service.go`, replace lines 148-155:

Old:
```go
	// Validate stratum is supported by the node
	stratumSupported := false
	for _, supported := range node.StrataSupported {
		if supported == sessionCtx.StratumLabel {
			stratumSupported = true
			break
		}
	}
```

New:
```go
	// Validate stratum is supported by the node (hierarchical matching)
	stratumSupported := StratumMatches(sessionCtx.StratumLabel, node.StrataSupported)
```

- [ ] **Step 7: Build KB-22 to verify compilation**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go build ./...`
Expected: Build succeeds (no errors)

- [ ] **Step 8: Commit B1 integration**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/session_service.go
git commit -m "fix(kb-22): use hierarchical StratumMatches in session init — fixes DM_HTN vs DM_HTN_base mismatch"
```

---

### Task 4: Patient ID Resolver — Write Failing Tests

**Files:**
- Create: `backend/services/api-gateway/tests/test_patient_resolver.py`

- [ ] **Step 9: Write resolver tests**

```python
"""Tests for FHIR UUID → ABHA patient ID resolution."""
import pytest
from unittest.mock import AsyncMock, patch, MagicMock

from app.api.patient_resolver import resolve_patient_id, UUID_PATTERN


def test_uuid_pattern_matches_fhir_uuid():
    assert UUID_PATTERN.match("550e8400-e29b-41d4-a716-446655440000")


def test_uuid_pattern_rejects_abha_id():
    assert not UUID_PATTERN.match("91-1001-2001-3001")


def test_uuid_pattern_rejects_plain_string():
    assert not UUID_PATTERN.match("patient-abc-123")


@pytest.mark.asyncio
async def test_non_uuid_returns_unchanged():
    """ABHA IDs pass through without any KB-20 call."""
    result = await resolve_patient_id("91-1001-2001-3001")
    assert result == "91-1001-2001-3001"


@pytest.mark.asyncio
@patch("app.api.patient_resolver._call_kb20_resolve")
async def test_uuid_calls_kb20(mock_kb20):
    """FHIR UUID triggers KB-20 resolution."""
    mock_kb20.return_value = "91-1001-2001-3001"
    result = await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
    assert result == "91-1001-2001-3001"
    mock_kb20.assert_called_once_with("550e8400-e29b-41d4-a716-446655440000")


@pytest.mark.asyncio
@patch("app.api.patient_resolver._call_kb20_resolve")
async def test_uuid_cached_on_second_call(mock_kb20):
    """Second call for same UUID uses cache, not KB-20."""
    mock_kb20.return_value = "91-1001-2001-3001"

    # Provide a fake in-memory cache
    cache = {}

    with patch("app.api.patient_resolver._cache_get", new_callable=AsyncMock) as mock_get, \
         patch("app.api.patient_resolver._cache_set", new_callable=AsyncMock) as mock_set:
        mock_get.return_value = None  # cache miss
        result1 = await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
        assert result1 == "91-1001-2001-3001"
        mock_set.assert_called_once()

        # Second call — cache hit
        mock_get.return_value = "91-1001-2001-3001"
        result2 = await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
        assert result2 == "91-1001-2001-3001"
        # KB-20 should NOT be called again
        assert mock_kb20.call_count == 1


@pytest.mark.asyncio
@patch("app.api.patient_resolver._call_kb20_resolve")
async def test_kb20_failure_raises_502(mock_kb20):
    """If KB-20 is unreachable, raise HTTP 502."""
    from fastapi import HTTPException
    mock_kb20.side_effect = HTTPException(status_code=502, detail="KB-20 service unavailable")
    with pytest.raises(HTTPException) as exc_info:
        await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
    assert exc_info.value.status_code == 502
```

- [ ] **Step 10: Run tests to verify they fail**

Run: `cd backend/services/api-gateway && python -m pytest tests/test_patient_resolver.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'app.api.patient_resolver'`

---

### Task 5: Patient ID Resolver — Implement

**Files:**
- Create: `backend/services/api-gateway/app/api/patient_resolver.py`

- [ ] **Step 11: Write the resolver module**

```python
"""FHIR UUID → ABHA patient ID resolution via KB-20.

The gateway receives FHIR UUIDs from client apps (Flutter, React). Only KB-20
has the DB mapping; other KB services store data by ABHA ID. This module
resolves UUIDs before forwarding to non-KB-20 services.
"""
import logging
import re
from typing import Optional

import httpx
from fastapi import HTTPException

from app.config import settings

logger = logging.getLogger(__name__)

UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
    re.IGNORECASE,
)

_CACHE_TTL = 3600  # 1 hour
_CACHE_PREFIX = "patient_resolve:"


async def resolve_patient_id(patient_id: str) -> str:
    """Resolve a FHIR UUID to its ABHA patient ID.

    If patient_id is not a UUID (already ABHA format), returns it unchanged.
    Results are cached in Redis with 1-hour TTL.
    Raises HTTPException(502) if KB-20 is unreachable.
    """
    if not UUID_PATTERN.match(patient_id):
        return patient_id

    # Try cache first
    cached = await _cache_get(patient_id)
    if cached is not None:
        return cached

    # Call KB-20
    abha_id = await _call_kb20_resolve(patient_id)

    # Cache the result
    await _cache_set(patient_id, abha_id)
    return abha_id


async def _call_kb20_resolve(fhir_uuid: str) -> str:
    """Call KB-20 profile endpoint to resolve FHIR UUID → ABHA ID."""
    url = f"{settings.KB20_SERVICE_URL}/api/v1/patient/{fhir_uuid}/profile"
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.get(url)
            if resp.status_code == 404:
                raise HTTPException(status_code=404, detail=f"Patient not found for FHIR ID {fhir_uuid}")
            resp.raise_for_status()
            data = resp.json()
            # KB-20 returns profile with patient_id field (ABHA format)
            abha_id = data.get("patient_id") or data.get("data", {}).get("patient_id")
            if not abha_id:
                raise HTTPException(status_code=502, detail="KB-20 response missing patient_id")
            return abha_id
    except httpx.ConnectError:
        logger.error("KB-20 unreachable for patient ID resolution: %s", fhir_uuid)
        raise HTTPException(status_code=502, detail="KB-20 service unavailable for patient ID resolution")
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Patient ID resolution failed: %s", e)
        raise HTTPException(status_code=502, detail="Patient ID resolution error")


async def _cache_get(fhir_uuid: str) -> Optional[str]:
    """Look up cached UUID → ABHA mapping. Returns None on miss or Redis unavailable."""
    try:
        redis = _get_redis()
        if redis is None:
            return None
        val = await redis.get(f"{_CACHE_PREFIX}{fhir_uuid}")
        if val:
            return val  # decode_responses=True ensures string return
    except Exception:
        logger.debug("Redis cache get failed (non-fatal), skipping cache")
    return None


async def _cache_set(fhir_uuid: str, abha_id: str) -> None:
    """Store UUID → ABHA mapping in Redis. Fails silently if Redis unavailable."""
    try:
        redis = _get_redis()
        if redis is None:
            return
        await redis.set(f"{_CACHE_PREFIX}{fhir_uuid}", abha_id, ex=_CACHE_TTL)
    except Exception:
        logger.debug("Redis cache set failed (non-fatal), skipping cache")


_redis_client = None


def _get_redis():
    """Get async Redis client singleton, or None if Redis is not configured/available.

    Follows gateway's existing optional-Redis pattern (see patient_app.py:21-26).
    Uses a module-level singleton to avoid creating a new connection per call.
    """
    global _redis_client
    if _redis_client is not None:
        return _redis_client
    if not settings.REDIS_URL:
        return None
    try:
        import redis.asyncio as aioredis
        _redis_client = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
        return _redis_client
    except Exception:
        return None
```

- [ ] **Step 12: Run resolver tests**

Run: `cd backend/services/api-gateway && python -m pytest tests/test_patient_resolver.py -v`
Expected: All 7 tests PASS

- [ ] **Step 13: Commit B3 patient resolver**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/services/api-gateway/app/api/patient_resolver.py backend/services/api-gateway/tests/test_patient_resolver.py
git commit -m "feat(gateway): add FHIR UUID → ABHA patient ID resolver with Redis cache"
```

---

### Task 6: Body/Response Transformers — Write Failing Tests

**Files:**
- Create: `backend/services/api-gateway/tests/test_transforms.py`

- [ ] **Step 14: Write transformer tests**

```python
"""Tests for gateway body/response transformers."""
import pytest
from app.api.transforms import checkin_to_session, extract_health_score


class TestCheckinToSession:
    def test_known_symptom(self):
        result = checkin_to_session("91-1001", {"symptom": "chest_pain"})
        assert result == {
            "patient_id": "91-1001",
            "node_id": "P01_CHEST_PAIN",
        }

    def test_breathlessness(self):
        result = checkin_to_session("91-1001", {"symptom": "breathlessness"})
        assert result == {
            "patient_id": "91-1001",
            "node_id": "P02_DYSPNEA",
        }

    def test_palpitations(self):
        result = checkin_to_session("91-1001", {"symptom": "palpitations"})
        assert result == {
            "patient_id": "91-1001",
            "node_id": "P03_PALPITATIONS",
        }

    def test_unknown_symptom_defaults_to_p01(self):
        result = checkin_to_session("91-1001", {"symptom": "headache"})
        assert result["node_id"] == "P01_CHEST_PAIN"

    def test_missing_symptom_defaults_to_p01(self):
        result = checkin_to_session("91-1001", {})
        assert result["node_id"] == "P01_CHEST_PAIN"


class TestExtractHealthScore:
    def test_normal_response(self):
        kb26_resp = {
            "data": {
                "mri_score": 72.5,
                "trend": "improving",
                "decomposition": {"sbp": 0.3, "hba1c": 0.4},
            }
        }
        result = extract_health_score(kb26_resp)
        assert result["score"] == 72.5
        assert result["trend"] == "improving"
        assert result["components"]["sbp"] == 0.3

    def test_flat_response(self):
        """KB-26 may return data at top level."""
        kb26_resp = {
            "composite_score": 65.0,
            "trend": "stable",
            "decomposition": {},
        }
        result = extract_health_score(kb26_resp)
        assert result["score"] == 65.0

    def test_missing_fields(self):
        result = extract_health_score({})
        assert result["score"] is None
        assert result["trend"] is None
```

- [ ] **Step 15: Run tests to verify they fail**

Run: `cd backend/services/api-gateway && python -m pytest tests/test_transforms.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'app.api.transforms'`

---

### Task 7: Body/Response Transformers — Implement

**Files:**
- Create: `backend/services/api-gateway/app/api/transforms.py`

- [ ] **Step 16: Write the transforms module**

```python
"""Body and response transformers for gateway endpoint translation.

These bridge the gap between product-level URLs (Patient App, Doctor Dashboard)
and internal KB service contracts.
"""


def checkin_to_session(patient_id: str, body: dict) -> dict:
    """Transform Patient App checkin → KB-22 CreateSessionRequest.

    Maps symptom keywords to HPI node IDs. Defaults to P01 (Chest Pain)
    for unmapped symptoms so the flow is never blocked.
    """
    symptom = body.get("symptom", "")
    node_map = {
        "chest_pain": "P01_CHEST_PAIN",
        "breathlessness": "P02_DYSPNEA",
        "palpitations": "P03_PALPITATIONS",
    }
    return {
        "patient_id": patient_id,
        "node_id": node_map.get(symptom, "P01_CHEST_PAIN"),
    }


def extract_health_score(kb26_mri_response: dict) -> dict:
    """Extract simplified health score from KB-26 MRI response.

    KB-26 returns a rich MRI payload; the Patient App needs a simplified view.
    """
    data = kb26_mri_response.get("data", kb26_mri_response)
    return {
        "score": data.get("mri_score") or data.get("composite_score"),
        "trend": data.get("trend"),
        "components": data.get("decomposition", {}),
    }
```

- [ ] **Step 17: Run transformer tests**

Run: `cd backend/services/api-gateway && python -m pytest tests/test_transforms.py -v`
Expected: All 8 tests PASS

- [ ] **Step 18: Commit transforms**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/services/api-gateway/app/api/transforms.py backend/services/api-gateway/tests/test_transforms.py
git commit -m "feat(gateway): add checkin_to_session and extract_health_score transformers"
```

---

### Task 8: Upgrade _forward() to Support Transforms

**Files:**
- Modify: `backend/services/api-gateway/app/api/endpoints/patient_app.py:119-165`

- [ ] **Step 19: Update `_forward()` signature and logic**

In `patient_app.py`, replace the `_forward()` function (lines 119-165) with:

```python
async def _forward(
    request: Request,
    service_url: str,
    path: str,
    method: str = None,
    body: bytes = None,
    response_transform=None,
) -> Any:
    """Forward request to a downstream service with X-User-* headers.

    Args:
        request: Incoming FastAPI request.
        service_url: Base URL of the target KB service.
        path: Internal path on the target service (must be the correct KB route).
        method: HTTP method override (defaults to request.method).
        body: Pre-built request body bytes. If None, reads from request.
        response_transform: Optional callable(dict) -> dict to transform the response.
    """
    method = method or request.method
    breaker = get_breaker(service_url)

    if not breaker.is_available():
        raise HTTPException(status_code=503, detail="Service temporarily unavailable")

    # Response cache check (GET only)
    cache = _get_cache()
    user_id = ""
    if hasattr(request.state, "user") and isinstance(request.state.user, dict):
        user_id = request.state.user.get("id", "")
    if cache and method == "GET":
        cached = await cache.get(method, path, user_id)
        if cached is not None:
            breaker.record_success()
            return cached

    headers = _build_headers(request)
    if body is None:
        body = await request.body() if method in ("POST", "PUT", "PATCH") else None

    try:
        async with httpx.AsyncClient(timeout=30) as client:
            resp = await client.request(
                method=method,
                url=f"{service_url}{path}",
                headers=headers,
                content=body,
                params=dict(request.query_params),
            )
            if resp.status_code >= 500:
                breaker.record_failure()
            else:
                breaker.record_success()
            result = resp.json()

            # Apply response transform if provided
            if response_transform and resp.status_code == 200:
                result = response_transform(result)

            # Cache successful GET responses
            if cache and method == "GET" and resp.status_code == 200:
                await cache.set(method, path, user_id, result)
            return result
    except httpx.ConnectError:
        breaker.record_failure()
        raise HTTPException(status_code=502, detail="Downstream service unavailable")
    except Exception as e:
        breaker.record_failure()
        logger.error("Proxy error: %s %s → %s", method, path, e)
        raise HTTPException(status_code=502, detail="Downstream service error")
```

- [ ] **Step 20: Verify gateway still starts**

Run: `cd backend/services/api-gateway && python -c "from app.api.endpoints.patient_app import _forward; print('OK')"`
Expected: `OK`

- [ ] **Step 21: Commit _forward upgrade**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/services/api-gateway/app/api/endpoints/patient_app.py
git commit -m "refactor(gateway): upgrade _forward() with body and response_transform params"
```

---

### Task 9: Fix Patient App Handler Paths

**Files:**
- Modify: `backend/services/api-gateway/app/api/endpoints/patient_app.py:1-117`

- [ ] **Step 22: Add imports and fix all handler paths**

Add imports at the top of `patient_app.py` (after existing imports):

```python
import json
from app.api.patient_resolver import resolve_patient_id
from app.api.transforms import checkin_to_session, extract_health_score
```

Replace the 8 protected route handlers (lines 53-98) with corrected paths:

```python
# --- Protected routes (JWT required, enforced by middleware) ---

@router.get("/patient/{patient_id}/health-score")
async def patient_health_score(patient_id: str, request: Request):
    """KB-26: Composite health score from Metabolic Digital Twin."""
    resolved_id = await resolve_patient_id(patient_id)
    return await _forward(
        request, settings.KB26_SERVICE_URL,
        f"/api/v1/kb26/mri/{resolved_id}",
        response_transform=extract_health_score,
    )


@router.get("/patient/{patient_id}/actions/today")
async def patient_actions_today(patient_id: str, request: Request):
    """KB-23: Today's action items from Decision Cards."""
    resolved_id = await resolve_patient_id(patient_id)
    return await _forward(
        request, settings.KB23_SERVICE_URL,
        f"/api/v1/patients/{resolved_id}/active-cards",
    )


@router.get("/patient/{patient_id}/health-drive")
async def patient_health_drive(patient_id: str, request: Request):
    """KB-25: Lifestyle Knowledge Graph recommendations."""
    resolved_id = await resolve_patient_id(patient_id)
    return await _forward(
        request, settings.KB25_SERVICE_URL,
        "/api/v1/kb25/recommend-lifestyle",
        method="POST",
        body=json.dumps({"patient_id": resolved_id}).encode(),
    )


@router.get("/patient/{patient_id}/progress")
async def patient_progress(patient_id: str, request: Request):
    """KB-20: Protocol progress from Patient Profile.

    Note: KB-20 has its own resolveFHIRPatientID() middleware, so no
    gateway-level resolution needed — pass patient_id as-is.
    """
    return await _forward(
        request, settings.KB20_SERVICE_URL,
        f"/api/v1/patient/{patient_id}/protocols",
    )


@router.get("/patient/{patient_id}/cause-effect")
async def patient_cause_effect(patient_id: str, request: Request):
    """KB-26: Cause-effect analysis (not yet implemented)."""
    raise HTTPException(
        status_code=501,
        detail={"error": "endpoint not yet implemented", "planned_service": "KB-26"},
    )


@router.get("/patient/{patient_id}/timeline")
async def patient_timeline(patient_id: str, request: Request):
    """KB-20: Patient clinical timeline (not yet implemented)."""
    raise HTTPException(
        status_code=501,
        detail={"error": "endpoint not yet implemented", "planned_service": "KB-20"},
    )


@router.get("/patient/{patient_id}/insights")
async def patient_insights(patient_id: str, request: Request):
    """KB-26: Patient-facing insights (not yet implemented)."""
    raise HTTPException(
        status_code=501,
        detail={"error": "endpoint not yet implemented", "planned_service": "KB-26"},
    )


@router.post("/patient/{patient_id}/checkin")
async def patient_checkin(patient_id: str, request: Request):
    """KB-22: Daily check-in via HPI Engine — transforms to session creation."""
    resolved_id = await resolve_patient_id(patient_id)
    req_body = await request.json()
    session_body = checkin_to_session(resolved_id, req_body)
    return await _forward(
        request, settings.KB22_SERVICE_URL,
        "/api/v1/sessions",
        method="POST",
        body=json.dumps(session_body).encode(),
    )
```

- [ ] **Step 23: Verify import chain**

Run: `cd backend/services/api-gateway && python -c "from app.api.endpoints.patient_app import router; print('OK')"`
Expected: `OK`

- [ ] **Step 24: Commit Patient App handler fixes**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/services/api-gateway/app/api/endpoints/patient_app.py
git commit -m "fix(gateway): correct Patient App handler paths to actual KB service routes"
```

---

### Task 10: Fix Doctor Dashboard Handler Paths

**Files:**
- Modify: `backend/services/api-gateway/app/api/endpoints/doctor_dashboard.py:1-75`

- [ ] **Step 25: Add import and fix handler paths**

Add import at top (after existing imports):

```python
from app.api.patient_resolver import resolve_patient_id
```

Replace the REST handlers (lines 29-74) with corrected paths:

```python
# --- Direct REST for latency-sensitive reads ---

@router.get("/patients/{patient_id}/summary")
async def patient_summary(patient_id: str, request: Request):
    """KB-20: Patient profile summary."""
    return await _forward(
        request, settings.KB20_SERVICE_URL,
        f"/api/v1/patient/{patient_id}/profile",
    )


@router.get("/patients/{patient_id}/mri")
async def patient_mri(patient_id: str, request: Request):
    """KB-26: Metabolic Risk Index."""
    resolved_id = await resolve_patient_id(patient_id)
    return await _forward(
        request, settings.KB26_SERVICE_URL,
        f"/api/v1/kb26/mri/{resolved_id}",
    )


@router.get("/patients/{patient_id}/cards")
async def patient_cards(patient_id: str, request: Request):
    """KB-23: Decision cards."""
    resolved_id = await resolve_patient_id(patient_id)
    return await _forward(
        request, settings.KB23_SERVICE_URL,
        f"/api/v1/patients/{resolved_id}/active-cards",
    )


@router.post("/cards/{card_id}/action")
async def card_action(card_id: str, request: Request):
    """KB-23: Physician action on card (approve/modify/escalate)."""
    return await _forward(
        request, settings.KB23_SERVICE_URL,
        f"/api/v1/cards/{card_id}/mcu-gate-resume",
        method="POST",
    )


# --- V-MCU safety traces ---

@router.get("/traces/{patient_id}")
async def safety_traces(patient_id: str, request: Request):
    """V-MCU: Safety trace audit log."""
    if not settings.VMCU_SERVICE_URL:
        raise HTTPException(status_code=503, detail="V-MCU service not configured")
    return await _forward(request, settings.VMCU_SERVICE_URL, f"/traces/{patient_id}")


# --- KB-20 projections (Channel B/C) ---

@router.get("/patients/{patient_id}/channel-b-inputs")
async def channel_b_inputs(patient_id: str, request: Request):
    """KB-20: Channel B projection data."""
    return await _forward(
        request, settings.KB20_SERVICE_URL,
        f"/api/v1/patient/{patient_id}/channel-b-inputs",
    )


@router.get("/patients/{patient_id}/channel-c-inputs")
async def channel_c_inputs(patient_id: str, request: Request):
    """KB-20: Channel C projection data."""
    return await _forward(
        request, settings.KB20_SERVICE_URL,
        f"/api/v1/patient/{patient_id}/channel-c-inputs",
    )
```

**Key path corrections:**
- `summary` → `/api/v1/patient/{id}/profile` (singular `/patient/`, correct KB-20 route)
- `mri` → `/api/v1/kb26/mri/{id}` (KB-26 route with kb26 prefix)
- `cards` → `/api/v1/patients/{id}/active-cards` (KB-23 route, plural `/patients/`)
- `card action` → `/api/v1/cards/{id}/mcu-gate-resume` (KB-23 route)
- `channel-b/c-inputs` → `/api/v1/patient/{id}/channel-b-inputs` (singular `/patient/`)

**Resolution note:** KB-20 routes use singular `/patient/` with its own FHIR middleware, so no gateway-level resolution needed for KB-20 calls. KB-23/KB-26 calls need `resolve_patient_id()`.

- [ ] **Step 26: Verify import chain**

Run: `cd backend/services/api-gateway && python -c "from app.api.endpoints.doctor_dashboard import router; print('OK')"`
Expected: `OK`

- [ ] **Step 27: Commit Doctor Dashboard handler fixes**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/services/api-gateway/app/api/endpoints/doctor_dashboard.py
git commit -m "fix(gateway): correct Doctor Dashboard handler paths — KB-20/23/26 route alignment"
```

---

### Task 11: Fix Catch-all Proxy — Add internal_prefix

**Files:**
- Modify: `backend/services/api-gateway/app/api/proxy.py:119-148,151-181`

- [ ] **Step 28: Add `internal_prefix` to Vaidshala SERVICE_ROUTES**

In `proxy.py`, replace the Vaidshala KB service entries (lines 118-148) with:

```python
    # Vaidshala Clinical Runtime — KB services
    "kb20_patient_profile": {
        "prefix": "/api/v1/kb20",
        "target": settings.KB20_SERVICE_URL,
        "strip_prefix": True,
        "internal_prefix": "/api/v1",
        "public_paths": []
    },
    "kb22_hpi_engine": {
        "prefix": "/api/v1/kb22",
        "target": settings.KB22_SERVICE_URL,
        "strip_prefix": True,
        "internal_prefix": "/api/v1",
        "public_paths": []
    },
    "kb23_decision_cards": {
        "prefix": "/api/v1/kb23",
        "target": settings.KB23_SERVICE_URL,
        "strip_prefix": True,
        "internal_prefix": "/api/v1",
        "public_paths": []
    },
    "kb25_lifestyle_graph": {
        "prefix": "/api/v1/kb25",
        "target": settings.KB25_SERVICE_URL,
        "strip_prefix": True,
        "internal_prefix": "/api/v1/kb25",
        "public_paths": []
    },
    "kb26_metabolic_twin": {
        "prefix": "/api/v1/kb26",
        "target": settings.KB26_SERVICE_URL,
        "strip_prefix": True,
        "internal_prefix": "/api/v1/kb26",
        "public_paths": []
    },
```

- [ ] **Step 29: Update `forward_request()` to use `internal_prefix`**

In `proxy.py`, update the `forward_request()` function's path stripping logic (around line 176-181). Replace:

```python
    target_path = path
    if strip_prefix and path.startswith(service_prefix):
        target_path = path[len(service_prefix):]
        # Ensure the path starts with a slash
        if not target_path.startswith('/'):
            target_path = '/' + target_path
```

With:

```python
    target_path = path
    if strip_prefix and path.startswith(service_prefix):
        remainder = path[len(service_prefix):]
        if not remainder.startswith('/'):
            remainder = '/' + remainder
        # Prepend the service's internal route prefix
        target_path = internal_prefix + remainder if internal_prefix else remainder
```

Also update the `forward_request()` signature to add the explicit `internal_prefix` parameter:

```python
async def forward_request(
    request: Request,
    target_url: str,
    path: str,
    strip_prefix: bool = False,
    service_prefix: str = "",
    internal_prefix: str = "",
) -> Response:
```

And update the call site in `proxy_endpoint()` (around line 700) to pass the route config:

```python
        return await forward_request(
            request=request,
            target_url=route_config["target"],
            path=full_path,
            strip_prefix=route_config["strip_prefix"],
            service_prefix=prefix,
            internal_prefix=route_config.get("internal_prefix", ""),
        )
```

- [ ] **Step 30: Verify proxy module loads**

Run: `cd backend/services/api-gateway && python -c "from app.api.proxy import SERVICE_ROUTES; print('kb22 internal_prefix:', SERVICE_ROUTES['kb22_hpi_engine'].get('internal_prefix')); print('kb25 internal_prefix:', SERVICE_ROUTES['kb25_lifestyle_graph'].get('internal_prefix'))"`
Expected:
```
kb22 internal_prefix: /api/v1
kb25 internal_prefix: /api/v1/kb25
```

- [ ] **Step 31: Commit proxy fix**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add backend/services/api-gateway/app/api/proxy.py
git commit -m "fix(gateway): add internal_prefix to catch-all proxy for correct KB route translation"
```

---

### Task 12: E2E Verification Checklist

This task verifies all three blockers are fixed by tracing through the key flows.

- [ ] **Step 32: Verify B1 — stratum hierarchy (Go unit tests)**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./internal/services/ -run TestStratumMatches -v -count=1`
Expected: All tests PASS

- [ ] **Step 33: Verify B1 — KB-22 builds cleanly**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go build ./...`
Expected: Build succeeds

- [ ] **Step 34: Verify B3 + B2 — Python tests**

Run: `cd backend/services/api-gateway && python -m pytest tests/test_patient_resolver.py tests/test_transforms.py -v`
Expected: All tests PASS

- [ ] **Step 35: Verify B2 — proxy path translation logic**

Run a quick Python smoke test:

```bash
cd backend/services/api-gateway && python -c "
from app.api.proxy import SERVICE_ROUTES

# Simulate proxy path translation
def simulate(full_path):
    for name, cfg in SERVICE_ROUTES.items():
        if full_path.startswith(cfg['prefix']):
            if cfg.get('strip_prefix'):
                remainder = full_path[len(cfg['prefix']):]
                if not remainder.startswith('/'):
                    remainder = '/' + remainder
                ip = cfg.get('internal_prefix', '')
                target = ip + remainder if ip else remainder
            else:
                target = full_path
            print(f'{full_path} → {name} → {cfg[\"target\"]}{target}')
            return
    print(f'{full_path} → NO MATCH')

simulate('/api/v1/kb22/sessions')
simulate('/api/v1/kb25/causal-chain/sbp')
simulate('/api/v1/kb26/mri/123')
simulate('/api/v1/kb20/patient/91-1001/profile')
simulate('/api/v1/ingest/fhir/Observation')
"
```

Expected output:
```
/api/v1/kb22/sessions → kb22_hpi_engine → http://localhost:8132/api/v1/sessions
/api/v1/kb25/causal-chain/sbp → kb25_lifestyle_graph → http://localhost:8136/api/v1/kb25/causal-chain/sbp
/api/v1/kb26/mri/123 → kb26_metabolic_twin → http://localhost:8137/api/v1/kb26/mri/123
/api/v1/kb20/patient/91-1001/profile → kb20_patient_profile → http://localhost:8131/api/v1/patient/91-1001/profile
/api/v1/ingest/fhir/Observation → ingestion → http://localhost:8140/fhir/Observation
```

- [ ] **Step 36: Final commit with verification notes**

No code changes — just verify all previous commits are clean:

```bash
cd /Users/apoorvabk/Downloads/cardiofit && git log --oneline -6
```

Expected: 6 commits for B1 hierarchy, B1 integration, B3 resolver, transforms, Patient App fixes, Doctor Dashboard fixes, proxy fix.

---

## Summary of Changes

| Blocker | Root Cause | Fix | Files |
|---------|-----------|-----|-------|
| B1 | KB-22 uses `==` for stratum validation; `DM_HTN` ≠ `DM_HTN_base` | `StratumMatches()` with parent-walking hierarchy | `stratum_hierarchy.go`, `session_service.go` |
| B3 | Only KB-20 resolves FHIR UUID → ABHA; KB-22/23/25/26 can't find patients | Gateway-level resolver calls KB-20 before forwarding | `patient_resolver.py` |
| B2 | Handler paths don't match KB service routes; proxy strips too much | Correct all `_forward()` paths; add `internal_prefix` to proxy | `patient_app.py`, `doctor_dashboard.py`, `proxy.py`, `transforms.py` |
