// KB-3 Neo4j Graph Schema for Guideline Relationships
// Creates comprehensive graph structure for guideline management with conflict resolution

// =============================================================================
// CONSTRAINTS AND INDEXES
// =============================================================================

// Unique constraints for core entities
CREATE CONSTRAINT guideline_unique IF NOT EXISTS
FOR (g:Guideline) REQUIRE g.guideline_id IS UNIQUE;

CREATE CONSTRAINT recommendation_unique IF NOT EXISTS
FOR (r:Recommendation) REQUIRE r.rec_id IS UNIQUE;

CREATE CONSTRAINT conflict_unique IF NOT EXISTS
FOR (c:Conflict) REQUIRE c.conflict_id IS UNIQUE;

CREATE CONSTRAINT safety_override_unique IF NOT EXISTS
FOR (s:SafetyOverride) REQUIRE s.override_id IS UNIQUE;

// Performance indexes
CREATE INDEX guideline_region IF NOT EXISTS
FOR (g:Guideline) ON (g.region);

CREATE INDEX guideline_condition IF NOT EXISTS
FOR (g:Guideline) ON (g.condition_primary);

CREATE INDEX guideline_status IF NOT EXISTS
FOR (g:Guideline) ON (g.status);

CREATE INDEX recommendation_domain IF NOT EXISTS
FOR (r:Recommendation) ON (r.domain);

CREATE INDEX recommendation_grade IF NOT EXISTS
FOR (r:Recommendation) ON (r.evidence_grade);

CREATE INDEX recommendation_population IF NOT EXISTS
FOR (r:Recommendation) ON (r.population_criteria);

// =============================================================================
// NODE DEFINITIONS WITH PROPERTIES
// =============================================================================

// Guidelines with comprehensive metadata
// (:Guideline {
//   guideline_id: String,           // Unique identifier (e.g., "ACC-AHA-HTN-2017")
//   organization: String,           // Publishing organization
//   title: String,                  // Full guideline title
//   region: String,                 // Geographic region (US, EU, WHO, etc.)
//   version: String,                // Semantic version
//   condition_primary: String,      // Primary condition addressed
//   icd10_codes: [String],         // Array of ICD-10 codes
//   effective_date: Date,           // When guideline becomes effective
//   superseded_date: Date,          // When guideline is superseded (nullable)
//   status: String,                 // active, superseded, draft, withdrawn
//   approval_status: String,        // approved, pending, draft
//   evidence_summary: Map,          // Overall evidence quality metrics
//   digital_signature: String,      // Integrity verification
//   created_at: DateTime,
//   updated_at: DateTime
// })

// Clinical recommendations with evidence backing
// (:Recommendation {
//   rec_id: String,                 // Unique recommendation identifier
//   statement: String,              // Full recommendation text
//   domain: String,                 // diagnosis, treatment, monitoring, etc.
//   evidence_grade: String,         // A, B, C, D, Expert Opinion
//   quality_score: Integer,         // 0-100 quality score
//   population_criteria: Map,       // Age, conditions, exclusions
//   safety_considerations: [Map],   // Safety warnings and monitoring
//   linked_kb_refs: Map,           // References to other KBs
//   created_at: DateTime,
//   updated_at: DateTime
// })

// Conflict tracking and resolution
// (:Conflict {
//   conflict_id: String,            // Unique conflict identifier
//   type: String,                   // direct_contradiction, evidence_disagreement, etc.
//   severity: String,               // critical, major, minor
//   description: String,            // Human-readable conflict description
//   detected_at: DateTime,          // When conflict was identified
//   resolution_status: String,      // pending, resolved, escalated
//   resolution_rule: String,        // Which rule was applied
//   rationale: String               // Why this resolution was chosen
// })

// Safety overrides with priority handling
// (:SafetyOverride {
//   override_id: String,            // Unique override identifier (SAFETY-001)
//   priority: Integer,              // 1-10 priority ranking
//   condition: String,              // Clinical condition triggering override
//   description: String,            // Human-readable description
//   trigger_conditions: Map,        // Lab values, conditions, medications
//   override_action: Map,           // Action to take and alternatives
//   requires_audit: Boolean,        // Whether audit logging is required
//   active: Boolean,                // Whether override is currently active
//   created_at: DateTime,
//   last_reviewed: DateTime
// })

// External knowledge base references
// (:ExternalReference {
//   kb_name: String,                // KB-1, KB-2, KB-4, etc.
//   reference_id: String,           // Target ID in external KB
//   reference_type: String,         // dosing, interaction, monitoring
//   validation_status: String,      // valid, invalid, pending
//   last_validated: DateTime
// })

// =============================================================================
// RELATIONSHIP DEFINITIONS
// =============================================================================

// Core guideline-recommendation relationship
// (g:Guideline)-[:CONTAINS]->(r:Recommendation)

// Conflict relationships between recommendations
// (r1:Recommendation)-[:CONFLICTS_WITH {
//   type: String,                   // Type of conflict
//   severity: String,               // Severity level
//   resolution_rule: String,        // Applied resolution rule
//   resolved_at: DateTime,          // When conflict was resolved
//   winning_rec: String             // Which recommendation won
// }]->(r2:Recommendation)

// Guideline succession tracking
// (g1:Guideline)-[:SUPERSEDES {
//   transition_date: Date,          // When succession occurred
//   major_changes: [String],        // List of major changes
//   clinical_impact: String,        // High, Medium, Low impact
//   migration_required: Boolean     // Whether active migration is needed
// }]->(g2:Guideline)

// Recommendation supersession
// (r1:Recommendation)-[:SUPERSEDED_BY {
//   reason: String,                 // Why it was superseded
//   effective_date: Date,           // When supersession becomes effective
//   grace_period_days: Integer      // Transition period allowed
// }]->(r2:Recommendation)

// Safety override relationships
// (s:SafetyOverride)-[:OVERRIDES {
//   priority: Integer,              // Override priority
//   condition_match: [String],      // Which conditions triggered
//   applied_at: DateTime,           // When override was applied
//   audit_trail: String             // Reference to audit log
// }]->(r:Recommendation)

// Cross-KB linkages with validation
// (r:Recommendation)-[:LINKS_TO {
//   kb: String,                     // Target knowledge base
//   target_id: String,              // Target entity ID
//   link_type: String,              // dosing, interaction, monitoring
//   validation_status: String,      // valid, invalid, pending
//   last_validated: DateTime,       // When link was last checked
//   validation_error: String        // Error message if invalid
// }]->(ref:ExternalReference)

// Conflict resolution audit trail
// (c:Conflict)-[:RESOLVED_BY {
//   resolver: String,               // System or user who resolved
//   resolution_time: Duration,      // How long resolution took
//   manual_override: Boolean,       // Whether manual intervention was needed
//   appeal_filed: Boolean           // Whether resolution was appealed
// }]->(r:Recommendation)

// =============================================================================
// SAMPLE QUERIES FOR COMMON OPERATIONS
// =============================================================================

// Query 1: Get applicable guidelines for condition with conflict resolution
// MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)
// WHERE g.region = $region 
//   AND r.domain = $domain
//   AND g.condition_primary = $condition
//   AND g.status = 'active'
//   AND NOT EXISTS((g)-[:SUPERSEDED_BY]->())
// OPTIONAL MATCH (r)-[conf:CONFLICTS_WITH]-(conflicting_r)
// OPTIONAL MATCH (so:SafetyOverride)-[override:OVERRIDES]->(r)
// RETURN g, r, 
//        collect(DISTINCT {conflict: conflicting_r, details: conf}) as conflicts,
//        collect(DISTINCT {override: so, details: override}) as overrides
// ORDER BY r.evidence_grade, g.effective_date DESC

// Query 2: Detect new conflicts between guidelines
// MATCH (g1:Guideline)-[:CONTAINS]->(r1:Recommendation)
// MATCH (g2:Guideline)-[:CONTAINS]->(r2:Recommendation)
// WHERE g1.guideline_id IN $new_guidelines
//   AND g2.guideline_id <> g1.guideline_id
//   AND r1.domain = r2.domain
//   AND NOT EXISTS((r1)-[:CONFLICTS_WITH]-(r2))
//   AND (
//     (r1.statement CONTAINS 'target' AND r2.statement CONTAINS 'target'
//      AND r1.statement <> r2.statement)
//     OR
//     (r1.evidence_grade <> r2.evidence_grade 
//      AND r1.rec_id = r2.rec_id)
//   )
// RETURN g1, r1, g2, r2

// Query 3: Get clinical pathway with resolved conflicts
// MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)
// WHERE g.region = $region 
//   AND r.domain IN $domains
//   AND any(code IN g.icd10_codes WHERE code IN $patient_conditions)
// CALL {
//   WITH r
//   OPTIONAL MATCH (r)-[conf:CONFLICTS_WITH]-(cr)
//   RETURN r, collect({conflicting_rec: cr, resolution: conf.resolution_rule}) as conflicts
// }
// CALL {
//   WITH r
//   OPTIONAL MATCH (so:SafetyOverride)-[ov:OVERRIDES]->(r)
//   WHERE so.active = true
//   RETURN r, collect({override: so, action: ov}) as overrides
// }
// RETURN r, conflicts, overrides
// ORDER BY r.evidence_grade, size(overrides) DESC