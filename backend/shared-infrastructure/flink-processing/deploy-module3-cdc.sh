#!/bin/bash

###############################################################################
# Deploy Module 3 with CDC BroadcastStream
# Tests hot-swapping clinical protocols without Flink restart
###############################################################################

set -e

echo "=========================================="
echo "Module 3 CDC BroadcastStream Deployment"
echo "=========================================="
echo ""

FLINK_UI="http://localhost:8081"
JAR_PATH="target/flink-ehr-intelligence-1.0.0.jar"

# Check if JAR exists
if [ ! -f "$JAR_PATH" ]; then
    echo "❌ JAR not found at $JAR_PATH"
    echo "Run: mvn package -DskipTests"
    exit 1
fi

echo "✅ JAR found: $JAR_PATH"
echo "JAR size: $(ls -lh $JAR_PATH | awk '{print $5}')"
echo ""

# Upload JAR to Flink
echo "[1/4] Uploading JAR to Flink..."
UPLOAD_RESPONSE=$(curl -s -X POST -H "Content-Type: application/x-java-archive" \
    -F "jarfile=@$JAR_PATH" \
    "$FLINK_UI/jars/upload")

JAR_ID=$(echo $UPLOAD_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['filename'].split('/')[-1])" 2>/dev/null || echo "")

if [ -z "$JAR_ID" ]; then
    echo "❌ Failed to upload JAR"
    echo "Response: $UPLOAD_RESPONSE"
    exit 1
fi

echo "✅ JAR uploaded: $JAR_ID"
echo ""

# Deploy Module 3 with CDC
echo "[2/4] Deploying Module 3 with CDC BroadcastStream..."
DEPLOY_RESPONSE=$(curl -s -X POST "$FLINK_UI/jars/$JAR_ID/run" \
    -H "Content-Type: application/json" \
    -d '{
        "entryClass": "com.cardiofit.flink.operators.Module3_ComprehensiveCDS_WithCDC",
        "parallelism": 2
    }')

JOB_ID=$(echo $DEPLOY_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin).get('jobid', ''))" 2>/dev/null || echo "")

if [ -z "$JOB_ID" ]; then
    echo "❌ Failed to deploy Module 3 CDC"
    echo "Response: $DEPLOY_RESPONSE"
    exit 1
fi

echo "✅ Module 3 CDC deployed"
echo "Job ID: $JOB_ID"
echo ""

# Wait for job to start
echo "[3/4] Waiting for job to start (5 seconds)..."
sleep 5

# Check job status
echo "[4/4] Verifying job status..."
JOB_STATUS=$(curl -s "$FLINK_UI/jobs/$JOB_ID" | python3 -c "import sys, json; print(json.load(sys.stdin).get('state', ''))" 2>/dev/null || echo "UNKNOWN")

echo "Job Status: $JOB_STATUS"
echo ""

if [ "$JOB_STATUS" == "RUNNING" ]; then
    echo "=========================================="
    echo "✅ Module 3 CDC Deployment SUCCESS"
    echo "=========================================="
    echo ""
    echo "📊 Flink Web UI: $FLINK_UI"
    echo "🔍 Job Details: $FLINK_UI/#/job/$JOB_ID/overview"
    echo ""
    echo "📡 CDC Source: kb3.clinical_protocols.changes"
    echo "📥 Input Topic: clinical-patterns.v1"
    echo "📤 Output Topic: comprehensive-cds-events-cdc.v1"
    echo ""
    echo "🧪 Test CDC Hot-Swap:"
    echo "1. Check current protocol count:"
    echo "   docker logs flink-taskmanager 2>&1 | grep 'protocols (from CDC BroadcastState)'"
    echo ""
    echo "2. Insert test protocol:"
    echo "   psql -h localhost -U cardiofit_user -d kb3 << EOF"
    echo "   INSERT INTO clinical_protocols (protocol_id, name, category, specialty, version, last_updated, source)"
    echo "   VALUES ('TEST-CDC-001', 'Test Protocol', 'INFECTIOUS', 'CRITICAL_CARE', '1.0', CURRENT_DATE, 'CDC Test');"
    echo "   EOF"
    echo ""
    echo "3. Verify CDC event captured:"
    echo "   docker logs flink-taskmanager 2>&1 | grep 'CREATED Protocol in BroadcastState'"
    echo ""
    echo "4. Verify protocol used in processing:"
    echo "   docker logs flink-taskmanager 2>&1 | tail -50 | grep 'protocols (from CDC BroadcastState)'"
    echo ""
else
    echo "=========================================="
    echo "⚠️  Module 3 CDC Job Status: $JOB_STATUS"
    echo "=========================================="
    echo ""
    echo "Check Flink logs for errors:"
    echo "docker logs flink-taskmanager 2>&1 | tail -100"
fi
