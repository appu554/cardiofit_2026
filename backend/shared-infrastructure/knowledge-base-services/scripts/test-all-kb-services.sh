#!/bin/bash

# Integration Test Script for All KB Services
# This script tests the integration between all Knowledge Base services
# and verifies their health, API endpoints, and inter-service communication

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Service configurations
declare -A SERVICES=(
    ["kb-1"]="http://localhost:8081"
    ["kb-3"]="http://localhost:8083"
    ["kb-4"]="http://localhost:8084"
    ["kb-5"]="http://localhost:8085"
    ["kb-7"]="http://localhost:8087"
)

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE} KB Services Integration Test Suite${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Function to print test result
print_test_result() {
    local test_name="$1"
    local result="$2"
    local details="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✅ PASS${NC}: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ FAIL${NC}: $test_name"
        if [ ! -z "$details" ]; then
            echo -e "   ${YELLOW}Details: $details${NC}"
        fi
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Function to test service health
test_service_health() {
    local service_name="$1"
    local service_url="$2"
    
    echo -e "${BLUE}Testing $service_name Health...${NC}"
    
    # Health check endpoint
    if curl -sf "$service_url/health" > /dev/null 2>&1; then
        print_test_result "$service_name Health Endpoint" "PASS"
        
        # Get health details
        health_response=$(curl -s "$service_url/health")
        if echo "$health_response" | grep -q "healthy\|success"; then
            print_test_result "$service_name Health Status" "PASS"
        else
            print_test_result "$service_name Health Status" "FAIL" "Service reports unhealthy status"
        fi
    else
        print_test_result "$service_name Health Endpoint" "FAIL" "Health endpoint not responding"
        print_test_result "$service_name Health Status" "FAIL" "Cannot reach service"
    fi
    
    # Metrics endpoint
    if curl -sf "$service_url/metrics" > /dev/null 2>&1; then
        print_test_result "$service_name Metrics Endpoint" "PASS"
    else
        print_test_result "$service_name Metrics Endpoint" "FAIL" "Metrics endpoint not responding"
    fi
}

# Function to test KB-1 Drug Rules specific endpoints
test_kb1_endpoints() {
    local service_url="${SERVICES[kb-1]}"
    
    echo -e "${BLUE}Testing KB-1 Drug Rules Specific Endpoints...${NC}"
    
    # Test drug rules retrieval
    if curl -sf "$service_url/v1/items/metformin" > /dev/null 2>&1; then
        print_test_result "KB-1 Drug Rules Query (metformin)" "PASS"
    else
        print_test_result "KB-1 Drug Rules Query (metformin)" "FAIL" "Cannot retrieve drug rules"
    fi
    
    # Test rules validation
    local validation_payload='{
        "content": "[meta]\ndrug_name=\"Test\"\ntherapeutic_class=[\"Test\"]\n[dose_calculation]\nbase_formula=\"100mg\"\nmax_daily_dose=200.0\nmin_daily_dose=50.0\n[safety_verification]\ncontraindications=[]\nwarnings=[]\nprecautions=[]\ninteraction_checks=[]\nlab_monitoring=[]\nmonitoring_requirements=[]\nregional_variations={}",
        "regions": ["US"]
    }'
    
    if curl -sf -X POST "$service_url/v1/validate" \
        -H "Content-Type: application/json" \
        -d "$validation_payload" > /dev/null 2>&1; then
        print_test_result "KB-1 Rules Validation" "PASS"
    else
        print_test_result "KB-1 Rules Validation" "FAIL" "Validation endpoint failed"
    fi
}

# Function to test KB-3 Guidelines specific endpoints
test_kb3_endpoints() {
    local service_url="${SERVICES[kb-3]}"
    
    echo -e "${BLUE}Testing KB-3 Guidelines Specific Endpoints...${NC}"
    
    # Test guidelines retrieval
    if curl -sf "$service_url/api/v1/guidelines" > /dev/null 2>&1; then
        print_test_result "KB-3 Guidelines List" "PASS"
    else
        print_test_result "KB-3 Guidelines List" "FAIL" "Cannot retrieve guidelines list"
    fi
    
    # Test evidence search
    if curl -sf "$service_url/api/v1/evidence/search?query=diabetes" > /dev/null 2>&1; then
        print_test_result "KB-3 Evidence Search" "PASS"
    else
        print_test_result "KB-3 Evidence Search" "FAIL" "Evidence search failed"
    fi
    
    # Test recommendations
    if curl -sf "$service_url/api/v1/recommendations" > /dev/null 2>&1; then
        print_test_result "KB-3 Clinical Recommendations" "PASS"
    else
        print_test_result "KB-3 Clinical Recommendations" "FAIL" "Recommendations endpoint failed"
    fi
}

# Function to test KB-4 Patient Safety specific endpoints
test_kb4_endpoints() {
    local service_url="${SERVICES[kb-4]}"
    
    echo -e "${BLUE}Testing KB-4 Patient Safety Specific Endpoints...${NC}"
    
    # Test alerts retrieval
    if curl -sf "$service_url/api/v1/alerts" > /dev/null 2>&1; then
        print_test_result "KB-4 Safety Alerts List" "PASS"
    else
        print_test_result "KB-4 Safety Alerts List" "FAIL" "Cannot retrieve safety alerts"
    fi
    
    # Test risk assessment
    local risk_payload='{
        "patient_id": "test-patient-123",
        "risk_factors": ["diabetes", "hypertension"],
        "medications": ["metformin", "lisinopril"]
    }'
    
    if curl -sf -X POST "$service_url/api/v1/risk-assessment" \
        -H "Content-Type: application/json" \
        -d "$risk_payload" > /dev/null 2>&1; then
        print_test_result "KB-4 Risk Assessment" "PASS"
    else
        print_test_result "KB-4 Risk Assessment" "FAIL" "Risk assessment failed"
    fi
    
    # Test monitoring rules
    if curl -sf "$service_url/api/v1/monitoring/rules" > /dev/null 2>&1; then
        print_test_result "KB-4 Monitoring Rules" "PASS"
    else
        print_test_result "KB-4 Monitoring Rules" "FAIL" "Monitoring rules endpoint failed"
    fi
}

# Function to test KB-5 Drug Interactions specific endpoints
test_kb5_endpoints() {
    local service_url="${SERVICES[kb-5]}"
    
    echo -e "${BLUE}Testing KB-5 Drug Interactions Specific Endpoints...${NC}"
    
    # Test interaction check
    local interaction_payload='{
        "drug_codes": ["warfarin", "aspirin"],
        "check_type": "comprehensive",
        "patient_id": "test-patient-123"
    }'
    
    if curl -sf -X POST "$service_url/api/v1/interactions/check" \
        -H "Content-Type: application/json" \
        -d "$interaction_payload" > /dev/null 2>&1; then
        print_test_result "KB-5 Interaction Check" "PASS"
    else
        print_test_result "KB-5 Interaction Check" "FAIL" "Interaction check failed"
    fi
    
    # Test quick check
    if curl -sf "$service_url/api/v1/interactions/quick-check?drugs=warfarin,aspirin" > /dev/null 2>&1; then
        print_test_result "KB-5 Quick Interaction Check" "PASS"
    else
        print_test_result "KB-5 Quick Interaction Check" "FAIL" "Quick check failed"
    fi
    
    # Test drug interactions lookup
    if curl -sf "$service_url/api/v1/drugs/warfarin/interactions" > /dev/null 2>&1; then
        print_test_result "KB-5 Drug Interactions Lookup" "PASS"
    else
        print_test_result "KB-5 Drug Interactions Lookup" "FAIL" "Drug lookup failed"
    fi
    
    # Test interaction statistics
    if curl -sf "$service_url/api/v1/interactions/statistics" > /dev/null 2>&1; then
        print_test_result "KB-5 Interaction Statistics" "PASS"
    else
        print_test_result "KB-5 Interaction Statistics" "FAIL" "Statistics endpoint failed"
    fi
}

# Function to test KB-7 Terminology specific endpoints
test_kb7_endpoints() {
    local service_url="${SERVICES[kb-7]}"
    
    echo -e "${BLUE}Testing KB-7 Terminology Specific Endpoints...${NC}"
    
    # Test terminology search
    if curl -sf "$service_url/api/v1/terminology/search?query=diabetes&system=snomed" > /dev/null 2>&1; then
        print_test_result "KB-7 Terminology Search" "PASS"
    else
        print_test_result "KB-7 Terminology Search" "FAIL" "Terminology search failed"
    fi
    
    # Test code validation
    if curl -sf "$service_url/api/v1/terminology/validate/snomed/73211009" > /dev/null 2>&1; then
        print_test_result "KB-7 Code Validation" "PASS"
    else
        print_test_result "KB-7 Code Validation" "FAIL" "Code validation failed"
    fi
    
    # Test mappings
    if curl -sf "$service_url/api/v1/terminology/mappings?from_system=icd10&to_system=snomed" > /dev/null 2>&1; then
        print_test_result "KB-7 Terminology Mappings" "PASS"
    else
        print_test_result "KB-7 Terminology Mappings" "FAIL" "Mappings endpoint failed"
    fi
    
    # Test value sets
    if curl -sf "$service_url/api/v1/terminology/valuesets" > /dev/null 2>&1; then
        print_test_result "KB-7 Value Sets" "PASS"
    else
        print_test_result "KB-7 Value Sets" "FAIL" "Value sets endpoint failed"
    fi
}

# Function to test cross-service integration
test_integration_scenarios() {
    echo -e "${BLUE}Testing Cross-Service Integration Scenarios...${NC}"
    
    # Scenario 1: Drug rule validation with interaction check
    echo -e "${YELLOW}Scenario 1: Drug Rules + Interaction Check${NC}"
    
    # First, check if we can get drug rules from KB-1
    local drug_rules_response=$(curl -s "${SERVICES[kb-1]}/v1/items/warfarin" 2>/dev/null)
    if [ $? -eq 0 ] && [ ! -z "$drug_rules_response" ]; then
        # Then check interactions with KB-5
        if curl -sf "${SERVICES[kb-5]}/api/v1/interactions/quick-check?drugs=warfarin,aspirin" > /dev/null 2>&1; then
            print_test_result "Integration: Drug Rules + Interactions" "PASS"
        else
            print_test_result "Integration: Drug Rules + Interactions" "FAIL" "Interaction check failed"
        fi
    else
        print_test_result "Integration: Drug Rules + Interactions" "FAIL" "Drug rules retrieval failed"
    fi
    
    # Scenario 2: Guidelines with Safety Monitoring
    echo -e "${YELLOW}Scenario 2: Guidelines + Safety Monitoring${NC}"
    
    # Check guidelines from KB-3 and safety alerts from KB-4
    local guidelines_available=false
    local safety_available=false
    
    if curl -sf "${SERVICES[kb-3]}/api/v1/guidelines" > /dev/null 2>&1; then
        guidelines_available=true
    fi
    
    if curl -sf "${SERVICES[kb-4]}/api/v1/alerts" > /dev/null 2>&1; then
        safety_available=true
    fi
    
    if [ "$guidelines_available" = true ] && [ "$safety_available" = true ]; then
        print_test_result "Integration: Guidelines + Safety Monitoring" "PASS"
    else
        print_test_result "Integration: Guidelines + Safety Monitoring" "FAIL" "One or both services unavailable"
    fi
    
    # Scenario 3: Terminology validation across services
    echo -e "${YELLOW}Scenario 3: Cross-Service Terminology Validation${NC}"
    
    # Check if KB-7 can validate terminology used by other services
    if curl -sf "${SERVICES[kb-7]}/api/v1/terminology/validate/snomed/73211009" > /dev/null 2>&1; then
        print_test_result "Integration: Terminology Validation" "PASS"
    else
        print_test_result "Integration: Terminology Validation" "FAIL" "Terminology validation failed"
    fi
}

# Function to test database connectivity
test_database_connectivity() {
    echo -e "${BLUE}Testing Database Connectivity...${NC}"
    
    # Check if services can connect to their databases
    for service in "${!SERVICES[@]}"; do
        local service_url="${SERVICES[$service]}"
        local health_response=$(curl -s "$service_url/health" 2>/dev/null)
        
        if echo "$health_response" | grep -q "database.*healthy\|db.*ok"; then
            print_test_result "$service Database Connection" "PASS"
        else
            print_test_result "$service Database Connection" "FAIL" "Database connection issue"
        fi
    done
}

# Function to test cache connectivity
test_cache_connectivity() {
    echo -e "${BLUE}Testing Cache Connectivity...${NC}"
    
    # Check if services can connect to Redis cache
    for service in "${!SERVICES[@]}"; do
        local service_url="${SERVICES[$service]}"
        local health_response=$(curl -s "$service_url/health" 2>/dev/null)
        
        if echo "$health_response" | grep -q "cache.*healthy\|redis.*ok"; then
            print_test_result "$service Cache Connection" "PASS"
        else
            print_test_result "$service Cache Connection" "FAIL" "Cache connection issue"
        fi
    done
}

# Function to test performance metrics
test_performance_metrics() {
    echo -e "${BLUE}Testing Performance Metrics Collection...${NC}"
    
    for service in "${!SERVICES[@]}"; do
        local service_url="${SERVICES[$service]}"
        local metrics_response=$(curl -s "$service_url/metrics" 2>/dev/null)
        
        if echo "$metrics_response" | grep -q "http_requests_total\|go_"; then
            print_test_result "$service Metrics Collection" "PASS"
        else
            print_test_result "$service Metrics Collection" "FAIL" "No metrics found"
        fi
    done
}

# Main test execution
main() {
    echo -e "${BLUE}Starting comprehensive KB services testing...${NC}"
    echo ""
    
    # Test individual service health
    for service in "${!SERVICES[@]}"; do
        test_service_health "$service" "${SERVICES[$service]}"
        echo ""
    done
    
    # Test service-specific endpoints
    test_kb1_endpoints
    echo ""
    test_kb3_endpoints
    echo ""
    test_kb4_endpoints
    echo ""
    test_kb5_endpoints
    echo ""
    test_kb7_endpoints
    echo ""
    
    # Test integration scenarios
    test_integration_scenarios
    echo ""
    
    # Test infrastructure connectivity
    test_database_connectivity
    echo ""
    test_cache_connectivity
    echo ""
    test_performance_metrics
    echo ""
    
    # Print final results
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE} Test Results Summary${NC}"
    echo -e "${BLUE}================================================${NC}"
    echo -e "Total Tests: $TOTAL_TESTS"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 All tests passed! KB services are running correctly.${NC}"
        exit 0
    else
        echo -e "${RED}⚠️  Some tests failed. Please check the services and try again.${NC}"
        exit 1
    fi
}

# Check if services are running
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}Error: curl is required but not installed.${NC}"
        exit 1
    fi
    
    # Check if any services are running
    local services_running=0
    for service in "${!SERVICES[@]}"; do
        if curl -sf "${SERVICES[$service]}/health" > /dev/null 2>&1; then
            services_running=$((services_running + 1))
        fi
    done
    
    if [ $services_running -eq 0 ]; then
        echo -e "${RED}Error: No KB services are running. Please start the services first.${NC}"
        echo -e "${YELLOW}Run: make run-kb-docker${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Prerequisites check passed. Found $services_running running services.${NC}"
    echo ""
}

# Run the tests
check_prerequisites
main