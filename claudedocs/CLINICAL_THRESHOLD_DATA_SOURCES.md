# Clinical Threshold Data Sources - Complete Guide

**Question**: Where do we get the data for the 4-layer threshold configuration?

---

## Data Source Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  LAYER 1: Universal Thresholds                                  │
│  Sources: Clinical Guidelines, Medical Literature, LOINC        │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 2: Geographic Adjustments                                │
│  Sources: National Clinical Guidelines, Regulatory Bodies       │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 3: Demographic Adjustments                               │
│  Sources: Clinical Calculators, Research Papers, eGFR Formulas  │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 4: Contextual Adjustments                                │
│  Sources: Specialized Guidelines, Drug References, Research     │
└─────────────────────────────────────────────────────────────────┘
```

---

## Layer 1: Universal Thresholds

### **Source 1: LOINC Database** (FREE - Primary Source for Lab Identifiers)
**URL**: https://loinc.org/downloads/

**What You Get**:
- **LOINC codes**: Unique identifiers for every lab test (e.g., "2524-7" = Lactate)
- **Lab test names**: Official names and synonyms
- **Preferred units**: Standard measurement units (mmol/L, mg/dL, etc.)
- **Reference ranges**: Some LOINC codes include typical reference ranges

**Download**:
```bash
# LOINC Core Table (free, requires registration)
wget https://loinc.org/downloads/loinc/loinc-table-csv-text-format.zip

# Extract relevant fields:
# - LOINC_NUM (code)
# - COMPONENT (lab name)
# - PROPERTY (type: Mass, Substance, etc.)
# - TIME_ASPCT (timing)
# - SYSTEM (specimen type: Blood, Urine, etc.)
# - SCALE_TYP (quantitative vs qualitative)
# - METHOD_TYP (measurement method)
# - EXAMPLE_UCUM_UNITS (preferred units)
```

**Example Entry**:
```csv
LOINC_NUM,COMPONENT,PROPERTY,SYSTEM,SCALE_TYP,EXAMPLE_UCUM_UNITS
2524-7,Lactate,SCnc,Bld,Qn,mmol/L
2160-0,Creatinine,MCnc,Ser/Plas,Qn,mg/dL
6690-2,Leukocytes,NCnc,Bld,Qn,10*3/uL
```

---

### **Source 2: Clinical Practice Guidelines** (FREE - Evidence-Based Thresholds)

#### **A. Surviving Sepsis Campaign** (Sepsis/Lactate Thresholds)
**URL**: https://www.sccm.org/SurvivingSepsisCampaign/Guidelines

**Key Data**:
- **Lactate threshold**: >2.0 mmol/L indicates tissue hypoperfusion
- **Severe lactate**: >4.0 mmol/L indicates septic shock
- **Guideline version**: 2021 (updated every 4 years)

**Citation**:
```
Evans L, Rhodes A, Alhazzani W, et al. Surviving Sepsis Campaign:
International Guidelines for Management of Sepsis and Septic Shock 2021.
Crit Care Med. 2021;49(11):e1063-e1143.
```

**How to Extract**:
```yaml
# config/thresholds/universal/lactate.yaml
"2524-7": # LOINC for Lactate
  loinc_code: "2524-7"
  lab_name: "Lactate"
  normal:
    low: 0.5
    high: 2.0
  warning:
    high: 2.5
  critical:
    high: 4.0
  preferred_unit: "mmol/L"
  evidence_source: "Surviving Sepsis Campaign 2021"
  guideline_url: "https://www.sccm.org/SurvivingSepsisCampaign/Guidelines"
  last_updated: "2021-10-01"
```

---

#### **B. KDIGO (Kidney Disease) Guidelines** (Creatinine/Kidney Function)
**URL**: https://kdigo.org/guidelines/

**Key Data**:
- **Creatinine normal**: 0.6-1.2 mg/dL (general adult population)
- **AKI (Acute Kidney Injury) criteria**: Creatinine increase ≥0.3 mg/dL within 48 hours OR ≥1.5x baseline
- **CKD staging**: Based on eGFR calculations

**Citation**:
```
KDIGO 2012 Clinical Practice Guideline for the Evaluation and Management
of Chronic Kidney Disease. Kidney Int Suppl. 2013;3(1):1-150.
```

**How to Extract**:
```yaml
"2160-0": # Creatinine
  loinc_code: "2160-0"
  lab_name: "Creatinine"
  normal:
    low: 0.6
    high: 1.2
  warning:
    high: 1.5
  critical:
    high: 3.0
  preferred_unit: "mg/dL"
  evidence_source: "KDIGO CKD Guidelines 2012"
  guideline_url: "https://kdigo.org/guidelines/ckd-evaluation-and-management/"
```

---

#### **C. AHA/ACC Guidelines** (Cardiovascular Biomarkers)
**URL**: https://www.acc.org/guidelines

**Key Data**:
- **Troponin threshold**: Varies by assay (0.04 ng/mL for conventional, 0.014 ng/mL for high-sensitivity)
- **BNP threshold**: >400 pg/mL indicates heart failure
- **CK-MB threshold**: >25 U/L indicates myocardial injury

**Citation**:
```
Januzzi JL Jr, et al. 2017 ACC Expert Consensus Decision Pathway for
Optimization of Heart Failure Treatment. J Am Coll Cardiol. 2018;71(2):201-230.
```

---

#### **D. WHO Reference Values** (Global Baseline Hematology)
**URL**: https://www.who.int/publications

**Key Data**:
- **Hemoglobin**: Males 13-17 g/dL, Females 12-15 g/dL
- **WBC**: 4.0-11.0 × 10^9/L
- **Platelets**: 150-400 × 10^9/L

**How to Extract**:
```yaml
"6690-2": # WBC (White Blood Cell Count)
  loinc_code: "6690-2"
  lab_name: "Leukocytes (WBC)"
  normal:
    low: 4.0
    high: 11.0
  warning:
    low: 3.5
    high: 12.0
  critical:
    low: 2.0
    high: 30.0
  preferred_unit: "10*3/uL"
  evidence_source: "WHO Reference Values 2020"
```

---

### **Source 3: LabCorp/Quest Reference Ranges** (Commercial Lab Data)
**URL**:
- LabCorp: https://www.labcorp.com/help/patient-test-info/reference-ranges
- Quest Diagnostics: https://www.questdiagnostics.com/

**What You Get**:
- **Detailed reference ranges** for thousands of tests
- **Age-stratified ranges** (pediatric vs adult)
- **Sex-specific ranges**
- **Free to access** (no login required for most tests)

**Example - LabCorp Creatinine Reference**:
```
Test: Creatinine, Serum (LOINC 2160-0)
Reference Range (Adult):
  Male: 0.7 - 1.3 mg/dL
  Female: 0.6 - 1.1 mg/dL

Age Adjustments:
  Pediatric (1-18 years): 0.3 - 1.0 mg/dL
  Elderly (>65 years): Lower by ~10-15%
```

**How to Scrape** (Legally via their public API if available, or manual entry):
```python
# Example: Extract from LabCorp reference PDF
import pandas as pd

labcorp_data = {
    "2160-0": {  # Creatinine
        "male_range": (0.7, 1.3),
        "female_range": (0.6, 1.1),
        "unit": "mg/dL",
        "source": "LabCorp Reference Ranges 2024"
    }
}
```

---

### **Source 4: UpToDate Clinical Decision Support** (PAID - Comprehensive)
**URL**: https://www.uptodate.com/ (Subscription required: ~$500/year)

**What You Get**:
- **Curated clinical thresholds** for all major lab tests
- **Evidence-based ranges** updated continuously
- **Clinical context** for when thresholds vary
- **Structured data** (can be extracted programmatically)

**Example Entry**:
```
Topic: Serum Lactate - Interpretation
Normal: <2.0 mmol/L
Elevated: 2.0-4.0 mmol/L (hypoperfusion)
Severe: >4.0 mmol/L (shock)
Context: Serial lactate measurements more useful than single value
```

---

## Layer 2: Geographic/Regional Adjustments

### **Source 1: US - AHA/ACC Guidelines** (FREE)
**URL**: https://www.heart.org/en/health-topics/high-blood-pressure

**Key Geographic Difference**:
```yaml
# 2017 AHA/ACC lowered hypertension threshold from 140 to 130 mmHg
geographic_profiles:
  US:
    region: "US"
    regulatory_body: "FDA"
    guideline_version: "AHA/ACC 2017"
    adjustments:
      systolic_bp:
        offset_high: -10.0  # 130 mmHg instead of 140
        rationale: "AHA/ACC 2017 Hypertension Guideline"
        evidence_source: "Whelton PK, et al. J Am Coll Cardiol. 2018;71(19):e127-e248"
```

---

### **Source 2: India - ICMR Guidelines** (FREE)
**URL**: https://www.icmr.gov.in/

**Key Documents**:
- **India Hypertension Control Initiative (IHCI)**: Uses 140/90 mmHg threshold (not 130/80)
- **India Diabetes Guidelines**: HbA1c targets differ from ADA guidelines

**Example**:
```yaml
geographic_profiles:
  India:
    region: "India"
    regulatory_body: "CDSCO"
    guideline_version: "ICMR Hypertension Guidelines 2020"
    adjustments:
      systolic_bp:
        offset_high: 0.0  # Uses universal 140 mmHg threshold
        rationale: "ICMR maintains 140/90 threshold for Indian population"
```

---

### **Source 3: Europe - ESC Guidelines** (FREE)
**URL**: https://www.escardio.org/Guidelines

**Key Data**:
- **Hypertension threshold**: 140/90 mmHg (aligned with India, not US)
- **Diabetes HbA1c**: <7.0% (53 mmol/mol)
- **Lipid targets**: LDL <1.4 mmol/L for very high-risk patients

---

### **Source 4: WHO Global Guidelines** (FREE)
**URL**: https://www.who.int/health-topics/hypertension

**Use Case**: Fallback for countries without specific guidelines

```yaml
geographic_profiles:
  WHO:
    region: "WHO"
    regulatory_body: "WHO"
    guideline_version: "WHO Global Guidelines 2021"
    adjustments:
      systolic_bp:
        offset_high: 0.0  # Uses 140/90 universal threshold
        rationale: "WHO Global HTN Guidelines"
```

---

## Layer 3: Demographic Adjustments

### **Source 1: Cockcroft-Gault & MDRD Equations** (Age/Sex Creatinine Adjustment)
**Reference**: FREE - Published formulas

**Cockcroft-Gault Formula** (Creatinine Clearance):
```
CrCl (mL/min) = [(140 - Age) × Weight (kg)] / [72 × Serum Cr (mg/dL)]
                × 0.85 (if female)
```

**Key Insight**: Elderly have lower creatinine production due to decreased muscle mass.

**Age Adjustment Rules**:
```yaml
demographic_adjustments:
  age_rules:
    - loinc_code: "2160-0" # Creatinine
      age_min: 65
      age_max: 120
      adjustment:
        offset_high: -0.2  # Lower threshold by 0.2 mg/dL
        rationale: "Decreased muscle mass with aging (Cockcroft-Gault)"
        evidence_source: "Cockcroft DW, Gault MH. Nephron. 1976;16(1):31-41"
```

---

### **Source 2: Sex-Specific Reference Ranges** (LabCorp/Quest/WHO)

**LabCorp Sex-Specific Ranges**:
```yaml
demographic_adjustments:
  sex_rules:
    - loinc_code: "2160-0" # Creatinine
      sex: "MALE"
      adjustment:
        offset_high: 0.1  # Males: 0.7-1.3 vs Females: 0.6-1.1
        rationale: "Higher muscle mass in males"
        evidence_source: "LabCorp Reference Ranges 2024"

    - loinc_code: "718-7" # Hemoglobin
      sex: "FEMALE"
      adjustment:
        offset_high: -2.0  # Females: 12-15 g/dL vs Males: 13-17 g/dL
        rationale: "Menstruation and lower androgen levels"
        evidence_source: "WHO Hemoglobin Reference Values"
```

---

### **Source 3: Ethnicity Adjustments** (MDRD & CKD-EPI eGFR Equations)

**IMPORTANT NOTE**: The **race coefficient** in eGFR equations is **controversial** and being phased out in many institutions.

**Historical CKD-EPI Formula** (Being Replaced):
```
eGFR = 141 × min(SCr/κ, 1)^α × max(SCr/κ, 1)^-1.209 × 0.993^Age
       × 1.018 [if female] × 1.159 [if Black]
```

**NEW 2021 CKD-EPI Formula** (Race-Free):
Uses **cystatin C** or **creatinine + cystatin C** instead of race coefficient.

**Current Approach** (Until Race-Free Adoption):
```yaml
demographic_adjustments:
  ethnicity_rules:
    - loinc_code: "2160-0" # Creatinine
      ethnicity: "AFRICAN"
      adjustment:
        offset_high: 0.15  # ~10-15% higher baseline
        rationale: "Higher muscle mass in African ancestry populations (CKD-EPI 2009)"
        evidence_source: "Levey AS, et al. Ann Intern Med. 2009;150(9):604-612"
        note: "CONTROVERSIAL - Being phased out per ASN/NKF 2021 recommendations"
```

**Recommendation**: **Do NOT use race-based adjustments** unless clinical team explicitly approves. Use cystatin C-based eGFR instead.

---

### **Source 4: BMI Adjustments** (Research Papers)

**Athletes & High Muscle Mass**:
```yaml
demographic_adjustments:
  bmi_rules:
    - loinc_code: "2160-0" # Creatinine
      bmi_min: 25.0
      bmi_max: 35.0
      adjustment:
        offset_high: 0.2  # Athletes may have Cr up to 1.4-1.5 mg/dL
        rationale: "Higher muscle mass in athletic populations"
        evidence_source: "Perrone RD, et al. Clin Chem. 1992;38(10):1933-1953"
        note: "Only apply if patient is athletic (requires clinical context)"
```

---

## Layer 4: Contextual Adjustments

### **Source 1: Pregnancy Reference Ranges** (ACOG Guidelines)
**URL**: https://www.acog.org/clinical/clinical-guidance

**ACOG Practice Bulletin No. 203** (Chronic Hypertension in Pregnancy):
```yaml
contextual_adjustments:
  pregnancy_rules:
    - loinc_code: "2823-3" # Potassium
      trimester: 0  # All trimesters
      adjustment:
        offset_low: -0.3  # Pregnancy K+ 3.0-3.5 mmol/L normal
        offset_high: -0.2
        rationale: "Hemodilution and increased renal clearance"
        evidence_source: "ACOG Practice Bulletin No. 203, 2019"

    - loinc_code: "6690-2" # WBC
      trimester: 3  # Third trimester
      adjustment:
        offset_high: 5.0  # Pregnancy WBC can be 5-16 K/uL
        rationale: "Physiological leukocytosis in late pregnancy"
        evidence_source: "Abbassi-Ghanavati M, et al. Obstet Gynecol. 2009;114(6):1326-1331"
```

---

### **Source 2: CKD Guidelines** (KDIGO - Chronic Kidney Disease Adjustments)

**KDIGO 2012 CKD Guidelines**:
```yaml
contextual_adjustments:
  chronic_condition_rules:
    - loinc_code: "2160-0" # Creatinine
      condition_code: "585.3" # ICD-10: CKD Stage 3
      condition_name: "Chronic Kidney Disease Stage 3"
      adjustment:
        offset_high: 0.3  # Accept Cr up to 1.5 mg/dL as "stable"
        rationale: "Stable CKD patients may have baseline Cr 1.3-1.5 mg/dL"
        evidence_source: "KDIGO 2012 CKD Guidelines"
        note: "Acute rise >0.3 mg/dL still triggers AKI alert"

    - loinc_code: "2823-3" # Potassium
      condition_code: "585.3" # CKD Stage 3
      condition_name: "Chronic Kidney Disease Stage 3"
      adjustment:
        offset_high: 0.3  # Accept K+ up to 5.8 mEq/L
        rationale: "Reduced renal K+ excretion in CKD"
        evidence_source: "Palmer BF. Clin J Am Soc Nephrol. 2015;10(6):1050-1060"
```

---

### **Source 3: Medication-Induced Adjustments** (DrugBank, Lexicomp, Micromedex)

#### **A. DrugBank** (FREE Academic License)
**URL**: https://go.drugbank.com/

**What You Get**:
- **RxNorm codes** for medication identification
- **Drug-lab interactions**: Which labs are affected by which drugs
- **Mechanism of action**: Why the interaction occurs

**Example - ACE Inhibitors**:
```
Drug: Lisinopril (RxNorm: 29046)
Lab Interactions:
  - Creatinine: Increase 10-20% (expected, not harmful)
  - Potassium: Increase 0.3-0.5 mEq/L (monitor for hyperkalemia)
  - Hemoglobin: May decrease slightly (rare)
```

**Configuration**:
```yaml
contextual_adjustments:
  medication_rules:
    - loinc_code: "2160-0" # Creatinine
      rx_norm_code: "29046" # Lisinopril (ACE-I)
      medication_name: "Lisinopril"
      adjustment:
        offset_high: 0.15  # Accept 10-15% increase
        rationale: "ACE-I reduces glomerular filtration pressure (expected drug effect)"
        evidence_source: "DrugBank Lisinopril Monograph + Palmer BF. Am J Kidney Dis. 2002;40(2):265-274"

    - loinc_code: "2823-3" # Potassium
      rx_norm_code: "29046" # Lisinopril
      medication_name: "Lisinopril"
      adjustment:
        offset_high: 0.4  # Accept K+ up to 5.9 mEq/L
        rationale: "ACE-I reduces renal K+ excretion (monitor for hyperkalemia)"
        evidence_source: "Palmer BF. N Engl J Med. 2004;351(6):585-592"
```

---

#### **B. Lexicomp / Micromedex** (PAID - Hospital Subscriptions)
**URL**:
- Lexicomp: https://www.wolterskluwer.com/en/solutions/lexicomp
- Micromedex: https://www.micromedexsolutions.com/

**Cost**: ~$1000-5000/year (hospital-wide license)

**What You Get**:
- **Comprehensive drug-lab interactions**
- **Clinical significance ratings**
- **Management recommendations**
- **Structured data export** (API access)

---

### **Source 4: Altitude Adjustments** (Research Literature)

**High-Altitude Physiology**:
```
Altitude → Lower atmospheric O2 → Compensatory polycythemia + higher hemoglobin
Altitude > 2500m → SpO2 baseline 88-92% (sea level: 95-100%)
```

**Reference**:
```
West JB. High altitude medicine. Am J Respir Crit Care Med. 2012;186(12):1229-1237.
```

**Configuration**:
```yaml
contextual_adjustments:
  altitude_rules:
    - loinc_code: "59408-5" # Oxygen Saturation
      altitude_meters: 2500  # 2500m = ~8200 feet
      adjustment:
        offset_low: -7.0  # Accept SpO2 down to 88% at high altitude
        rationale: "Physiological adaptation to high altitude (West 2012)"
        evidence_source: "West JB. Am J Respir Crit Care Med. 2012;186(12):1229-1237"

    - loinc_code: "718-7" # Hemoglobin
      altitude_meters: 2500
      adjustment:
        offset_high: 2.0  # Accept Hgb up to 19 g/dL at high altitude
        rationale: "Compensatory polycythemia at altitude"
        evidence_source: "León-Velarde F, et al. High Alt Med Biol. 2005;6(2):147-157"
```

---

## Practical Data Collection Strategy

### **Phase 1: Minimum Viable Configuration** (2-3 weeks)

**Goal**: Cover 80% of alerts with 20% of effort

**Priority Labs** (20 most common):
1. Creatinine (2160-0)
2. Potassium (2823-3)
3. Sodium (2951-2)
4. Glucose (2345-7)
5. Hemoglobin (718-7)
6. WBC (6690-2)
7. Platelets (777-3)
8. Lactate (2524-7)
9. Troponin I (10839-9)
10. BNP (30934-4)
11. ALT (1742-6)
12. AST (1920-8)
13. Bilirubin (1975-2)
14. Albumin (1751-7)
15. INR (6301-6)
16. SpO2 (59408-5)
17. Heart Rate (8867-4)
18. Blood Pressure Systolic (8480-6)
19. Respiratory Rate (9279-1)
20. Temperature (8310-5)

**Data Sources for MVP**:
1. **Universal thresholds**: LOINC + LabCorp reference ranges (free)
2. **Geographic**: US (AHA/ACC 2017) + India (ICMR 2020) + WHO fallback
3. **Demographic**: Age + Sex adjustments only (from LabCorp)
4. **Contextual**: Top 5 medications (ACE-I, ARB, Diuretics, Metformin, Statins) + Pregnancy

**Estimated Data Collection Time**: 40-60 hours (2-3 weeks part-time)

---

### **Phase 2: Production-Grade Configuration** (8-12 weeks)

**Goal**: Cover 95%+ of clinical scenarios

**Additional Data Sources**:
1. **UpToDate subscription** ($500/year) - Comprehensive clinical context
2. **DrugBank academic license** (free) - Medication interactions for all drugs
3. **Specialty guidelines**:
   - Cardiology: ACC/AHA full guideline library
   - Nephrology: KDIGO full guidelines
   - Critical Care: SCCM/ESICM guidelines
   - Obstetrics: ACOG practice bulletins
4. **Local lab calibration data**: Partner with hospital labs for assay-specific thresholds

---

### **Phase 3: AI-Assisted Data Extraction** (Ongoing)

**Use LLMs to Extract Data**:
```python
# Example: Use Claude/GPT-4 to extract thresholds from PDFs
import anthropic

client = anthropic.Anthropic(api_key="...")

guideline_pdf = load_pdf("AHA_Hypertension_Guideline_2017.pdf")

response = client.messages.create(
    model="claude-3-5-sonnet-20241022",
    max_tokens=4096,
    messages=[{
        "role": "user",
        "content": f"""Extract clinical thresholds from this guideline:

        {guideline_pdf}

        Format as YAML:
        - Lab test name
        - LOINC code (if available)
        - Normal range (low-high)
        - Warning threshold
        - Critical threshold
        - Units
        - Evidence citation
        """
    }]
)

# Parse response into YAML configuration
yaml_config = response.content[0].text
```

---

## Data Maintenance Strategy

### **Version Control**:
```yaml
threshold_metadata:
  version: "1.0.0"
  effective_date: "2025-01-01"
  next_review_date: "2026-01-01"
  changelog:
    - date: "2025-01-01"
      change: "Initial version based on AHA/ACC 2017, KDIGO 2012"
    - date: "2025-06-15"
      change: "Updated troponin threshold per Abbott hs-cTn assay"
```

### **Annual Review Process**:
1. **Q1**: Review new clinical guidelines published in prior year
2. **Q2**: Update configurations based on new evidence
3. **Q3**: Validate changes with clinical team
4. **Q4**: Deploy updated thresholds with A/B testing

---

## Free vs Paid Data Summary

### **FREE Sources** (Sufficient for MVP):
- ✅ LOINC database (lab identifiers + some reference ranges)
- ✅ LabCorp/Quest reference ranges (publicly available)
- ✅ WHO guidelines (global baselines)
- ✅ AHA/ACC guidelines (cardiovascular - free PDFs)
- ✅ KDIGO guidelines (kidney disease - free PDFs)
- ✅ Surviving Sepsis Campaign (sepsis/lactate - free)
- ✅ ACOG guidelines (pregnancy - free summaries)
- ✅ DrugBank academic license (drug-lab interactions)
- ✅ PubMed research papers (demographic adjustments)

**Estimated Cost**: $0 (just labor for data entry)

### **PAID Sources** (Better for Production):
- 💰 UpToDate ($500/year) - Comprehensive clinical context
- 💰 Lexicomp/Micromedex ($1000-5000/year) - Detailed drug interactions
- 💰 Elsevier ClinicalKey ($300-1000/year) - Broad guideline access

**Estimated Cost**: $1800-6500/year

---

## Recommended Starting Point

**Week 1-2**: Setup
1. Download LOINC database
2. Scrape LabCorp reference ranges for top 20 labs
3. Download free AHA/ACC, KDIGO, SCCM guidelines

**Week 3-4**: Data Entry
1. Create universal thresholds YAML for 20 priority labs
2. Create US + India geographic profiles
3. Create age/sex demographic rules for creatinine, hemoglobin

**Week 5-6**: Validation
1. Implement ClinicalThresholdResolver
2. Unit test with sample patients
3. Compare output to existing hardcoded thresholds

**Total Time**: 6 weeks for MVP covering 80% of use cases

---

## Example: Complete Creatinine Configuration from Free Sources

```yaml
# Assembled from: LOINC + LabCorp + KDIGO + Cockcroft-Gault
"2160-0": # Creatinine
  # Layer 1: Universal Baseline (LOINC + LabCorp)
  loinc_code: "2160-0"
  lab_name: "Creatinine, Serum"
  normal:
    low: 0.6
    high: 1.2
  warning:
    high: 1.5
  critical:
    high: 3.0
  preferred_unit: "mg/dL"
  evidence_source: "KDIGO CKD Guidelines 2012 + LabCorp 2024"
  last_updated: "2024-01-15"

  # Layer 2: Geographic (No adjustment for Creatinine)
  # Layer 3: Demographics
  demographic_adjustments:
    sex:
      male:
        offset_high: 0.1  # Males: 0.7-1.3 mg/dL
        source: "LabCorp Reference Ranges 2024"
      female:
        offset_high: -0.1  # Females: 0.6-1.1 mg/dL
        source: "LabCorp Reference Ranges 2024"
    age:
      - age_min: 65
        age_max: 120
        offset_high: -0.2  # Elderly: Lower threshold
        source: "Cockcroft DW, Gault MH. Nephron. 1976;16(1):31-41"

  # Layer 4: Contextual
  contextual_adjustments:
    medications:
      - rx_norm_code: "29046"  # Lisinopril (ACE-I)
        offset_high: 0.15
        source: "Palmer BF. Am J Kidney Dis. 2002;40(2):265-274"
      - rx_norm_code: "83367"  # Valsartan (ARB)
        offset_high: 0.15
        source: "Palmer BF. Am J Kidney Dis. 2002;40(2):265-274"
    chronic_conditions:
      - icd10_code: "N18.3"  # CKD Stage 3
        offset_high: 0.3
        source: "KDIGO 2012 CKD Guidelines"
```

**Data Sources Used** (All FREE):
1. LOINC: Lab identifier
2. LabCorp: Reference ranges, sex-specific
3. KDIGO: CKD thresholds, chronic condition adjustments
4. Cockcroft-Gault: Age-based adjustment formula
5. PubMed (Palmer paper): ACE-I/ARB medication adjustment

**Total Cost**: $0
**Time to Assemble**: ~2 hours

---

## Conclusion

**You can build a global, production-grade threshold system using 100% FREE data sources.**

**Recommended Path**:
1. **Start with free sources** (LOINC + LabCorp + Clinical Guidelines)
2. **Build MVP** covering 20 most common labs (~80% of alerts)
3. **Validate with clinical team**
4. **Expand coverage** to 50+ labs over 3-6 months
5. **Consider paid subscriptions** (UpToDate, Lexicomp) only after MVP proves value

**Time Investment**: ~40-60 hours for MVP, ~200-300 hours for comprehensive coverage

**Monetary Investment**: $0 for MVP, $1800-6500/year for production-grade paid subscriptions (optional)
