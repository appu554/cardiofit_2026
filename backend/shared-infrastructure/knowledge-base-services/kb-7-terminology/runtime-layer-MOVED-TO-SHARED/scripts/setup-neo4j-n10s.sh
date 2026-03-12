#!/bin/bash
#===============================================================================
# KB7 Neo4j + N10s Setup Script
#===============================================================================
#
# This script sets up Neo4j with the Neosemantics (n10s) plugin for KB7
# terminology synchronization.
#
# What it does:
# 1. Stops existing Neo4j container (if running)
# 2. Downloads n10s plugin JAR
# 3. Creates new Neo4j container with n10s plugin
# 4. Initializes n10s graph configuration
#
# Usage:
#   ./setup-neo4j-n10s.sh [--password YOUR_PASSWORD]
#
#===============================================================================

set -e

# Configuration
NEO4J_PASSWORD="${1:-kb7password}"
NEO4J_CONTAINER="kb7-neo4j"
NEO4J_VERSION="5.12.0"
N10S_VERSION="5.20.0"  # Compatible with Neo4j 5.x
NEO4J_HTTP_PORT=7474
NEO4J_BOLT_PORT=7687

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║           KB7 Neo4j + N10s Setup Script                       ║"
echo "║                                                               ║"
echo "║  Creates Neo4j with neosemantics plugin for RDF import        ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Create directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="${SCRIPT_DIR}/../data/neo4j"
PLUGINS_DIR="${DATA_DIR}/plugins"

echo -e "${YELLOW}📁 Creating directories...${NC}"
mkdir -p "${DATA_DIR}/data"
mkdir -p "${DATA_DIR}/logs"
mkdir -p "${DATA_DIR}/import"
mkdir -p "${PLUGINS_DIR}"

# Download n10s plugin
N10S_JAR="neosemantics-${N10S_VERSION}.jar"
N10S_URL="https://github.com/neo4j-labs/neosemantics/releases/download/${N10S_VERSION}/${N10S_JAR}"

if [ ! -f "${PLUGINS_DIR}/${N10S_JAR}" ]; then
    echo -e "${YELLOW}📥 Downloading n10s plugin v${N10S_VERSION}...${NC}"
    curl -L -o "${PLUGINS_DIR}/${N10S_JAR}" "${N10S_URL}" || {
        echo -e "${RED}❌ Failed to download n10s plugin${NC}"
        echo "Please download manually from: ${N10S_URL}"
        exit 1
    }
    echo -e "${GREEN}✅ n10s plugin downloaded${NC}"
else
    echo -e "${GREEN}✅ n10s plugin already exists${NC}"
fi

# Stop existing container if running
echo -e "${YELLOW}🛑 Checking for existing container...${NC}"
if docker ps -a --format '{{.Names}}' | grep -q "^${NEO4J_CONTAINER}$"; then
    echo "   Stopping existing ${NEO4J_CONTAINER}..."
    docker stop ${NEO4J_CONTAINER} 2>/dev/null || true
    docker rm ${NEO4J_CONTAINER} 2>/dev/null || true
fi

# Also stop the old 'neo4j' container if it's using our ports
if docker ps --format '{{.Names}}' | grep -q "^neo4j$"; then
    EXISTING_PORT=$(docker port neo4j 7474 2>/dev/null || echo "")
    if [[ "$EXISTING_PORT" == *"$NEO4J_HTTP_PORT"* ]]; then
        echo -e "${YELLOW}⚠️  Existing 'neo4j' container is using port ${NEO4J_HTTP_PORT}${NC}"
        read -p "   Stop it and continue? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker stop neo4j
        else
            echo "   Using alternative ports: HTTP=7475, Bolt=7688"
            NEO4J_HTTP_PORT=7475
            NEO4J_BOLT_PORT=7688
        fi
    fi
fi

# Create Neo4j container with n10s
echo -e "${YELLOW}🚀 Starting Neo4j with n10s plugin...${NC}"

docker run -d \
    --name ${NEO4J_CONTAINER} \
    -p ${NEO4J_HTTP_PORT}:7474 \
    -p ${NEO4J_BOLT_PORT}:7687 \
    -v "${DATA_DIR}/data:/data" \
    -v "${DATA_DIR}/logs:/logs" \
    -v "${DATA_DIR}/import:/var/lib/neo4j/import" \
    -v "${PLUGINS_DIR}:/plugins" \
    -e NEO4J_AUTH="neo4j/${NEO4J_PASSWORD}" \
    -e NEO4J_dbms_security_procedures_unrestricted="n10s.*" \
    -e NEO4J_dbms_security_procedures_allowlist="n10s.*" \
    -e NEO4J_dbms_memory_heap_initial__size="1G" \
    -e NEO4J_dbms_memory_heap_max__size="2G" \
    -e NEO4J_dbms_memory_pagecache_size="1G" \
    neo4j:${NEO4J_VERSION}-community

echo -e "${YELLOW}⏳ Waiting for Neo4j to start...${NC}"

# Wait for Neo4j to be ready
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${NEO4J_HTTP_PORT}" | grep -q "200"; then
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    echo "   Attempt $ATTEMPT/$MAX_ATTEMPTS..."
    sleep 2
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo -e "${RED}❌ Neo4j failed to start within timeout${NC}"
    docker logs ${NEO4J_CONTAINER}
    exit 1
fi

echo -e "${GREEN}✅ Neo4j is running${NC}"

# Wait a bit more for full initialization
sleep 5

# Verify n10s is installed
echo -e "${YELLOW}🔍 Verifying n10s installation...${NC}"

N10S_VERSION_CHECK=$(curl -s -u "neo4j:${NEO4J_PASSWORD}" \
    "http://localhost:${NEO4J_HTTP_PORT}/db/neo4j/tx/commit" \
    -H "Content-Type: application/json" \
    -d '{"statements":[{"statement":"RETURN n10s.version() as version"}]}' \
    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('results',[{}])[0].get('data',[{}])[0].get('row',['ERROR'])[0]) if not d.get('errors') else print('ERROR: ' + d['errors'][0]['message'][:50])" 2>/dev/null)

if [[ "$N10S_VERSION_CHECK" == ERROR* ]]; then
    echo -e "${RED}❌ n10s verification failed: ${N10S_VERSION_CHECK}${NC}"
    echo "   Check Neo4j logs: docker logs ${NEO4J_CONTAINER}"
    exit 1
fi

echo -e "${GREEN}✅ n10s version: ${N10S_VERSION_CHECK}${NC}"

# Initialize n10s graph config
echo -e "${YELLOW}🔧 Initializing n10s graph configuration...${NC}"

# Create constraint
curl -s -u "neo4j:${NEO4J_PASSWORD}" \
    "http://localhost:${NEO4J_HTTP_PORT}/db/neo4j/tx/commit" \
    -H "Content-Type: application/json" \
    -d '{"statements":[{"statement":"CREATE CONSTRAINT n10s_unique_uri IF NOT EXISTS FOR (r:Resource) REQUIRE r.uri IS UNIQUE"}]}' > /dev/null

# Initialize graph config
INIT_RESULT=$(curl -s -u "neo4j:${NEO4J_PASSWORD}" \
    "http://localhost:${NEO4J_HTTP_PORT}/db/neo4j/tx/commit" \
    -H "Content-Type: application/json" \
    -d '{"statements":[{"statement":"CALL n10s.graphconfig.init({handleVocabUris: \"SHORTEN\", applyNeo4jNaming: true, multivalPropList: [\"http://www.w3.org/2000/01/rdf-schema#label\"]})"}]}')

if echo "$INIT_RESULT" | grep -q '"errors":\[\]'; then
    echo -e "${GREEN}✅ n10s graph configuration initialized${NC}"
else
    # May already be initialized
    echo -e "${YELLOW}⚠️  Graph config may already exist (this is OK)${NC}"
fi

# Summary
echo ""
echo -e "${GREEN}"
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                    SETUP COMPLETE!                            ║"
echo "╠═══════════════════════════════════════════════════════════════╣"
echo "║                                                               ║"
echo "║  Neo4j Browser: http://localhost:${NEO4J_HTTP_PORT}                        ║"
echo "║  Bolt URI:      bolt://localhost:${NEO4J_BOLT_PORT}                        ║"
echo "║  Username:      neo4j                                         ║"
echo "║  Password:      ${NEO4J_PASSWORD}                                    ║"
echo "║                                                               ║"
echo "║  n10s Version:  ${N10S_VERSION_CHECK}                                     ║"
echo "║                                                               ║"
echo "╠═══════════════════════════════════════════════════════════════╣"
echo "║  To test the pipeline, run:                                   ║"
echo "║                                                               ║"
echo "║  python tests/test_kb7_n10s_pipeline.py \\                     ║"
echo "║      --neo4j-password ${NEO4J_PASSWORD} \\                            ║"
echo "║      --neo4j-uri bolt://localhost:${NEO4J_BOLT_PORT}                       ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"
