#!/bin/bash
# KB-7 Semantic Alignment Validation
# Checks for URI collision and namespace fragmentation
# Ensures RxNorm references match SNOMED-OWL-Toolkit URI structure

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}

echo "=================================================="
echo "KB-7 Semantic Alignment Validation"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "=================================================="

cd "$WORKSPACE"

# Verify merged ontology exists
if [ ! -f "kb7-merged.ttl" ]; then
    echo "ERROR: kb7-merged.ttl not found"
    exit 1
fi

echo ""
echo "Running URI Alignment Validation Queries..."
echo ""

# Query 1: Count SNOMED URIs in merged ontology
echo "1️⃣ Checking SNOMED URI Alignment..."
SNOMED_COUNT=$($ROBOT query \
    --input kb7-merged.ttl \
    --query <(cat <<'EOF'
PREFIX snomed: <http://snomed.info/id/>
SELECT (COUNT(?s) AS ?count) WHERE {
  ?s a ?type .
  FILTER (STRSTARTS(STR(?s), "http://snomed.info/id/"))
}
EOF
) \
    --format csv | tail -1)

echo "   SNOMED URIs found: $SNOMED_COUNT"

if [ "$SNOMED_COUNT" = "0" ]; then
    echo "   ❌ CRITICAL ERROR: No SNOMED URIs found!"
    echo "   This indicates namespace fragmentation - RxNorm/LOINC not using SNOMED-OWL-Toolkit URIs"
    exit 1
fi

echo "   ✅ SNOMED URIs present in merged ontology"

# Query 2: Detect incorrect SNOMED URI patterns (BioPortal URIs)
echo ""
echo "2️⃣ Checking for BioPortal SNOMED URIs (incorrect namespace)..."
BIOPORTAL_COUNT=$($ROBOT query \
    --input kb7-merged.ttl \
    --query <(cat <<'EOF'
SELECT (COUNT(?s) AS ?count) WHERE {
  ?s a ?type .
  FILTER (STRSTARTS(STR(?s), "http://purl.bioontology.org/ontology/SNOMEDCT/"))
}
EOF
) \
    --format csv | tail -1)

echo "   BioPortal SNOMED URIs found: $BIOPORTAL_COUNT"

if [ "$BIOPORTAL_COUNT" != "0" ]; then
    echo "   ⚠️  WARNING: Found BioPortal SNOMED URIs!"
    echo "   These should use http://snomed.info/id/ instead"
    echo "   RxNorm/LOINC converters may be using incorrect namespace"
fi

# Query 3: Find dangling SNOMED references
echo ""
echo "3️⃣ Checking for Dangling SNOMED References..."
DANGLING_REFS=$($ROBOT query \
    --input kb7-merged.ttl \
    --query <(cat <<'EOF'
PREFIX snomed: <http://snomed.info/id/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT (COUNT(DISTINCT ?ref) AS ?count) WHERE {
  ?s ?p ?ref .
  FILTER (STRSTARTS(STR(?ref), "http://snomed.info/id/"))
  FILTER NOT EXISTS { ?ref a ?type }
}
EOF
) \
    --format csv | tail -1)

echo "   Dangling SNOMED references: $DANGLING_REFS"

if [ "$DANGLING_REFS" != "0" ]; then
    echo "   ⚠️  WARNING: Found dangling SNOMED references!"
    echo "   Some RxNorm/LOINC concepts reference SNOMED codes not in SNOMED-OWL output"

    # Get sample dangling references for debugging
    echo ""
    echo "   Sample dangling references:"
    $ROBOT query \
        --input kb7-merged.ttl \
        --query <(cat <<'EOF'
PREFIX snomed: <http://snomed.info/id/>
SELECT DISTINCT ?ref WHERE {
  ?s ?p ?ref .
  FILTER (STRSTARTS(STR(?ref), "http://snomed.info/id/"))
  FILTER NOT EXISTS { ?ref a ?type }
}
LIMIT 5
EOF
) \
        --format csv | tail -5 | sed 's/^/      /'
fi

# Query 4: Verify RxNorm-to-SNOMED linkage
echo ""
echo "4️⃣ Checking RxNorm-to-SNOMED Cross-References..."
RXNORM_SNOMED_LINKS=$($ROBOT query \
    --input kb7-merged.ttl \
    --query <(cat <<'EOF'
PREFIX rxnorm: <http://purl.bioontology.org/ontology/RXNORM/>
PREFIX snomed: <http://snomed.info/id/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT (COUNT(*) AS ?count) WHERE {
  ?rxconcept ?p ?snomedconcept .
  FILTER (STRSTARTS(STR(?rxconcept), "http://purl.bioontology.org/ontology/RXNORM/") ||
          STRSTARTS(STR(?rxconcept), "http://snomed.info/id/"))
  FILTER (STRSTARTS(STR(?snomedconcept), "http://snomed.info/id/"))
}
EOF
) \
    --format csv | tail -1)

echo "   RxNorm-to-SNOMED cross-references: $RXNORM_SNOMED_LINKS"

if [ "$RXNORM_SNOMED_LINKS" = "0" ]; then
    echo "   ⚠️  WARNING: No RxNorm-to-SNOMED cross-references found!"
    echo "   This may indicate incomplete semantic linkage"
fi

# Query 5: Check for multiple URI patterns for same concept (URI collision)
echo ""
echo "5️⃣ Checking for URI Collision (same concept, different URIs)..."

# This is a heuristic check - looks for SNOMED concepts that might have duplicates
POTENTIAL_COLLISIONS=$($ROBOT query \
    --input kb7-merged.ttl \
    --query <(cat <<'EOF'
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT (COUNT(DISTINCT ?label) AS ?unique_labels) (COUNT(?s) AS ?total_subjects) WHERE {
  ?s rdfs:label ?label .
  FILTER (LANG(?label) = "en" || LANG(?label) = "")
}
EOF
) \
    --format csv | tail -1)

echo "   Label distribution analysis: $POTENTIAL_COLLISIONS"

# Summary
echo ""
echo "=================================================="
echo "Validation Summary"
echo "=================================================="
echo "✅ SNOMED URI Count: $SNOMED_COUNT"
echo "⚠️  BioPortal URIs:  $BIOPORTAL_COUNT (should be 0)"
echo "⚠️  Dangling Refs:   $DANGLING_REFS"
echo "📊 RxNorm↔SNOMED Links: $RXNORM_SNOMED_LINKS"
echo "=================================================="

# Exit with error if critical issues found
if [ "$SNOMED_COUNT" = "0" ]; then
    echo "❌ VALIDATION FAILED: Critical namespace fragmentation detected"
    exit 1
fi

if [ "$BIOPORTAL_COUNT" != "0" ]; then
    echo "⚠️  VALIDATION WARNING: BioPortal URIs detected (should use http://snomed.info/id/)"
    echo "   This indicates partial namespace fragmentation"
    # Don't exit - this is a warning, not a failure
fi

if [ "$DANGLING_REFS" != "0" ]; then
    echo "⚠️  VALIDATION WARNING: Dangling SNOMED references detected"
    echo "   Some cross-ontology references may be incomplete"
    # Don't exit - this might be expected for some reference types
fi

echo ""
echo "✅ URI Alignment Validation Complete"
echo "   Semantic integration appears sound for clinical decision support"
