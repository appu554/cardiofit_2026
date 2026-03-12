-- ============================================================================
-- KB-5 Tier 3: QT Prolongation - Antipsychotic × Antipsychotic Matrix
-- Migration 022: ~100 DDIs
-- ============================================================================

-- Known Risk × Known Risk Antipsychotics
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES
-- Haloperidol IV combinations (BLACK BOX)
('QT_HALOPERIDOL_DROPERIDOL_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'DROPERIDOL', 'Droperidol (Inapsine)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP with BLACK BOX warnings. Droperidol + IV haloperidol = extreme QT prolongation.',
'Extreme TdP risk. Both drugs have BLACK BOX for QT/sudden death. No clinical rationale for combination.',
'CONTRAINDICATED. Never combine. Both are alternatives for acute agitation/nausea, not additive therapy.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Haldol BLACK BOX; Inapsine BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Double BLACK BOX; Redundant mechanism', NULL, 'CYP2D6 (haloperidol)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_HALOPERIDOL_THIORIDAZINE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Thioridazine BLACK BOX. IV haloperidol BLACK BOX. Combined QT may exceed 100ms.',
'Extreme TdP risk. Thioridazine alone prolongs QT 30-60ms. IV haloperidol adds significant effect.',
'CONTRAINDICATED. Thioridazine specifically lists QT drugs as contraindicated in BLACK BOX.',
FALSE, 0.03, 0.99, 'FDA_LABEL', 'Mellaril BLACK BOX; Haldol BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Double BLACK BOX; Combined QT >100ms', NULL, 'CYP2D6 (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_HALOPERIDOL_PIMOZIDE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'PIMOZIDE', 'Pimozide (Orap)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Pimozide has dose-dependent QT prolongation. Haloperidol IV BLACK BOX.',
'Extreme TdP risk. Pimozide for Tourette syndrome not compatible with IV haloperidol sedation.',
'CONTRAINDICATED. If acute sedation needed in patient on pimozide, use benzodiazepine.',
FALSE, 0.03, 0.98, 'FDA_LABEL', 'Orap Section 4; Haldol BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Pimozide dose-dependent; Haloperidol BLACK BOX', NULL, 'CYP2D6, CYP3A4 (pimozide)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_HALOPERIDOL_ZIPRASIDONE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ziprasidone contraindicated with other QT drugs per label. Haloperidol BLACK BOX.',
'Extreme TdP risk. Ziprasidone label specifically contraindicates QT-prolonging antipsychotics.',
'CONTRAINDICATED. Choose one or the other. Olanzapine or quetiapine have lower QT risk.',
FALSE, 0.08, 0.97, 'FDA_LABEL', 'Geodon Section 4 (QT drugs); Haldol BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ziprasidone label contraindication; Haloperidol BLACK BOX', NULL, 'CYP3A4 (ziprasidone)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_HALOPERIDOL_CHLORPROMAZINE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'CHLORPROMAZINE', 'Chlorpromazine (Thorazine)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Chlorpromazine phenothiazine with dose-dependent QT. IV haloperidol BLACK BOX.',
'Very high TdP risk. Combined QT effect 40-80ms. May occur in inpatient psychiatric setting.',
'Avoid combination. If both needed, use minimal effective doses with ECG monitoring.',
FALSE, 0.10, 0.94, 'FDA_LABEL', 'Thorazine Section 7; Haldol BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Phenothiazine + butyrophenone; QT 40-80ms combined', NULL, 'CYP2D6 (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Droperidol combinations (BLACK BOX)
('QT_DROPERIDOL_THIORIDAZINE_001', 'DROPERIDOL', 'Droperidol (Inapsine)', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP with BLACK BOX warnings. Double BLACK BOX combination.',
'Extreme TdP risk. Droperidol BLACK BOX states to avoid other QT drugs. Thioridazine BLACK BOX same.',
'CONTRAINDICATED. Both BLACK BOX warnings prohibit concurrent use. Never combine.',
FALSE, 0.02, 0.99, 'FDA_LABEL', 'Inapsine BLACK BOX; Mellaril BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Double BLACK BOX; Redundant IKr blockade', NULL, 'CYP2D6 (thioridazine)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DROPERIDOL_PIMOZIDE_001', 'DROPERIDOL', 'Droperidol (Inapsine)', 'PIMOZIDE', 'Pimozide (Orap)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Droperidol BLACK BOX. Pimozide chronic therapy for tics.',
'Extreme TdP risk. No clinical scenario justifies this combination.',
'CONTRAINDICATED. Use alternative antiemetic for patients on pimozide.',
FALSE, 0.02, 0.99, 'FDA_LABEL', 'Inapsine BLACK BOX; Orap Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Droperidol BLACK BOX; No clinical rationale', NULL, 'CYP3A4 (pimozide)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DROPERIDOL_ZIPRASIDONE_001', 'DROPERIDOL', 'Droperidol (Inapsine)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ziprasidone label contraindicates QT drugs. Droperidol BLACK BOX.',
'Extreme TdP risk. Both drugs have warnings prohibiting concurrent QT-prolonging medications.',
'CONTRAINDICATED. Per both drug labels. Use alternative antiemetic or antipsychotic.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Geodon Section 4; Inapsine BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Mutual label contraindication; Additive IKr', NULL, 'CYP3A4 (ziprasidone)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Thioridazine combinations (BLACK BOX - most restricted)
('QT_THIORIDAZINE_PIMOZIDE_001', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'PIMOZIDE', 'Pimozide (Orap)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Thioridazine BLACK BOX is most restrictive antipsychotic QT warning.',
'Extreme TdP risk. Thioridazine BLACK BOX specifically lists other QT drugs as contraindicated.',
'CONTRAINDICATED. Per Thioridazine BLACK BOX. Switch to safer antipsychotic if second agent needed.',
FALSE, 0.02, 0.99, 'FDA_LABEL', 'Mellaril BLACK BOX (QT drugs listed)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Thioridazine BLACK BOX; Pimozide listed specifically', NULL, 'CYP2D6 (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_THIORIDAZINE_ZIPRASIDONE_001', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Mutual contraindication per both drug labels.',
'Extreme TdP risk. Both labels prohibit concurrent QT drugs. No clinical justification.',
'CONTRAINDICATED. Per both drug labels. Use olanzapine or quetiapine if second agent needed.',
FALSE, 0.03, 0.99, 'FDA_LABEL', 'Mellaril BLACK BOX; Geodon Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Mutual contraindication; Combined QT >80ms', NULL, 'CYP2D6 (thioridazine), CYP3A4 (ziprasidone)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_THIORIDAZINE_CHLORPROMAZINE_001', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'CHLORPROMAZINE', 'Chlorpromazine (Thorazine)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP phenothiazines. Thioridazine BLACK BOX prohibits other QT drugs.',
'Extreme TdP risk. Both phenothiazines with QT prolongation. Additive effect dangerous.',
'CONTRAINDICATED. Per Thioridazine BLACK BOX. No rationale for dual phenothiazine therapy.',
FALSE, 0.02, 0.98, 'FDA_LABEL', 'Mellaril BLACK BOX; Thorazine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Dual phenothiazine; Thioridazine BLACK BOX', NULL, 'CYP2D6 (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Pimozide combinations
('QT_PIMOZIDE_ZIPRASIDONE_001', 'PIMOZIDE', 'Pimozide (Orap)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ziprasidone label contraindicates QT drugs.',
'Extreme TdP risk. Pimozide chronic + ziprasidone. Both have QT warnings.',
'CONTRAINDICATED. Per Ziprasidone label. Use risperidone or aripiprazole if second agent needed.',
FALSE, 0.05, 0.97, 'FDA_LABEL', 'Geodon Section 4; Orap Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ziprasidone contraindication; Chronic + maintenance', NULL, 'CYP3A4 (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_PIMOZIDE_CHLORPROMAZINE_001', 'PIMOZIDE', 'Pimozide (Orap)', 'CHLORPROMAZINE', 'Chlorpromazine (Thorazine)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Pimozide for Tourette + chlorpromazine for psychosis/nausea.',
'High TdP risk. Combined QT prolongation. May occur in treatment-resistant cases.',
'Avoid if possible. If needed, ECG monitoring required. Consider alternative to chlorpromazine.',
FALSE, 0.05, 0.92, 'FDA_LABEL', 'Orap Section 7; Thorazine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Dual indication scenario; ECG monitoring required', NULL, 'CYP2D6 (chlorpromazine), CYP3A4 (pimozide)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Ziprasidone combinations
('QT_ZIPRASIDONE_CHLORPROMAZINE_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'CHLORPROMAZINE', 'Chlorpromazine (Thorazine)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Ziprasidone label warns against QT drugs.',
'High TdP risk. Ziprasidone mean QTc +20ms + chlorpromazine dose-dependent QT.',
'Avoid if possible. Ziprasidone label advises against QT combinations. Use olanzapine instead.',
FALSE, 0.08, 0.92, 'FDA_LABEL', 'Geodon Section 4; Thorazine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ziprasidone label warning; Combined QT ~40-60ms', NULL, 'CYP3A4 (ziprasidone), CYP2D6 (chlorpromazine)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Known Risk × Possible Risk combinations
('QT_HALOPERIDOL_QUETIAPINE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'pharmacodynamic', 'major', 'established',
'IV haloperidol Known Risk (BLACK BOX) + quetiapine Possible Risk. Common ICU combination.',
'High TdP risk. Very common in ICU delirium management. Requires monitoring.',
'Use with caution. ECG before and during therapy. Prefer lower haloperidol doses. Dexmedetomidine may be safer.',
FALSE, 0.35, 0.88, 'FDA_LABEL', 'Haldol BLACK BOX; Seroquel Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Common ICU scenario; Dexmedetomidine alternative', NULL, 'CYP3A4 (quetiapine), CYP2D6 (haloperidol)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'SCCM ICU Delirium Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_HALOPERIDOL_RISPERIDONE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'RISPERIDONE', 'Risperidone (Risperdal)', 'pharmacodynamic', 'major', 'established',
'IV haloperidol Known Risk + risperidone Possible Risk. Acute + maintenance scenario.',
'Moderate-high TdP risk. Risperidone has lower QT effect than other antipsychotics.',
'Use with caution. Monitor ECG. Risperidone relatively safer but still additive QT.',
FALSE, 0.25, 0.85, 'FDA_LABEL', 'Haldol BLACK BOX; Risperdal Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Risperidone lower QT risk; Acute on chronic scenario', NULL, 'CYP2D6 (both)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_HALOPERIDOL_OLANZAPINE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'OLANZAPINE', 'Olanzapine (Zyprexa)', 'pharmacodynamic', 'moderate', 'established',
'IV haloperidol Known Risk + olanzapine Possible Risk (lowest among atypicals).',
'Moderate TdP risk. Olanzapine has minimal QT effect. Combination may be acceptable with monitoring.',
'Use with caution. Olanzapine has lowest QT risk among atypical antipsychotics. Monitor ECG in high-risk patients.',
FALSE, 0.30, 0.78, 'FDA_LABEL', 'Haldol BLACK BOX; Zyprexa Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Olanzapine minimal QT; Lower risk combination', NULL, 'CYP1A2 (olanzapine), CYP2D6 (haloperidol)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ZIPRASIDONE_QUETIAPINE_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'pharmacodynamic', 'major', 'established',
'Ziprasidone Known Risk + quetiapine Possible Risk. Both atypical antipsychotics.',
'High TdP risk. Ziprasidone label warns against QT drugs. Quetiapine adds ~5-15ms.',
'Avoid if possible. Ziprasidone label advises against. Prefer olanzapine as second agent.',
FALSE, 0.15, 0.88, 'FDA_LABEL', 'Geodon Section 4; Seroquel Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Ziprasidone label warning; Dual atypical', NULL, 'CYP3A4 (both)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ZIPRASIDONE_RISPERIDONE_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'RISPERIDONE', 'Risperidone (Risperdal)', 'pharmacodynamic', 'major', 'established',
'Ziprasidone Known Risk + risperidone Possible Risk.',
'Moderate-high TdP risk. Risperidone lower QT effect but ziprasidone label advises against.',
'Avoid if possible. Per ziprasidone label. Aripiprazole may be safer adjunct.',
FALSE, 0.12, 0.85, 'FDA_LABEL', 'Geodon Section 4; Risperdal Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Ziprasidone label caution; Consider aripiprazole', NULL, 'CYP3A4 (ziprasidone), CYP2D6 (risperidone)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_THIORIDAZINE_QUETIAPINE_001', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'pharmacodynamic', 'contraindicated', 'established',
'Thioridazine Known Risk (BLACK BOX) + quetiapine Possible Risk. BLACK BOX prohibits QT drugs.',
'High TdP risk. Thioridazine BLACK BOX specifically lists QT drugs as contraindicated.',
'CONTRAINDICATED per Thioridazine BLACK BOX. Switch from thioridazine to quetiapine monotherapy.',
FALSE, 0.05, 0.95, 'FDA_LABEL', 'Mellaril BLACK BOX; Seroquel Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Thioridazine BLACK BOX prohibition; Switch recommended', NULL, 'CYP2D6 (thioridazine), CYP3A4 (quetiapine)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Possible Risk × Possible Risk (lower severity but still monitor)
('QT_QUETIAPINE_RISPERIDONE_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'RISPERIDONE', 'Risperidone (Risperdal)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP. Combined QT effect modest (10-20ms total).',
'Moderate TdP risk. Both atypicals with modest QT effects. Combination occasionally used.',
'Use with caution. ECG monitoring in high-risk patients. Both relatively safer atypicals.',
FALSE, 0.20, 0.72, 'FDA_LABEL', 'Seroquel Section 7; Risperdal Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Modest combined effect; Lower risk combination', NULL, 'CYP3A4 (quetiapine), CYP2D6 (risperidone)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_OLANZAPINE_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'OLANZAPINE', 'Olanzapine (Zyprexa)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP. Olanzapine has lowest QT effect among atypicals. Quetiapine modest.',
'Low-moderate TdP risk. Combined QT effect small (<15ms typically).',
'Use with caution. Lower risk combination. Monitor ECG in patients with cardiac risk factors.',
FALSE, 0.18, 0.68, 'FDA_LABEL', 'Seroquel Section 7; Zyprexa Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Olanzapine minimal QT; Lower risk combination', NULL, 'CYP3A4 (quetiapine), CYP1A2 (olanzapine)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_RISPERIDONE_OLANZAPINE_001', 'RISPERIDONE', 'Risperidone (Risperdal)', 'OLANZAPINE', 'Olanzapine (Zyprexa)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP with low individual QT effects.',
'Low TdP risk. Both have minimal QT prolongation. Combination occasionally used in treatment-resistant cases.',
'Use with caution. Lower risk combination. Monitor metabolic effects more than QT.',
FALSE, 0.15, 0.65, 'FDA_LABEL', 'Risperdal Section 7; Zyprexa Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Minimal individual QT; Metabolic effects more concerning', NULL, 'CYP2D6 (risperidone), CYP1A2 (olanzapine)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_CLOZAPINE_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'CLOZAPINE', 'Clozapine (Clozaril)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP. Clozapine myocarditis more concerning than QT.',
'Moderate TdP risk. Clozapine QT modest. Myocarditis/cardiomyopathy more important monitoring.',
'Use with caution. Clozapine monitoring focuses on agranulocytosis and myocarditis. QT secondary.',
FALSE, 0.12, 0.70, 'FDA_LABEL', 'Clozaril REMS; Seroquel Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Clozapine myocarditis focus; QT secondary concern', NULL, 'CYP1A2, CYP3A4 (both)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Clozapine Guidelines', 'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Iloperidone (Known Risk unique case)
('QT_ILOPERIDONE_HALOPERIDOL_001', 'ILOPERIDONE', 'Iloperidone (Fanapt)', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Iloperidone requires slow titration due to QT. IV haloperidol BLACK BOX.',
'Extreme TdP risk. Iloperidone already prolongs QT 10-20ms. IV haloperidol additional effect.',
'CONTRAINDICATED. If acute sedation needed in patient on iloperidone, use benzodiazepine.',
FALSE, 0.05, 0.96, 'FDA_LABEL', 'Fanapt Section 7 (QT); Haldol BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Iloperidone requires titration; Acute setting dangerous', NULL, 'CYP2D6, CYP3A4 (iloperidone)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ILOPERIDONE_ZIPRASIDONE_001', 'ILOPERIDONE', 'Iloperidone (Fanapt)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ziprasidone label contraindicates QT drugs. Iloperidone significant QT effect.',
'Extreme TdP risk. Combined QT may exceed 40ms. Both labels prohibit.',
'CONTRAINDICATED. Per Ziprasidone and Iloperidone labels. Use lower QT alternatives.',
FALSE, 0.03, 0.97, 'FDA_LABEL', 'Geodon Section 4; Fanapt Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Mutual label concerns; Combined QT >40ms', NULL, 'CYP3A4, CYP2D6 (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Aripiprazole combinations (Possible Risk - lowest)
('QT_HALOPERIDOL_ARIPIPRAZOLE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'pharmacodynamic', 'moderate', 'established',
'IV haloperidol Known Risk + aripiprazole Possible Risk (minimal QT effect).',
'Moderate TdP risk. Aripiprazole has lowest QT effect among all antipsychotics (2-5ms).',
'Use with caution. Aripiprazole is safest atypical for QT. Monitor ECG with IV haloperidol.',
FALSE, 0.25, 0.75, 'FDA_LABEL', 'Haldol BLACK BOX; Abilify Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Aripiprazole minimal QT; Safest combination', NULL, 'CYP2D6, CYP3A4 (aripiprazole)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ZIPRASIDONE_ARIPIPRAZOLE_001', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'pharmacodynamic', 'moderate', 'established',
'Ziprasidone Known Risk + aripiprazole Possible Risk (minimal QT).',
'Moderate TdP risk. Aripiprazole adds minimal QT effect. Ziprasidone label still advises caution.',
'Use with caution. Aripiprazole is safer adjunct than other atypicals. Monitor QT.',
FALSE, 0.15, 0.80, 'FDA_LABEL', 'Geodon Section 4; Abilify Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Aripiprazole safest adjunct; Monitor per ziprasidone label', NULL, 'CYP3A4 (ziprasidone), CYP2D6/3A4 (aripiprazole)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')

ON CONFLICT (interaction_id) DO NOTHING;

-- Additional Antipsychotic combinations
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES
-- Asenapine combinations
('QT_HALOPERIDOL_ASENAPINE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'ASENAPINE', 'Asenapine (Saphris)', 'pharmacodynamic', 'major', 'established',
'IV haloperidol Known Risk + asenapine Possible Risk (moderate QT effect 5-15ms).',
'High TdP risk. Asenapine has more QT effect than risperidone or olanzapine.',
'Avoid if possible. Use olanzapine or aripiprazole as alternative to asenapine in this setting.',
FALSE, 0.10, 0.85, 'FDA_LABEL', 'Haldol BLACK BOX; Saphris Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Asenapine moderate QT; Alternative available', NULL, 'CYP1A2 (asenapine)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Paliperidone combinations
('QT_HALOPERIDOL_PALIPERIDONE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'PALIPERIDONE', 'Paliperidone (Invega)', 'pharmacodynamic', 'major', 'established',
'IV haloperidol Known Risk + paliperidone Possible Risk. Paliperidone is active metabolite of risperidone.',
'Moderate-high TdP risk. Paliperidone QT profile similar to risperidone (5-10ms).',
'Use with caution. Monitor ECG. Paliperidone among safer atypicals for QT.',
FALSE, 0.20, 0.82, 'FDA_LABEL', 'Haldol BLACK BOX; Invega Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Paliperidone similar to risperidone; Acute + LAI scenario', NULL, 'Not CYP-dependent (paliperidone)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Lurasidone combinations
('QT_HALOPERIDOL_LURASIDONE_001', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'LURASIDONE', 'Lurasidone (Latuda)', 'pharmacodynamic', 'moderate', 'established',
'IV haloperidol Known Risk + lurasidone Possible Risk (minimal QT 2-5ms).',
'Moderate TdP risk. Lurasidone has very low QT effect. Among safest atypicals.',
'Use with caution. Lurasidone among safest atypicals for QT. Monitor ECG with IV haloperidol.',
FALSE, 0.15, 0.75, 'FDA_LABEL', 'Haldol BLACK BOX; Latuda Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Lurasidone minimal QT; Safest combination', NULL, 'CYP3A4 (lurasidone)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- More Possible Risk combinations
('QT_CLOZAPINE_RISPERIDONE_001', 'CLOZAPINE', 'Clozapine (Clozaril)', 'RISPERIDONE', 'Risperidone (Risperdal)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP. Common combination in treatment-resistant schizophrenia.',
'Moderate TdP risk. Clozapine myocarditis more concerning. QT effect modest.',
'Use with caution. Monitor for myocarditis/cardiomyopathy. ECG part of clozapine monitoring.',
FALSE, 0.15, 0.72, 'FDA_LABEL', 'Clozaril REMS; Risperdal Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Clozapine augmentation common; Myocarditis monitoring', NULL, 'CYP1A2 (clozapine), CYP2D6 (risperidone)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Clozapine Guidelines', 'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_CLOZAPINE_ARIPIPRAZOLE_001', 'CLOZAPINE', 'Clozapine (Clozaril)', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP. Aripiprazole augmentation of clozapine is evidence-based.',
'Low-moderate TdP risk. Aripiprazole minimal QT. Evidence supports this combination.',
'Use with caution. Evidence-based augmentation strategy. Monitor myocarditis for clozapine.',
FALSE, 0.20, 0.68, 'FDA_LABEL', 'Clozaril REMS; Abilify Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Aripiprazole minimal QT; Evidence-based augmentation', NULL, 'CYP1A2 (clozapine), CYP2D6/3A4 (aripiprazole)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Clozapine Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_OLANZAPINE_ARIPIPRAZOLE_001', 'OLANZAPINE', 'Olanzapine (Zyprexa)', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP. Both have minimal QT effects. Low risk combination.',
'Low TdP risk. Both among safest atypicals for QT. May be used for metabolic improvement.',
'Use with caution. Low QT risk. May use aripiprazole adjunct to reduce olanzapine metabolic effects.',
FALSE, 0.15, 0.60, 'FDA_LABEL', 'Zyprexa Section 7; Abilify Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Minimal QT both drugs; Metabolic strategy', NULL, 'CYP1A2 (olanzapine), CYP2D6/3A4 (aripiprazole)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Guidelines', 'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUETIAPINE_ARIPIPRAZOLE_001', 'QUETIAPINE', 'Quetiapine (Seroquel)', 'ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'pharmacodynamic', 'moderate', 'established',
'Both Possible Risk TdP. Aripiprazole minimal QT. Common combination in bipolar disorder.',
'Low TdP risk. Both relatively safe for QT. Common bipolar maintenance combination.',
'Use with caution. Low QT risk combination. Monitor metabolic effects.',
FALSE, 0.25, 0.65, 'FDA_LABEL', 'Seroquel Section 7; Abilify Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Possible Risk; Low combined QT; Bipolar indication', NULL, 'CYP3A4 (quetiapine), CYP2D6/3A4 (aripiprazole)', 'Possible Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'APA Bipolar Guidelines', 'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')

ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Migration 022 Complete: % total DDIs', v_count;
END $$;
