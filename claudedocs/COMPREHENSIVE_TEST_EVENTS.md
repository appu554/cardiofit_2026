# Comprehensive Clinical Intelligence Test Events

## Test Scenario: Patient Rohan with Worsening Sepsis

**Clinical Context:**
- 42-year-old male with hypertension and prediabetes
- Presenting with early sepsis signs
- Requires comprehensive monitoring across vitals, labs, and medications

**Expected System Behavior:**
1. **Vital signs** → Trigger NEWS2=8, qSOFA=1, SIRS alerts
2. **Lab results** → Detect elevated lactate, cardiac markers, electrolyte abnormalities
3. **Medications** → Track nephrotoxic drugs, recent changes, anticoagulation status
4. **Combined analysis** → Generate comprehensive clinical intelligence with high confidence

---

## Event 1: VITAL_SIGN (Baseline - Already Tested)

**Topic:** `vital-signs-events-v1`

**Expected Results:**
- ✅ NEWS2 score: 8 (HIGH RISK)
- ✅ qSOFA score: 1
- ✅ Risk indicators: tachycardia, fever, hypoxia, tachypnea (all true)
- ✅ Alerts: 6 alerts including HIGH priority NEWS2 and SIRS warnings
- ✅ Confidence: 0.95

```json
{
  "type": "VITAL_SIGN",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171000000,
  "source": "bedside_monitor",
  "payload": {
    "heartRate": 110,
    "systolicBP": 110,
    "diastolicBP": 70,
    "respiratoryRate": 28,
    "oxygenSaturation": 92,
    "temperature": 39.0,
    "consciousness": "Alert",
    "supplementalOxygen": false
  },
  "metadata": {
    "deviceId": "MONITOR-ICU-12",
    "location": "ICU-BED-5"
  }
}
```

---

## Event 2: LAB_RESULT (Critical Labs - Sepsis Workup)

**Topic:** `lab-result-events-v1`

**Clinical Rationale:**
- **Elevated Lactate (2.8 mmol/L)** → Tissue hypoperfusion, sepsis indicator
- **Elevated Troponin (0.06 ng/mL)** → Cardiac stress from sepsis
- **Elevated Creatinine (1.6 mg/dL)** → Early acute kidney injury
- **Leukocytosis (15.2 K/uL)** → Infection/inflammation response
- **Hypokalemia (3.2 mEq/L)** → Electrolyte disturbance
- **Elevated BNP (520 pg/mL)** → Cardiac stress/fluid overload

**Expected Results:**
- ✅ Risk indicators updated:
  - `elevatedLactate: true` (lactate=2.8 > 2.0)
  - `elevatedTroponin: true` (troponin=0.06 > 0.04)
  - `elevatedCreatinine: true` (creatinine=1.6 > 1.3)
  - `leukocytosis: true` (WBC=15.2 > 11.0)
  - `hypokalemia: true` (K=3.2 < 3.5)
  - `elevatedBNP: true` (BNP=520 > 400)
- ✅ New alerts generated:
  - LAB_CRITICAL_VALUE alerts for abnormal labs
  - Cardiac marker alerts
  - Electrolyte abnormality warnings
- ✅ Confidence score increases (labs data present)

### Lactate (LOINC: 2524-7)
```json
{
  "type": "LAB_RESULT",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171300000,
  "source": "lab_system",
  "payload": {
    "loincCode": "2524-7",
    "labName": "Lactate",
    "category": "METABOLIC",
    "value": 2.8,
    "unit": "mmol/L",
    "referenceRangeLow": 0.5,
    "referenceRangeHigh": 2.0,
    "abnormalFlag": "H",
    "specimenType": "blood",
    "collectionTime": 1760171200000,
    "resultTime": 1760171300000,
    "performingLab": "CENTRAL_LAB"
  },
  "metadata": {
    "orderId": "LAB-2025-001234",
    "priority": "STAT"
  }
}
```

### Troponin I (LOINC: 10839-9)
```json
{
  "type": "LAB_RESULT",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171300000,
  "source": "lab_system",
  "payload": {
    "loincCode": "10839-9",
    "labName": "Troponin I, High Sensitivity",
    "category": "CARDIAC",
    "value": 0.06,
    "unit": "ng/mL",
    "referenceRangeLow": 0.0,
    "referenceRangeHigh": 0.04,
    "abnormalFlag": "H",
    "specimenType": "serum",
    "collectionTime": 1760171200000,
    "resultTime": 1760171300000,
    "performingLab": "CENTRAL_LAB"
  },
  "metadata": {
    "orderId": "LAB-2025-001235",
    "priority": "STAT"
  }
}
```

### BNP (LOINC: 42757-5)
```json
{
  "type": "LAB_RESULT",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171300000,
  "source": "lab_system",
  "payload": {
    "loincCode": "42757-5",
    "labName": "B-type Natriuretic Peptide (BNP)",
    "category": "CARDIAC",
    "value": 520.0,
    "unit": "pg/mL",
    "referenceRangeLow": 0.0,
    "referenceRangeHigh": 100.0,
    "abnormalFlag": "HH",
    "specimenType": "plasma",
    "collectionTime": 1760171200000,
    "resultTime": 1760171300000,
    "performingLab": "CENTRAL_LAB"
  },
  "metadata": {
    "orderId": "LAB-2025-001236",
    "priority": "STAT"
  }
}
```

### Creatinine (LOINC: 2160-0)
```json
{
  "type": "LAB_RESULT",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171300000,
  "source": "lab_system",
  "payload": {
    "loincCode": "2160-0",
    "labName": "Creatinine",
    "category": "METABOLIC",
    "value": 1.6,
    "unit": "mg/dL",
    "referenceRangeLow": 0.7,
    "referenceRangeHigh": 1.3,
    "abnormalFlag": "H",
    "specimenType": "serum",
    "collectionTime": 1760171200000,
    "resultTime": 1760171300000,
    "performingLab": "CENTRAL_LAB"
  },
  "metadata": {
    "orderId": "LAB-2025-001237",
    "priority": "ROUTINE"
  }
}
```

### White Blood Cell Count (LOINC: 6690-2)
```json
{
  "type": "LAB_RESULT",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171300000,
  "source": "lab_system",
  "payload": {
    "loincCode": "6690-2",
    "labName": "White Blood Cell Count",
    "category": "HEMATOLOGY",
    "value": 15.2,
    "unit": "K/uL",
    "referenceRangeLow": 4.0,
    "referenceRangeHigh": 11.0,
    "abnormalFlag": "H",
    "specimenType": "blood",
    "collectionTime": 1760171200000,
    "resultTime": 1760171300000,
    "performingLab": "HEMATOLOGY_LAB"
  },
  "metadata": {
    "orderId": "LAB-2025-001238",
    "priority": "ROUTINE"
  }
}
```

### Potassium (LOINC: 2823-3)
```json
{
  "type": "LAB_RESULT",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171300000,
  "source": "lab_system",
  "payload": {
    "loincCode": "2823-3",
    "labName": "Potassium",
    "category": "METABOLIC",
    "value": 3.2,
    "unit": "mEq/L",
    "referenceRangeLow": 3.5,
    "referenceRangeHigh": 5.5,
    "abnormalFlag": "L",
    "specimenType": "serum",
    "collectionTime": 1760171200000,
    "resultTime": 1760171300000,
    "performingLab": "CHEMISTRY_LAB"
  },
  "metadata": {
    "orderId": "LAB-2025-001239",
    "priority": "ROUTINE"
  }
}
```

---

## Event 3: MEDICATION_UPDATE (Active Medications)

**Topic:** `medication-events-v1`

**Clinical Rationale:**
- **Telmisartan (ARB)** → Existing BP medication, continued
- **Vancomycin (Antibiotic)** → NEW - Started for sepsis treatment (NEPHROTOXIC!)
- **Furosemide (Diuretic)** → NEW - For fluid management
- **Heparin (Anticoagulant)** → NEW - DVT prophylaxis (bleeding risk!)

**Expected Results:**
- ✅ Risk indicators updated:
  - `onNephrotoxicMeds: true` (Vancomycin started)
  - `onAnticoagulation: true` (Heparin started)
  - `recentMedicationChange: true` (3 new meds in last 24h)
- ✅ New alerts generated:
  - DRUG_INTERACTION alert (Furosemide + hypokalemia)
  - Nephrotoxic medication alert (Vancomycin + elevated creatinine)
  - Anticoagulation monitoring alert
- ✅ Medication effectiveness monitoring begins

### Telmisartan (Continued - Baseline BP Med)
```json
{
  "type": "MEDICATION_UPDATE",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171000000,
  "source": "pharmacy_system",
  "payload": {
    "rxNormCode": "83367",
    "medicationName": "Telmisartan 40 mg Tablet",
    "genericName": "Telmisartan",
    "brandName": "Micardis",
    "therapeuticClass": "ARB",
    "drugClasses": ["ANTIHYPERTENSIVE", "ANGIOTENSIN_RECEPTOR_BLOCKER"],
    "dose": 40.0,
    "doseUnit": "mg",
    "route": "oral",
    "frequency": "daily",
    "administrationTime": 1760171000000,
    "administeredBy": "NURSE-456",
    "administrationStatus": "administered",
    "orderTime": 1759000000000,
    "startTime": 1759000000000,
    "stopTime": null,
    "prescriber": "DOC-101",
    "indication": "Hypertension",
    "nephrotoxic": false,
    "requiresMonitoring": true,
    "monitoringParameters": ["Blood Pressure", "Creatinine", "Potassium"]
  },
  "metadata": {
    "orderId": "MED-2025-RX-9001",
    "pharmacyVerified": true
  }
}
```

### Vancomycin (NEW - Antibiotic for Sepsis - NEPHROTOXIC!)
```json
{
  "type": "MEDICATION_UPDATE",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171400000,
  "source": "pharmacy_system",
  "payload": {
    "rxNormCode": "11124",
    "medicationName": "Vancomycin 1000 mg IV",
    "genericName": "Vancomycin",
    "brandName": null,
    "therapeuticClass": "GLYCOPEPTIDE_ANTIBIOTIC",
    "drugClasses": ["ANTIBIOTIC", "ANTI_INFECTIVE"],
    "dose": 1000.0,
    "doseUnit": "mg",
    "route": "IV",
    "frequency": "Q12H",
    "administrationTime": 1760171400000,
    "administeredBy": "NURSE-789",
    "administrationStatus": "administered",
    "orderTime": 1760171200000,
    "startTime": 1760171400000,
    "stopTime": null,
    "prescriber": "DOC-101",
    "indication": "Suspected sepsis",
    "nephrotoxic": true,
    "requiresMonitoring": true,
    "monitoringParameters": ["Creatinine", "Vancomycin Trough Level", "BUN", "Urine Output"]
  },
  "metadata": {
    "orderId": "MED-2025-RX-9010",
    "pharmacyVerified": true,
    "infusionRate": "10 mg/min",
    "diluentVolume": "250 mL NS"
  }
}
```

### Furosemide (NEW - Diuretic for Fluid Management)
```json
{
  "type": "MEDICATION_UPDATE",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171500000,
  "source": "pharmacy_system",
  "payload": {
    "rxNormCode": "4603",
    "medicationName": "Furosemide 40 mg IV",
    "genericName": "Furosemide",
    "brandName": "Lasix",
    "therapeuticClass": "DIURETIC_LOOP",
    "drugClasses": ["DIURETIC", "ANTIHYPERTENSIVE"],
    "dose": 40.0,
    "doseUnit": "mg",
    "route": "IV",
    "frequency": "daily",
    "administrationTime": 1760171500000,
    "administeredBy": "NURSE-789",
    "administrationStatus": "administered",
    "orderTime": 1760171200000,
    "startTime": 1760171500000,
    "stopTime": null,
    "prescriber": "DOC-101",
    "indication": "Fluid overload",
    "nephrotoxic": false,
    "requiresMonitoring": true,
    "monitoringParameters": ["Electrolytes", "Creatinine", "Blood Pressure", "Urine Output"]
  },
  "metadata": {
    "orderId": "MED-2025-RX-9011",
    "pharmacyVerified": true,
    "pushRate": "slow IV push over 2 minutes"
  }
}
```

### Heparin (NEW - DVT Prophylaxis - ANTICOAGULATION!)
```json
{
  "type": "MEDICATION_UPDATE",
  "patient_id": "PAT-ROHAN-001",
  "event_time": 1760171600000,
  "source": "pharmacy_system",
  "payload": {
    "rxNormCode": "5224",
    "medicationName": "Heparin 5000 units SC",
    "genericName": "Heparin",
    "brandName": null,
    "therapeuticClass": "ANTICOAGULANT",
    "drugClasses": ["ANTICOAGULANT", "ANTITHROMBOTIC"],
    "dose": 5000.0,
    "doseUnit": "units",
    "route": "SC",
    "frequency": "Q12H",
    "administrationTime": 1760171600000,
    "administeredBy": "NURSE-789",
    "administrationStatus": "administered",
    "orderTime": 1760171200000,
    "startTime": 1760171600000,
    "stopTime": null,
    "prescriber": "DOC-101",
    "indication": "DVT prophylaxis",
    "nephrotoxic": false,
    "requiresMonitoring": true,
    "monitoringParameters": ["Platelets", "aPTT", "Signs of bleeding"]
  },
  "metadata": {
    "orderId": "MED-2025-RX-9012",
    "pharmacyVerified": true,
    "injectionSite": "abdomen"
  }
}
```

---

## Expected Combined Clinical Intelligence Output

After processing all three event types, the system should produce:

### Clinical Scores
```json
{
  "news2Score": 8,
  "qsofaScore": 1,
  "combinedAcuityScore": 5.75
}
```

### Risk Indicators (Comprehensive)
```json
{
  "tachycardia": true,
  "fever": true,
  "hypoxia": true,
  "tachypnea": true,
  "elevatedLactate": true,
  "elevatedTroponin": true,
  "elevatedBNP": true,
  "elevatedCreatinine": true,
  "leukocytosis": true,
  "hypokalemia": true,
  "onNephrotoxicMeds": true,
  "onAnticoagulation": true,
  "recentMedicationChange": true,
  "confidenceScore": 1.0
}
```

### Alerts (Expected ~12-15 Total)
```json
{
  "alertCount": 12,
  "alertCategories": {
    "DETERIORATION_PATTERN": 1,
    "SEPSIS_PATTERN": 2,
    "RESPIRATORY_DISTRESS": 2,
    "VITAL_THRESHOLD_BREACH": 1,
    "LAB_CRITICAL_VALUE": 4,
    "DRUG_INTERACTION": 2
  }
}
```

### Confidence Score
```json
{
  "confidenceScore": 1.0,
  "breakdown": {
    "base": 0.5,
    "fhir": 0.2,
    "neo4j": 0.15,
    "vitals": 0.1,
    "labs": 0.05
  }
}
```

---

## How to Run Tests

### 1. Send Events Sequentially (Recommended)
```bash
# Event 1: Vital Signs (baseline)
echo '<VITAL_SIGN_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic vital-signs-events-v1

# Wait 2 seconds for processing
sleep 2

# Event 2: Lab Results (send all 6 labs)
echo '<LACTATE_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1
echo '<TROPONIN_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1
echo '<BNP_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1
echo '<CREATININE_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1
echo '<WBC_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1
echo '<POTASSIUM_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1

# Wait 2 seconds for processing
sleep 2

# Event 3: Medications (send all 4 meds)
echo '<TELMISARTAN_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
echo '<VANCOMYCIN_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
echo '<FUROSEMIDE_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
echo '<HEPARIN_JSON>' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1

# Wait 3 seconds for complete processing
sleep 3

# Consume final enriched output
timeout 10 docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic enriched-patient-events-v1 --from-beginning --max-messages 1
```

### 2. Verify Results
Check output for:
- ✅ All risk indicators set correctly
- ✅ ~12-15 alerts generated
- ✅ Confidence score = 1.0 (100% - all data sources present)
- ✅ Lab values in `recentLabs` map
- ✅ Medications in `activeMedications` map
- ✅ High acuity flag = true

---

## Clinical Decision Support Validation

This comprehensive test validates:

1. **Multi-Source Data Integration** ✅
   - Vitals + Labs + Medications all aggregated correctly
   - State persistence across event types

2. **Evidence-Based Scoring** ✅
   - NEWS2 = 8 (respiratory distress + hypoxia + tachycardia + fever)
   - qSOFA = 1 (tachypnea for sepsis screening)

3. **Lab Abnormality Detection** ✅
   - Cardiac markers (Troponin, BNP)
   - Metabolic panel (Lactate, Creatinine, Potassium)
   - Hematology (WBC)

4. **Medication Safety Monitoring** ✅
   - Nephrotoxic drug alert (Vancomycin + elevated creatinine)
   - Drug-lab interaction (Furosemide + hypokalemia)
   - Anticoagulation monitoring (Heparin)

5. **Clinical Correlation** ✅
   - Sepsis pattern recognition (SIRS + elevated lactate + leukocytosis)
   - Organ dysfunction detection (AKI from elevated creatinine)
   - Cardiac stress markers (Troponin + BNP elevation)

This represents a **realistic clinical scenario** requiring comprehensive monitoring and multi-disciplinary intervention!
