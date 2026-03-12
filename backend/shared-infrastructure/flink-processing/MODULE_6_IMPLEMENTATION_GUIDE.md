# Module 6: Advanced Analytics & Predictive Dashboards - Implementation Guide

## 📋 Executive Summary

**Module 6** extends the existing alert composition and routing infrastructure with comprehensive analytics, real-time dashboards, and predictive visualizations. This module builds on the successful implementation of Modules 1-5 to provide actionable intelligence for clinical teams, hospital administrators, and quality improvement initiatives.

**Timeline**: 3-4 weeks (120-160 hours)
**Complexity**: High (Real-time Analytics + Complex Visualizations + Multi-user Access)
**Prerequisites**: Modules 1-5 Complete (Especially Module 5 ML predictions operational)

---

## 🎯 Module 6 Architecture Overview

```
┌────────────────────────────────────────────────────────────────┐
│         MODULE 6: ANALYTICS & DASHBOARD ARCHITECTURE           │
└────────────────────────────────────────────────────────────────┘

                    ┌─────────────────────┐
                    │  Modules 1-5 Output │
                    │  (Kafka Topics)     │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              ▼                ▼                ▼
    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │  Raw Events  │  │  CEP Alerts  │  │  ML Predictions│
    │  (Module 1-3)│  │  (Module 4)  │  │  (Module 5)    │
    └──────┬───────┘  └──────┬───────┘  └──────┬─────────┘
           │                 │                  │
           └─────────────────┼──────────────────┘
                             │
                    ┌────────▼─────────┐
                    │  Flink SQL       │
                    │  Materialized    │
                    │  Views           │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
    │  Time-Series│  │  Aggregated │  │  Predictive │
    │  Metrics    │  │  KPIs       │  │  Analytics  │
    │  (InfluxDB) │  │  (Postgres) │  │  (Redis)    │
    └──────┬──────┘  └──────┬──────┘  └──────┬──────┘
           │                │                 │
           └────────────────┼─────────────────┘
                            │
                   ┌────────▼─────────┐
                   │  Analytics API   │
                   │  (GraphQL/REST)  │
                   └────────┬─────────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             ▼             ▼
    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
    │  Dashboard  │  │  Mobile App │  │  Pager/SMS  │
    │  (React)    │  │  (React     │  │  Alerts     │
    │             │  │   Native)   │  │             │
    └─────────────┘  └─────────────┘  └─────────────┘
```

---

## 🏗️ Implementation Phases

### Phase 1: Real-Time Analytics Engine (Week 1)
**Goal**: Create Flink SQL materialized views and time-series aggregations

### Phase 2: Dashboard API Layer (Week 2)
**Goal**: Implement GraphQL API service for dashboard data

### Phase 3: Multi-Channel Notifications (Week 2-3)
**Goal**: Build smart notification routing system

### Phase 4: Dashboard UI Components (Week 3-4)
**Goal**: Develop React dashboards with real-time updates

---

## 📦 Component 1: Real-Time Analytics Engine

### 1.1 Flink SQL Analytics Job

**File**: `src/main/java/com/cardiofit/flink/analytics/Module6_AnalyticsEngine.java`

```java
package com.cardiofit.flink.analytics;

import org.apache.flink.table.api.*;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.table.api.bridge.java.StreamTableEnvironment;

/**
 * Module 6A: Real-Time Analytics Engine
 *
 * Creates materialized views for:
 * - Patient census (1-min tumbling windows)
 * - Alert performance metrics
 * - ML model performance
 * - Department workload (1-hour sliding windows)
 * - Sepsis surveillance
 */
public class Module6_AnalyticsEngine {

    public static void main(String[] args) throws Exception {

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(16);
        env.enableCheckpointing(60000);

        StreamTableEnvironment tableEnv = StreamTableEnvironment.create(env);

        // Define source tables from existing Kafka topics
        createSourceTables(tableEnv);

        // Create materialized views
        createPatientCensusView(tableEnv);
        createAlertMetricsView(tableEnv);
        createMLPerformanceView(tableEnv);
        createDepartmentWorkloadView(tableEnv);
        createSepsisSurveillanceView(tableEnv);

        env.execute("Module 6: Real-Time Analytics Engine");
    }

    private static void createSourceTables(StreamTableEnvironment tableEnv) {
        // Raw events from Module 1-3
        tableEnv.executeSql(
            "CREATE TABLE raw_events (" +
            "  patient_id STRING," +
            "  event_type STRING," +
            "  event_timestamp TIMESTAMP(3)," +
            "  data_type STRING," +
            "  value DOUBLE," +
            "  unit STRING," +
            "  department STRING," +
            "  risk_level STRING," +
            "  WATERMARK FOR event_timestamp AS event_timestamp - INTERVAL '5' MINUTE" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'enriched-patient-events.v1'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'properties.group.id' = 'analytics-engine'," +
            "  'format' = 'json'" +
            ")"
        );

        // CEP alerts from Module 4
        tableEnv.executeSql(
            "CREATE TABLE cep_alerts (" +
            "  alert_id STRING," +
            "  patient_id STRING," +
            "  pattern_name STRING," +
            "  severity STRING," +
            "  confidence DOUBLE," +
            "  alert_timestamp TIMESTAMP(3)," +
            "  department STRING," +
            "  acknowledged BOOLEAN," +
            "  WATERMARK FOR alert_timestamp AS alert_timestamp - INTERVAL '1' MINUTE" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'clinical-patterns.v1'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'format' = 'json'" +
            ")"
        );

        // ML predictions from Module 5
        tableEnv.executeSql(
            "CREATE TABLE ml_predictions (" +
            "  prediction_id STRING," +
            "  patient_id STRING," +
            "  model_type STRING," +
            "  probability DOUBLE," +
            "  risk_category STRING," +
            "  prediction_timestamp TIMESTAMP(3)," +
            "  department STRING," +
            "  WATERMARK FOR prediction_timestamp AS prediction_timestamp - INTERVAL '2' MINUTE" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'ml-predictions.v1'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'format' = 'json'" +
            ")"
        );
    }

    private static void createPatientCensusView(StreamTableEnvironment tableEnv) {
        // Real-time patient census (1-minute tumbling window)
        tableEnv.executeSql(
            "CREATE VIEW patient_census AS " +
            "SELECT " +
            "  department," +
            "  COUNT(DISTINCT patient_id) AS active_patients," +
            "  SUM(CASE WHEN risk_level = 'HIGH' THEN 1 ELSE 0 END) AS high_risk_count," +
            "  SUM(CASE WHEN risk_level = 'CRITICAL' THEN 1 ELSE 0 END) AS critical_count," +
            "  TUMBLE_START(event_timestamp, INTERVAL '1' MINUTE) AS window_start " +
            "FROM raw_events " +
            "WHERE event_type IN ('vitals', 'labs', 'medications') " +
            "GROUP BY department, TUMBLE(event_timestamp, INTERVAL '1' MINUTE)"
        );

        // Sink to Kafka
        tableEnv.executeSql(
            "CREATE TABLE patient_census_sink (" +
            "  department STRING," +
            "  active_patients BIGINT," +
            "  high_risk_count BIGINT," +
            "  critical_count BIGINT," +
            "  window_start TIMESTAMP(3)" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'analytics-patient-census'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'format' = 'json'" +
            ")"
        );

        tableEnv.executeSql(
            "INSERT INTO patient_census_sink " +
            "SELECT * FROM patient_census"
        );
    }

    private static void createAlertMetricsView(StreamTableEnvironment tableEnv) {
        // Alert performance metrics (1-minute tumbling window)
        tableEnv.executeSql(
            "CREATE VIEW alert_metrics AS " +
            "SELECT " +
            "  department," +
            "  pattern_name," +
            "  COUNT(*) AS alert_count," +
            "  SUM(CASE WHEN acknowledged = true THEN 1 ELSE 0 END) AS acknowledged_count," +
            "  AVG(confidence) AS avg_confidence," +
            "  SUM(CASE WHEN severity = 'CRITICAL' THEN 1 ELSE 0 END) AS critical_alerts," +
            "  TUMBLE_START(alert_timestamp, INTERVAL '1' MINUTE) AS window_start " +
            "FROM cep_alerts " +
            "GROUP BY department, pattern_name, TUMBLE(alert_timestamp, INTERVAL '1' MINUTE)"
        );

        tableEnv.executeSql(
            "CREATE TABLE alert_metrics_sink (" +
            "  department STRING," +
            "  pattern_name STRING," +
            "  alert_count BIGINT," +
            "  acknowledged_count BIGINT," +
            "  avg_confidence DOUBLE," +
            "  critical_alerts BIGINT," +
            "  window_start TIMESTAMP(3)" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'analytics-alert-metrics'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'format' = 'json'" +
            ")"
        );

        tableEnv.executeSql(
            "INSERT INTO alert_metrics_sink " +
            "SELECT * FROM alert_metrics"
        );
    }

    private static void createMLPerformanceView(StreamTableEnvironment tableEnv) {
        // ML model performance (5-minute windows)
        tableEnv.executeSql(
            "CREATE VIEW ml_performance AS " +
            "SELECT " +
            "  model_type," +
            "  department," +
            "  COUNT(*) AS prediction_count," +
            "  AVG(probability) AS avg_probability," +
            "  SUM(CASE WHEN risk_category = 'HIGH' THEN 1 ELSE 0 END) AS high_risk_predictions," +
            "  SUM(CASE WHEN risk_category = 'CRITICAL' THEN 1 ELSE 0 END) AS critical_predictions," +
            "  TUMBLE_START(prediction_timestamp, INTERVAL '5' MINUTE) AS window_start " +
            "FROM ml_predictions " +
            "GROUP BY model_type, department, TUMBLE(prediction_timestamp, INTERVAL '5' MINUTE)"
        );

        tableEnv.executeSql(
            "CREATE TABLE ml_performance_sink (" +
            "  model_type STRING," +
            "  department STRING," +
            "  prediction_count BIGINT," +
            "  avg_probability DOUBLE," +
            "  high_risk_predictions BIGINT," +
            "  critical_predictions BIGINT," +
            "  window_start TIMESTAMP(3)" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'analytics-ml-performance'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'format' = 'json'" +
            ")"
        );

        tableEnv.executeSql(
            "INSERT INTO ml_performance_sink " +
            "SELECT * FROM ml_performance"
        );
    }

    private static void createDepartmentWorkloadView(StreamTableEnvironment tableEnv) {
        // Department workload (1-hour sliding window, 5-minute slide)
        tableEnv.executeSql(
            "CREATE VIEW department_workload AS " +
            "SELECT " +
            "  r.department," +
            "  COUNT(DISTINCT r.patient_id) AS total_patients," +
            "  COUNT(DISTINCT a.patient_id) AS alerted_patients," +
            "  COUNT(DISTINCT m.patient_id) AS ml_flagged_patients," +
            "  HOP_START(r.event_timestamp, INTERVAL '5' MINUTE, INTERVAL '1' HOUR) AS window_start," +
            "  HOP_END(r.event_timestamp, INTERVAL '5' MINUTE, INTERVAL '1' HOUR) AS window_end " +
            "FROM raw_events r " +
            "LEFT JOIN cep_alerts a " +
            "  ON r.patient_id = a.patient_id " +
            "  AND a.alert_timestamp BETWEEN r.event_timestamp - INTERVAL '1' HOUR AND r.event_timestamp " +
            "LEFT JOIN ml_predictions m " +
            "  ON r.patient_id = m.patient_id " +
            "  AND m.prediction_timestamp BETWEEN r.event_timestamp - INTERVAL '1' HOUR AND r.event_timestamp " +
            "  AND m.risk_category IN ('HIGH', 'CRITICAL') " +
            "GROUP BY r.department, HOP(r.event_timestamp, INTERVAL '5' MINUTE, INTERVAL '1' HOUR)"
        );

        tableEnv.executeSql(
            "CREATE TABLE department_workload_sink (" +
            "  department STRING," +
            "  total_patients BIGINT," +
            "  alerted_patients BIGINT," +
            "  ml_flagged_patients BIGINT," +
            "  window_start TIMESTAMP(3)," +
            "  window_end TIMESTAMP(3)" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'analytics-department-workload'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'format' = 'json'" +
            ")"
        );

        tableEnv.executeSql(
            "INSERT INTO department_workload_sink " +
            "SELECT * FROM department_workload"
        );
    }

    private static void createSepsisSurveillanceView(StreamTableEnvironment tableEnv) {
        // Sepsis surveillance dashboard (real-time)
        tableEnv.executeSql(
            "CREATE VIEW sepsis_surveillance AS " +
            "SELECT " +
            "  m.patient_id," +
            "  m.probability AS sepsis_probability," +
            "  m.risk_category," +
            "  r.department," +
            "  a.alert_id AS cep_alert_id," +
            "  a.severity AS alert_severity," +
            "  CASE " +
            "    WHEN a.alert_id IS NOT NULL AND m.risk_category = 'HIGH' THEN 'DUAL_CONFIRMED' " +
            "    WHEN m.risk_category = 'HIGH' THEN 'ML_ONLY' " +
            "    WHEN a.alert_id IS NOT NULL THEN 'CEP_ONLY' " +
            "    ELSE 'LOW_RISK' " +
            "  END AS detection_method," +
            "  m.prediction_timestamp " +
            "FROM ml_predictions m " +
            "LEFT JOIN raw_events r " +
            "  ON m.patient_id = r.patient_id " +
            "  AND r.event_timestamp BETWEEN m.prediction_timestamp - INTERVAL '5' MINUTE AND m.prediction_timestamp " +
            "LEFT JOIN cep_alerts a " +
            "  ON m.patient_id = a.patient_id " +
            "  AND a.pattern_name LIKE '%sepsis%' " +
            "  AND a.alert_timestamp BETWEEN m.prediction_timestamp - INTERVAL '10' MINUTE AND m.prediction_timestamp " +
            "WHERE m.model_type = 'SEPSIS_ONSET' " +
            "  AND m.probability > 0.25"
        );

        tableEnv.executeSql(
            "CREATE TABLE sepsis_surveillance_sink (" +
            "  patient_id STRING," +
            "  sepsis_probability DOUBLE," +
            "  risk_category STRING," +
            "  department STRING," +
            "  cep_alert_id STRING," +
            "  alert_severity STRING," +
            "  detection_method STRING," +
            "  prediction_timestamp TIMESTAMP(3)" +
            ") WITH (" +
            "  'connector' = 'kafka'," +
            "  'topic' = 'analytics-sepsis-surveillance'," +
            "  'properties.bootstrap.servers' = 'localhost:9092'," +
            "  'format' = 'json'" +
            ")"
        );

        tableEnv.executeSql(
            "INSERT INTO sepsis_surveillance_sink " +
            "SELECT * FROM sepsis_surveillance"
        );
    }
}
```

---

`★ Insight ─────────────────────────────────────`
**Flink SQL for Analytics**: This approach leverages Flink's SQL API to create materialized views without writing complex DataStream code. The key benefits are:
1. **Declarative Syntax**: SQL is more readable and maintainable than procedural stream processing code
2. **Automatic Optimization**: Flink's query optimizer handles execution planning
3. **Windowing Made Simple**: Tumbling and sliding windows are expressed naturally with SQL
4. **Join Operations**: Complex stream joins (raw events + alerts + predictions) are straightforward
`─────────────────────────────────────────────────`

### 1.2 Required Kafka Topics

**Create these topics for analytics outputs**:

```bash
# Create analytics output topics
kafka-topics --create --topic analytics-patient-census \
  --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1

kafka-topics --create --topic analytics-alert-metrics \
  --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1

kafka-topics --create --topic analytics-ml-performance \
  --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1

kafka-topics --create --topic analytics-department-workload \
  --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1

kafka-topics --create --topic analytics-sepsis-surveillance \
  --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1
```

---

## 📦 Component 2: GraphQL Dashboard API

### 2.1 Technology Choice

**Recommended Stack**: Node.js + Apollo Server + TypeScript + PostgreSQL

**Why Apollo Server?**
- Existing Apollo Federation infrastructure in the project
- Strong TypeScript support
- Real-time subscriptions for dashboard updates
- GraphQL schema stitching capabilities

### 2.2 Project Structure

```
backend/services/dashboard-api/
├── src/
│   ├── schema/
│   │   ├── hospital-kpis.graphql
│   │   ├── department-metrics.graphql
│   │   ├── patient-risk.graphql
│   │   └── analytics.graphql
│   ├── resolvers/
│   │   ├── hospital-kpis.resolver.ts
│   │   ├── department-metrics.resolver.ts
│   │   ├── patient-risk.resolver.ts
│   │   └── analytics.resolver.ts
│   ├── services/
│   │   ├── kafka-consumer.service.ts
│   │   ├── analytics-data.service.ts
│   │   └── cache.service.ts
│   ├── models/
│   │   └── analytics-models.ts
│   └── server.ts
├── package.json
├── tsconfig.json
└── README.md
```

### 2.3 Implementation Steps

**Step 1: Initialize Project**

```bash
cd backend/services
mkdir dashboard-api
cd dashboard-api
npm init -y
npm install @apollo/server graphql typescript ts-node @types/node
npm install kafkajs ioredis pg
npm install @types/ioredis @types/pg
```

**Step 2: GraphQL Schema** (`src/schema/hospital-kpis.graphql`)

```graphql
type HospitalKPIs {
  timestamp: String!
  totalPatients: Int!
  icuPatients: Int!
  edPatients: Int!
  lowRiskCount: Int!
  moderateRiskCount: Int!
  highRiskCount: Int!
  criticalRiskCount: Int!
  activeAlerts: Int!
  criticalAlerts: Int!
  acknowledgedRate: Float!
  avgResponseTimeMinutes: Float!
  mlPredictionsLast24h: Int!
  sepsisAlertsLast24h: Int!
  mortalityHighRiskCount: Int!
  mortality30d: Float!
  readmission30d: Float!
  avgLengthOfStay: Float!
  patientCensusTrend: String!
  riskLevelTrend: String!
  alertVolumeTrend: String!
  departmentSummaries: [DepartmentSummary!]!
}

type DepartmentSummary {
  departmentId: String!
  departmentName: String!
  patientCount: Int!
  highRiskCount: Int!
  activeAlerts: Int!
  avgRiskScore: Float!
}

type Query {
  hospitalKPIs: HospitalKPIs!
  departmentMetrics(departmentId: String!): DepartmentMetrics!
  patientRiskProfile(patientId: String!): PatientRiskProfile!
}
```

**Step 3: Kafka Consumer Service** (`src/services/kafka-consumer.service.ts`)

```typescript
import { Kafka, Consumer } from 'kafkajs';
import { Redis } from 'ioredis';

export class KafkaConsumerService {
  private kafka: Kafka;
  private consumers: Map<string, Consumer> = new Map();
  private redis: Redis;

  constructor() {
    this.kafka = new Kafka({
      clientId: 'dashboard-api',
      brokers: ['localhost:9092']
    });

    this.redis = new Redis({
      host: 'localhost',
      port: 6379
    });
  }

  async startConsumers() {
    // Patient Census Consumer
    await this.createConsumer(
      'analytics-patient-census',
      'dashboard-patient-census',
      async (message) => {
        const data = JSON.parse(message.value.toString());
        await this.redis.set(
          `census:${data.department}`,
          JSON.stringify(data),
          'EX',
          300 // 5-minute expiry
        );
      }
    );

    // Alert Metrics Consumer
    await this.createConsumer(
      'analytics-alert-metrics',
      'dashboard-alert-metrics',
      async (message) => {
        const data = JSON.parse(message.value.toString());
        await this.redis.set(
          `alerts:${data.department}`,
          JSON.stringify(data),
          'EX',
          300
        );
      }
    );

    // ML Performance Consumer
    await this.createConsumer(
      'analytics-ml-performance',
      'dashboard-ml-performance',
      async (message) => {
        const data = JSON.parse(message.value.toString());
        await this.redis.set(
          `ml:${data.model_type}:${data.department}`,
          JSON.stringify(data),
          'EX',
          300
        );
      }
    );
  }

  private async createConsumer(
    topic: string,
    groupId: string,
    messageHandler: (message: any) => Promise<void>
  ) {
    const consumer = this.kafka.consumer({ groupId });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: false });

    await consumer.run({
      eachMessage: async ({ message }) => {
        try {
          await messageHandler(message);
        } catch (error) {
          console.error(`Error processing message from ${topic}:`, error);
        }
      }
    });

    this.consumers.set(topic, consumer);
  }

  async shutdown() {
    for (const [topic, consumer] of this.consumers) {
      await consumer.disconnect();
    }
    await this.redis.quit();
  }
}
```

**Step 4: Analytics Data Service** (`src/services/analytics-data.service.ts`)

```typescript
import { Redis } from 'ioredis';
import { Pool } from 'pg';

export class AnalyticsDataService {
  private redis: Redis;
  private pg: Pool;

  constructor() {
    this.redis = new Redis({
      host: 'localhost',
      port: 6379
    });

    this.pg = new Pool({
      host: 'localhost',
      port: 5432,
      database: 'cardiofit',
      user: 'postgres',
      password: process.env.POSTGRES_PASSWORD
    });
  }

  async getHospitalKPIs() {
    // Fetch real-time metrics from Redis
    const censusData = await this.redis.mget(
      'census:ICU',
      'census:ED',
      'census:MED_SURG'
    );

    const alertData = await this.redis.mget(
      'alerts:ICU',
      'alerts:ED',
      'alerts:MED_SURG'
    );

    // Fetch historical trends from PostgreSQL
    const trendsQuery = `
      SELECT
        COUNT(DISTINCT patient_id) as patient_count_trend,
        AVG(risk_score) as avg_risk_trend
      FROM patient_metrics
      WHERE timestamp > NOW() - INTERVAL '7 days'
      GROUP BY DATE_TRUNC('day', timestamp)
      ORDER BY DATE_TRUNC('day', timestamp)
    `;

    const trends = await this.pg.query(trendsQuery);

    // Fetch 30-day outcomes
    const outcomeQuery = `
      SELECT
        COUNT(CASE WHEN died_30d = true THEN 1 END)::float / COUNT(*) as mortality_30d,
        COUNT(CASE WHEN readmitted_30d = true THEN 1 END)::float / COUNT(*) as readmission_30d,
        AVG(length_of_stay) as avg_los
      FROM patient_outcomes
      WHERE discharge_date > NOW() - INTERVAL '30 days'
    `;

    const outcomes = await this.pg.query(outcomeQuery);

    // Aggregate and return
    return {
      timestamp: new Date().toISOString(),
      totalPatients: this.aggregateCensus(censusData),
      mortality30d: outcomes.rows[0].mortality_30d,
      readmission30d: outcomes.rows[0].readmission_30d,
      avgLengthOfStay: outcomes.rows[0].avg_los,
      // ... more fields
    };
  }

  private aggregateCensus(censusData: (string | null)[]): number {
    return censusData
      .filter(d => d !== null)
      .map(d => JSON.parse(d!))
      .reduce((sum, d) => sum + d.active_patients, 0);
  }
}
```

**Step 5: Apollo Server Setup** (`src/server.ts`)

```typescript
import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { readFileSync } from 'fs';
import { join } from 'path';
import { KafkaConsumerService } from './services/kafka-consumer.service';
import { AnalyticsDataService } from './services/analytics-data.service';

// Load GraphQL schemas
const hospitalKPIsSchema = readFileSync(
  join(__dirname, 'schema/hospital-kpis.graphql'),
  'utf-8'
);

// Resolvers
const resolvers = {
  Query: {
    hospitalKPIs: async () => {
      const analyticsService = new AnalyticsDataService();
      return await analyticsService.getHospitalKPIs();
    },

    departmentMetrics: async (_: any, { departmentId }: { departmentId: string }) => {
      const analyticsService = new AnalyticsDataService();
      return await analyticsService.getDepartmentMetrics(departmentId);
    },

    patientRiskProfile: async (_: any, { patientId }: { patientId: string }) => {
      const analyticsService = new AnalyticsDataService();
      return await analyticsService.getPatientRiskProfile(patientId);
    }
  }
};

async function startServer() {
  // Start Kafka consumers
  const kafkaService = new KafkaConsumerService();
  await kafkaService.startConsumers();

  // Create Apollo Server
  const server = new ApolloServer({
    typeDefs: hospitalKPIsSchema,
    resolvers
  });

  const { url } = await startStandaloneServer(server, {
    listen: { port: 4001 }
  });

  console.log(`🚀 Dashboard API ready at ${url}`);

  // Graceful shutdown
  process.on('SIGTERM', async () => {
    await kafkaService.shutdown();
    await server.stop();
  });
}

startServer().catch(console.error);
```

---

`★ Insight ─────────────────────────────────────`
**Real-Time Data Architecture**: The dashboard API uses a hybrid approach:
1. **Redis Cache**: Stores real-time analytics from Kafka (last 5 minutes)
2. **PostgreSQL**: Stores historical trends and outcomes (7-30 days)
3. **Kafka Consumers**: Continuously update Redis with latest analytics

This design provides:
- **Low Latency**: Redis reads are <1ms
- **Historical Context**: PostgreSQL handles complex trend queries
- **Scalability**: Kafka consumers can run in parallel
`─────────────────────────────────────────────────`

---

## 📦 Component 3: Multi-Channel Notification System

### 3.1 Notification Router Implementation

**File**: `src/main/java/com/cardiofit/flink/notifications/NotificationRouter.java`

```java
package com.cardiofit.flink.notifications;

import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;

/**
 * Smart notification router
 * Routes alerts to appropriate channels based on:
 * - Alert severity
 * - User role and preferences
 * - On-call schedule
 * - Alert fatigue mitigation
 */
public class NotificationRouter extends ProcessFunction<Alert, Notification> {

    private final NotificationRuleEngine ruleEngine;
    private final OnCallScheduleService scheduleService;
    private final UserPreferenceService preferenceService;
    private final AlertFatigueTracker fatigueTracker;

    @Override
    public void processElement(
        Alert alert,
        Context ctx,
        Collector<Notification> out) throws Exception {

        // Step 1: Check alert fatigue
        if (fatigueTracker.shouldSuppress(alert)) {
            logSuppression(alert, "ALERT_FATIGUE");
            return;
        }

        // Step 2: Determine target users
        List<User> targetUsers = determineTargetUsers(alert);

        // Step 3: For each user, determine notification channels
        for (User user : targetUsers) {
            UserPreferences prefs = preferenceService.getPreferences(user.getUserId());
            List<NotificationChannel> channels = ruleEngine.determineChannels(
                alert, user, prefs
            );

            // Create notifications for each channel
            for (NotificationChannel channel : channels) {
                Notification notification = Notification.builder()
                    .notificationId(UUID.randomUUID().toString())
                    .alertId(alert.getAlertId())
                    .userId(user.getUserId())
                    .channel(channel)
                    .priority(calculatePriority(alert, user, channel))
                    .message(formatMessage(alert, channel))
                    .createdAt(System.currentTimeMillis())
                    .status("PENDING")
                    .build();

                out.collect(notification);
                fatigueTracker.recordNotification(user.getUserId(), alert);
            }
        }

        // Step 4: Check if escalation is needed
        if (requiresEscalation(alert)) {
            scheduleEscalation(alert, ctx);
        }
    }

    private List<User> determineTargetUsers(Alert alert) {
        List<User> users = new ArrayList<>();

        // Critical alerts → Attending physician + Charge nurse
        if (alert.getSeverity().equals("CRITICAL")) {
            users.addAll(scheduleService.getOnCallPhysicians(alert.getDepartment()));
            users.addAll(scheduleService.getChargeNurses(alert.getDepartment()));
        }
        // High severity → Primary nurse + Resident
        else if (alert.getSeverity().equals("HIGH")) {
            User primaryNurse = scheduleService.getPrimaryNurse(alert.getPatientId());
            if (primaryNurse != null) users.add(primaryNurse);

            User resident = scheduleService.getAssignedResident(alert.getDepartment());
            if (resident != null) users.add(resident);
        }

        return users;
    }
}
```

### 3.2 Alert Fatigue Mitigation

**File**: `src/main/java/com/cardiofit/flink/notifications/AlertFatigueTracker.java`

```java
package com.cardiofit.flink.notifications;

/**
 * Tracks alert volume and implements fatigue mitigation
 * - Rate limiting per user (max 20 alerts/hour)
 * - Duplicate suppression (5-minute window)
 * - Alert bundling (3+ similar alerts)
 */
public class AlertFatigueTracker {

    private final int MAX_ALERTS_PER_HOUR = 20;
    private final long DUPLICATE_WINDOW_MS = 5 * 60 * 1000;  // 5 minutes

    public boolean shouldSuppress(Alert alert) {
        String userId = alert.getAssignedUserId();
        if (userId == null) return false;

        UserAlertHistory history = getUserHistory(userId);

        // Check rate limit
        if (history.getAlertsInLastHour() >= MAX_ALERTS_PER_HOUR) {
            // Only allow CRITICAL alerts through
            if (!alert.getSeverity().equals("CRITICAL")) {
                return true;
            }
        }

        // Check for duplicates
        if (isDuplicate(alert, history)) {
            return true;
        }

        return false;
    }

    private boolean isDuplicate(Alert alert, UserAlertHistory history) {
        long now = System.currentTimeMillis();

        return history.getRecentAlerts().stream()
            .filter(a -> now - a.getCreatedAt() < DUPLICATE_WINDOW_MS)
            .anyMatch(a ->
                a.getPatientId().equals(alert.getPatientId()) &&
                a.getAlertType().equals(alert.getAlertType()) &&
                a.getSeverity().equals(alert.getSeverity())
            );
    }
}
```

### 3.3 Multi-Channel Delivery Service

**Technology**: Spring Boot service with Twilio/SendGrid integration

**File**: `backend/services/notification-delivery-service/`

```java
package com.cardiofit.notifications.delivery;

import org.springframework.stereotype.Service;
import com.twilio.Twilio;
import com.twilio.rest.api.v2010.account.Message;

@Service
public class NotificationDeliveryService {

    private final TwilioConfig twilioConfig;
    private final SendGridConfig sendGridConfig;

    public DeliveryResult send(Notification notification) {
        try {
            switch (notification.getChannel()) {
                case SMS:
                    return sendSMS(notification);
                case EMAIL:
                    return sendEmail(notification);
                case PUSH_NOTIFICATION:
                    return sendPush(notification);
                case PAGER:
                    return sendPager(notification);
                default:
                    return DeliveryResult.failure("Unsupported channel");
            }
        } catch (Exception e) {
            return DeliveryResult.failure(e.getMessage());
        }
    }

    private DeliveryResult sendSMS(Notification notification) {
        User user = getUserById(notification.getUserId());

        Message message = Message.creator(
            new PhoneNumber(user.getPhoneNumber()),
            new PhoneNumber(twilioConfig.getFromNumber()),
            notification.getMessage()
        ).create();

        return DeliveryResult.success(message.getSid());
    }
}
```

---

## 📦 Component 4: React Dashboard UI

### 4.1 Project Setup

```bash
cd backend/services
npx create-react-app dashboard-ui --template typescript
cd dashboard-ui
npm install @apollo/client graphql
npm install @mui/material @emotion/react @emotion/styled
npm install recharts @mui/icons-material
npm install ws
```

### 4.2 Apollo Client Setup

**File**: `src/apollo-client.ts`

```typescript
import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';

export const apolloClient = new ApolloClient({
  link: new HttpLink({
    uri: 'http://localhost:4001/graphql'
  }),
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'cache-and-network',
    },
  },
});
```

### 4.3 Executive Dashboard Component

**File**: `src/components/ExecutiveDashboard.tsx`

```typescript
import React, { useState, useEffect } from 'react';
import { useQuery, gql } from '@apollo/client';
import { Grid, Card, CardContent, Typography, Box } from '@mui/material';
import { LineChart, Line, BarChart, Bar, PieChart, Pie } from 'recharts';

const GET_HOSPITAL_KPIS = gql`
  query GetHospitalKPIs {
    hospitalKPIs {
      timestamp
      totalPatients
      criticalRiskCount
      activeAlerts
      mortality30d
      departmentSummaries {
        departmentName
        patientCount
        highRiskCount
      }
    }
  }
`;

export const ExecutiveDashboard: React.FC = () => {
  const { loading, error, data, refetch } = useQuery(GET_HOSPITAL_KPIS, {
    pollInterval: 30000  // Refresh every 30 seconds
  });

  useEffect(() => {
    // Setup WebSocket for real-time updates
    const ws = new WebSocket('ws://localhost:8080/dashboard/realtime');

    ws.onmessage = (event) => {
      const update = JSON.parse(event.data);
      if (update.type === 'KPI_UPDATE') {
        refetch();
      }
    };

    return () => ws.close();
  }, [refetch]);

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  const kpis = data.hospitalKPIs;

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Hospital-Wide Dashboard
      </Typography>

      <Grid container spacing={3}>
        {/* Total Census Card */}
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h6">Total Census</Typography>
              <Typography variant="h3">{kpis.totalPatients}</Typography>
            </CardContent>
          </Card>
        </Grid>

        {/* Critical Patients Card */}
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h6">Critical Risk</Typography>
              <Typography variant="h3" color="error">
                {kpis.criticalRiskCount}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        {/* Department Comparison Chart */}
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h6">Department Overview</Typography>
              <BarChart width={600} height={300} data={kpis.departmentSummaries}>
                <Bar dataKey="patientCount" fill="#1976d2" />
                <Bar dataKey="highRiskCount" fill="#f44336" />
              </BarChart>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};
```

---

## 🚀 Deployment Guide

### Step 1: Build and Deploy Flink Jobs

```bash
cd backend/shared-infrastructure/flink-processing
mvn clean package

# Deploy to Flink cluster
flink run -c com.cardiofit.flink.analytics.Module6_AnalyticsEngine \
  target/flink-ehr-intelligence-1.0.0.jar
```

### Step 2: Deploy Dashboard API

```bash
cd backend/services/dashboard-api
npm run build
pm2 start dist/server.js --name dashboard-api
```

### Step 3: Deploy Notification Service

```bash
cd backend/services/notification-delivery-service
mvn clean package
java -jar target/notification-delivery-service-1.0.0.jar
```

### Step 4: Deploy React Dashboard

```bash
cd backend/services/dashboard-ui
npm run build
# Serve with nginx or deploy to Vercel/Netlify
```

---

## 📊 Testing Strategy

### Integration Tests

```java
@Test
public void testAnalyticsPipeline() {
    // 1. Send test events to enriched-patient-events.v1
    // 2. Wait for analytics topics to receive aggregated data
    // 3. Verify Redis contains expected metrics
    // 4. Query GraphQL API and validate response
}
```

### Performance Tests

```bash
# Load test with k6
k6 run --vus 100 --duration 30s dashboard-load-test.js
```

---

## 📝 Next Steps

1. ✅ Complete Module 6A (Analytics Engine) - Week 1
2. ✅ Implement Module 6B (Dashboard API) - Week 2
3. ✅ Build Module 6C (Notifications) - Week 2-3
4. ✅ Develop Module 6D (Dashboard UI) - Week 3-4
5. 🔄 Integration testing across all modules
6. 📊 Performance optimization and monitoring
7. 📚 Documentation and training materials

---

## 🎓 Summary

Module 6 transforms raw clinical data into actionable intelligence through:
- **Real-time analytics** with Flink SQL materialized views
- **GraphQL API** for flexible dashboard queries
- **Smart notifications** with fatigue mitigation
- **Interactive dashboards** with live updates

This completes the CardioFit Flink processing pipeline from ingestion (Module 1) through predictive analytics (Module 5) to real-time dashboards (Module 6).
