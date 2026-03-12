# Device Data Ingestion Service

A high-performance, secure FastAPI service for ingesting device data from medical devices and wearables as part of the Clinical Synthesis Hub's event-driven architecture.

## Overview

This service serves as the "front door" for device data in the event-driven architecture implementation. It provides:

- **Secure API Key Authentication**: Each device vendor gets unique API keys with specific permissions
- **Rate Limiting**: Per-vendor and per-device rate limiting to prevent abuse
- **Real-time Processing**: Immediate publishing to Kafka for downstream ETL processing
- **Batch Support**: Efficient batch ingestion for high-volume scenarios
- **FHIR Compliance**: Data is structured for downstream FHIR Observation resource creation
- **Monitoring**: Built-in health checks and metrics endpoints

## Architecture

```
Device → Vendor System → Ingestion Service → Kafka → Declarative ETL Pipeline → FHIR Store + Elasticsearch
```

This service is part of Phase 2 of the Event-Driven Architecture implementation, specifically the "Continuum Write Path" vertical slice.

## Features

### Supported Device Types

- Heart Rate Monitors (`heart_rate`)
- Blood Pressure Monitors (`blood_pressure_systolic`, `blood_pressure_diastolic`)
- Blood Glucose Meters (`blood_glucose`)
- Temperature Sensors (`temperature`)
- Pulse Oximeters (`oxygen_saturation`)
- Smart Scales (`weight`)
- Activity Trackers (`steps`)
- Sleep Monitors (`sleep_duration`)
- Respiratory Rate Monitors (`respiratory_rate`)

### Authentication & Authorization

- API key-based authentication via `X-API-Key` header
- Vendor-specific permissions for device types
- Rate limiting per vendor and per device

### Rate Limiting

- Default: 1000 requests/minute per vendor
- Default: 100 requests/minute per device
- Configurable limits per vendor

## Installation

1. **Install Dependencies**:
   ```bash
   cd backend/services/device-data-ingestion-service
   pip install -r requirements.txt
   ```

2. **Configure Environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your Kafka credentials
   ```

3. **Run the Service**:
   ```bash
   python run_service.py
   ```

The service will start on `http://localhost:8015`

## API Endpoints

### POST /api/v1/ingest/device-data

Ingest a single device reading.

**Headers**:
```
X-API-Key: your-vendor-api-key
Content-Type: application/json
```

**Request Body**:
```json
{
  "device_id": "device-12345",
  "timestamp": 1703123456,
  "reading_type": "heart_rate",
  "value": 72.5,
  "unit": "bpm",
  "patient_id": "patient-67890",
  "metadata": {
    "battery_level": 85,
    "signal_quality": "good"
  }
}
```

**Response**:
```json
{
  "status": "accepted",
  "message": "Data queued for processing successfully",
  "ingestion_id": "uuid-here",
  "timestamp": "2023-12-21T10:30:56.123456"
}
```

### POST /api/v1/ingest/batch-device-data

Ingest multiple device readings in a single request (max 100).

**Request Body**:
```json
[
  {
    "device_id": "device-12345",
    "timestamp": 1703123456,
    "reading_type": "heart_rate",
    "value": 72.5,
    "unit": "bpm"
  },
  {
    "device_id": "device-12346", 
    "timestamp": 1703123457,
    "reading_type": "blood_glucose",
    "value": 95.0,
    "unit": "mg/dL"
  }
]
```

### GET /api/v1/health

Health check endpoint.

**Response**:
```json
{
  "status": "healthy",
  "service": "Device Data Ingestion Service",
  "version": "1.0.0",
  "timestamp": "2023-12-21T10:30:56.123456",
  "kafka_connected": true,
  "dependencies": {
    "kafka": "healthy"
  }
}
```

### GET /api/v1/metrics

Service metrics endpoint.

## Configuration

Key configuration options in `.env`:

```env
# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS="pkc-619z3.us-east1.gcp.confluent.cloud:9092"
KAFKA_API_KEY="your-api-key"
KAFKA_API_SECRET="your-api-secret"
KAFKA_TOPIC_DEVICE_DATA="raw-device-data.v1"

# Rate Limiting
RATE_LIMIT_PER_MINUTE=1000
RATE_LIMIT_PER_DEVICE_PER_MINUTE=100

# Service
PORT=8015
DEBUG=false
```

## Testing

### Test API Keys

For development/testing, the following API keys are pre-configured:

- **Vendor 1**: `dv1_test_key_12345`
  - Allowed types: `heart_rate`, `blood_pressure`, `blood_glucose`
  - Rate limit: 1000/minute

- **Vendor 2**: `dv2_test_key_67890`
  - Allowed types: `temperature`, `oxygen_saturation`, `weight`
  - Rate limit: 500/minute

### Example cURL Request

```bash
curl -X POST "http://localhost:8015/api/v1/ingest/device-data" \
  -H "X-API-Key: dv1_test_key_12345" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "test-device-001",
    "timestamp": 1703123456,
    "reading_type": "heart_rate",
    "value": 75.0,
    "unit": "bpm"
  }'
```

## Integration with ETL Pipeline

This service publishes data to the `raw-device-data.v1` Kafka topic with the following structure:

```json
{
  "data": {
    "device_id": "device-12345",
    "timestamp": 1703123456,
    "reading_type": "heart_rate",
    "value": 72.5,
    "unit": "bpm",
    "patient_id": "patient-67890",
    "metadata": {},
    "vendor_info": {
      "vendor_id": "device-vendor-1",
      "vendor_name": "Test Device Vendor 1"
    }
  },
  "metadata": {
    "ingestion_timestamp": "2023-12-21T10:30:56.123456",
    "service": "device-data-ingestion-service",
    "version": "1.0.0"
  }
}
```

This data is then consumed by the Declarative ETL Pipeline for transformation into FHIR Observation resources and Elasticsearch documents.

## Monitoring

- Health check endpoint: `/api/v1/health`
- Metrics endpoint: `/api/v1/metrics`
- Structured logging with correlation IDs
- Kafka producer health monitoring

## Security

- API key authentication with vendor-specific permissions
- Rate limiting to prevent abuse
- Input validation and sanitization
- Secure Kafka connection with SASL/SSL
- No sensitive data logging

## Production Considerations

1. **API Key Management**: Implement proper API key management system
2. **Rate Limiting**: Use Redis for distributed rate limiting
3. **Monitoring**: Integrate with Prometheus/Grafana
4. **Logging**: Use structured logging with correlation IDs
5. **Security**: Implement additional security measures (IP whitelisting, etc.)
6. **Scaling**: Deploy multiple instances behind load balancer
