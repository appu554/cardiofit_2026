#!/bin/bash

###############################################################################
# KB-7 Knowledge Factory - Multi-Region Workflow Deployment Script
###############################################################################
# Deploys the multi-region GCP Cloud Workflow and required Cloud Run Jobs
# for AU (Australia), IN (India), and US (United States) terminology regions
###############################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GCP_DIR="$(dirname "$SCRIPT_DIR")"
WORKFLOWS_DIR="$GCP_DIR/workflows"

# Default values
PROJECT_ID="${GCP_PROJECT_ID:-sincere-hybrid-477206-h2}"
REGION="${GCP_REGION:-us-central1}"
ENVIRONMENT="${ENVIRONMENT:-production}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}KB-7 Multi-Region Workflow Deployment${NC}"
echo -e "${BLUE}========================================${NC}\n"

###############################################################################
# Parse Arguments
###############################################################################

show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --project-id ID       GCP Project ID (default: ${PROJECT_ID})"
    echo "  --region REGION       GCP region (default: ${REGION})"
    echo "  --environment ENV     Environment name (default: ${ENVIRONMENT})"
    echo "  --workflow-only       Only deploy the workflow, skip job creation"
    echo "  --jobs-only           Only create Cloud Run Jobs, skip workflow"
    echo "  --dry-run             Show what would be done without making changes"
    echo "  --help                Show this help message"
}

DRY_RUN=false
WORKFLOW_ONLY=false
JOBS_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --project-id)
            PROJECT_ID="$2"
            shift 2
            ;;
        --region)
            REGION="$2"
            shift 2
            ;;
        --environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --workflow-only)
            WORKFLOW_ONLY=true
            shift
            ;;
        --jobs-only)
            JOBS_ONLY=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

###############################################################################
# Step 1: Validate Prerequisites
###############################################################################

echo -e "${YELLOW}Step 1: Validating prerequisites...${NC}"

command -v gcloud >/dev/null 2>&1 || { echo -e "${RED}Error: gcloud CLI not found${NC}"; exit 1; }

# Check authentication
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "@"; then
    echo -e "${RED}Error: Not authenticated with GCP. Run 'gcloud auth login'${NC}"
    exit 1
fi

# Set project
gcloud config set project "$PROJECT_ID" 2>/dev/null

echo -e "${GREEN}  Project: ${PROJECT_ID}${NC}"
echo -e "${GREEN}  Region: ${REGION}${NC}"
echo -e "${GREEN}  Environment: ${ENVIRONMENT}${NC}\n"

###############################################################################
# Step 2: Create Regional Cloud Run Jobs
###############################################################################

if [ "$WORKFLOW_ONLY" = false ]; then
    echo -e "${YELLOW}Step 2: Creating regional Cloud Run Jobs...${NC}\n"

    # Artifacts bucket for terminology downloads
    ARTIFACTS_BUCKET="${PROJECT_ID}-kb-artifacts-${ENVIRONMENT}"
    SOURCES_BUCKET="${PROJECT_ID}-kb-sources"

    # Docker image (shared snomed-toolkit image works for all SNOMED-based formats)
    TOOLKIT_IMAGE="ghcr.io/onkarshahi-ind/snomed-toolkit:latest"
    CONVERTER_IMAGE="ghcr.io/onkarshahi-ind/converters:latest"

    create_job() {
        local job_name=$1
        local terminology=$2
        local term_region=$3
        local description=$4
        local image=$5
        local extra_env=$6

        echo -e "${CYAN}  Creating job: ${job_name}${NC}"
        echo -e "    Terminology: ${terminology}"
        echo -e "    Region: ${term_region}"

        if [ "$DRY_RUN" = true ]; then
            echo -e "${YELLOW}    [DRY-RUN] Would create Cloud Run Job${NC}"
            return
        fi

        # Check if job exists
        if gcloud run jobs describe "$job_name" --region="$REGION" &>/dev/null; then
            echo -e "    Job exists, updating..."
            gcloud run jobs update "$job_name" \
                --region="$REGION" \
                --image="$image" \
                --set-env-vars="TERMINOLOGY=${terminology},REGION=${term_region},GCS_BUCKET=${SOURCES_BUCKET}/${term_region},OUTPUT_BUCKET=${ARTIFACTS_BUCKET}/${term_region}${extra_env}" \
                --memory=4Gi \
                --cpu=2 \
                --task-timeout=3600s \
                --max-retries=1 \
                --quiet
        else
            echo -e "    Creating new job..."
            gcloud run jobs create "$job_name" \
                --region="$REGION" \
                --image="$image" \
                --set-env-vars="TERMINOLOGY=${terminology},REGION=${term_region},GCS_BUCKET=${SOURCES_BUCKET}/${term_region},OUTPUT_BUCKET=${ARTIFACTS_BUCKET}/${term_region}${extra_env}" \
                --memory=4Gi \
                --cpu=2 \
                --task-timeout=3600s \
                --max-retries=1 \
                --quiet
        fi
        echo -e "${GREEN}    Created: ${job_name}${NC}"
    }

    echo -e "\n${BLUE}  --- AU Region Jobs (Australia) ---${NC}"

    # SNOMED CT-AU Job (Australian Extension - Module 32506021000036107)
    create_job \
        "kb7-snomed-au-job-${ENVIRONMENT}" \
        "snomed-ct-au" \
        "au" \
        "SNOMED CT Australian Extension" \
        "$TOOLKIT_IMAGE" \
        ",SNOMED_MODULE_ID=32506021000036107,SNOMED_EDITION=AU"

    # AMT Job (Australian Medicines Terminology - Module 900062011000036103)
    create_job \
        "kb7-amt-job-${ENVIRONMENT}" \
        "amt" \
        "au" \
        "Australian Medicines Terminology" \
        "$TOOLKIT_IMAGE" \
        ",SNOMED_MODULE_ID=900062011000036103,TERMINOLOGY_TYPE=AMT"

    echo -e "\n${BLUE}  --- IN Region Jobs (India) ---${NC}"

    # SNOMED CT-IN Job (International for India)
    create_job \
        "kb7-snomed-in-job-${ENVIRONMENT}" \
        "snomed-ct-int" \
        "in" \
        "SNOMED CT International for India" \
        "$TOOLKIT_IMAGE" \
        ",SNOMED_EDITION=INT"

    # CDCI Job (Central Drug Standard Control Index)
    create_job \
        "kb7-cdci-job-${ENVIRONMENT}" \
        "cdci" \
        "in" \
        "Central Drug Standard Control Index" \
        "$CONVERTER_IMAGE" \
        ",TERMINOLOGY_TYPE=CDCI,SOURCE_FORMAT=RF2"

    echo -e "\n${GREEN}  All regional Cloud Run Jobs created${NC}\n"
fi

###############################################################################
# Step 3: Deploy Multi-Region Workflow
###############################################################################

if [ "$JOBS_ONLY" = false ]; then
    echo -e "${YELLOW}Step 3: Deploying multi-region workflow...${NC}"

    WORKFLOW_FILE="$WORKFLOWS_DIR/kb-factory-multiregion-workflow.yaml"
    WORKFLOW_NAME="kb7-factory-multiregion-workflow-${ENVIRONMENT}"

    if [ ! -f "$WORKFLOW_FILE" ]; then
        echo -e "${RED}Error: Workflow file not found: $WORKFLOW_FILE${NC}"
        exit 1
    fi

    if [ "$DRY_RUN" = true ]; then
        echo -e "${YELLOW}  [DRY-RUN] Would deploy workflow: ${WORKFLOW_NAME}${NC}"
    else
        echo -e "  Deploying: ${WORKFLOW_NAME}"

        gcloud workflows deploy "$WORKFLOW_NAME" \
            --location="$REGION" \
            --source="$WORKFLOW_FILE" \
            --description="KB-7 Knowledge Factory Multi-Region Workflow (AU/IN/US)" \
            --quiet

        echo -e "${GREEN}  Workflow deployed: ${WORKFLOW_NAME}${NC}"
    fi
fi

echo ""

###############################################################################
# Step 4: Display Summary
###############################################################################

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${BLUE}========================================${NC}\n"

echo -e "${BLUE}Multi-Region Workflow:${NC}"
echo -e "  Name: kb7-factory-multiregion-workflow-${ENVIRONMENT}"
echo -e "  Region: ${REGION}"
echo -e ""

echo -e "${BLUE}Regional Cloud Run Jobs:${NC}"
echo -e ""
echo -e "  ${CYAN}AU (Australia):${NC}"
echo -e "    - kb7-snomed-au-job-${ENVIRONMENT}  (SNOMED CT-AU, Module 32506021000036107)"
echo -e "    - kb7-amt-job-${ENVIRONMENT}        (AMT, Module 900062011000036103)"
echo -e ""
echo -e "  ${CYAN}IN (India):${NC}"
echo -e "    - kb7-snomed-in-job-${ENVIRONMENT}  (SNOMED CT International)"
echo -e "    - kb7-cdci-job-${ENVIRONMENT}       (CDCI)"
echo -e ""
echo -e "  ${CYAN}US (United States):${NC} (existing jobs)"
echo -e "    - kb7-snomed-job-${ENVIRONMENT}     (SNOMED CT-US, Module 731000124108)"
echo -e "    - kb7-rxnorm-job-${ENVIRONMENT}     (RxNorm)"
echo -e "    - kb7-loinc-job-${ENVIRONMENT}      (LOINC)"
echo -e ""

echo -e "${BLUE}Execute Workflow by Region:${NC}"
echo -e ""
echo -e "  ${CYAN}US Region:${NC}"
echo -e "    ${YELLOW}gcloud workflows execute kb7-factory-multiregion-workflow-${ENVIRONMENT} \\${NC}"
echo -e "    ${YELLOW}  --location=${REGION} --data='{\"region\":\"us\"}'${NC}"
echo -e ""
echo -e "  ${CYAN}AU Region:${NC}"
echo -e "    ${YELLOW}gcloud workflows execute kb7-factory-multiregion-workflow-${ENVIRONMENT} \\${NC}"
echo -e "    ${YELLOW}  --location=${REGION} --data='{\"region\":\"au\"}'${NC}"
echo -e ""
echo -e "  ${CYAN}IN Region:${NC}"
echo -e "    ${YELLOW}gcloud workflows execute kb7-factory-multiregion-workflow-${ENVIRONMENT} \\${NC}"
echo -e "    ${YELLOW}  --location=${REGION} --data='{\"region\":\"in\"}'${NC}"
echo -e ""

echo -e "${BLUE}Monitor Execution:${NC}"
echo -e "  ${YELLOW}gcloud workflows executions list --workflow=kb7-factory-multiregion-workflow-${ENVIRONMENT} --location=${REGION}${NC}"
echo -e ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}NOTE: This was a dry run. No changes were made.${NC}"
    echo -e "${YELLOW}Remove --dry-run to apply changes.${NC}"
fi

echo -e "\n${GREEN}Script completed successfully!${NC}"
