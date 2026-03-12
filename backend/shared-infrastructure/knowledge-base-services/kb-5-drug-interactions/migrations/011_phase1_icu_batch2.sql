-- =============================================================================
-- KB-5 Phase-1 ICU Critical DDIs - Batch 2
-- Migration: 011_phase1_icu_batch2.sql
-- Categories: Transplant, Electrolytes, Extended QT, Chemotherapy
-- =============================================================================

-- =============================================================================
-- CATEGORY 7: TRANSPLANT IMMUNOSUPPRESSANTS (ICU Critical)
-- =============================================================================

-- Tacrolimus + Fluconazole (CYP3A4 Inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TACROLIMUS_FLUCONAZOLE_001',
    'TACROLIMUS', 'Tacrolimus (Prograf)',
    'FLUCONAZOLE', 'Fluconazole',
    'pharmacokinetic',
    'major',
    'established',
    'Fluconazole inhibits CYP3A4 (primary tacrolimus metabolism). Dose-dependent effect: 100mg fluconazole doubles tacrolimus levels.',
    'Tacrolimus toxicity: nephrotoxicity, neurotoxicity (tremor, seizures), hyperkalemia, hypertension. Narrow therapeutic index amplifies risk.',
    'Reduce tacrolimus dose by 50% when starting fluconazole. Monitor tacrolimus trough levels within 3-5 days. Target trough 5-10 ng/mL.',
    TRUE, 0.55, 0.94,
    'FDA_LABEL', 'Prograf - Section 7 (Azole antifungals)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Fluconazole: CYP3A4 inhibitor (Ki = 15 μM); Tacrolimus AUC increases 1.4-4x depending on fluconazole dose', 'P-gp substrate (tacrolimus)', 'CYP3A4 (major)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'KDIGO Transplant Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Tacrolimus + Voriconazole (CYP3A4 Inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TACROLIMUS_VORICONAZOLE_001',
    'TACROLIMUS', 'Tacrolimus (Prograf)',
    'VORICONAZOLE', 'Voriconazole',
    'pharmacokinetic',
    'major',
    'established',
    'Voriconazole is a potent CYP3A4 inhibitor. Tacrolimus levels increase 3-10 fold. One of the most significant drug interactions in transplant.',
    'Severe tacrolimus toxicity: acute nephrotoxicity (Cr may double), neurotoxicity, QT prolongation (both drugs). May require dialysis.',
    'Reduce tacrolimus dose to 1/3 of baseline when starting voriconazole. Monitor levels every 2-3 days initially. Target trough 5-8 ng/mL.',
    TRUE, 0.40, 0.98,
    'FDA_LABEL', 'Prograf - Section 7; Vfend - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Voriconazole: Strong CYP3A4 inhibitor; Tacrolimus AUC increases 3-10x; Also both have QT effects', 'P-gp inhibition additive', 'CYP3A4 (strong inhibition)', 'POSSIBLE (voriconazole)',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Aspergillosis Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Cyclosporine + Diltiazem (CYP3A4 Inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CYCLOSPORINE_DILTIAZEM_001',
    'CYCLOSPORINE', 'Cyclosporine (Neoral)',
    'DILTIAZEM', 'Diltiazem',
    'pharmacokinetic',
    'major',
    'established',
    'Diltiazem inhibits CYP3A4 and P-gp. Commonly used intentionally as "cyclosporine-sparing" strategy to reduce cyclosporine costs.',
    'Cyclosporine levels increase 50-100%. Risk of nephrotoxicity, hypertension, gingival hyperplasia if not monitored.',
    'If adding diltiazem: reduce cyclosporine dose by 25-50%. Monitor levels weekly until stable. Some centers use this interaction therapeutically.',
    TRUE, 0.50, 0.88,
    'FDA_LABEL', 'Neoral - Section 7 (Calcium channel blockers)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Diltiazem: Moderate CYP3A4 inhibitor; Cyclosporine bioavailability increases ~50%', 'P-gp inhibition contributes', 'CYP3A4 (moderate inhibition)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'KDIGO Transplant Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Cyclosporine + Aminoglycosides (Nephrotoxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CYCLOSPORINE_GENTAMICIN_001',
    'CYCLOSPORINE', 'Cyclosporine',
    'GENTAMICIN', 'Gentamicin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive nephrotoxicity: Cyclosporine causes afferent arteriolar vasoconstriction; Aminoglycosides cause proximal tubular necrosis.',
    'Acute kidney injury, may jeopardize transplant function. AKI incidence >30% when combined in transplant recipients.',
    'Avoid if possible. If aminoglycoside essential, use extended-interval dosing, monitor levels, aggressive hydration, check Cr daily.',
    FALSE, 0.25, 0.92,
    'FDA_LABEL', 'Neoral - Section 5 (Nephrotoxicity)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Cyclosporine: Afferent arteriolar vasoconstriction via endothelin; Aminoglycosides: Tubular necrosis via lysosomal accumulation', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'KDIGO Transplant Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Mycophenolate + Proton Pump Inhibitors (Reduced Absorption - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'MYCOPHENOLATE_OMEPRAZOLE_001',
    'MYCOPHENOLATE', 'Mycophenolate Mofetil (CellCept)',
    'OMEPRAZOLE', 'Omeprazole',
    'pharmacokinetic',
    'major',
    'established',
    'PPIs raise gastric pH, reducing mycophenolate mofetil (MMF) dissolution and absorption. MPA AUC reduced 25-35%.',
    'Subtherapeutic immunosuppression, risk of rejection. Effect less pronounced with mycophenolic acid (Myfortic) which is enteric-coated.',
    'Consider switching to mycophenolic acid (Myfortic) if PPI essential. Alternative: increase MMF dose by 25-50% with MPA level monitoring.',
    TRUE, 0.55, 0.82,
    'FDA_LABEL', 'CellCept - Section 7 (PPIs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'pH-dependent dissolution: MMF requires acidic environment for optimal absorption; PPI raises pH >4', NULL, 'Not CYP-mediated (absorption)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'KDIGO Transplant Guidelines 2023',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 8: ELECTROLYTE-AFFECTING DRUGS (ICU Critical)
-- =============================================================================

-- Digoxin + Amiodarone (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DIGOXIN_AMIODARONE_001',
    'DIGOXIN', 'Digoxin',
    'AMIODARONE', 'Amiodarone',
    'pharmacokinetic',
    'major',
    'established',
    'Amiodarone inhibits P-glycoprotein (digoxin efflux transporter) and reduces renal clearance. Digoxin levels increase 70-100%.',
    'Digoxin toxicity: arrhythmias (especially with hypokalemia from amiodarone), nausea, visual disturbances, bradycardia.',
    'Reduce digoxin dose by 50% when starting amiodarone. Monitor levels in 1 week. Target digoxin level 0.5-1.0 ng/mL.',
    TRUE, 0.45, 0.91,
    'FDA_LABEL', 'Digoxin - Section 7; Amiodarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Amiodarone inhibits P-gp (IC50 = 5 μM); Also reduces renal tubular secretion of digoxin', 'P-gp inhibition primary mechanism', 'Not CYP-mediated (P-gp)', 'KNOWN (Amiodarone)',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA AF Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Spironolactone + ACE Inhibitors (Hyperkalemia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'SPIRONOLACTONE_LISINOPRIL_001',
    'SPIRONOLACTONE', 'Spironolactone',
    'LISINOPRIL', 'Lisinopril',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs reduce potassium excretion: ACEi reduces aldosterone (K+ retention); Spironolactone blocks aldosterone receptor directly.',
    'Severe hyperkalemia (K+ >6.0 mEq/L), cardiac arrhythmias, cardiac arrest. Risk highest in CKD, diabetes, elderly.',
    'Standard of care in HFrEF (RALES trial) with careful monitoring. Start spironolactone 12.5-25mg. Check K+ and Cr at 3 days, 1 week, monthly.',
    FALSE, 0.50, 0.89,
    'FDA_LABEL', 'Spironolactone - Section 5 (Hyperkalemia)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'ACEi: reduces angiotensin II-stimulated aldosterone; Spironolactone: MR antagonism; Both retain K+', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACC/AHA HF Guidelines 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Potassium Chloride + ACE Inhibitors (Hyperkalemia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'KCL_LISINOPRIL_001',
    'POTASSIUM_CHLORIDE', 'Potassium Chloride',
    'LISINOPRIL', 'Lisinopril',
    'pharmacodynamic',
    'major',
    'established',
    'ACE inhibitors reduce aldosterone-mediated potassium excretion. Adding KCl supplementation further increases K+ load.',
    'Hyperkalemia with risk of fatal arrhythmias. Especially dangerous if K+ supplementation is aggressive or patient has CKD.',
    'Usually avoid K+ supplements in patients on ACEi with normal K+. If hypokalemia present, supplement cautiously with frequent monitoring.',
    TRUE, 0.40, 0.86,
    'FDA_LABEL', 'ACE inhibitor class labeling - Section 5 (Hyperkalemia)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'ACEi-induced K+ retention + exogenous K+ = risk of hyperkalemia', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'KDIGO CKD Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 9: EXTENDED QT PROLONGATION (ICU Critical)
-- =============================================================================

-- Ciprofloxacin + Amiodarone (QT - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CIPROFLOXACIN_AMIODARONE_001',
    'CIPROFLOXACIN', 'Ciprofloxacin',
    'AMIODARONE', 'Amiodarone',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs block hERG potassium channels. Amiodarone: Known Risk (CredibleMeds Category 1). Ciprofloxacin: Possible Risk (Category 2).',
    'Additive QT prolongation, risk of Torsades de Pointes. Ciprofloxacin also inhibits CYP1A2 (minor amiodarone pathway).',
    'Avoid if possible. If essential, baseline ECG, correct K+ >4.0 and Mg >2.0, monitor QTc. Consider alternative antibiotic (non-FQ).',
    FALSE, 0.30, 0.90,
    'FDA_LABEL', 'Ciprofloxacin - Section 5 (QT); Amiodarone - Section 5 (QT)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Amiodarone: Known Risk (Cat 1); Ciprofloxacin: Possible Risk (Cat 2) per CredibleMeds', NULL, 'hERG channel blockade', 'KNOWN + POSSIBLE',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'AHA QT Prolongation Statement 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Quetiapine + Methadone (QT - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QUETIAPINE_METHADONE_001',
    'QUETIAPINE', 'Quetiapine',
    'METHADONE', 'Methadone',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs prolong QT interval. Methadone: Known Risk (CredibleMeds Cat 1). Quetiapine: Possible Risk (Cat 2). Common combination in psychiatry.',
    'Additive QT prolongation with TdP risk. Both drugs used commonly in substance use disorder + psychiatric comorbidity.',
    'ECG at baseline and after dose changes. Correct electrolytes. Consider alternative antipsychotic (aripiprazole - no QT effect).',
    FALSE, 0.35, 0.88,
    'FDA_LABEL', 'Quetiapine - Section 5 (QT); Methadone - Black Box (QT)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Methadone: Known Risk (Cat 1); Quetiapine: Possible Risk (Cat 2) per CredibleMeds', NULL, 'hERG channel blockade', 'KNOWN + POSSIBLE',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Schizophrenia Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Azithromycin + Amiodarone (QT - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'AZITHROMYCIN_AMIODARONE_001',
    'AZITHROMYCIN', 'Azithromycin',
    'AMIODARONE', 'Amiodarone',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs prolong QT interval via hERG blockade. Azithromycin: Known Risk per CredibleMeds. FDA warning 2013.',
    'Additive QT prolongation, TdP risk. Azithromycin may be preferred over fluoroquinolones for CYP reasons but still has QT risk.',
    'Avoid if possible. Short course (3-5 days) may be acceptable with monitoring. Baseline ECG, correct K+/Mg, avoid other QT drugs.',
    FALSE, 0.40, 0.89,
    'FDA_LABEL', 'Azithromycin - FDA Drug Safety Communication 2013 (QT)', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-drug-safety-communication-azithromycin-zithromax-or-zmax-and-risk-potentially-fatal-heart', 'US',
    'CREDIBLEMEDS', 'Azithromycin: Known Risk (Cat 1); Amiodarone: Known Risk (Cat 1) per CredibleMeds', NULL, 'hERG channel blockade', 'KNOWN + KNOWN',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'AHA QT Statement 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 10: CHEMOTHERAPY HIGH-ALERT (ICU Critical)
-- =============================================================================

-- Methotrexate + NSAIDs (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'METHOTREXATE_IBUPROFEN_001',
    'METHOTREXATE', 'Methotrexate',
    'IBUPROFEN', 'Ibuprofen',
    'pharmacokinetic',
    'major',
    'established',
    'NSAIDs reduce renal blood flow (prostaglandin inhibition) and compete for tubular secretion. Methotrexate clearance reduced 30-50%.',
    'Methotrexate toxicity: severe mucositis, myelosuppression, hepatotoxicity, nephrotoxicity. Fatalities reported.',
    'Avoid NSAIDs during high-dose methotrexate. For low-dose (RA), can use with caution but avoid around MTX dosing day.',
    FALSE, 0.35, 0.94,
    'FDA_LABEL', 'Methotrexate - Section 7 (NSAIDs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'NSAIDs: reduce GFR via PG inhibition; Compete with MTX for OAT-mediated tubular secretion', 'OAT1/OAT3 competition', 'Not CYP-mediated (renal)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X for high-dose)', 'NCCN Lymphoma Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Methotrexate + Trimethoprim-Sulfamethoxazole (Myelosuppression - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'METHOTREXATE_TMPSMX_001',
    'METHOTREXATE', 'Methotrexate',
    'TMP_SMX', 'Trimethoprim-Sulfamethoxazole',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both drugs are antifolates. TMP inhibits dihydrofolate reductase (same target as MTX). SMX competes for renal secretion.',
    'Severe pancytopenia, aplastic anemia, death. Synergistic folate depletion causes profound bone marrow suppression.',
    'CONTRAINDICATED. Use alternative antibiotic. If TMP-SMX absolutely essential (PCP prophylaxis), reduce MTX dose and monitor CBC closely.',
    FALSE, 0.20, 0.98,
    'FDA_LABEL', 'Methotrexate - Section 7 (TMP-SMX)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'TMP: DHFR inhibition synergistic with MTX; SMX: competes for OAT-mediated renal secretion; Both reduce folate', 'OAT3 competition (SMX)', 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'NCCN Supportive Care Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Cisplatin + Aminoglycosides (Nephro/Ototoxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CISPLATIN_GENTAMICIN_001',
    'CISPLATIN', 'Cisplatin',
    'GENTAMICIN', 'Gentamicin',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs cause nephrotoxicity (tubular damage) and ototoxicity (cochlear hair cell death via oxidative stress). Additive/synergistic toxicity.',
    'Acute kidney injury (>50% risk combined), permanent hearing loss (high-frequency first). May limit future cisplatin dosing.',
    'Avoid combination. If aminoglycoside essential, use extended-interval dosing, aggressive hydration, monitor Cr and audiometry.',
    FALSE, 0.20, 0.95,
    'FDA_LABEL', 'Cisplatin - Black Box (Nephro/Oto); Gentamicin - Black Box (Nephro/Oto)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Both generate ROS in proximal tubule and cochlear hair cells; OCT2-mediated cisplatin uptake in kidney', 'OCT2, Megalin transport', 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'NCCN Supportive Care Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- VERIFICATION
-- =============================================================================
DO $$
DECLARE
    v_total INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '============================================';
    RAISE NOTICE 'Phase-1 ICU Critical DDIs - Batch 2 Complete';
    RAISE NOTICE 'Total active DDIs now: %', v_total;
    RAISE NOTICE '============================================';
END $$;
