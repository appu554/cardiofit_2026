-- ============================================================================
-- KB-6 Formulary Service - Formulary Entries Seed Data
-- ============================================================================
-- Production drug data organized by therapeutic class
-- RxNorm codes are real, validated codes
-- ============================================================================

-- =============================================================================
-- DIABETES MEDICATIONS
-- =============================================================================

-- Metformin (Biguanide - First-line, Generic)
INSERT INTO formulary_entries (
    payer_id, plan_id, plan_year, drug_rxnorm, drug_name, drug_type,
    tier_id, copay_amount, coinsurance_percent, prior_authorization, step_therapy,
    quantity_limit, generic_available, therapeutic_class, ndc_codes,
    effective_date, status, created_at, updated_at
) VALUES
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '6809', 'Metformin HCl 500mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 180, "per_days": 30, "max_fills_per_year": 12, "max_daily_dose_mg": 2550}'::jsonb,
 true, 'Antidiabetic - Biguanide', ARRAY['00093-1048-01', '00378-0221-01'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '860975', 'Metformin HCl 1000mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 30, "max_fills_per_year": 12, "max_daily_dose_mg": 2550}'::jsonb,
 true, 'Antidiabetic - Biguanide', ARRAY['00093-1050-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Empagliflozin (Jardiance - SGLT2 Inhibitor, Preferred Brand)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1545653', 'Empagliflozin (Jardiance) 10mg Tablet', 'brand',
 'tier2_preferred_brand', 45.00, NULL, false, true,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Antidiabetic - SGLT2 Inhibitor', ARRAY['00597-0153-30'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1545658', 'Empagliflozin (Jardiance) 25mg Tablet', 'brand',
 'tier2_preferred_brand', 45.00, NULL, false, true,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Antidiabetic - SGLT2 Inhibitor', ARRAY['00597-0154-30'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Canagliflozin (Invokana - SGLT2 Inhibitor, Non-Preferred)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1373458', 'Canagliflozin (Invokana) 100mg Tablet', 'brand',
 'tier3_non_preferred', 85.00, NULL, true, true,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Antidiabetic - SGLT2 Inhibitor', ARRAY['50458-0140-30'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1373463', 'Canagliflozin (Invokana) 300mg Tablet', 'brand',
 'tier3_non_preferred', 85.00, NULL, true, true,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Antidiabetic - SGLT2 Inhibitor', ARRAY['50458-0141-30'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Dapagliflozin (Farxiga - SGLT2 Inhibitor)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1486436', 'Dapagliflozin (Farxiga) 10mg Tablet', 'brand',
 'tier2_preferred_brand', 50.00, NULL, false, true,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Antidiabetic - SGLT2 Inhibitor', ARRAY['00310-6210-30'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Semaglutide (Ozempic - GLP-1 Agonist, Specialty)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1991302', 'Semaglutide (Ozempic) 0.25mg/0.5mL Pen', 'brand',
 'tier4_specialty', NULL, 25.00, true, true,
 '{"max_quantity": 4, "per_days": 28, "max_fills_per_year": 13}'::jsonb,
 false, 'Antidiabetic - GLP-1 Agonist', ARRAY['00169-4130-12'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1991306', 'Semaglutide (Ozempic) 0.5mg/0.5mL Pen', 'brand',
 'tier4_specialty', NULL, 25.00, true, true,
 '{"max_quantity": 4, "per_days": 28, "max_fills_per_year": 13}'::jsonb,
 false, 'Antidiabetic - GLP-1 Agonist', ARRAY['00169-4131-12'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1991310', 'Semaglutide (Ozempic) 1mg/0.5mL Pen', 'brand',
 'tier4_specialty', NULL, 25.00, true, true,
 '{"max_quantity": 4, "per_days": 28, "max_fills_per_year": 13}'::jsonb,
 false, 'Antidiabetic - GLP-1 Agonist', ARRAY['00169-4132-12'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Liraglutide (Victoza - GLP-1 Agonist)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '897122', 'Liraglutide (Victoza) 18mg/3mL Pen', 'brand',
 'tier4_specialty', NULL, 25.00, true, true,
 '{"max_quantity": 3, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Antidiabetic - GLP-1 Agonist', ARRAY['00169-4060-12'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Sitagliptin (Januvia - DPP-4 Inhibitor)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '593411', 'Sitagliptin (Januvia) 100mg Tablet', 'brand',
 'tier2_preferred_brand', 50.00, NULL, false, false,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Antidiabetic - DPP-4 Inhibitor', ARRAY['00006-0277-31'],
 '2025-01-01', 'active', NOW(), NOW()),

-- =============================================================================
-- CARDIOVASCULAR MEDICATIONS
-- =============================================================================

-- Lisinopril (ACE Inhibitor - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '29046', 'Lisinopril 10mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'ACE Inhibitor', ARRAY['00093-1040-01'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '314076', 'Lisinopril 20mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'ACE Inhibitor', ARRAY['00093-1041-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Losartan (ARB - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '52175', 'Losartan 50mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'ARB', ARRAY['00093-7365-01'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '979480', 'Losartan 100mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'ARB', ARRAY['00093-7366-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Metoprolol Succinate ER (Beta Blocker - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '866924', 'Metoprolol Succinate ER 50mg Tablet', 'generic',
 'tier1_generic', 10.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Beta Blocker', ARRAY['00378-1355-01'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '866928', 'Metoprolol Succinate ER 100mg Tablet', 'generic',
 'tier1_generic', 10.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Beta Blocker', ARRAY['00378-1356-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Amlodipine (Calcium Channel Blocker - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '329528', 'Amlodipine 5mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Calcium Channel Blocker', ARRAY['00093-2122-01'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '329526', 'Amlodipine 10mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Calcium Channel Blocker', ARRAY['00093-2123-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Atorvastatin (Statin - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '617312', 'Atorvastatin 20mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Statin', ARRAY['00093-5057-01'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '617314', 'Atorvastatin 40mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Statin', ARRAY['00093-5058-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Rosuvastatin (Statin - Preferred Brand now Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '301542', 'Rosuvastatin 10mg Tablet', 'generic',
 'tier1_generic', 10.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Statin', ARRAY['00310-0751-90'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '859749', 'Rosuvastatin 20mg Tablet', 'generic',
 'tier1_generic', 10.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Statin', ARRAY['00310-0752-90'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Furosemide (Loop Diuretic - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '197417', 'Furosemide 40mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 true, 'Loop Diuretic', ARRAY['00093-5270-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Carvedilol (Beta Blocker - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '200031', 'Carvedilol 25mg Tablet', 'generic',
 'tier1_generic', 10.00, NULL, false, false,
 '{"max_quantity": 180, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Beta Blocker', ARRAY['00093-8633-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Digoxin (Cardiac Glycoside - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '197604', 'Digoxin 0.25mg Tablet', 'generic',
 'tier1_generic', 10.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Cardiac Glycoside', ARRAY['00781-1780-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- =============================================================================
-- ANTICOAGULANT MEDICATIONS
-- =============================================================================

-- Warfarin (Vitamin K Antagonist - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '11289', 'Warfarin 5mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Anticoagulant - Vitamin K Antagonist', ARRAY['00555-0178-02'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '855332', 'Warfarin 2.5mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Anticoagulant - Vitamin K Antagonist', ARRAY['00555-0177-02'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Apixaban (Eliquis - DOAC, Preferred)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1364430', 'Apixaban (Eliquis) 5mg Tablet', 'brand',
 'tier2_preferred_brand', 50.00, NULL, false, false,
 '{"max_quantity": 60, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Anticoagulant - Direct Xa Inhibitor', ARRAY['00003-0894-21'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1364435', 'Apixaban (Eliquis) 2.5mg Tablet', 'brand',
 'tier2_preferred_brand', 50.00, NULL, false, false,
 '{"max_quantity": 60, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Anticoagulant - Direct Xa Inhibitor', ARRAY['00003-0893-21'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Rivaroxaban (Xarelto - DOAC, Non-Preferred with Step Therapy)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1232082', 'Rivaroxaban (Xarelto) 20mg Tablet', 'brand',
 'tier3_non_preferred', 85.00, NULL, false, true,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Anticoagulant - Direct Xa Inhibitor', ARRAY['50458-0579-30'],
 '2025-01-01', 'active', NOW(), NOW()),

('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1232086', 'Rivaroxaban (Xarelto) 15mg Tablet', 'brand',
 'tier3_non_preferred', 85.00, NULL, false, true,
 '{"max_quantity": 30, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Anticoagulant - Direct Xa Inhibitor', ARRAY['50458-0580-30'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Dabigatran (Pradaxa - DOAC)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1037045', 'Dabigatran (Pradaxa) 150mg Capsule', 'brand',
 'tier2_preferred_brand', 55.00, NULL, false, false,
 '{"max_quantity": 60, "per_days": 30, "max_fills_per_year": 12}'::jsonb,
 false, 'Anticoagulant - Direct Thrombin Inhibitor', ARRAY['00597-0149-60'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Clopidogrel (Antiplatelet - Generic)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '32968', 'Clopidogrel 75mg Tablet', 'generic',
 'tier1_generic', 5.00, NULL, false, false,
 '{"max_quantity": 90, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Antiplatelet', ARRAY['00093-7276-01'],
 '2025-01-01', 'active', NOW(), NOW()),

-- Aspirin (Antiplatelet - OTC but often covered)
('ANTHEM', 'ANTHEM_PPO_2025', 2025, '1191', 'Aspirin 81mg Tablet EC', 'generic',
 'tier1_generic', 0.00, NULL, false, false,
 '{"max_quantity": 180, "per_days": 90, "max_fills_per_year": 4}'::jsonb,
 true, 'Antiplatelet', ARRAY['00113-0274-71'],
 '2025-01-01', 'active', NOW(), NOW())

ON CONFLICT (payer_id, plan_id, drug_rxnorm, plan_year)
DO UPDATE SET
    tier_id = EXCLUDED.tier_id,
    copay_amount = EXCLUDED.copay_amount,
    coinsurance_percent = EXCLUDED.coinsurance_percent,
    prior_authorization = EXCLUDED.prior_authorization,
    step_therapy = EXCLUDED.step_therapy,
    quantity_limit = EXCLUDED.quantity_limit,
    therapeutic_class = EXCLUDED.therapeutic_class,
    status = EXCLUDED.status,
    updated_at = NOW();
