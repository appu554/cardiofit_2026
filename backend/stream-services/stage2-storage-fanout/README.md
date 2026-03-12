# 💾 Stage 2: Storage Fan-Out Service
## Multi-Sink Writer with FHIR Transformations

### 📋 **Overview**

This is **Stage 2** of our modular stream processing architecture that **replaces the monolithic Spark reactor**. This dedicated Python service handles:

1. **FHIR Transformations** - EXACT same logic as PySpark `create_fhir_observation_from_device_data_impl()`
2. **Multi-Sink Writes** - Parallel writes to FHIR Store + Elasticsearch + MongoDB
3. **Collect-Then-Dispatch Pattern** - Same pattern as PySpark ETL pipeline
4. **Independent Error Handling** - Per-sink failures with DLQ integration

### 🎯 **Core Responsibility**

**ONLY** consume "validated" events and handle complex sink writes. **NO** validation, **NO** enrichment - those are Stage 1's job. **Focus**: Reliable, parallel persistence to multiple storage systems.

---

## 🏗️ **Architecture**

### **Input/Output Flow**
```
Validated Device Data (validated-device-data.v1)
         ↓
   FHIR Transformation (same as PySpark)
         ↓
   ┌─────────────────┬─────────────────┬─────────────────┐
   ↓                 ↓                 ↓                 ↓
FHIR Store      Elasticsearch      MongoDB         DLQ Topics
(FHIR Obs)      (UI Document)      (Raw Data)      (Failures)
```

### **Technology Stack**
- **Framework**: Python FastAPI + Kafka Consumer
- **Language**: Python 3.11+
- **Port**: 8042
- **Dependencies**: Kafka, Google Healthcare API, Elasticsearch, MongoDB

---

## 🔄 **EXACT Same Business Logic as PySpark**

### **FHIR Transformations Preserved**
```python
# PySpark Implementation (Current):
from business_logic.transformations import create_fhir_observation_from_device_data_impl

@udf(StringType())
def create_fhir_observation_from_device_data(device_id, timestamp, reading_type, value, unit, patient_id, metadata, vendor_info):
    return create_fhir_observation_from_device_data_impl(device_id, timestamp, reading_type, value, unit, patient_id, metadata, vendor_info)

# Stage 2 Implementation (New):
from app.services.fhir_transformation import FHIRTransformationService

class FHIRTransformationService:
    def create_fhir_observation_from_device_data(self, device_data: Dict[str, Any]) -> str:
        # EXACT same business logic as PySpark implementation
        return create_fhir_observation_from_device_data_impl(
            device_data.get('device_id'), device_data.get('timestamp'), 
            device_data.get('reading_type'), device_data.get('value'),
            device_data.get('unit'), device_data.get('patient_id'),
            device_data.get('metadata'), device_data.get('vendor_info')
        )
```

### **Multi-Sink Pattern Preserved**
```python
# PySpark Implementation (Current):
# Parallel writes using ThreadPoolExecutor
futures = [
    executor.submit(write_to_fhir_store, fhir_observation),
    executor.submit(write_to_elasticsearch, ui_document),
    executor.submit(write_to_mongodb, raw_data)
]

# Stage 2 Implementation (New):
# EXACT same Collect-Then-Dispatch pattern
futures = [
    self.executor.submit(self.write_to_fhir_store, fhir_observation),
    self.executor.submit(self.write_to_elasticsearch, ui_document),
    self.executor.submit(self.write_to_mongodb, raw_data)
]
```

---

## 🔧 **Configuration**

### **Environment Variables**
```bash
# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092
KAFKA_API_KEY=LGJ3AQ2L6VRPW4S2
KAFKA_API_SECRET=your-secret
KAFKA_INPUT_TOPIC=validated-device-data.v1
KAFKA_DLQ_TOPIC=sink-write-failures.v1

# Google Healthcare API (FHIR Store)
GOOGLE_CLOUD_PROJECT=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=asia-south1
GOOGLE_CLOUD_DATASET=clinical-synthesis-hub
GOOGLE_CLOUD_FHIR_STORE=fhir-store
GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json

# Elasticsearch
ELASTICSEARCH_URL=https://my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud:443
ELASTICSEARCH_API_KEY=d0gyTG5aY0JGajhWTVBOTzkzeDk6VGxoNENEd29DZEtERXBxRXpRUXBEUQ==

# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=clinical_synthesis_hub
MONGODB_COLLECTION=device_readings_raw

# Multi-Sink Configuration
PARALLEL_WRITES=true
THREAD_POOL_SIZE=6
SINK_TIMEOUT_SECONDS=30
```

### **Kafka Topics**
```yaml
Input:  validated-device-data.v1       # From Stage 1 (Validator & Enricher)
DLQ:    sink-write-failures.v1         # Sink write failures
DLQ:    critical-sink-failures.v1      # Critical data failures
DLQ:    poison-messages-stage2.v1      # Repeated failures
```

---

## 🚀 **Running the Service**

### **Local Development**
```bash
# 1. Install dependencies
pip install -r requirements.txt

# 2. Set environment variables
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/gcp-credentials.json
export ELASTICSEARCH_URL=http://localhost:9200
export MONGODB_URI=mongodb://localhost:27017

# 3. Run the application
python -m uvicorn app.main:app --host 0.0.0.0 --port 8042 --reload
```

### **Production Deployment**
```bash
# Build Docker image
docker build -t stage2-storage-fanout:1.0.0 .

# Run container
docker run -d \
  --name stage2-storage-fanout \
  -p 8042:8042 \
  -e KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092 \
  -e KAFKA_API_KEY=LGJ3AQ2L6VRPW4S2 \
  -e KAFKA_API_SECRET=your-secret \
  -e GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json \
  -v /path/to/gcp-credentials.json:/app/credentials.json:ro \
  stage2-storage-fanout:1.0.0
```

---

## 🔍 **Monitoring & Health Checks**

### **Health Endpoints**
```bash
# Overall health
curl http://localhost:8042/api/v1/health

# Kafka consumer health
curl http://localhost:8042/api/v1/health/kafka

# Multi-sink writer health
curl http://localhost:8042/api/v1/health/sinks

# DLQ service health
curl http://localhost:8042/api/v1/health/dlq

# FHIR transformer health
curl http://localhost:8042/api/v1/health/fhir-transformer
```

### **Metrics Endpoints**
```bash
# Overall metrics
curl http://localhost:8042/api/v1/metrics

# Kafka metrics
curl http://localhost:8042/api/v1/metrics/kafka

# Sink metrics
curl http://localhost:8042/api/v1/metrics/sinks

# DLQ metrics
curl http://localhost:8042/api/v1/metrics/dlq

# Performance metrics
curl http://localhost:8042/api/v1/metrics/performance

# Prometheus metrics
curl http://localhost:8042/api/v1/metrics/prometheus
```

### **Key Metrics**
- **Kafka Consumer**: Messages processed, consumer lag, success rate
- **FHIR Transformations**: Transformation success rate, error types
- **Multi-Sink Writes**: Per-sink success rates, write latencies
- **DLQ**: Failed messages by category, retry attempts
- **Circuit Breakers**: Sink availability, failure thresholds

---

## 🛡️ **Error Handling & DLQ**

### **Error Categories**
1. **FHIR Transformation Failures**: Invalid LOINC codes, malformed resources
2. **Sink Write Failures**: Connection errors, authentication issues, quota limits
3. **Circuit Breaker Failures**: Service unavailability protection
4. **Timeout Failures**: Network latency, resource contention
5. **Poison Messages**: Repeatedly failing messages

### **DLQ Topics**
- `sink-write-failures.v1`: General sink write failures
- `critical-sink-failures.v1`: Critical medical data failures
- `poison-messages-stage2.v1`: Repeated processing failures

### **Recovery Mechanisms**
- **Retry Logic**: Exponential backoff with configurable limits
- **Circuit Breakers**: Per-sink failure protection
- **Independent Failures**: One sink failure doesn't affect others
- **Manual Recovery**: DLQ message replay capabilities

---

## 📊 **Performance Characteristics**

### **Throughput Targets**
- **15,000 writes/second** per sink (FHIR Store, Elasticsearch, MongoDB)
- **Sub-500ms latency** for FHIR transformation + multi-sink write
- **99%+ success rate** for sink writes under normal conditions

### **Resource Requirements**
- **CPU**: 4 cores minimum, 8 cores recommended
- **Memory**: 4GB minimum, 8GB recommended
- **Network**: High bandwidth to sinks (Google Cloud, Elasticsearch, MongoDB)

---

## 🔄 **Integration with Other Services**

### **Upstream Dependencies**
- **Stage 1: Validator & Enricher** (Port 8041) - Publishes validated device data

### **Downstream Sinks**
- **Google Healthcare API FHIR Store** - FHIR Observation resources
- **Elasticsearch** - UI-optimized documents for dashboards
- **MongoDB** - Raw device data backup and historical analysis

### **External Dependencies**
- **Confluent Cloud Kafka** - Message streaming
- **Google Cloud Healthcare API** - FHIR Store
- **Elasticsearch Cloud** - Search and analytics
- **MongoDB Atlas** - Document storage

---

## 🔧 **Troubleshooting**

### **Common Issues**

1. **FHIR Store Authentication Errors**
   ```bash
   # Check Google Cloud credentials
   gcloud auth application-default print-access-token
   
   # Verify FHIR Store permissions
   curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store
   ```

2. **Elasticsearch Connection Issues**
   ```bash
   # Test Elasticsearch connectivity
   curl -H "Authorization: ApiKey d0gyTG5aY0JGajhWTVBOTzkzeDk6VGxoNENEd29DZEtERXBxRXpRUXBEUQ==" \
     https://my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud:443/_cluster/health
   ```

3. **High DLQ Message Rate**
   ```bash
   # Check DLQ metrics
   curl http://localhost:8042/api/v1/metrics/dlq
   
   # Review error patterns
   kafka-console-consumer --bootstrap-server localhost:9092 \
     --topic sink-write-failures.v1 --from-beginning
   ```

This service provides the **exact same FHIR transformations and multi-sink functionality** as your PySpark pipeline, but with **better error handling**, **independent scaling**, and **production-ready monitoring**.
