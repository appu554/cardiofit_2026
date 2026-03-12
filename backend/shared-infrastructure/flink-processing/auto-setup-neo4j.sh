#!/bin/bash

# Automated Neo4j Password Setup for Module 2
# This script changes the Neo4j password non-interactively using expect

set -e

echo "🔧 Automated Neo4j Password Setup for Module 2"
echo "=============================================="
echo ""

# Check if expect is installed
if ! command -v expect &> /dev/null; then
    echo "⚠️  'expect' is not installed. Installing..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        brew install expect 2>/dev/null || echo "Please install expect: brew install expect"
    else
        # Linux
        sudo apt-get install -y expect 2>/dev/null || echo "Please install expect: sudo apt-get install expect"
    fi
fi

# Create expect script
cat > /tmp/neo4j_password_change.exp << 'EOF'
#!/usr/bin/expect -f

set timeout 10
spawn docker exec -it neo4j cypher-shell -u neo4j -p neo4j

expect {
    "password must be changed" {
        send "ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';\r"
        expect eof
    }
    "Connected to Neo4j" {
        send "ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';\r"
        expect eof
    }
    timeout {
        puts "Connection timeout"
        exit 1
    }
}
EOF

chmod +x /tmp/neo4j_password_change.exp

# Run the expect script
echo "📝 Attempting to change Neo4j password..."
if /tmp/neo4j_password_change.exp; then
    echo "✅ Password change script executed"
else
    echo "❌ Password change script failed"
fi

# Clean up
rm -f /tmp/neo4j_password_change.exp

echo ""
echo "🔍 Verifying new password..."

# Verify the new password works
if echo "RETURN 'Connection successful!' AS status;" | docker exec -i neo4j cypher-shell -u neo4j -p 'CardioFit2024!' 2>&1 | grep -q "Connection successful"; then
    echo "✅ Password verified successfully!"
    echo ""
    echo "✅ Neo4j is now configured for Module 2:"
    echo "   - Bolt URI (local): bolt://localhost:55002"
    echo "   - Bolt URI (Docker): bolt://neo4j:7687"
    echo "   - Username: neo4j"
    echo "   - Password: CardioFit2024!"
    echo ""
    echo "🚀 Ready for Module 2 Phase 5 testing!"
    exit 0
else
    echo ""
    echo "⚠️  Automated password change may have failed."
    echo ""
    echo "Please change the password manually:"
    echo "  1. Run: docker exec -it neo4j cypher-shell -u neo4j -p neo4j"
    echo "  2. Execute: ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';"
    echo "  3. Type: :exit"
    echo ""
    exit 1
fi
