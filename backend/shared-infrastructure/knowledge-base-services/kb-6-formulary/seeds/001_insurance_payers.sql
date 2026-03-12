-- ============================================================================
-- KB-6 Formulary Service - Insurance Payers Seed Data
-- ============================================================================
-- Production data for major US insurance payers
-- ============================================================================

-- Major US Commercial Payers
INSERT INTO insurance_payers (payer_id, payer_name, payer_type, status, created_at, updated_at) VALUES
('ANTHEM', 'Anthem Blue Cross Blue Shield', 'commercial', 'active', NOW(), NOW()),
('AETNA', 'Aetna', 'commercial', 'active', NOW(), NOW()),
('CIGNA', 'Cigna Healthcare', 'commercial', 'active', NOW(), NOW()),
('UNITED', 'UnitedHealthcare', 'commercial', 'active', NOW(), NOW()),
('HUMANA', 'Humana', 'commercial', 'active', NOW(), NOW()),
('KAISER', 'Kaiser Permanente', 'commercial', 'active', NOW(), NOW()),
('BCBS_CA', 'Blue Cross Blue Shield of California', 'commercial', 'active', NOW(), NOW()),
('BCBS_TX', 'Blue Cross Blue Shield of Texas', 'commercial', 'active', NOW(), NOW()),
('BCBS_FL', 'Florida Blue (BCBS Florida)', 'commercial', 'active', NOW(), NOW()),
('CENTENE', 'Centene Corporation', 'commercial', 'active', NOW(), NOW())
ON CONFLICT (payer_id) DO UPDATE SET
    payer_name = EXCLUDED.payer_name,
    payer_type = EXCLUDED.payer_type,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Medicare Plans
INSERT INTO insurance_payers (payer_id, payer_name, payer_type, status, created_at, updated_at) VALUES
('MEDICARE_A', 'Medicare Part A (Hospital)', 'medicare', 'active', NOW(), NOW()),
('MEDICARE_B', 'Medicare Part B (Medical)', 'medicare', 'active', NOW(), NOW()),
('MEDICARE_D', 'Medicare Part D (Prescription)', 'medicare', 'active', NOW(), NOW()),
('MEDICARE_ADV', 'Medicare Advantage', 'medicare', 'active', NOW(), NOW())
ON CONFLICT (payer_id) DO UPDATE SET
    payer_name = EXCLUDED.payer_name,
    payer_type = EXCLUDED.payer_type,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Medicaid Plans (State-based)
INSERT INTO insurance_payers (payer_id, payer_name, payer_type, status, created_at, updated_at) VALUES
('MEDICAID_CA', 'Medi-Cal (California Medicaid)', 'medicaid', 'active', NOW(), NOW()),
('MEDICAID_NY', 'NY Medicaid', 'medicaid', 'active', NOW(), NOW()),
('MEDICAID_TX', 'Texas Medicaid', 'medicaid', 'active', NOW(), NOW()),
('MEDICAID_FL', 'Florida Medicaid', 'medicaid', 'active', NOW(), NOW()),
('MEDICAID_PA', 'Pennsylvania Medicaid', 'medicaid', 'active', NOW(), NOW())
ON CONFLICT (payer_id) DO UPDATE SET
    payer_name = EXCLUDED.payer_name,
    payer_type = EXCLUDED.payer_type,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Insurance Plans
INSERT INTO insurance_plans (plan_id, payer_id, plan_name, plan_type, plan_year, status, created_at, updated_at) VALUES
-- Anthem Plans
('ANTHEM_PPO_2025', 'ANTHEM', 'Anthem Blue Cross PPO 2025', 'ppo', 2025, 'active', NOW(), NOW()),
('ANTHEM_HMO_2025', 'ANTHEM', 'Anthem Blue Cross HMO 2025', 'hmo', 2025, 'active', NOW(), NOW()),
('ANTHEM_HDHP_2025', 'ANTHEM', 'Anthem Blue Cross HDHP 2025', 'hdhp', 2025, 'active', NOW(), NOW()),
-- Aetna Plans
('AETNA_PPO_2025', 'AETNA', 'Aetna Choice PPO 2025', 'ppo', 2025, 'active', NOW(), NOW()),
('AETNA_HMO_2025', 'AETNA', 'Aetna Select HMO 2025', 'hmo', 2025, 'active', NOW(), NOW()),
-- Cigna Plans
('CIGNA_PPO_2025', 'CIGNA', 'Cigna Choice Plus PPO 2025', 'ppo', 2025, 'active', NOW(), NOW()),
('CIGNA_HMO_2025', 'CIGNA', 'Cigna LocalPlus HMO 2025', 'hmo', 2025, 'active', NOW(), NOW()),
-- United Plans
('UNITED_PPO_2025', 'UNITED', 'UnitedHealthcare Choice PPO 2025', 'ppo', 2025, 'active', NOW(), NOW()),
('UNITED_HMO_2025', 'UNITED', 'UnitedHealthcare Navigate HMO 2025', 'hmo', 2025, 'active', NOW(), NOW()),
-- Medicare Part D
('MEDICARE_D_BASIC_2025', 'MEDICARE_D', 'Medicare Part D Basic 2025', 'part_d', 2025, 'active', NOW(), NOW()),
('MEDICARE_D_ENHANCED_2025', 'MEDICARE_D', 'Medicare Part D Enhanced 2025', 'part_d', 2025, 'active', NOW(), NOW())
ON CONFLICT (plan_id) DO UPDATE SET
    plan_name = EXCLUDED.plan_name,
    plan_type = EXCLUDED.plan_type,
    plan_year = EXCLUDED.plan_year,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Plan Tier Structures
INSERT INTO formulary_tiers (tier_id, tier_name, tier_order, copay_type, description, created_at) VALUES
('tier1_generic', 'Tier 1 - Generic', 1, 'copay', 'Generic medications with lowest cost-sharing', NOW()),
('tier2_preferred_brand', 'Tier 2 - Preferred Brand', 2, 'copay', 'Preferred brand-name medications', NOW()),
('tier3_non_preferred', 'Tier 3 - Non-Preferred', 3, 'copay', 'Non-preferred brand-name medications', NOW()),
('tier4_specialty', 'Tier 4 - Specialty', 4, 'coinsurance', 'Specialty medications (biologics, high-cost)', NOW()),
('tier5_specialty_plus', 'Tier 5 - Specialty Plus', 5, 'coinsurance', 'Ultra-high-cost specialty medications', NOW())
ON CONFLICT (tier_id) DO UPDATE SET
    tier_name = EXCLUDED.tier_name,
    tier_order = EXCLUDED.tier_order,
    copay_type = EXCLUDED.copay_type,
    description = EXCLUDED.description;
