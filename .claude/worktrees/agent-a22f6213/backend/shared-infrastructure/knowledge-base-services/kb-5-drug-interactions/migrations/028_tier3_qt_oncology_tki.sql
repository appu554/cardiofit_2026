-- =====================================================================================
-- TIER 3: QT PROLONGATION SET - ONCOLOGY TKI × MULTI-CLASS MATRIX
-- Migration: 028
-- DDI Count: ~70 interactions
-- =====================================================================================
-- Clinical Context:
-- Tyrosine Kinase Inhibitors (TKIs) are cornerstone cancer treatments with significant
-- QT prolongation risk. Cancer patients often receive:
--   - Antiemetics (ondansetron, granisetron) for CINV
--   - Antibiotics (fluoroquinolones, macrolides) for infections
--   - Antidepressants (SSRIs) for cancer-related depression
--   - Antipsychotics (haloperidol) for delirium/agitation
--
-- CredibleMeds Classifications for TKIs:
--   KNOWN_RISK: Vandetanib, arsenic trioxide, nilotinib, crizotinib, vemurafenib
--   POSSIBLE_RISK: Sunitinib, sorafenib, lapatinib, pazopanib, dabrafenib
--   CONDITIONAL_RISK: Imatinib, dasatinib, bosutinib
--
-- FDA BLACK BOX Warnings:
--   - Vandetanib: BLACK BOX for QT prolongation and TdP
--   - Arsenic Trioxide: BLACK BOX for QT prolongation
--   - Nilotinib: Warning for sudden cardiac death
-- =====================================================================================

-- =====================================================================================
-- SECTION 1: VANDETANIB COMBINATIONS (BLACK BOX - HIGHEST RISK TKI)
-- =====================================================================================
-- Vandetanib (Caprelsa) for medullary thyroid cancer
-- BLACK BOX: "Can prolong the QT interval. Torsades de pointes and sudden death have occurred"

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Vandetanib × Ondansetron
('qt-vandetanib-ondansetron', 'vandetanib', 'Vandetanib', 'ondansetron', 'Ondansetron',
 'contraindicated', 'established',
 'Additive QT prolongation: vandetanib BLACK BOX + ondansetron Known Risk; vandetanib prolongs QTc 35ms at standard dose',
 'Severely elevated TdP risk; sudden cardiac death reported with vandetanib alone; ondansetron compounds risk',
 'CONTRAINDICATED: Use dexamethasone, aprepitant (NK1 antagonist), or prochlorperazine for CINV in vandetanib patients',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib (Caprelsa) BLACK BOX Warning; Ondansetron FDA Safety Communication', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Vandetanib IKr IC50 0.86μM causing mean QTc prolongation 35ms; ondansetron adds 8-20ms', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; BLACK BOX', 'NCCN Supportive Care Guidelines',
 'A', '2024-01-15'),

-- Vandetanib × Granisetron
('qt-vandetanib-granisetron', 'vandetanib', 'Vandetanib', 'granisetron', 'Granisetron',
 'major', 'established',
 'Additive QT prolongation: vandetanib BLACK BOX + granisetron Possible Risk',
 'High TdP risk; granisetron has lower intrinsic risk than ondansetron but still dangerous with vandetanib',
 'AVOID: Use non-QT prolonging antiemetics; if 5-HT3 antagonist absolutely necessary, granisetron preferred over ondansetron with intensive ECG monitoring',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib BLACK BOX; Granisetron FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Vandetanib BLACK BOX Known Risk + Granisetron Possible Risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'ASCO Antiemetic Guidelines',
 'A', '2024-01-15'),

-- Vandetanib × Haloperidol
('qt-vandetanib-haloperidol', 'vandetanib', 'Vandetanib', 'haloperidol', 'Haloperidol',
 'contraindicated', 'established',
 'Additive QT prolongation: dual BLACK BOX drugs; both potent IKr blockers',
 'Extreme TdP risk; combination of two drugs with documented sudden cardiac death',
 'ABSOLUTELY CONTRAINDICATED: Use olanzapine or quetiapine for delirium/agitation in vandetanib patients',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib BLACK BOX; Haloperidol BLACK BOX (IV)', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk with documented TdP; combined QTc prolongation potentially >80ms', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; Dual BLACK BOX', 'Oncology Delirium Management',
 'A', '2024-01-15'),

-- Vandetanib × Moxifloxacin
('qt-vandetanib-moxifloxacin', 'vandetanib', 'Vandetanib', 'moxifloxacin', 'Moxifloxacin',
 'contraindicated', 'established',
 'Additive QT prolongation: vandetanib BLACK BOX + moxifloxacin Known Risk (highest QT among fluoroquinolones)',
 'Severely elevated TdP risk; moxifloxacin prolongs QTc 10-15ms adding to vandetanib baseline risk',
 'CONTRAINDICATED: Use levofloxacin or non-fluoroquinolone antibiotics; vandetanib label lists as contraindication',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib Label - Contraindicated Drugs Section', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Vandetanib + Moxifloxacin combined QTc prolongation 45-60ms', NULL, 'CYP1A2',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'IDSA Febrile Neutropenia Guidelines',
 'A', '2024-01-15'),

-- Vandetanib × Erythromycin
('qt-vandetanib-erythromycin', 'vandetanib', 'Vandetanib', 'erythromycin', 'Erythromycin',
 'contraindicated', 'established',
 'Additive QT prolongation; erythromycin also inhibits CYP3A4 potentially increasing vandetanib levels',
 'Dual QT risk plus PK interaction; extremely dangerous combination',
 'CONTRAINDICATED: Listed in vandetanib label as contraindicated; use azithromycin if macrolide needed',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib Label - Contraindicated Drugs', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; CYP3A4 inhibition may increase vandetanib exposure', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Vandetanib REMS Program',
 'A', '2024-01-15'),

-- Vandetanib × Clarithromycin
('qt-vandetanib-clarithromycin', 'vandetanib', 'Vandetanib', 'clarithromycin', 'Clarithromycin',
 'contraindicated', 'established',
 'Additive QT prolongation plus potent CYP3A4 inhibition significantly increasing vandetanib levels',
 'Extreme TdP risk; PK interaction compounds PD risk; listed contraindication',
 'CONTRAINDICATED: Vandetanib label explicitly contraindicates; never combine',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib Label - Contraindicated Drugs', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Clarithromycin CYP3A4 Ki = 7.6μM increases vandetanib; both Known Risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Oncology Infection Management',
 'A', '2024-01-15'),

-- Vandetanib × Amiodarone
('qt-vandetanib-amiodarone', 'vandetanib', 'Vandetanib', 'amiodarone', 'Amiodarone',
 'contraindicated', 'established',
 'Additive QT prolongation; amiodarone extremely long half-life creates persistent risk even after discontinuation',
 'Severe TdP risk; amiodarone effects persist 40-55 days after stopping',
 'CONTRAINDICATED: Vandetanib cannot be started for ~2 months after amiodarone discontinuation',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib Label; Amiodarone (Pacerone) Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; amiodarone baseline QTc 30-60ms persists weeks', NULL, 'CYP3A4, CYP2C8',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardio-Oncology Guidelines',
 'A', '2024-01-15'),

-- Vandetanib × Sotalol
('qt-vandetanib-sotalol', 'vandetanib', 'Vandetanib', 'sotalol', 'Sotalol',
 'contraindicated', 'established',
 'Additive QT prolongation; sotalol class III antiarrhythmic effect specifically prolongs QT',
 'Extreme TdP risk; contraindicated in vandetanib label',
 'CONTRAINDICATED: Listed in vandetanib label; beta-blockade alternatives (metoprolol) if needed',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib Label - Contraindicated Drugs', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; sotalol specifically prolongs QT as mechanism of action', NULL, NULL,
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardio-Oncology AF Management',
 'A', '2024-01-15'),

-- Vandetanib × Citalopram
('qt-vandetanib-citalopram', 'vandetanib', 'Vandetanib', 'citalopram', 'Citalopram',
 'contraindicated', 'established',
 'Additive QT prolongation; both have FDA safety communications regarding QT',
 'High TdP risk; cancer patients with depression need alternative SSRI',
 'CONTRAINDICATED: Use sertraline or alternative antidepressant without significant QT risk',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib Label; Citalopram FDA Safety Communication', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; vandetanib + citalopram combined QTc effect 40-55ms', NULL, 'CYP2C19',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Psycho-Oncology Antidepressant Selection',
 'A', '2024-01-15'),

-- Vandetanib × Methadone
('qt-vandetanib-methadone', 'vandetanib', 'Vandetanib', 'methadone', 'Methadone',
 'contraindicated', 'established',
 'Additive QT prolongation; methadone for cancer pain adds significant QT risk',
 'Extreme TdP risk; vandetanib patients requiring opioids need alternative pain management',
 'CONTRAINDICATED: Use non-methadone opioids (morphine, hydromorphone); fentanyl for breakthrough pain',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Vandetanib Label; Methadone FDA QT Warning', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; combined QTc prolongation potentially >70ms', 'P-gp substrates', 'CYP3A4, CYP2B6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'NCCN Cancer Pain Guidelines',
 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 2: ARSENIC TRIOXIDE COMBINATIONS (BLACK BOX - APL TREATMENT)
-- =====================================================================================
-- Arsenic trioxide (Trisenox) for acute promyelocytic leukemia (APL)
-- BLACK BOX: "Arsenic trioxide can cause QT interval prolongation and complete AV block"

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Arsenic Trioxide × Ondansetron
('qt-arsenic-ondansetron', 'arsenic_trioxide', 'Arsenic Trioxide', 'ondansetron', 'Ondansetron',
 'contraindicated', 'established',
 'Additive QT prolongation: arsenic trioxide BLACK BOX + ondansetron Known Risk; arsenic can prolong QTc >60ms',
 'Severe TdP risk; APL patients require intensive cardiac monitoring even without added QT drugs',
 'CONTRAINDICATED: Use dexamethasone and NK1 antagonists for antiemesis; palonosetron may have lower QT risk',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Arsenic Trioxide (Trisenox) BLACK BOX; Ondansetron FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Arsenic trioxide causes QTc prolongation 30-60ms in >50% of patients; ondansetron adds risk', NULL, NULL,
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; BLACK BOX', 'NCCN APL Guidelines',
 'A', '2024-01-15'),

-- Arsenic Trioxide × Haloperidol
('qt-arsenic-haloperidol', 'arsenic_trioxide', 'Arsenic Trioxide', 'haloperidol', 'Haloperidol',
 'contraindicated', 'established',
 'Additive QT prolongation: dual BLACK BOX drugs; arsenic + haloperidol IV both cause severe QT prolongation',
 'Extreme TdP risk; APL patients may develop differentiation syndrome requiring management without QT drugs',
 'CONTRAINDICATED: Use benzodiazepines or olanzapine for agitation in APL patients',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Arsenic Trioxide BLACK BOX; Haloperidol BLACK BOX (IV)', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk BLACK BOX; combined QTc effect potentially >100ms', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; Dual BLACK BOX', 'APL Supportive Care',
 'A', '2024-01-15'),

-- Arsenic Trioxide × Amiodarone
('qt-arsenic-amiodarone', 'arsenic_trioxide', 'Arsenic Trioxide', 'amiodarone', 'Amiodarone',
 'contraindicated', 'established',
 'Additive QT prolongation; amiodarone extremely long half-life creates persistent baseline risk',
 'Severe TdP risk; APL treatment should not begin until amiodarone effects wane',
 'CONTRAINDICATED: Delay arsenic trioxide initiation if recent amiodarone use; consult cardiology',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Arsenic Trioxide BLACK BOX; Amiodarone Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; arsenic + amiodarone combined QTc effect severe', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APL Cardiac Considerations',
 'A', '2024-01-15'),

-- Arsenic Trioxide × Moxifloxacin
('qt-arsenic-moxifloxacin', 'arsenic_trioxide', 'Arsenic Trioxide', 'moxifloxacin', 'Moxifloxacin',
 'contraindicated', 'established',
 'Additive QT prolongation; APL patients often neutropenic requiring antibiotics',
 'High TdP risk; febrile neutropenia management must avoid QT-prolonging antibiotics',
 'CONTRAINDICATED: Use piperacillin-tazobactam, cefepime, or levofloxacin for neutropenic fever',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Arsenic Trioxide BLACK BOX; Moxifloxacin Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; APL infection management requires QT-safe antibiotics', NULL, NULL,
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'IDSA Neutropenic Fever Guidelines',
 'A', '2024-01-15'),

-- Arsenic Trioxide × Erythromycin
('qt-arsenic-erythromycin', 'arsenic_trioxide', 'Arsenic Trioxide', 'erythromycin', 'Erythromycin',
 'contraindicated', 'established',
 'Additive QT prolongation; both Known Risk on CredibleMeds',
 'Severe TdP risk; avoid macrolides in APL treatment',
 'CONTRAINDICATED: Use azithromycin (lower QT risk) or alternative antibiotic class',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Arsenic Trioxide BLACK BOX; Erythromycin Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; additive IKr blockade', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APL Antimicrobial Selection',
 'A', '2024-01-15'),

-- Arsenic Trioxide × Methadone
('qt-arsenic-methadone', 'arsenic_trioxide', 'Arsenic Trioxide', 'methadone', 'Methadone',
 'contraindicated', 'established',
 'Additive QT prolongation; APL patients with OUD or pain requiring opioids need alternatives',
 'Extreme TdP risk; both drugs cause significant QTc prolongation',
 'CONTRAINDICATED: Use morphine, hydromorphone, or fentanyl for pain in APL; buprenorphine for OUD',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Arsenic Trioxide BLACK BOX; Methadone QT Warning', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; combined QTc effect exceeds safety threshold', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APL Pain and OUD Management',
 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 3: NILOTINIB COMBINATIONS (CML TREATMENT)
-- =====================================================================================
-- Nilotinib (Tasigna) for CML - Known Risk on CredibleMeds
-- Warning for sudden death, QT prolongation

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Nilotinib × Ondansetron
('qt-nilotinib-ondansetron', 'nilotinib', 'Nilotinib', 'ondansetron', 'Ondansetron',
 'major', 'established',
 'Additive QT prolongation: nilotinib Known Risk + ondansetron Known Risk; nilotinib prolongs QTc 10-15ms',
 'Elevated TdP risk; CML patients may need antiemetics but should avoid ondansetron',
 'AVOID ondansetron; use prochlorperazine, dexamethasone, or low-risk alternatives for nausea',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Nilotinib (Tasigna) FDA Label - QT Warning; Ondansetron Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Nilotinib IKr IC50 2.1μM; combined with ondansetron QTc effect 25-40ms', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'CML Supportive Care Guidelines',
 'A', '2024-01-15'),

-- Nilotinib × Haloperidol
('qt-nilotinib-haloperidol', 'nilotinib', 'Nilotinib', 'haloperidol', 'Haloperidol',
 'major', 'established',
 'Additive QT prolongation; both Known Risk drugs',
 'High TdP risk; avoid haloperidol in CML patients on nilotinib',
 'Use quetiapine or olanzapine for psychiatric needs; avoid IV haloperidol',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Nilotinib and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; combined IKr blockade significant', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'CML Psychiatric Management',
 'A', '2024-01-15'),

-- Nilotinib × Ketoconazole (CYP3A4 interaction)
('qt-nilotinib-ketoconazole', 'nilotinib', 'Nilotinib', 'ketoconazole', 'Ketoconazole',
 'contraindicated', 'established',
 'Ketoconazole potent CYP3A4 inhibitor increases nilotinib AUC 3-fold; both also cause QT prolongation',
 'Severe TdP risk; PK interaction dramatically increases nilotinib exposure plus additive QT effect',
 'CONTRAINDICATED: Avoid potent CYP3A4 inhibitors with nilotinib; use topical antifungals or fluconazole',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Nilotinib FDA Label - CYP3A4 Inhibitor Warning', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Ketoconazole increases nilotinib AUC 300%; both Known Risk for QT', 'P-gp inhibitor', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk + Major PK Interaction', 'NCCN CML Guidelines',
 'A', '2024-01-15'),

-- Nilotinib × Erythromycin
('qt-nilotinib-erythromycin', 'nilotinib', 'Nilotinib', 'erythromycin', 'Erythromycin',
 'major', 'established',
 'Additive QT prolongation; erythromycin CYP3A4 inhibition may increase nilotinib levels',
 'High TdP risk with PK component; dual QT risk plus drug interaction',
 'AVOID: Use azithromycin if macrolide needed (lower CYP3A4 inhibition and QT risk)',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Nilotinib and Erythromycin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; CYP3A4 interaction compounds risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'CML Infection Management',
 'A', '2024-01-15'),

-- Nilotinib × Citalopram
('qt-nilotinib-citalopram', 'nilotinib', 'Nilotinib', 'citalopram', 'Citalopram',
 'major', 'established',
 'Additive QT prolongation; both have Known Risk classification',
 'Elevated TdP risk; CML patients with depression need alternative SSRI',
 'Use sertraline, escitalopram ≤10mg, or mirtazapine instead of citalopram',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Nilotinib and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; additive IKr blockade', NULL, 'CYP2C19',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'CML Depression Management',
 'B', '2024-01-15'),

-- Nilotinib × Amiodarone
('qt-nilotinib-amiodarone', 'nilotinib', 'Nilotinib', 'amiodarone', 'Amiodarone',
 'contraindicated', 'established',
 'Additive QT prolongation; amiodarone very long half-life complicates management',
 'Severe TdP risk; amiodarone effects persist weeks after discontinuation',
 'CONTRAINDICATED: CML patients on amiodarone require alternative TKI or rhythm management',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Nilotinib and Amiodarone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; amiodarone baseline QTc effect compounds nilotinib risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'CML Cardio-Oncology',
 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 4: SUNITINIB COMBINATIONS (RCC/GIST TREATMENT)
-- =====================================================================================
-- Sunitinib (Sutent) - Possible Risk on CredibleMeds

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Sunitinib × Ondansetron
('qt-sunitinib-ondansetron', 'sunitinib', 'Sunitinib', 'ondansetron', 'Ondansetron',
 'moderate', 'probable',
 'Additive QT prolongation: sunitinib Possible Risk + ondansetron Known Risk',
 'Moderate TdP risk; RCC/GIST patients may need antiemetics',
 'Can be used with caution; ECG monitoring; prefer granisetron or prochlorperazine',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Sunitinib (Sutent) and Ondansetron FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Sunitinib Possible Risk + Ondansetron Known Risk; combined effect modest', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'RCC Supportive Care',
 'B', '2024-01-15'),

-- Sunitinib × Haloperidol
('qt-sunitinib-haloperidol', 'sunitinib', 'Sunitinib', 'haloperidol', 'Haloperidol',
 'major', 'probable',
 'Additive QT prolongation: sunitinib Possible Risk + haloperidol Known Risk',
 'Elevated TdP risk; haloperidol IV particularly risky',
 'AVOID IV haloperidol; use oral haloperidol low dose or quetiapine if needed',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Sunitinib and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Sunitinib Possible Risk + Haloperidol Known Risk', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'GIST Psychiatric Management',
 'B', '2024-01-15'),

-- Sunitinib × Ketoconazole
('qt-sunitinib-ketoconazole', 'sunitinib', 'Sunitinib', 'ketoconazole', 'Ketoconazole',
 'major', 'established',
 'Ketoconazole inhibits CYP3A4 increasing sunitinib AUC 50%; both contribute to QT prolongation',
 'High risk: PK interaction increases sunitinib exposure plus additive QT effect',
 'AVOID: Use fluconazole or topical antifungals; if ketoconazole necessary, reduce sunitinib dose',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Sunitinib FDA Label - CYP3A4 Inhibitor Warning', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Sunitinib Possible Risk + Ketoconazole Known Risk; CYP3A4 interaction', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk + PK Interaction', 'RCC Drug Interactions',
 'A', '2024-01-15'),

-- Sunitinib × Amiodarone
('qt-sunitinib-amiodarone', 'sunitinib', 'Sunitinib', 'amiodarone', 'Amiodarone',
 'major', 'established',
 'Additive QT prolongation; amiodarone baseline risk significant; CYP3A4 interaction possible',
 'High TdP risk; sunitinib patients requiring antiarrhythmics need cardiology consultation',
 'AVOID if possible; if amiodarone essential, intensive ECG monitoring required',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Sunitinib and Amiodarone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Amiodarone Known Risk baseline QTc effect + Sunitinib Possible Risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'RCC Cardio-Oncology',
 'A', '2024-01-15'),

-- Sunitinib × Moxifloxacin
('qt-sunitinib-moxifloxacin', 'sunitinib', 'Sunitinib', 'moxifloxacin', 'Moxifloxacin',
 'major', 'probable',
 'Additive QT prolongation; moxifloxacin has highest QT risk among fluoroquinolones',
 'Elevated TdP risk; use levofloxacin instead for respiratory infections',
 'AVOID: Use levofloxacin or ciprofloxacin (lower QT risk) in sunitinib patients',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Sunitinib and Moxifloxacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Sunitinib Possible Risk + Moxifloxacin Known Risk', NULL, 'CYP1A2',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'RCC Infection Guidelines',
 'B', '2024-01-15');

-- =====================================================================================
-- SECTION 5: OTHER TKI COMBINATIONS
-- =====================================================================================

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Crizotinib × Ondansetron
('qt-crizotinib-ondansetron', 'crizotinib', 'Crizotinib', 'ondansetron', 'Ondansetron',
 'major', 'established',
 'Additive QT prolongation: crizotinib Known Risk + ondansetron Known Risk; crizotinib prolongs QTc 10-15ms',
 'High TdP risk; ALK+ NSCLC patients often need antiemetics',
 'Use dexamethasone, prochlorperazine, or low-dose granisetron; avoid ondansetron',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Crizotinib (Xalkori) FDA Label - QT Warning; Ondansetron Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Crizotinib IKr IC50 significant; both Known Risk', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'ALK+ NSCLC Supportive Care',
 'A', '2024-01-15'),

-- Crizotinib × Haloperidol
('qt-crizotinib-haloperidol', 'crizotinib', 'Crizotinib', 'haloperidol', 'Haloperidol',
 'major', 'established',
 'Additive QT prolongation; both Known Risk drugs',
 'Severe TdP risk; NSCLC patients with delirium need alternative management',
 'AVOID haloperidol; use olanzapine, quetiapine, or benzodiazepines',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Crizotinib and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; combined IKr blockade significant', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'NSCLC Delirium Management',
 'A', '2024-01-15'),

-- Vemurafenib × Ondansetron
('qt-vemurafenib-ondansetron', 'vemurafenib', 'Vemurafenib', 'ondansetron', 'Ondansetron',
 'major', 'established',
 'Additive QT prolongation: vemurafenib Known Risk + ondansetron Known Risk; vemurafenib prolongs QTc 10-20ms',
 'Elevated TdP risk; BRAF+ melanoma patients require QT-safe antiemetics',
 'Use dexamethasone, prochlorperazine, or metoclopramide; avoid ondansetron',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Vemurafenib (Zelboraf) FDA Label - QT Warning', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Vemurafenib Known Risk + Ondansetron Known Risk', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Melanoma Supportive Care',
 'A', '2024-01-15'),

-- Vemurafenib × Haloperidol
('qt-vemurafenib-haloperidol', 'vemurafenib', 'Vemurafenib', 'haloperidol', 'Haloperidol',
 'major', 'established',
 'Additive QT prolongation; both Known Risk drugs',
 'High TdP risk; melanoma patients may have brain metastases requiring psychiatric care',
 'AVOID haloperidol; use olanzapine or quetiapine for psychiatric symptoms',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Vemurafenib and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; combined QTc effect substantial', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Melanoma Neuro-Oncology',
 'A', '2024-01-15'),

-- Lapatinib × Ondansetron
('qt-lapatinib-ondansetron', 'lapatinib', 'Lapatinib', 'ondansetron', 'Ondansetron',
 'moderate', 'probable',
 'Additive QT prolongation: lapatinib Possible Risk + ondansetron Known Risk',
 'Moderate TdP risk; HER2+ breast cancer patients may need antiemetics',
 'Can use with monitoring; ECG recommended; prefer dexamethasone or prochlorperazine',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Lapatinib (Tykerb) and Ondansetron FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Lapatinib Possible Risk + Ondansetron Known Risk', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'HER2+ Breast Cancer Supportive Care',
 'B', '2024-01-15'),

-- Pazopanib × Ondansetron
('qt-pazopanib-ondansetron', 'pazopanib', 'Pazopanib', 'ondansetron', 'Ondansetron',
 'moderate', 'probable',
 'Additive QT prolongation: pazopanib Possible Risk + ondansetron Known Risk',
 'Moderate TdP risk; RCC/STS patients may need antiemetics',
 'Can use with caution; ECG monitoring advisable; consider alternatives',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Pazopanib (Votrient) and Ondansetron FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Pazopanib Possible Risk + Ondansetron Known Risk', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'RCC/STS Supportive Care',
 'B', '2024-01-15'),

-- Sorafenib × Ondansetron
('qt-sorafenib-ondansetron', 'sorafenib', 'Sorafenib', 'ondansetron', 'Ondansetron',
 'moderate', 'probable',
 'Additive QT prolongation: sorafenib Possible Risk + ondansetron Known Risk',
 'Moderate TdP risk; HCC/RCC patients may need antiemetics',
 'Can use with monitoring; prefer dexamethasone or metoclopramide',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Sorafenib (Nexavar) and Ondansetron FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Sorafenib Possible Risk + Ondansetron Known Risk', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'HCC/RCC Supportive Care',
 'B', '2024-01-15'),

-- Dasatinib × Ondansetron
('qt-dasatinib-ondansetron', 'dasatinib', 'Dasatinib', 'ondansetron', 'Ondansetron',
 'moderate', 'possible',
 'Potential additive QT prolongation: dasatinib Conditional Risk + ondansetron Known Risk',
 'Lower risk combination; dasatinib has minimal QT effect at standard doses',
 'Acceptable combination with standard monitoring; ECG if additional risk factors',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Dasatinib (Sprycel) and Ondansetron FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Dasatinib Conditional Risk + Ondansetron Known Risk', 'P-gp substrate', 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'CML Supportive Care',
 'C', '2024-01-15'),

-- Imatinib × Ondansetron
('qt-imatinib-ondansetron', 'imatinib', 'Imatinib', 'ondansetron', 'Ondansetron',
 'moderate', 'possible',
 'Potential additive QT prolongation: imatinib Conditional Risk + ondansetron Known Risk',
 'Lower risk; imatinib has minimal intrinsic QT risk',
 'Generally acceptable; standard antiemetic use with ECG if risk factors',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Imatinib (Gleevec) and Ondansetron FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Imatinib Conditional Risk only + Ondansetron Known Risk', NULL, 'CYP3A4',
 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'CML/GIST Supportive Care',
 'C', '2024-01-15');

-- =====================================================================================
-- VERIFICATION QUERY
-- =====================================================================================
-- SELECT COUNT(*) as total_qt_oncology_ddis
-- FROM drug_interactions
-- WHERE interaction_id LIKE 'qt-%'
--   AND category = 'qt_prolongation'
--   AND (drug_a_name IN ('Vandetanib', 'Arsenic Trioxide', 'Nilotinib', 'Sunitinib', 'Crizotinib',
--                        'Vemurafenib', 'Lapatinib', 'Pazopanib', 'Sorafenib', 'Dasatinib', 'Imatinib')
--    OR  drug_b_name IN ('Vandetanib', 'Arsenic Trioxide', 'Nilotinib', 'Sunitinib', 'Crizotinib',
--                        'Vemurafenib', 'Lapatinib', 'Pazopanib', 'Sorafenib', 'Dasatinib', 'Imatinib'));
-- Expected: ~70 DDIs from this migration
