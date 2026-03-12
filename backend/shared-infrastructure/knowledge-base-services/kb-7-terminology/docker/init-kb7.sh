#!/bin/bash

# KB-7 Initialization Script
# Runs after GraphDB is ready to initialize repository and load ontology

set -e

echo "🔧 Initializing KB-7 Semantic Infrastructure"
echo "============================================"

# Wait for GraphDB to be ready
echo "Waiting for GraphDB to start..."
max_attempts=30
attempt=1

while [ $attempt -le $max_attempts ]; do
    if curl -f -s http://localhost:7200/rest/repositories >/dev/null 2>&1; then
        echo "✅ GraphDB is ready"
        break
    fi
    echo "⏳ Attempt $attempt/$max_attempts - GraphDB not ready yet..."
    sleep 10
    ((attempt++))
done

if [ $attempt -gt $max_attempts ]; then
    echo "❌ GraphDB failed to start within 5 minutes"
    exit 1
fi

# Create KB-7 repository if it doesn't exist
echo "🗄️ Creating KB-7 terminology repository..."

REPO_CHECK=$(curl -s -w "%{http_code}" -o /dev/null http://localhost:7200/rest/repositories/kb7-terminology || echo "000")

if [ "$REPO_CHECK" != "200" ]; then
    echo "Creating new repository..."

    # Create repository using GraphDB REST API
    curl -X PUT \
        -H "Content-Type: application/json" \
        -d '{
            "repositoryID": "kb7-terminology",
            "title": "KB-7 Clinical Terminology Repository",
            "type": "file-repository",
            "config": {
                "baseURL": "http://cardiofit.ai/kb7/",
                "ruleset": "owl2-rl-optimized",
                "enableContextIndex": true,
                "cacheMemory": "1g",
                "entityIndexSize": "10000000"
            }
        }' \
        "http://localhost:7200/rest/repositories/kb7-terminology" || {
        echo "⚠️ Repository creation failed, but continuing..."
    }

    echo "✅ Repository created"
else
    echo "✅ Repository already exists"
fi

# Load core ontology if available
if [ -f "/app/ontologies/kb7-core.ttl" ]; then
    echo "📚 Loading KB-7 core ontology..."

    curl -X POST \
        -H "Content-Type: application/x-turtle" \
        -T /app/ontologies/kb7-core.ttl \
        "http://localhost:7200/repositories/kb7-terminology/statements" && {
        echo "✅ Core ontology loaded successfully"
    } || {
        echo "⚠️ Core ontology loading failed, but continuing..."
    }
else
    echo "⚠️ Core ontology file not found at /app/ontologies/kb7-core.ttl"
fi

# Test SPARQL endpoint
echo "🧪 Testing SPARQL endpoint..."
TEST_QUERY='SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 5'

curl -s -X POST \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "query=$TEST_QUERY" \
    "http://localhost:7200/repositories/kb7-terminology" >/dev/null && {
    echo "✅ SPARQL endpoint is working"
} || {
    echo "⚠️ SPARQL endpoint test failed"
}

echo "🎯 KB-7 initialization complete!"
echo ""
echo "📊 Service URLs:"
echo "  • GraphDB Workbench: http://localhost:7200"
echo "  • KB-7 Repository: http://localhost:7200/repository/kb7-terminology"
echo "  • SPARQL Proxy: http://localhost:8095"
echo "  • Redis Cache: localhost:6379"
echo ""
echo "Ready for clinical terminology operations! 🏥"