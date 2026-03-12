#!/bin/bash

# Deploy all 6 Flink modules for EHR Intelligence Platform
# This script deploys modules in sequence with proper logging

set -e

FLINK_URL="http://localhost:8081"
JAR_ID="8ebe5658-a03d-4f7b-b9fa-2e003a153af9_flink-ehr-intelligence-1.0.0.jar"

echo "==================================="
echo "Deploying All Flink Modules"
echo "==================================="
echo ""

# Module 1: Ingestion
echo "📥 Deploying Module 1: Ingestion (FHIR Event Ingestion)"
MODULE1_RESPONSE=$(curl -s -X POST "$FLINK_URL/jars/$JAR_ID/run" \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.operators.Module1_Ingestion",
    "parallelism": 2,
    "programArgs": "",
    "savepointPath": null
  }')
MODULE1_JOB_ID=$(echo $MODULE1_RESPONSE | jq -r '.jobid')
echo "✅ Module 1 deployed - Job ID: $MODULE1_JOB_ID"
echo ""
sleep 3

# Module 2: Enhanced Context Assembly
echo "🔗 Deploying Module 2: Enhanced Context Assembly"
MODULE2_RESPONSE=$(curl -s -X POST "$FLINK_URL/jars/$JAR_ID/run" \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.operators.Module2_Enhanced",
    "parallelism": 2,
    "programArgs": "",
    "savepointPath": null
  }')
MODULE2_JOB_ID=$(echo $MODULE2_RESPONSE | jq -r '.jobid')
echo "✅ Module 2 deployed - Job ID: $MODULE2_JOB_ID"
echo ""
sleep 3

# Module 3: Comprehensive CDS
echo "🏥 Deploying Module 3: Comprehensive CDS (Clinical Decision Support)"
MODULE3_RESPONSE=$(curl -s -X POST "$FLINK_URL/jars/$JAR_ID/run" \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.operators.Module3_ComprehensiveCDS",
    "parallelism": 2,
    "programArgs": "",
    "savepointPath": null
  }')
MODULE3_JOB_ID=$(echo $MODULE3_RESPONSE | jq -r '.jobid')
echo "✅ Module 3 deployed - Job ID: $MODULE3_JOB_ID"
echo ""
sleep 3

# Module 4: Pattern Detection
echo "🔍 Deploying Module 4: Pattern Detection (CEP)"
MODULE4_RESPONSE=$(curl -s -X POST "$FLINK_URL/jars/$JAR_ID/run" \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.operators.Module4_PatternDetection",
    "parallelism": 2,
    "programArgs": "",
    "savepointPath": null
  }')
MODULE4_JOB_ID=$(echo $MODULE4_RESPONSE | jq -r '.jobid')
echo "✅ Module 4 deployed - Job ID: $MODULE4_JOB_ID"
echo ""
sleep 3

# Module 5: ML Inference
echo "🤖 Deploying Module 5: ML Inference (ONNX Models)"
MODULE5_RESPONSE=$(curl -s -X POST "$FLINK_URL/jars/$JAR_ID/run" \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.operators.Module5_MLInference",
    "parallelism": 2,
    "programArgs": "",
    "savepointPath": null
  }')
MODULE5_JOB_ID=$(echo $MODULE5_RESPONSE | jq -r '.jobid')
echo "✅ Module 5 deployed - Job ID: $MODULE5_JOB_ID"
echo ""
sleep 3

# Module 6: Analytics Engine
echo "📊 Deploying Module 6: Analytics Engine (SQL Views + Dashboard)"
MODULE6_RESPONSE=$(curl -s -X POST "$FLINK_URL/jars/$JAR_ID/run" \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.analytics.Module6_AnalyticsEngine",
    "parallelism": 2,
    "programArgs": "",
    "savepointPath": null
  }')
MODULE6_JOB_ID=$(echo $MODULE6_RESPONSE | jq -r '.jobid')
echo "✅ Module 6 deployed - Job ID: $MODULE6_JOB_ID"
echo ""
sleep 2

# Verify all jobs are running
echo "==================================="
echo "✅ Deployment Summary"
echo "==================================="
echo "Module 1 (Ingestion):      $MODULE1_JOB_ID"
echo "Module 2 (Context):        $MODULE2_JOB_ID"
echo "Module 3 (CDS):            $MODULE3_JOB_ID"
echo "Module 4 (Pattern):        $MODULE4_JOB_ID"
echo "Module 5 (ML Inference):   $MODULE5_JOB_ID"
echo "Module 6 (Analytics):      $MODULE6_JOB_ID"
echo ""

# Check status
echo "🔍 Checking job status..."
curl -s "$FLINK_URL/jobs/overview" | jq '.jobs[] | {name: .name, state: .state, "job-id": .jid}'
echo ""
echo "✅ All modules deployed successfully!"
echo "🌐 Flink Dashboard: http://localhost:8081"
echo "📊 Analytics Dashboard API: http://localhost:8050/graphql"
