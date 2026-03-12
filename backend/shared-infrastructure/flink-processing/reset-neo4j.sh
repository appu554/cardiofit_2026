#!/bin/bash
# Reset Neo4j with correct password from docker-compose
# This script stops Neo4j, removes old data volume, and recreates with CardioFit2024! password

set -e

echo "🔄 Resetting Neo4j container and volumes..."

# Stop and remove Neo4j container
echo "⏹️  Stopping Neo4j container..."
docker stop neo4j || true
docker rm neo4j || true

# Find and remove Neo4j data volumes
echo "🗑️  Removing old Neo4j data volumes..."
NEO4J_DATA_VOLUME=$(docker volume ls -q | xargs docker volume inspect 2>/dev/null | \
  jq -r '.[] | select(.Mountpoint | contains("neo4j")) | select(.Mountpoint | contains("/data")) | .Name' | head -n1)

NEO4J_LOGS_VOLUME=$(docker volume ls -q | xargs docker volume inspect 2>/dev/null | \
  jq -r '.[] | select(.Mountpoint | contains("neo4j")) | select(.Mountpoint | contains("/logs")) | .Name' | head -n1)

if [ -n "$NEO4J_DATA_VOLUME" ]; then
  echo "  Removing data volume: $NEO4J_DATA_VOLUME"
  docker volume rm "$NEO4J_DATA_VOLUME" || true
fi

if [ -n "$NEO4J_LOGS_VOLUME" ]; then
  echo "  Removing logs volume: $NEO4J_LOGS_VOLUME"
  docker volume rm "$NEO4J_LOGS_VOLUME" || true
fi

# Restart Neo4j (docker-compose will create with NEO4J_AUTH=neo4j/CardioFit2024!)
echo "🚀 Starting fresh Neo4j container..."
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure
docker-compose -f docker-compose.hybrid-kafka.yml up -d neo4j

echo "⏳ Waiting for Neo4j to initialize (30 seconds)..."
sleep 30

# Verify password works
echo "✅ Testing connection with new password..."
if docker exec -i neo4j cypher-shell -u neo4j -p 'CardioFit2024!' -d neo4j "RETURN 'Password works!' AS status;" 2>/dev/null; then
  echo "✅ Neo4j reset successfully! Password is now: CardioFit2024!"
  echo ""
  echo "📊 Neo4j connection details:"
  echo "   URI: bolt://localhost:55002"
  echo "   Username: neo4j"
  echo "   Password: CardioFit2024!"
  echo "   Web UI: http://localhost:55001"
else
  echo "⚠️  Connection test failed. Check logs:"
  echo "   docker logs neo4j"
fi
