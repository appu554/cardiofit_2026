# 🔍 Stage 1: Validator & Enricher Service
## Lightweight Kafka Streams Application for Medical Data Validation

### 📋 **Overview**

This is **Stage 1** of our modular stream processing architecture that **replaces the monolithic Spark reactor**. This dedicated Kafka Streams application handles:

1. **Medical Data Validation** - Same validation rules as PySpark pipeline
2. **Patient Context Enrichment** - Redis-cached patient information
3. **Intelligent Routing** - Valid data to Stage 2, invalid data to DLQ
4. **Real-time Processing** - Lightweight, fast validation without bottlenecks

### 🎯 **Core Responsibility**

**ONLY** consume raw events, validate them, enrich from Redis cache, and publish clean "validated" events. **NO** complex processing, **NO** sink writes, **NO** analytics - those are other stages' jobs.

---

## 🏗️ **Architecture**

### **Input/Output Flow**
```
Raw Device Data (raw-device-data.v1)
         ↓
   Medical Validation
         ↓
   Patient Context Enrichment (Redis Cache)
         ↓
   ┌─────────────────┬─────────────────┐
   ↓                 ↓                 ↓
Valid Data      Invalid Data      Critical Data
(validated-     (failed-          (priority
device-data.v1) validation.v1)    processing)
```

### **Technology Stack**
- **Framework**: Kafka Streams + Spring Boot
- **Language**: Java 17
- **Port**: 8041
- **Dependencies**: Kafka, Redis, Patient Service

---

## 🔧 **Configuration**

### **Environment Variables**
```bash
# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092
KAFKA_API_KEY=LGJ3AQ2L6VRPW4S2
KAFKA_API_SECRET=your-secret

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Patient Service Configuration
PATIENT_SERVICE_URL=http://patient-service:8003/api/v1/patient
```

### **Kafka Topics**
```yaml
Input:  raw-device-data.v1          # From Global Outbox Service
Output: validated-device-data.v1    # To Stage 2 (Storage Fan-Out)
DLQ:    failed-validation.v1        # Invalid data for review
```

---

## 🚀 **Running the Service**

### **Local Development**
```bash
# 1. Start dependencies
docker-compose up redis

# 2. Set environment variables
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
export REDIS_HOST=localhost
export PATIENT_SERVICE_URL=http://localhost:8003/api/v1/patient

# 3. Run the application
mvn spring-boot:run -Dspring-boot.run.profiles=dev
```

### **Production Deployment**
```bash
# Build the application
mvn clean package

# Run with production profile
java -jar target/stage1-validator-enricher-1.0.0.jar --spring.profiles.active=prod
```

### **Docker Deployment**
```bash
# Build Docker image
docker build -t stage1-validator-enricher:1.0.0 .

# Run container
docker run -d \
  --name stage1-validator-enricher \
  -p 8041:8041 \
  -e KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092 \
  -e KAFKA_API_KEY=LGJ3AQ2L6VRPW4S2 \
  -e KAFKA_API_SECRET=your-secret \
  -e REDIS_HOST=redis \
  -e PATIENT_SERVICE_URL=http://patient-service:8003/api/v1/patient \
  stage1-validator-enricher:1.0.0
```

---

## 🔍 **Monitoring & Health Checks**

### **Health Endpoints**
```bash
# Overall health
curl http://localhost:8041/api/v1/health

# Validation service health
curl http://localhost:8041/api/v1/health/validation

# Patient context service health
curl http://localhost:8041/api/v1/health/patient-context

# Spring Boot Actuator
curl http://localhost:8041/actuator/health
curl http://localhost:8041/actuator/metrics
curl http://localhost:8041/actuator/kafka-streams
```

### **Key Metrics**
- **Kafka Streams Lag**: Consumer lag per partition
- **Validation Rate**: Messages validated per second
- **Cache Hit Rate**: Redis cache hit percentage
- **Error Rate**: Failed validations per minute

---

## 🧪 **Testing**

### **Unit Tests**
```bash
mvn test
```

### **Integration Tests**
```bash
mvn verify -P integration-tests
```

### **Manual Testing**
```bash
# Send test message to input topic
kafka-console-producer --bootstrap-server localhost:9092 --topic raw-device-data.v1
{"device_id":"test-001","timestamp":1703123456,"reading_type":"heart_rate","value":75,"unit":"bpm","patient_id":"patient-123"}

# Check output topic
kafka-console-consumer --bootstrap-server localhost:9092 --topic validated-device-data.v1 --from-beginning
```

---

## 📊 **Performance Characteristics**

### **Throughput Targets**
- **25,000 messages/second** validation processing
- **Sub-100ms latency** for validation + enrichment
- **95%+ cache hit rate** for patient context

### **Resource Requirements**
- **CPU**: 2 cores minimum, 4 cores recommended
- **Memory**: 2GB minimum, 4GB recommended
- **Network**: Low latency to Kafka and Redis

---

## 🔄 **Integration with Other Services**

### **Upstream Dependencies**
- **Global Outbox Service** (Port 8040) - Publishes raw device data
- **Device Data Ingestion Service** (Port 8015) - Original data source

### **Downstream Consumers**
- **Stage 2: Storage Fan-Out Service** (Port 8042) - Consumes validated data
- **Monitoring Services** - Consume DLQ data for alerting

### **External Dependencies**
- **Redis Cache** - Patient context caching
- **Patient Service** (Port 8003) - Patient information API
- **Confluent Cloud Kafka** - Message streaming

---

## 🛡️ **Error Handling**

### **Validation Failures**
- Invalid JSON → Dead Letter Queue
- Missing required fields → Dead Letter Queue
- Out-of-range values → Dead Letter Queue with alert

### **Enrichment Failures**
- Redis cache miss → Fallback to Patient Service API
- Patient Service timeout → Continue without enrichment
- Critical medical data → Always process regardless of enrichment

### **Circuit Breaker**
- **Failure Threshold**: 10 consecutive failures
- **Recovery Timeout**: 60 seconds
- **Medical Data Bypass**: Critical data always processes

---

## 🔧 **Troubleshooting**

### **Common Issues**

1. **High Consumer Lag**
   ```bash
   # Check Kafka Streams metrics
   curl http://localhost:8041/actuator/kafka-streams
   
   # Increase stream threads
   spring.kafka.streams.properties.num.stream.threads=4
   ```

2. **Redis Connection Issues**
   ```bash
   # Test Redis connectivity
   redis-cli -h localhost -p 6379 ping
   
   # Check Redis health
   curl http://localhost:8041/api/v1/health/patient-context
   ```

3. **Patient Service Timeouts**
   ```bash
   # Check Patient Service health
   curl http://patient-service:8003/health
   
   # Increase timeout
   app.patient-service.timeout=10000ms
   ```

This service is the foundation of our modular architecture - lightweight, focused, and optimized for real-time medical data validation and enrichment.
