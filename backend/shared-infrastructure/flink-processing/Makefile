# =============================================================================
# Flink M7-M13 Clinical Intelligence Pipeline — Makefile
# =============================================================================
# One-command deployment for the CardioFit Flink pipeline.
#
# Quick start:
#   make deploy          # Build + start infra + create topics + submit all jobs
#   make status          # Check all running jobs
#   make stop            # Stop everything
#   make e2e             # Run 3-patient 14-day E2E test
#
# Prerequisites:
#   - Docker + Docker Compose
#   - Maven 3.9+
#   - Java 17+
#   - Kafka running on cardiofit-lite network:
#     cd ../kafka && docker compose -f docker-compose.hpi-lite.yml up -d
# =============================================================================

SHELL := /bin/bash
.DEFAULT_GOAL := help

# --- Configuration ---
JAR            := target/flink-ehr-intelligence-1.0.0.jar
COMPOSE_E2E    := docker-compose.e2e-flink.yml
COMPOSE_PROD   := docker-compose.yml
JM_CONTAINER   := cardiofit-flink-jobmanager
TM_CONTAINER   := cardiofit-flink-taskmanager
KAFKA_CONTAINER := cardiofit-kafka-lite
KAFKA_BOOTSTRAP := localhost:29092
FLINK_REST     := http://localhost:8181

# All 10 M7-M13 module job arguments (positional args for FlinkJobOrchestrator)
MODULES := module7 module8 module9 module10 module10b module11 module11b intervention-window intervention-delta clinical-state-sync

# M7-M13 Kafka output topics (from KafkaTopics.java isV4OutputTopic)
M7_M13_TOPICS := \
	flink.bp-variability-metrics \
	flink.meal-response \
	flink.meal-patterns \
	flink.engagement-signals \
	clinical.intervention-events \
	clinical.intervention-window-signals \
	clinical.decision-cards \
	alerts.comorbidity-interactions \
	alerts.engagement-drop \
	alerts.relapse-risk \
	flink.activity-response \
	flink.fitness-patterns \
	flink.intervention-deltas \
	clinical.state-change-events

# Input topics consumed by M7-M13
INPUT_TOPICS := \
	ingestion.vitals \
	enriched-patient-events-v1 \
	clinical.intervention-events

# =============================================================================
#  HIGH-LEVEL TARGETS
# =============================================================================

.PHONY: help deploy deploy-prod stop status e2e clean

help: ## Show this help
	@echo ""
	@echo "CardioFit Flink M7-M13 Pipeline"
	@echo "================================"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

deploy: build start-e2e wait-flink create-topics submit-all verify ## Full deploy: build → start → topics → submit → verify
	@echo ""
	@echo "════════════════════════════════════════════════════"
	@echo "  ✓ Deployment complete — all 10 modules running"
	@echo "  Flink UI: $(FLINK_REST)"
	@echo "  Kafka UI: http://localhost:9080"
	@echo "════════════════════════════════════════════════════"

deploy-prod: build start-prod wait-flink-prod create-topics-prod submit-all-prod verify-prod ## Production deploy (2 TaskManagers)
	@echo "  ✓ Production deployment complete"

stop: ## Stop all Flink containers
	@echo "Stopping Flink containers..."
	docker compose -f $(COMPOSE_E2E) down 2>/dev/null || true
	docker compose -f $(COMPOSE_PROD) down 2>/dev/null || true
	@echo "  ✓ Stopped"

clean: stop ## Stop containers + clean build artifacts
	mvn clean -q
	@echo "  ✓ Cleaned"

# =============================================================================
#  BUILD
# =============================================================================

.PHONY: build build-quick

build: ## Build uber JAR (with tests)
	@echo "Building flink-ehr-intelligence JAR..."
	mvn clean package -q
	@echo "  ✓ Built: $(JAR) ($$(du -h $(JAR) | cut -f1))"

build-quick: ## Build uber JAR (skip tests)
	@echo "Building JAR (skip tests)..."
	mvn package -DskipTests -q
	@echo "  ✓ Built: $(JAR)"

# =============================================================================
#  INFRASTRUCTURE
# =============================================================================

.PHONY: start-e2e start-prod wait-flink wait-flink-prod

start-e2e: ## Start Flink (E2E mode: 1 TaskManager + auto-submitter)
	@echo "Starting Flink (E2E mode)..."
	docker compose -f $(COMPOSE_E2E) up -d
	@echo "  ✓ Flink containers started"

start-prod: ## Start Flink (Production: 2 TaskManagers)
	@echo "Starting Flink (Production mode)..."
	docker compose -f $(COMPOSE_PROD) up -d
	@echo "  ✓ Flink production containers started"

wait-flink: ## Wait for Flink JobManager to be ready
	@echo "Waiting for Flink JobManager..."
	@for i in $$(seq 1 60); do \
		if curl -sf $(FLINK_REST)/overview > /dev/null 2>&1; then \
			echo "  ✓ Flink ready ($$i s)"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "  ✗ Flink not ready after 60s"; exit 1

wait-flink-prod:
	@echo "Waiting for Flink JobManager (prod)..."
	@for i in $$(seq 1 60); do \
		if curl -sf http://localhost:8081/overview > /dev/null 2>&1; then \
			echo "  ✓ Flink ready ($$i s)"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "  ✗ Flink not ready after 60s"; exit 1

# =============================================================================
#  KAFKA TOPICS
# =============================================================================

.PHONY: create-topics create-topics-prod reset-topics list-topics

create-topics: ## Create all M7-M13 input + output Kafka topics
	@echo "Creating M7-M13 Kafka topics..."
	@for topic in $(INPUT_TOPICS) $(M7_M13_TOPICS); do \
		docker exec $(KAFKA_CONTAINER) kafka-topics --bootstrap-server localhost:29092 \
			--create --if-not-exists --topic $$topic \
			--partitions 4 --replication-factor 1 2>/dev/null && \
			echo "  ✓ $$topic" || echo "  · $$topic (exists)"; \
	done
	@echo "  ✓ All topics ready"

create-topics-prod:
	@echo "Creating M7-M13 Kafka topics (prod)..."
	@for topic in $(INPUT_TOPICS) $(M7_M13_TOPICS); do \
		docker exec kafka1 kafka-topics --bootstrap-server localhost:9092 \
			--create --if-not-exists --topic $$topic \
			--partitions 8 --replication-factor 3 2>/dev/null && \
			echo "  ✓ $$topic" || echo "  · $$topic (exists)"; \
	done

reset-topics: ## Delete and recreate all M7-M13 output topics (DESTRUCTIVE)
	@echo "⚠️  Resetting M7-M13 output topics (all messages will be lost)..."
	@read -p "Continue? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	@for topic in $(M7_M13_TOPICS); do \
		docker exec $(KAFKA_CONTAINER) kafka-topics --bootstrap-server localhost:29092 \
			--delete --topic $$topic 2>/dev/null; \
	done
	@sleep 2
	@$(MAKE) create-topics --no-print-directory

list-topics: ## List all Kafka topics with message counts
	@echo "Kafka topics:"
	@docker exec $(KAFKA_CONTAINER) kafka-topics --bootstrap-server localhost:29092 --list 2>/dev/null | \
		grep -E 'flink\.|clinical\.|alerts\.|enriched|ingestion' | sort

# =============================================================================
#  JOB SUBMISSION
# =============================================================================

.PHONY: submit-all submit-all-prod cancel-all

submit-all: ## Submit all 10 M7-M13 module jobs
	@echo "Submitting 10 module jobs..."
	@for mod in $(MODULES); do \
		echo "  Submitting $$mod..."; \
		docker exec $(JM_CONTAINER) /opt/flink/bin/flink run -d \
			-c com.cardiofit.flink.FlinkJobOrchestrator \
			/opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
			$$mod development 2>&1 | grep -o 'JobID [a-f0-9]*' || echo "  ✗ $$mod failed"; \
		sleep 3; \
	done
	@echo "  ✓ All jobs submitted"

submit-all-prod:
	@echo "Submitting 10 module jobs (production)..."
	@for mod in $(MODULES); do \
		echo "  Submitting $$mod..."; \
		docker exec flink-jobmanager-2.1 /opt/flink/bin/flink run -d \
			-c com.cardiofit.flink.FlinkJobOrchestrator \
			/opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
			$$mod production 2>&1 | grep -o 'JobID [a-f0-9]*' || echo "  ✗ $$mod failed"; \
		sleep 3; \
	done

cancel-all: ## Cancel all running Flink jobs
	@echo "Cancelling all running jobs..."
	@docker exec $(JM_CONTAINER) /opt/flink/bin/flink list 2>/dev/null | \
		grep RUNNING | awk '{print $$4}' | while read jid; do \
			docker exec $(JM_CONTAINER) /opt/flink/bin/flink cancel $$jid 2>/dev/null; \
			echo "  ✗ Cancelled $$jid"; \
		done
	@echo "  ✓ All jobs cancelled"

# =============================================================================
#  MONITORING & STATUS
# =============================================================================

.PHONY: verify verify-prod status jobs logs health topic-counts

verify: ## Verify all 10 jobs are RUNNING
	@echo "Verifying deployment..."
	@running=$$(docker exec $(JM_CONTAINER) /opt/flink/bin/flink list 2>/dev/null | grep -c RUNNING); \
	echo "  Running jobs: $$running / 10"; \
	if [ "$$running" -ge 10 ]; then \
		echo "  ✓ All modules running"; \
	else \
		echo "  ⚠ Expected 10, got $$running"; \
		docker exec $(JM_CONTAINER) /opt/flink/bin/flink list 2>/dev/null | grep RUNNING; \
	fi

verify-prod:
	@running=$$(docker exec flink-jobmanager-2.1 /opt/flink/bin/flink list 2>/dev/null | grep -c RUNNING); \
	echo "  Running jobs: $$running / 10"; \
	if [ "$$running" -ge 10 ]; then echo "  ✓ All modules running"; \
	else echo "  ⚠ Expected 10, got $$running"; fi

status: ## Show Flink job status + Kafka topic counts
	@echo ""
	@echo "═══ Flink Jobs ═══"
	@docker exec $(JM_CONTAINER) /opt/flink/bin/flink list 2>/dev/null | grep -E 'RUNNING|FAILED|CANCELED' || echo "  No jobs found"
	@echo ""
	@echo "═══ Docker Containers ═══"
	@docker ps --format "  {{.Names}}\t{{.Status}}" 2>/dev/null | grep -i flink || echo "  No Flink containers"
	@echo ""

jobs: ## List running jobs (compact)
	@docker exec $(JM_CONTAINER) /opt/flink/bin/flink list 2>/dev/null | grep RUNNING

logs: ## Tail Flink TaskManager logs
	docker logs -f --tail 50 $(TM_CONTAINER)

health: ## Check Flink cluster health
	@echo "JobManager:"
	@curl -sf $(FLINK_REST)/overview 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  slots: {d[\"slots-available\"]}/{d[\"slots-total\"]}  jobs: {d[\"jobs-running\"]} running, {d[\"jobs-finished\"]} finished')" 2>/dev/null || echo "  ✗ Not reachable"
	@echo "TaskManagers:"
	@curl -sf $(FLINK_REST)/taskmanagers 2>/dev/null | python3 -c "import sys,json; [print(f'  {tm[\"id\"][:8]}... slots={tm[\"slotsNumber\"]} free={tm[\"freeSlots\"]}') for tm in json.load(sys.stdin).get('taskmanagers',[])]" 2>/dev/null || echo "  ✗ Not reachable"

topic-counts: ## Show message counts for M7-M13 output topics
	@echo "M7-M13 output topic message counts:"
	@for topic in $(M7_M13_TOPICS); do \
		count=$$(docker exec $(KAFKA_CONTAINER) kafka-run-class kafka.tools.GetOffsetShell \
			--broker-list localhost:29092 --topic $$topic --time -1 2>/dev/null | \
			awk -F: '{sum+=$$3} END{print sum}'); \
		printf "  %-45s %s\n" "$$topic" "$${count:-0}"; \
	done

# =============================================================================
#  E2E TESTING
# =============================================================================

.PHONY: e2e e2e-generate e2e-run

e2e: e2e-generate e2e-run ## Run full 3-patient 14-day E2E test

e2e-generate: ## Regenerate E2E test dataset
	@echo "Generating E2E dataset..."
	python3 e2e_14day_generator.py --start-date $$(date +%Y-%m-%d)
	@echo "  ✓ Dataset generated"

e2e-run: ## Run E2E test (requires running Flink + all jobs)
	@echo "Running 3-patient 14-day E2E test..."
	python3 scripts/flink_e2e_3patient_14day.py --process-wait 50 --consume-timeout 20

# =============================================================================
#  DEVELOPMENT
# =============================================================================

.PHONY: test test-m13 redeploy

test: ## Run all unit tests
	mvn test

test-m13: ## Run Module 13 tests only
	mvn test -Dtest="Module13*"

redeploy: build-quick cancel-all submit-all verify ## Rebuild JAR + redeploy all jobs (no infra restart)
	@echo "  ✓ Redeployed"
