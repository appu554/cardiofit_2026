-- =============================================================================
-- KB-5 Phase 6: Comprehensive DDI Seed Data with Three-Layer Authority Sources
-- Migration: 008_comprehensive_ddi_seed.sql
-- Purpose: Add clinically important DDIs with full governance attribution
--
-- Categories:
--   1. Anticoagulant Interactions (Warfarin, DOACs)
--   2. QT Prolongation Combinations (CredibleMeds)
--   3. Serotonin Syndrome Risk
--   4. CYP450 Interactions (Major inhibitors/inducers)
--   5. Nephrotoxicity (Triple Whammy)
--   6. Opioid Safety (FDA Black Box)
--   7. Antiplatelet Interactions
--   8. Diabetes Medication Safety
-- =============================================================================

-- =============================================================================
-- CATEGORY 1: ANTICOAGULANT INTERACTIONS
-- =============================================================================

-- Warfarin + Fluconazole (CYP2C9 inhibition - MAJOR)
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
    'Fluconazole potently inhibits CYP2C9, the primary enzyme metabolizing S-warfarin (the more potent enantiomer). INR increases typically occur within 2-3 days.',
    'Significantly increased INR with high bleeding risk. INR may increase 2-4 fold. Risk of intracranial hemorrhage, GI bleeding.',
    'Reduce warfarin dose by 25-50% when starting fluconazole. Monitor INR within 3-5 days of initiation. Consider alternative antifungal (micafungin) if possible.',
    TRUE,
    0.70,
    0.92,
    'FDA_LABEL', 'Warfarin Sodium Tablets - Section 7; Fluconazole Tablets - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Fluconazole inhibits CYP2C9 (Ki = 7.1 μM); S-warfarin AUC increases 2.6-fold', NULL, 'CYP2C9 (major), CYP3A4 (minor)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (D)', 'ACCP Antithrombotic Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Warfarin + Rifampin (CYP inducer - MAJOR)
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
    'Rifampin is a potent inducer of CYP2C9, CYP3A4, and CYP1A2, dramatically increasing warfarin metabolism. Effect persists 2-3 weeks after rifampin discontinuation.',
    'Subtherapeutic INR with risk of thromboembolism. Warfarin requirements may increase 2-5 fold during rifampin therapy.',
    'Avoid combination if possible. If essential, increase warfarin dose significantly (may need 2-5x usual dose). Monitor INR twice weekly. When stopping rifampin, taper warfarin slowly over 2-3 weeks.',
    TRUE,
    0.40,
    0.95,
    'FDA_LABEL', 'Rifampin Capsules - Section 7 (Drug Interactions)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Rifampin induces CYP2C9, CYP3A4, CYP1A2; Warfarin clearance increases 3-4 fold', 'Also induces P-gp', 'CYP2C9, CYP3A4, CYP1A2 induction', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (X)', 'IDSA TB Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Rivaroxaban + Ketoconazole (CONTRAINDICATED)
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
    'RIVAROXABAN', 'Rivaroxaban',
    'KETOCONAZOLE', 'Ketoconazole',
    'pharmacokinetic',
    'contraindicated',
    'established',
    'Ketoconazole is a potent inhibitor of both CYP3A4 and P-glycoprotein. Rivaroxaban is a substrate of both, leading to dramatically increased exposure.',
    'Rivaroxaban AUC increases 2.6-fold, Cmax increases 1.7-fold. High risk of major bleeding including intracranial hemorrhage.',
    'CONTRAINDICATED. Use alternative antifungal (fluconazole for non-invasive, micafungin for invasive). If DOAC essential, consider apixaban with dose reduction.',
    FALSE,
    0.15,
    0.98,
    'FDA_LABEL', 'Xarelto (Rivaroxaban) - Section 4 (Contraindications), Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Ketoconazole inhibits CYP3A4 (Ki = 0.015 μM) and P-gp; Rivaroxaban AUC +160%', 'P-gp inhibition (IC50 = 1.2 μM)', 'CYP3A4, P-gp dual substrate', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Contraindicated (X)', 'ISTH DOAC Guidance 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- CATEGORY 2: QT PROLONGATION COMBINATIONS (CredibleMeds)
-- =============================================================================

-- Amiodarone + Levofloxacin (QT - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'AMIODARONE_LEVOFLOXACIN_001',
    'AMIODARONE', 'Amiodarone',
    'LEVOFLOXACIN', 'Levofloxacin',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs block cardiac hERG potassium channels (IKr), causing additive QT prolongation. Amiodarone also inhibits CYP metabolism of some fluoroquinolones.',
    'Risk of Torsades de Pointes (TdP), potentially fatal ventricular arrhythmia. Risk increased with hypokalemia, hypomagnesemia, bradycardia.',
    'Avoid combination. If essential, obtain baseline ECG, correct electrolytes (K+ >4.0, Mg2+ >2.0), monitor QTc daily. Discontinue if QTc >500ms or increases >60ms.',
    FALSE,
    0.35,
    0.94,
    'FDA_LABEL', 'Amiodarone - Section 5 (Warnings); Levofloxacin - Section 5 (QT Prolongation)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Amiodarone: Known Risk (Category 1); Levofloxacin: Possible Risk (Category 2) per CredibleMeds QT Drug Lists', NULL, 'hERG channel blockade', 'KNOWN + POSSIBLE',
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (D)', 'AHA/ACC QT Prolongation Statement 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Haloperidol + Methadone (QT - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HALOPERIDOL_METHADONE_001',
    'HALOPERIDOL', 'Haloperidol',
    'METHADONE', 'Methadone',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs are CredibleMeds "Known Risk" for QT prolongation. Haloperidol blocks hERG channels; Methadone blocks hERG and has prolonged half-life.',
    'Additive QT prolongation with high risk of Torsades de Pointes. Multiple fatalities reported. Risk highest with IV haloperidol and methadone doses >100mg/day.',
    'Avoid IV haloperidol with methadone. If oral haloperidol needed, obtain baseline ECG, correct electrolytes, start low dose. Monitor QTc; hold if >500ms.',
    FALSE,
    0.25,
    0.96,
    'FDA_LABEL', 'Haloperidol - Section 5 (QT Warning); Methadone - Black Box Warning (QT)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Haloperidol and Methadone are CredibleMeds Category 1 (Known Risk) for TdP', NULL, 'hERG channel blockade', 'KNOWN + KNOWN',
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (X)', 'SAMHSA Methadone Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Ondansetron + Droperidol (QT - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ONDANSETRON_DROPERIDOL_001',
    'ONDANSETRON', 'Ondansetron',
    'DROPERIDOL', 'Droperidol',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs prolong QT interval. Ondansetron blocks hERG at therapeutic concentrations; Droperidol has Black Box warning for QT prolongation.',
    'Additive QT prolongation. Droperidol Black Box warning cites cases of QT prolongation and TdP even at single doses ≤2.5mg.',
    'Avoid combination. Use alternative antiemetic (metoclopramide, prochlorperazine with caution). If droperidol needed for PONV, do not give ondansetron concurrently.',
    FALSE,
    0.20,
    0.93,
    'FDA_LABEL', 'Ondansetron - Section 5 (QT); Droperidol - Black Box Warning', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Ondansetron: Known Risk; Droperidol: Known Risk per CredibleMeds', NULL, 'hERG channel blockade', 'KNOWN + KNOWN',
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (X)', 'ASPAN PONV Guidelines 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- CATEGORY 3: SEROTONIN SYNDROME RISK
-- =============================================================================

-- SSRI + Tramadol (Serotonin Syndrome - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'SERTRALINE_TRAMADOL_001',
    'SERTRALINE', 'Sertraline',
    'TRAMADOL', 'Tramadol',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs increase serotonergic activity. SSRIs inhibit serotonin reuptake; Tramadol inhibits serotonin and norepinephrine reuptake. Also CYP2D6 interaction.',
    'Serotonin syndrome: hyperthermia, rigidity, myoclonus, autonomic instability, altered mental status. Can be life-threatening. Also seizure risk increased.',
    'Avoid combination if possible. If needed, use lowest effective doses, educate patient on serotonin syndrome symptoms, monitor closely first 24-72 hours.',
    FALSE,
    0.55,
    0.88,
    'FDA_LABEL', 'Tramadol - Section 5 (Serotonin Syndrome); Sertraline - Section 5 (Serotonin Syndrome)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Tramadol: serotonin reuptake inhibition (Ki = 1.4 μM); Sertraline: SERT Ki = 0.29 nM', NULL, 'CYP2D6 (Tramadol activation)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (D)', 'FDA Drug Safety Communication - Serotonin Syndrome 2016',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- SSRI + MAOI (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'FLUOXETINE_PHENELZINE_001',
    'FLUOXETINE', 'Fluoxetine',
    'PHENELZINE', 'Phenelzine',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'MAOIs prevent serotonin breakdown; SSRIs prevent reuptake. Combination causes massive serotonin accumulation. Fluoxetine has 5-week washout due to long half-life.',
    'Severe, potentially fatal serotonin syndrome with hyperthermia (>41°C), seizures, coma, cardiovascular collapse. Multiple fatalities reported.',
    'ABSOLUTELY CONTRAINDICATED. Wait 5 weeks after stopping fluoxetine before starting MAOI. Wait 2 weeks after stopping MAOI before starting any SSRI.',
    FALSE,
    0.05,
    1.00,
    'FDA_LABEL', 'Fluoxetine - Section 4 (Contraindications); Phenelzine - Section 4 (Contraindications)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Mechanism: SSRI + MAOI = massive synaptic serotonin accumulation', NULL, 'Not primarily CYP-mediated (pharmacodynamic)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Contraindicated (X)', 'APA Practice Guideline for MDD 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- CATEGORY 4: CYP450 INTERACTIONS
-- =============================================================================

-- Atorvastatin + Clarithromycin (CYP3A4 - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ATORVASTATIN_CLARITHROMYCIN_001',
    'ATORVASTATIN', 'Atorvastatin',
    'CLARITHROMYCIN', 'Clarithromycin',
    'pharmacokinetic',
    'major',
    'established',
    'Clarithromycin is a potent CYP3A4 inhibitor. Atorvastatin is extensively metabolized by CYP3A4. Combination increases statin exposure and myopathy risk.',
    'Increased risk of myopathy and rhabdomyolysis. Atorvastatin AUC increases ~4-fold. CK elevations, muscle pain, weakness may occur.',
    'Limit atorvastatin to 20mg/day during clarithromycin therapy. Consider azithromycin (no CYP3A4 inhibition) as alternative macrolide. Monitor for muscle symptoms.',
    TRUE,
    0.45,
    0.89,
    'FDA_LABEL', 'Lipitor (Atorvastatin) - Section 7 (Drug Interactions)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Clarithromycin: Strong CYP3A4 inhibitor (Ki = 0.5 μM); Atorvastatin AUC +380%', 'Also inhibits OATP1B1 (IC50 = 5 μM)', 'CYP3A4 substrate (major)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (C)', 'ACC/AHA Lipid Guidelines 2018',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Clopidogrel + Omeprazole (CYP2C19 - MAJOR)
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
    'CLOPIDOGREL', 'Clopidogrel',
    'OMEPRAZOLE', 'Omeprazole',
    'pharmacokinetic',
    'major',
    'established',
    'Clopidogrel is a prodrug requiring CYP2C19 activation. Omeprazole inhibits CYP2C19, reducing conversion to active metabolite and antiplatelet effect.',
    'Reduced antiplatelet effect (~40% reduction in active metabolite). Increased risk of cardiovascular events, stent thrombosis in ACS patients.',
    'Avoid omeprazole/esomeprazole. Use pantoprazole (weakest CYP2C19 inhibition) or H2 blocker. FDA boxed warning advises against combination.',
    FALSE,
    0.60,
    0.85,
    'FDA_LABEL', 'Plavix (Clopidogrel) - Black Box Warning; Omeprazole - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Omeprazole inhibits CYP2C19 (Ki = 2-6 μM); Clopidogrel active metabolite reduced ~45%', NULL, 'CYP2C19 (major activation pathway)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (X)', 'ACC/AHA DAPT Guidelines 2020',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Phenytoin + Fluoxetine (CYP2C9/CYP2C19 - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PHENYTOIN_FLUOXETINE_001',
    'PHENYTOIN', 'Phenytoin',
    'FLUOXETINE', 'Fluoxetine',
    'pharmacokinetic',
    'major',
    'established',
    'Fluoxetine and norfluoxetine inhibit CYP2C9 and CYP2C19, the primary enzymes metabolizing phenytoin. Phenytoin has narrow therapeutic index.',
    'Phenytoin toxicity: nystagmus, ataxia, lethargy, confusion, seizures. Levels may increase 50-100%. Effect may persist weeks after fluoxetine discontinuation.',
    'Monitor phenytoin levels closely when starting/stopping fluoxetine. May need 25-50% phenytoin dose reduction. Consider alternative antidepressant (sertraline, citalopram).',
    TRUE,
    0.30,
    0.91,
    'FDA_LABEL', 'Phenytoin - Section 7; Fluoxetine - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Fluoxetine/norfluoxetine: CYP2C9 Ki = 11 μM, CYP2C19 Ki = 2 μM', NULL, 'CYP2C9, CYP2C19 substrate', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (D)', 'AAN Epilepsy Guidelines 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- CATEGORY 5: NEPHROTOXICITY (Triple Whammy)
-- =============================================================================

-- Triple Whammy: ACEi + NSAID + Diuretic
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TRIPLE_WHAMMY_001',
    'LISINOPRIL', 'Lisinopril (ACEi)',
    'IBUPROFEN', 'Ibuprofen (NSAID) + Diuretic context',
    'pharmacodynamic',
    'major',
    'established',
    'Triple Whammy: ACEi dilates efferent arteriole, NSAIDs constrict afferent arteriole, diuretics reduce renal perfusion. Together they collapse GFR.',
    'Acute kidney injury (AKI), especially in elderly, diabetics, CKD patients. Risk increases 30% with triple therapy. May be irreversible.',
    'AVOID triple combination. If NSAID needed, use short course (<5 days), ensure euvolemia, hold ACEi/diuretic if possible. Check creatinine in 3-5 days.',
    FALSE,
    0.35,
    0.93,
    'FDA_LABEL', 'NSAID Class Labeling - Section 5 (Renal Effects); ACEi Class Labeling - Section 5', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Mechanism: Afferent vasoconstriction (NSAID) + Efferent vasodilation (ACEi) + Volume depletion (Diuretic) = GFR collapse', NULL, 'Not CYP-mediated (hemodynamic)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major', 'KDIGO AKI Guidelines 2022; Australian "Triple Whammy" Alert',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- CATEGORY 6: OPIOID SAFETY (FDA Black Box)
-- =============================================================================

-- Opioid + Benzodiazepine (FDA Black Box - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'OXYCODONE_ALPRAZOLAM_001',
    'OXYCODONE', 'Oxycodone',
    'ALPRAZOLAM', 'Alprazolam',
    'pharmacodynamic',
    'major',
    'established',
    'Additive CNS depression through different mechanisms. Opioids act on μ-receptors; benzodiazepines enhance GABA-A activity. Both depress respiratory drive.',
    'Profound sedation, respiratory depression, coma, death. >30% of opioid overdose deaths involve benzodiazepines. Risk highest in first 2 weeks.',
    'Avoid concurrent prescribing when possible. If combination needed, use lowest effective doses of each, limit duration, educate on overdose signs, consider naloxone prescription.',
    TRUE,
    0.40,
    0.97,
    'FDA_LABEL', 'FDA Black Box Warning (2016, updated 2020) - All opioids and benzodiazepines', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-drug-safety-communication-fda-warns-about-serious-risks', 'US',
    'DRUGBANK', 'Pharmacodynamic synergy: μ-opioid receptor + GABA-A receptor = synergistic respiratory depression', NULL, 'CYP3A4 (both substrates, but PD interaction dominates)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (Black Box)', 'CDC Opioid Prescribing Guidelines 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Methadone + CYP3A4 Inhibitor (QT + Pharmacokinetic - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'METHADONE_FLUCONAZOLE_001',
    'METHADONE', 'Methadone',
    'FLUCONAZOLE', 'Fluconazole',
    'pharmacokinetic',
    'major',
    'established',
    'Fluconazole inhibits CYP3A4 and CYP2C9, both involved in methadone metabolism. Increased methadone levels compound QT risk (methadone is CredibleMeds Known Risk).',
    'Increased methadone levels with enhanced respiratory depression AND QT prolongation risk. Dual mechanism makes this particularly dangerous.',
    'Reduce methadone dose 25-50% when starting fluconazole. Monitor for sedation, respiratory depression. Obtain baseline ECG, monitor QTc, correct electrolytes.',
    TRUE,
    0.25,
    0.92,
    'FDA_LABEL', 'Methadone - Section 7; Fluconazole - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Fluconazole inhibits CYP3A4, CYP2C9; Methadone AUC increased; Also additive QT effect', NULL, 'CYP3A4, CYP2C9 substrate', 'KNOWN (Methadone)',
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (D)', 'SAMHSA Methadone Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- CATEGORY 7: ANTIPLATELET INTERACTIONS
-- =============================================================================

-- Aspirin + Ibuprofen (Antiplatelet Antagonism - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ASPIRIN_IBUPROFEN_001',
    'ASPIRIN', 'Aspirin (Cardioprotective)',
    'IBUPROFEN', 'Ibuprofen',
    'pharmacodynamic',
    'major',
    'established',
    'Ibuprofen competes with aspirin for COX-1 binding site in platelets. If ibuprofen binds first, it blocks aspirin access, negating cardioprotective effect.',
    'Loss of aspirin cardioprotection. Increased risk of MI, stroke in patients taking aspirin for CV prevention. Effect depends on timing of administration.',
    'Take aspirin at least 30 minutes BEFORE ibuprofen OR 8 hours AFTER. If regular NSAID needed, consider naproxen (less interference) or use enteric-coated aspirin.',
    FALSE,
    0.50,
    0.82,
    'FDA_LABEL', 'FDA Science Paper 2006 - Ibuprofen Interference with Aspirin', 'https://www.fda.gov/drugs/postmarket-drug-safety-information-patients-and-providers/information-concomitant-use-ibuprofen-and-aspirin', 'US',
    'DRUGBANK', 'Competitive COX-1 binding: Ibuprofen reversibly binds COX-1; blocks irreversible aspirin acetylation of Ser530', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (C)', 'ACC/AHA CV Prevention Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- CATEGORY 8: DIABETES MEDICATION SAFETY
-- =============================================================================

-- Metformin + IV Contrast (Lactic Acidosis Risk - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'METFORMIN_CONTRAST_001',
    'METFORMIN', 'Metformin',
    'IODINATED_CONTRAST', 'Iodinated IV Contrast',
    'pharmacokinetic',
    'major',
    'established',
    'IV contrast can cause contrast-induced nephropathy, impairing metformin excretion. Metformin accumulation increases lactic acidosis risk.',
    'Metformin-associated lactic acidosis (MALA): pH <7.35, lactate >5 mmol/L. Mortality 30-50%. Risk highest in renal impairment, heart failure, sepsis.',
    'Check eGFR before contrast. If eGFR ≥30: hold metformin day of and 48h after, recheck creatinine before restarting. If eGFR <30: avoid contrast or hold metformin longer.',
    FALSE,
    0.35,
    0.90,
    'FDA_LABEL', 'Metformin - Section 5 (Lactic Acidosis); ACR Contrast Manual 2023', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Contrast-induced AKI reduces metformin renal clearance; Metformin inhibits mitochondrial Complex I causing lactate accumulation', 'OCT1/2 transporter-mediated uptake', 'Not CYP-mediated (renal elimination)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major', 'ACR Contrast Manual v10.3 (2023); ADA Diabetes Standards 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- Sulfonylurea + Fluoroquinolone (Hypoglycemia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'GLYBURIDE_LEVOFLOXACIN_001',
    'GLYBURIDE', 'Glyburide',
    'LEVOFLOXACIN', 'Levofloxacin',
    'pharmacodynamic',
    'major',
    'established',
    'Fluoroquinolones cause dysglycemia through multiple mechanisms: augment insulin secretion, inhibit gluconeogenesis, and may have direct pancreatic β-cell effects.',
    'Severe hypoglycemia, sometimes prolonged (>24h). Cases of hypoglycemic coma reported. Also can cause hyperglycemia in some patients. FDA warning 2018.',
    'Monitor blood glucose closely during fluoroquinolone therapy. Educate on hypoglycemia symptoms. Consider alternative antibiotic. May need temporary sulfonylurea dose reduction.',
    TRUE,
    0.30,
    0.87,
    'FDA_LABEL', 'FDA Safety Communication 2018 - Fluoroquinolone Dysglycemia; Glyburide - Section 7', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-reinforces-safety-information-about-serious-low-blood-sugar-levels-and-mental-health-side', 'US',
    'DRUGBANK', 'Fluoroquinolones augment sulfonylurea-induced insulin secretion; Block ATP-sensitive K+ channels in pancreatic β-cells', NULL, 'CYP2C9 (glyburide substrate)', NULL,
    'LEXICOMP', 'Lexicomp Drug Interactions 2024 - Severity: Major (C)', 'ADA Diabetes Standards 2024; IDSA Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
);

-- =============================================================================
-- VERIFICATION & SUMMARY
-- =============================================================================
DO $$
DECLARE
    v_total INTEGER;
    v_governed INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total FROM drug_interactions WHERE active = TRUE;
    SELECT COUNT(*) INTO v_governed FROM drug_interactions
    WHERE active = TRUE
      AND gov_regulatory_authority IS NOT NULL
      AND gov_pharmacology_authority IS NOT NULL
      AND gov_clinical_authority IS NOT NULL;

    RAISE NOTICE '===========================================';
    RAISE NOTICE 'KB-5 DDI GOVERNANCE STATUS';
    RAISE NOTICE '===========================================';
    RAISE NOTICE 'Total active DDIs: %', v_total;
    RAISE NOTICE 'Fully governed: %', v_governed;
    RAISE NOTICE 'Governance compliance: %', ROUND((v_governed::NUMERIC / v_total * 100), 1) || '%';
    RAISE NOTICE '===========================================';
END $$;

-- Show summary by category
SELECT
    CASE
        WHEN interaction_id LIKE 'WARFARIN%' OR interaction_id LIKE 'RIVAROXABAN%' THEN 'Anticoagulant'
        WHEN gov_qt_risk_category IS NOT NULL THEN 'QT Prolongation'
        WHEN interaction_id LIKE '%TRAMADOL%' OR interaction_id LIKE '%PHENELZINE%' THEN 'Serotonin Syndrome'
        WHEN interaction_id LIKE 'TRIPLE%' THEN 'Nephrotoxicity'
        WHEN interaction_id LIKE 'OXYCODONE%' OR interaction_id LIKE 'METHADONE%' THEN 'Opioid Safety'
        WHEN interaction_id LIKE 'METFORMIN%' OR interaction_id LIKE 'GLYBURIDE%' THEN 'Diabetes'
        ELSE 'Other CYP/DDI'
    END AS category,
    COUNT(*) as count,
    STRING_AGG(DISTINCT severity, ', ' ORDER BY severity) as severities
FROM drug_interactions
WHERE active = TRUE
GROUP BY 1
ORDER BY count DESC;
