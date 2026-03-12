#!/bin/bash
# =============================================================================
# PRODUCTION DATA ACQUISITION SCRIPT
# Downloads full datasets for Phase 1 ingestion
# =============================================================================
#
# Data Sources:
#   1. ONC High-Priority DDI     - https://www.healthit.gov
#   2. CMS Medicare Formulary    - https://data.cms.gov
#   3. LOINC Lab Reference       - https://loinc.org (requires registration)
#   4. OHDSI Athena DDI          - https://athena.ohdsi.org (requires registration)
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$SCRIPT_DIR/../data"
OHDSI_DIR="$DATA_DIR/ohdsi"

echo "============================================="
echo "PHASE 1 PRODUCTION DATA ACQUISITION"
echo "============================================="
echo ""
echo "Target: $DATA_DIR"
echo ""

mkdir -p "$DATA_DIR"
mkdir -p "$OHDSI_DIR"

# =============================================================================
# 1. ONC HIGH-PRIORITY DDI (~1,200 pairs)
# =============================================================================
echo "1. ONC High-Priority Drug-Drug Interactions"
echo "   Source: ONC/HHS"
echo "   URL: https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction"
echo ""

# ONC provides the list via their CDS support website
# We can fetch from the Clinical Decision Support GitHub
ONC_URL="https://raw.githubusercontent.com/AHRQ-CDS/AHRQ-CDS-Connect/main/drug-drug-interactions/onc-high-priority-ddi.csv"

if [ -f "$DATA_DIR/onc_ddi_production.csv" ]; then
    echo "   [SKIP] onc_ddi_production.csv already exists"
else
    echo "   Downloading ONC DDI data..."
    # Try fetching from CDS Connect (if available)
    if curl -fsSL "$ONC_URL" -o "$DATA_DIR/onc_ddi_production.csv" 2>/dev/null; then
        echo "   [OK] Downloaded ONC DDI data"
    else
        echo "   [MANUAL] ONC data requires manual download"
        echo "   Steps:"
        echo "     1. Go to: https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction"
        echo "     2. Download the DDI spreadsheet"
        echo "     3. Convert to CSV format with columns:"
        echo "        Drug1_RXCUI,Drug1_Name,Drug2_RXCUI,Drug2_Name,Severity,Clinical_Effect,Management,Evidence_Level,Documentation,Clinical_Source,ONC_Pair_ID,Last_Updated"
        echo "     4. Save as: $DATA_DIR/onc_ddi_production.csv"
    fi
fi
echo ""

# =============================================================================
# 2. CMS MEDICARE PART D FORMULARY (~100K entries)
# =============================================================================
echo "2. CMS Medicare Part D Formulary"
echo "   Source: CMS Data.gov"
echo "   URL: https://data.cms.gov/provider-summary-by-type-of-service/medicare-part-d-prescribers"
echo ""

# CMS Formulary Public Use Files
CMS_FORMULARY_URL="https://data.cms.gov/sites/default/files/2024-12/Basic_Drugs_Q4_2024.csv"

if [ -f "$DATA_DIR/cms_formulary_production.csv" ]; then
    echo "   [SKIP] cms_formulary_production.csv already exists"
else
    echo "   Downloading CMS Formulary data..."
    if curl -fsSL "$CMS_FORMULARY_URL" -o "$DATA_DIR/cms_raw.csv" 2>/dev/null; then
        echo "   [OK] Downloaded CMS Formulary data"
        echo "   [NOTE] May need transformation to match expected schema"
    else
        echo "   [MANUAL] CMS Formulary requires manual download"
        echo "   Steps:"
        echo "     1. Go to: https://data.cms.gov"
        echo "     2. Search for 'Part D Formulary'"
        echo "     3. Download the Formulary Reference file"
        echo "     4. Transform to CSV format with columns:"
        echo "        CONTRACT_ID,PLAN_ID,RXCUI,NDC,DRUG_NAME,TIER_LEVEL_CODE,QUANTITY_LIMIT,QUANTITY_LIMIT_AMOUNT,QUANTITY_LIMIT_DAYS,PRIOR_AUTH,STEP_THERAPY,COVERAGE_STATUS,EFFECTIVE_YEAR"
        echo "     5. Save as: $DATA_DIR/cms_formulary_production.csv"
    fi
fi
echo ""

# =============================================================================
# 3. LOINC LAB REFERENCE RANGES (~2,000 ranges)
# =============================================================================
echo "3. LOINC Lab Reference Ranges"
echo "   Source: LOINC (Regenstrief Institute)"
echo "   URL: https://loinc.org"
echo ""

if [ -f "$DATA_DIR/loinc_labs_production.csv" ]; then
    echo "   [SKIP] loinc_labs_production.csv already exists"
else
    echo "   [MANUAL] LOINC data requires free registration"
    echo "   Steps:"
    echo "     1. Register at: https://loinc.org/get-loinc/"
    echo "     2. Download the full LOINC table (LoincTableCore.csv)"
    echo "     3. Download LOINC document ontology"
    echo "     4. Extract lab reference ranges from:"
    echo "        - LOINC clinical panels"
    echo "        - NHANES reference ranges"
    echo "     5. Transform to CSV format with columns:"
    echo "        loinc_code,component,property,time_aspect,system,scale_type,method_type,class,short_name,long_name,unit,low_normal,high_normal,critical_low,critical_high,age_group,sex,clinical_category,interpretation_guidance,delta_check_percent,delta_check_hours,deprecated"
    echo "     6. Save as: $DATA_DIR/loinc_labs_production.csv"
fi
echo ""

# =============================================================================
# 4. OHDSI ATHENA DDI (~200K pairs)
# =============================================================================
echo "4. OHDSI Athena Drug-Drug Interactions"
echo "   Source: OHDSI (Observational Health Data Sciences)"
echo "   URL: https://athena.ohdsi.org"
echo ""

if [ -f "$OHDSI_DIR/CONCEPT.csv" ] && [ -f "$OHDSI_DIR/CONCEPT_RELATIONSHIP.csv" ]; then
    echo "   [SKIP] OHDSI Athena files already exist"
else
    echo "   [MANUAL] OHDSI Athena requires free registration"
    echo "   Steps:"
    echo "     1. Register at: https://athena.ohdsi.org"
    echo "     2. Request vocabulary download (select RxNorm, RxNorm Extension)"
    echo "     3. Download the vocabulary bundle (ZIP file)"
    echo "     4. Extract and place these files in $OHDSI_DIR:"
    echo "        - CONCEPT.csv"
    echo "        - CONCEPT_RELATIONSHIP.csv"
    echo "     5. The Go ETL will extract DDI relationships from these"
fi
echo ""

# =============================================================================
# SUMMARY
# =============================================================================
echo "============================================="
echo "DATA ACQUISITION SUMMARY"
echo "============================================="
echo ""
echo "Files in $DATA_DIR:"
ls -la "$DATA_DIR/" 2>/dev/null || echo "(directory empty)"
echo ""
echo "Files in $OHDSI_DIR:"
ls -la "$OHDSI_DIR/" 2>/dev/null || echo "(directory empty)"
echo ""

# Check what's ready
echo "============================================="
echo "READINESS CHECK"
echo "============================================="

check_file() {
    if [ -f "$1" ]; then
        lines=$(wc -l < "$1")
        echo "  [OK] $(basename "$1"): $lines lines"
        return 0
    else
        echo "  [MISSING] $(basename "$1")"
        return 1
    fi
}

echo ""
echo "Required files:"
check_file "$DATA_DIR/onc_ddi.csv" || check_file "$DATA_DIR/onc_ddi_production.csv" || true
check_file "$DATA_DIR/cms_formulary.csv" || check_file "$DATA_DIR/cms_formulary_production.csv" || true
check_file "$DATA_DIR/loinc_labs.csv" || check_file "$DATA_DIR/loinc_labs_production.csv" || true

echo ""
echo "Optional (for expanded DDI coverage):"
check_file "$OHDSI_DIR/CONCEPT.csv" || true
check_file "$OHDSI_DIR/CONCEPT_RELATIONSHIP.csv" || true

echo ""
echo "============================================="
echo "NEXT STEPS"
echo "============================================="
echo ""
echo "After acquiring all data files:"
echo "  1. Rename production files to standard names:"
echo "     mv onc_ddi_production.csv onc_ddi.csv"
echo "     mv cms_formulary_production.csv cms_formulary.csv"
echo "     mv loinc_labs_production.csv loinc_labs.csv"
echo ""
echo "  2. Run the Go ingestion CLI:"
echo "     cd $(dirname "$SCRIPT_DIR")"
echo "     make ingest-live"
echo ""
echo "  3. Or run directly:"
echo "     go run ./cmd/phase1-ingest/main.go --all"
echo ""
