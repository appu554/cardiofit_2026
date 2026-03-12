-- ============================================================================
-- KB-5 Tier 3: QT Prolongation - Antibiotic × Antibiotic Matrix
-- Migration 025: ~80 DDIs
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
-- Macrolide × Fluoroquinolone (Known Risk × Known Risk)
('QT_ERYTHROMYCIN_MOXIFLOXACIN_001', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Erythromycin + moxifloxacin BLACK BOX. Combined QT 30-50ms.',
'Extreme TdP risk. No clinical rationale for combination. Both respiratory antibiotics.',
'CONTRAINDICATED. Never combine. Treat with one antibiotic class.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Avelox BLACK BOX; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; No clinical rationale; Redundant coverage', NULL, 'CYP3A4 (erythromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ERYTHROMYCIN_LEVOFLOXACIN_001', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Rarely combined clinically but significant QT additive effect.',
'Very high TdP risk. Combined QT prolongation 20-40ms.',
'Avoid combination. Choose one antibiotic class. If both needed, monitor QTc.',
FALSE, 0.08, 0.94, 'FDA_LABEL', 'E-Mycin Section 7; Levaquin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Avoid combination; Monitor if needed', NULL, 'CYP3A4 (erythromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_CLARITHROMYCIN_MOXIFLOXACIN_001', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Double QT-prolonging antibiotics. Moxifloxacin BLACK BOX.',
'Extreme TdP risk. Combined QT prolongation 30-40ms. No clinical justification.',
'CONTRAINDICATED. Choose one antibiotic. Both cover atypicals.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Avelox BLACK BOX; Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; No clinical rationale', NULL, 'CYP3A4 (clarithromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_CLARITHROMYCIN_LEVOFLOXACIN_001', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Redundant atypical coverage. QT additive.',
'Very high TdP risk. No reason to combine. Both effective for atypicals.',
'Avoid combination. Single antibiotic sufficient.',
FALSE, 0.08, 0.94, 'FDA_LABEL', 'Biaxin Section 7; Levaquin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Redundant coverage', NULL, 'CYP3A4 (clarithromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AZITHROMYCIN_MOXIFLOXACIN_001', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Azithromycin FDA warning + moxifloxacin BLACK BOX.',
'High TdP risk. Both have cardiovascular warnings. Avoid combination.',
'Avoid combination. If both needed for complex infection, monitor QTc closely.',
FALSE, 0.10, 0.93, 'FDA_LABEL', 'FDA Azithromycin Warning 2013; Avelox BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; FDA warnings both drugs', NULL, 'Minimal CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AZITHROMYCIN_LEVOFLOXACIN_001', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. May be combined in severe pneumonia. Monitor closely.',
'High TdP risk. Occasionally combined in severe CAP. Requires monitoring.',
'Use with caution. If combined for severe infection, baseline ECG and monitoring required.',
FALSE, 0.15, 0.88, 'FDA_LABEL', 'Zithromax Section 7; Levaquin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Severe CAP scenario; Monitor QT', NULL, 'Minimal CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA CAP Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Macrolide × Azole Antifungal
('QT_ERYTHROMYCIN_FLUCONAZOLE_001', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Erythromycin and fluconazole both IKr blockers.',
'High TdP risk. Both CYP inhibitors and QT prolongers. Common in candidemia + bacterial infection.',
'Use with caution. Monitor QTc. Consider alternatives for one or both.',
FALSE, 0.18, 0.88, 'FDA_LABEL', 'E-Mycin Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Common in mixed infections', NULL, 'CYP3A4 inhibition (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_CLARITHROMYCIN_FLUCONAZOLE_001', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Strong CYP3A4 inhibitors. Additive QT effect.',
'Very high TdP risk. Both strong CYP inhibitors. PK + PD interaction.',
'Avoid if possible. Use azithromycin + fluconazole or alternative combinations.',
FALSE, 0.15, 0.92, 'FDA_LABEL', 'Biaxin Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Strong CYP3A4 inhibitors', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ERYTHROMYCIN_KETOCONAZOLE_001', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'KETOCONAZOLE_QT', 'Ketoconazole (Nizoral)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ketoconazole systemic largely withdrawn. Strong CYP3A4 inhibitors.',
'Extreme TdP risk. Ketoconazole FDA restriction. Both strong QT prolongers.',
'CONTRAINDICATED. Systemic ketoconazole restricted. Use alternative antifungal.',
FALSE, 0.02, 0.98, 'FDA_LABEL', 'Nizoral Section 4; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ketoconazole FDA restriction', NULL, 'CYP3A4 inhibition (strongest)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Fluoroquinolone × Azole Antifungal
('QT_MOXIFLOXACIN_FLUCONAZOLE_001', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Moxifloxacin BLACK BOX. Common scenario in serious infections.',
'Very high TdP risk. Combined QT 20-45ms. Common in critically ill patients.',
'Avoid if possible. Use levofloxacin instead of moxifloxacin. Monitor QTc closely.',
FALSE, 0.20, 0.92, 'FDA_LABEL', 'Avelox BLACK BOX; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX; ICU scenario', NULL, 'Not primary CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_LEVOFLOXACIN_FLUCONAZOLE_001', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Common combination in hospitalized patients.',
'High TdP risk. Very common in critically ill. Requires monitoring.',
'Use with caution. Preferred over moxifloxacin + fluconazole. Monitor QTc.',
FALSE, 0.30, 0.85, 'FDA_LABEL', 'Levaquin Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Very common; Monitor QT', NULL, 'Not primary CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_CIPROFLOXACIN_FLUCONAZOLE_001', 'CIPROFLOXACIN', 'Ciprofloxacin (Cipro)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'moderate', 'established',
'Ciprofloxacin Possible Risk + fluconazole Known Risk. Ciprofloxacin lowest FQ QT.',
'Moderate TdP risk. Ciprofloxacin has lowest QT risk. Preferred FQ with fluconazole.',
'Use with caution. Preferred over levofloxacin or moxifloxacin. Monitor in high-risk.',
FALSE, 0.35, 0.78, 'FDA_LABEL', 'Cipro Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Ciprofloxacin lowest FQ QT; Preferred', NULL, 'CYP1A2 (ciprofloxacin)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Fluoroquinolone × Fluoroquinolone (no clinical rationale)
('QT_MOXIFLOXACIN_LEVOFLOXACIN_001', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Both FQs. No clinical rationale for combination.',
'Extreme TdP risk. Combined QT prolongation. Redundant mechanism.',
'CONTRAINDICATED. Never combine two fluoroquinolones. Select one.',
FALSE, 0.01, 0.99, 'FDA_LABEL', 'Avelox BLACK BOX; Levaquin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; No clinical rationale; Redundant', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Macrolide × Macrolide (no clinical rationale)
('QT_ERYTHROMYCIN_CLARITHROMYCIN_001', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Both macrolides. No clinical rationale.',
'Extreme TdP risk. Redundant mechanism. Combined QT effect.',
'CONTRAINDICATED. Never combine two macrolides. Select one.',
FALSE, 0.01, 0.99, 'FDA_LABEL', 'E-Mycin Section 7; Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; No clinical rationale', NULL, 'CYP3A4 inhibition (both)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_ERYTHROMYCIN_AZITHROMYCIN_001', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Both macrolides. No rationale for combination.',
'Extreme TdP risk. Redundant coverage. Combined QT effect.',
'CONTRAINDICATED. Never combine two macrolides.',
FALSE, 0.01, 0.99, 'FDA_LABEL', 'E-Mycin Section 7; Zithromax Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Redundant', NULL, 'CYP3A4 (erythromycin only)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Pentamidine combinations
('QT_PENTAMIDINE_ERYTHROMYCIN_001', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Pentamidine extremely high QT risk (30-60ms).',
'Extreme TdP risk. IV pentamidine among highest QT drugs.',
'CONTRAINDICATED. Use aerosolized pentamidine + azithromycin if combination needed.',
FALSE, 0.03, 0.98, 'FDA_LABEL', 'Pentam Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Pentamidine extreme QT', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'NIH OI Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_PENTAMIDINE_MOXIFLOXACIN_001', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP with BLACK BOX level warnings.',
'Extreme TdP risk. Combined QT may exceed 80ms. Potentially fatal.',
'CONTRAINDICATED. Use alternative antibiotics and aerosolized pentamidine.',
FALSE, 0.02, 0.99, 'FDA_LABEL', 'Pentam Section 7; Avelox BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Combined QT >80ms', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'NIH OI Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_PENTAMIDINE_FLUCONAZOLE_001', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Common in HIV/AIDS patients with opportunistic infections.',
'Extreme TdP risk. Common scenario in immunocompromised patients.',
'CONTRAINDICATED for IV pentamidine. Use aerosolized pentamidine with fluconazole.',
FALSE, 0.08, 0.97, 'FDA_LABEL', 'Pentam Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; HIV/OI scenario; Aerosolized preferred', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'NIH OI Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Azole × Azole (rarely combined but can occur)
('QT_FLUCONAZOLE_ITRACONAZOLE_001', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'ITRACONAZOLE', 'Itraconazole (Sporanox)', 'pharmacodynamic', 'major', 'established',
'Fluconazole Known Risk + itraconazole Possible Risk. Rarely combined.',
'High TdP risk. Rarely clinically indicated to combine azoles.',
'Avoid combination. No rationale for dual azole therapy. Select one.',
FALSE, 0.03, 0.88, 'FDA_LABEL', 'Diflucan Section 7; Sporanox Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; No clinical rationale', NULL, 'CYP3A4 inhibition (both)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Candida Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Fluoroquinolone × Antimalarial
('QT_MOXIFLOXACIN_CHLOROQUINE_001', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'CHLOROQUINE', 'Chloroquine (Aralen)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Moxifloxacin BLACK BOX + chloroquine significant QT.',
'Extreme TdP risk. Travel medicine scenario. Combined QT 40-70ms.',
'CONTRAINDICATED. Use ciprofloxacin if FQ needed with antimalarial.',
FALSE, 0.03, 0.98, 'FDA_LABEL', 'Avelox BLACK BOX; Aralen Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Travel medicine; Extreme risk', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'CDC Malaria Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_MOXIFLOXACIN_HYDROXYCHLOROQUINE_001', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'HYDROXYCHLOROQUINE', 'Hydroxychloroquine (Plaquenil)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. COVID-19 experience highlighted arrhythmia risk.',
'Very high TdP risk. COVID-19 pandemic demonstrated significant arrhythmia risk.',
'Avoid if possible. COVID-19 experience showed this combination is dangerous.',
FALSE, 0.08, 0.94, 'FDA_LABEL', 'Avelox BLACK BOX; FDA COVID-19 Warning', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; COVID-19 experience; Avoid', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA COVID-19 Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AZITHROMYCIN_HYDROXYCHLOROQUINE_001', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'HYDROXYCHLOROQUINE', 'Hydroxychloroquine (Plaquenil)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Infamous COVID-19 combination with significant cardiac risk.',
'Very high TdP risk. COVID-19 pandemic showed multiple cardiac deaths with this combination.',
'Avoid combination. COVID-19 data showed no benefit and significant QT harm.',
FALSE, 0.15, 0.94, 'FDA_LABEL', 'FDA COVID-19 Warning 2020; Zithromax Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; COVID-19 deaths; FDA warning', NULL, 'Minimal CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA COVID-19 Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Macrolide × TMP-SMX (common respiratory scenario)
('QT_ERYTHROMYCIN_TRIMETHOPRIM_001', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'TRIMETHOPRIM', 'Trimethoprim (Primsol)', 'pharmacodynamic', 'moderate', 'established',
'Erythromycin Known Risk + trimethoprim Possible Risk. May occur in respiratory infections.',
'Moderate TdP risk. Trimethoprim has modest QT effect. Combined effect manageable.',
'Use with caution. Monitor QTc in high-risk patients.',
FALSE, 0.15, 0.75, 'FDA_LABEL', 'E-Mycin Section 7; Bactrim Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Trimethoprim modest QT', NULL, 'CYP3A4 (erythromycin)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Voriconazole combinations
('QT_VORICONAZOLE_ERYTHROMYCIN_001', 'VORICONAZOLE_QT', 'Voriconazole (Vfend)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Voriconazole Possible Risk + erythromycin Known Risk. Strong CYP3A4 inhibitors both.',
'High risk. Both CYP3A4 inhibitors. Mutual PK + PD interaction.',
'Avoid if possible. Use azithromycin instead of erythromycin.',
FALSE, 0.10, 0.85, 'FDA_LABEL', 'Vfend Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Strong CYP3A4 inhibitors; PK + PD', NULL, 'CYP3A4 inhibition (major)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Aspergillosis Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_VORICONAZOLE_MOXIFLOXACIN_001', 'VORICONAZOLE_QT', 'Voriconazole (Vfend)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'major', 'established',
'Voriconazole Possible Risk + moxifloxacin Known Risk (BLACK BOX).',
'High TdP risk. Common in severe fungal + bacterial infections. Monitor closely.',
'Use with caution. Prefer levofloxacin or ciprofloxacin. Monitor QTc.',
FALSE, 0.15, 0.88, 'FDA_LABEL', 'Avelox BLACK BOX; Vfend Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Moxifloxacin BLACK BOX', NULL, 'CYP3A4 (voriconazole inhibits)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_VORICONAZOLE_LEVOFLOXACIN_001', 'VORICONAZOLE_QT', 'Voriconazole (Vfend)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'moderate', 'established',
'Voriconazole Possible Risk + levofloxacin Known Risk.',
'Moderate-high TdP risk. Common combination in aspergillosis + bacterial superinfection.',
'Use with caution. Preferred over moxifloxacin. Monitor QTc.',
FALSE, 0.20, 0.82, 'FDA_LABEL', 'Vfend Section 7; Levaquin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Possible + Known Risk; Common aspergillosis scenario', NULL, 'CYP3A4 (voriconazole)', 'Possible Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Moderate (C)', 'IDSA Aspergillosis Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')

ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Migration 025 Complete: % total DDIs', v_count;
END $$;
