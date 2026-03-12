#!/bin/bash
# V3 Clinical Guideline Curation Pipeline - Full Production Runner
# Usage: ./run-pipeline.sh [command]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ═══════════════════════════════════════════════════════════════════════════════
# Helper Functions
# ═══════════════════════════════════════════════════════════════════════════════

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_env() {
    if [ ! -f .env ]; then
        log_error ".env file not found!"
        log_info "Copy .env.example to .env and configure your settings:"
        echo "    cp .env.example .env"
        exit 1
    fi

    # Source .env file
    export $(grep -v '^#' .env | xargs)

    if [ -z "$ANTHROPIC_API_KEY" ]; then
        log_error "ANTHROPIC_API_KEY not set in .env"
        exit 1
    fi

    if [ -z "$VAIDSHALA_LOCAL_PATH" ] || [ ! -d "$VAIDSHALA_LOCAL_PATH" ]; then
        log_warn "VAIDSHALA_LOCAL_PATH not set or directory doesn't exist"
        log_info "CQL validation (L5) will be limited"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Commands
# ═══════════════════════════════════════════════════════════════════════════════

cmd_start() {
    log_info "Starting V3 Pipeline Stack..."
    check_env

    log_info "Building Docker images..."
    docker compose build

    log_info "Starting services..."
    docker compose up -d

    log_info "Waiting for services to be healthy..."
    sleep 10

    # Check health
    cmd_health
}

cmd_stop() {
    log_info "Stopping V3 Pipeline Stack..."
    docker compose down
    log_success "All services stopped"
}

cmd_health() {
    log_info "Checking service health..."

    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo "  V3 Clinical Guideline Pipeline - Health Check"
    echo "═══════════════════════════════════════════════════════════════"
    echo ""

    # PostgreSQL
    if docker compose exec -T postgres pg_isready -U v3user -d v3_facts > /dev/null 2>&1; then
        echo -e "  PostgreSQL:    ${GREEN}✅ Healthy${NC}"
    else
        echo -e "  PostgreSQL:    ${RED}❌ Unhealthy${NC}"
    fi

    # Redis
    if docker compose exec -T redis redis-cli ping > /dev/null 2>&1; then
        echo -e "  Redis:         ${GREEN}✅ Healthy${NC}"
    else
        echo -e "  Redis:         ${RED}❌ Unhealthy${NC}"
    fi

    # Snow Owl
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/snowowl/admin/info | grep -q "200"; then
        echo -e "  Snow Owl:      ${GREEN}✅ Healthy${NC}"
    else
        echo -e "  Snow Owl:      ${YELLOW}⏳ Starting (may take 2-3 min)${NC}"
    fi

    # Pipeline
    if docker compose ps v3-pipeline | grep -q "Up"; then
        echo -e "  Pipeline:      ${GREEN}✅ Running${NC}"
    else
        echo -e "  Pipeline:      ${RED}❌ Not Running${NC}"
    fi

    echo ""
}

cmd_logs() {
    local service=${1:-""}
    if [ -z "$service" ]; then
        docker compose logs -f
    else
        docker compose logs -f "$service"
    fi
}

cmd_cli() {
    log_info "Starting Pipeline CLI..."
    check_env
    docker compose run --rm pipeline-cli
}

cmd_extract() {
    local pdf_file=$1
    local target_kb=${2:-"dosing"}

    if [ -z "$pdf_file" ]; then
        log_error "Usage: ./run-pipeline.sh extract <pdf_file> [target_kb]"
        log_info "  target_kb: dosing, safety, or monitoring (default: dosing)"
        exit 1
    fi

    check_env

    log_info "Running extraction pipeline..."
    log_info "  PDF: $pdf_file"
    log_info "  Target KB: $target_kb"

    docker compose exec v3-pipeline python -c "
from guideline_atomiser import MarkerExtractor, KBFactExtractor, create_extractor_from_env
from gliner import ClinicalNERExtractor
from cql import CQLCompatibilityChecker
import json

# L1: PDF Extraction
print('L1: Extracting PDF...')
marker = MarkerExtractor()
pdf_result = marker.extract('/data/pdfs/$pdf_file')
print(f'  Pages: {pdf_result.provenance.total_pages}')
print(f'  Blocks: {len(pdf_result.blocks)}')
print(f'  Tables: {len(pdf_result.tables)}')

# L2: Clinical NER
print('L2: Running clinical NER...')
ner = ClinicalNERExtractor()
ner_result = ner.extract_for_kb(pdf_result.markdown, '$target_kb')
print(f'  Entities: {len(ner_result.entities)}')

# L3: Claude Extraction
print('L3: Extracting structured facts...')
extractor = create_extractor_from_env()
facts = extractor.extract_facts(
    markdown_text=pdf_result.markdown,
    gliner_entities=ner_result.to_gliner_format(),
    target_kb='$target_kb',
    guideline_context={'authority': 'KDIGO'}
)
print(f'  Facts extracted: {len(facts.drugs) if hasattr(facts, \"drugs\") else \"N/A\"}')

# Save output
output_path = '/data/output/${pdf_file%.pdf}_${target_kb}.json'
with open(output_path, 'w') as f:
    json.dump(facts.model_dump(by_alias=True), f, indent=2)
print(f'Output saved to: {output_path}')
"

    log_success "Extraction complete!"
}

cmd_validate() {
    local facts_file=$1
    local cql_file=${2:-"T2DMGuidelines.cql"}

    if [ -z "$facts_file" ]; then
        log_error "Usage: ./run-pipeline.sh validate <facts_file> [cql_file]"
        exit 1
    fi

    log_info "Running L5 CQL Validation..."

    docker compose exec v3-pipeline python -c "
from cql import CQLCompatibilityChecker, CQLGapDetector
import json

with open('/data/output/$facts_file') as f:
    facts = json.load(f)

checker = CQLCompatibilityChecker(
    '/app/cql/registry/cql_guideline_registry.yaml',
    '/data/vaidshala'
)
report = checker.check_compatibility(facts, '$cql_file')

print('L5 Compatibility Report:')
print(f'  Compatible: {report.compatible}')
print(f'  Matches: {len(report.matches)}')
print(f'  Issues: {len(report.issues)}')

if report.issues:
    print('\\nIssues:')
    for issue in report.issues:
        print(f'  - {issue}')
"
}

cmd_terminology() {
    log_info "Loading terminology into Snow Owl..."
    log_warn "This requires manual import of RxNorm, LOINC, SNOMED-CT files"
    log_info "See: https://docs.b2ihealthcare.com/snow-owl/8.x/administration/import"

    echo ""
    echo "Snow Owl Admin UI: http://localhost:8080"
    echo "Username: snowowl"
    echo "Password: (from .env SNOW_OWL_PASSWORD)"
    echo ""
    echo "Download terminology files from:"
    echo "  - RxNorm: https://www.nlm.nih.gov/research/umls/rxnorm/"
    echo "  - LOINC: https://loinc.org/downloads/"
    echo "  - SNOMED-CT: https://www.snomed.org/snomed-ct/"
}

cmd_test() {
    log_info "Running pipeline tests..."
    docker compose exec v3-pipeline pytest /app/ -v --tb=short
}

cmd_shell() {
    log_info "Opening shell in pipeline container..."
    docker compose exec v3-pipeline bash
}

cmd_clean() {
    log_warn "This will remove all containers, volumes, and data!"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker compose down -v --rmi local
        rm -rf data/output/*
        log_success "Cleanup complete"
    else
        log_info "Cancelled"
    fi
}

cmd_help() {
    echo ""
    echo "V3 Clinical Guideline Curation Pipeline"
    echo "========================================"
    echo ""
    echo "Usage: ./run-pipeline.sh [command] [options]"
    echo ""
    echo "Commands:"
    echo "  start         Start all services (PostgreSQL, Redis, Snow Owl, Pipeline)"
    echo "  stop          Stop all services"
    echo "  health        Check health of all services"
    echo "  logs [svc]    View logs (optionally for specific service)"
    echo "  cli           Open interactive CLI in pipeline container"
    echo "  shell         Open bash shell in pipeline container"
    echo ""
    echo "Extraction:"
    echo "  extract <pdf> [kb]    Run full L1-L5 pipeline on a PDF"
    echo "                        kb: dosing, safety, or monitoring"
    echo "  validate <facts> [cql]  Validate facts against CQL"
    echo ""
    echo "Setup:"
    echo "  terminology   Instructions for loading terminology into Snow Owl"
    echo "  test          Run pipeline tests"
    echo "  clean         Remove all containers and data"
    echo ""
    echo "Examples:"
    echo "  ./run-pipeline.sh start"
    echo "  ./run-pipeline.sh extract kdigo_2022.pdf dosing"
    echo "  ./run-pipeline.sh validate kdigo_2022_dosing.json T2DMGuidelines.cql"
    echo ""
}

# ═══════════════════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════════════════

case "${1:-help}" in
    start)      cmd_start ;;
    stop)       cmd_stop ;;
    health)     cmd_health ;;
    logs)       cmd_logs "$2" ;;
    cli)        cmd_cli ;;
    shell)      cmd_shell ;;
    extract)    cmd_extract "$2" "$3" ;;
    validate)   cmd_validate "$2" "$3" ;;
    terminology) cmd_terminology ;;
    test)       cmd_test ;;
    clean)      cmd_clean ;;
    help|--help|-h) cmd_help ;;
    *)
        log_error "Unknown command: $1"
        cmd_help
        exit 1
        ;;
esac
