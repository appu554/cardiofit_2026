# KB-4 Patient Safety: Clinical Knowledge Governance SOP

**Standard Operating Procedure for Clinical Knowledge Management**

| Document Control | |
|-----------------|---|
| **Version** | 1.0 |
| **Effective Date** | 2025-01-01 |
| **Review Date** | 2026-01-01 |
| **Owner** | Chief Medical Officer (CMO) |
| **Approved By** | CTO + CMO |

---

## 1. Purpose

This SOP establishes the formal governance workflow for managing clinical safety knowledge within the KB-4 Patient Safety service. It ensures all clinical knowledge is:

- **Evidence-based**: Sourced from authoritative clinical references
- **Traceable**: Full audit trail from source to implementation
- **Defensible**: Regulatory-compliant with documented approval chain
- **Current**: Regularly reviewed and updated

---

## 2. Scope

This SOP applies to all clinical knowledge within KB-4:

| Knowledge Category | Primary Authority | Review Cycle |
|-------------------|-------------------|--------------|
| Black Box Warnings | FDA, TGA, EMA | Quarterly |
| Contraindications | FDA Label, SmPC | Quarterly |
| Dose Limits | FDA, Manufacturer | Semi-annual |
| Age Restrictions | FDA Pediatric | Annual |
| Pregnancy Safety | FDA PLLR, TGA | Quarterly |
| Lactation Safety | LactMed, WHO | Semi-annual |
| High-Alert Medications | ISMP | Annual |
| Beers Criteria | AGS | Upon publication |
| Anticholinergic Burden | ACB Scale | Annual |
| Lab Monitoring | FDA, NICE | Semi-annual |

---

## 3. Roles and Responsibilities

### 3.1 Chief Medical Officer (CMO)
- **Authority**: Final approval for all clinical knowledge changes
- **Responsibilities**:
  - Review and approve all knowledge additions/modifications
  - Validate clinical accuracy and relevance
  - Sign-off on governance metadata
  - Conduct periodic knowledge audits
  - Respond to clinical safety incidents

### 3.2 Clinical Informaticist
- **Authority**: Prepare and validate knowledge submissions
- **Responsibilities**:
  - Research authoritative sources
  - Draft YAML knowledge entries
  - Complete governance metadata
  - Validate RxNorm/ATC/SNOMED mappings
  - Prepare change documentation

### 3.3 Chief Technology Officer (CTO)
- **Authority**: Technical implementation approval
- **Responsibilities**:
  - Review technical accuracy of implementations
  - Approve deployment to production
  - Ensure system integrity and performance
  - Maintain audit infrastructure

### 3.4 Quality Assurance Lead
- **Authority**: Validation and testing approval
- **Responsibilities**:
  - Execute test cases for new knowledge
  - Validate alert firing behavior
  - Document edge cases
  - Sign-off on deployment readiness

---

## 4. Knowledge Lifecycle

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CLINICAL KNOWLEDGE LIFECYCLE                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐     │
│   │ IDENTIFY │───▶│  DRAFT   │───▶│  REVIEW  │───▶│ APPROVE  │     │
│   │  Source  │    │   YAML   │    │   CMO    │    │  Deploy  │     │
│   └──────────┘    └──────────┘    └──────────┘    └──────────┘     │
│        │                                                 │          │
│        │                                                 ▼          │
│        │          ┌──────────┐    ┌──────────┐    ┌──────────┐     │
│        │          │  RETIRE  │◀───│  AUDIT   │◀───│ MONITOR  │     │
│        │          │ Archive  │    │  Review  │    │   Live   │     │
│        └──────────┴──────────┴────┴──────────┴────┴──────────┘     │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.1 Phase 1: Identification

**Trigger Events**:
- New FDA drug approval or label update
- AGS Beers Criteria publication
- ISMP high-alert list update
- Clinical incident report
- Regulatory requirement change
- Scheduled review cycle

**Required Documentation**:
- Source document reference (URL, DOI, or publication)
- Date of source publication
- Jurisdiction applicability
- Clinical rationale for inclusion

### 4.2 Phase 2: Drafting

**YAML Entry Requirements**:

```yaml
# MANDATORY FIELDS - Every entry MUST include:
- rxnorm: "XXXXX"           # Primary drug identifier
- drugName: "Drug Name"     # Human-readable name
- governance:
    sourceAuthority: "FDA"  # Primary source (FDA, AGS, ISMP, TGA, WHO)
    sourceDocument: "..."   # Exact document title
    sourceSection: "..."    # Section/page reference
    sourceUrl: "..."        # Direct URL to source
    sourceVersion: "..."    # Document version/date
    jurisdiction: "US"      # Primary jurisdiction
    evidenceLevel: "A"      # A (high), B (moderate), C (low)
    effectiveDate: "..."    # When knowledge becomes active
    reviewDate: "..."       # Next mandatory review
    knowledgeVersion: "..." # Internal versioning
    approvalStatus: "DRAFT" # DRAFT → PENDING → ACTIVE → RETIRED
    approvedBy: ""          # Empty until CMO approval
    approvedAt: ""          # Empty until CMO approval
```

**Evidence Level Classification**:
| Level | Description | Source Types |
|-------|-------------|--------------|
| A | High quality | RCTs, Meta-analyses, FDA labels |
| B | Moderate quality | Observational studies, Expert consensus |
| C | Low quality | Case reports, Expert opinion |

### 4.3 Phase 3: Review

**CMO Review Checklist**:

```
□ Clinical accuracy verified against source document
□ RxNorm code validated (NLM RxNorm browser)
□ ATC code validated (WHO ATC Index)
□ SNOMED codes validated (if applicable)
□ Severity classification appropriate
□ Risk categories complete and accurate
□ Alternatives clinically appropriate
□ Jurisdiction correctly assigned
□ Evidence level justified
□ No conflicts with existing knowledge
□ Edge cases considered
□ Test cases reviewed and approved
```

**Review Timeline**:
- Standard additions: 5 business days
- Urgent safety updates: 24 hours
- Regulatory mandates: Per regulatory timeline

### 4.4 Phase 4: Approval

**Approval Workflow**:

1. **Clinical Informaticist** → Submits YAML + documentation
2. **QA Lead** → Validates test cases, signs off on functionality
3. **CMO** → Reviews clinical accuracy, approves if satisfactory
4. **CTO** → Approves technical deployment
5. **System** → Updates `approvalStatus`, `approvedBy`, `approvedAt`

**Approval Record Format**:
```yaml
governance:
  approvalStatus: "ACTIVE"
  approvedBy: "CMO"
  approvedAt: "2025-01-15T10:30:00Z"
  approvalNotes: "Reviewed against FDA label dated 2024-12-01"
```

### 4.5 Phase 5: Monitoring

**Post-Deployment Monitoring**:
- Alert firing rates by category
- Override patterns and reasons
- Clinical feedback collection
- Incident reports related to knowledge

**Key Metrics**:
| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Alert accuracy rate | >95% | <90% |
| False positive rate | <10% | >15% |
| Override rate | <20% | >30% |
| Clinician satisfaction | >4.0/5.0 | <3.5/5.0 |

### 4.6 Phase 6: Audit & Retirement

**Scheduled Audits**:
- Quarterly: High-risk categories (Black Box, Pregnancy)
- Semi-annual: All other categories
- Annual: Complete knowledge base review

**Retirement Criteria**:
- Drug withdrawn from market
- Superseded by new guideline
- Clinical evidence contradicted
- Regulatory requirement removed

**Retirement Process**:
```yaml
governance:
  approvalStatus: "RETIRED"
  retiredBy: "CMO"
  retiredAt: "2025-06-01T14:00:00Z"
  retirementReason: "Drug withdrawn from market - FDA safety alert"
```

---

## 5. Change Control

### 5.1 Change Categories

| Category | Description | Approval Required | Timeline |
|----------|-------------|-------------------|----------|
| **Critical** | Safety-impacting changes | CMO + CTO | Immediate |
| **Major** | New drug/category additions | CMO | 5 days |
| **Minor** | Text corrections, URL updates | Clinical Informaticist | 2 days |
| **Administrative** | Metadata updates | Automated | Immediate |

### 5.2 Change Request Template

```
CHANGE REQUEST: KB4-CR-YYYY-NNNN
==================================
Requested By: [Name]
Date: [Date]
Category: [Critical/Major/Minor/Administrative]

CURRENT STATE:
[Description of current knowledge entry]

PROPOSED CHANGE:
[Description of proposed change]

JUSTIFICATION:
[Clinical/regulatory rationale]

SOURCE REFERENCE:
[URL or document reference]

IMPACT ASSESSMENT:
- Affected RxNorm codes: [List]
- Estimated alert volume change: [±X%]
- Risk of false positives: [Low/Medium/High]
- Risk of false negatives: [Low/Medium/High]

APPROVALS:
□ Clinical Informaticist: _________ Date: _______
□ QA Lead: _________ Date: _______
□ CMO: _________ Date: _______
□ CTO: _________ Date: _______
```

---

## 6. Incident Management

### 6.1 Incident Classification

| Severity | Definition | Response Time |
|----------|------------|---------------|
| **SEV-1** | Patient harm occurred or imminent | 1 hour |
| **SEV-2** | Incorrect alert with clinical impact | 4 hours |
| **SEV-3** | Missing alert discovered | 24 hours |
| **SEV-4** | Minor inaccuracy reported | 5 days |

### 6.2 Incident Response Workflow

```
1. DETECT → Alert from monitoring, clinician report, or audit
2. TRIAGE → Classify severity, assign owner
3. CONTAIN → Temporary disable if SEV-1/SEV-2
4. INVESTIGATE → Root cause analysis
5. REMEDIATE → Fix knowledge entry
6. VERIFY → Test fix in staging
7. DEPLOY → Push to production
8. REPORT → Document lessons learned
```

### 6.3 Post-Incident Report Template

```
INCIDENT REPORT: KB4-IR-YYYY-NNNN
==================================
Severity: [SEV-1/2/3/4]
Date Detected: [Date/Time]
Date Resolved: [Date/Time]

DESCRIPTION:
[What happened]

ROOT CAUSE:
[Why it happened]

IMPACT:
- Patients affected: [Number]
- Alerts affected: [Number]
- Duration: [Time period]

REMEDIATION:
[What was done to fix it]

PREVENTION:
[What will prevent recurrence]

APPROVALS:
CMO Review: _________ Date: _______
```

---

## 7. Audit Trail Requirements

### 7.1 Required Audit Fields

Every knowledge change MUST record:

```json
{
  "auditId": "UUID",
  "timestamp": "ISO-8601",
  "action": "CREATE|UPDATE|DELETE|APPROVE|RETIRE",
  "performedBy": "User ID",
  "performedByRole": "CMO|CTO|Informaticist|System",
  "previousState": { /* snapshot */ },
  "newState": { /* snapshot */ },
  "changeReason": "Free text",
  "sourceReference": "URL or document",
  "approvalChain": [
    {"role": "Informaticist", "user": "...", "timestamp": "..."},
    {"role": "CMO", "user": "...", "timestamp": "..."}
  ]
}
```

### 7.2 Retention Requirements

| Record Type | Retention Period | Storage |
|-------------|------------------|---------|
| Active knowledge audit | Indefinite | Primary DB |
| Retired knowledge audit | 10 years | Archive |
| Incident reports | 10 years | Archive |
| Change requests | 7 years | Archive |

---

## 8. Compliance Mapping

### 8.1 Regulatory Alignment

| Regulation | Requirement | KB-4 Implementation |
|------------|-------------|---------------------|
| FDA 21 CFR Part 11 | Electronic records, signatures | Audit trail, approval workflow |
| HIPAA | Data integrity | Immutable audit logs |
| IEC 62304 | Medical device software | Traceability matrix |
| ISO 13485 | Quality management | Change control process |
| ACSQHC NSQHS | Clinical governance | CMO approval workflow |

### 8.2 Accreditation Support

This SOP supports hospital accreditation requirements for:
- Joint Commission (USA)
- ACSQHC (Australia)
- NABH (India)
- CQC (UK)

---

## 9. Training Requirements

### 9.1 Role-Based Training

| Role | Required Training | Frequency |
|------|-------------------|-----------|
| CMO | Governance SOP, Clinical validation | Annual |
| Clinical Informaticist | YAML authoring, Source research | Initial + Annual |
| QA Lead | Test case development, Validation | Initial + Annual |
| CTO | Technical review, Deployment | Initial + Annual |

### 9.2 Training Records

All training must be documented with:
- Trainee name and role
- Training date
- Training content
- Competency assessment result
- Trainer signature

---

## 10. Document Control

### 10.1 Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-01-01 | CMO/CTO | Initial release |

### 10.2 Review Schedule

This SOP must be reviewed:
- Annually by CMO
- Upon significant regulatory change
- After SEV-1 incident
- Upon system architecture change

---

## Appendix A: Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│           KB-4 GOVERNANCE QUICK REFERENCE                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  NEW KNOWLEDGE ENTRY:                                       │
│  1. Research source → 2. Draft YAML → 3. Submit CR          │
│  4. QA validates → 5. CMO reviews → 6. CTO deploys          │
│                                                             │
│  EMERGENCY UPDATE (SEV-1):                                  │
│  1. Notify CMO immediately                                  │
│  2. Draft emergency YAML                                    │
│  3. CMO verbal approval → document within 24h               │
│  4. Hotfix deployment                                       │
│  5. Post-incident report within 72h                         │
│                                                             │
│  MANDATORY GOVERNANCE FIELDS:                               │
│  ✓ sourceAuthority    ✓ sourceDocument                      │
│  ✓ jurisdiction       ✓ evidenceLevel                       │
│  ✓ effectiveDate      ✓ approvalStatus                      │
│  ✓ approvedBy         ✓ approvedAt                          │
│                                                             │
│  CONTACTS:                                                  │
│  CMO: [TBD]           CTO: [TBD]                            │
│  On-Call: [TBD]       Incident Line: [TBD]                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Appendix B: YAML Template

```yaml
# Template for new clinical knowledge entry
# Copy this template and fill in all required fields

entries:
  - rxnorm: ""                    # REQUIRED: RxNorm CUI
    drugName: ""                  # REQUIRED: Generic drug name
    atcCode: ""                   # RECOMMENDED: WHO ATC code
    drugClass: ""                 # RECOMMENDED: Therapeutic class

    # Category-specific fields here...

    governance:                   # REQUIRED: All governance fields
      sourceAuthority: ""         # FDA, AGS, ISMP, TGA, WHO, etc.
      sourceDocument: ""          # Exact document title
      sourceSection: ""           # Section or page reference
      sourceUrl: ""               # Direct URL to source
      sourceVersion: ""           # Document version or date
      jurisdiction: ""            # US, AU, IN, EU, GLOBAL
      additionalJurisdictions: [] # Other applicable regions
      evidenceLevel: ""           # A, B, or C
      effectiveDate: ""           # YYYY-MM-DD
      reviewDate: ""              # YYYY-MM-DD (next review)
      knowledgeVersion: ""        # e.g., "2025.1"
      approvalStatus: "DRAFT"     # DRAFT until CMO approval
      approvedBy: ""              # Empty until approved
      approvedAt: ""              # Empty until approved
```

---

**END OF DOCUMENT**

*This SOP is maintained by the Clinical Informatics team and approved by the CMO and CTO.*
