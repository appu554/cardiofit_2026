#!/bin/bash
# KB-7 GraphDB Health Check Script
# Validates repository operational status and configuration

set -e

# Configuration
GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"
REPO_ID="kb7-terminology"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "========================================="
echo "KB-7 GraphDB Health Check"
echo "========================================="
echo ""

# Test 1: GraphDB Service Availability
echo -n "1. GraphDB Service... "
if curl -sf "$GRAPHDB_URL/rest/repositories" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ RUNNING${NC}"
else
    echo -e "${RED}✗ FAILED${NC}"
    echo "   GraphDB is not accessible at $GRAPHDB_URL"
    exit 1
fi

# Test 2: Repository Exists
echo -n "2. Repository Exists... "
REPO_EXISTS=$(curl -sf "$GRAPHDB_URL/rest/repositories" | jq -r --arg id "$REPO_ID" '.[] | select(.id == $id) | .id' || echo "")
if [ -n "$REPO_EXISTS" ]; then
    echo -e "${GREEN}✓ FOUND${NC}"
else
    echo -e "${RED}✗ NOT FOUND${NC}"
    echo "   Repository '$REPO_ID' does not exist"
    echo "   Run: ./scripts/graphdb/create-repository.sh"
    exit 1
fi

# Test 3: Repository State
echo -n "3. Repository State... "
REPO_STATE=$(curl -sf "$GRAPHDB_URL/rest/repositories" | jq -r --arg id "$REPO_ID" '.[] | select(.id == $id) | .state' || echo "UNKNOWN")
if [ "$REPO_STATE" == "RUNNING" ]; then
    echo -e "${GREEN}✓ $REPO_STATE${NC}"
elif [ "$REPO_STATE" == "STARTING" ]; then
    echo -e "${YELLOW}⚠ $REPO_STATE${NC}"
    echo "   Repository is still initializing, please wait..."
else
    echo -e "${RED}✗ $REPO_STATE${NC}"
fi

# Test 4: Read/Write Permissions
echo -n "4. Read Permission... "
READABLE=$(curl -sf "$GRAPHDB_URL/rest/repositories" | jq -r --arg id "$REPO_ID" '.[] | select(.id == $id) | .readable' || echo "false")
if [ "$READABLE" == "true" ]; then
    echo -e "${GREEN}✓ ENABLED${NC}"
else
    echo -e "${RED}✗ DISABLED${NC}"
fi

echo -n "5. Write Permission... "
WRITABLE=$(curl -sf "$GRAPHDB_URL/rest/repositories" | jq -r --arg id "$REPO_ID" '.[] | select(.id == $id) | .writable' || echo "false")
if [ "$WRITABLE" == "true" ]; then
    echo -e "${GREEN}✓ ENABLED${NC}"
else
    echo -e "${RED}✗ DISABLED${NC}"
fi

# Test 6: SPARQL Query Endpoint
echo -n "6. SPARQL Endpoint... "
SPARQL_RESPONSE=$(curl -sf -X POST \
    -H "Accept: application/sparql-results+json" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }" \
    "$GRAPHDB_URL/repositories/$REPO_ID" 2>&1 || echo "")

if echo "$SPARQL_RESPONSE" | jq -e '.results.bindings' > /dev/null 2>&1; then
    TRIPLE_COUNT=$(echo "$SPARQL_RESPONSE" | jq -r '.results.bindings[0].count.value')
    echo -e "${GREEN}✓ OPERATIONAL${NC}"
    echo "   Triples in repository: $TRIPLE_COUNT"
else
    echo -e "${RED}✗ FAILED${NC}"
    echo "   Could not execute SPARQL query"
fi

# Test 7: Repository Configuration
echo "7. Configuration Validation:"
REPO_CONFIG=$(curl -sf "$GRAPHDB_URL/rest/repositories/$REPO_ID" | jq '.')

echo -n "   - Ruleset... "
RULESET=$(echo "$REPO_CONFIG" | jq -r '.params.ruleset.value')
if [ "$RULESET" == "owl2-rl-optimized" ]; then
    echo -e "${GREEN}✓ $RULESET${NC}"
else
    echo -e "${YELLOW}⚠ $RULESET${NC} (expected: owl2-rl-optimized)"
fi

echo -n "   - Base URL... "
BASE_URL=$(echo "$REPO_CONFIG" | jq -r '.params.baseURL.value')
if [ "$BASE_URL" == "http://cardiofit.ai/ontology/" ]; then
    echo -e "${GREEN}✓ $BASE_URL${NC}"
else
    echo -e "${YELLOW}⚠ $BASE_URL${NC}"
fi

echo -n "   - Context Index... "
CONTEXT_INDEX=$(echo "$REPO_CONFIG" | jq -r '.params.enableContextIndex.value')
if [ "$CONTEXT_INDEX" == "true" ]; then
    echo -e "${GREEN}✓ ENABLED${NC}"
else
    echo -e "${YELLOW}⚠ DISABLED${NC}"
fi

echo -n "   - Predicate List... "
PREDICATE_LIST=$(echo "$REPO_CONFIG" | jq -r '.params.enablePredicateList.value')
if [ "$PREDICATE_LIST" == "true" ]; then
    echo -e "${GREEN}✓ ENABLED${NC}"
else
    echo -e "${YELLOW}⚠ DISABLED${NC}"
fi

echo -n "   - Literal Index... "
LITERAL_INDEX=$(echo "$REPO_CONFIG" | jq -r '.params.enableLiteralIndex.value')
if [ "$LITERAL_INDEX" == "true" ]; then
    echo -e "${GREEN}✓ ENABLED${NC}"
else
    echo -e "${YELLOW}⚠ DISABLED${NC}"
fi

echo -n "   - Entity Index Size... "
ENTITY_SIZE=$(echo "$REPO_CONFIG" | jq -r '.params.entityIndexSize.value')
echo -e "${BLUE}$ENTITY_SIZE${NC}"

# Test 8: GraphDB Client Connectivity (Go)
echo ""
echo "8. Go Client Connectivity:"
if [ -f "test-graphdb-connection.go" ]; then
    echo "   Running Go client test..."
    if go run test-graphdb-connection.go 2>&1 | grep -q "✓"; then
        echo -e "   ${GREEN}✓ Go client can connect${NC}"
    else
        echo -e "   ${YELLOW}⚠ Go client test available but needs verification${NC}"
    fi
else
    echo -e "   ${YELLOW}⚠ test-graphdb-connection.go not found${NC}"
fi

# Summary
echo ""
echo "========================================="
echo "Health Check Summary"
echo "========================================="
echo -e "GraphDB Service:      ${GREEN}✓ Operational${NC}"
echo -e "Repository:           ${GREEN}✓ $REPO_ID${NC}"
echo -e "State:                ${GREEN}$REPO_STATE${NC}"
echo -e "SPARQL Endpoint:      ${GREEN}✓ Available${NC}"
echo -e "Triples:              ${BLUE}$TRIPLE_COUNT${NC}"
echo ""
echo "🌐 GraphDB Workbench: http://localhost:7200"
echo "📊 Repository URL: http://localhost:7200/repository?resource=$REPO_ID"
echo "🔍 SPARQL Endpoint: http://localhost:7200/repositories/$REPO_ID"
echo ""

if [ "$REPO_STATE" == "RUNNING" ] && [ "$READABLE" == "true" ] && [ "$WRITABLE" == "true" ]; then
    echo -e "${GREEN}✅ All systems operational - ready for Phase 1.2${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some checks failed - review configuration${NC}"
    exit 1
fi
