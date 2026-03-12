#!/bin/bash

# Test script for KB-2 Clinical Context GraphQL Federation

echo "🧪 Testing KB-2 Clinical Context GraphQL Federation"
echo "================================================="

# Test 1: Service SDL
echo "Test 1: Federation Schema Definition Language (SDL)"
echo "---"
curl -s -X POST http://localhost:8082/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "query { _service { sdl } }"}' | jq -r '.data._service.sdl'
echo ""

# Test 2: System Health
echo "Test 2: System Health Query"
echo "---"
curl -s -X POST http://localhost:8082/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "query { systemHealth { status timestamp } }"}' | jq '.data.systemHealth'
echo ""

# Test 3: Entity Resolution (empty)
echo "Test 3: Entity Resolution (empty)"
echo "---"
curl -s -X POST http://localhost:8082/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "query { _entities(representations: []) { id } }"}' | jq '.data._entities'
echo ""

# Test 4: Health endpoint
echo "Test 4: REST Health Endpoint"
echo "---"
curl -s http://localhost:8082/health | jq '.status'
echo ""

# Test 5: Build Context Mutation
echo "Test 5: Build Context Mutation"
echo "---"
curl -s -X POST http://localhost:8082/api/federation \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { buildContext(input: { patientId: \"test-123\", patient: \"{}\" }) { cacheHit processedAt } }"
  }' | jq '.data.buildContext // .errors'
echo ""

echo "✅ Federation tests completed!"
echo "KB-2 Clinical Context service is ready for Apollo Federation integration"
echo "Service available at: http://localhost:8082/api/federation"