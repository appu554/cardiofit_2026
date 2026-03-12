#!/bin/bash

# ============================================================================
# Deploy Modules 1 & 2 to Flink 2.1.0 Cluster
# ============================================================================

set -e

FLINK_JOBMANAGER="http://localhost:8081"
JAR_PATH="./target/flink-ehr-intelligence-1.0.0.jar"

echo "🚀 CardioFit Flink 2.1.0 - Module 1 & 2 Deployment"
echo "=================================================="

# Wait for Flink cluster to be ready
echo "⏳ Waiting for Flink cluster..."
for i in {1..30}; do
    if curl -s "$FLINK_JOBMANAGER/overview" > /dev/null 2>&1; then
        echo "✅ Flink cluster is ready!"
        break
    fi
    echo "   Attempt $i/30..."
    sleep 2
done

if ! curl -s "$FLINK_JOBMANAGER/overview" > /dev/null 2>&1; then
    echo "❌ Flink cluster not accessible at $FLINK_JOBMANAGER"
    exit 1
fi

# Upload JAR to Flink cluster
echo ""
echo "📦 Uploading JAR to Flink cluster..."
JAR_UPLOAD_RESPONSE=$(curl -s -X POST -F "jarfile=@$JAR_PATH" "$FLINK_JOBMANAGER/jars/upload")
JAR_FILENAME=$(echo "$JAR_UPLOAD_RESPONSE" | grep -o '"filename":"[^"]*"' | cut -d'"' -f4)

if [ -z "$JAR_FILENAME" ]; then
    echo "❌ Failed to upload JAR"
    echo "Response: $JAR_UPLOAD_RESPONSE"
    exit 1
fi

# URL encode the JAR path
JAR_ID=$(echo "$JAR_FILENAME" | sed 's|/|%2F|g')

echo "✅ JAR uploaded successfully!"
echo "   Filename: $JAR_FILENAME"
echo "   JAR ID: $JAR_ID"

# Deploy Module 1 - Ingestion & Validation
echo ""
echo "🎯 Deploying Module 1: Ingestion & Validation"
echo "----------------------------------------------"

MODULE1_RESPONSE=$(curl -s -X POST "$FLINK_JOBMANAGER/jars/$JAR_ID/run" \
    -H "Content-Type: application/json" \
    -d '{
        "entryClass": "com.cardiofit.flink.operators.Module1_IngestionValidation",
        "parallelism": 2,
        "programArgs": "ingestion-only development",
        "savepointPath": null,
        "allowNonRestoredState": false
    }')

MODULE1_JOB_ID=$(echo "$MODULE1_RESPONSE" | grep -o '"jobid":"[^"]*"' | cut -d'"' -f4)

if [ -z "$MODULE1_JOB_ID" ]; then
    echo "❌ Failed to deploy Module 1"
    echo "Response: $MODULE1_RESPONSE"
else
    echo "✅ Module 1 deployed successfully!"
    echo "   Job ID: $MODULE1_JOB_ID"
    echo "   View at: $FLINK_JOBMANAGER/#/job/$MODULE1_JOB_ID"
fi

# Wait a bit before deploying Module 2
sleep 3

# Deploy Module 2 - Context Assembly
echo ""
echo "🎯 Deploying Module 2: Context Assembly"
echo "----------------------------------------------"

MODULE2_RESPONSE=$(curl -s -X POST "$FLINK_JOBMANAGER/jars/$JAR_ID/run" \
    -H "Content-Type: application/json" \
    -d '{
        "entryClass": "com.cardiofit.flink.operators.Module2_ContextAssembly",
        "parallelism": 2,
        "programArgs": "context-assembly development",
        "savepointPath": null,
        "allowNonRestoredState": false
    }')

MODULE2_JOB_ID=$(echo "$MODULE2_RESPONSE" | grep -o '"jobid":"[^"]*"' | cut -d'"' -f4)

if [ -z "$MODULE2_JOB_ID" ]; then
    echo "❌ Failed to deploy Module 2"
    echo "Response: $MODULE2_RESPONSE"
else
    echo "✅ Module 2 deployed successfully!"
    echo "   Job ID: $MODULE2_JOB_ID"
    echo "   View at: $FLINK_JOBMANAGER/#/job/$MODULE2_JOB_ID"
fi

# Summary
echo ""
echo "🎉 Deployment Summary"
echo "===================="
echo "Module 1 (Ingestion):    ${MODULE1_JOB_ID:-FAILED}"
echo "Module 2 (Context):      ${MODULE2_JOB_ID:-FAILED}"
echo ""
echo "🌐 Flink Web UI: $FLINK_JOBMANAGER"
echo ""
echo "Next steps:"
echo "1. Verify jobs are RUNNING in Flink UI"
echo "2. Run ./create-kafka-topics.sh to create input topics"
echo "3. Run ./send-test-events.sh to test the pipeline"
