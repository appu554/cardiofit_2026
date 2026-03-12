#!/bin/bash
# ============================================================
# CQL to ELM Compiler Script
# ============================================================
# Compiles all CQL files to ELM (Expression Logical Model) JSON
# using the official HL7 CQL-to-ELM translator.
#
# Usage: ./compile-cql.sh [source-dir] [output-dir]
# ============================================================

set -e

SOURCE_DIR="${1:-.}"
OUTPUT_DIR="${2:-build/cql-to-elm}"

# CQL Translator version (official HL7)
CQL_TRANSLATOR_VERSION="2.11.0"
CQL_TRANSLATOR_JAR="cql-to-elm-${CQL_TRANSLATOR_VERSION}.jar"
CQL_TRANSLATOR_URL="https://github.com/cqframework/clinical_quality_language/releases/download/v${CQL_TRANSLATOR_VERSION}/cql-to-elm-${CQL_TRANSLATOR_VERSION}-all.jar"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Download CQL translator if not present
TOOLS_DIR="$(dirname "$0")/../.tools"
mkdir -p "$TOOLS_DIR"

if [ ! -f "$TOOLS_DIR/$CQL_TRANSLATOR_JAR" ]; then
    echo "📥 Downloading CQL-to-ELM translator v${CQL_TRANSLATOR_VERSION}..."
    curl -L -o "$TOOLS_DIR/$CQL_TRANSLATOR_JAR" "$CQL_TRANSLATOR_URL" 2>/dev/null || {
        echo "⚠️  Could not download CQL translator. Using Docker fallback..."
        USE_DOCKER=true
    }
fi

# Find all CQL files across tiers
echo "🔍 Finding CQL files..."
CQL_FILES=$(find "$SOURCE_DIR" -name "*.cql" -type f 2>/dev/null || true)

if [ -z "$CQL_FILES" ]; then
    echo "ℹ️  No CQL files found. Skipping compilation."
    exit 0
fi

# Count files
FILE_COUNT=$(echo "$CQL_FILES" | wc -l | tr -d ' ')
echo "📄 Found $FILE_COUNT CQL files to compile"

# Compile each CQL file
COMPILED=0
FAILED=0

for cql_file in $CQL_FILES; do
    filename=$(basename "$cql_file" .cql)
    rel_path=$(dirname "$cql_file" | sed "s|$SOURCE_DIR/||")
    output_subdir="$OUTPUT_DIR/$rel_path"
    mkdir -p "$output_subdir"

    echo "  Compiling: $cql_file"

    if [ "$USE_DOCKER" = true ]; then
        # Docker-based compilation
        docker run --rm \
            -v "$(pwd):/workspace" \
            -w /workspace \
            cqframework/cql-translator:latest \
            -i "$cql_file" \
            -o "$output_subdir/${filename}.json" \
            --format JSON 2>/dev/null && {
            ((COMPILED++))
        } || {
            echo "    ❌ Failed to compile $filename"
            ((FAILED++))
        }
    else
        # Local Java-based compilation
        if command -v java &> /dev/null; then
            java -jar "$TOOLS_DIR/$CQL_TRANSLATOR_JAR" \
                --input "$cql_file" \
                --output "$output_subdir" \
                --format JSON \
                --date-range-optimization \
                --annotations \
                --locators 2>/dev/null && {
                ((COMPILED++))
            } || {
                echo "    ❌ Failed to compile $filename"
                ((FAILED++))
            }
        else
            echo "⚠️  Java not found. Please install Java 11+ or use 'make docker-build'"
            exit 1
        fi
    fi
done

echo ""
echo "📊 Compilation Summary:"
echo "   ✅ Compiled: $COMPILED"
echo "   ❌ Failed: $FAILED"
echo "   📁 Output: $OUTPUT_DIR"

if [ $FAILED -gt 0 ]; then
    exit 1
fi
