# KB-5 Drug-Drug Interactions: Authoritative Sources for KB-0

## CTO/CMO Analysis

**Date**: January 2026
**Status**: FINAL SPECIFICATION

---

## Executive Summary

Your KB-5 implementation is **functionally solid** but **governance-incomplete**.

**Current State**:
- 50+ drug-drug interactions loaded
- CYP profiles present
- Drug-food, drug-disease interactions covered
- Severity classification implemented

**Missing**:
- **No governance metadata** on each interaction
- **No source citations** (FDA Section, DrugBank ID, etc.)
- **No audit trail** for clinical defensibility

---

## The Three-Layer Authority Model

Every DDI in KB-5 must reference **three authority layers**:

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                         KB-5 THREE-LAYER AUTHORITY MODEL                                    │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                             │
│  LAYER 1: REGULATORY AUTHORITY                                                             │
│  ═════════════════════════════                                                             │
│  "What is officially prohibited or warned about?"                                          │
│                                                                                             │
│  Sources:                                                                                  │
│  • FDA Drug Labels (DailyMed SPL) - Sections 4, 5, 7, 12                                  │
│  • TGA Product Information (Australia)                                                    │
│  • CDSCO Package Inserts (India)                                                          │
│  • EMA SmPC (Europe)                                                                      │
│                                                                                             │
│  This is the LEGAL basis for the interaction.                                             │
│                                                                                             │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                             │
│  LAYER 2: PHARMACOLOGY AUTHORITY                                                           │
│  ═══════════════════════════════                                                           │
│  "Why does this interaction happen biologically?"                                          │
│                                                                                             │
│  Sources:                                                                                  │
│  • FDA Clinical Pharmacology (Label Section 12)                                           │
│  • DrugBank (CYP enzymes, transporters)                                                   │
│  • Flockhart CYP Table (Indiana University)                                               │
│  • University of Washington DDI Database                                                   │
│  • PharmGKB (Pharmacogenomics)                                                            │
│                                                                                             │
│  This is the SCIENTIFIC basis for the interaction.                                        │
│                                                                                             │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                             │
│  LAYER 3: CLINICAL PRACTICE AUTHORITY                                                      │
│  ════════════════════════════════════                                                      │
│  "How do clinicians manage this interaction?"                                              │
│                                                                                             │
│  Sources:                                                                                  │
│  • Lexicomp (US)                                                                          │
│  • Micromedex (US)                                                                        │
│  • UpToDate (US)                                                                          │
│  • Australian Medicines Handbook (AU)                                                     │
│  • Therapeutic Guidelines eTG (AU)                                                        │
│  • British National Formulary BNF (UK)                                                    │
│  • ASHP Interaction Guidelines                                                            │
│  • ACCP Antithrombotic Guidelines                                                         │
│                                                                                             │
│  This is the CLINICAL basis for management.                                               │
│                                                                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Complete Authority Catalog for KB-0

### LAYER 1: Regulatory Authorities

| Authority ID | Full Name | Jurisdiction | URL | Data Type |
|--------------|-----------|--------------|-----|-----------|
| `FDA.DailyMed` | FDA Structured Product Labeling | US | https://dailymed.nlm.nih.gov | Drug Labels |
| `FDA.Section4` | FDA Label - Contraindications | US | DailyMed Section 4 | Absolute contraindications |
| `FDA.Section5` | FDA Label - Warnings | US | DailyMed Section 5 | Boxed warnings, precautions |
| `FDA.Section7` | FDA Label - Drug Interactions | US | DailyMed Section 7 | DDI text |
| `FDA.Section12` | FDA Label - Clinical Pharmacology | US | DailyMed Section 12 | PK/PD data |
| `TGA.PI` | TGA Product Information | AU | https://www.tga.gov.au | Australian labels |
| `TGA.Section4.3` | TGA PI - Contraindications | AU | TGA PI Section 4.3 | AU contraindications |
| `TGA.Section4.5` | TGA PI - Interactions | AU | TGA PI Section 4.5 | AU DDI text |
| `CDSCO.Insert` | CDSCO Package Insert | IN | https://cdsco.gov.in | Indian labels |
| `EMA.SmPC` | EMA Summary of Product Characteristics | EU | https://www.ema.europa.eu | EU labels |
| `EMA.Section4.3` | SmPC - Contraindications | EU | SmPC Section 4.3 | EU contraindications |
| `EMA.Section4.5` | SmPC - Interactions | EU | SmPC Section 4.5 | EU DDI text |
| `MHRA.SmPC` | MHRA UK SmPC | UK | https://www.gov.uk/medicines | UK labels |

### LAYER 2: Pharmacology Authorities

| Authority ID | Full Name | Scope | URL | Data Type |
|--------------|-----------|-------|-----|-----------|
| `DrugBank` | DrugBank Database | Global | https://go.drugbank.com | CYP, transporters, targets |
| `Flockhart.CYP` | Indiana University Flockhart CYP Table | Global | https://drug-interactions.medicine.iu.edu | CYP substrate/inhibitor/inducer |
| `UW.DDI` | University of Washington DDI Database | Global | https://www.druginteractioninfo.org | Transporter-based DDI |
| `PharmGKB` | Pharmacogenomics Knowledge Base | Global | https://www.pharmgkb.org | Gene-drug interactions |
| `DIDB` | Drug Interaction Database (UW) | Global | https://www.druginteractionsolutions.org | Quantitative DDI predictions |
| `PubChem` | NIH PubChem | Global | https://pubchem.ncbi.nlm.nih.gov | Chemical properties |

### LAYER 3: Clinical Practice Authorities

| Authority ID | Full Name | Jurisdiction | URL | Data Type |
|--------------|-----------|--------------|-----|-----------|
| `Lexicomp` | Lexicomp Drug Information | US | https://online.lexi.com | DDI severity, management |
| `Micromedex` | IBM Micromedex | US | https://www.micromedexsolutions.com | DDI severity, management |
| `UpToDate` | UpToDate Clinical Decision Support | US | https://www.uptodate.com | Clinical guidance |
| `AHFS.DI` | AHFS Drug Information | US | https://www.ahfsdruginformation.com | Comprehensive drug info |
| `AMH` | Australian Medicines Handbook | AU | https://amhonline.amh.net.au | AU DDI guidance |
| `eTG` | Therapeutic Guidelines | AU | https://www.tg.org.au | AU clinical guidelines |
| `BNF` | British National Formulary | UK | https://bnf.nice.org.uk | UK DDI guidance |
| `ASHP.DDI` | ASHP Drug Interaction Guidelines | US | ASHP publications | DDI classification |
| `ACCP.Antithrombotic` | ACCP Antithrombotic Guidelines | US | https://www.accp.com | Anticoagulant DDI |
| `Stockley` | Stockley's Drug Interactions | UK | https://www.medicinescomplete.com | DDI reference text |
| `Hansten.Horn` | Hansten and Horn's DDI Analysis | US | Published text | DDI analysis method |

---

## KB-0 Authority Registration Schema

Each authority must be registered in KB-0 with this structure:

```yaml
# kb-0/authorities/ddi_authorities.yaml

authorities:
  # LAYER 1: REGULATORY
  - id: "FDA.Section7"
    name: "FDA Drug Label Section 7 - Drug Interactions"
    layer: "REGULATORY"
    jurisdiction: "US"
    authority_type: "PRIMARY"
    url: "https://dailymed.nlm.nih.gov"
    data_format: "SPL XML"
    update_frequency: "As labels updated"
    legal_weight: "HIGHEST"
    description: "Official FDA-approved drug interaction information"
    
  - id: "TGA.Section4.5"
    name: "TGA Product Information Section 4.5 - Interactions"
    layer: "REGULATORY"
    jurisdiction: "AU"
    authority_type: "PRIMARY"
    url: "https://www.tga.gov.au"
    data_format: "PDF/XML"
    update_frequency: "As PIs updated"
    legal_weight: "HIGHEST"
    description: "Official TGA-approved drug interaction information"

  # LAYER 2: PHARMACOLOGY
  - id: "DrugBank"
    name: "DrugBank Database"
    layer: "PHARMACOLOGY"
    jurisdiction: "GLOBAL"
    authority_type: "SECONDARY"
    url: "https://go.drugbank.com"
    data_format: "XML/JSON API"
    update_frequency: "Continuous"
    legal_weight: "SUPPORTIVE"
    description: "Comprehensive CYP enzyme, transporter, and target data"
    
  - id: "Flockhart.CYP"
    name: "Indiana University Flockhart CYP Table"
    layer: "PHARMACOLOGY"
    jurisdiction: "GLOBAL"
    authority_type: "SECONDARY"
    url: "https://drug-interactions.medicine.iu.edu"
    data_format: "HTML Table"
    update_frequency: "Periodic"
    legal_weight: "SUPPORTIVE"
    description: "Gold standard CYP450 substrate/inhibitor/inducer classification"

  # LAYER 3: CLINICAL PRACTICE
  - id: "Lexicomp"
    name: "Lexicomp Drug Interactions"
    layer: "CLINICAL_PRACTICE"
    jurisdiction: "US"
    authority_type: "SECONDARY"
    url: "https://online.lexi.com"
    data_format: "Proprietary"
    update_frequency: "Continuous"
    legal_weight: "SUPPORTIVE"
    description: "Clinical severity classification and management guidance"
    
  - id: "AMH"
    name: "Australian Medicines Handbook"
    layer: "CLINICAL_PRACTICE"
    jurisdiction: "AU"
    authority_type: "SECONDARY"
    url: "https://amhonline.amh.net.au"
    data_format: "HTML/PDF"
    update_frequency: "Annual + updates"
    legal_weight: "SUPPORTIVE"
    description: "Australian clinical practice DDI guidance"
```

---

## KB-5 Governance Schema (Required Addition)

Your KB-5 `DrugInteraction` struct needs governance metadata:

```go
// DrugInteraction with governance (REQUIRED UPDATE)
type DrugInteraction struct {
    ID              string              `json:"id"`
    
    // Interacting drugs (existing)
    Drug1RxNorm     string              `json:"drug1RxNorm"`
    Drug1Name       string              `json:"drug1Name"`
    Drug2RxNorm     string              `json:"drug2RxNorm"`
    Drug2Name       string              `json:"drug2Name"`
    
    // Interaction details (existing)
    Severity        InteractionSeverity `json:"severity"`
    EvidenceLevel   EvidenceLevel       `json:"evidenceLevel"`
    MechanismType   MechanismType       `json:"mechanismType"`
    
    // Clinical details (existing)
    Description     string              `json:"description"`
    ClinicalEffect  string              `json:"clinicalEffect"`
    Management      string              `json:"management"`
    
    // ═══════════════════════════════════════════════════════════════════════
    // NEW: GOVERNANCE METADATA (REQUIRED)
    // ═══════════════════════════════════════════════════════════════════════
    Governance      DDIGovernance       `json:"governance" yaml:"governance"`
}

// DDIGovernance contains the three-layer authority provenance
type DDIGovernance struct {
    // Layer 1: Regulatory Source (REQUIRED)
    RegulatorySource    string   `json:"regulatorySource" yaml:"regulatorySource"`       // e.g., "FDA.Section7"
    RegulatoryReference string   `json:"regulatoryReference" yaml:"regulatoryReference"` // e.g., "Warfarin label, Section 7.1"
    RegulatoryDate      string   `json:"regulatoryDate" yaml:"regulatoryDate"`           // Label revision date
    
    // Layer 2: Pharmacology Source (REQUIRED for PK interactions)
    PharmacologySource    string   `json:"pharmacologySource,omitempty" yaml:"pharmacologySource,omitempty"`       // e.g., "DrugBank", "Flockhart.CYP"
    PharmacologyReference string   `json:"pharmacologyReference,omitempty" yaml:"pharmacologyReference,omitempty"` // e.g., "DrugBank DB00682"
    MechanismEvidence     string   `json:"mechanismEvidence,omitempty" yaml:"mechanismEvidence,omitempty"`         // e.g., "CYP2C9 Ki = 2.3 μM"
    
    // Layer 3: Clinical Practice Source (REQUIRED)
    ClinicalSource    string   `json:"clinicalSource" yaml:"clinicalSource"`       // e.g., "Lexicomp", "AMH"
    ClinicalReference string   `json:"clinicalReference" yaml:"clinicalReference"` // e.g., "Lexicomp 2024"
    SeveritySource    string   `json:"severitySource" yaml:"severitySource"`       // Who classified severity
    
    // Jurisdiction
    Jurisdiction     string   `json:"jurisdiction" yaml:"jurisdiction"`           // "US", "AU", "IN", "GLOBAL"
    
    // Quality Metadata
    EvidenceGrade    string   `json:"evidenceGrade" yaml:"evidenceGrade"`         // "HIGH", "MODERATE", "LOW"
    LastReviewed     string   `json:"lastReviewed" yaml:"lastReviewed"`           // Date of last clinical review
    ReviewedBy       string   `json:"reviewedBy" yaml:"reviewedBy"`               // CMO, PharmD name
    
    // Version Control
    Version          string   `json:"version" yaml:"version"`                     // "1.0.0"
    EffectiveDate    string   `json:"effectiveDate" yaml:"effectiveDate"`         // When this rule became active
}
```

---

## Example: Governed DDI Entry

### Before (Current - No Governance)

```go
{
    ID: "ddi-warfarin-fluconazole",
    Drug1RxNorm: "11289", Drug1Name: "Warfarin",
    Drug2RxNorm: "4083", Drug2Name: "Fluconazole",
    Severity: SeverityMajor,
    ClinicalEffect: "INR can more than double, high bleeding risk",
    Management: "Reduce warfarin dose by 25-50%. Monitor INR closely.",
    // NO GOVERNANCE - WHERE DID THIS COME FROM?
}
```

### After (With Governance - Legally Defensible)

```yaml
- id: "ddi-warfarin-fluconazole"
  drug1RxNorm: "11289"
  drug1Name: "Warfarin"
  drug2RxNorm: "4083"
  drug2Name: "Fluconazole"
  
  severity: "MAJOR"
  evidenceLevel: "ESTABLISHED"
  mechanismType: "PHARMACOKINETIC"
  pkMechanism: "METABOLISM_CYP"
  cypEnzymes: ["CYP2C9"]
  inhibitor: "Fluconazole"
  substrate: "Warfarin"
  
  description: "Fluconazole strongly inhibits CYP2C9, the primary enzyme metabolizing S-warfarin"
  clinicalEffect: "INR can increase 2-3 fold, significantly elevated bleeding risk"
  management: "Reduce warfarin dose by 25-50%. Monitor INR every 2-3 days during fluconazole therapy and 1 week after."
  
  governance:
    # Layer 1: Regulatory
    regulatorySource: "FDA.Section7"
    regulatoryReference: "Warfarin (Coumadin) Label, Section 7.1 - CYP2C9 Inhibitors"
    regulatoryDate: "2023-08"
    
    # Layer 2: Pharmacology
    pharmacologySource: "DrugBank"
    pharmacologyReference: "DrugBank DB00196 (Fluconazole) - Strong CYP2C9 inhibitor"
    mechanismEvidence: "Fluconazole CYP2C9 Ki = 7.1 μM (strong inhibition)"
    
    # Layer 3: Clinical Practice
    clinicalSource: "Lexicomp"
    clinicalReference: "Lexicomp 2024 - Warfarin Drug Interactions"
    severitySource: "Lexicomp"
    
    # Metadata
    jurisdiction: "US"
    evidenceGrade: "HIGH"
    lastReviewed: "2024-01-15"
    reviewedBy: "Dr. Clinical Pharmacist, PharmD"
    version: "1.0.0"
    effectiveDate: "2024-01-15"
```

---

## Authority Hierarchy by Interaction Type

### Drug-Drug Interactions (Pharmacokinetic)

| Authority Layer | Primary Source | Secondary Source |
|-----------------|---------------|------------------|
| Regulatory | FDA Section 7, 12 | TGA Section 4.5 |
| Pharmacology | **DrugBank** | Flockhart CYP |
| Clinical | Lexicomp | Micromedex |

### Drug-Drug Interactions (Pharmacodynamic)

| Authority Layer | Primary Source | Secondary Source |
|-----------------|---------------|------------------|
| Regulatory | FDA Section 5, 7 | TGA Section 4.4, 4.5 |
| Pharmacology | FDA Section 12 | Literature |
| Clinical | Lexicomp | UpToDate |

### Drug-Food Interactions

| Authority Layer | Primary Source | Secondary Source |
|-----------------|---------------|------------------|
| Regulatory | FDA Section 7 | TGA Section 4.5 |
| Pharmacology | DrugBank | Literature |
| Clinical | **Lexicomp** | AMH |

### Drug-Disease Contraindications

| Authority Layer | Primary Source | Secondary Source |
|-----------------|---------------|------------------|
| Regulatory | **FDA Section 4** | TGA Section 4.3 |
| Pharmacology | FDA Section 12 | N/A |
| Clinical | UpToDate | Specialty guidelines |

### QT Prolongation Interactions

| Authority Layer | Primary Source | Secondary Source |
|-----------------|---------------|------------------|
| Regulatory | FDA Section 5 | TGA Section 4.4 |
| Pharmacology | **CredibleMeds** | ICH E14 guidance |
| Clinical | CredibleMeds | AHA Guidelines |

### Serotonin Syndrome Interactions

| Authority Layer | Primary Source | Secondary Source |
|-----------------|---------------|------------------|
| Regulatory | **FDA Black Box** | TGA Section 4.4 |
| Pharmacology | Literature | DrugBank targets |
| Clinical | Lexicomp | Serotonin syndrome criteria |

---

## Special Authorities by Mechanism

### CYP450 Interactions

| Source | Type | Use |
|--------|------|-----|
| **Flockhart CYP Table** | Gold standard | Substrate/inhibitor/inducer classification |
| **DrugBank** | Comprehensive | Ki values, quantitative inhibition |
| **FDA Section 12** | Regulatory | Official PK data |
| **UW DIDB** | Quantitative | DDI predictions |

### Transporter Interactions

| Source | Type | Use |
|--------|------|-----|
| **UW DDI Database** | Primary | P-gp, OATP, BCRP interactions |
| **DrugBank** | Secondary | Transporter targets |
| **FDA Section 12** | Regulatory | Official transporter data |

### QT Prolongation

| Source | Type | Use |
|--------|------|-----|
| **CredibleMeds** | Primary | QT risk classification |
| **FDA Section 5** | Regulatory | QT warnings |
| **Arizona CERT** | Classification | Known/Possible/Conditional QT risk |

---

## Data Flow: KB-5 → KB-0 → Audit

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              KB-5 → KB-0 → AUDIT FLOW                                       │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                             │
│  KB-5 DDI CHECK                                                                            │
│  ══════════════                                                                            │
│                                                                                             │
│     Input: Warfarin + Fluconazole                                                          │
│              │                                                                              │
│              ▼                                                                              │
│     ┌─────────────────────────────────────────────────────────────────────────────────┐    │
│     │  KB-5 Finds Interaction                                                          │    │
│     │                                                                                  │    │
│     │  Severity: MAJOR                                                                 │    │
│     │  Mechanism: CYP2C9 inhibition                                                    │    │
│     │  Management: Reduce warfarin 25-50%                                              │    │
│     │                                                                                  │    │
│     │  Governance:                                                                     │    │
│     │  • Regulatory: FDA.Section7 (Warfarin label 2023-08)                            │    │
│     │  • Pharmacology: DrugBank DB00196                                               │    │
│     │  • Clinical: Lexicomp 2024                                                      │    │
│     └─────────────────────────────────────────────────────────────────────────────────┘    │
│              │                                                                              │
│              │ Governance metadata                                                         │
│              ▼                                                                              │
│     ┌─────────────────────────────────────────────────────────────────────────────────┐    │
│     │  KB-0 Authority Validation                                                       │    │
│     │                                                                                  │    │
│     │  ✓ FDA.Section7 is registered authority (PRIMARY, US)                           │    │
│     │  ✓ DrugBank is registered authority (SECONDARY, GLOBAL)                         │    │
│     │  ✓ Lexicomp is registered authority (SECONDARY, US)                             │    │
│     │                                                                                  │    │
│     │  Authority chain VALID                                                           │    │
│     └─────────────────────────────────────────────────────────────────────────────────┘    │
│              │                                                                              │
│              ▼                                                                              │
│     ┌─────────────────────────────────────────────────────────────────────────────────┐    │
│     │  AUDIT TRAIL OUTPUT                                                              │    │
│     │                                                                                  │    │
│     │  "DDI Alert: Warfarin + Fluconazole                                             │    │
│     │   Severity: MAJOR                                                                │    │
│     │   Action: Reduce warfarin 25-50%                                                │    │
│     │                                                                                  │    │
│     │   Evidence Chain:                                                                │    │
│     │   1. FDA Warfarin Label Section 7.1 (Aug 2023) - CYP2C9 inhibitor warning       │    │
│     │   2. DrugBank DB00196 - Fluconazole strong CYP2C9 inhibitor (Ki 7.1 μM)         │    │
│     │   3. Lexicomp 2024 - Severity MAJOR, reduce dose 25-50%                         │    │
│     │                                                                                  │    │
│     │   This alert meets medico-legal documentation standards."                        │    │
│     └─────────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## KB-5 Missing: What You Need to Add

### 1. Governance Metadata (CRITICAL)

Every interaction must have:
- `regulatorySource` (FDA/TGA/CDSCO label reference)
- `pharmacologySource` (DrugBank/Flockhart for CYP)
- `clinicalSource` (Lexicomp/AMH for severity)

### 2. Authority Registration in KB-0

Create `/kb-0/authorities/ddi_authorities.yaml` with all sources listed above.

### 3. Jurisdiction-Specific Rules

Your current data.go is US-centric. Add:
- AU-specific TGA references
- IN-specific CDSCO references
- Regional severity variations

### 4. CYP Evidence

Your CYP profiles lack:
- Ki values (inhibition strength)
- Induction fold-change
- Clinical significance thresholds

---

## Summary: KB-5 Authority Stack for KB-0

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                         KB-5 AUTHORITY STACK FOR KB-0                                       │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                             │
│  LAYER 1: REGULATORY (Legal basis)                                                         │
│  ═════════════════════════════════                                                         │
│  • FDA.DailyMed / FDA.Section4 / FDA.Section5 / FDA.Section7 / FDA.Section12               │
│  • TGA.PI / TGA.Section4.3 / TGA.Section4.5                                                │
│  • CDSCO.Insert                                                                            │
│  • EMA.SmPC / MHRA.SmPC                                                                    │
│                                                                                             │
│  LAYER 2: PHARMACOLOGY (Scientific basis)                                                  │
│  ═════════════════════════════════════════                                                 │
│  • DrugBank (CYP, transporters, targets)                                                  │
│  • Flockhart.CYP (CYP classification)                                                     │
│  • UW.DDI (Transporter DDI)                                                               │
│  • PharmGKB (Pharmacogenomics)                                                            │
│  • CredibleMeds (QT risk)                                                                 │
│                                                                                             │
│  LAYER 3: CLINICAL PRACTICE (Management basis)                                             │
│  ═════════════════════════════════════════════                                             │
│  • Lexicomp / Micromedex / UpToDate (US)                                                  │
│  • AMH / eTG (AU)                                                                         │
│  • BNF / Stockley (UK)                                                                    │
│  • ACCP.Antithrombotic / ASHP.DDI (Specialty)                                             │
│                                                                                             │
│  TOTAL AUTHORITIES: 25+                                                                    │
│                                                                                             │
│  REQUIRED FOR EACH DDI:                                                                    │
│  • At least 1 REGULATORY source (REQUIRED)                                                │
│  • At least 1 PHARMACOLOGY source (for PK interactions)                                   │
│  • At least 1 CLINICAL source (REQUIRED)                                                  │
│                                                                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## CTO/CMO Final Verdict

Your KB-5 is **functionally complete** but **governance-incomplete**.

**Action Required**:

| Priority | Action | Effort |
|----------|--------|--------|
| P0 | Add `DDIGovernance` struct to all interactions | 2 days |
| P0 | Register authorities in KB-0 | 1 day |
| P1 | Add jurisdiction-specific references (TGA, CDSCO) | 3 days |
| P2 | Add CYP Ki values from DrugBank | 2 days |

Without governance metadata, your DDI alerts are **"software opinions"**.

With governance metadata, they become **"medico-legal clinical guidance"**.

**DECISION**: Add governance before production deployment.
