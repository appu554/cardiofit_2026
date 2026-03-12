#!/bin/bash
# Apollo Federation Required Services Health Check

echo "🔍 Apollo Federation Required Services Health Check"
echo "================================================="

services=(
  "http://localhost:8003/api/federation|patients|Patient Service"
  "http://localhost:8005/api/federation|medications|Medication Service V2" 
  "http://localhost:8117/api/federation|context-gateway|Context Gateway"
  "http://localhost:8118/api/federation|clinical-data-hub|Clinical Data Hub"
  "http://localhost:8082/api/federation|kb2-clinical-context|KB2 Clinical Context"
  "http://localhost:8085/graphql|kb3-guidelines|KB3 Guidelines"
)

echo "Required Services:"
for service in "${services[@]}"; do
  IFS='|' read -r url name description <<< "$service"
  echo -n "  $description ($name) at $url ... "

  # Different testing for GraphQL endpoints vs regular endpoints
  if [[ "$url" == *"/graphql" ]]; then
    # Test GraphQL endpoint with POST request
    if curl -f -s --connect-timeout 5 -X POST "$url" -H "Content-Type: application/json" -d '{"query":"{ __typename }"}' > /dev/null 2>&1; then
      echo "✅ OK"
    else
      echo "❌ FAILED"
    fi
  elif [[ "$url" == *"/api/federation" ]]; then
    # Test Federation endpoint with POST request
    if curl -f -s --connect-timeout 5 -X POST "$url" -H "Content-Type: application/json" -d '{"query":"{ __typename }"}' > /dev/null 2>&1; then
      echo "✅ OK"
    else
      echo "❌ FAILED"
    fi
  else
    # Regular GET request for health endpoints
    if curl -f -s --connect-timeout 5 "$url" > /dev/null 2>&1; then
      echo "✅ OK"
    else
      echo "❌ FAILED"
    fi
  fi
done

echo ""
echo "Checking if any services are running on expected ports:"
netstat -an 2>/dev/null | grep -E ":(8003|8005|8117|8118|8082|8085)" | grep LISTEN || echo "No services found on expected ports"
