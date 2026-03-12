#!/bin/bash
# Upload KB-7 Kernel to S3
# Uploads versioned kernel and updates 'latest' pointer
# Requires: AWS CLI configured with credentials

set -e

WORKSPACE=${WORKSPACE:-/workspace}
S3_BUCKET=${S3_BUCKET_ARTIFACTS:-cardiofit-kb-artifacts}
VERSION=${VERSION:-$(date +%Y%m%d)}

echo "=================================================="
echo "KB-7 Kernel Upload to S3"
echo "=================================================="
echo "Bucket:    s3://$S3_BUCKET"
echo "Version:   $VERSION"
echo "=================================================="

cd "$WORKSPACE"

# Verify files exist
if [ ! -f "kb7-kernel.ttl" ]; then
    echo "ERROR: kb7-kernel.ttl not found"
    exit 1
fi

if [ ! -f "kb7-manifest.json" ]; then
    echo "ERROR: kb7-manifest.json not found"
    exit 1
fi

echo ""
echo "Uploading versioned kernel..."

# Upload versioned files
aws s3 cp kb7-kernel.ttl \
    s3://$S3_BUCKET/$VERSION/kb7-kernel.ttl \
    --metadata "version=$VERSION,build-date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --storage-class STANDARD

aws s3 cp kb7-manifest.json \
    s3://$S3_BUCKET/$VERSION/kb7-manifest.json \
    --content-type "application/json" \
    --metadata "version=$VERSION"

# Upload checksums
if [ -f "kb7-kernel.ttl.sha256" ]; then
    aws s3 cp kb7-kernel.ttl.sha256 \
        s3://$S3_BUCKET/$VERSION/kb7-kernel.ttl.sha256
fi

echo ""
echo "Updating 'latest' pointer..."

# Update latest pointers
aws s3 cp kb7-kernel.ttl \
    s3://$S3_BUCKET/latest/kb7-kernel.ttl

aws s3 cp kb7-manifest.json \
    s3://$S3_BUCKET/latest/kb7-manifest.json

# Create version list
echo ""
echo "Updating version registry..."

cat > version-registry.json <<EOF
{
  "latest_version": "$VERSION",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "s3_uri": "s3://$S3_BUCKET/$VERSION/kb7-kernel.ttl"
}
EOF

aws s3 cp version-registry.json \
    s3://$S3_BUCKET/version-registry.json \
    --content-type "application/json"

# Generate public URL (if bucket has public read)
PUBLIC_URL="https://$S3_BUCKET.s3.amazonaws.com/$VERSION/kb7-kernel.ttl"

echo ""
echo "=================================================="
echo "Upload Complete"
echo "=================================================="
echo "Versioned:  s3://$S3_BUCKET/$VERSION/kb7-kernel.ttl"
echo "Latest:     s3://$S3_BUCKET/latest/kb7-kernel.ttl"
echo "Public URL: $PUBLIC_URL"
echo "=================================================="

echo ""
echo "✅ KB-7 kernel uploaded successfully"
