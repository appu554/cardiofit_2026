#!/bin/bash

# KB-7 Terminology Service - Data Loading Script
# Loads SNOMED CT, RxNorm, and LOINC datasets into the terminology database

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$SCRIPT_DIR/data"
ETL_CMD="$SCRIPT_DIR/cmd/etl/main.go"

# Default values
SYSTEMS="snomed,rxnorm,loinc"
BATCH_SIZE=10000
WORKERS=4
VALIDATE_ONLY=false
DEBUG=false
FORCE=false

# Function to print colored output
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

# Function to show usage
show_usage() {
    cat << EOF
KB-7 Terminology Data Loading Script

Usage: $0 [OPTIONS]

Options:
    --systems SYSTEMS     Comma-separated list of systems to load (default: snomed,rxnorm,loinc)
    --batch-size SIZE     Number of records per batch (default: 10000)
    --workers NUM         Number of concurrent workers (default: 4)
    --validate-only       Only validate data files, don't load
    --debug               Enable debug logging
    --force               Force reload even if data exists
    --help                Show this help message

Examples:
    # Load all systems with default settings
    $0

    # Load only SNOMED CT with debug logging
    $0 --systems snomed --debug

    # Validate all data files without loading
    $0 --validate-only

    # Force reload with custom batch size
    $0 --force --batch-size 5000

Supported Systems:
    - snomed   : SNOMED CT International Edition
    - rxnorm   : RxNorm drug terminology
    - loinc    : LOINC laboratory codes

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --systems)
            SYSTEMS="$2"
            shift 2
            ;;
        --batch-size)
            BATCH_SIZE="$2"
            shift 2
            ;;
        --workers)
            WORKERS="$2"
            shift 2
            ;;
        --validate-only)
            VALIDATE_ONLY=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if data directory exists
    if [ ! -d "$DATA_DIR" ]; then
        print_error "Data directory not found: $DATA_DIR"
        exit 1
    fi
    
    # Check if ETL tool exists
    if [ ! -f "$ETL_CMD" ]; then
        print_error "ETL tool not found: $ETL_CMD"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to validate system data directories
validate_data_directories() {
    print_status "Validating data directories..."
    
    IFS=',' read -ra SYSTEM_LIST <<< "$SYSTEMS"
    for system in "${SYSTEM_LIST[@]}"; do
        case $system in
            snomed)
                if [ ! -d "$DATA_DIR/snomed" ]; then
                    print_error "SNOMED CT data directory not found: $DATA_DIR/snomed"
                    exit 1
                fi
                # Check for key SNOMED files
                if [ ! -f "$DATA_DIR/snomed/snapshot/sct2_Concept_Snapshot_INT.txt" ]; then
                    print_warning "SNOMED concept file not found in expected location"
                    print_status "Looking for SNOMED files in extracted directory..."
                    if [ ! -d "$DATA_DIR/snomed/extracted" ]; then
                        print_error "No SNOMED data files found"
                        exit 1
                    fi
                fi
                ;;
            rxnorm)
                if [ ! -d "$DATA_DIR/rxnorm" ]; then
                    print_error "RxNorm data directory not found: $DATA_DIR/rxnorm"
                    exit 1
                fi
                # Check for key RxNorm files
                if [ ! -f "$DATA_DIR/rxnorm/rrf/RXNCONSO.RRF" ]; then
                    print_warning "RxNorm RXNCONSO.RRF file not found in rrf directory"
                    if [ ! -d "$DATA_DIR/rxnorm/extracted" ]; then
                        print_error "No RxNorm data files found"
                        exit 1
                    fi
                fi
                ;;
            loinc)
                if [ ! -d "$DATA_DIR/loinc" ]; then
                    print_error "LOINC data directory not found: $DATA_DIR/loinc"
                    exit 1
                fi
                ;;
            *)
                print_error "Unsupported system: $system"
                exit 1
                ;;
        esac
    done
    
    print_success "Data directories validation passed"
}

# Function to load a specific system
load_system() {
    local system=$1
    local data_path=""
    
    case $system in
        snomed)
            # Try snapshot first, then extracted
            if [ -d "$DATA_DIR/snomed/snapshot" ] && [ -f "$DATA_DIR/snomed/snapshot/sct2_Concept_Snapshot_INT.txt" ]; then
                data_path="$DATA_DIR/snomed/snapshot"
            elif [ -d "$DATA_DIR/snomed/extracted" ]; then
                # Find the extracted SNOMED directory
                snomed_extracted=$(find "$DATA_DIR/snomed/extracted" -maxdepth 1 -type d -name "SnomedCT_*" | head -1)
                if [ -n "$snomed_extracted" ]; then
                    data_path="$snomed_extracted/Snapshot/Terminology"
                else
                    print_error "Could not find SNOMED extracted data"
                    return 1
                fi
            else
                print_error "Could not find SNOMED data files"
                return 1
            fi
            ;;
        rxnorm)
            # Try rrf first, then extracted
            if [ -d "$DATA_DIR/rxnorm/rrf" ] && [ -f "$DATA_DIR/rxnorm/rrf/RXNCONSO.RRF" ]; then
                data_path="$DATA_DIR/rxnorm/rrf"
            elif [ -d "$DATA_DIR/rxnorm/extracted/rrf" ]; then
                data_path="$DATA_DIR/rxnorm/extracted/rrf"
            else
                print_error "Could not find RxNorm data files"
                return 1
            fi
            ;;
        loinc)
            if [ -d "$DATA_DIR/loinc/snapshot" ]; then
                data_path="$DATA_DIR/loinc/snapshot"
            else
                print_error "Could not find LOINC data files"
                return 1
            fi
            ;;
    esac
    
    print_status "Loading $system from: $data_path"
    
    # Build ETL command
    local etl_args="--data=$data_path --system=$system --batch=$BATCH_SIZE --workers=$WORKERS"
    
    if [ "$VALIDATE_ONLY" = true ]; then
        etl_args="$etl_args --validate-only"
    fi
    
    if [ "$DEBUG" = true ]; then
        etl_args="$etl_args --debug"
    fi
    
    if [ "$FORCE" = true ]; then
        etl_args="$etl_args --force"
    fi
    
    # Execute ETL command
    print_status "Running: go run $ETL_CMD $etl_args"
    
    if go run "$ETL_CMD" $etl_args; then
        print_success "Successfully loaded $system"
    else
        print_error "Failed to load $system"
        return 1
    fi
}

# Function to show data summary
show_data_summary() {
    print_status "Data Summary:"
    echo "=============="
    
    if [[ $SYSTEMS == *"snomed"* ]]; then
        echo "SNOMED CT:"
        if [ -d "$DATA_DIR/snomed/snapshot" ]; then
            concept_count=$(wc -l < "$DATA_DIR/snomed/snapshot/sct2_Concept_Snapshot_INT.txt" 2>/dev/null || echo "Unknown")
            echo "  - Concepts: $concept_count"
        fi
        echo "  - Location: $DATA_DIR/snomed"
    fi
    
    if [[ $SYSTEMS == *"rxnorm"* ]]; then
        echo "RxNorm:"
        if [ -f "$DATA_DIR/rxnorm/rrf/RXNCONSO.RRF" ]; then
            concept_count=$(wc -l < "$DATA_DIR/rxnorm/rrf/RXNCONSO.RRF" 2>/dev/null || echo "Unknown")
            echo "  - Concepts: $concept_count"
        fi
        echo "  - Location: $DATA_DIR/rxnorm"
    fi
    
    if [[ $SYSTEMS == *"loinc"* ]]; then
        echo "LOINC:"
        if [ -f "$DATA_DIR/loinc/snapshot/sct2_Concept_Snapshot_LO1010000_20250321.txt" ]; then
            concept_count=$(wc -l < "$DATA_DIR/loinc/snapshot/sct2_Concept_Snapshot_LO1010000_20250321.txt" 2>/dev/null || echo "Unknown")
            echo "  - Concepts: $concept_count"
        fi
        echo "  - Location: $DATA_DIR/loinc"
    fi
    
    echo ""
}

# Main execution
main() {
    print_status "KB-7 Terminology Data Loading Script"
    print_status "======================================"
    
    # Check prerequisites
    check_prerequisites
    
    # Validate data directories
    validate_data_directories
    
    # Show data summary
    show_data_summary
    
    # Parse systems to load
    IFS=',' read -ra SYSTEM_LIST <<< "$SYSTEMS"
    
    if [ "$VALIDATE_ONLY" = true ]; then
        print_status "Validation mode - no data will be loaded"
    fi
    
    # Load each system
    local failed_systems=()
    local success_count=0
    
    for system in "${SYSTEM_LIST[@]}"; do
        print_status "Processing system: $system"
        
        if load_system "$system"; then
            ((success_count++))
        else
            failed_systems+=("$system")
        fi
        
        echo ""
    done
    
    # Summary
    print_status "Loading Summary:"
    print_status "================"
    print_success "Successfully loaded: $success_count/${#SYSTEM_LIST[@]} systems"
    
    if [ ${#failed_systems[@]} -gt 0 ]; then
        print_error "Failed systems: ${failed_systems[*]}"
        exit 1
    else
        print_success "All systems loaded successfully!"
    fi
}

# Run main function
main "$@"