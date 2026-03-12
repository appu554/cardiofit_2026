# 🗄️ Knowledge Base Missing Database Technologies - Docker Setup Guide

## Overview
This guide provides comprehensive Docker configurations for the missing database technologies required by the Knowledge Base services according to the KB Implementation Guide.

## Missing Database Technologies

| KB Service | Required Database | Current Status | Port | Purpose |
|------------|------------------|----------------|------|---------|
| KB-2 Clinical Context | MongoDB | ❌ Not Implemented | 27017 | Document store for complex clinical contexts |
| KB-3 Guidelines | Neo4j | ❌ Not Implemented | 7474/7687 | Graph database for clinical pathways |
| KB-4 Patient Safety | TimescaleDB | ❌ Not Implemented | 5434 | Time-series analytics for safety events |
| KB-6 Formulary | Elasticsearch | ❌ Not Implemented | 9200/9300 | Full-text search for formulary data |

---

## 📦 Complete Docker Compose Configuration

Create a new file: `docker-compose.databases.yml`

```yaml
version: '3.8'

services:
  # ==================== MongoDB for KB-2 Clinical Context ====================
  mongodb:
    image: mongo:7.0
    container_name: kb-mongodb
    restart: unless-stopped
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: mongodb_admin_password
      MONGO_INITDB_DATABASE: kb_clinical_context
    volumes:
      - mongodb_data:/data/db
      - ./init-scripts/mongodb:/docker-entrypoint-initdb.d:ro
    networks:
      - kb-network
    healthcheck:
      test: echo 'db.runCommand("ping").ok' | mongosh localhost:27017/kb_clinical_context --quiet
      interval: 10s
      timeout: 5s
      retries: 5

  # MongoDB Express - Web UI for MongoDB
  mongo-express:
    image: mongo-express:1.0.2
    container_name: kb-mongo-express
    restart: unless-stopped
    ports:
      - "8090:8081"
    environment:
      ME_CONFIG_MONGODB_ADMINUSERNAME: admin
      ME_CONFIG_MONGODB_ADMINPASSWORD: mongodb_admin_password
      ME_CONFIG_MONGODB_URL: mongodb://admin:mongodb_admin_password@mongodb:27017/
      ME_CONFIG_BASICAUTH: false
    depends_on:
      - mongodb
    networks:
      - kb-network

  # ==================== Neo4j for KB-3 Guidelines ====================
  neo4j:
    image: neo4j:5.15-enterprise
    container_name: kb-neo4j
    restart: unless-stopped
    ports:
      - "7474:7474"  # HTTP
      - "7687:7687"  # Bolt
    environment:
      NEO4J_AUTH: neo4j/neo4j_password
      NEO4J_ACCEPT_LICENSE_AGREEMENT: "yes"
      NEO4J_dbms_memory_pagecache_size: "1G"
      NEO4J_dbms_memory_heap_initial__size: "1G"
      NEO4J_dbms_memory_heap_max__size: "2G"
      NEO4J_dbms_connector_bolt_enabled: "true"
      NEO4J_dbms_connector_http_enabled: "true"
      NEO4J_dbms_security_procedures_unrestricted: "gds.*,apoc.*"
      NEO4J_dbms_security_procedures_allowlist: "gds.*,apoc.*"
      NEO4JLABS_PLUGINS: '["apoc", "graph-data-science"]'
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
      - neo4j_import:/var/lib/neo4j/import
      - neo4j_plugins:/plugins
      - ./init-scripts/neo4j:/var/lib/neo4j/import/init:ro
    networks:
      - kb-network
    healthcheck:
      test: wget -q --spider http://localhost:7474 || exit 1
      interval: 10s
      timeout: 5s
      retries: 5

  # ==================== TimescaleDB for KB-4 Patient Safety ====================
  timescaledb:
    image: timescale/timescaledb:latest-pg15
    container_name: kb-timescaledb
    restart: unless-stopped
    ports:
      - "5434:5432"
    environment:
      POSTGRES_USER: timescale
      POSTGRES_PASSWORD: timescale_password
      POSTGRES_DB: kb_patient_safety
      TS_TUNE_MAX_CONNS: 100
      TS_TUNE_MAX_BG_WORKERS: 8
    volumes:
      - timescaledb_data:/var/lib/postgresql/data
      - ./init-scripts/timescaledb:/docker-entrypoint-initdb.d:ro
    networks:
      - kb-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U timescale -d kb_patient_safety"]
      interval: 10s
      timeout: 5s
      retries: 5
    command: >
      postgres
      -c shared_preload_libraries=timescaledb
      -c max_connections=100
      -c shared_buffers=1GB
      -c effective_cache_size=3GB
      -c maintenance_work_mem=256MB
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c default_statistics_target=100
      -c random_page_cost=1.1
      -c effective_io_concurrency=200
      -c work_mem=10MB
      -c timescaledb.max_background_workers=8

  # ==================== Elasticsearch for KB-6 Formulary ====================
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.3
    container_name: kb-elasticsearch
    restart: unless-stopped
    ports:
      - "9200:9200"
      - "9300:9300"
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - xpack.security.enrollment.enabled=false
      - "ES_JAVA_OPTS=-Xms1g -Xmx1g"
      - cluster.name=kb-formulary-cluster
      - node.name=kb-es-node1
      - bootstrap.memory_lock=true
    ulimits:
      memlock:
        soft: -1
        hard: -1
      nofile:
        soft: 65536
        hard: 65536
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data
      - ./init-scripts/elasticsearch:/usr/share/elasticsearch/init:ro
    networks:
      - kb-network
    healthcheck:
      test: curl -s http://localhost:9200 >/dev/null || exit 1
      interval: 30s
      timeout: 10s
      retries: 5

  # Kibana - Elasticsearch UI
  kibana:
    image: docker.elastic.co/kibana/kibana:8.11.3
    container_name: kb-kibana
    restart: unless-stopped
    ports:
      - "5601:5601"
    environment:
      ELASTICSEARCH_HOSTS: '["http://elasticsearch:9200"]'
      ELASTICSEARCH_USERNAME: kibana_system
      ELASTICSEARCH_PASSWORD: kibana_password
      xpack.security.enabled: "false"
    depends_on:
      - elasticsearch
    networks:
      - kb-network
    healthcheck:
      test: curl -s http://localhost:5601/api/status || exit 1
      interval: 30s
      timeout: 10s
      retries: 5

networks:
  kb-network:
    driver: bridge

volumes:
  mongodb_data:
  neo4j_data:
  neo4j_logs:
  neo4j_import:
  neo4j_plugins:
  timescaledb_data:
  elasticsearch_data:
```

---

## 🚀 Initialization Scripts

### MongoDB Initialization Script
Create `init-scripts/mongodb/01-init-kb-clinical-context.js`:

```javascript
// Switch to the kb_clinical_context database
db = db.getSiblingDB('kb_clinical_context');

// Create user for the application
db.createUser({
  user: 'kb_context_user',
  pwd: 'kb_context_password',
  roles: [
    {
      role: 'readWrite',
      db: 'kb_clinical_context'
    }
  ]
});

// Create collections with validation
db.createCollection('phenotype_definitions', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['phenotype_id', 'name', 'version', 'criteria', 'status'],
      properties: {
        phenotype_id: {
          bsonType: 'string',
          description: 'Unique identifier for phenotype'
        },
        name: {
          bsonType: 'string',
          description: 'Human-readable phenotype name'
        },
        version: {
          bsonType: 'string',
          pattern: '^\\d+\\.\\d+\\.\\d+$'
        },
        criteria: {
          bsonType: 'object',
          description: 'Phenotype detection criteria'
        },
        status: {
          enum: ['active', 'draft', 'deprecated']
        }
      }
    }
  }
});

db.createCollection('patient_contexts', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['patient_id', 'context_id', 'timestamp'],
      properties: {
        patient_id: { bsonType: 'string' },
        context_id: { bsonType: 'string' },
        timestamp: { bsonType: 'date' }
      }
    }
  }
});

// Create indexes
db.phenotype_definitions.createIndex({ 'phenotype_id': 1, 'version': -1 });
db.phenotype_definitions.createIndex({ 'status': 1 });
db.patient_contexts.createIndex({ 'patient_id': 1, 'timestamp': -1 });
db.patient_contexts.createIndex({ 'context_id': 1 }, { unique: true });

print('MongoDB initialization complete for KB-2 Clinical Context');
```

### Neo4j Initialization Script
Create `init-scripts/neo4j/01-init-guidelines.cypher`:

```cypher
// Create constraints for data integrity
CREATE CONSTRAINT guideline_id IF NOT EXISTS 
    FOR (g:Guideline) REQUIRE g.id IS UNIQUE;

CREATE CONSTRAINT recommendation_id IF NOT EXISTS 
    FOR (r:Recommendation) REQUIRE r.id IS UNIQUE;

CREATE CONSTRAINT evidence_id IF NOT EXISTS 
    FOR (e:Evidence) REQUIRE e.id IS UNIQUE;

CREATE CONSTRAINT condition_id IF NOT EXISTS 
    FOR (c:Condition) REQUIRE c.id IS UNIQUE;

// Create indexes for performance
CREATE INDEX guideline_condition IF NOT EXISTS 
    FOR (g:Guideline) ON (g.condition);

CREATE INDEX recommendation_grade IF NOT EXISTS 
    FOR (r:Recommendation) ON (r.grade);

CREATE INDEX evidence_level IF NOT EXISTS 
    FOR (e:Evidence) ON (e.level);

// Load sample ACC/AHA Hypertension Guideline
MERGE (g:Guideline {
    id: 'ACC_AHA_HTN_2017',
    title: '2017 ACC/AHA Guideline for High Blood Pressure',
    publisher: 'ACC/AHA',
    publication_date: date('2017-11-13'),
    condition: 'Hypertension',
    version: '2017.1',
    status: 'active'
})

MERGE (htn:Condition {
    id: 'CONDITION_HTN',
    name: 'Hypertension',
    icd10: 'I10',
    snomed: '38341003'
})

MERGE (stage2:Condition {
    id: 'CONDITION_HTN_STAGE2',
    name: 'Stage 2 Hypertension',
    parent: 'CONDITION_HTN',
    criteria: 'BP ≥140/90 mmHg'
})

MERGE (ckd:Condition {
    id: 'CONDITION_CKD',
    name: 'Chronic Kidney Disease',
    icd10: 'N18',
    snomed: '709044004'
})

// Create recommendations
MERGE (r1:Recommendation {
    id: 'HTN_REC_001',
    text: 'Initiate antihypertensive therapy for Stage 2 HTN',
    grade: 'I',
    level_of_evidence: 'A',
    priority: 1
})

MERGE (r2:Recommendation {
    id: 'HTN_REC_002',
    text: 'Use ACEi/ARB as first-line for HTN with CKD',
    grade: 'I',
    level_of_evidence: 'B',
    priority: 1
})

// Create evidence
MERGE (e1:Evidence {
    id: 'EVIDENCE_001',
    study_type: 'RCT',
    pmid: '28146533',
    summary: 'SPRINT trial demonstrates benefit of intensive BP control',
    quality_score: 0.95
})

// Create relationships
MERGE (g)-[:CONTAINS]->(r1)
MERGE (g)-[:CONTAINS]->(r2)
MERGE (r1)-[:APPLIES_TO]->(stage2)
MERGE (r2)-[:APPLIES_TO]->(htn)
MERGE (r2)-[:APPLIES_TO]->(ckd)
MERGE (r1)-[:SUPPORTED_BY]->(e1)
MERGE (r1)-[:FOLLOWED_BY {condition: 'if_ckd_present'}]->(r2)

// Create clinical pathway
CREATE (pathway:ClinicalPathway {
    id: 'HTN_PATHWAY_001',
    name: 'Hypertension Management Pathway',
    created_at: datetime(),
    version: '1.0.0'
})

MERGE (pathway)-[:STARTS_WITH]->(r1)
MERGE (pathway)-[:INCLUDES]->(r2);
```

### TimescaleDB Initialization Script
Create `init-scripts/timescaledb/01-init-patient-safety.sql`:

```sql
-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create schema for KB-4 Patient Safety
CREATE SCHEMA IF NOT EXISTS patient_safety;

-- Safety alerts time-series table
CREATE TABLE patient_safety.safety_alerts (
    time TIMESTAMPTZ NOT NULL,
    alert_id UUID DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    description TEXT,
    source_system VARCHAR(50),
    triggering_values JSONB,
    recommendations JSONB,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    metadata JSONB
);

-- Convert to hypertable for time-series optimization
SELECT create_hypertable('patient_safety.safety_alerts', 'time', 
    chunk_time_interval => INTERVAL '1 day');

-- Create indexes
CREATE INDEX idx_safety_alerts_patient 
    ON patient_safety.safety_alerts(patient_id, time DESC);
CREATE INDEX idx_safety_alerts_type 
    ON patient_safety.safety_alerts(alert_type, time DESC);
CREATE INDEX idx_safety_alerts_severity 
    ON patient_safety.safety_alerts(severity, time DESC);

-- Patient risk profiles
CREATE TABLE patient_safety.patient_risk_profiles (
    patient_id VARCHAR(100) PRIMARY KEY,
    risk_scores JSONB NOT NULL DEFAULT '{}',
    risk_factors JSONB,
    contraindications TEXT[],
    safety_flags JSONB,
    last_calculated TIMESTAMPTZ DEFAULT NOW(),
    version INTEGER DEFAULT 1
);

-- Safety rules repository
CREATE TABLE patient_safety.safety_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(200) UNIQUE NOT NULL,
    rule_type VARCHAR(50),
    condition_logic JSONB NOT NULL,
    action_logic JSONB NOT NULL,
    severity VARCHAR(20),
    active BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create continuous aggregate for hourly alert summary
CREATE MATERIALIZED VIEW patient_safety.safety_alerts_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    alert_type,
    severity,
    COUNT(*) as alert_count,
    COUNT(DISTINCT patient_id) as unique_patients
FROM patient_safety.safety_alerts
WHERE time > NOW() - INTERVAL '30 days'
GROUP BY hour, alert_type, severity
WITH NO DATA;

-- Refresh policy for continuous aggregate
SELECT add_continuous_aggregate_policy('patient_safety.safety_alerts_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- Retention policy (keep raw data for 90 days)
SELECT add_retention_policy('patient_safety.safety_alerts', INTERVAL '90 days');

-- Create application user
CREATE USER kb_safety_user WITH PASSWORD 'kb_safety_password';
GRANT ALL PRIVILEGES ON SCHEMA patient_safety TO kb_safety_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA patient_safety TO kb_safety_user;
```

### Elasticsearch Initialization Script
Create `init-scripts/elasticsearch/01-init-formulary.sh`:

```bash
#!/bin/bash

# Wait for Elasticsearch to be ready
until curl -s http://localhost:9200/_cluster/health | grep -q '"status":"yellow\|green"'; do
  echo "Waiting for Elasticsearch..."
  sleep 5
done

# Create formulary index with mappings
curl -X PUT "localhost:9200/formulary" -H 'Content-Type: application/json' -d'
{
  "settings": {
    "number_of_shards": 2,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "drug_name_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "synonym_filter", "edge_ngram_filter"]
        }
      },
      "filter": {
        "edge_ngram_filter": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 20
        },
        "synonym_filter": {
          "type": "synonym",
          "synonyms": [
            "acetaminophen,tylenol,paracetamol",
            "ibuprofen,advil,motrin",
            "asa,aspirin"
          ]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "drug_rxnorm": {
        "type": "keyword"
      },
      "drug_name": {
        "type": "text",
        "analyzer": "drug_name_analyzer",
        "fields": {
          "keyword": {
            "type": "keyword"
          }
        }
      },
      "generic_name": {
        "type": "text",
        "analyzer": "drug_name_analyzer"
      },
      "brand_names": {
        "type": "text",
        "analyzer": "drug_name_analyzer"
      },
      "drug_class": {
        "type": "keyword"
      },
      "tier": {
        "type": "keyword"
      },
      "payer_id": {
        "type": "keyword"
      },
      "plan_id": {
        "type": "keyword"
      },
      "copay_amount": {
        "type": "float"
      },
      "prior_authorization": {
        "type": "boolean"
      },
      "quantity_limit": {
        "type": "object"
      },
      "effective_date": {
        "type": "date"
      },
      "termination_date": {
        "type": "date"
      }
    }
  }
}'

# Create drug pricing index
curl -X PUT "localhost:9200/drug_pricing" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "drug_rxnorm": {
        "type": "keyword"
      },
      "price_type": {
        "type": "keyword"
      },
      "price": {
        "type": "float"
      },
      "effective_date": {
        "type": "date"
      }
    }
  }
}'

echo "Elasticsearch initialization complete"
```

---

## 🔧 Service Configuration Updates

### KB-2 Clinical Context MongoDB Configuration
Update `kb-2-clinical-context/internal/config/config.go`:

```go
type Config struct {
    Port        string
    MongoDBURL  string
    MongoDBName string
    RedisURL    string
}

func LoadConfig() (*Config, error) {
    return &Config{
        Port:        getEnv("PORT", "8082"),
        MongoDBURL:  getEnv("MONGODB_URL", "mongodb://kb_context_user:kb_context_password@localhost:27017/kb_clinical_context"),
        MongoDBName: getEnv("MONGODB_NAME", "kb_clinical_context"),
        RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/2"),
    }, nil
}
```

### KB-3 Guidelines Neo4j Configuration
Update `kb-guideline-evidence/internal/config/config.go`:

```go
type Config struct {
    Port      string
    Neo4jURL  string
    Neo4jUser string
    Neo4jPass string
    RedisURL  string
}

func LoadConfig() (*Config, error) {
    return &Config{
        Port:      getEnv("PORT", "8083"),
        Neo4jURL:  getEnv("NEO4J_URL", "bolt://localhost:7687"),
        Neo4jUser: getEnv("NEO4J_USER", "neo4j"),
        Neo4jPass: getEnv("NEO4J_PASSWORD", "neo4j_password"),
        RedisURL:  getEnv("REDIS_URL", "redis://localhost:6379/3"),
    }, nil
}
```

### KB-4 Patient Safety TimescaleDB Configuration
Update `kb-4-patient-safety/internal/config/config.go`:

```go
type Config struct {
    Port           string
    TimescaleDBURL string
    KafkaBrokers   []string
    RedisURL       string
}

func LoadConfig() (*Config, error) {
    return &Config{
        Port:           getEnv("PORT", "8084"),
        TimescaleDBURL: getEnv("TIMESCALEDB_URL", "postgres://kb_safety_user:kb_safety_password@localhost:5434/kb_patient_safety"),
        KafkaBrokers:   strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
        RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/4"),
    }, nil
}
```

### KB-6 Formulary Elasticsearch Configuration
Update `kb-6-formulary/internal/config/config.go`:

```go
type Config struct {
    Port              string
    PostgreSQLURL     string
    ElasticsearchURL  []string
    RedisURL          string
}

func LoadConfig() (*Config, error) {
    return &Config{
        Port:             getEnv("PORT", "8086"),
        PostgreSQLURL:    getEnv("POSTGRESQL_URL", "postgres://postgres:password@localhost:5432/kb_formulary"),
        ElasticsearchURL: strings.Split(getEnv("ELASTICSEARCH_URL", "http://localhost:9200"), ","),
        RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379/6"),
    }, nil
}
```

---

## 📝 Updated Main Docker Compose

Create `docker-compose.complete.yml` that includes all services:

```yaml
version: '3.8'

services:
  # Include existing PostgreSQL and Redis from original docker-compose.yml
  postgres:
    extends:
      file: docker-compose.yml
      service: db
  
  redis:
    extends:
      file: docker-compose.yml
      service: redis

  # Include new databases
  mongodb:
    extends:
      file: docker-compose.databases.yml
      service: mongodb

  neo4j:
    extends:
      file: docker-compose.databases.yml
      service: neo4j

  timescaledb:
    extends:
      file: docker-compose.databases.yml
      service: timescaledb

  elasticsearch:
    extends:
      file: docker-compose.databases.yml
      service: elasticsearch

  # KB Services with correct database connections
  kb-2-clinical-context:
    build: ./kb-2-clinical-context
    ports:
      - "8082:8082"
    environment:
      - MONGODB_URL=mongodb://kb_context_user:kb_context_password@mongodb:27017/kb_clinical_context
      - REDIS_URL=redis://redis:6379/2
    depends_on:
      - mongodb
      - redis

  kb-3-guidelines:
    build: ./kb-guideline-evidence
    ports:
      - "8083:8083"
    environment:
      - NEO4J_URL=bolt://neo4j:7687
      - NEO4J_USER=neo4j
      - NEO4J_PASSWORD=neo4j_password
      - REDIS_URL=redis://redis:6379/3
    depends_on:
      - neo4j
      - redis

  kb-4-patient-safety:
    build: ./kb-4-patient-safety
    ports:
      - "8084:8084"
    environment:
      - TIMESCALEDB_URL=postgres://kb_safety_user:kb_safety_password@timescaledb:5432/kb_patient_safety
      - KAFKA_BROKERS=kafka:9092
      - REDIS_URL=redis://redis:6379/4
    depends_on:
      - timescaledb
      - redis
      - kafka

  kb-6-formulary:
    build: ./kb-6-formulary
    ports:
      - "8086:8086"
    environment:
      - POSTGRESQL_URL=postgres://postgres:password@postgres:5432/kb_formulary
      - ELASTICSEARCH_URL=http://elasticsearch:9200
      - REDIS_URL=redis://redis:6379/6
    depends_on:
      - postgres
      - elasticsearch
      - redis
```

---

## 🚀 Running the Complete Stack

### Start all database services:
```bash
# Start the missing databases
docker-compose -f docker-compose.databases.yml up -d

# Verify all services are healthy
docker-compose -f docker-compose.databases.yml ps

# Check logs
docker-compose -f docker-compose.databases.yml logs -f
```

### Initialize databases:
```bash
# MongoDB initialization
docker exec -it kb-mongodb mongosh -u admin -p mongodb_admin_password --authenticationDatabase admin /docker-entrypoint-initdb.d/01-init-kb-clinical-context.js

# Neo4j initialization
docker exec -it kb-neo4j cypher-shell -u neo4j -p neo4j_password < init-scripts/neo4j/01-init-guidelines.cypher

# TimescaleDB initialization (automatic via docker-entrypoint-initdb.d)

# Elasticsearch initialization
docker exec -it kb-elasticsearch bash /usr/share/elasticsearch/init/01-init-formulary.sh
```

### Access Web UIs:
- **MongoDB Express**: http://localhost:8090
- **Neo4j Browser**: http://localhost:7474
- **Kibana (Elasticsearch)**: http://localhost:5601

### Test connections:
```bash
# Test MongoDB
mongosh "mongodb://kb_context_user:kb_context_password@localhost:27017/kb_clinical_context" --eval "db.stats()"

# Test Neo4j
curl -u neo4j:neo4j_password http://localhost:7474/db/data/

# Test TimescaleDB
psql -h localhost -p 5434 -U kb_safety_user -d kb_patient_safety -c "SELECT version();"

# Test Elasticsearch
curl -X GET "localhost:9200/_cluster/health?pretty"
```

---

## 🔍 Troubleshooting

### Common Issues and Solutions

#### MongoDB Connection Issues
```bash
# Check if MongoDB is running
docker logs kb-mongodb

# Test connection
docker exec -it kb-mongodb mongosh -u admin -p mongodb_admin_password --authenticationDatabase admin
```

#### Neo4j Memory Issues
```yaml
# Adjust memory settings in docker-compose if needed
environment:
  NEO4J_dbms_memory_heap_max__size: "4G"
  NEO4J_dbms_memory_pagecache_size: "2G"
```

#### TimescaleDB Performance
```sql
-- Check chunk size
SELECT show_chunks('patient_safety.safety_alerts');

-- Optimize chunk size if needed
SELECT set_chunk_time_interval('patient_safety.safety_alerts', INTERVAL '7 days');
```

#### Elasticsearch Heap Size
```yaml
# Adjust heap size based on available memory
environment:
  - "ES_JAVA_OPTS=-Xms2g -Xmx2g"
```

---

## 📋 Health Check Commands

```bash
# Create a health check script
cat > check-databases.sh << 'EOF'
#!/bin/bash

echo "Checking database health..."

# MongoDB
echo -n "MongoDB: "
docker exec kb-mongodb mongosh --quiet --eval "db.adminCommand('ping')" > /dev/null 2>&1 && echo "✅ OK" || echo "❌ Failed"

# Neo4j
echo -n "Neo4j: "
curl -s -u neo4j:neo4j_password http://localhost:7474/db/data/ > /dev/null && echo "✅ OK" || echo "❌ Failed"

# TimescaleDB
echo -n "TimescaleDB: "
docker exec kb-timescaledb pg_isready -U timescale > /dev/null 2>&1 && echo "✅ OK" || echo "❌ Failed"

# Elasticsearch
echo -n "Elasticsearch: "
curl -s http://localhost:9200/_cluster/health | grep -q '"status":"yellow\|green"' && echo "✅ OK" || echo "❌ Failed"
EOF

chmod +x check-databases.sh
./check-databases.sh
```

---

## 🎯 Next Steps

1. **Update KB service code** to use the new database connections
2. **Implement database-specific logic**:
   - MongoDB document operations for KB-2
   - Neo4j graph queries for KB-3
   - TimescaleDB time-series operations for KB-4
   - Elasticsearch search queries for KB-6
3. **Load initial data** into each database
4. **Configure monitoring** with Prometheus/Grafana
5. **Set up backup strategies** for each database

---

## 📚 References

- [MongoDB Docker Documentation](https://hub.docker.com/_/mongo)
- [Neo4j Docker Documentation](https://neo4j.com/docs/operations-manual/current/docker/)
- [TimescaleDB Docker Documentation](https://docs.timescale.com/self-hosted/latest/install/docker/)
- [Elasticsearch Docker Documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/docker.html)