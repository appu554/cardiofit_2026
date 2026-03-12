#!/bin/bash

# Setup Kafka Topics for Stage 1 & Stage 2 Testing
# Run this script to create all required Kafka topics

set -e

echo "🚀 Setting up Kafka topics for Stage 1 & Stage 2 testing..."

# Kafka configuration
BOOTSTRAP_SERVERS="pkc-619z3.us-east1.gcp.confluent.cloud:9092"
API_KEY="LGJ3AQ2L6VRPW4S2"
API_SECRET="2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl"

# Check if confluent CLI is installed
if ! command -v confluent &> /dev/null; then
    echo "❌ Confluent CLI not found. Installing..."
    curl -sL --http1.1 https://cnfl.io/cli | sh -s -- latest
    export PATH=$PATH:$HOME/.confluent/bin
fi

# Login to Confluent Cloud (if not already logged in)
echo "🔐 Logging into Confluent Cloud..."
confluent login --save || echo "Already logged in"

# Set environment and cluster
echo "🌐 Setting Confluent environment..."
confluent environment use env-your-env-id || echo "Using current environment"
confluent kafka cluster use lkc-x86njx || echo "Using current cluster"

echo "📝 Creating Kafka topics..."

# Stage 1 Topics
echo "Creating Stage 1 topics..."

# Validated device data topic (Stage 1 output)
confluent kafka topic create validated-device-data.v1 \
  --partitions 12 \
  --config retention.ms=259200000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2 \
  || echo "Topic validated-device-data.v1 already exists"

# Failed validation topic (Stage 1 DLQ)
confluent kafka topic create failed-validation.v1 \
  --partitions 4 \
  --config retention.ms=2592000000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2 \
  || echo "Topic failed-validation.v1 already exists"

# Critical data DLQ topic
confluent kafka topic create critical-data-dlq.v1 \
  --partitions 4 \
  --config retention.ms=7776000000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2 \
  || echo "Topic critical-data-dlq.v1 already exists"

# Poison messages topic (Stage 1)
confluent kafka topic create poison-messages.v1 \
  --partitions 2 \
  --config retention.ms=31536000000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2 \
  || echo "Topic poison-messages.v1 already exists"

# Stage 2 Topics
echo "Creating Stage 2 topics..."

# Sink write failures topic (Stage 2 DLQ)
confluent kafka topic create sink-write-failures.v1 \
  --partitions 6 \
  --config retention.ms=1209600000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2 \
  || echo "Topic sink-write-failures.v1 already exists"

# Critical sink failures topic
confluent kafka topic create critical-sink-failures.v1 \
  --partitions 4 \
  --config retention.ms=7776000000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2 \
  || echo "Topic critical-sink-failures.v1 already exists"

# Poison messages topic (Stage 2)
confluent kafka topic create poison-messages-stage2.v1 \
  --partitions 2 \
  --config retention.ms=31536000000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2 \
  || echo "Topic poison-messages-stage2.v1 already exists"

echo "✅ Kafka topics created successfully!"

# List created topics
echo "📋 Listing created topics:"
confluent kafka topic list | grep -E "(validated-device-data|failed-validation|critical-data-dlq|poison-messages|sink-write-failures|critical-sink-failures)"

echo "🎉 Kafka topics setup complete!"
echo ""
echo "Next steps:"
echo "1. Update your Kafka API secret in the script"
echo "2. Run Stage 1: cd backend/stream-services/stage1-validator-enricher && mvn spring-boot:run"
echo "3. Run Stage 2: cd backend/stream-services/stage2-storage-fanout && python -m uvicorn app.main:app --host 0.0.0.0 --port 8042"
