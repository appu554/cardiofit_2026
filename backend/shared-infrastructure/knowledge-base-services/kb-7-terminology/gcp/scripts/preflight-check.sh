#!/bin/bash
# KB-7 GCP Deployment Pre-Flight Check
# Verifies all prerequisites before deployment

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

CHECKS_PASSED=0
CHECKS_FAILED=0
WARNINGS=0

echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  KB-7 GCP Deployment - Pre-Flight Check             ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to print check result
check_pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((CHECKS_PASSED++))
}

check_fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    echo -e "   ${YELLOW}→ $2${NC}"
    ((CHECKS_FAILED++))
}

check_warn() {
    echo -e "${YELLOW}⚠️  WARN${NC}: $1"
    echo -e "   ${YELLOW}→ $2${NC}"
    ((WARNINGS++))
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1️⃣  Checking GCP CLI Tools"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check gcloud
if command -v gcloud &> /dev/null; then
    GCLOUD_VERSION=$(gcloud version --format="value(version)" 2>/dev/null | head -1)
    check_pass "gcloud CLI installed (version $GCLOUD_VERSION)"
else
    check_fail "gcloud CLI not installed" "Install from: https://cloud.google.com/sdk/docs/install"
fi

# Check gsutil
if command -v gsutil &> /dev/null; then
    check_pass "gsutil installed"
else
    check_warn "gsutil not found" "Usually bundled with gcloud SDK"
fi

# Check terraform
if command -v terraform &> /dev/null; then
    TF_VERSION=$(terraform version -json 2>/dev/null | grep terraform_version | cut -d'"' -f4)
    check_pass "Terraform installed (version $TF_VERSION)"
else
    check_fail "Terraform not installed" "Install from: https://www.terraform.io/downloads"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2️⃣  Checking GCP Authentication"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check gcloud auth
if gcloud auth list --filter=status:ACTIVE --format="value(account)" &> /dev/null; then
    ACTIVE_ACCOUNT=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | head -1)
    if [ -n "$ACTIVE_ACCOUNT" ]; then
        check_pass "Authenticated as: $ACTIVE_ACCOUNT"
    else
        check_fail "No active gcloud authentication" "Run: gcloud auth login"
    fi
else
    check_fail "Cannot check gcloud auth" "Run: gcloud auth login"
fi

# Check application default credentials
if [ -f "$HOME/.config/gcloud/application_default_credentials.json" ]; then
    check_pass "Application default credentials configured"
else
    check_warn "Application default credentials not found" "Run: gcloud auth application-default login"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3️⃣  Checking GCP Project Configuration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check active project
ACTIVE_PROJECT=$(gcloud config get-value project 2>/dev/null)
if [ -n "$ACTIVE_PROJECT" ] && [ "$ACTIVE_PROJECT" != "(unset)" ]; then
    check_pass "Active project: $ACTIVE_PROJECT"

    # Check billing
    BILLING_ENABLED=$(gcloud billing projects describe $ACTIVE_PROJECT --format="value(billingEnabled)" 2>/dev/null)
    if [ "$BILLING_ENABLED" = "True" ]; then
        check_pass "Billing enabled for project"
    else
        check_fail "Billing not enabled" "Link billing: gcloud billing projects link $ACTIVE_PROJECT --billing-account=BILLING_ACCOUNT_ID"
    fi
else
    check_fail "No active GCP project" "Set project: gcloud config set project PROJECT_ID"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4️⃣  Checking Terraform Configuration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check terraform directory
if [ -d "../terraform" ]; then
    check_pass "Terraform directory exists"

    # Check terraform.tfvars
    if [ -f "../terraform/terraform.tfvars" ]; then
        check_pass "terraform.tfvars configuration file exists"

        # Check for placeholder values
        if grep -q "your-nhs-trud-api-key-here" ../terraform/terraform.tfvars 2>/dev/null; then
            check_warn "terraform.tfvars contains placeholder values" "Update with real API keys"
        else
            check_pass "terraform.tfvars appears configured"
        fi
    else
        check_fail "terraform.tfvars not found" "Copy from: terraform/terraform.tfvars.example"
    fi

    # Check terraform init
    if [ -d "../terraform/.terraform" ]; then
        check_pass "Terraform initialized"
    else
        check_warn "Terraform not initialized" "Run: cd ../terraform && terraform init"
    fi
else
    check_fail "Terraform directory not found" "Ensure you're in gcp/scripts/ directory"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5️⃣  Checking Required API Credentials"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ -f "../terraform/terraform.tfvars" ]; then
    # Check NHS TRUD API key
    if grep -q "nhs_trud_api_key.*=" ../terraform/terraform.tfvars | grep -v "your-nhs" &>/dev/null; then
        check_pass "NHS TRUD API key configured"
    else
        check_warn "NHS TRUD API key not configured" "Get from: https://isd.digital.nhs.uk/trud/"
    fi

    # Check UMLS API key
    if grep -q "umls_api_key.*=" ../terraform/terraform.tfvars | grep -v "your-umls" &>/dev/null; then
        check_pass "UMLS API key configured"
    else
        check_warn "UMLS API key not configured" "Get from: https://uts.nlm.nih.gov/"
    fi

    # Check LOINC credentials
    if grep -q "loinc_username.*=" ../terraform/terraform.tfvars | grep -v "your-loinc" &>/dev/null; then
        check_pass "LOINC credentials configured"
    else
        check_warn "LOINC credentials not configured" "Get from: https://loinc.org/"
    fi

    # Check GitHub PAT
    if grep -q "github_pat.*=" ../terraform/terraform.tfvars | grep -v "ghp_your" &>/dev/null; then
        check_pass "GitHub Personal Access Token configured"
    else
        check_warn "GitHub PAT not configured" "Get from: https://github.com/settings/tokens"
    fi
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "6️⃣  Checking Local Environment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check Python
if command -v python3 &> /dev/null; then
    PYTHON_VERSION=$(python3 --version | cut -d' ' -f2)
    check_pass "Python 3 installed (version $PYTHON_VERSION)"
else
    check_warn "Python 3 not found" "Required for local function testing"
fi

# Check jq
if command -v jq &> /dev/null; then
    check_pass "jq installed (for JSON parsing)"
else
    check_warn "jq not found" "Install for better log parsing: brew install jq"
fi

# Check curl
if command -v curl &> /dev/null; then
    check_pass "curl installed"
else
    check_fail "curl not found" "Required for API calls"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Pre-Flight Check Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo -e "Checks passed:  ${GREEN}$CHECKS_PASSED${NC}"
echo -e "Checks failed:  ${RED}$CHECKS_FAILED${NC}"
echo -e "Warnings:       ${YELLOW}$WARNINGS${NC}"
echo ""

if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All critical checks passed!${NC}"
    echo ""
    echo "You're ready to deploy. Run these commands:"
    echo ""
    echo -e "  ${BLUE}cd ../terraform${NC}"
    echo -e "  ${BLUE}terraform init${NC}"
    echo -e "  ${BLUE}terraform plan${NC}"
    echo -e "  ${BLUE}terraform apply${NC}"
    echo ""
    echo "Or use the automated deployment script:"
    echo -e "  ${BLUE}./deploy-infrastructure.sh${NC}"
    echo ""
    exit 0
else
    echo -e "${RED}❌ Some critical checks failed!${NC}"
    echo ""
    echo "Fix the issues above before deploying."
    echo ""
    echo "Quick fixes:"
    echo ""
    if ! command -v gcloud &> /dev/null; then
        echo "  1. Install gcloud: https://cloud.google.com/sdk/docs/install"
    fi
    if ! command -v terraform &> /dev/null; then
        echo "  2. Install terraform: https://www.terraform.io/downloads"
    fi
    if [ -z "$ACTIVE_ACCOUNT" ]; then
        echo "  3. Login to GCP: gcloud auth login"
        echo "  4. Set application credentials: gcloud auth application-default login"
    fi
    if [ -z "$ACTIVE_PROJECT" ] || [ "$ACTIVE_PROJECT" = "(unset)" ]; then
        echo "  5. Set project: gcloud config set project PROJECT_ID"
    fi
    if [ "$BILLING_ENABLED" != "True" ]; then
        echo "  6. Enable billing: gcloud billing projects link PROJECT_ID --billing-account=BILLING_ACCOUNT_ID"
    fi
    if [ ! -f "../terraform/terraform.tfvars" ]; then
        echo "  7. Create terraform.tfvars: cp ../terraform/terraform.tfvars.example ../terraform/terraform.tfvars"
    fi
    echo ""
    exit 1
fi
