# ✅ Corrected Alert Architecture

## 📋 Summary of Changes

### ❌ Problem Identified
- **Module 5** was outputting ML predictions to `alert-management.v1`
- **Module 4** was also outputting PatternEvents to `alert-management.v1`
- **Module 6 Alert Composition** expected SimpleAlerts from `alert-management.v1`
- **Result**: Topic had mixed data formats → Deserialization failures

### ✅ Solution Applied

**Changed Module 5 output** from `alert-management.v1` → `ml-risk-alerts.v1`

**File**: `Module5_MLInference.java` Line 1013

---

## 🎯 Corrected Data Flow

### Module 2 (Enhanced Clinical Reasoning)
```
Purpose: Context building and enrichment
Outputs:
├─ EnrichedEvent → clinical-patterns.v1 ✅
└─ ProtocolEvent → protocol-triggers.v1 ✅

NO ALERTS created by Module 2
```

### Module 4 (Pattern Detection / CEP)
```
Purpose: Create alerts from clinical patterns
Outputs:
└─ PatternEvent → alert-management.v1 ✅

This is the CORRECT use of alert-management.v1
```

### Module 5 (ML Inference)
```
Purpose: ML risk predictions
Outputs:
├─ MLPrediction → inference-results.v1 ✅ (all predictions)
└─ MLPrediction → ml-risk-alerts.v1 ✅ (high-risk only) NEW!

FIXED: No longer pollutes alert-management.v1
```

### Module 6 Alert Composition
```
Purpose: Process and deduplicate alerts
Inputs:
├─ SimpleAlert from alert-management.v1 (expects PatternEvent)
└─ PatternEvent from clinical-patterns.v1 ✅

Outputs:
└─ ComposedAlert → composed-alerts.v1
```

### Module 6.3 Analytics
```
Purpose: Aggregate metrics for dashboards
Inputs:
├─ ComposedAlert from composed-alerts.v1
└─ MLPrediction from inference-results.v1

Outputs:
├─ Alert metrics → analytics-alert-metrics (with patient_ids) ✅
└─ ML performance → analytics-ml-performance (with patient_ids) ✅
```

---

## 📊 Topic Usage Map

```
clinical-patterns.v1:
├─ Source: Module 2
├─ Format: EnrichedEvent (with clinical patterns)
└─ Consumers: Module 6 Alert Composition ✅

alert-management.v1:
├─ Source: Module 4 ONLY ✅
├─ Format: PatternEvent (clinical alerts)
└─ Consumers: Module 6 Alert Composition ✅

ml-risk-alerts.v1: NEW!
├─ Source: Module 5
├─ Format: MLPrediction (high-risk predictions)
└─ Consumers: (future alert routing system)

inference-results.v1:
├─ Source: Module 5
├─ Format: MLPrediction (all predictions)
└─ Consumers: Module 6.3 Analytics ✅

composed-alerts.v1:
├─ Source: Module 6 Alert Composition
├─ Format: ComposedAlert (deduplicated/enhanced alerts)
└─ Consumers: Module 6.3 Analytics ✅

analytics-alert-metrics:
├─ Source: Module 6.3 Analytics
├─ Format: Alert metrics with patient_ids
└─ Consumers: Dashboard applications

analytics-ml-performance:
├─ Source: Module 6.3 Analytics
├─ Format: ML performance metrics with patient_ids
└─ Consumers: Dashboard applications
```

---

## 🔧 Next Steps

1. **Rebuild JAR** with Module 5 fix:
   ```bash
   cd backend/shared-infrastructure/flink-processing
   mvn clean package
   ```

2. **Create new Kafka topic** (if not auto-created):
   ```bash
   docker exec kafka kafka-topics --create \
     --topic ml-risk-alerts.v1 \
     --bootstrap-server localhost:9092 \
     --partitions 4 \
     --replication-factor 1
   ```

3. **Fix Module 6 Alert Composition**:
   - Remove SimpleAlert source from alert-management.v1
   - Use only PatternEvent source from clinical-patterns.v1

4. **Deploy updated modules**:
   - Upload new JAR to Flink
   - Deploy Module 5 with ml-risk-alerts.v1 output
   - Deploy Module 6 Alert Composition (after fixing)
   - Deploy Module 6.3 Analytics (already has patient_ids field)

5. **Verify data flow**:
   - Check alert-management.v1 has only PatternEvents
   - Check ml-risk-alerts.v1 receives high-risk ML predictions
   - Check composed-alerts.v1 receives new ComposedAlerts
   - Check analytics-alert-metrics produces metrics with patient_ids

---

## ✅ Expected Results After Fix

```
✅ Module 2 → clinical-patterns.v1 (EnrichedEvent)
✅ Module 4 → alert-management.v1 (PatternEvent ONLY)
✅ Module 5 → ml-risk-alerts.v1 (MLPrediction high-risk)
✅ Module 5 → inference-results.v1 (MLPrediction all)
✅ Module 6 Alert Composition → composed-alerts.v1 (ComposedAlert)
✅ Module 6.3 Analytics → analytics-alert-metrics (with patient_ids)
```

No more data format mismatches!
No more deserialization failures!
Alert metrics will update in real-time!
