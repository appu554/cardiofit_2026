#!/bin/bash

# Quick Patient Test Script
# Run ML predictions on preset patient scenarios

cd "$(dirname "$0")"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     CardioFit ML Inference - Quick Patient Test               ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Run the quick comparison test (option 4 from the menu)
mvn exec:java -Dexec.mainClass="ManualMLTest" \
              -Dexec.classpathScope="test" \
              -Dexec.args="4" \
              -q

echo ""
echo "✅ Test complete!"
echo ""
echo "To run the interactive version with custom patient data:"
echo "  mvn exec:java -Dexec.mainClass=\"ManualMLTest\" -Dexec.classpathScope=\"test\""
