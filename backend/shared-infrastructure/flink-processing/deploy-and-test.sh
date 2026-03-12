#!/bin/bash

# Deploy and Test Module 1 & 2 with FHIR/Neo4j Enrichment
# This script deploys the updated modules and validates the enrichment

set -e  # Exit on error

echo "=================================================="
echo "Deploying Module 1 & 2 with FHIR/Neo4j Enrichment"
echo "=================================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
JAR_PATH="/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/target/flink-ehr-intelligence-1.0.0.jar"
FLINK_URL="http://localhost:8081"

# Step 1: Verify JAR exists
echo -e "${BLUE}1. Checking JAR file...${NC}"
if [ ! -f "$JAR_PATH" ]; then
    echo -e "${RED}❌ JAR not found at $JAR_PATH${NC}"
    echo "Please run: mvn clean package -DskipTests"
    exit 1
fi
echo -e "${GREEN}✅ JAR found${NC}"

# Step 2: Check Flink status
echo ""
echo -e "${BLUE}2. Checking Flink cluster status...${NC}"
FLINK_STATUS=$(curl -s $FLINK_URL/overview | python3 -c "import sys, json; data = json.load(sys.stdin); print('RUNNING' if data.get('flink-version') else 'DOWN')" 2>/dev/null || echo "DOWN")
if [ "$FLINK_STATUS" != "RUNNING" ]; then
    echo -e "${RED}❌ Flink is not running at $FLINK_URL${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Flink cluster is running${NC}"

# Step 3: Cancel existing jobs (optional cleanup)
echo ""
echo -e "${BLUE}3. Checking for existing Module 1 & 2 jobs...${NC}"
RUNNING_JOBS=$(curl -s $FLINK_URL/jobs | python3 -c "
import sys, json
data = json.load(sys.stdin)
for job in data.get('jobs', []):
    if job['status'] == 'RUNNING':
        if 'Module 1' in job.get('name', '') or 'Module 2' in job.get('name', ''):
            print(f\"{job['id']}:{job.get('name', 'Unknown')}\")
" 2>/dev/null)

if [ ! -z "$RUNNING_JOBS" ]; then
    echo -e "${YELLOW}Found running Module jobs:${NC}"
    echo "$RUNNING_JOBS"
    echo -e "${YELLOW}Cancelling existing jobs...${NC}"

    echo "$RUNNING_JOBS" | while IFS=':' read -r job_id job_name; do
        echo "  Cancelling: $job_name"
        curl -s -X PATCH "$FLINK_URL/jobs/$job_id?mode=cancel" > /dev/null
    done

    echo "Waiting for jobs to stop..."
    sleep 5
fi

# Step 4: Upload JAR to Flink
echo ""
echo -e "${BLUE}4. Uploading JAR to Flink...${NC}"
UPLOAD_RESPONSE=$(curl -s -X POST \
    -H "Expect:" \
    -F "jarfile=@$JAR_PATH" \
    "$FLINK_URL/jars/upload")

JAR_ID=$(echo $UPLOAD_RESPONSE | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'filename' in data:
    # Extract just the filename part after the last /
    filename = data['filename'].split('/')[-1]
    print(filename)
else:
    print('ERROR')
" 2>/dev/null)

if [ "$JAR_ID" == "ERROR" ]; then
    echo -e "${RED}❌ Failed to upload JAR${NC}"
    echo "Response: $UPLOAD_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✅ JAR uploaded successfully: $JAR_ID${NC}"

# Step 5: Deploy Module 1 (Ingestion)
echo ""
echo -e "${BLUE}5. Deploying Module 1: Ingestion & Validation...${NC}"

MODULE1_RESPONSE=$(curl -s -X POST \
    "$FLINK_URL/jars/$JAR_ID/run" \
    -H "Content-Type: application/json" \
    -d '{
        "entryClass": "com.cardiofit.flink.operators.Module1_Ingestion",
        "parallelism": 2,
        "programArgs": "",
        "savepointPath": null,
        "allowNonRestoredState": false
    }')

MODULE1_JOB_ID=$(echo $MODULE1_RESPONSE | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data.get('jobid', 'ERROR'))
" 2>/dev/null)

if [ "$MODULE1_JOB_ID" == "ERROR" ]; then
    echo -e "${RED}❌ Failed to deploy Module 1${NC}"
    echo "Response: $MODULE1_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✅ Module 1 deployed: Job ID $MODULE1_JOB_ID${NC}"

# Step 6: Deploy Module 2 (Context Assembly with FHIR/Neo4j)
echo ""
echo -e "${BLUE}6. Deploying Module 2: Context Assembly & Enrichment...${NC}"

MODULE2_RESPONSE=$(curl -s -X POST \
    "$FLINK_URL/jars/$JAR_ID/run" \
    -H "Content-Type: application/json" \
    -d '{
        "entryClass": "com.cardiofit.flink.operators.Module2_ContextAssembly",
        "parallelism": 2,
        "programArgs": "",
        "savepointPath": null,
        "allowNonRestoredState": false
    }')

MODULE2_JOB_ID=$(echo $MODULE2_RESPONSE | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data.get('jobid', 'ERROR'))
" 2>/dev/null)

if [ "$MODULE2_JOB_ID" == "ERROR" ]; then
    echo -e "${RED}❌ Failed to deploy Module 2${NC}"
    echo "Response: $MODULE2_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✅ Module 2 deployed: Job ID $MODULE2_JOB_ID${NC}"

# Step 7: Verify jobs are running
echo ""
echo -e "${BLUE}7. Verifying job status...${NC}"
sleep 3

MODULE1_STATUS=$(curl -s "$FLINK_URL/jobs/$MODULE1_JOB_ID" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data.get('state', 'UNKNOWN'))
" 2>/dev/null)

MODULE2_STATUS=$(curl -s "$FLINK_URL/jobs/$MODULE2_JOB_ID" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data.get('state', 'UNKNOWN'))
" 2>/dev/null)

echo -e "Module 1: $MODULE1_STATUS"
echo -e "Module 2: $MODULE2_STATUS"

if [ "$MODULE1_STATUS" != "RUNNING" ] || [ "$MODULE2_STATUS" != "RUNNING" ]; then
    echo -e "${RED}❌ Jobs are not running properly${NC}"
    echo "Check Flink UI at $FLINK_URL for details"
    exit 1
fi

echo -e "${GREEN}✅ Both modules are running${NC}"

# Step 8: Send test event
echo ""
echo -e "${BLUE}8. Sending test event with patient data...${NC}"

TEST_EVENT='{
  "id": "test-enrichment-'$(date +%s)'",
  "patient_id": "PAT-ROHAN-001",
  "encounter_id": "ENC-001",
  "event_type": "VITAL_SIGN",
  "event_time": '$(($(date +%s) * 1000))',
  "source_system": "test-deployment",
  "payload": {
    "heart_rate": 72,
    "blood_pressure": "120/80",
    "temperature": 37.0,
    "oxygen_saturation": 98,
    "respiratory_rate": 16
  }
}'

# Send to raw events topic (Module 1 input)
echo "$TEST_EVENT" | docker exec -i kafka kafka-console-producer \
    --broker-list localhost:9092 \
    --topic patient-events-v1

echo -e "${GREEN}✅ Test event sent to patient-events-v1${NC}"

# Step 9: Monitor enriched output
echo ""
echo -e "${BLUE}9. Waiting for enriched output...${NC}"
sleep 5

echo ""
echo -e "${YELLOW}Checking enriched output from clinical-patterns-v1 topic:${NC}"
echo "================================================================"

ENRICHED_OUTPUT=$(timeout 5 docker exec kafka kafka-console-consumer \
    --bootstrap-server localhost:9092 \
    --topic clinical-patterns-v1 \
    --from-beginning \
    --max-messages 1 \
    2>/dev/null | tail -1)

if [ -z "$ENRICHED_OUTPUT" ]; then
    echo -e "${RED}❌ No enriched output found${NC}"
    echo "Pipeline may still be processing or there may be an issue."
else
    echo "$ENRICHED_OUTPUT" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(json.dumps(data, indent=2))

    # Check for enrichment_data
    if 'enrichment_data' in data and data['enrichment_data']:
        print('\n${GREEN}✅ ENRICHMENT DATA FOUND:${NC}')
        ed = data['enrichment_data']

        if 'fhir_demographics' in ed:
            print('  ✅ FHIR demographics present')
        if 'fhir_medications' in ed:
            print('  ✅ FHIR medications present')
        if 'fhir_conditions' in ed:
            print('  ✅ FHIR conditions present')
        if 'neo4j_care_team' in ed:
            print('  ✅ Neo4j care team present')
        if 'neo4j_risk_cohorts' in ed:
            print('  ✅ Neo4j risk cohorts present')
        if 'latest_vitals' in ed:
            print('  ✅ Latest vitals present')

        print(f'\n  Total enrichment fields: {len(ed)}')
    else:
        print('\n${RED}❌ ENRICHMENT DATA MISSING OR EMPTY${NC}')

except:
    print('Raw output:', sys.stdin.read())
    " 2>/dev/null || echo "$ENRICHED_OUTPUT"
fi

echo ""
echo "================================================================"
echo -e "${GREEN}Deployment Complete!${NC}"
echo ""
echo "📊 Monitoring Options:"
echo "  - Flink UI: http://localhost:8081"
echo "  - Kafka UI: http://localhost:8080"
echo ""
echo "To check more events:"
echo "  docker exec kafka kafka-console-consumer \\"
echo "    --bootstrap-server localhost:9092 \\"
echo "    --topic clinical-patterns-v1 \\"
echo "    --from-beginning --max-messages 5 | python3 -m json.tool"
echo ""
echo "Job IDs for reference:"
echo "  Module 1: $MODULE1_JOB_ID"
echo "  Module 2: $MODULE2_JOB_ID"