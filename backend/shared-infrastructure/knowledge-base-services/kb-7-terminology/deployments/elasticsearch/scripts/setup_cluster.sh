#!/bin/bash

# Setup Elasticsearch Cluster for KB7 Terminology Service
# Phase 4.1 - Week 1.2: Create clinical_terms index with medical analyzers

set -e

ELASTICSEARCH_URL=${ELASTICSEARCH_URL:-"http://localhost:9200"}
CONFIG_DIR="$(dirname "$0")/../config"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Wait for Elasticsearch to be ready
wait_for_elasticsearch() {
    log_info "Waiting for Elasticsearch to be ready at $ELASTICSEARCH_URL..."

    for i in {1..60}; do
        if curl -s "$ELASTICSEARCH_URL/_cluster/health" > /dev/null 2>&1; then
            log_success "Elasticsearch is ready!"
            return 0
        fi
        echo -n "."
        sleep 2
    done

    log_error "Elasticsearch failed to start within 120 seconds"
    exit 1
}

# Check cluster health
check_cluster_health() {
    log_info "Checking cluster health..."

    HEALTH=$(curl -s "$ELASTICSEARCH_URL/_cluster/health?wait_for_status=yellow&timeout=30s")
    STATUS=$(echo $HEALTH | jq -r '.status')

    case $STATUS in
        "green")
            log_success "Cluster health: GREEN - All shards are allocated"
            ;;
        "yellow")
            log_warning "Cluster health: YELLOW - Some replica shards are not allocated"
            ;;
        "red")
            log_error "Cluster health: RED - Some primary shards are not allocated"
            exit 1
            ;;
        *)
            log_error "Unable to determine cluster health status"
            exit 1
            ;;
    esac
}

# Create index template
create_index_template() {
    log_info "Creating index template for clinical terminology..."

    if [ ! -f "$CONFIG_DIR/index_templates.json" ]; then
        log_error "Index template file not found: $CONFIG_DIR/index_templates.json"
        exit 1
    fi

    RESPONSE=$(curl -s -w "%{http_code}" -X PUT \
        "$ELASTICSEARCH_URL/_index_template/clinical_terminology_template" \
        -H "Content-Type: application/json" \
        -d "@$CONFIG_DIR/index_templates.json")

    HTTP_CODE="${RESPONSE: -3}"

    if [ "$HTTP_CODE" -eq 200 ] || [ "$HTTP_CODE" -eq 201 ]; then
        log_success "Index template created successfully"
    else
        log_error "Failed to create index template. HTTP Code: $HTTP_CODE"
        echo "${RESPONSE%???}"
        exit 1
    fi
}

# Create clinical_terms index
create_clinical_terms_index() {
    log_info "Creating clinical_terms index..."

    if [ ! -f "$CONFIG_DIR/clinical_terms_mapping.json" ]; then
        log_error "Clinical terms mapping file not found: $CONFIG_DIR/clinical_terms_mapping.json"
        exit 1
    fi

    # Check if index already exists
    if curl -s -f "$ELASTICSEARCH_URL/clinical_terms" > /dev/null 2>&1; then
        log_warning "Index 'clinical_terms' already exists"
        read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Deleting existing index..."
            curl -s -X DELETE "$ELASTICSEARCH_URL/clinical_terms"
            log_success "Existing index deleted"
        else
            log_info "Skipping index creation"
            return 0
        fi
    fi

    RESPONSE=$(curl -s -w "%{http_code}" -X PUT \
        "$ELASTICSEARCH_URL/clinical_terms" \
        -H "Content-Type: application/json" \
        -d "@$CONFIG_DIR/clinical_terms_mapping.json")

    HTTP_CODE="${RESPONSE: -3}"

    if [ "$HTTP_CODE" -eq 200 ] || [ "$HTTP_CODE" -eq 201 ]; then
        log_success "Clinical terms index created successfully"
    else
        log_error "Failed to create clinical terms index. HTTP Code: $HTTP_CODE"
        echo "${RESPONSE%???}"
        exit 1
    fi
}

# Test analyzers
test_analyzers() {
    log_info "Testing medical analyzers..."

    # Test medical_standard analyzer
    log_info "Testing medical_standard analyzer with 'myocardial infarction'..."
    RESPONSE=$(curl -s -X POST "$ELASTICSEARCH_URL/clinical_terms/_analyze" \
        -H "Content-Type: application/json" \
        -d '{
            "analyzer": "medical_standard",
            "text": "myocardial infarction"
        }')

    if echo "$RESPONSE" | jq -e '.tokens' > /dev/null 2>&1; then
        log_success "Medical standard analyzer working correctly"
        echo "$RESPONSE" | jq -r '.tokens[] | "  Token: \(.token), Type: \(.type)"'
    else
        log_error "Medical standard analyzer test failed"
        echo "$RESPONSE"
    fi

    # Test medical_search analyzer with synonym
    log_info "Testing medical_search analyzer with 'heart attack'..."
    RESPONSE=$(curl -s -X POST "$ELASTICSEARCH_URL/clinical_terms/_analyze" \
        -H "Content-Type: application/json" \
        -d '{
            "analyzer": "medical_search",
            "text": "heart attack"
        }')

    if echo "$RESPONSE" | jq -e '.tokens' > /dev/null 2>&1; then
        log_success "Medical search analyzer working correctly"
        echo "$RESPONSE" | jq -r '.tokens[] | "  Token: \(.token), Type: \(.type)"'
    else
        log_error "Medical search analyzer test failed"
        echo "$RESPONSE"
    fi

    # Test medical_autocomplete analyzer
    log_info "Testing medical_autocomplete analyzer with 'cardio'..."
    RESPONSE=$(curl -s -X POST "$ELASTICSEARCH_URL/clinical_terms/_analyze" \
        -H "Content-Type: application/json" \
        -d '{
            "analyzer": "medical_autocomplete",
            "text": "cardiology"
        }')

    if echo "$RESPONSE" | jq -e '.tokens' > /dev/null 2>&1; then
        log_success "Medical autocomplete analyzer working correctly"
        echo "$RESPONSE" | jq -r '.tokens[] | "  Token: \(.token), Position: \(.position)"'
    else
        log_error "Medical autocomplete analyzer test failed"
        echo "$RESPONSE"
    fi
}

# Create sample data for testing
create_sample_data() {
    log_info "Creating sample clinical terminology data..."

    # Sample SNOMED CT terms
    SAMPLE_DATA='[
        {
            "index": {
                "_index": "clinical_terms",
                "_id": "sct_22298006"
            }
        },
        {
            "term_id": "22298006",
            "concept_id": "22298006",
            "term": "Myocardial infarction",
            "preferred_term": "Myocardial infarction",
            "synonyms": ["Heart attack", "MI", "Cardiac infarction"],
            "definition": "Irreversible necrosis of heart muscle secondary to prolonged ischemia",
            "terminology_system": "SNOMED_CT",
            "terminology_version": "20240301",
            "semantic_tags": ["disorder"],
            "clinical_domain": "cardiology",
            "status": "active",
            "effective_date": "2024-01-01",
            "complexity_score": 0.8,
            "usage_frequency": 15670,
            "last_updated": "2024-01-15T10:30:00Z",
            "fhir_mappings": {
                "code": "22298006",
                "system": "http://snomed.info/sct",
                "display": "Myocardial infarction",
                "version": "20240301"
            }
        },
        {
            "index": {
                "_index": "clinical_terms",
                "_id": "sct_38341003"
            }
        },
        {
            "term_id": "38341003",
            "concept_id": "38341003",
            "term": "Hypertensive disorder",
            "preferred_term": "Hypertensive disorder",
            "synonyms": ["High blood pressure", "Hypertension", "HTN"],
            "definition": "Condition characterized by elevated arterial blood pressure",
            "terminology_system": "SNOMED_CT",
            "terminology_version": "20240301",
            "semantic_tags": ["disorder"],
            "clinical_domain": "cardiology",
            "status": "active",
            "effective_date": "2024-01-01",
            "complexity_score": 0.6,
            "usage_frequency": 23890,
            "last_updated": "2024-01-15T10:30:00Z",
            "fhir_mappings": {
                "code": "38341003",
                "system": "http://snomed.info/sct",
                "display": "Hypertensive disorder",
                "version": "20240301"
            }
        },
        {
            "index": {
                "_index": "clinical_terms",
                "_id": "rxnorm_313782"
            }
        },
        {
            "term_id": "313782",
            "concept_id": "313782",
            "term": "Aspirin 81 MG Oral Tablet",
            "preferred_term": "Aspirin 81 MG Oral Tablet",
            "synonyms": ["Baby aspirin", "Low-dose aspirin", "Aspirin 81mg"],
            "definition": "Low-dose aspirin tablet for cardiovascular protection",
            "terminology_system": "RXNORM",
            "terminology_version": "20240301",
            "semantic_tags": ["medication"],
            "clinical_domain": "pharmacy",
            "status": "active",
            "effective_date": "2024-01-01",
            "complexity_score": 0.4,
            "usage_frequency": 45230,
            "last_updated": "2024-01-15T10:30:00Z",
            "fhir_mappings": {
                "code": "313782",
                "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                "display": "Aspirin 81 MG Oral Tablet",
                "version": "20240301"
            }
        }
    ]'

    RESPONSE=$(curl -s -w "%{http_code}" -X POST \
        "$ELASTICSEARCH_URL/_bulk" \
        -H "Content-Type: application/x-ndjson" \
        -d "$SAMPLE_DATA")

    HTTP_CODE="${RESPONSE: -3}"

    if [ "$HTTP_CODE" -eq 200 ] || [ "$HTTP_CODE" -eq 201 ]; then
        log_success "Sample data created successfully"

        # Force refresh to make data searchable immediately
        curl -s -X POST "$ELASTICSEARCH_URL/clinical_terms/_refresh" > /dev/null

        # Show document count
        sleep 1
        COUNT=$(curl -s "$ELASTICSEARCH_URL/clinical_terms/_count" | jq -r '.count')
        log_info "Documents in clinical_terms index: $COUNT"
    else
        log_error "Failed to create sample data. HTTP Code: $HTTP_CODE"
        echo "${RESPONSE%???}"
    fi
}

# Test search functionality
test_search() {
    log_info "Testing search functionality..."

    # Test exact term search
    log_info "Testing exact term search for 'Myocardial infarction'..."
    RESPONSE=$(curl -s -X POST "$ELASTICSEARCH_URL/clinical_terms/_search" \
        -H "Content-Type: application/json" \
        -d '{
            "query": {
                "match": {
                    "term": "Myocardial infarction"
                }
            }
        }')

    HITS=$(echo "$RESPONSE" | jq -r '.hits.total.value')
    log_info "Found $HITS results for exact term search"

    # Test synonym search
    log_info "Testing synonym search for 'heart attack'..."
    RESPONSE=$(curl -s -X POST "$ELASTICSEARCH_URL/clinical_terms/_search" \
        -H "Content-Type: application/json" \
        -d '{
            "query": {
                "match": {
                    "term": "heart attack"
                }
            }
        }')

    HITS=$(echo "$RESPONSE" | jq -r '.hits.total.value')
    log_info "Found $HITS results for synonym search"

    if [ "$HITS" -gt 0 ]; then
        log_success "Synonym search is working correctly"
    else
        log_warning "Synonym search may not be working as expected"
    fi

    # Test autocomplete search
    log_info "Testing autocomplete search for 'cardio'..."
    RESPONSE=$(curl -s -X POST "$ELASTICSEARCH_URL/clinical_terms/_search" \
        -H "Content-Type: application/json" \
        -d '{
            "query": {
                "match": {
                    "term.autocomplete": "cardio"
                }
            }
        }')

    HITS=$(echo "$RESPONSE" | jq -r '.hits.total.value')
    log_info "Found $HITS results for autocomplete search"
}

# Main execution
main() {
    log_info "Starting Elasticsearch cluster setup for KB7 Terminology Service"
    log_info "Target URL: $ELASTICSEARCH_URL"

    # Check if jq is installed
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed. Please install jq first."
        exit 1
    fi

    wait_for_elasticsearch
    check_cluster_health
    create_index_template
    create_clinical_terms_index
    test_analyzers
    create_sample_data
    test_search

    log_success "Elasticsearch cluster setup completed successfully!"
    log_info "Cluster endpoints:"
    log_info "  - Elasticsearch: $ELASTICSEARCH_URL"
    log_info "  - Kibana: http://localhost:5601"
    log_info "  - Elasticsearch Head: http://localhost:9100"
    log_info "  - Prometheus Metrics: http://localhost:9114/metrics"
}

# Run main function
main "$@"