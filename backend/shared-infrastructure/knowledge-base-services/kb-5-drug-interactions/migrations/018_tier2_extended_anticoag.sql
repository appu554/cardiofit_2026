-- ============================================================================
-- KB-5 Tier 2: Anticoagulant Panel - Migration 018
-- Extended Anticoagulant Combinations (25 DDIs)
-- Completes Tier 2 Target: 200 DDIs
-- ============================================================================
-- Option C: Iterative Clinical Expansion
-- Three-Layer Governance: Regulatory + Pharmacology + Clinical
-- ============================================================================

-- ============================================================================
-- ADDITIONAL WARFARIN INTERACTIONS
-- ============================================================================

-- 1. Warfarin + Fish Oil/Omega-3 (Bleeding - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_FISHOIL_001',
    'WARFARIN', 'Warfarin',
    'FISHOIL', 'Fish Oil / Omega-3 Fatty Acids',
    'pharmacodynamic',
    'moderate',
    'probable',
    'Omega-3 fatty acids have antiplatelet effects. High doses (>3g/day) may also affect vitamin K-dependent clotting factors.',
    'Potential for increased INR and bleeding risk, especially at high omega-3 doses.',
    'Monitor INR when starting or significantly changing omega-3 intake. Counsel patients about supplement use.',
    FALSE, 0.60, 0.70,
    'FDA_LABEL', 'Lovaza - Section 7 (Anticoagulants)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Omega-3: antiplatelet effect + possible vitamin K displacement at high doses', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'AHA Omega-3 Scientific Statement',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 2. Warfarin + Coenzyme Q10 (Reduced effect - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_COQ10_001',
    'WARFARIN', 'Warfarin',
    'COQ10', 'Coenzyme Q10',
    'pharmacodynamic',
    'moderate',
    'probable',
    'CoQ10 is structurally similar to vitamin K2. May have pro-coagulant effect opposing warfarin.',
    'Potential for reduced INR and warfarin effect. Case reports of INR reduction.',
    'Monitor INR when starting CoQ10 supplementation. May need warfarin dose increase.',
    TRUE, 0.35, 0.65,
    'FDA_LABEL', 'Coumadin - Section 7 (Dietary supplements)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'CoQ10: structural similarity to vitamin K; May compete at VKORC1', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'Natural Medicines Database',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 3. Warfarin + Aprepitant (CYP2C9 inducer - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_APREPITANT_001',
    'WARFARIN', 'Warfarin',
    'APREPITANT', 'Aprepitant (Emend)',
    'pharmacokinetic',
    'moderate',
    'established',
    'Aprepitant is a CYP2C9 inducer. Reduces S-warfarin levels during treatment and for 2 weeks after.',
    'INR may decrease during aprepitant administration. Common scenario in chemotherapy patients on warfarin.',
    'Monitor INR closely during and for 2 weeks after aprepitant. May need warfarin dose increase.',
    TRUE, 0.30, 0.75,
    'FDA_LABEL', 'Emend - Section 7 (Warfarin)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Aprepitant: CYP2C9 inducer; S-warfarin Cmin reduced 34%', NULL, 'CYP2C9 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ASCO Antiemetic Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 4. Warfarin + Azathioprine (Reduced effect - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_AZATHIOPRINE_001',
    'WARFARIN', 'Warfarin',
    'AZATHIOPRINE', 'Azathioprine',
    'unknown',
    'moderate',
    'probable',
    'Mechanism unclear. May involve inhibition of warfarin absorption or enhanced hepatic enzyme activity.',
    'Reduced warfarin effect requiring higher doses. Common in IBD and transplant patients.',
    'Monitor INR when starting/stopping azathioprine. May need significant warfarin dose adjustment.',
    TRUE, 0.25, 0.72,
    'FDA_LABEL', 'Coumadin - Section 7; Imuran - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Unknown mechanism; Consistent clinical observations of reduced warfarin effect', NULL, 'Unknown pathway', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACG IBD Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 5. Warfarin + Prednisone (Variable effect - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_PREDNISONE_001',
    'WARFARIN', 'Warfarin',
    'PREDNISONE', 'Prednisone/Corticosteroids',
    'pharmacodynamic',
    'moderate',
    'established',
    'High-dose corticosteroids may have procoagulant effect. Also may affect clotting factor synthesis. Variable effects reported.',
    'INR may increase or decrease. Effect is dose-dependent and patient-variable. Often overlooked interaction.',
    'Monitor INR when starting high-dose steroids (>40mg/day prednisone equivalent). Adjust warfarin as needed.',
    TRUE, 0.70, 0.70,
    'FDA_LABEL', 'Coumadin - Section 7 (Corticosteroids)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Variable: high-dose steroids may enhance or reduce warfarin effect depending on patient', NULL, 'Not primarily CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACCP Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- ADDITIONAL DOAC INTERACTIONS
-- ============================================================================

-- 6. Apixaban + Naproxen (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_NAPROXEN_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'NAPROXEN', 'Naproxen',
    'pharmacodynamic',
    'major',
    'established',
    'Naproxen has long-duration antiplatelet effect and GI toxicity. Higher GI bleeding risk than ibuprofen.',
    'Significantly elevated GI bleeding risk. Naproxen longer half-life extends risk window.',
    'Avoid if possible. Use acetaminophen. If NSAID needed, prefer short-acting with PPI protection.',
    FALSE, 0.65, 0.88,
    'FDA_LABEL', 'Eliquis - Section 7 (NSAIDs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: antiplatelet + GI toxicity; Long naproxen t1/2 extends risk', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR NSAID Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 7. Rivaroxaban + Fluconazole (CYP3A4/P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_FLUCONAZOLE_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'FLUCONAZOLE', 'Fluconazole',
    'pharmacokinetic',
    'moderate',
    'established',
    'Fluconazole is a moderate CYP3A4 inhibitor. Rivaroxaban exposure increased approximately 40%.',
    'Increased bleeding risk, particularly with higher fluconazole doses (≥400mg).',
    'Use caution, especially with high-dose fluconazole. Monitor for bleeding. Short courses usually tolerable.',
    TRUE, 0.40, 0.78,
    'FDA_LABEL', 'Xarelto - Section 7 (Moderate CYP3A4 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Fluconazole: moderate CYP3A4 inhibitor; Rivaroxaban AUC +42%', 'P-gp inhibitor (weak)', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Candidiasis Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 8. Edoxaban + Quinidine (P-gp inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EDOXABAN_QUINIDINE_001',
    'EDOXABAN', 'Edoxaban (Savaysa)',
    'QUINIDINE', 'Quinidine',
    'pharmacokinetic',
    'major',
    'established',
    'Quinidine is a potent P-gp inhibitor. Edoxaban AUC increased approximately 77%.',
    'Nearly doubled edoxaban exposure with significantly elevated bleeding risk.',
    'Reduce edoxaban dose to 30mg once daily when used with quinidine.',
    TRUE, 0.15, 0.90,
    'FDA_LABEL', 'Savaysa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Quinidine: potent P-gp inhibitor; Edoxaban AUC +77%', 'P-gp inhibition', 'Not CYP-mediated for edoxaban', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP DOAC Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 9. Dabigatran + Quinidine (P-gp inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_QUINIDINE_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'QUINIDINE', 'Quinidine',
    'pharmacokinetic',
    'major',
    'established',
    'Quinidine is a potent P-gp inhibitor. Dabigatran AUC increased approximately 50%.',
    'Significantly elevated bleeding risk from increased dabigatran exposure.',
    'Consider dabigatran dose reduction or alternative anticoagulation strategy.',
    TRUE, 0.15, 0.88,
    'FDA_LABEL', 'Pradaxa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Quinidine: potent P-gp inhibitor; Dabigatran AUC +53%', 'P-gp inhibition', 'No CYP metabolism', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 10. Apixaban + Diltiazem (CYP3A4/P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_DILTIAZEM_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'DILTIAZEM', 'Diltiazem',
    'pharmacokinetic',
    'moderate',
    'established',
    'Diltiazem is a moderate CYP3A4 inhibitor. Apixaban AUC increased approximately 40%.',
    'Modestly increased bleeding risk. Common combination in AF patients.',
    'No routine dose adjustment but be aware of interaction. Monitor for bleeding in high-risk patients.',
    FALSE, 0.60, 0.72,
    'FDA_LABEL', 'Eliquis - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Diltiazem: moderate CYP3A4 inhibitor; Apixaban AUC +40%', 'Weak P-gp inhibitor', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA AF Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- REVERSAL AGENT AND SPECIAL CONSIDERATIONS
-- ============================================================================

-- 11. Idarucizumab + Dabigatran (Reversal - THERAPEUTIC)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'IDARUCIZUMAB_DABIGATRAN_001',
    'IDARUCIZUMAB', 'Idarucizumab (Praxbind)',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'pharmacodynamic',
    'moderate',
    'established',
    'Idarucizumab is a humanized antibody fragment that binds dabigatran with 350x higher affinity than thrombin.',
    'Complete reversal of dabigatran anticoagulation within minutes. Used for life-threatening bleeding or urgent surgery.',
    'Dose: 5g IV (two 2.5g vials). Can restart dabigatran 24 hours after idarucizumab if clinically appropriate.',
    FALSE, 0.15, 0.99,
    'FDA_LABEL', 'Praxbind - Section 1 (Specific reversal agent for dabigatran)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Monoclonal antibody fragment; Binds dabigatran with Kd = 2.1 pM', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Therapeutic Interaction', 'REVERSE-AD Trial',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 12. Andexanet alfa + Factor Xa Inhibitors (Reversal - THERAPEUTIC)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ANDEXANET_FACTORXA_001',
    'ANDEXANET', 'Andexanet alfa (Andexxa)',
    'FACTORXA_INHIBITORS', 'Factor Xa Inhibitors (rivaroxaban, apixaban)',
    'pharmacodynamic',
    'moderate',
    'established',
    'Andexanet alfa is a modified recombinant Factor Xa that binds and sequesters Factor Xa inhibitors.',
    'Reverses anticoagulant effect of rivaroxaban and apixaban. Used for life-threatening or uncontrolled bleeding.',
    'Dose depends on last dose and timing of Factor Xa inhibitor. Low dose: 400mg bolus + 480mg infusion. High dose: 800mg bolus + 960mg infusion.',
    FALSE, 0.10, 0.98,
    'FDA_LABEL', 'Andexxa - Section 1 (Factor Xa inhibitor reversal)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Decoy Factor Xa; Binds and sequesters rivaroxaban/apixaban', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Therapeutic Interaction', 'ANNEXA-4 Trial',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 13. Vitamin K + Warfarin (Reversal - THERAPEUTIC)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'VITAMINK_WARFARIN_002',
    'VITAMINK', 'Vitamin K (phytonadione)',
    'WARFARIN', 'Warfarin',
    'pharmacodynamic',
    'moderate',
    'established',
    'Vitamin K directly opposes warfarin by restoring vitamin K-dependent clotting factor synthesis.',
    'Reverses warfarin anticoagulation. Time to effect: 6-24 hours (oral), 1-6 hours (IV). Can cause warfarin resistance.',
    'Dose based on INR and bleeding severity. Low-dose oral (1-2.5mg) for non-bleeding elevated INR. IV 5-10mg for serious bleeding.',
    TRUE, 0.50, 0.99,
    'FDA_LABEL', 'Mephyton/Vitamin K1 - Section 1', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Vitamin K: direct antagonist of warfarin; Restores VKORC1 function', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Therapeutic Interaction', 'ACCP Chest Guidelines - Warfarin Reversal',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 14. Protamine + Heparin (Reversal - THERAPEUTIC)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PROTAMINE_HEPARIN_001',
    'PROTAMINE', 'Protamine Sulfate',
    'HEPARIN', 'Heparin (UFH)',
    'pharmacodynamic',
    'moderate',
    'established',
    'Protamine is a positively charged protein that binds and neutralizes negatively charged heparin.',
    'Immediate reversal of heparin anticoagulation. 100% neutralization of UFH. Partial reversal of LMWH.',
    'Dose: 1mg protamine per 100 units heparin given in past 2 hours. Maximum 50mg. Caution: can cause hypotension and anaphylaxis.',
    TRUE, 0.40, 0.98,
    'FDA_LABEL', 'Protamine Sulfate - Section 1', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Electrostatic neutralization of heparin; Positively charged protamine binds anionic heparin', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Therapeutic Interaction', 'Cardiac Surgery Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- ADDITIONAL ANTIPLATELET AND SPECIAL COMBINATIONS
-- ============================================================================

-- 15. Clopidogrel + Morphine (Reduced absorption - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CLOPIDOGREL_MORPHINE_001',
    'CLOPIDOGREL', 'Clopidogrel',
    'MORPHINE', 'Morphine',
    'pharmacokinetic',
    'moderate',
    'established',
    'Morphine delays gastric emptying, reducing clopidogrel absorption. Active metabolite levels reduced 30-50%.',
    'Delayed and reduced antiplatelet effect in ACS. May contribute to stent thrombosis risk.',
    'Consider IV P2Y12 inhibitor (cangrelor) or crushed/chewed prasugrel in STEMI. Avoid morphine if possible.',
    FALSE, 0.60, 0.80,
    'FDA_LABEL', 'ESC STEMI Guidelines 2023', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Morphine: delays gastric emptying; Clopidogrel absorption and activation delayed', NULL, 'Not CYP interaction but affects absorption', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ESC STEMI Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 16. Ticagrelor + Morphine (Reduced absorption - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TICAGRELOR_MORPHINE_001',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'MORPHINE', 'Morphine',
    'pharmacokinetic',
    'moderate',
    'established',
    'Morphine delays gastric emptying, reducing ticagrelor absorption. Effect on active drug levels.',
    'Delayed onset of antiplatelet effect in ACS. ATLANTIC trial showed impact.',
    'Consider cangrelor bridge in high-risk STEMI. Crushed ticagrelor may help. Minimize morphine use.',
    FALSE, 0.55, 0.78,
    'FDA_LABEL', 'Brilinta - Section 7; ATLANTIC Trial', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Morphine: delayed gastric emptying; Ticagrelor Cmax reduced ~30%', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ESC STEMI Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 17. Prasugrel + Morphine (Reduced absorption - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PRASUGREL_MORPHINE_001',
    'PRASUGREL', 'Prasugrel (Effient)',
    'MORPHINE', 'Morphine',
    'pharmacokinetic',
    'moderate',
    'established',
    'Morphine delays gastric emptying, reducing prasugrel absorption. Similar mechanism to clopidogrel.',
    'Delayed antiplatelet effect onset in ACS. Prasugrel may be less affected than clopidogrel.',
    'Crushed prasugrel may mitigate effect. Consider cangrelor bridge in high-risk patients.',
    FALSE, 0.50, 0.75,
    'FDA_LABEL', 'Effient - Clinical Pharmacology', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Morphine: delays gastric emptying; Prasugrel absorption affected', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ESC STEMI Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 18. Rivaroxaban + Amiodarone (P-gp/CYP3A4 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_AMIODARONE_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'AMIODARONE', 'Amiodarone',
    'pharmacokinetic',
    'moderate',
    'established',
    'Amiodarone is a moderate P-gp and weak CYP3A4 inhibitor. Rivaroxaban exposure modestly increased.',
    'Increased bleeding risk. Common combination in AF patients. Long amiodarone t1/2 prolongs effect.',
    'No routine dose adjustment but monitor for bleeding, especially in elderly or renal impairment.',
    FALSE, 0.50, 0.75,
    'FDA_LABEL', 'Xarelto - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Amiodarone: P-gp inhibitor; Rivaroxaban exposure modestly increased', 'P-gp inhibition', 'CYP3A4 (weak)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA AF Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 19. Apixaban + Amiodarone (P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_AMIODARONE_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'AMIODARONE', 'Amiodarone',
    'pharmacokinetic',
    'moderate',
    'established',
    'Amiodarone inhibits P-gp and weakly CYP3A4. Apixaban exposure modestly increased.',
    'Modestly increased bleeding risk. Common combination in AF management.',
    'No routine dose adjustment. Be aware of interaction in high-risk patients.',
    FALSE, 0.55, 0.72,
    'FDA_LABEL', 'Eliquis - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Amiodarone: P-gp + weak CYP3A4 inhibitor; Apixaban exposure modestly increased', 'P-gp inhibition', 'CYP3A4 (weak)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA AF Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 20. Bivalirudin + Antiplatelets (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'BIVALIRUDIN_ANTIPLATELET_001',
    'BIVALIRUDIN', 'Bivalirudin (Angiomax)',
    'ANTIPLATELETS', 'Antiplatelets (class)',
    'pharmacodynamic',
    'major',
    'established',
    'Bivalirudin is a direct thrombin inhibitor. Combined with antiplatelets = high bleeding risk at access sites.',
    'Standard combination in PCI but requires meticulous hemostasis. Lower bleeding than heparin+GPI.',
    'Use radial access to minimize bleeding. Meticulous hemostasis at access sites. HORIZONS-AMI protocol.',
    FALSE, 0.45, 0.88,
    'FDA_LABEL', 'Angiomax - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Direct thrombin inhibitor + antiplatelet = additive antithrombotic effect', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'HORIZONS-AMI Trial; SCAI PCI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 21. Argatroban + Warfarin (INR falsely elevated - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ARGATROBAN_WARFARIN_001',
    'ARGATROBAN', 'Argatroban',
    'WARFARIN', 'Warfarin',
    'pharmacodynamic',
    'major',
    'established',
    'Argatroban elevates INR independent of warfarin. Combined INR does not reflect warfarin anticoagulation alone.',
    'INR is falsely elevated during transition from argatroban to warfarin. Risk of stopping argatroban too early.',
    'Use chromogenic Factor X assay or wait until INR >4 on combination. Target INR on warfarin alone after argatroban stop.',
    TRUE, 0.30, 0.90,
    'FDA_LABEL', 'Argatroban - Section 7 (Warfarin transition)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Both drugs affect INR; Combined INR overestimates warfarin effect', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASH HIT Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 22. Dalteparin + Aspirin (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DALTEPARIN_ASPIRIN_001',
    'DALTEPARIN', 'Dalteparin (Fragmin)',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive antithrombotic effect. LMWH + aspirin = increased bleeding risk vs either alone.',
    'Increased bleeding risk. Standard of care in ACS but requires monitoring.',
    'Use per ACS guidelines. Monitor for bleeding signs. Use lowest effective aspirin dose.',
    FALSE, 0.65, 0.86,
    'FDA_LABEL', 'Fragmin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: dual antithrombotic mechanism', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA ACS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 23. Tinzaparin + NSAIDs (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TINZAPARIN_NSAIDS_001',
    'TINZAPARIN', 'Tinzaparin (Innohep)',
    'NSAIDS', 'NSAIDs (class)',
    'pharmacodynamic',
    'major',
    'established',
    'NSAIDs cause antiplatelet effect and GI mucosal injury. Combined with LMWH = high bleeding risk.',
    'Significantly increased bleeding risk, especially GI.',
    'Avoid NSAIDs. Use acetaminophen. If NSAID necessary, use lowest dose with PPI.',
    FALSE, 0.55, 0.85,
    'FDA_LABEL', 'Innohep - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: antiplatelet + GI toxicity from NSAIDs', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR NSAID Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 24. Betrixaban + P-gp Inhibitors (Elevated levels - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'BETRIXABAN_PGPINH_001',
    'BETRIXABAN', 'Betrixaban (Bevyxxa)',
    'PGP_INHIBITORS', 'P-gp Inhibitors (class)',
    'pharmacokinetic',
    'major',
    'established',
    'Betrixaban is primarily eliminated unchanged via P-gp efflux and biliary excretion. P-gp inhibitors increase exposure.',
    'Increased betrixaban levels and bleeding risk.',
    'Reduce betrixaban dose from 160mg to 80mg loading, 40mg daily when used with P-gp inhibitors.',
    TRUE, 0.25, 0.88,
    'FDA_LABEL', 'Bevyxxa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Betrixaban: P-gp substrate; Minimal CYP metabolism; P-gp inhibition increases levels', 'P-gp substrate', 'Minimal CYP involvement', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APEX Trial',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 25. Sulfinpyrazone + Warfarin (CYP2C9 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'SULFINPYRAZONE_WARFARIN_001',
    'SULFINPYRAZONE', 'Sulfinpyrazone',
    'WARFARIN', 'Warfarin',
    'pharmacokinetic',
    'major',
    'established',
    'Sulfinpyrazone inhibits CYP2C9 (S-warfarin metabolism) and may displace warfarin from protein binding.',
    'Significantly increased INR and bleeding risk. Historically significant interaction.',
    'Avoid combination. If sulfinpyrazone started, reduce warfarin dose significantly and monitor INR closely.',
    TRUE, 0.10, 0.90,
    'FDA_LABEL', 'Coumadin - Section 7 (Sulfinpyrazone)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Sulfinpyrazone: CYP2C9 inhibitor + protein binding displacement', NULL, 'CYP2C9', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'Historical interaction literature',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Final Verification for Tier 2 Target
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    IF v_count >= 200 THEN
        RAISE NOTICE '★★★ TIER 2 TARGET ACHIEVED: % DDIs (Target: 200) ★★★', v_count;
    ELSE
        RAISE NOTICE '★ Total active DDIs: % (Target: 200)', v_count;
    END IF;
END $$;
