#!/bin/bash
###############################################################################
# Quick Deploy AU Downloader - Run this after 'gcloud auth login'
###############################################################################

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ID="sincere-hybrid-477206-h2"
REGION="us-central1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}KB-7 AU Region Quick Deployment${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Set project
gcloud config set project "$PROJECT_ID"

###############################################################################
# Step 1: Create NTS Secret
###############################################################################
echo -e "${YELLOW}Step 1: Creating NTS credentials secret...${NC}"

SECRET_NAME="kb7-nts-australia-credentials"
CREDENTIALS_JSON='{"username": "onkarshahi@gmail.com", "password": "19Snomed-au$47"}'

if gcloud secrets describe "$SECRET_NAME" &>/dev/null; then
    echo "Secret exists, adding new version..."
    echo "$CREDENTIALS_JSON" | gcloud secrets versions add "$SECRET_NAME" --data-file=-
else
    gcloud secrets create "$SECRET_NAME" \
        --replication-policy="automatic" \
        --labels="service=kb7,region=au,type=credentials"
    echo "$CREDENTIALS_JSON" | gcloud secrets versions add "$SECRET_NAME" --data-file=-
fi
echo -e "${GREEN}✅ Secret created/updated${NC}\n"

###############################################################################
# Step 2: Create Artifact Registry
###############################################################################
echo -e "${YELLOW}Step 2: Creating Artifact Registry repository...${NC}"

REPOSITORY="kb7-terminology"

if gcloud artifacts repositories describe "$REPOSITORY" --location="$REGION" &>/dev/null; then
    echo "Repository already exists"
else
    gcloud artifacts repositories create "$REPOSITORY" \
        --repository-format=docker \
        --location="$REGION" \
        --description="KB-7 Terminology Service Docker images"
fi
echo -e "${GREEN}✅ Repository ready${NC}\n"

###############################################################################
# Step 3: Build and Push Docker Image
###############################################################################
echo -e "${YELLOW}Step 3: Building Docker image with Cloud Build...${NC}"

IMAGE="$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/snomed-au-downloader:latest"
DOWNLOADER_DIR="$SCRIPT_DIR/../functions/snomed-au-downloader"

cd "$DOWNLOADER_DIR"
gcloud builds submit --tag "$IMAGE" --quiet

echo -e "${GREEN}✅ Image built: $IMAGE${NC}\n"

###############################################################################
# Step 4: Deploy Cloud Run Job
###############################################################################
echo -e "${YELLOW}Step 4: Deploying Cloud Run Job...${NC}"

JOB_NAME="kb7-snomed-au-job-production"
SOURCE_BUCKET="${PROJECT_ID}-kb-sources"

# Grant Secret access to compute service account
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
COMPUTE_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

gcloud secrets add-iam-policy-binding "$SECRET_NAME" \
    --member="serviceAccount:$COMPUTE_SA" \
    --role="roles/secretmanager.secretAccessor" --quiet 2>/dev/null || true

# Create or update job
if gcloud run jobs describe "$JOB_NAME" --region="$REGION" &>/dev/null; then
    echo "Updating existing job..."
    gcloud run jobs update "$JOB_NAME" \
        --region="$REGION" \
        --image="$IMAGE" \
        --set-env-vars="PROJECT_ID=${PROJECT_ID},SOURCE_BUCKET=${SOURCE_BUCKET},ENVIRONMENT=production,NTS_SECRET_NAME=${SECRET_NAME}" \
        --memory=4Gi \
        --cpu=2 \
        --task-timeout=3600s \
        --max-retries=2 \
        --quiet
else
    echo "Creating new job..."
    gcloud run jobs create "$JOB_NAME" \
        --region="$REGION" \
        --image="$IMAGE" \
        --set-env-vars="PROJECT_ID=${PROJECT_ID},SOURCE_BUCKET=${SOURCE_BUCKET},ENVIRONMENT=production,NTS_SECRET_NAME=${SECRET_NAME}" \
        --memory=4Gi \
        --cpu=2 \
        --task-timeout=3600s \
        --max-retries=2 \
        --quiet
fi

echo -e "${GREEN}✅ Job deployed: $JOB_NAME${NC}\n"

###############################################################################
# Step 5: Create GCS Bucket folders
###############################################################################
echo -e "${YELLOW}Step 5: Creating AU folder in sources bucket...${NC}"

gsutil ls "gs://${SOURCE_BUCKET}/au/" &>/dev/null || \
    echo "# AU Region Placeholder" | gsutil cp - "gs://${SOURCE_BUCKET}/au/.placeholder"

echo -e "${GREEN}✅ Bucket structure ready${NC}\n"

###############################################################################
# Summary
###############################################################################
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${BLUE}========================================${NC}\n"

echo -e "To run the AU downloader:"
echo -e "  ${YELLOW}gcloud run jobs execute $JOB_NAME --region=$REGION${NC}\n"

echo -e "To trigger full AU workflow:"
echo -e "  ${YELLOW}gcloud workflows execute kb7-factory-multiregion-workflow-production \\${NC}"
echo -e "  ${YELLOW}  --location=$REGION --data='{\"region\":\"au\"}'${NC}\n"
