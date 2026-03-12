// Neo4j Graph Schema Initialization
// Creates constraints and indexes for patient journey graphs

// ============================================================================
// Unique Constraints on nodeId
// ============================================================================

CREATE CONSTRAINT patient_id IF NOT EXISTS
FOR (p:Patient) REQUIRE p.nodeId IS UNIQUE;

CREATE CONSTRAINT event_id IF NOT EXISTS
FOR (e:ClinicalEvent) REQUIRE e.nodeId IS UNIQUE;

CREATE CONSTRAINT condition_id IF NOT EXISTS
FOR (c:Condition) REQUIRE c.nodeId IS UNIQUE;

CREATE CONSTRAINT medication_id IF NOT EXISTS
FOR (m:Medication) REQUIRE m.nodeId IS UNIQUE;

CREATE CONSTRAINT procedure_id IF NOT EXISTS
FOR (p:Procedure) REQUIRE p.nodeId IS UNIQUE;

CREATE CONSTRAINT department_id IF NOT EXISTS
FOR (d:Department) REQUIRE d.nodeId IS UNIQUE;

CREATE CONSTRAINT device_id IF NOT EXISTS
FOR (d:Device) REQUIRE d.nodeId IS UNIQUE;

// ============================================================================
// Performance Indexes
// ============================================================================

CREATE INDEX patient_last_updated IF NOT EXISTS
FOR (p:Patient) ON (p.lastUpdated);

CREATE INDEX event_timestamp IF NOT EXISTS
FOR (e:ClinicalEvent) ON (e.timestamp);

CREATE INDEX event_patient IF NOT EXISTS
FOR (e:ClinicalEvent) ON (e.patientId);

CREATE INDEX condition_patient IF NOT EXISTS
FOR (c:Condition) ON (c.patientId);

CREATE INDEX medication_patient IF NOT EXISTS
FOR (m:Medication) ON (m.patientId);

// ============================================================================
// Example Queries
// ============================================================================

// Get patient journey (chronological events)
// MATCH (p:Patient {nodeId: 'P12345'})-[:HAS_EVENT]->(e:ClinicalEvent)
// RETURN p, e ORDER BY e.timestamp;

// Find temporal sequence (event chain)
// MATCH path = (e1:ClinicalEvent)-[:NEXT_EVENT*]->(e2:ClinicalEvent)
// WHERE e1.patientId = 'P12345'
// RETURN path;

// Clinical pathway analysis
// MATCH (p:Patient)-[:HAS_CONDITION]->(c:Condition),
//       (p)-[:HAS_EVENT]->(e:ClinicalEvent)-[:TRIGGERED_BY]->(c)
// RETURN p, c, e;
