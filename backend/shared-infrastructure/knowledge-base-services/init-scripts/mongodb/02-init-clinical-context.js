// KB-2 Clinical Context MongoDB Initialization
// Clinical context and phenotype data setup

// Switch to clinical_context database
db = db.getSiblingDB('clinical_context');

// Drop existing collections (for clean initialization)
db.clinical_contexts.drop();
db.phenotype_definitions.drop();
db.patient_profiles.drop();
db.contextual_insights.drop();
db.population_cohorts.drop();
db.clinical_patterns.drop();
db.context_cache.drop();

print("Initializing KB-2 Clinical Context Collections...");

// Insert sample phenotype definitions
db.phenotype_definitions.insertMany([
  {
    phenotype_id: "diabetes_t2_phenotype",
    name: "Type 2 Diabetes Mellitus Phenotype",
    description: "Algorithmic identification of patients with Type 2 Diabetes based on clinical indicators",
    category: "endocrine",
    severity: "moderate",
    criteria: {
      required_conditions: ["E11", "E11.9"],  // ICD-10 codes
      exclusion_conditions: ["E10"],          // Type 1 diabetes
      lab_value_rules: [
        {
          lab_code: "HBA1C",
          loinc_code: "4548-4",
          operator: "gte",
          value: 6.5,
          unit: "%",
          required: false,
          weight: 0.8
        },
        {
          lab_code: "GLUCOSE_FASTING",
          loinc_code: "1558-6", 
          operator: "gte",
          value: 126,
          unit: "mg/dL",
          required: false,
          weight: 0.7
        }
      ],
      medication_rules: [
        {
          medication_class: ["antidiabetic", "insulin"],
          required: false,
          weight: 0.6
        }
      ],
      age_restrictions: {
        min_age: 18
      },
      time_windows: {
        lookback_days: 365,
        required_within: 90
      }
    },
    icd10_codes: ["E11", "E11.0", "E11.1", "E11.2", "E11.9"],
    snomed_codes: ["44054006"],
    algorithm_type: "rule_based",
    algorithm: {
      type: "rule_based",
      rule_logic: "OR",
      thresholds: {
        minimum_score: 0.7,
        high_confidence: 0.85
      }
    },
    validation_data: {
      validation_dataset: "EHR_VALIDATION_2024",
      ppv: 0.92,
      npv: 0.88,
      sensitivity: 0.85,
      specificity: 0.90,
      f1_score: 0.88,
      validated_at: new Date("2024-01-15"),
      validated_by: "clinical_informatics_team"
    },
    created_at: new Date(),
    updated_at: new Date(),
    version: "1.2.0",
    status: "active"
  },
  {
    phenotype_id: "hypertension_phenotype",
    name: "Hypertension Phenotype",
    description: "Clinical phenotype for essential hypertension identification",
    category: "cardiovascular", 
    severity: "mild",
    criteria: {
      required_conditions: ["I10"],
      exclusion_conditions: ["I15"],  // Secondary hypertension
      lab_value_rules: [],
      medication_rules: [
        {
          medication_class: ["ace_inhibitor", "arb", "beta_blocker", "ccb", "diuretic"],
          required: false,
          weight: 0.5
        }
      ],
      age_restrictions: {
        min_age: 18
      },
      time_windows: {
        lookback_days: 730,  // 2 years lookback
        required_within: 180
      }
    },
    icd10_codes: ["I10", "I11", "I12", "I13"],
    snomed_codes: ["38341003", "59621000"],
    algorithm_type: "hybrid",
    algorithm: {
      type: "hybrid",
      rule_logic: "AND",
      model_details: {
        model_type: "logistic_regression",
        features: ["systolic_bp", "diastolic_bp", "age", "bmi", "medications"],
        version: "1.0.0",
        accuracy: 0.89,
        sensitivity: 0.87,
        specificity: 0.91
      },
      thresholds: {
        minimum_score: 0.6,
        high_confidence: 0.8
      }
    },
    validation_data: {
      validation_dataset: "CLINICAL_HTN_COHORT_2024", 
      ppv: 0.89,
      npv: 0.91,
      sensitivity: 0.87,
      specificity: 0.91,
      f1_score: 0.89,
      auc: 0.93,
      validated_at: new Date("2024-02-01"),
      validated_by: "cardiology_team"
    },
    created_at: new Date(),
    updated_at: new Date(),
    version: "2.0.1",
    status: "active"
  },
  {
    phenotype_id: "ckd_phenotype",
    name: "Chronic Kidney Disease Phenotype",
    description: "Multi-stage chronic kidney disease phenotype with GFR-based staging",
    category: "renal",
    severity: "moderate",
    criteria: {
      required_conditions: ["N18"],
      exclusion_conditions: ["N17"],  // Acute kidney injury
      lab_value_rules: [
        {
          lab_code: "CREATININE",
          loinc_code: "2160-0",
          operator: "gt",
          value: 1.2,
          unit: "mg/dL",
          required: false,
          weight: 0.7
        },
        {
          lab_code: "EGFR",
          loinc_code: "33914-3", 
          operator: "lt",
          value: 60,
          unit: "mL/min/1.73m2",
          required: false,
          weight: 0.9
        }
      ],
      medication_rules: [
        {
          medication_class: ["ace_inhibitor", "arb"],
          required: false,
          weight: 0.4
        }
      ],
      age_restrictions: {
        min_age: 18
      },
      time_windows: {
        lookback_days: 365,
        required_within: 90
      }
    },
    icd10_codes: ["N18", "N18.1", "N18.2", "N18.3", "N18.4", "N18.5", "N18.6"],
    snomed_codes: ["431855005", "46177005"],
    algorithm_type: "rule_based",
    algorithm: {
      type: "rule_based",
      rule_logic: "OR",
      thresholds: {
        minimum_score: 0.65,
        high_confidence: 0.8
      },
      parameters: {
        stage_thresholds: {
          "stage_3a": 45,
          "stage_3b": 30, 
          "stage_4": 15,
          "stage_5": 15
        }
      }
    },
    validation_data: {
      validation_dataset: "KIDNEY_DISEASE_REGISTRY_2024",
      ppv: 0.85,
      npv: 0.92,
      sensitivity: 0.82,
      specificity: 0.93,
      f1_score: 0.85,
      validated_at: new Date("2024-01-20"),
      validated_by: "nephrology_team"
    },
    created_at: new Date(),
    updated_at: new Date(),
    version: "1.1.0", 
    status: "active"
  }
]);

// Insert sample clinical contexts
db.clinical_contexts.insertMany([
  {
    patient_id: "PATIENT_001",
    context_type: "admission",
    context_id: "ADM_2024_001",
    clinical_indicators: {
      vital_signs: {
        systolic_bp: 158,
        diastolic_bp: 95,
        heart_rate: 78,
        temperature: 98.6,
        respiratory_rate: 16,
        oxygen_sat: 98,
        bmi: 28.5,
        weight: 185,
        height: 70
      },
      lab_values: [
        {
          test_code: "HBA1C",
          test_name: "Hemoglobin A1C", 
          value: 7.2,
          unit: "%",
          reference_range: "4.0-5.6",
          status: "abnormal",
          collected_at: new Date("2024-01-15"),
          loinc_code: "4548-4"
        },
        {
          test_code: "CREATININE",
          test_name: "Serum Creatinine",
          value: 1.1,
          unit: "mg/dL", 
          reference_range: "0.6-1.2",
          status: "normal",
          collected_at: new Date("2024-01-15"),
          loinc_code: "2160-0"
        }
      ],
      medications: [
        {
          medication_id: "MED_001",
          medication_name: "Metformin",
          rxnorm_code: "6809",
          dosage: "1000mg",
          frequency: "BID",
          start_date: new Date("2023-06-01"),
          status: "active",
          indication: "Type 2 Diabetes"
        },
        {
          medication_id: "MED_002",
          medication_name: "Lisinopril",
          rxnorm_code: "29046", 
          dosage: "10mg",
          frequency: "QD",
          start_date: new Date("2023-08-15"),
          status: "active",
          indication: "Hypertension"
        }
      ],
      condition_codes: [
        {
          code: "E11.9",
          code_system: "ICD10",
          description: "Type 2 diabetes mellitus without complications",
          severity: "moderate",
          status: "active",
          diagnosed_at: new Date("2023-05-20"),
          is_primary: true
        },
        {
          code: "I10",
          code_system: "ICD10", 
          description: "Essential hypertension",
          severity: "mild",
          status: "active",
          diagnosed_at: new Date("2023-08-10"),
          is_primary: false
        }
      ],
      procedures: [],
      allergies: [
        {
          allergen: "Penicillin",
          allergen_type: "drug",
          severity: "moderate",
          reaction: ["rash", "itching"],
          status: "active"
        }
      ],
      social_history: {
        smoking_status: "former_smoker",
        alcohol_use: "occasional",
        marital_status: "married",
        employment_status: "employed",
        insurance_type: "commercial"
      }
    },
    demographics: {
      age_range: {
        min: 55,
        max: 64,
        category: "adult"
      },
      gender: "male",
      ethnicity: "hispanic",
      geographic_region: "southwest_us"
    },
    risk_factors: [
      {
        factor_type: "cardiovascular",
        factor_name: "Diabetes mellitus",
        risk_score: 0.75,
        risk_category: "high",
        evidence: ["E11.9", "HBA1C=7.2"],
        assessed_at: new Date(),
        valid_until: new Date("2024-12-31")
      },
      {
        factor_type: "cardiovascular", 
        factor_name: "Hypertension",
        risk_score: 0.65,
        risk_category: "moderate", 
        evidence: ["I10", "BP=158/95"],
        assessed_at: new Date(),
        valid_until: new Date("2024-12-31")
      }
    ],
    phenotypes: [
      {
        phenotype_id: "diabetes_t2_phenotype",
        phenotype_name: "Type 2 Diabetes Mellitus Phenotype",
        match_score: 0.89,
        confidence: 0.92,
        matched_criteria: ["E11.9", "HBA1C>=6.5", "metformin"],
        evidence: [
          {
            evidence_type: "condition",
            description: "Type 2 diabetes diagnosis",
            value: "E11.9",
            timestamp: new Date("2023-05-20"),
            weight: 0.8,
            confidence: 0.95
          },
          {
            evidence_type: "lab",
            description: "Elevated HbA1c",
            value: "7.2%",
            timestamp: new Date("2024-01-15"),
            weight: 0.9,
            confidence: 0.9
          }
        ],
        matched_at: new Date(),
        algorithm_version: "1.2.0"
      },
      {
        phenotype_id: "hypertension_phenotype",
        phenotype_name: "Hypertension Phenotype", 
        match_score: 0.84,
        confidence: 0.88,
        matched_criteria: ["I10", "lisinopril", "BP_elevated"],
        evidence: [
          {
            evidence_type: "condition",
            description: "Essential hypertension diagnosis",
            value: "I10",
            timestamp: new Date("2023-08-10"),
            weight: 0.8,
            confidence: 0.9
          },
          {
            evidence_type: "medication",
            description: "ACE inhibitor therapy",
            value: "Lisinopril 10mg",
            timestamp: new Date("2023-08-15"),
            weight: 0.5,
            confidence: 0.85
          }
        ],
        matched_at: new Date(),
        algorithm_version: "2.0.1"
      }
    ],
    contextual_insights: [
      {
        insight_id: "INSIGHT_CVD_RISK_001",
        insight_type: "risk_alert",
        title: "Elevated Cardiovascular Risk",
        description: "Patient has multiple cardiovascular risk factors including diabetes and hypertension with suboptimal control",
        priority: "high",
        confidence_score: 0.87,
        evidence: [
          {
            evidence_type: "lab",
            description: "HbA1c above target (7.2%)",
            timestamp: new Date("2024-01-15"),
            weight: 0.8,
            confidence: 0.9
          },
          {
            evidence_type: "vital",
            description: "Blood pressure above target (158/95)", 
            timestamp: new Date(),
            weight: 0.7,
            confidence: 0.85
          }
        ],
        recommendations: [
          "Consider intensifying diabetes management",
          "Evaluate hypertension medication optimization",
          "Assess for cardiovascular disease screening"
        ],
        generated_at: new Date(),
        expires_at: new Date("2024-04-15"),
        status: "active"
      }
    ],
    created_at: new Date(),
    updated_at: new Date(),
    version: 1
  }
]);

// Insert sample population cohorts
db.population_cohorts.insertMany([
  {
    cohort_id: "DIABETES_COHORT_2024",
    name: "Adult Type 2 Diabetes Cohort",
    description: "Adults with diagnosed Type 2 Diabetes for population health management",
    criteria: {
      inclusion_criteria: [
        {
          rule_type: "phenotype",
          field: "phenotypes.phenotype_id",
          operator: "eq",
          value: "diabetes_t2_phenotype",
          weight: 1.0,
          required: true
        }
      ],
      exclusion_criteria: [
        {
          rule_type: "condition",
          field: "condition_codes.code", 
          operator: "eq",
          value: "E10",
          weight: 1.0,
          required: false
        }
      ],
      phenotypes: ["diabetes_t2_phenotype"],
      age_range: {
        min: 18,
        max: 85,
        category: "adult"
      },
      gender: ["male", "female"],
      geographic_regions: ["northeast_us", "southeast_us", "midwest_us", "southwest_us", "west_us"]
    },
    statistics: {
      member_count: 0,  // Will be updated by aggregation
      demographics: {
        age_groups: {},
        gender_split: {},
        ethnicity_split: {}
      },
      top_phenotypes: [],
      avg_risk_score: 0,
      last_computed: new Date()
    },
    created_at: new Date(),
    updated_at: new Date(),
    status: "active"
  },
  {
    cohort_id: "HIGH_CVD_RISK_COHORT",
    name: "High Cardiovascular Risk Cohort",
    description: "Patients with elevated cardiovascular disease risk for targeted interventions",
    criteria: {
      inclusion_criteria: [
        {
          rule_type: "risk_score",
          field: "risk_factors.risk_score",
          operator: "gte",
          value: 0.7,
          weight: 1.0,
          required: true
        }
      ],
      exclusion_criteria: [],
      phenotypes: ["diabetes_t2_phenotype", "hypertension_phenotype", "ckd_phenotype"],
      age_range: {
        min: 40,
        max: 85,
        category: "adult"
      }
    },
    statistics: {
      member_count: 0,
      demographics: {
        age_groups: {},
        gender_split: {},
        ethnicity_split: {}
      },
      top_phenotypes: [],
      avg_risk_score: 0,
      last_computed: new Date()
    },
    created_at: new Date(),
    updated_at: new Date(),
    status: "active"
  }
]);

// Insert sample patient profiles
db.patient_profiles.insertMany([
  {
    patient_id: "PATIENT_001",
    demographics: {
      age_range: {
        min: 55,
        max: 64,
        category: "adult"
      },
      gender: "male",
      ethnicity: "hispanic",
      geographic_region: "southwest_us"
    },
    phenotypes: [
      {
        phenotype_id: "diabetes_t2_phenotype",
        phenotype_name: "Type 2 Diabetes Mellitus Phenotype",
        match_score: 0.89,
        confidence: 0.92,
        matched_at: new Date(),
        algorithm_version: "1.2.0"
      },
      {
        phenotype_id: "hypertension_phenotype", 
        phenotype_name: "Hypertension Phenotype",
        match_score: 0.84,
        confidence: 0.88,
        matched_at: new Date(),
        algorithm_version: "2.0.1"
      }
    ],
    risk_profile: {
      overall_risk_score: 0.78,
      risk_category: "high",
      domain_risk_scores: {
        "cardiovascular": 0.82,
        "diabetes_complications": 0.75,
        "renal": 0.45
      },
      active_risk_factors: [
        {
          factor_type: "cardiovascular",
          factor_name: "Diabetes mellitus", 
          risk_score: 0.75,
          risk_category: "high"
        },
        {
          factor_type: "cardiovascular",
          factor_name: "Hypertension",
          risk_score: 0.65,
          risk_category: "moderate"
        }
      ],
      next_assessment_due: new Date("2024-07-15")
    },
    cohort_memberships: [
      {
        cohort_id: "DIABETES_COHORT_2024",
        cohort_name: "Adult Type 2 Diabetes Cohort",
        joined_at: new Date(),
        match_score: 0.95,
        status: "active"
      },
      {
        cohort_id: "HIGH_CVD_RISK_COHORT",
        cohort_name: "High Cardiovascular Risk Cohort", 
        joined_at: new Date(),
        match_score: 0.87,
        status: "active"
      }
    ],
    last_updated: new Date(),
    data_version: 1
  }
]);

// Create indexes for performance optimization
print("Creating performance indexes...");

// Clinical contexts indexes
db.clinical_contexts.createIndex({"patient_id": 1, "context_type": 1});
db.clinical_contexts.createIndex({"created_at": -1});
db.clinical_contexts.createIndex({"clinical_indicators.condition_codes.code": 1});
db.clinical_contexts.createIndex({"phenotypes.phenotype_id": 1});
db.clinical_contexts.createIndex({"demographics.age_range.min": 1, "demographics.age_range.max": 1});

// Phenotype definitions indexes  
db.phenotype_definitions.createIndex({"phenotype_id": 1}, {unique: true});
db.phenotype_definitions.createIndex({"category": 1, "status": 1});
db.phenotype_definitions.createIndex({"icd10_codes": 1});
db.phenotype_definitions.createIndex({"snomed_codes": 1});

// Patient profiles indexes
db.patient_profiles.createIndex({"patient_id": 1}, {unique: true});
db.patient_profiles.createIndex({"phenotypes.phenotype_id": 1});
db.patient_profiles.createIndex({"risk_profile.overall_risk_score": -1});
db.patient_profiles.createIndex({"cohort_memberships.cohort_id": 1});

// Population cohorts indexes
db.population_cohorts.createIndex({"cohort_id": 1}, {unique: true});
db.population_cohorts.createIndex({"criteria.phenotypes": 1});
db.population_cohorts.createIndex({"status": 1});

// Contextual insights indexes
db.contextual_insights.createIndex({"patient_id": 1, "insight_type": 1});
db.contextual_insights.createIndex({"priority": 1, "status": 1});
db.contextual_insights.createIndex({"generated_at": -1});
db.contextual_insights.createIndex({"expires_at": 1});

// Context cache with TTL index
db.context_cache.createIndex({"cache_key": 1}, {unique: true});
db.context_cache.createIndex({"expires_at": 1}, {expireAfterSeconds: 0});

print("Created performance indexes successfully");

// Aggregate pipeline example for cohort statistics
print("Computing initial cohort statistics...");

// Update diabetes cohort statistics
var diabetesCohortStats = db.clinical_contexts.aggregate([
  {
    $match: {
      "phenotypes.phenotype_id": "diabetes_t2_phenotype"
    }
  },
  {
    $group: {
      _id: null,
      member_count: { $sum: 1 },
      age_groups: {
        $push: "$demographics.age_range.category"
      },
      gender_split: {
        $push: "$demographics.gender"
      }
    }
  }
]).toArray();

if (diabetesCohortStats.length > 0) {
  var stats = diabetesCohortStats[0];
  db.population_cohorts.updateOne(
    { cohort_id: "DIABETES_COHORT_2024" },
    {
      $set: {
        "statistics.member_count": stats.member_count,
        "statistics.last_computed": new Date()
      }
    }
  );
}

print("KB-2 Clinical Context initialization completed successfully!");
print("Collections created: clinical_contexts, phenotype_definitions, patient_profiles, contextual_insights, population_cohorts");
print("Sample data inserted and indexes created");

// Validation queries
print("\n=== Validation Results ===");
print("Phenotype definitions:", db.phenotype_definitions.countDocuments());
print("Clinical contexts:", db.clinical_contexts.countDocuments()); 
print("Patient profiles:", db.patient_profiles.countDocuments());
print("Population cohorts:", db.population_cohorts.countDocuments());
print("Indexes created:", db.clinical_contexts.getIndexes().length + db.phenotype_definitions.getIndexes().length);