# 🚀 Quick Start Guide - Outbox Pattern
## Device Data Ingestion Service

### ⚡ **TL;DR - Get Running in 5 Minutes**

```bash
# 1. Start the service
python run_service.py

# 2. Test with Postman
POST http://localhost:8030/api/v1/ingest/device-data-smart

# 3. Check processing
GET http://localhost:8030/api/v1/outbox/queue-depths
```

---

## 🎯 **Essential Commands**

### **Start Service**
```bash
python run_service.py
# Service runs on http://localhost:8030
# Background publisher runs automatically every 2 seconds
```

### **Test Database**
```bash
python test_db_connection.py
```

### **Check Status**
```bash
# Queue depths
curl http://localhost:8030/api/v1/outbox/queue-depths

# Service health
curl http://localhost:8030/api/v1/health
```

---

## 📱 **Test Requests (Copy-Paste Ready)**

### **Fitbit Heart Rate**
```json
POST http://localhost:8030/api/v1/ingest/device-data-smart
Content-Type: application/json

{
  "device_id": "fitbit_test_001",
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

### **Medical Blood Pressure**
```json
POST http://localhost:8030/api/v1/ingest/device-data-smart
Content-Type: application/json

{
  "device_id": "omron_bp_001",
  "timestamp": 1735416000,
  "reading_type": "blood_pressure",
  "systolic": 120,
  "diastolic": 80,
  "unit": "mmHg",
  "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
  "metadata": {
    "medical_grade": true,
    "device_model": "Omron BP7000"
  }
}
```

### **Apple Watch ECG**
```json
POST http://localhost:8030/api/v1/ingest/device-data-smart
Content-Type: application/json

{
  "device_id": "apple_watch_001",
  "timestamp": 1735416000,
  "reading_type": "ecg",
  "waveform": [0.1, 0.3, 0.8, 0.2, -0.1],
  "heart_rate": 74,
  "unit": "mV",
  "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
  "metadata": {
    "vendor": "apple",
    "medical_grade": true,
    "device_model": "Apple Watch Series 9"
  }
}
```

---

## 🔍 **Expected Flow**

### **1. Send Request → Smart Detection**
```
Request → Universal Handler → Vendor Detection (e.g., "fitbit", 76% confidence)
```

### **2. Outbox Storage**
```
Vendor Routing → fitbit_outbox table → Status: 'pending'
```

### **3. Background Processing (Automatic)**
```
Background Publisher (every 2s) → Poll outbox → Kafka publish → Status: 'completed'
```

### **4. Verification**
```bash
# Check if processed
curl http://localhost:8030/api/v1/outbox/queue-depths
# Should show 0 pending messages if processed
```

---

## 📊 **Success Indicators**

### **In Service Logs:**
```
✅ Smart detection result: fitbit (0.76) via universal_handler
✅ Stored device data in fitbit_outbox
✅ Processing 1 pending messages across all vendors
✅ Successfully published message to Kafka
```

### **In API Response:**
```json
{
  "status": "accepted",
  "message": "Device data routed to Fitbit via universal_handler",
  "outbox_id": "uuid-here",
  "metadata": {
    "vendor_detection": {
      "vendor_id": "fitbit",
      "confidence": 0.76,
      "outbox_table": "fitbit_outbox"
    }
  }
}
```

---

## 🔧 **Common Issues & Fixes**

### **Database Connection Failed**
```bash
# Check connection
python test_db_connection.py

# Fix: Use pooled connection for Supabase free tier
DATABASE_URL=postgresql://postgres.PROJECT_ID:PASSWORD@aws-0-ap-south-1.pooler.supabase.com:5432/postgres
```

### **Messages Not Processing**
```bash
# Check queue
curl http://localhost:8030/api/v1/outbox/queue-depths

# Manual process (if needed)
python test_background_publisher.py
```

### **Service Won't Start**
```bash
# Check port availability
netstat -ano | findstr :8030

# Kill existing process if needed
taskkill /PID <process-id> /F
```

---

## 🎯 **Vendor Routing Logic**

| Device Type | Vendor Detected | Outbox Table |
|-------------|----------------|--------------|
| Fitbit devices | `fitbit` | `fitbit_outbox` |
| Garmin devices | `garmin` | `garmin_outbox` |
| Apple Health | `apple` | `apple_health_outbox` |
| Medical grade | `medical_device` | `medical_device_outbox` |
| Unknown/Generic | `generic_device` | `generic_device_outbox` |

---

## 📈 **Monitoring Endpoints**

```bash
# Service health
GET http://localhost:8030/api/v1/health

# Queue status
GET http://localhost:8030/api/v1/outbox/queue-depths

# Vendor detection test
POST http://localhost:8030/api/v1/vendors/detect

# Supported vendors
GET http://localhost:8030/api/v1/vendors/supported
```

---

## 🚀 **Production Checklist**

- ✅ Database connection working
- ✅ Kafka credentials configured
- ✅ Outbox tables created
- ✅ Background publisher running
- ✅ API endpoints responding
- ✅ Test requests successful
- ✅ Messages reaching Kafka

**Ready for ETL pipeline consumption!** 🎉

---

## 📞 **Need Help?**

1. **Check logs**: Service logs show detailed processing info
2. **Test connection**: `python test_db_connection.py`
3. **Manual processing**: `python test_background_publisher.py`
4. **Queue status**: `GET /api/v1/outbox/queue-depths`

**The outbox pattern ensures guaranteed delivery - no data loss even during failures!**
