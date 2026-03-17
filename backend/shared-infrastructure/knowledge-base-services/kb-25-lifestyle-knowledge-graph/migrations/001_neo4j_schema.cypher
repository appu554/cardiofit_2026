// KB-25 Lifestyle Knowledge Graph — Neo4j Schema (spec §7.1)
// Constraints
CREATE CONSTRAINT food_code IF NOT EXISTS FOR (f:Food) REQUIRE f.code IS UNIQUE;
CREATE CONSTRAINT exercise_code IF NOT EXISTS FOR (e:Exercise) REQUIRE e.code IS UNIQUE;
CREATE CONSTRAINT nutrient_code IF NOT EXISTS FOR (n:Nutrient) REQUIRE n.code IS UNIQUE;
CREATE CONSTRAINT physprocess_code IF NOT EXISTS FOR (p:PhysProcess) REQUIRE p.code IS UNIQUE;
CREATE CONSTRAINT clinvar_code IF NOT EXISTS FOR (c:ClinVar) REQUIRE c.code IS UNIQUE;
CREATE CONSTRAINT drugclass_code IF NOT EXISTS FOR (d:DrugClass) REQUIRE d.code IS UNIQUE;
CREATE CONSTRAINT patientctx_code IF NOT EXISTS FOR (p:PatientCtx) REQUIRE p.code IS UNIQUE;

// Indexes for traversal performance
CREATE INDEX food_region IF NOT EXISTS FOR (f:Food) ON (f.region);
CREATE INDEX food_diet_type IF NOT EXISTS FOR (f:Food) ON (f.diet_type);
CREATE INDEX exercise_safety_tier IF NOT EXISTS FOR (e:Exercise) ON (e.safety_tier);
CREATE INDEX clinvar_kb20_field IF NOT EXISTS FOR (c:ClinVar) ON (c.kb20_field);
