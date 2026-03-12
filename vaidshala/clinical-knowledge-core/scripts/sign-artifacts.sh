#!/bin/bash
# ============================================================
# Artifact Signing Script
# ============================================================
# Signs build artifacts using Ed25519.
# Creates detached signatures and updates manifest.
#
# Usage: ./sign-artifacts.sh [build-dir] [private-key]
# ============================================================

set -e

BUILD_DIR="${1:-build}"
PRIVATE_KEY="${2:-.keys/private.pem}"

echo "🔐 Artifact Signing Pipeline"
echo "============================"
echo ""

# Check for private key
if [ ! -f "$PRIVATE_KEY" ]; then
    echo "❌ Private key not found: $PRIVATE_KEY"
    echo "   Run 'make generate-keys' first, or provide key path"
    exit 1
fi

# Find manifest to sign
MANIFEST_DIR="$BUILD_DIR/manifests"
LATEST_MANIFEST=$(ls -t "$MANIFEST_DIR"/manifest-*.json 2>/dev/null | head -1)

if [ -z "$LATEST_MANIFEST" ]; then
    echo "❌ No manifest found. Run 'make build' first."
    exit 1
fi

echo "📋 Signing manifest: $(basename "$LATEST_MANIFEST")"

# Create signature directory
SIGNATURES_DIR="$BUILD_DIR/signatures"
mkdir -p "$SIGNATURES_DIR"

# Sign the manifest
MANIFEST_BASENAME=$(basename "$LATEST_MANIFEST" .json)
SIGNATURE_FILE="$SIGNATURES_DIR/${MANIFEST_BASENAME}.sig"

echo "🖊️  Creating signature..."

# Sign using OpenSSL Ed25519
openssl pkeyutl -sign \
    -inkey "$PRIVATE_KEY" \
    -in "$LATEST_MANIFEST" \
    -out "$SIGNATURE_FILE"

# Create Base64-encoded signature for embedding
SIGNATURE_B64=$(base64 < "$SIGNATURE_FILE" | tr -d '\n')

# Update manifest with signature
echo "📝 Embedding signature in manifest..."

# Create signed manifest
SIGNED_MANIFEST="${LATEST_MANIFEST%.json}-signed.json"

jq --arg sig "$SIGNATURE_B64" \
   --arg sigTime "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
   '. + {
     signature: {
       algorithm: "Ed25519",
       signedAt: $sigTime,
       value: $sig
     }
   }' "$LATEST_MANIFEST" > "$SIGNED_MANIFEST"

# Sign individual artifact bundles
echo ""
echo "📦 Signing artifact bundles..."

# Create tarball of ELM files
if [ -d "$BUILD_DIR/cql-to-elm" ] && [ "$(ls -A "$BUILD_DIR/cql-to-elm" 2>/dev/null)" ]; then
    ELM_TARBALL="$BUILD_DIR/elm-bundle.tar.gz"
    tar -czf "$ELM_TARBALL" -C "$BUILD_DIR" cql-to-elm

    openssl pkeyutl -sign \
        -inkey "$PRIVATE_KEY" \
        -in "$ELM_TARBALL" \
        -out "${ELM_TARBALL}.sig"

    echo "   ✅ Signed: elm-bundle.tar.gz"
fi

# Create tarball of ValueSets
if [ -d "$BUILD_DIR/valueset-expansion" ] && [ "$(ls -A "$BUILD_DIR/valueset-expansion" 2>/dev/null)" ]; then
    VS_TARBALL="$BUILD_DIR/valueset-bundle.tar.gz"
    tar -czf "$VS_TARBALL" -C "$BUILD_DIR" valueset-expansion

    openssl pkeyutl -sign \
        -inkey "$PRIVATE_KEY" \
        -in "$VS_TARBALL" \
        -out "${VS_TARBALL}.sig"

    echo "   ✅ Signed: valueset-bundle.tar.gz"
fi

# Generate signature manifest
cat > "$SIGNATURES_DIR/signatures.json" << EOF
{
  "signatureVersion": "1.0.0",
  "algorithm": "Ed25519",
  "signedAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "artifacts": [
    {
      "file": "$(basename "$LATEST_MANIFEST")",
      "signature": "${MANIFEST_BASENAME}.sig"
    }
$(if [ -f "$BUILD_DIR/elm-bundle.tar.gz.sig" ]; then
echo '    ,{
      "file": "elm-bundle.tar.gz",
      "signature": "elm-bundle.tar.gz.sig"
    }'
fi)
$(if [ -f "$BUILD_DIR/valueset-bundle.tar.gz.sig" ]; then
echo '    ,{
      "file": "valueset-bundle.tar.gz",
      "signature": "valueset-bundle.tar.gz.sig"
    }'
fi)
  ]
}
EOF

echo ""
echo "✅ Signing complete!"
echo ""
echo "📊 Signed Artifacts:"
echo "   📋 Manifest: $SIGNED_MANIFEST"
echo "   🔏 Signature: $SIGNATURE_FILE"
[ -f "$BUILD_DIR/elm-bundle.tar.gz" ] && echo "   📦 ELM Bundle: elm-bundle.tar.gz"
[ -f "$BUILD_DIR/valueset-bundle.tar.gz" ] && echo "   📦 ValueSet Bundle: valueset-bundle.tar.gz"
echo ""
echo "🔍 To verify: make verify"
