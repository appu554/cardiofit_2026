# KB-5 DDI Authoritative Sources Reference

## Production-Grade Data Sources

This document verifies the authoritative sources used for Drug-Drug Interaction (DDI) governance data. All sources are **real, production-grade clinical references** used by hospital pharmacy systems globally.

---

## Layer 1: REGULATORY AUTHORITIES

### FDA DailyMed (United States)
- **URL**: https://dailymed.nlm.nih.gov/dailymed/
- **What it is**: Official FDA drug labeling database
- **Section Used**: Section 7 (Drug Interactions) of each drug label
- **Authority**: U.S. Food and Drug Administration
- **Access**: Free, public access
- **Update Frequency**: Continuous as labels are updated

**Example Verification**:
```
Warfarin Drug Label → Section 7 Drug Interactions
https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=e97c0469-06cc-4a64-8ee3-12f8f90a9a30
```

### TGA (Australia)
- **URL**: https://www.tga.gov.au/products/medicines
- **What it is**: Australian equivalent of FDA
- **Document**: Product Information (PI) documents
- **Authority**: Therapeutic Goods Administration

### EMA (European Union)
- **URL**: https://www.ema.europa.eu/en/medicines
- **What it is**: European Medicines Agency
- **Document**: Summary of Product Characteristics (SmPC)
- **Authority**: European regulatory body

### CDSCO (India)
- **URL**: https://cdsco.gov.in/
- **What it is**: Central Drugs Standard Control Organisation
- **Document**: Prescribing Information
- **Authority**: Ministry of Health & Family Welfare, India

---

## Layer 2: PHARMACOLOGY AUTHORITIES

### DrugBank
- **URL**: https://go.drugbank.com/
- **What it is**: Comprehensive drug database with pharmacokinetic data
- **Data Available**:
  - CYP enzyme Ki values (inhibition constants)
  - IC50 values for enzyme inhibition
  - Transporter substrates/inhibitors (P-gp, OATP, OAT)
  - Protein binding data
- **Authority**: University of Alberta / OMx Personal Health Analytics
- **Citation**: Wishart DS et al. DrugBank 5.0. Nucleic Acids Res. 2018

**Example Data Points Used**:
```
Fluconazole CYP2C9 inhibition: Ki = 7.1 μM
Source: DrugBank DB00196
https://go.drugbank.com/drugs/DB00196

Ketoconazole CYP3A4 inhibition: Ki = 0.015 μM
Source: DrugBank DB01026
https://go.drugbank.com/drugs/DB01026
```

### Indiana University Flockhart CYP Table
- **URL**: https://drug-interactions.medicine.iu.edu/MainTable.aspx
- **What it is**: Gold-standard CYP450 enzyme interaction reference
- **Data Available**:
  - CYP enzyme substrates (major/minor)
  - CYP inhibitors (strong/moderate/weak)
  - CYP inducers (strong/moderate/weak)
- **Authority**: Indiana University School of Medicine
- **Citation**: Flockhart DA. Drug Interactions: Cytochrome P450 Drug Interaction Table

**Why It's Authoritative**:
- Referenced in FDA drug interaction guidance documents
- Used by clinical pharmacology textbooks
- Updated regularly based on peer-reviewed literature

### CredibleMeds (Arizona CERT)
- **URL**: https://crediblemeds.org/
- **What it is**: QT prolongation risk classification database
- **Categories**:
  - **Known Risk**: Drugs that prolong QT and cause TdP
  - **Possible Risk**: Drugs that may prolong QT
  - **Conditional Risk**: QT risk under certain conditions
  - **Avoid if congenital LQTS**: Special population warning
- **Authority**: Arizona Center for Education and Research on Therapeutics
- **Citation**: Woosley RL, et al. CredibleMeds.org. QTDrugs List.

**Example Classifications Used**:
```
Amiodarone: Known Risk (Category 1)
Methadone: Known Risk (Category 1)
Levofloxacin: Possible Risk (Category 2)
Haloperidol: Known Risk (Category 1)
```

### University of Washington DDI Database
- **URL**: https://www.druginteractioninfo.org/
- **What it is**: Research-grade DDI database with transporter data
- **Data Available**:
  - P-glycoprotein (P-gp/ABCB1) interactions
  - OATP1B1/1B3 transporter interactions
  - OAT1/OAT3 interactions
  - BCRP interactions
- **Authority**: UW Drug Interaction Database (DIDB)
- **Access**: Academic/commercial licensing

### PharmGKB
- **URL**: https://www.pharmgkb.org/
- **What it is**: Pharmacogenomics knowledge resource
- **Data Available**:
  - Gene-drug relationships
  - CYP2C19, CYP2D6 poor/rapid metabolizer impacts
  - Dosing guidelines based on genotype
- **Authority**: Stanford University
- **Citation**: Whirl-Carrillo M, et al. Clin Pharmacol Ther. 2012

---

## Layer 3: CLINICAL PRACTICE AUTHORITIES

### Lexicomp (Wolters Kluwer)
- **URL**: https://www.wolterskluwer.com/en/solutions/lexicomp
- **What it is**: Industry-standard clinical drug reference
- **Data Available**:
  - DDI severity ratings (A, B, C, D, X)
  - Management recommendations
  - Evidence citations
- **Authority**: Wolters Kluwer Health
- **Used By**: >75% of US hospitals

**Severity Rating Scale**:
```
A = No known interaction
B = Minor; no action needed
C = Moderate; monitor therapy
D = Major; consider therapy modification
X = Contraindicated; avoid combination
```

### Micromedex (IBM/Merative)
- **URL**: https://www.micromedexsolutions.com/
- **What it is**: Clinical decision support database
- **Data Available**:
  - DDI severity and documentation ratings
  - Onset timing (rapid, delayed)
  - Clinical management
- **Authority**: Merative (formerly IBM Watson Health)
- **Used By**: Major health systems globally

### Stockley's Drug Interactions
- **URL**: https://about.medicinescomplete.com/publication/stockleys-drug-interactions/
- **What it is**: Comprehensive DDI reference text
- **Publisher**: Pharmaceutical Press (Royal Pharmaceutical Society)
- **Authority**: Gold-standard DDI reference in UK/Commonwealth
- **Citation**: Baxter K. Stockley's Drug Interactions. 12th ed.

### Australian Medicines Handbook (AMH)
- **URL**: https://amhonline.amh.net.au/
- **What it is**: Independent Australian drug reference
- **Authority**: Australian Medicines Handbook Pty Ltd
- **Governance**: Independent of pharmaceutical industry

### British National Formulary (BNF)
- **URL**: https://bnf.nice.org.uk/
- **What it is**: UK national drug formulary
- **Publisher**: BMJ Group and Pharmaceutical Press
- **Authority**: NICE (National Institute for Health and Care Excellence)
- **Access**: Free online access for UK healthcare

---

## Clinical Practice Guidelines Used

### Cardiovascular
- **ACC/AHA Lipid Guidelines 2018**: Statin dosing recommendations
- **ACC/AHA DAPT Guidelines 2020**: Dual antiplatelet therapy
- **AHA/ACC QT Statement 2022**: QT prolongation management

### Anticoagulation
- **ACCP Antithrombotic Guidelines 2024**: 9th Edition
- **ISTH DOAC Guidance 2023**: Direct oral anticoagulants

### Infectious Disease
- **IDSA Guidelines**: Various infection-specific recommendations
- **SAMHSA Methadone Guidelines 2023**: OUD treatment

### Pain/Psychiatry
- **CDC Opioid Prescribing Guidelines 2022**
- **FDA Black Box Warnings**: Opioid + benzodiazepine
- **APA Practice Guidelines**: Depression treatment

### Nephrology
- **KDIGO AKI Guidelines 2022**: Acute kidney injury
- **ACR Contrast Manual 2023**: v10.3 metformin guidance

### Diabetes
- **ADA Standards of Care 2024**: Diabetes management

---

## Data Validation Process

### How to Verify DDI Data

1. **Regulatory Layer Verification**:
   ```bash
   # Look up drug on DailyMed
   https://dailymed.nlm.nih.gov/dailymed/search.cfm?query={DRUG_NAME}
   # Navigate to Section 7: Drug Interactions
   ```

2. **Pharmacology Layer Verification**:
   ```bash
   # DrugBank lookup for Ki/IC50 values
   https://go.drugbank.com/drugs/DB{NUMBER}

   # Flockhart Table for CYP classification
   https://drug-interactions.medicine.iu.edu/

   # CredibleMeds for QT risk
   https://crediblemeds.org/pdftemp/pdf/CombinedList.pdf
   ```

3. **Clinical Layer Verification**:
   - Lexicomp/Micromedex: Institutional subscription required
   - BNF: https://bnf.nice.org.uk/interactions/
   - Stockley: Available via MedicinesComplete subscription

---

## Production Deployment Considerations

### Licensing Requirements

| Source | License Type | Production Use |
|--------|-------------|----------------|
| FDA DailyMed | Public Domain | ✅ Free |
| DrugBank | Academic/Commercial | ⚠️ License needed for commercial |
| CredibleMeds | Research/Clinical | ✅ Free with attribution |
| Flockhart Table | Academic | ✅ Free with citation |
| Lexicomp | Commercial | ⚠️ Subscription required |
| Micromedex | Commercial | ⚠️ Subscription required |
| BNF | NHS/Commercial | ⚠️ License for non-UK |
| AMH | Commercial | ⚠️ Subscription required |

### Recommended Approach for Production

1. **Minimum Viable (Free Sources)**:
   - FDA DailyMed for regulatory
   - DrugBank (academic license) for pharmacology
   - CredibleMeds (free) for QT classification
   - Flockhart Table (free) for CYP data

2. **Full Commercial Deployment**:
   - Lexicomp or Micromedex subscription
   - AMH for Australian market
   - BNF for UK market
   - DrugBank commercial license

---

## Update Schedule

| Source | Update Frequency | KB-5 Sync Recommendation |
|--------|-----------------|-------------------------|
| FDA DailyMed | Continuous | Weekly check |
| CredibleMeds | Quarterly | Quarterly sync |
| Flockhart Table | As needed | Monthly check |
| DrugBank | Bi-annual major | Bi-annual sync |
| Lexicomp | Daily | Real-time integration preferred |

---

## References

1. Wishart DS, et al. DrugBank 5.0: a major update. Nucleic Acids Res. 2018;46(D1):D1074-D1082.
2. Flockhart DA. Drug Interactions: Cytochrome P450 Drug Interaction Table. Indiana University School of Medicine.
3. Woosley RL, et al. CredibleMeds.org, QTDrugs List. AZCERT, Inc.
4. Baxter K. Stockley's Drug Interactions. 12th ed. Pharmaceutical Press; 2019.
5. FDA Guidance: Drug Interaction Studies. January 2020.
