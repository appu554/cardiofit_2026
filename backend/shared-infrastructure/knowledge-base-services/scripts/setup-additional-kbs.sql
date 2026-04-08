-- setup-additional-kbs.sql
-- Creates databases for KB services added by docker-compose.gateway-e2e.yml
-- Runs as 02-additional-kbs.sql after 01-setup.sql (which creates kb_drug_rules + kb10_rules)

-- Generic KB user for all additional services
CREATE USER kb_user WITH PASSWORD 'kb_password';

-- ─── Core KB databases ───────────────────────────
CREATE DATABASE kb_clinical_context;
GRANT ALL PRIVILEGES ON DATABASE kb_clinical_context TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_clinical_context TO postgres;

CREATE DATABASE kb_guidelines;
GRANT ALL PRIVILEGES ON DATABASE kb_guidelines TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_guidelines TO postgres;

CREATE DATABASE kb_patient_safety;
GRANT ALL PRIVILEGES ON DATABASE kb_patient_safety TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_patient_safety TO postgres;

CREATE DATABASE kb_drug_interactions;
GRANT ALL PRIVILEGES ON DATABASE kb_drug_interactions TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_drug_interactions TO postgres;

CREATE DATABASE kb_formulary;
GRANT ALL PRIVILEGES ON DATABASE kb_formulary TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_formulary TO postgres;

CREATE DATABASE kb_calculator;
GRANT ALL PRIVILEGES ON DATABASE kb_calculator TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_calculator TO postgres;

-- ─── Extended KB databases ───────────────────────
CREATE DATABASE kb_population_health;
GRANT ALL PRIVILEGES ON DATABASE kb_population_health TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_population_health TO postgres;

CREATE DATABASE kb_ordersets;
GRANT ALL PRIVILEGES ON DATABASE kb_ordersets TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_ordersets TO postgres;

CREATE DATABASE kb_quality_measures;
GRANT ALL PRIVILEGES ON DATABASE kb_quality_measures TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_quality_measures TO postgres;

CREATE DATABASE kb_care_navigator;
GRANT ALL PRIVILEGES ON DATABASE kb_care_navigator TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_care_navigator TO postgres;

CREATE DATABASE kb_lab_interpretation;
GRANT ALL PRIVILEGES ON DATABASE kb_lab_interpretation TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_lab_interpretation TO postgres;

CREATE DATABASE kb_population_registry;
GRANT ALL PRIVILEGES ON DATABASE kb_population_registry TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_population_registry TO postgres;

CREATE DATABASE kb_governance_engine;
GRANT ALL PRIVILEGES ON DATABASE kb_governance_engine TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_governance_engine TO postgres;

-- ─── Vaidshala Runtime KB databases ──────────────
CREATE DATABASE kb_protocol_orchestrator;
GRANT ALL PRIVILEGES ON DATABASE kb_protocol_orchestrator TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_protocol_orchestrator TO postgres;

CREATE DATABASE kb_patient_profile;
GRANT ALL PRIVILEGES ON DATABASE kb_patient_profile TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_patient_profile TO postgres;

CREATE DATABASE kb_behavioral_intel;
GRANT ALL PRIVILEGES ON DATABASE kb_behavioral_intel TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_behavioral_intel TO postgres;

CREATE DATABASE kb_hpi_engine;
GRANT ALL PRIVILEGES ON DATABASE kb_hpi_engine TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_hpi_engine TO postgres;

CREATE DATABASE kb_decision_cards;
GRANT ALL PRIVILEGES ON DATABASE kb_decision_cards TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_decision_cards TO postgres;

CREATE DATABASE kb_safety_constraints;
GRANT ALL PRIVILEGES ON DATABASE kb_safety_constraints TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_safety_constraints TO postgres;

CREATE DATABASE kb_lifestyle_graph;
GRANT ALL PRIVILEGES ON DATABASE kb_lifestyle_graph TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_lifestyle_graph TO postgres;

CREATE DATABASE kb_metabolic_twin;
GRANT ALL PRIVILEGES ON DATABASE kb_metabolic_twin TO kb_user;
GRANT ALL PRIVILEGES ON DATABASE kb_metabolic_twin TO postgres;

-- KB-7 uses clinical_governance — already created by kb-only setup if available
-- If not, create it here as a safety net
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'clinical_governance') THEN
        PERFORM dblink_exec('dbname=postgres', 'CREATE DATABASE clinical_governance');
    END IF;
EXCEPTION WHEN OTHERS THEN
    -- Ignore if dblink not available; clinical_governance is created by kb-7 init
    NULL;
END $$;

SELECT 'Additional KB databases created successfully!' AS status;
