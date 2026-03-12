#!/bin/bash
# ============================================================
# Signature Verification Script
# ============================================================
# Verifies Ed25519 signatures on all artifacts.
# Used by runtime platform before loading artifacts.
#
# Usage: ./verify-signatures.sh [build-dir] [public-key]
# ============================================================

set -e

BUILD_DIR="${1:-build}"
PUBLIC_KEY="${2:-.keys/public.pem}"

echo "🔍 Signature Verification"
echo "========================="
echo ""

# Check for public key
if [ ! -f "$PUBLIC_KEY" ]; then
    echo "❌ Public key not found: $PUBLIC_KEY"
    exit 1
fi

VERIFIED=0
FAILED=0

# Verify manifest signature
SIGNATURES_DIR="$BUILD_DIR/signatures"
MANIFESTS_DIR="$BUILD_DIR/manifests"

if [ -d "$SIGNATURES_DIR" ]; then
    for sig_file in "$SIGNATURES_DIR"/*.sig; do
        if [ -f "$sig_file" ]; then
            # Find corresponding artifact
            base_name=$(basename "$sig_file" .sig)
            artifact_file="$MANIFESTS_DIR/${base_name}.json"

            if [ -f "$artifact_file" ]; then
                echo "🔐 Verifying: $(basename "$artifact_file")"

                if openssl pkeyutl -verify \
                    -pubin -inkey "$PUBLIC_KEY" \
                    -in "$artifact_file" \
                    -sigfile "$sig_file" 2>/dev/null; then
                    echo "   ✅ Valid signature"
                    ((VERIFIED++))
                else
                    echo "   ❌ INVALID SIGNATURE!"
                    ((FAILED++))
                fi
            fi
        fi
    done
fi

# Verify bundle signatures
for bundle in "$BUILD_DIR"/*.tar.gz; do
    if [ -f "$bundle" ] && [ -f "${bundle}.sig" ]; then
        echo "🔐 Verifying: $(basename "$bundle")"

        if openssl pkeyutl -verify \
            -pubin -inkey "$PUBLIC_KEY" \
            -in "$bundle" \
            -sigfile "${bundle}.sig" 2>/dev/null; then
            echo "   ✅ Valid signature"
            ((VERIFIED++))
        else
            echo "   ❌ INVALID SIGNATURE!"
            ((FAILED++))
        fi
    fi
done

echo ""
echo "📊 Verification Summary:"
echo "   ✅ Verified: $VERIFIED"
echo "   ❌ Failed: $FAILED"

if [ $FAILED -gt 0 ]; then
    echo ""
    echo "⚠️  WARNING: Some signatures are invalid!"
    echo "   Do NOT use these artifacts in production."
    exit 1
else
    echo ""
    echo "✅ All signatures verified successfully!"
    echo "   Artifacts are safe to deploy."
fi
