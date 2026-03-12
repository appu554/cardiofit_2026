# 🌐 Global Outbox Service

## Overview

The Global Outbox Service is a centralized event publishing service for all Clinical Synthesis Hub microservices. It implements the transactional outbox pattern at scale, providing guaranteed event delivery to Kafka while simplifying microservice development.

## Key Features

- **Centralized Event Publishing**: Single service handles all event publishing
- **Guaranteed Delivery**: Transactional outbox pattern ensures no data loss
- **Service Isolation**: Partitioned tables prevent cross-service interference
- **High Performance**: Optimized for >10,000 events/second throughput
- **Operational Excellence**: Comprehensive monitoring and observability

## Architecture

```
Microservices → gRPC API → Partitioned Database → Background Publisher → Kafka
```

## Quick Start

1. **Install Dependencies**
   ```bash
   pip install -r requirements.txt
   ```

2. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Run Database Migration**
   ```bash
   python run_migration.py
   ```

4. **Start Service**
   ```bash
   python run_service.py
   ```

## Configuration

- **Port**: 8040 (HTTP), 50051 (gRPC)
- **Database**: Supabase PostgreSQL
- **Message Queue**: Kafka (Confluent Cloud)

## API

### gRPC API
- `PublishEvent`: Publish an event to the outbox
- `HealthCheck`: Service health status
- `GetOutboxStats`: Queue statistics

### REST API
- `GET /health`: Health check
- `GET /metrics`: Prometheus metrics
- `GET /stats`: Outbox statistics

## Monitoring

- **Queue Depths**: Per-service queue monitoring
- **Success Rates**: Event publishing success rates
- **Latency**: End-to-end publishing latency
- **Error Rates**: Failed event tracking

## Documentation

- [Implementation Plan](../GLOBAL_OUTBOX_SERVICE_IMPLEMENTATION_PLAN.md)
- [Benefits Analysis](../GLOBAL_OUTBOX_BENEFITS_ANALYSIS.md)
- [Quick Start Guide](../GLOBAL_OUTBOX_QUICK_START.md)
