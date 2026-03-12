#!/bin/bash

###############################################################################
# KB-7 GitHub Dispatcher AU Deployment Script
###############################################################################
# Builds and deploys the Australian GitHub Dispatcher as a Cloud Run Job
# Triggers GitHub Actions on knowledge-factory-au repository
###############################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FUNCTIONS_DIR="$(dirname "$SCRIPT_DIR")/functions"
DISPATCHER_DIR="$FUNCTIONS_DIR/github-dispatcher-au"

PROJECT_ID="${GCP_PROJECT_ID:-sincere-hybrid-477206-h2}"
# AU dispatcher runs in australia-southeast1 to maintain regional affinity
REGION="${GCP_REGION:-australia-southeast1}"
ENVIRONMENT="${ENVIRONMENT:-production}"

# Artifact Registry configuration
# Use us-central1 for Artifact Registry (faster builds, shared with other images)
REGISTRY_REGION="us-central1"
REPOSITORY="kb7-terminology"
IMAGE_NAME="github-dispatcher-au"
IMAGE_TAG="${IMAGE_TAG:-latest}"
FULL_IMAGE="$REGISTRY_REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE_NAME:$IMAGE_TAG"

# Cloud Run Job configuration
JOB_NAME="kb7-github-dispatcher-au-job-${ENVIRONMENT}"
SOURCE_BUCKET="${PROJECT_ID}-kb-sources-${ENVIRONMENT}"
GITHUB_TOKEN_SECRET="kb7-github-token-${ENVIRONMENT}"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}KB-7 GitHub Dispatcher AU Deployment${NC}"
echo -e "${BLUE}============================================${NC}\n"

###############################################################################
# Parse Arguments
###############################################################################

show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --project-id ID       GCP Project ID (default: ${PROJECT_ID})"
    echo "  --region REGION       Deploy region (default: ${REGION})"
    echo "  --environment ENV     Environment (default: ${ENVIRONMENT})"
    echo "  --tag TAG             Docker image tag (default: ${IMAGE_TAG})"
    echo "  --skip-build          Skip Docker build, only deploy"
    echo "  --build-only          Only build image, don't deploy"
    echo "  --run-job             Run the job after deployment"
    echo "  --dry-run             Show what would be done"
    echo "  --help                Show this help"
}

SKIP_BUILD=false
BUILD_ONLY=false
RUN_JOB=false
DRY_RUN=false

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
        --tag)
            IMAGE_TAG="$2"
            FULL_IMAGE="$REGISTRY_REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE_NAME:$IMAGE_TAG"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        --run-job)
            RUN_JOB=true
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

# Update derived variables
SOURCE_BUCKET="${PROJECT_ID}-kb-sources-${ENVIRONMENT}"
JOB_NAME="kb7-github-dispatcher-au-job-${ENVIRONMENT}"
FULL_IMAGE="$REGISTRY_REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE_NAME:$IMAGE_TAG"

###############################################################################
# Step 1: Validate Prerequisites
###############################################################################

echo -e "${YELLOW}Step 1: Validating prerequisites...${NC}"

command -v gcloud >/dev/null 2>&1 || { echo -e "${RED}Error: gcloud CLI not found${NC}"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo -e "${RED}Error: docker not found${NC}"; exit 1; }

if [ ! -d "$DISPATCHER_DIR" ]; then
    echo -e "${RED}Error: Dispatcher directory not found: $DISPATCHER_DIR${NC}"
    exit 1
fi

gcloud config set project "$PROJECT_ID" 2>/dev/null

echo -e "${GREEN}  Project: ${PROJECT_ID}${NC}"
echo -e "${GREEN}  Deploy Region: ${REGION} (australia-southeast1)${NC}"
echo -e "${GREEN}  Registry Region: ${REGISTRY_REGION}${NC}"
echo -e "${GREEN}  Environment: ${ENVIRONMENT}${NC}"
echo -e "${GREEN}  Image: ${FULL_IMAGE}${NC}"
echo -e "${GREEN}  Target Repo: onkarshahi-IND/knowledge-factory-au${NC}\n"

###############################################################################
# Step 2: Create Artifact Registry Repository (if needed)
###############################################################################

echo -e "${YELLOW}Step 2: Ensuring Artifact Registry repository exists...${NC}"

if [ "$DRY_RUN" = true ]; then
    echo -e "${CYAN}  [DRY-RUN] Would ensure repository: $REPOSITORY${NC}"
else
    if ! gcloud artifacts repositories describe "$REPOSITORY" --location="$REGISTRY_REGION" &>/dev/null; then
        echo -e "  Creating repository: $REPOSITORY"
        gcloud artifacts repositories create "$REPOSITORY" \
            --repository-format=docker \
            --location="$REGISTRY_REGION" \
            --description="KB-7 Terminology Service Docker images"
    else
        echo -e "  Repository exists: $REPOSITORY"
    fi
fi

echo ""

###############################################################################
# Step 3: Build Docker Image
###############################################################################

if [ "$SKIP_BUILD" = false ]; then
    echo -e "${YELLOW}Step 3: Building Docker image...${NC}"

    if [ "$DRY_RUN" = true ]; then
        echo -e "${CYAN}  [DRY-RUN] Would build: ${FULL_IMAGE}${NC}"
    else
        echo -e "  Building: ${FULL_IMAGE}"

        # Configure Docker for Artifact Registry
        gcloud auth configure-docker "$REGISTRY_REGION-docker.pkg.dev" --quiet

        # Build and push using Cloud Build
        cd "$DISPATCHER_DIR"

        gcloud builds submit \
            --tag "$FULL_IMAGE" \
            --project "$PROJECT_ID" \
            --region "$REGISTRY_REGION" \
            --quiet

        echo -e "${GREEN}  Image built and pushed: ${FULL_IMAGE}${NC}"
    fi
else
    echo -e "${YELLOW}Step 3: Skipping Docker build (--skip-build)${NC}"
fi

echo ""

if [ "$BUILD_ONLY" = true ]; then
    echo -e "${GREEN}Build complete (--build-only specified)${NC}"
    exit 0
fi

###############################################################################
# Step 4: Deploy Cloud Run Job
###############################################################################

echo -e "${YELLOW}Step 4: Deploying Cloud Run Job to ${REGION}...${NC}"

if [ "$DRY_RUN" = true ]; then
    echo -e "${CYAN}  [DRY-RUN] Would deploy job: ${JOB_NAME} in ${REGION}${NC}"
else
    echo -e "  Job Name: ${JOB_NAME}"
    echo -e "  Region: ${REGION}"

    # Check if job exists
    if gcloud run jobs describe "$JOB_NAME" --region="$REGION" &>/dev/null; then
        echo -e "  Updating existing job..."
        ACTION="update"
    else
        echo -e "  Creating new job..."
        ACTION="create"
    fi

    gcloud run jobs $ACTION "$JOB_NAME" \
        --region="$REGION" \
        --image="$FULL_IMAGE" \
        --set-env-vars="PROJECT_ID=${PROJECT_ID},ENVIRONMENT=${ENVIRONMENT},SECRET_NAME=${GITHUB_TOKEN_SECRET},GITHUB_REPO=onkarshahi-IND/knowledge-factory-au,GCS_BUCKET_SOURCES=${SOURCE_BUCKET}" \
        --memory=512Mi \
        --cpu=1 \
        --task-timeout=300s \
        --max-retries=2 \
        --quiet

    echo -e "${GREEN}  Job deployed: ${JOB_NAME}${NC}"
fi

echo ""

###############################################################################
# Step 5: Run Job (if requested)
###############################################################################

if [ "$RUN_JOB" = true ]; then
    echo -e "${YELLOW}Step 5: Running job...${NC}"

    if [ "$DRY_RUN" = true ]; then
        echo -e "${CYAN}  [DRY-RUN] Would execute: ${JOB_NAME}${NC}"
    else
        echo -e "  Executing: ${JOB_NAME}"

        EXECUTION=$(gcloud run jobs execute "$JOB_NAME" \
            --region="$REGION" \
            --format="value(metadata.name)" \
            --async)

        echo -e "${GREEN}  Execution started: ${EXECUTION}${NC}"
        echo -e ""
        echo -e "  Monitor execution:"
        echo -e "    ${BLUE}gcloud run jobs executions describe ${EXECUTION} --region=${REGION}${NC}"
        echo -e ""
        echo -e "  View logs:"
        echo -e "    ${BLUE}gcloud run jobs executions logs ${EXECUTION} --region=${REGION}${NC}"
    fi

    echo ""
fi

###############################################################################
# Summary
###############################################################################

echo -e "${BLUE}============================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${BLUE}============================================${NC}\n"

echo -e "${BLUE}GitHub Dispatcher AU:${NC}"
echo -e "  Job Name: ${GREEN}${JOB_NAME}${NC}"
echo -e "  Image: ${GREEN}${FULL_IMAGE}${NC}"
echo -e "  Deploy Region: ${GREEN}${REGION}${NC}"
echo -e ""
echo -e "${BLUE}Target Repository:${NC}"
echo -e "  GitHub Repo: ${GREEN}onkarshahi-IND/knowledge-factory-au${NC}"
echo -e "  Event Type: ${GREEN}terminology-update-au${NC}"
echo -e ""
echo -e "${BLUE}Terminology Coverage:${NC}"
echo -e "  - SNOMED CT-AU (Module 32506021000036107)"
echo -e "  - AMT - Australian Medicines Terminology (Module 900062011000036103)"
echo -e "  - LOINC (International - shared)"
echo -e ""
echo -e "${BLUE}Execute Job:${NC}"
echo -e "  ${YELLOW}gcloud run jobs execute ${JOB_NAME} --region=${REGION}${NC}"
echo -e ""
echo -e "${BLUE}View Logs:${NC}"
echo -e "  ${YELLOW}gcloud logging read 'resource.type=\"cloud_run_job\" AND resource.labels.job_name=\"${JOB_NAME}\"' --limit=50${NC}"
echo -e ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}NOTE: This was a dry run. No changes were made.${NC}"
fi
