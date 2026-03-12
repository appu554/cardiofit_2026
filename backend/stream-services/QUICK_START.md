# 🚀 Quick Start Guide: Stage 1 & Stage 2 Testing

## 📋 **Prerequisites**

Before running the tests, ensure you have:

1. **Java 17+** (for Stage 1)
2. **Maven 3.6+** (for Stage 1)
3. **Python 3.11+** (for Stage 2)
4. **Redis** (running on localhost:6379)
5. **MongoDB** (running on localhost:27017)
6. **Kafka Access** (Confluent Cloud credentials)

---

## ⚡ **Quick Start (5 Minutes)**

### **Step 1: Configuration Ready! ✅**
```bash
# ✅ All Kafka credentials are already configured with your settings:
# - API Key: LGJ3AQ2L6VRPW4S2
# - Bootstrap Server: pkc-619z3.us-east1.gcp.confluent.cloud:9092
# - Resource: lkc-x86njx
# No manual configuration needed!
```

### **Step 2: Quick Setup (2 Minutes)**
```bash
# Make scripts executable
chmod +x start-services.sh

# Run the quick setup
./start-services.sh
# Select option 6 (Full setup)
```

### **Step 3: Start Services**

**Terminal 1 - Stage 1:**
```bash
./start-services.sh
# Select option 4 (Start Stage 1)
# Credentials are automatically configured!
```

**Terminal 2 - Stage 2:**
```bash
./start-services.sh
# Select option 5 (Start Stage 2)
# Credentials are automatically configured!
```

### **Step 4: Verify Services**
```bash
# Check Stage 1 health
curl http://localhost:8041/api/v1/health

# Check Stage 2 health
curl http://localhost:8042/api/v1/health
```

### **Step 5: Send Test Data**
```bash
# Terminal 3 - Send test data
python3 test-data-generator.py
# Select scenario 5 (mixed) and send 10 messages
```

---

## 🔍 **Monitoring & Verification**

### **Health Checks**
```bash
# Stage 1 Health
curl http://localhost:8041/api/v1/health
curl http://localhost:8041/api/v1/health/validation
curl http://localhost:8041/api/v1/health/patient-context

# Stage 2 Health
curl http://localhost:8042/api/v1/health
curl http://localhost:8042/api/v1/health/kafka
curl http://localhost:8042/api/v1/health/sinks
curl http://localhost:8042/api/v1/health/dlq
```

### **Metrics**
```bash
# Stage 1 Metrics
curl http://localhost:8041/actuator/metrics
curl http://localhost:8041/actuator/kafka-streams

# Stage 2 Metrics
curl http://localhost:8042/api/v1/metrics
curl http://localhost:8042/api/v1/metrics/kafka
curl http://localhost:8042/api/v1/metrics/sinks
curl http://localhost:8042/api/v1/metrics/dlq
```

### **Kafka Topic Monitoring**
```bash
# Monitor validated data (Stage 1 output)
confluent kafka topic consume validated-device-data.v1 --from-beginning

# Monitor validation failures (Stage 1 DLQ)
confluent kafka topic consume failed-validation.v1 --from-beginning

# Monitor sink failures (Stage 2 DLQ)
confluent kafka topic consume sink-write-failures.v1 --from-beginning
```

---

## 🧪 **Test Scenarios**

### **1. Normal Flow Test**
```bash
# Send normal readings
python3 test-data-generator.py
# Select scenario 1 (normal), send 5 messages

# Expected: Messages flow through Stage 1 → Stage 2 → MongoDB
```

### **2. Validation Failure Test**
```bash
# Send invalid readings
python3 test-data-generator.py
# Select scenario 4 (invalid), send 3 messages

# Expected: Messages fail in Stage 1 → failed-validation.v1 topic
```

### **3. Critical Data Test**
```bash
# Send critical readings
python3 test-data-generator.py
# Select scenario 3 (emergency), send 2 messages

# Expected: Messages processed with high priority flags
```

### **4. Mixed Scenario Test**
```bash
# Send mixed readings
python3 test-data-generator.py
# Select scenario 5 (mixed), send 20 messages

# Expected: Mix of successful processing and DLQ routing
```

---

## 📊 **Expected Results**

### **Successful Processing Flow:**
```
Raw Device Data → Stage 1 (Validation + Enrichment) → Stage 2 (FHIR + Multi-Sink)
                                                    → MongoDB (Raw Data)
```

### **Validation Failure Flow:**
```
Invalid Data → Stage 1 (Validation Fails) → failed-validation.v1 (DLQ)
```

### **Sink Failure Flow:**
```
Valid Data → Stage 1 → Stage 2 (Sink Write Fails) → sink-write-failures.v1 (DLQ)
```

---

## 🔧 **Troubleshooting**

### **Common Issues:**

1. **Stage 1 Won't Start**
   - Check Java version: `java -version`
   - Check Maven: `mvn -version`
   - Check Kafka credentials in application-dev.yml

2. **Stage 2 Won't Start**
   - Check Python version: `python3 --version`
   - Install dependencies: `pip install -r requirements.txt`
   - Check environment variables in .env.dev

3. **Kafka Connection Issues**
   - Verify Confluent Cloud credentials
   - Check network connectivity
   - Ensure topics are created

4. **Redis Connection Issues**
   - Start Redis: `redis-server`
   - Check connection: `redis-cli ping`

5. **MongoDB Connection Issues**
   - Start MongoDB: `mongod`
   - Check connection: `mongo --eval "db.runCommand('ping')"`

### **Logs Location:**
- **Stage 1**: Console output + `logs/stage1-validator-enricher.log`
- **Stage 2**: Console output (structured JSON logs)

### **Debug Commands:**
```bash
# Check Kafka consumer lag
confluent kafka consumer group describe stage1-validator-enricher
confluent kafka consumer group describe stage2-storage-fanout-dev

# Check topic messages
confluent kafka topic consume raw-device-data.v1 --from-beginning --max-messages 5
confluent kafka topic consume validated-device-data.v1 --from-beginning --max-messages 5
```

---

## 🎯 **Success Criteria**

✅ **Stage 1 Success:**
- Service starts on port 8041
- Health checks pass
- Consumes from `raw-device-data.v1`
- Produces to `validated-device-data.v1`
- Invalid data routed to `failed-validation.v1`

✅ **Stage 2 Success:**
- Service starts on port 8042
- Health checks pass
- Consumes from `validated-device-data.v1`
- FHIR transformations work
- Data written to MongoDB
- Failures routed to `sink-write-failures.v1`

✅ **End-to-End Success:**
- Test data flows through both stages
- Valid data reaches MongoDB
- Invalid data reaches appropriate DLQ topics
- Metrics show processing statistics
- No critical errors in logs

---

## 📞 **Need Help?**

If you encounter issues:
1. Check the logs for error messages
2. Verify all prerequisites are installed
3. Ensure Kafka credentials are correct
4. Check that Redis and MongoDB are running
5. Review the troubleshooting section above

The services are designed to be resilient and provide detailed error messages to help with debugging.
