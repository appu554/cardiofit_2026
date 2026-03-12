#!/bin/bash
set -e

# KB-7 Knowledge Factory - Build and Push Docker Images to GHCR
# This script builds all three Docker images and pushes them to GitHub Container Registry

REGISTRY="ghcr.io"
OWNER="onkarshahi-ind"  # Lowercase (required by Docker)
TAG="latest"

echo "=================================================="
echo "KB-7 Knowledge Factory - Docker Image Builder"
echo "=================================================="
echo ""
echo "Registry: $REGISTRY"
echo "Owner: $OWNER"
echo "Tag: $TAG"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if we're in the right directory
if [ ! -d "docker" ] || [ ! -d "scripts" ]; then
    echo "❌ Error: Must run from knowledge-factory/ directory"
    echo "Current directory: $(pwd)"
    exit 1
fi

# Function to build and push an image
build_and_push() {
    local dockerfile=$1
    local image_name=$2
    local full_image="$REGISTRY/$OWNER/$image_name:$TAG"

    echo "---------------------------------------------------"
    echo "Building: $image_name"
    echo "---------------------------------------------------"

    # Build the image
    docker build \
        -f "docker/$dockerfile" \
        -t "$full_image" \
        --platform linux/amd64 \
        .

    if [ $? -ne 0 ]; then
        echo "❌ Failed to build $image_name"
        return 1
    fi

    echo "✅ Built: $full_image"
    echo ""

    # Show image size
    docker images "$full_image" --format "Size: {{.Size}}"
    echo ""

    return 0
}

# Build all three images
echo "🔨 Building Docker images..."
echo ""

build_and_push "Dockerfile.snomed-toolkit" "snomed-toolkit" || exit 1
build_and_push "Dockerfile.robot" "robot" || exit 1
build_and_push "Dockerfile.converters" "converters" || exit 1

echo "=================================================="
echo "✅ All images built successfully!"
echo "=================================================="
echo ""
echo "Built images:"
docker images | grep "$REGISTRY/$OWNER" | grep "$TAG"
echo ""

# Ask if user wants to push
echo "---------------------------------------------------"
echo "Ready to push images to GHCR?"
echo "---------------------------------------------------"
echo ""
echo "This will push the images to:"
echo "  - $REGISTRY/$OWNER/snomed-toolkit:$TAG"
echo "  - $REGISTRY/$OWNER/robot:$TAG"
echo "  - $REGISTRY/$OWNER/converters:$TAG"
echo ""
read -p "Continue with push? (y/n): " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "ℹ️  Push cancelled. Images are built locally."
    echo ""
    echo "To push manually later:"
    echo "  1. docker login ghcr.io -u onkarshahi-IND"
    echo "  2. docker push $REGISTRY/$OWNER/snomed-toolkit:$TAG"
    echo "  3. docker push $REGISTRY/$OWNER/robot:$TAG"
    echo "  4. docker push $REGISTRY/$OWNER/converters:$TAG"
    exit 0
fi

# Check if user is logged in to GHCR
echo ""
echo "🔐 Checking GHCR authentication..."
if ! docker pull "$REGISTRY/$OWNER/snomed-toolkit:$TAG" 2>/dev/null; then
    echo "⚠️  Not authenticated to GHCR. Logging in..."
    echo ""
    echo "You'll need a GitHub Personal Access Token (PAT) with 'write:packages' scope."
    echo "Create one at: https://github.com/settings/tokens"
    echo ""

    docker login "$REGISTRY" -u onkarshahi-IND

    if [ $? -ne 0 ]; then
        echo "❌ GHCR login failed"
        exit 1
    fi
fi

# Push all images
echo ""
echo "📤 Pushing images to GHCR..."
echo ""

push_image() {
    local image_name=$1
    local full_image="$REGISTRY/$OWNER/$image_name:$TAG"

    echo "Pushing: $full_image"
    docker push "$full_image"

    if [ $? -ne 0 ]; then
        echo "❌ Failed to push $image_name"
        return 1
    fi

    echo "✅ Pushed: $full_image"
    echo ""
    return 0
}

push_image "snomed-toolkit" || exit 1
push_image "robot" || exit 1
push_image "converters" || exit 1

echo "=================================================="
echo "✅ All images pushed successfully!"
echo "=================================================="
echo ""
echo "Published images:"
echo "  • $REGISTRY/$OWNER/snomed-toolkit:$TAG"
echo "  • $REGISTRY/$OWNER/robot:$TAG"
echo "  • $REGISTRY/$OWNER/converters:$TAG"
echo ""
echo "Verify at: https://github.com/onkarshahi-IND?tab=packages"
echo ""
echo "🚀 GitHub Actions workflow is now ready to run!"
echo ""
echo "Next steps:"
echo "  1. Trigger workflow: gcloud workflows run kb7-factory-workflow-production"
echo "  2. Monitor: https://github.com/onkarshahi-IND/knowledge-factory/actions"
echo "  3. Watch Stage 2 (Transform) succeed with Docker images!"
echo ""
