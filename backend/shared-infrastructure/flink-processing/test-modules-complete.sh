#!/bin/bash
set -e

echo "========================================="
echo "Module 1 & 2 Complete Test Pipeline"
echo "========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Recreate topics fresh
echo -e "${YELLOW}Step 1: Recreating Kafka topics...${NC}"
docker exec kafka kafka-topics --delete --topic patient-events-v1 --bootstrap-server localhost:9092 2>/dev/null || true
docker exec kafka kafka-topics --delete --topic enriched-patient-events-v1 --bootstrap-server localhost:9092 2>/dev/null || true
docker exec kafka kafka-topics --delete --topic clinical-patterns.v1 --bootstrap-server localhost:9092 2>/dev/null || true

sleep 2

docker exec kafka kafka-topics --create --topic patient-events-v1 --bootstrap-server localhost:9092 --partitions 2 --replication-factor 1
docker exec kafka kafka-topics --create --topic enriched-patient-events-v1 --bootstrap-server localhost:9092 --partitions 2 --replication-factor 1
docker exec kafka kafka-topics --create --topic clinical-patterns.v1 --bootstrap-server localhost:9092 --partitions 2 --replication-factor 1

echo -e "${GREEN}✓ Topics created${NC}"

# Step 2: Cancel existing jobs
echo -e "${YELLOW}Step 2: Canceling existing Flink jobs...${NC}"
for job_id in $(curl -s http://localhost:8081/jobs | python3 -c "import sys, json; data=json.load(sys.stdin); [print(job['id']) for job in data.get('jobs', []) if job.get('status') == 'RUNNING']" 2>/dev/null); do
  curl -s -X PATCH "http://localhost:8081/jobs/${job_id}?mode=cancel" > /dev/null
  echo "  Cancelled job: $job_id"
done

sleep 5
echo -e "${GREEN}✓ Jobs cancelled${NC}"

# Step 3: Deploy Module 1
echo -e "${YELLOW}Step 3: Deploying Module 1...${NC}"
JAR_ID=$(curl -s http://localhost:8081/jars | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['files'][0]['id'])" 2>/dev/null)

MODULE1_JOB=$(curl -s -X POST "http://localhost:8081/jars/${JAR_ID}/run" \
  -H "Content-Type: application/json" \
  -d '{"entryClass":"com.cardiofit.flink.operators.Module1_Ingestion","parallelism":2}' | python3 -c "import sys, json; print(json.load(sys.stdin)['jobid'])" 2>/dev/null)

echo -e "${GREEN}✓ Module 1 deployed: $MODULE1_JOB${NC}"

# Step 4: Deploy Module 2
echo -e "${YELLOW}Step 4: Deploying Module 2...${NC}"
MODULE2_JOB=$(curl -s -X POST "http://localhost:8081/jars/${JAR_ID}/run" \
  -H "Content-Type: application/json" \
  -d '{"entryClass":"com.cardiofit.flink.operators.Module2_ContextAssembly","parallelism":2}' | python3 -c "import sys, json; print(json.load(sys.stdin)['jobid'])" 2>/dev/null)

echo -e "${GREEN}✓ Module 2 deployed: $MODULE2_JOB${NC}"

# Wait for jobs to stabilize
sleep 5

# Step 5: Send test event
echo -e "${YELLOW}Step 5: Sending test patient event...${NC}"
echo '{"patient_id":"test-patient-001","event_time":1728518400000,"type":"vital_signs","source":"bedside-monitor","payload":{"heart_rate":105,"blood_pressure_systolic":145,"blood_pressure_diastolic":92,"oxygen_saturation":94,"temperature":38.2,"respiratory_rate":24},"metadata":{"unit":"ICU-2A","encounter_id":"ENC-12345"}}' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

echo -e "${GREEN}✓ Test event sent${NC}"

# Step 6: Wait and check processing
echo -e "${YELLOW}Step 6: Waiting for processing (10 seconds)...${NC}"
sleep 10

# Step 7: Verify results
echo -e "${YELLOW}Step 7: Verifying results...${NC}"

INPUT_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic patient-events-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
MODULE1_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
MODULE2_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic clinical-patterns.v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')

echo "Message counts:"
echo "  patient-events-v1: $INPUT_COUNT"
echo "  enriched-patient-events-v1 (Module 1 output): $MODULE1_COUNT"
echo "  clinical-patterns.v1 (Module 2 output): $MODULE2_COUNT"

# Step 8: Display outputs
if [ "$MODULE2_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ Pipeline SUCCESS!${NC}"
  echo ""
  echo -e "${YELLOW}Module 2 FHIR-Enriched Output:${NC}"
  docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic clinical-patterns.v1 --from-beginning --max-messages 1 --timeout-ms 3000 2>&1 | python3 -m json.tool 2>/dev/null || echo "(JSON formatting failed - raw output shown above)"
else
  echo -e "${RED}✗ Pipeline issue - checking for errors...${NC}"

  # Check for exceptions
  echo "Module 1 exceptions:"
  curl -s "http://localhost:8081/jobs/${MODULE1_JOB}/exceptions" | python3 -c "import sys, json; data=json.load(sys.stdin); entries=data.get('exceptionHistory', {}).get('entries', []); print(f'  Count: {len(entries)}'); [print(f'  - {e[\"exceptionName\"]}') for e in entries[:3]]" 2>/dev/null

  echo "Module 2 exceptions:"
  curl -s "http://localhost:8081/jobs/${MODULE2_JOB}/exceptions" | python3 -c "import sys, json; data=json.load(sys.stdin); entries=data.get('exceptionHistory', {}).get('entries', []); print(f'  Count: {len(entries)}'); [print(f'  - {e[\"exceptionName\"]}') for e in entries[:3]]" 2>/dev/null
fi

echo ""
echo "========================================="
echo "Test Complete!"
echo "========================================="
