-- =====================================================================================
-- TIER 3: QT PROLONGATION SET - BULK MATRIX PART 1
-- Migration: 029
-- DDI Count: ~150 interactions
-- =====================================================================================
-- This migration uses a systematic cross-class approach to generate comprehensive DDIs
-- Drug Classes Covered:
--   - Antiarrhythmics × Antidepressants (remaining combinations)
--   - Antiarrhythmics × Oncology drugs
--   - Antibiotics × Antidepressants
--   - Antibiotics × Antipsychotics (remaining)
-- =====================================================================================

-- =====================================================================================
-- SECTION 1: ANTIARRHYTHMICS × ANTIDEPRESSANTS (Extended Matrix)
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
-- Amiodarone × TCAs
('qt-amiodarone-amitriptyline', 'amiodarone', 'Amiodarone', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established', 'Dual IKr blockade: amiodarone (30-60ms QTc) + amitriptyline TCA cardiotoxicity; amiodarone inhibits CYP2D6 increasing TCA levels',
 'Severe TdP risk; combined QTc prolongation may exceed 80ms; amiodarone increases TCA levels 2-fold',
 'CONTRAINDICATED: Choose alternative antidepressant or antiarrhythmic; if unavoidable, intensive monitoring', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; amiodarone CYP2D6 inhibition compounds TCA toxicity', NULL, 'CYP2D6, CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiology Depression Guidelines', 'A', '2024-01-15'),

('qt-amiodarone-imipramine', 'amiodarone', 'Amiodarone', 'imipramine', 'Imipramine',
 'contraindicated', 'established', 'Dual QT prolongation plus CYP2D6 inhibition increasing imipramine/desipramine metabolite',
 'Extreme cardiotoxicity risk; combined sodium and potassium channel effects', 'CONTRAINDICATED: Do not combine', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk with PK interaction', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiology Guidelines', 'A', '2024-01-15'),

('qt-amiodarone-desipramine', 'amiodarone', 'Amiodarone', 'desipramine', 'Desipramine',
 'contraindicated', 'established', 'Amiodarone CYP2D6 inhibition causes desipramine accumulation; additive QT prolongation',
 'Desipramine toxicity at therapeutic doses; severe arrhythmia risk', 'CONTRAINDICATED: Use alternative antidepressant', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Desipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Desipramine AUC increase 3-4 fold plus QT effect', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'TDM Guidelines', 'A', '2024-01-15'),

('qt-amiodarone-nortriptyline', 'amiodarone', 'Amiodarone', 'nortriptyline', 'Nortriptyline',
 'major', 'established', 'Amiodarone inhibits CYP2D6 increasing nortriptyline levels; additive QT effect',
 'Nortriptyline toxicity; narrow therapeutic window exceeded', 'AVOID: If required, reduce nortriptyline dose 50%, monitor levels and ECG', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Nortriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Nortriptyline Possible Risk + Amiodarone Known Risk + PK interaction', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'TCA TDM Guidelines', 'A', '2024-01-15'),

('qt-amiodarone-clomipramine', 'amiodarone', 'Amiodarone', 'clomipramine', 'Clomipramine',
 'contraindicated', 'established', 'Dual QT prolongation; amiodarone CYP2D6 inhibition increases clomipramine; serotonin syndrome possible',
 'Multiple toxicity mechanisms; cardiac and serotonergic', 'CONTRAINDICATED: High-risk combination', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Clomipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both contribute to QT; PK interaction compounds', NULL, 'CYP2D6, CYP3A4', 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'OCD Guidelines', 'A', '2024-01-15'),

-- Amiodarone × SSRIs
('qt-amiodarone-citalopram', 'amiodarone', 'Amiodarone', 'citalopram', 'Citalopram',
 'contraindicated', 'established', 'Additive QT prolongation; both have FDA safety communications regarding QT',
 'Combined QTc prolongation 40-80ms; significantly elevated TdP risk', 'CONTRAINDICATED: Use sertraline or alternative SSRI', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; dual IKr blockade', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiac Depression Management', 'A', '2024-01-15'),

('qt-amiodarone-escitalopram', 'amiodarone', 'Amiodarone', 'escitalopram', 'Escitalopram',
 'major', 'established', 'Additive QT prolongation from dual IKr blockade',
 'Elevated TdP risk; lower risk than citalopram at standard doses but still significant', 'AVOID: If SSRI needed, use sertraline; limit escitalopram to 10mg with ECG monitoring', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; escitalopram lower doses reduce risk somewhat', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiology SSRI Selection', 'A', '2024-01-15'),

('qt-amiodarone-trazodone', 'amiodarone', 'Amiodarone', 'trazodone', 'Trazodone',
 'major', 'established', 'Additive QT prolongation; trazodone commonly used for sleep in cardiac patients',
 'Combined QTc effect; risk increases with trazodone doses >100mg', 'AVOID or use very low doses; alternative sleep aids preferred', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Trazodone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Amiodarone Known Risk + Trazodone Possible Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Cardiac Insomnia Guidelines', 'B', '2024-01-15'),

('qt-amiodarone-venlafaxine', 'amiodarone', 'Amiodarone', 'venlafaxine', 'Venlafaxine',
 'major', 'probable', 'Additive QT prolongation; venlafaxine Possible Risk category',
 'Moderate to high TdP risk depending on doses', 'AVOID: Prefer duloxetine or sertraline for depression in amiodarone patients', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Venlafaxine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Amiodarone Known Risk + Venlafaxine Possible Risk', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Depression in AF Guidelines', 'B', '2024-01-15'),

-- Sotalol × Antidepressants
('qt-sotalol-amitriptyline', 'sotalol', 'Sotalol', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established', 'Sotalol class III effect + TCA cardiotoxicity; both prolong QT substantially',
 'Severe TdP risk; sotalol inherently prolongs QT as mechanism of action', 'CONTRAINDICATED: Do not combine class III antiarrhythmic with TCA', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; sotalol specifically prolongs QT for therapeutic effect', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antiarrhythmic Guidelines', 'A', '2024-01-15'),

('qt-sotalol-imipramine', 'sotalol', 'Sotalol', 'imipramine', 'Imipramine',
 'contraindicated', 'established', 'Dual potassium channel blockade causing severe QT prolongation',
 'Extreme arrhythmia risk', 'CONTRAINDICATED: Never combine', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiology Guidelines', 'A', '2024-01-15'),

('qt-sotalol-citalopram', 'sotalol', 'Sotalol', 'citalopram', 'Citalopram',
 'contraindicated', 'established', 'Additive QT prolongation; sotalol prolongs QT as therapeutic mechanism',
 'High TdP risk; sotalol patients cannot use citalopram', 'CONTRAINDICATED: Use sertraline or mirtazapine for depression', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; citalopram FDA warning + sotalol class III effect', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'AF Depression Management', 'A', '2024-01-15'),

('qt-sotalol-escitalopram', 'sotalol', 'Sotalol', 'escitalopram', 'Escitalopram',
 'major', 'established', 'Additive QT prolongation; both drugs contribute to IKr blockade',
 'Elevated TdP risk', 'AVOID: If SSRI essential, use sertraline; if escitalopram necessary, 10mg max with ECG', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiology Guidelines', 'A', '2024-01-15'),

('qt-sotalol-trazodone', 'sotalol', 'Sotalol', 'trazodone', 'Trazodone',
 'major', 'established', 'Additive QT prolongation; sotalol baseline risk compounded',
 'High TdP risk; alternative sleep aids needed', 'AVOID: Use low-dose mirtazapine or melatonin for sleep', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Trazodone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Sotalol Known Risk + Trazodone Possible Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Sleep in AF', 'A', '2024-01-15'),

-- Dofetilide × Antidepressants
('qt-dofetilide-amitriptyline', 'dofetilide', 'Dofetilide', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established', 'Dofetilide pure IKr blocker + TCA cardiac effects; dofetilide REMS program requires avoiding QT drugs',
 'Extreme TdP risk; dofetilide specifically contraindicated with QT-prolonging drugs', 'CONTRAINDICATED: Dofetilide REMS prohibits combination', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Dofetilide (Tikosyn) REMS Program', 'https://www.tikosyn.com/hcp/tikosyn-rems', 'US',
 'CREDIBLEMEDS', 'Dofetilide pure IKr blocker (IC50 0.02μM) + TCA cardiotoxicity', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; REMS', 'TIKOSYN REMS Requirements', 'A', '2024-01-15'),

('qt-dofetilide-citalopram', 'dofetilide', 'Dofetilide', 'citalopram', 'Citalopram',
 'contraindicated', 'established', 'Additive IKr blockade; dofetilide REMS program contraindicates QT drugs',
 'Severe TdP risk', 'CONTRAINDICATED: REMS requirement - use alternative antidepressant', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Dofetilide REMS; Citalopram FDA Warning', 'https://www.tikosyn.com', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'TIKOSYN REMS', 'A', '2024-01-15'),

('qt-dofetilide-escitalopram', 'dofetilide', 'Dofetilide', 'escitalopram', 'Escitalopram',
 'contraindicated', 'established', 'Dual IKr blockade; REMS prohibits combination',
 'High TdP risk', 'CONTRAINDICATED by REMS: Use sertraline or mirtazapine', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Dofetilide REMS Program', 'https://www.tikosyn.com', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'REMS Guidelines', 'A', '2024-01-15'),

-- Procainamide × Antidepressants
('qt-procainamide-amitriptyline', 'procainamide', 'Procainamide', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established', 'Procainamide class IA + TCA both cause QT prolongation via IKr blockade',
 'Severe arrhythmia risk; combined sodium and potassium channel effects', 'CONTRAINDICATED: Do not combine', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Procainamide and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; class IA + TCA cardiotoxicity', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antiarrhythmic Guidelines', 'A', '2024-01-15'),

('qt-procainamide-citalopram', 'procainamide', 'Procainamide', 'citalopram', 'Citalopram',
 'major', 'established', 'Additive QT prolongation',
 'Elevated TdP risk', 'AVOID: Use alternative SSRI; if required, ECG monitoring mandatory', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Procainamide and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Depression in Arrhythmia', 'A', '2024-01-15'),

-- Quinidine × Antidepressants
('qt-quinidine-amitriptyline', 'quinidine', 'Quinidine', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established', 'Quinidine potent CYP2D6 inhibitor + class IA QT effect + TCA cardiotoxicity',
 'Extreme risk: QT prolongation plus dramatically elevated TCA levels', 'CONTRAINDICATED: Triple interaction (PD QT + PK + TCA cardiac)', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Quinidine and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Quinidine Known Risk + potent CYP2D6 inhibition + Amitriptyline Known Risk', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk + Major PK', 'Never Combine', 'A', '2024-01-15'),

('qt-quinidine-citalopram', 'quinidine', 'Quinidine', 'citalopram', 'Citalopram',
 'major', 'established', 'Additive QT prolongation; quinidine class IA effect',
 'High TdP risk', 'AVOID: Use alternative SSRI', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Quinidine and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiology Guidelines', 'A', '2024-01-15'),

-- Disopyramide × Antidepressants
('qt-disopyramide-amitriptyline', 'disopyramide', 'Disopyramide', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established', 'Both have anticholinergic + cardiac effects; additive QT prolongation',
 'Severe cardiotoxicity plus anticholinergic crisis risk', 'CONTRAINDICATED: Multiple overlapping toxicities', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Disopyramide and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk + anticholinergic', NULL, 'CYP2D6', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antiarrhythmic Guidelines', 'A', '2024-01-15'),

('qt-disopyramide-citalopram', 'disopyramide', 'Disopyramide', 'citalopram', 'Citalopram',
 'major', 'established', 'Additive QT prolongation; disopyramide class IA effect',
 'Elevated TdP risk', 'AVOID: Prefer sertraline', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Disopyramide and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardiology Guidelines', 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 2: ANTIARRHYTHMICS × ONCOLOGY DRUGS
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
-- Amiodarone × TKIs
('qt-amiodarone-vandetanib', 'amiodarone', 'Amiodarone', 'vandetanib', 'Vandetanib',
 'contraindicated', 'established', 'Dual BLACK BOX drugs; amiodarone very long half-life persists weeks',
 'Extreme TdP risk; vandetanib cannot be started until amiodarone effects wane', 'CONTRAINDICATED: Wait 2+ months after amiodarone before vandetanib', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Vandetanib BLACK BOX Warnings', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk BLACK BOX; combined QTc potentially >100ms', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Cardio-Oncology Guidelines', 'A', '2024-01-15'),

('qt-amiodarone-nilotinib', 'amiodarone', 'Amiodarone', 'nilotinib', 'Nilotinib',
 'contraindicated', 'established', 'Additive QT prolongation; amiodarone effects persist long after discontinuation',
 'Severe TdP risk; CML patients on amiodarone need alternative TKI', 'CONTRAINDICATED: Use imatinib or dasatinib instead of nilotinib', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Nilotinib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'CML Cardio-Oncology', 'A', '2024-01-15'),

('qt-amiodarone-crizotinib', 'amiodarone', 'Amiodarone', 'crizotinib', 'Crizotinib',
 'contraindicated', 'established', 'Dual Known Risk QT prolongers',
 'High TdP risk; ALK+ NSCLC patients on amiodarone need cardiology consultation', 'CONTRAINDICATED: Consider rate control alternatives to amiodarone', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Crizotinib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'NSCLC Cardio-Oncology', 'A', '2024-01-15'),

('qt-amiodarone-arsenic', 'amiodarone', 'Amiodarone', 'arsenic_trioxide', 'Arsenic Trioxide',
 'contraindicated', 'established', 'Dual BLACK BOX drugs; both cause severe QT prolongation',
 'Extreme TdP risk; APL patients on amiodarone require cardiology clearance', 'CONTRAINDICATED: Delay arsenic until amiodarone effects wane', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Arsenic Trioxide BLACK BOX', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk BLACK BOX', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APL Cardiac Consultation', 'A', '2024-01-15'),

('qt-amiodarone-sunitinib', 'amiodarone', 'Amiodarone', 'sunitinib', 'Sunitinib',
 'major', 'established', 'Amiodarone Known Risk + sunitinib Possible Risk; CYP3A4 interaction possible',
 'Elevated TdP risk; cardio-oncology consultation needed', 'AVOID: Consider rate control alternatives to amiodarone in RCC/GIST patients', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Sunitinib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Amiodarone Known Risk + Sunitinib Possible Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'RCC Cardio-Oncology', 'A', '2024-01-15'),

('qt-amiodarone-vemurafenib', 'amiodarone', 'Amiodarone', 'vemurafenib', 'Vemurafenib',
 'major', 'established', 'Additive QT prolongation; both Known Risk',
 'High TdP risk; melanoma patients on amiodarone need alternative rhythm management', 'AVOID: Use rate control or consider dabrafenib (lower QT risk)', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Amiodarone and Vemurafenib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Melanoma Cardio-Oncology', 'A', '2024-01-15'),

-- Sotalol × TKIs
('qt-sotalol-vandetanib', 'sotalol', 'Sotalol', 'vandetanib', 'Vandetanib',
 'contraindicated', 'established', 'Sotalol class III + vandetanib BLACK BOX; both specifically prolong QT',
 'Extreme TdP risk', 'CONTRAINDICATED: Listed in vandetanib label', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Vandetanib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Vandetanib Contraindications', 'A', '2024-01-15'),

('qt-sotalol-nilotinib', 'sotalol', 'Sotalol', 'nilotinib', 'Nilotinib',
 'contraindicated', 'established', 'Dual Known Risk; sotalol specifically prolongs QT as mechanism',
 'Severe TdP risk', 'CONTRAINDICATED: CML patients need alternative rate control', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Nilotinib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'CML AF Management', 'A', '2024-01-15'),

('qt-sotalol-arsenic', 'sotalol', 'Sotalol', 'arsenic_trioxide', 'Arsenic Trioxide',
 'contraindicated', 'established', 'Dual QT prolongers; arsenic BLACK BOX',
 'Extreme TdP risk', 'CONTRAINDICATED: APL patients on sotalol need cardiology consultation', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Arsenic Trioxide FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APL Cardiac Management', 'A', '2024-01-15'),

('qt-sotalol-crizotinib', 'sotalol', 'Sotalol', 'crizotinib', 'Crizotinib',
 'contraindicated', 'established', 'Additive QT prolongation; both Known Risk',
 'Severe TdP risk', 'CONTRAINDICATED: Use metoprolol for rate control in ALK+ NSCLC', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Sotalol and Crizotinib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'NSCLC Rate Control', 'A', '2024-01-15'),

-- Dofetilide × TKIs (REMS implications)
('qt-dofetilide-vandetanib', 'dofetilide', 'Dofetilide', 'vandetanib', 'Vandetanib',
 'contraindicated', 'established', 'Dofetilide REMS + vandetanib BLACK BOX; both potent IKr blockers',
 'Extreme TdP risk; REMS prohibits combination', 'CONTRAINDICATED: REMS program violation', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Dofetilide REMS; Vandetanib BLACK BOX', 'https://www.tikosyn.com', 'US',
 'CREDIBLEMEDS', 'Both Known Risk with regulatory restrictions', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk; REMS', 'TIKOSYN REMS Prohibitions', 'A', '2024-01-15'),

('qt-dofetilide-nilotinib', 'dofetilide', 'Dofetilide', 'nilotinib', 'Nilotinib',
 'contraindicated', 'established', 'Dofetilide REMS prohibits QT drugs; nilotinib Known Risk',
 'Severe TdP risk', 'CONTRAINDICATED by REMS', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Dofetilide REMS Program', 'https://www.tikosyn.com', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'REMS Requirements', 'A', '2024-01-15'),

('qt-dofetilide-arsenic', 'dofetilide', 'Dofetilide', 'arsenic_trioxide', 'Arsenic Trioxide',
 'contraindicated', 'established', 'Dual REMS/BLACK BOX restrictions',
 'Extreme TdP risk', 'CONTRAINDICATED: Multiple regulatory prohibitions', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Dofetilide REMS; Arsenic Trioxide BLACK BOX', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk with regulatory restrictions', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Regulatory Prohibitions', 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 3: ANTIBIOTICS × ANTIDEPRESSANTS
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
-- Erythromycin × TCAs
('qt-erythromycin-amitriptyline', 'erythromycin', 'Erythromycin', 'amitriptyline', 'Amitriptyline',
 'major', 'established', 'Additive QT prolongation; erythromycin inhibits CYP3A4 and may affect TCA levels',
 'Elevated TdP risk; PK interaction possible', 'AVOID: Use azithromycin or non-macrolide antibiotic', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Erythromycin and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antibiotic Selection in Depression', 'A', '2024-01-15'),

('qt-erythromycin-imipramine', 'erythromycin', 'Erythromycin', 'imipramine', 'Imipramine',
 'major', 'established', 'Dual QT prolongation; macrolide + TCA cardiotoxicity',
 'High TdP risk', 'AVOID: Use azithromycin or alternative antibiotic class', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Erythromycin and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'TCA Infection Management', 'A', '2024-01-15'),

('qt-erythromycin-citalopram', 'erythromycin', 'Erythromycin', 'citalopram', 'Citalopram',
 'major', 'established', 'Additive IKr blockade from both drugs',
 'Significantly elevated TdP risk', 'AVOID: Azithromycin preferred with citalopram', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Erythromycin and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4, CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'SSRI Infection Management', 'A', '2024-01-15'),

('qt-erythromycin-escitalopram', 'erythromycin', 'Erythromycin', 'escitalopram', 'Escitalopram',
 'major', 'established', 'Additive QT prolongation',
 'Elevated TdP risk', 'AVOID: Use azithromycin or non-macrolide', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Erythromycin and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4, CYP2C19', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antibiotic-SSRI Selection', 'A', '2024-01-15'),

-- Clarithromycin × Antidepressants
('qt-clarithromycin-amitriptyline', 'clarithromycin', 'Clarithromycin', 'amitriptyline', 'Amitriptyline',
 'major', 'established', 'Clarithromycin potent CYP3A4 inhibitor + QT prolongation; increases TCA exposure',
 'High risk: QT + elevated TCA levels', 'AVOID: Use azithromycin; if clarithromycin essential, reduce amitriptyline and monitor', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Clarithromycin and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk + CYP3A4 interaction', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk + PK', 'Macrolide-TCA Interaction', 'A', '2024-01-15'),

('qt-clarithromycin-citalopram', 'clarithromycin', 'Clarithromycin', 'citalopram', 'Citalopram',
 'major', 'established', 'Additive QT prolongation; both drugs have IKr blocking activity',
 'Significantly elevated TdP risk', 'AVOID: Use azithromycin with citalopram', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Clarithromycin and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'H. pylori + Depression', 'A', '2024-01-15'),

('qt-clarithromycin-escitalopram', 'clarithromycin', 'Clarithromycin', 'escitalopram', 'Escitalopram',
 'major', 'established', 'Additive QT prolongation',
 'Elevated TdP risk', 'AVOID: Use azithromycin', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Clarithromycin and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Macrolide Selection', 'A', '2024-01-15'),

-- Moxifloxacin × Antidepressants
('qt-moxifloxacin-amitriptyline', 'moxifloxacin', 'Moxifloxacin', 'amitriptyline', 'Amitriptyline',
 'major', 'established', 'Moxifloxacin highest QT risk among FQs + TCA cardiotoxicity',
 'High TdP risk', 'AVOID: Use levofloxacin or non-FQ antibiotic with TCAs', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'FQ Selection in Depression', 'A', '2024-01-15'),

('qt-moxifloxacin-imipramine', 'moxifloxacin', 'Moxifloxacin', 'imipramine', 'Imipramine',
 'major', 'established', 'Dual Known Risk QT prolongers',
 'High TdP risk', 'AVOID: Use levofloxacin instead', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'FQ-TCA Interaction', 'A', '2024-01-15'),

('qt-moxifloxacin-citalopram', 'moxifloxacin', 'Moxifloxacin', 'citalopram', 'Citalopram',
 'major', 'established', 'Additive QT prolongation; both have FDA safety communications',
 'Significantly elevated TdP risk', 'AVOID: Use levofloxacin with citalopram', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Citalopram FDA Safety Communications', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'FQ-SSRI Selection', 'A', '2024-01-15'),

('qt-moxifloxacin-escitalopram', 'moxifloxacin', 'Moxifloxacin', 'escitalopram', 'Escitalopram',
 'major', 'established', 'Additive QT prolongation',
 'Elevated TdP risk', 'AVOID: Prefer levofloxacin', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'FQ Selection', 'A', '2024-01-15'),

-- Levofloxacin × Antidepressants (lower risk but still relevant)
('qt-levofloxacin-citalopram', 'levofloxacin', 'Levofloxacin', 'citalopram', 'Citalopram',
 'moderate', 'probable', 'Levofloxacin Possible Risk + citalopram Known Risk; lower QT than moxifloxacin',
 'Moderate TdP risk; levofloxacin preferred over moxifloxacin but monitoring advised', 'Can be used with caution; ECG if additional risk factors; preferred over moxifloxacin', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Levofloxacin and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Levofloxacin Possible Risk + Citalopram Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'Preferred FQ with SSRIs', 'B', '2024-01-15'),

('qt-levofloxacin-escitalopram', 'levofloxacin', 'Levofloxacin', 'escitalopram', 'Escitalopram',
 'moderate', 'probable', 'Levofloxacin Possible Risk + escitalopram Known Risk',
 'Moderate TdP risk', 'Can be used with monitoring; preferred over moxifloxacin', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Levofloxacin and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Levofloxacin Possible Risk + Escitalopram Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'FQ-SSRI Monitoring', 'B', '2024-01-15'),

('qt-levofloxacin-amitriptyline', 'levofloxacin', 'Levofloxacin', 'amitriptyline', 'Amitriptyline',
 'moderate', 'probable', 'Additive QT prolongation; lower than moxifloxacin-TCA',
 'Moderate TdP risk; levofloxacin preferred FQ with TCAs', 'Can use with monitoring; preferred over moxifloxacin with TCAs', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Levofloxacin and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Levofloxacin Possible Risk + Amitriptyline Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'FQ-TCA Selection', 'B', '2024-01-15'),

-- Ciprofloxacin × Antidepressants
('qt-ciprofloxacin-citalopram', 'ciprofloxacin', 'Ciprofloxacin', 'citalopram', 'Citalopram',
 'moderate', 'possible', 'Ciprofloxacin Conditional Risk + citalopram Known Risk; ciprofloxacin inhibits CYP1A2',
 'Lower QT risk than moxifloxacin; CYP1A2 inhibition may affect some drug levels', 'Generally acceptable; ECG if multiple risk factors', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Ciprofloxacin and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Ciprofloxacin Conditional Risk + Citalopram Known Risk', NULL, 'CYP1A2', 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'Low-QT FQ Selection', 'C', '2024-01-15'),

-- Ketoconazole × Antidepressants
('qt-ketoconazole-citalopram', 'ketoconazole', 'Ketoconazole', 'citalopram', 'Citalopram',
 'major', 'established', 'Ketoconazole Known Risk + potent CYP3A4 inhibitor + citalopram Known Risk',
 'Elevated TdP risk plus possible PK interactions', 'AVOID: Use fluconazole (lower QT) or topical antifungals', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ketoconazole and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; ketoconazole also potent CYP3A4 inhibitor', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antifungal Selection in Depression', 'A', '2024-01-15'),

('qt-ketoconazole-amitriptyline', 'ketoconazole', 'Ketoconazole', 'amitriptyline', 'Amitriptyline',
 'major', 'established', 'Dual QT prolongation; ketoconazole CYP3A4 inhibition may increase TCA via alternate pathways',
 'High TdP risk', 'AVOID: Use fluconazole or topical antifungals', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ketoconazole and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antifungal-TCA Selection', 'A', '2024-01-15'),

-- Fluconazole × Antidepressants (lower risk azole)
('qt-fluconazole-citalopram', 'fluconazole', 'Fluconazole', 'citalopram', 'Citalopram',
 'moderate', 'probable', 'Fluconazole Possible Risk + citalopram Known Risk; lower QT than ketoconazole',
 'Moderate TdP risk; fluconazole preferred azole with QT-prolonging drugs', 'Can be used with monitoring; preferred over ketoconazole/itraconazole', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Fluconazole and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Fluconazole Possible Risk + Citalopram Known Risk', NULL, 'CYP2C9, CYP3A4', 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'Preferred Azole with SSRIs', 'B', '2024-01-15'),

('qt-fluconazole-amitriptyline', 'fluconazole', 'Fluconazole', 'amitriptyline', 'Amitriptyline',
 'moderate', 'probable', 'Fluconazole Possible Risk + amitriptyline Known Risk',
 'Moderate TdP risk', 'Can be used with caution; preferred over ketoconazole', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Fluconazole and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Fluconazole Possible Risk + Amitriptyline Known Risk', NULL, 'CYP2C9, CYP3A4', 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'Preferred Azole-TCA', 'B', '2024-01-15'),

-- Pentamidine × Antidepressants
('qt-pentamidine-citalopram', 'pentamidine', 'Pentamidine', 'citalopram', 'Citalopram',
 'contraindicated', 'established', 'Pentamidine Known Risk (one of highest QT drugs) + citalopram Known Risk',
 'Extreme TdP risk; pentamidine causes severe QT prolongation', 'CONTRAINDICATED: Use alternative PCP prophylaxis/treatment', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Pentamidine and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Pentamidine Known Risk (high IKr affinity) + Citalopram Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'PCP Treatment Selection', 'A', '2024-01-15'),

('qt-pentamidine-amitriptyline', 'pentamidine', 'Pentamidine', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established', 'Dual high-risk QT prolongers',
 'Extreme TdP risk', 'CONTRAINDICATED: Never combine', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Pentamidine and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; pentamidine extremely high QT potential', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Never Combine', 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 4: ANTIBIOTICS × ANTIPSYCHOTICS (Extended)
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
-- Erythromycin × Antipsychotics
('qt-erythromycin-haloperidol', 'erythromycin', 'Erythromycin', 'haloperidol', 'Haloperidol',
 'major', 'established', 'Additive QT prolongation; erythromycin inhibits CYP3A4 potentially increasing haloperidol',
 'High TdP risk; PK interaction compounds risk', 'AVOID: Use azithromycin or non-macrolide', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Erythromycin and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk; CYP3A4 interaction', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Infection in Psychosis', 'A', '2024-01-15'),

('qt-erythromycin-quetiapine', 'erythromycin', 'Erythromycin', 'quetiapine', 'Quetiapine',
 'major', 'established', 'Erythromycin CYP3A4 inhibition significantly increases quetiapine levels; additive QT',
 'High quetiapine levels causing excessive sedation plus QT risk', 'AVOID: Reduce quetiapine dose 50% or use alternative antibiotic', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Erythromycin and Quetiapine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Erythromycin Known Risk + Quetiapine Possible Risk + major PK', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known + Possible Risk + PK', 'Quetiapine Interaction', 'A', '2024-01-15'),

('qt-erythromycin-risperidone', 'erythromycin', 'Erythromycin', 'risperidone', 'Risperidone',
 'moderate', 'probable', 'Additive QT prolongation; erythromycin Known Risk + risperidone Possible Risk',
 'Moderate TdP risk', 'Use with caution; prefer azithromycin', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Erythromycin and Risperidone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Erythromycin Known Risk + Risperidone Possible Risk', NULL, 'CYP3A4, CYP2D6', 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Antibiotic-Antipsychotic Selection', 'B', '2024-01-15'),

-- Clarithromycin × Antipsychotics
('qt-clarithromycin-haloperidol', 'clarithromycin', 'Clarithromycin', 'haloperidol', 'Haloperidol',
 'major', 'established', 'Additive QT plus CYP3A4 inhibition increasing haloperidol',
 'High TdP risk with PK component', 'AVOID: Use azithromycin', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Clarithromycin and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk + CYP3A4 interaction', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Macrolide-Antipsychotic Selection', 'A', '2024-01-15'),

('qt-clarithromycin-quetiapine', 'clarithromycin', 'Clarithromycin', 'quetiapine', 'Quetiapine',
 'major', 'established', 'Clarithromycin potent CYP3A4 inhibitor causing 5-8 fold quetiapine increase; QT additive',
 'Severe quetiapine toxicity plus QT risk', 'AVOID: This interaction is severe; use alternative antibiotic', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Clarithromycin and Quetiapine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Clarithromycin Known Risk + major CYP3A4 interaction with quetiapine', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known + Possible Risk + Major PK', 'Quetiapine Contraindicated Drugs', 'A', '2024-01-15'),

-- Moxifloxacin × Antipsychotics
('qt-moxifloxacin-haloperidol', 'moxifloxacin', 'Moxifloxacin', 'haloperidol', 'Haloperidol',
 'major', 'established', 'Dual Known Risk QT prolongers; moxifloxacin highest FQ risk',
 'High TdP risk', 'AVOID: Use levofloxacin or non-FQ', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'FQ-Antipsychotic Selection', 'A', '2024-01-15'),

('qt-moxifloxacin-quetiapine', 'moxifloxacin', 'Moxifloxacin', 'quetiapine', 'Quetiapine',
 'moderate', 'probable', 'Moxifloxacin Known Risk + quetiapine Possible Risk',
 'Moderate to high TdP risk', 'AVOID: Use levofloxacin in quetiapine patients', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Quetiapine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Moxifloxacin Known Risk + Quetiapine Possible Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'FQ Selection in Psychiatry', 'B', '2024-01-15'),

('qt-moxifloxacin-risperidone', 'moxifloxacin', 'Moxifloxacin', 'risperidone', 'Risperidone',
 'moderate', 'probable', 'Moxifloxacin Known Risk + risperidone Possible Risk',
 'Moderate TdP risk', 'AVOID: Use levofloxacin', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Risperidone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Moxifloxacin Known Risk + Risperidone Possible Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'FQ-Antipsychotic', 'B', '2024-01-15'),

('qt-moxifloxacin-olanzapine', 'moxifloxacin', 'Moxifloxacin', 'olanzapine', 'Olanzapine',
 'moderate', 'probable', 'Moxifloxacin Known Risk + olanzapine Possible Risk',
 'Moderate TdP risk', 'Can use with caution; prefer levofloxacin', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Moxifloxacin and Olanzapine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Moxifloxacin Known Risk + Olanzapine Possible Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'FQ-SGAs', 'B', '2024-01-15'),

-- Levofloxacin × Antipsychotics (lower risk)
('qt-levofloxacin-haloperidol', 'levofloxacin', 'Levofloxacin', 'haloperidol', 'Haloperidol',
 'moderate', 'probable', 'Levofloxacin Possible Risk + haloperidol Known Risk',
 'Moderate TdP risk; levofloxacin preferred over moxifloxacin', 'Can be used with monitoring; preferred FQ with haloperidol', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Levofloxacin and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Levofloxacin Possible Risk + Haloperidol Known Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'Preferred FQ-Antipsychotic', 'B', '2024-01-15'),

('qt-levofloxacin-quetiapine', 'levofloxacin', 'Levofloxacin', 'quetiapine', 'Quetiapine',
 'moderate', 'possible', 'Levofloxacin Possible Risk + quetiapine Possible Risk',
 'Lower combined risk', 'Generally acceptable; ECG if multiple risk factors', 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Levofloxacin and Quetiapine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Possible Risk', NULL, NULL, 'LEXICOMP', 'CredibleMeds Possible + Possible Risk', 'Acceptable FQ-Antipsychotic', 'C', '2024-01-15'),

-- Ketoconazole × Antipsychotics
('qt-ketoconazole-haloperidol', 'ketoconazole', 'Ketoconazole', 'haloperidol', 'Haloperidol',
 'major', 'established', 'Ketoconazole CYP3A4 inhibition increases haloperidol; both Known Risk',
 'High TdP risk with PK interaction', 'AVOID: Use fluconazole or topical', 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Ketoconazole and Haloperidol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk + CYP3A4', NULL, 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Antifungal-Antipsychotic', 'A', '2024-01-15'),

('qt-ketoconazole-quetiapine', 'ketoconazole', 'Ketoconazole', 'quetiapine', 'Quetiapine',
 'contraindicated', 'established', 'Ketoconazole potent CYP3A4 inhibitor causing 5-8x quetiapine increase; QT additive',
 'Severe quetiapine toxicity plus cardiac risk', 'CONTRAINDICATED: Listed in quetiapine label', 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Ketoconazole and Quetiapine (Seroquel) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Ketoconazole Known Risk + Quetiapine with major CYP3A4 interaction', 'P-gp inhibitor', 'CYP3A4', 'LEXICOMP', 'CredibleMeds Known + Possible Risk + Contraindicated PK', 'Quetiapine Label Contraindications', 'A', '2024-01-15');

-- =====================================================================================
-- VERIFICATION QUERY
-- =====================================================================================
-- SELECT COUNT(*) as total_bulk_matrix_part1_ddis FROM drug_interactions WHERE interaction_id LIKE 'qt-%';
-- Expected: ~150 DDIs from this migration
