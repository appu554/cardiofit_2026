# CAE Clinical Intelligence - Comprehensive Testing Guide

This guide provides step-by-step instructions for importing and testing the comprehensive clinical data for your CAE system with the specific patient ID: **905a60cb-8241-418f-b29b-5b020e851392**.

## 📊 **Dataset Overview**

### **Primary Test Patient**
- **Patient ID**: `905a60cb-8241-418f-b29b-5b020e851392`
- **Profile**: 67-year-old male with complex cardiovascular conditions
- **Conditions**: Atrial fibrillation, hypertension, coronary artery disease, diabetes type 2, hyperlipidemia
- **Medications**: Warfarin, lisinopril, metoprolol, atorvastatin, metformin, aspirin
- **Critical Test Case**: Warfarin + Aspirin interaction (critical severity)

### **Complete Dataset Includes**
- **9 Patients** with diverse clinical profiles (pediatric, geriatric, pregnancy, complex comorbidities)
- **20+ Medications** across all major therapeutic classes
- **25+ Clinical Conditions** with SNOMED CT codes
- **8 Drug Interactions** with learning data (critical, high, moderate severity)
- **10 Clinical Assertions** covering all CAE reasoner types
- **8 Clinical Outcomes** for learning and validation
- **6 Clinicians** with different specialties
- **Multiple Facilities and Encounters** for context testing

## 🚀 **Step 1: Import Data into GraphDB**

### **Quick Import**
```bash
# Navigate to the graph directory
cd backend/services/clinical-reasoning-service/app/graph

# Run the import script
python import_to_graphdb.py
```

### **Manual Import (if needed)**
1. Open GraphDB Workbench at `http://localhost:7200`
2. Create repository: `cae-clinical-intelligence`
3. Import files in order:
   - First: `cae-clinical-schema.ttl` (schema)
   - Second: `cae-sample-data.ttl` (data)

## 🧪 **Step 2: Verify Import with Test Queries**

### **Query 1: Find Your Primary Test Patient**
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?patient ?patientId ?age ?gender ?weight WHERE {
    ?patient cae:hasPatientId "905a60cb-8241-418f-b29b-5b020e851392" ;
             cae:hasAge ?age ;
             cae:hasGender ?gender ;
             cae:hasWeight ?weight .
    BIND("905a60cb-8241-418f-b29b-5b020e851392" AS ?patientId)
}
```

**Expected Result**: 1 patient (67-year-old male, 78.5 kg)

### **Query 2: Find Patient's Medications**
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?medication ?genericName ?therapeuticClass WHERE {
    ?patient cae:hasPatientId "905a60cb-8241-418f-b29b-5b020e851392" ;
             cae:prescribedMedication ?medication .
    ?medication cae:hasGenericName ?genericName ;
                cae:hasTherapeuticClass ?therapeuticClass .
}
```

**Expected Result**: 6 medications (warfarin, lisinopril, metoprolol, atorvastatin, metformin, aspirin)

### **Query 3: Find Patient's Conditions**
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?condition ?conditionName ?severity WHERE {
    ?patient cae:hasPatientId "905a60cb-8241-418f-b29b-5b020e851392" ;
             cae:hasCondition ?condition .
    ?condition cae:hasConditionName ?conditionName ;
               cae:hasSeverity ?severity .
}
```

**Expected Result**: 5 conditions (atrial fibrillation, hypertension, coronary artery disease, diabetes type 2, hyperlipidemia)

### **Query 4: Find Critical Drug Interactions**
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?interaction ?severity ?confidence ?patientCount WHERE {
    ?interaction a cae:DrugInteraction ;
                 cae:hasInteractionSeverity "critical" ;
                 cae:hasConfidenceScore ?confidence ;
                 cae:hasPatientCount ?patientCount .
    BIND("critical" AS ?severity)
}
ORDER BY DESC(?confidence)
```

**Expected Result**: 3 critical interactions (warfarin+aspirin, warfarin+ibuprofen, rivaroxaban+aspirin)

## 🔍 **Step 3: CAE Reasoner Testing Scenarios**

### **Test Case 1: Drug Interaction Detection**
**Patient**: 905a60cb-8241-418f-b29b-5b020e851392  
**Medications**: Warfarin + Aspirin  
**Expected Alert**: Critical drug interaction (bleeding risk)  
**Confidence**: 95%

### **Test Case 2: Contraindication Detection**
**Patient**: patient_008  
**Medication**: Amoxicillin  
**Allergy**: Penicillin allergy  
**Expected Alert**: Critical contraindication

### **Test Case 3: Dose Adjustment (Renal)**
**Patient**: patient_009  
**Medication**: Metformin  
**Condition**: Chronic kidney disease  
**Expected Alert**: High severity dose adjustment needed

### **Test Case 4: Duplicate Therapy**
**Patient**: Any patient with multiple ACE inhibitors  
**Expected Alert**: Duplicate therapy detection

### **Test Case 5: Age-Based Dosing**
**Patient**: patient_004 (8 years old)  
**Expected Alert**: Pediatric dosing considerations

### **Test Case 6: Pregnancy Safety**
**Patient**: patient_006 (pregnant)  
**Expected Alert**: Pregnancy category warnings

## 📈 **Step 4: Learning System Testing**

### **Test Override Learning**
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?override ?reason ?timestamp ?clinician WHERE {
    ?override a cae:ClinicalOverride ;
              cae:hasOverrideReason ?reason ;
              cae:hasOverrideTimestamp ?timestamp ;
              cae:performedBy ?clinician .
}
```

### **Test Outcome Correlation**
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?outcome ?outcomeType ?severity ?assertion WHERE {
    ?outcome a cae:ClinicalOutcome ;
             cae:hasOutcomeType ?outcomeType ;
             cae:hasOutcomeSeverity ?severity ;
             cae:resultedFrom ?assertion .
}
ORDER BY DESC(?severity)
```

### **Test Patient Similarity**
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?patient1 ?patient2 WHERE {
    ?patient1 cae:hasPatientId "905a60cb-8241-418f-b29b-5b020e851392" ;
              cae:similarTo ?patient2 .
}
```

## 🔧 **Step 5: CAE Service Integration Testing**

### **Update CAE Configuration**
```python
# In your CAE service configuration
GRAPHDB_ENDPOINT = "http://localhost:7200"
GRAPHDB_REPOSITORY = "cae-clinical-intelligence"
PRIMARY_TEST_PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"
```

### **Test gRPC Calls**
```python
# Example test call for your primary patient
patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
medications = ["warfarin", "aspirin"]  # Should trigger critical interaction

# Call your CAE service
result = await cae_client.check_medication_interactions(
    patient_id=patient_id,
    medications=medications
)

# Expected: Critical interaction alert
assert result.severity == "critical"
assert result.confidence > 0.9
```

## 📊 **Step 6: Performance Testing**

### **Load Testing Queries**
```sparql
# Test query performance with complex joins
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?patient ?medication ?interaction ?severity WHERE {
    ?patient a cae:Patient ;
             cae:prescribedMedication ?med1 ;
             cae:prescribedMedication ?med2 .
    ?med1 cae:interactsWith ?med2 .
    ?interaction a cae:DrugInteraction ;
                 cae:hasInteractionSeverity ?severity .
    FILTER(?med1 != ?med2)
}
```

### **Expected Performance Metrics**
- **Simple patient lookup**: < 10ms
- **Drug interaction queries**: < 50ms
- **Complex similarity queries**: < 100ms
- **Learning pattern queries**: < 200ms

## 🎯 **Step 7: Validation Checklist**

### **Data Integrity**
- [ ] All patients have required demographics
- [ ] All medications have RxNorm codes
- [ ] All conditions have SNOMED codes
- [ ] All interactions have confidence scores
- [ ] All assertions have timestamps

### **Clinical Logic**
- [ ] Critical interactions detected correctly
- [ ] Contraindications trigger appropriate alerts
- [ ] Dose adjustments calculated properly
- [ ] Duplicate therapy identified
- [ ] Age-based rules applied correctly

### **Learning System**
- [ ] Override data captured
- [ ] Outcomes linked to assertions
- [ ] Patient similarity calculated
- [ ] Confidence scores updated
- [ ] Temporal patterns recognized

## 🚨 **Common Issues and Solutions**

### **Import Issues**
- **Problem**: RDF syntax errors
- **Solution**: Validate TTL files with online RDF validator

### **Query Issues**
- **Problem**: No results returned
- **Solution**: Check namespace prefixes and URI format

### **Performance Issues**
- **Problem**: Slow queries
- **Solution**: Add indexes on frequently queried properties

### **Integration Issues**
- **Problem**: CAE service can't connect to GraphDB
- **Solution**: Verify GraphDB is running and repository exists

## 📞 **Support Resources**

- **GraphDB Documentation**: https://graphdb.ontotext.com/documentation/
- **SPARQL Tutorial**: https://www.w3.org/TR/sparql11-query/
- **RDF/Turtle Guide**: https://www.w3.org/TR/turtle/

This comprehensive dataset provides everything needed to thoroughly test your CAE system with realistic clinical scenarios! 🎉
