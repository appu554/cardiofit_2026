#!/bin/bash

# Test PubMed API Key Configuration
# This script verifies that your NCBI API key is working correctly

API_KEY="3ddce7afddefb52bd45a79f3a4416dabaf0a"
TEST_PMID="26903338"  # Sepsis-3 study

echo "=========================================="
echo "PubMed API Key Verification Test"
echo "=========================================="
echo ""
echo "API Key: ${API_KEY:0:10}...${API_KEY: -5}"
echo "Test PMID: $TEST_PMID (Sepsis-3 study)"
echo ""

# Test 1: Fetch citation
echo "Test 1: Fetching citation metadata..."
RESPONSE=$(curl -s "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/efetch.fcgi?db=pubmed&id=$TEST_PMID&retmode=xml&api_key=$API_KEY")

if echo "$RESPONSE" | grep -q "<PubmedArticleSet>"; then
    echo "✅ SUCCESS: Citation fetched successfully"

    # Extract title
    TITLE=$(echo "$RESPONSE" | grep -oP '<ArticleTitle>\K[^<]+' | head -1)
    echo "   Title: $TITLE"

    # Extract journal
    JOURNAL=$(echo "$RESPONSE" | grep -oP '<Title>\K[^<]+' | head -1)
    echo "   Journal: $JOURNAL"

    # Extract year
    YEAR=$(echo "$RESPONSE" | grep -oP '<PubDate>.*?<Year>\K[^<]+' | head -1)
    echo "   Year: $YEAR"
else
    echo "❌ FAILED: Could not fetch citation"
    echo "   Response preview: ${RESPONSE:0:200}"
fi

echo ""

# Test 2: Search query
echo "Test 2: Testing search functionality..."
SEARCH_RESPONSE=$(curl -s "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&term=sepsis&retmax=5&api_key=$API_KEY")

if echo "$SEARCH_RESPONSE" | grep -q "<IdList>"; then
    COUNT=$(echo "$SEARCH_RESPONSE" | grep -oP '<Count>\K[^<]+' | head -1)
    echo "✅ SUCCESS: Search query executed"
    echo "   Found: $COUNT results for 'sepsis'"

    # Extract first few PMIDs
    PMIDS=$(echo "$SEARCH_RESPONSE" | grep -oP '<Id>\K[^<]+' | head -3)
    echo "   Sample PMIDs:"
    while IFS= read -r pmid; do
        echo "     - $pmid"
    done <<< "$PMIDS"
else
    echo "❌ FAILED: Search query failed"
fi

echo ""

# Test 3: Rate limit verification
echo "Test 3: Testing rate limits (10 requests/second with API key)..."
START_TIME=$(date +%s%N)

for i in {1..10}; do
    curl -s "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&id=$TEST_PMID&api_key=$API_KEY" > /dev/null
done

END_TIME=$(date +%s%N)
DURATION=$(( (END_TIME - START_TIME) / 1000000 ))  # Convert to milliseconds

echo "   10 requests completed in: ${DURATION}ms"

if [ $DURATION -lt 2000 ]; then
    echo "✅ SUCCESS: Rate limit allows 10 req/sec (expected: ~1000ms)"
else
    echo "⚠️  WARNING: Slower than expected (may indicate rate limiting)"
fi

echo ""
echo "=========================================="
echo "API Key Verification Complete"
echo "=========================================="
echo ""
echo "Next Steps:"
echo "1. If all tests passed, your API key is configured correctly"
echo "2. Set environment variable: export PUBMED_API_KEY=\"$API_KEY\""
echo "3. Start Evidence Repository service: mvn spring-boot:run"
echo "4. Test REST API: curl http://localhost:8015/api/citations/fetch/26903338"
echo ""
