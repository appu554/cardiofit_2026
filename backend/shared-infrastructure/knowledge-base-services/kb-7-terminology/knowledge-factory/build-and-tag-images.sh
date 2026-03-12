#!/bin/bash
# Production-Ready Docker Image Builder with Git SHA Tagging
#
# This script builds and tags Docker images with IMMUTABLE git commit SHAs
# instead of :latest tags, enabling:
# - Reproducible deployments
# - Easy rollbacks
# - Audit trail
# - GitOps compatibility (ArgoCD, Flux)
#
# Usage: ./build-and-tag-images.sh [--push] [--registry ghcr.io/owner]

set -e

# Configuration
REGISTRY="${DOCKER_REGISTRY:-ghcr.io/onkarshahi-ind}"
PUSH_IMAGES=false
GIT_SHA=$(git rev-parse HEAD)  # Full SHA to match GitHub Actions ${{ github.sha }}
GIT_SHA_SHORT=$(git rev-parse --short HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --push)
      PUSH_IMAGES=true
      shift
      ;;
    --registry)
      REGISTRY="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

echo "=============================================="
echo "Production Docker Image Builder"
echo "=============================================="
echo "Git SHA:      $GIT_SHA (short: $GIT_SHA_SHORT)"
echo "Git Branch:   $GIT_BRANCH"
echo "Registry:     $REGISTRY"
echo "Push Images:  $PUSH_IMAGES"
echo "=============================================="
echo ""

# Build and tag images (bash 3.2 compatible)
IMAGES="snomed-toolkit:docker/Dockerfile.snomed-toolkit converters:docker/Dockerfile.converters robot:docker/Dockerfile.robot"

for IMAGE_SPEC in $IMAGES; do
  IMAGE_NAME="${IMAGE_SPEC%%:*}"
  DOCKERFILE="${IMAGE_SPEC#*:}"

  echo "📦 Building $IMAGE_NAME..."
  echo "   Dockerfile: $DOCKERFILE"
  echo "   Context: ."

  # Build with multi-platform support
  docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --file "$DOCKERFILE" \
    --tag "$REGISTRY/$IMAGE_NAME:$GIT_SHA" \
    --tag "$REGISTRY/$IMAGE_NAME:$GIT_BRANCH" \
    $([ "$PUSH_IMAGES" = true ] && echo "--push" || echo "--load") \
    .

  echo "✅ Tagged: $REGISTRY/$IMAGE_NAME:$GIT_SHA"
  echo "✅ Tagged: $REGISTRY/$IMAGE_NAME:$GIT_BRANCH"
  echo ""
done

echo "=============================================="
echo "✅ Build Complete!"
echo "=============================================="
echo ""
echo "Images built with SHA: $GIT_SHA"
echo ""
echo "To use in your workflow:"
echo "  snomed-toolkit:$GIT_SHA"
echo "  converters:$GIT_SHA"
echo "  robot:$GIT_SHA"
echo ""

if [ "$PUSH_IMAGES" = true ]; then
  echo "✅ Images pushed to $REGISTRY"
  echo ""
  echo "Update your workflow file to reference:"
  echo "  ghcr.io/$REGISTRY/snomed-toolkit:$GIT_SHA"
else
  echo "💡 To push to registry, run:"
  echo "   ./build-and-tag-images.sh --push"
fi

echo ""
echo "=============================================="
