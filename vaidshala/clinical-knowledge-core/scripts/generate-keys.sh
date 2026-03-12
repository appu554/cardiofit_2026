#!/bin/bash
# ============================================================
# Ed25519 Keypair Generation Script
# ============================================================
# Generates Ed25519 keypair for signing clinical artifacts.
#
# ⚠️  IMPORTANT: Run this ONCE. Store private key securely!
#     The private key should NEVER be committed to git.
#     Store in: GitHub Secrets, AWS Secrets Manager, or Vault.
#
# Usage: ./generate-keys.sh [output-dir]
# ============================================================

set -e

OUTPUT_DIR="${1:-.keys}"

echo "🔑 Ed25519 Keypair Generation"
echo "=============================="
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Check if keys already exist
if [ -f "$OUTPUT_DIR/private.pem" ]; then
    echo "⚠️  Keys already exist in $OUTPUT_DIR"
    echo "   To regenerate, delete existing keys first:"
    echo "   rm -rf $OUTPUT_DIR/*.pem"
    exit 1
fi

# Generate Ed25519 keypair using OpenSSL
echo "📝 Generating Ed25519 private key..."
openssl genpkey -algorithm Ed25519 -out "$OUTPUT_DIR/private.pem"

echo "📝 Extracting public key..."
openssl pkey -in "$OUTPUT_DIR/private.pem" -pubout -out "$OUTPUT_DIR/public.pem"

# Set secure permissions
chmod 600 "$OUTPUT_DIR/private.pem"
chmod 644 "$OUTPUT_DIR/public.pem"

# Generate key fingerprint
FINGERPRINT=$(openssl pkey -in "$OUTPUT_DIR/public.pem" -pubin -outform DER 2>/dev/null | shasum -a 256 | cut -d' ' -f1)

# Create key metadata
cat > "$OUTPUT_DIR/key-metadata.json" << EOF
{
  "algorithm": "Ed25519",
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "publicKeyFingerprint": "sha256:${FINGERPRINT}",
  "usage": "clinical-knowledge-core artifact signing",
  "publicKeyPath": "public.pem",
  "privateKeyPath": "private.pem (DO NOT COMMIT)"
}
EOF

echo ""
echo "✅ Keypair generated successfully!"
echo ""
echo "📁 Files created:"
echo "   $OUTPUT_DIR/private.pem  (KEEP SECRET!)"
echo "   $OUTPUT_DIR/public.pem   (safe to distribute)"
echo "   $OUTPUT_DIR/key-metadata.json"
echo ""
echo "🔐 Public Key Fingerprint:"
echo "   sha256:${FINGERPRINT}"
echo ""
echo "⚠️  SECURITY REMINDERS:"
echo "   1. NEVER commit private.pem to git"
echo "   2. Store private key in a secure location:"
echo "      - GitHub Secrets: CLINICAL_SIGNING_KEY"
echo "      - AWS Secrets Manager"
echo "      - HashiCorp Vault"
echo "   3. Distribute public.pem to runtime platform"
echo "   4. Back up private.pem securely (offline)"
echo ""
echo "📋 Next steps:"
echo "   1. Add to GitHub Secrets:"
echo "      cat $OUTPUT_DIR/private.pem | base64 | pbcopy"
echo "      Then add as CLINICAL_SIGNING_KEY in repo settings"
echo ""
echo "   2. Run 'make sign' to sign artifacts"
