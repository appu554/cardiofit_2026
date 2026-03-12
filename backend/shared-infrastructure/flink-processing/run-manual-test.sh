#!/bin/bash

# Manual ML Testing Tool - Easy Runner
# This script compiles and runs the interactive ML testing tool

cd "$(dirname "$0")"

echo "════════════════════════════════════════════════════════════════"
echo "  CardioFit ML Inference - Manual Testing Tool"
echo "════════════════════════════════════════════════════════════════"
echo ""

# Check if models exist
if [ ! -f "models/sepsis_risk_v1.0.0.onnx" ]; then
    echo "❌ Error: ONNX models not found in models/ directory"
    echo ""
    echo "Please ensure the following models exist:"
    echo "  - models/sepsis_risk_v1.0.0.onnx"
    echo "  - models/deterioration_risk_v1.0.0.onnx"
    echo "  - models/mortality_risk_v1.0.0.onnx"
    echo "  - models/readmission_risk_v1.0.0.onnx"
    echo ""
    exit 1
fi

echo "✅ ONNX models found"
echo ""

# Compile if needed
if [ ! -f "ManualMLTest.class" ] || [ "ManualMLTest.java" -nt "ManualMLTest.class" ]; then
    echo "🔨 Compiling ManualMLTest.java..."

    # Build classpath
    CP="target/classes:target/test-classes"
    CP="$CP:$HOME/.m2/repository/com/microsoft/onnxruntime/onnxruntime/1.17.0/onnxruntime-1.17.0.jar"
    CP="$CP:$HOME/.m2/repository/org/apache/flink/flink-streaming-java/1.18.0/flink-streaming-java-1.18.0.jar"
    CP="$CP:$HOME/.m2/repository/org/apache/flink/flink-core/1.18.0/flink-core-1.18.0.jar"

    javac -cp "$CP" ManualMLTest.java

    if [ $? -ne 0 ]; then
        echo "❌ Compilation failed"
        exit 1
    fi

    echo "✅ Compilation successful"
    echo ""
fi

# Run the test
echo "🚀 Starting interactive ML testing tool..."
echo ""

# Build runtime classpath
CP=".:target/classes:target/test-classes"
CP="$CP:$HOME/.m2/repository/com/microsoft/onnxruntime/onnxruntime/1.17.0/onnxruntime-1.17.0.jar"
CP="$CP:$HOME/.m2/repository/org/apache/flink/flink-streaming-java/1.18.0/flink-streaming-java-1.18.0.jar"
CP="$CP:$HOME/.m2/repository/org/apache/flink/flink-core/1.18.0/flink-core-1.18.0.jar"
CP="$CP:$HOME/.m2/repository/org/slf4j/slf4j-api/2.0.9/slf4j-api-2.0.9.jar"

java -cp "$CP" ManualMLTest

echo ""
echo "👋 Thanks for using CardioFit ML Testing Tool!"
