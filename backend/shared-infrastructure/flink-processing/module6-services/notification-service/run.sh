#!/bin/bash

# CardioFit Notification Service - Run Script

set -e

echo "========================================="
echo "CardioFit Notification Service"
echo "========================================="

# Load environment variables
if [ -f .env ]; then
    echo "Loading environment variables from .env..."
    export $(cat .env | grep -v '^#' | xargs)
else
    echo "Warning: .env file not found. Using default values."
fi

# Check if JAR exists
JAR_FILE="target/notification-service-1.0.0.jar"

if [ ! -f "$JAR_FILE" ]; then
    echo "JAR file not found. Building application..."
    mvn clean package -DskipTests
fi

# Verify Firebase credentials
if [ "$FIREBASE_ENABLED" = "true" ] && [ ! -f "$FIREBASE_CREDENTIALS_PATH" ]; then
    echo "Warning: Firebase is enabled but credentials file not found: $FIREBASE_CREDENTIALS_PATH"
    echo "Push notifications will not work. Set FIREBASE_ENABLED=false to disable."
fi

# Start application
echo ""
echo "Starting Notification Service on port ${SERVER_PORT:-8070}..."
echo "Kafka: ${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
echo "Redis: ${REDIS_HOST:-localhost}:${REDIS_PORT:-6379}"
echo ""

java -jar "$JAR_FILE" \
  --spring.kafka.bootstrap-servers="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}" \
  --spring.data.redis.host="${REDIS_HOST:-localhost}" \
  --spring.data.redis.port="${REDIS_PORT:-6379}" \
  --server.port="${SERVER_PORT:-8070}"
