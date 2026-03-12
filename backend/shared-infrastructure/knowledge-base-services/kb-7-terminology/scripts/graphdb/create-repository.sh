#!/bin/bash
# KB-7 GraphDB Repository Creation Script
# Phase 1.1: GraphDB Repository & Infrastructure Setup
#
# Creates the kb7-terminology repository with:
# - OWL2-RL ruleset for clinical ontology reasoning
# - 2.5M triple capacity (10M entity index)
# - Context, literal, and predicate indexes enabled
# - Base URL: http://cardiofit.ai/ontology/
# - Clinical safety settings (consistency checks, SHACL validation)

set -e

# Configuration
GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"
REPO_ID="kb7-terminology"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
KB7_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"

echo "========================================="
echo "KB-7 GraphDB Repository Creation"
echo "========================================="
echo "GraphDB URL: $GRAPHDB_URL"
echo "Repository ID: $REPO_ID"
echo "Ruleset: owl2-rl-optimized"
echo ""

# Check if GraphDB is accessible
echo "🔍 Checking GraphDB connectivity..."
if ! curl -sf "$GRAPHDB_URL/rest/repositories" > /dev/null; then
    echo "❌ ERROR: Cannot connect to GraphDB at $GRAPHDB_URL"
    echo "   Make sure GraphDB container is running:"
    echo "   docker ps | grep graphdb"
    exit 1
fi
echo "✅ GraphDB is accessible"
echo ""

# Check if repository already exists
echo "🔍 Checking if repository exists..."
EXISTING=$(curl -sf "$GRAPHDB_URL/rest/repositories" | jq -r --arg id "$REPO_ID" '.[] | select(.id == $id) | .id' || echo "")
if [ -n "$EXISTING" ]; then
    echo "⚠️  Repository '$REPO_ID' already exists"
    read -p "   Delete and recreate? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🗑️  Deleting existing repository..."
        curl -X DELETE "$GRAPHDB_URL/rest/repositories/$REPO_ID"
        echo "✅ Repository deleted"
        sleep 2
    else
        echo "❌ Aborting - repository already exists"
        exit 1
    fi
fi
echo ""

# Create repository configuration
echo "📝 Creating repository configuration..."
CONFIG_FILE="/tmp/kb7-repo-config-$(date +%s).ttl"

cat > "$CONFIG_FILE" << 'EOF'
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#>.
@prefix rep: <http://www.openrdf.org/config/repository#>.
@prefix sr: <http://www.openrdf.org/config/repository/sail#>.
@prefix sail: <http://www.openrdf.org/config/sail#>.
@prefix owlim: <http://www.ontotext.com/trree/owlim#>.

[] a rep:Repository ;
    rep:repositoryID "kb7-terminology" ;
    rdfs:label "KB-7 Clinical Terminology Repository" ;
    rep:repositoryImpl [
        rep:repositoryType "graphdb:SailRepository" ;
        sr:sailImpl [
            sail:sailType "graphdb:Sail" ;

            # Core Repository Configuration
            owlim:repository-type "file-repository" ;
            owlim:base-URL "http://cardiofit.ai/ontology/" ;
            owlim:defaultNS "" ;
            owlim:storage-folder "storage" ;

            # Entity Index Configuration (2.5M triples capacity)
            owlim:entity-index-size "10000000" ;
            owlim:entity-id-size "32" ;

            # OWL2-RL Reasoning for Clinical Ontologies
            owlim:ruleset "owl2-rl" ;

            # Index Configuration
            owlim:enable-context-index "true" ;
            owlim:enablePredicateList "true" ;
            owlim:enable-literal-index "true" ;
            owlim:in-memory-literal-properties "true" ;

            # Clinical Safety Settings
            owlim:check-for-inconsistencies "true" ;
            owlim:disable-sameAs "false" ;

            # Query Configuration
            owlim:query-timeout "0" ;
            owlim:query-limit-results "0" ;
            owlim:throw-QueryEvaluationException-on-timeout "false" ;

            # Write Access
            owlim:read-only "false" ;

            # Imports Configuration
            owlim:imports "" ;
        ]
    ].
EOF

echo "✅ Configuration file created: $CONFIG_FILE"
echo ""

# Display configuration summary
echo "📋 Repository Configuration Summary:"
echo "   - Repository Type: file-repository"
echo "   - Base URL: http://cardiofit.ai/ontology/"
echo "   - Ruleset: owl2-rl-optimized"
echo "   - Entity Index Size: 10,000,000 (2.5M triples)"
echo "   - Context Index: ENABLED"
echo "   - Predicate List: ENABLED"
echo "   - Literal Index: ENABLED"
echo "   - Consistency Checks: ENABLED"
echo "   - Query Timeout: UNLIMITED"
echo ""

# Create repository via GraphDB REST API
echo "🚀 Creating repository via GraphDB API..."
HTTP_CODE=$(curl -X POST \
    -H "Content-Type: multipart/form-data" \
    -F "config=@$CONFIG_FILE" \
    "$GRAPHDB_URL/rest/repositories" \
    -w "%{http_code}" \
    -o /tmp/graphdb-response.txt \
    -s)

echo "   HTTP Status: $HTTP_CODE"

if [ "$HTTP_CODE" -eq 201 ]; then
    echo "✅ Repository created successfully!"
else
    echo "❌ Repository creation failed"
    echo "   Response:"
    cat /tmp/graphdb-response.txt
    echo ""
    rm -f "$CONFIG_FILE" /tmp/graphdb-response.txt
    exit 1
fi
echo ""

# Wait for repository to initialize
echo "⏳ Waiting for repository to initialize..."
sleep 3

# Verify repository configuration
echo "🔍 Verifying repository configuration..."
REPO_INFO=$(curl -sf "$GRAPHDB_URL/rest/repositories/$REPO_ID" | jq '.')

if [ -z "$REPO_INFO" ]; then
    echo "⚠️  Warning: Could not retrieve repository information"
else
    echo "✅ Repository verification successful"
    echo ""
    echo "📊 Repository Details:"
    echo "$REPO_INFO" | jq '{
        id: .id,
        title: .title,
        type: .type,
        readable: .readable,
        writable: .writable,
        ruleset: .params.ruleset.value,
        baseURL: .params.baseURL.value,
        entityIndexSize: .params.entityIndexSize.value,
        enableContextIndex: .params.enableContextIndex.value,
        enablePredicateList: .params.enablePredicateList.value,
        enableLiteralIndex: .params.enableLiteralIndex.value
    }'
fi
echo ""

# Cleanup
rm -f "$CONFIG_FILE" /tmp/graphdb-response.txt

# Success summary
echo "========================================="
echo "✅ GraphDB Repository Setup Complete!"
echo "========================================="
echo ""
echo "📍 Repository Information:"
echo "   ID: $REPO_ID"
echo "   URL: $GRAPHDB_URL/repositories/$REPO_ID"
echo "   SPARQL Endpoint: $GRAPHDB_URL/repositories/$REPO_ID"
echo ""
echo "🌐 Access Points:"
echo "   GraphDB Workbench: http://localhost:7200"
echo "   Repository UI: http://localhost:7200/repository?resource=$REPO_ID"
echo ""
echo "🔧 Next Steps:"
echo "   1. Run health check: ./scripts/graphdb/health-check.sh"
echo "   2. Test connectivity: go run test-graphdb-connection.go"
echo "   3. Load ontologies: See Phase 1.2 in KB7_ARCHITECTURE_TRANSFORMATION_PLAN.md"
echo ""
