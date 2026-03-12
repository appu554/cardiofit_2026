#!/bin/bash

###############################################################################
# KB-7 Knowledge Factory - Multi-Region GCS Bucket Setup Script
###############################################################################
# This script sets up the GCS bucket folder structure for multi-region
# terminology deployment (AU=Australia, IN=India, US=United States)
#
# Folder Structure:
#   gs://{bucket}/au/{version}/   - Australian AMT terminology
#   gs://{bucket}/in/{version}/   - Indian CDCI terminology
#   gs://{bucket}/us/{version}/   - US RxNorm/SNOMED/LOINC terminology
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

# Default values (can be overridden via environment variables)
PROJECT_ID="${GCP_PROJECT_ID:-sincere-hybrid-477206-h2}"
ARTIFACTS_BUCKET="${KB7_ARTIFACTS_BUCKET:-${PROJECT_ID}-kb-artifacts-production}"
SOURCES_BUCKET="${KB7_SOURCES_BUCKET:-${PROJECT_ID}-kb-sources}"
REGION="${GCP_REGION:-us-central1}"

# Supported terminology regions
REGIONS=("au" "in" "us")

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}KB-7 Multi-Region GCS Bucket Setup${NC}"
echo -e "${BLUE}========================================${NC}\n"

###############################################################################
# Step 1: Parse Arguments
###############################################################################

show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --project-id ID       GCP Project ID (default: ${PROJECT_ID})"
    echo "  --artifacts-bucket    Artifacts bucket name (default: ${ARTIFACTS_BUCKET})"
    echo "  --sources-bucket      Sources bucket name (default: ${SOURCES_BUCKET})"
    echo "  --region REGION       GCP region for buckets (default: ${REGION})"
    echo "  --dry-run             Show what would be done without making changes"
    echo "  --help                Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  GCP_PROJECT_ID        Override project ID"
    echo "  KB7_ARTIFACTS_BUCKET  Override artifacts bucket name"
    echo "  KB7_SOURCES_BUCKET    Override sources bucket name"
    echo "  GCP_REGION            Override GCP region"
}

DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --project-id)
            PROJECT_ID="$2"
            shift 2
            ;;
        --artifacts-bucket)
            ARTIFACTS_BUCKET="$2"
            shift 2
            ;;
        --sources-bucket)
            SOURCES_BUCKET="$2"
            shift 2
            ;;
        --region)
            REGION="$2"
            shift 2
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
# Step 2: Validate Prerequisites
###############################################################################

echo -e "${YELLOW}Step 1: Validating prerequisites...${NC}"

# Check for required tools
command -v gcloud >/dev/null 2>&1 || { echo -e "${RED}Error: gcloud CLI not found${NC}"; exit 1; }
command -v gsutil >/dev/null 2>&1 || { echo -e "${RED}Error: gsutil not found${NC}"; exit 1; }

echo -e "${GREEN}  All prerequisites satisfied${NC}\n"

###############################################################################
# Step 3: Verify GCP Authentication
###############################################################################

echo -e "${YELLOW}Step 2: Verifying GCP authentication...${NC}"

if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "@"; then
    echo -e "${RED}Error: Not authenticated with GCP. Run 'gcloud auth login'${NC}"
    exit 1
fi

CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)
echo -e "${GREEN}  Authenticated with GCP${NC}"
echo -e "  Current project: ${CURRENT_PROJECT}"

# Set project if different
if [ "$CURRENT_PROJECT" != "$PROJECT_ID" ]; then
    echo -e "  Switching to project: ${PROJECT_ID}"
    if [ "$DRY_RUN" = false ]; then
        gcloud config set project "$PROJECT_ID"
    fi
fi

echo ""

###############################################################################
# Step 4: Create/Verify Buckets
###############################################################################

echo -e "${YELLOW}Step 3: Creating/verifying buckets...${NC}"

create_bucket_if_not_exists() {
    local bucket_name=$1
    local bucket_uri="gs://${bucket_name}"

    if gsutil ls -b "$bucket_uri" &>/dev/null; then
        echo -e "${GREEN}  Bucket exists: ${bucket_uri}${NC}"
    else
        echo -e "${CYAN}  Creating bucket: ${bucket_uri}${NC}"
        if [ "$DRY_RUN" = false ]; then
            gsutil mb -p "$PROJECT_ID" -l "$REGION" -b on "$bucket_uri"
            echo -e "${GREEN}  Created: ${bucket_uri}${NC}"
        else
            echo -e "${YELLOW}  [DRY-RUN] Would create: ${bucket_uri}${NC}"
        fi
    fi
}

create_bucket_if_not_exists "$ARTIFACTS_BUCKET"
create_bucket_if_not_exists "$SOURCES_BUCKET"

echo ""

###############################################################################
# Step 5: Create Regional Folder Structure
###############################################################################

echo -e "${YELLOW}Step 4: Creating regional folder structure...${NC}"

create_regional_structure() {
    local bucket=$1
    local region=$2

    # Create placeholder file to establish folder structure
    local placeholder_content="# KB-7 Regional Terminology - ${region^^}\n# Created: $(date -u +%Y-%m-%dT%H:%M:%SZ)\n"

    # Folders for each region
    local folders=(
        "${region}/latest/"
        "${region}/archive/"
    )

    for folder in "${folders[@]}"; do
        local folder_path="gs://${bucket}/${folder}"
        local placeholder_path="${folder_path}.placeholder"

        if gsutil ls "$folder_path" &>/dev/null 2>&1; then
            echo -e "${GREEN}    Folder exists: ${folder_path}${NC}"
        else
            echo -e "${CYAN}    Creating folder: ${folder_path}${NC}"
            if [ "$DRY_RUN" = false ]; then
                echo -e "$placeholder_content" | gsutil cp - "$placeholder_path"
                echo -e "${GREEN}    Created: ${folder_path}${NC}"
            else
                echo -e "${YELLOW}    [DRY-RUN] Would create: ${folder_path}${NC}"
            fi
        fi
    done
}

echo -e "\n${BLUE}  Artifacts Bucket: gs://${ARTIFACTS_BUCKET}${NC}"
for region in "${REGIONS[@]}"; do
    echo -e "  Region: ${region^^}"
    create_regional_structure "$ARTIFACTS_BUCKET" "$region"
done

echo -e "\n${BLUE}  Sources Bucket: gs://${SOURCES_BUCKET}${NC}"
for region in "${REGIONS[@]}"; do
    echo -e "  Region: ${region^^}"
    create_regional_structure "$SOURCES_BUCKET" "$region"
done

echo ""

###############################################################################
# Step 6: Set Bucket Labels and CORS
###############################################################################

echo -e "${YELLOW}Step 5: Configuring bucket settings...${NC}"

configure_bucket() {
    local bucket_name=$1
    local bucket_type=$2

    echo -e "  Configuring: gs://${bucket_name}"

    if [ "$DRY_RUN" = false ]; then
        # Set labels
        gsutil label ch \
            -l "project:kb7" \
            -l "environment:production" \
            -l "component:terminology" \
            -l "type:${bucket_type}" \
            "gs://${bucket_name}" 2>/dev/null || true

        # Enable versioning for artifacts bucket
        if [ "$bucket_type" = "artifacts" ]; then
            gsutil versioning set on "gs://${bucket_name}"
            echo -e "${GREEN}    Versioning enabled${NC}"
        fi

        # Set lifecycle rules (archive old versions after 90 days)
        cat > /tmp/lifecycle.json << 'EOF'
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "SetStorageClass", "storageClass": "NEARLINE"},
        "condition": {"age": 90, "matchesPrefix": ["au/archive/", "in/archive/", "us/archive/"]}
      },
      {
        "action": {"type": "SetStorageClass", "storageClass": "COLDLINE"},
        "condition": {"age": 365, "matchesPrefix": ["au/archive/", "in/archive/", "us/archive/"]}
      }
    ]
  }
}
EOF
        gsutil lifecycle set /tmp/lifecycle.json "gs://${bucket_name}"
        rm /tmp/lifecycle.json
        echo -e "${GREEN}    Lifecycle rules configured${NC}"
    else
        echo -e "${YELLOW}    [DRY-RUN] Would configure labels and lifecycle${NC}"
    fi
}

configure_bucket "$ARTIFACTS_BUCKET" "artifacts"
configure_bucket "$SOURCES_BUCKET" "sources"

echo ""

###############################################################################
# Step 7: Display Summary
###############################################################################

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Setup Complete!${NC}"
echo -e "${BLUE}========================================${NC}\n"

echo -e "${BLUE}Regional Bucket Structure:${NC}"
echo -e ""
echo -e "  ${CYAN}Artifacts Bucket:${NC} gs://${ARTIFACTS_BUCKET}"
echo -e "  ├── au/                    # Australian AMT terminology"
echo -e "  │   ├── latest/            # Current production kernel"
echo -e "  │   └── archive/           # Historical versions"
echo -e "  ├── in/                    # Indian CDCI terminology"
echo -e "  │   ├── latest/"
echo -e "  │   └── archive/"
echo -e "  └── us/                    # US RxNorm/SNOMED/LOINC"
echo -e "      ├── latest/"
echo -e "      └── archive/"
echo -e ""
echo -e "  ${CYAN}Sources Bucket:${NC} gs://${SOURCES_BUCKET}"
echo -e "  ├── au/                    # Downloaded AMT source files"
echo -e "  ├── in/                    # Downloaded CDCI source files"
echo -e "  └── us/                    # Downloaded RxNorm/SNOMED/LOINC files"
echo -e ""

echo -e "${BLUE}Regional Terminology Mapping:${NC}"
echo -e ""
echo -e "  ${CYAN}AU (Australia):${NC}"
echo -e "    - AMT (Australian Medicines Terminology)"
echo -e "    - SNOMED CT-AU (Australian Extension)"
echo -e "    - Module ID: 900062011000036103"
echo -e ""
echo -e "  ${CYAN}IN (India):${NC}"
echo -e "    - CDCI (Central Drug Standard Control Index)"
echo -e "    - SNOMED CT (International)"
echo -e ""
echo -e "  ${CYAN}US (United States):${NC}"
echo -e "    - RxNorm (Drug terminology)"
echo -e "    - SNOMED CT-US (US Extension)"
echo -e "    - LOINC (Lab codes)"
echo -e "    - Module ID: 731000124108"
echo -e ""

echo -e "${BLUE}Next Steps:${NC}"
echo -e "  1. Trigger regional factory workflow:"
echo -e "     ${YELLOW}gh workflow run kb-factory.yml -f region=au${NC}"
echo -e ""
echo -e "  2. Monitor workflow execution:"
echo -e "     ${YELLOW}gh run list --workflow=kb-factory.yml${NC}"
echo -e ""
echo -e "  3. View artifacts:"
echo -e "     ${YELLOW}gsutil ls gs://${ARTIFACTS_BUCKET}/au/latest/${NC}"
echo -e ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}NOTE: This was a dry run. No changes were made.${NC}"
    echo -e "${YELLOW}Remove --dry-run to apply changes.${NC}"
fi

echo -e "\n${GREEN}Script completed successfully!${NC}"
