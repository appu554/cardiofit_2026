# Production Data Download Guide

**Phase 1: Full Dataset Acquisition**

This guide walks you through downloading the complete production datasets for the Clinical Knowledge OS.

---

## 1. ONC High-Priority DDI (~1,200 pairs)

### Source Information
- **Organization**: Office of the National Coordinator for Health IT (ONC/HHS)
- **License**: Public Domain (US Government Work)
- **Update Frequency**: Quarterly

### Download Steps

1. **Navigate to ONC Website**
   ```
   https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction
   ```

2. **Find the DDI Spreadsheet**
   - Look for "High Priority Drug-Drug Interactions" or "DDI Clinical Decision Support"
   - The file is typically an Excel (.xlsx) or CSV format
   - Current version should have ~1,200 drug pairs

3. **Alternative: AHRQ CDS Connect**
   ```
   https://cds.ahrq.gov/cdsconnect
   ```
   - Search for "drug-drug interaction"
   - Download the CDS artifact which includes the DDI list

4. **Transform to Required Format**

   Your CSV must have these columns:
   ```
   Drug1_RXCUI,Drug1_Name,Drug2_RXCUI,Drug2_Name,Severity,Clinical_Effect,Management,Evidence_Level,Documentation,Clinical_Source,ONC_Pair_ID,Last_Updated
   ```

   **Severity Values** (must match exactly):
   - `CONTRAINDICATED`
   - `HIGH`
   - `MODERATE`
   - `LOW`

5. **Save the File**
   ```bash
   # Save as:
   /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/cmd/phase1-ingest/data/onc_ddi.csv
   ```

### Validation Command
```bash
# Check row count (should be ~1,200)
wc -l data/onc_ddi.csv

# Verify header format
head -1 data/onc_ddi.csv
```

---

## 2. CMS Medicare Part D Formulary (~100K entries)

### Source Information
- **Organization**: Centers for Medicare & Medicaid Services (CMS)
- **License**: Public Domain (US Government Work)
- **Update Frequency**: Quarterly

### Download Steps

1. **Navigate to CMS Data Portal**
   ```
   https://data.cms.gov
   ```

2. **Search for Formulary Data**
   - Search: "Medicare Part D Formulary"
   - Or navigate: Provider Data → Part D → Formulary Reference File

3. **Direct Download Links** (check for latest version)
   ```
   # 2024 Basic Drugs File
   https://data.cms.gov/provider-summary-by-type-of-service/medicare-part-d-prescribers

   # Alternative: Medicare Plan Finder Files
   https://www.cms.gov/medicare/search?keywords=formulary%20file
   ```

4. **Download the Files**
   - Look for: `Basic_Drugs_Q4_2024.csv` or similar
   - Or: `Formulary_Reference_File.zip`

5. **Transform to Required Format**

   Your CSV must have these columns:
   ```
   CONTRACT_ID,PLAN_ID,RXCUI,NDC,DRUG_NAME,TIER_LEVEL_CODE,QUANTITY_LIMIT,QUANTITY_LIMIT_AMOUNT,QUANTITY_LIMIT_DAYS,PRIOR_AUTH,STEP_THERAPY,COVERAGE_STATUS,EFFECTIVE_YEAR
   ```

   **Tier Level Codes**:
   - `1` = Generic
   - `2` = Preferred Brand
   - `3` = Non-Preferred Brand
   - `4` = Specialty Tier
   - `5` = Specialty Tier (highest)

   **Boolean Fields** (PRIOR_AUTH, STEP_THERAPY):
   - `true` or `false` (lowercase)

   **Coverage Status**:
   - `COVERED`
   - `NOT_COVERED`
   - `PRIOR_AUTH_REQUIRED`

6. **Save the File**
   ```bash
   # Save as:
   /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/cmd/phase1-ingest/data/cms_formulary.csv
   ```

### Validation Command
```bash
# Check row count (should be ~100K)
wc -l data/cms_formulary.csv

# Check unique RxCUIs
cut -d',' -f3 data/cms_formulary.csv | sort -u | wc -l
```

---

## 3. LOINC Lab Reference Ranges (~2,000 ranges)

### Source Information
- **Organization**: Regenstrief Institute
- **License**: LOINC License (free registration required)
- **Update Frequency**: Biannual

### Registration Steps

1. **Create LOINC Account**
   ```
   https://loinc.org/get-loinc/
   ```
   - Click "Get LOINC"
   - Fill out registration form (free)
   - Verify email

2. **Download LOINC Files**

   After login, download:
   - `LoincTableCore.csv` - Main LOINC table
   - `LoincPartLink.csv` - Part linkages
   - `PanelsAndForms.csv` - Lab panels

3. **Extract Lab Reference Ranges**

   LOINC doesn't include reference ranges directly. You need to:

   a) **Use LOINC Lab Panels** - Extract common lab codes

   b) **Merge with NHANES Data** - CDC provides population reference ranges
      ```
      https://wwwn.cdc.gov/nchs/nhanes/
      ```

   c) **Use Clinical Guidelines** - Standard medical references for ranges

4. **Transform to Required Format**

   Your CSV must have these columns:
   ```
   loinc_code,component,property,time_aspect,system,scale_type,method_type,class,short_name,long_name,unit,low_normal,high_normal,critical_low,critical_high,age_group,sex,clinical_category,interpretation_guidance,delta_check_percent,delta_check_hours,deprecated
   ```

   **Age Groups**:
   - `ADULT` (18+)
   - `PEDIATRIC` (0-17)
   - `NEONATE` (0-28 days)
   - `ALL`

   **Sex**:
   - `M`, `F`, `ALL`

   **Clinical Categories**:
   - `hematology`, `chemistry`, `coagulation`, `endocrine`, `lipid`, `liver`, `renal`, `cardiac`, `inflammatory`, `immunology`, etc.

5. **Save the File**
   ```bash
   # Save as:
   /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/cmd/phase1-ingest/data/loinc_labs.csv
   ```

### Validation Command
```bash
# Check row count (should be ~2,000)
wc -l data/loinc_labs.csv

# Check unique LOINC codes
cut -d',' -f1 data/loinc_labs.csv | sort -u | wc -l
```

---

## 4. OHDSI Athena DDI (~200K pairs)

### Source Information
- **Organization**: OHDSI (Observational Health Data Sciences and Informatics)
- **License**: OHDSI License (free registration required)
- **Update Frequency**: Monthly

### Registration Steps

1. **Create OHDSI Athena Account**
   ```
   https://athena.ohdsi.org
   ```
   - Click "Sign Up"
   - Complete registration
   - Verify email

2. **Request Vocabulary Download**

   After login:
   - Click "Download" tab
   - Select vocabularies:
     - ✅ RxNorm
     - ✅ RxNorm Extension
     - ✅ ATC (for drug classes)
   - Click "Download Vocabularies"
   - Wait for email (can take 1-24 hours)

3. **Download the Bundle**

   When ready, you'll receive a link to download a ZIP file containing:
   - `CONCEPT.csv` (~2-5 GB)
   - `CONCEPT_RELATIONSHIP.csv` (~1-3 GB)
   - Other vocabulary files

4. **Extract Required Files**
   ```bash
   # Create OHDSI directory
   mkdir -p data/ohdsi

   # Extract from ZIP
   unzip athena_vocab_*.zip -d data/ohdsi/

   # Verify files exist
   ls -la data/ohdsi/CONCEPT.csv data/ohdsi/CONCEPT_RELATIONSHIP.csv
   ```

5. **No Transformation Needed**

   The Go ETL (`extraction/etl/ohdsi_ddi.go`) will:
   - Parse CONCEPT.csv for drug concepts
   - Extract DDI relationships from CONCEPT_RELATIONSHIP.csv
   - Map OHDSI concept IDs to RxCUIs
   - Generate ~200K DDI pairs

### Validation Command
```bash
# Check files exist and have content
wc -l data/ohdsi/CONCEPT.csv
wc -l data/ohdsi/CONCEPT_RELATIONSHIP.csv

# Should see millions of lines
# CONCEPT.csv: ~5-10 million rows
# CONCEPT_RELATIONSHIP.csv: ~20-50 million rows
```

---

## Quick Reference: File Locations

After downloading, your data directory should look like:

```
data/
├── MANIFEST.yaml          # Provenance tracking (update after downloads)
├── onc_ddi.csv            # ~1,200 rows (ONC DDI)
├── cms_formulary.csv      # ~100,000 rows (CMS Formulary)
├── loinc_labs.csv         # ~2,000 rows (LOINC Labs)
└── ohdsi/
    ├── CONCEPT.csv        # ~5-10 million rows
    └── CONCEPT_RELATIONSHIP.csv  # ~20-50 million rows
```

---

## After Download: Run Full Ingestion

Once all files are downloaded:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared

# Update checksums in MANIFEST.yaml
shasum -a 256 cmd/phase1-ingest/data/*.csv

# Run the full ingestion
./cmd/phase1-ingest/run_production_ingestion.sh

# Or use the Go CLI for OHDSI (larger dataset)
go run cmd/phase1-ingest/main.go --all
```

---

## Estimated Download Sizes

| Dataset | Download Size | Extracted Size | Time to Download |
|---------|---------------|----------------|------------------|
| ONC DDI | ~500 KB | ~500 KB | < 1 minute |
| CMS Formulary | ~50 MB | ~100 MB | 2-5 minutes |
| LOINC Full | ~500 MB | ~1 GB | 10-20 minutes |
| OHDSI Athena | ~2 GB | ~8-10 GB | 30-60 minutes |

---

## Troubleshooting

### ONC DDI
- If Excel format, save as CSV with UTF-8 encoding
- Remove any BOM (byte order mark) from the file

### CMS Formulary
- Large files may timeout - use download manager
- Files may be in ZIP format - extract first

### LOINC
- Registration may take 1-2 business days for approval
- Reference ranges require external sources (NHANES, clinical guidelines)

### OHDSI
- Download request may take up to 24 hours to process
- Files are very large - ensure adequate disk space (15+ GB)
- Consider using `pigz` for faster decompression
