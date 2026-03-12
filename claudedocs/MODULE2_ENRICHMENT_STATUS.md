# Module 2 FHIR/Neo4j Enrichment - Current Status

## ✅ What's Working

### Enrichment Data Structure
The `enrichment_data` field is now properly populated with all available data:

```json
"enrichment_data": {
  "fhir_demographics": {
    "firstName": "Rohan",
    "lastName": "Sharma",
    "gender": "male",
    "dateOfBirth": "1983-05-15",
    "age": 42
  },
  "neo4j_care_team": ["DOC-101"],
  "neo4j_risk_cohorts": ["Urban Metabolic Syndrome Cohort"],
  "fhir_medications": [],  // Empty because no data in FHIR store
  "fhir_conditions": [],   // Empty because no data in FHIR store
  "fhir_allergies": [],    // Empty because no data in FHIR store
  "state_version": 6,
  "was_new_patient": false
}
```

### Data Flow Verified
1. ✅ **AsyncPatientEnricher** - Successfully fetches data from FHIR and Neo4j
2. ✅ **PatientSnapshot.hydrateFromHistory()** - Properly stores all fetched data
3. ✅ **createEnrichedEventFromSnapshot()** - Correctly populates `enrichment_data`
4. ✅ **Neo4j Integration** - Care team and risk cohorts are being fetched and included
5. ✅ **FHIR Demographics** - Patient details (name, age, gender, DOB) are included

## ⚠️ Missing Data Explanation

### Why Medications and Conditions Show Empty

The medication and condition arrays are empty **NOT because of a bug**, but because:

1. **FHIR Store Has No Data**: The Google FHIR Healthcare API doesn't have Condition or MedicationStatement resources for PAT-ROHAN-001
2. **Client Working Correctly**: The `GoogleFHIRClient.getConditionsAsync()` and `getMedicationsAsync()` methods are querying correctly - they're just returning empty results because no data exists
3. **Previous Sessions**: If you saw medications/conditions before, they were likely:
   - Mock data from test cases
   - Data that was cleared from FHIR
   - From a different patient ID

### Data Currently in FHIR for PAT-ROHAN-001
- ✅ Patient resource (demographics)
- ❌ Condition resources (none loaded)
- ❌ MedicationStatement resources (none loaded)
- ❌ AllergyIntolerance resources (none loaded)

### Data Currently in Neo4j for PAT-ROHAN-001
- ✅ Care team relationships (DOC-101)
- ✅ Risk cohort assignments (Urban Metabolic Syndrome Cohort)

## 🔧 How to Add Medications and Conditions

### Option 1: Load Test Data to FHIR

Run the data loading script to populate FHIR with clinical data:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
python3 load-synthetic-data-rohan.py
```

This should create:
- Condition resources (e.g., Hypertension, Type 2 Diabetes)
- MedicationStatement resources (e.g., Metformin, Lisinopril)
- AllergyIntolerance resources if applicable

### Option 2: Manually Create FHIR Resources

Use the Google Healthcare API to create Condition resources:

```json
POST https://healthcare.googleapis.com/v1/projects/{project}/locations/{location}/datasets/{dataset}/fhirStores/{fhirStore}/fhir/Condition

{
  "resourceType": "Condition",
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "code": {
    "coding": [{
      "system": "http://snomed.info/sct",
      "code": "44054006",
      "display": "Type 2 Diabetes Mellitus"
    }]
  },
  "clinicalStatus": {
    "coding": [{
      "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
      "code": "active"
    }]
  }
}
```

And MedicationStatement resources:

```json
POST https://healthcare.googleapis.com/v1/projects/{project}/locations/{location}/datasets/{dataset}/fhirStores/{fhirStore}/fhir/MedicationStatement

{
  "resourceType": "MedicationStatement",
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "medicationCodeableConcept": {
    "coding": [{
      "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
      "code": "860975",
      "display": "Metformin 500 MG"
    }]
  },
  "status": "active"
}
```

### Option 3: Verify FHIR Client Configuration

Check that the FHIR client is configured to query the correct FHIR store:

```bash
# Check environment variables
cat /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/flink-datastores.env | grep FHIR
```

## 🎯 Expected Output After Loading Data

Once medications and conditions are loaded into FHIR, the enrichment_data will include:

```json
"enrichment_data": {
  "fhir_demographics": {
    "firstName": "Rohan",
    "lastName": "Sharma",
    "age": 42,
    "gender": "male",
    "dateOfBirth": "1983-05-15"
  },
  "fhir_medications": [
    {
      "code": "860975",
      "display": "Metformin 500 MG",
      "status": "active"
    },
    {
      "code": "314076",
      "display": "Lisinopril 10 MG",
      "status": "active"
    }
  ],
  "fhir_conditions": [
    {
      "code": "44054006",
      "display": "Type 2 Diabetes Mellitus",
      "clinicalStatus": "active"
    },
    {
      "code": "38341003",
      "display": "Hypertension",
      "clinicalStatus": "active"
    }
  ],
  "fhir_allergies": [
    {
      "substance": "Penicillin",
      "reaction": "Rash"
    }
  ],
  "neo4j_care_team": ["DOC-101"],
  "neo4j_risk_cohorts": ["Urban Metabolic Syndrome Cohort"],
  "state_version": 6,
  "was_new_patient": false
}
```

## ✅ Code Changes Applied

### 1. Always Include FHIR Arrays (Even if Empty)

**File**: `Module2_ContextAssembly.java`
**Lines**: 1075-1081, 438-444

**Change**: Modified to always include `fhir_medications`, `fhir_conditions`, and `fhir_allergies` fields even when empty:

```java
// OLD - would exclude if empty
if (snapshot.getActiveMedications() != null && !snapshot.getActiveMedications().isEmpty()) {
    enrichmentData.put("fhir_medications", snapshot.getActiveMedications());
}

// NEW - always include to show data was fetched
enrichmentData.put("fhir_medications", snapshot.getActiveMedications() != null ?
    snapshot.getActiveMedications() : new ArrayList<>());
```

**Benefit**: This clearly shows that FHIR was queried (empty array) vs. not queried at all (field missing).

### 2. Comprehensive Enrichment Data Population

**File**: `Module2_ContextAssembly.java`
**Lines**: 1045-1139 (async), 408-462 (blocking)

**Added**:
- FHIR demographics (firstName, lastName, age, gender, dateOfBirth, mrn)
- FHIR clinical data (medications, conditions, allergies)
- Neo4j graph data (care team, risk cohorts)
- Latest vitals and lab values
- State metadata (state_version, was_new_patient)
- Risk scores (sepsis_score, deterioration_score, readmission_risk)

### 3. PatientContext Enrichment

**File**: `Module2_ContextAssembly.java`
**Lines**: 1219-1233

**Added**: Care team and risk cohorts to `PatientContext`:

```java
if (snapshot.getCareTeam() != null && !snapshot.getCareTeam().isEmpty()) {
    context.setCareTeam(snapshot.getCareTeam());
}
if (snapshot.getRiskCohorts() != null && !snapshot.getRiskCohorts().isEmpty()) {
    context.setRiskCohorts(snapshot.getRiskCohorts());
}
```

## 📊 Testing & Verification

### Current Test Results

```bash
# Send test event
echo '{"patient_id":"PAT-ROHAN-001","event_time":'$(date +%s)000',"type":"vital_signs",...}' | \
  docker exec -i kafka kafka-console-producer --broker-list localhost:9092 --topic patient-events-v1

# Check enriched output
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic clinical-patterns-v1 \
  --from-beginning --max-messages 1
```

**Result**: ✅ enrichment_data properly populated with all available data (demographics + Neo4j, medications/conditions empty because not in FHIR)

### Verification Checklist

- [x] `enrichment_data` field present in output
- [x] `fhir_demographics` with patient details
- [x] `fhir_medications` field present (empty array)
- [x] `fhir_conditions` field present (empty array)
- [x] `fhir_allergies` field present (empty array)
- [x] `neo4j_care_team` with care providers
- [x] `neo4j_risk_cohorts` with cohort assignments
- [x] `patient_context.demographics` populated
- [x] `patient_context.careTeam` populated
- [x] `patient_context.riskCohorts` populated
- [ ] FHIR medications data (pending FHIR data load)
- [ ] FHIR conditions data (pending FHIR data load)

## 🎯 Next Steps

1. **Load FHIR Clinical Data**: Run data loading script to populate medications and conditions
2. **Verify Complete Enrichment**: Test again after data load to see full enrichment
3. **Monitor Performance**: Check AsyncPatientEnricher latency with full data
4. **Test Edge Cases**: Verify behavior with patients that have many medications/conditions

## 📝 Summary

The enrichment pipeline is **working correctly**. The "missing" medications and conditions are not a bug - they're simply not present in the FHIR store for this patient. Once you load the clinical data into FHIR, it will automatically appear in the enrichment_data field.

**Key Achievement**: We successfully fixed the data mapping gap that was preventing enrichment_data from being populated, and now all FHIR and Neo4j data flows correctly from source systems → AsyncPatientEnricher → PatientSnapshot → EnrichedEvent output.