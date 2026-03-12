# Google FHIR Healthcare API Integration for KB7 Terminology Service

This integration provides seamless connectivity between the KB7 Terminology Service and Google Cloud Healthcare API, enabling hybrid terminology operations across PostgreSQL, GraphDB, and Google FHIR stores.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   FastAPI       │    │   Hybrid Query   │    │   Google FHIR       │
│   Endpoints     │◄──►│   Router         │◄──►│   Healthcare API    │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
         │                       │                        │
         ▼                       ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Local FHIR    │    │   PostgreSQL     │    │   Google Cloud      │
│   Client        │    │   + GraphDB      │    │   FHIR Store        │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
```

## 📁 Files Created

### Core Integration Files

1. **`google_config.py`** - Configuration management for Google Cloud Healthcare API
2. **`google_fhir_terminology_client.py`** - Direct Google FHIR API client
3. **`google_fhir_service.py`** - Hybrid service combining Google FHIR with local KB7 query router
4. **`requirements.txt`** - Enhanced dependencies including Google Cloud Healthcare API
5. **`endpoints.py`** - Updated FastAPI endpoints with Google FHIR integration

### Configuration and Testing

6. **`.env.google-fhir.example`** - Example environment configuration
7. **`test_google_integration.py`** - Comprehensive integration test suite

## 🚀 Setup Instructions

### 1. Install Dependencies

```bash
cd /path/to/kb-7-terminology/fhir
pip install -r requirements.txt
```

### 2. Configure Google Cloud Credentials

#### Option A: Service Account Key File
```bash
# Download service account key from Google Cloud Console
cp /path/to/your-service-account-key.json credentials/google-credentials.json
```

#### Option B: Environment Variable
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

### 3. Set Up Environment Variables

```bash
cp .env.google-fhir.example .env
# Edit .env with your specific configuration
```

Required variables:
```env
GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=asia-south1
GOOGLE_CLOUD_DATASET_ID=clinical-synthesis-hub
GOOGLE_CLOUD_FHIR_STORE_ID=fhir-store
GOOGLE_CLOUD_CREDENTIALS_PATH=credentials/google-credentials.json
```

### 4. Verify Setup

```bash
python test_google_integration.py
```

## 🔧 Usage Examples

### CodeSystem $lookup with Intelligent Routing

```bash
# Prefer Google FHIR for official terminologies
curl "http://localhost:8000/fhir/CodeSystem/\$lookup?system=http://snomed.info/sct&code=73211009&prefer_source=google"

# Use local store for custom terminologies
curl "http://localhost:8000/fhir/CodeSystem/\$lookup?system=http://example.com/custom&code=CUSTOM_001&prefer_source=local"

# Let the system decide optimal routing
curl "http://localhost:8000/fhir/CodeSystem/\$lookup?system=http://loinc.org&code=33747-0"
```

### ValueSet $expand with Fallback

```bash
# Expand official FHIR ValueSet
curl "http://localhost:8000/fhir/ValueSet/\$expand?url=http://hl7.org/fhir/ValueSet/administrative-gender"

# Expand with filtering
curl "http://localhost:8000/fhir/ValueSet/\$expand?url=http://hl7.org/fhir/ValueSet/condition-clinical&filter=active"
```

### Health Check and Monitoring

```bash
# Check overall service health
curl "http://localhost:8000/fhir/terminology/health"

# Get Google FHIR specific statistics
curl "http://localhost:8000/fhir/terminology/google-stats"
```

## 🎯 Key Features

### Intelligent Routing

The hybrid service automatically determines the optimal backend for each request:

- **Google FHIR**: Official terminologies (SNOMED CT, LOINC, ICD-10)
- **Local PostgreSQL**: Fast exact lookups and custom terminologies
- **GraphDB**: Semantic reasoning and complex relationships

### Automatic Fallback

If the preferred service fails:
1. Attempt primary service (Google or local)
2. Fall back to alternative service
3. Return results with metadata about source and fallback usage

### Response Metadata

All responses include operational metadata:

```json
{
  "resourceType": "Parameters",
  "parameter": [...],
  "_metadata": {
    "source": "google",
    "latency_ms": 45,
    "cached": false,
    "fallback_used": false
  }
}
```

### Caching and Performance

- Redis-based caching for frequent operations
- Connection pooling for Google FHIR API
- Circuit breaker patterns for resilience
- Configurable timeout and retry logic

## 🔒 Security Configuration

### Required IAM Permissions

The service account needs these Google Cloud IAM roles:

```yaml
# Minimum required permissions
roles:
  - roles/healthcare.fhirResourceReader
  - roles/healthcare.fhirResourceEditor  # If write operations needed

# Specific permissions
permissions:
  - healthcare.fhirStores.searchResources
  - healthcare.fhirResources.create
  - healthcare.fhirResources.read
  - healthcare.fhirResources.update
  - healthcare.fhirResources.search
```

### Security Best Practices

1. **Credentials Management**
   - Store service account keys securely
   - Use Google Secret Manager in production
   - Rotate credentials regularly

2. **Network Security**
   - Use VPC Service Controls
   - Enable audit logging
   - Monitor API usage and quotas

3. **Access Control**
   - Apply principle of least privilege
   - Use resource-level IAM policies
   - Enable Cloud Healthcare API audit logs

## 🔍 Monitoring and Observability

### Health Check Endpoints

```bash
# Overall service health
GET /fhir/terminology/health

# Response includes status of all components:
{
  "healthy": true,
  "overall_status": "healthy",
  "services": {
    "google_fhir_hybrid": {"status": "healthy", ...},
    "local_terminology": {"status": "healthy", ...}
  }
}
```

### Performance Metrics

```bash
# Google FHIR specific metrics
GET /fhir/terminology/google-stats

# Response includes:
{
  "hybrid_service": {
    "google_requests": 150,
    "local_requests": 75,
    "fallback_count": 5,
    "cache_hit_ratio": 0.85
  },
  "google_fhir": {
    "request_count": 150,
    "success_rate": 0.97,
    "cache_hits": 45
  }
}
```

### Logging

Enable comprehensive logging for debugging:

```env
GOOGLE_FHIR_LOG_LEVEL=DEBUG
GOOGLE_FHIR_DEBUG_LOGGING=true
```

## 🧪 Testing

### Integration Tests

Run the comprehensive test suite:

```bash
python test_google_integration.py
```

Tests include:
- Configuration validation
- Health checks
- FHIR operations (lookup, expand, validate)
- Fallback mechanisms
- Performance benchmarks
- Statistics collection

### Manual Testing

Test individual components:

```python
import asyncio
from google_fhir_service import create_hybrid_service
from models import CodeSystemLookupRequest

async def test_lookup():
    async with create_hybrid_service() as service:
        request = CodeSystemLookupRequest(
            system_url="http://snomed.info/sct",
            code="73211009"
        )
        result = await service.lookup_code(request)
        print(f"Result: {result}")

asyncio.run(test_lookup())
```

## 📊 Performance Optimization

### Recommended Configuration

```env
# Production settings
GOOGLE_FHIR_MAX_CONCURRENT=20
GOOGLE_FHIR_BATCH_SIZE=200
GOOGLE_FHIR_CACHE_TTL=7200
GOOGLE_FHIR_TIMEOUT=15
```

### Caching Strategy

1. **Redis Caching**: Frequently accessed codes and valuesets
2. **Application Caching**: Connection pooling and metadata
3. **CDN Caching**: Static terminology resources

### Expected Performance

- **CodeSystem $lookup**: < 50ms (cached), < 200ms (uncached)
- **ValueSet $expand**: < 100ms (small sets), < 500ms (large sets)
- **Fallback Operations**: < 300ms additional latency

## 🔄 Synchronization

The hybrid service supports bidirectional synchronization:

```env
GOOGLE_FHIR_SYNC_ENABLED=true
GOOGLE_FHIR_SYNC_INTERVAL=300  # 5 minutes
```

Synchronization features:
- Automatic sync of changed resources
- Conflict resolution strategies
- Audit trail of sync operations
- Manual sync triggers via API

## 🚨 Troubleshooting

### Common Issues

1. **Authentication Errors**
   ```
   Error: Failed to initialize credentials
   Solution: Check service account key file and IAM permissions
   ```

2. **Network Connectivity**
   ```
   Error: Request timeout
   Solution: Check firewall rules and network connectivity to googleapis.com
   ```

3. **Quota Exceeded**
   ```
   Error: API quota exceeded
   Solution: Monitor usage in Google Cloud Console and request quota increase
   ```

### Debug Mode

Enable debug logging:

```env
GOOGLE_FHIR_LOG_LEVEL=DEBUG
GOOGLE_FHIR_DEBUG_LOGGING=true
```

### Support Channels

- Check Google Cloud Healthcare API documentation
- Review audit logs in Google Cloud Console
- Monitor API quotas and limits
- Use Cloud Logging for detailed request tracing

## 📈 Future Enhancements

### Planned Features

1. **Advanced Synchronization**
   - Real-time sync with Cloud Pub/Sub
   - Conflict resolution UI
   - Selective sync filters

2. **Enhanced Security**
   - Workload Identity for GKE
   - Customer-managed encryption keys
   - VPC Service Controls integration

3. **Performance Optimizations**
   - Batch operation support
   - Streaming responses for large valuesets
   - Intelligent prefetching

4. **Monitoring Improvements**
   - Prometheus metrics export
   - Grafana dashboards
   - Custom alerting rules

---

## 📝 Summary

This Google FHIR integration provides a robust, scalable solution for hybrid terminology operations that:

- ✅ Maintains compatibility with existing KB7 architecture
- ✅ Provides intelligent routing between local and cloud stores
- ✅ Includes comprehensive fallback and error handling
- ✅ Offers production-ready security and monitoring
- ✅ Supports FHIR R4 compliance with Google Healthcare API

The integration seamlessly combines the speed of local PostgreSQL lookups, the semantic capabilities of GraphDB, and the comprehensive terminology coverage of Google FHIR Healthcare API.