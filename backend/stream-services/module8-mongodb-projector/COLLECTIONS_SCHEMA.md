# MongoDB Collections Schema Reference

## Database: `module8_clinical`

### Overview

The MongoDB Projector creates and maintains three collections optimized for different access patterns:

1. **clinical_documents**: Full event storage with rich metadata
2. **patient_timelines**: Aggregated patient event histories (max 1000 events)
3. **ml_explanations**: ML model predictions with interpretability data

---

## Collection 1: `clinical_documents`

### Purpose
Primary storage for all clinical events with complete enrichment data, vital signs, lab results, and ML predictions.

### Document Schema

```javascript
{
  // Primary identifiers
  _id: "evt_20250115_123456_abc123",           // Event ID (unique)
  patientId: "patient_12345",                  // Patient identifier
  timestamp: ISODate("2025-01-15T10:30:45Z"),  // Event timestamp
  eventType: "vital_signs",                    // Event type enum
  deviceType: "patient_monitor",               // Device type (optional)

  // Original clinical data
  vitalSigns: {
    heartRate: 85,                             // bpm
    bloodPressureSystolic: 120,                // mmHg
    bloodPressureDiastolic: 80,                // mmHg
    temperature: 37.2,                         // Celsius
    oxygenSaturation: 98,                      // %
    respiratoryRate: 16                        // breaths/min
  },

  labResults: {                                // Lab test results (optional)
    whiteBloodCellCount: 12.5,
    cReactiveProtein: 45,
    lactate: 2.1
  },

  // Semantic enrichments from Module 8
  enrichments: {
    riskLevel: "NORMAL",                       // NORMAL|ELEVATED|HIGH|CRITICAL
    earlyWarningScore: 2,                      // NEWS score
    clinicalContext: {
      setting: "ICU",
      admissionDate: "2024-01-01T00:00:00Z",
      diagnosis: "Sepsis"
    },
    deviceContext: {
      manufacturer: "Philips",
      model: "IntelliVue MX800",
      firmwareVersion: "2.1.0"
    }
  },

  // ML predictions and interpretability
  mlPredictions: {
    predictions: {
      sepsis_risk_24h: {
        modelName: "sepsis_xgboost_v1",
        prediction: 0.35,                      // Risk score [0-1]
        confidence: 0.82,                      // Model confidence
        threshold: 0.5,                        // Alert threshold
        alertTriggered: false,                 // Alert status
        shapValues: {                          // SHAP feature importance
          heart_rate: 0.05,
          temperature: 0.12,
          wbc_count: 0.08
        },
        limeExplanation: {                     // LIME local explanation
          features: ["heart_rate", "temperature"],
          weights: [0.05, 0.12]
        }
      },
      mortality_risk_48h: {
        modelName: "mortality_lgbm_v1",
        prediction: 0.15,
        confidence: 0.88,
        threshold: 0.3,
        alertTriggered: false
      }
    },
    featureImportance: {                       // Global feature importance
      heart_rate: 0.25,
      temperature: 0.18,
      wbc_count: 0.15
    }
  },

  // Processing metadata
  ingestionTime: ISODate("2025-01-15T10:30:40Z"),
  processingTime: ISODate("2025-01-15T10:30:42Z"),
  createdAt: ISODate("2025-01-15T10:30:45Z"),

  // Human-readable summary (auto-generated)
  summary: "vital_signs event | Vitals: HR 85, BP 120/80, Temp 37.2°C, SpO2 98% | Risk: NORMAL"
}
```

### Indexes

```javascript
// 1. Patient timeline queries (compound)
{ patientId: 1, timestamp: -1 }
// Usage: db.clinical_documents.find({patientId: "patient_123"}).sort({timestamp: -1})

// 2. Event type filtering
{ eventType: 1 }
// Usage: db.clinical_documents.find({eventType: "vital_signs"})

// 3. Risk level queries
{ "enrichments.riskLevel": 1 }
// Usage: db.clinical_documents.find({"enrichments.riskLevel": "CRITICAL"})

// 4. Temporal queries
{ timestamp: -1 }
// Usage: db.clinical_documents.find().sort({timestamp: -1}).limit(100)
```

### Estimated Size
- **Document Size**: ~2-5 KB per document (varies with enrichments)
- **1M events**: ~3-5 GB storage
- **Index Size**: ~200-400 MB per 1M documents

---

## Collection 2: `patient_timelines`

### Purpose
Fast patient timeline retrieval without scanning full event history. Maintains latest 1000 events per patient in sorted array.

### Document Schema

```javascript
{
  // Patient ID as primary key
  _id: "patient_12345",

  // Array of latest 1000 events (sorted newest first)
  events: [
    {
      eventId: "evt_20250115_123456_abc123",
      timestamp: ISODate("2025-01-15T10:30:45Z"),
      eventType: "vital_signs",
      summary: "HR 85, BP 120/80, Temp 37.2°C",
      riskLevel: "NORMAL",                     // Optional
      vitalSigns: {                            // Optional summary
        heartRate: 85,
        bloodPressureSystolic: 120,
        bloodPressureDiastolic: 80,
        temperature: 37.2,
        oxygenSaturation: 98
      },
      predictions: {                           // Optional ML scores
        sepsis_risk_24h: 0.35,
        mortality_risk_48h: 0.15
      }
    },
    // ... up to 1000 events (oldest removed automatically)
  ],

  // Timeline metadata
  lastUpdated: ISODate("2025-01-15T10:30:45Z"),
  eventCount: 1523,                            // Total events ever recorded
  firstEventTime: ISODate("2024-01-01T08:00:00Z"),
  latestEventTime: ISODate("2025-01-15T10:30:45Z")
}
```

### Update Strategy

Uses MongoDB's `$push` with array operators for automatic array management:

```javascript
{
  $push: {
    events: {
      $each: [new_event],        // Add new event
      $sort: { timestamp: -1 },  // Keep sorted (newest first)
      $slice: 1000               // Limit to 1000 events (auto-remove oldest)
    }
  },
  $set: {
    lastUpdated: new Date(),
    latestEventTime: event.timestamp
  },
  $inc: { eventCount: 1 },
  $setOnInsert: { firstEventTime: event.timestamp }
}
```

### Indexes

```javascript
// 1. Primary key (patient ID)
{ _id: 1 }
// Usage: db.patient_timelines.findOne({_id: "patient_123"})

// 2. Recently active patients
{ lastUpdated: -1 }
// Usage: db.patient_timelines.find().sort({lastUpdated: -1}).limit(10)
```

### Estimated Size
- **Document Size**: ~100-200 KB per patient (1000 events * ~150 bytes)
- **10K patients**: ~1-2 GB storage
- **Index Size**: ~10 MB per 10K patients

### Performance Characteristics
- **Single query retrieval**: Get patient's full timeline in one document read
- **Automatic cleanup**: Oldest events automatically removed when limit exceeded
- **Fast updates**: In-place array updates without document relocation

---

## Collection 3: `ml_explanations`

### Purpose
ML model predictions with interpretability data (SHAP values, LIME explanations) for clinical AI audit trail.

### Document Schema

```javascript
{
  // Identifiers
  patientId: "patient_12345",
  eventId: "evt_20250115_123456_abc123",
  timestamp: ISODate("2025-01-15T10:30:45Z"),

  // All model predictions for this event
  predictions: {
    sepsis_risk_24h: {
      model_name: "sepsis_xgboost_v1",
      prediction: 0.35,                        // Risk score [0-1]
      confidence: 0.82,                        // Model confidence
      threshold: 0.5,                          // Alert threshold
      alert_triggered: false,                  // Alert status

      // SHAP (SHapley Additive exPlanations) values
      shap_values: {
        heart_rate: 0.05,                      // Contribution to prediction
        temperature: 0.12,
        wbc_count: 0.08,
        blood_pressure: -0.03,                 // Negative = decreases risk
        lactate: 0.15
      },

      // LIME (Local Interpretable Model-agnostic Explanations)
      lime_explanation: {
        features: [
          "heart_rate",
          "temperature",
          "wbc_count"
        ],
        weights: [0.05, 0.12, 0.08],
        intercept: 0.02
      }
    },

    mortality_risk_48h: {
      model_name: "mortality_lgbm_v1",
      prediction: 0.15,
      confidence: 0.88,
      threshold: 0.3,
      alert_triggered: false,
      shap_values: {
        age: 0.08,
        sofa_score: 0.12,
        mechanical_ventilation: 0.05
      }
    },

    ards_risk_72h: {
      model_name: "ards_rf_v1",
      prediction: 0.22,
      confidence: 0.75,
      threshold: 0.4,
      alert_triggered: false
    }
  },

  // Global feature importance (across all models)
  feature_importance: {
    heart_rate: 0.25,
    temperature: 0.18,
    wbc_count: 0.15,
    lactate: 0.12,
    blood_pressure: 0.10
  },

  // Metadata
  created_at: ISODate("2025-01-15T10:30:45Z")
}
```

### Indexes

```javascript
// 1. Patient prediction history
{ patientId: 1, timestamp: -1 }
// Usage: db.ml_explanations.find({patientId: "patient_123"}).sort({timestamp: -1})

// 2. High sepsis risk queries
{ "predictions.sepsis_risk_24h.prediction": -1 }
// Usage: db.ml_explanations.find({"predictions.sepsis_risk_24h.prediction": {$gte: 0.7}})
```

### Estimated Size
- **Document Size**: ~1-3 KB per document (varies with number of models)
- **1M predictions**: ~1.5-3 GB storage
- **Index Size**: ~100-200 MB per 1M documents

### Use Cases

1. **Clinical AI Audit Trail**
   ```javascript
   // Find all sepsis alerts for a patient
   db.ml_explanations.find({
     patientId: "patient_123",
     "predictions.sepsis_risk_24h.alert_triggered": true
   })
   ```

2. **Model Performance Analysis**
   ```javascript
   // Get distribution of sepsis predictions
   db.ml_explanations.aggregate([
     {$bucket: {
       groupBy: "$predictions.sepsis_risk_24h.prediction",
       boundaries: [0, 0.3, 0.5, 0.7, 1.0],
       default: "other"
     }}
   ])
   ```

3. **Feature Importance Tracking**
   ```javascript
   // Find events where heart_rate was most important
   db.ml_explanations.find({
     "predictions.sepsis_risk_24h.shap_values.heart_rate": {$gte: 0.1}
   }).sort({"predictions.sepsis_risk_24h.shap_values.heart_rate": -1})
   ```

---

## Collection Statistics

### Expected Document Counts (Example Production System)

| Collection | Documents | Avg Size | Total Size | Indexes | Query Pattern |
|------------|-----------|----------|------------|---------|---------------|
| clinical_documents | 10M | 3 KB | 30 GB | 1.5 GB | Point queries, time-range scans |
| patient_timelines | 50K | 150 KB | 7.5 GB | 50 MB | Single document retrieval |
| ml_explanations | 8M | 2 KB | 16 GB | 800 MB | Patient history, risk analysis |

### Storage Estimates by Volume

**Low Volume (Hospital Ward)**:
- 100 patients, 10 events/day/patient = 1K events/day
- 30 days: ~30K events, ~90 MB storage

**Medium Volume (Hospital)**:
- 1000 patients, 20 events/day/patient = 20K events/day
- 90 days: ~1.8M events, ~5.4 GB storage

**High Volume (Hospital Network)**:
- 10000 patients, 50 events/day/patient = 500K events/day
- 365 days: ~182M events, ~550 GB storage

---

## Query Performance Characteristics

### clinical_documents

| Query Type | Index Used | Latency | Throughput |
|------------|-----------|---------|------------|
| Patient timeline | patientId + timestamp | <10ms | 10K/sec |
| Risk level filter | enrichments.riskLevel | <5ms | 20K/sec |
| Recent events | timestamp | <5ms | 20K/sec |
| Event type | eventType | <5ms | 20K/sec |

### patient_timelines

| Query Type | Index Used | Latency | Throughput |
|------------|-----------|---------|------------|
| Get patient timeline | _id | <5ms | 50K/sec |
| Active patients | lastUpdated | <10ms | 10K/sec |

### ml_explanations

| Query Type | Index Used | Latency | Throughput |
|------------|-----------|---------|------------|
| Patient predictions | patientId + timestamp | <10ms | 10K/sec |
| High-risk patients | predictions.sepsis_risk_24h.prediction | <20ms | 5K/sec |

---

## Data Retention Strategies

### TTL Indexes (Optional)

```javascript
// Auto-delete clinical documents older than 2 years
db.clinical_documents.createIndex(
  { createdAt: 1 },
  { expireAfterSeconds: 63072000 }  // 2 years
)

// Auto-delete ML explanations older than 1 year
db.ml_explanations.createIndex(
  { created_at: 1 },
  { expireAfterSeconds: 31536000 }  // 1 year
)
```

### Archival Strategy

```javascript
// Archive old events to separate collection
db.clinical_documents.aggregate([
  { $match: { timestamp: { $lt: ISODate("2024-01-01") } } },
  { $out: "clinical_documents_archive_2024" }
])
```

---

## Backup Recommendations

### Full Backup
```bash
mongodump --uri="mongodb://localhost:27017" --db=module8_clinical --out=/backup/
```

### Incremental Backup (Oplog-based)
```bash
mongodump --uri="mongodb://localhost:27017" --oplog --out=/backup/incremental/
```

### Restore
```bash
mongorestore --uri="mongodb://localhost:27017" /backup/module8_clinical/
```

---

## Monitoring Queries

### Collection Health Check
```javascript
// Check collection sizes
db.stats()

// Index usage statistics
db.clinical_documents.aggregate([{ $indexStats: {} }])

// Slow query analysis
db.setProfilingLevel(1, { slowms: 100 })
db.system.profile.find().sort({ ts: -1 }).limit(5)
```

### Data Quality Checks
```javascript
// Find documents missing enrichments
db.clinical_documents.countDocuments({ enrichments: null })

// Find timelines exceeding event limit
db.patient_timelines.find({ "events.1000": { $exists: true } })

// Check for orphaned explanations
db.ml_explanations.aggregate([
  {
    $lookup: {
      from: "clinical_documents",
      localField: "eventId",
      foreignField: "_id",
      as: "event"
    }
  },
  { $match: { event: { $eq: [] } } }
])
```
