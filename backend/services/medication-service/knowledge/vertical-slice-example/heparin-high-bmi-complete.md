# Vertical Slice Example: Heparin Infusion for High-BMI Patient

## Clinical Scenario
**Patient**: 45-year-old male, 150kg, BMI 42, presenting with acute STEMI requiring heparin anticoagulation

## Complete Knowledge Base Integration

### 1. Medication Knowledge Core (MKC)
```json
{
  "heparin_sodium": {
    "rxnorm_code": "5224",
    "generic_name": "heparin sodium",
    "brand_names": ["Hep-Lock", "Monoject"],
    "therapeutic_class": "anticoagulant",
    "mechanism": "antithrombin_activation",
    "indications": ["acute_coronary_syndrome", "pulmonary_embolism", "dvt"],
    "pharmacokinetics": {
      "half_life_hours": 1.5,
      "protein_binding_percent": 95,
      "renal_elimination_percent": 10,
      "hepatic_metabolism_percent": 90,
      "volume_distribution_l_per_kg": 0.07
    },
    "dosing": {
      "loading_dose_units_per_kg": 80,
      "initial_infusion_units_per_kg_per_hr": 18,
      "max_loading_dose_units": 10000,
      "route": "intravenous"
    },
    "safety_profile": {
      "bleeding_risk": true,
      "hit_risk": true,
      "requires_monitoring": true,
      "black_box_warnings": ["bleeding_risk", "hit_syndrome"],
      "pregnancy_category": "C"
    },
    "monitoring_requirements": [
      "aptt",
      "platelet_count",
      "bleeding_assessment",
      "hit_antibodies"
    ],
    "contraindications": [
      "active_bleeding",
      "severe_thrombocytopenia",
      "hit_history"
    ]
  }
}
```

### 2. Orchestrator Rule Base (ORB)
```yaml
# High-BMI Heparin Rule (Priority 100)
- id: "heparin-obesity-adjusted"
  priority: 100
  medication_code: "heparin_sodium"
  rule_name: "Heparin Obesity-Adjusted Dosing"
  
  conditions:
    medication:
      code: "heparin_sodium"
    patient_demographics:
      bmi: ">=40"
    clinical_context:
      indication: ["acute_coronary_syndrome", "stemi", "nstemi"]
      
  intent_manifest:
    recipe_id: "heparin-infusion-adult-v2.0"
    variant: "obesity_adjusted"
    data_requirements:
      - "actual_weight_kg"
      - "height_cm"
      - "bmi"
      - "baseline_aptt"
      - "platelet_count"
      - "bleeding_risk_factors"
    priority: "high"
    rationale: "High BMI requires weight-adjusted heparin dosing to prevent under-anticoagulation"

# Standard Heparin Rule (Priority 50)
- id: "heparin-standard"
  priority: 50
  medication_code: "heparin_sodium"
  rule_name: "Heparin Standard Dosing"
  
  conditions:
    medication:
      code: "heparin_sodium"
    patient_demographics:
      bmi: "<40"
      
  intent_manifest:
    recipe_id: "heparin-infusion-adult-v2.0"
    variant: "standard"
    data_requirements:
      - "actual_weight_kg"
      - "baseline_aptt"
      - "platelet_count"
    priority: "medium"
    rationale: "Standard weight-based heparin dosing protocol"
```

### 3. Clinical Recipe Book (CRB)
```yaml
recipe_id: "heparin-infusion-adult-v2.0"
version: "2.0.0"
medication_code: "heparin_sodium"
recipe_name: "Adult Heparin Infusion Protocol"

calculation_variants:
  standard:
    loading_dose:
      formula: "patient_weight_kg * 80"
      max_dose: 10000
      units: "units"
    
    initial_infusion:
      formula: "patient_weight_kg * 18"
      units: "units_per_hour"
    
    weight_source: "actual_weight"
    
  obesity_adjusted:
    loading_dose:
      formula: "adjusted_weight_kg * 80"
      max_dose: 10000
      units: "units"
    
    initial_infusion:
      formula: "adjusted_weight_kg * 18"
      units: "units_per_hour"
    
    weight_calculation:
      adjusted_weight_formula: "ideal_weight + 0.4 * (actual_weight - ideal_weight)"
      weight_cap_kg: 140
    
    weight_source: "adjusted_weight"

titration_protocol:
  target_aptt_range: [60, 80]
  monitoring_frequency: "6_hours_initially"
  
  adjustments:
    - aptt_range: [0, 35]
      action: "increase_by_4_units_per_kg_per_hr"
      recheck: "6_hours"
    - aptt_range: [36, 45]
      action: "increase_by_2_units_per_kg_per_hr"
      recheck: "6_hours"
    - aptt_range: [46, 70]
      action: "increase_by_1_units_per_kg_per_hr"
      recheck: "6_hours"
    - aptt_range: [71, 90]
      action: "no_change"
      recheck: "24_hours"
    - aptt_range: [91, 100]
      action: "decrease_by_1_units_per_kg_per_hr"
      recheck: "6_hours"
    - aptt_range: [101, 999]
      action: "hold_1_hour_then_decrease_by_3_units_per_kg_per_hr"
      recheck: "6_hours"

safety_checks:
  contraindications:
    absolute:
      - "active_bleeding"
      - "platelet_count < 50000"
    relative:
      - "recent_surgery < 24_hours"
      - "severe_hypertension"
  
  monitoring_requirements:
    aptt:
      frequency: "6_hours_until_stable"
      target_range: [60, 80]
    platelet_count:
      frequency: "daily"
      alert_threshold: "50%_decrease_or_<100000"
    bleeding_assessment:
      frequency: "every_shift"
```

### 4. Context Service Recipe Book (CSRB)
```yaml
recipe_id: "heparin-context-v1"
medication_code: "heparin_sodium"

base_requirements:
  patient_demographics:
    - field: "actual_weight_kg"
      source: "patient_service"
      required: true
    - field: "height_cm"
      source: "patient_service"
      required: true
    - field: "bmi"
      source: "patient_service"
      required: true
      calculation: "weight_kg / (height_m^2)"

  laboratory_values:
    - field: "baseline_aptt"
      source: "lab_service"
      required: true
      max_age_hours: 24
    - field: "platelet_count"
      source: "lab_service"
      required: true
      max_age_hours: 24

recipe_specific_requirements:
  "heparin-infusion-adult-v2.0":
    obesity_adjusted:
      additional_requirements:
        - field: "ideal_body_weight"
          calculation: "height_based_ibw"
          required: true
        - field: "adjusted_body_weight"
          calculation: "ibw + 0.4 * (actual_weight - ibw)"
          required: true
```

### 5. Evidence Repository (ER)
```json
{
  "guideline_id": "acc_aha_stemi_2023",
  "title": "2023 ACC/AHA STEMI Guidelines",
  "organization": "American College of Cardiology",
  "publication_date": "2023-01-15",
  "evidence_level": "Class_I_Level_A",
  "recommendations": {
    "heparin_dosing": {
      "standard_patients": "80 units/kg bolus, 18 units/kg/hr infusion",
      "obesity_adjustment": "Use adjusted body weight for patients BMI >40",
      "monitoring": "aPTT 6 hours after initiation, target 60-80 seconds"
    }
  },
  "references": [
    "PMID:36876737",
    "DOI:10.1161/CIR.0000000000001127"
  ]
}
```

### 6. Monitoring Requirements Database (MRD)
```json
{
  "protocol_id": "heparin_monitoring_protocol_v2",
  "medication": "heparin_sodium",
  "monitoring_schedule": {
    "aptt": {
      "initial": "6_hours_after_start",
      "frequency": "every_6_hours_until_therapeutic",
      "stable": "daily_when_stable",
      "target_range": [60, 80],
      "critical_values": {
        "low": "<35",
        "high": ">100"
      }
    },
    "platelet_count": {
      "baseline": "before_initiation",
      "frequency": "daily",
      "hit_surveillance": "every_2_days_after_day_4",
      "critical_threshold": "50%_decrease_or_<100000"
    },
    "bleeding_assessment": {
      "frequency": "every_shift",
      "parameters": ["hemoglobin", "hematocrit", "clinical_bleeding"]
    }
  },
  "alert_conditions": [
    "aptt_>100_seconds",
    "platelet_drop_>50_percent",
    "clinical_bleeding",
    "hemoglobin_drop_>2_g_dl"
  ]
}
```

### 7. Formulary & Cost Database (FCD)
```json
{
  "medication_code": "heparin_sodium",
  "formulations": [
    {
      "strength": "5000_units_per_ml",
      "package_size": "10_ml_vial",
      "ndc": "0409-2720-01",
      "formulary_status": "preferred",
      "acquisition_cost_usd": 12.50,
      "billing_code": "J1644",
      "restrictions": "none"
    }
  ],
  "insurance_coverage": {
    "medicare": "covered",
    "medicaid": "covered",
    "commercial": "covered",
    "prior_authorization": false
  }
}
```

## Integration Flow Example

**Input**: 150kg patient with BMI 42 needs heparin for STEMI

**ORB Decision**: 
- Evaluates BMI ≥40 → Matches "heparin-obesity-adjusted" rule (Priority 100)
- Generates Intent Manifest: recipe_id="heparin-infusion-adult-v2.0", variant="obesity_adjusted"

**Context Fetch**: 
- Retrieves: actual_weight=150kg, height=185cm, BMI=42, baseline_aPTT=32sec, platelets=250k

**Recipe Execution**:
- Calculates IBW = 79kg
- Calculates adjusted weight = 79 + 0.4*(150-79) = 107.4kg
- Loading dose = 107.4 * 80 = 8,592 units
- Initial infusion = 107.4 * 18 = 1,933 units/hr

**Output**: Complete heparin protocol with obesity-adjusted dosing, monitoring plan, and safety alerts
