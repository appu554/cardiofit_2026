#!/bin/bash
#
# KB-7 Knowledge Factory - Lambda Function Testing Script
# Tests each Lambda function individually with mock events
#
# Usage:
#   ./test-lambda-functions.sh [environment]
#   environment: production (default), staging, development
#

set -e  # Exit on error

# Configuration
ENVIRONMENT="${1:-production}"
AWS_REGION="${AWS_REGION:-us-east-1}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Function names
SNOMED_FUNCTION="kb7-snomed-downloader-${ENVIRONMENT}"
RXNORM_FUNCTION="kb7-rxnorm-downloader-${ENVIRONMENT}"
LOINC_FUNCTION="kb7-loinc-downloader-${ENVIRONMENT}"
GITHUB_FUNCTION="kb7-github-dispatcher-${ENVIRONMENT}"

log_info "Starting KB-7 Lambda function tests"
log_info "Environment: $ENVIRONMENT"
log_info "Region: $AWS_REGION"
echo ""

# Test 1: SNOMED Downloader
log_test "Test 1/4: Testing SNOMED downloader..."
echo ""

SNOMED_EVENT='{
  "release_date": "20250131",
  "edition": "international"
}'

log_info "Invoking SNOMED downloader with test event..."
SNOMED_RESULT=$(aws lambda invoke \
    --function-name "$SNOMED_FUNCTION" \
    --payload "$SNOMED_EVENT" \
    --region "$AWS_REGION" \
    /tmp/snomed-response.json 2>&1)

if [ $? -eq 0 ]; then
    SNOMED_RESPONSE=$(cat /tmp/snomed-response.json)
    STATUS_CODE=$(echo "$SNOMED_RESPONSE" | jq -r '.statusCode')

    if [ "$STATUS_CODE" == "200" ]; then
        log_success "SNOMED downloader: PASSED"
        echo "Response: $(echo "$SNOMED_RESPONSE" | jq -c '.')"
    else
        log_error "SNOMED downloader: FAILED (HTTP $STATUS_CODE)"
        echo "Response: $SNOMED_RESPONSE"
    fi
else
    log_error "SNOMED downloader: FAILED (Lambda invocation error)"
fi
echo ""

# Test 2: RxNorm Downloader
log_test "Test 2/4: Testing RxNorm downloader..."
echo ""

RXNORM_EVENT='{
  "version": "01042025",
  "subset": "full"
}'

log_info "Invoking RxNorm downloader with test event..."
RXNORM_RESULT=$(aws lambda invoke \
    --function-name "$RXNORM_FUNCTION" \
    --payload "$RXNORM_EVENT" \
    --region "$AWS_REGION" \
    /tmp/rxnorm-response.json 2>&1)

if [ $? -eq 0 ]; then
    RXNORM_RESPONSE=$(cat /tmp/rxnorm-response.json)
    STATUS_CODE=$(echo "$RXNORM_RESPONSE" | jq -r '.statusCode')

    if [ "$STATUS_CODE" == "200" ]; then
        log_success "RxNorm downloader: PASSED"
        echo "Response: $(echo "$RXNORM_RESPONSE" | jq -c '.')"
    else
        log_error "RxNorm downloader: FAILED (HTTP $STATUS_CODE)"
        echo "Response: $RXNORM_RESPONSE"
    fi
else
    log_error "RxNorm downloader: FAILED (Lambda invocation error)"
fi
echo ""

# Test 3: LOINC Downloader
log_test "Test 3/4: Testing LOINC downloader..."
echo ""

LOINC_EVENT='{
  "version": "2.77",
  "format": "csv"
}'

log_info "Invoking LOINC downloader with test event..."
LOINC_RESULT=$(aws lambda invoke \
    --function-name "$LOINC_FUNCTION" \
    --payload "$LOINC_EVENT" \
    --region "$AWS_REGION" \
    /tmp/loinc-response.json 2>&1)

if [ $? -eq 0 ]; then
    LOINC_RESPONSE=$(cat /tmp/loinc-response.json)
    STATUS_CODE=$(echo "$LOINC_RESPONSE" | jq -r '.statusCode')

    if [ "$STATUS_CODE" == "200" ]; then
        log_success "LOINC downloader: PASSED"
        echo "Response: $(echo "$LOINC_RESPONSE" | jq -c '.')"
    else
        log_error "LOINC downloader: FAILED (HTTP $STATUS_CODE)"
        echo "Response: $LOINC_RESPONSE"
    fi
else
    log_error "LOINC downloader: FAILED (Lambda invocation error)"
fi
echo ""

# Test 4: GitHub Dispatcher
log_test "Test 4/4: Testing GitHub dispatcher..."
echo ""

GITHUB_EVENT='{
  "downloads": [
    {
      "snomed": {
        "statusCode": 200,
        "s3_key": "snomed-ct/20250131/SnomedCT_international_20250131.zip",
        "checksum": "abc123def456"
      }
    },
    {
      "rxnorm": {
        "statusCode": 200,
        "s3_key": "rxnorm/01042025/RxNorm_full_01042025.zip",
        "checksum": "def456ghi789"
      }
    },
    {
      "loinc": {
        "statusCode": 200,
        "s3_key": "loinc/2.77/LOINC_2.77.csv.zip",
        "checksum": "ghi789jkl012"
      }
    }
  ]
}'

log_info "Invoking GitHub dispatcher with test event..."
GITHUB_RESULT=$(aws lambda invoke \
    --function-name "$GITHUB_FUNCTION" \
    --payload "$GITHUB_EVENT" \
    --region "$AWS_REGION" \
    /tmp/github-response.json 2>&1)

if [ $? -eq 0 ]; then
    GITHUB_RESPONSE=$(cat /tmp/github-response.json)
    STATUS_CODE=$(echo "$GITHUB_RESPONSE" | jq -r '.statusCode')

    if [ "$STATUS_CODE" == "200" ]; then
        log_success "GitHub dispatcher: PASSED"
        echo "Response: $(echo "$GITHUB_RESPONSE" | jq -c '.')"
    else
        log_error "GitHub dispatcher: FAILED (HTTP $STATUS_CODE)"
        echo "Response: $GITHUB_RESPONSE"
    fi
else
    log_error "GitHub dispatcher: FAILED (Lambda invocation error)"
fi
echo ""

# Summary
log_info "========================================="
log_info "Lambda function test summary"
log_info "========================================="
echo ""
log_info "Tested functions:"
log_info "  1. $SNOMED_FUNCTION"
log_info "  2. $RXNORM_FUNCTION"
log_info "  3. $LOINC_FUNCTION"
log_info "  4. $GITHUB_FUNCTION"
echo ""
log_warn "Next steps:"
log_warn "  1. Review CloudWatch logs for each function:"
log_warn "     aws logs tail /aws/lambda/$SNOMED_FUNCTION --follow"
log_warn "  2. Verify S3 uploads (if functions succeeded)"
log_warn "  3. Check Secrets Manager access (ensure credentials are set)"
log_warn "  4. Test Step Functions workflow end-to-end"
echo ""
log_info "Clean up test files:"
log_info "  rm /tmp/snomed-response.json /tmp/rxnorm-response.json /tmp/loinc-response.json /tmp/github-response.json"
