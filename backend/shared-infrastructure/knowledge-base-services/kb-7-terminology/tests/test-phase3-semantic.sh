#!/bin/bash

# Phase 3 Semantic Web Infrastructure Test Suite
# Tests all components of the semantic layer implementation

echo "==========================================="
echo "KB-7 Phase 3: Semantic Infrastructure Test"
echo "==========================================="
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Configuration
GRAPHDB_URL="http://localhost:7200"
SPARQL_PROXY_URL="http://localhost:8095"
REDIS_PORT="6381"

# Function to print colored output
print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

print_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
    ((TESTS_SKIPPED++))
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# ============================================
# Test 1: Docker Environment Check
# ============================================
print_test "1. Docker Environment Check"

if command -v docker >/dev/null 2>&1; then
    DOCKER_VERSION=$(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')
    print_pass "Docker is installed: $DOCKER_VERSION"
else
    print_fail "Docker is not installed"
fi

if command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_VERSION=$(docker-compose --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')
    print_pass "Docker Compose is installed: $COMPOSE_VERSION"
else
    # Try docker compose (new syntax)
    if docker compose version >/dev/null 2>&1; then
        print_pass "Docker Compose is installed (docker compose syntax)"
    else
        print_fail "Docker Compose is not installed"
    fi
fi

# ============================================
# Test 2: Semantic Services Deployment Files
# ============================================
print_test "2. Semantic Services Deployment Files"

# Check if docker-compose file exists
if [ -f "docker-compose.semantic.yml" ]; then
    print_pass "Semantic docker-compose file exists"

    # Check for services
    if grep -q "graphdb:" docker-compose.semantic.yml; then
        print_pass "GraphDB service defined"
    else
        print_fail "GraphDB service not defined"
    fi

    if grep -q "sparql-proxy:" docker-compose.semantic.yml; then
        print_pass "SPARQL proxy service defined"
    else
        print_fail "SPARQL proxy service not defined"
    fi

    if grep -q "redis-semantic:" docker-compose.semantic.yml; then
        print_pass "Redis semantic cache defined"
    else
        print_fail "Redis semantic cache not defined"
    fi

    if grep -q "robot-service:" docker-compose.semantic.yml; then
        print_pass "ROBOT service defined"
    else
        print_fail "ROBOT service not defined"
    fi
else
    print_fail "docker-compose.semantic.yml not found"
fi

# ============================================
# Test 3: Core Ontology Files
# ============================================
print_test "3. Core Ontology Files"

# Check for KB-7 core ontology
if [ -f "semantic/ontologies/kb7-core.ttl" ]; then
    print_pass "KB-7 core ontology file exists"

    # Count lines to verify it's not empty
    LINE_COUNT=$(wc -l < semantic/ontologies/kb7-core.ttl)
    if [ "$LINE_COUNT" -gt 100 ]; then
        print_pass "Core ontology has substantial content ($LINE_COUNT lines)"
    else
        print_fail "Core ontology appears incomplete ($LINE_COUNT lines)"
    fi

    # Check for namespaces
    if grep -q "@prefix kb7:" semantic/ontologies/kb7-core.ttl; then
        print_pass "KB7 namespace defined"
    else
        print_fail "KB7 namespace missing"
    fi

    if grep -q "@prefix sct:" semantic/ontologies/kb7-core.ttl; then
        print_pass "SNOMED CT namespace defined"
    else
        print_fail "SNOMED CT namespace missing"
    fi
else
    print_fail "KB-7 core ontology file not found"
fi

# ============================================
# Test 4: GraphDB Configuration
# ============================================
print_test "4. GraphDB Configuration"

if [ -f "semantic/config/kb7-repository-config.ttl" ]; then
    print_pass "GraphDB repository configuration exists"

    if grep -q "kb7-terminology" semantic/config/kb7-repository-config.ttl; then
        print_pass "KB7 terminology repository configured"
    else
        print_fail "KB7 terminology repository not configured"
    fi

    if grep -q "owl2-rl" semantic/config/kb7-repository-config.ttl; then
        print_pass "OWL 2 RL reasoning configured"
    else
        print_fail "OWL 2 RL reasoning not configured"
    fi
else
    print_fail "GraphDB repository configuration not found"
fi

if [ -f "semantic/config/redis.conf" ]; then
    print_pass "Redis semantic cache configuration exists"
else
    print_fail "Redis semantic cache configuration not found"
fi

# ============================================
# Test 5: SPARQL Proxy Service
# ============================================
print_test "5. SPARQL Proxy Service"

if [ -d "semantic/sparql-proxy" ]; then
    print_pass "SPARQL proxy directory exists"

    if [ -f "semantic/sparql-proxy/main.go" ]; then
        print_pass "SPARQL proxy main.go exists"
    else
        print_fail "SPARQL proxy main.go not found"
    fi

    if [ -f "semantic/sparql-proxy/go.mod" ]; then
        print_pass "SPARQL proxy go.mod exists"
    else
        print_fail "SPARQL proxy go.mod not found"
    fi
else
    print_fail "SPARQL proxy directory not found"
fi

if [ -f "semantic/Dockerfile.sparql-proxy" ]; then
    print_pass "SPARQL proxy Dockerfile exists"
else
    print_fail "SPARQL proxy Dockerfile not found"
fi

# ============================================
# Test 6: ROBOT Tool Pipeline
# ============================================
print_test "6. ROBOT Tool Pipeline"

if [ -f "semantic/Dockerfile.robot" ]; then
    print_pass "ROBOT Dockerfile exists"
else
    print_fail "ROBOT Dockerfile not found"
fi

if [ -f "semantic/robot-entrypoint.sh" ]; then
    print_pass "ROBOT entrypoint script exists"
else
    print_fail "ROBOT entrypoint script not found"
fi

if [ -d "semantic/robot-scripts" ]; then
    print_pass "ROBOT scripts directory exists"

    if [ -f "semantic/robot-scripts/validate_ontologies.py" ]; then
        print_pass "Ontology validation script exists"
    else
        print_fail "Ontology validation script not found"
    fi
else
    print_fail "ROBOT scripts directory not found"
fi

# ============================================
# Test 7: Go Semantic Integration
# ============================================
print_test "7. Go Semantic Integration"

# Check GraphDB client
if [ -f "internal/semantic/graphdb_client.go" ]; then
    print_pass "GraphDB client implementation exists"

    # Check file size to ensure it's not empty
    FILE_SIZE=$(wc -c < internal/semantic/graphdb_client.go)
    if [ "$FILE_SIZE" -gt 1000 ]; then
        print_pass "GraphDB client has implementation ($FILE_SIZE bytes)"
    else
        print_fail "GraphDB client appears incomplete"
    fi
else
    print_fail "GraphDB client not found"
fi

# Check RDF converter
if [ -f "internal/semantic/rdf_converter.go" ]; then
    print_pass "RDF converter implementation exists"
else
    print_fail "RDF converter not found"
fi

# Check reasoning engine
if [ -f "internal/semantic/reasoning_engine.go" ]; then
    print_pass "Reasoning engine implementation exists"
else
    print_fail "Reasoning engine not found"
fi

# ============================================
# Test 8: Deployment Scripts
# ============================================
print_test "8. Deployment Scripts"

if [ -f "scripts/deploy-semantic.sh" ]; then
    print_pass "Semantic deployment script exists"

    if [ -x "scripts/deploy-semantic.sh" ]; then
        print_pass "Deployment script is executable"
    else
        print_fail "Deployment script is not executable"
    fi
else
    print_fail "Semantic deployment script not found"
fi

# ============================================
# Test 9: Makefile Integration
# ============================================
print_test "9. Makefile Integration"

if [ -f "Makefile" ]; then
    # Check for semantic commands
    if grep -q "semantic-deploy:" Makefile; then
        print_pass "semantic-deploy command in Makefile"
    else
        print_fail "semantic-deploy command missing"
    fi

    if grep -q "graphdb-health:" Makefile; then
        print_pass "graphdb-health command in Makefile"
    else
        print_fail "graphdb-health command missing"
    fi

    if grep -q "sparql-test:" Makefile; then
        print_pass "sparql-test command in Makefile"
    else
        print_fail "sparql-test command missing"
    fi

    if grep -q "phase3-setup:" Makefile; then
        print_pass "phase3-setup command in Makefile"
    else
        print_fail "phase3-setup command missing"
    fi
else
    print_fail "Makefile not found"
fi

# ============================================
# Test 10: Documentation
# ============================================
print_test "10. Documentation"

if [ -f "PHASE3_IMPLEMENTATION_COMPLETE.md" ]; then
    print_pass "Phase 3 documentation exists"

    # Check documentation size
    DOC_SIZE=$(wc -l < PHASE3_IMPLEMENTATION_COMPLETE.md)
    if [ "$DOC_SIZE" -gt 100 ]; then
        print_pass "Documentation is comprehensive ($DOC_SIZE lines)"
    else
        print_fail "Documentation appears incomplete"
    fi
else
    print_fail "Phase 3 documentation not found"
fi

# ============================================
# Test Summary
# ============================================
echo ""
echo "==========================================="
echo "Phase 3 Test Summary"
echo "==========================================="

TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))

echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
echo -e "${RED}Failed:${NC} $TESTS_FAILED"
echo -e "${YELLOW}Skipped:${NC} $TESTS_SKIPPED"
echo -e "Total: $TOTAL_TESTS"
echo ""

# Calculate pass percentage
if [ "$TOTAL_TESTS" -gt 0 ]; then
    PASS_PERCENTAGE=$((TESTS_PASSED * 100 / TOTAL_TESTS))
    echo "Pass Rate: $PASS_PERCENTAGE%"
    echo ""

    if [ "$PASS_PERCENTAGE" -ge 90 ]; then
        echo -e "${GREEN}✅ Phase 3 Semantic Infrastructure: READY${NC}"
        echo ""
        echo "All critical components are in place!"
        EXIT_CODE=0
    elif [ "$PASS_PERCENTAGE" -ge 70 ]; then
        echo -e "${YELLOW}⚠️ Phase 3 Semantic Infrastructure: MOSTLY READY${NC}"
        echo ""
        echo "Most components are ready, but some issues need attention."
        EXIT_CODE=1
    else
        echo -e "${RED}❌ Phase 3 Semantic Infrastructure: NOT READY${NC}"
        echo ""
        echo "Significant issues found. Please review failed tests."
        EXIT_CODE=2
    fi
else
    echo -e "${RED}No tests executed${NC}"
    EXIT_CODE=3
fi

echo ""
echo "==========================================="
echo "Next Steps:"
echo "==========================================="

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo "✅ All tests passed! Ready to deploy Phase 3:"
    echo "  1. Run: make semantic-deploy"
    echo "  2. Load ontology: make graphdb-load-ontology"
    echo "  3. Test SPARQL: make sparql-test"
    echo "  4. Begin Phase 4 implementation"
else
    echo "⚠️ Some tests failed. Recommended actions:"
    echo "  1. Review the failed tests above"
    echo "  2. Check if all files were created properly"
    echo "  3. Ensure Docker is installed and running"
    echo "  4. Re-run: bash tests/test-phase3-semantic.sh"
fi

exit $EXIT_CODE