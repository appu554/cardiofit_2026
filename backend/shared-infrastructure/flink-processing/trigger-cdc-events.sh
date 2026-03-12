#!/bin/bash

###############################################################################
# Trigger CDC Events Script
# Generates CDC events by making database changes to all KB services
###############################################################################

set -e

echo "=================================================="
echo "Triggering CDC Events Across All KB Services"
echo "=================================================="
echo ""

###############################################################################
# KB3: Clinical Protocols
###############################################################################

echo "[1/7] Triggering KB3 Clinical Protocol CDC event..."

# Check if kb3 database exists
if psql -h localhost -U cardiofit_user -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw kb3; then
    psql -h localhost -U cardiofit_user -d kb3 << EOF
-- Insert a test protocol (will trigger CDC INSERT event)
INSERT INTO clinical_protocols (
    protocol_id, name, category, specialty, version, last_updated, source,
    activation_criteria, priority_determination
) VALUES (
    'TEST-CDC-001',
    'Test Protocol for CDC',
    'INFECTIOUS',
    'CRITICAL_CARE',
    '1.0',
    CURRENT_DATE,
    'CDC Test Script',
    'Test criteria',
    'HIGH'
) ON CONFLICT (protocol_id) DO UPDATE SET
    version = EXCLUDED.version || '.1',
    updated_at = CURRENT_TIMESTAMP;

SELECT 'KB3 CDC event triggered: ' || protocol_id FROM clinical_protocols WHERE protocol_id = 'TEST-CDC-001';
EOF
    echo "✅ KB3 Protocol CDC event triggered"
else
    echo "⚠️  KB3 database not found - skipping"
fi

echo ""

###############################################################################
# KB2: Clinical Phenotypes
###############################################################################

echo "[2/7] Triggering KB2 Clinical Phenotype CDC event..."

if psql -h localhost -U cardiofit_user -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw kb2; then
    psql -h localhost -U cardiofit_user -d kb2 << EOF
-- Insert test phenotype
INSERT INTO clinical_phenotypes (
    phenotype_id, name, description, priority, version
) VALUES (
    'TEST-PHENO-001',
    'Test Phenotype for CDC',
    'High-risk cardiac patient archetype',
    'HIGH',
    '1.0'
) ON CONFLICT (phenotype_id) DO UPDATE SET
    version = EXCLUDED.version || '.1',
    updated_at = CURRENT_TIMESTAMP;

SELECT 'KB2 CDC event triggered: ' || phenotype_id FROM clinical_phenotypes WHERE phenotype_id = 'TEST-PHENO-001';
EOF
    echo "✅ KB2 Phenotype CDC event triggered"
else
    echo "⚠️  KB2 database not found - skipping"
fi

echo ""

###############################################################################
# KB1: Drug Rules
###############################################################################

echo "[3/7] Triggering KB1 Drug Rule CDC event..."

if psql -h localhost -U cardiofit_user -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw kb1; then
    psql -h localhost -U cardiofit_user -d kb1 << EOF
-- Insert test drug rule
INSERT INTO drug_rule_packs (
    drug_id, version, content_sha, signed_by, signature_valid, regions, content
) VALUES (
    'TEST-DRUG-001',
    '1.0',
    'test-sha256',
    'CDC Test',
    true,
    '["US", "EU"]'::jsonb,
    '{"dosing": "test"}'::jsonb
) ON CONFLICT (drug_id, version) DO UPDATE SET
    updated_at = CURRENT_TIMESTAMP;

SELECT 'KB1 CDC event triggered: ' || drug_id FROM drug_rule_packs WHERE drug_id = 'TEST-DRUG-001';
EOF
    echo "✅ KB1 Drug Rule CDC event triggered"
else
    echo "⚠️  KB1 database not found - skipping"
fi

echo ""

###############################################################################
# KB5: Drug Interactions
###############################################################################

echo "[4/7] Triggering KB5 Drug Interaction CDC event..."

if psql -h localhost -U cardiofit_user -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw kb5; then
    psql -h localhost -U cardiofit_user -d kb5 << EOF
-- Insert test drug interaction
INSERT INTO drug_interactions (
    interaction_id, drug_a, drug_b, severity, mechanism, clinical_effect, evidence_level
) VALUES (
    'TEST-INT-001',
    'WARFARIN',
    'ASPIRIN',
    'HIGH',
    'Synergistic anticoagulant effect',
    'Increased bleeding risk',
    'STRONG'
) ON CONFLICT (interaction_id) DO UPDATE SET
    severity = EXCLUDED.severity,
    updated_at = CURRENT_TIMESTAMP;

SELECT 'KB5 CDC event triggered: ' || interaction_id FROM drug_interactions WHERE interaction_id = 'TEST-INT-001';
EOF
    echo "✅ KB5 Interaction CDC event triggered"
else
    echo "⚠️  KB5 database not found - skipping"
fi

echo ""

###############################################################################
# KB6: Formulary
###############################################################################

echo "[5/7] Triggering KB6 Formulary CDC event..."

if psql -h localhost -U cardiofit_user -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw kb6; then
    psql -h localhost -U cardiofit_user -d kb6 << EOF
-- Insert test formulary drug
INSERT INTO formulary_drugs (
    drug_id, drug_name, generic_name, formulary_status, tier, requires_prior_auth, therapeutic_class
) VALUES (
    'TEST-FORM-001',
    'Test Medication',
    'test-generic',
    'PREFERRED',
    1,
    false,
    'CARDIOVASCULAR'
) ON CONFLICT (drug_id) DO UPDATE SET
    formulary_status = EXCLUDED.formulary_status,
    updated_at = CURRENT_TIMESTAMP;

SELECT 'KB6 CDC event triggered: ' || drug_id FROM formulary_drugs WHERE drug_id = 'TEST-FORM-001';
EOF
    echo "✅ KB6 Formulary CDC event triggered"
else
    echo "⚠️  KB6 database not found - skipping"
fi

echo ""

###############################################################################
# KB7: Terminology
###############################################################################

echo "[6/7] Triggering KB7 Terminology CDC event..."

if psql -h localhost -U cardiofit_user -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw kb7; then
    psql -h localhost -U cardiofit_user -d kb7 << EOF
-- Insert test terminology concept
INSERT INTO terminology_concepts (
    concept_id, concept_code, display_name, code_system, code_system_version, definition, status
) VALUES (
    'TEST-TERM-001',
    '12345-6',
    'Test Laboratory Result',
    'LOINC',
    '2.74',
    'Test concept for CDC verification',
    'ACTIVE'
) ON CONFLICT (concept_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    updated_at = CURRENT_TIMESTAMP;

SELECT 'KB7 CDC event triggered: ' || concept_id FROM terminology_concepts WHERE concept_id = 'TEST-TERM-001';
EOF
    echo "✅ KB7 Terminology CDC event triggered"
else
    echo "⚠️  KB7 database not found - skipping"
fi

echo ""

###############################################################################
# Verification
###############################################################################

echo "[7/7] Verifying CDC events in Kafka topics..."
echo ""

sleep 2  # Wait for Debezium to process

for topic in kb3.clinical_protocols.changes kb2.clinical_phenotypes.changes kb1.drug_rule_packs.changes kb5.drug_interactions.changes kb6.formulary_drugs.changes kb7.terminology.changes
do
    echo -n "  $topic: "
    COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic $topic --time -1 2>/dev/null | awk -F ":" '{sum += $3} END {print sum}')
    if [ "$COUNT" -gt 0 ]; then
        echo "✅ $COUNT events"
    else
        echo "⚠️  0 events (topic may not exist or Debezium not configured)"
    fi
done

echo ""
echo "=================================================="
echo "CDC Event Trigger Complete!"
echo "=================================================="
echo ""
echo "Next steps:"
echo "1. Run CDC Consumer Test: mvn clean package && flink run --class com.cardiofit.flink.test.CDCConsumerTest target/flink-ehr-intelligence-1.0.0.jar"
echo "2. Check Flink logs for CDC event processing"
echo "3. Verify deserialization works correctly"
