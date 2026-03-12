# Usage Examples - Neo4j Multi-KB Stream Manager

This document provides practical examples for common usage patterns of the Multi-KB Stream Manager in clinical applications.

## Table of Contents

- [Basic Setup and Initialization](#basic-setup-and-initialization)
- [Data Loading Patterns](#data-loading-patterns)
- [Single KB Queries](#single-kb-queries)
- [Cross-KB Clinical Scenarios](#cross-kb-clinical-scenarios)
- [Health Monitoring](#health-monitoring)
- [Integration Patterns](#integration-patterns)

## Basic Setup and Initialization

### Standard Initialization

```python
import asyncio
from multi_kb_stream_manager import MultiKBStreamManager, KnowledgeBase, StreamType

async def setup_manager():
    config = {
        'neo4j_uri': 'bolt://localhost:7687',
        'neo4j_user': 'neo4j',
        'neo4j_password': 'your_password'
    }

    manager = MultiKBStreamManager(config)

    # Initialize all knowledge base streams
    success = await manager.initialize_all_streams()
    if not success:
        raise Exception("Failed to initialize KB streams")

    return manager

# Usage
manager = await setup_manager()
```

### Production Configuration

```python
import os
from multi_kb_stream_manager import MultiKBStreamManager

async def setup_production_manager():
    config = {
        'neo4j_uri': os.getenv('NEO4J_URI', 'bolt://neo4j-cluster:7687'),
        'neo4j_user': os.getenv('NEO4J_USER', 'cardiofit'),
        'neo4j_password': os.getenv('NEO4J_PASSWORD')
    }

    manager = MultiKBStreamManager(config)
    await manager.initialize_all_streams()

    # Verify all KBs are healthy
    health_status = await manager.health_check_all_streams()
    unhealthy_kbs = [kb for kb, status in health_status.items()
                     if not status.get('healthy', False)]

    if unhealthy_kbs:
        raise Exception(f"Unhealthy KBs detected: {unhealthy_kbs}")

    return manager
```

## Data Loading Patterns

### Loading Patient Data (KB1)

```python
async def load_patient(manager, patient_data):
    """Load patient into KB1 with comprehensive data"""

    patient_record = {
        'entity_type': 'Patient',
        'name': patient_data['name'],
        'mrn': patient_data['mrn'],
        'date_of_birth': patient_data['dob'],
        'gender': patient_data['gender'],
        'conditions': patient_data.get('conditions', []),
        'allergies': patient_data.get('allergies', []),
        'current_medications': patient_data.get('medications', [])
    }

    success = await manager.load_kb_data(
        KnowledgeBase.KB1_PATIENT,
        StreamType.PATIENT,
        patient_data['mrn'],
        patient_record
    )

    if success:
        print(f"Patient {patient_data['name']} loaded successfully")
    return success

# Example usage
patient_data = {
    'name': 'John Doe',
    'mrn': 'MRN123456',
    'dob': '1975-06-15',
    'gender': 'M',
    'conditions': ['hypertension', 'diabetes_type_2'],
    'allergies': ['penicillin'],
    'medications': ['lisinopril', 'metformin']
}

await load_patient(manager, patient_data)
```

### Loading Clinical Guidelines (KB2)

```python
async def load_clinical_guideline(manager, guideline_data):
    """Load clinical practice guideline into KB2"""

    guideline_record = {
        'entity_type': 'Guideline',
        'title': guideline_data['title'],
        'organization': guideline_data['organization'],
        'version': guideline_data['version'],
        'effective_date': guideline_data['effective_date'],
        'condition_category': guideline_data['condition_category'],
        'recommendations': guideline_data['recommendations'],
        'evidence_level': guideline_data['evidence_level']
    }

    success = await manager.load_kb_data(
        KnowledgeBase.KB2_GUIDELINES,
        StreamType.PATIENT,  # Primary stream for operational data
        guideline_data['guideline_id'],
        guideline_record
    )

    return success

# Example usage
guideline_data = {
    'guideline_id': 'AHA_HTN_2023',
    'title': 'Hypertension Management Guidelines',
    'organization': 'American Heart Association',
    'version': '2023.1',
    'effective_date': '2023-01-01',
    'condition_category': 'cardiovascular',
    'recommendations': [
        'First-line: ACE inhibitor or ARB',
        'Target BP: <130/80 for most adults',
        'Lifestyle modifications recommended'
    ],
    'evidence_level': 'A'
}

await load_clinical_guideline(manager, guideline_data)
```

### Loading Drug Interaction Data (KB5)

```python
async def load_drug_interaction(manager, interaction_data):
    """Load drug interaction data into KB5"""

    interaction_record = {
        'entity_type': 'Interaction',
        'drug1_rxnorm': interaction_data['drug1_rxnorm'],
        'drug1_name': interaction_data['drug1_name'],
        'drug2_rxnorm': interaction_data['drug2_rxnorm'],
        'drug2_name': interaction_data['drug2_name'],
        'severity': interaction_data['severity'],
        'mechanism': interaction_data['mechanism'],
        'clinical_effect': interaction_data['clinical_effect'],
        'management': interaction_data['management'],
        'source': interaction_data['source']
    }

    interaction_id = f"{interaction_data['drug1_rxnorm']}_{interaction_data['drug2_rxnorm']}"

    success = await manager.load_kb_data(
        KnowledgeBase.KB5_DRUG_INTERACTIONS,
        StreamType.PATIENT,
        interaction_id,
        interaction_record
    )

    return success

# Example usage
interaction_data = {
    'drug1_rxnorm': 'RX123456',
    'drug1_name': 'lisinopril',
    'drug2_rxnorm': 'RX789012',
    'drug2_name': 'potassium supplements',
    'severity': 'moderate',
    'mechanism': 'Additive hyperkalemic effects',
    'clinical_effect': 'Increased risk of hyperkalemia',
    'management': 'Monitor potassium levels closely',
    'source': 'Clinical Pharmacology Database'
}

await load_drug_interaction(manager, interaction_data)
```

## Single KB Queries

### Finding Patients with Specific Conditions

```python
async def find_patients_with_condition(manager, condition):
    """Find all patients with a specific condition"""

    results = await manager.query_kb_stream(
        KnowledgeBase.KB1_PATIENT,
        StreamType.PATIENT,
        "WHERE $condition IN n.conditions RETURN n.name, n.mrn, n.conditions",
        {'condition': condition}
    )

    patients = [{'name': r['n.name'], 'mrn': r['n.mrn'], 'conditions': r['n.conditions']}
                for r in results]

    return patients

# Example usage
hypertensive_patients = await find_patients_with_condition(manager, 'hypertension')
print(f"Found {len(hypertensive_patients)} patients with hypertension")
```

### Querying Drug Interactions by Severity

```python
async def get_severe_drug_interactions(manager):
    """Get all severe drug interactions"""

    results = await manager.query_kb_stream(
        KnowledgeBase.KB5_DRUG_INTERACTIONS,
        StreamType.PATIENT,
        """
        WHERE n.severity = 'severe'
        RETURN n.drug1_name, n.drug2_name, n.clinical_effect, n.management
        ORDER BY n.drug1_name
        """,
        {}
    )

    return [{
        'drug1': r['n.drug1_name'],
        'drug2': r['n.drug2_name'],
        'effect': r['n.clinical_effect'],
        'management': r['n.management']
    } for r in results]

# Example usage
severe_interactions = await get_severe_drug_interactions(manager)
for interaction in severe_interactions:
    print(f"⚠️  {interaction['drug1']} + {interaction['drug2']}: {interaction['effect']}")
```

## Cross-KB Clinical Scenarios

### Patient Safety Check: Drug Interactions

```python
async def check_patient_drug_safety(manager, patient_mrn):
    """Check for drug interactions in patient's current medications"""

    results = await manager.cross_kb_query(
        [KnowledgeBase.KB1_PATIENT, KnowledgeBase.KB5_DRUG_INTERACTIONS],
        """
        MATCH (p:Patient:KB1_PatientStream {mrn: $patient_mrn})
        UNWIND p.current_medications AS med1
        UNWIND p.current_medications AS med2
        WITH p, med1, med2
        WHERE med1 < med2  // Avoid duplicate pairs

        MATCH (i:Interaction:KB5_InteractionStream)
        WHERE (i.drug1_name = med1 AND i.drug2_name = med2) OR
              (i.drug1_name = med2 AND i.drug2_name = med1)

        RETURN p.name, p.mrn, med1, med2, i.severity, i.clinical_effect, i.management
        ORDER BY
          CASE i.severity
            WHEN 'severe' THEN 1
            WHEN 'moderate' THEN 2
            WHEN 'mild' THEN 3
          END
        """,
        {'patient_mrn': patient_mrn}
    )

    interactions = []
    for r in results:
        interactions.append({
            'patient': r['p.name'],
            'drug1': r['med1'],
            'drug2': r['med2'],
            'severity': r['i.severity'],
            'effect': r['i.clinical_effect'],
            'management': r['i.management']
        })

    return interactions

# Example usage
safety_alerts = await check_patient_drug_safety(manager, 'MRN123456')
for alert in safety_alerts:
    print(f"🚨 {alert['severity'].upper()}: {alert['drug1']} + {alert['drug2']}")
    print(f"   Effect: {alert['effect']}")
    print(f"   Management: {alert['management']}")
```

### Clinical Decision Support: Guideline Recommendations

```python
async def get_treatment_recommendations(manager, patient_mrn):
    """Get evidence-based treatment recommendations for patient conditions"""

    results = await manager.cross_kb_query(
        [KnowledgeBase.KB1_PATIENT, KnowledgeBase.KB2_GUIDELINES],
        """
        MATCH (p:Patient:KB1_PatientStream {mrn: $patient_mrn})
        UNWIND p.conditions AS condition

        MATCH (g:Guideline:KB2_GuidelineStream)
        WHERE condition IN g.condition_category OR
              any(keyword IN split(g.title, ' ') WHERE toLower(keyword) CONTAINS toLower(condition))

        RETURN p.name, condition, g.title, g.organization, g.recommendations, g.evidence_level
        ORDER BY g.evidence_level, g.effective_date DESC
        """,
        {'patient_mrn': patient_mrn}
    )

    recommendations = []
    for r in results:
        recommendations.append({
            'patient': r['p.name'],
            'condition': r['condition'],
            'guideline': r['g.title'],
            'organization': r['g.organization'],
            'recommendations': r['g.recommendations'],
            'evidence_level': r['g.evidence_level']
        })

    return recommendations

# Example usage
recommendations = await get_treatment_recommendations(manager, 'MRN123456')
for rec in recommendations:
    print(f"📋 {rec['condition'].title()} - {rec['guideline']}")
    print(f"   Organization: {rec['organization']} (Evidence Level: {rec['evidence_level']})")
    for recommendation in rec['recommendations']:
        print(f"   • {recommendation}")
```

### Drug Dosage Calculation with Safety Rules

```python
async def calculate_safe_dosage(manager, patient_mrn, drug_rxnorm):
    """Calculate drug dosage considering patient factors and safety rules"""

    results = await manager.cross_kb_query(
        [KnowledgeBase.KB1_PATIENT, KnowledgeBase.KB3_DRUG_CALCULATIONS, KnowledgeBase.KB4_SAFETY_RULES],
        """
        MATCH (p:Patient:KB1_PatientStream {mrn: $patient_mrn})
        MATCH (c:CalculationRule:KB3_DrugCalculationStream {drug_rxnorm: $drug_rxnorm})
        OPTIONAL MATCH (s:SafetyRule:KB4_SafetyStream)
        WHERE $drug_rxnorm IN s.applicable_drugs AND
              any(condition IN p.conditions WHERE condition IN s.contraindicated_conditions)

        RETURN p.name, p.conditions, p.allergies,
               c.base_dose, c.unit, c.indication, c.adjustment_factors,
               collect(s.rule_text) AS safety_warnings
        """,
        {'patient_mrn': patient_mrn, 'drug_rxnorm': drug_rxnorm}
    )

    if not results:
        return None

    result = results[0]
    return {
        'patient': result['p.name'],
        'base_dose': result['c.base_dose'],
        'unit': result['c.unit'],
        'indication': result['c.indication'],
        'safety_warnings': result['safety_warnings'],
        'patient_conditions': result['p.conditions'],
        'patient_allergies': result['p.allergies']
    }

# Example usage
dosage_info = await calculate_safe_dosage(manager, 'MRN123456', 'RX123456')
if dosage_info:
    print(f"💊 Dosage for {dosage_info['patient']}:")
    print(f"   Base dose: {dosage_info['base_dose']} {dosage_info['unit']}")
    if dosage_info['safety_warnings']:
        print("   ⚠️  Safety warnings:")
        for warning in dosage_info['safety_warnings']:
            print(f"      • {warning}")
```

## Health Monitoring

### Comprehensive System Health Check

```python
async def system_health_report(manager):
    """Generate comprehensive health report for all knowledge bases"""

    health_status = await manager.health_check_all_streams()

    report = {
        'timestamp': datetime.utcnow().isoformat(),
        'overall_healthy': True,
        'knowledge_bases': {}
    }

    for kb_name, status in health_status.items():
        kb_report = {
            'healthy': status.get('healthy', False),
            'primary_nodes': status.get('primary_nodes', 0),
            'semantic_nodes': status.get('semantic_nodes', 0),
            'total_nodes': status.get('primary_nodes', 0) + status.get('semantic_nodes', 0)
        }

        if not kb_report['healthy']:
            report['overall_healthy'] = False
            kb_report['error'] = status.get('error', 'Unknown error')

        report['knowledge_bases'][kb_name] = kb_report

    return report

# Example usage with monitoring
async def monitor_system_health(manager):
    report = await system_health_report(manager)

    print(f"🏥 CardioFit KB Health Report - {report['timestamp']}")
    print(f"Overall Status: {'✅ HEALTHY' if report['overall_healthy'] else '❌ UNHEALTHY'}")
    print()

    for kb_name, status in report['knowledge_bases'].items():
        status_icon = '✅' if status['healthy'] else '❌'
        print(f"{status_icon} {kb_name.upper()}")
        print(f"   Primary nodes: {status['primary_nodes']:,}")
        print(f"   Semantic nodes: {status['semantic_nodes']:,}")
        print(f"   Total nodes: {status['total_nodes']:,}")

        if not status['healthy']:
            print(f"   Error: {status.get('error', 'Unknown')}")
        print()

    return report

# Example usage
health_report = await monitor_system_health(manager)
```

## Integration Patterns

### Service Integration Pattern

```python
class ClinicalDecisionService:
    """Example service integrating with Multi-KB Stream Manager"""

    def __init__(self, neo4j_config):
        self.manager = MultiKBStreamManager(neo4j_config)
        self.initialized = False

    async def initialize(self):
        """Initialize the service and KB streams"""
        if not self.initialized:
            success = await self.manager.initialize_all_streams()
            if not success:
                raise Exception("Failed to initialize knowledge base streams")
            self.initialized = True

    async def get_clinical_alerts(self, patient_mrn):
        """Get all clinical alerts for a patient"""
        if not self.initialized:
            await self.initialize()

        # Get drug interaction alerts
        drug_alerts = await check_patient_drug_safety(self.manager, patient_mrn)

        # Get treatment recommendations
        treatment_recs = await get_treatment_recommendations(self.manager, patient_mrn)

        return {
            'patient_mrn': patient_mrn,
            'drug_interaction_alerts': drug_alerts,
            'treatment_recommendations': treatment_recs,
            'alert_count': len(drug_alerts),
            'recommendation_count': len(treatment_recs)
        }

    async def close(self):
        """Clean shutdown"""
        if self.manager:
            await self.manager.close()

# Example usage in a FastAPI service
from fastapi import FastAPI

app = FastAPI()
clinical_service = ClinicalDecisionService(neo4j_config)

@app.on_event("startup")
async def startup_event():
    await clinical_service.initialize()

@app.on_event("shutdown")
async def shutdown_event():
    await clinical_service.close()

@app.get("/patient/{patient_mrn}/clinical-alerts")
async def get_patient_alerts(patient_mrn: str):
    return await clinical_service.get_clinical_alerts(patient_mrn)
```

### Batch Data Loading Pattern

```python
async def bulk_load_patients(manager, patient_list):
    """Efficiently load multiple patients"""

    successful_loads = 0
    failed_loads = []

    for patient_data in patient_list:
        try:
            success = await load_patient(manager, patient_data)
            if success:
                successful_loads += 1
            else:
                failed_loads.append(patient_data['mrn'])
        except Exception as e:
            failed_loads.append({'mrn': patient_data['mrn'], 'error': str(e)})

    return {
        'total_patients': len(patient_list),
        'successful_loads': successful_loads,
        'failed_loads': failed_loads,
        'success_rate': successful_loads / len(patient_list) * 100
    }

# Example usage
patient_batch = [
    {'name': 'Alice Johnson', 'mrn': 'MRN001', 'dob': '1980-03-15', 'gender': 'F'},
    {'name': 'Bob Smith', 'mrn': 'MRN002', 'dob': '1975-08-22', 'gender': 'M'},
    # ... more patients
]

load_results = await bulk_load_patients(manager, patient_batch)
print(f"Loaded {load_results['successful_loads']} of {load_results['total_patients']} patients")
print(f"Success rate: {load_results['success_rate']:.1f}%")
```

This examples document demonstrates real-world usage patterns for integrating the Neo4j Multi-KB Stream Manager into clinical applications, emphasizing practical scenarios that healthcare systems commonly encounter.