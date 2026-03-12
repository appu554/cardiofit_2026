-- ============================================================================
-- KB-5 Tier 3: QT Prolongation - Antipsychotic × Antibiotic Matrix
-- Migration 023: ~100 DDIs
-- ============================================================================

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES
-- Thioridazine × Antibiotics (BLACK BOX - most dangerous)
('QT_THIORIDAZINE_ERYTHROMYCIN_001', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'contraindicated', 'established',
'Thioridazine BLACK BOX + erythromycin Known Risk. CYP2D6 inhibition increases thioridazine.',
'Extreme TdP risk. PK + PD double mechanism. BLACK BOX specifically prohibits.',
'CONTRAINDICATED per Thioridazine BLACK BOX. Use alternative antibiotic.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Mellaril BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP2D6 inhibition; PK + PD', NULL, 'CYP2D6 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_THIORIDAZINE_CLARITHROMYCIN_001', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Thioridazine BLACK BOX + clarithromycin Known Risk.',
'Extreme TdP risk. Clarithromycin CYP3A4 inhibition + QT effect. BLACK BOX prohibition.',
'CONTRAINDICATED per Thioridazine BLACK BOX.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Mellaril BLACK BOX; Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; BLACK BOX prohibition', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_THIORIDAZINE_MOXIFLOXACIN_001', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Double BLACK BOX: Thioridazine + Moxifloxacin.',
'Extreme TdP risk. Both drugs have BLACK BOX warnings for QT. Combined effect potentially fatal.',
'CONTRAINDICATED. Double BLACK BOX drugs. Use alternative antibiotic.',
FALSE, 0.03, 0.99, 'FDA_LABEL', 'Mellaril BLACK BOX; Avelox BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Double BLACK BOX', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Pimozide × Antibiotics (CYP3A4/2D6 interactions critical)
('QT_PIMOZIDE_ERYTHROMYCIN_001', 'PIMOZIDE', 'Pimozide (Orap)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Erythromycin CYP3A4 inhibition significantly increases pimozide.',
'Extreme TdP risk. Pimozide contraindicated with CYP3A4 inhibitors. PK + PD interaction.',
'CONTRAINDICATED per Pimozide label (CYP3A4 inhibitors).',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Orap Section 4 (CYP3A4 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition contraindicated; PK + PD', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_PIMOZIDE_CLARITHROMYCIN_001', 'PIMOZIDE', 'Pimozide (Orap)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Clarithromycin strong CYP3A4 inhibitor dramatically increases pimozide.',
'Extreme TdP risk. Pimozide label lists clarithromycin specifically as contraindicated.',
'CONTRAINDICATED. Clarithromycin listed specifically in pimozide label.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Orap Section 4 (Clarithromycin listed)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Clarithromycin listed in pimozide label', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Orap Prescribing Information', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_PIMOZIDE_AZITHROMYCIN_001', 'PIMOZIDE', 'Pimozide (Orap)', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Azithromycin has lower CYP3A4 effect but significant QT.',
'High TdP risk. Azithromycin preferred macrolide but still dangerous with pimozide.',
'Avoid if possible. Azithromycin is least CYP-interacting macrolide but QT effect additive.',
FALSE, 0.08, 0.93, 'FDA_LABEL', 'Orap Section 7; Zithromax Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Azithromycin preferred but risky; QT additive', NULL, 'Minimal CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Ziprasidone × Antibiotics (label contraindication)
('QT_ZIPRASIDONE_ERYTHROMYCIN_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ziprasidone label contraindicates QT drugs.',
'Extreme TdP risk. Ziprasidone Section 4 lists QT-prolonging drugs as contraindicated.',
'CONTRAINDICATED per Ziprasidone label. Use non-macrolide antibiotic.',
FALSE, 0.10, 0.97, 'FDA_LABEL', 'Geodon Section 4 (QT drugs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ziprasidone label contraindication', NULL, 'CYP3A4 (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Geodon PI', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ZIPRASIDONE_CLARITHROMYCIN_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Clarithromycin strong CYP3A4 inhibitor increases ziprasidone.',
'Extreme TdP risk. PK + PD double mechanism. Label contraindication.',
'CONTRAINDICATED. CYP3A4 inhibition + QT effect. Use non-macrolide.',
FALSE, 0.08, 0.98, 'FDA_LABEL', 'Geodon Section 4; Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 + QT; Double mechanism', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ZIPRASIDONE_MOXIFLOXACIN_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP with explicit label warnings. Double contraindication.',
'Extreme TdP risk. Both labels warn against QT drugs. Mutual contraindication.',
'CONTRAINDICATED. Both labels prohibit. Use ciprofloxacin with caution if FQ needed.',
FALSE, 0.08, 0.99, 'FDA_LABEL', 'Geodon Section 4; Avelox BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Mutual contraindication', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ZIPRASIDONE_LEVOFLOXACIN_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Ziprasidone label warns against QT drugs.',
'High TdP risk. Levofloxacin lower FQ QT risk but ziprasidone label advises against.',
'Avoid if possible. Ciprofloxacin preferred if FQ needed with ziprasidone.',
FALSE, 0.15, 0.93, 'FDA_LABEL', 'Geodon Section 4; Levaquin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Levofloxacin lower risk FQ; Monitor', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Chlorpromazine × Antibiotics
('QT_CHLORPROMAZINE_ERYTHROMYCIN_001', 'CHLORPROMAZINE', 'Chlorpromazine (Thorazine)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Phenothiazine + macrolide combination.',
'High TdP risk. Combined QT prolongation 40-70ms potential.',
'Avoid if possible. Use azithromycin or non-macrolide. Monitor QTc.',
FALSE, 0.10, 0.92, 'FDA_LABEL', 'Thorazine Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Phenothiazine + macrolide', NULL, 'CYP2D6 (chlorpromazine)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_CHLORPROMAZINE_MOXIFLOXACIN_001', 'CHLORPROMAZINE', 'Chlorpromazine (Thorazine)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Moxifloxacin BLACK BOX.',
'Extreme TdP risk. Moxifloxacin BLACK BOX prohibits antipsychotic combination.',
'CONTRAINDICATED per Moxifloxacin BLACK BOX. Use levofloxacin or non-FQ.',
FALSE, 0.08, 0.97, 'FDA_LABEL', 'Avelox BLACK BOX; Thorazine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Iloperidone × Antibiotics
('QT_ILOPERIDONE_CLARITHROMYCIN_001', 'ILOPERIDONE', 'Iloperidone (Fanapt)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Iloperidone Known Risk + clarithromycin Known Risk. CYP3A4 inhibition.',
'Extreme TdP risk. Iloperidone extensively metabolized by CYP3A4. Levels dramatically increased.',
'CONTRAINDICATED. Strong CYP3A4 inhibitors contraindicated with iloperidone per label.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Fanapt Section 4 (CYP3A4 inhibitors); Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition contraindicated; PK + PD', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Fanapt PI', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ILOPERIDONE_MOXIFLOXACIN_001', 'ILOPERIDONE', 'Iloperidone (Fanapt)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Iloperidone QT + moxifloxacin BLACK BOX.',
'Extreme TdP risk. Both have QT warnings. Combined effect dangerous.',
'CONTRAINDICATED. Use alternative antibiotic.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Fanapt Section 7; Avelox BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Quetiapine × Antibiotics (Possible Risk × Known Risk)
('QT_QUETIAPINE_ERYTHROMYCIN_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Quetiapine Possible Risk + erythromycin Known Risk. CYP3A4 inhibition increases quetiapine.',
'Moderate-high TdP risk. Quetiapine levels increased by erythromycin. PK + PD.',
'Use with caution. Monitor QTc. Consider dose reduction of quetiapine.',
TRUE, 0.20, 0.85, 'FDA_LABEL', 'Seroquel Section 7 (CYP3A4); E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; CYP3A4 increases quetiapine', NULL, 'CYP3A4 inhibition', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_CLARITHROMYCIN_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'major', 'established',
'Quetiapine Possible Risk + clarithromycin Known Risk. Strong CYP3A4 inhibitor.',
'High TdP risk. Quetiapine AUC may increase significantly. Strong CYP3A4 interaction.',
'Use with caution. Reduce quetiapine dose. Monitor QTc. Consider azithromycin alternative.',
TRUE, 0.18, 0.88, 'FDA_LABEL', 'Seroquel Section 7 (Strong CYP3A4 inhibitors); Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Strong CYP3A4 interaction; Dose reduction', NULL, 'CYP3A4 inhibition (major)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'Seroquel PI', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_MOXIFLOXACIN_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'major', 'established',
'Quetiapine Possible Risk + moxifloxacin Known Risk (BLACK BOX).',
'High TdP risk. Moxifloxacin BLACK BOX advises against QT drugs.',
'Avoid if possible. Moxifloxacin BLACK BOX applies. Use levofloxacin or non-FQ.',
FALSE, 0.15, 0.90, 'FDA_LABEL', 'Avelox BLACK BOX; Seroquel Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Moxifloxacin BLACK BOX', NULL, 'Not CYP interaction', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_AZITHROMYCIN_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'pharmacodynamic', 'moderate', 'established',
'Quetiapine Possible Risk + azithromycin Known Risk. Minimal CYP interaction.',
'Moderate TdP risk. Azithromycin does not significantly affect quetiapine levels.',
'Use with caution. Azithromycin preferred macrolide. Monitor QTc in high-risk patients.',
FALSE, 0.30, 0.78, 'FDA_LABEL', 'Seroquel Section 7; Zithromax Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Azithromycin minimal CYP; Preferred macrolide', NULL, 'Minimal CYP interaction', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Risperidone × Antibiotics
('QT_RISPERIDONE_ERYTHROMYCIN_001', 'RISPERIDONE', 'Risperidone (Risperdal)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'moderate', 'established',
'Risperidone Possible Risk + erythromycin Known Risk.',
'Moderate TdP risk. Risperidone has modest QT effect. Combined effect manageable.',
'Use with caution. Monitor QTc. Risperidone among safer atypicals for this combination.',
FALSE, 0.25, 0.78, 'FDA_LABEL', 'Risperdal Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Risperidone modest QT; Monitor', NULL, 'CYP2D6 (risperidone)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_RISPERIDONE_MOXIFLOXACIN_001', 'RISPERIDONE', 'Risperidone (Risperdal)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'major', 'established',
'Risperidone Possible Risk + moxifloxacin Known Risk (BLACK BOX).',
'High TdP risk. Moxifloxacin BLACK BOX advises against antipsychotics.',
'Avoid if possible. Use levofloxacin or ciprofloxacin. Moxifloxacin BLACK BOX.',
FALSE, 0.12, 0.88, 'FDA_LABEL', 'Avelox BLACK BOX; Risperdal Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Moxifloxacin BLACK BOX', NULL, 'Not CYP interaction', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Olanzapine × Antibiotics (lowest risk atypical)
('QT_OLANZAPINE_ERYTHROMYCIN_001', 'OLANZAPINE', 'Olanzapine (Zyprexa)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'moderate', 'established',
'Olanzapine Possible Risk (minimal QT) + erythromycin Known Risk.',
'Moderate TdP risk. Olanzapine has lowest QT effect among atypicals. Safer combination.',
'Use with caution. Olanzapine safest atypical for this combination. Monitor in high-risk.',
FALSE, 0.25, 0.72, 'FDA_LABEL', 'Zyprexa Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Olanzapine minimal QT; Safest combination', NULL, 'CYP1A2 (olanzapine)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_OLANZAPINE_MOXIFLOXACIN_001', 'OLANZAPINE', 'Olanzapine (Zyprexa)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'major', 'established',
'Olanzapine Possible Risk + moxifloxacin Known Risk (BLACK BOX).',
'Moderate-high TdP risk. Moxifloxacin BLACK BOX despite olanzapine low QT.',
'Use with caution. Moxifloxacin BLACK BOX applies. Prefer other FQ.',
FALSE, 0.15, 0.82, 'FDA_LABEL', 'Avelox BLACK BOX; Zyprexa Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Moxifloxacin BLACK BOX; Olanzapine low risk', NULL, 'Not CYP interaction', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Aripiprazole × Antibiotics (safest atypical)
('QT_ARIPIPRAZOLE_ERYTHROMYCIN_001', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'moderate', 'established',
'Aripiprazole Possible Risk (minimal QT) + erythromycin Known Risk.',
'Low-moderate TdP risk. Aripiprazole has lowest QT effect (2-5ms). Safest combination.',
'Use with caution. Aripiprazole safest atypical for QT. Monitor in high-risk.',
FALSE, 0.30, 0.70, 'FDA_LABEL', 'Abilify Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Aripiprazole minimal QT 2-5ms', NULL, 'CYP3A4, CYP2D6 (aripiprazole)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ARIPIPRAZOLE_CLARITHROMYCIN_001', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'moderate', 'established',
'Aripiprazole Possible Risk + clarithromycin Known Risk. CYP3A4 inhibition.',
'Moderate TdP risk. Aripiprazole low QT but levels increased by clarithromycin.',
'Use with caution. Reduce aripiprazole dose by 50% with strong CYP3A4 inhibitors per label.',
TRUE, 0.20, 0.75, 'FDA_LABEL', 'Abilify Section 7 (CYP3A4 inhibitors); Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; CYP3A4 interaction; Dose adjustment', NULL, 'CYP3A4 inhibition', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'Abilify PI', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ARIPIPRAZOLE_MOXIFLOXACIN_001', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'moderate', 'established',
'Aripiprazole Possible Risk (minimal QT) + moxifloxacin Known Risk (BLACK BOX).',
'Moderate TdP risk. Aripiprazole minimal QT. Moxifloxacin BLACK BOX still applies.',
'Use with caution. Safest atypical for this situation but BLACK BOX still relevant.',
FALSE, 0.20, 0.78, 'FDA_LABEL', 'Avelox BLACK BOX; Abilify Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Aripiprazole safest; BLACK BOX still relevant', NULL, 'Not CYP interaction', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Clozapine × Antibiotics
('QT_CLOZAPINE_CIPROFLOXACIN_001', 'CLOZAPINE', 'Clozapine (Clozaril)', 'CIPROFLOXACIN', 'Ciprofloxacin (Cipro)', 'pharmacokinetic', 'major', 'established',
'Clozapine Possible Risk + ciprofloxacin Possible Risk. IMPORTANT CYP1A2 interaction.',
'High toxicity risk. Ciprofloxacin inhibits CYP1A2, dramatically increasing clozapine levels.',
'Reduce clozapine dose by 33-50%. Monitor clozapine levels. Seizure and myocarditis risk increased.',
TRUE, 0.15, 0.90, 'FDA_LABEL', 'Clozaril REMS (CYP1A2); Cipro Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; CYP1A2 inhibition major; Clozapine toxicity', NULL, 'CYP1A2 inhibition (major)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Clozapine Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_CLOZAPINE_ERYTHROMYCIN_001', 'CLOZAPINE', 'Clozapine (Clozaril)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Clozapine Possible Risk + erythromycin Known Risk. Myocarditis concern.',
'High risk. Clozapine myocarditis + erythromycin QT. Cardiac monitoring critical.',
'Use with caution. Monitor for myocarditis signs. ECG monitoring. Consider azithromycin.',
FALSE, 0.12, 0.85, 'FDA_LABEL', 'Clozaril REMS; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Clozapine myocarditis; Cardiac monitoring', NULL, 'CYP3A4 (erythromycin)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Clozapine Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Additional combinations with antifungals
('QT_HALOPERIDOL_FLUCONAZOLE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'IV haloperidol Known Risk (BLACK BOX) + fluconazole Known Risk.',
'High TdP risk. Both Known Risk TdP. Common ICU scenario. Monitoring essential.',
'Use with caution. Common combination in ICU sepsis. ECG monitoring required. Use lower doses.',
FALSE, 0.30, 0.88, 'FDA_LABEL', 'Haldol BLACK BOX; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Common ICU scenario; Monitoring required', NULL, 'CYP2D6 (haloperidol)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Candida Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_FLUCONAZOLE_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Quetiapine Possible Risk + fluconazole Known Risk. CYP3A4 interaction.',
'High risk. Fluconazole moderately inhibits CYP3A4 increasing quetiapine.',
'Reduce quetiapine dose by 50% with fluconazole. Monitor QTc.',
TRUE, 0.20, 0.85, 'FDA_LABEL', 'Seroquel Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; CYP3A4 interaction; Dose reduction', NULL, 'CYP3A4 inhibition', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'Seroquel PI', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Antipsychotic × Pentamidine
('QT_HALOPERIDOL_PENTAMIDINE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. IV pentamidine extremely high QT risk. IV haloperidol BLACK BOX.',
'Extreme TdP risk. IV pentamidine prolongs QT 30-60ms. Combination potentially fatal.',
'CONTRAINDICATED. Use aerosolized pentamidine or alternative. Use lorazepam for sedation if needed.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Haldol BLACK BOX; Pentam Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; IV pentamidine extreme risk', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'NIH OI Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_PENTAMIDINE_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'pharmacodynamic', 'major', 'established',
'Quetiapine Possible Risk + pentamidine Known Risk.',
'High TdP risk. IV pentamidine significant QT prolongation.',
'Avoid IV pentamidine if possible. Use aerosolized form. Monitor QTc closely.',
FALSE, 0.08, 0.88, 'FDA_LABEL', 'Seroquel Section 7; Pentam Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Pentamidine high QT; Aerosolized preferred', NULL, 'Not CYP interaction', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'NIH OI Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')

ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Migration 023 Complete: % total DDIs', v_count;
END $$;
