-- ============================================================================
-- KB-5 Tier 2: Anticoagulant Panel - Migration 017
-- Heparin, LMWH, and Antiplatelet Combinations (25 DDIs)
-- ============================================================================
-- Option C: Iterative Clinical Expansion
-- Three-Layer Governance: Regulatory + Pharmacology + Clinical
-- ============================================================================

-- ============================================================================
-- HEPARIN INTERACTIONS
-- ============================================================================

-- 1. Heparin + Aspirin (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_ASPIRIN_001',
    'HEPARIN', 'Heparin (UFH)',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive antithrombotic effect: heparin activates antithrombin, aspirin inhibits platelet COX-1.',
    'Significantly increased bleeding risk. Common combination in ACS but requires careful monitoring.',
    'Use in ACS per protocol. Monitor closely for bleeding. Ensure appropriate heparin dosing with aPTT.',
    FALSE, 0.70, 0.90,
    'FDA_LABEL', 'Heparin - Section 7; ACS Guidelines', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: dual antithrombotic mechanism; No PK interaction', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA ACS Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 2. Heparin + Clopidogrel (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_CLOPIDOGREL_001',
    'HEPARIN', 'Heparin (UFH)',
    'CLOPIDOGREL', 'Clopidogrel',
    'pharmacodynamic',
    'major',
    'established',
    'Clopidogrel irreversibly inhibits P2Y12 receptor. Combined with heparin anticoagulation = major bleeding risk.',
    'High bleeding risk, especially at vascular access sites. Standard in PCI but requires vigilance.',
    'Use per ACS/PCI protocols. Apply meticulous hemostasis at access sites. Monitor for bleeding complications.',
    FALSE, 0.55, 0.88,
    'FDA_LABEL', 'Heparin - Section 7; Plavix - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: dual antithrombotic mechanism', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'SCAI PCI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 3. Heparin + Ketorolac (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_KETOROLAC_001',
    'HEPARIN', 'Heparin (UFH)',
    'KETOROLAC', 'Ketorolac',
    'pharmacodynamic',
    'major',
    'established',
    'Ketorolac has potent antiplatelet effect + GI toxicity. One of the highest bleeding risk NSAIDs.',
    'Very high bleeding risk. Ketorolac use while anticoagulated often results in hemorrhagic complications.',
    'Avoid combination. Ketorolac is contraindicated with therapeutic anticoagulation. Use alternative analgesia.',
    FALSE, 0.35, 0.95,
    'FDA_LABEL', 'Heparin - Section 7; Toradol Black Box Warning (bleeding risk)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Ketorolac: potent COX inhibitor with high GI bleeding risk', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Toradol Prescribing Information',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 4. Heparin + Dextran (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_DEXTRAN_001',
    'HEPARIN', 'Heparin (UFH)',
    'DEXTRAN', 'Dextran',
    'pharmacodynamic',
    'major',
    'established',
    'Dextran impairs platelet function and reduces Factor VIII activity. Combined anticoagulant effect.',
    'High bleeding risk, especially surgical patients receiving dextran for volume expansion.',
    'Avoid dextran in heparinized patients. Use crystalloid or albumin for volume replacement.',
    FALSE, 0.15, 0.85,
    'FDA_LABEL', 'Heparin - Section 7; Dextran - Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Dextran: antiplatelet effect + hemodilution; Additive with heparin', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'Surgical Hemostasis Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 5. Heparin + Glycoprotein IIb/IIIa Inhibitors (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_GPIIBIIIA_001',
    'HEPARIN', 'Heparin (UFH)',
    'ABCIXIMAB', 'GP IIb/IIIa Inhibitors (abciximab, eptifibatide, tirofiban)',
    'pharmacodynamic',
    'major',
    'established',
    'GP IIb/IIIa inhibitors block final common pathway of platelet aggregation. Additive with heparin anticoagulation.',
    'Very high bleeding risk. Standard of care in high-risk PCI but requires dose adjustment of heparin.',
    'Reduce heparin target ACT to 200-250 sec when using GP IIb/IIIa inhibitors. Monitor closely at access sites.',
    TRUE, 0.30, 0.95,
    'FDA_LABEL', 'Heparin - Section 7; ReoPro/Integrilin/Aggrastat - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Complete platelet inhibition + anticoagulation = high bleeding risk', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'SCAI High-Risk PCI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- ENOXAPARIN (LMWH) INTERACTIONS
-- ============================================================================

-- 6. Enoxaparin + Aspirin (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ENOXAPARIN_ASPIRIN_001',
    'ENOXAPARIN', 'Enoxaparin (Lovenox)',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive antithrombotic effect. Standard in ACS but increases bleeding risk.',
    'Increased bleeding risk, especially at injection sites and GI tract.',
    'Use per ACS guidelines. Low-dose aspirin (81-100mg). Monitor for bleeding. Consider PPI for GI protection.',
    FALSE, 0.75, 0.88,
    'FDA_LABEL', 'Lovenox - Section 7 (Antiplatelets)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: dual antithrombotic mechanism', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA ACS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 7. Enoxaparin + Clopidogrel (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ENOXAPARIN_CLOPIDOGREL_001',
    'ENOXAPARIN', 'Enoxaparin (Lovenox)',
    'CLOPIDOGREL', 'Clopidogrel',
    'pharmacodynamic',
    'major',
    'established',
    'Triple antithrombotic therapy when aspirin also used. High bleeding risk.',
    'Significantly increased bleeding risk. Common scenario in NSTEMI management.',
    'Standard of care in ACS. Assess bleeding risk vs ischemic benefit. Shortest duration possible.',
    FALSE, 0.60, 0.90,
    'FDA_LABEL', 'Lovenox - Section 7; Plavix - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: dual/triple antithrombotic therapy', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA NSTEMI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 8. Enoxaparin + NSAIDs (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ENOXAPARIN_NSAIDS_001',
    'ENOXAPARIN', 'Enoxaparin (Lovenox)',
    'NSAIDS', 'NSAIDs (class)',
    'pharmacodynamic',
    'major',
    'established',
    'NSAIDs inhibit platelet function and cause GI mucosal injury. Additive bleeding risk.',
    'Significantly increased GI bleeding risk. Spinal/epidural hematoma risk if neuraxial anesthesia.',
    'Avoid NSAIDs if possible. Use acetaminophen. If NSAID needed, add PPI protection.',
    FALSE, 0.65, 0.88,
    'FDA_LABEL', 'Lovenox - Section 7 (NSAIDs); Black Box re: spinal hematoma', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: antiplatelet + GI toxicity; No PK interaction', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR NSAID Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 9. Enoxaparin + Spinal/Epidural Anesthesia (Spinal Hematoma - BLACK BOX)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ENOXAPARIN_NEURAXIAL_001',
    'ENOXAPARIN', 'Enoxaparin (Lovenox)',
    'NEURAXIAL', 'Spinal/Epidural Anesthesia',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Risk of spinal/epidural hematoma with neuraxial anesthesia in anticoagulated patients. Can cause permanent paralysis.',
    'Spinal/epidural hematoma risk → permanent neurological injury including paralysis.',
    'Wait 12 hours after prophylactic dose, 24 hours after therapeutic dose before neuraxial. Remove catheter before restart.',
    TRUE, 0.40, 0.99,
    'FDA_LABEL', 'Lovenox - BLACK BOX WARNING (Spinal/Epidural Hematomas)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Anticoagulation in confined spinal space = hematoma risk with compression', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ASRA Regional Anesthesia Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- ============================================================================
-- ANTIPLATELET INTERACTIONS (CLOPIDOGREL, PRASUGREL, TICAGRELOR)
-- ============================================================================

-- 10. Clopidogrel + Omeprazole (CYP2C19 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CLOPIDOGREL_OMEPRAZOLE_001',
    'CLOPIDOGREL', 'Clopidogrel (Plavix)',
    'OMEPRAZOLE', 'Omeprazole',
    'pharmacokinetic',
    'major',
    'established',
    'Omeprazole inhibits CYP2C19, which activates clopidogrel prodrug. Active metabolite reduced 45%.',
    'Reduced antiplatelet effect. Potential for increased cardiovascular events. FDA warning issued.',
    'Use pantoprazole (least CYP2C19 inhibition) instead. If omeprazole necessary, separate by 12 hours.',
    TRUE, 0.80, 0.85,
    'FDA_LABEL', 'Plavix - Section 7 (FDA Box Warning on CYP2C19 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Omeprazole: CYP2C19 inhibitor; Clopidogrel active metabolite reduced 45%', NULL, 'CYP2C19', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ACC/AHA Antiplatelet Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 11. Clopidogrel + Esomeprazole (CYP2C19 inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CLOPIDOGREL_ESOMEPRAZOLE_001',
    'CLOPIDOGREL', 'Clopidogrel (Plavix)',
    'ESOMEPRAZOLE', 'Esomeprazole (Nexium)',
    'pharmacokinetic',
    'major',
    'established',
    'Esomeprazole (S-isomer of omeprazole) also inhibits CYP2C19. Similar concern to omeprazole.',
    'Reduced clopidogrel activation and antiplatelet effect.',
    'Use pantoprazole instead. Avoid esomeprazole in patients requiring clopidogrel.',
    TRUE, 0.70, 0.85,
    'FDA_LABEL', 'Plavix - Section 7 (CYP2C19 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Esomeprazole: CYP2C19 inhibitor; Similar mechanism to omeprazole', NULL, 'CYP2C19', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ACC/AHA Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 12. Clopidogrel + Fluconazole (CYP2C19 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CLOPIDOGREL_FLUCONAZOLE_001',
    'CLOPIDOGREL', 'Clopidogrel (Plavix)',
    'FLUCONAZOLE', 'Fluconazole',
    'pharmacokinetic',
    'moderate',
    'established',
    'Fluconazole inhibits CYP2C19 (and CYP2C9). May reduce clopidogrel activation.',
    'Reduced antiplatelet effect. Less studied than PPI interaction but mechanistically similar.',
    'Short courses acceptable with monitoring. Consider alternative antifungal for prolonged treatment.',
    TRUE, 0.35, 0.75,
    'FDA_LABEL', 'Plavix - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Fluconazole: CYP2C19 inhibitor; May reduce clopidogrel active metabolite', NULL, 'CYP2C19', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Candidiasis Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 13. Clopidogrel + Aspirin (DAPT - Bleeding - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CLOPIDOGREL_ASPIRIN_001',
    'CLOPIDOGREL', 'Clopidogrel (Plavix)',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'moderate',
    'established',
    'DAPT (Dual Antiplatelet Therapy): synergistic antiplatelet effect via two different mechanisms.',
    'Increased bleeding risk but superior efficacy in preventing stent thrombosis. Cornerstone of post-PCI care.',
    'Standard of care post-PCI. Duration based on stent type and bleeding risk. Consider PPI for GI protection.',
    FALSE, 0.90, 0.85,
    'FDA_LABEL', 'Plavix - Section 7; CURE/CREDO trials', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'DAPT: synergistic antiplatelet effect; COX-1 + P2Y12 inhibition', NULL, 'Not CYP-mediated interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C) - Expected', 'ACC/AHA DAPT Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 14. Prasugrel + NSAIDs (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PRASUGREL_NSAIDS_001',
    'PRASUGREL', 'Prasugrel (Effient)',
    'NSAIDS', 'NSAIDs (class)',
    'pharmacodynamic',
    'major',
    'established',
    'Prasugrel is a more potent P2Y12 inhibitor than clopidogrel. NSAIDs add antiplatelet and GI bleeding risk.',
    'Very high GI bleeding risk. Prasugrel has higher bleeding rates than clopidogrel.',
    'Avoid NSAIDs. Use acetaminophen for pain. If NSAID unavoidable, use lowest dose + PPI.',
    FALSE, 0.60, 0.92,
    'FDA_LABEL', 'Effient - Section 7 (NSAIDs); Black Box Warning (bleeding)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: potent antiplatelet + NSAID bleeding risk', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'TRITON-TIMI 38 Trial Data',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 15. Prasugrel + Warfarin (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PRASUGREL_WARFARIN_001',
    'PRASUGREL', 'Prasugrel (Effient)',
    'WARFARIN', 'Warfarin',
    'pharmacodynamic',
    'major',
    'established',
    'Triple therapy (anticoagulant + DAPT) has very high bleeding risk. Prasugrel more potent than clopidogrel.',
    'Very high major bleeding rate. Generally prasugrel is avoided in triple therapy scenarios.',
    'Use clopidogrel instead of prasugrel in triple therapy. Shortest possible duration. Target INR 2.0-2.5.',
    FALSE, 0.30, 0.95,
    'FDA_LABEL', 'Effient - Section 7; Black Box Warning', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: potent antiplatelet + anticoagulant = very high bleeding', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D-Avoid if possible)', 'ESC AF + PCI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 16. Ticagrelor + Digoxin (P-gp inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TICAGRELOR_DIGOXIN_001',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'DIGOXIN', 'Digoxin',
    'pharmacokinetic',
    'moderate',
    'established',
    'Ticagrelor is a P-gp inhibitor. Digoxin is a P-gp substrate. Digoxin levels increase approximately 28%.',
    'Potential for digoxin toxicity. May be clinically significant in patients with renal impairment.',
    'Monitor digoxin levels when starting ticagrelor. May need digoxin dose reduction.',
    TRUE, 0.45, 0.75,
    'FDA_LABEL', 'Brilinta - Section 7 (P-gp substrates)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Ticagrelor: P-gp inhibitor; Digoxin AUC +28%', 'P-gp inhibition', 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA AF Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 17. Ticagrelor + Simvastatin (CYP3A4 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TICAGRELOR_SIMVASTATIN_001',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'SIMVASTATIN', 'Simvastatin',
    'pharmacokinetic',
    'moderate',
    'established',
    'Ticagrelor is a weak CYP3A4 inhibitor. Simvastatin is a CYP3A4 substrate. Simvastatin levels increase.',
    'Increased risk of statin-related myopathy and rhabdomyolysis.',
    'Limit simvastatin to 40mg/day with ticagrelor. Alternatively, use atorvastatin (less CYP3A4 dependent).',
    TRUE, 0.70, 0.80,
    'FDA_LABEL', 'Brilinta - Section 7 (CYP3A4 substrates); Zocor - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ticagrelor: weak CYP3A4 inhibitor; Simvastatin highly CYP3A4-dependent', NULL, 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA Statin Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 18. Ticagrelor + Strong CYP3A4 Inhibitors (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TICAGRELOR_CYP3A4STRONG_001',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'CYP3A4_STRONG_INHIBITORS', 'Strong CYP3A4 Inhibitors (itraconazole, clarithromycin)',
    'pharmacokinetic',
    'major',
    'established',
    'Ticagrelor is a CYP3A4 substrate. Strong inhibitors significantly increase ticagrelor levels.',
    'Excessive bleeding risk due to elevated ticagrelor exposure.',
    'Avoid strong CYP3A4 inhibitors with ticagrelor. Use alternative antiplatelet or antimicrobial.',
    TRUE, 0.25, 0.92,
    'FDA_LABEL', 'Brilinta - Section 7 (Strong CYP3A4 inhibitors - Avoid)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ticagrelor: CYP3A4 substrate; AUC significantly increased by strong inhibitors', NULL, 'CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D-Avoid)', 'ACCP Antiplatelet Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 19. Ticagrelor + Rifampin (CYP3A4 induction - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TICAGRELOR_RIFAMPIN_001',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'RIFAMPIN', 'Rifampin',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin is a potent CYP3A4 inducer. Ticagrelor AUC decreased by 86%.',
    'Markedly reduced antiplatelet effect. High risk of stent thrombosis and cardiovascular events.',
    'Avoid combination. Use prasugrel (not CYP3A4-dependent) or consider alternative TB regimen.',
    TRUE, 0.15, 0.98,
    'FDA_LABEL', 'Brilinta - Section 7 (Strong CYP3A4 inducers - Avoid)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Rifampin: potent CYP3A4 inducer; Ticagrelor AUC reduced 86%', NULL, 'CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X-Avoid)', 'ATS TB Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 20. Cilostazol + Omeprazole (CYP2C19 inhibition - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CILOSTAZOL_OMEPRAZOLE_001',
    'CILOSTAZOL', 'Cilostazol (Pletal)',
    'OMEPRAZOLE', 'Omeprazole',
    'pharmacokinetic',
    'moderate',
    'established',
    'Omeprazole inhibits CYP2C19, one of the metabolic pathways for cilostazol. Cilostazol levels increase.',
    'Potential for increased cilostazol effects: headache, palpitations, bleeding.',
    'Consider reducing cilostazol to 50mg BID when used with omeprazole. Use pantoprazole as alternative.',
    TRUE, 0.40, 0.75,
    'FDA_LABEL', 'Pletal - Section 7 (CYP2C19 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Cilostazol: CYP3A4 + CYP2C19 substrate; Omeprazole inhibits CYP2C19', NULL, 'CYP2C19', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC PAD Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 21. Dipyridamole + Adenosine (Enhanced Effect - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DIPYRIDAMOLE_ADENOSINE_002',
    'DIPYRIDAMOLE', 'Dipyridamole',
    'ADENOSINE', 'Adenosine',
    'pharmacokinetic',
    'major',
    'established',
    'Dipyridamole inhibits adenosine uptake and adenosine deaminase, dramatically prolonging adenosine effect.',
    'Profound and prolonged AV block, bronchospasm, hypotension. Effect is 4-5x longer.',
    'Reduce adenosine dose by 75% if must use. Alternatively, avoid adenosine - use regadenoson for cardiac stress testing.',
    TRUE, 0.30, 0.95,
    'FDA_LABEL', 'Adenocard - Section 7 (Dipyridamole)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Dipyridamole: adenosine uptake inhibitor; Adenosine effect prolonged 4-5x', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASNC Cardiac Imaging Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 22. Vorapaxar + Anticoagulants (Bleeding - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'VORAPAXAR_ANTICOAG_001',
    'VORAPAXAR', 'Vorapaxar (Zontivity)',
    'ANTICOAGULANTS', 'Anticoagulants (class)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Vorapaxar is a PAR-1 antagonist (third antiplatelet mechanism). Triple antithrombotic = extreme bleeding.',
    'Very high bleeding risk including intracranial hemorrhage. Not studied with anticoagulants.',
    'Do not use vorapaxar with any anticoagulant. It is contraindicated per labeling.',
    FALSE, 0.10, 0.99,
    'FDA_LABEL', 'Zontivity - Section 4 (Contraindication: Active bleeding or stroke)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Triple antithrombotic mechanism: PAR-1 + P2Y12 + COX-1 + anticoagulant = extreme risk', NULL, 'Not CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'TRA 2P-TIMI 50 Trial',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 23. Cangrelor + Clopidogrel (Timing-dependent - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CANGRELOR_CLOPIDOGREL_001',
    'CANGRELOR', 'Cangrelor (Kengreal)',
    'CLOPIDOGREL', 'Clopidogrel',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs bind P2Y12 receptor. Cangrelor blocks clopidogrel binding if given simultaneously.',
    'If clopidogrel given during cangrelor infusion, clopidogrel is rendered ineffective.',
    'Give clopidogrel loading dose AFTER cangrelor infusion ends (or immediately before starting cangrelor).',
    TRUE, 0.25, 0.90,
    'FDA_LABEL', 'Kengreal - Section 7 (Thienopyridines)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Competitive P2Y12 binding; Cangrelor prevents clopidogrel from binding if co-administered', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'CHAMPION-PHOENIX Trial',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 24. Aspirin + ACE Inhibitors (Reduced efficacy - MODERATE)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ASPIRIN_ACEI_001',
    'ASPIRIN', 'Aspirin',
    'ACEI', 'ACE Inhibitors (class)',
    'pharmacodynamic',
    'moderate',
    'probable',
    'Aspirin inhibits prostaglandin synthesis, which may attenuate vasodilatory and cardioprotective effects of ACE inhibitors.',
    'Potentially reduced cardiovascular benefit of ACE inhibitors. Clinical significance debated.',
    'Use low-dose aspirin (81mg). Benefits of combination generally outweigh risks in appropriate patients.',
    FALSE, 0.85, 0.70,
    'FDA_LABEL', 'Multiple ACE inhibitor labels note interaction', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Prostaglandin-mediated interaction; Aspirin blocks ACEi-induced PG vasodilation', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ACC/AHA Heart Failure Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 25. Fondaparinux + Antiplatelets (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'FONDAPARINUX_ANTIPLATELET_001',
    'FONDAPARINUX', 'Fondaparinux (Arixtra)',
    'ANTIPLATELETS', 'Antiplatelets (class)',
    'pharmacodynamic',
    'major',
    'established',
    'Fondaparinux is a selective Factor Xa inhibitor. Combined with antiplatelets = additive bleeding risk.',
    'Increased bleeding risk. Standard combination in NSTEMI per OASIS-5 trial.',
    'Use per ACS guidelines. Monitor closely for bleeding. Fondaparinux has lower bleeding than enoxaparin.',
    FALSE, 0.55, 0.88,
    'FDA_LABEL', 'Arixtra - Section 7 (Antiplatelets)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic: Factor Xa inhibition + antiplatelet = additive antithrombotic', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'OASIS-5 Trial; ACC/AHA NSTEMI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Total active DDIs after migration 017: %', v_count;
END $$;
