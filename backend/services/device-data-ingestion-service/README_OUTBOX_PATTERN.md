# 📦 Transactional Outbox Pattern Implementation
## Device Data Ingestion Service

### 🎯 **Overview**

This service implements a **production-ready Transactional Outbox Pattern** for reliable medical device data ingestion with guaranteed delivery to Kafka. The pattern ensures **ACID compliance**, **fault tolerance**, and **vendor isolation** for enterprise-grade healthcare data processing.

---

## 🏗️ **Architecture**

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   API Request   │───▶│  Smart Detection │───▶│ Vendor Routing  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Background      │◀───│ Transactional    │◀───│ Outbox Storage  │
│ Publisher       │    │ Outbox Tables    │    │ (Per Vendor)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │
         ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Kafka Producer  │───▶│ Confluent Cloud  │───▶│ ETL Pipeline    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

---

## 🚀 **Key Features**

### ✅ **Enterprise-Grade Reliability**
- **ACID Transactions**: Guaranteed data consistency
- **Fault Tolerance**: Automatic retry with exponential backoff
- **Vendor Isolation**: Separate outbox tables per vendor
- **Guaranteed Delivery**: No data loss even during failures

### ✅ **Smart Device Detection**
- **Universal Device Handler**: Automatic vendor detection
- **Confidence Scoring**: ML-based device classification
- **Fallback Routing**: Generic handling for unknown devices
- **Medical Grade Classification**: Automatic medical device identification

### ✅ **Performance Optimizations**
- **Adaptive Batching**: Dynamic batch size optimization
- **Connection Pooling**: Efficient database connections
- **Background Processing**: Non-blocking async operations
- **Monitoring & Metrics**: Real-time observability

---

## 📋 **Database Schema**

### **Vendor Registry**
```sql
vendor_outbox_registry (
    vendor_id VARCHAR(100) PRIMARY KEY,
    vendor_name VARCHAR(255) NOT NULL,
    outbox_table_name VARCHAR(255) NOT NULL,
    kafka_topic VARCHAR(255) DEFAULT 'raw-device-data.v1',
    max_retries INT DEFAULT 3,
    is_active BOOLEAN DEFAULT true
)
```

### **Vendor-Specific Outbox Tables**
```sql
{vendor}_outbox (
    id UUID PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    correlation_id UUID,
    trace_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0
)
```

**Supported Vendors:**
- `fitbit_outbox` - Fitbit devices
- `garmin_outbox` - Garmin devices  
- `apple_health_outbox` - Apple Health devices
- `medical_device_outbox` - Medical grade devices
- `generic_device_outbox` - Unknown/generic devices

---

## 🔧 **Configuration**

### **Environment Variables**
```bash
# Database (Supabase)
DATABASE_URL=postgresql://postgres.auugxeqzgrnknklgwqrh:PASSWORD@aws-0-ap-south-1.pooler.supabase.com:5432/postgres

# Kafka (Confluent Cloud)
KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092
KAFKA_API_KEY=LGJ3AQ2L6VRPW4S2
KAFKA_API_SECRET=your-secret

# Service
PORT=8030
ENVIRONMENT=development
```

### **Supabase Setup**
1. **Create outbox tables**: Run `migrations/001_create_outbox_tables.sql`
2. **Disable RLS**: For service access to outbox tables
3. **Connection pooling**: Use pooled connection for free tier

---

## 🚀 **Quick Start**

### **1. Install Dependencies**
```bash
pip install -r requirements.txt
```

### **2. Configure Database**
```bash
# Update config.py with your Supabase credentials
DATABASE_URL=postgresql://postgres.PROJECT_ID:PASSWORD@aws-0-ap-south-1.pooler.supabase.com:5432/postgres
```

### **3. Run Database Migration**
```bash
python run_migration.py
```

### **4. Start Service**
```bash
python run_service.py
```

**Service starts on:** `http://localhost:8030`

---

## 📡 **API Endpoints**

### **Smart Device Data Ingestion**
```http
POST /api/v1/ingest/device-data-smart
Content-Type: application/json

{
  "device_id": "fitbit_charge5_001",
  "timestamp": 1735416000,
  "reading_type": "heart_rate",
  "value": 75,
  "unit": "bpm",
  "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
  "metadata": {
    "vendor": "fitbit",
    "device_model": "Charge 5",
    "battery_level": 85
  }
}
```

**Response:**
```json
{
  "status": "accepted",
  "message": "Device data routed to Fitbit via universal_handler",
  "ingestion_id": "correlation-id",
  "outbox_id": "uuid",
  "metadata": {
    "vendor_detection": {
      "vendor_id": "fitbit",
      "confidence": 0.76,
      "detection_method": "universal_handler",
      "outbox_table": "fitbit_outbox"
    }
  }
}
```

### **Queue Status**
```http
GET /api/v1/outbox/queue-depths
```

### **Vendor Detection**
```http
POST /api/v1/vendors/detect
```

---

## 🔄 **Data Flow**

### **1. Ingestion Phase**
```
API Request → Smart Detection → Vendor Routing → Outbox Storage
```

### **2. Processing Phase**
```
Background Publisher (every 2s) → Poll Outbox → Kafka Publishing → Status Update
```

### **3. Message States**
- `pending` - Awaiting processing
- `processing` - Currently being processed
- `completed` - Successfully published to Kafka
- `failed` - Failed after max retries

---

## 🧪 **Testing**

### **Test Database Connection**
```bash
python test_db_connection.py
```

### **Test Background Publisher**
```bash
python test_background_publisher.py
```

### **Test Complete Flow**
```bash
python test_complete_flow.py
```

### **Sample Test Requests**

**Fitbit Device:**
```json
{
  "device_id": "fitbit_test_001",
  "reading_type": "heart_rate",
  "value": 75,
  "unit": "bpm",
  "patient_id": "test-patient-id",
  "metadata": {"vendor": "fitbit"}
}
```

**Medical Device:**
```json
{
  "device_id": "omron_bp_001",
  "reading_type": "blood_pressure",
  "systolic": 120,
  "diastolic": 80,
  "unit": "mmHg",
  "patient_id": "test-patient-id",
  "metadata": {"medical_grade": true}
}
```

---

## 📊 **Monitoring**

### **Service Health**
```http
GET /api/v1/health
```

### **Queue Depths**
```http
GET /api/v1/outbox/queue-depths
```

### **Metrics**
- Message processing rate
- Vendor detection confidence
- Queue depths per vendor
- Error rates and retry counts

### **Logs**
```bash
# Service logs
tail -f logs/device-data-ingestion.log

# Background publisher logs
tail -f background_publisher.log
```

---

## 🔧 **Troubleshooting**

### **Common Issues**

**1. Database Connection Failed**
```bash
# Check Supabase connection
python test_db_connection.py

# Verify pooled connection for free tier
DATABASE_URL=postgresql://postgres.PROJECT_ID:PASSWORD@aws-0-ap-south-1.pooler.supabase.com:5432/postgres
```

**2. Messages Not Processing**
```bash
# Check queue depths
curl http://localhost:8030/api/v1/outbox/queue-depths

# Manual processing
python test_background_publisher.py
```

**3. Kafka Connection Issues**
```bash
# Verify Kafka credentials
KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092
KAFKA_API_KEY=your-key
```

### **Performance Tuning**

**Background Publisher:**
- `poll_interval`: Adjust processing frequency (default: 2s)
- `batch_size`: Messages per batch (default: 50)
- `max_retries`: Retry attempts (default: 3)

**Database:**
- `pool_size`: Connection pool size (default: 10)
- `max_overflow`: Additional connections (default: 20)

---

## 🏆 **Production Deployment**

### **Environment Setup**
```bash
# Production environment
ENVIRONMENT=production
LOG_LEVEL=INFO
ENABLE_METRICS=true

# Database
DATABASE_URL=postgresql://postgres.PROJECT_ID:PASSWORD@aws-0-ap-south-1.pooler.supabase.com:5432/postgres

# Kafka
KAFKA_BOOTSTRAP_SERVERS=your-production-kafka
```

### **Scaling Considerations**
- **Horizontal Scaling**: Multiple service instances
- **Database Sharding**: Partition by vendor_id
- **Kafka Partitioning**: Distribute load across partitions
- **Monitoring**: Prometheus + Grafana dashboards

---

## 📚 **Related Documentation**

- [TRANSACTIONAL_OUTBOX_PATTERN_IMPLEMENTATION_PLAN.md](./TRANSACTIONAL_OUTBOX_PATTERN_IMPLEMENTATION_PLAN.md)
- [API Documentation](./docs/api.md)
- [Database Schema](./docs/schema.md)
- [Deployment Guide](./docs/deployment.md)

---

## ✅ **Implementation Status**

- ✅ **Phase 1**: Foundation Setup - COMPLETE
- ✅ **Phase 2**: Outbox Tables - COMPLETE  
- ✅ **Phase 3**: Smart Detection - COMPLETE
- ✅ **Phase 4**: Transactional Storage - COMPLETE
- ✅ **Phase 5**: Vendor Isolation - COMPLETE
- ✅ **Phase 6**: Background Publishing - COMPLETE
- ✅ **Phase 7**: End-to-End Kafka - COMPLETE

**🎉 Production Ready!**
