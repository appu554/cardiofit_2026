# Complete Clinical Knowledge Base Architecture

## Executive Summary

This document provides the authoritative architecture for the Clinical Decision Support System (CDSS) comprising 19 Knowledge Bases (KBs), a 6-tier CQL library stack, and the Vaidshala clinical runtime platform.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CLINICAL KNOWLEDGE PLATFORM                              │
│                                                                             │
│                         "From Data to Decision"                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     PRESENTATION LAYER                              │   │
│  │   CDSS UI │ AI Scribe │ CDI Engine │ Conditions Advisor │ EHR      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      CDS INTERFACE LAYER                            │   │
│  │     CDS Hooks (real-time) │ $evaluate-measure │ $care-gaps          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    APPLICATION LAYER (Tier 6)                       │   │
│  │                                                                      │   │
│  │   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐           │   │
│  │   │   KB-19      │   │   Vaidshala  │   │   KB-12      │           │   │
│  │   │  Protocol    │◄─►│  CQL Engine  │◄─►│  OrderSets   │           │   │
│  │   │  Orchestrator│   │   Runtime    │   │  CarePlans   │           │   │
│  │   └──────────────┘   └──────────────┘   └──────────────┘           │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    KNOWLEDGE SERVICES (19 KBs)                      │   │
│  │                                                                      │   │
│  │   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐     │   │
│  │   │  KB-1   │ │  KB-2   │ │  KB-3   │ │  KB-4   │ │  KB-5   │     │   │
│  │   │ Patient │ │Diagnosis│ │Temporal │ │ Safety  │ │ NLP/    │     │   │
│  │   │ Profile │ │ Engine  │ │ Service │ │ Checks  │ │ Scribe  │     │   │
│  │   └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘     │   │
│  │   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐     │   │
│  │   │  KB-6   │ │  KB-7   │ │  KB-8   │ │  KB-9   │ │  KB-10  │     │   │
│  │   │Specialty│ │Chronic  │ │Clinical │ │ FHIR    │ │ Rules   │     │   │
│  │   │Formulary│ │ Care    │ │ Calcs   │ │ Mapper  │ │ Engine  │     │   │
│  │   └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘     │   │
│  │   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐     │   │
│  │   │  KB-11  │ │  KB-12  │ │  KB-13  │ │  KB-14  │ │  KB-15  │     │   │
│  │   │ Prior   │ │OrderSet │ │Quality  │ │Ontology │ │Evidence │     │   │
│  │   │  Auth   │ │CarePlan │ │Measures │ │ Engine  │ │ Engine  │     │   │
│  │   └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘     │   │
│  │   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐                 │   │
│  │   │  KB-16  │ │  KB-17  │ │  KB-18  │ │  KB-19  │                 │   │
│  │   │   Lab   │ │Wellness │ │Payer/   │ │Protocol │                 │   │
│  │   │Interpret│ │ Engine  │ │ Policy  │ │ Engine  │                 │   │
│  │   └─────────┘ └─────────┘ └─────────┘ └─────────┘                 │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    CQL LIBRARY STACK (6 Tiers)                      │   │
│  │                                                                      │   │
│  │   Tier 0: FHIRHelpers, FHIRCommon          [Foundation]             │   │
│  │   Tier 1: QICorePatterns, StatusCQL        [Primitives] ✓           │   │
│  │   Tier 2: CQMCommon, Hospice               [CQM Infrastructure]     │   │
│  │   Tier 3: ClinicalCalculators              [Domain Logic] ✓         │   │
│  │   Tier 4: ClinicalGuidelines               [Knowledge] ✓ NEW        │   │
│  │   Tier 5: IndiaTermAdapter                 [Localization] ✓         │   │
│  │   Tier 6: CDSSOrchestrator                 [Application]            │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         DATA LAYER                                   │   │
│  │                                                                      │   │
│  │   FHIR Server │ Terminology Server │ Clinical Results Store         │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## All 19 Knowledge Bases

### KB Matrix by Category

| Category | KBs | Focus |
|----------|-----|-------|
| **Patient Context** | KB-1, KB-7, KB-17 | Profile, chronic care, wellness |
| **Clinical Logic** | KB-2, KB-3, KB-8, KB-16 | Diagnosis, temporal, calculators, labs |
| **Safety** | KB-4, KB-6 | Drug interactions, formulary |
| **Integration** | KB-5, KB-9, KB-10 | NLP, FHIR mapping, rules |
| **Workflow** | KB-11, KB-12, KB-18 | Prior auth, ordersets, payer |
| **Quality** | KB-13, KB-15 | Measures, evidence |
| **Knowledge** | KB-14, KB-19 | Ontology, protocols |

---

### Detailed KB Specifications

#### KB-1: Patient Profile Service
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-1: PATIENT PROFILE SERVICE                                               │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Unified patient context for all clinical decisions                 │
│                                                                             │
│ Inputs:                          │ Outputs:                                │
│ • Demographics (FHIR Patient)    │ • Risk stratification                   │
│ • Problem list                   │ • Active medications summary            │
│ • Medication list                │ • Allergy alerts                        │
│ • Allergies                      │ • Care gaps                             │
│ • Lab results                    │ • Comorbidity index                     │
│ • Vitals                         │                                         │
│                                                                             │
│ Key Functions:                                                              │
│ • BuildPatientContext(patientId) → PatientContext                          │
│ • GetRiskFactors(patientId) → []RiskFactor                                 │
│ • GetActiveProblems(patientId) → []Condition                               │
│                                                                             │
│ Consumers: All other KBs, CQL Engine                                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-2: Diagnosis Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-2: DIAGNOSIS ENGINE                                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Differential diagnosis and diagnostic workup recommendations       │
│                                                                             │
│ Inputs:                          │ Outputs:                                │
│ • Chief complaint                │ • Differential diagnosis list          │
│ • Symptoms                       │ • Recommended workup                    │
│ • Physical exam findings         │ • Red flags                             │
│ • Initial labs/imaging           │ • Urgency classification               │
│                                                                             │
│ Key Functions:                                                              │
│ • GenerateDifferential(symptoms) → []DiagnosisHypothesis                   │
│ • RecommendWorkup(differential) → []DiagnosticOrder                        │
│ • IdentifyRedFlags(presentation) → []RedFlag                               │
│                                                                             │
│ Algorithms: Bayesian inference, symptom clustering                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-3: Clinical Temporal Service
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-3: CLINICAL TEMPORAL SERVICE                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Track WHEN things are due, manage deadlines, schedule alerts       │
│                                                                             │
│ Focus: WHEN / SEQUENCE (not WHAT to do - that's KB-19)                     │
│                                                                             │
│ Inputs:                          │ Outputs:                                │
│ • Protocol definitions           │ • Due dates                             │
│ • Patient events                 │ • Overdue alerts                        │
│ • Time constraints               │ • Schedule recommendations              │
│                                                                             │
│ Key Functions:                                                              │
│ • CalculateDueDate(event, constraint) → DateTime                           │
│ • CheckDeadline(patientId, protocol) → DeadlineStatus                      │
│ • GetOverdueItems(patientId) → []OverdueItem                               │
│                                                                             │
│ Examples:                                                                   │
│ • Door-to-balloon < 90 minutes (STEMI)                                     │
│ • HbA1c every 3-6 months (Diabetes)                                        │
│ • Colonoscopy every 10 years (CRC screening)                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-4: Patient Safety Service
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-4: PATIENT SAFETY SERVICE                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Drug-drug interactions, allergy checks, contraindications          │
│                                                                             │
│ Inputs:                          │ Outputs:                                │
│ • Current medications            │ • Interaction alerts                    │
│ • New prescription               │ • Severity levels                       │
│ • Allergies                      │ • Alternative suggestions               │
│ • Diagnoses                      │ • Documentation                         │
│                                                                             │
│ Key Functions:                                                              │
│ • CheckInteractions(meds) → []Interaction                                  │
│ • CheckAllergies(drug, allergies) → []AllergyAlert                         │
│ • CheckContraindications(drug, conditions) → []Contraindication            │
│                                                                             │
│ Integration: Vaidshala medication-advisor-engine                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-5: Clinical NLP / AI Scribe
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-5: CLINICAL NLP / AI SCRIBE                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Extract structured data from clinical narratives                   │
│                                                                             │
│ Inputs:                          │ Outputs:                                │
│ • Clinical notes                 │ • Extracted entities (ICD-10, RxNorm)  │
│ • Dictation audio                │ • Structured FHIR resources            │
│ • Discharge summaries            │ • Suggested codes                       │
│                                                                             │
│ Key Functions:                                                              │
│ • ExtractEntities(text) → []ClinicalEntity                                 │
│ • GenerateNote(encounter) → ClinicalNote                                   │
│ • SuggestCodes(note) → []DiagnosisCode                                     │
│                                                                             │
│ Models: NER, relation extraction, code suggestion                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-6: Specialty Formulary Service
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-6: SPECIALTY FORMULARY SERVICE                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Drug formulary, tier checking, therapeutic alternatives            │
│                                                                             │
│ Inputs:                          │ Outputs:                                │
│ • Drug request                   │ • Formulary status                      │
│ • Insurance plan                 │ • Tier level                            │
│ • Patient coverage               │ • Prior auth requirement                │
│                                  │ • Therapeutic alternatives              │
│                                                                             │
│ Key Functions:                                                              │
│ • CheckFormulary(drug, plan) → FormularyStatus                             │
│ • GetAlternatives(drug, plan) → []TherapeuticAlternative                   │
│ • GetCopay(drug, plan) → CopayInfo                                         │
│                                                                             │
│ Data: NLEM (India), commercial formularies                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-7: Chronic Care Management
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-7: CHRONIC CARE MANAGEMENT                                               │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Disease-specific chronic condition management pathways             │
│                                                                             │
│ Conditions Covered:              │ Functions:                              │
│ • Diabetes (T1DM, T2DM)          │ • Care plan generation                  │
│ • Hypertension                   │ • Goal tracking                         │
│ • Heart Failure                  │ • Medication optimization               │
│ • CKD                            │ • Complication screening                │
│ • COPD                           │ • Patient education                     │
│ • Asthma                         │                                         │
│                                                                             │
│ Integration: KB-13 (quality measures), KB-19 (protocols)                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-8: Clinical Calculator Service
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-8: CLINICAL CALCULATOR SERVICE                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Validated clinical scoring and risk calculations                   │
│                                                                             │
│ Calculators Implemented:                                                    │
│ ┌─────────────────────────────────────────────────────────────────────┐    │
│ │ Category     │ Calculator      │ Use Case                          │    │
│ ├─────────────────────────────────────────────────────────────────────┤    │
│ │ Renal        │ eGFR CKD-EPI    │ Kidney function, drug dosing      │    │
│ │ Cardiac      │ CHA₂DS₂-VASc    │ Stroke risk in AFib               │    │
│ │ Cardiac      │ ASCVD 10-year   │ Statin eligibility                │    │
│ │ Sepsis       │ qSOFA           │ Bedside sepsis screen             │    │
│ │ Sepsis       │ SOFA            │ ICU mortality                     │    │
│ │ VTE          │ Padua           │ Medical VTE risk                  │    │
│ │ VTE          │ Caprini         │ Surgical VTE risk                 │    │
│ │ ACS          │ TIMI            │ NSTEMI risk stratification        │    │
│ │ General      │ BMI             │ With India-adjusted categories    │    │
│ │ Labs         │ Corrected Ca    │ Adjusts for albumin               │    │
│ │ Labs         │ Anion Gap       │ Metabolic acidosis                │    │
│ └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│ SaMD Compliance: FDA 21 CFR Part 11, CE Mark ready                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-9: FHIR Interoperability Mapper
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-9: FHIR INTEROPERABILITY MAPPER                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Transform between FHIR versions, legacy formats, and standards     │
│                                                                             │
│ Mappings:                                                                   │
│ • FHIR R4 ↔ FHIR R5                                                        │
│ • HL7 v2 → FHIR                                                            │
│ • CDA → FHIR                                                                │
│ • Custom EHR → FHIR                                                         │
│                                                                             │
│ Key Functions:                                                              │
│ • TransformResource(resource, targetVersion) → Resource                    │
│ • MapHL7v2ToFHIR(message) → Bundle                                         │
│ • ValidateProfile(resource, profile) → ValidationResult                    │
│                                                                             │
│ Profiles: US Core, QI-Core, India NDHM                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-10: Clinical Rules Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-10: CLINICAL RULES ENGINE                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Execute configurable clinical rules and decision logic             │
│                                                                             │
│ Rule Types:                                                                 │
│ • Alert rules (high potassium → notify)                                    │
│ • Inference rules (A + B → suggest C)                                      │
│ • Validation rules (order requires indication)                             │
│ • Escalation rules (critical value → page)                                 │
│                                                                             │
│ Key Functions:                                                              │
│ • EvaluateRules(context, ruleSet) → []RuleResult                           │
│ • AddRule(rule) → RuleId                                                   │
│ • GetActiveAlerts(patientId) → []Alert                                     │
│                                                                             │
│ Format: YAML/TOML rule definitions, hot-reloadable                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-11: Prior Authorization Service
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-11: PRIOR AUTHORIZATION SERVICE                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Automate prior authorization determination and submission          │
│                                                                             │
│ Inputs:                          │ Outputs:                                │
│ • Order (drug/procedure)         │ • PA required (yes/no)                 │
│ • Clinical documentation         │ • Auto-approval if criteria met        │
│ • Payer rules                    │ • Required documentation list          │
│                                  │ • X12 278 submission                   │
│                                                                             │
│ Key Functions:                                                              │
│ • CheckPARequired(order, payer) → PARequirement                            │
│ • EvaluateCriteria(order, docs) → ApprovalStatus                           │
│ • SubmitPA(request) → PAResponse                                           │
│                                                                             │
│ Standards: Da Vinci CRD, X12 278, NCPDP                                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-12: OrderSets & Care Plans
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-12: ORDERSETS & CARE PLANS                                               │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Condition-specific order bundles and longitudinal care plans       │
│                                                                             │
│ OrderSet Categories:                                                        │
│ ┌─────────────────────────────────────────────────────────────────────┐    │
│ │ Admission       │ CHF, STEMI, Stroke, Sepsis, DKA, Pneumonia       │    │
│ │ Emergency       │ Code Blue, Anaphylaxis, Trauma, RRT              │    │
│ │ Perioperative   │ Pre-op, Post-op by surgery type                  │    │
│ │ Discharge       │ Condition-specific discharge bundles             │    │
│ └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│ Care Plan Types:                                                            │
│ • Chronic disease management                                                │
│ • Post-hospitalization                                                      │
│ • Preventive care                                                           │
│                                                                             │
│ Key Functions:                                                              │
│ • GetOrderSet(condition, setting) → OrderSet                               │
│ • ActivateOrders(orderSet, patient) → []Order                              │
│ • GenerateCarePlan(patient, conditions) → CarePlan                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-13: Quality Measures Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-13: QUALITY MEASURES ENGINE                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Calculate quality measures, identify care gaps                     │
│                                                                             │
│ Measure Sets:                                                               │
│ • CMS eCQM (MIPS, Hospital IQR)                                            │
│ • HEDIS                                                                     │
│ • India NQAS                                                                │
│ • Custom measures                                                           │
│                                                                             │
│ Key Measures:                                                               │
│ ┌─────────────────────────────────────────────────────────────────────┐    │
│ │ CMS122 │ Diabetes: Hemoglobin A1c Poor Control                     │    │
│ │ CMS165 │ Controlling High Blood Pressure                           │    │
│ │ CMS134 │ Diabetes: Medical Attention for Nephropathy               │    │
│ │ CMS347 │ Statin Therapy for Prevention of CV Disease               │    │
│ │ CMS139 │ Falls: Screening for Future Fall Risk                     │    │
│ └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│ Key Functions:                                                              │
│ • EvaluateMeasure(patientId, measureId) → MeasureResult                    │
│ • GetCareGaps(patientId) → []CareGap                                       │
│ • CalculatePopulationMetrics(population, measure) → PopulationResult       │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-14: Clinical Ontology Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-14: CLINICAL ONTOLOGY ENGINE                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Semantic reasoning over clinical concepts and relationships        │
│                                                                             │
│ Ontologies:                                                                 │
│ • SNOMED CT (concepts, relationships)                                       │
│ • ICD-10-CM/PCS (diagnosis, procedures)                                     │
│ • RxNorm (medications)                                                      │
│ • LOINC (labs, observations)                                                │
│                                                                             │
│ Key Functions:                                                              │
│ • GetParentConcepts(code) → []Concept                                      │
│ • GetChildConcepts(code) → []Concept                                       │
│ • IsSubtypeOf(code, parentCode) → bool                                     │
│ • FindRelatedConcepts(code, relationship) → []Concept                      │
│                                                                             │
│ Use Cases:                                                                  │
│ • Code grouping for measures                                                │
│ • Semantic search                                                           │
│ • Concept expansion for queries                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-15: Evidence Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-15: EVIDENCE ENGINE                                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Clinical evidence retrieval, grading, and citation                 │
│                                                                             │
│ Evidence Sources:                                                           │
│ • PubMed / MEDLINE                                                          │
│ • Cochrane Reviews                                                          │
│ • Clinical practice guidelines                                              │
│ • UpToDate / DynaMed                                                        │
│                                                                             │
│ Evidence Grading:                                                           │
│ ┌─────────────────────────────────────────────────────────────────────┐    │
│ │ System         │ Levels                                            │    │
│ ├─────────────────────────────────────────────────────────────────────┤    │
│ │ GRADE          │ HIGH, MODERATE, LOW, VERY_LOW                     │    │
│ │ ACC/AHA        │ Class I, IIa, IIb, III (Benefit/Harm)             │    │
│ │ Oxford CEBM    │ 1a, 1b, 2a, 2b, 3a, 3b, 4, 5                      │    │
│ └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│ Key Functions:                                                              │
│ • SearchEvidence(query) → []EvidenceResult                                 │
│ • GetCitations(recommendation) → []Citation                                │
│ • GradeEvidence(study) → EvidenceGrade                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-16: Lab Interpretation Service
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-16: LAB INTERPRETATION SERVICE                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Context-aware lab result interpretation and trending               │
│                                                                             │
│ Capabilities:                                                               │
│ • Age/sex-specific reference ranges                                         │
│ • Condition-adjusted interpretation                                         │
│ • Trend analysis (improving/worsening)                                      │
│ • Panel interpretation (CBC, CMP, LFTs)                                     │
│ • Critical value alerting                                                   │
│                                                                             │
│ Key Functions:                                                              │
│ • InterpretResult(lab, context) → Interpretation                           │
│ • GetTrend(patientId, labCode, period) → TrendAnalysis                     │
│ • InterpretPanel(panelResults, context) → PanelInterpretation              │
│                                                                             │
│ Examples:                                                                   │
│ • Creatinine 1.5 + known CKD Stage 3 → "Stable at baseline"                │
│ • Creatinine 1.5 + new onset → "Acute kidney injury"                       │
│ • HbA1c 7.2 + on metformin → "Above ADA target, consider intensification"  │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-17: Wellness & Prevention Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-17: WELLNESS & PREVENTION ENGINE                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Preventive care recommendations based on demographics and risk     │
│                                                                             │
│ Screening Guidelines:                                                       │
│ ┌─────────────────────────────────────────────────────────────────────┐    │
│ │ Cancer       │ Colorectal, Breast, Cervical, Lung, Prostate        │    │
│ │ Cardiovascular │ Lipids, BP, Diabetes, AAA                         │    │
│ │ Infectious   │ HIV, Hepatitis, STIs, TB                            │    │
│ │ Behavioral   │ Depression, Alcohol, Tobacco                        │    │
│ │ Pediatric    │ Developmental, Vision, Hearing, Lead               │    │
│ └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│ Sources: USPSTF, ACIP, AAP, India NHM                                      │
│                                                                             │
│ Key Functions:                                                              │
│ • GetScreeningRecommendations(patient) → []ScreeningRec                    │
│ • GetImmunizationSchedule(patient) → []Immunization                        │
│ • CalculateWellnessScore(patient) → WellnessScore                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-18: Payer Policy Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-18: PAYER POLICY ENGINE                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Payer-specific coverage rules and medical policies                 │
│                                                                             │
│ Policy Types:                                                               │
│ • Medical necessity criteria                                                │
│ • Step therapy requirements                                                 │
│ • Quantity limits                                                           │
│ • Age/gender restrictions                                                   │
│ • Site of service requirements                                              │
│                                                                             │
│ Key Functions:                                                              │
│ • CheckCoverage(service, payer) → CoverageResult                           │
│ • GetMedicalPolicy(service, payer) → MedicalPolicy                         │
│ • EvaluateMedicalNecessity(service, docs, payer) → Determination           │
│                                                                             │
│ Integration: KB-11 (prior auth), KB-6 (formulary)                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### KB-19: Clinical Protocol Execution Engine
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ KB-19: CLINICAL PROTOCOL EXECUTION ENGINE                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ Purpose: Evidence-based guideline execution with recommendations            │
│                                                                             │
│ Focus: WHAT to do / WHY (vs KB-3 which handles WHEN)                       │
│                                                                             │
│ Protocols Implemented:                                                      │
│ ┌─────────────────────────────────────────────────────────────────────┐    │
│ │ Cardiac     │ HF GDMT, ACS/STEMI, AFib Anticoagulation             │    │
│ │ Metabolic   │ Type 2 Diabetes, Hypertension, CKD                   │    │
│ │ Acute Care  │ Sepsis, VTE Prophylaxis, COPD/Asthma Exacerbation   │    │
│ └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│ Features:                                                                   │
│ • Decision tree execution                                                   │
│ • ACC/AHA Class I-III recommendation grading                               │
│ • GRADE evidence levels (HIGH/MODERATE/LOW)                                │
│ • Contraindication checking                                                 │
│ • Multi-protocol execution for complex patients                            │
│ • Audit logging                                                             │
│                                                                             │
│ Key Functions:                                                              │
│ • ExecuteProtocol(protocolId, patientContext) → Recommendations            │
│ • ExecuteAllApplicable(patientContext) → MultiProtocolResult               │
│ • CheckApplicability(patientContext) → []ApplicabilityResult               │
│                                                                             │
│ Integration: Calls Vaidshala for CQL truth (Tier 4)                        │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## CQL Library Stack (6 Tiers)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CQL LIBRARY LOAD ORDER                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TIER 0: FHIR FOUNDATION                           [░░░░░░░░] Stubs │   │
│  │                                                                      │   │
│  │   FHIRHelpers     → FHIR type conversions                           │   │
│  │   FHIRCommon      → Common FHIR patterns                            │   │
│  │                                                                      │   │
│  │   Status: Need to download from CMS eCQI                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TIER 1: CQL PRIMITIVES                            [██████████] ✓   │   │
│  │                                                                      │   │
│  │   QICorePatterns  → Fluent FHIR accessors (the Fluency Patch)       │   │
│  │   StatusCQL       → Workflow status helpers                         │   │
│  │                                                                      │   │
│  │   // One line, chainable, null-safe                                 │   │
│  │   define "Diabetes Onset":                                          │   │
│  │     Condition.onset.toInterval()                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TIER 2: CQM INFRASTRUCTURE                        [░░░░░░░░] Stubs │   │
│  │                                                                      │   │
│  │   CQMCommon       → Measure calculation patterns                    │   │
│  │   Hospice         → Hospice exclusion logic                         │   │
│  │   CumulativeMedDur → Medication duration calculations               │   │
│  │                                                                      │   │
│  │   Status: Need to download from CMS eCQI                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TIER 3: DOMAIN COMMONS                            [██████████] ✓   │   │
│  │                                                                      │   │
│  │   ClinicalCalculators → KB-8 as CQL functions                       │   │
│  │   DiabetesCommon      → Shared diabetes logic                       │   │
│  │   CardiovascularCommon → Shared CV logic                            │   │
│  │                                                                      │   │
│  │   define "eGFR": Calc.eGFR_CKD_EPI(age, sex, creatinine, race)      │   │
│  │   define "CHA2DS2VASc": Calc.CHA2DS2VASc(...)                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TIER 4: MEASURES & GUIDELINES                     [██████████] ✓   │   │
│  │                                                                      │   │
│  │   ClinicalGuidelines → ACC/AHA, ADA, KDIGO, SSC protocols           │   │
│  │   CMS122            → Diabetes HbA1c Poor Control                   │   │
│  │   CMS165            → Controlling High Blood Pressure               │   │
│  │   CMS134            → Diabetes Nephropathy                          │   │
│  │                                                                      │   │
│  │   define "Has HFrEF": "Has HF" and "LVEF" <= 40                     │   │
│  │   define "Missing GDMT Pillars": [...]                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TIER 5: INDIA ADAPTATION                          [██████████] ✓   │   │
│  │                                                                      │   │
│  │   IndiaTermAdapter → ICD-10 India, NLEM, NDHM                       │   │
│  │   IndiaDrugAdapter → RxNorm + NLEM + Brand Names                    │   │
│  │   IndiaLabAdapter  → LOINC + India Local codes                      │   │
│  │                                                                      │   │
│  │   ┌─────────────────────────────────────────────────────────────┐   │   │
│  │   │ Metric        │ US Value    │ India Value                  │   │   │
│  │   │ BMI Overweight│ ≥25         │ ≥23 (WHO Asia-Pacific)       │   │   │
│  │   │ BMI Obese     │ ≥30         │ ≥25                          │   │   │
│  │   │ Waist (M)     │ 102 cm      │ 90 cm                        │   │   │
│  │   │ Waist (F)     │ 88 cm       │ 80 cm                        │   │   │
│  │   └─────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TIER 6: APPLICATION ENGINES                       [░░░░░░░░] Stubs │   │
│  │                                                                      │   │
│  │   CDSSOrchestrator   → Coordinates all KBs                          │   │
│  │   ConditionsAdvisor  → Medication advisor                           │   │
│  │   MeasureCalculator  → Quality measure execution                    │   │
│  │                                                                      │   │
│  │   Status: Need KB engine orchestrators                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  Legend: ██████████ = Production Ready │ ░░░░░░░░ = Stub (needs content)   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## KB Interaction Patterns

### Pattern 1: Protocol Execution (KB-19 + Vaidshala)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PROTOCOL EXECUTION FLOW                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. EHR/App → KB-19: "What should I do for patient with HF?"               │
│                              │                                              │
│                              ▼                                              │
│  2. KB-19 → Vaidshala: POST /api/v1/cql/evaluate                           │
│     {                                                                       │
│       "patientId": "P001",                                                 │
│       "library": "ClinicalGuidelines",                                     │
│       "expressions": ["Complete Clinical Evaluation"]                       │
│     }                                                                       │
│                              │                                              │
│                              ▼                                              │
│  3. Vaidshala → KB-8: Calculate eGFR, CHA2DS2VASc (via CQL include)        │
│                              │                                              │
│                              ▼                                              │
│  4. Vaidshala → KB-19: CQL truth results                                   │
│     {                                                                       │
│       "hasHFrEF": true,                                                    │
│       "missingGDMTPillars": ["ARNI", "BB", "MRA", "SGLT2i"],              │
│       "eGFR": 55,                                                          │
│       "potassium": 4.2                                                     │
│     }                                                                       │
│                              │                                              │
│                              ▼                                              │
│  5. KB-19: Traverse decision tree, check contraindications                 │
│                              │                                              │
│                              ▼                                              │
│  6. KB-19 → EHR/App: Recommendations with evidence grading                 │
│     [                                                                       │
│       { "ARNI": "Class I, Level A, JACC 2022" },                           │
│       { "BB": "Class I, Level A" },                                        │
│       { "MRA": "Class I, Level A" },                                       │
│       { "SGLT2i": "Class I, Level A, DAPA-HF trial" }                      │
│     ]                                                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Pattern 2: Order Entry with Safety (KB-12 + KB-4 + KB-6)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    ORDER ENTRY SAFETY FLOW                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Clinician selects medication from order set                            │
│                              │                                              │
│                              ▼                                              │
│  2. KB-12 → KB-4: Check drug interactions                                  │
│     "Lisinopril + Spironolactone + current K+ = ?"                         │
│                              │                                              │
│                              ▼                                              │
│  3. KB-4: Returns interaction alert                                        │
│     { "severity": "HIGH", "message": "Risk of hyperkalemia" }              │
│                              │                                              │
│                              ▼                                              │
│  4. KB-12 → KB-6: Check formulary                                          │
│     "Is brand Entresto covered?"                                           │
│                              │                                              │
│                              ▼                                              │
│  5. KB-6: Returns formulary status                                         │
│     { "covered": true, "tier": 3, "priorAuth": true }                      │
│                              │                                              │
│                              ▼                                              │
│  6. KB-12 → KB-11: Initiate prior auth                                     │
│     Submit clinical documentation for PA                                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Pattern 3: Quality Measure (KB-13 + Vaidshala)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    QUALITY MEASURE FLOW                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Quality Dashboard → KB-13: Calculate CMS122 for population             │
│                              │                                              │
│                              ▼                                              │
│  2. KB-13 → Vaidshala: $evaluate-measure                                   │
│     {                                                                       │
│       "measure": "CMS122v12",                                              │
│       "periodStart": "2024-01-01",                                         │
│       "periodEnd": "2024-12-31",                                           │
│       "subject": "Group/all-diabetic-patients"                             │
│     }                                                                       │
│                              │                                              │
│                              ▼                                              │
│  3. Vaidshala executes CQL (Tier 2 + Tier 4):                              │
│     - Initial Population: Has diabetes                                      │
│     - Denominator: Age 18-75, 2+ encounters                                │
│     - Numerator: Most recent HbA1c > 9%                                    │
│     - Exclusions: Hospice, ESRD                                            │
│                              │                                              │
│                              ▼                                              │
│  4. KB-13 → Dashboard: MeasureReport                                       │
│     { "rate": 12.3%, "numerator": 123, "denominator": 1000 }              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Service Ports & Endpoints

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SERVICE REGISTRY                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Service          │ Port  │ Key Endpoints                                  │
│  ─────────────────┼───────┼────────────────────────────────────────────────│
│  KB-1  Patient    │ 8081  │ /api/v1/patients/{id}/context                 │
│  KB-2  Diagnosis  │ 8082  │ /api/v1/differential                          │
│  KB-3  Temporal   │ 8083  │ /api/v1/deadlines, /api/v1/schedule           │
│  KB-4  Safety     │ 8084  │ /api/v1/interactions, /api/v1/allergies       │
│  KB-5  NLP        │ 8085  │ /api/v1/extract, /api/v1/generate-note        │
│  KB-6  Formulary  │ 8086  │ /api/v1/formulary, /api/v1/alternatives       │
│  KB-7  Chronic    │ 8087  │ /api/v1/careplan, /api/v1/goals               │
│  KB-8  Calculators│ 8088  │ /api/v1/calculate/{calculator}                │
│  KB-9  FHIR       │ 8089  │ /api/v1/transform, /api/v1/validate           │
│  KB-10 Rules      │ 8090  │ /api/v1/evaluate, /api/v1/alerts              │
│  KB-11 PriorAuth  │ 8091  │ /api/v1/check-pa, /api/v1/submit              │
│  KB-12 OrderSets  │ 8092  │ /api/v1/ordersets, /api/v1/careplans          │
│  KB-13 Quality    │ 8093  │ /api/v1/measure, /api/v1/care-gaps            │
│  KB-14 Ontology   │ 8094  │ /api/v1/concepts, /api/v1/relationships       │
│  KB-15 Evidence   │ 8095  │ /api/v1/search, /api/v1/citations             │
│  KB-16 Labs       │ 8096  │ /api/v1/interpret, /api/v1/trends             │
│  KB-17 Wellness   │ 8097  │ /api/v1/screenings, /api/v1/immunizations     │
│  KB-18 Payer      │ 8098  │ /api/v1/coverage, /api/v1/policy              │
│  KB-19 Protocols  │ 8099  │ /api/v1/execute, /api/v1/protocols            │
│  ─────────────────┼───────┼────────────────────────────────────────────────│
│  Vaidshala        │ 8096  │ /api/v1/cql/evaluate, /api/v1/measure         │
│  FHIR Server      │ 8080  │ /{Resource}, /$operation                      │
│  Terminology      │ 8180  │ /CodeSystem, /ValueSet/$expand                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Data Flow Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         COMPLETE DATA FLOW                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                            ┌───────────┐                                   │
│                            │    EHR    │                                   │
│                            │    App    │                                   │
│                            └─────┬─────┘                                   │
│                                  │                                          │
│                  ┌───────────────┼───────────────┐                         │
│                  │               │               │                          │
│                  ▼               ▼               ▼                          │
│           ┌──────────┐   ┌──────────┐   ┌──────────┐                       │
│           │ CDS Hooks│   │ SMART    │   │ Custom   │                       │
│           │ (hook)   │   │ on FHIR  │   │   API    │                       │
│           └────┬─────┘   └────┬─────┘   └────┬─────┘                       │
│                │              │              │                              │
│                └──────────────┼──────────────┘                             │
│                               │                                             │
│                               ▼                                             │
│                    ┌──────────────────────┐                                │
│                    │       KB-19          │                                │
│                    │   Protocol Engine    │                                │
│                    │    (Orchestrator)    │                                │
│                    └──────────┬───────────┘                                │
│                               │                                             │
│           ┌───────────────────┼───────────────────┐                        │
│           │                   │                   │                         │
│           ▼                   ▼                   ▼                         │
│    ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                  │
│    │  Vaidshala   │   │    KB-8      │   │   KB-12      │                  │
│    │ CQL Engine   │   │ Calculators  │   │  OrderSets   │                  │
│    │  (Tier 4)    │   │              │   │              │                  │
│    └──────┬───────┘   └──────────────┘   └──────┬───────┘                  │
│           │                                      │                          │
│           │                                      ▼                          │
│           │                              ┌──────────────┐                  │
│           │                              │    KB-4      │                  │
│           │                              │   Safety     │                  │
│           │                              └──────────────┘                  │
│           │                                                                 │
│           ▼                                                                 │
│    ┌──────────────────────────────────────────────────────────────────┐    │
│    │                       FHIR SERVER                                 │    │
│    │                                                                   │    │
│    │  Patient │ Condition │ MedicationRequest │ Observation │ ...     │    │
│    │                                                                   │    │
│    └──────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Status

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    IMPLEMENTATION STATUS                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  KB     │ Name                  │ Status       │ Lines   │ Notes           │
│  ───────┼───────────────────────┼──────────────┼─────────┼─────────────────│
│  KB-1   │ Patient Profile       │ ⬜ Planned   │ -       │                 │
│  KB-2   │ Diagnosis Engine      │ ⬜ Planned   │ -       │                 │
│  KB-3   │ Temporal Service      │ ✅ Complete  │ 3,500+  │ In Vaidshala   │
│  KB-4   │ Patient Safety        │ ✅ Complete  │ 2,800+  │ In Vaidshala   │
│  KB-5   │ Clinical NLP          │ ⬜ Planned   │ -       │                 │
│  KB-6   │ Specialty Formulary   │ ⬜ Planned   │ -       │                 │
│  KB-7   │ Chronic Care          │ ⬜ Planned   │ -       │                 │
│  KB-8   │ Clinical Calculators  │ ✅ Complete  │ 2,200+  │ Standalone     │
│  KB-9   │ FHIR Mapper           │ ⬜ Planned   │ -       │                 │
│  KB-10  │ Rules Engine          │ ⬜ Planned   │ -       │                 │
│  KB-11  │ Prior Authorization   │ ⬜ Planned   │ -       │                 │
│  KB-12  │ OrderSets/CarePlans   │ ✅ Complete  │ 5,500+  │ Standalone     │
│  KB-13  │ Quality Measures      │ 🔶 Partial   │ 1,500+  │ In Vaidshala   │
│  KB-14  │ Ontology Engine       │ ⬜ Planned   │ -       │                 │
│  KB-15  │ Evidence Engine       │ ⬜ Planned   │ -       │                 │
│  KB-16  │ Lab Interpretation    │ ⬜ Planned   │ -       │                 │
│  KB-17  │ Wellness Engine       │ ⬜ Planned   │ -       │                 │
│  KB-18  │ Payer Policy          │ ⬜ Planned   │ -       │                 │
│  KB-19  │ Protocol Execution    │ ✅ Complete  │ 7,000+  │ Standalone     │
│  ───────┼───────────────────────┼──────────────┼─────────┼─────────────────│
│  Tier 4 │ ClinicalGuidelines    │ ✅ Complete  │ 1,900+  │ CQL Library    │
│  ───────┴───────────────────────┴──────────────┴─────────┴─────────────────│
│                                                                             │
│  Legend: ✅ Complete │ 🔶 Partial │ ⬜ Planned                              │
│                                                                             │
│  Total Implemented: ~24,400+ lines across completed services               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Key Architecture Decisions

### 1. Separation of Truth vs Action

```
CQL (Tier 4)                    KB-19
═══════════                     ═════
"Is this true?"                 "What to do about it"
Boolean/Value answers           Recommendations + evidence
Declarative                     Imperative
Reusable knowledge              Action-oriented
```

### 2. Single Calculator Service

All clinical calculations go through KB-8 to ensure:
- Consistent formulas across all KBs
- SaMD compliance (FDA 21 CFR Part 11)
- Audit trail for score calculations
- Version control for formula updates

### 3. India Localization Layer

Tier 5 provides the "shim" for India-specific:
- BMI thresholds (23/25 vs 25/30)
- Drug codes (RxNorm + NLEM)
- Lab codes (LOINC + local)
- Guidelines (NHM, ICMR)

### 4. Protocol Engine as Orchestrator

KB-19 is the "front door" for clinical decisions:
- Calls Vaidshala for CQL truth
- Calls KB-8 for calculations
- Calls KB-4 for safety checks
- Returns unified recommendations

---

## Next Steps

### Immediate (Tier 4 Completion)

1. Deploy ClinicalGuidelines.cql to Vaidshala
2. Test KB-19 → Vaidshala integration
3. Download CMS eCQM libraries (Tier 0, Tier 2)

### Short-term (More KBs)

1. KB-1 Patient Profile (foundation for all)
2. KB-13 Quality Measures (with CMS libraries)
3. KB-2 Diagnosis Engine

### Long-term (Complete Platform)

1. Remaining KBs (5, 6, 7, 9, 10, 11, 14-18)
2. CDS Hooks integration
3. AI/ML enhancement layer
4. Real-time analytics

---

*Document Version: 1.0*
*Last Updated: January 2025*
*Author: CTO/CMO Architecture Team*
