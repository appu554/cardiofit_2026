# KB-6 Formulary Docker Setup

This document provides comprehensive instructions for running the KB-6 Formulary Management Service using Docker.

## 🏗️ Architecture

The Docker setup includes:

- **KB-6 Formulary Service** (Go) - gRPC (8086) + REST API (8087)
- **PostgreSQL 16** - Primary database (5433)
- **Redis 7** - Caching layer (6380)
- **Elasticsearch 8.11** - Search engine (9200)
- **Kibana 8.11** - Elasticsearch UI (5601)
- **Prometheus** - Metrics collection (9090)
- **Grafana** - Monitoring dashboards (3000)

## 🚀 Quick Start

### Prerequisites

- Docker 24.0+ and Docker Compose 2.0+
- Make (optional, for convenient commands)
- 4GB+ available RAM
- 10GB+ available disk space

### 1. Start All Services

```bash
# Using Make (recommended)
make up

# Or using Docker Compose directly
docker-compose up -d
```

### 2. Verify Services

```bash
# Check all service status
make status

# Check health endpoints
make health

# View logs
make logs
```

### 3. Access Endpoints

| Service | URL | Credentials |
|---------|-----|-------------|
| REST API | http://localhost:8087 | None |
| gRPC API | localhost:8086 | None |
| API Docs | http://localhost:8087/api/v1/docs | None |
| Elasticsearch | http://localhost:9200 | elastic/changeme |
| Kibana | http://localhost:5601 | elastic/changeme |
| Grafana | http://localhost:3000 | admin/admin |
| Prometheus | http://localhost:9090 | None |

## 📋 Available Commands

### Service Management
```bash
make build          # Build all images
make up             # Start all services
make down           # Stop all services
make restart        # Restart all services
make status         # Show service status
```

### Monitoring & Logs
```bash
make health         # Check service health
make logs           # View all logs
make logs-kb6       # View KB-6 service logs only
make logs-es        # View Elasticsearch logs
make monitor        # Open monitoring dashboards
make metrics        # Show current metrics
```

### Development
```bash
make test           # Run Go tests
make test-integration # Run integration tests
make dev-up         # Start in development mode
```

### Data Management
```bash
make seed           # Load sample formulary data
make backup         # Backup database and ES data
make restore BACKUP_FILE=path.sql  # Restore from backup
```

### Maintenance
```bash
make clean          # Clean containers and images
make clean-data     # Remove all data (destructive!)
make reset          # Full reset and rebuild
make security-scan  # Security scan on images
```

## 🔧 Configuration

### Environment Variables

The service uses these key environment variables:

```bash
# Service Configuration
PORT=8086
ENVIRONMENT=development
DEBUG=true

# Database
DB_HOST=postgres
DB_PORT=5432
DB_NAME=kb_formulary
DB_USER=postgres
DB_PASSWORD=password

# Redis Cache
REDIS_URL=redis:6379
REDIS_PASSWORD=

# Elasticsearch
ELASTICSEARCH_URL=http://elasticsearch:9200
ELASTICSEARCH_USERNAME=elastic
ELASTICSEARCH_PASSWORD=changeme
ELASTICSEARCH_ENABLED=true
```

### Custom Configuration

1. **Elasticsearch Settings**: Modify `config/elasticsearch/elasticsearch.yml`
2. **Redis Settings**: Modify `config/redis.conf`
3. **Prometheus**: Modify `config/prometheus/prometheus.yml`

## 🏥 Health Checks

All services include comprehensive health checks:

- **KB-6 Service**: HTTP health endpoint with component status
- **PostgreSQL**: Connection and database readiness
- **Redis**: Ping and memory status
- **Elasticsearch**: Cluster health and node status

Monitor health via:
```bash
curl http://localhost:8087/health
```

## 🔍 Troubleshooting

### Common Issues

1. **Port Conflicts**: Ensure ports 3000, 5601, 6380, 8086, 8087, 9090, 9200 are available
2. **Memory Issues**: Elasticsearch requires at least 2GB RAM
3. **Disk Space**: Monitor Docker volume usage with `docker system df`

### Service-Specific Troubleshooting

#### KB-6 Service Won't Start
```bash
# Check logs
make logs-kb6

# Verify dependencies
docker-compose ps postgres redis elasticsearch

# Restart service
docker-compose restart kb6-formulary
```

#### Elasticsearch Issues
```bash
# Check ES logs
make logs-es

# Verify cluster health
curl -u elastic:changeme http://localhost:9200/_cluster/health

# Reset ES data (destructive)
docker-compose stop elasticsearch
docker volume rm kb6_elasticsearch_data
docker-compose up -d elasticsearch
```

#### Database Connection Issues
```bash
# Check DB logs
make logs-db

# Connect to database
docker-compose exec postgres psql -U postgres -d kb_formulary

# Reset database (destructive)
make clean-data
```

### Log Locations

- **KB-6 Service**: `docker-compose logs kb6-formulary`
- **Elasticsearch**: `docker-compose logs elasticsearch`
- **PostgreSQL**: `docker-compose logs postgres`
- **All Services**: `docker-compose logs`

## 🚦 Production Considerations

### Security
- Change default passwords
- Enable TLS/SSL
- Configure firewalls
- Run security scans: `make security-scan`

### Performance
- Adjust JVM heap size for Elasticsearch
- Configure PostgreSQL connection pools
- Monitor resource usage
- Scale services horizontally

### Backup Strategy
- Regular database backups: `make backup`
- Elasticsearch snapshots
- Configuration backups
- Container registry for images

### Monitoring
- Set up alerting rules in Prometheus
- Configure Grafana dashboards
- Monitor disk usage and performance
- Set up log aggregation

## 🔗 Integration

### External Services
The KB-6 service can integrate with:
- Existing medication services
- FHIR servers
- Authentication providers
- External formulary data sources

### API Usage
```bash
# Test formulary coverage
curl "http://localhost:8087/api/v1/formulary/coverage?drug_id=123&payer_id=456"

# Search drugs
curl "http://localhost:8087/api/v1/formulary/search?q=aspirin"

# Check inventory
curl "http://localhost:8087/api/v1/inventory/stock?drug_id=123"
```

## 📚 Additional Resources

- [KB-6 Implementation Workflow](./docs/KB-6-Implementation-Workflow.md)
- [API Documentation](http://localhost:8087/api/v1/docs)
- [Elasticsearch Documentation](https://www.elastic.co/guide/index.html)
- [Redis Documentation](https://redis.io/documentation)
- [Prometheus Documentation](https://prometheus.io/docs/)

## 🆘 Support

For issues and questions:
1. Check logs: `make logs`
2. Verify health: `make health`
3. Review troubleshooting section above
4. Check service documentation