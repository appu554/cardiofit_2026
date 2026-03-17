// Causal edges with EffectDescriptors (spec §2.2.1)

// Brisk Walking → AMPK Activation → Insulin Sensitivity → FBG
MATCH (e:Exercise {code: 'EX001'}), (p:PhysProcess {code: 'AMPK_ACTIVATION'})
CREATE (e)-[:STIMULATES {effect_size: 0.35, effect_unit: 'fold_increase', evidence_grade: 'A', onset_days: 7, peak_effect_days: 42, steady_state_days: 90, source_pmids: ['PMID:28246350']}]->(p);

MATCH (p:PhysProcess {code: 'AMPK_ACTIVATION'}), (c:ClinVar {code: 'IS'})
CREATE (p)-[:IMPROVES {effect_size: 0.20, effect_unit: 'index_change', evidence_grade: 'A', onset_days: 14, peak_effect_days: 56, steady_state_days: 120, source_pmids: ['PMID:28246350']}]->(c);

MATCH (c1:ClinVar {code: 'IS'}), (c2:ClinVar {code: 'FBG'})
CREATE (c1)-[:REDUCES {effect_size: -12.0, effect_unit: 'mg/dL', evidence_grade: 'A', onset_days: 14, peak_effect_days: 60, steady_state_days: 90, source_pmids: ['PMID:28246350']}]->(c2);

// Ragi → GLP-1 Secretion → PPBG
MATCH (f:Food {code: 'F002'}), (p:PhysProcess {code: 'GLP1_SECRETION'})
CREATE (f)-[:STIMULATES {effect_size: 0.25, effect_unit: 'fold_increase', evidence_grade: 'B', onset_days: 1, peak_effect_days: 1, steady_state_days: 7, source_pmids: ['PMID:23022602']}]->(p);

MATCH (p:PhysProcess {code: 'GLP1_SECRETION'}), (c:ClinVar {code: 'PPBG'})
CREATE (p)-[:REDUCES {effect_size: -18.0, effect_unit: 'mg/dL', evidence_grade: 'B', onset_days: 1, peak_effect_days: 7, steady_state_days: 30, source_pmids: ['PMID:23022602']}]->(c);

// Bitter Gourd → Peripheral Glucose Uptake → FBG
MATCH (f:Food {code: 'F006'}), (p:PhysProcess {code: 'PERIPHERAL_GLUCOSE_UPTAKE'})
CREATE (f)-[:STIMULATES {effect_size: 0.15, effect_unit: 'fold_increase', evidence_grade: 'B', onset_days: 7, peak_effect_days: 30, steady_state_days: 60, source_pmids: ['PMID:21425411']}]->(p);

MATCH (p:PhysProcess {code: 'PERIPHERAL_GLUCOSE_UPTAKE'}), (c:ClinVar {code: 'FBG'})
CREATE (p)-[:REDUCES {effect_size: -15.0, effect_unit: 'mg/dL', evidence_grade: 'B', onset_days: 14, peak_effect_days: 42, steady_state_days: 60, source_pmids: ['PMID:21425411']}]->(c);

// Resistance Training → Muscle Protein Synthesis → Muscle Mass → Insulin Sensitivity
MATCH (e:Exercise {code: 'EX007'}), (p:PhysProcess {code: 'MUSCLE_PROTEIN_SYNTHESIS'})
CREATE (e)-[:STIMULATES {effect_size: 0.40, effect_unit: 'fold_increase', evidence_grade: 'A', onset_days: 1, peak_effect_days: 28, steady_state_days: 90, source_pmids: ['PMID:29929465']}]->(p);

MATCH (p:PhysProcess {code: 'MUSCLE_PROTEIN_SYNTHESIS'}), (c:ClinVar {code: 'MM'})
CREATE (p)-[:IMPROVES {effect_size: 0.10, effect_unit: 'index_change', evidence_grade: 'A', onset_days: 28, peak_effect_days: 90, steady_state_days: 180, source_pmids: ['PMID:29929465']}]->(c);

MATCH (c1:ClinVar {code: 'MM'}), (c2:ClinVar {code: 'IS'})
CREATE (c1)-[:IMPROVES {effect_size: 0.08, effect_unit: 'index_change', evidence_grade: 'A', onset_days: 30, peak_effect_days: 90, steady_state_days: 180, source_pmids: ['PMID:29929465']}]->(c2);

// Flaxseed → Endothelial Function → SBP
MATCH (f:Food {code: 'F010'}), (p:PhysProcess {code: 'ENDOTHELIAL_FUNCTION'})
CREATE (f)-[:IMPROVES {effect_size: 0.12, effect_unit: 'fold_increase', evidence_grade: 'A', onset_days: 14, peak_effect_days: 60, steady_state_days: 180, source_pmids: ['PMID:24126178']}]->(p);

MATCH (p:PhysProcess {code: 'ENDOTHELIAL_FUNCTION'}), (c:ClinVar {code: 'SBP'})
CREATE (p)-[:REDUCES {effect_size: -7.0, effect_unit: 'mmHg', evidence_grade: 'A', onset_days: 30, peak_effect_days: 90, steady_state_days: 180, source_pmids: ['PMID:24126178']}]->(c);
