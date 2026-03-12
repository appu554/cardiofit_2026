# ============================================================================
# NCTS RF2 Import Automation - Makefile
# ============================================================================
# Usage:
#   make import FILE=/path/to/ncts.zip       # Standard import with backup
#   make import-dry-run FILE=/path/to/ncts.zip  # Validate only
#   make import-force FILE=/path/to/ncts.zip    # Force reimport
#   make list-versions                       # View import history
#   make verify                              # Verify current import
#   make rollback                            # Delete all refset data
#   make setup-cron                          # Setup automated updates
# ============================================================================

.PHONY: help import import-dry-run import-force list-versions verify rollback \
        setup-cron create-indexes delete-all stats clean

# Load environment
-include .env

# Default Neo4j settings
NEO4J_URI ?= bolt://localhost:7687
NEO4J_USER ?= neo4j
NEO4J_DATABASE ?= neo4j
BATCH_SIZE ?= 10000

# Script paths
IMPORT_SCRIPT := ./ncts_rf2_import.sh
ROLLBACK_SCRIPT := ./ncts_rollback.sh
CRON_SCRIPT := ./setup_cron.sh

# Colors
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m

# ============================================================================
# Help
# ============================================================================

help:
	@echo ""
	@echo "$(GREEN)NCTS RF2 Import Automation$(NC)"
	@echo "========================================"
	@echo ""
	@echo "$(YELLOW)Import Commands:$(NC)"
	@echo "  make import FILE=<zip>           Standard import with backup"
	@echo "  make import-dry-run FILE=<zip>   Validate without importing"
	@echo "  make import-force FILE=<zip>     Force reimport (same version)"
	@echo "  make import-skip-backup FILE=<zip>  Import without backup"
	@echo ""
	@echo "$(YELLOW)Management Commands:$(NC)"
	@echo "  make list-versions               View import history"
	@echo "  make verify                      Verify current import"
	@echo "  make stats                       Show import statistics"
	@echo "  make create-indexes              Create Neo4j indexes"
	@echo ""
	@echo "$(YELLOW)Rollback Commands:$(NC)"
	@echo "  make rollback                    Delete all refset data"
	@echo "  make delete-all                  Force delete all (no confirm)"
	@echo ""
	@echo "$(YELLOW)Automation Commands:$(NC)"
	@echo "  make setup-cron                  Setup automated monthly updates"
	@echo "  make clean                       Clean temp files"
	@echo ""
	@echo "$(YELLOW)Environment Variables:$(NC)"
	@echo "  NEO4J_URI       = $(NEO4J_URI)"
	@echo "  NEO4J_USER      = $(NEO4J_USER)"
	@echo "  NEO4J_DATABASE  = $(NEO4J_DATABASE)"
	@echo "  BATCH_SIZE      = $(BATCH_SIZE)"
	@echo ""
	@echo "$(YELLOW)Example:$(NC)"
	@echo "  make import FILE=/path/to/SnomedCT_AU_20240930.zip"
	@echo ""

# ============================================================================
# Import Commands
# ============================================================================

import:
ifndef FILE
	$(error FILE is required. Usage: make import FILE=/path/to/ncts.zip)
endif
	@echo "Starting NCTS RF2 import..."
	NEO4J_URI=$(NEO4J_URI) NEO4J_USER=$(NEO4J_USER) \
	NEO4J_PASSWORD=$(NEO4J_PASSWORD) NEO4J_DATABASE=$(NEO4J_DATABASE) \
	BATCH_SIZE=$(BATCH_SIZE) \
	$(IMPORT_SCRIPT) $(FILE)

import-dry-run:
ifndef FILE
	$(error FILE is required. Usage: make import-dry-run FILE=/path/to/ncts.zip)
endif
	@echo "Running dry-run validation..."
	NEO4J_URI=$(NEO4J_URI) NEO4J_USER=$(NEO4J_USER) \
	NEO4J_PASSWORD=$(NEO4J_PASSWORD) NEO4J_DATABASE=$(NEO4J_DATABASE) \
	$(IMPORT_SCRIPT) --dry-run $(FILE)

import-force:
ifndef FILE
	$(error FILE is required. Usage: make import-force FILE=/path/to/ncts.zip)
endif
	@echo "Force reimporting NCTS RF2 data..."
	NEO4J_URI=$(NEO4J_URI) NEO4J_USER=$(NEO4J_USER) \
	NEO4J_PASSWORD=$(NEO4J_PASSWORD) NEO4J_DATABASE=$(NEO4J_DATABASE) \
	BATCH_SIZE=$(BATCH_SIZE) \
	$(IMPORT_SCRIPT) --force $(FILE)

import-skip-backup:
ifndef FILE
	$(error FILE is required. Usage: make import-skip-backup FILE=/path/to/ncts.zip)
endif
	@echo "Importing without backup..."
	NEO4J_URI=$(NEO4J_URI) NEO4J_USER=$(NEO4J_USER) \
	NEO4J_PASSWORD=$(NEO4J_PASSWORD) NEO4J_DATABASE=$(NEO4J_DATABASE) \
	BATCH_SIZE=$(BATCH_SIZE) \
	$(IMPORT_SCRIPT) --skip-backup $(FILE)

# ============================================================================
# Management Commands
# ============================================================================

list-versions:
	@echo "Import History:"
	@echo "========================================"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) --format plain \
		"MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) \
		 RETURN m.version AS version, m.importedAt AS imported_at, \
		        m.fileCount AS files, m.relationshipCount AS relationships \
		 ORDER BY m.importedAt DESC"

verify:
	@echo "Verifying NCTS Import..."
	@echo "========================================"
	@echo "Current Version:"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) --format plain \
		"MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) \
		 RETURN m.version ORDER BY m.importedAt DESC LIMIT 1"
	@echo ""
	@echo "Relationship Counts:"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) --format plain \
		"MATCH ()-[r:IN_REFSET]->() RETURN 'IN_REFSET' AS type, count(r) AS count \
		 UNION ALL \
		 MATCH (r:Refset) RETURN 'Refset nodes' AS type, count(r) AS count"
	@echo ""
	@echo "Sample Refsets:"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) --format plain \
		"MATCH (r:Refset) RETURN r.id AS id, r.name AS name LIMIT 5"

stats:
	@echo "NCTS Import Statistics"
	@echo "========================================"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) --format plain \
		"MATCH (r:Refset) \
		 WITH count(r) AS refsetCount \
		 MATCH ()-[rel:IN_REFSET]->() \
		 WITH refsetCount, count(rel) AS membershipCount \
		 MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) \
		 WITH refsetCount, membershipCount, m \
		 ORDER BY m.importedAt DESC LIMIT 1 \
		 RETURN refsetCount AS refset_nodes, \
		        membershipCount AS in_refset_relationships, \
		        m.version AS current_version, \
		        m.importedAt AS last_import"

create-indexes:
	@echo "Creating Neo4j indexes..."
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"CREATE INDEX refset_id_idx IF NOT EXISTS FOR (r:Refset) ON (r.id)"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"CREATE INDEX import_metadata_idx IF NOT EXISTS FOR (m:ImportMetadata) ON (m.type, m.version)"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"CREATE CONSTRAINT refset_unique IF NOT EXISTS FOR (r:Refset) REQUIRE r.id IS UNIQUE"
	@echo "Indexes created."

# ============================================================================
# Rollback Commands
# ============================================================================

rollback:
	@echo "$(YELLOW)WARNING: This will delete ALL refset data!$(NC)"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	@echo "Deleting refset data..."
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"CALL apoc.periodic.iterate( \
			'MATCH ()-[r:IN_REFSET]->() RETURN r', \
			'DELETE r', \
			{batchSize: $(BATCH_SIZE), parallel: false}) \
		 YIELD batches, total \
		 RETURN batches, total"
	@echo "Deleting Refset nodes..."
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"MATCH (r:Refset) DETACH DELETE r"
	@echo "Deleting metadata..."
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) DELETE m"
	@echo "$(GREEN)Rollback complete.$(NC)"

delete-all:
	@echo "Force deleting all refset data..."
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"CALL apoc.periodic.iterate( \
			'MATCH ()-[r:IN_REFSET]->() RETURN r', \
			'DELETE r', \
			{batchSize: $(BATCH_SIZE), parallel: false})"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"MATCH (r:Refset) DETACH DELETE r"
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) \
		"MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) DELETE m"
	@echo "All refset data deleted."

# ============================================================================
# Automation Commands
# ============================================================================

setup-cron:
	@echo "Setting up cron job for monthly updates..."
	@chmod +x $(CRON_SCRIPT) 2>/dev/null || true
	@if [ -f $(CRON_SCRIPT) ]; then \
		$(CRON_SCRIPT); \
	else \
		echo "Cron script not found. Creating default..."; \
		echo "0 2 15 * * $(PWD)/ncts_cron_update.sh >> /var/log/ncts-import.log 2>&1" | crontab -; \
	fi
	@echo "Cron job configured for 15th of each month at 2 AM"

clean:
	@echo "Cleaning temporary files..."
	@rm -rf .temp
	@rm -f *.log
	@echo "Clean complete."

# ============================================================================
# Development Commands
# ============================================================================

test-connection:
	@echo "Testing Neo4j connection..."
	@cypher-shell -a $(NEO4J_URI) -u $(NEO4J_USER) -p $(NEO4J_PASSWORD) \
		-d $(NEO4J_DATABASE) "RETURN 'Connection successful!' AS status"
