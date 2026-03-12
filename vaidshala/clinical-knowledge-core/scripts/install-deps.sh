#!/bin/bash
# ============================================================
# Dependency Installation Script
# ============================================================
# Installs all required dependencies for local development.
#
# Usage: ./install-deps.sh
# ============================================================

set -e

echo "Installing Dependencies"
echo "========================"
echo ""

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Darwin*)    PLATFORM="macos" ;;
    Linux*)     PLATFORM="linux" ;;
    *)          PLATFORM="unknown" ;;
esac

echo "Detected platform: $PLATFORM"
echo ""

# Check for Java 17+
echo "Checking Java..."
if command -v java &> /dev/null; then
    JAVA_VERSION=$(java -version 2>&1 | head -1 | cut -d'"' -f2 | cut -d'.' -f1)
    if [ "$JAVA_VERSION" -ge 17 ] 2>/dev/null; then
        echo "   Java $JAVA_VERSION found"
    else
        echo "   Java 17+ required (found version $JAVA_VERSION)"
        echo "   Install: https://adoptium.net/"
        exit 1
    fi
else
    echo "   Java not found. Install Java 17+:"
    echo "   https://adoptium.net/"
    exit 1
fi

# Check for jq
echo "Checking jq..."
if command -v jq &> /dev/null; then
    echo "   jq found: $(jq --version)"
else
    echo "   Installing jq..."
    if [ "$PLATFORM" = "macos" ]; then
        brew install jq
    elif [ "$PLATFORM" = "linux" ]; then
        sudo apt-get update && sudo apt-get install -y jq
    else
        echo "   Please install jq manually: https://stedolan.github.io/jq/"
        exit 1
    fi
fi

# Check for OpenSSL
echo "Checking OpenSSL..."
if command -v openssl &> /dev/null; then
    echo "   OpenSSL found: $(openssl version)"
else
    echo "   OpenSSL not found. Please install OpenSSL."
    exit 1
fi

# Check for GitHub CLI (optional)
echo "Checking GitHub CLI (optional)..."
if command -v gh &> /dev/null; then
    echo "   GitHub CLI found: $(gh --version | head -1)"
else
    echo "   GitHub CLI not found (needed for publishing)"
    echo "   Install: https://cli.github.com/"
fi

# Download CQL-to-ELM translator
CQL_VERSION="2.11.0"
CQL_DIR="$HOME/.cql-translator"
CQL_JAR="$CQL_DIR/cql-to-elm-$CQL_VERSION.jar"

echo ""
echo "Checking CQL-to-ELM translator..."
if [ -f "$CQL_JAR" ]; then
    echo "   CQL translator found: $CQL_JAR"
else
    echo "   Downloading CQL-to-ELM translator v$CQL_VERSION..."
    mkdir -p "$CQL_DIR"

    # Try primary download location
    URL="https://github.com/cqframework/clinical_quality_language/releases/download/v${CQL_VERSION}/cql-to-elm-${CQL_VERSION}.jar"

    if command -v curl &> /dev/null; then
        curl -fsSL "$URL" -o "$CQL_JAR" 2>/dev/null || {
            echo "   Primary download failed, trying alternative..."
            # Alternative: Maven Central
            ALT_URL="https://repo1.maven.org/maven2/info/cqframework/cql-to-elm/${CQL_VERSION}/cql-to-elm-${CQL_VERSION}.jar"
            curl -fsSL "$ALT_URL" -o "$CQL_JAR" || {
                echo "   Could not download CQL translator."
                echo "   Download manually from: $URL"
                exit 1
            }
        }
        echo "   Downloaded successfully!"
    else
        echo "   curl not found. Please download CQL translator manually:"
        echo "   $URL"
        exit 1
    fi
fi

echo ""
echo "All dependencies installed!"
echo ""
echo "Next steps:"
echo "   1. Run 'make build' to compile CQL and expand ValueSets"
echo "   2. Run 'make generate-keys' to create signing keypair"
echo "   3. Run 'make sign' to sign artifacts"
echo "   4. Run 'make verify' to verify signatures"
