#!/bin/bash

# ========================================
# Module 8 Complete Integration Test
# ========================================

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# ========================================
# Helper Functions
# ========================================

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_test() {
    echo -n "Testing: $1 ... "
    ((TESTS_RUN++))
}

pass_test() {
    echo -e "${GREEN}PASS${NC}"
    ((TESTS_PASSED++))
}

fail_test() {
    echo -e "${RED}FAIL${NC}"
    [ -n "$1" ] && echo -e "  ${RED}Error: $1${NC}"
    ((TESTS_FAILED++))
}

# ========================================
# Test Functions
# ========================================

test_files_exist() {
    print_header "File Existence Tests"

    local files=(
        "docker-compose.module8-complete.yml"
        ".env.module8.example"
        "start-module8-projectors.sh"
        "stop-module8-projectors.sh"
        "health-check-module8.sh"
        "logs-module8.sh"
        "configure-network-module8.sh"
        "MODULE8_ORCHESTRATION_COMPLETE.md"
        "MODULE8_QUICK_REFERENCE.md"
    )

    for file in "${files[@]}"; do
        print_test "$file exists"
        if [ -f "$SCRIPT_DIR/$file" ]; then
            pass_test
        else
            fail_test "File not found"
        fi
    done
}

test_scripts_executable() {
    print_header "Script Permissions Tests"

    local scripts=(
        "start-module8-projectors.sh"
        "stop-module8-projectors.sh"
        "health-check-module8.sh"
        "logs-module8.sh"
        "configure-network-module8.sh"
    )

    for script in "${scripts[@]}"; do
        print_test "$script is executable"
        if [ -x "$SCRIPT_DIR/$script" ]; then
            pass_test
        else
            fail_test "Not executable"
        fi
    done
}

test_docker_requirements() {
    print_header "Docker Requirements Tests"

    print_test "Docker installed"
    if command -v docker &> /dev/null; then
        pass_test
    else
        fail_test "Docker not found"
    fi

    print_test "Docker Compose installed"
    if command -v docker-compose &> /dev/null; then
        pass_test
    else
        fail_test "Docker Compose not found"
    fi

    print_test "Docker daemon running"
    if docker info &> /dev/null; then
        pass_test
    else
        fail_test "Docker daemon not running"
    fi
}

test_external_containers() {
    print_header "External Container Tests"

    print_test "PostgreSQL container (a2f55d83b1fa) running"
    if docker ps --format '{{.ID}}' | grep -q "^a2f55d83b1fa"; then
        pass_test
    else
        fail_test "Container not running"
    fi

    print_test "InfluxDB container (8502fd5d078d) running"
    if docker ps --format '{{.ID}}' | grep -q "^8502fd5d078d"; then
        pass_test
    else
        fail_test "Container not running"
    fi

    print_test "Neo4j container (e8b3df4d8a02) running"
    if docker ps --format '{{.ID}}' | grep -q "^e8b3df4d8a02"; then
        pass_test
    else
        fail_test "Container not running"
    fi
}

test_network_configuration() {
    print_header "Network Configuration Tests"

    print_test "module8-network exists"
    if docker network ls | grep -q "module8-network"; then
        pass_test
    else
        fail_test "Network not found"
    fi

    print_test "PostgreSQL IP detection"
    local postgres_ip=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' a2f55d83b1fa 2>/dev/null | head -1)
    if [ -n "$postgres_ip" ]; then
        pass_test
        echo "    IP: $postgres_ip"
    else
        fail_test "Cannot detect IP"
    fi

    print_test "InfluxDB IP detection"
    local influxdb_ip=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 8502fd5d078d 2>/dev/null | head -1)
    if [ -n "$influxdb_ip" ]; then
        pass_test
        echo "    IP: $influxdb_ip"
    else
        fail_test "Cannot detect IP"
    fi
}

test_environment_file() {
    print_header "Environment File Tests"

    print_test ".env.module8.example exists"
    if [ -f "$SCRIPT_DIR/.env.module8.example" ]; then
        pass_test
    else
        fail_test "Example file not found"
    fi

    print_test "Example file has required variables"
    local required_vars=(
        "KAFKA_BOOTSTRAP_SERVERS"
        "KAFKA_SASL_USERNAME"
        "KAFKA_SASL_PASSWORD"
        "POSTGRES_HOST"
        "POSTGRES_PASSWORD"
        "INFLUXDB_URL"
        "NEO4J_URI"
    )

    local all_found=true
    for var in "${required_vars[@]}"; do
        if ! grep -q "^$var=" "$SCRIPT_DIR/.env.module8.example" 2>/dev/null; then
            all_found=false
            break
        fi
    done

    if [ "$all_found" = true ]; then
        pass_test
    else
        fail_test "Missing required variables"
    fi
}

test_docker_compose_syntax() {
    print_header "Docker Compose Syntax Tests"

    print_test "docker-compose.module8-complete.yml syntax"
    if docker-compose -f "$SCRIPT_DIR/docker-compose.module8-complete.yml" config > /dev/null 2>&1; then
        pass_test
    else
        fail_test "Invalid YAML syntax"
    fi

    print_test "All 8 projectors defined"
    local projectors=(
        "postgresql-projector"
        "mongodb-projector"
        "elasticsearch-projector"
        "clickhouse-projector"
        "influxdb-projector"
        "ups-projector"
        "fhir-store-projector"
        "neo4j-graph-projector"
    )

    local all_defined=true
    for projector in "${projectors[@]}"; do
        if ! grep -q "^  $projector:" "$SCRIPT_DIR/docker-compose.module8-complete.yml"; then
            all_defined=false
            break
        fi
    done

    if [ "$all_defined" = true ]; then
        pass_test
    else
        fail_test "Not all projectors defined"
    fi

    print_test "Infrastructure services defined"
    local infra_services=("mongodb" "elasticsearch" "clickhouse" "redis")

    local all_defined=true
    for service in "${infra_services[@]}"; do
        if ! grep -q "^  $service:" "$SCRIPT_DIR/docker-compose.module8-complete.yml"; then
            all_defined=false
            break
        fi
    done

    if [ "$all_defined" = true ]; then
        pass_test
    else
        fail_test "Not all infrastructure services defined"
    fi
}

test_projector_directories() {
    print_header "Projector Directory Tests"

    local projectors=(
        "module8-postgresql-projector"
        "module8-mongodb-projector"
        "module8-elasticsearch-projector"
        "module8-clickhouse-projector"
        "module8-influxdb-projector"
        "module8-ups-projector"
        "module8-fhir-store-projector"
        "module8-neo4j-graph-projector"
    )

    for projector in "${projectors[@]}"; do
        print_test "$projector directory exists"
        if [ -d "$SCRIPT_DIR/$projector" ]; then
            pass_test
        else
            fail_test "Directory not found"
        fi

        print_test "$projector has Dockerfile"
        if [ -f "$SCRIPT_DIR/$projector/Dockerfile" ]; then
            pass_test
        else
            fail_test "Dockerfile not found"
        fi
    done
}

test_shared_module() {
    print_header "Shared Module Tests"

    print_test "module8-shared directory exists"
    if [ -d "$SCRIPT_DIR/module8-shared" ]; then
        pass_test
    else
        fail_test "Directory not found"
    fi

    print_test "module8-shared has __init__.py"
    if [ -f "$SCRIPT_DIR/module8-shared/__init__.py" ]; then
        pass_test
    else
        fail_test "__init__.py not found"
    fi
}

test_documentation() {
    print_header "Documentation Tests"

    print_test "Orchestration guide exists"
    if [ -f "$SCRIPT_DIR/MODULE8_ORCHESTRATION_COMPLETE.md" ]; then
        pass_test
    else
        fail_test "Orchestration guide not found"
    fi

    print_test "Quick reference exists"
    if [ -f "$SCRIPT_DIR/MODULE8_QUICK_REFERENCE.md" ]; then
        pass_test
    else
        fail_test "Quick reference not found"
    fi

    print_test "Orchestration guide has service ports"
    if grep -q "8050.*8057" "$SCRIPT_DIR/MODULE8_ORCHESTRATION_COMPLETE.md" 2>/dev/null; then
        pass_test
    else
        fail_test "Service ports not documented"
    fi
}

# ========================================
# Main Test Execution
# ========================================

main() {
    print_header "🧪 Module 8 Complete Integration Test Suite"

    echo "Starting comprehensive tests..."
    echo ""

    # Run all tests
    test_files_exist
    test_scripts_executable
    test_docker_requirements
    test_external_containers
    test_network_configuration
    test_environment_file
    test_docker_compose_syntax
    test_projector_directories
    test_shared_module
    test_documentation

    # Print summary
    print_header "Test Summary"

    echo "Total Tests Run:    $TESTS_RUN"
    echo -e "Tests Passed:       ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests Failed:       ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✅ All tests passed!${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Configure network: ./configure-network-module8.sh"
        echo "  2. Setup environment: cp .env.module8.example .env.module8"
        echo "  3. Edit credentials: nano .env.module8"
        echo "  4. Start services: ./start-module8-projectors.sh"
        echo "  5. Check health: ./health-check-module8.sh"
        echo ""
        exit 0
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        echo ""
        echo "Please review failed tests and fix issues before proceeding."
        echo ""
        exit 1
    fi
}

# Run tests
main "$@"
