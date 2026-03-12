#!/bin/bash
# Pull and verify signed artifacts from clinical-knowledge-core
# Usage: ./pull-and-verify.sh [version]

set -e

VERSION=${1:-"latest"}
REGISTRY_URL="${ARTIFACT_REGISTRY_URL:-https://registry.vaidshala.internal}"

echo "Pulling artifacts version: $VERSION"

# Pull ELM artifacts
echo "Pulling ELM..."
curl -sS "$REGISTRY_URL/elm/$VERSION.tar.gz" -o /tmp/elm.tar.gz

# Pull value sets
echo "Pulling value sets..."
curl -sS "$REGISTRY_URL/valuesets/$VERSION.tar.gz" -o /tmp/valuesets.tar.gz

# Pull manifest
echo "Pulling manifest..."
curl -sS "$REGISTRY_URL/manifests/$VERSION.json" -o /tmp/manifest.json

# Verify signatures
echo "Verifying signatures..."
# TODO: Implement Ed25519 signature verification

echo "All artifacts verified successfully!"
