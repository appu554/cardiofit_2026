# Module 1 Input Event Format

## Required Topics
- **Input**: `patient-events-v1` (and other event topics)
- **Output**: `enriched-patient-events-v1`
- **DLQ**: `dlq.processing-errors.v1` (MUST exist or Module 1 crashes!)

## Event Validation Requirements

Module 1 validates these fields (validation logic at lines 202-238):

### ✅ REQUIRED Fields:
1. **`patient_id`** (string): Must not be null or blank
2. **`event_time`** (long/number): Must be > 0
   - Cannot be > 1 hour in the future
   - Cannot be > 30 days old
3. **`payload`** (object): Must not be null or empty

### ⚠️ OPTIONAL Fields:
- **`type`** (string): Can be missing (defaults to "UNKNOWN")
- **`source`** (string): Optional
- **`encounter_id`** (string): Optional (Module 2 will handle)
- **`metadata`** (object): Optional
- **`id`** (string): Optional (auto-generated UUID if missing)

## Correct Event Format

```json
{
  "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
  "event_time": 1728518400000,
  "type": "vital_signs",
  "source": "bedside-monitor",
  "payload": {
    "heart_rate": 115,
    "blood_pressure_systolic": 155,
    "blood_pressure_diastolic": 95,
    "oxygen_saturation": 92,
    "temperature": 38.5,
    "respiratory_rate": 26
  },
  "metadata": {
    "unit": "ICU-3B"
  }
}
```

## Key Points to Avoid Crashes:

1. **DLQ Topic**: `dlq.processing-errors.v1` MUST exist before starting Module 1
2. **Timestamp**: Use current timestamp in milliseconds (e.g., `Date.now()` in JavaScript or `System.currentTimeMillis()` in Java)
3. **Patient ID**: Must be a non-empty string
4. **Payload**: Must contain at least one field

## Send Event Command:

```bash
echo '{"patient_id":"905a60cb-8241-418f-b29b-5b020e851392","event_time":1728518400000,"type":"vital_signs","source":"bedside-monitor","payload":{"heart_rate":115,"blood_pressure_systolic":155,"blood_pressure_diastolic":95,"oxygen_saturation":92,"temperature":38.5,"respiratory_rate":26},"metadata":{"unit":"ICU-3B"}}' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1
```

## Common Validation Failures (sent to DLQ):

❌ **Missing patient_id**
```json
{"event_time": 1728518400000, "payload": {...}}
```

❌ **Zero or negative timestamp**
```json
{"patient_id": "123", "event_time": 0, "payload": {...}}
```

❌ **Empty payload**
```json
{"patient_id": "123", "event_time": 1728518400000, "payload": {}}
```

❌ **Timestamp too old (>30 days)**
```json
{"patient_id": "123", "event_time": 1000000000, "payload": {...}}
```

## Module 2 FHIR Enrichment

Once Module 1 successfully processes the event:
- Output goes to: `enriched-patient-events-v1`
- Module 2 reads from this topic
- Module 2 enriches with FHIR data from Google Healthcare API
- Final output: `clinical-patterns.v1` (with FHIR patient demographics, medications, conditions, etc.)
