// ============================================================================
// NCTS RF2 Operations - Neo4j Cypher Scripts
// ============================================================================
// Manual Cypher scripts for NCTS refset data management.
// These can be run directly in Neo4j Browser or via cypher-shell.
// ============================================================================

// ============================================================================
// INDEXES AND CONSTRAINTS
// ============================================================================
// Run these BEFORE importing data for optimal performance

// Index for Refset node lookups
CREATE INDEX refset_id_idx IF NOT EXISTS FOR (r:Refset) ON (r.id);

// Index for concept code lookups (should already exist)
CREATE INDEX concept_code_idx IF NOT EXISTS FOR (c:Concept) ON (c.code);

// Index for import metadata
CREATE INDEX import_metadata_idx IF NOT EXISTS FOR (m:ImportMetadata) ON (m.type, m.version);

// Unique constraint for Refset IDs
CREATE CONSTRAINT refset_unique IF NOT EXISTS FOR (r:Refset) REQUIRE r.id IS UNIQUE;

// ============================================================================
// VERSION TRACKING QUERIES
// ============================================================================

// Get current imported version
MATCH (m:ImportMetadata {type: 'NCTS_REFSET'})
RETURN m.version AS version, m.importedAt AS imported_at,
       m.fileCount AS files, m.relationshipCount AS relationships
ORDER BY m.importedAt DESC
LIMIT 1;

// Get all import history
MATCH (m:ImportMetadata {type: 'NCTS_REFSET'})
RETURN m.version AS version, m.importedAt AS imported_at,
       m.fileCount AS files, m.relationshipCount AS relationships
ORDER BY m.importedAt DESC;

// Record new import metadata
// Replace $version, $fileCount, $relationshipCount with actual values
MERGE (m:ImportMetadata {type: 'NCTS_REFSET', version: $version})
SET m.importedAt = datetime(),
    m.fileCount = $fileCount,
    m.relationshipCount = $relationshipCount,
    m.importedBy = 'manual';

// ============================================================================
// REFSET MEMBERSHIP QUERIES
// ============================================================================

// Get all members of a refset
// Replace $refsetId with actual refset ID (e.g., '32570581000036109')
MATCH (c:Concept)-[m:IN_REFSET]->(r:Refset {id: $refsetId})
WHERE m.active = true
RETURN c.code AS code, c.preferredLabel AS label,
       m.effectiveTime AS effective_time, m.memberId AS member_id
ORDER BY c.preferredLabel
LIMIT 100;

// Get all refsets a concept belongs to
// Replace $conceptCode with actual SNOMED code (e.g., '123456789')
MATCH (c:Concept {code: $conceptCode})-[m:IN_REFSET]->(r:Refset)
WHERE m.active = true
RETURN r.id AS refset_id, r.name AS refset_name,
       m.effectiveTime AS effective_time
ORDER BY r.name;

// Check if concept is in a specific refset (O(1) lookup)
MATCH (c:Concept {code: $conceptCode})-[m:IN_REFSET {active: true}]->(r:Refset {id: $refsetId})
RETURN count(m) > 0 AS is_member;

// Count members in a refset
MATCH (c:Concept)-[m:IN_REFSET {active: true}]->(r:Refset {id: $refsetId})
RETURN count(c) AS member_count;

// ============================================================================
// REFSET LISTING QUERIES
// ============================================================================

// List all refsets with member counts
MATCH (r:Refset)
OPTIONAL MATCH (c:Concept)-[m:IN_REFSET {active: true}]->(r)
WITH r, count(m) AS memberCount
RETURN r.id AS id, r.name AS name, memberCount
ORDER BY memberCount DESC
LIMIT 50;

// List refsets by module
MATCH (c:Concept)-[m:IN_REFSET]->(r:Refset)
WHERE m.moduleId = '32506021000036107'  // SNOMED-AU module
WITH r, count(m) AS memberCount
RETURN r.id AS id, r.name AS name, memberCount
ORDER BY memberCount DESC;

// ============================================================================
// BATCH DELETE OPERATIONS
// ============================================================================
// Use APOC for efficient batch deletion

// Delete all IN_REFSET relationships (batch)
CALL apoc.periodic.iterate(
  'MATCH ()-[r:IN_REFSET]->() RETURN r',
  'DELETE r',
  {batchSize: 10000, parallel: false}
) YIELD batches, total, timeTaken
RETURN batches, total, timeTaken;

// Delete all Refset nodes
MATCH (r:Refset) DETACH DELETE r;

// Delete import metadata
MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) DELETE m;

// ============================================================================
// BATCH IMPORT EXAMPLE
// ============================================================================
// Example using LOAD CSV (file must be in Neo4j import directory)

// Import Simple Refset from RF2 file
// File format: id, effectiveTime, active, moduleId, refsetId, referencedComponentId
CALL apoc.periodic.iterate(
  'LOAD CSV WITH HEADERS FROM "file:///der2_Refset_SimpleSnapshot.txt" AS row FIELDTERMINATOR "\t"
   WHERE row.active = "1"
   RETURN row',
  'MATCH (c:Concept {code: row.referencedComponentId})
   MERGE (r:Refset {id: row.refsetId})
   ON CREATE SET r.name = "Unknown Refset"
   CREATE (c)-[:IN_REFSET {
       memberId: row.id,
       effectiveTime: date(substring(row.effectiveTime, 0, 4) + "-" +
                          substring(row.effectiveTime, 4, 2) + "-" +
                          substring(row.effectiveTime, 6, 2)),
       active: true,
       moduleId: row.moduleId
   }]->(r)',
  {batchSize: 5000, parallel: false}
) YIELD batches, total, timeTaken
RETURN batches, total, timeTaken;

// ============================================================================
// ASSOCIATION REFSET OPERATIONS
// ============================================================================

// Import association refset (REPLACED_BY relationships)
// File: der2_cRefset_AssociationSnapshot.txt
// Columns: id, effectiveTime, active, moduleId, refsetId, referencedComponentId, targetComponentId
CALL apoc.periodic.iterate(
  'LOAD CSV WITH HEADERS FROM "file:///der2_cRefset_AssociationSnapshot.txt" AS row FIELDTERMINATOR "\t"
   WHERE row.active = "1" AND row.refsetId = "900000000000526001"  // REPLACED_BY refset
   RETURN row',
  'MATCH (source:Concept {code: row.referencedComponentId})
   MATCH (target:Concept {code: row.targetComponentId})
   CREATE (source)-[:REPLACED_BY {
       memberId: row.id,
       effectiveTime: date(substring(row.effectiveTime, 0, 4) + "-" +
                          substring(row.effectiveTime, 4, 2) + "-" +
                          substring(row.effectiveTime, 6, 2)),
       moduleId: row.moduleId
   }]->(target)',
  {batchSize: 5000, parallel: false}
);

// ============================================================================
// STATISTICS QUERIES
// ============================================================================

// Overall import statistics
MATCH (r:Refset)
WITH count(r) AS refsetCount
MATCH ()-[rel:IN_REFSET]->()
WITH refsetCount, count(rel) AS membershipCount
MATCH ()-[rel2:IN_REFSET {active: true}]->()
WITH refsetCount, membershipCount, count(rel2) AS activeCount
MATCH (m:ImportMetadata {type: 'NCTS_REFSET'})
WITH refsetCount, membershipCount, activeCount, m
ORDER BY m.importedAt DESC
LIMIT 1
RETURN {
    refset_nodes: refsetCount,
    total_memberships: membershipCount,
    active_memberships: activeCount,
    current_version: m.version,
    last_import: m.importedAt
} AS statistics;

// Membership by module
MATCH ()-[m:IN_REFSET]->()
RETURN m.moduleId AS module_id,
       CASE m.moduleId
         WHEN '32506021000036107' THEN 'SNOMED-AU'
         WHEN '900062011000036103' THEN 'AMT'
         WHEN '900000000000207008' THEN 'SNOMED-INT'
         ELSE 'Other'
       END AS module_name,
       count(m) AS membership_count
ORDER BY membership_count DESC;

// Top 10 largest refsets
MATCH (c:Concept)-[m:IN_REFSET {active: true}]->(r:Refset)
WITH r, count(m) AS memberCount
RETURN r.id AS refset_id, r.name AS refset_name, memberCount
ORDER BY memberCount DESC
LIMIT 10;
