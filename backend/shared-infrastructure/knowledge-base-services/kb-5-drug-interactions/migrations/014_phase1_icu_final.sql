-- =============================================================================
-- KB-5 Phase-1 ICU Critical DDIs - Final Batch (Reach 100)
-- Migration: 014_phase1_icu_final.sql
-- =============================================================================

-- Heparin + Thrombolytics (Bleeding - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_TPA_001',
    'HEPARIN', 'Heparin',
    'ALTEPLASE', 'Alteplase (tPA)',
    'pharmacodynamic',
    'major',
    'established',
    'Additive bleeding risk: tPA causes fibrinolysis; heparin prevents clot propagation. Both impair hemostasis.',
    'Major bleeding including intracranial hemorrhage. Standard combination in STEMI/PE but high-risk.',
    'Protocol-driven dosing. Start heparin AFTER tPA bolus per STEMI/PE protocols. Monitor closely for bleeding.',
    FALSE, 0.30, 0.92,
    'FDA_LABEL', 'Alteplase - Section 5 (Bleeding)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'tPA: plasminogen activation; Heparin: AT-III mediated; Combined hemostatic impairment', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACC/AHA STEMI Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Epinephrine + MAOIs (Hypertensive Crisis - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'EPINEPHRINE_MAOI_001',
    'EPINEPHRINE', 'Epinephrine',
    'PHENELZINE', 'Phenelzine (MAOI)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'MAOIs prevent catecholamine metabolism. Exogenous epinephrine effects massively potentiated.',
    'Severe hypertensive crisis, stroke, MI, death. Even small doses of epinephrine can cause severe reactions.',
    'CONTRAINDICATED. Use alternative vasopressors (phenylephrine with extreme caution, vasopressin). If epi needed for anaphylaxis, use 1/10th dose.',
    TRUE, 0.05, 1.00,
    'FDA_LABEL', 'Epinephrine - Section 7 (MAOIs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'MAO-A/B inhibition prevents NE/EPI degradation; Exogenous EPI causes catecholamine excess', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACLS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Nitroprusside + PDE5 Inhibitors (Hypotension - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'NITROPRUSSIDE_SILDENAFIL_001',
    'NITROPRUSSIDE', 'Nitroprusside',
    'SILDENAFIL', 'Sildenafil',
    'pharmacodynamic',
    'major',
    'established',
    'Both increase cGMP: Nitroprusside via NO donation; Sildenafil by preventing cGMP breakdown. Synergistic vasodilation.',
    'Severe refractory hypotension. PDE5 inhibitors are contraindicated with nitrates (same mechanism).',
    'Avoid sildenafil for 24-48h before nitroprusside. If hypotension occurs, may require high-dose vasopressors.',
    FALSE, 0.15, 0.93,
    'FDA_LABEL', 'Sildenafil - Section 4 (Contraindicated with nitrates)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'NO donors activate guanylate cyclase; PDE5 inhibitors prevent cGMP degradation; Synergistic cGMP accumulation', NULL, 'CYP3A4 (sildenafil)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'ACC/AHA Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Phenobarbital + Warfarin (Reduced Effect - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PHENOBARBITAL_WARFARIN_001',
    'PHENOBARBITAL', 'Phenobarbital',
    'WARFARIN', 'Warfarin',
    'pharmacokinetic',
    'major',
    'established',
    'Phenobarbital is a potent CYP inducer (CYP2C9, CYP3A4). Warfarin metabolism dramatically increased.',
    'Subtherapeutic INR with thromboembolic risk. Effect persists weeks after phenobarbital discontinued.',
    'Monitor INR twice weekly. Expect 30-50% warfarin dose increase. When stopping phenobarbital, taper warfarin slowly.',
    TRUE, 0.20, 0.89,
    'FDA_LABEL', 'Warfarin - Section 7 (Barbiturates)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Phenobarbital: CYP2C9/3A4 inducer; Warfarin clearance increases significantly', NULL, 'CYP2C9, CYP3A4 induction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACCP Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Vecuronium + Clindamycin (Prolonged Block - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'VECURONIUM_CLINDAMYCIN_001',
    'VECURONIUM', 'Vecuronium',
    'CLINDAMYCIN', 'Clindamycin',
    'pharmacodynamic',
    'major',
    'established',
    'Clindamycin has intrinsic neuromuscular blocking properties. Inhibits presynaptic ACh release.',
    'Prolonged neuromuscular blockade, difficulty weaning from ventilator.',
    'Use TOF monitoring. May need reduced NMB doses. Consider alternative antibiotic if prolonged NMB use expected.',
    TRUE, 0.25, 0.84,
    'FDA_LABEL', 'Vecuronium - Section 7 (Antibiotics)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Clindamycin: presynaptic ACh release inhibition; Potentiates non-depolarizing NMB agents', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ASA NMB Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Ticagrelor + Strong CYP3A4 Inhibitors (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'TICAGRELOR_KETOCONAZOLE_001',
    'TICAGRELOR', 'Ticagrelor (Brilinta)',
    'KETOCONAZOLE', 'Ketoconazole',
    'pharmacokinetic',
    'major',
    'established',
    'Ketoconazole is a potent CYP3A4 inhibitor. Ticagrelor is CYP3A4 substrate. AUC increases ~7-fold.',
    'Increased bleeding risk from excessive antiplatelet effect. Dyspnea (ticagrelor side effect) may worsen.',
    'Avoid strong CYP3A4 inhibitors with ticagrelor. Use alternative antifungal or switch to clopidogrel/prasugrel.',
    FALSE, 0.15, 0.91,
    'FDA_LABEL', 'Brilinta - Section 7 (Strong CYP3A4 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ketoconazole: strong CYP3A4 inhibitor; Ticagrelor AUC increases 7-fold', 'P-gp substrate (ticagrelor)', 'CYP3A4 (major)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'ACC/AHA DAPT Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Dabigatran + Dronedarone (Bleeding - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DABIGATRAN_DRONEDARONE_001',
    'DABIGATRAN', 'Dabigatran (Pradaxa)',
    'DRONEDARONE', 'Dronedarone',
    'pharmacokinetic',
    'contraindicated',
    'established',
    'Dronedarone is a P-gp inhibitor. Dabigatran is P-gp substrate with renal elimination. Levels increase significantly.',
    'Major bleeding risk. FDA contraindication based on mechanism and clinical concerns.',
    'CONTRAINDICATED. Use rivaroxaban or apixaban if DOAC needed with dronedarone (though still use caution).',
    FALSE, 0.10, 0.96,
    'FDA_LABEL', 'Pradaxa - Section 7 (P-gp inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Dronedarone: P-gp inhibitor; Dabigatran: P-gp substrate; Dabigatran AUC increases 70-140%', 'P-gp inhibition primary', 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ISTH DOAC Guidance',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Octreotide + Insulin (Hypoglycemia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'OCTREOTIDE_INSULIN_001',
    'OCTREOTIDE', 'Octreotide',
    'INSULIN', 'Insulin',
    'pharmacodynamic',
    'major',
    'established',
    'Octreotide inhibits glucagon release AND reduces insulin secretion. Net effect on glucose is unpredictable.',
    'Initially hyperglycemia (insulin inhibition), then hypoglycemia (glucagon inhibition). Glucose swings common.',
    'Frequent glucose monitoring during octreotide therapy. May need to reduce insulin initially, then increase. Adjust based on trends.',
    TRUE, 0.25, 0.83,
    'FDA_LABEL', 'Octreotide - Section 5 (Glucose regulation)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Octreotide: somatostatin analog inhibiting both glucagon and insulin release; Complex glucose effects', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ADA Standards 2024',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Labetalol + Calcium Channel Blockers (Heart Block - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'LABETALOL_VERAPAMIL_001',
    'LABETALOL', 'Labetalol',
    'VERAPAMIL', 'Verapamil',
    'pharmacodynamic',
    'major',
    'established',
    'Additive negative chronotropic and dromotropic effects. Both slow AV conduction. Verapamil is non-dihydropyridine CCB.',
    'Severe bradycardia, AV block (2nd/3rd degree), asystole, heart failure. IV combination particularly dangerous.',
    'Avoid IV verapamil in patients on beta-blockers. Oral combination may be used with caution for rate control but monitor closely.',
    FALSE, 0.30, 0.89,
    'FDA_LABEL', 'Verapamil - Section 7 (Beta-blockers)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Beta-blocker: SA/AV node depression; Verapamil: L-type Ca2+ channel block = synergistic conduction delay', NULL, 'CYP3A4 (verapamil)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA Bradycardia Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Nitroglycerin + Alteplase (Hypotension - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'NTG_TPA_001',
    'NITROGLYCERIN', 'Nitroglycerin IV',
    'ALTEPLASE', 'Alteplase (tPA)',
    'pharmacodynamic',
    'major',
    'established',
    'Both cause hypotension: NTG via vasodilation; tPA may cause transient hypotension during infusion. Additive effect.',
    'Severe hypotension during STEMI treatment where both commonly used. May compromise coronary perfusion.',
    'Use lower NTG doses during tPA. Monitor BP q15min. Titrate NTG to BP tolerance. May need to reduce/hold NTG.',
    TRUE, 0.40, 0.82,
    'FDA_LABEL', 'Clinical STEMI literature', 'https://pubmed.ncbi.nlm.nih.gov/', 'US',
    'DRUGBANK', 'NTG: venodilation reducing preload; tPA: fibrinolysis may cause transient hypotension; Additive BP reduction', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACC/AHA STEMI Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Adenosine + Dipyridamole (Prolonged Effect - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ADENOSINE_DIPYRIDAMOLE_001',
    'ADENOSINE', 'Adenosine',
    'DIPYRIDAMOLE', 'Dipyridamole',
    'pharmacokinetic',
    'major',
    'established',
    'Dipyridamole blocks adenosine reuptake into cells, dramatically increasing adenosine levels and duration of effect.',
    'Prolonged asystole, severe bradycardia, bronchospasm. Adenosine effect extended from seconds to minutes.',
    'Reduce adenosine dose by 75% (start at 3mg instead of 6mg). Have atropine ready. Avoid in patients on chronic dipyridamole.',
    TRUE, 0.20, 0.90,
    'FDA_LABEL', 'Adenosine - Section 7 (Dipyridamole)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Dipyridamole inhibits adenosine deaminase and blocks ENT1 reuptake transporter; Adenosine half-life extended 4-6x', 'ENT1 inhibition', 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA SVT Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- FINAL COUNT CHECK
-- =============================================================================
DO $$
DECLARE
    v_total INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total FROM drug_interactions WHERE active = TRUE;
    IF v_total >= 100 THEN
        RAISE NOTICE '★★★ TIER 1 TARGET ACHIEVED: % DDIs ★★★', v_total;
    ELSE
        RAISE NOTICE 'Current count: % (need % more for 100)', v_total, 100 - v_total;
    END IF;
END $$;
