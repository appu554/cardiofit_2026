-- ============================================================================
-- KB-5 Tier 2: Anticoagulant Panel - Migration 019
-- Supplemental DDIs to Complete Tier 2 (8 DDIs)
-- Completes Tier 2 Target: 200 DDIs
-- ============================================================================

-- 1. Warfarin + Vitamin E (Bleeding - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_VITAMINE_001',
    'WARFARIN', 'Warfarin',
    'VITAMINE', 'Vitamin E (high dose)',
    'pharmacodynamic',
    'moderate',
    'probable',
    'High-dose vitamin E (>400 IU/day) may have antiplatelet effects and interfere with vitamin K cycle.',
    'Potential for increased INR and bleeding risk with high-dose supplementation.',
    'Monitor INR when starting high-dose vitamin E. Counsel patients about supplement use.',
    FALSE, 0.40, 0.68,
    'FDA_LABEL', 'Coumadin - Section 7 (Vitamins)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Vitamin E: may inhibit vitamin K-dependent carboxylation; Antiplatelet effect', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'Natural Medicines Database',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 2. Warfarin + Garlic supplements (Bleeding - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_GARLIC_001',
    'WARFARIN', 'Warfarin',
    'GARLIC', 'Garlic (supplements)',
    'pharmacodynamic',
    'moderate',
    'probable',
    'Garlic has antiplatelet activity (ajoene inhibits platelet aggregation). May potentiate warfarin bleeding.',
    'Increased bleeding risk with concentrated garlic supplements. Case reports of excessive INR.',
    'Advise consistent intake. Monitor INR with significant changes in garlic consumption.',
    FALSE, 0.35, 0.65,
    'FDA_LABEL', 'Coumadin - Section 7 (Herbal products)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Garlic (ajoene): inhibits platelet aggregation; Additive bleeding risk', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'Natural Medicines Database',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 3. Apixaban + Aspirin + Clopidogrel Triple Therapy (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_TRIPLE_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'TRIPLE_THERAPY', 'DAPT (Aspirin + P2Y12 inhibitor)',
    'pharmacodynamic',
    'major',
    'established',
    'Triple antithrombotic therapy: Factor Xa inhibitor + ASA + P2Y12 inhibitor. Maximum bleeding risk.',
    'Very high major bleeding risk. AUGUSTUS trial showed apixaban preferable to warfarin in this setting.',
    'Use apixaban 5mg BID (2.5mg if dose reduction criteria met). Shorten dual antiplatelet duration. Add PPI.',
    FALSE, 0.35, 0.95,
    'FDA_LABEL', 'Eliquis - Section 7; AUGUSTUS Trial', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Triple antithrombotic: Factor Xa + COX-1 + P2Y12 = very high bleeding', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC AF + PCI Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 4. Rivaroxaban + Prasugrel (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_PRASUGREL_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'PRASUGREL', 'Prasugrel (Effient)',
    'pharmacodynamic',
    'major',
    'established',
    'Rivaroxaban is a Factor Xa inhibitor. Prasugrel is a potent P2Y12 inhibitor. Combined = very high bleeding.',
    'Very high bleeding risk. Generally prasugrel avoided in AF + PCI due to bleeding concerns.',
    'Use clopidogrel instead of prasugrel with rivaroxaban. If prasugrel necessary, shortest possible duration.',
    FALSE, 0.20, 0.94,
    'FDA_LABEL', 'Xarelto - Section 7; Effient - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Dual antithrombotic: Factor Xa + potent antiplatelet = high bleeding risk', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC AF + PCI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 5. Dabigatran + Ticagrelor (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_TICAGRELOR_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'pharmacodynamic',
    'major',
    'established',
    'Direct thrombin inhibitor + potent P2Y12 inhibitor. Ticagrelor also inhibits P-gp (may increase dabigatran).',
    'High bleeding risk with possible PK interaction. Limited data on this combination.',
    'Consider apixaban instead (AUGUSTUS data). If dabigatran used, monitor closely for bleeding.',
    FALSE, 0.25, 0.90,
    'FDA_LABEL', 'Pradaxa - Section 7; Brilinta - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'DTI + potent antiplatelet; Ticagrelor P-gp effect may increase dabigatran', 'Ticagrelor: P-gp inhibitor', 'Not primary CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC AF Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 6. Edoxaban + Ticagrelor (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EDOXABAN_TICAGRELOR_001',
    'EDOXABAN', 'Edoxaban (Savaysa)',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'pharmacodynamic',
    'major',
    'established',
    'Factor Xa inhibitor + potent P2Y12 inhibitor. Ticagrelor P-gp inhibition may increase edoxaban.',
    'High bleeding risk. Limited clinical data for this specific combination.',
    'Reduce edoxaban if P-gp interaction significant. Monitor closely for bleeding.',
    FALSE, 0.20, 0.88,
    'FDA_LABEL', 'Savaysa - Section 7; Brilinta - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Factor Xa inhibitor + potent antiplatelet; P-gp-mediated PK interaction possible', 'Ticagrelor: P-gp inhibitor', 'Not CYP interaction for edoxaban', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC AF Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 7. Heparin + Thrombolytics (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_THROMBOLYTIC_001',
    'HEPARIN', 'Heparin (UFH)',
    'THROMBOLYTIC', 'Thrombolytics (alteplase, tenecteplase)',
    'pharmacodynamic',
    'major',
    'established',
    'Heparin anticoagulation + fibrinolysis = dramatically increased bleeding risk.',
    'Very high bleeding risk including intracranial hemorrhage. Standard in STEMI but critical risk management.',
    'Use per STEMI protocols. Reduced heparin dose after thrombolysis. Monitor aPTT closely.',
    TRUE, 0.30, 0.96,
    'FDA_LABEL', 'tPA/Alteplase - Section 7; Heparin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Anticoagulation + fibrinolysis = additive bleeding; Critical hemostatic impairment', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA STEMI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 8. Enoxaparin + Fondaparinux (Overlapping anticoagulation - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ENOXAPARIN_FONDAPARINUX_001',
    'ENOXAPARIN', 'Enoxaparin (Lovenox)',
    'FONDAPARINUX', 'Fondaparinux (Arixtra)',
    'pharmacodynamic',
    'major',
    'established',
    'Both are Factor Xa inhibitors (LMWH + selective Xa). No therapeutic rationale for combination.',
    'Overlapping anticoagulation with doubled bleeding risk. No clinical benefit.',
    'Never combine. These are alternative anticoagulants, not additive therapies.',
    FALSE, 0.05, 0.98,
    'FDA_LABEL', 'Lovenox - Section 7; Arixtra - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Redundant Factor Xa inhibition; No therapeutic rationale; Pure bleeding risk', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ACCP Anticoagulation Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Final Verification for Tier 2 Target
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    IF v_count >= 200 THEN
        RAISE NOTICE '★★★ TIER 2 TARGET ACHIEVED: % DDIs ★★★', v_count;
    ELSE
        RAISE NOTICE '★ Total active DDIs: % (Target: 200)', v_count;
    END IF;
END $$;
