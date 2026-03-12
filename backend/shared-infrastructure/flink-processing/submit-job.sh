#!/bin/bash

# Flink Job Submission Script
# Submits the CardioFit EHR Intelligence job to the Flink cluster

set -e

FLINK_DIR="/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing"
JAR_PATH="/opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar"
MAIN_CLASS="com.cardiofit.flink.FlinkJobOrchestrator"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_usage() {
    echo "Usage: $0 [JOB_TYPE] [ENVIRONMENT]"
    echo
    echo "Job Types:"
    echo "  full-pipeline      - Complete 6-module EHR intelligence pipeline (default)"
    echo "  ingestion-only     - Module 1: Ingestion & Gateway only"
    echo "  context-assembly   - Module 2: Context Assembly only"
    echo "  semantic-mesh      - Module 3: Semantic Mesh only"
    echo "  pattern-detection  - Module 4: Pattern Detection only"
    echo "  ml-inference       - Module 5: ML Inference only"
    echo "  egress-routing     - Module 6: Egress Routing only"
    echo
    echo "Environments:"
    echo "  development        - Dev mode with reduced parallelism (default)"
    echo "  production         - Prod mode with full parallelism"
    echo
    echo "Examples:"
    echo "  $0                                    # Full pipeline in dev mode"
    echo "  $0 full-pipeline production           # Full pipeline in prod mode"
    echo "  $0 ingestion-only development         # Test ingestion only"
}

# Parse arguments
JOB_TYPE="${1:-full-pipeline}"
ENVIRONMENT="${2:-development}"

if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    print_usage
    exit 0
fi

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         Flink Job Submission - CardioFit Platform            ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo
echo "Job Type:    $JOB_TYPE"
echo "Environment: $ENVIRONMENT"
echo

# ============================================================================
# Pre-submission Checks
# ============================================================================
echo -e "${BLUE}[Step 1/4]${NC} Running pre-submission checks..."

# Check if Flink cluster is running
if ! curl -s -f http://localhost:8081/overview >/dev/null 2>&1; then
    echo -e "${RED}✗${NC} Flink cluster is not accessible"
    echo "Please start the cluster first:"
    echo "  cd $FLINK_DIR && docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✓${NC} Flink cluster is running"

# Check TaskManagers
TASKMANAGERS=$(curl -s http://localhost:8081/taskmanagers 2>/dev/null | \
    python3 -c "import sys,json; data=json.load(sys.stdin); print(len(data.get('taskmanagers', [])))" 2>/dev/null)

if [ -z "$TASKMANAGERS" ] || [ "$TASKMANAGERS" -eq 0 ]; then
    echo -e "${YELLOW}⚠${NC} Warning: No TaskManagers registered"
    echo "Jobs require TaskManagers to execute. Continue anyway? (y/N)"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo -e "${GREEN}✓${NC} TaskManagers available: $TASKMANAGERS"
fi

# Check if JAR exists in container
if ! docker exec cardiofit-flink-jobmanager test -f "$JAR_PATH" 2>/dev/null; then
    echo -e "${RED}✗${NC} JAR not found in container at: $JAR_PATH"
    echo "Please ensure the JAR is built and mounted:"
    echo "  cd $FLINK_DIR && mvn clean package"
    exit 1
fi
echo -e "${GREEN}✓${NC} JAR file is accessible in container"

# Check Kafka connectivity
echo
echo "Testing Kafka connectivity..."
KAFKA_REACHABLE=false
KAFKA_HOSTS=("kafka:9092" "kafka1:29092" "localhost:9092")

for host in "${KAFKA_HOSTS[@]}"; do
    HOST_NAME=$(echo $host | cut -d: -f1)
    HOST_PORT=$(echo $host | cut -d: -f2)

    if docker exec cardiofit-flink-jobmanager timeout 2 nc -zv $HOST_NAME $HOST_PORT 2>&1 | grep -q "succeeded"; then
        echo -e "${GREEN}✓${NC} Kafka reachable at: $host"
        KAFKA_REACHABLE=true
        break
    fi
done

if [ "$KAFKA_REACHABLE" = false ]; then
    echo -e "${RED}✗${NC} Warning: Kafka not reachable from Flink"
    echo "Job may fail to start. Continue anyway? (y/N)"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# ============================================================================
# Cancel Existing Jobs (Optional)
# ============================================================================
echo
echo -e "${BLUE}[Step 2/4]${NC} Checking for existing jobs..."

RUNNING_JOBS=$(docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r 2>&1)

if echo "$RUNNING_JOBS" | grep -qv "No running jobs"; then
    echo "Found running jobs:"
    echo "$RUNNING_JOBS" | grep -E "^[0-9a-f]" | sed 's/^/  /'
    echo
    echo "Cancel existing jobs before submitting new one? (y/N)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo "Canceling jobs..."
        JOB_IDS=$(echo "$RUNNING_JOBS" | grep -oE "^[0-9a-f]{32}" || true)
        for job_id in $JOB_IDS; do
            docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink cancel "$job_id" && \
                echo -e "${GREEN}✓${NC} Canceled job: $job_id" || \
                echo -e "${RED}✗${NC} Failed to cancel job: $job_id"
        done
        echo "Waiting 5 seconds for cleanup..."
        sleep 5
    fi
else
    echo -e "${GREEN}✓${NC} No running jobs"
fi

# ============================================================================
# Submit Job
# ============================================================================
echo
echo -e "${BLUE}[Step 3/4]${NC} Submitting job..."

echo "Executing command:"
echo "  flink run --detached --class $MAIN_CLASS $JAR_PATH $JOB_TYPE $ENVIRONMENT"
echo

SUBMIT_OUTPUT=$(docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
    --detached \
    --class "$MAIN_CLASS" \
    "$JAR_PATH" \
    "$JOB_TYPE" "$ENVIRONMENT" 2>&1)

if echo "$SUBMIT_OUTPUT" | grep -q "Job has been submitted with JobID"; then
    JOB_ID=$(echo "$SUBMIT_OUTPUT" | grep -oE "JobID [0-9a-f]{32}" | cut -d' ' -f2)
    echo -e "${GREEN}✓${NC} Job submitted successfully!"
    echo "Job ID: $JOB_ID"
else
    echo -e "${RED}✗${NC} Job submission failed!"
    echo "$SUBMIT_OUTPUT"
    exit 1
fi

# ============================================================================
# Monitor Job Startup
# ============================================================================
echo
echo -e "${BLUE}[Step 4/4]${NC} Monitoring job startup..."

echo "Waiting for job to initialize (10 seconds)..."
sleep 10

# Check job status
JOB_STATUS=$(curl -s "http://localhost:8081/jobs/$JOB_ID" 2>/dev/null | \
    python3 -c "import sys,json; data=json.load(sys.stdin); print(data.get('state', 'UNKNOWN'))" 2>/dev/null)

case "$JOB_STATUS" in
    RUNNING)
        echo -e "${GREEN}✓${NC} Job is RUNNING"
        ;;
    CREATED|SCHEDULED|DEPLOYING)
        echo -e "${YELLOW}⚠${NC} Job is starting: $JOB_STATUS"
        echo "Check status in a few moments"
        ;;
    FAILED|CANCELED)
        echo -e "${RED}✗${NC} Job has $JOB_STATUS"
        echo "Check logs for details"
        ;;
    *)
        echo -e "${YELLOW}⚠${NC} Job status: $JOB_STATUS"
        ;;
esac

# Get job metrics
echo
echo "Job Overview:"
curl -s "http://localhost:8081/jobs/$JOB_ID" 2>/dev/null | \
    python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(f\"  Name: {data.get('name', 'N/A')}\")
    print(f\"  Start Time: {data.get('start-time', 'N/A')}\")
    print(f\"  State: {data.get('state', 'N/A')}\")
    vertices = data.get('vertices', [])
    print(f\"  Vertices: {len(vertices)}\")
except:
    print('  Could not retrieve job details')
" 2>/dev/null || echo "  Could not retrieve job overview"

# ============================================================================
# Success Summary
# ============================================================================
echo
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                     Submission Complete                       ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo

echo "Job Details:"
echo "  Job ID:      $JOB_ID"
echo "  Job Type:    $JOB_TYPE"
echo "  Environment: $ENVIRONMENT"
echo

echo "Monitoring Commands:"
echo "  Flink Web UI:   http://localhost:8081/#/job/$JOB_ID/overview"
echo "  Job Status:     docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list"
echo "  Job Logs:       docker logs -f cardiofit-flink-jobmanager"
echo "  Cancel Job:     docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink cancel $JOB_ID"
echo

echo "Test Event Commands:"
echo "  # Send a test patient event to Kafka"
echo '  docker exec <kafka-container> kafka-console-producer --broker-list localhost:9092 --topic patient-events.v1 << EOF'
echo '  {"patientId":"test-123","eventType":"vital-signs","timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","data":{"heartRate":75},"critical":false,"clinicalEvent":true}'
echo '  EOF'
echo

if [ "$JOB_STATUS" = "RUNNING" ]; then
    echo -e "${GREEN}✓${NC} Job is running successfully!"
else
    echo -e "${YELLOW}⚠${NC} Job may still be initializing. Check Web UI for current status."
    echo "If job fails, run diagnostics: ./diagnose-flink-kafka.sh"
fi
echo