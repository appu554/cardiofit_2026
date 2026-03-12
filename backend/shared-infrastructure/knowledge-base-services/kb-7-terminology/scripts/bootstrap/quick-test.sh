#!/bin/bash
set -e

# Quick test script for bootstrap migration
# Tests migration with 10 concepts to verify everything works

echo "=========================================="
echo "KB-7 Bootstrap Migration Quick Test"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
echo "Checking prerequisites..."

# Check PostgreSQL
if ! psql "${DATABASE_URL:-postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology}" -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${RED}❌ PostgreSQL is not accessible${NC}"
    echo "Expected: postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology"
    exit 1
fi
echo -e "${GREEN}✓${NC} PostgreSQL connection OK"

# Check GraphDB
GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"
if ! curl -s -f "${GRAPHDB_URL}/rest/repositories" > /dev/null; then
    echo -e "${RED}❌ GraphDB is not accessible${NC}"
    echo "Expected: ${GRAPHDB_URL}"
    exit 1
fi
echo -e "${GREEN}✓${NC} GraphDB connection OK"

# Check GraphDB repository
REPO="${GRAPHDB_REPOSITORY:-kb7-terminology}"
if ! curl -s -f "${GRAPHDB_URL}/rest/repositories/${REPO}" > /dev/null; then
    echo -e "${YELLOW}⚠${NC}  GraphDB repository '${REPO}' not found"
    echo "Creating repository..."

    # Try to create repository
    curl -X POST "${GRAPHDB_URL}/rest/repositories" \
        -H "Content-Type: application/json" \
        -d "{\"id\":\"${REPO}\",\"title\":\"KB-7 Terminology Service\",\"type\":\"graphdb\",\"params\":{\"ruleset\":{\"label\":\"OWL2-RL (Optimized)\",\"value\":\"owl2-rl-optimized\"}}}" \
        > /dev/null 2>&1 || {
        echo -e "${RED}❌ Failed to create repository${NC}"
        echo "Please create repository manually via GraphDB Workbench"
        exit 1
    }
    echo -e "${GREEN}✓${NC} Repository created"
fi
echo -e "${GREEN}✓${NC} GraphDB repository '${REPO}' exists"

# Check concept count
CONCEPT_COUNT=$(psql "${DATABASE_URL:-postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology}" -t -c "SELECT COUNT(*) FROM terminology_concepts WHERE status = 'active';" | xargs)
echo -e "${GREEN}✓${NC} Found ${CONCEPT_COUNT} concepts in PostgreSQL"

if [ "$CONCEPT_COUNT" -lt 10 ]; then
    echo -e "${RED}❌ Need at least 10 concepts for testing${NC}"
    exit 1
fi

echo ""
echo "=========================================="
echo "Running Test Migration (10 concepts)"
echo "=========================================="
echo ""

# Clear existing bootstrap data
echo "Clearing existing bootstrap data..."
curl -s -X POST "${GRAPHDB_URL}/repositories/${REPO}/statements" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "update=CLEAR GRAPH <http://cardiofit.ai/bootstrap>" \
    > /dev/null 2>&1 || true

# Run migration
go run scripts/bootstrap/postgres-to-graphdb.go \
    --max 10 \
    --batch 10 \
    --log-interval 5

EXIT_CODE=$?

echo ""
echo "=========================================="
echo "Verification"
echo "=========================================="
echo ""

if [ $EXIT_CODE -eq 0 ]; then
    # Count concepts in GraphDB
    GRAPHDB_COUNT=$(curl -s -X POST "${GRAPHDB_URL}/repositories/${REPO}" \
        --data-urlencode "query=PREFIX kb7: <http://cardiofit.ai/kb7/ontology#> SELECT (COUNT(?concept) AS ?count) WHERE { ?concept a kb7:ClinicalConcept . }" \
        -H "Accept: application/sparql-results+json" | \
        jq -r '.results.bindings[0].count.value' 2>/dev/null || echo "0")

    echo "Concepts in GraphDB: ${GRAPHDB_COUNT}"

    if [ "$GRAPHDB_COUNT" -eq 10 ]; then
        echo -e "${GREEN}✅ Test migration successful!${NC}"
        echo ""
        echo "Sample SPARQL query:"
        curl -s -X POST "${GRAPHDB_URL}/repositories/${REPO}" \
            --data-urlencode "query=PREFIX kb7: <http://cardiofit.ai/kb7/ontology#> PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#> SELECT ?code ?label WHERE { ?concept a kb7:ClinicalConcept ; kb7:code ?code ; rdfs:label ?label . } LIMIT 3" \
            -H "Accept: application/sparql-results+json" | jq -r '.results.bindings[] | "  - \(.code.value): \(.label.value)"'

        echo ""
        echo -e "${GREEN}Ready for full migration!${NC}"
        echo ""
        echo "To run full migration (520K concepts):"
        echo "  go run scripts/bootstrap/postgres-to-graphdb.go"
        echo ""
        echo "Estimated time: 2-4 hours"
        exit 0
    else
        echo -e "${RED}❌ Expected 10 concepts, found ${GRAPHDB_COUNT}${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ Migration failed${NC}"
    exit 1
fi
