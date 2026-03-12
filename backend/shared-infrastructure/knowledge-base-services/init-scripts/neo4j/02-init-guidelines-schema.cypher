// KB-3 Guidelines Neo4j Schema Initialization
// Clinical guidelines knowledge graph setup

// Create constraints for unique identifiers
CREATE CONSTRAINT guideline_id_unique IF NOT EXISTS FOR (g:Guideline) REQUIRE g.id IS UNIQUE;
CREATE CONSTRAINT recommendation_id_unique IF NOT EXISTS FOR (r:Recommendation) REQUIRE r.id IS UNIQUE;
CREATE CONSTRAINT condition_id_unique IF NOT EXISTS FOR (c:Condition) REQUIRE c.id IS UNIQUE;
CREATE CONSTRAINT evidence_id_unique IF NOT EXISTS FOR (e:Evidence) REQUIRE e.id IS UNIQUE;

// Create indexes for performance
CREATE INDEX guideline_title_index IF NOT EXISTS FOR (g:Guideline) ON (g.title);
CREATE INDEX guideline_organization_index IF NOT EXISTS FOR (g:Guideline) ON (g.organization);
CREATE INDEX recommendation_grade_index IF NOT EXISTS FOR (r:Recommendation) ON (r.grade);
CREATE INDEX recommendation_priority_index IF NOT EXISTS FOR (r:Recommendation) ON (r.priority);
CREATE INDEX condition_name_index IF NOT EXISTS FOR (c:Condition) ON (c.name);
CREATE INDEX condition_icd10_index IF NOT EXISTS FOR (c:Condition) ON (c.icd10_codes);

// Create full-text search indexes
CREATE FULLTEXT INDEX guidelineFulltextIndex IF NOT EXISTS 
FOR (g:Guideline) 
ON EACH [g.title, g.description, g.keywords];

CREATE FULLTEXT INDEX recommendationFulltextIndex IF NOT EXISTS 
FOR (r:Recommendation) 
ON EACH [r.text, r.summary, r.keywords];

CREATE FULLTEXT INDEX conditionFulltextIndex IF NOT EXISTS 
FOR (c:Condition) 
ON EACH [c.name, c.synonyms, c.description];

// Insert sample clinical guidelines data

// Create major guideline organizations
CREATE (who:Organization {
  id: "who",
  name: "World Health Organization",
  acronym: "WHO",
  country: "International",
  type: "International Health Organization"
});

CREATE (aha:Organization {
  id: "aha", 
  name: "American Heart Association",
  acronym: "AHA",
  country: "US",
  type: "Professional Medical Association"
});

CREATE (esc:Organization {
  id: "esc",
  name: "European Society of Cardiology", 
  acronym: "ESC",
  country: "EU",
  type: "Professional Medical Association"
});

CREATE (nice:Organization {
  id: "nice",
  name: "National Institute for Health and Care Excellence",
  acronym: "NICE", 
  country: "UK",
  type: "National Health Institute"
});

// Create sample conditions
CREATE (htn:Condition {
  id: "hypertension",
  name: "Hypertension",
  icd10_codes: ["I10", "I11", "I12", "I13", "I15"],
  snomed_codes: ["38341003", "59621000"],
  synonyms: ["High blood pressure", "Arterial hypertension"],
  description: "Persistently elevated blood pressure in the systemic arteries"
});

CREATE (dm2:Condition {
  id: "diabetes_mellitus_type2", 
  name: "Type 2 Diabetes Mellitus",
  icd10_codes: ["E11"],
  snomed_codes: ["44054006"],
  synonyms: ["Type 2 diabetes", "Adult-onset diabetes", "Non-insulin dependent diabetes"],
  description: "A metabolic disorder characterized by insulin resistance and relative insulin deficiency"
});

CREATE (cad:Condition {
  id: "coronary_artery_disease",
  name: "Coronary Artery Disease", 
  icd10_codes: ["I25"],
  snomed_codes: ["414545008"],
  synonyms: ["CAD", "Ischemic heart disease", "Coronary heart disease"],
  description: "Disease of the coronary arteries that supply blood to the heart muscle"
});

CREATE (hf:Condition {
  id: "heart_failure",
  name: "Heart Failure",
  icd10_codes: ["I50"],
  snomed_codes: ["84114007"],
  synonyms: ["Congestive heart failure", "CHF", "Cardiac failure"],
  description: "A complex syndrome resulting from impaired ability of the heart to pump blood"
});

// Create sample guidelines
CREATE (aha_htn_guideline:Guideline {
  id: "aha_acc_htn_2017",
  title: "2017 AHA/ACC/AAPA/ABC/ACPM/AGS/APhA/ASH/ASPC/NMA/PCNA Guideline for the Prevention, Detection, Evaluation, and Management of High Blood Pressure in Adults",
  organization: "AHA/ACC", 
  publication_date: "2017-11-13",
  version: "2017.1",
  doi: "10.1161/HYP.0000000000000065",
  status: "active",
  evidence_level: "high",
  description: "Comprehensive guideline for hypertension management in adults",
  keywords: ["hypertension", "blood pressure", "cardiovascular risk"]
});

CREATE (esc_hf_guideline:Guideline {
  id: "esc_hf_2021", 
  title: "2021 ESC Guidelines for the diagnosis and treatment of acute and chronic heart failure",
  organization: "ESC",
  publication_date: "2021-08-27", 
  version: "2021.1",
  doi: "10.1093/eurheartj/ehab368",
  status: "active",
  evidence_level: "high",
  description: "Evidence-based recommendations for heart failure diagnosis and treatment",
  keywords: ["heart failure", "cardiac dysfunction", "treatment"]
});

// Create sample recommendations with evidence grades
CREATE (htn_bp_target:Recommendation {
  id: "aha_htn_bp_target_2017",
  text: "For most adults with confirmed high BP, the recommended treatment goal is <130/80 mmHg.",
  summary: "BP target <130/80 mmHg for most adults",
  grade: "A", 
  level_of_evidence: "1",
  priority: 100,
  strength: "strong",
  keywords: ["blood pressure target", "treatment goal"],
  clinical_context: ["general_adult_population", "confirmed_hypertension"]
});

CREATE (htn_lifestyle:Recommendation {
  id: "aha_htn_lifestyle_2017",
  text: "Implement lifestyle modifications including weight management, DASH diet, sodium restriction, potassium supplementation, physical activity, and limited alcohol intake.",
  summary: "Comprehensive lifestyle modifications for BP reduction", 
  grade: "A",
  level_of_evidence: "1", 
  priority: 95,
  strength: "strong",
  keywords: ["lifestyle", "diet", "exercise", "weight management"],
  clinical_context: ["all_hypertensive_patients"]
});

CREATE (hf_acei_arb:Recommendation {
  id: "esc_hf_acei_arb_2021",
  text: "ACE inhibitors (or ARBs if ACE inhibitor intolerant) are recommended for all patients with HFrEF to reduce morbidity and mortality.",
  summary: "ACE inhibitors/ARBs for HFrEF patients",
  grade: "A",
  level_of_evidence: "1",
  priority: 100, 
  strength: "strong",
  keywords: ["ACE inhibitor", "ARB", "heart failure", "reduced ejection fraction"],
  clinical_context: ["heart_failure_reduced_ef"]
});

CREATE (hf_beta_blocker:Recommendation {
  id: "esc_hf_beta_blocker_2021", 
  text: "Beta-blockers are recommended for all stable patients with HFrEF to reduce morbidity and mortality.",
  summary: "Beta-blockers for stable HFrEF patients",
  grade: "A",
  level_of_evidence: "1",
  priority: 95,
  strength: "strong", 
  keywords: ["beta blocker", "heart failure", "mortality reduction"],
  clinical_context: ["heart_failure_reduced_ef", "stable_patients"]
});

// Create evidence nodes
CREATE (meta_analysis_bp:Evidence {
  id: "bp_meta_analysis_2015",
  description: "Meta-analysis of blood pressure lowering trials showing cardiovascular benefit",
  study_type: "Meta-analysis",
  quality: "High",
  citation: "Ettehad D, et al. Blood pressure lowering for prevention of cardiovascular disease and death. BMJ. 2016;354:i4098",
  doi: "10.1136/bmj.i4098",
  year: 2016,
  sample_size: 613815,
  study_duration_months: 48
});

CREATE (rct_acei_hf:Evidence {
  id: "consensus_trial_1987",
  description: "CONSENSUS Trial: Effects of enalapril on mortality in severe congestive heart failure",
  study_type: "RCT",
  quality: "High", 
  citation: "CONSENSUS Trial Study Group. Effects of enalapril on mortality in severe congestive heart failure. N Engl J Med. 1987;316(23):1429-35",
  doi: "10.1056/NEJM198706043162301",
  year: 1987,
  sample_size: 253,
  study_duration_months: 6
});

// Create relationships
// Guideline contains recommendations
CREATE (aha_htn_guideline)-[:CONTAINS]->(htn_bp_target);
CREATE (aha_htn_guideline)-[:CONTAINS]->(htn_lifestyle);
CREATE (esc_hf_guideline)-[:CONTAINS]->(hf_acei_arb);
CREATE (esc_hf_guideline)-[:CONTAINS]->(hf_beta_blocker);

// Recommendations apply to conditions
CREATE (htn_bp_target)-[:APPLIES_TO]->(htn);
CREATE (htn_lifestyle)-[:APPLIES_TO]->(htn);
CREATE (hf_acei_arb)-[:APPLIES_TO]->(hf);
CREATE (hf_beta_blocker)-[:APPLIES_TO]->(hf);

// Recommendations have supporting evidence
CREATE (htn_bp_target)-[:HAS_EVIDENCE]->(meta_analysis_bp);
CREATE (htn_lifestyle)-[:HAS_EVIDENCE]->(meta_analysis_bp);
CREATE (hf_acei_arb)-[:HAS_EVIDENCE]->(rct_acei_hf);

// Cross-references between recommendations
CREATE (htn_bp_target)-[:SUPPORTS]->(htn_lifestyle);
CREATE (hf_acei_arb)-[:COMPLEMENTS]->(hf_beta_blocker);

// Organizations publish guidelines
CREATE (aha)-[:PUBLISHES]->(aha_htn_guideline);
CREATE (esc)-[:PUBLISHES]->(esc_hf_guideline);

// Create procedure for updating recommendation priorities
CREATE OR REPLACE PROCEDURE updateRecommendationPriorities()
LANGUAGE cypher AS
$$
  MATCH (r:Recommendation)
  SET r.calculated_priority = 
    CASE r.grade
      WHEN 'A' THEN 100
      WHEN 'B' THEN 80  
      WHEN 'C' THEN 60
      WHEN 'D' THEN 40
      ELSE 20
    END +
    CASE r.level_of_evidence
      WHEN '1' THEN 20
      WHEN '2a' THEN 15
      WHEN '2b' THEN 10
      WHEN '3' THEN 5
      ELSE 0
    END
$$;

// Create procedure for conflict detection
CREATE OR REPLACE PROCEDURE detectRecommendationConflicts()
LANGUAGE cypher AS
$$
  MATCH (r1:Recommendation)-[:APPLIES_TO]->(c:Condition)<-[:APPLIES_TO]-(r2:Recommendation)
  WHERE r1 <> r2 
    AND r1.grade <> r2.grade
    AND (
      (r1.grade IN ['A', 'B'] AND r2.grade IN ['C', 'D']) OR
      (r1.grade IN ['C', 'D'] AND r2.grade IN ['A', 'B'])
    )
  MERGE (conflict:Conflict {
    id: r1.id + '_' + r2.id,
    recommendation_a: r1.id,
    recommendation_b: r2.id,
    condition: c.name,
    conflict_type: 'GRADE_INCONSISTENCY',
    severity: CASE 
      WHEN (r1.grade = 'A' AND r2.grade = 'D') OR (r1.grade = 'D' AND r2.grade = 'A') THEN 'HIGH'
      ELSE 'MEDIUM'
    END,
    detected_at: datetime(),
    status: 'ACTIVE'
  })
$$;

// Initial data quality checks
CALL updateRecommendationPriorities();
CALL detectRecommendationConflicts();

// Create sample regional variations
CREATE (us_htn_variation:GuidelineVariation {
  id: "us_htn_variation",
  guideline_id: "aha_acc_htn_2017", 
  region: "US",
  variation_type: "IMPLEMENTATION",
  description: "US-specific implementation guidance for hypertension management",
  modifications: ["Insurance coverage considerations", "Healthcare system integration"]
});

CREATE (eu_htn_variation:GuidelineVariation {
  id: "eu_htn_variation",
  guideline_id: "aha_acc_htn_2017",
  region: "EU", 
  variation_type: "ADAPTATION",
  description: "European adaptation considering ESC guidelines alignment", 
  modifications: ["Drug availability differences", "Cost-effectiveness considerations"]
});

// Final statistics query
MATCH (g:Guideline)
OPTIONAL MATCH (g)-[:CONTAINS]->(r:Recommendation)
OPTIONAL MATCH (r)-[:APPLIES_TO]->(c:Condition)
OPTIONAL MATCH (r)-[:HAS_EVIDENCE]->(e:Evidence)
RETURN 
  count(DISTINCT g) as total_guidelines,
  count(DISTINCT r) as total_recommendations, 
  count(DISTINCT c) as total_conditions,
  count(DISTINCT e) as total_evidence,
  'KB-3 Guidelines Neo4j initialization complete' as status;