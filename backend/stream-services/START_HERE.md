# 🚀 START HERE: Complete Testing Guide
## Stage 1 & Stage 2 Services with Your Exact Configuration

### 📋 **What You Have**

✅ **Python Scripts Created** (Windows-compatible):
- `setup-kafka-topics.py` - Creates all required Kafka topics
- `run-stage1.py` - Launches Stage 1 (Validator & Enricher)
- `run-stage2.py` - Launches Stage 2 (Storage Fan-Out)
- `run-tests.py` - Test runner and monitoring
- `test-data-generator.py` - Sends test data

✅ **Your Exact Configuration Applied**:
- **Patient ID**: `905a60cb-8241-418f-b29b-5b020e851392`
- **Kafka**: Your Confluent Cloud credentials
- **FHIR Store**: `projects/cardiofit-905a8/.../fhir-store`
- **Elasticsearch**: `my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud`
- **MongoDB**: `cluster0.yqdzbvb.mongodb.net/clinical_synthesis_hub`

---

## ⚡ **Quick Start (5 Steps)**

### **Step 1: Setup Kafka Topics**
```powershell
python setup-kafka-topics.py
# Select option 5 (Full setup)
```

### **Step 2: Start Stage 1 (Terminal 1)**
```powershell
python run-stage1.py
# Select option 5 (Full setup and start)
```

### **Step 3: Start Stage 2 (Terminal 2)**
```powershell
python run-stage2.py
# Select option 6 (Full setup and start)
```

### **Step 4: Verify Services (Terminal 3)**
```powershell
python run-tests.py
# Select option 2 (Check service health)
```

### **Step 5: Send Test Data**
```powershell
python run-tests.py
# Select option 3 (Send test data)
```

---

## 🎯 **Expected Results**

### **✅ Successful Flow:**
```
Raw Device Data → Stage 1 (Validation + Enrichment) → Stage 2 (FHIR + Multi-Sink)
                                                    → FHIR Store (Google Healthcare API)
                                                    → Elasticsearch (Your Elastic Cloud)
                                                    → MongoDB (Your Atlas cluster)
```

### **📊 Test Data:**
- **Patient ID**: `905a60cb-8241-418f-b29b-5b020e851392`
- **Device Types**: Heart rate, blood pressure, glucose, temperature, oxygen saturation
- **Scenarios**: Normal, critical, emergency, invalid readings
- **FHIR Compliance**: Same transformations as your PySpark ETL

### **🔍 Monitoring:**
```powershell
# Health checks
curl http://localhost:8041/api/v1/health  # Stage 1
curl http://localhost:8042/api/v1/health  # Stage 2

# Sink health (your PySpark sinks)
curl http://localhost:8042/api/v1/health/sinks

# Metrics
curl http://localhost:8042/api/v1/metrics/summary
```

---

## 🛠️ **Prerequisites**

### **Required Software:**
- **Java 17+** (for Stage 1)
- **Maven 3.6+** (for Stage 1)
- **Python 3.11+** (for Stage 2 and scripts)

### **Check Prerequisites:**
```powershell
java -version    # Should show Java 17+
mvn -version     # Should show Maven 3.6+
python --version # Should show Python 3.11+
```

### **Optional (for full testing):**
- **Redis** (for patient context caching)
- **MongoDB** (local instance for testing)

---

## 📁 **Script Details**

### **1. setup-kafka-topics.py**
- Creates 8 Kafka topics for the complete pipeline
- Uses your Confluent Cloud credentials
- Handles topic existence gracefully
- Shows topic configurations

### **2. run-stage1.py**
- Builds and runs Java Spring Boot service
- Validates device readings using medical rules
- Enriches with patient context
- Routes to appropriate topics

### **3. run-stage2.py**
- Runs Python FastAPI service
- FHIR transformations (same as PySpark)
- Multi-sink writes to your exact sinks
- Comprehensive error handling

### **4. run-tests.py**
- End-to-end testing
- Health monitoring
- Metrics collection
- Troubleshooting guide

### **5. test-data-generator.py**
- Generates realistic medical device data
- Uses your patient ID
- Multiple test scenarios
- Sends to Kafka topics

---

## 🔧 **Troubleshooting**

### **Common Issues:**

#### **1. Java/Maven Issues:**
```powershell
# Check Java
java -version

# Install Java 17 if needed
# Download from: https://adoptium.net/

# Check Maven
mvn -version

# Install Maven if needed
# Download from: https://maven.apache.org/download.cgi
```

#### **2. Python Issues:**
```powershell
# Check Python
python --version

# Install required packages
pip install confluent-kafka requests uvicorn fastapi motor pymongo elasticsearch google-cloud-healthcare

# If pip fails, try:
python -m pip install --upgrade pip
```

#### **3. Kafka Connection Issues:**
- Verify your Confluent Cloud credentials
- Check network connectivity
- Ensure topics are created

#### **4. Service Won't Start:**
```powershell
# Stage 1 logs
# Look for errors in the Maven output

# Stage 2 logs
# Look for errors in the Python output

# Common fixes:
# - Check port availability (8041, 8042)
# - Verify environment variables
# - Check dependency installation
```

---

## 📊 **Success Indicators**

### **✅ Stage 1 Success:**
- Service starts on port 8041
- Health endpoint returns `{"status": "UP"}`
- Kafka Streams topology builds successfully
- Processes raw device data
- Validates using medical rules

### **✅ Stage 2 Success:**
- Service starts on port 8042
- Health endpoint returns `{"status": "UP"}`
- All sinks show healthy status
- FHIR transformations work
- Multi-sink writes succeed

### **✅ End-to-End Success:**
- Test data flows through both stages
- Valid data reaches your sinks (FHIR Store, Elasticsearch, MongoDB)
- Invalid data goes to DLQ topics
- Metrics show processing statistics
- No critical errors in logs

---

## 🎉 **What This Proves**

1. **✅ Same Business Logic**: Identical FHIR transformations as PySpark
2. **✅ Same Data Quality**: Medical validation and enrichment preserved
3. **✅ Same Sinks**: Writes to your exact PySpark destinations
4. **✅ Better Architecture**: Modular, scalable, fault-tolerant
5. **✅ Enhanced Monitoring**: Production-ready observability
6. **✅ Independent Scaling**: Scale validation and storage separately

---

## 🚀 **Ready to Start?**

1. **Open PowerShell** in the `backend/stream-services` directory
2. **Run**: `python setup-kafka-topics.py` (select option 5)
3. **Open 3 terminals** and follow the Quick Start steps above
4. **Watch the magic happen!** 🎯

Your modular architecture will demonstrate the **exact same functionality** as your monolithic PySpark reactor, but with **better reliability**, **independent scaling**, and **comprehensive error handling**!

**Need help?** Run `python run-tests.py` and select option 7 for troubleshooting! 🛠️
