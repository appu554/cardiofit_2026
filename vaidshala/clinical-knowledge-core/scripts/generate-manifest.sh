#!/bin/bash
# ============================================================
# Manifest Generation Script
# ============================================================
# Generates a build manifest with checksums for all artifacts.
# This manifest is later signed to ensure integrity.
#
# Usage: ./generate-manifest.sh [version] [output-dir]
# ============================================================

set -e

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo '0.0.1-dev')}"
OUTPUT_DIR="${2:-build/manifests}"
BUILD_DIR="build"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

echo "📋 Generating Build Manifest"
echo "============================"
echo "Version: $VERSION"
echo ""

# Collect artifact information
MANIFEST_FILE="$OUTPUT_DIR/manifest-${VERSION}.json"

# Calculate checksums for all build artifacts
echo "🔐 Calculating checksums..."

ELM_CHECKSUMS=""
if [ -d "$BUILD_DIR/cql-to-elm" ]; then
    shopt -s nullglob globstar 2>/dev/null || true
    for f in "$BUILD_DIR/cql-to-elm"/**/*.json; do
        if [ -f "$f" ]; then
            checksum=$(shasum -a 256 "$f" | cut -d' ' -f1)
            filename=$(basename "$f")
            ELM_CHECKSUMS="${ELM_CHECKSUMS}    \"${filename}\": \"sha256:${checksum}\",\n"
        fi
    done
    shopt -u nullglob globstar 2>/dev/null || true
fi

VS_CHECKSUMS=""
if [ -d "$BUILD_DIR/valueset-expansion/expanded" ]; then
    shopt -s nullglob 2>/dev/null || true
    for f in "$BUILD_DIR/valueset-expansion/expanded"/*.json; do
        if [ -f "$f" ]; then
            checksum=$(shasum -a 256 "$f" | cut -d' ' -f1)
            filename=$(basename "$f")
            VS_CHECKSUMS="${VS_CHECKSUMS}    \"${filename}\": \"sha256:${checksum}\",\n"
        fi
    done
    shopt -u nullglob 2>/dev/null || true
fi

# Get git information
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY=$(git diff --quiet 2>/dev/null && echo "false" || echo "true")

# Count artifacts
ELM_COUNT=$(find "$BUILD_DIR/cql-to-elm" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
VS_COUNT=$(find "$BUILD_DIR/valueset-expansion/expanded" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')

# Generate manifest
cat > "$MANIFEST_FILE" << EOF
{
  "manifestVersion": "1.0.0",
  "version": "$VERSION",
  "generatedAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "git": {
    "commit": "$GIT_COMMIT",
    "branch": "$GIT_BRANCH",
    "dirty": $GIT_DIRTY
  },
  "tiers": [
    "tier-0-fhir",
    "tier-0.5-terminology",
    "tier-1-primitives",
    "tier-2-cqm-infra",
    "tier-3-domain-commons",
    "tier-4-guidelines",
    "tier-5-regional-adapters"
  ],
  "artifacts": {
    "elm": {
      "count": $ELM_COUNT,
      "directory": "cql-to-elm",
      "checksums": {
$(echo -e "$ELM_CHECKSUMS" | sed '$ s/,$//')
      }
    },
    "valuesets": {
      "count": $VS_COUNT,
      "directory": "valueset-expansion",
      "checksums": {
$(echo -e "$VS_CHECKSUMS" | sed '$ s/,$//')
      }
    }
  },
  "signature": null
}
EOF

echo "✅ Manifest generated: $MANIFEST_FILE"
echo ""
echo "📊 Manifest Contents:"
echo "   Version: $VERSION"
echo "   Git Commit: ${GIT_COMMIT:0:8}"
echo "   ELM Files: $ELM_COUNT"
echo "   ValueSets: $VS_COUNT"
