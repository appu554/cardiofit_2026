#!/bin/bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

export DATABASE_URL="postgres://postgres:password@localhost:5432/kb_terminology?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export GRAPHDB_ENABLED=false
export NEO4J_MULTI_REGION_ENABLED=false

echo "Starting KB7 service to test migrations..."
go run ./cmd/server/main.go 2>&1
