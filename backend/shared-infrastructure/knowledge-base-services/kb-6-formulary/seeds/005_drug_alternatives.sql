-- ============================================================================
-- KB-6 Formulary Service - Drug Alternatives Seed Data
-- ============================================================================
-- Generic and therapeutic alternatives for cost optimization
-- ============================================================================

-- =============================================================================
-- GENERIC ALTERNATIVES (Brand → Generic substitution)
-- =============================================================================

-- Note: Most drugs in our formulary are already generic, but we document
-- the brand-generic relationships for reference and AB-rating lookups

INSERT INTO drug_alternatives (
    source_rxnorm, source_name, alternative_rxnorm, alternative_name,
    alternative_type, efficacy_rating, cost_savings_percent,
    therapeutic_class, clinical_notes, ab_rated,
    effective_date, status
) VALUES
-- Lipitor → Atorvastatin (AB-rated generic)
('83367', 'Lipitor (Atorvastatin) 20mg', '617312', 'Atorvastatin 20mg',
 'generic', 1.00, 95.00,
 'Statin', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

('83369', 'Lipitor (Atorvastatin) 40mg', '617314', 'Atorvastatin 40mg',
 'generic', 1.00, 95.00,
 'Statin', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

-- Crestor → Rosuvastatin (AB-rated generic)
('859747', 'Crestor (Rosuvastatin) 10mg', '301542', 'Rosuvastatin 10mg',
 'generic', 1.00, 90.00,
 'Statin', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

-- Glucophage → Metformin (AB-rated generic)
('6809', 'Glucophage (Metformin) 500mg', '6809', 'Metformin 500mg',
 'generic', 1.00, 85.00,
 'Antidiabetic - Biguanide', 'Same molecule, different branding', true,
 '2025-01-01', 'active'),

-- Toprol XL → Metoprolol Succinate ER (AB-rated generic)
('866924', 'Toprol XL 50mg', '866924', 'Metoprolol Succinate ER 50mg',
 'generic', 1.00, 80.00,
 'Beta Blocker', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

-- Norvasc → Amlodipine (AB-rated generic)
('329528', 'Norvasc 5mg', '329528', 'Amlodipine 5mg',
 'generic', 1.00, 90.00,
 'Calcium Channel Blocker', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

-- Prinivil/Zestril → Lisinopril (AB-rated generic)
('29046', 'Zestril (Lisinopril) 10mg', '29046', 'Lisinopril 10mg',
 'generic', 1.00, 92.00,
 'ACE Inhibitor', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

-- Cozaar → Losartan (AB-rated generic)
('52175', 'Cozaar (Losartan) 50mg', '52175', 'Losartan 50mg',
 'generic', 1.00, 88.00,
 'ARB', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

-- Plavix → Clopidogrel (AB-rated generic)
('32968', 'Plavix (Clopidogrel) 75mg', '32968', 'Clopidogrel 75mg',
 'generic', 1.00, 85.00,
 'Antiplatelet', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active'),

-- Coumadin → Warfarin (AB-rated generic)
('11289', 'Coumadin 5mg', '11289', 'Warfarin 5mg',
 'generic', 1.00, 75.00,
 'Anticoagulant', 'FDA AB-rated therapeutic equivalent', true,
 '2025-01-01', 'active')

ON CONFLICT (source_rxnorm, alternative_rxnorm) DO UPDATE SET
    efficacy_rating = EXCLUDED.efficacy_rating,
    cost_savings_percent = EXCLUDED.cost_savings_percent,
    status = EXCLUDED.status,
    updated_at = NOW();

-- =============================================================================
-- THERAPEUTIC ALTERNATIVES (Same class, different molecules)
-- =============================================================================

INSERT INTO drug_alternatives (
    source_rxnorm, source_name, alternative_rxnorm, alternative_name,
    alternative_type, efficacy_rating, cost_savings_percent,
    therapeutic_class, clinical_notes, ab_rated,
    effective_date, status
) VALUES
-- SGLT2 Inhibitor Alternatives
('1373458', 'Invokana (Canagliflozin) 100mg', '1545653', 'Jardiance (Empagliflozin) 10mg',
 'therapeutic', 0.95, 45.00,
 'SGLT2 Inhibitor', 'Preferred SGLT2i with similar CV outcomes. EMPA-REG demonstrated CV mortality benefit.',
 false, '2025-01-01', 'active'),

('1373458', 'Invokana (Canagliflozin) 100mg', '1486436', 'Farxiga (Dapagliflozin) 10mg',
 'therapeutic', 0.95, 40.00,
 'SGLT2 Inhibitor', 'Alternative SGLT2i. DAPA-HF showed heart failure benefit.',
 false, '2025-01-01', 'active'),

-- GLP-1 Agonist Alternatives
('1991302', 'Ozempic (Semaglutide) 0.25mg', '897122', 'Victoza (Liraglutide) 18mg',
 'therapeutic', 0.85, 20.00,
 'GLP-1 Agonist', 'Daily injection vs weekly. Semaglutide shows greater A1c reduction in SUSTAIN trials.',
 false, '2025-01-01', 'active'),

-- Statin Therapeutic Alternatives (Intensity-based)
('617312', 'Atorvastatin 20mg', '301542', 'Rosuvastatin 10mg',
 'therapeutic', 0.95, -5.00,
 'Statin', 'Moderate-intensity statin. Similar LDL reduction. Consider patient preference.',
 false, '2025-01-01', 'active'),

('617314', 'Atorvastatin 40mg', '859749', 'Rosuvastatin 20mg',
 'therapeutic', 0.95, 0.00,
 'Statin', 'High-intensity statin alternatives. Equivalent LDL reduction expected.',
 false, '2025-01-01', 'active'),

-- ACE Inhibitor Alternatives
('29046', 'Lisinopril 10mg', '52175', 'Losartan 50mg',
 'therapeutic', 0.90, 0.00,
 'RAAS Inhibitor', 'ARB alternative for patients with ACE inhibitor cough. Similar CV protection.',
 false, '2025-01-01', 'active'),

-- DOAC Alternatives
('1232082', 'Xarelto (Rivaroxaban) 20mg', '1364430', 'Eliquis (Apixaban) 5mg',
 'therapeutic', 0.95, 40.00,
 'DOAC', 'Preferred DOAC on formulary. ARISTOTLE trial showed lower bleeding risk vs warfarin.',
 false, '2025-01-01', 'active'),

('1232082', 'Xarelto (Rivaroxaban) 20mg', '1037045', 'Pradaxa (Dabigatran) 150mg',
 'therapeutic', 0.90, 35.00,
 'DOAC', 'Alternative DOAC. RE-LY trial demonstrated non-inferiority to warfarin. Has reversal agent.',
 false, '2025-01-01', 'active'),

('1364430', 'Eliquis (Apixaban) 5mg', '11289', 'Warfarin 5mg',
 'therapeutic', 0.85, 90.00,
 'Anticoagulant', 'Traditional anticoagulant. Requires INR monitoring. Lower cost but more management.',
 false, '2025-01-01', 'active'),

-- Beta Blocker Alternatives
('866924', 'Metoprolol Succinate ER 50mg', '200031', 'Carvedilol 25mg',
 'therapeutic', 0.90, 0.00,
 'Beta Blocker', 'Alpha/beta blocker. Preferred in heart failure with reduced EF per guidelines.',
 false, '2025-01-01', 'active'),

-- DPP-4 vs SGLT2 (Therapeutic class alternative)
('593411', 'Januvia (Sitagliptin) 100mg', '1545653', 'Jardiance (Empagliflozin) 10mg',
 'therapeutic', 0.85, 10.00,
 'Oral Antidiabetic', 'SGLT2i preferred for patients with CVD or HF per ADA guidelines. Different mechanism.',
 false, '2025-01-01', 'active')

ON CONFLICT (source_rxnorm, alternative_rxnorm) DO UPDATE SET
    efficacy_rating = EXCLUDED.efficacy_rating,
    cost_savings_percent = EXCLUDED.cost_savings_percent,
    clinical_notes = EXCLUDED.clinical_notes,
    updated_at = NOW();

-- =============================================================================
-- TIER-OPTIMIZED ALTERNATIVES (Lower tier = lower cost)
-- =============================================================================

INSERT INTO drug_alternatives (
    source_rxnorm, source_name, alternative_rxnorm, alternative_name,
    alternative_type, efficacy_rating, cost_savings_percent,
    therapeutic_class, clinical_notes, tier_improvement, ab_rated,
    effective_date, status
) VALUES
-- Tier 4 → Tier 2 alternatives for diabetes
('1991302', 'Ozempic (Semaglutide)', '1545653', 'Jardiance (Empagliflozin)',
 'tier_optimized', 0.85, 60.00,
 'Antidiabetic', 'If glycemic target can be met with SGLT2i alone. Consider for patients without specific GLP-1 indication.',
 2, false, '2025-01-01', 'active'),

('1991302', 'Ozempic (Semaglutide)', '593411', 'Januvia (Sitagliptin)',
 'tier_optimized', 0.80, 55.00,
 'Antidiabetic', 'DPP-4 inhibitor for patients needing modest additional A1c reduction without injection.',
 2, false, '2025-01-01', 'active'),

-- Tier 3 → Tier 2 for SGLT2i
('1373458', 'Invokana (Canagliflozin)', '1545653', 'Jardiance (Empagliflozin)',
 'tier_optimized', 0.95, 45.00,
 'SGLT2 Inhibitor', 'Preferred SGLT2i with excellent CV outcomes data (EMPA-REG OUTCOME).',
 1, false, '2025-01-01', 'active'),

-- Tier 3 → Tier 2 for DOACs
('1232082', 'Xarelto (Rivaroxaban)', '1364430', 'Eliquis (Apixaban)',
 'tier_optimized', 0.95, 40.00,
 'DOAC', 'Preferred DOAC with favorable bleeding profile in ARISTOTLE trial.',
 1, false, '2025-01-01', 'active'),

-- Tier 2 → Tier 1 (Brand to Generic within class)
('1364430', 'Eliquis (Apixaban)', '11289', 'Warfarin',
 'tier_optimized', 0.85, 90.00,
 'Anticoagulant', 'Generic warfarin for cost-sensitive patients willing to accept INR monitoring.',
 3, false, '2025-01-01', 'active'),

('1545653', 'Jardiance (Empagliflozin)', '6809', 'Metformin',
 'tier_optimized', 0.80, 88.00,
 'Antidiabetic', 'First-line metformin if not yet tried. Excellent efficacy and safety profile.',
 1, false, '2025-01-01', 'active')

ON CONFLICT (source_rxnorm, alternative_rxnorm) DO UPDATE SET
    tier_improvement = EXCLUDED.tier_improvement,
    cost_savings_percent = EXCLUDED.cost_savings_percent,
    updated_at = NOW();
