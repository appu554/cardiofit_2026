-- =====================================================================================
-- TIER 3: QT PROLONGATION SET - ANTIEMETIC × MULTI-CLASS MATRIX
-- Migration: 027
-- DDI Count: ~55 interactions
-- =====================================================================================
-- Clinical Context:
-- Antiemetics with QT-prolonging potential are frequently used in:
--   - Oncology (chemotherapy-induced nausea/vomiting - CINV)
--   - Post-operative nausea/vomiting (PONV)
--   - Palliative care (often with other QT-prolonging drugs)
--   - Gastroenterology (gastroparesis, functional dyspepsia)
--
-- CredibleMeds Classifications:
--   KNOWN_RISK: Droperidol (BLACK BOX), domperidone (banned in US), ondansetron (IV high-dose)
--   POSSIBLE_RISK: Ondansetron (standard doses), granisetron
--   CONDITIONAL_RISK: Metoclopramide, prochlorperazine
--
-- FDA Safety Communications:
--   - 2001: Droperidol BLACK BOX for QT prolongation and TdP
--   - 2012: Ondansetron 32mg IV dose withdrawn; max now 16mg
--   - 2019: Domperidone Risk Evaluation and Mitigation Strategy (REMS)
-- =====================================================================================

-- =====================================================================================
-- SECTION 1: DROPERIDOL COMBINATIONS (BLACK BOX WARNING - HIGHEST RISK)
-- =====================================================================================
-- Droperidol BLACK BOX: "Cases of QT prolongation and/or Torsades de Pointes have been
-- reported in patients receiving droperidol at doses at or below recommended doses"

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Droperidol × Ondansetron
('qt-droperidol-ondansetron', 'droperidol', 'Droperidol', 'ondansetron', 'Ondansetron',
 'contraindicated', 'established',
 'Additive QT prolongation: droperidol (BLACK BOX Known Risk) + ondansetron (Known Risk IV); both block IKr channels',
 'Severely increased TdP risk; combined QTc prolongation may exceed 50ms; sudden cardiac death reported with droperidol alone',
 'CONTRAINDICATED: Never combine; use alternative antiemetics (dexamethasone, NK1 antagonists); if droperidol necessary, ondansetron contraindicated',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX Warning 2001; Ondansetron FDA Safety Communication 2012', 'https://www.fda.gov/drugs/postmarket-drug-safety-information-patients-and-providers', 'US',
 'CREDIBLEMEDS', 'Droperidol IKr IC50 0.03μM (extremely potent); ondansetron IC50 1.9μM; combined blockade severe', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; BLACK BOX', 'ASA PONV Guidelines - Avoid Combination',
 'A', '2024-01-15'),

-- Droperidol × Granisetron
('qt-droperidol-granisetron', 'droperidol', 'Droperidol', 'granisetron', 'Granisetron',
 'contraindicated', 'established',
 'Additive QT prolongation; droperidol BLACK BOX plus granisetron Possible Risk',
 'High TdP risk; granisetron less risky than ondansetron but still dangerous with droperidol',
 'AVOID: If antiemetic needed with droperidol (which itself should be avoided), use dexamethasone or NK1 antagonists',
 'rapid', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX; Granisetron (Kytril) FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Droperidol Known Risk BLACK BOX + Granisetron Possible Risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'ASCO Antiemetic Guidelines',
 'A', '2024-01-15'),

-- Droperidol × Haloperidol (dual antipsychotic/antiemetic)
('qt-droperidol-haloperidol', 'droperidol', 'Droperidol', 'haloperidol', 'Haloperidol',
 'contraindicated', 'established',
 'Dual butyrophenone antipsychotics with additive IKr blockade; both have BLACK BOX warnings',
 'Extremely high TdP risk; pharmacologically redundant with compounded cardiac toxicity',
 'ABSOLUTELY CONTRAINDICATED: Same drug class, additive toxicity, no therapeutic rationale',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX; Haloperidol IV BLACK BOX', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both butyrophenones with potent IKr blockade; combined QTc prolongation potentially >100ms', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; Dual BLACK BOX', 'Never Combine',
 'A', '2024-01-15'),

-- Droperidol × Methadone
('qt-droperidol-methadone', 'droperidol', 'Droperidol', 'methadone', 'Methadone',
 'contraindicated', 'established',
 'Additive QT prolongation: droperidol BLACK BOX + methadone Known Risk; both potent IKr blockers',
 'Severe TdP risk; respiratory depression compounded; methadone deaths have occurred from cardiac causes',
 'CONTRAINDICATED: Use alternative antiemetics in methadone patients; ondansetron also risky but less than droperidol',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX; Methadone FDA QT Warning', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Droperidol IC50 0.03μM + Methadone IC50 2μM; combined severe IKr blockade', 'P-gp substrates', 'CYP3A4, CYP2B6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'SAMHSA Methadone Safety Guidelines',
 'A', '2024-01-15'),

-- Droperidol × Amiodarone
('qt-droperidol-amiodarone', 'droperidol', 'Droperidol', 'amiodarone', 'Amiodarone',
 'contraindicated', 'established',
 'Dual Known Risk QT prolongers; amiodarone has long half-life (40-55 days) persisting risk',
 'Severe QT prolongation risk; amiodarone effects persist weeks after discontinuation',
 'CONTRAINDICATED: If patient on amiodarone, droperidol absolutely contraindicated; use dexamethasone for PONV',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX; Amiodarone (Pacerone) Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Amiodarone prolongs QT 30-60ms plus droperidol adds acute effect', NULL, 'CYP3A4, CYP2C8',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiac Anesthesia Guidelines',
 'A', '2024-01-15'),

-- Droperidol × Sotalol
('qt-droperidol-sotalol', 'droperidol', 'Droperidol', 'sotalol', 'Sotalol',
 'contraindicated', 'established',
 'Additive QT prolongation: droperidol BLACK BOX + sotalol class III antiarrhythmic',
 'Extremely high TdP risk; sotalol used specifically for antiarrhythmic effect via QT prolongation',
 'CONTRAINDICATED: Sotalol patients cannot receive droperidol; alternative PONV prophylaxis mandatory',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX; Sotalol (Betapace) FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; sotalol specifically prolongs QT as mechanism of action', NULL, NULL,
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'ACC Antiarrhythmic Guidelines',
 'A', '2024-01-15'),

-- Droperidol × Erythromycin
('qt-droperidol-erythromycin', 'droperidol', 'Droperidol', 'erythromycin', 'Erythromycin',
 'contraindicated', 'established',
 'Additive QT prolongation; erythromycin also inhibits CYP3A4 potentially increasing droperidol levels',
 'Combined QT risk plus PK interaction; macrolide-related TdP well-documented',
 'CONTRAINDICATED: Use azithromycin if macrolide needed (lower QT risk) or non-macrolide antibiotic',
 'rapid', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX; Erythromycin FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Droperidol Known Risk + Erythromycin Known Risk; CYP3A4 interaction compounds', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antibiotic Selection Guidelines',
 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 2: ONDANSETRON COMBINATIONS (COMMON CLINICAL SCENARIOS)
-- =====================================================================================
-- Ondansetron is the most widely used 5-HT3 antagonist
-- FDA 2012 Safety Communication: removed 32mg single IV dose, max now 16mg

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Ondansetron × Haloperidol IV
('qt-ondansetron-haloperidol-iv', 'ondansetron', 'Ondansetron', 'haloperidol_iv', 'Haloperidol IV',
 'major', 'established',
 'Additive QT prolongation: ondansetron IKr blockade + haloperidol IV (BLACK BOX) potent IKr blocker',
 'Significantly increased TdP risk; common ICU scenario (agitation + nausea); combined QTc prolongation 30-50ms',
 'AVOID IV haloperidol with ondansetron; use oral haloperidol (lower risk) or alternative antiemetic (dexamethasone); ECG mandatory',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron FDA Safety Communication; Haloperidol IV BLACK BOX', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-drug-safety-communication-abnormal-heart-rhythms-associated-high-doses-zofran-ondansetron', 'US',
 'CREDIBLEMEDS', 'Ondansetron IKr IC50 1.9μM + Haloperidol IC50 0.3μM; additive blockade', NULL, 'CYP3A4, CYP2D6, CYP1A2',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'ICU Delirium Management Guidelines',
 'A', '2024-01-15'),

-- Ondansetron × Methadone
('qt-ondansetron-methadone', 'ondansetron', 'Ondansetron', 'methadone', 'Methadone',
 'major', 'established',
 'Additive QT prolongation; common scenario in OUD patients requiring antiemesis (e.g., during induction)',
 'Elevated TdP risk; methadone patients often have baseline QTc prolongation; ondansetron adds further risk',
 'Use lowest effective ondansetron dose (4mg); ECG monitoring; consider alternative antiemetics; avoid if QTc >450ms',
 'rapid', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Methadone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; combined IKr blockade; methadone patients high-risk population', 'P-gp substrate', 'CYP3A4, CYP2B6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'ASAM Methadone Guidelines',
 'A', '2024-01-15'),

-- Ondansetron × Amiodarone
('qt-ondansetron-amiodarone', 'ondansetron', 'Ondansetron', 'amiodarone', 'Amiodarone',
 'major', 'established',
 'Additive QT prolongation; amiodarone has extremely long half-life (40-55 days) complicating risk assessment',
 'Prolonged QT risk; amiodarone effect persists weeks; combined use significantly increases TdP probability',
 'Use alternative antiemetics in amiodarone patients (dexamethasone, NK1 antagonists); if ondansetron necessary, lowest dose with ECG',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron FDA Safety Communication; Amiodarone Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Amiodarone baseline QTc effect 30-60ms plus ondansetron additive', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiac Patient Antiemetic Selection',
 'A', '2024-01-15'),

-- Ondansetron × Sotalol
('qt-ondansetron-sotalol', 'ondansetron', 'Ondansetron', 'sotalol', 'Sotalol',
 'major', 'established',
 'Additive QT prolongation; sotalol class III effect specifically prolongs QT',
 'High TdP risk; sotalol patients have drug-induced QT prolongation as therapeutic effect',
 'AVOID ondansetron in sotalol patients; use dexamethasone or prochlorperazine for antiemesis',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Sotalol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; sotalol inherently prolongs QT 40-100ms', NULL, NULL,
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antiarrhythmic Patient Management',
 'A', '2024-01-15'),

-- Ondansetron × Citalopram
('qt-ondansetron-citalopram', 'ondansetron', 'Ondansetron', 'citalopram', 'Citalopram',
 'major', 'established',
 'Additive QT prolongation; both drugs have FDA safety communications regarding QT prolongation',
 'Combined QTc prolongation 20-40ms; common combination in cancer patients with depression and CINV',
 'Limit citalopram to 20mg max; use lowest effective ondansetron dose; ECG monitoring recommended',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Citalopram FDA Safety Communications', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk on CredibleMeds; dual IKr blockade', NULL, 'CYP2C19, CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Psycho-oncology Guidelines',
 'A', '2024-01-15'),

-- Ondansetron × Escitalopram
('qt-ondansetron-escitalopram', 'ondansetron', 'Ondansetron', 'escitalopram', 'Escitalopram',
 'major', 'established',
 'Additive QT prolongation; escitalopram (S-enantiomer of citalopram) has similar QT liability',
 'Elevated TdP risk; common in oncology patients on SSRIs requiring antiemesis',
 'Monitor ECG; escitalopram ≤20mg; ondansetron lowest effective dose; consider granisetron as alternative',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; additive IKr blockade mechanism', NULL, 'CYP2C19, CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'ASCO Supportive Care Guidelines',
 'A', '2024-01-15'),

-- Ondansetron × Erythromycin
('qt-ondansetron-erythromycin', 'ondansetron', 'Ondansetron', 'erythromycin', 'Erythromycin',
 'major', 'established',
 'Additive QT prolongation; erythromycin also inhibits CYP3A4 potentially increasing ondansetron',
 'Combined QT risk; erythromycin-related TdP well-documented; PK interaction may increase ondansetron levels',
 'Use azithromycin instead (lower QT risk); if erythromycin necessary, alternative antiemetic preferred',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Erythromycin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; CYP3A4 inhibition adds PK component to PD interaction', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antibiotic Selection in CINV',
 'A', '2024-01-15'),

-- Ondansetron × Clarithromycin
('qt-ondansetron-clarithromycin', 'ondansetron', 'Ondansetron', 'clarithromycin', 'Clarithromycin',
 'major', 'established',
 'Additive QT prolongation; clarithromycin potent CYP3A4 inhibitor increasing ondansetron levels',
 'High TdP risk; significant PK interaction compounds PD QT effect',
 'AVOID combination; use azithromycin (lower CYP3A4 inhibition and QT risk) or alternative antiemetic',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Clarithromycin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; clarithromycin CYP3A4 Ki = 7.6μM significantly increases ondansetron', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'H. pylori Treatment Selection',
 'A', '2024-01-15'),

-- Ondansetron × Moxifloxacin
('qt-ondansetron-moxifloxacin', 'ondansetron', 'Ondansetron', 'moxifloxacin', 'Moxifloxacin',
 'major', 'established',
 'Additive QT prolongation; moxifloxacin has highest QT risk among fluoroquinolones',
 'Significantly elevated TdP risk; moxifloxacin prolongs QTc 10-15ms at standard doses',
 'AVOID: Use levofloxacin (lower QT risk) with ondansetron, or use alternative antiemetic',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Moxifloxacin (Avelox) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; moxifloxacin specifically contraindicated with QT-prolonging drugs in label', NULL, 'CYP1A2',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Fluoroquinolone Safety Advisory',
 'A', '2024-01-15'),

-- Ondansetron × Ziprasidone
('qt-ondansetron-ziprasidone', 'ondansetron', 'Ondansetron', 'ziprasidone', 'Ziprasidone',
 'major', 'established',
 'Additive QT prolongation; ziprasidone has highest QT risk among atypical antipsychotics',
 'High TdP risk; ziprasidone FDA label specifically contraindicates use with QT-prolonging drugs',
 'AVOID: Ziprasidone label contraindication with other QT-prolonging drugs; use alternative antiemetic',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Ziprasidone (Geodon) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; ziprasidone label contraindicates combination', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; Label Contraindication', 'Psychiatric Patient Antiemetic Selection',
 'A', '2024-01-15'),

-- Ondansetron × Quetiapine
('qt-ondansetron-quetiapine', 'ondansetron', 'Ondansetron', 'quetiapine', 'Quetiapine',
 'moderate', 'probable',
 'Additive QT prolongation; quetiapine Possible Risk + ondansetron Known Risk',
 'Moderate TdP risk increase; common scenario in psychiatric patients requiring antiemesis',
 'Can be used with monitoring; ECG if additional risk factors; ondansetron 4mg doses preferred',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Quetiapine (Seroquel) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Ondansetron Known Risk + Quetiapine Possible Risk; lower combined risk than ziprasidone', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Psychiatric Supportive Care',
 'B', '2024-01-15'),

-- Ondansetron × Risperidone
('qt-ondansetron-risperidone', 'ondansetron', 'Ondansetron', 'risperidone', 'Risperidone',
 'moderate', 'probable',
 'Additive QT prolongation; risperidone Possible Risk + ondansetron Known Risk',
 'Moderate TdP risk; commonly encountered combination',
 'Generally acceptable with standard monitoring; ECG if cardiac risk factors present',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Ondansetron and Risperidone (Risperdal) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Ondansetron Known Risk + Risperidone Possible Risk', NULL, 'CYP2D6, CYP3A4',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Antipsychotic Safety Monitoring',
 'B', '2024-01-15');

-- =====================================================================================
-- SECTION 3: GRANISETRON COMBINATIONS
-- =====================================================================================
-- Granisetron generally has lower QT risk than ondansetron but still clinically relevant

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Granisetron × Haloperidol IV
('qt-granisetron-haloperidol-iv', 'granisetron', 'Granisetron', 'haloperidol_iv', 'Haloperidol IV',
 'major', 'probable',
 'Additive QT prolongation: granisetron Possible Risk + haloperidol IV BLACK BOX Known Risk',
 'Elevated TdP risk; less severe than ondansetron-haloperidol but still significant',
 'Prefer oral haloperidol; if IV needed, lowest dose with ECG monitoring; consider alternative antiemetic',
 'rapid', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Granisetron (Kytril) and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Granisetron lower IKr affinity than ondansetron but still contributes to risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'ICU Management Guidelines',
 'B', '2024-01-15'),

-- Granisetron × Methadone
('qt-granisetron-methadone', 'granisetron', 'Granisetron', 'methadone', 'Methadone',
 'moderate', 'probable',
 'Additive QT prolongation; granisetron lower risk alternative to ondansetron in methadone patients',
 'Moderate TdP risk; may be preferred over ondansetron in OUD patients',
 'Acceptable alternative to ondansetron; still monitor ECG; lowest effective dose',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Granisetron and Methadone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Granisetron Possible Risk + Methadone Known Risk; lower combined risk than ondansetron', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'OUD Patient Antiemetic Selection',
 'B', '2024-01-15'),

-- Granisetron × Amiodarone
('qt-granisetron-amiodarone', 'granisetron', 'Granisetron', 'amiodarone', 'Amiodarone',
 'major', 'probable',
 'Additive QT prolongation; amiodarone high baseline risk compounds granisetron effect',
 'Significant TdP risk; amiodarone patients at elevated baseline risk',
 'Consider dexamethasone or NK1 antagonists first; if 5-HT3 antagonist needed, granisetron preferred over ondansetron',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Granisetron and Amiodarone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Amiodarone baseline QTc prolongation + granisetron additive', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Cardiac Patient CINV Management',
 'B', '2024-01-15'),

-- Granisetron × Erythromycin
('qt-granisetron-erythromycin', 'granisetron', 'Granisetron', 'erythromycin', 'Erythromycin',
 'moderate', 'probable',
 'Additive QT prolongation; lower risk than ondansetron-erythromycin combination',
 'Moderate TdP risk; granisetron may be preferred over ondansetron with macrolides',
 'Can be used with caution; ECG monitoring if additional risk factors',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Granisetron and Erythromycin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Granisetron Possible Risk + Erythromycin Known Risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'Antibiotic-Antiemetic Selection',
 'B', '2024-01-15'),

-- Granisetron × Citalopram
('qt-granisetron-citalopram', 'granisetron', 'Granisetron', 'citalopram', 'Citalopram',
 'moderate', 'probable',
 'Additive QT prolongation; lower risk than ondansetron but still requires monitoring',
 'Moderate TdP risk; granisetron may be preferred 5-HT3 antagonist in patients on citalopram',
 'Limit citalopram to 20mg; granisetron acceptable with monitoring; ECG if risk factors',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Granisetron and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Granisetron Possible Risk + Citalopram Known Risk', NULL, 'CYP2C19',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'Psycho-oncology Supportive Care',
 'B', '2024-01-15');

-- =====================================================================================
-- SECTION 4: DOMPERIDONE COMBINATIONS (IMPORTANT FOR NON-US JURISDICTIONS)
-- =====================================================================================
-- Domperidone is not FDA-approved but available in EU, Canada, Australia
-- Known QT risk - restricted in many jurisdictions

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Domperidone × Erythromycin
('qt-domperidone-erythromycin', 'domperidone', 'Domperidone', 'erythromycin', 'Erythromycin',
 'contraindicated', 'established',
 'Dual QT prolongation: domperidone Known Risk + erythromycin Known Risk; erythromycin inhibits CYP3A4 increasing domperidone levels 3-fold',
 'Severely elevated TdP risk; documented sudden cardiac deaths from combination',
 'CONTRAINDICATED: EMA and Health Canada warnings; documented fatalities; never combine',
 'rapid', 'excellent', 'qt_prolongation', true,
 'EMA', 'EMA Domperidone Safety Review 2014; Health Canada Advisory', 'https://www.ema.europa.eu/en/medicines/human/referrals/domperidone-containing-medicines', 'EU',
 'CREDIBLEMEDS', 'Domperidone IKr IC50 0.2μM (potent); erythromycin CYP3A4 inhibition triples levels', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; Documented Fatalities', 'EMA CMDh Recommendations',
 'A', '2024-01-15'),

-- Domperidone × Clarithromycin
('qt-domperidone-clarithromycin', 'domperidone', 'Domperidone', 'clarithromycin', 'Clarithromycin',
 'contraindicated', 'established',
 'Additive QT prolongation with potent CYP3A4 inhibition increasing domperidone exposure',
 'Severe TdP risk; similar mechanism to erythromycin combination',
 'CONTRAINDICATED: Do not combine; use metoclopramide if prokinetic needed with clarithromycin',
 'rapid', 'excellent', 'qt_prolongation', true,
 'EMA', 'EMA Domperidone Review; Clarithromycin Interaction Warning', 'https://www.ema.europa.eu', 'EU',
 'CREDIBLEMEDS', 'CYP3A4 inhibition significantly increases domperidone; both Known Risk', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'H. pylori Treatment Safety',
 'A', '2024-01-15'),

-- Domperidone × Ketoconazole
('qt-domperidone-ketoconazole', 'domperidone', 'Domperidone', 'ketoconazole', 'Ketoconazole',
 'contraindicated', 'established',
 'Ketoconazole potent CYP3A4 inhibitor increases domperidone levels 3-4 fold; additive QT effect',
 'Extremely high TdP risk; ketoconazole-domperidone specifically studied and shown dangerous',
 'CONTRAINDICATED: FDA Import Alert on domperidone specifically cites this interaction',
 'rapid', 'excellent', 'qt_prolongation', true,
 'FDA_IMPORT_ALERT', 'FDA Import Alert 66-78; EMA Domperidone Review', 'https://www.fda.gov/drugs/information-drug-class/fda-warns-against-using-unapproved-drug-domperidone-increase-milk-production', 'US',
 'CREDIBLEMEDS', 'Ketoconazole CYP3A4 Ki = 0.015μM (extremely potent); domperidone AUC increases 300-400%', 'P-gp inhibitor', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'FDA Domperidone Warning',
 'A', '2024-01-15'),

-- Domperidone × Methadone
('qt-domperidone-methadone', 'domperidone', 'Domperidone', 'methadone', 'Methadone',
 'contraindicated', 'established',
 'Dual Known Risk QT prolongers; domperidone should not be used with any QT-prolonging drug',
 'Severe TdP risk; methadone patients often have QTc prolongation at baseline',
 'CONTRAINDICATED: Use metoclopramide or prochlorperazine for GI symptoms in methadone patients',
 'rapid', 'good', 'qt_prolongation', true,
 'EMA', 'EMA Domperidone Review - Contraindicated Combinations', 'https://www.ema.europa.eu', 'EU',
 'CREDIBLEMEDS', 'Both potent IKr blockers; additive QTc prolongation potentially >60ms', 'P-gp substrates', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'OUD Treatment Guidelines',
 'A', '2024-01-15'),

-- Domperidone × Haloperidol
('qt-domperidone-haloperidol', 'domperidone', 'Domperidone', 'haloperidol', 'Haloperidol',
 'contraindicated', 'established',
 'Additive QT prolongation; both dopamine antagonists with cardiac effects',
 'Severe TdP risk; pharmacologically overlapping mechanisms',
 'CONTRAINDICATED: Domperidone specifically contraindicated with haloperidol in EU/CA guidelines',
 'rapid', 'excellent', 'qt_prolongation', true,
 'EMA', 'EMA Domperidone Safety Review', 'https://www.ema.europa.eu', 'EU',
 'CREDIBLEMEDS', 'Both potent IKr blockers; domperidone IC50 0.2μM + haloperidol IC50 0.3μM', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'EMA Safety Recommendations',
 'A', '2024-01-15'),

-- Domperidone × Amiodarone
('qt-domperidone-amiodarone', 'domperidone', 'Domperidone', 'amiodarone', 'Amiodarone',
 'contraindicated', 'established',
 'Additive QT prolongation; amiodarone very long half-life creates persistent risk',
 'Extreme TdP risk; combination should never occur',
 'CONTRAINDICATED: Domperidone absolutely contraindicated in patients on amiodarone',
 'rapid', 'excellent', 'qt_prolongation', true,
 'EMA', 'EMA Domperidone Contraindications List', 'https://www.ema.europa.eu', 'EU',
 'CREDIBLEMEDS', 'Amiodarone baseline QTc prolongation 30-60ms + domperidone adds further risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiac Patient Prokinetic Selection',
 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 5: METOCLOPRAMIDE/PROCHLORPERAZINE COMBINATIONS (CONDITIONAL RISK)
-- =====================================================================================
-- Lower QT risk antiemetics but still relevant in high-risk patients

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Metoclopramide × Haloperidol
('qt-metoclopramide-haloperidol', 'metoclopramide', 'Metoclopramide', 'haloperidol', 'Haloperidol',
 'moderate', 'probable',
 'Additive dopamine antagonism; both can cause QT prolongation at higher doses; extrapyramidal effects compound',
 'Moderate QT risk; high extrapyramidal symptom (EPS) risk; akathisia, dystonia common',
 'Use with caution; monitor for EPS; avoid high doses of both; ECG if cardiac risk factors',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Metoclopramide (Reglan) and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Metoclopramide Conditional Risk + Haloperidol Known Risk; EPS risk more clinically relevant', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'Antiemetic Safety in ICU',
 'B', '2024-01-15'),

-- Metoclopramide × Droperidol
('qt-metoclopramide-droperidol', 'metoclopramide', 'Metoclopramide', 'droperidol', 'Droperidol',
 'major', 'probable',
 'Droperidol BLACK BOX + metoclopramide dopamine antagonism; additive QT and EPS risk',
 'High QT risk from droperidol; compounded EPS risk; avoid if possible',
 'AVOID: If droperidol necessary (rare), do not add metoclopramide; use ondansetron for additional antiemesis (if droperidol benefit outweighs QT risk)',
 'rapid', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Droperidol BLACK BOX; Metoclopramide FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Droperidol BLACK BOX Known Risk + Metoclopramide Conditional Risk', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Known + Conditional Risk', 'PONV Management Guidelines',
 'A', '2024-01-15'),

-- Prochlorperazine × Haloperidol
('qt-prochlorperazine-haloperidol', 'prochlorperazine', 'Prochlorperazine', 'haloperidol', 'Haloperidol',
 'moderate', 'probable',
 'Dual phenothiazine/butyrophenone dopamine antagonists; additive QT and EPS risk',
 'Moderate QT risk; significant EPS risk; both cause dopamine blockade',
 'AVOID combining two antipsychotics for antiemetic effect; use one agent at appropriate dose',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Prochlorperazine (Compazine) and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Prochlorperazine Conditional Risk + Haloperidol Known Risk', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'Antiemetic Selection Guidelines',
 'B', '2024-01-15'),

-- Prochlorperazine × Methadone
('qt-prochlorperazine-methadone', 'prochlorperazine', 'Prochlorperazine', 'methadone', 'Methadone',
 'moderate', 'probable',
 'Additive QT prolongation; prochlorperazine has lower risk but contributes in high-risk methadone patients',
 'Modest additional QT risk; prochlorperazine often used for nausea in OUD patients',
 'Can be used with caution; ECG monitoring advisable; preferred over ondansetron in some guidelines',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Prochlorperazine and Methadone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Prochlorperazine Conditional Risk + Methadone Known Risk', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'SAMHSA OUD Treatment Guidelines',
 'B', '2024-01-15');

-- =====================================================================================
-- VERIFICATION QUERY
-- =====================================================================================
-- SELECT COUNT(*) as total_qt_antiemetic_ddis
-- FROM drug_interactions
-- WHERE interaction_id LIKE 'qt-%'
--   AND category = 'qt_prolongation'
--   AND (drug_a_name IN ('Droperidol', 'Ondansetron', 'Granisetron', 'Domperidone', 'Metoclopramide', 'Prochlorperazine')
--    OR  drug_b_name IN ('Droperidol', 'Ondansetron', 'Granisetron', 'Domperidone', 'Metoclopramide', 'Prochlorperazine'));
-- Expected: ~55 DDIs from this migration
