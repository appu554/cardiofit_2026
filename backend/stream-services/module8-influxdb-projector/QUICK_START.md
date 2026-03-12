# InfluxDB Projector - Quick Start Guide

## 1-Minute Setup

### Prerequisites
- InfluxDB running on port 8086 (container: cardiofit-influxdb)
- Kafka cluster accessible
- Python 3.9+

### Start Service

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services/module8-influxdb-projector

# Install dependencies
python3 -m pip install -r requirements.txt

# Start service (uses .env configuration)
python3 run_service.py
```

Service will start on **port 8054**

---

## Quick Verification

### 1. Check Health
```bash
curl http://localhost:8054/health
```

Expected: `"status": "healthy"`

### 2. View Statistics
```bash
curl http://localhost:8054/stats
```

Watch `vitals_written` counter increase as events are processed.

### 3. Verify Buckets
```bash
curl http://localhost:8054/buckets
```

Should show 3 buckets: `vitals_realtime`, `vitals_1min`, `vitals_1hour`

---

## Data Flow

```
Kafka Topic
    ↓
Filter VITAL_SIGNS events
    ↓
Extract vitals (HR, BP, SpO2, Temp)
    ↓
Write to InfluxDB vitals_realtime
    ↓
Auto-downsample to vitals_1min (every 1m)
    ↓
Auto-downsample to vitals_1hour (every 1h)
```

---

## Query Examples

### Recent Heart Rate (Last Hour)
```bash
docker exec cardiofit-influxdb influx query \
  'from(bucket: "vitals_realtime")
   |> range(start: -1h)
   |> filter(fn: (r) => r["_measurement"] == "heart_rate")
   |> filter(fn: (r) => r["patient_id"] == "P12345")' \
  --org cardiofit
```

### Blood Pressure Trends (24 Hours)
```bash
docker exec cardiofit-influxdb influx query \
  'from(bucket: "vitals_1min")
   |> range(start: -24h)
   |> filter(fn: (r) => r["_measurement"] == "blood_pressure")
   |> filter(fn: (r) => r["department_id"] == "ICU_01")' \
  --org cardiofit
```

---

## Key Configuration

**File**: `.env`

```bash
# InfluxDB
INFLUXDB_URL=http://localhost:8086
INFLUXDB_TOKEN=cardiofit-influx-token-123456
INFLUXDB_ORG=cardiofit

# Kafka
KAFKA_TOPIC=prod.ehr.events.enriched
KAFKA_CONSUMER_GROUP=influxdb-projector-group

# Service
SERVICE_PORT=8054
```

---

## Monitoring

### Watch Live Stats
```bash
watch -n 2 'curl -s http://localhost:8054/stats | python3 -m json.tool'
```

### Key Metrics
- `vitals_written`: Total data points written
- `heart_rate_count`: Heart rate measurements
- `blood_pressure_count`: BP measurements
- `errors`: Write failures (should be 0)

---

## Troubleshooting

### Service Won't Start

1. **Check InfluxDB**:
   ```bash
   docker ps | grep influxdb
   ```

2. **Verify Token**:
   ```bash
   docker exec cardiofit-influxdb influx auth list
   ```

3. **Check Logs**:
   ```bash
   # Service logs will show connection errors
   ```

### No Data Appearing

1. **Verify Kafka Consumer**:
   ```bash
   curl http://localhost:8054/stats
   # Check total_events_processed > 0
   ```

2. **Check Event Type**:
   - Only VITAL_SIGNS events are processed
   - Check `non_vital_skipped` counter

3. **Query InfluxDB Directly**:
   ```bash
   docker exec cardiofit-influxdb influx query \
     'from(bucket: "vitals_realtime") |> range(start: -1m)' \
     --org cardiofit
   ```

---

## Architecture

```
┌─────────────────────────┐
│  Kafka: enriched topic  │
└───────────┬─────────────┘
            │
            ↓
┌─────────────────────────┐
│  InfluxDBProjector:8054 │
│  - Filter VITAL_SIGNS   │
│  - Extract vitals       │
│  - Batch write (200/5s) │
└───────────┬─────────────┘
            │
            ↓
┌─────────────────────────┐
│  InfluxDB:8086          │
│  ├─ vitals_realtime (7d)│
│  ├─ vitals_1min (90d)   │
│  └─ vitals_1hour (2y)   │
└─────────────────────────┘
```

---

## Production Checklist

- [ ] InfluxDB container running and healthy
- [ ] Correct token in `.env` file
- [ ] Kafka credentials configured
- [ ] Service health endpoint returns "healthy"
- [ ] Buckets created with correct retention
- [ ] Downsampling tasks active
- [ ] Monitoring/alerting configured

---

## Support

**Documentation**: See `README.md` for comprehensive guide
**Verification**: Run `python3 test_influxdb_setup.py`
**Status Report**: See `SETUP_COMPLETE.md`
