# CDS Hooks Response Shape (v2.0)

**Closes:** the hidden gap identified in the 2026-05 Layer 2 / Layer 3
gap analysis (`claudedocs/audits/2026-05-Layer2-Layer3-Gap-Analysis.md`).
The CDS Hooks emitter (`shared/cql-toolchain/cds_hooks_emitter.py`)
produces conforming output; this doc is the dedicated reference for
authors and reviewers.

## Scope

This document covers the response payload produced by
`emit_cds_hooks_response()` for a single rule fire, the upstream
PlanDefinition `$apply` operation that produces the embedded FHIR
`RequestOrchestration` resource, the validation rules applied by
`validate_cds_hooks_v2_response()`, and the common emission errors
plus their remediation.

## Spec reference

- **CDS Hooks v2.0** — https://cds-hooks.org/specification/current/
- **FHIR Clinical Reasoning Module — `PlanDefinition/$apply`** — https://www.hl7.org/fhir/clinicalreasoning-module.html

## Card structure (per CDS Hooks v2.0)

The emitter returns a top-level object with a single key, `cards`,
whose value is a list of one card per rule fire:

```json
{
  "cards": [
    {
      "uuid": "<uuid5(NAMESPACE_URL, 'vaidshala/<rule_id>')>",
      "summary": "<<=140-char clinician-facing summary>",
      "indicator": "info" | "warning" | "critical",
      "detail": "<full markdown-friendly detail>",
      "source": {
        "label": "Vaidshala Clinical Reasoning",
        "url": "https://vaidshala.cardiofit/rules/"
      },
      "suggestions": [
        {
          "label": "<recommendation_text>",
          "uuid": "<uuid4>",
          "actions": [
            {
              "type": "create",
              "description": "<detail or summary>",
              "resource": { "<embedded RequestOrchestration FHIR>" : "..." }
            }
          ]
        }
      ],
      "selectionBehavior": "any" | "at-most-one",
      "links": [
        {
          "label": "Rule: <rule_id>",
          "url": "https://vaidshala.cardiofit/rules/<rule_id>",
          "type": "absolute"
        }
      ]
    }
  ]
}
```

### Required keys per card (validator-enforced)

| Key | Notes |
|---|---|
| `uuid` | Stable across fires of the same rule (uuid5 of NAMESPACE_URL + `vaidshala/<rule_id>`). Lets the EMR de-duplicate cards. |
| `summary` | <=140 chars, clinician-facing. Truncate the spec `summary` if needed. |
| `indicator` | One of `{"info","warning","critical"}`. Mapped from rule tier — see below. |
| `source` | Object with required `label`. `url` is recommended. |

### Recommended keys

| Key | Notes |
|---|---|
| `detail` | Long-form explanation. Defaults to `summary` if not supplied. |
| `suggestions[]` | Each suggestion requires `label`, `uuid`, `actions[]`. |
| `links[]` | Each link requires `label`, `url`, `type`. |
| `selectionBehavior` | Set to `any` when suggestions are present, otherwise `at-most-one`. |

### Indicator mapping (recommended convention)

| Tier | Indicator |
|---|---|
| Tier 1 immediate-safety | `critical` (default) or `warning` |
| Tier 2 deprescribing | `warning` |
| Tier 3 quality-gap | `info` |
| Tier 4 surveillance | `info` |

Authors override per-rule when the clinical context warrants escalation.

## PlanDefinition $apply integration

`emit_cds_hooks_response()` accepts an optional
`request_orchestration` parameter — the FHIR `RequestOrchestration`
resource produced by `apply_plan_definition()`. When supplied, the
resource is embedded as the `actions[0].resource` of the suggestion.

### Bundle requirements

`apply_plan_definition()` reduces a Bundle containing **at minimum**:

- One `PlanDefinition` (carries `url` referenced via `instantiatesCanonical`)
- One `ActivityDefinition` (carries `code.text` mapped to suggestion `title`)

Optional resources may be present (e.g. `Library`); the reducer ignores
them. Missing `PlanDefinition` or `ActivityDefinition` raises
`ValueError("Bundle must include PlanDefinition + ActivityDefinition")`.

### Output FHIR `RequestOrchestration` shape

```json
{
  "resourceType": "RequestOrchestration",
  "status": "draft",
  "intent": "proposal",
  "instantiatesCanonical": ["<PlanDefinition.url>"],
  "action": [
    {
      "title": "<ActivityDefinition.code.text or rule_fire.recommendation_text>",
      "definitionCanonical": {
        "reference": "ActivityDefinition/<ActivityDefinition.id>"
      }
    }
  ]
}
```

This resource is then embedded in the suggestion `actions[0].resource`
so the EMR can POST it back to the FHIR server when the clinician
accepts the suggestion.

## Sample card payloads

### Sample 1: PPI deprescribe (Tier 2, warning)

```json
{
  "cards": [
    {
      "uuid": "8b3c0f5d-8c0a-5e6e-b1c2-2a4f0a8d9e5e",
      "summary": "PPI active >8 weeks for uncomplicated PUD per STOPP v3 F2 — consider step-down/cessation",
      "indicator": "warning",
      "detail": "Resident on ATC A02BC PPI without active GORD_COMPLICATED, BARRETT_OESOPHAGUS, or NSAID GI prophylaxis indication. STOPP v3 §F2 (O'Mahony 2023) recommends dose reduction or earlier discontinuation.",
      "source": {
        "label": "Vaidshala Clinical Reasoning",
        "url": "https://vaidshala.cardiofit/rules/"
      },
      "suggestions": [
        {
          "label": "Consider PPI step-down/cessation per STOPP v3 F2",
          "uuid": "9f8d0c2b-3a1f-4b8a-9d3e-7c5b1a2e4f6c",
          "actions": [
            {
              "type": "create",
              "description": "Draft deprescribing recommendation (PPI taper)",
              "resource": {
                "resourceType": "RequestOrchestration",
                "status": "draft",
                "intent": "proposal",
                "instantiatesCanonical": ["https://vaidshala.cardiofit/PlanDefinition/stopp-f2-ppi-deprescribe"],
                "action": [{
                  "title": "Draft PPI deprescribing recommendation",
                  "definitionCanonical": {"reference": "ActivityDefinition/draft-deprescribing"}
                }]
              }
            }
          ]
        }
      ],
      "selectionBehavior": "any",
      "links": [{
        "label": "Rule: STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS",
        "url": "https://vaidshala.cardiofit/rules/STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS",
        "type": "absolute"
      }]
    }
  ]
}
```

### Sample 2: Hyperkalemia trajectory (Tier 1, critical)

```json
{
  "cards": [
    {
      "uuid": "2e3a5c1f-0d7e-5c8d-9b2a-4f6e8d1c3b5a",
      "summary": "Potassium trending up with delta >+0.5 mmol/L over 30d — review ACEi/ARB",
      "indicator": "critical",
      "detail": "Resident potassium baseline trajectory crossed the +0.5 mmol/L delta threshold over 30 days. Substrate-aware fire (DeltaFromBaseline + IsTrending). Consider ACEi/ARB hold and recheck within 24-72h.",
      "source": {"label": "Vaidshala Clinical Reasoning", "url": "https://vaidshala.cardiofit/rules/"},
      "suggestions": [],
      "selectionBehavior": "at-most-one",
      "links": [{
        "label": "Rule: VAIDSHALA_T1_HYPERKALEMIA_TRAJECTORY_30D",
        "url": "https://vaidshala.cardiofit/rules/VAIDSHALA_T1_HYPERKALEMIA_TRAJECTORY_30D",
        "type": "absolute"
      }]
    }
  ]
}
```

### Sample 3: Antipsychotic consent gathering (Tier 1, warning + suggestion)

```json
{
  "cards": [
    {
      "uuid": "5c1f8d2a-7e4b-5b3c-8d9e-6f1a2b3c4d5e",
      "summary": "Antipsychotic in dementia without documented SDM consent — Quality of Care Principles 2025",
      "indicator": "warning",
      "detail": "Restrictive-practice authorisation requires active SDM consent for chemical restraint. Draft a consent-gathering recommendation routed to the resident's recorded SDM.",
      "source": {"label": "Vaidshala Clinical Reasoning", "url": "https://vaidshala.cardiofit/rules/"},
      "suggestions": [
        {
          "label": "Draft consent-gathering action for SDM",
          "uuid": "1a2b3c4d-5e6f-4a8b-9c0d-1e2f3a4b5c6d",
          "actions": [{
            "type": "create",
            "description": "Create consent_gathering_recommendation routed to SDM",
            "resource": {
              "resourceType": "RequestOrchestration",
              "status": "draft",
              "intent": "proposal",
              "instantiatesCanonical": ["https://vaidshala.cardiofit/PlanDefinition/restrictive-practice-consent"],
              "action": [{
                "title": "Draft consent-gathering recommendation",
                "definitionCanonical": {"reference": "ActivityDefinition/consent-gather"}
              }]
            }
          }]
        }
      ],
      "selectionBehavior": "any",
      "links": [{
        "label": "Rule: VAIDSHALA_T1_RESTRICTIVE_PRACTICE_CONSENT",
        "url": "https://vaidshala.cardiofit/rules/VAIDSHALA_T1_RESTRICTIVE_PRACTICE_CONSENT",
        "type": "absolute"
      }]
    }
  ]
}
```

## Validation rules

`validate_cds_hooks_v2_response()` returns a list of error strings;
empty list == valid.

| Check | Failure mode |
|---|---|
| Top-level `cards` key present | `"response missing required key 'cards'"` |
| `cards` is a list | `"response.cards must be a list"` |
| Each card has `uuid`, `summary`, `indicator`, `source` | `"cards[i] missing required key '<k>'"` |
| `indicator` in `{"info","warning","critical"}` | `"cards[i].indicator '<v>' not in {...}"` |
| `card.source.label` present | `"cards[i].source missing required key 'label'"` |
| Each suggestion has `label`, `uuid`, `actions` | `"cards[i].suggestions[s] missing key '<k>'"` |
| Each link has `label`, `url`, `type` | `"cards[i].links[l] missing key '<k>'"` |

The validator is **structural** — it does not enforce CDS Hooks
service registration / hook-type semantics. The two recognised hook
types (`order-select`, `order-sign`) are checked at emission time by
`emit_cds_hooks_response()` and raise `ValueError` if violated.

## Common emission errors and remediation

### 1. `ValueError: indicator '<v>' not in {'info','warning','critical'}`

**Cause:** the `RuleFire.indicator` value is not one of the three
allowed strings (e.g. spelling drift like `"warn"` or tier-name
leakage like `"tier_2_deprescribing"`).

**Remediation:** map your tier to one of the three indicators
(see *Indicator mapping* above). The emitter does not fall back.

### 2. `ValueError: hook_type '<v>' not in {'order-select','order-sign'}`

**Cause:** caller passed an unsupported `hook_type` parameter
(only `order-select` and `order-sign` are emitted today).

**Remediation:** use `"order-select"` for advisory cards on order
selection / draft creation, and `"order-sign"` for the final-sign
gate. `patient-view` and other CDS Hooks types are not yet emitted
(planned for a future wave).

### 3. `ValueError: Bundle must include PlanDefinition + ActivityDefinition`

**Cause:** `apply_plan_definition()` was called with a Bundle missing
either resource type. This usually indicates the Bundle was loaded
from disk but the wrong path / a malformed JSON file.

**Remediation:** confirm `bundle["entry"][i]["resource"]["resourceType"]`
includes both `"PlanDefinition"` and `"ActivityDefinition"`. Use
`load_bundle()` to load and `_find_resource(bundle, "PlanDefinition")`
to verify out of band.

### 4. Validator reports `cards[i] missing required key 'source'`

**Cause:** the card was constructed manually (not via
`emit_cds_hooks_response()`) and the `source` object was omitted.

**Remediation:** use the emitter — it always populates
`source.label` and `source.url` from `RuleFire.source_label` / 
`RuleFire.source_url` (defaults are sane).

### 5. Validator reports `cards[i].suggestions[s] missing key 'uuid'`

**Cause:** custom suggestion objects were appended after emission
without a `uuid`.

**Remediation:** generate a `uuid4()` per suggestion. The CDS Hooks
spec uses suggestion uuids to correlate clinician acceptance back to
the originating card.

### 6. Validator reports `cards[i].links[l] missing key 'type'`

**Cause:** custom links were appended without a `type` field. CDS
Hooks v2.0 requires `type` to disambiguate `absolute` vs `smart`
links.

**Remediation:** use `"absolute"` for plain HTTP links and `"smart"`
for SMART-on-FHIR launch URLs. The emitter sets `type: "absolute"`
for the auto-generated rule-documentation link.

### 7. Card `summary` exceeds 140 characters

**Cause:** authors copy a long spec `summary` directly into
`RuleFire.summary`. While the validator does not enforce the 140-char
limit (CDS Hooks recommendation, not requirement), some EMR cards
truncate at 140 chars and lose information.

**Remediation:** truncate to 140 chars at the call site
(`spec.get("summary","")[:140]`), and put the long-form text into
`RuleFire.detail`.

### 8. EMR ignores `selectionBehavior`

**Cause:** the field is set to a value other than `"at-most-one"` or
`"any"`.

**Remediation:** the emitter sets `selectionBehavior` automatically
based on suggestion count — `"any"` when suggestions exist,
`"at-most-one"` otherwise. Don't override unless you have a specific
EMR requirement.

## Related files

- Emitter: `shared/cql-toolchain/cds_hooks_emitter.py`
- Tests: `shared/cql-toolchain/tests/test_cds_hooks_emitter.py`
- Wave 3 / 4A / 5 batch acceptance tests exercise the emitter end-to-end:
  - `tests/test_tier2_wave3_batch.py`
  - `tests/test_tier3_wave4a_batch.py`
  - `tests/test_tier4_wave5_batch.py`
