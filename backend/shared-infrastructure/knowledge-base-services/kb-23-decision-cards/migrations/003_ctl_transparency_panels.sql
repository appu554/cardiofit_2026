-- Migration 003: Clinical Transparency Layer (CTL) panel columns
-- Adds four JSONB/varchar columns to decision_cards for structured transparency:
--   Panel 1: patient_state_snapshot   (KB-20 patient data)
--   Panel 2: guideline_condition_status (overall criteria met/partial/not-met)
--   Panel 3: safety_check_summary     (gate decision + safety flags)
--   Panel 4: reasoning_chain          (KB-22 Bayesian reasoning steps)
-- Also adds condition_criteria and condition_status to card_recommendations.

-- decision_cards: CTL panel columns
ALTER TABLE decision_cards
    ADD COLUMN IF NOT EXISTS patient_state_snapshot     JSONB,
    ADD COLUMN IF NOT EXISTS guideline_condition_status VARCHAR(20),
    ADD COLUMN IF NOT EXISTS safety_check_summary       JSONB,
    ADD COLUMN IF NOT EXISTS reasoning_chain             JSONB;

COMMENT ON COLUMN decision_cards.patient_state_snapshot     IS 'CTL Panel 1: Structured KB-20 patient state (stratum, eGFR, HbA1c, medications)';
COMMENT ON COLUMN decision_cards.guideline_condition_status IS 'CTL Panel 2: Overall guideline criteria status (CRITERIA_MET, CRITERIA_PARTIAL, CRITERIA_NOT_MET)';
COMMENT ON COLUMN decision_cards.safety_check_summary       IS 'CTL Panel 3: Safety check summary (gate decision, observation reliability, safety flags)';
COMMENT ON COLUMN decision_cards.reasoning_chain             IS 'CTL Panel 4: Ordered reasoning steps from KB-22 Bayesian engine';

-- card_recommendations: condition criteria for Panel 2 per-recommendation detail
ALTER TABLE card_recommendations
    ADD COLUMN IF NOT EXISTS condition_criteria JSONB,
    ADD COLUMN IF NOT EXISTS condition_status   VARCHAR(20);

COMMENT ON COLUMN card_recommendations.condition_criteria IS 'CTL Panel 2: Array of guideline criteria with individual met/partial/not-met status';
COMMENT ON COLUMN card_recommendations.condition_status   IS 'CTL Panel 2: Overall condition status for this recommendation';
