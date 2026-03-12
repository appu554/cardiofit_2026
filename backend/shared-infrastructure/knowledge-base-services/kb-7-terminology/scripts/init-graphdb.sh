#!/bin/bash

# GraphDB Repository Initialization Script for KB-7 Terminology Service
# This script creates and configures GraphDB repositories with OWL 2 RL reasoning

set -euo pipefail

# Configuration
GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"
TERMINOLOGY_REPO="${TERMINOLOGY_REPO:-kb7-terminology}"
ONTOLOGY_REPO="${ONTOLOGY_REPO:-kb7-ontologies}"
ENABLE_OWL2RL="${ENABLE_OWL2RL:-true}"
BATCH_SIZE="${BATCH_SIZE:-10000}"
MAX_RETRIES=30
RETRY_DELAY=10

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Wait for GraphDB to be ready
wait_for_graphdb() {
    log_info "Waiting for GraphDB to be ready at ${GRAPHDB_URL}..."

    for i in $(seq 1 $MAX_RETRIES); do
        if curl -s -f "${GRAPHDB_URL}/rest/repositories" > /dev/null 2>&1; then
            log_success "GraphDB is ready!"
            return 0
        fi

        log_warning "GraphDB not ready yet (attempt $i/$MAX_RETRIES). Waiting ${RETRY_DELAY}s..."
        sleep $RETRY_DELAY
    done

    log_error "GraphDB failed to become ready after $MAX_RETRIES attempts"
    exit 1
}

# Check if repository exists
repository_exists() {
    local repo_id="$1"
    curl -s -f "${GRAPHDB_URL}/rest/repositories/${repo_id}" > /dev/null 2>&1
}

# Create repository with OWL 2 RL reasoning
create_repository() {
    local repo_id="$1"
    local repo_title="$2"
    local ruleset="$3"

    log_info "Creating repository: ${repo_id} with ruleset: ${ruleset}"

    # Repository configuration JSON
    local config=$(cat <<EOF
{
    "id": "${repo_id}",
    "title": "${repo_title}",
    "type": "graphdb",
    "params": {
        "imports": "",
        "defaultNS": "",
        "repositoryType": "file-repository",
        "id": "${repo_id}",
        "title": "${repo_title}",
        "ruleset": "${ruleset}",
        "disableSameAs": false,
        "checkForInconsistencies": true,
        "enableContextIndex": true,
        "cacheSelectNodes": true,
        "entityIndexSize": 10000000,
        "entityIdSize": 32,
        "predicateMemory": 64,
        "ftsMemory": 0,
        "ftsIndexPolicy": "never",
        "ftsLiteralsOnly": true,
        "storageFolder": "storage",
        "enablePredicateList": true,
        "enableLiteralIndex": true,
        "indexCompressionRatio": -1,
        "enableRdfRank": false,
        "inMemoryLiteralProperties": true,
        "throwQueryEvaluationExceptionOnTimeout": true,
        "queryTimeout": 0,
        "queryLimitResults": 0,
        "readOnly": false,
        "enableGeoIndex": false,
        "enableOptimalIndex": true,
        "transactionMode": "safe",
        "transactionIsolation": true,
        "queryReportMode": "off"
    }
}
EOF
    )

    # Create repository
    local response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "${config}" \
        "${GRAPHDB_URL}/rest/repositories")

    local http_code="${response: -3}"
    local body="${response%???}"

    if [[ "$http_code" == "201" ]] || [[ "$http_code" == "200" ]]; then
        log_success "Repository ${repo_id} created successfully"
    else
        log_error "Failed to create repository ${repo_id}. HTTP Code: ${http_code}, Response: ${body}"
        return 1
    fi
}

# Load initial ontologies and terminologies
load_initial_data() {
    local repo_id="$1"

    log_info "Loading initial data into repository: ${repo_id}"

    # Define base ontologies and terminologies to load
    local base_ontologies=(
        "http://www.w3.org/2004/02/skos/core"
        "http://purl.obolibrary.org/obo/iao.owl"
        "http://www.w3.org/2002/07/owl"
    )

    # Load FHIR terminology ontologies if available
    if [[ -d "/import/ontologies" ]]; then
        log_info "Loading FHIR terminology ontologies..."
        for ontology_file in /import/ontologies/*.{owl,ttl,rdf}; do
            if [[ -f "$ontology_file" ]]; then
                load_rdf_file "$repo_id" "$ontology_file"
            fi
        done
    fi

    # Load terminology data if available
    if [[ -d "/import/terminology" ]]; then
        log_info "Loading terminology data..."
        for data_file in /import/terminology/*.{ttl,nt,rdf,jsonld}; do
            if [[ -f "$data_file" ]]; then
                load_rdf_file "$repo_id" "$data_file"
            fi
        done
    fi
}

# Load individual RDF file
load_rdf_file() {
    local repo_id="$1"
    local file_path="$2"
    local filename=$(basename "$file_path")
    local extension="${filename##*.}"

    # Determine content type based on file extension
    local content_type
    case "$extension" in
        "owl"|"rdf"|"xml")
            content_type="application/rdf+xml"
            ;;
        "ttl"|"turtle")
            content_type="text/turtle"
            ;;
        "nt"|"ntriples")
            content_type="application/n-triples"
            ;;
        "jsonld"|"json-ld")
            content_type="application/ld+json"
            ;;
        *)
            log_warning "Unknown file extension: $extension. Assuming Turtle format."
            content_type="text/turtle"
            ;;
    esac

    log_info "Loading file: ${filename} (${content_type})"

    # Load the file
    local response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: ${content_type}" \
        --data-binary "@${file_path}" \
        "${GRAPHDB_URL}/repositories/${repo_id}/statements")

    local http_code="${response: -3}"

    if [[ "$http_code" == "200" ]] || [[ "$http_code" == "204" ]]; then
        log_success "Loaded file: ${filename}"
    else
        log_error "Failed to load file: ${filename}. HTTP Code: ${http_code}"
    fi
}

# Create SPARQL queries for initial setup
create_initial_queries() {
    local repo_id="$1"

    log_info "Creating initial SPARQL queries and prefixes for ${repo_id}"

    # Set up common prefixes
    local prefixes_query=$(cat <<'EOF'
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>
PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
PREFIX fhir: <http://hl7.org/fhir/>
PREFIX sct: <http://snomed.info/sct/>
PREFIX rxnorm: <http://purl.bioontology.org/ontology/RXNORM/>
PREFIX loinc: <http://loinc.org/>
PREFIX icd10: <http://hl7.org/fhir/sid/icd-10/>
PREFIX amt: <http://ns.electronichealth.net.au/id/amt/>

INSERT DATA {
    <http://kb7.terminology/graph/metadata> {
        <http://kb7.terminology/metadata> a owl:Ontology ;
            rdfs:label "KB-7 Terminology Service Metadata" ;
            rdfs:comment "Metadata and configuration for KB-7 Terminology Service" ;
            owl:versionInfo "1.0.0" .
    }
}
EOF
    )

    # Execute prefixes query
    execute_sparql_update "$repo_id" "$prefixes_query"

    # Create indexes for better performance
    create_custom_indexes "$repo_id"
}

# Execute SPARQL UPDATE query
execute_sparql_update() {
    local repo_id="$1"
    local query="$2"

    local response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "update=$(printf '%s' "$query" | jq -sRr @uri)" \
        "${GRAPHDB_URL}/repositories/${repo_id}/statements")

    local http_code="${response: -3}"

    if [[ "$http_code" == "200" ]] || [[ "$http_code" == "204" ]]; then
        log_success "SPARQL update executed successfully"
    else
        log_error "Failed to execute SPARQL update. HTTP Code: ${http_code}"
    fi
}

# Create custom indexes for terminology queries
create_custom_indexes() {
    local repo_id="$1"

    log_info "Creating custom indexes for terminology queries in ${repo_id}"

    # This would typically be done through GraphDB's management interface
    # For now, we'll use SPARQL to create some useful inference rules

    local inference_rules=$(cat <<'EOF'
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

INSERT {
    ?concept skos:inScheme ?scheme .
} WHERE {
    ?concept a skos:Concept .
    ?scheme a skos:ConceptScheme .
    ?concept skos:topConceptOf ?scheme .
}
EOF
    )

    execute_sparql_update "$repo_id" "$inference_rules"
}

# Verify repository health and performance
verify_repository() {
    local repo_id="$1"

    log_info "Verifying repository health: ${repo_id}"

    # Test basic query
    local test_query="SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
    local response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -H "Accept: application/sparql-results+json" \
        -d "query=${test_query}" \
        "${GRAPHDB_URL}/repositories/${repo_id}")

    local http_code="${response: -3}"
    local body="${response%???}"

    if [[ "$http_code" == "200" ]]; then
        local count=$(echo "$body" | jq -r '.results.bindings[0].count.value' 2>/dev/null || echo "unknown")
        log_success "Repository ${repo_id} is healthy. Total triples: ${count}"
    else
        log_error "Repository health check failed for ${repo_id}"
        return 1
    fi

    # Test reasoning capabilities if OWL 2 RL is enabled
    if [[ "$ENABLE_OWL2RL" == "true" ]]; then
        verify_reasoning "$repo_id"
    fi
}

# Verify OWL 2 RL reasoning capabilities
verify_reasoning() {
    local repo_id="$1"

    log_info "Verifying OWL 2 RL reasoning capabilities for ${repo_id}"

    # Insert test data for reasoning
    local test_data=$(cat <<'EOF'
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>
PREFIX test: <http://kb7.terminology/test/>

INSERT DATA {
    test:Drug rdfs:subClassOf test:MedicalProduct .
    test:Aspirin a test:Drug .
}
EOF
    )

    execute_sparql_update "$repo_id" "$test_data"

    # Query with inference to see if reasoning works
    local reasoning_query="PREFIX test: <http://kb7.terminology/test/> SELECT ?type WHERE { test:Aspirin a ?type }"
    local response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -H "Accept: application/sparql-results+json" \
        -d "query=${reasoning_query}&infer=true" \
        "${GRAPHDB_URL}/repositories/${repo_id}")

    local http_code="${response: -3}"
    local body="${response%???}"

    if [[ "$http_code" == "200" ]]; then
        # Check if inferred type is present
        if echo "$body" | grep -q "MedicalProduct"; then
            log_success "OWL 2 RL reasoning is working correctly"
        else
            log_warning "OWL 2 RL reasoning may not be working as expected"
        fi
    fi

    # Clean up test data
    local cleanup_query=$(cat <<'EOF'
PREFIX test: <http://kb7.terminology/test/>
DELETE DATA {
    test:Drug rdfs:subClassOf test:MedicalProduct .
    test:Aspirin a test:Drug .
}
EOF
    )
    execute_sparql_update "$repo_id" "$cleanup_query"
}

# Setup monitoring and health checks
setup_monitoring() {
    log_info "Setting up monitoring and health check endpoints"

    # Create health check queries for each repository
    local repos=("$TERMINOLOGY_REPO" "$ONTOLOGY_REPO")

    for repo in "${repos[@]}"; do
        if repository_exists "$repo"; then
            log_info "Setting up monitoring for repository: $repo"
            # This could include setting up specific health check queries
            # or configuring GraphDB's monitoring features
        fi
    done
}

# Main execution
main() {
    log_info "Starting GraphDB initialization for KB-7 Terminology Service"
    log_info "GraphDB URL: ${GRAPHDB_URL}"
    log_info "Terminology Repository: ${TERMINOLOGY_REPO}"
    log_info "Ontology Repository: ${ONTOLOGY_REPO}"
    log_info "OWL 2 RL Reasoning: ${ENABLE_OWL2RL}"

    # Wait for GraphDB to be ready
    wait_for_graphdb

    # Create terminology repository
    if repository_exists "$TERMINOLOGY_REPO"; then
        log_warning "Terminology repository ${TERMINOLOGY_REPO} already exists"
    else
        if [[ "$ENABLE_OWL2RL" == "true" ]]; then
            create_repository "$TERMINOLOGY_REPO" "KB-7 Terminology Repository" "owl2-rl-optimized"
        else
            create_repository "$TERMINOLOGY_REPO" "KB-7 Terminology Repository" "empty"
        fi
    fi

    # Create ontology repository
    if repository_exists "$ONTOLOGY_REPO"; then
        log_warning "Ontology repository ${ONTOLOGY_REPO} already exists"
    else
        if [[ "$ENABLE_OWL2RL" == "true" ]]; then
            create_repository "$ONTOLOGY_REPO" "KB-7 Ontology Repository" "owl2-rl-optimized"
        else
            create_repository "$ONTOLOGY_REPO" "KB-7 Ontology Repository" "rdfs-plus-optimized"
        fi
    fi

    # Load initial data
    load_initial_data "$TERMINOLOGY_REPO"
    load_initial_data "$ONTOLOGY_REPO"

    # Create initial queries and setup
    create_initial_queries "$TERMINOLOGY_REPO"
    create_initial_queries "$ONTOLOGY_REPO"

    # Verify repositories
    verify_repository "$TERMINOLOGY_REPO"
    verify_repository "$ONTOLOGY_REPO"

    # Setup monitoring
    setup_monitoring

    log_success "GraphDB initialization completed successfully!"
    log_info "Terminology Repository: ${GRAPHDB_URL}/repository/${TERMINOLOGY_REPO}"
    log_info "Ontology Repository: ${GRAPHDB_URL}/repository/${ONTOLOGY_REPO}"
    log_info "GraphDB Workbench: ${GRAPHDB_URL}"
}

# Handle script interruption
trap 'log_error "Script interrupted"; exit 1' INT TERM

# Execute main function
main "$@"