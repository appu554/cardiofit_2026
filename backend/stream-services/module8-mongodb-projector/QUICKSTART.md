# MongoDB Projector - Quick Start Guide

## Prerequisites

- Python 3.11+
- MongoDB 7.0+ (running locally or via Docker)
- Kafka cluster with `prod.ehr.events.enriched` topic
- module8-shared installed

## 1. Install Dependencies

```bash
cd module8-mongodb-projector
pip install -r requirements.txt
```

## 2. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your settings
nano .env
```

Key settings:
- `KAFKA_BOOTSTRAP_SERVERS`: Your Kafka broker address
- `MONGODB_URI`: Your MongoDB connection string
- `BATCH_SIZE`: Number of events to process per batch (default: 50)

## 3. Start MongoDB (if needed)

### Option A: Docker
```bash
docker run -d \
  --name mongodb \
  -p 27017:27017 \
  -v mongodb_data:/data/db \
  mongo:7
```

### Option B: Docker Compose (with web UI)
```bash
docker-compose up -d mongodb
# Or with Mongo Express UI:
docker-compose --profile ui up -d
```

Access Mongo Express at http://localhost:8081 (admin/admin123)

## 4. Run the Service

```bash
python run_projector.py
```

The service will:
1. Connect to MongoDB
2. Create collections and indexes
3. Start consuming from Kafka
4. Begin projecting events to MongoDB

## 5. Verify Service Health

```bash
# Health check
curl http://localhost:8051/health

# Get metrics
curl http://localhost:8051/metrics

# Get detailed status
curl http://localhost:8051/status

# Get collection statistics
curl http://localhost:8051/collections/stats
```

## 6. Test with Sample Data

```bash
# Produce test events and verify MongoDB
python test_projector.py
```

This will:
1. Produce 20 test enriched events to Kafka
2. Wait for processing
3. Verify data in MongoDB collections
4. Show statistics and sample documents

## 7. Query MongoDB Data

### Connect to MongoDB
```bash
mongosh mongodb://localhost:27017/module8_clinical
```

### Sample Queries

**View recent events:**
```javascript
db.clinical_documents.find().sort({ timestamp: -1 }).limit(5).pretty()
```

**Get patient timeline:**
```javascript
db.patient_timelines.findOne({ _id: "test_patient_1" })
```

**Find high-risk events:**
```javascript
db.clinical_documents.find({
  "enrichments.riskLevel": "CRITICAL"
}).sort({ timestamp: -1 })
```

**Get ML predictions with alerts:**
```javascript
db.ml_explanations.find({
  "predictions.sepsis_risk_24h.alert_triggered": true
}).sort({ timestamp: -1 })
```

**Risk level distribution:**
```javascript
db.clinical_documents.aggregate([
  { $group: { _id: "$enrichments.riskLevel", count: { $sum: 1 } } },
  { $sort: { count: -1 } }
])
```

**Check collection counts:**
```javascript
db.clinical_documents.countDocuments()
db.patient_timelines.countDocuments()
db.ml_explanations.countDocuments()
```

**View indexes:**
```javascript
db.clinical_documents.getIndexes()
db.patient_timelines.getIndexes()
db.ml_explanations.getIndexes()
```

## 8. Monitor Processing

### Watch Logs
```bash
# Service logs show batch processing
tail -f logs/mongodb-projector.log
```

### Metrics Endpoint
```bash
# Get real-time metrics
watch -n 2 'curl -s http://localhost:8051/metrics | jq'
```

Expected metrics:
```json
{
  "messages_consumed": 1523,
  "batches_processed": 31,
  "documents_written": 1523,
  "timelines_updated": 234,
  "explanations_written": 1245,
  "errors": 0,
  "total_clinical_docs": 125678,
  "total_patient_timelines": 2341,
  "total_ml_explanations": 98765
}
```

## 9. Docker Deployment

### Build and Run
```bash
# Build image
docker build -t mongodb-projector:latest .

# Run with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f mongodb-projector

# Check health
docker-compose exec mongodb-projector curl http://localhost:8051/health
```

### Scale for Higher Throughput
```bash
# Run multiple instances (different group IDs)
docker-compose up -d --scale mongodb-projector=3
```

## 10. Troubleshooting

### Service won't start
```bash
# Check MongoDB connection
mongosh mongodb://localhost:27017 --eval "db.adminCommand('ping')"

# Check Kafka connectivity
kafka-topics --bootstrap-server localhost:9092 --list

# Verify shared module installed
pip show module8-shared
```

### No data in MongoDB
```bash
# Check if events are in Kafka topic
kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched \
  --max-messages 1

# Check service logs for errors
docker-compose logs mongodb-projector | grep ERROR

# Verify consumer group is active
kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe \
  --group mongodb-projector-group
```

### Performance issues
```bash
# Check batch settings in .env
echo $BATCH_SIZE
echo $BATCH_TIMEOUT_SECONDS

# Monitor MongoDB performance
mongosh mongodb://localhost:27017/module8_clinical \
  --eval "db.currentOp()"

# Check index usage
mongosh mongodb://localhost:27017/module8_clinical \
  --eval "db.clinical_documents.aggregate([{ \$indexStats: {} }])"
```

## Next Steps

1. **Production Configuration**: Review and adjust batch size, connection pools, and indexes
2. **Monitoring**: Set up Prometheus/Grafana for metrics visualization
3. **Backup Strategy**: Configure MongoDB backups and retention policies
4. **Query Optimization**: Add indexes for your specific query patterns
5. **Data Retention**: Implement TTL indexes for automatic cleanup

## Useful Commands

```bash
# Service management
python run_projector.py                    # Start service
docker-compose up -d                       # Start with Docker
docker-compose down                        # Stop service

# Testing
python test_projector.py                   # Run test suite
curl http://localhost:8051/health          # Health check

# MongoDB operations
mongosh mongodb://localhost:27017/module8_clinical
db.clinical_documents.find().limit(5)      # View documents
db.stats()                                 # Database stats

# Kafka operations
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --describe --group mongodb-projector-group  # Check consumer lag
```

## Support

For issues or questions:
1. Check the main README.md for detailed documentation
2. Review service logs for error messages
3. Verify all prerequisites are met
4. Check MongoDB and Kafka connectivity
