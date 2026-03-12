#!/bin/bash

###############################################################################
# KB-7 Knowledge Factory - GCP Infrastructure Deployment Script
###############################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GCP_DIR="$(dirname "$SCRIPT_DIR")"
TERRAFORM_DIR="$GCP_DIR/terraform"
FUNCTIONS_DIR="$GCP_DIR/functions"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}KB-7 Knowledge Factory - GCP Deployment${NC}"
echo -e "${BLUE}========================================${NC}\n"

###############################################################################
# Step 1: Validate Prerequisites
###############################################################################

echo -e "${YELLOW}Step 1: Validating prerequisites...${NC}"

# Check for required tools
command -v gcloud >/dev/null 2>&1 || { echo -e "${RED}Error: gcloud CLI not found${NC}"; exit 1; }
command -v terraform >/dev/null 2>&1 || { echo -e "${RED}Error: terraform not found${NC}"; exit 1; }
command -v zip >/dev/null 2>&1 || { echo -e "${RED}Error: zip not found${NC}"; exit 1; }

echo -e "${GREEN}✓ All prerequisites satisfied${NC}\n"

###############################################################################
# Step 2: Verify GCP Authentication
###############################################################################

echo -e "${YELLOW}Step 2: Verifying GCP authentication...${NC}"

if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "@"; then
    echo -e "${RED}Error: Not authenticated with GCP. Run 'gcloud auth login'${NC}"
    exit 1
fi

CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)
echo -e "${GREEN}✓ Authenticated with GCP${NC}"
echo -e "  Current project: ${CURRENT_PROJECT}\n"

###############################################################################
# Step 3: Check Terraform Configuration
###############################################################################

echo -e "${YELLOW}Step 3: Checking Terraform configuration...${NC}"

if [ ! -f "$TERRAFORM_DIR/terraform.tfvars" ]; then
    echo -e "${RED}Error: terraform.tfvars not found${NC}"
    echo -e "  Copy terraform.tfvars.example and configure with your values"
    exit 1
fi

echo -e "${GREEN}✓ Terraform configuration found${NC}\n"

###############################################################################
# Step 4: Package Cloud Functions
###############################################################################

echo -e "${YELLOW}Step 4: Packaging Cloud Functions...${NC}"

package_function() {
    local func_name=$1
    local func_dir="$FUNCTIONS_DIR/$func_name"
    local output_zip="$func_dir/function.zip"

    echo -e "  Packaging ${func_name}..."

    if [ ! -d "$func_dir" ]; then
        echo -e "${RED}Error: Function directory not found: $func_dir${NC}"
        exit 1
    fi

    # Remove old zip if exists
    rm -f "$output_zip"

    # Create zip with main.py and requirements.txt
    cd "$func_dir"
    zip -q -r function.zip main.py requirements.txt
    cd - > /dev/null

    echo -e "${GREEN}  ✓ ${func_name} packaged${NC}"
}

package_function "snomed-downloader"
package_function "rxnorm-downloader"
package_function "loinc-downloader"
package_function "github-dispatcher"

echo -e "\n${GREEN}✓ All functions packaged${NC}\n"

###############################################################################
# Step 5: Initialize Terraform
###############################################################################

echo -e "${YELLOW}Step 5: Initializing Terraform...${NC}"

cd "$TERRAFORM_DIR"

terraform init

echo -e "${GREEN}✓ Terraform initialized${NC}\n"

###############################################################################
# Step 6: Plan Infrastructure Changes
###############################################################################

echo -e "${YELLOW}Step 6: Planning infrastructure changes...${NC}"

terraform plan -out=tfplan

echo -e "\n${BLUE}Review the plan above. Continue with deployment? (yes/no)${NC}"
read -r CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo -e "${YELLOW}Deployment cancelled${NC}"
    exit 0
fi

echo ""

###############################################################################
# Step 7: Apply Infrastructure
###############################################################################

echo -e "${YELLOW}Step 7: Deploying infrastructure...${NC}"
echo -e "${YELLOW}This may take 5-10 minutes...${NC}\n"

terraform apply tfplan

echo -e "\n${GREEN}✓ Infrastructure deployed successfully${NC}\n"

###############################################################################
# Step 8: Output Deployment Summary
###############################################################################

echo -e "${YELLOW}Step 8: Deployment summary${NC}\n"

# Extract outputs from Terraform
PROJECT_ID=$(terraform output -raw project_id)
REGION=$(terraform output -raw region)
SOURCES_BUCKET=$(terraform output -raw sources_bucket_name)
ARTIFACTS_BUCKET=$(terraform output -raw artifacts_bucket_name)
WORKFLOW_NAME=$(terraform output -raw workflow_name)
SCHEDULER_NAME=$(terraform output -raw scheduler_job_name)

echo -e "${GREEN}Deployment Complete!${NC}\n"

echo -e "${BLUE}Project Information:${NC}"
echo -e "  Project ID: ${PROJECT_ID}"
echo -e "  Region: ${REGION}"
echo -e ""

echo -e "${BLUE}Cloud Storage Buckets:${NC}"
echo -e "  Sources: gs://${SOURCES_BUCKET}"
echo -e "  Artifacts: gs://${ARTIFACTS_BUCKET}"
echo -e ""

echo -e "${BLUE}Cloud Functions:${NC}"
terraform output -json function_urls | jq -r 'to_entries[] | "  \(.key): \(.value)"'
echo -e ""

echo -e "${BLUE}Cloud Workflows:${NC}"
echo -e "  Workflow: ${WORKFLOW_NAME}"
echo -e ""

echo -e "${BLUE}Cloud Scheduler:${NC}"
echo -e "  Job: ${SCHEDULER_NAME}"
echo -e "  Schedule: $(terraform output -raw scheduler_schedule)"
echo -e ""

###############################################################################
# Step 9: Next Steps
###############################################################################

echo -e "${BLUE}Next Steps:${NC}"
echo -e "  1. Test individual functions:"
echo -e "     ${YELLOW}./test-functions.sh${NC}"
echo -e ""
echo -e "  2. Test the complete workflow:"
echo -e "     ${YELLOW}gcloud workflows execute ${WORKFLOW_NAME} --location=${REGION} --data='{\"trigger\":\"manual-test\"}'${NC}"
echo -e ""
echo -e "  3. View logs:"
echo -e "     ${YELLOW}gcloud logging read 'resource.type=\"cloud_function\"' --limit=50${NC}"
echo -e ""
echo -e "  4. Monitor workflow:"
echo -e "     ${YELLOW}https://console.cloud.google.com/workflows?project=${PROJECT_ID}${NC}"
echo -e ""

echo -e "${GREEN}Deployment script completed successfully!${NC}"
