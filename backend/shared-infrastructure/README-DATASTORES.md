# CardioFit Shared Infrastructure

Global data stores and infrastructure services shared across all CardioFit microservices and processing pipelines.

## Architecture Overview

```
CardioFit Platform
├── Microservices (Patient, Medication, etc.)
├── Flink Processing Pipeline
├── Apollo Federation Gateway
└── Shared Infrastructure (THIS COMPONENT)
    ├── Neo4j (Clinical Knowledge Graph)
    ├── ClickHouse (Time-Series Analytics)
    ├── Ontotext GraphDB (Semantic Knowledge)
    ├── Elasticsearch (Search & Analytics)
    ├── Redis (Caching & Sessions)
    └── Monitoring (Prometheus & Grafana)
```

## Services Overview

### 🗄️ **Neo4j** - Clinical Knowledge Graph
- **Purpose**: Patient relationships, clinical pathways, treatment patterns
- **Port**: 7474 (HTTP), 7687 (Bolt)
- **Database**: `cardiofit`
- **Use Cases**: Graph-based clinical decision support, pathway analytics

### 📈 **ClickHouse** - Time-Series Analytics
- **Purpose**: OLAP analytics, aggregated metrics, performance monitoring
- **Port**: 8123 (HTTP), 9000 (Native)
- **Database**: `cardiofit_analytics`
- **Use Cases**: Real-time dashboards, clinical metrics aggregation

### 🧠 **Ontotext GraphDB** - Semantic Knowledge
- **Purpose**: Clinical ontologies, semantic reasoning, FHIR terminology
- **Port**: 7200 (HTTP/REST)
- **Use Cases**: Clinical concept mapping, semantic validation, SNOMED CT

### 🔍 **Elasticsearch** - Search & Analytics
- **Purpose**: Full-text search, clinical document indexing
- **Port**: 9200 (HTTP), 9300 (Transport)
- **Use Cases**: Patient record search, clinical note analysis

### 🚀 **Redis** - Caching & Sessions
- **Ports**: 6379 (Master), 6380 (Replica)
- **Purpose**: Real-time caching, session management, critical alerts
- **Use Cases**: Patient context caching, ML prediction storage

### 📊 **Monitoring Stack**
- **Prometheus**: 9090 - Metrics collection
- **Grafana**: 3000 - Dashboards and visualization
- **Kibana**: 5601 - Elasticsearch visualization
- **Redis Insight**: 8001 - Redis management

## Quick Start

### Start All Services
```bash
cd backend/shared-infrastructure
./start-datastores.sh
```

### Stop All Services
```bash
# Stop containers but preserve data
./stop-datastores.sh

# Stop and remove all data (WARNING: Data loss!)
./stop-datastores.sh --remove-volumes

# Stop and remove images
./stop-datastores.sh --remove-images
```

### Using Docker Compose Directly
```bash
# Start all services
docker-compose -f docker-compose.datastores.yml up -d

# View logs
docker-compose -f docker-compose.datastores.yml logs -f

# Stop services
docker-compose -f docker-compose.datastores.yml down
```

## Service Access

### Web Interfaces

| Service | URL | Username | Password |
|---------|-----|----------|----------|
| Neo4j Browser | http://localhost:7474 | neo4j | CardioFit2024! |
| ClickHouse | http://localhost:8123 | cardiofit_user | ClickHouse2024! |
| GraphDB Workbench | http://localhost:7200 | admin | admin2024 |
| Kibana | http://localhost:5601 | elastic | ElasticCardioFit2024! |
| Redis Insight | http://localhost:8001 | - | RedisCardioFit2024! |
| Grafana | http://localhost:3000 | admin | GrafanaCardioFit2024! |
| Prometheus | http://localhost:9090 | - | - |

### Database Connections

| Service | Connection String | Notes |
|---------|------------------|-------|
| Neo4j | `bolt://localhost:7687` | Use Neo4j driver |
| ClickHouse | `localhost:8123` (HTTP)<br>`localhost:9000` (Native) | HTTP for REST, Native for JDBC |
| GraphDB | `http://localhost:7200` | REST API endpoint |
| Elasticsearch | `http://localhost:9200` | REST API with authentication |
| Redis Master | `localhost:6379` | Primary read/write |
| Redis Replica | `localhost:6380` | Read-only replica |

## Data Persistence

All data is stored in named Docker volumes:

```bash
# View all CardioFit volumes
docker volume ls | grep cardiofit

# Backup a volume (example for Neo4j)
docker run --rm -v cardiofit_neo4j_data:/data -v $(pwd):/backup alpine tar czf /backup/neo4j-backup.tar.gz /data

# Restore a volume
docker run --rm -v cardiofit_neo4j_data:/data -v $(pwd):/backup alpine tar xzf /backup/neo4j-backup.tar.gz -C /
```

## Configuration

### Environment-Specific Settings

For production deployments, override default settings:

```bash
# Example production override
export NEO4J_AUTH=neo4j/production_password_here
export CLICKHOUSE_PASSWORD=production_clickhouse_password
export REDIS_PASSWORD=production_redis_password

# Then start services
./start-datastores.sh
```

### Custom Configuration Files

- **Redis**: `redis/config/redis.conf`
- **ClickHouse**: `clickhouse/config/custom.xml`
- **Neo4j**: `neo4j/conf/neo4j.conf`
- **Prometheus**: `monitoring/prometheus.yml`

## Integration with Services

### From Microservices

```javascript
// Example: Patient Service using Neo4j
const neo4j = require('neo4j-driver');
const driver = neo4j.driver('bolt://localhost:7687',
  neo4j.auth.basic('neo4j', 'CardioFit2024!'));

// Example: Using ClickHouse for analytics
const { ClickHouse } = require('clickhouse');
const ch = new ClickHouse({
  url: 'http://localhost',
  port: 8123,
  debug: false,
  basicAuth: {
    username: 'cardiofit_user',
    password: 'ClickHouse2024!'
  }
});
```

### From Flink Processing

```java
// Example: Neo4j Sink in Flink
transformedEvents
    .filter(event -> event.hasDestination("neo4j"))
    .addSink(new Neo4jGraphSink())
    .name("Neo4j Graph Sink");

// Example: ClickHouse Sink
transformedEvents
    .filter(event -> event.hasDestination("analytics"))
    .addSink(new ClickHouseSink())
    .name("ClickHouse Analytics Sink");
```

## Monitoring & Health Checks

### Health Check Endpoints

| Service | Health Check URL |
|---------|-----------------|
| Neo4j | `http://localhost:7474/db/system/tx/commit` |
| ClickHouse | `http://localhost:8123/ping` |
| GraphDB | `http://localhost:7200/rest/monitor/healthcheck` |
| Elasticsearch | `http://localhost:9200/_cluster/health` |
| Redis | `redis-cli -p 6379 ping` |

### Prometheus Metrics

All services expose metrics at `/metrics` endpoints:
- Neo4j: `http://localhost:2004/metrics`
- ClickHouse: `http://localhost:8123/metrics`
- Elasticsearch: `http://localhost:9200/_prometheus/metrics`

### Grafana Dashboards

Pre-configured dashboards available at `http://localhost:3000`:
- Infrastructure Overview
- Neo4j Performance
- ClickHouse Analytics
- Redis Performance
- Elasticsearch Cluster Health

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 3000, 5601, 6379-6380, 7200, 7474, 7687, 8001, 8123, 9000, 9090, 9200, 9300 are available

2. **Memory issues**: Increase Docker memory limits:
   ```bash
   # macOS/Windows: Docker Desktop -> Settings -> Resources -> Memory
   # Linux: Adjust systemd service or docker daemon settings
   ```

3. **Volume permissions**: On Linux, ensure Docker has permissions:
   ```bash
   sudo chown -R $USER:$USER /var/lib/docker/volumes/cardiofit_*
   ```

4. **Service startup order**: Services have dependencies. Use the provided scripts which handle proper startup order.

### Logs

```bash
# View all service logs
docker-compose -f docker-compose.datastores.yml logs -f

# View specific service logs
docker-compose -f docker-compose.datastores.yml logs -f neo4j
docker-compose -f docker-compose.datastores.yml logs -f clickhouse
```

### Reset Everything

```bash
# Complete reset (WARNING: All data lost!)
./stop-datastores.sh --remove-volumes --remove-images
./start-datastores.sh
```

## Security Notes

### Default Passwords

⚠️ **Change default passwords in production!**

- All passwords are currently stored in the Docker Compose file
- For production, use Docker secrets or external secret management
- Consider using OAuth/SSO integration where available

### Network Security

- All services run on a dedicated Docker network (`cardiofit-network`)
- Only necessary ports are exposed to the host
- Consider using reverse proxy (nginx/traefik) for production

### Data Security

- Enable SSL/TLS for all services in production
- Use encrypted volumes for sensitive data
- Implement proper backup encryption
- Follow HIPAA compliance guidelines for healthcare data

## Performance Tuning

### Memory Allocation

Default memory limits are conservative. For production:

```yaml
# Increase in docker-compose.datastores.yml
services:
  neo4j:
    environment:
      NEO4J_server_memory_heap_max__size: 8G
      NEO4J_server_memory_pagecache__size: 4G

  clickhouse:
    environment:
      CLICKHOUSE_MAX_MEMORY_USAGE: 16000000000  # 16GB

  elasticsearch:
    environment:
      ES_JAVA_OPTS: "-Xms8g -Xmx8g"
```

### Disk I/O

- Use SSD storage for better performance
- Consider separate volumes for data and logs
- Monitor disk usage with provided Grafana dashboards

## Support & Documentation

- **Neo4j**: [Official Documentation](https://neo4j.com/docs/)
- **ClickHouse**: [Official Documentation](https://clickhouse.com/docs/)
- **Ontotext GraphDB**: [Documentation](https://graphdb.ontotext.com/documentation/)
- **Elasticsearch**: [Official Guide](https://www.elastic.co/guide/)
- **Redis**: [Official Documentation](https://redis.io/documentation)

For CardioFit-specific integration questions, refer to individual service documentation in their respective directories.