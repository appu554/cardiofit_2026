// PatientCtx nodes for safety rules
CREATE (:PatientCtx {code: 'CTX_CKD45', name: 'CKD Stage 4-5', condition: 'eGFR < 30'});
CREATE (:PatientCtx {code: 'CTX_HYPERTENSIVE_CRISIS', name: 'Hypertensive Crisis', condition: 'SBP > 180 OR DBP > 110'});
CREATE (:PatientCtx {code: 'CTX_RECENT_HYPO', name: 'Recent Hypoglycemia', condition: 'FBG < 70 in last 7d'});
CREATE (:PatientCtx {code: 'CTX_SU_INSULIN', name: 'SU or Insulin User', condition: 'current_meds includes SU or insulin'});
CREATE (:PatientCtx {code: 'CTX_SGLT2I', name: 'SGLT2i User', condition: 'current_meds includes SGLT2i'});
CREATE (:PatientCtx {code: 'CTX_RETINOPATHY', name: 'Proliferative Retinopathy', condition: 'retinopathy = PROLIFERATIVE'});
CREATE (:PatientCtx {code: 'CTX_NEUROPATHY', name: 'Peripheral Neuropathy', condition: 'neuropathy = true'});
CREATE (:PatientCtx {code: 'CTX_PREGNANCY', name: 'Pregnancy + T2DM/GDM', condition: 'pregnant = true AND diabetes'});
CREATE (:PatientCtx {code: 'CTX_HYPERKALEMIA', name: 'Hyperkalemia', condition: 'potassium > 5.5'});
CREATE (:PatientCtx {code: 'CTX_RECENT_CARDIAC', name: 'Recent Cardiac Event', condition: 'cardiac_event within 30d'});
CREATE (:PatientCtx {code: 'CTX_HBA1C_EXTREME', name: 'HbA1c > 13%', condition: 'HbA1c > 13'});
CREATE (:PatientCtx {code: 'CTX_LOW_BMR', name: 'Low BMR', condition: 'BMR < 1200 kcal'});
CREATE (:PatientCtx {code: 'CTX_GASTROPARESIS', name: 'Gastroparesis', condition: 'gastroparesis = true'});
CREATE (:PatientCtx {code: 'CTX_EATING_DISORDER', name: 'Eating Disorder History', condition: 'eating_disorder_history = true'});

// Safety rules as CONTRAINDICATED_FOR edges
MATCH (f:Food), (ctx:PatientCtx {code: 'CTX_CKD45'})
WHERE f.code IN ['F020', 'F008']
CREATE (f)-[:CONTRAINDICATED_FOR {rule_code: 'LS-01', severity: 'HARD_STOP', description: 'Protein > 0.6 g/kg/day blocked when eGFR < 30'}]->(ctx);

MATCH (e:Exercise), (ctx:PatientCtx {code: 'CTX_HYPERTENSIVE_CRISIS'})
WHERE e.met_value > 6
CREATE (e)-[:CONTRAINDICATED_FOR {rule_code: 'LS-02', severity: 'HARD_STOP', description: 'Vigorous exercise (MET > 6) blocked when SBP > 180 or DBP > 110'}]->(ctx);

MATCH (e:Exercise), (ctx:PatientCtx {code: 'CTX_RETINOPATHY'})
WHERE e.category = 'RESISTANCE'
CREATE (e)-[:CONTRAINDICATED_FOR {rule_code: 'LS-06', severity: 'HARD_STOP', description: 'Resistance training blocked with proliferative retinopathy'}]->(ctx);

MATCH (e:Exercise), (ctx:PatientCtx {code: 'CTX_RECENT_CARDIAC'})
CREATE (e)-[:CONTRAINDICATED_FOR {rule_code: 'LS-10', severity: 'HARD_STOP', description: 'All exercise blocked within 30d of cardiac event'}]->(ctx);

MATCH (f:Food), (ctx:PatientCtx {code: 'CTX_HYPERKALEMIA'})
WHERE f.potassium_mg > 400
CREATE (f)-[:CONTRAINDICATED_FOR {rule_code: 'LS-09', severity: 'HARD_STOP', description: 'High-potassium foods blocked when K+ > 5.5'}]->(ctx);

// LS-15: Underweight (South Asian BMI < 22) — VFRP blocked
CREATE (ctx_underweight:PatientCtx {code: 'CTX_UNDERWEIGHT_SA', name: 'South Asian Underweight', condition: 'BMI < 22', description: 'BMI below South Asian underweight threshold'})
WITH ctx_underweight
MATCH (vfrp_exercise:Exercise) WHERE vfrp_exercise.code STARTS WITH 'EX_'
CREATE (vfrp_exercise)-[:CONTRAINDICATED_FOR {rule_code: 'LS-15', severity: 'HARD_STOP', description: 'Underweight patients: caloric deficit protocols blocked'}]->(ctx_underweight);
