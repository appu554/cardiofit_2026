#!/bin/bash

# ============================================================================
# Deploy Module 5: ML Inference Engine (MIMIC-IV ONNX Models)
# ============================================================================

set -e

FLINK_JOBMANAGER="http://localhost:8081"
JAR_PATH="./target/flink-ehr-intelligence-1.0.0.jar"

echo "🚀 CardioFit Flink 2.1.0 - Module 5 ML Inference Deployment"
echo "============================================================"
echo ""
echo "📋 Deployment Configuration:"
echo "   - MIMIC-IV Models: Sepsis, Deterioration, Mortality"
echo "   - Input Topic: clinical-patterns.v1 (Module 2 output)"
echo "   - Output Topics: inference-results.v1, alert-management.v1"
echo "   - Parallelism: 6 workers"
echo "   - Feature Scaling: Z-score standardization (37 features)"
echo ""

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

# Show current cluster status
echo ""
echo "📊 Cluster Status:"
CLUSTER_INFO=$(curl -s "$FLINK_JOBMANAGER/overview")
echo "   Task Managers: $(echo $CLUSTER_INFO | grep -o '"taskmanagers":[0-9]*' | cut -d':' -f2)"
echo "   Available Slots: $(echo $CLUSTER_INFO | grep -o '"slots-available":[0-9]*' | cut -d':' -f2)"
echo "   Running Jobs: $(echo $CLUSTER_INFO | grep -o '"jobs-running":[0-9]*' | cut -d':' -f2)"

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

# Deploy Module 5 - ML Inference Engine
echo ""
echo "🎯 Deploying Module 5: ML Inference Engine (MIMIC-IV)"
echo "------------------------------------------------------"

MODULE5_RESPONSE=$(curl -s -X POST "$FLINK_JOBMANAGER/jars/$JAR_ID/run" \
    -H "Content-Type: application/json" \
    -d '{
        "entryClass": "com.cardiofit.flink.operators.Module5_MLInference",
        "parallelism": 6,
        "programArgs": "ml-inference production",
        "savepointPath": null,
        "allowNonRestoredState": false
    }')

MODULE5_JOB_ID=$(echo "$MODULE5_RESPONSE" | grep -o '"jobid":"[^"]*"' | cut -d'"' -f4)

if [ -z "$MODULE5_JOB_ID" ]; then
    echo "❌ Failed to deploy Module 5"
    echo "Response: $MODULE5_RESPONSE"
    exit 1
else
    echo "✅ Module 5 deployed successfully!"
    echo "   Job ID: $MODULE5_JOB_ID"
    echo "   View at: $FLINK_JOBMANAGER/#/job/$MODULE5_JOB_ID"
fi

# Wait for job to start
echo ""
echo "⏳ Waiting for job to start..."
sleep 5

# Check job status
echo ""
echo "📊 Job Status Check:"
JOB_STATUS=$(curl -s "$FLINK_JOBMANAGER/jobs/$MODULE5_JOB_ID")
JOB_STATE=$(echo "$JOB_STATUS" | grep -o '"state":"[^"]*"' | cut -d'"' -f4)

echo "   Current State: $JOB_STATE"

if [ "$JOB_STATE" = "RUNNING" ]; then
    echo "   ✅ Job is RUNNING successfully!"
else
    echo "   ⚠️  Job state: $JOB_STATE (may still be initializing)"
fi

# Summary
echo ""
echo "🎉 Deployment Summary"
echo "===================="
echo "Module 5 Job ID: $MODULE5_JOB_ID"
echo "Status: $JOB_STATE"
echo ""
echo "📋 Pipeline Architecture:"
echo "   Module 2 → clinical-patterns.v1"
echo "            ↓"
echo "   Module 5 (MIMIC-IV ML Inference)"
echo "            ├→ inference-results.v1 (all predictions)"
echo "            └→ alert-management.v1 (HIGH risk only)"
echo ""
echo "🔮 ML Models Loaded:"
echo "   1. Sepsis Risk v2.0.0 (AUROC 98.55%)"
echo "   2. Deterioration Risk v2.0.0 (AUROC 78.96%)"
echo "   3. Mortality Risk v2.0.0 (AUROC 95.70%)"
echo ""
echo "🌐 Flink Web UI: $FLINK_JOBMANAGER"
echo ""
echo "Next steps:"
echo "1. Monitor job in Flink UI: $FLINK_JOBMANAGER/#/job/$MODULE5_JOB_ID"
echo "2. Verify predictions: docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic inference-results.v1 --from-beginning"
echo "3. Test with patient: ./test-mimic-inference.sh"
echo ""
