#!/bin/bash

# Load Rohan Sharma data into Neo4j via HTTP API
# This bypasses authentication issues with cypher-shell

set -e

NEO4J_HOST="localhost:7474"
NEO4J_USER="neo4j"
NEO4J_PASS="CardioFit2024!"

echo "========================================================================"
echo "Loading Rohan Sharma Graph Data into Neo4j (via HTTP API)"
echo "========================================================================"
echo "Neo4j HTTP: http://$NEO4J_HOST"
echo ""

# Function to execute Cypher query
execute_cypher() {
    local query="$1"
    local description="$2"

    echo "📋 $description..."

    response=$(curl -s -u "$NEO4J_USER:$NEO4J_PASS" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -X POST "http://$NEO4J_HOST/db/neo4j/tx/commit" \
        -d "{\"statements\":[{\"statement\":\"$query\"}]}")

    if echo "$response" | grep -q "\"errors\":\[\]"; then
        echo "✅ Success"
        return 0
    else
        echo "❌ Error: $response"
        return 1
    fi
}

# Clear existing data
execute_cypher "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) DETACH DELETE p" \
    "Clearing existing Rohan data"

# Create patient node
execute_cypher "CREATE (p:Patient {patientId: 'PAT-ROHAN-001', name: 'Rohan Sharma', birthYear: 1983, gender: 'male', city: 'Bengaluru'})" \
    "Creating Patient node"

# Create conditions
execute_cypher "CREATE (c1:Condition {code: '38341003', name: 'Hypertension'}) CREATE (c2:Condition {code: '15777000', name: 'Prediabetes'})" \
    "Creating Condition nodes"

# Link conditions to patient
execute_cypher "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) MATCH (c1:Condition {code: '38341003'}) MATCH (c2:Condition {code: '15777000'}) MERGE (p)-[:HAS_CONDITION]->(c1) MERGE (p)-[:HAS_CONDITION]->(c2)" \
    "Linking Conditions to Patient"

# Create lifestyle factors
execute_cypher "CREATE (lf1:LifestyleFactor {name: 'Sedentary Lifestyle'}) CREATE (lf2:LifestyleFactor {name: 'High Stress'}) CREATE (lf3:LifestyleFactor {name: 'Low Fruit/Veg Intake'})" \
    "Creating Lifestyle Factor nodes"

# Link lifestyle to patient
execute_cypher "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) MATCH (lf1:LifestyleFactor {name: 'Sedentary Lifestyle'}) MATCH (lf2:LifestyleFactor {name: 'High Stress'}) MATCH (lf3:LifestyleFactor {name: 'Low Fruit/Veg Intake'}) MERGE (p)-[:EXHIBITS_LIFESTYLE]->(lf1) MERGE (p)-[:EXHIBITS_LIFESTYLE]->(lf2) MERGE (p)-[:EXHIBITS_LIFESTYLE]->(lf3)" \
    "Linking Lifestyle to Patient"

# Create clinician
execute_cypher "CREATE (doc:Provider {providerId: 'DOC-101', name: 'Dr. Priya Rao', specialty: 'Cardiology'})" \
    "Creating Provider node"

# Link clinician to patient
execute_cypher "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) MATCH (doc:Provider {providerId: 'DOC-101'}) MERGE (p)-[:HAS_PROVIDER]->(doc)" \
    "Linking Provider to Patient"

# Create risk cohort
execute_cypher "CREATE (cohort:Cohort {name: 'Urban Metabolic Syndrome Cohort', region: 'South India'})" \
    "Creating Cohort node"

# Link cohort to patient
execute_cypher "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) MATCH (cohort:Cohort {name: 'Urban Metabolic Syndrome Cohort'}) MERGE (p)-[:IN_COHORT]->(cohort)" \
    "Linking Cohort to Patient"

# Create family history
execute_cypher "CREATE (f:FamilyCondition {condition: 'Myocardial Infarction', onsetAge: 52, relation: 'Father'})" \
    "Creating Family History node"

# Link family history to patient
execute_cypher "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) MATCH (f:FamilyCondition {condition: 'Myocardial Infarction'}) MERGE (p)-[:FAMILY_HISTORY_OF]->(f)" \
    "Linking Family History to Patient"

echo ""
echo "========================================================================"
echo "📊 Verification"
echo "========================================================================"

# Verify graph
verify_query='MATCH (p:Patient {patientId: \"PAT-ROHAN-001\"}) OPTIONAL MATCH (p)-[:HAS_CONDITION]->(c:Condition) OPTIONAL MATCH (p)-[:EXHIBITS_LIFESTYLE]->(lf:LifestyleFactor) OPTIONAL MATCH (p)-[:HAS_PROVIDER]->(prov:Provider) OPTIONAL MATCH (p)-[:IN_COHORT]->(cohort:Cohort) OPTIONAL MATCH (p)-[:FAMILY_HISTORY_OF]->(fh:FamilyCondition) RETURN p.name as patient, collect(DISTINCT c.name) as conditions, collect(DISTINCT lf.name) as lifestyle, collect(DISTINCT prov.name) as providers, collect(DISTINCT cohort.name) as cohorts, collect(DISTINCT fh.condition) as family_history'

result=$(curl -s -u "$NEO4J_USER:$NEO4J_PASS" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -X POST "http://$NEO4J_HOST/db/neo4j/tx/commit" \
    -d "{\"statements\":[{\"statement\":\"$verify_query\"}]}")

echo "$result" | python3 -m json.tool 2>/dev/null || echo "$result"

echo ""
echo "✅ Neo4j Graph Data Loaded Successfully!"
echo ""
echo "🔍 Next Steps:"
echo "  1. Verify in Neo4j Browser: http://localhost:7474"
echo "  2. Load FHIR data: python3 load-synthetic-data-rohan.py"
echo "  3. Test Module 2: ./test-rohan-enrichment.sh"
echo ""
