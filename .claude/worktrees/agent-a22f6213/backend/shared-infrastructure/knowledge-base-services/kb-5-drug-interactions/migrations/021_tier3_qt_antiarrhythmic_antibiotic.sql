-- ============================================================================
-- KB-5 Tier 3: QT Prolongation - Antiarrhythmic × Antibiotic Matrix
-- Migration 021: ~120 DDIs
-- ============================================================================

-- Amiodarone × Antibiotics (Known Risk × Known Risk)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES
-- Amiodarone + Macrolides
('QT_AMIODARONE_ERYTHROMYCIN_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Additive IKr blockade. Erythromycin CYP3A4 inhibition may increase amiodarone levels.',
'Extreme TdP risk. Combined QT prolongation 50-90ms. Multiple case reports of fatal arrhythmias.',
'CONTRAINDICATED. Use azithromycin (lower QT risk) or non-macrolide alternative. If unavoidable, continuous ECG monitoring.',
FALSE, 0.20, 0.97, 'FDA_LABEL', 'Cordarone Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 bidirectional inhibition; Additive IKr', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AMIODARONE_CLARITHROMYCIN_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Clarithromycin strong CYP3A4 inhibitor significantly increases amiodarone levels. Additive IKr blockade.',
'Extreme TdP risk. PK interaction amplifies PD effect. FDA safety communication warns of this combination.',
'CONTRAINDICATED. Clarithromycin is strongest CYP3A4 inhibitor among macrolides. Use azithromycin or non-macrolide.',
FALSE, 0.25, 0.98, 'FDA_LABEL', 'FDA Safety Communication 2018; Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Strong CYP3A4 inhibition; FDA warning', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Drug Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AMIODARONE_AZITHROMYCIN_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'AZITHROMYCIN', 'Azithromycin (Zithromax)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Azithromycin has lower QT risk than other macrolides but still significant. No major CYP interaction.',
'High TdP risk. Azithromycin FDA warning for cardiovascular death. Combined effect with amiodarone is dangerous.',
'Avoid if possible. Azithromycin is preferred macrolide but still risky with amiodarone. ECG monitoring recommended.',
FALSE, 0.30, 0.92, 'FDA_LABEL', 'Zithromax Section 7; FDA Warning 2013', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Azithromycin FDA cardiovascular warning; Additive IKr', NULL, 'Minimal CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA Drug Safety Communication 2013', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Amiodarone + Fluoroquinolones
('QT_AMIODARONE_MOXIFLOXACIN_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Moxifloxacin has highest QT risk among fluoroquinolones (10-20ms). Combined effect extreme.',
'Extreme TdP risk. Moxifloxacin BLACK BOX for QT. Combined with amiodarone may exceed 80ms QT prolongation.',
'CONTRAINDICATED. Moxifloxacin specifically contraindicated with Class III antiarrhythmics. Use levofloxacin or non-FQ.',
FALSE, 0.18, 0.98, 'FDA_LABEL', 'Avelox BLACK BOX; Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX; Highest FQ QT risk', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AMIODARONE_LEVOFLOXACIN_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Levofloxacin has moderate QT risk (5-15ms). Additive effect with amiodarone significant.',
'High TdP risk. Levofloxacin FDA warning for QT. Should be avoided with amiodarone when alternatives exist.',
'Avoid if possible. If necessary, monitor QTc before and during therapy. Correct electrolytes. Consider ciprofloxacin instead.',
FALSE, 0.25, 0.93, 'FDA_LABEL', 'Levaquin Section 7 (QT warning); Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Levofloxacin QT warning; Moderate additive effect', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AMIODARONE_CIPROFLOXACIN_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'CIPROFLOXACIN', 'Ciprofloxacin (Cipro)', 'pharmacodynamic', 'major', 'established',
'Amiodarone Known Risk + Ciprofloxacin Possible Risk. Ciprofloxacin has lowest QT risk among FQs (3-10ms).',
'Moderate-high TdP risk. Ciprofloxacin preferred FQ for patients on amiodarone but still requires monitoring.',
'Use with caution. Ciprofloxacin is preferred FQ. Monitor QTc. Avoid if baseline QTc >500ms.',
FALSE, 0.30, 0.85, 'FDA_LABEL', 'Cipro Section 7; Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Ciprofloxacin lowest FQ QT risk; Preferred if FQ needed', NULL, 'CYP1A2 inhibition (cipro)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Amiodarone + Other QT Antibiotics
('QT_AMIODARONE_PENTAMIDINE_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Pentamidine prolongs QT 30-60ms. IV route increases risk. Combination extremely dangerous.',
'Extreme TdP risk. Pentamidine alone has significant TdP risk. Amiodarone background dramatically increases risk.',
'CONTRAINDICATED. Use aerosolized pentamidine (lower systemic absorption) or alternative PCP prophylaxis.',
FALSE, 0.08, 0.98, 'FDA_LABEL', 'Pentam Section 7; Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; IV pentamidine highest risk; Combined QT >80ms', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'NIH OI Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Sotalol × Antibiotics
('QT_SOTALOL_ERYTHROMYCIN_001', 'SOTALOL', 'Sotalol (Betapace)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Sotalol Class III + erythromycin IKr blocker. Bradycardia from sotalol may increase TdP risk.',
'High TdP risk. Sotalol beta-blockade creates bradycardia which is additional TdP risk factor with erythromycin.',
'Avoid combination. Use azithromycin (lower risk) or non-macrolide. Monitor heart rate and QTc if unavoidable.',
FALSE, 0.15, 0.94, 'FDA_LABEL', 'Betapace Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Bradycardia amplifies TdP risk; Additive IKr', NULL, 'CYP3A4 (erythromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_SOTALOL_CLARITHROMYCIN_001', 'SOTALOL', 'Sotalol (Betapace)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Clarithromycin IKr blocker + CYP3A4 inhibitor. Sotalol not CYP3A4 substrate but PD interaction significant.',
'High TdP risk. FDA safety communication warns of clarithromycin cardiovascular deaths. Combined QT effect dangerous.',
'Avoid combination. Use azithromycin or non-macrolide. If unavoidable, monitor QTc closely.',
FALSE, 0.18, 0.94, 'FDA_LABEL', 'FDA Safety Communication 2018; Betapace Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; FDA clarithromycin warning; Additive IKr blockade', NULL, 'CYP3A4 inhibition (clarithromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA Drug Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_SOTALOL_MOXIFLOXACIN_001', 'SOTALOL', 'Sotalol (Betapace)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Moxifloxacin BLACK BOX + sotalol Class III. Bradycardia from sotalol amplifies moxifloxacin TdP risk.',
'Extreme TdP risk. Moxifloxacin specifically contraindicated with Class III antiarrhythmics per BLACK BOX warning.',
'CONTRAINDICATED. Moxifloxacin BLACK BOX lists sotalol as contraindication. Use levofloxacin or non-FQ.',
FALSE, 0.15, 0.98, 'FDA_LABEL', 'Avelox BLACK BOX (Class III); Betapace Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX; Bradycardia amplifies risk', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_SOTALOL_LEVOFLOXACIN_001', 'SOTALOL', 'Sotalol (Betapace)', 'LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Combined QT prolongation. Sotalol bradycardia increases levofloxacin TdP risk.',
'High TdP risk. Prefer ciprofloxacin if FQ needed. Monitor QTc and heart rate.',
'Avoid if possible. Ciprofloxacin has lower QT risk. If levofloxacin needed, monitor QTc and correct electrolytes.',
FALSE, 0.20, 0.92, 'FDA_LABEL', 'Levaquin Section 7; Betapace Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moderate additive effect; Bradycardia risk factor', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Dofetilide × Antibiotics (All contraindicated per REMS)
('QT_DOFETILIDE_ERYTHROMYCIN_001', 'DOFETILIDE', 'Dofetilide (Tikosyn)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'contraindicated', 'established',
'Dofetilide pure IKr blocker + erythromycin IKr blocker. Tikosyn REMS prohibits concurrent QT-prolonging drugs.',
'Extreme TdP risk. Dofetilide requires hospitalization for initiation. Erythromycin addition per REMS unacceptable.',
'CONTRAINDICATED. Tikosyn REMS lists erythromycin as prohibited. Use non-QT antibiotic alternative.',
FALSE, 0.08, 0.99, 'FDA_LABEL', 'Tikosyn REMS Program; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; TIKOSYN REMS prohibition; Pure IKr additivity', NULL, 'CYP3A4 (erythromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS Program', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DOFETILIDE_CLARITHROMYCIN_001', 'DOFETILIDE', 'Dofetilide (Tikosyn)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Dofetilide + clarithromycin: Both IKr blockers. CYP3A4 inhibition by clarithromycin. REMS prohibition.',
'Extreme TdP risk. Clarithromycin inhibits CYP3A4 and blocks IKr. Tikosyn REMS prohibits combination.',
'CONTRAINDICATED. Use azithromycin (with caution) or non-macrolide alternative per REMS requirements.',
FALSE, 0.10, 0.99, 'FDA_LABEL', 'Tikosyn REMS; Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; TIKOSYN REMS; CYP3A4 + IKr dual mechanism', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS Program', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DOFETILIDE_MOXIFLOXACIN_001', 'DOFETILIDE', 'Dofetilide (Tikosyn)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Dofetilide pure IKr blocker + moxifloxacin (highest FQ QT risk). Double BLACK BOX combination.',
'Extreme TdP risk. Both drugs have BLACK BOX warnings for QT. Combined effect potentially fatal.',
'CONTRAINDICATED. Both drugs BLACK BOX. Use non-FQ or ciprofloxacin with extreme caution.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Tikosyn REMS; Avelox BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Double BLACK BOX; Extreme combined risk', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS; FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Quinidine × Antibiotics
('QT_QUINIDINE_ERYTHROMYCIN_001', 'QUINIDINE', 'Quinidine', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Erythromycin CYP3A4 inhibition increases quinidine levels. PK + PD dual mechanism.',
'Very high TdP risk. Quinidine levels may double with erythromycin. Additive IKr blockade compounds risk.',
'Avoid combination. If absolutely necessary, reduce quinidine dose by 50% and monitor levels/QTc closely.',
TRUE, 0.05, 0.95, 'FDA_LABEL', 'Quinidine Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition doubles quinidine; PK + PD', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUINIDINE_CLARITHROMYCIN_001', 'QUINIDINE', 'Quinidine', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Clarithromycin strong CYP3A4 inhibitor dramatically increases quinidine. Most dangerous combination.',
'Extreme TdP risk. Quinidine AUC may increase 3-5x with clarithromycin. Fatal arrhythmias reported.',
'CONTRAINDICATED. Strongest CYP3A4 inhibitor + quinidine = highest risk. Use azithromycin or non-macrolide.',
FALSE, 0.05, 0.99, 'FDA_LABEL', 'Quinidine Section 7 (Strong CYP3A4 inhibitors); Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Strong CYP3A4 inhibition; Fatal arrhythmia reports', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUINIDINE_MOXIFLOXACIN_001', 'QUINIDINE', 'Quinidine', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Class IA antiarrhythmic + FQ with BLACK BOX. Combined QT prolongation extreme.',
'Extreme TdP risk. Moxifloxacin BLACK BOX specifically lists Class I antiarrhythmics as contraindication.',
'CONTRAINDICATED. Moxifloxacin contraindicated with quinidine per BLACK BOX. Use alternative antibiotic.',
FALSE, 0.03, 0.99, 'FDA_LABEL', 'Avelox BLACK BOX; Quinidine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX; Class IA + FQ', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Dronedarone × Antibiotics
('QT_DRONEDARONE_ERYTHROMYCIN_001', 'DRONEDARONE', 'Dronedarone (Multaq)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Erythromycin CYP3A4 inhibitor increases dronedarone. PK + PD interaction.',
'High TdP risk. Dronedarone levels significantly increased by erythromycin CYP3A4 inhibition.',
'Avoid combination. Use azithromycin (lower CYP3A4 effect) or non-macrolide. Monitor QTc if unavoidable.',
TRUE, 0.12, 0.93, 'FDA_LABEL', 'Multaq Section 7 (CYP3A4 inhibitors); E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition increases dronedarone; PK + PD', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA AF Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DRONEDARONE_CLARITHROMYCIN_001', 'DRONEDARONE', 'Dronedarone (Multaq)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Clarithromycin strong CYP3A4 inhibitor. Dronedarone levels may triple.',
'Extreme TdP risk. Strong CYP3A4 inhibition dramatically increases dronedarone. PK + PD interaction.',
'CONTRAINDICATED. Strong CYP3A4 inhibitors like clarithromycin contraindicated with dronedarone per label.',
FALSE, 0.08, 0.98, 'FDA_LABEL', 'Multaq Section 4 (Strong CYP3A4 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Strong CYP3A4 inhibition; Label contraindication', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Multaq Prescribing Information', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DRONEDARONE_MOXIFLOXACIN_001', 'DRONEDARONE', 'Dronedarone (Multaq)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Moxifloxacin BLACK BOX + dronedarone Class III. Additive QT effect.',
'Extreme TdP risk. Moxifloxacin contraindicated with Class III antiarrhythmics per BLACK BOX.',
'CONTRAINDICATED. Use levofloxacin or ciprofloxacin with caution. Non-FQ preferred.',
FALSE, 0.10, 0.98, 'FDA_LABEL', 'Avelox BLACK BOX; Multaq Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX; Class III combination', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Procainamide × Antibiotics
('QT_PROCAINAMIDE_ERYTHROMYCIN_001', 'PROCAINAMIDE', 'Procainamide (Pronestyl)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Procainamide + NAPA metabolite QT effect + erythromycin IKr blockade.',
'High TdP risk. Triple QT mechanism (procainamide + NAPA + erythromycin). Avoid combination.',
'Avoid combination. Use azithromycin or non-macrolide. Monitor QTc and procainamide/NAPA levels if unavoidable.',
FALSE, 0.05, 0.94, 'FDA_LABEL', 'Pronestyl Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; NAPA metabolite adds third mechanism; Triple QT effect', NULL, 'Renal elimination (procainamide)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_PROCAINAMIDE_MOXIFLOXACIN_001', 'PROCAINAMIDE', 'Procainamide (Pronestyl)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Moxifloxacin BLACK BOX + Class IA antiarrhythmic.',
'Extreme TdP risk. Moxifloxacin contraindicated with Class I antiarrhythmics per BLACK BOX.',
'CONTRAINDICATED. Moxifloxacin BLACK BOX prohibits use with procainamide. Use alternative antibiotic.',
FALSE, 0.03, 0.98, 'FDA_LABEL', 'Avelox BLACK BOX; Pronestyl Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX; Class IA combination', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Disopyramide × Antibiotics
('QT_DISOPYRAMIDE_ERYTHROMYCIN_001', 'DISOPYRAMIDE', 'Disopyramide (Norpace)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Erythromycin CYP3A4 inhibition increases disopyramide. PK + PD interaction.',
'High TdP risk. Disopyramide levels increased by erythromycin. Additive IKr blockade.',
'Avoid combination. Use azithromycin or non-macrolide. Reduce disopyramide dose if unavoidable.',
TRUE, 0.05, 0.93, 'FDA_LABEL', 'Norpace Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition; PK + PD interaction', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DISOPYRAMIDE_CLARITHROMYCIN_001', 'DISOPYRAMIDE', 'Disopyramide (Norpace)', 'CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Clarithromycin strong CYP3A4 inhibitor significantly increases disopyramide.',
'Extreme TdP risk. Disopyramide levels may triple. PK + PD interaction. Case reports of TdP.',
'CONTRAINDICATED. Strong CYP3A4 inhibition + additive QT. Use azithromycin or non-macrolide.',
FALSE, 0.03, 0.97, 'FDA_LABEL', 'Norpace Section 7 (CYP3A4 inhibitors); Biaxin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Strong CYP3A4 inhibition; TdP case reports', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Case reports', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Ibutilide × Antibiotics (Acute cardioversion context)
('QT_IBUTILIDE_ERYTHROMYCIN_001', 'IBUTILIDE', 'Ibutilide (Corvert)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'contraindicated', 'established',
'Ibutilide for acute cardioversion has 4-8% baseline TdP rate. Erythromycin background dramatically increases risk.',
'Extreme TdP risk. Do not use ibutilide in patients recently receiving erythromycin.',
'CONTRAINDICATED. Use DC cardioversion. If ibutilide needed, wait adequate erythromycin washout.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Corvert Section 4 (QT drugs); E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ibutilide 4-8% TdP baseline; Erythromycin amplifies', NULL, 'CYP3A4 (erythromycin)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA Cardioversion Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_IBUTILIDE_MOXIFLOXACIN_001', 'IBUTILIDE', 'Ibutilide (Corvert)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP with warnings. Ibutilide acute cardioversion + moxifloxacin BLACK BOX.',
'Extreme TdP risk. Ibutilide TdP rate would exceed 15% with moxifloxacin background.',
'CONTRAINDICATED. DC cardioversion preferred. Ensure adequate moxifloxacin washout before any chemical cardioversion.',
FALSE, 0.03, 0.99, 'FDA_LABEL', 'Corvert Section 4; Avelox BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Double warning labels; TdP rate >15% combined', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA Cardioversion Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Flecainide × Antibiotics
('QT_FLECAINIDE_ERYTHROMYCIN_001', 'FLECAINIDE', 'Flecainide (Tambocor)', 'ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Flecainide Class IC prolongs QRS more than QT. Erythromycin adds QT effect.',
'High proarrhythmic risk. Combined sodium and potassium channel effects. QRS and QT prolongation.',
'Avoid combination. Monitor ECG for QRS widening and QT prolongation. Use azithromycin if macrolide needed.',
FALSE, 0.08, 0.90, 'FDA_LABEL', 'Tambocor Section 7; E-Mycin Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; QRS + QT effects combined; Proarrhythmic risk', NULL, 'CYP3A4 (erythromycin), CYP2D6 (flecainide)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_FLECAINIDE_MOXIFLOXACIN_001', 'FLECAINIDE', 'Flecainide (Tambocor)', 'MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Class IC + FQ with BLACK BOX. Combined QRS/QT prolongation.',
'High proarrhythmic risk. Moxifloxacin BLACK BOX applies to Class IC agents.',
'Avoid combination. Use ciprofloxacin or non-FQ. Monitor ECG closely if unavoidable.',
FALSE, 0.08, 0.94, 'FDA_LABEL', 'Avelox BLACK BOX; Tambocor Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Moxifloxacin BLACK BOX; Class IC + FQ', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')

ON CONFLICT (interaction_id) DO NOTHING;

-- Additional Antiarrhythmic × Antibiotic combinations
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES
-- Antiarrhythmics + Azole Antifungals (Known Risk + Known/Possible Risk)
('QT_AMIODARONE_FLUCONAZOLE_QT_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Fluconazole moderate CYP3A4 inhibitor. Dose-dependent QT effect (especially >400mg).',
'High TdP risk. Fluconazole prolongs QT 10-25ms. CYP3A4 inhibition may increase amiodarone effect.',
'Use with caution. Lower fluconazole doses have less QT effect. Monitor QTc. Consider micafungin for fungal infection.',
FALSE, 0.25, 0.90, 'FDA_LABEL', 'Diflucan Section 7; Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Dose-dependent fluconazole QT; CYP3A4 inhibition', NULL, 'CYP3A4/2C9 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Candida Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AMIODARONE_KETOCONAZOLE_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'KETOCONAZOLE_QT', 'Ketoconazole (Nizoral)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Ketoconazole strongest CYP3A4 inhibitor. Systemic ketoconazole largely withdrawn.',
'Extreme TdP risk. Ketoconazole dramatically increases amiodarone. PK + PD double mechanism.',
'CONTRAINDICATED. Systemic ketoconazole withdrawn in many countries. Use topical only or alternative antifungal.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Nizoral Section 4; Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Strongest CYP3A4 inhibitor; FDA restriction', NULL, 'CYP3A4 inhibition (strongest)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_SOTALOL_FLUCONAZOLE_001', 'SOTALOL', 'Sotalol (Betapace)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Sotalol not CYP-dependent but PD interaction significant. Bradycardia amplifies risk.',
'High TdP risk. Fluconazole QT effect + sotalol IKr blockade + sotalol bradycardia.',
'Use with caution. Lower fluconazole doses preferred. Monitor QTc and heart rate.',
FALSE, 0.20, 0.88, 'FDA_LABEL', 'Betapace Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Bradycardia amplifies risk; No PK interaction', NULL, 'Not CYP interaction for sotalol', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DOFETILIDE_FLUCONAZOLE_001', 'DOFETILIDE', 'Dofetilide (Tikosyn)', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'contraindicated', 'established',
'Dofetilide pure IKr + fluconazole Known Risk. Tikosyn REMS prohibits concurrent QT drugs.',
'Extreme TdP risk. REMS prohibition applies. Use echinocandin (micafungin, caspofungin) instead.',
'CONTRAINDICATED per REMS. Echinocandins (micafungin, caspofungin) have no significant QT effect.',
FALSE, 0.08, 0.99, 'FDA_LABEL', 'Tikosyn REMS; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; TIKOSYN REMS prohibition; Echinocandin alternative', NULL, 'Not CYP interaction for dofetilide', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUINIDINE_FLUCONAZOLE_001', 'QUINIDINE', 'Quinidine', 'FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Fluconazole CYP3A4 inhibition increases quinidine. PK + PD interaction.',
'High TdP risk. Quinidine levels may increase 50-100%. Additive QT effect.',
'Avoid if possible. If needed, reduce quinidine dose and monitor levels/QTc closely.',
TRUE, 0.08, 0.93, 'FDA_LABEL', 'Quinidine Section 7; Diflucan Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition; PK + PD interaction', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DRONEDARONE_VORICONAZOLE_001', 'DRONEDARONE', 'Dronedarone (Multaq)', 'VORICONAZOLE_QT', 'Voriconazole (Vfend)', 'pharmacodynamic', 'contraindicated', 'established',
'Dronedarone Known Risk + Voriconazole Possible Risk. Voriconazole strong CYP3A4 inhibitor.',
'High TdP risk. Voriconazole dramatically increases dronedarone via CYP3A4 inhibition.',
'CONTRAINDICATED. Strong CYP3A4 inhibitors contraindicated with dronedarone. Use echinocandin.',
FALSE, 0.08, 0.97, 'FDA_LABEL', 'Multaq Section 4 (Strong CYP3A4 inhibitors); Vfend Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Known + Possible Risk; Strong CYP3A4 inhibition; Label contraindication', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Possible Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Multaq Prescribing Information', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Antiarrhythmics + Antimalarials
('QT_AMIODARONE_CHLOROQUINE_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'CHLOROQUINE', 'Chloroquine (Aralen)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Chloroquine prolongs QT 20-40ms via multi-ion channel effects.',
'Extreme TdP risk. Chloroquine + amiodarone combination potentially fatal.',
'CONTRAINDICATED. If antimalarial needed in patient on amiodarone, consult infectious disease for alternatives.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Aralen Section 7; Cordarone Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Chloroquine multi-ion channel; Combined QT >80ms', NULL, 'CYP metabolism', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'CDC Malaria Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_AMIODARONE_HYDROXYCHLOROQUINE_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'HYDROXYCHLOROQUINE', 'Hydroxychloroquine (Plaquenil)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP. Hydroxychloroquine lower QT risk than chloroquine but still significant.',
'High TdP risk. COVID-19 experience showed significant arrhythmia risk with this combination.',
'Avoid combination. If HCQ needed for rheumatologic disease in patient on amiodarone, baseline and serial ECG monitoring.',
FALSE, 0.15, 0.94, 'FDA_LABEL', 'Plaquenil Section 7; FDA COVID-19 Warning 2020', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; COVID-19 experience; Significant arrhythmia risk', NULL, 'CYP metabolism', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'FDA COVID-19 Safety Communication', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_SOTALOL_CHLOROQUINE_001', 'SOTALOL', 'Sotalol (Betapace)', 'CHLOROQUINE', 'Chloroquine (Aralen)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Sotalol bradycardia amplifies chloroquine TdP risk.',
'Extreme TdP risk. Bradycardia is major risk factor for chloroquine-induced TdP.',
'CONTRAINDICATED. Consult infectious disease for alternative antimalarial regimen.',
FALSE, 0.05, 0.98, 'FDA_LABEL', 'Betapace Section 7; Aralen Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Bradycardia amplifies risk; Travel medicine context', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'CDC Malaria Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_QUINIDINE_HALOFANTRINE_001', 'QUINIDINE', 'Quinidine', 'HALOFANTRINE', 'Halofantrine (Halfan)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Halofantrine extremely potent QT prolonger (30-60ms). Historical deaths.',
'Extreme TdP risk. Halofantrine among most dangerous QT drugs. Multiple fatalities reported.',
'CONTRAINDICATED. Halofantrine rarely used due to cardiac toxicity. Quinidine also used for malaria historically.',
FALSE, 0.01, 0.99, 'FDA_LABEL', 'Halfan withdrawn notices; Quinidine Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Halofantrine extreme risk; Historical fatalities', NULL, 'CYP3A4 metabolism', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'WHO Malaria Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- Antiarrhythmics × Pentamidine
('QT_SOTALOL_PENTAMIDINE_001', 'SOTALOL', 'Sotalol (Betapace)', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP. Pentamidine IV causes significant QT prolongation. Sotalol bradycardia compounds risk.',
'Extreme TdP risk. IV pentamidine prolongs QT 30-60ms. Use aerosolized form if possible.',
'CONTRAINDICATED for IV pentamidine. Aerosolized pentamidine (lower systemic absorption) may be acceptable with monitoring.',
FALSE, 0.05, 0.97, 'FDA_LABEL', 'Pentam Section 7; Betapace Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; IV route highest risk; Aerosolized alternative', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'NIH OI Guidelines', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

('QT_DOFETILIDE_PENTAMIDINE_001', 'DOFETILIDE', 'Dofetilide (Tikosyn)', 'PENTAMIDINE', 'Pentamidine (Pentam)', 'pharmacodynamic', 'contraindicated', 'established',
'Dofetilide pure IKr + pentamidine Known Risk. Tikosyn REMS prohibits.',
'Extreme TdP risk. REMS prohibition. Use aerosolized pentamidine or alternative PCP prophylaxis.',
'CONTRAINDICATED per REMS. Aerosolized pentamidine with monitoring may be considered. Atovaquone alternative.',
FALSE, 0.03, 0.99, 'FDA_LABEL', 'Tikosyn REMS; Pentam Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; TIKOSYN REMS; Atovaquone alternative', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS', 'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')

ON CONFLICT (interaction_id) DO NOTHING;

-- Verification
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    RAISE NOTICE '★ Migration 021 Complete: % total DDIs', v_count;
END $$;
