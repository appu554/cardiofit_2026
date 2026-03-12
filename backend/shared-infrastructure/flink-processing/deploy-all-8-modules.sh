#!/bin/bash

###############################################################################
# Flink Complete Pipeline Deployment - All 8 Modules
# Phase 1: Foundation Deployment (Static YAML Loading)
###############################################################################

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FLINK_HOME="${FLINK_HOME:-http://localhost:8081}"
JAR_PATH="target/flink-ehr-intelligence-1.0.0.jar"
PARALLELISM=2

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Flink EHR Intelligence Platform - Complete Deployment        ║${NC}"
echo -e "${BLUE}║  Phase 1: Static YAML Loading (8 Modules)                    ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

###############################################################################
# Step 1: Pre-Deployment Checks
###############################################################################

echo -e "${YELLOW}[Step 1/4] Pre-Deployment Checks...${NC}"

# Check Flink cluster
echo -n "  ⏳ Checking Flink cluster..."
if curl -s "${FLINK_HOME}" > /dev/null 2>&1; then
    echo -e " ${GREEN}✓ Running${NC}"
else
    echo -e " ${RED}✗ Not accessible at ${FLINK_HOME}${NC}"
    echo -e "${YELLOW}Please start Flink cluster first:${NC}"
    echo "    docker-compose up -d"
    exit 1
fi

# Check Kafka
echo -n "  ⏳ Checking Kafka..."
if docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; then
    echo -e " ${GREEN}✓ Running${NC}"
else
    echo -e " ${RED}✗ Not running${NC}"
    echo -e "${YELLOW}Please start Kafka first:${NC}"
    echo "    docker-compose up -d kafka"
    exit 1
fi

# Check JAR exists
echo -n "  ⏳ Checking JAR file..."
if [ -f "$JAR_PATH" ]; then
    JAR_SIZE=$(ls -lh "$JAR_PATH" | awk '{print $5}')
    echo -e " ${GREEN}✓ Found ($JAR_SIZE)${NC}"
else
    echo -e " ${RED}✗ Not found${NC}"
    echo -e "${YELLOW}Building JAR...${NC}"
    mvn clean package -DskipTests
    if [ $? -ne 0 ]; then
        echo -e "${RED}Build failed!${NC}"
        exit 1
    fi
fi

echo ""

###############################################################################
# Step 2: Cancel Existing Jobs (if any)
###############################################################################

echo -e "${YELLOW}[Step 2/4] Cleaning up existing jobs...${NC}"

RUNNING_JOBS=$(curl -s "${FLINK_HOME}/jobs/overview" | jq -r '.jobs[] | select(.state=="RUNNING") | .jid' 2>/dev/null || echo "")

if [ -n "$RUNNING_JOBS" ]; then
    echo "  ⏳ Found running jobs, cancelling..."
    for job_id in $RUNNING_JOBS; do
        echo -n "    Cancelling job $job_id..."
        curl -s -X PATCH "${FLINK_HOME}/jobs/$job_id?mode=cancel" > /dev/null
        echo -e " ${GREEN}✓${NC}"
        sleep 2
    done
else
    echo -e "  ${GREEN}✓ No running jobs${NC}"
fi

echo ""

###############################################################################
# Step 3: Upload JAR to Flink
###############################################################################

echo -e "${YELLOW}[Step 3/4] Uploading JAR to Flink...${NC}"

echo -n "  ⏳ Uploading $JAR_PATH..."
UPLOAD_RESPONSE=$(curl -s -X POST -F "jarfile=@${JAR_PATH}" "${FLINK_HOME}/jars/upload")
JAR_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.filename' | sed 's/.*\///')

if [ -z "$JAR_ID" ] || [ "$JAR_ID" == "null" ]; then
    echo -e " ${RED}✗ Upload failed${NC}"
    echo "Response: $UPLOAD_RESPONSE"
    exit 1
fi

echo -e " ${GREEN}✓ Uploaded${NC}"
echo "  📦 JAR ID: $JAR_ID"
echo ""

###############################################################################
# Step 4: Deploy All 8 Modules
###############################################################################

echo -e "${YELLOW}[Step 4/4] Deploying modules...${NC}"
echo ""

# Array of modules to deploy
declare -a MODULES=(
    "com.cardiofit.flink.operators.Module1_Ingestion:Module 1: Ingestion & Gateway"
    "com.cardiofit.flink.operators.Module2_Enhanced:Module 2: Enhanced Context Assembly"
    "com.cardiofit.flink.operators.Module3_ComprehensiveCDS:Module 3: Comprehensive CDS"
    "com.cardiofit.flink.operators.Module4_PatternDetection:Module 4: Pattern Detection (CEP)"
    "com.cardiofit.flink.operators.Module5_MLInference:Module 5: ML Inference Engine"
    "com.cardiofit.flink.operators.Module6_EgressRouting:Module 6: Egress Routing"
    "com.cardiofit.flink.operators.Module6_AlertComposition:Module 6: Alert Composition"
    "com.cardiofit.flink.analytics.Module6_AnalyticsEngine:Module 6: Analytics Engine"
)

DEPLOYED_JOBS=()
FAILED_JOBS=()

for i in "${!MODULES[@]}"; do
    MODULE_INFO="${MODULES[$i]}"
    CLASS_NAME="${MODULE_INFO%%:*}"
    MODULE_NAME="${MODULE_INFO#*:}"
    MODULE_NUM=$((i + 1))

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Deploying ${MODULE_NAME} (${MODULE_NUM}/8)${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    echo "  📋 Class: $CLASS_NAME"
    echo "  ⚙️  Parallelism: $PARALLELISM"
    echo ""

    # Deploy the job
    echo -n "  ⏳ Submitting job..."
    DEPLOY_RESPONSE=$(curl -s -X POST "${FLINK_HOME}/jars/${JAR_ID}/run" \
        -H "Content-Type: application/json" \
        -d "{
            \"entryClass\": \"${CLASS_NAME}\",
            \"parallelism\": ${PARALLELISM}
        }")

    JOB_ID=$(echo "$DEPLOY_RESPONSE" | jq -r '.jobid' 2>/dev/null)

    if [ -z "$JOB_ID" ] || [ "$JOB_ID" == "null" ]; then
        echo -e " ${RED}✗ Failed${NC}"
        echo "  Response: $DEPLOY_RESPONSE"
        FAILED_JOBS+=("${MODULE_NAME}")
    else
        echo -e " ${GREEN}✓ Running${NC}"
        echo "  🆔 Job ID: $JOB_ID"
        DEPLOYED_JOBS+=("${MODULE_NAME}:${JOB_ID}")

        # Wait before deploying next module
        if [ $MODULE_NUM -lt 8 ]; then
            echo "  ⏱️  Waiting 3 seconds before next deployment..."
            sleep 3
        fi
    fi

    echo ""
done

###############################################################################
# Deployment Summary
###############################################################################

echo ""
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                   DEPLOYMENT SUMMARY                          ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

if [ ${#DEPLOYED_JOBS[@]} -eq 8 ]; then
    echo -e "${GREEN}✅ SUCCESS: All 8 modules deployed!${NC}"
    echo ""
    echo "Deployed Jobs:"
    for job in "${DEPLOYED_JOBS[@]}"; do
        MODULE_NAME="${job%%:*}"
        JOB_ID="${job#*:}"
        echo "  ✓ ${MODULE_NAME}"
        echo "    Job ID: ${JOB_ID}"
    done
else
    echo -e "${YELLOW}⚠️  PARTIAL: ${#DEPLOYED_JOBS[@]}/8 modules deployed${NC}"
    echo ""
    if [ ${#DEPLOYED_JOBS[@]} -gt 0 ]; then
        echo "✅ Deployed:"
        for job in "${DEPLOYED_JOBS[@]}"; do
            echo "  ✓ ${job%%:*}"
        done
        echo ""
    fi
    if [ ${#FAILED_JOBS[@]} -gt 0 ]; then
        echo "❌ Failed:"
        for module in "${FAILED_JOBS[@]}"; do
            echo "  ✗ ${module}"
        done
    fi
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Next Steps:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "1. 🌐 View Flink Web UI:"
echo "   ${FLINK_HOME}"
echo ""
echo "2. 📊 Check job status:"
echo "   curl -s ${FLINK_HOME}/jobs/overview | jq"
echo ""
echo "3. 🧪 Test the pipeline:"
echo "   ./test-complete-pipeline.sh"
echo ""
echo "4. 📈 Monitor topics:"
echo "   docker exec kafka kafka-console-consumer \\"
echo "     --bootstrap-server localhost:9092 \\"
echo "     --topic prod.ehr.events.enriched \\"
echo "     --from-beginning --max-messages 10"
echo ""

exit 0
