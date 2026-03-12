# Module 1 - Quick Start Guide

**Status**: ✅ Production Ready
**Last Updated**: October 10, 2025

---

## 🚀 Quick Deployment (5 Minutes)

### Prerequisites
- Docker & Docker Compose installed
- Kafka cluster running on `kafka_cardiofit-network`
- Maven 3.6+ and Java 11+

### 1. Start Flink Cluster
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
docker-compose up -d
```

### 2. Create Kafka Topics
```bash
# Required input topics
for topic in patient-events-v1 medication-events-v1 observation-events-v1 \
             vital-signs-events-v1 lab-result-events-v1 validated-device-data-v1; do
  docker exec kafka kafka-topics --create --topic $topic \
    --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1
done

# Output topics
docker exec kafka kafka-topics --create --topic enriched-patient-events-v1 \
  --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1

docker exec kafka kafka-topics --create --topic dlq.processing-errors.v1 \
  --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1
```

### 3. Build & Upload JAR
```bash
mvn clean package -DskipTests -Dmaven.test.skip=true

curl -X POST -H "Expect:" \
  -F "jarfile=@target/flink-ehr-intelligence-1.0.0.jar" \
  http://localhost:8081/jars/upload
```

### 4. Deploy Module 1
```bash
JAR_ID=$(curl -s http://localhost:8081/jars | \
  python3 -c "import sys, json; files=json.load(sys.stdin)['files']; print(files[-1]['id'])")

curl -s -X POST "http://localhost:8081/jars/${JAR_ID}/run" \
  -H "Content-Type: application/json" \
  -d '{"entryClass":"com.cardiofit.flink.operators.Module1_Ingestion","parallelism":2}'
```

### 5. Test It
```bash
chmod +x test-module1-production.sh
./test-module1-production.sh
```

---

## 📊 Monitoring URLs

- **Flink Web UI**: http://localhost:8081
- **Kafka UI**: http://localhost:8080

---

## ✅ Health Check

```bash
# Check Flink job status
curl -s http://localhost:8081/jobs | python3 -c \
  "import sys, json; jobs=json.load(sys.stdin)['jobs']; print(f\"Status: {jobs[0]['status'] if jobs else 'No jobs'}\")"

# Check message counts
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1
```

---

## 📝 Send Test Event

```bash
echo '{"patient_id":"TEST-001","event_time":1760066833774,"type":"vital_signs","source":"test","payload":{"heart_rate":120},"metadata":{}}' | \
  docker exec -i kafka kafka-console-producer \
  --bootstrap-server localhost:9092 --topic patient-events-v1

# Check output (wait 2 seconds)
sleep 2
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning --max-messages 1 --timeout-ms 5000
```

---

## 🔧 Troubleshooting

**Job keeps restarting?**
→ Check `docker logs flink-jobmanager-2.1` for errors

**No output messages?**
→ Ensure job is RUNNING before sending events

**DLQ messages appearing?**
→ Check validation rules in MODULE1_INPUT_FORMAT.md

---

## 📚 Full Documentation

See `MODULE1_PRODUCTION_VALIDATION.md` for complete details.
