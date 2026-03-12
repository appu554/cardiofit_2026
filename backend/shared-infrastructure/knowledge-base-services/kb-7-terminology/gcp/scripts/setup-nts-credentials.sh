#!/bin/bash

###############################################################################
# KB-7 Australian NTS Credentials Setup Script
###############################################################################
# Creates the Secret Manager secret for Australian NTS authentication
# Required for downloading SNOMED CT-AU from healthterminologies.gov.au
###############################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
PROJECT_ID="${GCP_PROJECT_ID:-sincere-hybrid-477206-h2}"
SECRET_NAME="kb7-nts-australia-credentials"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}KB-7 Australian NTS Credentials Setup${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Check gcloud
command -v gcloud >/dev/null 2>&1 || { echo -e "${RED}Error: gcloud CLI not found${NC}"; exit 1; }

# Set project
gcloud config set project "$PROJECT_ID" 2>/dev/null

echo -e "${YELLOW}This script will create a Secret Manager secret for Australian NTS authentication.${NC}"
echo -e "Secret Name: ${GREEN}${SECRET_NAME}${NC}"
echo -e "Project: ${GREEN}${PROJECT_ID}${NC}\n"

# Prompt for credentials
echo -e "${YELLOW}Enter your Australian NTS credentials:${NC}"
echo -e "(Register at https://www.healthterminologies.gov.au if you don't have an account)\n"

read -p "Username (email): " NTS_USERNAME
read -s -p "Password: " NTS_PASSWORD
echo ""

if [ -z "$NTS_USERNAME" ] || [ -z "$NTS_PASSWORD" ]; then
    echo -e "${RED}Error: Username and password are required${NC}"
    exit 1
fi

# Create JSON payload
CREDENTIALS_JSON=$(cat <<EOF
{
    "username": "${NTS_USERNAME}",
    "password": "${NTS_PASSWORD}"
}
EOF
)

# Check if secret exists
if gcloud secrets describe "$SECRET_NAME" --project="$PROJECT_ID" &>/dev/null; then
    echo -e "\n${YELLOW}Secret already exists. Adding new version...${NC}"

    echo "$CREDENTIALS_JSON" | gcloud secrets versions add "$SECRET_NAME" \
        --project="$PROJECT_ID" \
        --data-file=-

    echo -e "${GREEN}New secret version added successfully${NC}"
else
    echo -e "\n${YELLOW}Creating new secret...${NC}"

    # Create the secret
    gcloud secrets create "$SECRET_NAME" \
        --project="$PROJECT_ID" \
        --replication-policy="automatic" \
        --labels="service=kb7,region=au,type=credentials"

    # Add the first version
    echo "$CREDENTIALS_JSON" | gcloud secrets versions add "$SECRET_NAME" \
        --project="$PROJECT_ID" \
        --data-file=-

    echo -e "${GREEN}Secret created successfully${NC}"
fi

# Grant access to Cloud Run service account
COMPUTE_SA="${PROJECT_ID//-/_}@appspot.gserviceaccount.com"
DEFAULT_COMPUTE_SA="$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')-compute@developer.gserviceaccount.com"

echo -e "\n${YELLOW}Granting secret access to service accounts...${NC}"

for SA in "$COMPUTE_SA" "$DEFAULT_COMPUTE_SA"; do
    gcloud secrets add-iam-policy-binding "$SECRET_NAME" \
        --project="$PROJECT_ID" \
        --member="serviceAccount:${SA}" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet 2>/dev/null || true
done

echo -e "${GREEN}Secret access granted${NC}"

echo -e "\n${BLUE}========================================${NC}"
echo -e "${GREEN}Setup Complete!${NC}"
echo -e "${BLUE}========================================${NC}\n"

echo -e "Secret: ${GREEN}${SECRET_NAME}${NC}"
echo -e "Project: ${GREEN}${PROJECT_ID}${NC}"
echo -e ""
echo -e "${YELLOW}The SNOMED CT-AU downloader will automatically retrieve${NC}"
echo -e "${YELLOW}these credentials from Secret Manager.${NC}"
echo -e ""
echo -e "To verify the secret:"
echo -e "  ${BLUE}gcloud secrets versions list ${SECRET_NAME} --project=${PROJECT_ID}${NC}"
echo -e ""
