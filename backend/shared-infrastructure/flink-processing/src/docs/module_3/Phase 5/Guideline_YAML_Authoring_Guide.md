# Guideline YAML Authoring Guide

## Table of Contents
1. [Overview](#overview)
2. [YAML Structure](#yaml-structure)
3. [Required vs Optional Fields](#required-vs-optional-fields)
4. [Recommendation Format](#recommendation-format)
5. [Evidence Quality Mapping](#evidence-quality-mapping)
6. [Linking to Protocols](#linking-to-protocols)
7. [Superseded Guidelines](#superseded-guidelines)
8. [Validation Checklist](#validation-checklist)

---

## Overview

This guide provides instructions for creating YAML files that represent clinical practice guidelines in the CardioFit knowledge base. Following these standards ensures consistency, traceability, and integration with the clinical decision support system.

### Purpose of Guideline YAMLs

- **Structured Storage**: Machine-readable format for guideline data
- **Version Control**: Track guideline changes over time
- **Evidence Linkage**: Connect recommendations to research citations (PMIDs)
- **Protocol Integration**: Link guidelines to executable clinical protocols
- **Quality Assessment**: Store GRADE-based evidence quality ratings

### File Naming Convention

```
{organization}-{topic}-{year}.yaml

Examples:
- accaha-stemi-2023.yaml
- ssc-2021.yaml
- esc-stemi-2023.yaml
- nice-sepsis-2024.yaml
```

### Directory Structure

```
knowledge-base/guidelines/
├── cardiac/
│   ├── accaha-stemi-2023.yaml
│   ├── accaha-stemi-2013.yaml (superseded)
│   └── esc-stemi-2023.yaml
├── sepsis/
│   ├── ssc-2021.yaml
│   ├── ssc-2016.yaml (superseded)
│   └── nice-sepsis-2024.yaml
├── respiratory/
│   ├── bts-cap-2019.yaml
│   ├── ats-ards-2023.yaml
│   └── gold-copd-2024.yaml
└── cross-cutting/
    ├── grade-methodology.yaml
    └── acr-appropriateness.yaml
```

---

## YAML Structure

### Complete Template with Field Descriptions

```yaml
# ==================================================================
# GUIDELINE HEADER
# ==================================================================
guidelineId: "GUIDE-{ORG}-{TOPIC}-{YEAR}"
  # Unique identifier following pattern: GUIDE-{ORG}-{TOPIC}-{YEAR}
  # Examples: GUIDE-ACCAHA-STEMI-2023, GUIDE-SSC-2021
  # REQUIRED

name: "Full guideline title from official publication"
  # Complete formal name as published
  # Example: "2023 ACC/AHA/SCAI Guideline for the Management of Patients With Acute Myocardial Infarction"
  # REQUIRED

shortName: "Abbreviated name for UI display"
  # Brief version for clinical interface
  # Example: "ACC/AHA STEMI 2023"
  # REQUIRED

organization: "Publishing organization(s)"
  # Full name of authoring organization
  # Example: "American College of Cardiology / American Heart Association"
  # REQUIRED

topic: "Clinical domain or condition"
  # Clinical area covered
  # Example: "ST-Elevation Myocardial Infarction (STEMI) Management"
  # REQUIRED


# ==================================================================
# VERSIONING
# ==================================================================
version: "YYYY.N"
  # Version number: YYYY.N where N increments for revisions
  # Example: "2023.1" for first version, "2023.2" for amendment
  # REQUIRED

publicationDate: "YYYY-MM-DD"
  # Official publication date in ISO 8601 format
  # REQUIRED

lastReviewDate: "YYYY-MM-DD"
  # Date of last review by authoring body
  # REQUIRED

nextReviewDate: "YYYY-MM-DD"
  # Scheduled next review date (typically 3-5 years)
  # REQUIRED

status: "CURRENT | SUPERSEDED | UNDER_REVIEW | WITHDRAWN | ARCHIVED"
  # Current status of guideline
  # REQUIRED

supersededBy: "GUIDE-{ID}"
  # If status is SUPERSEDED, ID of replacement guideline
  # OPTIONAL (required if status = SUPERSEDED)

supersededDate: "YYYY-MM-DD"
  # Date this guideline was superseded
  # OPTIONAL (required if status = SUPERSEDED)


# ==================================================================
# PUBLICATION DETAILS
# ==================================================================
publication:
  journal: "Journal name"
    # REQUIRED
  year: 2023
    # Integer publication year
    # REQUIRED
  volume: 81
    # Integer volume number
    # OPTIONAL
  issue: 14
    # Integer issue number
    # OPTIONAL
  pages: "1372-1424"
    # Page range as string
    # OPTIONAL
  doi: "10.1016/j.jacc.2023.04.001"
    # Digital Object Identifier
    # REQUIRED
  pmid: "37079885"
    # PubMed ID (if available)
    # OPTIONAL
  url: "https://..."
    # Full text URL
    # REQUIRED
  pdfUrl: "https://..."
    # PDF download URL
    # OPTIONAL


# ==================================================================
# SCOPE & APPLICABILITY
# ==================================================================
scope:
  clinicalDomain: "Specialty area(s)"
    # Example: "Cardiology / Emergency Medicine / Critical Care"
    # REQUIRED

  targetPopulations:
    # List of patient populations covered
    # REQUIRED
    - "Adults (≥18 years) with condition X"
    - "Pediatrics with specific criteria"

  targetSettings:
    # List of clinical settings
    # REQUIRED
    - "Emergency Department"
    - "ICU / CCU"
    - "Inpatient Ward"

  exclusions:
    # List of populations/settings NOT covered
    # OPTIONAL
    - "Pediatric patients (<18 years)"
    - "Pregnancy (separate guideline)"

  geographicScope: "Geographic applicability"
    # Example: "United States", "Europe", "International"
    # REQUIRED


# ==================================================================
# METHODOLOGY
# ==================================================================
methodology:
  approachUsed: "Methodology framework"
    # Example: "GRADE", "ACC/AHA Methodology", "SIGN"
    # REQUIRED

  evidenceSearchStrategy: "Description of literature search"
    # OPTIONAL

  evidenceSearchDate: "YYYY-MM-DD"
    # Date through which evidence was searched
    # OPTIONAL

  panelSize: 21
    # Number of guideline panel members
    # OPTIONAL

  panelComposition:
    # List describing panel makeup
    # OPTIONAL
    - "17 cardiologists"
    - "2 emergency physicians"
    - "1 methodologist"

  conflictOfInterestPolicy: "Description of COI management"
    # OPTIONAL

  externalReview: true
    # Boolean: was guideline externally reviewed?
    # OPTIONAL

  numberOfReviewers: 42
    # Number of external reviewers
    # OPTIONAL


# ==================================================================
# RECOMMENDATIONS
# ==================================================================
recommendations:
  # List of individual recommendations
  # REQUIRED - must have at least one

  - recommendationId: "{GUIDELINE-ID}-REC-{NUMBER}"
      # Unique ID: {GUIDELINE-ID}-REC-{sequential number}
      # Example: "ACC-STEMI-2023-REC-001"
      # REQUIRED

    number: "1.1"
      # Section numbering from guideline
      # OPTIONAL

    section: "Section title"
      # Section heading from guideline
      # OPTIONAL

    title: "Short recommendation title"
      # Brief descriptive title
      # OPTIONAL

    statement: "Full recommendation text"
      # Complete recommendation statement in natural language
      # REQUIRED

    strength: "STRONG | WEAK | CONDITIONAL"
      # GRADE recommendation strength
      # REQUIRED

    classOfRecommendation: "Class I | Class IIa | Class IIb | Class III"
      # ACC/AHA classification (if applicable)
      # OPTIONAL

    evidenceQuality: "HIGH | MODERATE | LOW | VERY_LOW"
      # GRADE evidence quality
      # REQUIRED

    gradeLevel: "High | Moderate | Low | Very Low"
      # Text version of evidence quality
      # OPTIONAL

    levelOfEvidence: "A | B-R | B-NR | C-LD | C-EO"
      # ACC/AHA level of evidence (if applicable)
      # A = High-quality evidence from multiple RCTs
      # B-R = Moderate-quality from single RCT or meta-analysis
      # B-NR = Moderate-quality from non-randomized studies
      # C-LD = Limited data
      # C-EO = Expert opinion
      # OPTIONAL

    rationale: |
      Multi-line explanation of why this recommendation
      is made. Include key supporting evidence and
      clinical reasoning.
      # OPTIONAL but highly recommended

    keyEvidence:
      # List of PubMed IDs (PMIDs) supporting this recommendation
      # REQUIRED - must have at least one
      - "37079885"  # Primary guideline PMID
      - "12517460"  # Key supporting trial
      - "27282490"  # Additional evidence

    linkedProtocolActions:
      # List of protocol action IDs that implement this recommendation
      # OPTIONAL but required for CDS integration
      - "STEMI-ACT-001"
      - "STEMI-ACT-002"

    clinicalConsiderations: |
      Additional clinical context, warnings, or
      implementation guidance.
      # OPTIONAL


# ==================================================================
# RELATED GUIDELINES
# ==================================================================
relatedGuidelines:
  # List of related guidelines with relationship type
  # OPTIONAL

  - guidelineId: "GUIDE-ACCAHA-STEMI-2013"
    relationship: "SUPERSEDES | SUPERSEDED_BY | COMPLEMENTARY | CONFLICTING"
    note: "Brief explanation of relationship"


# ==================================================================
# QUALITY INDICATORS
# ==================================================================
qualityIndicators:
  # Performance measures derived from guideline
  # OPTIONAL

  - indicatorId: "QI-{TOPIC}-{NUMBER}"
    measure: "Description of quality measure"
    target: "Target performance level"
    rationale: "Why this measure matters"


# ==================================================================
# ALGORITHM SUMMARY
# ==================================================================
algorithmSummary: |
  Text or ASCII representation of guideline decision algorithm.
  Useful for quick reference and clinical implementation.
  # OPTIONAL


# ==================================================================
# MAJOR UPDATES
# ==================================================================
majorUpdates:
  # List of key changes from previous version
  # OPTIONAL but recommended for updated guidelines
  - "Description of change 1"
  - "Description of change 2"


# ==================================================================
# METADATA
# ==================================================================
lastUpdated: "YYYY-MM-DD"
  # Date YAML file was last modified
  # REQUIRED

source: "Organization name"
  # Authoring organization
  # REQUIRED

version: "1.0"
  # YAML file version (for tracking file format changes)
  # REQUIRED
```

---

## Required vs Optional Fields

### Minimum Required Fields

To create a valid guideline YAML, you **must** include:

```yaml
guidelineId: "GUIDE-{ORG}-{TOPIC}-{YEAR}"
name: "Full title"
shortName: "Brief name"
organization: "Organization"
topic: "Clinical topic"
version: "YYYY.N"
publicationDate: "YYYY-MM-DD"
lastReviewDate: "YYYY-MM-DD"
nextReviewDate: "YYYY-MM-DD"
status: "CURRENT"

publication:
  journal: "Journal"
  year: 2023
  doi: "10.xxxx/xxxxx"
  url: "https://..."

scope:
  clinicalDomain: "Domain"
  targetPopulations:
    - "Population"
  targetSettings:
    - "Setting"
  geographicScope: "Region"

methodology:
  approachUsed: "Framework"

recommendations:
  - recommendationId: "REC-001"
    statement: "Recommendation text"
    strength: "STRONG"
    evidenceQuality: "HIGH"
    keyEvidence:
      - "12345678"

lastUpdated: "YYYY-MM-DD"
source: "Organization"
version: "1.0"
```

### Recommended Optional Fields

For complete clinical utility, also include:

- `recommendation.rationale`: Explains why recommendation matters
- `recommendation.linkedProtocolActions`: Enables CDS integration
- `recommendation.clinicalConsiderations`: Implementation guidance
- `relatedGuidelines`: Shows guideline evolution
- `algorithmSummary`: Quick clinical reference

---

## Recommendation Format

### Anatomy of a Recommendation

```yaml
- recommendationId: "ACC-STEMI-2023-REC-003"
  number: "3.1"
  section: "Antiplatelet Therapy"
  title: "Aspirin 162-325 mg Loading Dose"

  statement: >
    Aspirin 162 to 325 mg should be given as soon as possible
    to all patients with STEMI who do not have a true aspirin allergy

  strength: "STRONG"
  classOfRecommendation: "Class I"
  evidenceQuality: "HIGH"
  gradeLevel: "High"
  levelOfEvidence: "A"

  rationale: |
    Aspirin reduces mortality in STEMI by approximately 23% (ISIS-2 trial).
    Should be given immediately upon STEMI recognition. Non-enteric coated
    formulation chewed for rapid absorption.

  keyEvidence:
    - "37079885"  # ACC/AHA STEMI 2023 guideline
    - "3081859"   # ISIS-2 trial - Aspirin mortality benefit
    - "18160631"  # De Luca G - Aspirin in primary PCI

  linkedProtocolActions:
    - "STEMI-ACT-002"  # Aspirin 324 mg PO chewable

  clinicalConsiderations: |
    - Chewable aspirin preferred for faster absorption
    - Continue 81 mg daily indefinitely after loading dose
    - True aspirin allergy rare (not aspirin sensitivity/GI upset)
```

### Writing Effective Recommendation Statements

**Good Recommendation Statements:**
- Use clear action verbs: "should be given", "is recommended", "should be performed"
- Specify who/what/when/where
- Include dosing/timing when applicable
- State contraindications if relevant

**Examples:**

✅ **Good:**
```yaml
statement: >
  A 12-lead ECG should be performed and evaluated for STEMI
  within 10 minutes of first medical contact in patients with
  symptoms suggestive of STEMI
```

❌ **Too vague:**
```yaml
statement: "ECG should be done quickly"
```

✅ **Good:**
```yaml
statement: >
  Norepinephrine is recommended as the first-choice vasopressor
  for patients with septic shock
```

❌ **Too ambiguous:**
```yaml
statement: "Use vasopressors for shock"
```

### Strength and Evidence Quality Combinations

| Strength | Evidence Quality | Interpretation |
|----------|------------------|----------------|
| STRONG | HIGH | Strong confidence, clear benefit > risk |
| STRONG | MODERATE | High confidence despite moderate evidence quality |
| WEAK | HIGH | High-quality evidence but benefit/risk closely balanced |
| WEAK | MODERATE | Moderate evidence, individualized decisions |
| WEAK | LOW | Limited evidence, expert consensus |
| CONDITIONAL | Any | Depends heavily on patient values/preferences |

---

## Evidence Quality Mapping

### How to Assign Evidence Quality Levels

Use the GRADE framework to determine evidence quality based on study types:

#### HIGH Evidence

Assign HIGH when:
- Multiple large, well-designed RCTs with consistent results
- High-quality systematic reviews/meta-analyses
- No serious limitations, inconsistency, or indirectness

**Examples:**
```yaml
evidenceQuality: "HIGH"
keyEvidence:
  - "3081859"   # ISIS-2: RCT with 17,187 patients
  - "18160631"  # Meta-analysis of primary PCI trials
```

#### MODERATE Evidence

Assign MODERATE when:
- Single RCT or multiple RCTs with limitations
- Very strong observational evidence
- Downgraded from HIGH due to limitations

**Reasons for downgrading:**
- Study limitations (risk of bias)
- Inconsistency between studies
- Indirectness of evidence
- Imprecision (wide confidence intervals)

**Example:**
```yaml
evidenceQuality: "MODERATE"
levelOfEvidence: "B-R"
keyEvidence:
  - "24635773"  # Single RCT with some limitations
```

#### LOW Evidence

Assign LOW when:
- Observational studies (cohort, case-control)
- RCTs with serious limitations
- Inconsistent findings

**Example:**
```yaml
evidenceQuality: "LOW"
levelOfEvidence: "C-LD"
keyEvidence:
  - "21378355"  # Observational cohort study
```

#### VERY_LOW Evidence

Assign VERY_LOW when:
- Case series or case reports
- Expert opinion
- Indirect evidence
- Very serious study limitations

**Example:**
```yaml
evidenceQuality: "VERY_LOW"
levelOfEvidence: "C-EO"
rationale: "Based on expert consensus due to lack of RCT data"
```

### Study Type to Evidence Quality Mapping

| Study Type | Typical Quality | Notes |
|------------|----------------|-------|
| Meta-analysis of RCTs | HIGH | If well-conducted |
| Multiple RCTs | HIGH | If consistent results |
| Single large RCT | HIGH-MODERATE | Depends on quality |
| Small RCT | MODERATE-LOW | Sample size matters |
| Cohort study | LOW-MODERATE | Can be upgraded if very strong |
| Case-control | LOW | Rarely higher |
| Case series | VERY_LOW | |
| Expert opinion | VERY_LOW | |

---

## Linking to Protocols

### How to Identify Protocol Actions

Protocol actions are executable clinical steps defined in protocol YAML files. To link a guideline recommendation to a protocol action:

1. **Find the protocol file** in `protocols/` directory (e.g., `stemi-protocol.yaml`)

2. **Identify the relevant action ID** from the protocol:
   ```yaml
   # stemi-protocol.yaml
   actions:
     - actionId: "STEMI-ACT-002"
       description: "Aspirin 324 mg PO chewable"
   ```

3. **Add the action ID** to your guideline recommendation:
   ```yaml
   # accaha-stemi-2023.yaml
   recommendations:
     - recommendationId: "ACC-STEMI-2023-REC-003"
       statement: "Aspirin 162 to 325 mg should be given..."
       linkedProtocolActions:
         - "STEMI-ACT-002"
   ```

4. **Create bidirectional link** in the protocol file:
   ```yaml
   # stemi-protocol.yaml
   actions:
     - actionId: "STEMI-ACT-002"
       description: "Aspirin 324 mg PO chewable"
       guidelineReferences:
         - "ACC-STEMI-2023-REC-003"
   ```

### Multiple Actions per Recommendation

A single recommendation may map to multiple protocol actions:

```yaml
# Guideline: antibiotic recommendation
recommendations:
  - recommendationId: "SSC-2021-REC-003"
    statement: "Broad-spectrum antibiotics within 1 hour"
    linkedProtocolActions:
      - "SEPSIS-ACT-003"  # Order blood cultures
      - "SEPSIS-ACT-004"  # Administer ceftriaxone
      - "SEPSIS-ACT-005"  # Administer vancomycin
```

### Multiple Recommendations per Action

An action may be supported by multiple recommendations:

```yaml
# Protocol action
actions:
  - actionId: "STEMI-ACT-002"
    description: "Aspirin 324 mg"
    guidelineReferences:
      - "ACC-STEMI-2023-REC-003"  # ACC/AHA guideline
      - "ESC-STEMI-2023-REC-004"  # ESC guideline
```

---

## Superseded Guidelines

### Marking a Guideline as Superseded

When a new guideline replaces an old one:

1. **Update the old guideline:**
   ```yaml
   # accaha-stemi-2013.yaml
   guidelineId: "GUIDE-ACCAHA-STEMI-2013"
   status: "SUPERSEDED"
   supersededBy: "GUIDE-ACCAHA-STEMI-2023"
   supersededDate: "2023-04-20"
   ```

2. **Reference in new guideline:**
   ```yaml
   # accaha-stemi-2023.yaml
   guidelineId: "GUIDE-ACCAHA-STEMI-2023"
   status: "CURRENT"

   relatedGuidelines:
     - guidelineId: "GUIDE-ACCAHA-STEMI-2013"
       relationship: "SUPERSEDES"
       note: "This 2023 guideline updates and replaces the 2013 guideline"
   ```

3. **Document major changes:**
   ```yaml
   # accaha-stemi-2023.yaml
   majorUpdates:
     - "Fibrinolysis threshold changed from >90 min to >120 min"
     - "Ticagrelor/prasugrel now preferred over clopidogrel"
     - "High-intensity statin strengthened from Class IIa to Class I"
   ```

### Guideline Evolution Tracking

```yaml
# Historical progression
relatedGuidelines:
  - guidelineId: "GUIDE-ACCAHA-STEMI-2004"
    relationship: "SUPERSEDES"
    note: "Original guideline"

  - guidelineId: "GUIDE-ACCAHA-STEMI-2013"
    relationship: "SUPERSEDES"
    note: "Major update with PCI time thresholds"

  - guidelineId: "GUIDE-ESC-STEMI-2023"
    relationship: "COMPLEMENTARY"
    note: "European perspective - some differences in approach"
```

---

## Validation Checklist

### Pre-Submission Validation

Before adding a guideline YAML to the knowledge base, verify:

#### ✅ Structure Validation

- [ ] File name follows pattern: `{org}-{topic}-{year}.yaml`
- [ ] Saved in correct directory: `guidelines/{category}/`
- [ ] YAML syntax is valid (use YAML validator)
- [ ] All required fields present
- [ ] Field values use correct types (string, integer, list, etc.)

#### ✅ Content Validation

- [ ] `guidelineId` follows pattern: `GUIDE-{ORG}-{TOPIC}-{YEAR}`
- [ ] `guidelineId` is unique across all guidelines
- [ ] `recommendationId` follows pattern: `{GUIDELINE-ID}-REC-{NUMBER}`
- [ ] All `recommendationId` values are unique
- [ ] `status` is one of: CURRENT, SUPERSEDED, UNDER_REVIEW, WITHDRAWN, ARCHIVED
- [ ] If status = SUPERSEDED, `supersededBy` field is present

#### ✅ Evidence Validation

- [ ] All PMIDs in `keyEvidence` are valid PubMed IDs
- [ ] Citation YAML files exist for all referenced PMIDs
- [ ] `strength` uses valid GRADE terms: STRONG, WEAK, CONDITIONAL
- [ ] `evidenceQuality` uses valid GRADE terms: HIGH, MODERATE, LOW, VERY_LOW
- [ ] Evidence quality matches supporting citations

#### ✅ Protocol Linkage Validation

- [ ] All `linkedProtocolActions` reference existing protocol action IDs
- [ ] Bidirectional links exist (protocol actions reference guidelines)
- [ ] At least one recommendation links to a protocol action (for CDS)

#### ✅ Publication Validation

- [ ] `doi` is valid and accessible
- [ ] `pmid` (if present) is valid PubMed ID
- [ ] `url` is accessible
- [ ] `publicationDate` is in YYYY-MM-DD format
- [ ] `nextReviewDate` is set (typically 3-5 years from publication)

#### ✅ Relationship Validation

- [ ] If guideline supersedes another, old guideline has `supersededBy` field
- [ ] Related guidelines in `relatedGuidelines` exist
- [ ] Relationship types are valid: SUPERSEDES, SUPERSEDED_BY, COMPLEMENTARY, CONFLICTING

#### ✅ Clinical Validation

- [ ] Recommendation statements accurately reflect guideline source
- [ ] Dosing/timing information matches published guideline
- [ ] Contraindications mentioned if critical
- [ ] Clinical considerations provide implementation guidance

### Automated Validation Script

```bash
#!/bin/bash
# validate-guideline.sh

GUIDELINE_FILE=$1

echo "Validating guideline: $GUIDELINE_FILE"

# 1. YAML syntax check
yamllint $GUIDELINE_FILE
if [ $? -ne 0 ]; then
    echo "❌ YAML syntax error"
    exit 1
fi

# 2. Check required fields
REQUIRED_FIELDS="guidelineId name shortName organization topic version publicationDate status"
for field in $REQUIRED_FIELDS; do
    if ! grep -q "^$field:" $GUIDELINE_FILE; then
        echo "❌ Missing required field: $field"
        exit 1
    fi
done

# 3. Validate PMIDs have citation files
PMIDS=$(grep -oP '- "\K\d+(?=")' $GUIDELINE_FILE)
for pmid in $PMIDS; do
    if [ ! -f "citations/pmid-$pmid.yaml" ]; then
        echo "⚠️  Warning: Citation file missing for PMID $pmid"
    fi
done

# 4. Check protocol action references exist
ACTIONS=$(grep -A1 "linkedProtocolActions:" $GUIDELINE_FILE | grep -oP '- "\K[^"]+')
for action in $ACTIONS; do
    # Search for action in protocol files
    if ! grep -r "actionId: \"$action\"" protocols/; then
        echo "⚠️  Warning: Protocol action $action not found"
    fi
done

echo "✅ Validation complete"
```

---

## Examples

### Example 1: Minimal Valid Guideline

```yaml
guidelineId: "GUIDE-EXAMPLE-TEST-2024"
name: "Example Guideline for Testing"
shortName: "Example 2024"
organization: "Example Medical Society"
topic: "Test Condition Management"
version: "2024.1"
publicationDate: "2024-01-15"
lastReviewDate: "2024-01-15"
nextReviewDate: "2029-01-15"
status: "CURRENT"

publication:
  journal: "Journal of Example Medicine"
  year: 2024
  doi: "10.1234/example.2024"
  url: "https://example.com/guideline"

scope:
  clinicalDomain: "Example Specialty"
  targetPopulations:
    - "Adults with test condition"
  targetSettings:
    - "Outpatient"
  geographicScope: "International"

methodology:
  approachUsed: "GRADE"

recommendations:
  - recommendationId: "EXAMPLE-2024-REC-001"
    statement: "Test intervention is recommended for eligible patients"
    strength: "STRONG"
    evidenceQuality: "HIGH"
    keyEvidence:
      - "12345678"

lastUpdated: "2024-01-15"
source: "Example Medical Society"
version: "1.0"
```

### Example 2: Complete Guideline with All Optional Fields

See existing guidelines in the knowledge base:
- `guidelines/cardiac/accaha-stemi-2023.yaml` (comprehensive cardiac guideline)
- `guidelines/sepsis/ssc-2021.yaml` (comprehensive sepsis guideline)

---

## Common Mistakes and How to Avoid Them

### Mistake 1: Inconsistent IDs

❌ **Wrong:**
```yaml
guidelineId: "ACCAHA-STEMI-2023"
recommendations:
  - recommendationId: "REC-001"  # Doesn't match guideline ID pattern
```

✅ **Correct:**
```yaml
guidelineId: "GUIDE-ACCAHA-STEMI-2023"
recommendations:
  - recommendationId: "ACC-STEMI-2023-REC-001"
```

### Mistake 2: Invalid GRADE Terms

❌ **Wrong:**
```yaml
strength: "Very Strong"
evidenceQuality: "Good"
```

✅ **Correct:**
```yaml
strength: "STRONG"
evidenceQuality: "HIGH"
```

### Mistake 3: Missing Citation Files

❌ **Wrong:**
```yaml
keyEvidence:
  - "12345678"  # No corresponding citation file
```

✅ **Correct:**
```yaml
keyEvidence:
  - "12345678"
# AND create file: citations/pmid-12345678.yaml
```

### Mistake 4: Broken Protocol Links

❌ **Wrong:**
```yaml
linkedProtocolActions:
  - "STEMI-ACTION-002"  # Typo in action ID
```

✅ **Correct:**
```yaml
linkedProtocolActions:
  - "STEMI-ACT-002"  # Matches protocol file
```

---

## Conclusion

Following this authoring guide ensures that guideline YAMLs are:
- **Consistent**: Uniform structure across all guidelines
- **Complete**: All required and recommended fields included
- **Traceable**: Proper linkage to citations and protocols
- **Valid**: Pass automated validation checks
- **Maintainable**: Easy to update when guidelines change

For additional guidance, see:
- [Evidence Chain Implementation Guide](./Evidence_Chain_Implementation_Guide.md)
- [Citation Management Guide](./Citation_Management_Guide.md)
- [Testing and Validation Guide](./Testing_Validation_Guide.md)

### Quick Start Checklist

1. Copy template from this guide
2. Fill in all required fields
3. Add recommendations with PMIDs
4. Link to protocol actions
5. Run validation script
6. Commit to version control

**Ready to create your first guideline? Start with the minimal template and expand from there!**
