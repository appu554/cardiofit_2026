#!/bin/bash
# Create GraphDB repository for KB-7 Terminology Service
# Repository: kb7-terminology
# Ruleset: rdfs-plus-optimized (kernel is already materialized via ELK reasoning)

set -e

GRAPHDB_URL="http://localhost:7200"
REPO_ID="kb7-terminology"

echo "=== Creating GraphDB Repository: $REPO_ID ==="

# Repository configuration with OWL2-RL ruleset
cat > /tmp/kb7-repo-config.ttl << 'EOF'
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

            owlim:base-URL "http://cardiofit.ai/ontology/" ;
            owlim:defaultNS "" ;
            owlim:entity-index-size "10000000" ;
            owlim:entity-id-size "32" ;
            owlim:imports "" ;
            owlim:repository-type "file-repository" ;
            owlim:ruleset "rdfs-plus-optimized" ;
            owlim:storage-folder "storage" ;

            owlim:enable-context-index "true" ;
            owlim:enablePredicateList "true" ;
            owlim:in-memory-literal-properties "true" ;
            owlim:enable-literal-index "true" ;
            owlim:check-for-inconsistencies "false" ;
            owlim:disable-sameAs "false" ;
            owlim:query-timeout "0" ;
            owlim:query-limit-results "0" ;
            owlim:throw-QueryEvaluationException-on-timeout "false" ;
            owlim:read-only "false" ;
        ]
    ].
EOF

echo "📝 Repository configuration created at /tmp/kb7-repo-config.ttl"

# Create repository via GraphDB REST API
echo "🚀 Creating repository via GraphDB API..."

curl -X POST \
    -H "Content-Type: multipart/form-data" \
    -F "config=@/tmp/kb7-repo-config.ttl" \
    "$GRAPHDB_URL/rest/repositories" \
    -w "\nHTTP Status: %{http_code}\n" \
    -o /tmp/graphdb-response.txt

echo ""
echo "📄 Response:"
cat /tmp/graphdb-response.txt
echo ""

# Verify repository was created
echo "✅ Verifying repository..."
curl -s "$GRAPHDB_URL/rest/repositories/$REPO_ID" | jq '.' || echo "⚠️  Repository verification failed - may need manual check"

echo ""
echo "🎉 GraphDB repository creation complete!"
echo "📍 Repository ID: $REPO_ID"
echo "🌐 GraphDB UI: http://localhost:7200"
echo "🔍 SPARQL Endpoint: http://localhost:7200/repositories/$REPO_ID"

# Cleanup
rm -f /tmp/kb7-repo-config.ttl /tmp/graphdb-response.txt

echo ""
echo "Next steps:"
echo "1. Visit http://localhost:7200 to verify repository in UI"
echo "2. Run test-graphdb-connection.go to validate connectivity"
echo "3. Load sample SNOMED CT data for proof-of-concept"
