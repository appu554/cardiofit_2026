# Module 6: Quick Start Guide

## 🚀 Getting Started in 5 Minutes

This guide helps you quickly start implementing Module 6 components.

---

## Prerequisites Checklist

- [ ] Modules 1-5 are running and producing data
- [ ] Kafka cluster is operational
- [ ] Java 17+ installed
- [ ] Node.js 18+ installed
- [ ] Maven configured
- [ ] PostgreSQL database available
- [ ] Redis cache available

---

## Component Startup Order

### 1️⃣ Kafka Topics (2 minutes)

```bash
# Navigate to flink-processing directory
cd backend/shared-infrastructure/flink-processing

# Run the topic creation script
./create-module6-topics.sh
```

**Script content** (create this file):

```bash
#!/bin/bash
# create-module6-topics.sh

KAFKA_BROKER="localhost:9092"
PARTITIONS=4
REPLICATION=1

echo "Creating Module 6 Analytics Topics..."

kafka-topics --create --topic analytics-patient-census \
  --bootstrap-server $KAFKA_BROKER \
  --partitions $PARTITIONS \
  --replication-factor $REPLICATION \
  --if-not-exists

kafka-topics --create --topic analytics-alert-metrics \
  --bootstrap-server $KAFKA_BROKER \
  --partitions $PARTITIONS \
  --replication-factor $REPLICATION \
  --if-not-exists

kafka-topics --create --topic analytics-ml-performance \
  --bootstrap-server $KAFKA_BROKER \
  --partitions $PARTITIONS \
  --replication-factor $REPLICATION \
  --if-not-exists

kafka-topics --create --topic analytics-department-workload \
  --bootstrap-server $KAFKA_BROKER \
  --partitions $PARTITIONS \
  --replication-factor $REPLICATION \
  --if-not-exists

kafka-topics --create --topic analytics-sepsis-surveillance \
  --bootstrap-server $KAFKA_BROKER \
  --partitions $PARTITIONS \
  --replication-factor $REPLICATION \
  --if-not-exists

echo "✅ All Module 6 topics created successfully!"

# List topics to verify
kafka-topics --list --bootstrap-server $KAFKA_BROKER | grep analytics
```

Make executable:
```bash
chmod +x create-module6-topics.sh
```

---

### 2️⃣ Analytics Engine (Flink Job) (5 minutes)

```bash
# Build the Flink job
cd backend/shared-infrastructure/flink-processing
mvn clean package -DskipTests

# Submit to Flink
flink run \
  -c com.cardiofit.flink.analytics.Module6_AnalyticsEngine \
  target/flink-ehr-intelligence-1.0.0.jar
```

**Verify it's running**:
```bash
# Check Flink Web UI
open http://localhost:8081

# Or check via CLI
flink list
```

**Expected output**: You should see "Module 6: Real-Time Analytics Engine" job running.

---

### 3️⃣ Dashboard API Service (10 minutes)

```bash
# Create the service directory
cd backend/services
mkdir -p dashboard-api/src/services
cd dashboard-api

# Initialize Node.js project
npm init -y

# Install dependencies
npm install @apollo/server graphql typescript ts-node @types/node
npm install kafkajs ioredis pg
npm install @types/ioredis @types/pg

# Create tsconfig.json
cat > tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules"]
}
EOF

# Create package.json scripts
npm pkg set scripts.build="tsc"
npm pkg set scripts.start="node dist/server.js"
npm pkg set scripts.dev="ts-node src/server.ts"

# Create .env file
cat > .env << 'EOF'
KAFKA_BROKERS=localhost:9092
REDIS_HOST=localhost
REDIS_PORT=6379
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=cardiofit
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_password_here
PORT=4001
EOF

echo "✅ Dashboard API project initialized!"
```

**Copy the implementation files from the guide**, then:

```bash
# Build and start
npm run build
npm start

# Or for development with auto-reload
npm run dev
```

**Verify it's running**:
```bash
curl http://localhost:4001/graphql
```

---

### 4️⃣ React Dashboard (15 minutes)

```bash
# Create React app
cd backend/services
npx create-react-app dashboard-ui --template typescript

cd dashboard-ui

# Install dependencies
npm install @apollo/client graphql
npm install @mui/material @emotion/react @emotion/styled
npm install recharts @mui/icons-material
npm install ws

# Create .env file
cat > .env << 'EOF'
REACT_APP_GRAPHQL_ENDPOINT=http://localhost:4001/graphql
REACT_APP_WS_ENDPOINT=ws://localhost:8080/dashboard/realtime
EOF

echo "✅ Dashboard UI project initialized!"
```

**Copy the React components from the guide**, then:

```bash
# Start development server
npm start

# Open browser
open http://localhost:3000
```

---

## Verification Steps

### 1. Check Kafka Topics Have Data

```bash
# Monitor analytics topics
kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic analytics-patient-census \
  --from-beginning \
  --max-messages 5
```

**Expected**: You should see JSON messages with patient census data.

### 2. Check Redis Cache

```bash
# Connect to Redis
redis-cli

# Check for analytics data
KEYS census:*
GET census:ICU

# Should show cached analytics data
```

### 3. Test GraphQL API

```bash
# Test query
curl -X POST http://localhost:4001/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ hospitalKPIs { totalPatients criticalRiskCount } }"
  }'
```

**Expected**: JSON response with hospital KPIs.

### 4. Open Dashboard UI

```
http://localhost:3000
```

**Expected**: Executive dashboard showing:
- Total patient census
- Critical risk patient count
- Active alerts
- Department comparison charts

---

## Common Issues & Solutions

### Issue: Kafka topics not receiving data

**Solution**:
```bash
# Check if Modules 1-5 are running
flink list

# Verify topics exist
kafka-topics --list --bootstrap-server localhost:9092

# Check consumer lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --describe --group analytics-engine
```

### Issue: Dashboard API can't connect to Kafka

**Solution**:
```bash
# Check Kafka is accessible
kafka-broker-api-versions --bootstrap-server localhost:9092

# Update .env file with correct Kafka broker address
# Restart dashboard API
```

### Issue: Redis cache is empty

**Solution**:
```bash
# Check if Kafka consumers are running
# Check dashboard API logs for errors
# Verify Redis is running
redis-cli ping  # Should return PONG
```

### Issue: React dashboard shows "Loading..." forever

**Solution**:
```bash
# Check GraphQL API is running
curl http://localhost:4001/graphql

# Check browser console for CORS errors
# Verify .env has correct REACT_APP_GRAPHQL_ENDPOINT
```

---

## Performance Tuning

### Flink Analytics Engine

```bash
# Increase parallelism for higher throughput
flink run -p 24 -c com.cardiofit.flink.analytics.Module6_AnalyticsEngine \
  target/flink-ehr-intelligence-1.0.0.jar
```

### Dashboard API

```javascript
// Increase polling interval for less load
pollInterval: 60000  // 1 minute instead of 30 seconds
```

### Redis Cache

```bash
# Increase memory limit
redis-cli CONFIG SET maxmemory 2gb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

---

## Monitoring Commands

```bash
# Monitor Flink job
flink list -r  # Running jobs
watch -n 5 'flink list -r'  # Auto-refresh every 5s

# Monitor Kafka topics
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic analytics-patient-census

# Monitor Redis memory
redis-cli INFO memory

# Monitor API logs
tail -f logs/dashboard-api.log

# Monitor React build
npm run build -- --stats
```

---

## Next Steps

1. ✅ Verify all components are running
2. 📊 Open dashboard and confirm live data
3. 🔔 Test notification system
4. 📈 Add more analytics views as needed
5. 🎨 Customize dashboard UI
6. 📱 Build mobile app version
7. 🚀 Deploy to production

---

## Support

- **Implementation Guide**: [MODULE_6_IMPLEMENTATION_GUIDE.md](./MODULE_6_IMPLEMENTATION_GUIDE.md)
- **Full Documentation**: [Module_6_Advanced_Analytics_&_Predictive_Dashboards.txt](./src/docs/module_6/Module_6_Advanced_Analytics_&_Predictive_Dashboards.txt)
- **Architecture Diagram**: See implementation guide
- **Troubleshooting**: Check logs in `logs/` directory

---

## Health Check Checklist

- [ ] Flink job "Module 6: Real-Time Analytics Engine" is running
- [ ] All 5 analytics Kafka topics exist and have data
- [ ] Redis cache contains keys like `census:ICU`, `alerts:ED`
- [ ] Dashboard API responds to GraphQL queries
- [ ] React UI loads and shows live data
- [ ] WebSocket connection established for real-time updates
- [ ] Notifications are being delivered (check logs)

**All green?** 🎉 Module 6 is fully operational!
