#!/bin/bash

###############################################################################
# KB-7 Knowledge Factory - Infrastructure Teardown Script
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

echo -e "${RED}========================================${NC}"
echo -e "${RED}KB-7 Infrastructure Teardown${NC}"
echo -e "${RED}========================================${NC}\n"

echo -e "${YELLOW}WARNING: This will delete ALL KB-7 infrastructure!${NC}"
echo -e "${YELLOW}This includes:${NC}"
echo -e "  - Cloud Functions (4)"
echo -e "  - Cloud Workflows (1)"
echo -e "  - Cloud Scheduler (1)"
echo -e "  - Cloud Storage buckets (3)"
echo -e "  - Secret Manager secrets (4)"
echo -e "  - Service Accounts (4)"
echo -e "  - Monitoring alerts"
echo -e ""

cd "$TERRAFORM_DIR"

if [ ! -f "terraform.tfstate" ]; then
    echo -e "${RED}Error: Terraform state not found. Nothing to teardown.${NC}"
    exit 1
fi

# Extract configuration
PROJECT_ID=$(terraform output -raw project_id 2>/dev/null || echo "unknown")
ENVIRONMENT=$(terraform output -raw environment 2>/dev/null || echo "unknown")

echo -e "${BLUE}Project:${NC} ${PROJECT_ID}"
echo -e "${BLUE}Environment:${NC} ${ENVIRONMENT}"
echo -e ""

echo -e "${RED}Are you ABSOLUTELY sure you want to destroy this infrastructure?${NC}"
echo -e "${YELLOW}Type 'destroy' to confirm:${NC} "
read -r CONFIRM

if [ "$CONFIRM" != "destroy" ]; then
    echo -e "${GREEN}Teardown cancelled${NC}"
    exit 0
fi

echo ""

###############################################################################
# Step 1: Disable Cloud Scheduler
###############################################################################

echo -e "${YELLOW}Step 1: Disabling Cloud Scheduler...${NC}"

SCHEDULER_NAME=$(terraform output -raw scheduler_job_name 2>/dev/null || echo "")

if [ -n "$SCHEDULER_NAME" ]; then
    gcloud scheduler jobs pause "$SCHEDULER_NAME" --location="$(terraform output -raw region)" 2>/dev/null || true
    echo -e "${GREEN}✓ Scheduler disabled${NC}\n"
else
    echo -e "${YELLOW}  Scheduler not found, skipping${NC}\n"
fi

###############################################################################
# Step 2: Empty Storage Buckets (if needed)
###############################################################################

echo -e "${YELLOW}Step 2: Checking storage buckets...${NC}"

SOURCES_BUCKET=$(terraform output -raw sources_bucket_name 2>/dev/null || echo "")
ARTIFACTS_BUCKET=$(terraform output -raw artifacts_bucket_name 2>/dev/null || echo "")

echo -e "${BLUE}Do you want to delete bucket contents? (yes/no)${NC}"
echo -e "${YELLOW}If 'no', Terraform will fail to delete non-empty buckets${NC}"
read -r DELETE_CONTENTS

if [ "$DELETE_CONTENTS" == "yes" ]; then
    if [ -n "$SOURCES_BUCKET" ]; then
        echo -e "  Emptying sources bucket..."
        gsutil -m rm -r "gs://${SOURCES_BUCKET}/**" 2>/dev/null || true
        echo -e "${GREEN}  ✓ Sources bucket emptied${NC}"
    fi

    if [ -n "$ARTIFACTS_BUCKET" ]; then
        echo -e "  Emptying artifacts bucket..."
        gsutil -m rm -r "gs://${ARTIFACTS_BUCKET}/**" 2>/dev/null || true
        echo -e "${GREEN}  ✓ Artifacts bucket emptied${NC}"
    fi
else
    echo -e "${YELLOW}  Buckets will not be emptied. Terraform may fail if buckets are not empty.${NC}"
fi

echo ""

###############################################################################
# Step 3: Run Terraform Destroy
###############################################################################

echo -e "${YELLOW}Step 3: Running Terraform destroy...${NC}"
echo -e "${YELLOW}This may take 5-10 minutes...${NC}\n"

terraform destroy -auto-approve

echo -e "\n${GREEN}✓ Infrastructure destroyed${NC}\n"

###############################################################################
# Step 4: Clean Up Local Files
###############################################################################

echo -e "${YELLOW}Step 4: Cleaning up local files...${NC}"

# Remove Terraform state files
rm -f terraform.tfstate*
rm -f tfplan
rm -rf .terraform/

# Remove function zip files
FUNCTIONS_DIR="$(dirname "$SCRIPT_DIR")/functions"
find "$FUNCTIONS_DIR" -name "function.zip" -delete

echo -e "${GREEN}✓ Local files cleaned${NC}\n"

###############################################################################
# Summary
###############################################################################

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Teardown Complete${NC}"
echo -e "${BLUE}========================================${NC}\n"

echo -e "${GREEN}All KB-7 infrastructure has been destroyed.${NC}\n"

echo -e "${BLUE}Verification Steps:${NC}"
echo -e "  1. Check for remaining functions:"
echo -e "     ${YELLOW}gcloud functions list --filter='name~kb7'${NC}"
echo -e ""
echo -e "  2. Check for remaining buckets:"
echo -e "     ${YELLOW}gcloud storage buckets list --filter='name~kb7'${NC}"
echo -e ""
echo -e "  3. Check for remaining secrets:"
echo -e "     ${YELLOW}gcloud secrets list --filter='name~kb7'${NC}"
echo -e ""

echo -e "${YELLOW}Note: Some resources may take a few minutes to fully delete.${NC}"
