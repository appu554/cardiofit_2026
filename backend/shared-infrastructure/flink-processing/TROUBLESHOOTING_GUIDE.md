# Flink JAR Deployment Troubleshooting Guide

## Problem Overview
You've uploaded the JAR file `flink-ehr-intelligence-1.0.0.jar` (180MB) to your Flink cluster, but the pipeline is failing in development with Docker-based Kafka and Flink.

## Identified Issues from Logs

### 1. **Job Not Found Errors** (Primary Issue)
```
ERROR org.apache.flink.runtime.rest.handler.job.JobDetailsHandler - Job b642ec839633f97dbf94d68870669418 not found
```
**Root Cause**: Job was submitted but failed immediately, possibly due to:
- Kafka connection failure
- Missing dependencies
- Configuration errors
- Class loading issues

### 2. **TaskManager Connection Failures**
```
Failed to connect to [jobmanager/172.18.0.3:6123] from local address [taskmanager-1/172.18.0.6]
Connection refused (Connection refused)
```
**Root Cause**: Network connectivity issues between JobManager and TaskManagers

### 3. **Kafka Connectivity Issues** (Most Likely)
Your configuration shows:
```
KAFKA_BOOTSTRAP_SERVERS=kafka:9092
KAFKA_BOOTSTRAP_SERVERS_DOCKER=kafka1:29092,kafka2:29093,kafka3:29094
```
The code uses `kafka:9092` but your Docker setup might be using the 3-node cluster with different ports.

---

## Diagnostic Steps

### Step 1: Verify Flink Cluster Health

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Check if containers are running
docker ps | grep -E "flink|kafka"

# Check Flink Web UI
curl http://localhost:8081/overview

# Check TaskManager registration
curl http://localhost:8081/taskmanagers
```

**Expected Output**: You should see 3 TaskManagers registered with 4 slots each (12 total slots).

### Step 2: Verify Kafka Connectivity from Flink Containers

```bash
# Test Kafka connection from JobManager container
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka 9092"

# If that fails, try the multi-node setup
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka1 29092"
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka2 29093"
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka3 29094"

# List Kafka topics (if kafkacat/kcat is available)
docker exec cardiofit-flink-jobmanager bash -c "kafkacat -b kafka:9092 -L" 2>/dev/null || \
  echo "kafkacat not available - install it or use kafka-topics.sh"
```

**Fix if Kafka Connection Fails**:
1. Check that both Flink and Kafka are on the same Docker network
2. Verify Kafka advertised listeners configuration
3. Update `flink-datastores.env` with correct Kafka bootstrap servers

### Step 3: Check JAR Dependencies

```bash
# Extract JAR contents to inspect
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
mkdir -p /tmp/flink-jar-inspect
cd /tmp/flink-jar-inspect
jar -xf /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/target/flink-ehr-intelligence-1.0.0.jar

# Check for Kafka connector
find . -name "*kafka*" | grep -i connector

# Verify main class is present
find . -name "FlinkJobOrchestrator.class"

# Check META-INF/MANIFEST.MF for Main-Class
cat META-INF/MANIFEST.MF | grep Main-Class
```

**Expected**:
- Should find `flink-connector-kafka` library
- Main class should be `com.cardiofit.flink.FlinkJobOrchestrator`

### Step 4: Review Job Submission Logs

```bash
# Check the most recent JobManager logs for job submission
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
tail -200 logs/flink--standalonesession-0-jobmanager.log | grep -A 10 -B 10 "Job submission"

# Look for ClassNotFoundException or NoClassDefFoundError
tail -500 logs/flink--standalonesession-0-jobmanager.log | grep -i "exception\|error" | grep -v "Job.*not found"
```

### Step 5: Test Job Submission Manually

```bash
# Submit the job with explicit configuration
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
  --detached \
  --class com.cardiofit.flink.FlinkJobOrchestrator \
  /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
  full-pipeline production

# Check submission status
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r
```

---

## Common Fixes

### Fix 1: Update Kafka Bootstrap Servers Configuration

**Issue**: Flink can't resolve `kafka:9092` from within containers.

**Solution 1 - Edit flink-datastores.env**:
```bash
# Determine your Kafka network setup
docker network inspect kafka_cardiofit-network

# Update the environment file
nano /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/flink-datastores.env

# Change this line based on your Kafka setup:
# For single-node Kafka: KAFKA_BOOTSTRAP_SERVERS=kafka:9092
# For 3-node cluster: KAFKA_BOOTSTRAP_SERVERS=kafka1:29092,kafka2:29093,kafka3:29094
```

**Solution 2 - Add Kafka host to JobManager**:
```bash
# Add to docker-compose.yml under jobmanager.extra_hosts:
extra_hosts:
  - "kafka:172.x.x.x"  # Replace with actual Kafka container IP
```

### Fix 2: Ensure Both Networks Are Accessible

**Issue**: Flink containers need access to both `cardiofit-network` and `kafka-network`.

**Verify networks exist**:
```bash
docker network ls | grep -E "cardiofit|kafka"
```

**Create networks if missing**:
```bash
docker network create cardiofit-network
docker network create kafka_cardiofit-network
```

**Restart Flink with corrected networks**:
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
docker-compose down
docker-compose up -d
```

### Fix 3: Add Kafka Connector to Classpath (if missing from JAR)

**Issue**: Flink-Kafka connector might be marked as `provided` scope and not included.

**Solution - Add to Flink lib directory**:
```bash
# Download Flink Kafka connector
wget https://repo1.maven.org/maven2/org/apache/flink/flink-connector-kafka/1.17.1/flink-connector-kafka-1.17.1.jar

# Copy to JobManager
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-jobmanager:/opt/flink/lib/

# Copy to TaskManagers
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-taskmanager-1:/opt/flink/lib/
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-taskmanager-2:/opt/flink/lib/
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-taskmanager-3:/opt/flink/lib/

# Restart Flink cluster
docker restart cardiofit-flink-jobmanager cardiofit-flink-taskmanager-1 cardiofit-flink-taskmanager-2 cardiofit-flink-taskmanager-3
```

### Fix 4: Rebuild JAR with Proper Shading

**Issue**: Maven shade plugin might have excluded critical dependencies.

**Solution - Modify pom.xml**:
```xml
<!-- In pom.xml, update the shade plugin excludes section -->
<artifactSet>
    <excludes>
        <exclude>org.apache.flink:flink-shaded-force-shading</exclude>
        <exclude>com.google.code.findbugs:jsr305</exclude>
        <!-- REMOVE these exclusions for included dependencies -->
        <!-- <exclude>org.slf4j:*</exclude> -->
        <!-- <exclude>org.apache.logging.log4j:*</exclude> -->
    </excludes>
</artifactSet>
```

**Rebuild**:
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
mvn clean package -DskipTests
```

### Fix 5: Check Kafka Topic Existence

**Issue**: Source topics don't exist yet.

**Solution - Create required topics**:
```bash
# List existing topics
docker exec <kafka-container-name> kafka-topics --list --bootstrap-server localhost:9092

# Create missing topics
for topic in patient-events.v1 medication-events.v1 observation-events.v1 \
             vital-signs-events.v1 lab-result-events.v1 safety-events.v1; do
  docker exec <kafka-container-name> kafka-topics --create \
    --topic $topic \
    --bootstrap-server localhost:9092 \
    --partitions 4 \
    --replication-factor 1
done
```

---

## Testing the Fix

### 1. Submit Test Job
```bash
# Start with a simple ingestion-only job first
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
  --detached \
  --class com.cardiofit.flink.FlinkJobOrchestrator \
  /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
  ingestion-only development
```

### 2. Monitor Job Status
```bash
# Watch job in Flink UI
open http://localhost:8081

# Check running jobs
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list

# Tail logs for errors
docker logs -f cardiofit-flink-jobmanager 2>&1 | grep -i "error\|exception"
```

### 3. Send Test Event
```bash
# Produce a test event to Kafka
docker exec <kafka-container> kafka-console-producer \
  --broker-list localhost:9092 \
  --topic patient-events.v1 << EOF
{
  "patientId": "test-patient-123",
  "eventType": "vital-signs",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "data": {
    "heartRate": 75,
    "bloodPressure": "120/80"
  },
  "critical": false,
  "clinicalEvent": true
}
EOF
```

### 4. Verify Processing
```bash
# Check Flink metrics
curl http://localhost:8081/jobs/<job-id>/metrics

# Check if events are flowing
docker logs cardiofit-flink-taskmanager-1 | grep -i "processed\|enriched"
```

---

## Quick Diagnostic Script

Save this as `diagnose-flink-kafka.sh`:

```bash
#!/bin/bash

echo "=== Flink Kafka Pipeline Diagnostics ==="
echo

echo "1. Docker Containers Status:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "flink|kafka"
echo

echo "2. Flink Cluster Overview:"
curl -s http://localhost:8081/overview | jq '.' 2>/dev/null || echo "Flink UI not accessible"
echo

echo "3. TaskManager Count:"
curl -s http://localhost:8081/taskmanagers | jq '.taskmanagers | length' 2>/dev/null || echo "Unable to get TaskManager count"
echo

echo "4. Running Jobs:"
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r 2>/dev/null || echo "Unable to list jobs"
echo

echo "5. Kafka Connectivity from Flink:"
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka 9092" 2>&1 || echo "Kafka not reachable as 'kafka:9092'"
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka1 29092" 2>&1 || echo "Kafka not reachable as 'kafka1:29092'"
echo

echo "6. Docker Networks:"
docker network ls | grep -E "cardiofit|kafka"
echo

echo "7. Recent JobManager Errors (last 50):"
tail -50 /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/logs/flink--standalonesession-0-jobmanager.log | \
  grep -i "error\|exception" | grep -v "Job.*not found" | tail -10
echo

echo "=== Diagnostic Complete ==="
```

Make it executable and run:
```bash
chmod +x diagnose-flink-kafka.sh
./diagnose-flink-kafka.sh
```

---

## Expected Error Messages and Solutions

| Error Message | Likely Cause | Solution |
|--------------|--------------|----------|
| `org.apache.kafka.common.errors.TimeoutException: Topic not present` | Kafka topics don't exist | Create topics using kafka-topics CLI |
| `java.net.UnknownHostException: kafka` | DNS resolution failure | Update bootstrap servers or add to /etc/hosts |
| `ClassNotFoundException: org.apache.kafka.clients.consumer.ConsumerRecord` | Missing Kafka client library | Add Kafka connector to Flink lib/ |
| `NoClassDefFoundError: org/apache/flink/connector/kafka/source/KafkaSource` | Wrong Flink version or missing connector | Rebuild JAR or add connector to classpath |
| `Connection refused (Connection refused)` | Network misconfiguration | Verify Docker networks and Kafka advertised listeners |
| `Job submission failed` | Various (check full stack trace) | Review full exception in JobManager logs |

---

## Recommended Actions (Priority Order)

1. **RUN DIAGNOSTIC SCRIPT**: Get full picture of current state
2. **VERIFY KAFKA CONNECTIVITY**: Most common issue with Docker setups
3. **CHECK KAFKA TOPICS**: Ensure all 6 source topics exist
4. **TEST MANUAL SUBMISSION**: Submit job via CLI for better error visibility
5. **REVIEW FULL LOGS**: Check last 500 lines of JobManager logs for complete error
6. **UPDATE CONFIGURATION**: Fix Kafka bootstrap servers if needed
7. **RESTART CLUSTER**: After configuration changes

---

## Next Steps

**Please provide the following information:**

1. **Run the diagnostic script** and share the output
2. **Kafka setup details**:
   - Single node or multi-node cluster?
   - Container names for Kafka brokers
   - Output of `docker network inspect kafka_cardiofit-network`
3. **Full error message**: Complete stack trace from when you tested the pipeline
4. **Job submission method**: How did you upload/submit the JAR? (Web UI, CLI, REST API?)

With this information, I can provide a precise fix for your specific setup.

---

## Reference: Flink-Kafka-Docker Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Docker Host                          │
│                                                          │
│  ┌──────────────────┐         ┌──────────────────┐     │
│  │  Flink JobManager│         │   Kafka Cluster   │     │
│  │  cardiofit-flink │         │   kafka:9092      │     │
│  │  Port: 8081      │◄───────►│   kafka1:29092    │     │
│  └────────┬─────────┘         │   kafka2:29093    │     │
│           │                   │   kafka3:29094    │     │
│           │ RPC 6123          └──────────────────┘     │
│           │                             ▲              │
│  ┌────────▼─────────┐                   │              │
│  │  TaskManager 1-3 │                   │              │
│  │  4 slots each    │◄──────────────────┘              │
│  └──────────────────┘        Kafka Network             │
│           │                  (kafka_cardiofit-network) │
│           │                                             │
│  ┌────────▼─────────┐                                  │
│  │   JAR: flink-ehr-│                                  │
│  │   intelligence   │                                  │
│  │   180MB          │                                  │
│  └──────────────────┘                                  │
└─────────────────────────────────────────────────────────┘
```

The key is ensuring network connectivity between Flink and Kafka containers!