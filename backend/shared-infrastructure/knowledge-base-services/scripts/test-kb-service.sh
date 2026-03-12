#!/bin/bash

# Test script for KB-Drug-Rules service with local PostgreSQL
# This script tests the complete setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
KB_SERVICE_URL="http://localhost:8081"
MAX_RETRIES=30
RETRY_DELAY=2

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Wait for service to be ready
wait_for_service() {
    print_status "Waiting for KB-Drug-Rules service to be ready..."
    
    for i in $(seq 1 $MAX_RETRIES); do
        if curl -s "$KB_SERVICE_URL/health" >/dev/null 2>&1; then
            print_success "Service is ready!"
            return 0
        fi
        
        if [ $i -lt $MAX_RETRIES ]; then
            echo "  Attempt $i/$MAX_RETRIES - retrying in ${RETRY_DELAY}s..."
            sleep $RETRY_DELAY
        fi
    done
    
    print_error "Service not ready after $((MAX_RETRIES * RETRY_DELAY)) seconds"
    return 1
}

# Test health endpoint
test_health() {
    print_status "Testing health endpoint..."
    
    response=$(curl -s "$KB_SERVICE_URL/health")
    status=$(echo "$response" | jq -r '.status' 2>/dev/null || echo "unknown")
    
    if [ "$status" = "healthy" ]; then
        print_success "Health check passed"
        
        # Check database status
        db_status=$(echo "$response" | jq -r '.checks.database' 2>/dev/null || echo "unknown")
        print_status "Database status: $db_status"
        
        # Check cache status
        cache_status=$(echo "$response" | jq -r '.checks.cache' 2>/dev/null || echo "unknown")
        print_status "Cache status: $cache_status"
        
        return 0
    else
        print_error "Health check failed: $status"
        echo "Response: $response"
        return 1
    fi
}

# Test drug rules retrieval
test_drug_rules() {
    print_status "Testing drug rules retrieval..."
    
    local drugs=("metformin" "lisinopril" "warfarin")
    local success_count=0
    
    for drug in "${drugs[@]}"; do
        print_status "Testing $drug..."
        
        response=$(curl -s "$KB_SERVICE_URL/v1/items/$drug")
        drug_name=$(echo "$response" | jq -r '.content.meta.drug_name' 2>/dev/null || echo "")
        
        if [ -n "$drug_name" ] && [ "$drug_name" != "null" ]; then
            print_success "Retrieved $drug: $drug_name"
            
            # Show some details
            version=$(echo "$response" | jq -r '.version' 2>/dev/null || echo "unknown")
            regions=$(echo "$response" | jq -r '.regions | join(", ")' 2>/dev/null || echo "unknown")
            signature_valid=$(echo "$response" | jq -r '.signature_valid' 2>/dev/null || echo "unknown")
            
            echo "  Version: $version"
            echo "  Regions: $regions"
            echo "  Signature Valid: $signature_valid"
            
            ((success_count++))
        else
            print_warning "Failed to retrieve $drug"
            echo "Response: $response"
        fi
        echo ""
    done
    
    if [ $success_count -eq ${#drugs[@]} ]; then
        print_success "All drug rules retrieved successfully"
        return 0
    else
        print_warning "Retrieved $success_count/${#drugs[@]} drug rules"
        return 1
    fi
}

# Test rule validation
test_validation() {
    print_status "Testing rule validation..."
    
    # Sample TOML for validation
    local toml_content='[meta]
drug_name = "Test Drug"
therapeutic_class = ["Test Class"]
evidence_sources = ["Test Guidelines 2024"]
last_major_update = "2024-01-01T00:00:00Z"
update_rationale = "Test validation"

[dose_calculation]
base_formula = "100mg daily"
max_daily_dose = 200.0
min_daily_dose = 50.0

[[dose_calculation.adjustment_factors]]
factor = "age"
condition = "age > 65"
multiplier = 0.8

[safety_verification]
contraindications = []
warnings = []
precautions = []
interaction_checks = []
lab_monitoring = []

monitoring_requirements = []
regional_variations = {}'
    
    # Create JSON payload
    local json_payload=$(jq -n \
        --arg content "$toml_content" \
        --argjson regions '["US"]' \
        '{content: $content, regions: $regions}')
    
    response=$(curl -s -X POST "$KB_SERVICE_URL/v1/validate" \
        -H "Content-Type: application/json" \
        -d "$json_payload")
    
    valid=$(echo "$response" | jq -r '.valid' 2>/dev/null || echo "false")
    
    if [ "$valid" = "true" ]; then
        print_success "Rule validation passed"
        
        errors=$(echo "$response" | jq -r '.errors | length' 2>/dev/null || echo "0")
        warnings=$(echo "$response" | jq -r '.warnings | length' 2>/dev/null || echo "0")
        
        echo "  Errors: $errors"
        echo "  Warnings: $warnings"
        
        return 0
    else
        print_error "Rule validation failed"
        echo "Response: $response"
        return 1
    fi
}

# Test metrics endpoint
test_metrics() {
    print_status "Testing metrics endpoint..."
    
    response=$(curl -s "$KB_SERVICE_URL/metrics")
    
    if echo "$response" | grep -q "kb_"; then
        metric_count=$(echo "$response" | grep -c "kb_" || echo "0")
        print_success "Metrics endpoint working: $metric_count KB metrics found"
        return 0
    else
        print_warning "Metrics endpoint not working or no KB metrics found"
        return 1
    fi
}

# Main test function
main() {
    echo "🧪 KB-Drug-Rules Service Test Suite"
    echo "===================================="
    echo ""
    
    local test_results=()
    
    # Run tests
    if wait_for_service; then
        test_results+=("wait_for_service:PASS")
    else
        test_results+=("wait_for_service:FAIL")
        print_error "Service not available, skipping other tests"
        exit 1
    fi
    
    echo ""
    if test_health; then
        test_results+=("health:PASS")
    else
        test_results+=("health:FAIL")
    fi
    
    echo ""
    if test_drug_rules; then
        test_results+=("drug_rules:PASS")
    else
        test_results+=("drug_rules:FAIL")
    fi
    
    echo ""
    if test_validation; then
        test_results+=("validation:PASS")
    else
        test_results+=("validation:FAIL")
    fi
    
    echo ""
    if test_metrics; then
        test_results+=("metrics:PASS")
    else
        test_results+=("metrics:FAIL")
    fi
    
    # Summary
    echo ""
    echo "===================================="
    echo "📊 Test Results Summary"
    echo "===================================="
    
    local pass_count=0
    local total_count=${#test_results[@]}
    
    for result in "${test_results[@]}"; do
        test_name=$(echo "$result" | cut -d: -f1)
        test_status=$(echo "$result" | cut -d: -f2)
        
        if [ "$test_status" = "PASS" ]; then
            print_success "$test_name: PASSED"
            ((pass_count++))
        else
            print_error "$test_name: FAILED"
        fi
    done
    
    echo ""
    if [ $pass_count -eq $total_count ]; then
        print_success "All tests passed! ($pass_count/$total_count)"
        echo ""
        echo "🎉 Your KB-Drug-Rules service is working perfectly!"
        echo ""
        echo "Next steps:"
        echo "  1. Integrate with your Flow2 orchestrator"
        echo "  2. Add more drug rules via the API"
        echo "  3. Set up monitoring and alerting"
        echo "  4. Deploy to production"
        
        exit 0
    else
        print_error "Some tests failed ($pass_count/$total_count passed)"
        echo ""
        echo "Please check the service logs and configuration."
        exit 1
    fi
}

# Check dependencies
if ! command -v curl >/dev/null 2>&1; then
    print_error "curl is required but not installed"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    print_warning "jq is not installed - some output will be less detailed"
fi

# Run main function
main "$@"
