# Clinical Knowledge Core - Build Pipeline
# ==========================================
# This Makefile orchestrates the CQL→ELM compilation, ValueSet expansion,
# manifest generation, and artifact signing.

.PHONY: all build compile-cql expand-valuesets sign verify clean test help

# Configuration
BUILD_DIR := build
CQL_DIR := .
ELM_OUTPUT := $(BUILD_DIR)/cql-to-elm
VS_OUTPUT := $(BUILD_DIR)/valueset-expansion
MANIFEST_OUTPUT := $(BUILD_DIR)/manifests
KEYS_DIR := .keys

# Version (from git tag or default)
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "0.0.1-dev")

# ============================================================
# Main Targets
# ============================================================

all: build sign ## Build and sign all artifacts

build: compile-cql expand-valuesets manifest ## Compile CQL, expand ValueSets, generate manifest

help: ## Show this help
	@echo "Clinical Knowledge Core - Build Pipeline"
	@echo "========================================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================================
# CQL Compilation
# ============================================================

compile-cql: $(ELM_OUTPUT) ## Compile all CQL files to ELM JSON
	@echo "🔨 Compiling CQL to ELM..."
	@./scripts/compile-cql.sh $(CQL_DIR) $(ELM_OUTPUT)
	@echo "✅ CQL compilation complete"

$(ELM_OUTPUT):
	@mkdir -p $(ELM_OUTPUT)

# ============================================================
# ValueSet Expansion
# ============================================================

expand-valuesets: $(VS_OUTPUT) ## Expand all ValueSet definitions
	@echo "📦 Expanding ValueSets..."
	@./scripts/expand-valuesets.sh $(VS_OUTPUT)
	@echo "✅ ValueSet expansion complete"

$(VS_OUTPUT):
	@mkdir -p $(VS_OUTPUT)

# ============================================================
# Manifest Generation
# ============================================================

manifest: $(MANIFEST_OUTPUT) ## Generate build manifest
	@echo "📋 Generating manifest..."
	@./scripts/generate-manifest.sh $(VERSION) $(MANIFEST_OUTPUT)
	@echo "✅ Manifest generated"

$(MANIFEST_OUTPUT):
	@mkdir -p $(MANIFEST_OUTPUT)

# ============================================================
# Signing
# ============================================================

sign: build $(KEYS_DIR)/private.pem ## Sign all artifacts with Ed25519
	@echo "🔐 Signing artifacts..."
	@./scripts/sign-artifacts.sh $(BUILD_DIR) $(KEYS_DIR)/private.pem
	@echo "✅ Artifacts signed"

verify: ## Verify artifact signatures
	@echo "🔍 Verifying signatures..."
	@./scripts/verify-signatures.sh $(BUILD_DIR) $(KEYS_DIR)/public.pem
	@echo "✅ Signatures verified"

generate-keys: $(KEYS_DIR) ## Generate new Ed25519 keypair (ONE TIME ONLY)
	@echo "🔑 Generating Ed25519 keypair..."
	@./scripts/generate-keys.sh $(KEYS_DIR)
	@echo "⚠️  IMPORTANT: Store private.pem securely! Never commit to git!"

$(KEYS_DIR):
	@mkdir -p $(KEYS_DIR)

# ============================================================
# Testing
# ============================================================

test: ## Run all tests
	@echo "🧪 Running tests..."
	@./scripts/run-tests.sh
	@echo "✅ Tests complete"

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@./scripts/run-tests.sh unit

test-integration: ## Run integration tests only
	@echo "🧪 Running integration tests..."
	@./scripts/run-tests.sh integration

# ============================================================
# Publishing
# ============================================================

publish: sign ## Publish artifacts to registry
	@echo "📤 Publishing to artifact registry..."
	@./scripts/publish.sh $(VERSION)
	@echo "✅ Published version $(VERSION)"

# ============================================================
# Cleanup
# ============================================================

clean: ## Remove all build artifacts
	@echo "🧹 Cleaning build directory..."
	@rm -rf $(BUILD_DIR)/cql-to-elm/*.json
	@rm -rf $(BUILD_DIR)/valueset-expansion/*.json
	@rm -rf $(BUILD_DIR)/manifests/*.json
	@echo "✅ Clean complete"

clean-all: clean ## Remove all build artifacts including keys
	@rm -rf $(BUILD_DIR)
	@echo "✅ Full clean complete"

# ============================================================
# Development Helpers
# ============================================================

lint: ## Lint CQL files for style issues
	@echo "🔍 Linting CQL files..."
	@./scripts/lint-cql.sh

format: ## Format CQL files
	@echo "✨ Formatting CQL files..."
	@./scripts/format-cql.sh

deps: ## Install build dependencies
	@echo "📦 Installing dependencies..."
	@./scripts/install-deps.sh

docker-build: ## Build using Docker (no local Java required)
	@echo "🐳 Building with Docker..."
	@docker-compose -f docker/docker-compose.build.yml up --build

# ============================================================
# Info
# ============================================================

info: ## Show build information
	@echo "Clinical Knowledge Core Build Info"
	@echo "==================================="
	@echo "Version: $(VERSION)"
	@echo "Build Dir: $(BUILD_DIR)"
	@echo "CQL Dir: $(CQL_DIR)"
	@echo ""
	@echo "Tiers:"
	@echo "  - tier-0-fhir"
	@echo "  - tier-0.5-terminology"
	@echo "  - tier-1-primitives"
	@echo "  - tier-2-cqm-infra"
	@echo "  - tier-3-domain-commons"
	@echo "  - tier-4-guidelines"
	@echo "  - tier-5-regional-adapters"
