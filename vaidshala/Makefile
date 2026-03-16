.PHONY: test-simulation test-production test-bridge test-vmcu test-kb20 test-all

test-simulation:
	cd simulation && go test ./pkg/scenarios/ -run TestScenario -v -count=1

test-production:
	cd simulation && go test ./pkg/scenarios/ -run TestProductionScenarios -v -count=1

test-bridge:
	cd simulation && go test ./bridge/ -v -count=1

test-vmcu:
	cd clinical-runtime-platform && go test ./engines/vmcu/... -v -count=1

test-kb20:
	cd ../backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go test ./... -v -count=1

test-all: test-simulation test-production test-bridge test-vmcu test-kb20
	@echo "All V-MCU tests passed"
