# 🚀 Kafka Topics Architecture Setup
## Modular Stream Processing - Stage 1 & 2 Topics

### 📋 **Topic Strategy Overview**

This document defines the Kafka topics architecture for our **3-stage modular pipeline**, replacing the monolithic Spark reactor with specialized topics for each processing stage.

---

## 🎯 **Topic Architecture**

### **Input Topics (From Global Outbox)**
```yaml
# Existing topic - already in use
raw-device-data.v1:
  description: "Raw device readings from Global Outbox Service"
  partitions: 12
  replication: 3
  retention: 7d
  cleanup.policy: delete
  compression.type: snappy
```

### **Stage 1 Output Topics (Validator & Enricher)**
```yaml
# New topic - validated and enriched data
validated-device-data.v1:
  description: "Clean, validated device data with patient context"
  partitions: 12
  replication: 3
  retention: 3d
  cleanup.policy: delete
  compression.type: snappy
  
# New topic - dead letter queue for invalid data
failed-validation.v1:
  description: "Invalid device data that failed validation"
  partitions: 4
  replication: 3
  retention: 30d
  cleanup.policy: delete
  compression.type: snappy
```

### **Stage 2 Output Topics (Storage Fan-Out)**
```yaml
# New topic - FHIR processing results
fhir-processing-results.v1:
  description: "Results of FHIR transformation and sink writes"
  partitions: 6
  replication: 3
  retention: 7d
  cleanup.policy: delete
  compression.type: snappy
  
# New topic - sink write failures
sink-write-failures.v1:
  description: "Failed sink writes for retry processing"
  partitions: 4
  replication: 3
  retention: 14d
  cleanup.policy: delete
  compression.type: snappy
```

---

## 🔧 **Confluent Cloud Topic Creation Commands**

### **Prerequisites**
```bash
# Install Confluent CLI
curl -sL --http1.1 https://cnfl.io/cli | sh -s -- latest

# Login to Confluent Cloud
confluent login --save

# Set environment and cluster
confluent environment use <your-environment-id>
confluent kafka cluster use lkc-x86njx
```

### **Create Stage 1 Topics**
```bash
# Validated device data topic
confluent kafka topic create validated-device-data.v1 \
  --partitions 12 \
  --config retention.ms=259200000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2

# Failed validation topic (DLQ)
confluent kafka topic create failed-validation.v1 \
  --partitions 4 \
  --config retention.ms=2592000000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2
```

### **Create Stage 2 Topics**
```bash
# FHIR processing results topic
confluent kafka topic create fhir-processing-results.v1 \
  --partitions 6 \
  --config retention.ms=604800000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2

# Sink write failures topic
confluent kafka topic create sink-write-failures.v1 \
  --partitions 4 \
  --config retention.ms=1209600000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2
```

---

## 📊 **Topic Partitioning Strategy**

### **Partitioning Keys**
```yaml
raw-device-data.v1:
  key: device_id
  reason: "Ensures ordered processing per device"
  
validated-device-data.v1:
  key: device_id
  reason: "Maintains device ordering through pipeline"
  
failed-validation.v1:
  key: device_id
  reason: "Groups validation failures by device"
  
fhir-processing-results.v1:
  key: patient_id
  reason: "Groups FHIR resources by patient"
  
sink-write-failures.v1:
  key: sink_name
  reason: "Groups failures by sink for targeted retry"
```

### **Partition Count Rationale**
- **12 partitions** for high-throughput device data (raw + validated)
- **6 partitions** for FHIR results (lower volume, processed data)
- **4 partitions** for error/failure topics (low volume)

---

## 🔍 **Topic Monitoring Setup**

### **Key Metrics to Monitor**
```yaml
Topic Metrics:
  - kafka.topic.partition.under_replicated_partitions
  - kafka.topic.partition.in_sync_replicas
  - kafka.topic.bytes_in_per_sec
  - kafka.topic.messages_in_per_sec
  - kafka.topic.consumer_lag_sum

Consumer Group Metrics:
  - kafka.consumer.lag_sum
  - kafka.consumer.records_consumed_rate
  - kafka.consumer.fetch_rate
```

### **Alerting Thresholds**
```yaml
Critical Alerts:
  - Under-replicated partitions > 0
  - Consumer lag > 10,000 messages
  - Failed validation rate > 5%
  - Sink write failure rate > 1%

Warning Alerts:
  - Consumer lag > 5,000 messages
  - Topic size growth > 100MB/hour
  - Failed validation rate > 1%
```

---

## 🚀 **Topic Verification**

### **List Created Topics**
```bash
confluent kafka topic list | grep -E "(validated-device-data|failed-validation|fhir-processing-results|sink-write-failures)"
```

### **Describe Topic Configuration**
```bash
confluent kafka topic describe validated-device-data.v1
confluent kafka topic describe failed-validation.v1
confluent kafka topic describe fhir-processing-results.v1
confluent kafka topic describe sink-write-failures.v1
```

### **Test Topic Connectivity**
```bash
# Test producer
echo '{"test": "message"}' | confluent kafka topic produce validated-device-data.v1

# Test consumer
confluent kafka topic consume validated-device-data.v1 --from-beginning --max-messages 1
```

---

## 📋 **Next Steps**

1. ✅ **Create Topics**: Execute the Confluent Cloud commands above
2. ⏳ **Verify Topics**: Confirm all topics are created with correct configuration
3. ⏳ **Setup Monitoring**: Configure topic monitoring and alerting
4. ⏳ **Implement Stage 1**: Build Validator & Enricher service
5. ⏳ **Implement Stage 2**: Build Storage Fan-Out service

This topic architecture provides the foundation for our modular stream processing pipeline, ensuring proper data flow between stages while maintaining ordering and reliability guarantees.
