-- ============================================================================
-- KB-5 Tier 2: Anticoagulant Panel - Migration 015
-- Warfarin Interaction Matrix (25 DDIs)
-- ============================================================================
-- Option C: Iterative Clinical Expansion
-- Three-Layer Governance: Regulatory + Pharmacology + Clinical
-- ============================================================================

-- 1. Warfarin + Fluconazole (CYP2C9 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_FLUCONAZOLE_001',
    'WARFARIN', 'Warfarin',
    'FLUCONAZOLE', 'Fluconazole',
    'pharmacokinetic',
    'major',
    'established',
    'Fluconazole potently inhibits CYP2C9, the primary metabolic pathway for S-warfarin (more potent enantiomer). INR increases 2-3 fold.',
    'Significantly elevated INR with high bleeding risk. Effect is dose-dependent (200mg fluconazole has greater effect than 100mg).',
    'Reduce warfarin dose by 25-50% when starting fluconazole. Check INR within 3-5 days. Consider alternative antifungal if possible.',
    TRUE, 0.65, 0.95,
    'FDA_LABEL', 'Coumadin - Section 7 (Drug Interactions); Diflucan - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Fluconazole: CYP2C9 inhibitor (Ki = 7 μM); S-warfarin AUC increases 2-3x', NULL, 'CYP2C9 (major), CYP3A4 (minor)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP Anticoagulation Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 2. Warfarin + Metronidazole (CYP2C9 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_METRONIDAZOLE_001',
    'WARFARIN', 'Warfarin',
    'METRONIDAZOLE', 'Metronidazole',
    'pharmacokinetic',
    'major',
    'established',
    'Metronidazole inhibits CYP2C9 metabolism of S-warfarin. INR typically increases within 3-7 days.',
    'Elevated INR with bleeding risk. Common combination in intra-abdominal infections where patients may also be on warfarin.',
    'Reduce warfarin dose by 25-35%. Monitor INR within 5-7 days of starting metronidazole. Resume normal dose after course.',
    TRUE, 0.55, 0.90,
    'FDA_LABEL', 'Coumadin - Section 7; Flagyl - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Metronidazole: CYP2C9 inhibitor; S-warfarin clearance reduced 30-40%', NULL, 'CYP2C9', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Intra-abdominal Infection Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 3. Warfarin + Sulfamethoxazole/TMP (CYP2C9 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_TMPSMX_001',
    'WARFARIN', 'Warfarin',
    'TMPSMX', 'Trimethoprim-Sulfamethoxazole (Bactrim)',
    'pharmacokinetic',
    'major',
    'established',
    'Sulfamethoxazole inhibits CYP2C9; trimethoprim may also contribute. One of the most significant warfarin drug interactions.',
    'INR can increase 50-100%. High risk of serious bleeding, especially in elderly patients. Hospitalizations reported.',
    'Avoid if alternative exists. If must use: reduce warfarin by 25-50%, check INR in 3-4 days, monitor closely throughout.',
    TRUE, 0.70, 0.98,
    'FDA_LABEL', 'Coumadin - Section 7 (explicitly lists TMP-SMX)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Sulfamethoxazole: potent CYP2C9 inhibitor; INR increases average 70%', NULL, 'CYP2C9', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ACCP Chest Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 4. Warfarin + Rifampin (CYP inducer - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_RIFAMPIN_001',
    'WARFARIN', 'Warfarin',
    'RIFAMPIN', 'Rifampin',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin is the most potent CYP inducer known. Induces CYP2C9, CYP3A4, CYP1A2. Warfarin metabolism increased 3-5 fold.',
    'Dramatically reduced anticoagulation effect. INR may drop to subtherapeutic within 5-7 days. Thromboembolism risk.',
    'Warfarin doses often need to be doubled or tripled. Monitor INR twice weekly during rifampin. Recheck after stopping (takes 2-3 weeks to normalize).',
    TRUE, 0.40, 0.99,
    'FDA_LABEL', 'Coumadin - Section 7; Rifadin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Rifampin: pan-CYP inducer; Warfarin clearance increased 300-500%', NULL, 'CYP2C9, CYP3A4, CYP1A2 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ATS/IDSA TB Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 5. Warfarin + Ciprofloxacin (CYP1A2 inhibition - MODERATE to MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_CIPROFLOXACIN_001',
    'WARFARIN', 'Warfarin',
    'CIPROFLOXACIN', 'Ciprofloxacin',
    'pharmacokinetic',
    'major',
    'established',
    'Ciprofloxacin inhibits CYP1A2 (R-warfarin metabolism). Also may affect gut flora vitamin K synthesis.',
    'INR elevation typically 20-50%. Risk increases with longer courses. Bleeding events reported.',
    'Monitor INR within 3-5 days of starting ciprofloxacin. Consider 10-20% warfarin dose reduction prophylactically.',
    TRUE, 0.60, 0.85,
    'FDA_LABEL', 'Coumadin - Section 7 (Fluoroquinolones); Cipro - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ciprofloxacin: CYP1A2 inhibitor; Variable effect on warfarin', NULL, 'CYP1A2', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA UTI Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 6. Warfarin + Amiodarone (CYP2C9/CYP3A4 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_AMIODARONE_001',
    'WARFARIN', 'Warfarin',
    'AMIODARONE', 'Amiodarone',
    'pharmacokinetic',
    'major',
    'established',
    'Amiodarone inhibits CYP2C9 and CYP3A4. Very long half-life (40-55 days) means effect persists weeks after discontinuation.',
    'INR increases 30-50%. Effect develops over 1-2 weeks and persists for months after amiodarone stopped.',
    'Reduce warfarin dose by 30-50% when starting amiodarone. Monitor INR weekly x 4 weeks, then every 2 weeks.',
    TRUE, 0.55, 0.95,
    'FDA_LABEL', 'Coumadin - Section 7; Cordarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Amiodarone: CYP2C9 + CYP3A4 inhibitor; Long t1/2 prolongs interaction', NULL, 'CYP2C9, CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA AF Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 7. Warfarin + Aspirin (Bleeding risk - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_ASPIRIN_001',
    'WARFARIN', 'Warfarin',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Dual mechanism: antiplatelet effect + GI mucosal damage. Aspirin irreversibly inhibits COX-1 in platelets.',
    'Significantly increased bleeding risk, especially GI bleeding. Risk increases with aspirin dose >100mg/day.',
    'Use only when benefit outweighs risk (mechanical valves, recent ACS). Use lowest effective aspirin dose (81mg). Add PPI for GI protection.',
    FALSE, 0.75, 0.92,
    'FDA_LABEL', 'Coumadin - Section 7 (Aspirin); Multiple FDA warnings', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Additive bleeding risk; No PK interaction; Pharmacodynamic synergy', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP Antithrombotic Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 8. Warfarin + Ibuprofen (Bleeding + CYP2C9 - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_IBUPROFEN_001',
    'WARFARIN', 'Warfarin',
    'IBUPROFEN', 'Ibuprofen',
    'pharmacodynamic',
    'major',
    'established',
    'NSAIDs inhibit platelet function (reversible COX-1), cause GI mucosal damage, and ibuprofen may inhibit CYP2C9.',
    'Increased GI bleeding risk 3-6 fold. Also risk of increased INR from CYP2C9 inhibition with higher ibuprofen doses.',
    'Avoid if possible. Use acetaminophen for pain. If NSAID necessary, use lowest dose for shortest duration with PPI.',
    FALSE, 0.80, 0.90,
    'FDA_LABEL', 'Coumadin - Section 7 (NSAIDs); NSAID Black Box Warning', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Dual mechanism: GI toxicity + antiplatelet effect; Variable CYP2C9 effect', NULL, 'CYP2C9 (minor)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR NSAID Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 9. Warfarin + Naproxen (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_NAPROXEN_001',
    'WARFARIN', 'Warfarin',
    'NAPROXEN', 'Naproxen',
    'pharmacodynamic',
    'major',
    'established',
    'Naproxen inhibits platelet COX-1 and causes GI mucosal injury. Long half-life (12-17h) increases cumulative bleeding risk.',
    'Significantly increased GI and other bleeding risk. Naproxen has longer duration of platelet inhibition than ibuprofen.',
    'Avoid combination. Use acetaminophen. If NSAID required, prefer short-acting with PPI. Monitor for bleeding signs.',
    FALSE, 0.70, 0.88,
    'FDA_LABEL', 'Coumadin - Section 7 (NSAIDs); Naprosyn - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Antiplatelet + GI toxicity; Long t1/2 increases risk duration', NULL, 'Not primarily CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR Pain Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 10. Warfarin + Celecoxib (Bleeding + CYP2C9 - MODERATE to MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_CELECOXIB_001',
    'WARFARIN', 'Warfarin',
    'CELECOXIB', 'Celecoxib',
    'pharmacokinetic',
    'major',
    'established',
    'Celecoxib is a CYP2C9 substrate and inhibitor. COX-2 selectivity reduces but does not eliminate GI bleeding risk.',
    'INR elevation from CYP2C9 inhibition. Less GI toxicity than non-selective NSAIDs but still increased bleeding risk.',
    'If NSAID needed in warfarin patient, celecoxib may be preferred. Still monitor INR; may need warfarin reduction.',
    TRUE, 0.45, 0.80,
    'FDA_LABEL', 'Coumadin - Section 7; Celebrex - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Celecoxib: CYP2C9 inhibitor; S-warfarin levels may increase 10-20%', NULL, 'CYP2C9', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACR NSAID Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 11. Warfarin + Clopidogrel (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_CLOPIDOGREL_001',
    'WARFARIN', 'Warfarin',
    'CLOPIDOGREL', 'Clopidogrel',
    'pharmacodynamic',
    'major',
    'established',
    'Triple antithrombotic effect when combined. Clopidogrel irreversibly inhibits P2Y12 receptor on platelets.',
    'Major bleeding risk increased 2-3 fold. Triple therapy (warfarin + aspirin + clopidogrel) has very high bleeding rate.',
    'Use only when strongly indicated (recent stent + AF). Shortest possible duration. Lower INR target (2.0-2.5). Add PPI.',
    FALSE, 0.50, 0.95,
    'FDA_LABEL', 'Coumadin - Section 7; Plavix - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Additive antithrombotic effect; No significant PK interaction', NULL, 'Not CYP-mediated interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC AF + ACS Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 12. Warfarin + Cranberry Juice (CYP2C9 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_CRANBERRY_001',
    'WARFARIN', 'Warfarin',
    'CRANBERRY', 'Cranberry Juice/Extract',
    'pharmacokinetic',
    'moderate',
    'probable',
    'Cranberry flavonoids may inhibit CYP2C9. Case reports of significant INR elevation, including fatalities.',
    'INR may increase with large or concentrated cranberry consumption. Effect is variable and unpredictable.',
    'Warn patients to avoid large quantities of cranberry juice. Extra INR monitoring if regular consumption. Moderation is key.',
    FALSE, 0.25, 0.70,
    'FDA_LABEL', 'CSM/MHRA UK Warning 2003 (cited in Coumadin literature)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Flavonoid CYP2C9 inhibition; Variable effect; Case reports of fatal bleeding', NULL, 'CYP2C9', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACCP Chest Guidelines 2023',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 13. Warfarin + Ginkgo biloba (Bleeding - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_GINKGO_001',
    'WARFARIN', 'Warfarin',
    'GINKGO', 'Ginkgo biloba',
    'pharmacodynamic',
    'moderate',
    'probable',
    'Ginkgo has antiplatelet activity (inhibits PAF). May also affect CYP enzymes. Spontaneous bleeding reported with ginkgo alone.',
    'Increased bleeding risk. Case reports of serious hemorrhage including intracranial bleeding.',
    'Advise patients to avoid ginkgo. If used, extra monitoring. Stop 2-3 weeks before planned surgery.',
    FALSE, 0.30, 0.75,
    'FDA_LABEL', 'Coumadin - Section 7 (Herbal Products)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'PAF inhibition (antiplatelet); Variable CYP effects; Supplement quality varies', NULL, 'Possible CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'Natural Medicines Database',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 14. Warfarin + St. John's Wort (CYP inducer - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_STJOHNSWORT_001',
    'WARFARIN', 'Warfarin',
    'STJOHNSWORT', 'St. Johns Wort',
    'pharmacokinetic',
    'major',
    'established',
    'St. Johns Wort induces CYP3A4, CYP2C9, and P-glycoprotein. Significant reduction in warfarin levels.',
    'INR decreases significantly. Thromboembolism risk. Effect develops over 1-2 weeks of use.',
    'Contraindicated combination. Advise patients to avoid. If stopped, monitor for INR increase over 2-3 weeks.',
    TRUE, 0.20, 0.95,
    'FDA_LABEL', 'Coumadin - Section 7; Multiple FDA alerts on St. Johns Wort', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Hyperforin: potent CYP3A4/CYP2C9 inducer via PXR; Warfarin AUC reduced 25-50%', 'P-gp inducer', 'CYP3A4, CYP2C9 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ACCP Anticoagulation Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 15. Warfarin + Miconazole (topical) (CYP2C9 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_MICONAZOLE_001',
    'WARFARIN', 'Warfarin',
    'MICONAZOLE', 'Miconazole (topical/oral gel)',
    'pharmacokinetic',
    'moderate',
    'established',
    'Even topical/oral gel miconazole can inhibit CYP2C9 through systemic absorption. Often underrecognized interaction.',
    'INR elevation reported even with topical use. Effect usually modest but can be clinically significant.',
    'Use clotrimazole troches as alternative for oral candidiasis. If miconazole used, monitor INR more frequently.',
    TRUE, 0.35, 0.75,
    'FDA_LABEL', 'FDA Warning on oral miconazole gel + warfarin', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Miconazole: CYP2C9 inhibitor; Systemic absorption from mucosa sufficient for interaction', NULL, 'CYP2C9', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Candidiasis Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 16. Warfarin + Levofloxacin (CYP1A2 - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_LEVOFLOXACIN_001',
    'WARFARIN', 'Warfarin',
    'LEVOFLOXACIN', 'Levofloxacin',
    'pharmacokinetic',
    'moderate',
    'established',
    'Levofloxacin has less CYP1A2 inhibition than ciprofloxacin but still may affect warfarin. Gut flora disruption contributes.',
    'Variable INR changes, usually modest elevation. Less predictable than ciprofloxacin interaction.',
    'Monitor INR during levofloxacin course. Less likely to require dose adjustment than ciprofloxacin.',
    FALSE, 0.50, 0.70,
    'FDA_LABEL', 'Coumadin - Section 7; Levaquin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Levofloxacin: weak CYP1A2 inhibitor; Also affects gut vitamin K production', NULL, 'CYP1A2 (weak)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA CAP Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 17. Warfarin + Clarithromycin (CYP3A4 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_CLARITHROMYCIN_001',
    'WARFARIN', 'Warfarin',
    'CLARITHROMYCIN', 'Clarithromycin',
    'pharmacokinetic',
    'major',
    'established',
    'Clarithromycin is a potent CYP3A4 inhibitor affecting R-warfarin. Also reduces gut flora vitamin K synthesis.',
    'INR increases typically 20-40%. Onset within 3-5 days. Bleeding events reported.',
    'Use azithromycin as alternative (less interaction). If clarithromycin necessary, reduce warfarin 15-25% and monitor INR.',
    TRUE, 0.45, 0.88,
    'FDA_LABEL', 'Coumadin - Section 7 (Macrolides); Biaxin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Clarithromycin: potent CYP3A4 inhibitor; R-warfarin AUC increased', NULL, 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Respiratory Infection Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 18. Warfarin + Erythromycin (CYP3A4 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_ERYTHROMYCIN_001',
    'WARFARIN', 'Warfarin',
    'ERYTHROMYCIN', 'Erythromycin',
    'pharmacokinetic',
    'moderate',
    'established',
    'Erythromycin inhibits CYP3A4. Less potent than clarithromycin but still causes INR elevation.',
    'INR typically increases 10-30%. Variable depending on erythromycin dose and duration.',
    'Monitor INR within 5 days. May need warfarin dose reduction 10-15%. Use azithromycin as alternative.',
    TRUE, 0.30, 0.78,
    'FDA_LABEL', 'Coumadin - Section 7 (Macrolides)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Erythromycin: moderate CYP3A4 inhibitor', NULL, 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACCP Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 19. Warfarin + Voriconazole (CYP2C9/CYP3A4 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_VORICONAZOLE_001',
    'WARFARIN', 'Warfarin',
    'VORICONAZOLE', 'Voriconazole',
    'pharmacokinetic',
    'major',
    'established',
    'Voriconazole is a potent inhibitor of CYP2C9, CYP2C19, and CYP3A4. Affects both R- and S-warfarin.',
    'INR can increase 2-3 fold. More pronounced effect than fluconazole. High bleeding risk.',
    'Reduce warfarin dose by 50% when starting voriconazole. Check INR within 3 days. Monitor twice weekly initially.',
    TRUE, 0.35, 0.96,
    'FDA_LABEL', 'Coumadin - Section 7; Vfend - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Voriconazole: CYP2C9 + CYP3A4 + CYP2C19 inhibitor; Warfarin AUC increases 200%+', NULL, 'CYP2C9, CYP3A4, CYP2C19', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Aspergillosis Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 20. Warfarin + Omeprazole (CYP2C19 - MINOR to MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_OMEPRAZOLE_001',
    'WARFARIN', 'Warfarin',
    'OMEPRAZOLE', 'Omeprazole',
    'pharmacokinetic',
    'moderate',
    'probable',
    'Omeprazole inhibits CYP2C19, which metabolizes R-warfarin. Effect is usually modest but can be clinically relevant.',
    'INR may increase slightly (5-15%). Clinical significance is debated but monitoring warranted.',
    'Monitor INR when starting or stopping PPI. Pantoprazole may have less interaction potential.',
    FALSE, 0.85, 0.60,
    'FDA_LABEL', 'Coumadin - Section 7 (PPIs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Omeprazole: CYP2C19 inhibitor; R-warfarin clearance slightly reduced', NULL, 'CYP2C19', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACG PPI Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 21. Warfarin + Phenytoin (Complex interaction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_PHENYTOIN_002',
    'WARFARIN', 'Warfarin',
    'PHENYTOIN', 'Phenytoin',
    'pharmacokinetic',
    'major',
    'established',
    'Complex bidirectional interaction: Phenytoin induces warfarin metabolism (CYP2C9/3A4). Warfarin inhibits phenytoin metabolism. Net effect varies.',
    'Usually reduced warfarin effect (induction predominates). But initial phase may show increased INR. Phenytoin levels may rise.',
    'Monitor both INR and phenytoin levels closely. Adjust warfarin based on INR. May need 2-3x warfarin dose long-term.',
    TRUE, 0.35, 0.90,
    'FDA_LABEL', 'Coumadin - Section 7; Dilantin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Phenytoin: CYP2C9/3A4 inducer; Warfarin: CYP2C9 inhibitor; Bidirectional interaction', NULL, 'CYP2C9, CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'AES Epilepsy Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 22. Warfarin + Carbamazepine (CYP inducer - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_CARBAMAZEPINE_002',
    'WARFARIN', 'Warfarin',
    'CARBAMAZEPINE', 'Carbamazepine',
    'pharmacokinetic',
    'major',
    'established',
    'Carbamazepine is a potent CYP3A4 and moderate CYP2C9 inducer. Significantly increases warfarin metabolism.',
    'Reduced anticoagulation effect. INR may drop 30-50%. Thromboembolism risk if not dose-adjusted.',
    'Increase warfarin dose as needed (often 50-100% increase). Monitor INR weekly during initiation. Recheck after CBZ stopped.',
    TRUE, 0.30, 0.92,
    'FDA_LABEL', 'Coumadin - Section 7; Tegretol - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Carbamazepine: CYP3A4 inducer + moderate CYP2C9 inducer; Warfarin clearance increased', NULL, 'CYP3A4, CYP2C9 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'AES Epilepsy Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 23. Warfarin + Bosentan (CYP inducer - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_BOSENTAN_001',
    'WARFARIN', 'Warfarin',
    'BOSENTAN', 'Bosentan',
    'pharmacokinetic',
    'major',
    'established',
    'Bosentan induces CYP3A4 and CYP2C9. Reduces warfarin levels by 30-40%. Common in pulmonary hypertension patients.',
    'Reduced anticoagulation. INR decreases over 1-2 weeks. May need significant warfarin dose increase.',
    'Monitor INR weekly x 4 weeks when starting bosentan. Increase warfarin dose as needed. Riociguat may be alternative.',
    TRUE, 0.25, 0.88,
    'FDA_LABEL', 'Coumadin - Section 7; Tracleer - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Bosentan: CYP3A4 + CYP2C9 inducer; Warfarin AUC reduced 30-40%', NULL, 'CYP3A4, CYP2C9 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ESC/ERS PAH Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 24. Warfarin + Tamoxifen (CYP2C9 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_TAMOXIFEN_001',
    'WARFARIN', 'Warfarin',
    'TAMOXIFEN', 'Tamoxifen',
    'pharmacokinetic',
    'major',
    'established',
    'Tamoxifen inhibits CYP2C9. Also has weak estrogenic effects that may affect clotting factors.',
    'Significant INR elevation. Cases of serious bleeding reported. May also affect tamoxifen efficacy (CYP2D6).',
    'Monitor INR closely. Consider LMWH or DOAC as alternative anticoagulation in breast cancer patients.',
    TRUE, 0.30, 0.90,
    'FDA_LABEL', 'Coumadin - Section 7; Nolvadex - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Tamoxifen: CYP2C9 inhibitor; S-warfarin levels significantly increased', NULL, 'CYP2C9', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASCO Breast Cancer Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 25. Warfarin + Levothyroxine (Pharmacodynamic - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_LEVOTHYROXINE_001',
    'WARFARIN', 'Warfarin',
    'LEVOTHYROXINE', 'Levothyroxine',
    'pharmacodynamic',
    'moderate',
    'established',
    'Thyroid hormones increase catabolism of vitamin K-dependent clotting factors, enhancing warfarin sensitivity.',
    'Hyperthyroidism increases warfarin effect; hypothyroidism decreases it. Dose changes in either direction affect INR.',
    'Monitor INR when thyroid status changes (starting, stopping, or adjusting thyroid hormone). Adjust warfarin accordingly.',
    TRUE, 0.60, 0.75,
    'FDA_LABEL', 'Coumadin - Section 7 (Thyroid drugs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: Thyroid hormone increases clotting factor catabolism; Not a CYP interaction', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ATA Thyroid Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Total active DDIs after migration 015: %', v_count;
END $$;
