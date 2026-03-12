#!/bin/bash

###############################################################################
# KB-7 Knowledge Factory - Function Testing Script
###############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$(dirname "$SCRIPT_DIR")/terraform"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}KB-7 Function Testing${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Extract configuration from Terraform
cd "$TERRAFORM_DIR"

if [ ! -f "terraform.tfstate" ]; then
    echo -e "${RED}Error: Terraform state not found. Deploy infrastructure first.${NC}"
    exit 1
fi

PROJECT_ID=$(terraform output -raw project_id)
REGION=$(terraform output -raw region)
ENVIRONMENT=$(terraform output -raw environment)

echo -e "${YELLOW}Testing environment: ${ENVIRONMENT}${NC}"
echo -e "${YELLOW}Project: ${PROJECT_ID}${NC}"
echo -e "${YELLOW}Region: ${REGION}${NC}\n"

###############################################################################
# Test Function
###############################################################################

test_function() {
    local func_name=$1
    local timeout=$2

    echo -e "${YELLOW}Testing ${func_name}...${NC}"

    # Get function URL
    local func_url=$(terraform output -json function_urls | jq -r ".\"${func_name}\"")

    if [ -z "$func_url" ] || [ "$func_url" == "null" ]; then
        echo -e "${RED}  ✗ Function URL not found${NC}\n"
        return 1
    fi

    # Call function with test payload
    local test_payload='{"test": true, "trigger": "manual-test"}'

    echo -e "  Calling: ${func_url}"
    echo -e "  Payload: ${test_payload}"
    echo -e "  Timeout: ${timeout}s"

    local response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        --max-time "$timeout" \
        "$func_url")

    local http_code=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        echo -e "${GREEN}  ✓ Function executed successfully (HTTP ${http_code})${NC}"
        echo -e "  Response: ${body}" | jq '.' 2>/dev/null || echo "  Response: ${body}"
        echo ""
        return 0
    else
        echo -e "${RED}  ✗ Function failed (HTTP ${http_code})${NC}"
        echo -e "  Response: ${body}"
        echo ""
        return 1
    fi
}

###############################################################################
# Test All Functions
###############################################################################

FAILED_TESTS=0

echo -e "${BLUE}Starting function tests...${NC}\n"

# Test SNOMED downloader (long timeout)
if ! test_function "snomed_downloader" 60; then
    ((FAILED_TESTS++))
fi

# Test RxNorm downloader
if ! test_function "rxnorm_downloader" 60; then
    ((FAILED_TESTS++))
fi

# Test LOINC downloader
if ! test_function "loinc_downloader" 30; then
    ((FAILED_TESTS++))
fi

# Test GitHub dispatcher (requires download results)
echo -e "${YELLOW}Testing github_dispatcher...${NC}"
echo -e "  ${YELLOW}Note: Skipping dispatcher test (requires download results)${NC}"
echo -e "  Use full workflow test instead\n"

###############################################################################
# Summary
###############################################################################

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}\n"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}\n"

    echo -e "${BLUE}Next steps:${NC}"
    echo -e "  1. Test the complete workflow:"
    echo -e "     ${YELLOW}gcloud workflows execute kb7-factory-workflow-${ENVIRONMENT} --location=${REGION} --data='{\"trigger\":\"manual-test\"}'${NC}"
    echo -e ""
    echo -e "  2. View workflow execution:"
    echo -e "     ${YELLOW}gcloud workflows executions list kb7-factory-workflow-${ENVIRONMENT} --location=${REGION}${NC}"
    echo -e ""
    echo -e "  3. View function logs:"
    echo -e "     ${YELLOW}gcloud logging read 'resource.type=\"cloud_function\" AND resource.labels.function_name=~\"kb7-.*\"' --limit=50${NC}"
    echo -e ""

    exit 0
else
    echo -e "${RED}✗ ${FAILED_TESTS} test(s) failed${NC}\n"

    echo -e "${BLUE}Troubleshooting:${NC}"
    echo -e "  1. Check function logs:"
    echo -e "     ${YELLOW}gcloud logging read 'resource.type=\"cloud_function\" AND severity>=ERROR' --limit=20${NC}"
    echo -e ""
    echo -e "  2. Verify secrets are configured:"
    echo -e "     ${YELLOW}gcloud secrets list --filter='name~kb7'${NC}"
    echo -e ""
    echo -e "  3. Check function status:"
    echo -e "     ${YELLOW}gcloud functions list --filter='name~kb7'${NC}"
    echo -e ""

    exit 1
fi
