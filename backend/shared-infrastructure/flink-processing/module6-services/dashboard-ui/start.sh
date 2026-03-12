#!/bin/bash

# CardioFit Dashboard UI - Startup Script
# This script handles development and production startup

set -e

echo "=========================================="
echo "  CardioFit Dashboard UI"
echo "  Clinical Intelligence Platform"
echo "=========================================="
echo ""

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
    echo "✅ Dependencies installed"
    echo ""
fi

# Check if .env exists
if [ ! -f ".env" ]; then
    echo "⚠️  No .env file found. Copying from .env.example..."
    cp .env.example .env
    echo "✅ Created .env file"
    echo "   Please review and update the configuration"
    echo ""
fi

# Determine startup mode
MODE="${1:-dev}"

case "$MODE" in
    "dev")
        echo "🚀 Starting development server..."
        echo "   Dashboard will be available at: http://localhost:3000"
        echo ""
        npm run dev
        ;;
    "build")
        echo "🔨 Building production bundle..."
        npm run build
        echo "✅ Build complete! Output in ./dist"
        echo ""
        ;;
    "preview")
        echo "👀 Starting production preview..."
        echo "   Dashboard will be available at: http://localhost:4173"
        echo ""
        npm run preview
        ;;
    "docker")
        echo "🐳 Building and starting Docker container..."
        docker-compose up --build -d
        echo "✅ Container started"
        echo "   Dashboard: http://localhost:3000"
        echo "   Health: http://localhost:3000/health"
        echo ""
        echo "📊 View logs with: docker-compose logs -f"
        echo "🛑 Stop with: docker-compose down"
        ;;
    "docker-logs")
        echo "📊 Showing container logs..."
        docker-compose logs -f
        ;;
    "docker-stop")
        echo "🛑 Stopping Docker container..."
        docker-compose down
        echo "✅ Container stopped"
        ;;
    "health")
        echo "🏥 Checking health status..."
        curl -f http://localhost:3000/health || echo "❌ Health check failed"
        echo ""
        ;;
    "type-check")
        echo "🔍 Running TypeScript type checking..."
        npm run type-check
        ;;
    "lint")
        echo "🔍 Running ESLint..."
        npm run lint
        ;;
    *)
        echo "Usage: ./start.sh [mode]"
        echo ""
        echo "Available modes:"
        echo "  dev            - Start development server (default)"
        echo "  build          - Build production bundle"
        echo "  preview        - Preview production build"
        echo "  docker         - Build and start Docker container"
        echo "  docker-logs    - View Docker container logs"
        echo "  docker-stop    - Stop Docker container"
        echo "  health         - Check application health"
        echo "  type-check     - Run TypeScript type checking"
        echo "  lint           - Run ESLint"
        echo ""
        exit 1
        ;;
esac
