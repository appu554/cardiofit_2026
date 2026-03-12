# L1 Cache and Prefetcher Service

Ultra-fast clinical data caching with ML-based intelligent prefetching for sub-10ms response times.

## Features

- **<10ms Response Times**: Ultra-fast L1 in-memory cache with optimized data structures
- **ML-Based Prefetching**: Machine learning predictions for intelligent data preloading
- **Session Awareness**: Context-aware caching with per-user and per-session optimization
- **Multi-Layer Architecture**: L1 (in-memory) + L2 (Redis) caching strategy
- **Access Pattern Learning**: Continuous learning from user behavior patterns
- **Resource Management**: Memory pressure monitoring and intelligent eviction
- **Real-time Monitoring**: Prometheus metrics and Grafana dashboards

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    L1 Cache Service                         │
├─────────────────┬─────────────────┬─────────────────────────┤
│   FastAPI App   │  L1 Cache Mgr   │  Prefetch Manager      │
│   (Port 8030)   │  (<10ms cache)  │  (ML Predictions)      │
└─────────────────┴─────────────────┴─────────────────────────┘
         │                │                        │
         ▼                ▼                        ▼
┌─────────────────┬─────────────────┬─────────────────────────┐
│  External APIs  │   Redis L2      │    ML Predictor        │
│  (Data Sources) │   (256MB)       │   (Pattern Learning)   │
└─────────────────┴─────────────────┴─────────────────────────┘
         │                │                        │
         ▼                ▼                        ▼
┌─────────────────┬─────────────────┬─────────────────────────┐
│ Patient Service │ Clinical Service│     Prometheus          │
│ Medication Svc  │ Guideline Svc   │     (Monitoring)        │
└─────────────────┴─────────────────┴─────────────────────────┘
```

## Performance Characteristics

- **L1 Cache Hit**: <10ms response time
- **L2 Cache Hit**: <50ms response time
- **ML Prediction Accuracy**: >70% for active sessions
- **Memory Efficiency**: 512MB L1 + 256MB L2 default
- **Prefetch Success Rate**: >80% for predicted items
- **Concurrent Requests**: 1000+ req/sec sustained

## Quick Start

### Using Docker Compose (Recommended)

1. **Start the service:**
   ```bash
   docker-compose -f docker-compose.l1-cache-prefetcher.yml up -d
   ```

2. **Check service health:**
   ```bash
   curl http://localhost:8030/health
   ```

3. **View API documentation:**
   - Swagger UI: http://localhost:8030/docs
   - ReDoc: http://localhost:8030/redoc

4. **Access monitoring:**
   - Grafana: http://localhost:3002 (admin/l1-cache-grafana-admin)
   - Prometheus: http://localhost:9091

### Local Development

1. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

2. **Set up environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start Redis (L2 cache):**
   ```bash
   docker run -d --name redis-l2 -p 6380:6379 redis:7.2-alpine
   ```

4. **Run the service:**
   ```bash
   python -m uvicorn src.main:app --host 0.0.0.0 --port 8030 --reload
   ```

## API Endpoints

### Health & Monitoring
- `GET /health` - Basic health check
- `GET /health/ready` - Readiness check with dependencies
- `GET /metrics` - Prometheus metrics

### L1 Cache Operations
- `GET /cache/{key}` - Get cached data (<10ms target)
- `POST /cache` - Store data in cache
- `DELETE /cache/{key}` - Invalidate cached item
- `DELETE /cache/sessions/{session_id}` - Invalidate session cache

### Intelligent Prefetching
- `POST /prefetch/predict` - ML-based predictive prefetching
- `POST /prefetch/explicit` - Explicit prefetch for specific keys

### Analytics & Insights
- `GET /metrics/cache` - Detailed cache performance metrics
- `GET /metrics/prefetch` - Prefetch accuracy and performance
- `GET /analytics/access-patterns` - Access pattern insights for ML
- `GET /analytics/sessions` - Session-based analytics

## Usage Examples

### Basic Cache Operations

```bash
# Store data in cache
curl -X POST "http://localhost:8030/cache" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "patient_context_12345",
    "key_type": "patient_context",
    "data": {
      "patient_id": "12345",
      "demographics": {...},
      "medical_history": {...}
    },
    "ttl_seconds": 30,
    "session_id": "session_abc123"
  }'

# Retrieve data from cache
curl "http://localhost:8030/cache/patient_context_12345?session_id=session_abc123" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### ML-Based Prefetching

```bash
# Trigger predictive prefetching
curl -X POST "http://localhost:8030/prefetch/predict?session_id=session_abc123&max_items=20&confidence_threshold=0.7" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Response shows prefetched keys and performance metrics
{
  "requested_keys": ["predicted_key_1", "predicted_key_2", ...],
  "prefetched_keys": ["predicted_key_1", "predicted_key_3", ...],
  "skipped_keys": ["predicted_key_2"],
  "total_prefetched": 15,
  "total_size_mb": 2.3,
  "processing_time_ms": 45.2,
  "predictions_used": 18
}
```

### Explicit Prefetching

```bash
curl -X POST "http://localhost:8030/prefetch/explicit" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "keys": [
      "medication_data_aspirin",
      "guideline_cardiology_2024",
      "semantic_mesh_hypertension"
    ],
    "session_context": {
      "session_id": "session_abc123",
      "user_id": "doctor_jones",
      "workflow_type": "medication_review"
    },
    "max_prefetch_items": 50,
    "prefetch_budget_mb": 100
  }'
```

## Configuration

Key configuration options:

| Setting | Default | Description |
|---------|---------|-------------|
| `L1_CACHE_SIZE_MB` | 512 | L1 cache memory limit |
| `L1_CACHE_DEFAULT_TTL` | 10 | Default TTL in seconds |
| `L1_CACHE_MAX_ENTRIES` | 50000 | Maximum cache entries |
| `MAX_CONCURRENT_FETCHES` | 20 | Concurrent prefetch limit |
| `PREFETCH_BUDGET_MB` | 100 | Prefetch memory budget |
| `MIN_CONFIDENCE_THRESHOLD` | 0.6 | ML prediction confidence threshold |
| `ML_MODEL_UPDATE_INTERVAL_HOURS` | 6 | ML model retraining frequency |

## Machine Learning Features

### Access Pattern Learning
- **Temporal Patterns**: Hour-of-day and day-of-week access patterns
- **Session Correlations**: Co-accessed data within sessions
- **User Behavior**: Individual user access preferences
- **Workflow Context**: Clinical workflow-specific patterns

### Prediction Models
- **Random Forest Regressor**: Predicts time-to-access for cache keys
- **Collaborative Filtering**: Finds related data based on session patterns
- **Feature Engineering**: 19 engineered features from access patterns
- **Continuous Learning**: Models retrain every 6 hours with new data

### Prediction Features
```python
# Example feature vector for ML prediction
{
  "hour_of_day": 14,           # 2 PM
  "day_of_week": 2,            # Tuesday
  "time_since_last_access": 0.5,  # 30 minutes
  "total_access_count": 45,
  "access_frequency": 2.3,     # accesses per hour
  "session_activity_level": 5.2,
  "correlated_keys_accessed": 3,
  "key_type_patient": 1,       # one-hot encoded
  "session_pattern_strength": 0.7
}
```

## Performance Monitoring

### Key Metrics
- **Cache Hit Rate**: Target >85% for L1, >95% for L1+L2
- **Response Time**: P95 <10ms for L1 hits, P95 <50ms for L2 hits
- **Prefetch Accuracy**: >70% of prefetched items accessed
- **Memory Utilization**: <85% of allocated memory
- **ML Model Performance**: R² >0.6 for time prediction

### Monitoring Dashboards
The service includes comprehensive Grafana dashboards:

1. **Cache Performance Dashboard**
   - Hit rates, response times, memory usage
   - Cache operations per second
   - Eviction and expiration rates

2. **Prefetch Intelligence Dashboard**
   - ML prediction accuracy over time
   - Prefetch hit rates by confidence threshold
   - Feature importance analysis

3. **Session Analytics Dashboard**
   - Active sessions and user patterns
   - Workflow-specific cache performance
   - Session lifecycle metrics

## Troubleshooting

### Common Issues

1. **High memory usage**
   - Check cache size configuration
   - Monitor memory pressure threshold
   - Review TTL settings for data types

2. **Low cache hit rates**
   - Analyze access patterns
   - Adjust TTL values per data type
   - Review prefetch configuration

3. **ML model performance issues**
   - Check training data volume (>1000 samples recommended)
   - Monitor feature quality
   - Review model update frequency

4. **Slow response times**
   - Check system memory and CPU
   - Monitor external data source latency
   - Review concurrent request limits

### Health Checks

```bash
# Basic health
curl http://localhost:8030/health

# Detailed readiness
curl http://localhost:8030/health/ready

# Cache metrics
curl http://localhost:8030/metrics/cache

# Prefetch performance
curl http://localhost:8030/metrics/prefetch
```

## Security

### Authentication
All endpoints require JWT authentication:
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Data Protection
- In-memory data encryption for sensitive content
- Secure session isolation
- Access pattern anonymization for ML training

## Development

### Running Tests
```bash
pytest tests/ -v --cov=src
```

### Code Quality
```bash
black src/ tests/
isort src/ tests/
flake8 src/ tests/
mypy src/
```

### ML Model Development
```bash
# Analyze access patterns
python scripts/analyze_patterns.py

# Train custom models
python scripts/train_models.py

# Evaluate prediction accuracy
python scripts/evaluate_models.py
```

## License

Proprietary - CardioFit Clinical Synthesis Hub