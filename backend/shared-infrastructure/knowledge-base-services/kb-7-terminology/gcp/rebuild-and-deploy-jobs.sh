#!/bin/bash

# KB-7 Cloud Run Jobs - Rebuild and Deploy Script
# Rebuilds container images with fixed code and updates Cloud Run Jobs
# Date: 2025-11-25

set -e  # Exit on error

PROJECT_ID="sincere-hybrid-477206-h2"
REGION="us-central1"
REPOSITORY="cloud-run-source-deploy"
FUNCTIONS_DIR="$(cd "$(dirname "$0")/functions" && pwd)"

echo "========================================"
echo "KB-7 Cloud Run Jobs Rebuild & Deploy"
echo "========================================"
echo ""
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo "Repository: $REPOSITORY"
echo ""

# Function to build and push image
build_and_push() {
    local service_name=$1
    local function_dir=$2

    echo "----------------------------------------"
    echo "Building: $service_name"
    echo "----------------------------------------"

    cd "$FUNCTIONS_DIR/$function_dir"

    # Build image using Cloud Build
    echo "→ Building container image..."
    gcloud builds submit \
        --project=$PROJECT_ID \
        --tag="$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$service_name:latest" \
        .

    if [ $? -eq 0 ]; then
        echo "✅ Image built successfully: $service_name"
    else
        echo "❌ Failed to build image: $service_name"
        return 1
    fi

    echo ""
}

# Function to update Cloud Run Job
update_job() {
    local job_name=$1
    local service_name=$2

    echo "→ Updating Cloud Run Job: $job_name..."
    gcloud run jobs update $job_name \
        --region=$REGION \
        --image="$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$service_name:latest"

    if [ $? -eq 0 ]; then
        echo "✅ Job updated successfully: $job_name"
    else
        echo "⚠️ Warning: Failed to update job: $job_name (job may not exist yet)"
    fi

    echo ""
}

# Build and deploy all services
echo "========================================"
echo "Phase 1: Building Container Images"
echo "========================================"
echo ""

build_and_push "kb7-snomed-downloader" "snomed-downloader"
build_and_push "kb7-rxnorm-downloader" "rxnorm-downloader"
build_and_push "kb7-loinc-downloader" "loinc-downloader"
build_and_push "kb7-github-dispatcher" "github-dispatcher"

echo ""
echo "========================================"
echo "Phase 2: Updating Cloud Run Jobs"
echo "========================================"
echo ""

update_job "kb7-snomed-job-production" "kb7-snomed-downloader"
update_job "kb7-rxnorm-job-production" "kb7-rxnorm-downloader"
update_job "kb7-loinc-job-production" "kb7-loinc-downloader"
update_job "kb7-github-dispatcher-job-production" "kb7-github-dispatcher"

echo ""
echo "========================================"
echo "Deployment Complete!"
echo "========================================"
echo ""
echo "Next Steps:"
echo "1. Test individual jobs manually:"
echo "   gcloud run jobs execute kb7-loinc-job-production --region=$REGION --wait"
echo ""
echo "2. Execute full workflow:"
echo "   gcloud workflows execute kb7-factory-workflow-production --location=$REGION"
echo ""
echo "3. Monitor job logs:"
echo "   gcloud logging read \"resource.type=cloud_run_job\" --limit=50 --format=\"table(timestamp,resource.labels.job_name,textPayload)\""
echo ""
