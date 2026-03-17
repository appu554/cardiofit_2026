// Lifestyle-Drug Interaction edges (spec §8)
MATCH (f:Food {code: 'F002'}), (d:DrugClass {code: 'DC_METFORMIN'})
CREATE (f)-[:INTERACTS_WITH {interaction: 'High fiber delays metformin absorption', severity: 'LOW', action: 'MONITOR', description: 'Space fiber-rich meals 1h from metformin dose'}]->(d);

MATCH (f:Food {code: 'F006'}), (d:DrugClass {code: 'DC_SU'})
CREATE (f)-[:INTERACTS_WITH {interaction: 'Additive glucose-lowering', severity: 'MODERATE', action: 'MODIFY', description: 'Bitter gourd adds ~15mg/dL FBG reduction — monitor for hypoglycemia with SU'}]->(d);

MATCH (f:Food {code: 'F006'}), (d:DrugClass {code: 'DC_INSULIN'})
CREATE (f)-[:INTERACTS_WITH {interaction: 'Additive glucose-lowering', severity: 'MODERATE', action: 'MODIFY', description: 'Bitter gourd adds ~15mg/dL FBG reduction — monitor for hypoglycemia with insulin'}]->(d);

MATCH (e:Exercise {code: 'EX011'}), (d:DrugClass {code: 'DC_SU'})
CREATE (e)-[:INTERACTS_WITH {interaction: 'Exercise-induced hypoglycemia', severity: 'HIGH', action: 'MODIFY', description: 'HIIT with SU requires pre-exercise carb snack and dose timing adjustment'}]->(d);

MATCH (e:Exercise {code: 'EX011'}), (d:DrugClass {code: 'DC_INSULIN'})
CREATE (e)-[:INTERACTS_WITH {interaction: 'Exercise-induced hypoglycemia', severity: 'HIGH', action: 'MODIFY', description: 'HIIT with insulin requires pre-exercise carb and possible dose reduction'}]->(d);

MATCH (f:Food), (d:DrugClass {code: 'DC_ACEI'})
WHERE f.potassium_mg > 400
CREATE (f)-[:INTERACTS_WITH {interaction: 'Hyperkalemia risk', severity: 'MODERATE', action: 'MONITOR', description: 'High-K foods with ACEi may elevate serum potassium — monitor K+ levels'}]->(d);

MATCH (f:Food), (d:DrugClass {code: 'DC_MRA'})
WHERE f.potassium_mg > 400
CREATE (f)-[:INTERACTS_WITH {interaction: 'Hyperkalemia risk', severity: 'HIGH', action: 'AVOID', description: 'High-K foods with MRA — significant hyperkalemia risk, restrict intake'}]->(d);

MATCH (f:Food {code: 'F010'}), (d:DrugClass {code: 'DC_ANTICOAG'})
CREATE (f)-[:INTERACTS_WITH {interaction: 'Omega-3 antiplatelet effect', severity: 'MODERATE', action: 'MONITOR', description: 'Flaxseed omega-3 may potentiate anticoagulant effect — monitor INR'}]->(d);

MATCH (e:Exercise), (d:DrugClass {code: 'DC_SGLT2I'})
WHERE e.met_value > 6
CREATE (e)-[:INTERACTS_WITH {interaction: 'Dehydration and euglycemic DKA risk', severity: 'HIGH', action: 'MODIFY', description: 'Vigorous exercise + SGLT2i requires extra hydration and carb intake'}]->(d);
