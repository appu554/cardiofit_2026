#!/bin/bash

# Flink-Kafka Pipeline Diagnostic Script
# Performs comprehensive health checks on Flink and Kafka integration

set -e

FLINK_DIR="/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing"
LOG_DIR="$FLINK_DIR/logs"

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║     Flink Kafka Pipeline Diagnostics - CardioFit Platform    ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
    fi
}

print_section() {
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  $1"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# ============================================================================
# Section 1: Docker Containers Status
# ============================================================================
print_section "1. Docker Containers Status"

echo "Flink Containers:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null | grep flink || echo "No Flink containers found"
print_status $? "Flink containers check"

echo
echo "Kafka Containers:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null | grep kafka || echo "No Kafka containers found"
print_status $? "Kafka containers check"

# ============================================================================
# Section 2: Flink Cluster Health
# ============================================================================
print_section "2. Flink Cluster Health"

# Check Flink Web UI accessibility
echo "Testing Flink Web UI (http://localhost:8081)..."
if curl -s -f http://localhost:8081/overview >/dev/null 2>&1; then
    print_status 0 "Flink Web UI is accessible"

    # Get cluster overview
    echo
    echo "Cluster Overview:"
    curl -s http://localhost:8081/overview | python3 -m json.tool 2>/dev/null || \
        echo "Could not parse cluster overview JSON"
else
    print_status 1 "Flink Web UI is NOT accessible"
    echo "   → Check if JobManager is running: docker ps | grep jobmanager"
    echo "   → Check logs: docker logs cardiofit-flink-jobmanager"
fi

# Check TaskManagers
echo
echo "TaskManager Status:"
TASKMANAGERS=$(curl -s http://localhost:8081/taskmanagers 2>/dev/null | \
    python3 -c "import sys,json; data=json.load(sys.stdin); print(len(data.get('taskmanagers', [])))" 2>/dev/null)

if [ -n "$TASKMANAGERS" ] && [ "$TASKMANAGERS" -gt 0 ]; then
    print_status 0 "Found $TASKMANAGERS TaskManager(s)"

    # Get available slots
    SLOTS=$(curl -s http://localhost:8081/overview 2>/dev/null | \
        python3 -c "import sys,json; data=json.load(sys.stdin); print(data.get('slots-available', 'N/A'))" 2>/dev/null)
    echo "   → Available task slots: $SLOTS"
else
    print_status 1 "No TaskManagers found"
    echo "   → TaskManagers should auto-register with JobManager"
    echo "   → Check TaskManager logs for connection errors"
fi

# ============================================================================
# Section 3: Running Jobs
# ============================================================================
print_section "3. Flink Jobs Status"

if docker exec cardiofit-flink-jobmanager test -f /opt/flink/bin/flink 2>/dev/null; then
    echo "Running Jobs:"
    JOBS=$(docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r 2>&1)
    echo "$JOBS"

    if echo "$JOBS" | grep -q "No running jobs"; then
        print_status 1 "No jobs are currently running"
        echo "   → Job may have failed to start or crashed"
    else
        print_status 0 "Jobs are running"
    fi
else
    print_status 1 "Cannot access Flink CLI"
fi

# Check for failed jobs
echo
echo "Failed/Canceled Jobs (last 5):"
curl -s http://localhost:8081/jobs/overview 2>/dev/null | \
    python3 -c "import sys,json; data=json.load(sys.stdin);
jobs=[j for j in data.get('jobs', []) if j.get('state') in ['FAILED', 'CANCELED']];
print('\n'.join([f\"  {j.get('jid')}: {j.get('name')} - {j.get('state')}\" for j in jobs[:5]])) if jobs else print('  No failed jobs')" 2>/dev/null || \
    echo "  Could not retrieve job history"

# ============================================================================
# Section 4: Kafka Connectivity
# ============================================================================
print_section "4. Kafka Connectivity from Flink"

# Test basic Kafka connectivity with multiple possible addresses
KAFKA_HOSTS=("kafka:9092" "kafka1:29092" "kafka2:29093" "kafka3:29094" "localhost:9092")

echo "Testing Kafka connectivity from JobManager container..."
for host in "${KAFKA_HOSTS[@]}"; do
    HOST_NAME=$(echo $host | cut -d: -f1)
    HOST_PORT=$(echo $host | cut -d: -f2)

    if docker exec cardiofit-flink-jobmanager bash -c "timeout 2 nc -zv $HOST_NAME $HOST_PORT" 2>&1 | grep -q "succeeded"; then
        print_status 0 "Kafka reachable at $host"
    else
        print_status 1 "Kafka NOT reachable at $host"
    fi
done

# Check environment variable
echo
echo "Kafka Bootstrap Servers Configuration:"
if [ -f "$FLINK_DIR/flink-datastores.env" ]; then
    KAFKA_CONFIG=$(grep "^KAFKA_BOOTSTRAP_SERVERS" "$FLINK_DIR/flink-datastores.env" | head -1)
    echo "   → $KAFKA_CONFIG"
else
    echo "   → flink-datastores.env not found"
fi

# ============================================================================
# Section 5: Docker Networks
# ============================================================================
print_section "5. Docker Network Configuration"

echo "Available Networks:"
docker network ls 2>/dev/null | grep -E "cardiofit|kafka" || echo "No CardioFit/Kafka networks found"

echo
echo "Flink JobManager Network Connections:"
docker inspect cardiofit-flink-jobmanager 2>/dev/null | \
    python3 -c "import sys,json; data=json.load(sys.stdin);
networks=data[0].get('NetworkSettings', {}).get('Networks', {});
print('\n'.join([f'  - {name}' for name in networks.keys()]))" 2>/dev/null || \
    echo "Could not inspect JobManager networks"

echo
echo "Network Isolation Check:"
if docker network inspect kafka_cardiofit-network >/dev/null 2>&1; then
    print_status 0 "kafka_cardiofit-network exists"

    # Check if Flink is connected
    if docker network inspect kafka_cardiofit-network | grep -q "cardiofit-flink-jobmanager"; then
        print_status 0 "Flink JobManager is connected to kafka_cardiofit-network"
    else
        print_status 1 "Flink JobManager is NOT connected to kafka_cardiofit-network"
        echo "   → This may prevent Flink from reaching Kafka"
    fi
else
    print_status 1 "kafka_cardiofit-network does not exist"
fi

if docker network inspect cardiofit-network >/dev/null 2>&1; then
    print_status 0 "cardiofit-network exists"
else
    print_status 1 "cardiofit-network does not exist"
fi

# ============================================================================
# Section 6: JAR File Analysis
# ============================================================================
print_section "6. JAR File Analysis"

JAR_PATH="$FLINK_DIR/target/flink-ehr-intelligence-1.0.0.jar"

if [ -f "$JAR_PATH" ]; then
    print_status 0 "JAR file exists at $JAR_PATH"

    JAR_SIZE=$(du -h "$JAR_PATH" | cut -f1)
    echo "   → Size: $JAR_SIZE"

    # Check if JAR is accessible from container
    if docker exec cardiofit-flink-jobmanager test -f /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar 2>/dev/null; then
        print_status 0 "JAR is mounted in JobManager container at /opt/flink/usrlib/"
    else
        print_status 1 "JAR is NOT accessible from JobManager container"
        echo "   → Check volume mount in docker-compose.yml"
    fi

    # Check manifest
    echo
    echo "JAR Manifest Main-Class:"
    unzip -p "$JAR_PATH" META-INF/MANIFEST.MF 2>/dev/null | grep "Main-Class" || echo "   → Could not read manifest"

    # Check for Kafka connector
    echo
    echo "Kafka Connector Check:"
    if jar tf "$JAR_PATH" 2>/dev/null | grep -q "org/apache/flink/connector/kafka"; then
        print_status 0 "Kafka connector classes found in JAR"
    else
        print_status 1 "Kafka connector classes NOT found in JAR"
        echo "   → May need to rebuild with proper shading"
        echo "   → Or add flink-connector-kafka to Flink lib/"
    fi
else
    print_status 1 "JAR file not found at $JAR_PATH"
    echo "   → Run: mvn clean package"
fi

# ============================================================================
# Section 7: Kafka Topics
# ============================================================================
print_section "7. Kafka Topics Status"

# Find Kafka container name
KAFKA_CONTAINER=$(docker ps --format "{{.Names}}" 2>/dev/null | grep -E "kafka[^-]|kafka$" | head -1)

if [ -n "$KAFKA_CONTAINER" ]; then
    echo "Using Kafka container: $KAFKA_CONTAINER"
    echo
    echo "Required Topics (from PatientEventEnrichmentJob.java):"
    REQUIRED_TOPICS=("patient-events.v1" "medication-events.v1" "observation-events.v1"
                     "vital-signs-events.v1" "lab-result-events.v1" "safety-events.v1")

    # List existing topics
    EXISTING_TOPICS=$(docker exec "$KAFKA_CONTAINER" kafka-topics --list --bootstrap-server localhost:9092 2>/dev/null || echo "")

    for topic in "${REQUIRED_TOPICS[@]}"; do
        if echo "$EXISTING_TOPICS" | grep -q "^$topic$"; then
            print_status 0 "Topic exists: $topic"
        else
            print_status 1 "Topic MISSING: $topic"
        fi
    done

    echo
    echo "All Existing Topics:"
    echo "$EXISTING_TOPICS" | head -20
else
    print_status 1 "Could not find Kafka container"
    echo "   → Manual check: docker ps | grep kafka"
fi

# ============================================================================
# Section 8: Recent Errors Analysis
# ============================================================================
print_section "8. Recent Error Analysis"

if [ -f "$LOG_DIR/flink--standalonesession-0-jobmanager.log" ]; then
    echo "Recent JobManager Errors (excluding 'Job not found'):"
    tail -200 "$LOG_DIR/flink--standalonesession-0-jobmanager.log" | \
        grep -i "ERROR\|Exception" | \
        grep -v "Job.*not found" | \
        tail -10 | \
        sed 's/^/   /' || echo "   No recent errors found"
else
    echo "JobManager log not found at: $LOG_DIR/flink--standalonesession-0-jobmanager.log"
fi

echo
if [ -f "$LOG_DIR/flink--taskexecutor-0-taskmanager-1.log" ]; then
    echo "Recent TaskManager Errors:"
    tail -200 "$LOG_DIR/flink--taskexecutor-0-taskmanager-1.log" | \
        grep -i "ERROR\|Exception" | \
        tail -5 | \
        sed 's/^/   /' || echo "   No recent errors found"
else
    echo "TaskManager log not found"
fi

# ============================================================================
# Section 9: System Resources
# ============================================================================
print_section "9. System Resources"

echo "Flink Container Resource Usage:"
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null | \
    grep -E "NAME|flink" || echo "Could not get stats"

# ============================================================================
# Summary & Recommendations
# ============================================================================
print_section "10. Diagnostic Summary & Recommendations"

echo "Based on the diagnostic results, here are the likely issues:"
echo

# Analyze results and provide recommendations
ISSUES_FOUND=0

# Check 1: Flink Web UI
if ! curl -s -f http://localhost:8081/overview >/dev/null 2>&1; then
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
    echo "${YELLOW}⚠${NC} ISSUE $ISSUES_FOUND: Flink Web UI is not accessible"
    echo "   FIX: Check if JobManager is running and healthy"
    echo "        → docker logs cardiofit-flink-jobmanager"
    echo "        → docker restart cardiofit-flink-jobmanager"
    echo
fi

# Check 2: TaskManagers
if [ -z "$TASKMANAGERS" ] || [ "$TASKMANAGERS" -eq 0 ]; then
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
    echo "${YELLOW}⚠${NC} ISSUE $ISSUES_FOUND: No TaskManagers registered"
    echo "   FIX: TaskManagers should auto-connect to JobManager"
    echo "        → Check network connectivity between containers"
    echo "        → Verify JOB_MANAGER_RPC_ADDRESS=jobmanager in docker-compose.yml"
    echo
fi

# Check 3: Kafka connectivity
KAFKA_REACHABLE=false
for host in "${KAFKA_HOSTS[@]}"; do
    HOST_NAME=$(echo $host | cut -d: -f1)
    HOST_PORT=$(echo $host | cut -d: -f2)
    if docker exec cardiofit-flink-jobmanager bash -c "timeout 2 nc -zv $HOST_NAME $HOST_PORT" 2>&1 | grep -q "succeeded"; then
        KAFKA_REACHABLE=true
        break
    fi
done

if [ "$KAFKA_REACHABLE" = false ]; then
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
    echo "${YELLOW}⚠${NC} ISSUE $ISSUES_FOUND: Kafka is not reachable from Flink"
    echo "   FIX: Update KAFKA_BOOTSTRAP_SERVERS in flink-datastores.env"
    echo "        → Ensure both use the same Docker network"
    echo "        → Check: docker network inspect kafka_cardiofit-network"
    echo
fi

# Check 4: No running jobs
if [ -n "$JOBS" ] && echo "$JOBS" | grep -q "No running jobs"; then
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
    echo "${YELLOW}⚠${NC} ISSUE $ISSUES_FOUND: No Flink jobs are currently running"
    echo "   FIX: Job may have failed to start - check for exceptions"
    echo "        → Review full JobManager logs"
    echo "        → Test manual submission: docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run ..."
    echo
fi

if [ $ISSUES_FOUND -eq 0 ]; then
    echo "${GREEN}✓${NC} No critical issues detected!"
    echo "   If your job is still failing, check:"
    echo "   → Application-specific exceptions in JobManager logs"
    echo "   → Job-specific configuration issues"
else
    echo "Found $ISSUES_FOUND potential issue(s) that need attention."
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  For detailed fixes, see: TROUBLESHOOTING_GUIDE.md"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

# Save diagnostics to file
DIAG_FILE="$FLINK_DIR/diagnostics-$(date +%Y%m%d-%H%M%S).log"
echo "Full diagnostic output saved to: $DIAG_FILE"