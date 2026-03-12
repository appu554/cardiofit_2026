# Dashboard Comprehensive Fix Guide

## Executive Summary

**Date**: November 10, 2025
**Status**: Partial Implementation - Config Updated, Consumer Logic Needed

### Root Cause Analysis

1. **Executive Dashboard Issues**:
   - ✅ Hospital KPIs query works perfectly - returns data from 718 Kafka messages
   - ⚠️ Department metrics has timestamp resolver bug (FIXED but needs rebuild/restart)
   - ✅ GraphQL queries match schema correctly

2. **Clinical Dashboard Patient List Issue**:
   - ❌ Patient Risk Profiles consumer reads from `analytics-patient-census` (aggregated data)
   - ✅ Individual patient data exists in `patient-events-v1` topic (3,209 messages)
   - ❌ No consumer implemented for individual patient events

### Data Available in Kafka

```
✅ analytics-patient-census (718 msgs)     → Hospital aggregated data (WORKING)
✅ analytics-department-workload           → Department aggregated data (FIXED)
✅ analytics-ml-performance                → Quality metrics (WORKING)
✅ analytics-sepsis-surveillance           → Sepsis alerts
✅ patient-events-v1 (3,209 msgs)          → Individual patient events (NEEDS CONSUMER)
```

### Sample Patient Event Structure

```json
{
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1762668737000,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 77,
    "bp": "136/64"
  },
  "metadata": {
    "source": "ContinuousTest",
    "batch": 1
  }
}
```

## Changes Made So Far

### 1. Fixed Department Metrics Timestamp Resolver
**File**: `src/resolvers/department-metrics.resolver.ts`

```typescript
DepartmentMetrics: {
  timestamp: (parent: DepartmentMetrics) => {
    if (parent.timestamp instanceof Date) {
      return parent.timestamp.toISOString();
    }
    if (typeof parent.timestamp === 'string') {
      return parent.timestamp;
    }
    if (typeof parent.timestamp === 'number') {
      return new Date(parent.timestamp).toISOString();
    }
    return new Date().toISOString();
  },
}
```

### 2. Updated Kafka Consumer Configuration
**File**: `src/config/index.ts`

```typescript
kafka: {
  // ... existing config
  topics: {
    hospitalKpis: 'analytics-patient-census',
    departmentMetrics: 'analytics-department-workload',
    patientRiskProfiles: 'analytics-patient-census', // Keep for now
    patientEvents: 'patient-events-v1', // NEW: Raw patient events
    sepsisSurveillance: 'analytics-sepsis-surveillance',
    qualityMetrics: 'analytics-ml-performance',
  },
}
```

### 3. Updated Kafka Consumers to Read from Beginning
**File**: `src/services/kafka-consumer.service.ts`
**File**: `.env`

All consumers now use:
- `fromBeginning: true` - Read historical messages
- `KAFKA_GROUP_ID=dashboard-api-consumers-v2` - Fresh consumer group

## Required Implementation

### Step 1: Add Patient Events Consumer

**Location**: `src/services/kafka-consumer.service.ts`

Add to `start()` method (line 44):
```typescript
async start(): Promise<void> {
  // ... existing consumers
  await this.startPatientEventsConsumer(); // ADD THIS LINE

  this.isRunning = true;
  logger.info('All Kafka consumers started successfully');
}
```

Add new consumer method after line 186:
```typescript
private async startPatientEventsConsumer(): Promise<void> {
  const topic = config.kafka.topics.patientEvents;
  const consumer = this.kafka.consumer({
    groupId: `${config.kafka.groupId}-patient-events`,
  });

  await consumer.connect();
  await consumer.subscribe({ topic, fromBeginning: true });

  await consumer.run({
    eachMessage: async (payload: EachMessagePayload) => {
      try {
        const message = this.parseMessage<any>(payload);
        if (message) {
          logger.info({ topic, patientId: message.patient_id }, 'Processing patient event');
          await this.processPatientEvent(message);
        }
      } catch (error) {
        logger.error({ error, topic }, 'Error processing patient event');
      }
    },
  });

  this.consumers.set(topic, consumer);
  logger.info({ topic }, 'Patient events consumer started');
}
```

### Step 2: Add Patient Event Transformation Logic

Add after the existing transformation methods (around line 297):

```typescript
/**
 * Transform raw patient events into patient risk profiles
 * Aggregates events per patient and calculates risk scores
 */
private transformPatientEventToRiskProfile(event: any): PatientRiskProfile | null {
  try {
    const patientId = event.patient_id;
    if (!patientId) return null;

    // Calculate basic risk score from vital signs
    let riskScore = 50; // Base risk
    let riskLevel: 'LOW' | 'MODERATE' | 'HIGH' | 'CRITICAL' = 'MODERATE';

    if (event.type === 'vital_signs' && event.payload) {
      const heartRate = event.payload.heart_rate;

      // Elevated heart rate increases risk
      if (heartRate > 100) riskScore += 20;
      if (heartRate > 120) riskScore += 10;

      // Parse blood pressure
      if (event.payload.bp) {
        const [systolic] = event.payload.bp.split('/').map(Number);
        if (systolic > 140) riskScore += 15;
        if (systolic < 90) riskScore += 20;
      }
    }

    // Determine risk level
    if (riskScore < 40) riskLevel = 'LOW';
    else if (riskScore < 60) riskLevel = 'MODERATE';
    else if (riskScore < 80) riskLevel = 'HIGH';
    else riskLevel = 'CRITICAL';

    return {
      patientId,
      hospitalId: 'HOSPITAL-001',
      departmentId: this.determineDepartment(patientId), // Helper method
      timestamp: new Date(event.event_time || Date.now()),
      overallRiskScore: riskScore,
      riskLevel,
      riskCategory: `${riskLevel}_RISK`,
      mortalityRisk: riskScore * 0.8,
      readmissionRisk: riskScore * 0.6,
      deteriorationRisk: riskScore * 0.9,
      sepsisRisk: riskScore * 0.5,
      fallRisk: riskScore * 0.4,
      bleedingRisk: riskScore * 0.3,
      vitalSigns: event.type === 'vital_signs' ? {
        heartRate: event.payload?.heart_rate,
        bloodPressureSystolic: event.payload?.bp ? parseInt(event.payload.bp.split('/')[0]) : undefined,
        bloodPressureDiastolic: event.payload?.bp ? parseInt(event.payload.bp.split('/')[1]) : undefined,
        timestamp: new Date(event.event_time || Date.now()),
      } : undefined,
      activeAlerts: [],
      alertCount: riskLevel === 'CRITICAL' ? 2 : riskLevel === 'HIGH' ? 1 : 0,
      criticalAlertCount: riskLevel === 'CRITICAL' ? 1 : 0,
      comorbidities: [],
      activeConditions: [],
      recommendedInterventions: this.getInterventions(riskLevel),
      activeInterventions: [],
      lastUpdated: new Date(),
    };
  } catch (error) {
    logger.error({ error, event }, 'Failed to transform patient event');
    return null;
  }
}

private determineDepartment(patientId: string): string {
  // Simple hash-based department assignment for demo
  const hash = patientId.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  const departments = ['ICU', 'Emergency', 'Cardiology', 'General'];
  return departments[hash % departments.length];
}

private getInterventions(riskLevel: string): string[] {
  switch (riskLevel) {
    case 'CRITICAL':
      return ['Immediate physician review', 'ICU transfer evaluation', 'Vital signs q15min'];
    case 'HIGH':
      return ['Frequent monitoring', 'Lab work review', 'Medication adjustment'];
    case 'MODERATE':
      return ['Standard monitoring', 'Daily assessment'];
    default:
      return ['Routine care'];
  }
}
```

### Step 3: Add Processing Method

Add after transformation methods:

```typescript
private async processPatientEvent(event: any): Promise<void> {
  try {
    const patientProfile = this.transformPatientEventToRiskProfile(event);
    if (!patientProfile) return;

    // Cache in Redis
    const cacheKey = `patient-risk:${patientProfile.patientId}:latest`;
    await cacheService.set(cacheKey, patientProfile);

    // Store in PostgreSQL
    await analyticsDataService.storePatientRiskProfile(patientProfile);

    logger.info(
      { patientId: patientProfile.patientId, riskLevel: patientProfile.riskLevel },
      'Patient risk profile updated'
    );
  } catch (error) {
    logger.error({ error, event }, 'Failed to process patient event');
  }
}
```

### Step 4: Update Health Check

Update line 506:
```typescript
async healthCheck(): Promise<boolean> {
  return this.isRunning && this.consumers.size === 6; // Changed from 5 to 6
}
```

## Deployment Steps

### 1. Apply Code Changes
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/module6-services/dashboard-api

# Apply the changes above to:
# - src/services/kafka-consumer.service.ts
# - src/config/index.ts (already done)
```

### 2. Rebuild and Restart
```bash
# Clean up old processes
lsof -ti :8050 | xargs kill -9

# Rebuild with new changes
npm run build

# Start server
npm start &

# Wait for consumers to start (watch logs)
tail -f logs/dashboard-api.log
```

### 3. Verify Consumer Started
```bash
# Should show 6 consumers started
curl http://localhost:8050/health
```

### 4. Check Patient Data Flowing
```bash
# Query for patients
curl -X POST http://localhost:8050/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ highRiskPatients(hospitalId: \"HOSPITAL-001\", limit: 10) { patientId riskLevel overallRiskScore } }"}'
```

### 5. Test Frontend Dashboards
```bash
# Start Dashboard UI
cd ../dashboard-ui
npm run dev &

# Open browser to http://localhost:3000
# Navigate to:
# 1. Executive Dashboard - Should show hospital KPIs and department metrics
# 2. Clinical Dashboard - Should show patient list with 3,209 patients
# 3. Quality Dashboard - Should show quality metrics
```

## Expected Results

### Executive Dashboard
- ✅ Hospital KPIs card showing: 100 beds, 1 patient, 4 admissions
- ✅ Real-time stats: 1 active patient, 99 available beds
- ✅ Department metrics table with multiple departments

### Clinical Dashboard
- ✅ Patient list table with up to 100 patients (limit)
- ✅ Filterable by department (ICU, Emergency, Cardiology, General)
- ✅ Risk level filtering (LOW, MODERATE, HIGH, CRITICAL)
- ✅ Search by patient ID
- ✅ Auto-refresh every 30 seconds

### Quality Metrics Dashboard
- ✅ Sepsis bundle compliance metrics
- ✅ Quality indicators
- ✅ Compliance rates

## Troubleshooting

### Issue: Patient list still empty
**Check**:
```bash
# Verify consumer is running
docker exec kafka kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group dashboard-api-consumers-v2-patient-events

# Check database
psql -h localhost -p 5433 -U cardiofit -d cardiofit_analytics
SELECT COUNT(*) FROM patient_risk_profiles;
```

### Issue: High memory usage
**Solution**: Implement event aggregation window
```typescript
// Keep in-memory map of last event per patient
private patientEventCache = new Map<string, any>();

// Only process if event is newer than cached version
if (this.shouldProcessEvent(event)) {
  await this.processPatientEvent(event);
}
```

### Issue: Duplicate patients
**Solution**: Use UPSERT in database
```sql
INSERT INTO patient_risk_profiles (patient_id, ...)
VALUES ($1, ...)
ON CONFLICT (patient_id)
DO UPDATE SET ...
```

## Performance Optimizations

### 1. Batch Processing
Process events in batches of 100:
```typescript
private eventBatch: any[] = [];

private async processPatientEvent(event: any): Promise<void> {
  this.eventBatch.push(event);

  if (this.eventBatch.length >= 100) {
    await this.flushBatch();
  }
}
```

### 2. Selective Storage
Only store patients with events in last 24 hours:
```typescript
const eventAge = Date.now() - event.event_time;
if (eventAge > 24 * 60 * 60 * 1000) {
  return; // Skip old events
}
```

### 3. Redis Expiry
Set TTL on patient cache entries:
```typescript
await cacheService.set(cacheKey, patientProfile, 3600); // 1 hour TTL
```

## Next Steps

1. **Implement the code changes above**
2. **Test each dashboard individually**
3. **Optimize based on performance metrics**
4. **Add monitoring and alerting**
5. **Document for production deployment**

## Files Modified

1. ✅ `src/config/index.ts` - Added patientEvents topic
2. ✅ `src/resolvers/department-metrics.resolver.ts` - Fixed timestamp resolver
3. ✅ `.env` - Updated consumer group ID
4. ⏳ `src/services/kafka-consumer.service.ts` - Need to add patient events consumer

## Success Criteria

- [x] Hospital KPIs display correctly
- [ ] Department metrics display correctly (needs server restart)
- [ ] Patient list populates with 3,209 patients
- [ ] All dashboards auto-refresh every 30 seconds
- [ ] No errors in console logs
- [ ] GraphQL queries return data within 2 seconds
- [ ] Memory usage stays under 512MB
