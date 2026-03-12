# 🛡️ Dead Letter Queue (DLQ) Implementation
## Comprehensive Error Handling for Modular Stream Processing

### 📋 **Overview**

This document outlines the comprehensive Dead Letter Queue (DLQ) implementation for both **Stage 1 (Validator & Enricher)** and **Stage 2 (Storage Fan-Out)** services. The DLQ system provides robust error handling, categorization, and recovery mechanisms for failed message processing.

---

## 🎯 **DLQ Architecture**

### **Stage 1: Validator & Enricher DLQ**
```
Raw Device Data → Validation → Enrichment
                      ↓              ↓
                 [Validation     [Enrichment
                  Failures]       Failures]
                      ↓              ↓
                failed-validation.v1 Topic
                      ↓
                critical-data-dlq.v1 (Critical Data)
                      ↓
                poison-messages.v1 (Repeated Failures)
```

### **Stage 2: Storage Fan-Out DLQ**
```
Validated Data → FHIR Transform → Multi-Sink Write
                      ↓                    ↓
                [Transform           [Sink Write
                 Failures]            Failures]
                      ↓                    ↓
                sink-write-failures.v1 Topic
                      ↓
                critical-sink-failures.v1 (Critical Data)
                      ↓
                poison-messages-stage2.v1 (Repeated Failures)
```

---

## 🔧 **DLQ Topics Structure**

### **Stage 1 Topics**
```yaml
failed-validation.v1:
  description: "Validation failures and parsing errors"
  partitions: 4
  retention: 30d
  
critical-data-dlq.v1:
  description: "Critical medical data that failed processing"
  partitions: 4
  retention: 90d
  
poison-messages.v1:
  description: "Messages that repeatedly fail processing"
  partitions: 2
  retention: 365d
```

### **Stage 2 Topics**
```yaml
sink-write-failures.v1:
  description: "Failed writes to FHIR Store, Elasticsearch, MongoDB"
  partitions: 6
  retention: 14d
  
critical-sink-failures.v1:
  description: "Critical medical data sink write failures"
  partitions: 4
  retention: 90d
  
poison-messages-stage2.v1:
  description: "Messages that repeatedly fail sink writes"
  partitions: 2
  retention: 365d
```

---

## 🚨 **Error Categories & Routing**

### **Stage 1: Validation Errors**

#### **1. Validation Failures**
```java
// Medical validation failures
DLQRecord record = createValidationFailureRecord(
    originalData, validationResult, deviceId, error
);

// Route based on criticality
String topic = record.isCriticalData() ? 
    CRITICAL_DATA_DLQ_TOPIC : FAILED_VALIDATION_TOPIC;
```

**Error Types:**
- Invalid physiological ranges (heart rate < 40 or > 150)
- Missing required fields (device_id, timestamp, value)
- Data type validation failures
- Medical threshold violations

#### **2. Parsing Failures**
```java
// JSON parsing errors
dlqService.sendParsingFailure(originalData, key, error);
```

**Error Types:**
- Malformed JSON
- Missing required JSON fields
- Invalid data types
- Encoding issues

#### **3. Enrichment Failures**
```java
// Patient context enrichment errors
dlqService.sendEnrichmentFailure(originalData, deviceId, patientId, error);
```

**Error Types:**
- Redis cache failures
- Patient Service API timeouts
- Invalid patient data
- Network connectivity issues

#### **4. Poison Messages**
```java
// Messages that repeatedly fail
dlqService.sendPoisonMessage(originalData, key, reason, retryCount);
```

**Criteria:**
- 3+ consecutive validation failures
- Repeated parsing errors
- Persistent enrichment failures

### **Stage 2: Storage Errors**

#### **1. FHIR Transformation Failures**
```python
await dlq_service.send_fhir_transformation_failure(
    original_data, error, device_id
)
```

**Error Types:**
- Invalid FHIR resource structure
- Missing LOINC codes
- JSON serialization errors
- Business logic failures

#### **2. Sink Write Failures**
```python
await dlq_service.send_sink_write_failure(
    original_data, sink_name, error, device_id, is_critical
)
```

**Error Types:**
- **FHIR Store**: Authentication, quota limits, invalid resources
- **Elasticsearch**: Connection errors, mapping conflicts, index failures
- **MongoDB**: Connection timeouts, write conflicts, storage limits

#### **3. Circuit Breaker Failures**
```python
await dlq_service.send_circuit_breaker_failure(
    original_data, sink_name, device_id
)
```

**Triggers:**
- 10+ consecutive sink failures
- Service unavailability
- Network partitions

#### **4. Timeout Failures**
```python
await dlq_service.send_timeout_failure(
    original_data, sink_name, timeout_seconds, device_id
)
```

**Scenarios:**
- Sink write timeouts (>30 seconds)
- Network latency issues
- Resource contention

---

## 📊 **DLQ Record Structure**

### **Standard DLQ Record**
```json
{
  "original_data": { /* Original message data */ },
  "error_type": "VALIDATION_FAILURE",
  "error_message": "Heart rate value 200 exceeds critical threshold",
  "device_id": "device-001",
  "patient_id": "patient-123",
  "failure_timestamp": 1703123456,
  "failure_datetime": "2023-12-21T10:30:56Z",
  "processing_stage": "stage1-validator-enricher",
  "is_critical_data": true,
  "retryable": true,
  "max_retries": 3,
  "retry_count": 0,
  "alert_level": "critical",
  "error_details": {
    "exception_type": "ValidationException",
    "exception_message": "Value exceeds critical range",
    "validation_rules_version": "1.0.0",
    "physiological_range": "40-150 bpm"
  },
  "dlq_version": "1.0"
}
```

---

## 🔄 **Retry Logic & Recovery**

### **Retry Strategy**
```yaml
Retryable Errors:
  - Validation failures: 3 retries
  - Enrichment failures: 2 retries  
  - Sink write failures: 3 retries
  - Circuit breaker: 5 retries (after recovery)
  - Timeout failures: 2 retries

Non-Retryable Errors:
  - Parsing failures (malformed JSON)
  - Poison messages
  - Authentication failures
  - Invalid FHIR resources
```

### **Exponential Backoff**
```python
retry_delay = base_delay * (backoff_factor ** retry_count)
# Example: 1s, 2s, 4s, 8s, 16s
```

### **Critical Data Bypass**
```java
// Critical medical data always processes
if (deviceReading.isCriticalMedicalData()) {
    // Bypass circuit breakers
    // Reduce retry delays
    // Escalate to manual review
}
```

---

## 📈 **Monitoring & Alerting**

### **Key Metrics**
```yaml
Stage 1 Metrics:
  - total_dlq_messages
  - validation_failures
  - critical_data_failures
  - poison_messages
  - enrichment_failures

Stage 2 Metrics:
  - total_dlq_messages
  - sink_failures
  - fhir_failures
  - elasticsearch_failures
  - mongodb_failures
  - transformation_failures
```

### **Alert Thresholds**
```yaml
Critical Alerts:
  - Critical medical data failures > 0
  - Poison message rate > 1%
  - All sinks failing simultaneously
  - DLQ topic lag > 1000 messages

Warning Alerts:
  - Validation failure rate > 5%
  - Sink failure rate > 10%
  - Enrichment failure rate > 20%
  - Circuit breaker open > 5 minutes
```

### **Health Check Endpoints**
```bash
# Stage 1 DLQ Health
curl http://localhost:8041/api/v1/health/dlq

# Stage 2 DLQ Health  
curl http://localhost:8042/api/v1/health/dlq

# DLQ Metrics
curl http://localhost:8042/api/v1/metrics/dlq
```

---

## 🛠️ **DLQ Management Operations**

### **Manual DLQ Processing**
```bash
# Consume DLQ messages for review
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic failed-validation.v1 \
  --from-beginning

# Replay DLQ messages after fixing issues
kafka-console-producer --bootstrap-server localhost:9092 \
  --topic raw-device-data.v1 < fixed_messages.json
```

### **DLQ Analytics**
```sql
-- Example queries for DLQ analysis
SELECT error_type, COUNT(*) as failure_count
FROM dlq_messages 
WHERE failure_timestamp > NOW() - INTERVAL '1 day'
GROUP BY error_type;

SELECT device_id, COUNT(*) as failure_count
FROM dlq_messages
WHERE is_critical_data = true
GROUP BY device_id
ORDER BY failure_count DESC;
```

### **Automated Recovery**
```python
# DLQ processor for automated recovery
class DLQProcessor:
    async def process_dlq_messages(self):
        for message in dlq_consumer:
            if self.is_recoverable(message):
                await self.retry_processing(message)
            elif self.requires_manual_review(message):
                await self.escalate_to_ops(message)
```

---

## 🔍 **Troubleshooting Guide**

### **Common DLQ Scenarios**

1. **High Validation Failures**
   - Check device data quality
   - Verify medical validation rules
   - Review physiological thresholds

2. **FHIR Store Write Failures**
   - Check Google Cloud credentials
   - Verify FHIR Store permissions
   - Monitor quota limits

3. **Elasticsearch Connection Issues**
   - Verify Elasticsearch cluster health
   - Check API key validity
   - Monitor index capacity

4. **Circuit Breaker Activation**
   - Check downstream service health
   - Review failure thresholds
   - Monitor recovery timeouts

### **DLQ Recovery Procedures**

1. **Identify Root Cause**
   - Analyze DLQ error patterns
   - Check service health
   - Review recent deployments

2. **Fix Underlying Issues**
   - Update validation rules
   - Fix service configurations
   - Resolve infrastructure problems

3. **Replay Failed Messages**
   - Extract recoverable messages
   - Apply fixes to message format
   - Replay through original topics

4. **Monitor Recovery**
   - Watch DLQ message rates
   - Verify successful processing
   - Update alerting thresholds

This comprehensive DLQ implementation ensures **robust error handling**, **data integrity**, and **operational visibility** across the entire modular stream processing pipeline.
