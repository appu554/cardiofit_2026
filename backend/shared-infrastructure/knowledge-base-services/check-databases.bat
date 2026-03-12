@echo off
echo Checking database health...
echo.

echo MongoDB: 
docker exec kb-mongodb mongosh --quiet --eval "db.adminCommand('ping')" >nul 2>&1 && echo ✅ OK || echo ❌ Failed

echo Neo4j: 
curl -s -u neo4j:neo4j_password http://localhost:7474/db/data/ >nul 2>&1 && echo ✅ OK || echo ❌ Failed

echo TimescaleDB: 
docker exec kb-timescaledb pg_isready -U timescale >nul 2>&1 && echo ✅ OK || echo ❌ Failed

echo Elasticsearch: 
curl -s http://localhost:9200/_cluster/health | findstr "yellow green" >nul && echo ✅ OK || echo ❌ Failed

echo.
echo Health check complete!