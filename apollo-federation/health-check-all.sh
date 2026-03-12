#!/bin/bash
# health-check-all.sh from FULL_ECOSYSTEM_TESTING.md

services=(
  "http://localhost:4000/graphql"     # Apollo Federation
  "http://localhost:8001/health"      # Auth Service
  "http://localhost:8003/health"      # Patient Service
  "http://localhost:8004/health"      # Medication Service V2
  "http://localhost:8016/health"      # Context Gateway
  "http://localhost:8018/health"      # Safety Gateway
  "http://localhost:8080/health"      # Flow2 Go Engine
  "http://localhost:8090/health"      # Flow2 Rust Engine
  "http://localhost:8020/health"      # Workflow Engine
)

echo "🔍 Testing all service health endpoints..."
for service in "${services[@]}"; do
  echo -n "Testing $service ... "
  if curl -f -s --connect-timeout 5 "$service" > /dev/null 2>&1; then
    echo "✅ OK"
  else
    echo "❌ FAILED"
  fi
done
