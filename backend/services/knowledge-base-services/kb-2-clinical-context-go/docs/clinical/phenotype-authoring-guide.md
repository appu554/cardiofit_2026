# Phenotype Authoring Guide

## Overview

This guide provides comprehensive instructions for clinical informaticists on creating, validating, and maintaining clinical phenotype rules using CEL (Common Expression Language) within the KB-2 Clinical Context service.

## Table of Contents

1. [Phenotype Development Lifecycle](#phenotype-development-lifecycle)
2. [CEL Syntax for Clinical Rules](#cel-syntax-for-clinical-rules)
3. [Clinical Data Model](#clinical-data-model)
4. [Rule Development Standards](#rule-development-standards)
5. [Validation Procedures](#validation-procedures)
6. [Testing Framework](#testing-framework)
7. [Version Control](#version-control)
8. [Quality Assurance](#quality-assurance)
9. [Examples and Templates](#examples-and-templates)
10. [Troubleshooting](#troubleshooting)

## Phenotype Development Lifecycle

### Phase 1: Clinical Requirements Gathering

#### 1.1 Clinical Need Assessment
- **Stakeholder Identification**: Clinical champions, informaticists, quality teams
- **Use Case Definition**: Specific clinical scenarios and decision points
- **Outcome Objectives**: Expected clinical outcomes and benefits
- **Success Metrics**: Measurable criteria for phenotype effectiveness

#### 1.2 Evidence Review
- **Literature Search**: Systematic review of clinical evidence
- **Guideline Analysis**: Current clinical practice guidelines
- **Expert Consultation**: Board-certified physician input
- **Best Practice Review**: Industry standards and benchmarks

#### 1.3 Requirements Documentation
```yaml
phenotype_requirements:
  id: "high_cardiovascular_risk"
  name: "High Cardiovascular Risk Assessment"
  clinical_domain: "cardiovascular"
  use_cases:
    - "Primary care preventive screening"
    - "Specialist referral triage"
    - "Population health management"
  target_population: "Adults 40-79 years"
  evidence_base: "ACC/AHA 2019 Prevention Guidelines"
  expected_prevalence: "15-25% in primary care"
  clinical_champion: "Dr. Sarah Johnson, Cardiology"
```

### Phase 2: Technical Implementation

#### 2.1 Data Requirements Analysis
- **Required Data Elements**: Age, conditions, labs, medications, vitals
- **Data Quality Assessment**: Completeness, accuracy, timeliness requirements
- **Missing Data Strategy**: Handling incomplete or missing clinical data
- **Data Source Mapping**: EHR field mapping and transformation rules

#### 2.2 Rule Logic Development
- **CEL Expression Creation**: Translate clinical logic to CEL syntax
- **Business Rules Documentation**: Clear natural language rule description
- **Edge Case Handling**: Boundary conditions and exceptional scenarios
- **Performance Optimization**: Efficient rule evaluation strategies

#### 2.3 Testing Strategy
- **Unit Testing**: Individual rule component validation
- **Integration Testing**: End-to-end phenotype evaluation
- **Clinical Testing**: Real patient data validation
- **Performance Testing**: Scalability and response time validation

### Phase 3: Clinical Validation

#### 3.1 Expert Review Process
- **Clinical Expert Panel**: 2-3 board-certified physicians
- **Rule Logic Review**: Clinical accuracy and appropriateness assessment
- **Evidence Base Validation**: Supporting literature and guidelines review
- **Use Case Scenarios**: Clinical workflow integration assessment

#### 3.2 Test Case Development
- **Positive Cases**: Patients who should match the phenotype
- **Negative Cases**: Patients who should not match
- **Edge Cases**: Boundary conditions and complex scenarios
- **Real-World Cases**: De-identified patient examples

#### 3.3 Accuracy Validation
- **Inter-rater Reliability**: Multiple clinician agreement assessment
- **Sensitivity Analysis**: True positive rate measurement
- **Specificity Analysis**: True negative rate measurement
- **Clinical Utility**: Provider feedback and workflow impact

### Phase 4: Production Deployment

#### 4.1 Pilot Testing
- **Limited Rollout**: Small clinical user group testing
- **Performance Monitoring**: System performance and accuracy tracking
- **User Feedback**: Clinical user experience and satisfaction
- **Iterative Improvement**: Rule refinement based on feedback

#### 4.2 Full Deployment
- **Rollout Planning**: Phased deployment across clinical sites
- **Training Delivery**: Clinical user education and competency
- **Support Planning**: Help desk and clinical informaticist support
- **Monitoring Setup**: Continuous performance and quality monitoring

#### 4.3 Ongoing Maintenance
- **Performance Monitoring**: Continuous accuracy and utility assessment
- **Clinical Updates**: Evidence-based rule updates and improvements
- **Version Management**: Controlled rule versioning and deployment
- **Quality Assurance**: Regular clinical validation and review cycles

## CEL Syntax for Clinical Rules

### Basic CEL Concepts

#### 4.1 Data Types and Operations

**Primitive Types**
```cel
// Numeric operations
age >= 65
lab_value('hba1c') > 8.0
bmi < 18.5 || bmi > 30.0

// String operations
gender == 'male'
medication_name.contains('metformin')
condition_code.startsWith('E11')

// Boolean operations
has_diabetes && has_hypertension
smoking_status == 'current' || smoking_status == 'former'
!has_allergy('penicillin')

// Date/time operations
days_since(last_hba1c_date) < 90
medication_duration('statin') > duration('6M')
age_at_diagnosis('diabetes') < 30
```

#### 4.2 Clinical Helper Functions

**Condition Functions**
```cel
// Check if patient has specific condition
has_condition('diabetes_type_2')
has_condition('hypertension', 'active')

// Check condition by ICD-10 code
has_icd10_code('E11.9')  // Type 2 diabetes without complications
has_icd10_range('I10', 'I15')  // Hypertensive diseases

// Condition with time constraints
has_condition_since('diabetes', duration('1Y'))
condition_onset_age('diabetes') < 30
```

**Laboratory Value Functions**
```cel
// Get most recent lab value
lab_value('hba1c')
lab_value('creatinine', 'mg/dL')

// Lab values with time constraints
lab_value('hba1c', within_days(90))
lab_value('ldl_cholesterol', within_days(365))

// Lab value trends
lab_trend('hba1c', '6M') > 0.5  // Increasing trend
lab_average('glucose', '30D') > 180

// Lab value comparisons
lab_value('creatinine') > reference_range('creatinine').upper
egfr_calculated() < 60
```

**Medication Functions**
```cel
// Current medications
has_medication('metformin')
has_medication_class('ace_inhibitor')
has_medication_ingredient('lisinopril')

// Medication history and duration
medication_duration('statin') > duration('6M')
has_medication_history('insulin', within_years(2))
medication_adherence('metformin') > 0.8

// Drug interactions and contraindications
has_drug_interaction('warfarin', 'aspirin')
has_contraindication('metformin', 'kidney_disease')
```

**Vital Signs Functions**
```cel
// Recent vitals
vitals.systolic_bp > 140
vitals.diastolic_bp > 90
vitals.heart_rate < 60 || vitals.heart_rate > 100

// Vital trends
bp_average('3M').systolic > 140
weight_change('6M') > 10  // 10 kg weight gain

// BMI calculations
bmi_calculated() > 30
bmi_category() == 'obese'
```

### Advanced CEL Patterns

#### 5.1 Complex Clinical Logic

**Cardiovascular Risk Assessment**
```cel
// High cardiovascular risk phenotype
(age >= 65 && gender == 'male') || (age >= 75 && gender == 'female') &&
(
  // Diabetes with additional risk factors
  (has_condition('diabetes') && 
   (lab_value('hba1c') > 7.0 || 
    has_condition('diabetic_nephropathy') || 
    medication_duration('insulin') > duration('1Y'))) ||
    
  // Multiple cardiovascular risk factors
  (count_conditions(['hypertension', 'hyperlipidemia', 'smoking']) >= 2 &&
   (lab_value('ldl_cholesterol') > 130 || 
    vitals.systolic_bp > 140 || 
    smoking_status == 'current')) ||
    
  // Established cardiovascular disease
  has_condition(['coronary_artery_disease', 'cerebrovascular_disease', 
                 'peripheral_arterial_disease'])
) &&
// Exclude patients with contraindications or optimal therapy
!has_condition('end_stage_renal_disease') &&
!has_medication_class('statin', therapeutic_dose=true) &&
!has_condition('active_cancer')
```

**Diabetes Complications Screening**
```cel
// Diabetes complications phenotype
has_condition('diabetes') &&
diabetes_duration() > duration('5Y') &&
(
  // Poor glycemic control
  (lab_value('hba1c', within_days(180)) > 8.0 ||
   glucose_readings_above(250, count=3, within_days(30))) ||
   
  // Evidence of complications
  has_condition(['diabetic_retinopathy', 'diabetic_nephropathy', 
                 'diabetic_neuropathy']) ||
                 
  // Risk factors for complications
  (lab_value('urine_microalbumin') > 30 &&
   !has_medication_class('ace_inhibitor') &&
   !has_medication_class('arb')) ||
   
  // Suboptimal monitoring
  days_since_last('ophthalmology_exam') > 365 ||
  days_since_last('foot_exam') > 365
) &&
// Age and life expectancy considerations
age < 80 && !has_condition('limited_life_expectancy')
```

#### 5.2 Data Quality and Validation

**Missing Data Handling**
```cel
// Require essential data for evaluation
has_data(['age', 'gender']) &&
(
  // Phenotype logic with data quality checks
  age >= 18 && age <= 120 &&  // Reasonable age range
  
  // Use available data with fallbacks
  (has_lab('hba1c', within_days(365)) ? 
   lab_value('hba1c') > 7.0 : 
   (has_condition('diabetes') && !has_optimal_diabetes_therapy())) &&
   
  // Handle missing medication data
  (has_medication_data() ? 
   !has_medication_class('statin') : 
   lab_value('ldl_cholesterol', within_days(365)) > 100)
)
```

**Confidence Scoring**
```cel
// Calculate rule confidence based on data availability
confidence_score = 
  (has_lab('hba1c', within_days(90)) ? 0.3 : 0.1) +  // Recent labs
  (medication_data_complete() ? 0.2 : 0.05) +        // Medication history
  (has_condition_detail('diabetes') ? 0.3 : 0.15) +  // Condition specificity
  (vitals_recent(within_days(30)) ? 0.2 : 0.1)       // Recent vitals

// Only trigger phenotype if confidence is adequate
phenotype_positive && confidence_score >= 0.7
```

### Rule Development Standards

#### 6.1 Naming Conventions

**Phenotype Identifiers**
```yaml
# Pattern: [domain]_[condition]_[specificity]
high_cardiovascular_risk          # Cardiovascular domain, high risk
diabetes_poor_control            # Diabetes domain, control status
medication_interaction_risk      # Medication domain, safety risk
ckd_stage_3_progression         # Renal domain, specific stage

# Version suffixes for rule iterations
high_cardiovascular_risk_v2      # Second version
diabetes_poor_control_aha2023    # Guideline-specific version
```

**Rule Variables**
```cel
// Use descriptive variable names
diabetes_duration_years = years_since_diagnosis('diabetes')
optimal_statin_therapy = has_medication_class('statin', high_intensity=true)
cardiovascular_risk_factors = count_conditions(['hypertension', 'diabetes', 'smoking'])

// Avoid abbreviations and single letters
// Good: current_medications, recent_hba1c
// Bad: cur_meds, hba1c_rec
```

#### 6.2 Documentation Standards

**Rule Header Documentation**
```yaml
phenotype:
  id: "high_cardiovascular_risk"
  version: "1.2"
  name: "High Cardiovascular Risk Assessment"
  description: "Identifies patients at high risk for cardiovascular events based on ACC/AHA guidelines"
  
  clinical_rationale: |
    This phenotype implements the 2019 ACC/AHA Primary Prevention Guidelines
    for cardiovascular risk assessment. It identifies patients who would benefit
    from statin therapy or intensive lifestyle interventions.
  
  evidence_base:
    - "2019 ACC/AHA Guideline on Primary Prevention (DOI: 10.1161/CIR.0000000000000678)"
    - "Pooled Cohort Equations for 10-year ASCVD risk"
  
  target_population: "Adults 40-79 years without established ASCVD"
  expected_prevalence: "15-20% in primary care populations"
  
  clinical_champion: "Dr. Sarah Johnson, Cardiology"
  informaticist: "John Smith, MS, Clinical Informatics"
  last_updated: "2025-01-15"
  next_review: "2025-07-15"
  
  validation_status:
    clinical_expert_review: "Approved - Dr. Johnson, Dr. Brown"
    test_case_validation: "Passed - 98.5% accuracy"
    pilot_deployment: "Completed - 3 sites, positive feedback"
    production_status: "Active"
```

**Inline Rule Documentation**
```cel
// High Cardiovascular Risk Assessment
// Based on 2019 ACC/AHA Primary Prevention Guidelines

// Age criteria (evidence: age is strongest predictor)
(age >= 65 && gender == 'male') || (age >= 75 && gender == 'female') &&

// Major risk factors (evidence: each doubles CV risk)
(
  // Diabetes with additional risk (evidence: diabetes equivalent to CHD)
  has_condition('diabetes') && 
  (lab_value('hba1c') > 7.0 ||  // Poor control increases risk
   duration_condition('diabetes') > duration('10Y')) ||  // Long duration
   
  // Multiple traditional risk factors (evidence: multiplicative risk)
  (count_risk_factors(['hypertension', 'smoking', 'family_history_cad']) >= 2 &&
   lab_value('ldl_cholesterol') > 130)  // LDL threshold per guidelines
) &&

// Exclusions (evidence: limited benefit or contraindications)
!has_condition('active_cancer') &&  // Limited life expectancy
age < 80  // Age limit per guidelines for primary prevention
```

#### 6.3 Testing Standards

**Test Case Requirements**
```yaml
test_cases:
  positive_cases:
    - name: "Elderly male with diabetes and hypertension"
      patient_data:
        age: 68
        gender: "male"
        conditions: ["diabetes_type_2", "hypertension"]
        labs:
          hba1c: {"value": 8.2, "unit": "%", "date": "2025-01-10"}
          ldl_cholesterol: {"value": 145, "unit": "mg/dL", "date": "2025-01-10"}
      expected_result: true
      rationale: "Meets age, diabetes, and poor control criteria"
      
  negative_cases:
    - name: "Young healthy adult"
      patient_data:
        age: 35
        gender: "female"
        conditions: []
        labs:
          total_cholesterol: {"value": 180, "unit": "mg/dL", "date": "2025-01-10"}
      expected_result: false
      rationale: "Below age threshold with no risk factors"
      
  edge_cases:
    - name: "Borderline age with single risk factor"
      patient_data:
        age: 64
        gender: "male"
        conditions: ["hypertension"]
        vitals:
          systolic_bp: 145
      expected_result: false
      rationale: "Age threshold is 65 for males"
```

## Validation Procedures

### Clinical Validation Process

#### 7.1 Expert Review Panel

**Panel Composition**
- **Primary Reviewer**: Board-certified physician in relevant specialty
- **Secondary Reviewer**: Clinical informaticist or physician champion
- **Quality Reviewer**: Quality assurance or patient safety representative
- **Optional**: Pharmacist (for medication-related phenotypes)

**Review Criteria**
```yaml
clinical_review_checklist:
  clinical_accuracy:
    - "Does the rule logic accurately represent clinical reasoning?"
    - "Are the thresholds clinically appropriate and evidence-based?"
    - "Does the rule handle edge cases appropriately?"
    
  evidence_base:
    - "Is the evidence base current and high-quality?"
    - "Are guideline recommendations correctly implemented?"
    - "Are any evidence gaps appropriately acknowledged?"
    
  clinical_utility:
    - "Will this phenotype improve clinical decision-making?"
    - "Is the expected prevalence reasonable for the target population?"
    - "Are there potential unintended consequences?"
    
  integration:
    - "Does this fit well within existing clinical workflows?"
    - "Are there dependencies on other systems or data sources?"
    - "Is the clinical context appropriate for the rule complexity?"
```

#### 7.2 Test Case Validation

**Validation Dataset Requirements**
- **Size**: Minimum 1,000 patients for common phenotypes, 500 for rare conditions
- **Diversity**: Representative age, gender, comorbidity distribution
- **Data Quality**: Complete data for all rule-required elements
- **Ground Truth**: Expert-adjudicated expected outcomes

**Validation Metrics**
```yaml
validation_metrics:
  accuracy_metrics:
    sensitivity: ">= 0.95"      # True positive rate
    specificity: ">= 0.90"      # True negative rate
    positive_predictive_value: ">= 0.85"  # Precision
    negative_predictive_value: ">= 0.95"  # NPV
    
  clinical_metrics:
    clinical_agreement: ">= 0.90"  # Inter-rater reliability
    clinical_utility: ">= 4.0/5.0"  # Provider satisfaction
    workflow_integration: ">= 4.0/5.0"  # Workflow fit
    
  performance_metrics:
    evaluation_time: "< 100ms"    # Response time SLA
    cache_hit_rate: "> 0.95"      # Caching effectiveness
    error_rate: "< 0.01"          # Technical error rate
```

#### 7.3 Pilot Testing

**Pilot Design**
- **Sites**: 2-3 representative clinical sites
- **Duration**: 4-6 weeks minimum
- **Users**: 10-20 clinical users per site
- **Monitoring**: Real-time performance and feedback collection

**Pilot Success Criteria**
```yaml
pilot_success_criteria:
  technical_performance:
    - "Response time SLA met >99% of requests"
    - "Zero critical errors or system failures"
    - "Cache hit rate >95%"
    
  clinical_performance:
    - "Clinical accuracy >95% in real-world scenarios"
    - "Provider satisfaction >4.0/5.0"
    - "No patient safety incidents"
    
  workflow_integration:
    - "Successful EHR integration with <5% alert override rate"
    - "Training completion >95% of users"
    - "Support ticket rate <5 per 100 users"
```

## Testing Framework

### Automated Testing

#### 8.1 Unit Testing

**Rule Component Testing**
```go
// Example unit test for CEL rule components
func TestCardiovascularRiskAgeRequirement(t *testing.T) {
    tests := []struct {
        name     string
        patient  Patient
        expected bool
    }{
        {
            name: "Male 65 meets age requirement",
            patient: Patient{Age: 65, Gender: "male"},
            expected: true,
        },
        {
            name: "Male 64 does not meet age requirement", 
            patient: Patient{Age: 64, Gender: "male"},
            expected: false,
        },
        {
            name: "Female 75 meets age requirement",
            patient: Patient{Age: 75, Gender: "female"},
            expected: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := evaluateAgeRequirement(tt.patient)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Data Quality Testing**
```go
func TestDataQualityValidation(t *testing.T) {
    // Test missing required data
    patient := Patient{
        ID: "test_001",
        // Missing age and gender
    }
    
    result, err := evaluatePhenotype("high_cardiovascular_risk", patient)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "required data missing")
    
    // Test data out of range
    patient = Patient{
        ID: "test_002", 
        Age: 150,  // Invalid age
        Gender: "male",
    }
    
    result, err = evaluatePhenotype("high_cardiovascular_risk", patient)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "age out of valid range")
}
```

#### 8.2 Integration Testing

**End-to-End Phenotype Testing**
```go
func TestPhenotypeEvaluationEndToEnd(t *testing.T) {
    // Test complete phenotype evaluation pipeline
    testCases := loadTestCases("cardiovascular_risk_test_cases.yaml")
    
    for _, tc := range testCases {
        t.Run(tc.Name, func(t *testing.T) {
            // Execute complete evaluation
            result, err := evaluatePhenotypes(tc.PatientData, []string{"cardiovascular"})
            
            assert.NoError(t, err)
            assert.Equal(t, tc.ExpectedResult, result.Phenotypes[0].Positive)
            assert.GreaterOrEqual(t, result.Confidence, tc.MinConfidence)
            assert.Less(t, result.ProcessingTimeMs, 100)  // SLA requirement
        })
    }
}
```

#### 8.3 Performance Testing

**Load Testing**
```go
func TestPhenotypeEvaluationPerformance(t *testing.T) {
    // Generate test patient data
    patients := generateTestPatients(1000)
    
    startTime := time.Now()
    
    // Execute batch evaluation
    results, err := evaluatePhenotypesBatch(patients, []string{"cardiovascular", "diabetes"})
    
    processingTime := time.Since(startTime)
    
    assert.NoError(t, err)
    assert.Len(t, results, 1000)
    assert.Less(t, processingTime, 5*time.Second)  // 5 seconds for 1000 patients
    
    // Verify individual response times
    for _, result := range results {
        assert.Less(t, result.ProcessingTimeMs, 100)
    }
}
```

### Manual Testing Procedures

#### 9.1 Clinical Scenario Testing

**Test Scenario Template**
```yaml
clinical_scenario:
  id: "cv_risk_scenario_001"
  name: "Elderly diabetic with poor control"
  description: "65-year-old male with diabetes, hypertension, and elevated HbA1c"
  
  patient_profile:
    demographics:
      age: 65
      gender: "male"
      ethnicity: "hispanic"
    
    medical_history:
      conditions:
        - name: "diabetes_type_2"
          diagnosis_date: "2020-03-15"
          status: "active"
        - name: "hypertension" 
          diagnosis_date: "2018-06-20"
          status: "active"
    
    current_medications:
      - name: "metformin"
        dosage: "1000mg"
        frequency: "twice_daily"
        start_date: "2020-03-15"
      - name: "lisinopril"
        dosage: "10mg"
        frequency: "once_daily"  
        start_date: "2018-06-20"
    
    recent_labs:
      - name: "hba1c"
        value: 8.7
        unit: "%"
        date: "2025-01-10"
      - name: "ldl_cholesterol"
        value: 155
        unit: "mg/dL"
        date: "2025-01-10"
    
    recent_vitals:
      - systolic_bp: 148
        diastolic_bp: 92
        date: "2025-01-15"
  
  expected_outcomes:
    phenotypes:
      - id: "high_cardiovascular_risk"
        expected: true
        rationale: "Age ≥65, diabetes with poor control, hypertension, elevated LDL"
      - id: "diabetes_poor_control"
        expected: true
        rationale: "HbA1c 8.7% indicates poor glycemic control"
    
    risk_assessments:
      - category: "cardiovascular"
        expected_risk_level: "high" 
        expected_score_range: [0.7, 1.0]
    
    treatment_preferences:
      - condition: "diabetes"
        expected_recommendations: ["sglt2_inhibitor", "statin_therapy"]
  
  test_execution:
    steps:
      1. "Load patient data into test environment"
      2. "Execute phenotype evaluation API call"
      3. "Verify phenotype results match expected outcomes"
      4. "Execute risk assessment API call"
      5. "Verify risk scores within expected ranges"
      6. "Execute treatment preference API call" 
      7. "Verify recommendations align with clinical expectations"
    
    success_criteria:
      - "All expected phenotypes correctly identified"
      - "Risk scores within expected ranges"
      - "Treatment recommendations clinically appropriate"
      - "Response times <100ms for phenotype evaluation"
      - "No technical errors during execution"
```

#### 9.2 User Acceptance Testing

**Clinical User Testing Protocol**
```yaml
user_acceptance_testing:
  participants:
    - role: "Primary Care Physician"
      count: 3
      experience: "5+ years clinical experience"
    - role: "Clinical Pharmacist" 
      count: 2
      experience: "3+ years clinical experience"
    - role: "Nurse Practitioner"
      count: 2
      experience: "3+ years clinical experience"
  
  testing_scenarios:
    - scenario: "Routine primary care visit"
      description: "Evaluate phenotypes during standard patient encounter"
      duration: "30 minutes"
      success_criteria:
        - "Phenotype results useful for clinical decision-making"
        - "Integration with EHR workflow seamless"
        - "No workflow disruption or delays"
    
    - scenario: "Medication prescribing"
      description: "Use treatment preferences during medication selection"
      duration: "20 minutes" 
      success_criteria:
        - "Treatment recommendations clinically appropriate"
        - "Drug interaction alerts accurate and actionable"
        - "Formulary preferences align with institutional guidelines"
  
  evaluation_criteria:
    usability:
      - "System easy to learn and use"
      - "Information presented clearly and intuitively"
      - "Workflow integration smooth and efficient"
    
    clinical_utility:
      - "Phenotype results clinically relevant and actionable"
      - "Risk assessments inform clinical decision-making"
      - "Treatment recommendations evidence-based and appropriate"
    
    performance:
      - "Response times acceptable for clinical workflow"
      - "System reliable with no technical failures"
      - "Alert frequency appropriate (not excessive)"
```

## Version Control and Change Management

### Rule Versioning Strategy

#### 10.1 Versioning Scheme

**Semantic Versioning for Clinical Rules**
```
Version Format: MAJOR.MINOR.PATCH

MAJOR: Incompatible clinical logic changes
  - New evidence significantly changes rule behavior
  - Threshold changes affecting >20% of population
  - Rule structure or data requirements changes

MINOR: Backward compatible clinical enhancements
  - Additional risk factors or refinements
  - Improved edge case handling
  - Performance optimizations

PATCH: Bug fixes and minor corrections
  - Syntax corrections
  - Documentation updates
  - Minor threshold adjustments (<5% population impact)

Examples:
  high_cardiovascular_risk_v1.0.0  # Initial version
  high_cardiovascular_risk_v1.1.0  # Added family history factor
  high_cardiovascular_risk_v1.1.1  # Fixed age threshold bug
  high_cardiovascular_risk_v2.0.0  # Updated to 2025 guidelines
```

#### 10.2 Change Control Process

**Change Request Documentation**
```yaml
change_request:
  id: "CR-2025-001"
  phenotype_id: "high_cardiovascular_risk"
  current_version: "1.1.1"
  proposed_version: "2.0.0"
  
  change_description: "Update to incorporate 2025 ACC/AHA guidelines"
  
  clinical_justification: |
    The 2025 ACC/AHA guidelines introduce new risk factors including
    kidney disease markers and refined age thresholds based on recent
    clinical trial evidence.
  
  evidence_base:
    - "2025 ACC/AHA Primary Prevention Guidelines (DOI: pending)"
    - "PREVENT equations for cardiovascular risk estimation"
    - "Clinical trial data supporting CKD as risk enhancer"
  
  impact_analysis:
    population_affected: "Estimated 15% change in phenotype prevalence"
    clinical_workflow: "No workflow changes required"
    technical_dependencies: "Requires eGFR calculation capability"
    training_required: "1-hour clinical update session"
  
  implementation_plan:
    development_effort: "40 hours"
    testing_effort: "24 hours"
    clinical_validation: "2 weeks"
    deployment_timeline: "6 weeks total"
  
  approval_required:
    - "Clinical Governance Committee"
    - "Cardiovascular Clinical Champion"
    - "Clinical Informatics Director"
    - "Medical Director"
```

**Change Approval Workflow**
```
Change Request → Clinical Review → Technical Assessment → 
Impact Analysis → Stakeholder Approval → Implementation → 
Testing → Clinical Validation → Production Deployment → 
Post-deployment Monitoring
```

#### 10.3 Migration and Rollback Procedures

**Version Migration Strategy**
```yaml
migration_plan:
  migration_type: "blue_green"  # Zero-downtime deployment
  
  pre_migration:
    - "Backup current rule definitions"
    - "Prepare rollback procedures"
    - "Validate test environment deployment"
    - "Notify clinical stakeholders"
  
  migration_execution:
    - "Deploy new version to production environment"
    - "Run parallel evaluation for 24 hours"
    - "Compare results between versions"
    - "Monitor performance metrics"
    - "Validate clinical accuracy"
  
  post_migration:
    - "Monitor system performance for 72 hours"
    - "Collect clinical user feedback"
    - "Document any issues or improvements"
    - "Update documentation and training materials"
  
  rollback_criteria:
    - "Clinical accuracy drops below 95%"
    - "Performance SLA violations >5%"
    - "Critical patient safety concerns"
    - "User acceptance issues"
  
  rollback_procedure:
    - "Immediate: Switch to previous version"
    - "Notify clinical stakeholders within 30 minutes"
    - "Document rollback rationale and timeline"
    - "Plan remediation and re-deployment"
```

## Quality Assurance

### Continuous Quality Monitoring

#### 11.1 Performance Monitoring

**Real-time Quality Metrics**
```yaml
quality_metrics:
  clinical_accuracy:
    metric: "phenotype_accuracy_rate"
    target: ">= 98%"
    measurement: "Expert review of random sample (n=100 weekly)"
    alert_threshold: "< 95%"
    escalation: "Clinical Informatics Director"
  
  system_performance:
    metric: "response_time_p95"
    target: "< 100ms"
    measurement: "Continuous monitoring"
    alert_threshold: "> 150ms"
    escalation: "Platform Engineering"
  
  clinical_utility:
    metric: "provider_satisfaction"
    target: ">= 4.0/5.0"
    measurement: "Monthly user survey"
    alert_threshold: "< 3.5/5.0"
    escalation: "Clinical Champion"
```

**Quality Dashboard Components**
- **Accuracy Trends**: Clinical validation results over time
- **Performance Metrics**: Response time and throughput monitoring
- **User Satisfaction**: Provider feedback and satisfaction scores
- **Usage Analytics**: Phenotype evaluation frequency and patterns
- **Error Tracking**: Technical and clinical error rates

#### 11.2 Clinical Validation Cycles

**Quarterly Validation Process**
```yaml
quarterly_validation:
  scope: "All active phenotypes"
  methodology: "Expert review + retrospective analysis"
  
  validation_activities:
    week_1:
      - "Generate random patient sample (n=200 per phenotype)"
      - "Execute phenotype evaluations"
      - "Prepare cases for expert review"
    
    week_2:
      - "Expert panel review sessions"
      - "Calculate agreement metrics"
      - "Identify discrepancies and edge cases"
    
    week_3:
      - "Analyze performance trends"
      - "Review user feedback and support tickets"
      - "Assess clinical outcome correlations"
    
    week_4:
      - "Prepare validation report"
      - "Develop improvement recommendations"
      - "Present findings to Clinical Governance Committee"
  
  deliverables:
    - "Validation accuracy report"
    - "Performance trend analysis"
    - "Clinical utility assessment"
    - "Improvement action plan"
```

#### 11.3 Continuous Improvement Process

**Improvement Identification**
- **User Feedback**: Clinical user suggestions and pain points
- **Performance Analysis**: System metrics and optimization opportunities  
- **Evidence Updates**: New clinical evidence and guideline changes
- **Technology Advances**: New capabilities and integration opportunities

**Improvement Implementation**
```yaml
improvement_lifecycle:
  identification:
    sources: ["user_feedback", "performance_metrics", "clinical_evidence", "technology_updates"]
    prioritization_criteria: ["clinical_impact", "patient_safety", "user_experience", "technical_feasibility"]
  
  assessment:
    clinical_review: "Clinical Governance Committee evaluation"
    technical_assessment: "Engineering feasibility and effort estimation"
    business_case: "Cost-benefit analysis and ROI calculation"
  
  implementation:
    development: "Agile development process with clinical validation"
    testing: "Comprehensive testing including clinical scenarios"
    deployment: "Controlled rollout with monitoring"
  
  evaluation:
    success_metrics: "Pre-defined success criteria measurement"
    user_feedback: "Post-implementation user satisfaction assessment"
    performance_impact: "System performance and clinical outcome analysis"
```

---

**Document Control**
- **Version**: 1.0
- **Effective Date**: 2025-01-15
- **Review Date**: 2025-07-15
- **Owner**: Clinical Informatics Director
- **Approved By**: Clinical Governance Committee + Medical Director

**Clinical Oversight**
- **Medical Director**: Dr. Jennifer Walsh, MD, CMIO
- **Clinical Champion**: Dr. Sarah Johnson, MD, Cardiology
- **Clinical Informaticist**: John Smith, MS, RN, Clinical Informatics