#!/bin/bash
# KB-7 GCP Quick Setup Script
# Automates Steps 5-8 of the deployment process

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  KB-7 GCP Quick Setup - Steps 5-8 Automation        ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# ══════════════════════════════════════════════════════════════════
# Step 5: Verify Authentication
# ══════════════════════════════════════════════════════════════════

echo -e "${YELLOW}━━━ Step 5: Verifying GCP Authentication ━━━${NC}"

if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}❌ gcloud CLI not found!${NC}"
    echo "Install: brew install --cask google-cloud-sdk"
    exit 1
fi

ACTIVE_ACCOUNT=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | head -1)
if [ -z "$ACTIVE_ACCOUNT" ]; then
    echo -e "${YELLOW}⚠️  Not authenticated. Opening browser for login...${NC}"
    gcloud auth login
    gcloud auth application-default login
    echo -e "${GREEN}✅ Authentication complete${NC}"
else
    echo -e "${GREEN}✅ Already authenticated as: $ACTIVE_ACCOUNT${NC}"
fi

# ══════════════════════════════════════════════════════════════════
# Step 6: Project Setup
# ══════════════════════════════════════════════════════════════════

echo ""
echo -e "${YELLOW}━━━ Step 6: GCP Project Setup ━━━${NC}"

CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)
if [ -n "$CURRENT_PROJECT" ] && [ "$CURRENT_PROJECT" != "(unset)" ]; then
    echo -e "${GREEN}✅ Active project: $CURRENT_PROJECT${NC}"
    read -p "Use this project? (y/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Available projects:"
        gcloud projects list --format="table(projectId,name)"
        echo ""
        read -p "Enter project ID to use (or 'new' to create): " PROJECT_CHOICE
        
        if [ "$PROJECT_CHOICE" = "new" ]; then
            read -p "Enter new project ID: " NEW_PROJECT_ID
            read -p "Enter project name: " NEW_PROJECT_NAME
            
            echo "Creating project $NEW_PROJECT_ID..."
            gcloud projects create $NEW_PROJECT_ID --name="$NEW_PROJECT_NAME"
            gcloud config set project $NEW_PROJECT_ID
            export PROJECT_ID=$NEW_PROJECT_ID
        else
            gcloud config set project $PROJECT_CHOICE
            export PROJECT_ID=$PROJECT_CHOICE
        fi
    else
        export PROJECT_ID=$CURRENT_PROJECT
    fi
else
    echo -e "${YELLOW}⚠️  No active project${NC}"
    read -p "Create new project? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Enter project ID: " NEW_PROJECT_ID
        read -p "Enter project name: " NEW_PROJECT_NAME
        
        echo "Creating project $NEW_PROJECT_ID..."
        gcloud projects create $NEW_PROJECT_ID --name="$NEW_PROJECT_NAME"
        gcloud config set project $NEW_PROJECT_ID
        export PROJECT_ID=$NEW_PROJECT_ID
    else
        echo "List of your projects:"
        gcloud projects list --format="table(projectId,name)"
        read -p "Enter project ID to use: " PROJECT_ID
        gcloud config set project $PROJECT_ID
    fi
fi

echo -e "${GREEN}✅ Using project: $PROJECT_ID${NC}"

# ══════════════════════════════════════════════════════════════════
# Step 7: Billing Setup
# ══════════════════════════════════════════════════════════════════

echo ""
echo -e "${YELLOW}━━━ Step 7: Verifying Billing ━━━${NC}"

BILLING_ENABLED=$(gcloud billing projects describe $PROJECT_ID --format="value(billingEnabled)" 2>/dev/null || echo "False")
if [ "$BILLING_ENABLED" = "True" ]; then
    echo -e "${GREEN}✅ Billing already enabled${NC}"
else
    echo -e "${YELLOW}⚠️  Billing not enabled${NC}"
    echo "Available billing accounts:"
    gcloud billing accounts list
    echo ""
    read -p "Enter billing account ID: " BILLING_ACCOUNT
    
    echo "Linking billing account..."
    gcloud billing projects link $PROJECT_ID --billing-account=$BILLING_ACCOUNT
    echo -e "${GREEN}✅ Billing enabled${NC}"
fi

# ══════════════════════════════════════════════════════════════════
# Step 8: Enable Required APIs
# ══════════════════════════════════════════════════════════════════

echo ""
echo -e "${YELLOW}━━━ Step 8: Enabling GCP APIs (this takes 2-3 minutes) ━━━${NC}"

REQUIRED_APIS=(
    "cloudfunctions.googleapis.com"
    "cloudbuild.googleapis.com"
    "cloudscheduler.googleapis.com"
    "workflows.googleapis.com"
    "storage.googleapis.com"
    "secretmanager.googleapis.com"
    "monitoring.googleapis.com"
    "logging.googleapis.com"
    "compute.googleapis.com"
    "artifactregistry.googleapis.com"
    "run.googleapis.com"
    "eventarc.googleapis.com"
)

echo "Enabling ${#REQUIRED_APIS[@]} APIs..."

for api in "${REQUIRED_APIS[@]}"; do
    echo -n "  • $api ... "
    if gcloud services enable $api --project=$PROJECT_ID 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi
done

echo ""
echo -e "${YELLOW}⏳ Waiting 60 seconds for APIs to be fully enabled...${NC}"
for i in {60..1}; do
    echo -ne "\r  ${i} seconds remaining...  "
    sleep 1
done
echo ""

# Verify APIs
echo "Verifying critical APIs..."
CRITICAL_APIS=("cloudfunctions" "storage" "workflows" "secretmanager")
for api in "${CRITICAL_APIS[@]}"; do
    if gcloud services list --enabled --project=$PROJECT_ID 2>/dev/null | grep -q "$api"; then
        echo -e "  ${GREEN}✓${NC} $api"
    else
        echo -e "  ${RED}✗${NC} $api"
    fi
done

# ══════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  ✅ Quick Setup Complete!                            ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "📋 ${BLUE}Setup Summary:${NC}"
echo -e "   Project ID: ${GREEN}$PROJECT_ID${NC}"
echo -e "   Region: ${GREEN}us-central1${NC}"
echo -e "   Billing: ${GREEN}Enabled${NC}"
echo -e "   APIs: ${GREEN}12 services enabled${NC}"
echo ""
echo -e "📝 ${BLUE}Next Steps:${NC}"
echo ""
echo -e "  ${YELLOW}Step 9:${NC} Gather API credentials (may take 24-48 hours)"
echo "    • NHS TRUD API key: https://isd.digital.nhs.uk/trud/"
echo "    • UMLS API key: https://uts.nlm.nih.gov/"
echo "    • LOINC credentials: https://loinc.org/downloads/"
echo "    • GitHub PAT: https://github.com/settings/tokens"
echo ""
echo -e "  ${YELLOW}Step 10:${NC} Configure Terraform variables"
echo "    cd ../terraform"
echo "    cp terraform.tfvars.example terraform.tfvars"
echo "    nano terraform.tfvars  # Add your API credentials"
echo ""
echo -e "  ${YELLOW}Step 11:${NC} Deploy infrastructure"
echo "    terraform init"
echo "    terraform plan"
echo "    terraform apply"
echo ""
echo -e "${BLUE}📚 Documentation:${NC}"
echo "  • Full guide: ../INSTALLATION_GUIDE.md"
echo "  • Quick start: ../QUICKSTART_DEPLOYMENT.md"
echo "  • Checklist: ../DEPLOYMENT_CHECKLIST.md"
echo ""

# Save project info for later use
cat > /tmp/kb7-gcp-setup.env << EOL
export PROJECT_ID="$PROJECT_ID"
export REGION="us-central1"
export ENVIRONMENT="production"
EOL

echo -e "${GREEN}💾 Project configuration saved to: /tmp/kb7-gcp-setup.env${NC}"
echo -e "   Load with: ${BLUE}source /tmp/kb7-gcp-setup.env${NC}"
echo ""
