-- ============================================================================
-- KB-5 Tier 2: Anticoagulant Panel - Migration 016
-- DOAC (Direct Oral Anticoagulant) Interactions (25 DDIs)
-- Rivaroxaban, Apixaban, Edoxaban, Dabigatran
-- ============================================================================
-- Option C: Iterative Clinical Expansion
-- Three-Layer Governance: Regulatory + Pharmacology + Clinical
-- ============================================================================

-- ============================================================================
-- RIVAROXABAN INTERACTIONS (CYP3A4 + P-gp substrate)
-- ============================================================================

-- 1. Rivaroxaban + Ketoconazole (CYP3A4 + P-gp inhibition - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_KETOCONAZOLE_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'KETOCONAZOLE', 'Ketoconazole',
    'pharmacokinetic',
    'contraindicated',
    'established',
    'Ketoconazole is a potent dual CYP3A4 and P-gp inhibitor. Rivaroxaban AUC increased 160% and Cmax 70%.',
    'Dramatically increased rivaroxaban exposure with high bleeding risk. Effect is dose-dependent.',
    'Avoid combination. Use alternative antifungal (terbinafine for dermatophytes). If systemic azole needed, consider interrupting rivaroxaban.',
    TRUE, 0.20, 0.99,
    'FDA_LABEL', 'Xarelto - Section 7 (Strong CYP3A4/P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ketoconazole: potent CYP3A4 inhibitor (Ki = 0.015 μM) + P-gp inhibitor; Rivaroxaban AUC +160%', 'P-gp inhibition', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACCP DOAC Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 2. Rivaroxaban + Ritonavir (CYP3A4 + P-gp inhibition - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_RITONAVIR_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'RITONAVIR', 'Ritonavir',
    'pharmacokinetic',
    'contraindicated',
    'established',
    'Ritonavir is the most potent CYP3A4 inhibitor and also inhibits P-gp. Rivaroxaban AUC increased 150%+.',
    'Extreme bleeding risk due to massively elevated rivaroxaban levels. HIV/HCV patients at high risk.',
    'Avoid combination. Use warfarin with close INR monitoring, or LMWH in HIV patients requiring anticoagulation.',
    TRUE, 0.15, 0.99,
    'FDA_LABEL', 'Xarelto - Section 7 (HIV protease inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ritonavir: most potent CYP3A4 inhibitor + P-gp inhibitor; DOAC contraindicated', 'P-gp inhibition', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'DHHS HIV Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 3. Rivaroxaban + Rifampin (CYP3A4 + P-gp induction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_RIFAMPIN_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'RIFAMPIN', 'Rifampin',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin induces both CYP3A4 and P-gp. Rivaroxaban AUC decreased by approximately 50%.',
    'Significantly reduced anticoagulation. High risk of thromboembolism. Cannot overcome with dose increase.',
    'Avoid combination. Use warfarin (can adjust dose) or LMWH during rifampin therapy.',
    TRUE, 0.25, 0.98,
    'FDA_LABEL', 'Xarelto - Section 7 (Strong CYP3A4/P-gp inducers - Avoid)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Rifampin: potent CYP3A4 + P-gp inducer; Rivaroxaban AUC reduced 50%', 'P-gp induction', 'CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ATS TB Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 4. Rivaroxaban + Verapamil (P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_VERAPAMIL_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'VERAPAMIL', 'Verapamil',
    'pharmacokinetic',
    'moderate',
    'established',
    'Verapamil is a moderate P-gp inhibitor with some CYP3A4 inhibition. Rivaroxaban exposure modestly increased.',
    'Increased rivaroxaban levels by approximately 30-40%. Clinical significance varies by patient risk.',
    'Monitor for bleeding signs. Consider dose reduction in patients with additional bleeding risk factors.',
    FALSE, 0.45, 0.75,
    'FDA_LABEL', 'Xarelto - Section 7 (Moderate P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Verapamil: moderate P-gp inhibitor; Rivaroxaban AUC increased ~35%', 'P-gp inhibition', 'CYP3A4 (minor)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ESC AF Guidelines 2023',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 5. Rivaroxaban + Diltiazem (P-gp/CYP3A4 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_DILTIAZEM_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'DILTIAZEM', 'Diltiazem',
    'pharmacokinetic',
    'moderate',
    'established',
    'Diltiazem is a moderate CYP3A4 and weak P-gp inhibitor. Common combination in AF patients.',
    'Rivaroxaban exposure increased approximately 20-30%. Generally well tolerated but monitor bleeding.',
    'No routine dose adjustment but monitor for bleeding, especially in elderly or renal impairment.',
    FALSE, 0.55, 0.70,
    'FDA_LABEL', 'Xarelto - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Diltiazem: moderate CYP3A4 inhibitor; Rivaroxaban AUC +20-30%', 'Weak P-gp inhibitor', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA AF Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- APIXABAN INTERACTIONS (CYP3A4 + P-gp substrate)
-- ============================================================================

-- 6. Apixaban + Ketoconazole (CYP3A4 + P-gp inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_KETOCONAZOLE_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'KETOCONAZOLE', 'Ketoconazole',
    'pharmacokinetic',
    'major',
    'established',
    'Ketoconazole is a dual strong CYP3A4 and P-gp inhibitor. Apixaban AUC increased approximately 2-fold.',
    'Doubled apixaban exposure significantly increases bleeding risk. Effect is clinically significant.',
    'Reduce apixaban dose by 50% (e.g., 5mg BID → 2.5mg BID). Alternatively, avoid ketoconazole.',
    TRUE, 0.20, 0.95,
    'FDA_LABEL', 'Eliquis - Section 7 (Strong dual CYP3A4/P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ketoconazole: strong CYP3A4 + P-gp inhibitor; Apixaban AUC increased 2x', 'P-gp inhibition', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP DOAC Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 7. Apixaban + Ritonavir (CYP3A4 + P-gp inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_RITONAVIR_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'RITONAVIR', 'Ritonavir',
    'pharmacokinetic',
    'major',
    'established',
    'Ritonavir is the strongest CYP3A4 inhibitor and also inhibits P-gp. Apixaban AUC increased 2-fold or more.',
    'Major bleeding risk. Common issue in HIV patients with AF or VTE.',
    'Reduce apixaban dose by 50% with close monitoring. Some experts avoid apixaban with ritonavir entirely.',
    TRUE, 0.15, 0.96,
    'FDA_LABEL', 'Eliquis - Section 7 (HIV protease inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ritonavir: potent CYP3A4 + P-gp inhibitor; Apixaban exposure significantly increased', 'P-gp inhibition', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'DHHS HIV Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 8. Apixaban + Rifampin (CYP3A4 + P-gp induction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_RIFAMPIN_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'RIFAMPIN', 'Rifampin',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin induces CYP3A4 and P-gp. Apixaban AUC decreased by approximately 54%.',
    'Markedly reduced anticoagulation effect. High risk of stroke/thromboembolism.',
    'Avoid combination. Use warfarin (dose adjustable) or LMWH during rifampin therapy.',
    TRUE, 0.25, 0.98,
    'FDA_LABEL', 'Eliquis - Section 7 (Strong CYP3A4/P-gp inducers - Avoid)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Rifampin: strong CYP3A4 + P-gp inducer; Apixaban AUC reduced 54%', 'P-gp induction', 'CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ATS TB Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 9. Apixaban + Carbamazepine (CYP3A4 induction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_CARBAMAZEPINE_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'CARBAMAZEPINE', 'Carbamazepine',
    'pharmacokinetic',
    'major',
    'established',
    'Carbamazepine is a strong CYP3A4 inducer with modest P-gp induction. Apixaban levels significantly reduced.',
    'Reduced anticoagulation, risk of thromboembolism. Common issue in epilepsy + AF patients.',
    'Avoid combination. Consider levetiracetam (no enzyme interaction) for seizures, warfarin for anticoagulation.',
    TRUE, 0.20, 0.92,
    'FDA_LABEL', 'Eliquis - Section 7 (Strong CYP3A4 inducers)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Carbamazepine: strong CYP3A4 inducer; Apixaban AUC reduced ~50%', 'P-gp induction (moderate)', 'CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D-Avoid)', 'ILAE Epilepsy Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 10. Apixaban + Phenytoin (CYP3A4 induction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_PHENYTOIN_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'PHENYTOIN', 'Phenytoin',
    'pharmacokinetic',
    'major',
    'established',
    'Phenytoin is a strong CYP3A4 inducer. Expected to significantly reduce apixaban levels.',
    'Reduced anticoagulation effect with increased thromboembolism risk.',
    'Avoid combination. Use non-enzyme-inducing AED or warfarin for anticoagulation.',
    TRUE, 0.18, 0.90,
    'FDA_LABEL', 'Eliquis - Section 7 (Strong CYP3A4 inducers)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Phenytoin: strong CYP3A4 inducer; Expected apixaban AUC reduction >50%', NULL, 'CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D-Avoid)', 'ILAE Epilepsy Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 11. Apixaban + Clarithromycin (CYP3A4 + P-gp inhibition - MODERATE to MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_CLARITHROMYCIN_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'CLARITHROMYCIN', 'Clarithromycin',
    'pharmacokinetic',
    'major',
    'established',
    'Clarithromycin is a strong CYP3A4 inhibitor and moderate P-gp inhibitor. Apixaban exposure increased.',
    'Increased bleeding risk. Short courses (5-7 days) may be acceptable with monitoring.',
    'Use azithromycin as alternative (minimal CYP3A4 effect). If clarithromycin used, consider 50% dose reduction.',
    TRUE, 0.35, 0.85,
    'FDA_LABEL', 'Eliquis - Section 7 (CYP3A4/P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Clarithromycin: strong CYP3A4 + moderate P-gp inhibitor; Apixaban AUC increased ~50%', 'P-gp inhibition', 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA CAP Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- EDOXABAN INTERACTIONS (Primarily P-gp substrate)
-- ============================================================================

-- 12. Edoxaban + Dronedarone (P-gp inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EDOXABAN_DRONEDARONE_001',
    'EDOXABAN', 'Edoxaban (Savaysa)',
    'DRONEDARONE', 'Dronedarone',
    'pharmacokinetic',
    'major',
    'established',
    'Dronedarone is a potent P-gp inhibitor. Edoxaban is primarily eliminated via P-gp. AUC increased ~85%.',
    'Nearly doubled edoxaban exposure with significantly increased bleeding risk.',
    'Reduce edoxaban dose to 30mg once daily when co-administered with dronedarone.',
    TRUE, 0.30, 0.92,
    'FDA_LABEL', 'Savaysa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Dronedarone: P-gp inhibitor; Edoxaban AUC +85%', 'P-gp inhibition (primary mechanism)', 'CYP3A4 (minor)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP DOAC Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 13. Edoxaban + Cyclosporine (P-gp inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EDOXABAN_CYCLOSPORINE_001',
    'EDOXABAN', 'Edoxaban (Savaysa)',
    'CYCLOSPORINE', 'Cyclosporine',
    'pharmacokinetic',
    'major',
    'established',
    'Cyclosporine is a potent P-gp inhibitor. Edoxaban AUC increased approximately 70-85%.',
    'Significantly elevated bleeding risk in transplant patients.',
    'Reduce edoxaban dose to 30mg daily. Consider alternative anticoagulation strategy in transplant patients.',
    TRUE, 0.20, 0.90,
    'FDA_LABEL', 'Savaysa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Cyclosporine: potent P-gp inhibitor; Edoxaban AUC +73%', 'P-gp inhibition', 'Not primarily CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'KDIGO Transplant Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 14. Edoxaban + Erythromycin (P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EDOXABAN_ERYTHROMYCIN_001',
    'EDOXABAN', 'Edoxaban (Savaysa)',
    'ERYTHROMYCIN', 'Erythromycin',
    'pharmacokinetic',
    'moderate',
    'established',
    'Erythromycin is a moderate P-gp inhibitor. Edoxaban exposure modestly increased.',
    'Increased bleeding risk, though less severe than with strong P-gp inhibitors.',
    'No routine dose reduction but monitor for bleeding. Consider azithromycin as alternative.',
    FALSE, 0.25, 0.75,
    'FDA_LABEL', 'Savaysa - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Erythromycin: moderate P-gp inhibitor; Edoxaban exposure modestly increased', 'P-gp inhibition', 'CYP3A4 (minor)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACCP Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 15. Edoxaban + Rifampin (P-gp induction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EDOXABAN_RIFAMPIN_001',
    'EDOXABAN', 'Edoxaban (Savaysa)',
    'RIFAMPIN', 'Rifampin',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin induces P-gp. Edoxaban exposure decreased significantly.',
    'Reduced anticoagulation effect with thromboembolism risk.',
    'Avoid combination. Use warfarin or LMWH during TB treatment.',
    TRUE, 0.20, 0.95,
    'FDA_LABEL', 'Savaysa - Section 7 (P-gp inducers)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Rifampin: potent P-gp inducer; Edoxaban AUC significantly reduced', 'P-gp induction', 'Not CYP-mediated for edoxaban', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D-Avoid)', 'ATS TB Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- DABIGATRAN INTERACTIONS (P-gp substrate - renal elimination)
-- ============================================================================

-- 16. Dabigatran + Ketoconazole (P-gp inhibition - CONTRAINDICATED in renal impairment)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_KETOCONAZOLE_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'KETOCONAZOLE', 'Ketoconazole',
    'pharmacokinetic',
    'contraindicated',
    'established',
    'Ketoconazole is a potent P-gp inhibitor. Dabigatran is a P-gp substrate with no CYP metabolism. AUC increased 140-150%.',
    'Dramatically elevated dabigatran levels. Contraindicated in patients with CrCl <50 mL/min.',
    'Avoid in renal impairment (CrCl <50). In normal renal function, reduce dabigatran to 75mg BID if must use.',
    TRUE, 0.15, 0.98,
    'FDA_LABEL', 'Pradaxa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Ketoconazole: potent P-gp inhibitor; Dabigatran AUC +140-150%', 'P-gp inhibition (primary)', 'No CYP metabolism', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X) in renal impairment', 'ACCP DOAC Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 17. Dabigatran + Rifampin (P-gp induction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_RIFAMPIN_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'RIFAMPIN', 'Rifampin',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin is a potent P-gp inducer. Dabigatran AUC decreased by 66%.',
    'Markedly reduced anticoagulation. High thromboembolism risk.',
    'Avoid combination. Use warfarin or LMWH for anticoagulation during rifampin therapy.',
    TRUE, 0.20, 0.98,
    'FDA_LABEL', 'Pradaxa - Section 7 (P-gp inducers - Avoid)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Rifampin: potent P-gp inducer; Dabigatran AUC reduced 66%', 'P-gp induction', 'No CYP metabolism', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ATS TB Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 18. Dabigatran + Verapamil (P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_VERAPAMIL_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'VERAPAMIL', 'Verapamil',
    'pharmacokinetic',
    'moderate',
    'established',
    'Verapamil is a moderate P-gp inhibitor. Effect depends on timing: AUC +180% if taken together, less if separated.',
    'Significantly increased dabigatran exposure when co-administered. Increased bleeding risk.',
    'Reduce dabigatran dose (consider 110mg BID for AF). Administer dabigatran 2 hours before verapamil if possible.',
    TRUE, 0.40, 0.85,
    'FDA_LABEL', 'Pradaxa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Verapamil: moderate P-gp inhibitor; Timing-dependent: AUC +12% (2h apart) to +180% (together)', 'P-gp inhibition', 'No CYP metabolism', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ESC AF Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 19. Dabigatran + Amiodarone (P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_AMIODARONE_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'AMIODARONE', 'Amiodarone',
    'pharmacokinetic',
    'moderate',
    'established',
    'Amiodarone is a moderate P-gp inhibitor. Dabigatran AUC increased approximately 60%. Long amiodarone t1/2.',
    'Increased bleeding risk. Effect persists for weeks after amiodarone discontinuation.',
    'Consider dabigatran dose reduction in patients with additional bleeding risk factors. Monitor closely.',
    TRUE, 0.35, 0.80,
    'FDA_LABEL', 'Pradaxa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Amiodarone: moderate P-gp inhibitor; Dabigatran AUC +60%; Long t1/2 = prolonged effect', 'P-gp inhibition', 'No CYP metabolism for dabigatran', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA AF Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- DOAC + ANTIPLATELET/NSAID COMBINATIONS
-- ============================================================================

-- 20. Rivaroxaban + Aspirin (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_ASPIRIN_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive antithrombotic effect. No PK interaction but significantly increased bleeding risk.',
    'Major bleeding risk increased 2-3 fold compared to rivaroxaban alone. COMPASS trial data available.',
    'Use only when benefit outweighs risk (vascular disease). Use rivaroxaban 2.5mg BID with ASA 100mg (COMPASS regimen).',
    FALSE, 0.60, 0.92,
    'FDA_LABEL', 'Xarelto - Section 7 (Antiplatelets); COMPASS Trial', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic interaction; No PK interaction; Additive antithrombotic + bleeding', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC CVD Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 21. Apixaban + Clopidogrel (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_CLOPIDOGREL_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'CLOPIDOGREL', 'Clopidogrel',
    'pharmacodynamic',
    'major',
    'established',
    'Additive antithrombotic effect. No direct PK interaction but major increase in bleeding.',
    'Significantly increased bleeding risk. Common in post-PCI AF patients.',
    'Use shortest duration possible. Consider apixaban 2.5mg BID. AUGUSTUS trial supports apixaban in this setting.',
    FALSE, 0.45, 0.90,
    'FDA_LABEL', 'Eliquis - Section 7 (Antiplatelets); AUGUSTUS Trial', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic interaction; Dual antithrombotic therapy; AUGUSTUS trial data', NULL, 'Not CYP-mediated interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC AF Guidelines + Antithrombotic Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 22. Dabigatran + NSAIDs (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_NSAIDS_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'NSAIDS', 'NSAIDs (class)',
    'pharmacodynamic',
    'major',
    'established',
    'NSAIDs inhibit platelet function and cause GI mucosal damage, compounding DOAC bleeding risk.',
    'GI bleeding risk significantly increased. NSAIDs are common OTC medications often overlooked.',
    'Avoid chronic NSAID use. Use acetaminophen for pain. If NSAID necessary, use lowest dose + PPI protection.',
    FALSE, 0.70, 0.88,
    'FDA_LABEL', 'Pradaxa - Section 7 (NSAIDs); NSAID Black Box Warning', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: NSAID antiplatelet effect + GI toxicity; No PK interaction', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR NSAID Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 23. Edoxaban + Aspirin (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EDOXABAN_ASPIRIN_001',
    'EDOXABAN', 'Edoxaban (Savaysa)',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive antithrombotic effect. ENGAGE AF-TIMI 48 showed increased bleeding with concomitant aspirin.',
    'Major bleeding rate doubled with aspirin co-administration in clinical trials.',
    'Avoid unless clearly indicated. Reassess need for aspirin in AF patients on edoxaban.',
    FALSE, 0.55, 0.90,
    'FDA_LABEL', 'Savaysa - Section 7; ENGAGE AF-TIMI 48', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: additive bleeding risk; Trial data supports increased bleeding', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC AF Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 24. Apixaban + SSRIs (Bleeding - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_SSRI_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'SSRI', 'SSRIs (class)',
    'pharmacodynamic',
    'moderate',
    'probable',
    'SSRIs deplete platelet serotonin, impairing platelet aggregation. Additive bleeding effect with anticoagulants.',
    'Increased risk of GI and other bleeding. Common combination in elderly AF patients with depression.',
    'Monitor for bleeding signs. Consider gastroprotection. Do not necessarily avoid combination but counsel patient.',
    FALSE, 0.65, 0.75,
    'FDA_LABEL', 'Eliquis - Section 7 (Drugs affecting hemostasis)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'SSRIs inhibit platelet serotonin reuptake; Impaired platelet aggregation; Additive bleeding', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Depression Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 25. Rivaroxaban + Ibuprofen (Bleeding + CYP3A4 - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIVAROXABAN_IBUPROFEN_001',
    'RIVAROXABAN', 'Rivaroxaban (Xarelto)',
    'IBUPROFEN', 'Ibuprofen',
    'pharmacodynamic',
    'major',
    'established',
    'Ibuprofen inhibits platelet function (reversible COX-1) and causes GI mucosal injury. Dual bleeding mechanism.',
    'Significantly increased GI bleeding risk. Common scenario with OTC ibuprofen use.',
    'Avoid if possible. Use acetaminophen for pain. If NSAID required, shortest course with PPI protection.',
    FALSE, 0.75, 0.88,
    'FDA_LABEL', 'Xarelto - Section 7 (NSAIDs); NSAID Black Box', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: antiplatelet + GI toxicity; No significant PK interaction', NULL, 'Not primarily CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR NSAID Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Total active DDIs after migration 016: %', v_count;
END $$;
