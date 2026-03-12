#!/bin/bash
# =============================================================================
# KB-1 Full Ingestion Script
# Ingests drug rules from all supported sources (FDA, TGA, CDSCO)
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=============================================="
echo "  KB-1 Full Drug Registry Ingestion"
echo "=============================================="
echo ""
echo "This script will ingest drugs from:"
echo "  - FDA DailyMed (US) - ~40,000+ drugs"
echo "  - TGA (AU) - Coming soon"
echo "  - CDSCO (IN) - Coming soon"
echo ""
echo "Estimated time: Several hours"
echo ""

read -p "Continue? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

# Run FDA ingestion
echo ""
echo "=== Phase 1: FDA DailyMed (US) ==="
"$SCRIPT_DIR/ingest-fda.sh"

# TGA ingestion (not yet implemented)
echo ""
echo "=== Phase 2: TGA (AU) ==="
echo "TGA ingestion not yet implemented"

# CDSCO ingestion (not yet implemented)
echo ""
echo "=== Phase 3: CDSCO (IN) ==="
echo "CDSCO ingestion not yet implemented"

echo ""
echo "=============================================="
echo "  Full Ingestion Complete"
echo "=============================================="
