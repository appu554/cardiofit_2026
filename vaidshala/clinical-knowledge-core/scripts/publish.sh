#!/bin/bash
# ============================================================
# Artifact Publishing Script
# ============================================================
# Publishes signed artifacts to GitHub Releases.
#
# Usage: ./publish.sh [version]
# ============================================================

set -e

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo '0.0.1-dev')}"
BUILD_DIR="build"

echo "📤 Publishing Artifacts"
echo "======================="
echo "Version: $VERSION"
echo ""

# Check for GitHub CLI
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) not found."
    echo "   Install: https://cli.github.com/"
    exit 1
fi

# Check for signed manifest
SIGNED_MANIFEST=$(ls -t "$BUILD_DIR/manifests"/*-signed.json 2>/dev/null | head -1)

if [ -z "$SIGNED_MANIFEST" ]; then
    echo "❌ No signed manifest found. Run 'make sign' first."
    exit 1
fi

echo "📋 Manifest: $(basename "$SIGNED_MANIFEST")"
echo ""

# Create release notes
RELEASE_NOTES=$(cat << EOF
## Clinical Knowledge Core v${VERSION}

### Artifacts Included
- 📦 ELM Compiled Libraries (cql-to-elm)
- 📦 Expanded ValueSets
- 📋 Signed Manifest

### Verification
All artifacts are signed with Ed25519. Verify signatures before use:
\`\`\`bash
make verify
\`\`\`

### Tiers Included
- tier-0-fhir: FHIR Foundation
- tier-0.5-terminology: Terminology (SNOMED, ICD, LOINC)
- tier-1-primitives: Utility Functions
- tier-2-cqm-infra: Quality Measure Infrastructure
- tier-3-domain-commons: Clinical Calculators
- tier-4-guidelines: Clinical Guidelines
- tier-5-regional-adapters: Regional Adaptations

---
Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF
)

# Collect artifacts to upload
ARTIFACTS=("$SIGNED_MANIFEST")

[ -f "$BUILD_DIR/elm-bundle.tar.gz" ] && ARTIFACTS+=("$BUILD_DIR/elm-bundle.tar.gz")
[ -f "$BUILD_DIR/elm-bundle.tar.gz.sig" ] && ARTIFACTS+=("$BUILD_DIR/elm-bundle.tar.gz.sig")
[ -f "$BUILD_DIR/valueset-bundle.tar.gz" ] && ARTIFACTS+=("$BUILD_DIR/valueset-bundle.tar.gz")
[ -f "$BUILD_DIR/valueset-bundle.tar.gz.sig" ] && ARTIFACTS+=("$BUILD_DIR/valueset-bundle.tar.gz.sig")
[ -f "$BUILD_DIR/signatures/signatures.json" ] && ARTIFACTS+=("$BUILD_DIR/signatures/signatures.json")

echo "📦 Artifacts to upload:"
for artifact in "${ARTIFACTS[@]}"; do
    echo "   - $(basename "$artifact")"
done
echo ""

# Create or update release
echo "🚀 Creating GitHub Release..."

gh release create "v${VERSION}" \
    --title "Clinical Knowledge Core v${VERSION}" \
    --notes "$RELEASE_NOTES" \
    --draft \
    "${ARTIFACTS[@]}" || {
    echo ""
    echo "⚠️  Release may already exist. Uploading to existing release..."
    gh release upload "v${VERSION}" "${ARTIFACTS[@]}" --clobber
}

echo ""
echo "✅ Published successfully!"
echo ""
echo "📍 Release URL:"
echo "   https://github.com/$(gh repo view --json nameWithOwner -q '.nameWithOwner')/releases/tag/v${VERSION}"
echo ""
echo "📋 Next steps:"
echo "   1. Review the draft release on GitHub"
echo "   2. Edit release notes if needed"
echo "   3. Publish the release"
echo "   4. Update runtime platform to pull new artifacts"
