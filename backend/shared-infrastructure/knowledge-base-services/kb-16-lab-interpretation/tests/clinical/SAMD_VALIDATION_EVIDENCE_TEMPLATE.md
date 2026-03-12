# Software as a Medical Device (SaMD) Validation Evidence Package
## KB-16 Lab Interpretation & Trending Service

---

**Document ID**: KB16-SAMD-VEP-001
**Version**: 1.0
**Classification**: Regulatory Validation Evidence
**Applicable Standards**: IEC 62304, FDA SaMD Guidance, ISO 14971, 21 CFR Part 820

---

## 1. DEVICE IDENTIFICATION

### 1.1 Product Information

| Field | Value |
|-------|-------|
| **Product Name** | KB-16 Lab Interpretation & Trending Service |
| **Product Version** | 1.0.0 |
| **UDI-DI** | [To be assigned] |
| **Manufacturer** | CardioFit Clinical Systems |
| **Date of Manufacture** | [Date] |

### 1.2 Software Identification

| Component | Version | Build | Hash |
|-----------|---------|-------|------|
| KB-16 Server | 1.0.0 | _____ | _____ |
| KB-8 Calculator | 1.0.0 | _____ | _____ |
| Database Schema | 1.0.0 | _____ | _____ |
| Reference Data | 2024.01 | _____ | _____ |

### 1.3 Classification

| Regulatory Body | Classification | Basis |
|-----------------|----------------|-------|
| FDA (US) | Class II (510(k)) | Clinical Decision Support |
| EU MDR | Class IIa | Rule 11 - Software for diagnosis |
| TGA (AU) | Class IIa | Active medical device |
| Health Canada | Class II | Risk Level 2 |

---

## 2. INTENDED USE STATEMENT

### 2.1 Intended Use

KB-16 Lab Interpretation & Trending Service is a clinical decision support system intended to assist licensed healthcare professionals in:

1. **Interpreting laboratory test results** by comparing values against age-, sex-, and context-adjusted reference ranges
2. **Detecting critical/panic values** requiring immediate clinical attention
3. **Identifying clinical patterns** across laboratory panels (BMP, CBC, LFT, etc.)
4. **Tracking trends** in patient laboratory values over time
5. **Generating care gap alerts** for overdue monitoring tests

### 2.2 Indications for Use

- Processing and interpretation of clinical laboratory results
- Identification of values outside normal reference ranges
- Detection of critical values per CAP (College of American Pathologists) guidelines
- Pattern recognition across common laboratory panels
- Trending analysis for chronic disease monitoring
- Care gap detection for preventive health monitoring

### 2.3 Contraindications

- **NOT intended for standalone diagnosis** - all interpretations require clinician review
- **NOT validated for research-only assays** - validated tests listed in Appendix A
- **NOT for pediatric patients <1 year** without age-specific reference data
- **NOT for point-of-care (POC) devices** - validated for central laboratory results only

### 2.4 Target Users

| User Type | Training Requirement | Supervision |
|-----------|---------------------|-------------|
| Physicians | Licensed medical degree | Independent |
| Nurse Practitioners | NP license + lab interpretation training | Per state regulations |
| Physician Assistants | PA license + lab interpretation training | Physician supervision |
| Clinical Pharmacists | PharmD + clinical training | Per protocol |
| Laboratory Directors | MD/DO/PhD with CLIA certification | Independent |

---

## 3. RISK MANAGEMENT (ISO 14971)

### 3.1 Risk Analysis Summary

| Hazard ID | Hazard | Severity | Probability | Risk Level | Mitigation |
|-----------|--------|----------|-------------|------------|------------|
| H-001 | False negative on critical value | Catastrophic (5) | Remote (2) | HIGH | Validated panic thresholds per CAP |
| H-002 | False positive on critical value | Minor (2) | Occasional (3) | MEDIUM | Context-aware interpretation |
| H-003 | Incorrect eGFR calculation | Moderate (3) | Remote (2) | MEDIUM | KB-8 formula validation |
| H-004 | Missed care gap | Minor (2) | Remote (2) | LOW | Rule-based detection |
| H-005 | System unavailability | Moderate (3) | Remote (2) | MEDIUM | Health checks, failover |
| H-006 | Incorrect trend direction | Moderate (3) | Remote (2) | MEDIUM | Statistical validation |
| H-007 | Data integrity compromise | Major (4) | Improbable (1) | MEDIUM | Audit logging |

### 3.2 Risk Mitigation Evidence

| Mitigation | Verification Method | Test Reference | Result |
|------------|---------------------|----------------|--------|
| Critical value thresholds | Functional testing | Phase 2: P2.1-P2.10 | ☐ Pass ☐ Fail |
| Context-aware interpretation | Functional testing | Phase 4: P4.1-P4.4 | ☐ Pass ☐ Fail |
| eGFR formula accuracy | Unit + integration testing | Phase 1: P1.1-P1.6 | ☐ Pass ☐ Fail |
| Care gap detection | Functional testing | Phase 6: P6.1-P6.2 | ☐ Pass ☐ Fail |
| System availability | Performance testing | Phase 8: P8.1-P8.3 | ☐ Pass ☐ Fail |
| Trending accuracy | Statistical analysis | Phase 5: P5.1-P5.3 | ☐ Pass ☐ Fail |
| Audit trail integrity | Functional testing | Phase 7: P7.1-P7.2 | ☐ Pass ☐ Fail |

### 3.3 Residual Risk Acceptance

| Residual Risk | Benefit | Risk-Benefit Analysis |
|---------------|---------|----------------------|
| Rare false negative | Timely critical value notification | Acceptable: sensitivity >99.5% |
| Occasional false positive | Pattern recognition support | Acceptable: specificity >95% |
| Context limitations | Reduced alert fatigue | Acceptable: requires clinician input |

**Risk Management File Reference**: KB16-RMF-001

---

## 4. SOFTWARE DEVELOPMENT (IEC 62304)

### 4.1 Software Safety Classification

| Classification | Justification |
|----------------|---------------|
| **Class B** | Software contributes to clinical decisions but cannot directly cause harm without clinician intervention |

### 4.2 Development Process Evidence

| IEC 62304 Requirement | Evidence Document | Status |
|-----------------------|-------------------|--------|
| 5.1 Software Development Planning | KB16-SDP-001 | ☐ Complete |
| 5.2 Software Requirements Analysis | KB16-SRS-001 | ☐ Complete |
| 5.3 Software Architectural Design | KB16-SAD-001 | ☐ Complete |
| 5.4 Software Detailed Design | KB16-SDD-001 | ☐ Complete |
| 5.5 Software Unit Implementation | Source code repository | ☐ Complete |
| 5.6 Software Integration | KB16-SIT-001 | ☐ Complete |
| 5.7 Software System Testing | KB16-SST-001 | ☐ Complete |
| 5.8 Software Release | KB16-SRN-001 | ☐ Complete |

### 4.3 SOUP (Software of Unknown Provenance) Analysis

| SOUP Component | Version | Risk Level | Validation |
|----------------|---------|------------|------------|
| Go Runtime | 1.22 | Low | Manufacturer validation |
| PostgreSQL | 15 | Low | Manufacturer validation |
| Redis | 7 | Low | Manufacturer validation |
| Gin Framework | 1.9.1 | Low | Open source audit |
| gorm | 1.25 | Low | Open source audit |

### 4.4 Cybersecurity Evidence

| Control | Implementation | Verification |
|---------|----------------|--------------|
| Authentication | JWT tokens | ☐ Verified |
| Authorization | Role-based access | ☐ Verified |
| Data encryption | TLS 1.3 in transit | ☐ Verified |
| Audit logging | Complete request/response | ☐ Verified |
| Input validation | Strict schema validation | ☐ Verified |

---

## 5. VERIFICATION & VALIDATION

### 5.1 Test Strategy Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    V-Model Test Coverage                         │
├─────────────────────────────────────────────────────────────────┤
│  Requirements ←→ System Validation (Clinical Acceptance)        │
│  Architecture ←→ Integration Testing (Panel/Context Tests)      │
│  Design ←→ Component Testing (Interpretation Engine)            │
│  Implementation ←→ Unit Testing (Functions/Methods)             │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Test Coverage Matrix

| Requirement Category | Test Phase | Test Count | Coverage |
|---------------------|------------|------------|----------|
| Critical Value Detection | Phase 2 | 10 | 100% |
| Reference Range Application | Phase 2, 4 | 14 | 100% |
| Panel Pattern Recognition | Phase 3 | 6 | 100% |
| Context-Aware Interpretation | Phase 4 | 4 | 100% |
| Delta Check/Trending | Phase 5 | 3 | 100% |
| Care Gap Detection | Phase 6 | 2 | 100% |
| Audit Trail | Phase 7 | 2 | 100% |
| Performance SLAs | Phase 8 | 3 | 100% |
| Edge Cases | Phase 9 | 3 | 100% |

### 5.3 Test Execution Summary

| Phase | Description | Total | Passed | Failed | Status |
|-------|-------------|-------|--------|--------|--------|
| 0 | Health Checks | 3 | ___ | ___ | ☐ Pass |
| 1 | KB-8 Dependency | 7 | ___ | ___ | ☐ Pass |
| 2 | Critical Values | 10 | ___ | ___ | ☐ Pass |
| 3 | Panel Patterns | 6 | ___ | ___ | ☐ Pass |
| 4 | Context-Aware | 4 | ___ | ___ | ☐ Pass |
| 5 | Delta/Trending | 3 | ___ | ___ | ☐ Pass |
| 6 | Care Gaps | 2 | ___ | ___ | ☐ Pass |
| 7 | Governance | 2 | ___ | ___ | ☐ Pass |
| 8 | Performance | 3 | ___ | ___ | ☐ Pass |
| 9 | Edge Cases | 3 | ___ | ___ | ☐ Pass |
| **TOTAL** | | **43** | **___** | **___** | **☐ Pass** |

### 5.4 Clinical Validation Evidence

| Validation Activity | Evidence | Reference |
|--------------------|----------|-----------|
| Critical value accuracy | Newman test report | Phase 2 results |
| Panel pattern recognition | Newman test report | Phase 3 results |
| Context adjustment accuracy | Newman test report | Phase 4 results |
| Formula validation (eGFR, AG) | KB-8 unit tests | KB8-UT-001 |
| Reference range accuracy | Clinical review | CMO sign-off |
| Performance under load | Newman performance tests | Phase 8 results |

### 5.5 Traceability Matrix

| Requirement ID | Requirement | Test ID | Result |
|----------------|-------------|---------|--------|
| REQ-001 | Detect K >6.5 as critical | P2.2 | ☐ Pass |
| REQ-002 | Detect K <2.5 as critical | P2.3 | ☐ Pass |
| REQ-003 | Calculate eGFR per CKD-EPI 2021 | P1.1, P1.2 | ☐ Pass |
| REQ-004 | Calculate Anion Gap | P1.3, P1.4 | ☐ Pass |
| REQ-005 | Apply pregnancy-adjusted ranges | P4.1 | ☐ Pass |
| REQ-006 | Apply pediatric ranges | P4.2 | ☐ Pass |
| REQ-007 | Detect HAGMA pattern | P3.2 | ☐ Pass |
| REQ-008 | Detect pancytopenia pattern | P3.3 | ☐ Pass |
| REQ-009 | Perform delta check on Hgb | P5.1 | ☐ Pass |
| REQ-010 | Calculate trending trajectory | P5.2 | ☐ Pass |
| REQ-011 | Detect HbA1c care gap | P6.1 | ☐ Pass |
| REQ-012 | Create KB-14 task for critical | P7.1 | ☐ Pass |
| REQ-013 | Maintain audit trail | P7.2 | ☐ Pass |
| REQ-014 | Response time <200ms | P8.1 | ☐ Pass |
| REQ-015 | Handle specimen quality issues | P9.1, P9.2 | ☐ Pass |

---

## 6. DESIGN HISTORY FILE (DHF) CONTENTS

### 6.1 Document Index

| Document ID | Title | Version | Status |
|-------------|-------|---------|--------|
| KB16-SRS-001 | Software Requirements Specification | 1.0 | ☐ Approved |
| KB16-SAD-001 | Software Architecture Document | 1.0 | ☐ Approved |
| KB16-SDD-001 | Software Detailed Design | 1.0 | ☐ Approved |
| KB16-STP-001 | Software Test Plan | 1.0 | ☐ Approved |
| KB16-STR-001 | Software Test Report | 1.0 | ☐ Approved |
| KB16-RMF-001 | Risk Management File | 1.0 | ☐ Approved |
| KB16-CAT-001 | CMO Clinical Acceptance | 1.0 | ☐ Approved |
| KB16-SRN-001 | Software Release Notes | 1.0 | ☐ Approved |

### 6.2 Change Control

| Change ID | Description | Impact Assessment | Approval |
|-----------|-------------|-------------------|----------|
| | | | |

---

## 7. POST-MARKET SURVEILLANCE PLAN

### 7.1 Monitoring Activities

| Activity | Frequency | Responsible |
|----------|-----------|-------------|
| Error log review | Daily | DevOps |
| Performance metrics | Weekly | Engineering |
| Clinical feedback review | Monthly | Clinical Affairs |
| Adverse event investigation | As reported | Quality |
| Reference range updates | Annually | Clinical |

### 7.2 Key Performance Indicators

| KPI | Target | Measurement |
|-----|--------|-------------|
| Critical value sensitivity | >99.5% | Monthly audit |
| False positive rate | <5% | Monthly audit |
| System availability | >99.9% | Automated monitoring |
| Response time P95 | <200ms | Automated monitoring |
| User reported issues | <5/month | Help desk tracking |

### 7.3 Adverse Event Reporting

| Event Type | Reporting Timeline | Regulatory Notification |
|------------|-------------------|------------------------|
| Death or serious injury | Within 24 hours | FDA MDR, MHRA, TGA |
| Malfunction with potential harm | Within 30 days | FDA MDR |
| Near-miss events | Quarterly summary | Internal review |

---

## 8. REGULATORY SUBMISSIONS

### 8.1 US FDA 510(k) Evidence

| Section | Evidence |
|---------|----------|
| Device Description | Section 1, 2 of this document |
| Substantial Equivalence | [Predicate device comparison] |
| Non-clinical Testing | Sections 3, 5 of this document |
| Clinical Evidence | CMO Clinical Acceptance Sheet |
| Labeling | USAGE.md, API documentation |

### 8.2 EU MDR Technical Documentation

| Annex | Content | Reference |
|-------|---------|-----------|
| Annex II | Technical documentation | This document |
| Annex III | Post-market surveillance | Section 7 |
| Annex XIV | Clinical evaluation | CMO Acceptance |

### 8.3 Applicable Standards Compliance

| Standard | Requirement | Evidence |
|----------|-------------|----------|
| IEC 62304 | Software lifecycle | Section 4 |
| ISO 14971 | Risk management | Section 3 |
| IEC 62366 | Usability | [Usability study] |
| ISO 13485 | QMS | [QMS certificate] |
| 21 CFR Part 820 | Design controls | DHF |

---

## 9. APPROVAL SIGNATURES

### Design Review Approval

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Software Lead | _________________ | _________________ | _________ |
| QA Manager | _________________ | _________________ | _________ |
| Regulatory Affairs | _________________ | _________________ | _________ |
| Clinical Lead | _________________ | _________________ | _________ |

### Release Approval

| Role | Name | Signature | Date |
|------|------|-----------|------|
| VP Engineering | _________________ | _________________ | _________ |
| VP Quality | _________________ | _________________ | _________ |
| CMO | _________________ | _________________ | _________ |
| CEO | _________________ | _________________ | _________ |

---

## APPENDIX A: Validated Laboratory Tests

| LOINC Code | Test Name | Specimen | Reference |
|------------|-----------|----------|-----------|
| 2823-3 | Potassium | Serum/Plasma | Validated |
| 2951-2 | Sodium | Serum/Plasma | Validated |
| 2075-0 | Chloride | Serum/Plasma | Validated |
| 1963-8 | CO2 (Bicarbonate) | Serum/Plasma | Validated |
| 3094-0 | BUN | Serum/Plasma | Validated |
| 2160-0 | Creatinine | Serum/Plasma | Validated |
| 2345-7 | Glucose | Serum/Plasma | Validated |
| 718-7 | Hemoglobin | Whole Blood | Validated |
| 777-3 | Platelets | Whole Blood | Validated |
| 6690-2 | WBC | Whole Blood | Validated |
| 789-8 | RBC | Whole Blood | Validated |
| 4544-3 | Hematocrit | Whole Blood | Validated |
| 787-2 | MCV | Whole Blood | Validated |
| 1920-8 | AST | Serum/Plasma | Validated |
| 1742-6 | ALT | Serum/Plasma | Validated |
| 6768-6 | ALP | Serum/Plasma | Validated |
| 1975-2 | Total Bilirubin | Serum/Plasma | Validated |
| 1751-7 | Albumin | Serum/Plasma | Validated |
| 2093-3 | Total Cholesterol | Serum/Plasma | Validated |
| 2571-8 | Triglycerides | Serum/Plasma | Validated |
| 2085-9 | HDL | Serum/Plasma | Validated |
| 13457-7 | LDL (calculated) | Serum/Plasma | Validated |
| 3016-3 | TSH | Serum/Plasma | Validated |
| 3026-2 | Free T4 | Serum/Plasma | Validated |
| 10839-9 | Troponin I | Serum/Plasma | Validated |
| 34714-6 | INR | Plasma | Validated |
| 2524-7 | Lactate | Whole Blood/Plasma | Validated |
| 17861-6 | Calcium | Serum/Plasma | Validated |
| 2601-3 | Magnesium | Serum/Plasma | Validated |
| 2777-1 | Phosphorus | Serum/Plasma | Validated |

---

## APPENDIX B: Test Report Attachments

☐ Newman HTML Report (kb16-report-[timestamp].html)
☐ Newman JSON Results (kb16-results-[timestamp].json)
☐ JUnit XML Results (kb16-junit-[timestamp].xml)
☐ Go Test Coverage Report
☐ Performance Test Results
☐ Security Scan Results

---

## APPENDIX C: Reference Standards

1. **CAP Laboratory Accreditation Program** - Critical value thresholds
2. **CKD-EPI Collaboration** - eGFR calculation formula (2021)
3. **KDIGO Guidelines** - CKD staging and AKI definitions
4. **ACOG Guidelines** - Pregnancy-related reference ranges
5. **Nelson Textbook of Pediatrics** - Pediatric reference ranges
6. **ADA Standards of Care** - Diabetes monitoring intervals
7. **CHEST Guidelines** - Anticoagulation monitoring

---

**Document Control**

| Field | Value |
|-------|-------|
| Document Owner | Quality Assurance |
| Review Cycle | Annual |
| Retention Period | 10 years post-EOL |
| Distribution | Controlled |
| Storage Location | [Document Management System] |

---

*This document is prepared in accordance with 21 CFR Part 820 (Quality System Regulation), IEC 62304 (Medical Device Software Lifecycle), and ISO 14971 (Risk Management for Medical Devices).*
