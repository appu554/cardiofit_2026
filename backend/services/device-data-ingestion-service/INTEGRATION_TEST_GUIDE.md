# 🚀 Complete Integration Test Guide: Device Data → Outbox → Kafka

This guide shows you exactly how the outbox pattern is integrated into your device ingestion service and how to test the complete flow from device data ingestion to Kafka delivery.

## 🏗️ Architecture Overview

```
Device Data → API Endpoint → Vendor Detection → Outbox Table → Publisher Service → Kafka → ETL Pipeline
```

### Integration Points:

1. **Supabase PostgreSQL**: Stores outbox tables with vendor isolation
2. **Device Ingestion Service**: Three API endpoints for different use cases
3. **Publisher Service**: Background service that processes outbox → Kafka
4. **Kafka**: Final destination for processed device data

## 📋 Prerequisites

### 1. Install Dependencies
```bash
cd backend/services/device-data-ingestion-service
pip install asyncpg psycopg2-binary sqlalchemy[asyncio] httpx
```

### 2. Database Setup
```bash
# Run the migration to create outbox tables in Supabase
python run_migration.py
```

### 3. Environment Variables
Make sure these are set in your environment:
```bash
DATABASE_URL=postgresql://postgres:Cardiofit@123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres
KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092
KAFKA_API_KEY=LGJ3AQ2L6VRPW4S2
KAFKA_API_SECRET=your-kafka-secret
```

## 🧪 Step-by-Step Testing

### Step 1: Start the Device Ingestion Service

```bash
# Terminal 1: Start the main ingestion service
python run_service.py
```

The service will start on `http://localhost:8015` with these endpoints:

- **Legacy**: `POST /api/v1/ingest/device-data` (direct Kafka)
- **Outbox**: `POST /api/v1/ingest/device-data-outbox` (with API key)
- **Smart**: `POST /api/v1/ingest/device-data-smart` (auto-detection) ⭐

### Step 2: Start the Publisher Service (Optional)

```bash
# Terminal 2: Start the background publisher
python run_outbox_publisher.py
```

This processes messages from outbox tables → Kafka.

### Step 3: Run the Integration Tests

```bash
# Terminal 3: Run the complete end-to-end test
python test_end_to_end_kafka.py
```

## 🔍 What Each Test Does

### 1. Service Health Test
- Verifies the ingestion service is running
- Checks API endpoints are responding
- Validates basic connectivity

### 2. Legacy Endpoint Test
```bash
curl -X POST http://localhost:8015/api/v1/ingest/device-data \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-vendor-api-key-123" \
  -d '{
    "device_id": "fitbit_charge5_001",
    "reading_type": "heart_rate",
    "value": 75.5,
    "unit": "bpm",
    "timestamp": 1703123456,
    "patient_id": "test-patient-123"
  }'
```

### 3. Outbox Endpoint Test
```bash
curl -X POST http://localhost:8015/api/v1/ingest/device-data-outbox \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-vendor-api-key-123" \
  -d '{
    "device_id": "omron_bp7000_001",
    "reading_type": "blood_pressure",
    "systolic": 120,
    "diastolic": 80,
    "unit": "mmHg",
    "timestamp": 1703123456,
    "patient_id": "test-patient-123",
    "metadata": {"vendor": "omron", "medical_grade": true}
  }'
```

### 4. Smart Detection Test
```bash
curl -X POST http://localhost:8015/api/v1/ingest/device-data-smart \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "hospital_glucose_001",
    "reading_type": "blood_glucose",
    "value": 95.0,
    "unit": "mg/dL",
    "timestamp": 1703123456,
    "patient_id": "test-patient-123",
    "metadata": {"medical_grade": true, "hospital_device": true}
  }'
```

**Expected Response:**
```json
{
  "status": "accepted",
  "message": "Device data routed to Medical Device via medical_grade_routing",
  "ingestion_id": "uuid-here",
  "metadata": {
    "vendor_detection": {
      "vendor_id": "medical_device",
      "vendor_name": "Medical Device",
      "device_type": "blood_glucose",
      "confidence": 0.75,
      "is_medical_grade": true,
      "detection_method": "medical_grade_routing",
      "outbox_table": "medical_device_outbox"
    }
  }
}
```

### 5. Vendor Detection Test
```bash
curl http://localhost:8015/api/v1/vendors/supported
```

**Expected Response:**
```json
{
  "status": "success",
  "vendors": {
    "fitbit": {
      "supported_device_types": ["heart_rate", "steps", "sleep_duration", "weight"],
      "outbox_table": "fitbit_outbox",
      "is_medical_grade": false
    },
    "omron": {
      "supported_device_types": ["blood_pressure", "heart_rate", "weight"],
      "outbox_table": "omron_outbox", 
      "is_medical_grade": true
    },
    "medical_device": {
      "supported_device_types": ["ecg", "blood_pressure", "blood_glucose", "temperature", "oxygen_saturation", "heart_rate"],
      "outbox_table": "medical_device_outbox",
      "is_medical_grade": true
    }
  }
}
```

### 6. Outbox Queue Status
```bash
curl http://localhost:8015/api/v1/outbox/queue-depths
```

**Expected Response:**
```json
{
  "status": "success",
  "queue_depths": {
    "fitbit_outbox": {"pending": 5, "processing": 0, "failed": 0},
    "omron_outbox": {"pending": 2, "processing": 0, "failed": 0},
    "medical_device_outbox": {"pending": 1, "processing": 0, "failed": 0}
  }
}
```

## 🔄 Complete Flow Verification

### 1. Check Supabase Tables
```sql
-- Check if data is stored in outbox tables
SELECT vendor_id, COUNT(*) as pending_count 
FROM vendor_outbox_registry r
JOIN (
  SELECT 'fitbit' as vendor, COUNT(*) as cnt FROM fitbit_outbox WHERE status = 'pending'
  UNION ALL
  SELECT 'omron' as vendor, COUNT(*) as cnt FROM omron_outbox WHERE status = 'pending'
  UNION ALL  
  SELECT 'medical_device' as vendor, COUNT(*) as cnt FROM medical_device_outbox WHERE status = 'pending'
) counts ON r.vendor_id = counts.vendor
WHERE r.is_active = true;
```

### 2. Monitor Publisher Processing
```bash
# Check publisher health
curl http://localhost:8015/api/v1/outbox/health
```

### 3. Verify Kafka Delivery
The test will attempt to publish a test message to Kafka and verify connectivity.

## 🎯 Expected Test Results

When you run `python test_end_to_end_kafka.py`, you should see:

```
🚀 Starting End-to-End Kafka Integration Test
======================================================================

🔍 Testing service health...
✅ Service running: Device Data Ingestion Service v1.0.0
✅ API health check passed

🔍 Testing legacy direct Kafka endpoint...
✅ Legacy endpoint: success

🔍 Testing transactional outbox endpoint...
✅ Outbox endpoint: accepted - ID: uuid-here

🔍 Testing smart detection endpoint...
✅ Smart detection for fitbit_heart_rate:
   Vendor: fitbit
   Confidence: 0.95
   Method: explicit_metadata
✅ Smart detection for omron_blood_pressure:
   Vendor: omron
   Confidence: 0.95
   Method: explicit_metadata
✅ Smart detection for medical_glucose:
   Vendor: medical_device
   Confidence: 0.75
   Method: medical_grade_routing

🔍 Testing vendor detection capabilities...
✅ Found 7 supported vendors

🔍 Testing outbox queue status...
✅ Outbox queue depths:
   fitbit_outbox: 1 pending
   omron_outbox: 1 pending
   medical_device_outbox: 1 pending
   Total pending: 3

🔍 Testing publisher service...
✅ Publisher service is running
   Messages processed: 15
   Success rate: 100.00%

🔍 Testing Kafka integration...
✅ Kafka producer health check passed
✅ Test message published to Kafka: message-id-here

======================================================================
📊 END-TO-END KAFKA INTEGRATION TEST RESULTS
======================================================================

🏥 SERVICE HEALTH:
Service Health: ✅
API Health: ✅

📡 INGESTION ENDPOINTS:
Legacy Direct Kafka: ✅
Transactional Outbox: ✅

🧠 SMART DETECTION:
fitbit_heart_rate: ✅ → fitbit (0.95)
omron_blood_pressure: ✅ → omron (0.95)
medical_glucose: ✅ → medical_device (0.75)
apple_ecg: ✅ → medical_device (0.75)
unknown_steps: ✅ → generic_device (0.10)

📊 OUTBOX QUEUES:
Total Pending Messages: 3

🔄 PUBLISHER SERVICE:
Publisher Status: ✅ Running

📨 KAFKA INTEGRATION:
Kafka Connection: ✅
Test Message ID: message-id-here

======================================================================
📈 SUMMARY:
Core Tests: 6
Smart Detection Tests: 5
Total Device Types Tested: 5

🎉 END-TO-END INTEGRATION SUCCESSFUL!
✅ Device data can flow from ingestion → outbox → Kafka
======================================================================
```

## 🚨 Troubleshooting

### Common Issues:

1. **Database Connection Failed**
   - Check Supabase credentials
   - Ensure `asyncpg` is installed
   - Run migration: `python run_migration.py`

2. **Kafka Connection Failed**
   - Verify Kafka credentials in config
   - Check network connectivity to Confluent Cloud

3. **Publisher Not Running**
   - Start publisher service: `python run_outbox_publisher.py`
   - Check publisher health: `curl http://localhost:8015/api/v1/outbox/health`

4. **API Key Issues**
   - The test uses a dummy API key
   - For production, configure real API keys in the auth system

## 🎉 Success Criteria

✅ **Integration is working if:**
- All service health checks pass
- Smart detection correctly identifies device vendors
- Messages are stored in appropriate outbox tables
- Publisher service processes messages to Kafka
- Kafka connectivity is verified

This confirms your outbox pattern is fully integrated and ready for production! 🚀
