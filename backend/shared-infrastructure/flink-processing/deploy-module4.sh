#!/bin/bash
# Module 4 Deployment Script
# Deploys flink-ehr-intelligence JAR to Flink cluster with Module 4 configuration

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Module 4: Clinical Pattern Engine${NC}"
echo -e "${BLUE}Deployment to Flink Cluster${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Configuration
FLINK_JOBMANAGER="http://localhost:8081"
JAR_PATH="target/flink-ehr-intelligence-1.0.0.jar"
ENV_FILE="flink-datastores.env"
DEPLOYMENT_MODE="${1:-ingestion-only}"  # Default to ingestion-only, can pass "full" or "module4-only"

# Step 1: Verify JAR exists
echo -e "${YELLOW}[1/6]${NC} Verifying JAR file..."
if [ ! -f "$JAR_PATH" ]; then
    echo -e "${RED}ERROR: JAR file not found at $JAR_PATH${NC}"
    echo -e "${YELLOW}Run 'mvn clean package -DskipTests' to build the JAR${NC}"
    exit 1
fi

JAR_SIZE=$(ls -lh "$JAR_PATH" | awk '{print $5}')
echo -e "${GREEN}✓${NC} JAR found: $JAR_PATH ($JAR_SIZE)"
echo ""

# Step 2: Load environment variables
echo -e "${YELLOW}[2/6]${NC} Loading environment variables..."
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${RED}ERROR: Environment file not found at $ENV_FILE${NC}"
    exit 1
fi

# Export environment variables
set -a
source "$ENV_FILE"
set +a

echo -e "${GREEN}✓${NC} Environment variables loaded from $ENV_FILE"
echo -e "  - Kafka Bootstrap: $KAFKA_BOOTSTRAP_SERVERS"
echo -e "  - Daily Risk Score Topic: $MODULE4_DAILY_RISK_SCORE_TOPIC"
echo -e "  - Clinical Patterns Topic: $MODULE4_CLINICAL_PATTERNS_TOPIC"
echo ""

# Step 3: Verify Flink is running
echo -e "${YELLOW}[3/6]${NC} Checking Flink JobManager connectivity..."
if ! curl -s "$FLINK_JOBMANAGER/overview" > /dev/null; then
    echo -e "${RED}ERROR: Cannot connect to Flink JobManager at $FLINK_JOBMANAGER${NC}"
    echo -e "${YELLOW}Is Flink running? Check: docker ps | grep flink${NC}"
    exit 1
fi

FLINK_VERSION=$(curl -s "$FLINK_JOBMANAGER/config" | grep -o '"flink-version":"[^"]*"' | cut -d'"' -f4)
echo -e "${GREEN}✓${NC} Flink JobManager accessible (version: $FLINK_VERSION)"
echo ""

# Step 4: Verify Kafka topics exist
echo -e "${YELLOW}[4/6]${NC} Verifying Kafka topics..."

check_topic() {
    local topic=$1
    if docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null | grep -q "^${topic}$"; then
        echo -e "${GREEN}  ✓${NC} Topic exists: $topic"
        return 0
    else
        echo -e "${RED}  ✗${NC} Topic missing: $topic"
        return 1
    fi
}

TOPICS_OK=true
check_topic "$MODULE4_DAILY_RISK_SCORE_TOPIC" || TOPICS_OK=false
check_topic "$MODULE4_CLINICAL_PATTERNS_TOPIC" || TOPICS_OK=false
check_topic "patient-events-v1" || TOPICS_OK=false
check_topic "enriched-patient-events-v1" || TOPICS_OK=false

if [ "$TOPICS_OK" = false ]; then
    echo -e "${RED}ERROR: Required Kafka topics are missing${NC}"
    echo -e "${YELLOW}Run topic creation scripts before deploying${NC}"
    exit 1
fi
echo ""

# Step 5: Upload JAR to Flink
echo -e "${YELLOW}[5/6]${NC} Uploading JAR to Flink cluster..."

# Upload JAR
UPLOAD_RESPONSE=$(curl -s -X POST -F "jarfile=@$JAR_PATH" "$FLINK_JOBMANAGER/jars/upload")
JAR_FILENAME=$(echo "$UPLOAD_RESPONSE" | grep -o '"filename":"[^"]*"' | cut -d'"' -f4 | sed 's/\\/\\\\/g')

# Extract just the filename (not full path)
JAR_ID=$(basename "$JAR_FILENAME")

if [ -z "$JAR_ID" ]; then
    echo -e "${RED}ERROR: Failed to upload JAR${NC}"
    echo -e "${YELLOW}Response: $UPLOAD_RESPONSE${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} JAR uploaded successfully"
echo -e "  JAR ID: $JAR_ID"
echo ""

# Step 6: Start Flink job
echo -e "${YELLOW}[6/6]${NC} Starting Flink job..."

# Prepare job configuration
JOB_ENTRY_CLASS="com.cardiofit.flink.operators.Module4_PatternDetection"
JOB_PARALLELISM=2
PROGRAM_ARGS="$DEPLOYMENT_MODE development"

# Submit job
SUBMIT_RESPONSE=$(curl -s -X POST "$FLINK_JOBMANAGER/jars/$JAR_ID/run" \
    -H "Content-Type: application/json" \
    -d "{
        \"entryClass\": \"$JOB_ENTRY_CLASS\",
        \"parallelism\": $JOB_PARALLELISM,
        \"programArgs\": \"$PROGRAM_ARGS\",
        \"savepointPath\": null,
        \"allowNonRestoredState\": false
    }")

JOB_ID=$(echo "$SUBMIT_RESPONSE" | grep -o '"jobid":"[^"]*"' | cut -d'"' -f4)

if [ -z "$JOB_ID" ]; then
    echo -e "${RED}ERROR: Failed to start Flink job${NC}"
    echo -e "${YELLOW}Response: $SUBMIT_RESPONSE${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Flink job started successfully"
echo -e "  Job ID: $JOB_ID"
echo ""

# Wait a moment for job to initialize
echo -e "${BLUE}Waiting for job initialization (5 seconds)...${NC}"
sleep 5

# Check job status
JOB_STATUS=$(curl -s "$FLINK_JOBMANAGER/jobs/$JOB_ID" | grep -o '"state":"[^"]*"' | cut -d'"' -f4)

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ DEPLOYMENT SUCCESSFUL${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${BLUE}Job Details:${NC}"
echo -e "  Job ID: $JOB_ID"
echo -e "  Status: $JOB_STATUS"
echo -e "  Entry Class: $JOB_ENTRY_CLASS"
echo -e "  Parallelism: $JOB_PARALLELISM"
echo -e "  Mode: $DEPLOYMENT_MODE"
echo ""
echo -e "${BLUE}Monitoring:${NC}"
echo -e "  Flink Web UI: ${GREEN}http://localhost:8081/#/job/$JOB_ID/overview${NC}"
echo -e "  Kafka UI: ${GREEN}http://localhost:8080${NC}"
echo -e "  Daily Risk Scores Topic: ${GREEN}daily-risk-scores.v1${NC}"
echo ""
echo -e "${BLUE}Kafka Topics to Monitor:${NC}"
echo -e "  Input:  enriched-patient-events-v1"
echo -e "  Output: clinical-patterns.v1"
echo -e "  Output: daily-risk-scores.v1 (24-hour windows)"
echo ""
echo -e "${YELLOW}⏳ Note: Daily risk scores will appear after first 24-hour window completes${NC}"
echo -e "${YELLOW}📊 Monitor Flink metrics to see event processing and pattern detection${NC}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo -e "  1. Send test patient events to patient-events-v1"
echo -e "  2. Monitor enriched-patient-events-v1 for processed events"
echo -e "  3. Check clinical-patterns.v1 for detected patterns"
echo -e "  4. Wait 24 hours for daily-risk-scores.v1 to populate"
echo ""
