#!/bin/bash

# Setup Neo4j password for Module 2
# This script changes the default Neo4j password to CardioFit2024!

echo "Setting up Neo4j password for Module 2..."

# Method 1: Try via cypher-shell with password change
echo "ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';" | docker exec -i neo4j cypher-shell -u neo4j -p neo4j 2>/dev/null

if [ $? -eq 0 ]; then
    echo "✅ Password changed successfully!"

    # Verify new password works
    echo "RETURN 'Connection successful' AS status;" | docker exec -i neo4j cypher-shell -u neo4j -p 'CardioFit2024!' 2>/dev/null

    if [ $? -eq 0 ]; then
        echo "✅ New password verified!"
    else
        echo "❌ New password verification failed"
    fi
else
    echo "❌ Failed to change password via cypher-shell"
    echo ""
    echo "Please run this command manually:"
    echo "docker exec -it neo4j cypher-shell -u neo4j -p neo4j"
    echo "Then run: ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';"
fi

echo ""
echo "Neo4j Configuration for Module 2:"
echo "  - Bolt URI (local): bolt://localhost:55002"
echo "  - Bolt URI (Docker): bolt://neo4j:7687"
echo "  - Username: neo4j"
echo "  - Password: CardioFit2024!"
