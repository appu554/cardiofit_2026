# InfluxDB Projector Service

Time-series projector that consumes enriched EHR events from Kafka and writes vital signs data to InfluxDB with automatic downsampling.

## Overview

The InfluxDB Projector is part of the Module 8 streaming architecture:

```
Kafka Topic: prod.ehr.events.enriched
         ↓
    InfluxDBProjector (Filter VITAL_SIGNS events)
         ↓
    Extract vital signs → Create InfluxDB Points
         ↓
    Write to vitals_realtime bucket (7-day retention)
         ↓
    Automatic downsampling:
    - vitals_1min (90-day retention, 1-minute averages)
    - vitals_1hour (2-year retention, 1-hour averages)
```

## Features

- **High-Frequency Ingestion**: 10,000+ data points per second
- **Automatic Downsampling**: Flux tasks for 1-minute and 1-hour aggregation
- **Multi-Tier Retention**: 7 days raw → 90 days 1-min → 2 years 1-hour
- **Tag-Based Filtering**: Efficient queries by patient, device, department
- **Batch Processing**: Configurable batch size and flush intervals
- **Retry Logic**: Exponential backoff for failed writes

## Data Model

### Measurements

#### heart_rate
```
Tags: patient_id, device_id, department_id
Fields: value (float)
```

#### blood_pressure
```
Tags: patient_id, device_id, department_id
Fields: systolic (float), diastolic (float)
```

#### spo2
```
Tags: patient_id, device_id, department_id
Fields: value (float)
```

#### temperature
```
Tags: patient_id, device_id, department_id
Fields: value (float)
```

## Configuration

Create `.env` file (copy from `.env.example`):

```bash
# InfluxDB Configuration
INFLUXDB_URL=http://localhost:8086
INFLUXDB_ORG=cardiofit
INFLUXDB_TOKEN=your_influxdb_token_here

# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS=pkc-9q8rv.ap-south-2.aws.confluent.cloud:9092
KAFKA_SASL_USERNAME=your_kafka_key
KAFKA_SASL_PASSWORD=your_kafka_secret
KAFKA_CONSUMER_GROUP=influxdb-projector-group
```

## Installation

```bash
# Install dependencies
pip install -r requirements.txt

# Install shared module in development mode
cd ../module8-shared
pip install -e .
cd ../module8-influxdb-projector
```

## Usage

### Start Service

```bash
python run_service.py
```

Or with uvicorn directly:

```bash
uvicorn main:app --host 0.0.0.0 --port 8054
```

### Verify Setup

```bash
# Check health
curl http://localhost:8054/health

# Get statistics
curl http://localhost:8054/stats

# List buckets
curl http://localhost:8054/buckets
```

## API Endpoints

### GET /health
Health check with InfluxDB and Kafka status

**Response:**
```json
{
  "status": "healthy",
  "service": "influxdb-projector",
  "influxdb_status": "pass",
  "kafka_status": "running",
  "statistics": {
    "total_events_processed": 1500,
    "vitals_written": 4200,
    "heart_rate_count": 1500,
    "blood_pressure_count": 1400,
    "spo2_count": 700,
    "temperature_count": 600
  }
}
```

### GET /stats
Detailed projector statistics

### GET /buckets
InfluxDB bucket information and retention policies

### POST /reset
Reset statistics counters

## InfluxDB Buckets

### vitals_realtime
- **Retention**: 7 days
- **Purpose**: High-frequency raw data points
- **Downsampled to**: vitals_1min

### vitals_1min
- **Retention**: 90 days
- **Purpose**: 1-minute averaged data
- **Source**: vitals_realtime (aggregateWindow every 1m)
- **Downsampled to**: vitals_1hour

### vitals_1hour
- **Retention**: 2 years
- **Purpose**: 1-hour averaged data for long-term trends
- **Source**: vitals_1min (aggregateWindow every 1h)

## Downsampling Tasks

Automatically created Flux tasks:

### downsample_1min (runs every 1 minute)
```flux
from(bucket: "vitals_realtime")
    |> range(start: -2m)
    |> aggregateWindow(every: 1m, fn: mean)
    |> to(bucket: "vitals_1min")
```

### downsample_1hour (runs every 1 hour)
```flux
from(bucket: "vitals_1min")
    |> range(start: -2h)
    |> aggregateWindow(every: 1h, fn: mean)
    |> to(bucket: "vitals_1hour")
```

## Query Examples

### Recent Heart Rate (Real-time)
```flux
from(bucket: "vitals_realtime")
    |> range(start: -1h)
    |> filter(fn: (r) => r["_measurement"] == "heart_rate")
    |> filter(fn: (r) => r["patient_id"] == "P12345")
```

### Daily Trends (1-minute aggregates)
```flux
from(bucket: "vitals_1min")
    |> range(start: -24h)
    |> filter(fn: (r) => r["_measurement"] == "blood_pressure")
    |> filter(fn: (r) => r["department_id"] == "ICU_01")
```

### Long-term Analysis (1-hour aggregates)
```flux
from(bucket: "vitals_1hour")
    |> range(start: -30d)
    |> filter(fn: (r) => r["_measurement"] == "temperature")
    |> filter(fn: (r) => r["patient_id"] == "P12345")
```

## Performance Tuning

### Batch Settings

```python
# High throughput (default)
INFLUXDB_BATCH_SIZE=200
INFLUXDB_FLUSH_INTERVAL=5000  # 5 seconds

# Low latency
INFLUXDB_BATCH_SIZE=50
INFLUXDB_FLUSH_INTERVAL=1000  # 1 second

# High volume
INFLUXDB_BATCH_SIZE=500
INFLUXDB_FLUSH_INTERVAL=10000  # 10 seconds
```

### Kafka Consumer Settings

```python
# Increase throughput
KAFKA_MAX_POLL_RECORDS=500

# Reduce latency
KAFKA_MAX_POLL_RECORDS=50
```

## Monitoring

### Key Metrics

- **vitals_written**: Total data points written to InfluxDB
- **heart_rate_count**: Number of heart rate measurements
- **blood_pressure_count**: Number of BP measurements
- **errors**: Write failures and processing errors

### Logs

```bash
# Follow logs
tail -f logs/influxdb-projector.log

# Filter errors
grep ERROR logs/influxdb-projector.log
```

## Troubleshooting

### InfluxDB Connection Failed

```bash
# Verify InfluxDB is running
curl http://localhost:8086/health

# Check token permissions
# Token needs read/write access to buckets and tasks
```

### No Data Written

```bash
# Check Kafka consumer is running
curl http://localhost:8054/health

# Verify enriched topic has VITAL_SIGNS events
# Check stats endpoint for non_vital_skipped count

# Validate InfluxDB bucket exists
curl http://localhost:8054/buckets
```

### High Memory Usage

- Reduce `INFLUXDB_BATCH_SIZE`
- Decrease `INFLUXDB_FLUSH_INTERVAL`
- Lower `KAFKA_MAX_POLL_RECORDS`

## Architecture

```
┌─────────────────────────────────────────────────┐
│         Kafka: prod.ehr.events.enriched         │
└─────────────────────┬───────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────┐
│          InfluxDBProjector Consumer             │
│  - Filter VITAL_SIGNS events                    │
│  - Extract heart_rate, BP, SpO2, temp           │
│  - Create tagged Points                         │
│  - Batch write with retry                       │
└─────────────────────┬───────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────┐
│            InfluxDB Time-Series DB              │
│                                                 │
│  vitals_realtime (7d)                           │
│       ↓ (downsample every 1m)                   │
│  vitals_1min (90d)                              │
│       ↓ (downsample every 1h)                   │
│  vitals_1hour (2y)                              │
└─────────────────────────────────────────────────┘
```

## License

Part of the CardioFit Clinical Synthesis Hub Platform
