# LLM Constitution for Clinical Knowledge Extraction

**Document Version**: 1.0
**Status**: APPROVED - BINDING
**Effective Date**: 2026-01-20

---

## Preamble

This Constitution establishes the governance framework for Large Language Model (LLM) usage in clinical knowledge extraction for the CardioFit Knowledge Base system. It is designed to ensure patient safety, regulatory compliance, and clinical accuracy.

---

## Article I: Fundamental Principle

### Section 1.1: The Prime Directive

> **LLMs generate DRAFT facts only.**
>
> No LLM-extracted clinical fact may influence patient care without human validation.

### Section 1.2: Rationale

LLMs are powerful tools for extracting structured data from unstructured clinical text (SPL labels, guidelines, literature). However, they:

- May hallucinate clinical details
- Cannot verify pharmacological accuracy
- Lack accountability for patient outcomes

Therefore, LLM output is **advisory input** to human experts, not **clinical truth**.

---

## Article II: Status Lifecycle

### Section 2.1: Mandatory Status Progression

```
LLM Extraction → DRAFT → [Human Review] → APPROVED → [Activation] → ACTIVE
                   ↓
              REJECTED (with reason)
```

### Section 2.2: Status Definitions

| Status | Meaning | Who Can Set |
|--------|---------|-------------|
| `DRAFT` | LLM-generated, awaiting review | System (automatic) |
| `APPROVED` | Human-validated, ready for activation | Pharmacist, Clinical Informaticist |
| `ACTIVE` | In production use | System (via `activate_fact()`) |
| `REJECTED` | Failed validation | Reviewer |

### Section 2.3: Database Enforcement

This Constitution is enforced at the database level via trigger:

```sql
-- LLM-extracted facts cannot be ACTIVE without human validation
IF NEW.extraction_source = 'LLM' AND
   NEW.status = 'ACTIVE' AND
   NEW.validated_by IS NULL THEN
    RAISE EXCEPTION 'LLM GOVERNANCE VIOLATION...';
END IF;
```

**This cannot be bypassed by application code.**

---

## Article III: Extraction Source Classification

### Section 3.1: Source Types

| Source | Governance | Examples |
|--------|------------|----------|
| `MANUAL` | Standard review | Hand-entered facts |
| `LLM` | **Strict review required** | SPL extraction, guideline parsing |
| `API_SYNC` | Automated verification | ONC DDI API, RxNav |
| `ETL` | Schema validation | CMS Formulary CSV, LOINC files |

### Section 3.2: LLM-Specific Requirements

For any fact where `extraction_source = 'LLM'`:

1. **Confidence Score Required**: Must have `confidence_score` between 0.0 and 1.0
2. **Confidence Signals Required**: Must document what contributed to confidence
3. **Source Citation Required**: Must reference specific SPL section or document
4. **Extraction Method Versioned**: Must record model version (e.g., `claude-renal-v2.3`)

---

## Article IV: Human Validation Requirements

### Section 4.1: Qualified Reviewers

Only the following roles may approve LLM-extracted facts:

| Role | Scope |
|------|-------|
| Clinical Pharmacist (PharmD) | All drug-related facts |
| Clinical Informaticist | All facts |
| Physician (MD/DO) | Clinical guidelines, safety signals |
| P&T Committee | Formulary decisions |

### Section 4.2: Review Documentation

Each approval must record:

```json
{
  "validated_by": "jane.smith@hospital.org",
  "validated_at": "2026-01-20T14:30:00Z",
  "reviewer_role": "PHARMACIST",
  "review_decision": "APPROVE",
  "review_notes": "Verified against DailyMed SPL section 8.6"
}
```

### Section 4.3: Rejection Handling

Rejected facts must include:

- Rejection reason (structured)
- Specific errors identified
- Recommendation (re-extract, manual entry, or discard)

---

## Article V: Confidence Thresholds

### Section 5.1: Confidence Bands

| Band | Score Range | Handling |
|------|-------------|----------|
| `HIGH` | ≥ 0.85 | Standard review queue |
| `MEDIUM` | 0.65 - 0.84 | Priority review queue |
| `LOW` | < 0.65 | Expert review required |

### Section 5.2: Auto-Rejection Threshold

Facts with `confidence_score < 0.50` are automatically marked for re-extraction or manual entry.

### Section 5.3: Confidence Signal Requirements

The `confidence_signals` JSONB must include:

```json
{
  "source_quality": "HIGH|MEDIUM|LOW",
  "extraction_clarity": "CLEAR|AMBIGUOUS|CONFLICTING",
  "cross_reference_match": true|false,
  "model_certainty": 0.0-1.0,
  "flags": ["needs_clinical_context", "multiple_interpretations"]
}
```

---

## Article VI: Prohibited Actions

### Section 6.1: Absolute Prohibitions

The following actions are **FORBIDDEN** and will trigger system alerts:

1. **Direct ACTIVE Status**: Setting `status = 'ACTIVE'` for LLM facts without `validated_by`
2. **Bypassing Review**: Any attempt to circumvent the DRAFT → APPROVED flow
3. **Modifying Extraction Source**: Changing `extraction_source` from 'LLM' to 'MANUAL'
4. **Backdating Validation**: Setting `validated_at` to a past timestamp

### Section 6.2: Technical Enforcement

```sql
-- These triggers enforce Article VI
CREATE TRIGGER trg_llm_governance
BEFORE INSERT OR UPDATE ON clinical_facts
FOR EACH ROW
EXECUTE FUNCTION enforce_llm_governance();
```

---

## Article VII: Audit Trail

### Section 7.1: Required Logging

All LLM extraction activities must be logged:

| Event | Log Destination |
|-------|-----------------|
| Extraction started | `audit.extraction_log` |
| Extraction completed | `audit.extraction_log` |
| Fact created (DRAFT) | `audit.fact_audit_log` |
| Review decision | `fact_reviews` |
| Activation | `audit.fact_audit_log` |

### Section 7.2: Retention

- Extraction logs: 7 years (FDA requirement)
- Review decisions: 7 years
- Audit trail: Indefinite

---

## Article VIII: Model Versioning

### Section 8.1: Version Tracking

Each LLM extraction must record:

```
extraction_method = "{model}-{task}-v{version}"
Examples:
- claude-renal-v2.3
- gpt4-safety-v1.0
- gemini-guideline-v3.1
```

### Section 8.2: Model Retirement

When a model version is retired:

1. All DRAFT facts from that version enter **priority review**
2. No new extractions use the retired version
3. Retirement is logged in `schema_version_registry`

---

## Article IX: Emergency Procedures

### Section 9.1: Safety Signal Override

In case of urgent patient safety signal:

1. **Clinical Leadership** may authorize immediate activation
2. Must be documented with `emergency_override = true`
3. Post-activation review required within 24 hours

### Section 9.2: Mass Extraction Failure

If an LLM produces systematically incorrect extractions:

1. Halt all extractions from that model
2. Quarantine all DRAFT facts from affected batch
3. Notify Clinical Informatics Lead
4. Root cause analysis required before resumption

---

## Article X: Amendments

### Section 10.1: Change Process

Amendments to this Constitution require:

1. Proposal by Clinical Informatics or Engineering Lead
2. Review by Patient Safety Committee
3. Approval by Chief Medical Information Officer
4. 30-day notice before implementation

### Section 10.2: Version Control

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-01-20 | Initial Constitution |

---

## Signatures

By implementing this system, the following parties acknowledge and agree to this Constitution:

- [ ] Chief Medical Information Officer
- [ ] Director of Clinical Informatics
- [ ] Chief Technology Officer
- [ ] Patient Safety Officer

---

## Appendix A: Database Schema Alignment

```sql
-- clinical_facts table columns supporting this Constitution
extraction_source   VARCHAR(50)  -- 'MANUAL', 'LLM', 'API_SYNC', 'ETL'
confidence_score    NUMERIC(3,2) -- 0.00 to 1.00
confidence_band     ENUM         -- 'HIGH', 'MEDIUM', 'LOW'
confidence_signals  JSONB        -- Detailed confidence breakdown
validated_by        VARCHAR(255) -- Reviewer identity
validated_at        TIMESTAMP    -- Validation timestamp
extraction_method   VARCHAR(100) -- Model version string
```

## Appendix B: Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│                    LLM CONSTITUTION                         │
│                    QUICK REFERENCE                          │
├─────────────────────────────────────────────────────────────┤
│  ✅ LLMs generate DRAFT only                                │
│  ✅ Human validation required for ACTIVE                    │
│  ✅ Confidence score mandatory                              │
│  ✅ Source citation mandatory                               │
│  ✅ Model version tracked                                   │
├─────────────────────────────────────────────────────────────┤
│  ❌ NO direct ACTIVE status for LLM facts                   │
│  ❌ NO bypassing review queue                               │
│  ❌ NO changing extraction_source                           │
│  ❌ NO backdating validation                                │
├─────────────────────────────────────────────────────────────┤
│  Database enforces these rules via triggers.                │
│  Application code CANNOT bypass.                            │
└─────────────────────────────────────────────────────────────┘
```
