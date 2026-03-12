@echo off
echo ==========================================
echo KB Services Database Deployment Script
echo ==========================================
echo.

echo Step 1: Starting missing databases...
docker-compose -f docker-compose.databases.yml up -d

echo.
echo Step 2: Waiting for services to be ready...
timeout /t 30 /nobreak > nul

echo.
echo Step 3: Checking service health...

echo Checking MongoDB...
docker exec kb-mongodb mongosh --quiet --eval "db.adminCommand('ping')" 2>nul && echo MongoDB: OK || echo MongoDB: Failed

echo Checking Neo4j...
curl -s -u neo4j:neo4j_password http://localhost:7474/db/data/ >nul 2>&1 && echo Neo4j: OK || echo Neo4j: Failed

echo Checking TimescaleDB...
docker exec kb-timescaledb pg_isready -U timescale >nul 2>&1 && echo TimescaleDB: OK || echo TimescaleDB: Failed

echo Checking Elasticsearch...
curl -s http://localhost:9200/_cluster/health | findstr "yellow green" >nul && echo Elasticsearch: OK || echo Elasticsearch: Failed

echo.
echo Step 4: Initializing databases...

echo Initializing MongoDB...
docker exec -it kb-mongodb mongosh -u admin -p mongodb_admin_password --authenticationDatabase admin /docker-entrypoint-initdb.d/01-init-kb-clinical-context.js

echo Initializing Neo4j...
docker exec -it kb-neo4j cypher-shell -u neo4j -p neo4j_password < init-scripts/neo4j/01-init-guidelines.cypher

echo Initializing Elasticsearch...
docker exec -it kb-elasticsearch bash /usr/share/elasticsearch/init/01-init-formulary.sh

echo.
echo ==========================================
echo Database Deployment Complete!
echo ==========================================
echo.
echo Access Web UIs:
echo MongoDB Express: http://localhost:8090
echo Neo4j Browser: http://localhost:7474
echo Kibana: http://localhost:5601
echo.
echo To test connections manually:
echo mongosh "mongodb://kb_context_user:kb_context_password@localhost:27017/kb_clinical_context"
echo curl -u neo4j:neo4j_password http://localhost:7474/db/data/
echo psql -h localhost -p 5434 -U kb_safety_user -d kb_patient_safety
echo curl -X GET "localhost:9200/_cluster/health?pretty"
echo.