#!/bin/bash
# =============================================================================
# KB-5 DDI Database Verification Script
# Run this after starting the services to verify data was loaded
# =============================================================================

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5433}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-kb5_drug_interactions}"

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}       KB-5 DDI DATABASE VERIFICATION                          ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo "⚠️  psql not found. Using docker exec instead..."
    PSQL_CMD="docker exec kb5-postgres psql -U $DB_USER -d $DB_NAME"
else
    PSQL_CMD="psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"
fi

echo -e "${YELLOW}📊 TOTAL DDI RECORDS:${NC}"
echo "────────────────────────────────────────"
$PSQL_CMD -c "
SELECT
    COUNT(*) as total_interactions,
    COUNT(*) FILTER (WHERE active = TRUE) as active_interactions,
    COUNT(*) FILTER (WHERE severity = 'contraindicated') as contraindicated,
    COUNT(*) FILTER (WHERE severity = 'major') as major,
    COUNT(*) FILTER (WHERE severity = 'moderate') as moderate
FROM drug_interactions;
"

echo ""
echo -e "${YELLOW}✅ GOVERNANCE COMPLIANCE STATUS:${NC}"
echo "────────────────────────────────────────"
$PSQL_CMD -c "
SELECT
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE gov_regulatory_authority IS NOT NULL) as has_regulatory,
    COUNT(*) FILTER (WHERE gov_pharmacology_authority IS NOT NULL) as has_pharmacology,
    COUNT(*) FILTER (WHERE gov_clinical_authority IS NOT NULL) as has_clinical,
    COUNT(*) FILTER (
        WHERE gov_regulatory_authority IS NOT NULL
        AND gov_pharmacology_authority IS NOT NULL
        AND gov_clinical_authority IS NOT NULL
    ) as fully_governed
FROM drug_interactions
WHERE active = TRUE;
"

echo ""
echo -e "${YELLOW}📋 DDI RECORDS BY CATEGORY:${NC}"
echo "────────────────────────────────────────"
$PSQL_CMD -c "
SELECT
    CASE
        WHEN interaction_id LIKE 'WARFARIN%' OR interaction_id LIKE 'RIVAROXABAN%' THEN 'Anticoagulant'
        WHEN gov_qt_risk_category IS NOT NULL THEN 'QT Prolongation'
        WHEN interaction_id LIKE '%TRAMADOL%' OR interaction_id LIKE '%PHENELZINE%' THEN 'Serotonin Syndrome'
        WHEN interaction_id LIKE 'TRIPLE%' OR interaction_id LIKE 'METHOTREXATE%' THEN 'Nephrotoxicity'
        WHEN interaction_id LIKE 'OXYCODONE%' OR interaction_id LIKE 'METHADONE%' THEN 'Opioid Safety'
        WHEN interaction_id LIKE 'METFORMIN%' OR interaction_id LIKE 'GLYBURIDE%' THEN 'Diabetes'
        WHEN interaction_id LIKE 'ATORVASTATIN%' OR interaction_id LIKE 'STATINS%' THEN 'Statin DDI'
        ELSE 'Other'
    END AS category,
    COUNT(*) as count,
    STRING_AGG(DISTINCT severity, ', ' ORDER BY severity) as severities
FROM drug_interactions
WHERE active = TRUE
GROUP BY 1
ORDER BY count DESC;
"

echo ""
echo -e "${YELLOW}🔗 SOURCE AUTHORITY BREAKDOWN:${NC}"
echo "────────────────────────────────────────"
$PSQL_CMD -c "
SELECT
    'Layer 1 (Regulatory)' as layer,
    gov_regulatory_authority as authority,
    COUNT(*) as count
FROM drug_interactions
WHERE active = TRUE AND gov_regulatory_authority IS NOT NULL
GROUP BY gov_regulatory_authority

UNION ALL

SELECT
    'Layer 2 (Pharmacology)' as layer,
    gov_pharmacology_authority as authority,
    COUNT(*) as count
FROM drug_interactions
WHERE active = TRUE AND gov_pharmacology_authority IS NOT NULL
GROUP BY gov_pharmacology_authority

UNION ALL

SELECT
    'Layer 3 (Clinical)' as layer,
    gov_clinical_authority as authority,
    COUNT(*) as count
FROM drug_interactions
WHERE active = TRUE AND gov_clinical_authority IS NOT NULL
GROUP BY gov_clinical_authority

ORDER BY layer, count DESC;
"

echo ""
echo -e "${YELLOW}📝 SAMPLE DDI WITH FULL GOVERNANCE:${NC}"
echo "────────────────────────────────────────"
$PSQL_CMD -c "
SELECT
    interaction_id,
    drug_a_name || ' + ' || drug_b_name as drug_pair,
    severity,
    gov_regulatory_authority as regulatory,
    gov_pharmacology_authority as pharmacology,
    gov_clinical_authority as clinical,
    gov_evidence_grade as grade
FROM drug_interactions
WHERE active = TRUE
  AND gov_regulatory_authority IS NOT NULL
  AND gov_pharmacology_authority IS NOT NULL
ORDER BY severity DESC
LIMIT 5;
"

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}       VERIFICATION COMPLETE                                   ${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
