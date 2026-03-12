// Create constraints for data integrity
CREATE CONSTRAINT guideline_id IF NOT EXISTS 
    FOR (g:Guideline) REQUIRE g.id IS UNIQUE;

CREATE CONSTRAINT recommendation_id IF NOT EXISTS 
    FOR (r:Recommendation) REQUIRE r.id IS UNIQUE;

CREATE CONSTRAINT evidence_id IF NOT EXISTS 
    FOR (e:Evidence) REQUIRE e.id IS UNIQUE;

CREATE CONSTRAINT condition_id IF NOT EXISTS 
    FOR (c:Condition) REQUIRE c.id IS UNIQUE;

// Create indexes for performance
CREATE INDEX guideline_condition IF NOT EXISTS 
    FOR (g:Guideline) ON (g.condition);

CREATE INDEX recommendation_grade IF NOT EXISTS 
    FOR (r:Recommendation) ON (r.grade);

CREATE INDEX evidence_level IF NOT EXISTS 
    FOR (e:Evidence) ON (e.level);

// Load sample ACC/AHA Hypertension Guideline
MERGE (g:Guideline {
    id: 'ACC_AHA_HTN_2017',
    title: '2017 ACC/AHA Guideline for High Blood Pressure',
    publisher: 'ACC/AHA',
    publication_date: date('2017-11-13'),
    condition: 'Hypertension',
    version: '2017.1',
    status: 'active'
})

MERGE (htn:Condition {
    id: 'CONDITION_HTN',
    name: 'Hypertension',
    icd10: 'I10',
    snomed: '38341003'
})

MERGE (stage2:Condition {
    id: 'CONDITION_HTN_STAGE2',
    name: 'Stage 2 Hypertension',
    parent: 'CONDITION_HTN',
    criteria: 'BP ≥140/90 mmHg'
})

MERGE (ckd:Condition {
    id: 'CONDITION_CKD',
    name: 'Chronic Kidney Disease',
    icd10: 'N18',
    snomed: '709044004'
})

// Create recommendations
MERGE (r1:Recommendation {
    id: 'HTN_REC_001',
    text: 'Initiate antihypertensive therapy for Stage 2 HTN',
    grade: 'I',
    level_of_evidence: 'A',
    priority: 1
})

MERGE (r2:Recommendation {
    id: 'HTN_REC_002',
    text: 'Use ACEi/ARB as first-line for HTN with CKD',
    grade: 'I',
    level_of_evidence: 'B',
    priority: 1
})

// Create evidence
MERGE (e1:Evidence {
    id: 'EVIDENCE_001',
    study_type: 'RCT',
    pmid: '28146533',
    summary: 'SPRINT trial demonstrates benefit of intensive BP control',
    quality_score: 0.95
})

// Create relationships
MERGE (g)-[:CONTAINS]->(r1)
MERGE (g)-[:CONTAINS]->(r2)
MERGE (r1)-[:APPLIES_TO]->(stage2)
MERGE (r2)-[:APPLIES_TO]->(htn)
MERGE (r2)-[:APPLIES_TO]->(ckd)
MERGE (r1)-[:SUPPORTED_BY]->(e1)
MERGE (r1)-[:FOLLOWED_BY {condition: 'if_ckd_present'}]->(r2)

// Create clinical pathway
CREATE (pathway:ClinicalPathway {
    id: 'HTN_PATHWAY_001',
    name: 'Hypertension Management Pathway',
    created_at: datetime(),
    version: '1.0.0'
})

MERGE (pathway)-[:STARTS_WITH]->(r1)
MERGE (pathway)-[:INCLUDES]->(r2);