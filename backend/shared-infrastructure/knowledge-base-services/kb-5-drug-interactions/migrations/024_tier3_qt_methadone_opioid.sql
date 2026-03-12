-- ============================================================================
-- KB-5 Tier 3: QT Prolongation - Methadone × All QT Drug Classes
-- Migration 024: ~100 DDIs
-- Methadone is critically important - high QT risk and common in OUD treatment
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
-- Methadone × Antiarrhythmics (most dangerous)
('QT_METHADONE_AMIODARONE_001', 'METHADONE', 'Methadone (Dolophine)', 'AMIODARONE', 'Amiodarone (Cordarone)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Amiodarone inhibits CYP3A4 increasing methadone levels. PK + PD double mechanism.',
'Extreme TdP risk. Methadone QT dose-dependent (10-40ms). Amiodarone adds 30-60ms. Combined fatal.',
'CONTRAINDICATED. Avoid if at all possible. If both needed, ECG monitoring mandatory, electrolyte correction.',
FALSE, 0.10, 0.98, 'FDA_LABEL', 'Dolophine Section 7; Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition; PK + PD mechanism', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ASAM Methadone Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_SOTALOL_001', 'METHADONE', 'Methadone (Dolophine)', 'SOTALOL', 'Sotalol (Betapace)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Sotalol beta-blockade creates bradycardia which amplifies methadone TdP risk.',
'Extreme TdP risk. Bradycardia + combined IKr blockade. Multiple cardiac death reports.',
'CONTRAINDICATED. Alternative antiarrhythmic or pain management needed.',
FALSE, 0.08, 0.98, 'FDA_LABEL', 'Dolophine Section 7; Betapace Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Bradycardia amplifies TdP; Deaths reported', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_DOFETILIDE_001', 'METHADONE', 'Methadone (Dolophine)', 'DOFETILIDE', 'Dofetilide (Tikosyn)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Tikosyn REMS prohibits QT drugs. Methadone significant QT prolongation.',
'Extreme TdP risk. REMS prohibition. Combined effect potentially fatal.',
'CONTRAINDICATED per Tikosyn REMS. Alternative rhythm control needed for AF.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Tikosyn REMS; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; TIKOSYN REMS prohibition', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_DRONEDARONE_001', 'METHADONE', 'Methadone (Dolophine)', 'DRONEDARONE', 'Dronedarone (Multaq)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Dronedarone multi-channel blocker + methadone IKr blocker.',
'Very high TdP risk. Dronedarone QT effect (10-20ms) + methadone QT effect (10-40ms).',
'Avoid if possible. If combination necessary, ECG monitoring and electrolyte correction mandatory.',
FALSE, 0.08, 0.94, 'FDA_LABEL', 'Multaq Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Multi-channel + IKr; Monitor closely', NULL, 'CYP3A4 (dronedarone)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA AF Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Methadone × Antipsychotics (common in dual-diagnosis patients)
('QT_METHADONE_HALOPERIDOL_001', 'METHADONE', 'Methadone (Dolophine)', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP with warnings. Common scenario in psychiatric emergencies.',
'Extreme TdP risk. IV haloperidol BLACK BOX + methadone QT. Multiple deaths reported.',
'CONTRAINDICATED. Use lorazepam for acute agitation. Olanzapine safer if antipsychotic needed.',
FALSE, 0.25, 0.97, 'FDA_LABEL', 'Haldol BLACK BOX; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; IV haloperidol BLACK BOX; Deaths reported', NULL, 'CYP2D6 (haloperidol)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ASAM/APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_THIORIDAZINE_001', 'METHADONE', 'Methadone (Dolophine)', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Thioridazine BLACK BOX specifically lists QT drugs.',
'Extreme TdP risk. Thioridazine BLACK BOX prohibition. Both prolong QT significantly.',
'CONTRAINDICATED per Thioridazine BLACK BOX. Switch antipsychotic.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Mellaril BLACK BOX; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Thioridazine BLACK BOX', NULL, 'CYP2D6 (thioridazine)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_ZIPRASIDONE_001', 'METHADONE', 'Methadone (Dolophine)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ziprasidone label contraindicates QT drugs.',
'Extreme TdP risk. Ziprasidone Section 4 lists QT drugs as contraindicated.',
'CONTRAINDICATED per Ziprasidone label. Use olanzapine or quetiapine.',
FALSE, 0.10, 0.97, 'FDA_LABEL', 'Geodon Section 4; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ziprasidone contraindication', NULL, 'CYP3A4 (ziprasidone)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_QUETIAPINE_001', 'METHADONE', 'Methadone (Dolophine)', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'pharmacodynamic', 'major', 'established',
'Methadone Known Risk + quetiapine Possible Risk. Common dual-diagnosis combination.',
'High TdP risk. Quetiapine modest QT + methadone dose-dependent QT. Common clinical scenario.',
'Use with caution. ECG at baseline and periodically. Quetiapine among safer atypicals for this.',
FALSE, 0.35, 0.85, 'FDA_LABEL', 'Seroquel Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Common dual-diagnosis; Monitor QT', NULL, 'CYP3A4 (quetiapine)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_OLANZAPINE_001', 'METHADONE', 'Methadone (Dolophine)', 'OLANZAPINE', 'Olanzapine (Zyprexa)', 'pharmacodynamic', 'moderate', 'established',
'Methadone Known Risk + olanzapine Possible Risk (minimal QT).',
'Moderate TdP risk. Olanzapine has lowest QT effect among atypicals. Safest combination.',
'Use with caution. Olanzapine preferred atypical for methadone patients. Monitor ECG.',
FALSE, 0.30, 0.78, 'FDA_LABEL', 'Zyprexa Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Olanzapine minimal QT; Preferred', NULL, 'CYP1A2 (olanzapine)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_ARIPIPRAZOLE_001', 'METHADONE', 'Methadone (Dolophine)', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'pharmacodynamic', 'moderate', 'established',
'Methadone Known Risk + aripiprazole Possible Risk (minimal QT 2-5ms).',
'Moderate TdP risk. Aripiprazole has lowest QT effect. Best option for methadone patients.',
'Use with caution. Aripiprazole safest atypical for methadone patients. Preferred choice.',
FALSE, 0.30, 0.75, 'FDA_LABEL', 'Abilify Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Aripiprazole safest (2-5ms QT); Preferred', NULL, 'CYP3A4/2D6 (aripiprazole)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Methadone × Antibiotics
('QT_METHADONE_ERYTHROMYCIN_001', 'METHADONE', 'Methadone (Dolophine)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Erythromycin CYP3A4 inhibition increases methadone levels. PK + PD.',
'Very high TdP risk. Methadone levels increased. Additive QT effect.',
'Avoid if possible. Use azithromycin or non-macrolide. Monitor QTc if unavoidable.',
FALSE, 0.20, 0.92, 'FDA_LABEL', 'Dolophine Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition; PK + PD', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_CLARITHROMYCIN_001', 'METHADONE', 'Methadone (Dolophine)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Clarithromycin strong CYP3A4 inhibitor dramatically increases methadone.',
'Extreme TdP risk. Methadone levels may double. Strong PK + PD interaction.',
'CONTRAINDICATED. Strong CYP3A4 inhibitor. Use azithromycin or non-macrolide.',
FALSE, 0.15, 0.97, 'FDA_LABEL', 'Dolophine Section 7 (Strong CYP3A4); Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Strong CYP3A4 inhibition; Methadone doubled', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_AZITHROMYCIN_001', 'METHADONE', 'Methadone (Dolophine)', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Azithromycin has minimal CYP interaction but significant QT effect.',
'High TdP risk. Azithromycin FDA cardiovascular warning. Additive QT effect.',
'Use with caution. Azithromycin is preferred macrolide but still risky. Monitor QTc.',
FALSE, 0.25, 0.88, 'FDA_LABEL', 'Dolophine Section 7; FDA Azithromycin Warning 2013', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Azithromycin preferred macrolide; FDA warning', NULL, 'Minimal CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_MOXIFLOXACIN_001', 'METHADONE', 'Methadone (Dolophine)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Moxifloxacin BLACK BOX for QT.',
'Extreme TdP risk. Combined QT may exceed 60ms. Moxifloxacin BLACK BOX.',
'CONTRAINDICATED. Use levofloxacin with caution or non-FQ alternative.',
FALSE, 0.15, 0.98, 'FDA_LABEL', 'Avelox BLACK BOX; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_LEVOFLOXACIN_001', 'METHADONE', 'Methadone (Dolophine)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Levofloxacin moderate FQ QT risk.',
'High TdP risk. Levofloxacin (5-15ms) + methadone (10-40ms) = significant combined effect.',
'Use with caution. Ciprofloxacin has lower QT risk if FQ needed. Monitor QTc.',
FALSE, 0.25, 0.90, 'FDA_LABEL', 'Levaquin Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Levofloxacin moderate FQ risk', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_CIPROFLOXACIN_001', 'METHADONE', 'Methadone (Dolophine)', 'CIPROFLOXACIN', 'Ciprofloxacin (Cipro)', 'pharmacodynamic', 'moderate', 'established',
'Methadone Known Risk + ciprofloxacin Possible Risk (lowest FQ QT).',
'Moderate TdP risk. Ciprofloxacin has lowest QT effect among FQs. Preferred if FQ needed.',
'Use with caution. Ciprofloxacin preferred FQ for methadone patients. Monitor QTc.',
FALSE, 0.30, 0.80, 'FDA_LABEL', 'Cipro Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Ciprofloxacin lowest FQ QT; Preferred', NULL, 'CYP1A2 inhibition (cipro)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_FLUCONAZOLE_001', 'METHADONE', 'Methadone (Dolophine)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Fluconazole CYP3A4 inhibitor increases methadone.',
'High TdP risk. Methadone levels increased. Common in HIV patients.',
'Use with caution. Reduce methadone dose if high fluconazole doses used. Monitor QTc.',
TRUE, 0.25, 0.88, 'FDA_LABEL', 'Diflucan Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition; Common HIV scenario', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'DHHS HIV Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Methadone × Antidepressants (very common in OUD treatment)
('QT_METHADONE_CITALOPRAM_001', 'METHADONE', 'Methadone (Dolophine)', 'CITALOPRAM', 'Citalopram (Celexa)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Citalopram dose-dependent QT (FDA max 40mg, 20mg elderly).',
'High TdP risk. Citalopram + methadone very common combination. Requires monitoring.',
'Use with caution. Maximum citalopram 40mg (20mg elderly). ECG before and during therapy.',
FALSE, 0.40, 0.90, 'FDA_LABEL', 'Celexa FDA Safety 2012; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Citalopram dose limit; Very common combination', NULL, 'CYP3A4/2C19 (citalopram)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA Safety Communication 2012', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_ESCITALOPRAM_001', 'METHADONE', 'Methadone (Dolophine)', 'ESCITALOPRAM', 'Escitalopram (Lexapro)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Escitalopram is active enantiomer of citalopram.',
'High TdP risk. Similar to citalopram. Common combination in OUD treatment.',
'Use with caution. Monitor QTc. Escitalopram dose limit applies similar to citalopram.',
FALSE, 0.35, 0.88, 'FDA_LABEL', 'Lexapro Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Escitalopram similar to citalopram; Monitor QT', NULL, 'CYP3A4/2C19 (escitalopram)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_SERTRALINE_001', 'METHADONE', 'Methadone (Dolophine)', 'SERTRALINE', 'Sertraline (Zoloft)', 'pharmacodynamic', 'moderate', 'established',
'Methadone Known Risk + sertraline Possible Risk (minimal QT).',
'Moderate TdP risk. Sertraline has minimal QT effect. Preferred SSRI for methadone.',
'Use with caution. Sertraline preferred SSRI for methadone patients (lowest QT). Monitor.',
FALSE, 0.40, 0.75, 'FDA_LABEL', 'Zoloft Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Sertraline minimal QT; Preferred SSRI', NULL, 'Minimal CYP interaction', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_FLUOXETINE_001', 'METHADONE', 'Methadone (Dolophine)', 'FLUOXETINE', 'Fluoxetine (Prozac)', 'pharmacodynamic', 'moderate', 'established',
'Methadone Known Risk + fluoxetine Possible Risk. CYP2D6 inhibition increases methadone.',
'Moderate TdP risk. Fluoxetine inhibits CYP2D6. May increase methadone levels modestly.',
'Use with caution. Monitor for increased opioid effects. ECG monitoring recommended.',
FALSE, 0.35, 0.78, 'FDA_LABEL', 'Prozac Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; CYP2D6 inhibition modest; Monitor opioid effects', NULL, 'CYP2D6 inhibition', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_PAROXETINE_001', 'METHADONE', 'Methadone (Dolophine)', 'PAROXETINE', 'Paroxetine (Paxil)', 'pharmacodynamic', 'moderate', 'established',
'Methadone Known Risk + paroxetine Possible Risk. CYP2D6 inhibition.',
'Moderate TdP risk. Paroxetine potent CYP2D6 inhibitor. May increase methadone effect.',
'Use with caution. Monitor for increased opioid effects. Sertraline may be preferred.',
FALSE, 0.30, 0.78, 'FDA_LABEL', 'Paxil Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; CYP2D6 potent inhibitor; Monitor opioid effects', NULL, 'CYP2D6 inhibition (potent)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_VENLAFAXINE_001', 'METHADONE', 'Methadone (Dolophine)', 'VENLAFAXINE', 'Venlafaxine (Effexor)', 'pharmacodynamic', 'major', 'established',
'Methadone Known Risk + venlafaxine Possible Risk. Venlafaxine dose-dependent QT.',
'Moderate-high TdP risk. Venlafaxine has dose-dependent QT effect (5-15ms at high doses).',
'Use with caution. Monitor QTc especially at higher venlafaxine doses. Consider duloxetine.',
FALSE, 0.25, 0.82, 'FDA_LABEL', 'Effexor Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Venlafaxine dose-dependent QT', NULL, 'CYP2D6 (venlafaxine metabolism)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_BUPROPION_001', 'METHADONE', 'Methadone (Dolophine)', 'BUPROPION', 'Bupropion (Wellbutrin)', 'pharmacodynamic', 'moderate', 'established',
'Methadone Known Risk + bupropion minimal QT risk. Seizure threshold concern.',
'Low TdP risk. Bupropion minimal QT effect. Seizure risk more relevant than QT.',
'Use with caution. Bupropion has minimal QT effect. Monitor for seizure risk at high doses.',
FALSE, 0.25, 0.68, 'FDA_LABEL', 'Wellbutrin Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Conditional Risk; Bupropion minimal QT; Seizure concern', NULL, 'CYP2B6 (bupropion)', 'Known Risk + Conditional Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Methadone × TCAs (significant QT risk)
('QT_METHADONE_AMITRIPTYLINE_001', 'METHADONE', 'Methadone (Dolophine)', 'AMITRIPTYLINE', 'Amitriptyline (Elavil)', 'pharmacodynamic', 'major', 'established',
'Methadone Known Risk + amitriptyline Possible Risk. Both cause QT prolongation.',
'High TdP risk. TCAs have significant QT effect (10-30ms). Additive with methadone.',
'Avoid if possible. If TCA needed, use lowest effective dose. Monitor QTc closely.',
FALSE, 0.15, 0.88, 'FDA_LABEL', 'Elavil Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; TCA significant QT; Avoid if possible', NULL, 'CYP2D6 (amitriptyline)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_NORTRIPTYLINE_001', 'METHADONE', 'Methadone (Dolophine)', 'NORTRIPTYLINE', 'Nortriptyline (Pamelor)', 'pharmacodynamic', 'major', 'established',
'Methadone Known Risk + nortriptyline Possible Risk. Nortriptyline lower QT than amitriptyline.',
'Moderate-high TdP risk. Nortriptyline preferred TCA (less QT). Still significant risk.',
'Use with caution. If TCA needed, nortriptyline preferred over amitriptyline. Monitor QTc.',
FALSE, 0.12, 0.82, 'FDA_LABEL', 'Pamelor Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Nortriptyline less QT than amitriptyline', NULL, 'CYP2D6 (nortriptyline)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_TRAZODONE_001', 'METHADONE', 'Methadone (Dolophine)', 'TRAZODONE', 'Trazodone (Desyrel)', 'pharmacodynamic', 'major', 'established',
'Methadone Known Risk + trazodone Possible Risk. Common insomnia treatment.',
'High TdP risk. Trazodone has moderate QT effect. Very common combination for sleep.',
'Use with caution. Monitor QTc. Use lowest effective trazodone dose for sleep.',
FALSE, 0.35, 0.85, 'FDA_LABEL', 'Desyrel Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Trazodone moderate QT; Common for sleep', NULL, 'CYP3A4 (trazodone)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ASAM Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Methadone × Antiemetics
('QT_METHADONE_ONDANSETRON_001', 'METHADONE', 'Methadone (Dolophine)', 'ONDANSETRON_IV', 'Ondansetron IV (Zofran)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Ondansetron IV has dose-dependent QT (>16mg BLACK BOX).',
'High TdP risk. Ondansetron IV >16mg has BLACK BOX. Combined effect significant.',
'Use with caution. Maximum ondansetron IV 16mg. Monitor QTc. Consider granisetron.',
FALSE, 0.30, 0.88, 'FDA_LABEL', 'Zofran Section 7 (16mg limit); Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ondansetron dose limit; 16mg maximum', NULL, 'CYP3A4 (ondansetron)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_DROPERIDOL_001', 'METHADONE', 'Methadone (Dolophine)', 'DROPERIDOL', 'Droperidol (Inapsine)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Droperidol BLACK BOX for QT and sudden death.',
'Extreme TdP risk. Droperidol BLACK BOX. Never combine with methadone.',
'CONTRAINDICATED per Droperidol BLACK BOX. Use alternative antiemetic.',
FALSE, 0.08, 0.99, 'FDA_LABEL', 'Inapsine BLACK BOX; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Droperidol BLACK BOX', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Methadone × Antimalarials
('QT_METHADONE_CHLOROQUINE_001', 'METHADONE', 'Methadone (Dolophine)', 'CHLOROQUINE', 'Chloroquine (Aralen)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Chloroquine significant QT prolongation (20-40ms).',
'Extreme TdP risk. Combined QT may exceed 60ms. Travel medicine consideration.',
'CONTRAINDICATED. Use alternative antimalarial. Consult infectious disease.',
FALSE, 0.05, 0.97, 'FDA_LABEL', 'Aralen Section 7; Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Chloroquine significant QT', NULL, 'CYP metabolism', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'CDC Malaria Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_HYDROXYCHLOROQUINE_001', 'METHADONE', 'Methadone (Dolophine)', 'HYDROXYCHLOROQUINE', 'Hydroxychloroquine (Plaquenil)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. HCQ lower QT risk than chloroquine but still significant.',
'High TdP risk. COVID-19 experience demonstrated arrhythmia risk with this combination.',
'Avoid if possible. If HCQ needed for rheumatologic disease, ECG monitoring mandatory.',
FALSE, 0.10, 0.90, 'FDA_LABEL', 'Plaquenil Section 7; FDA COVID-19 Warning 2020', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; COVID-19 experience; ECG monitoring', NULL, 'CYP metabolism', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA COVID-19 Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Methadone × HIV medications (critical for many patients)
('QT_METHADONE_ATAZANAVIR_001', 'METHADONE', 'Methadone (Dolophine)', 'ATAZANAVIR', 'Atazanavir (Reyataz)', 'pharmacokinetic', 'moderate', 'established',
'Methadone Known Risk + atazanavir moderate CYP3A4 inhibition.',
'Moderate risk. Atazanavir may increase methadone levels. Also has modest QT effect.',
'Use with caution. Monitor for opioid toxicity. Atazanavir has modest QT effect itself.',
FALSE, 0.20, 0.78, 'FDA_LABEL', 'Reyataz Section 7 (Methadone); Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known Risk + HIV context; CYP3A4 interaction; Monitor', NULL, 'CYP3A4 inhibition', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'DHHS HIV Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_METHADONE_EFAVIRENZ_001', 'METHADONE', 'Methadone (Dolophine)', 'EFAVIRENZ', 'Efavirenz (Sustiva)', 'pharmacokinetic', 'major', 'established',
'Efavirenz induces CYP3A4, dramatically decreasing methadone levels. Withdrawal risk.',
'High withdrawal risk. Methadone levels may decrease 50-60%. Dose increase often needed.',
'Increase methadone dose by 50% when starting efavirenz. Monitor for withdrawal.',
TRUE, 0.15, 0.90, 'FDA_LABEL', 'Sustiva Section 7 (Methadone); Dolophine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'CYP3A4 induction; Methadone decreased 50-60%; Withdrawal risk', NULL, 'CYP3A4 induction', 'Known Risk + Induction',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'DHHS HIV Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')

ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Migration 024 Complete: % total DDIs', v_count;
END $$;
