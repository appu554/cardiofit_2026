package graph

const (
	CypherGetCausalChains = `
		MATCH path = (source)-[rels*1..4]->(target:ClinVar {code: $target_code})
		WHERE (source:Food OR source:Exercise) AND source.code = $source_code
		RETURN path, [r IN rels | type(r)] AS edge_types,
		       [r IN rels | r.effect_size] AS effect_sizes,
		       [r IN rels | r.evidence_grade] AS grades
		ORDER BY length(path)
		LIMIT 10
	`

	CypherGetAllChainsToTarget = `
		MATCH path = (source)-[rels*1..4]->(target:ClinVar {code: $target_code})
		WHERE source:Food OR source:Exercise
		RETURN source.code AS source_code, labels(source)[0] AS source_type,
		       path, [r IN rels | type(r)] AS edge_types,
		       [r IN rels | r.effect_size] AS effect_sizes,
		       [r IN rels | r.evidence_grade] AS grades
		ORDER BY length(path)
		LIMIT 100
	`

	CypherSearchFoods = `
		MATCH (f:Food)
		WHERE ($name = '' OR toLower(f.name) CONTAINS toLower($name))
		  AND ($region = '' OR f.region = $region OR f.region = 'ALL')
		  AND ($diet_type = '' OR f.diet_type = $diet_type)
		RETURN f
		ORDER BY f.name
		LIMIT $limit
	`

	CypherGetSafetyRules = `
		MATCH (source {code: $code})-[r:CONTRAINDICATED_FOR]->(ctx:PatientCtx)
		RETURN r.rule_code AS rule_code, r.condition AS condition,
		       r.severity AS severity, r.description AS description,
		       ctx.code AS context_code
	`

	CypherGetDrugInteractions = `
		MATCH (source {code: $lifestyle_code})-[r:INTERACTS_WITH]->(d:DrugClass)
		WHERE d.code IN $drug_codes
		RETURN d.code AS drug_class_code, r.interaction AS interaction,
		       r.severity AS severity, r.action AS action,
		       r.description AS description
	`

	CypherGetFoodByCode = `
		MATCH (f:Food {code: $code})
		RETURN f
	`

	CypherGetExerciseByCode = `
		MATCH (e:Exercise {code: $code})
		RETURN e
	`
)
