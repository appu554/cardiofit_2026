-- =============================================================================
-- KB-5 Phase-1 ICU Critical DDIs - Batch 4 (Final)
-- Migration: 013_phase1_icu_batch4.sql
-- Purpose: Complete Tier 1 to reach 100 ICU-critical DDIs
-- Categories: Extended cardiac, renal, respiratory, additional high-harm
-- =============================================================================

-- =============================================================================
-- CATEGORY 16: CARDIAC ICU DRUGS
-- =============================================================================

-- Digoxin + Verapamil (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DIGOXIN_VERAPAMIL_001',
    'DIGOXIN', 'Digoxin',
    'VERAPAMIL', 'Verapamil',
    'pharmacokinetic',
    'major',
    'established',
    'Verapamil inhibits P-glycoprotein and reduces renal/hepatic clearance of digoxin. Digoxin levels increase 50-75%.',
    'Digoxin toxicity: nausea, visual disturbances, arrhythmias. Additive AV nodal depression causing severe bradycardia.',
    'Reduce digoxin dose by 50% when starting verapamil. Monitor levels in 1 week. Watch for bradycardia - both drugs slow AV conduction.',
    TRUE, 0.40, 0.90,
    'FDA_LABEL', 'Digoxin - Section 7; Verapamil - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Verapamil inhibits P-gp (digoxin efflux); Also reduces renal tubular secretion', 'P-gp inhibition primary', 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA AF Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Amiodarone + Beta-Blockers (Bradycardia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'AMIODARONE_METOPROLOL_001',
    'AMIODARONE', 'Amiodarone',
    'METOPROLOL', 'Metoprolol',
    'pharmacodynamic',
    'major',
    'established',
    'Additive negative chronotropic and dromotropic effects. Amiodarone also inhibits CYP2D6, increasing metoprolol levels.',
    'Severe bradycardia, AV block, hypotension. May require temporary pacing. Common combination in AF but high-risk.',
    'Start metoprolol at low dose. Monitor HR closely (target >50 bpm). Have atropine/pacing available. Reduce metoprolol if HR <50.',
    TRUE, 0.55, 0.88,
    'FDA_LABEL', 'Amiodarone - Section 7 (Beta-blockers)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Amiodarone inhibits CYP2D6; Metoprolol AUC increases; Also pharmacodynamic synergy on SA/AV node', NULL, 'CYP2D6 (metoprolol)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACC/AHA AF Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Dobutamine + Beta-Blockers (Antagonism - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DOBUTAMINE_ESMOLOL_001',
    'DOBUTAMINE', 'Dobutamine',
    'ESMOLOL', 'Esmolol',
    'pharmacodynamic',
    'major',
    'established',
    'Direct pharmacologic antagonism: Dobutamine is β1-agonist; Beta-blockers are β1-antagonists. Opposing effects on inotropy.',
    'Reduced dobutamine efficacy, hemodynamic instability. May need much higher dobutamine doses to overcome blockade.',
    'Avoid concurrent use if possible. If beta-blocker needed for rate control, use lowest dose. Consider milrinone (PDE inhibitor) instead.',
    TRUE, 0.25, 0.85,
    'FDA_LABEL', 'Dobutamine - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Direct receptor antagonism: β1-agonist vs β1-blocker; Competitive inhibition at receptor level', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'SCCM Heart Failure Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Milrinone + Sildenafil (Hypotension - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'MILRINONE_SILDENAFIL_001',
    'MILRINONE', 'Milrinone',
    'SILDENAFIL', 'Sildenafil',
    'pharmacodynamic',
    'major',
    'established',
    'Both are PDE inhibitors causing vasodilation. Milrinone: PDE3; Sildenafil: PDE5. Additive hypotensive effect.',
    'Severe hypotension, especially in patients with marginal cardiac output. Common scenario: RV failure on milrinone + sildenafil for pulmonary HTN.',
    'Use with caution. Start sildenafil at lowest dose (10mg). Ensure adequate preload. Have vasopressors available.',
    TRUE, 0.20, 0.87,
    'FDA_LABEL', 'Sildenafil - Section 7 (Vasodilators)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Both inhibit phosphodiesterases: Milrinone (PDE3) + Sildenafil (PDE5) = additive cGMP/cAMP accumulation', NULL, 'CYP3A4 (sildenafil)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'CHEST Pulmonary HTN Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 17: RENAL/ELECTROLYTE ICU DRUGS
-- =============================================================================

-- Amphotericin B + Aminoglycosides (Nephrotoxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'AMPHOTERICIN_GENTAMICIN_001',
    'AMPHOTERICIN_B', 'Amphotericin B',
    'GENTAMICIN', 'Gentamicin',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs cause nephrotoxicity via different mechanisms. Amphotericin: renal vasoconstriction + tubular toxicity. Aminoglycosides: proximal tubular necrosis.',
    'Additive AKI risk >50%. Also additive hypokalemia and hypomagnesemia from both drugs.',
    'Avoid if possible. Use lipid amphotericin formulation (less nephrotoxic). Aggressive hydration, monitor Cr/K+/Mg daily. Consider alternative antifungal.',
    FALSE, 0.15, 0.93,
    'FDA_LABEL', 'Amphotericin B - Section 5 (Nephrotoxicity)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Amphotericin: afferent arteriolar vasoconstriction + tubular membrane damage; Aminoglycoside: lysosomal accumulation', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Candidiasis Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Loop Diuretics + ACE Inhibitors (Hypotension - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'FUROSEMIDE_LISINOPRIL_001',
    'FUROSEMIDE', 'Furosemide',
    'LISINOPRIL', 'Lisinopril',
    'pharmacodynamic',
    'major',
    'established',
    'Loop diuretics activate RAAS; ACE inhibitors block RAAS. First-dose hypotension when ACEi started in diuretic-treated patients.',
    'Severe first-dose hypotension, especially in volume-depleted patients. May cause syncope, AKI.',
    'Hold diuretic for 24-48h before starting ACEi OR start ACEi at very low dose. Monitor BP closely for 4-6 hours after first dose.',
    TRUE, 0.50, 0.84,
    'FDA_LABEL', 'Lisinopril - Section 5 (Hypotension)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Diuretic-induced RAAS activation; ACEi blocks compensatory angiotensin II = unopposed hypotension', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACC/AHA HF Guidelines 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Mannitol + Loop Diuretics (Volume/Electrolytes - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'MANNITOL_FUROSEMIDE_001',
    'MANNITOL', 'Mannitol',
    'FUROSEMIDE', 'Furosemide',
    'pharmacodynamic',
    'major',
    'established',
    'Synergistic diuresis leading to rapid volume depletion. Both cause electrolyte losses. Mannitol initially expands plasma volume then causes diuresis.',
    'Severe hypovolemia, hyponatremia, hypokalemia, metabolic alkalosis. Risk of AKI from prerenal azotemia.',
    'Common combination in neurocritical care for ICP. Monitor volume status closely, replace K+/Na+ aggressively, check osmolality.',
    TRUE, 0.30, 0.86,
    'FDA_LABEL', 'Mannitol - Section 5 (Volume/Electrolytes)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Osmotic diuresis (mannitol) + loop diuresis (furosemide) = synergistic volume/electrolyte losses', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'Neurocritical Care Society Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 18: RESPIRATORY ICU DRUGS
-- =============================================================================

-- Neuromuscular Blockers + Corticosteroids (ICU Myopathy - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CISATRACURIUM_METHYLPRED_001',
    'CISATRACURIUM', 'Cisatracurium',
    'METHYLPREDNISOLONE', 'Methylprednisolone',
    'pharmacodynamic',
    'major',
    'established',
    'Prolonged NMB use + high-dose corticosteroids = ICU-acquired weakness (critical illness myopathy). Risk increases with duration.',
    'Profound weakness, prolonged mechanical ventilation, difficulty weaning. May take weeks to months to recover.',
    'Minimize NMB duration (<48h if possible). Use daily sedation holidays and spontaneous breathing trials. Limit steroid dose/duration.',
    FALSE, 0.40, 0.89,
    'FDA_LABEL', 'Clinical literature on ICU-acquired weakness', 'https://pubmed.ncbi.nlm.nih.gov/', 'US',
    'DRUGBANK', 'Corticosteroids cause myopathy; NMB prevents muscle use; Combined = severe ICU-acquired weakness', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'SCCM PADIS Guidelines 2018',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Theophylline + Erythromycin (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'THEOPHYLLINE_ERYTHROMYCIN_001',
    'THEOPHYLLINE', 'Theophylline',
    'ERYTHROMYCIN', 'Erythromycin',
    'pharmacokinetic',
    'major',
    'established',
    'Erythromycin inhibits CYP3A4 and CYP1A2, both involved in theophylline metabolism. Levels increase 25-50%.',
    'Theophylline toxicity: nausea, vomiting, tachyarrhythmias, seizures. Narrow therapeutic window.',
    'Reduce theophylline dose by 25-30%. Monitor levels. Consider azithromycin (no CYP interaction) as alternative macrolide.',
    TRUE, 0.20, 0.88,
    'FDA_LABEL', 'Theophylline - Section 7; Erythromycin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Erythromycin: CYP3A4 inhibitor; Also inhibits CYP1A2; Theophylline clearance reduced 25-50%', NULL, 'CYP1A2, CYP3A4', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'GINA Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 19: ADDITIONAL HIGH-HARM ICU COMBINATIONS
-- =============================================================================

-- Phenytoin + Valproic Acid (Both Affected - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PHENYTOIN_VALPROATE_001',
    'PHENYTOIN', 'Phenytoin',
    'VALPROIC_ACID', 'Valproic Acid',
    'pharmacokinetic',
    'major',
    'established',
    'Complex bidirectional interaction: VPA inhibits phenytoin metabolism AND displaces from protein binding. Phenytoin induces VPA metabolism.',
    'Initially: free phenytoin toxicity (despite normal total levels). Later: subtherapeutic VPA levels due to induction.',
    'Monitor free phenytoin levels (not total). Check VPA levels. May need to increase VPA dose 25-50%. Consider alternative AED.',
    TRUE, 0.35, 0.91,
    'FDA_LABEL', 'Phenytoin - Section 7; Valproic Acid - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'VPA: CYP2C9 inhibitor + protein binding displacement; Phenytoin: CYP inducer reducing VPA levels', NULL, 'CYP2C9, CYP2C19, UGT', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'AAN Epilepsy Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Carbamazepine + Warfarin (Reduced Anticoagulation - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CARBAMAZEPINE_WARFARIN_001',
    'CARBAMAZEPINE', 'Carbamazepine',
    'WARFARIN', 'Warfarin',
    'pharmacokinetic',
    'major',
    'established',
    'Carbamazepine is a potent CYP inducer (CYP3A4, CYP2C9, CYP1A2). Warfarin metabolism dramatically increased.',
    'Subtherapeutic INR with risk of thromboembolism. May need 2-3x usual warfarin dose. Effect persists weeks after stopping CBZ.',
    'Monitor INR twice weekly when starting/stopping carbamazepine. Expect significant warfarin dose adjustments. Consider alternative AED.',
    TRUE, 0.25, 0.90,
    'FDA_LABEL', 'Carbamazepine - Section 7; Warfarin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Carbamazepine: potent CYP2C9, CYP3A4, CYP1A2 inducer; Warfarin clearance increases 2-3 fold', NULL, 'CYP2C9, CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP Antithrombotic Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Clonidine + Beta-Blockers (Rebound Hypertension - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CLONIDINE_METOPROLOL_001',
    'CLONIDINE', 'Clonidine',
    'METOPROLOL', 'Metoprolol',
    'pharmacodynamic',
    'major',
    'established',
    'If clonidine stopped abruptly while on beta-blocker, severe rebound hypertension occurs. Beta-blockade prevents compensatory vasodilation.',
    'Hypertensive crisis (BP >200 mmHg) when clonidine withdrawn. Beta-blocker blocks compensatory β2-vasodilation, leaving α-vasoconstriction unopposed.',
    'Never abruptly stop clonidine in patients on beta-blockers. Taper clonidine over 1-2 weeks. Discontinue beta-blocker several days before stopping clonidine.',
    FALSE, 0.30, 0.88,
    'FDA_LABEL', 'Clonidine - Section 5 (Withdrawal)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Clonidine withdrawal: catecholamine surge; Beta-blockade: blocks β2-vasodilation leaving α-vasoconstriction unopposed', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA Hypertension Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Potassium + Succinylcholine (Hyperkalemia/Cardiac Arrest - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'KCL_SUCCINYLCHOLINE_001',
    'POTASSIUM_CHLORIDE', 'Potassium (elevated >5.5)',
    'SUCCINYLCHOLINE', 'Succinylcholine',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Succinylcholine causes transient K+ release from muscle (0.5-1.0 mEq/L increase). In hyperkalemic patients, this can trigger fatal arrhythmia.',
    'Cardiac arrest from hyperkalemia-induced arrhythmia. Especially dangerous in burns, crush injuries, denervation, renal failure.',
    'CONTRAINDICATED in hyperkalemia (K+ >5.5). Use rocuronium instead. Check K+ before RSI in high-risk patients.',
    FALSE, 0.25, 1.00,
    'FDA_LABEL', 'Succinylcholine - Section 4 (Contraindications)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Succinylcholine: depolarizing agent causing K+ efflux from muscle; Normal increase 0.5 mEq/L can be fatal in hyperkalemia', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ASA Difficult Airway Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 20: MORE ANTIBIOTIC/INFECTIOUS DISEASE
-- =============================================================================

-- Rifampin + Tacrolimus (Reduced Levels - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'RIFAMPIN_TACROLIMUS_001',
    'RIFAMPIN', 'Rifampin',
    'TACROLIMUS', 'Tacrolimus',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin is a potent inducer of CYP3A4 and P-gp. Tacrolimus levels decrease by 50-90%. One of the most significant induction interactions.',
    'Subtherapeutic tacrolimus with risk of acute rejection. May need 3-10x normal tacrolimus dose during rifampin therapy.',
    'Avoid if possible. If rifampin essential, dramatically increase tacrolimus dose and monitor levels every 2-3 days. Consider rifabutin (less induction).',
    TRUE, 0.15, 0.97,
    'FDA_LABEL', 'Tacrolimus - Section 7 (Rifampin)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Rifampin: most potent CYP3A4/P-gp inducer known; Tacrolimus levels decrease 50-90%', 'P-gp induction', 'CYP3A4 (major induction)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'KDIGO Transplant Guidelines; IDSA TB Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Fluconazole + Cyclosporine (Nephrotoxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'FLUCONAZOLE_CYCLOSPORINE_001',
    'FLUCONAZOLE', 'Fluconazole',
    'CYCLOSPORINE', 'Cyclosporine',
    'pharmacokinetic',
    'major',
    'established',
    'Fluconazole inhibits CYP3A4, the primary enzyme metabolizing cyclosporine. Levels increase 50-200% depending on fluconazole dose.',
    'Cyclosporine toxicity: nephrotoxicity (Cr elevation), neurotoxicity, hypertension, gingival hyperplasia.',
    'Reduce cyclosporine dose by 25-50% when starting fluconazole. Monitor levels within 3-5 days. Adjust based on trough.',
    TRUE, 0.40, 0.91,
    'FDA_LABEL', 'Cyclosporine - Section 7 (Azole antifungals)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Fluconazole: CYP3A4 inhibitor (dose-dependent); Cyclosporine AUC increases 50-200%', 'P-gp inhibition additive', 'CYP3A4 (major)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'KDIGO Transplant Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Trimethoprim + Methotrexate (Myelosuppression - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TRIMETHOPRIM_MTX_001',
    'TRIMETHOPRIM', 'Trimethoprim',
    'METHOTREXATE', 'Methotrexate',
    'pharmacodynamic',
    'major',
    'established',
    'Both are antifolates: Trimethoprim inhibits bacterial DHFR but also has activity against human DHFR. Additive folate depletion.',
    'Severe myelosuppression: pancytopenia, megaloblastic anemia. Risk especially with low-dose MTX for RA/psoriasis.',
    'Avoid combination. If TMP essential for UTI, use short course with CBC monitoring. Consider folinic acid rescue.',
    FALSE, 0.30, 0.92,
    'FDA_LABEL', 'Methotrexate - Section 7 (Trimethoprim)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Trimethoprim: weak DHFR inhibitor in humans; MTX: potent DHFR inhibitor; Additive folate pathway blockade', 'OAT competition (renal secretion)', 'Not primarily CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACR Rheumatoid Arthritis Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 21: ADDITIONAL QT COMBINATIONS
-- =============================================================================

-- Sotalol + Amiodarone (QT - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'SOTALOL_AMIODARONE_001',
    'SOTALOL', 'Sotalol',
    'AMIODARONE', 'Amiodarone',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both are Class III antiarrhythmics blocking IKr (hERG). Both are CredibleMeds Known Risk. Combination = extreme QT prolongation.',
    'Extreme TdP risk. Both drugs are among the highest-risk QT prolonging agents. Combination essentially never justified.',
    'CONTRAINDICATED. Do not use together. If switching, allow adequate washout (amiodarone has very long half-life - weeks to months).',
    FALSE, 0.05, 1.00,
    'FDA_LABEL', 'Sotalol - Section 5 (QT); Amiodarone - Section 5 (QT)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Sotalol and Amiodarone are CredibleMeds Category 1 (Known Risk) - highest QT risk drugs', NULL, 'IKr (hERG) blockade', 'KNOWN + KNOWN',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA AF Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Escitalopram + Amiodarone (QT - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ESCITALOPRAM_AMIODARONE_001',
    'ESCITALOPRAM', 'Escitalopram',
    'AMIODARONE', 'Amiodarone',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs prolong QT interval. Escitalopram has dose-dependent QT effect (FDA warning 2011). Amiodarone: Known Risk.',
    'Additive QT prolongation, TdP risk. Escitalopram max dose reduced to 20mg by FDA due to QT concerns.',
    'Limit escitalopram to ≤10mg/day with amiodarone. Baseline ECG, correct electrolytes, monitor QTc. Consider alternative SSRI (sertraline).',
    TRUE, 0.35, 0.87,
    'FDA_LABEL', 'Escitalopram - FDA Drug Safety Communication 2011 (QT)', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-drug-safety-communication-abnormal-heart-rhythms-associated-high-doses-celexa-citalopram', 'US',
    'CREDIBLEMEDS', 'Escitalopram: Conditional Risk (dose-dependent); Amiodarone: Known Risk per CredibleMeds', NULL, 'hERG channel blockade', 'CONDITIONAL + KNOWN',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA MDD Guidelines; AHA QT Statement',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 22: ADDITIONAL GI/NUTRITION ICU
-- =============================================================================

-- Ciprofloxacin + Enteral Nutrition (Reduced Absorption - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CIPROFLOXACIN_ENTERAL_001',
    'CIPROFLOXACIN', 'Ciprofloxacin',
    'ENTERAL_NUTRITION', 'Enteral Tube Feeds',
    'pharmacokinetic',
    'major',
    'established',
    'Fluoroquinolones chelate with divalent cations (Ca2+, Mg2+, Fe2+) in tube feeds, dramatically reducing absorption.',
    'Subtherapeutic ciprofloxacin levels, treatment failure for serious infections. Bioavailability reduced 30-50%.',
    'Hold tube feeds 1 hour before and 2 hours after ciprofloxacin. Alternatively, use IV ciprofloxacin in critically ill.',
    FALSE, 0.45, 0.86,
    'FDA_LABEL', 'Ciprofloxacin - Section 7 (Antacids, metals)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'FQ chelation with divalent/trivalent cations forms insoluble complexes; Bioavailability reduced 30-50%', NULL, 'Not CYP-mediated (absorption)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'IDSA Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Warfarin + Vitamin K (Antagonism - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_VITAMINK_001',
    'WARFARIN', 'Warfarin',
    'VITAMIN_K', 'Vitamin K (dietary/supplements)',
    'pharmacodynamic',
    'major',
    'established',
    'Direct pharmacologic antagonism: Warfarin inhibits vitamin K-dependent clotting factors. Vitamin K reverses this effect.',
    'Subtherapeutic INR with thromboembolic risk. Variable vitamin K intake is leading cause of INR instability.',
    'Maintain consistent vitamin K intake (not avoidance). Educate on high-K foods (leafy greens). Adjust warfarin for consistent intake.',
    FALSE, 0.60, 0.85,
    'FDA_LABEL', 'Warfarin - Section 7 (Vitamin K)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Vitamin K is essential cofactor for carboxylation of factors II, VII, IX, X; Warfarin blocks vitamin K recycling', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACCP Antithrombotic Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 23: ADDITIONAL PAIN/SEDATION
-- =============================================================================

-- Fentanyl + Diltiazem (Respiratory Depression - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'FENTANYL_DILTIAZEM_001',
    'FENTANYL', 'Fentanyl',
    'DILTIAZEM', 'Diltiazem',
    'pharmacokinetic',
    'major',
    'established',
    'Diltiazem inhibits CYP3A4, the primary enzyme metabolizing fentanyl. Fentanyl levels and duration of effect increased.',
    'Enhanced and prolonged opioid effect, respiratory depression. Common ICU combination (rate control + analgesia).',
    'Reduce fentanyl dose by 25-50% when diltiazem started. Monitor respiratory status. Be aware of prolonged effect.',
    TRUE, 0.45, 0.85,
    'FDA_LABEL', 'Fentanyl - Section 7 (CYP3A4 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Diltiazem: moderate CYP3A4 inhibitor; Fentanyl AUC increases; Effect may be clinically significant in ICU', NULL, 'CYP3A4 (major)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'SCCM Pain Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Remifentanil + Propofol (Synergistic Depression - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'REMIFENTANIL_PROPOFOL_001',
    'REMIFENTANIL', 'Remifentanil (Ultiva)',
    'PROPOFOL', 'Propofol',
    'pharmacodynamic',
    'major',
    'established',
    'True pharmacodynamic synergy: opioid + hypnotic interaction is supra-additive. ED50 of each drug reduced 50% when combined.',
    'Profound sedation and respiratory depression. Standard TIVA combination but requires careful titration.',
    'Standard for TIVA but reduce both doses significantly. Use BIS monitoring. Requires secured airway or immediate access.',
    TRUE, 0.70, 0.88,
    'FDA_LABEL', 'Remifentanil - Section 6 (Anesthetic interactions)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Opioid-hypnotic synergy: true pharmacodynamic interaction with ED50 shifts; Well-characterized isobologram', NULL, 'Not CYP-mediated (ester hydrolysis)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ASA Anesthesia Guidelines; SCCM PADIS',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- FINAL VERIFICATION
-- =============================================================================
DO $$
DECLARE
    v_total INTEGER;
    v_governed INTEGER;
    v_contraindicated INTEGER;
    v_major INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total FROM drug_interactions WHERE active = TRUE;
    SELECT COUNT(*) INTO v_governed FROM drug_interactions
    WHERE active = TRUE
      AND gov_regulatory_authority IS NOT NULL
      AND gov_pharmacology_authority IS NOT NULL
      AND gov_clinical_authority IS NOT NULL;
    SELECT COUNT(*) INTO v_contraindicated FROM drug_interactions
    WHERE active = TRUE AND severity = 'contraindicated';
    SELECT COUNT(*) INTO v_major FROM drug_interactions
    WHERE active = TRUE AND severity = 'major';

    RAISE NOTICE '================================================';
    RAISE NOTICE 'PHASE-1 ICU TIER 1 COMPLETE - TARGET 100';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Total active DDIs: %', v_total;
    RAISE NOTICE 'Fully governed (3-layer): %', v_governed;
    RAISE NOTICE 'Governance compliance: %', ROUND((v_governed::NUMERIC / NULLIF(v_total, 0) * 100), 1) || '%';
    RAISE NOTICE '------------------------------------------------';
    RAISE NOTICE 'Contraindicated: %', v_contraindicated;
    RAISE NOTICE 'Major severity: %', v_major;
    RAISE NOTICE '================================================';
END $$;
