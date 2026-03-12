-- =====================================================================================
-- TIER 3: QT PROLONGATION SET - ANTIDEPRESSANT × ANTIDEPRESSANT MATRIX
-- Migration: 026
-- DDI Count: ~45 interactions
-- =====================================================================================
-- Clinical Context:
-- Antidepressant combinations are common in treatment-resistant depression (TRD),
-- anxiety disorders with comorbid depression, and augmentation strategies.
-- QT prolongation risk varies significantly:
--   - TCAs: Highest risk (sodium/potassium channel blockade)
--   - Citalopram/Escitalopram: FDA black box warning for dose-dependent QT prolongation
--   - Other SSRIs: Lower/minimal risk
--   - Trazodone: Moderate risk, common as sleep adjunct
--   - SNRIs: Generally lower risk (venlafaxine has some reports)
--
-- CredibleMeds Classification:
--   KNOWN_RISK: Citalopram, Escitalopram (>40mg/20mg), amitriptyline, imipramine, desipramine
--   POSSIBLE_RISK: Trazodone, venlafaxine, clomipramine, nortriptyline
--   CONDITIONAL_RISK: Most other antidepressants (risk with overdose or drug interactions)
-- =====================================================================================

-- =====================================================================================
-- SECTION 1: TCA × TCA COMBINATIONS (CONTRAINDICATED - Additive Cardiotoxicity)
-- =====================================================================================
-- TCAs are rarely combined but important to document due to high risk
-- Mechanism: Dual sodium/potassium channel blockade, additive cardiac conduction delay

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Amitriptyline × Imipramine (Both Known Risk)
('qt-amitriptyline-imipramine', 'amitriptyline', 'Amitriptyline', 'imipramine', 'Imipramine',
 'contraindicated', 'established',
 'Additive QT prolongation via dual IKr blockade and sodium channel inhibition; both TCAs cause dose-dependent cardiac conduction delays',
 'Severely increased risk of Torsades de Pointes, ventricular arrhythmias, and sudden cardiac death; combined QTc prolongation may exceed 100ms',
 'CONTRAINDICATED: Never combine two TCAs; if switching, allow complete washout (5 half-lives); ECG monitoring mandatory during any transition',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amitriptyline and Imipramine FDA Labels - Cardiovascular Warnings', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both TCAs cause IKr blockade (IC50: 1-5μM) plus sodium channel block causing QRS widening; combined cardiotoxicity risk', 'P-gp substrates', 'CYP2D6, CYP3A4, CYP1A2',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APA Practice Guidelines - TCA Safety',
 'A', '2024-01-15'),

-- Amitriptyline × Desipramine
('qt-amitriptyline-desipramine', 'amitriptyline', 'Amitriptyline', 'desipramine', 'Desipramine',
 'contraindicated', 'established',
 'Additive QT prolongation; desipramine is active metabolite of imipramine with similar cardiotoxicity profile',
 'Markedly increased risk of cardiac arrhythmias; combined sodium and potassium channel blockade',
 'CONTRAINDICATED: Do not combine TCAs; use single agent at lowest effective dose with ECG monitoring',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'TCA Class Labeling - Cardiac Warnings', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Desipramine QTc effect 15-30ms; combined with amitriptyline exceeds safety threshold', 'P-gp substrates', 'CYP2D6 (primary)',
 'LEXICOMP', 'CredibleMeds Known Risk Classification', 'Tricyclic Antidepressant Prescribing Guide',
 'A', '2024-01-15'),

-- Imipramine × Desipramine (parent-metabolite combination)
('qt-imipramine-desipramine', 'imipramine', 'Imipramine', 'desipramine', 'Desipramine',
 'contraindicated', 'established',
 'Desipramine is active metabolite of imipramine; exogenous desipramine adds to endogenous metabolite levels',
 'Excessive TCA exposure with severe cardiotoxicity risk; effectively supratherapeutic combined levels',
 'CONTRAINDICATED: Pharmacologically redundant and dangerous; desipramine already present as imipramine metabolite',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Imipramine FDA Label - Metabolism Section', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Parent-metabolite overlap creates supratherapeutic exposure; combined IKr blockade', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk', 'Clinical Pharmacology Reference',
 'A', '2024-01-15'),

-- Amitriptyline × Nortriptyline (parent-metabolite)
('qt-amitriptyline-nortriptyline', 'amitriptyline', 'Amitriptyline', 'nortriptyline', 'Nortriptyline',
 'contraindicated', 'established',
 'Nortriptyline is active metabolite of amitriptyline; exogenous addition creates supratherapeutic levels',
 'Excessive combined exposure; severe cardiac conduction abnormalities including heart block',
 'CONTRAINDICATED: Nortriptyline already present as amitriptyline metabolite; redundant and dangerous',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Amitriptyline FDA Label - Metabolism Section', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Parent drug plus exogenous metabolite exceeds therapeutic window; additive IKr/INa blockade', NULL, 'CYP2D6, CYP3A4',
 'LEXICOMP', 'CredibleMeds Known Risk', 'TCA Pharmacology Reference',
 'A', '2024-01-15'),

-- Clomipramine × Amitriptyline
('qt-clomipramine-amitriptyline', 'clomipramine', 'Clomipramine', 'amitriptyline', 'Amitriptyline',
 'contraindicated', 'established',
 'Additive QT prolongation; clomipramine has significant serotonergic activity adding serotonin syndrome risk',
 'Dual cardiotoxicity plus serotonin syndrome risk; life-threatening combination',
 'CONTRAINDICATED: Multiple overlapping toxicities; choose single agent for OCD (clomipramine) or depression (alternatives preferred)',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Clomipramine (Anafranil) FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Clomipramine IKr IC50 2.1μM plus amitriptyline cardiotoxicity; additive risks', 'P-gp substrates', 'CYP2D6, CYP3A4, CYP1A2',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'WFSBP OCD Treatment Guidelines',
 'A', '2024-01-15'),

-- Doxepin × Amitriptyline
('qt-doxepin-amitriptyline', 'doxepin', 'Doxepin', 'amitriptyline', 'Amitriptyline',
 'major', 'established',
 'Additive TCA cardiotoxicity; both cause QT prolongation and cardiac conduction delays',
 'Increased risk of arrhythmias, heart block, and TdP; excessive anticholinergic burden',
 'AVOID: No clinical rationale for combining TCAs; if using doxepin for sleep, ensure no concurrent TCA',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Doxepin FDA Label - Cardiac Warnings', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Doxepin lower QT risk than other TCAs but still contributes to additive effect', NULL, 'CYP2D6, CYP2C19',
 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'APA Depression Guidelines',
 'B', '2024-01-15');

-- =====================================================================================
-- SECTION 2: CITALOPRAM/ESCITALOPRAM × TCA COMBINATIONS (MAJOR - FDA BLACK BOX + TCA)
-- =====================================================================================
-- Most concerning SSRI combinations due to citalopram's dose-dependent QT prolongation
-- FDA issued safety communication in 2011 limiting citalopram to 40mg max (20mg in elderly)

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Citalopram × Amitriptyline
('qt-citalopram-amitriptyline', 'citalopram', 'Citalopram', 'amitriptyline', 'Amitriptyline',
 'major', 'established',
 'Dual QT prolongation: citalopram causes dose-dependent IKr blockade (FDA black box); amitriptyline adds TCA cardiotoxicity; citalopram inhibits CYP2D6 increasing amitriptyline levels',
 'Significantly increased TdP risk; QTc prolongation 30-60ms combined; also serotonin syndrome risk',
 'AVOID if possible; if necessary: citalopram ≤20mg, lowest amitriptyline dose; baseline and serial ECGs; discontinue if QTc >500ms',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Citalopram FDA Safety Communication 2011 - QT Prolongation', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-drug-safety-communication-revised-recommendations-celexa-citalopram-hydrobromide', 'US',
 'CREDIBLEMEDS', 'Citalopram IKr IC50 2.4μM causing dose-dependent QTc prolongation (8-18ms at 20-60mg); CYP2D6 inhibition raises TCA levels', NULL, 'CYP2C19 (citalopram), CYP2D6 (amitriptyline)',
 'LEXICOMP', 'FDA Black Box + CredibleMeds Known Risk', 'APA Guidelines - Antidepressant Combinations',
 'A', '2024-01-15'),

-- Citalopram × Imipramine
('qt-citalopram-imipramine', 'citalopram', 'Citalopram', 'imipramine', 'Imipramine',
 'major', 'established',
 'Additive QT prolongation via IKr blockade; citalopram inhibits CYP2D6 increasing imipramine and desipramine levels 2-3 fold',
 'High TdP risk; significant QTc prolongation; elevated TCA levels increase both efficacy and toxicity',
 'AVOID combination; if required: limit citalopram to 20mg; reduce imipramine dose by 50%; monitor ECG and TCA levels',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Citalopram FDA Label - Drug Interactions', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'CYP2D6 inhibition by citalopram increases imipramine AUC 2-3x; combined IKr blockade', NULL, 'CYP2D6, CYP2C19',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'Clinical Pharmacology Drug Interactions',
 'A', '2024-01-15'),

-- Citalopram × Desipramine
('qt-citalopram-desipramine', 'citalopram', 'Citalopram', 'desipramine', 'Desipramine',
 'major', 'established',
 'Citalopram inhibits CYP2D6 causing 2-fold increase in desipramine levels; additive QT prolongation',
 'Desipramine toxicity at therapeutic doses; combined QTc prolongation exceeds individual drug effects',
 'AVOID: If combining, reduce desipramine dose by 50%; monitor plasma levels; baseline/serial ECGs',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Desipramine and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Desipramine is CYP2D6 substrate; citalopram inhibition causes significant level increase plus additive IKr block', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'TDM Guidelines for TCAs',
 'A', '2024-01-15'),

-- Citalopram × Nortriptyline
('qt-citalopram-nortriptyline', 'citalopram', 'Citalopram', 'nortriptyline', 'Nortriptyline',
 'major', 'established',
 'CYP2D6 inhibition increases nortriptyline levels; both drugs cause QT prolongation independently',
 'Nortriptyline toxicity risk; additive cardiac effects; therapeutic window for nortriptyline is narrow (50-150 ng/mL)',
 'Use with caution; reduce nortriptyline dose; target trough 50-100 ng/mL; ECG monitoring required',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Nortriptyline FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Nortriptyline Possible Risk; CYP2D6 inhibition increases levels into toxic range', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'TDM Guidelines for Nortriptyline',
 'B', '2024-01-15'),

-- Escitalopram × Amitriptyline
('qt-escitalopram-amitriptyline', 'escitalopram', 'Escitalopram', 'amitriptyline', 'Amitriptyline',
 'major', 'established',
 'Escitalopram causes dose-dependent QT prolongation (S-enantiomer of citalopram); moderate CYP2D6 inhibition increases amitriptyline',
 'Increased TdP risk; combined QTc prolongation 20-40ms; escitalopram >20mg contraindicated',
 'Limit escitalopram to 10-20mg; use lowest effective amitriptyline dose; ECG monitoring; watch for serotonin syndrome',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Escitalopram (Lexapro) FDA Label - QT Warning', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Escitalopram IKr IC50 similar to citalopram but lower doses used clinically; still Known Risk', NULL, 'CYP2C19, CYP2D6',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APA Guidelines',
 'A', '2024-01-15'),

-- Escitalopram × Imipramine
('qt-escitalopram-imipramine', 'escitalopram', 'Escitalopram', 'imipramine', 'Imipramine',
 'major', 'established',
 'Dual QT prolongation mechanism; escitalopram inhibits CYP2D6 moderately increasing imipramine/desipramine',
 'Elevated TdP risk; imipramine toxicity possible; cardiac conduction delays',
 'AVOID if possible; if required: escitalopram ≤10mg, reduced imipramine dose; ECG and TCA level monitoring',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Escitalopram FDA Label - Drug Interactions', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Known Risk on CredibleMeds; additive IKr blockade with PK interaction', NULL, 'CYP2D6, CYP2C19',
 'LEXICOMP', 'CredibleMeds Known Risk × Known Risk', 'APA Guidelines - Drug Interactions',
 'A', '2024-01-15'),

-- Escitalopram × Clomipramine
('qt-escitalopram-clomipramine', 'escitalopram', 'Escitalopram', 'clomipramine', 'Clomipramine',
 'major', 'established',
 'Additive QT prolongation; significant serotonin syndrome risk (clomipramine is serotonergic TCA); CYP2D6 inhibition',
 'High risk of both TdP and serotonin syndrome; dual life-threatening mechanisms',
 'AVOID: High-risk combination with multiple overlapping toxicities; consider alternatives for OCD',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Clomipramine (Anafranil) and Escitalopram Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Clomipramine Possible Risk + Escitalopram Known Risk; serotonergic overlap adds toxicity', 'P-gp substrate (clomipramine)', 'CYP2D6, CYP3A4',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'WFSBP OCD Guidelines',
 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 3: CITALOPRAM/ESCITALOPRAM × OTHER ANTIDEPRESSANTS
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
-- Citalopram × Trazodone
('qt-citalopram-trazodone', 'citalopram', 'Citalopram', 'trazodone', 'Trazodone',
 'major', 'established',
 'Additive QT prolongation: citalopram (Known Risk) + trazodone (Possible Risk); both block IKr channels',
 'Increased TdP risk particularly at higher doses; trazodone commonly used for sleep with SSRIs',
 'Use lowest effective doses; citalopram ≤20mg with trazodone ≤100mg; ECG if risk factors; avoid in elderly',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Citalopram and Trazodone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Trazodone IKr IC50 ~10μM (Possible Risk); combined with citalopram increases QTc 15-25ms', NULL, 'CYP3A4 (trazodone), CYP2C19 (citalopram)',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Sleep Medicine Guidelines',
 'B', '2024-01-15'),

-- Citalopram × Venlafaxine
('qt-citalopram-venlafaxine', 'citalopram', 'Citalopram', 'venlafaxine', 'Venlafaxine',
 'major', 'probable',
 'Additive QT prolongation (both Known/Possible Risk); significant serotonin syndrome risk; CYP2D6 interaction',
 'Combined QTc prolongation; high serotonin syndrome risk with dual serotonergic agents',
 'AVOID combining; if switching, allow adequate washout; no clinical rationale for combination',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Citalopram and Venlafaxine (Effexor) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Venlafaxine Possible Risk for QT; combined serotonergic effect plus cardiac risk', NULL, 'CYP2D6 (venlafaxine), CYP2C19 (citalopram)',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'APA Depression Guidelines',
 'B', '2024-01-15'),

-- Escitalopram × Trazodone
('qt-escitalopram-trazodone', 'escitalopram', 'Escitalopram', 'trazodone', 'Trazodone',
 'major', 'established',
 'Additive QT prolongation from dual IKr blockade; common combination for depression with insomnia',
 'Increased arrhythmia risk; monitor for excessive sedation and cardiac effects',
 'Commonly used but requires monitoring; limit escitalopram to 10-20mg; trazodone ≤100mg for sleep',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Escitalopram and Trazodone FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Combined IKr blockade; lower risk than citalopram due to lower escitalopram doses', NULL, 'CYP3A4, CYP2C19',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'Insomnia Treatment Guidelines',
 'B', '2024-01-15'),

-- Escitalopram × Venlafaxine
('qt-escitalopram-venlafaxine', 'escitalopram', 'Escitalopram', 'venlafaxine', 'Venlafaxine',
 'major', 'probable',
 'Additive QT prolongation; dual serotonergic mechanism creating serotonin syndrome risk',
 'Combined cardiac and serotonergic toxicity risk; no therapeutic rationale for combination',
 'AVOID: No clinical indication for combining two first-line antidepressants; switch rather than add',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Escitalopram and Venlafaxine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both drugs on CredibleMeds lists; serotonin syndrome risk compounds cardiac risk', NULL, 'CYP2D6, CYP2C19',
 'LEXICOMP', 'CredibleMeds Known + Possible Risk', 'APA Guidelines',
 'B', '2024-01-15'),

-- Citalopram × Mirtazapine
('qt-citalopram-mirtazapine', 'citalopram', 'Citalopram', 'mirtazapine', 'Mirtazapine',
 'moderate', 'probable',
 'Citalopram causes QT prolongation; mirtazapine has lower cardiac risk but some QT reports; used in "California Rocket Fuel" combination',
 'Modest additive QT risk; combination used for treatment-resistant depression',
 'Acceptable with monitoring; limit citalopram to 20mg; ECG if cardiac risk factors present',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Citalopram and Mirtazapine (Remeron) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Mirtazapine Conditional Risk only; citalopram is primary QT concern in combination', NULL, 'CYP2D6, CYP3A4, CYP1A2 (mirtazapine)',
 'LEXICOMP', 'CredibleMeds Known + Conditional Risk', 'TRD Augmentation Strategies',
 'B', '2024-01-15'),

-- Citalopram × Bupropion
('qt-citalopram-bupropion', 'citalopram', 'Citalopram', 'bupropion', 'Bupropion',
 'moderate', 'possible',
 'Citalopram QT prolongation is primary concern; bupropion has minimal QT effect but inhibits CYP2D6',
 'Citalopram-related QT risk; bupropion CYP2D6 inhibition is minor; combination commonly used',
 'Generally acceptable; monitor for citalopram QT effects; limit citalopram dose; bupropion adds seizure risk consideration',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Citalopram and Bupropion (Wellbutrin) FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Bupropion not on QT risk lists; citalopram is sole QT concern; commonly combined', NULL, 'CYP2B6 (bupropion), CYP2C19 (citalopram)',
 'LEXICOMP', 'CredibleMeds Known Risk (citalopram only)', 'Combination Antidepressant Guidelines',
 'C', '2024-01-15');

-- =====================================================================================
-- SECTION 4: TRAZODONE × OTHER ANTIDEPRESSANTS
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
-- Trazodone × Amitriptyline
('qt-trazodone-amitriptyline', 'trazodone', 'Trazodone', 'amitriptyline', 'Amitriptyline',
 'major', 'established',
 'Additive QT prolongation: trazodone (Possible Risk) + amitriptyline (Known Risk TCA); excessive sedation',
 'Increased arrhythmia risk; profound sedation; orthostatic hypotension; anticholinergic burden',
 'AVOID if possible; if using trazodone for sleep with TCA: minimum doses, ECG monitoring, warn about sedation',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Trazodone and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Combined IKr blockade from both drugs; trazodone adds serotonergic effect to TCA', NULL, 'CYP3A4 (trazodone), CYP2D6 (amitriptyline)',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'APA Guidelines',
 'A', '2024-01-15'),

-- Trazodone × Imipramine
('qt-trazodone-imipramine', 'trazodone', 'Trazodone', 'imipramine', 'Imipramine',
 'major', 'established',
 'Additive QT prolongation from dual IKr blockade; sedation and serotonergic effects compound',
 'Elevated TdP risk; imipramine cardiotoxicity enhanced; excessive sedation',
 'AVOID: No clear rationale for combination; if transitioning, taper one before starting other',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Trazodone and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Trazodone Possible Risk + Imipramine Known Risk; combined cardiac effects', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'TCA Prescribing Guidelines',
 'A', '2024-01-15'),

-- Trazodone × Nortriptyline
('qt-trazodone-nortriptyline', 'trazodone', 'Trazodone', 'nortriptyline', 'Nortriptyline',
 'moderate', 'probable',
 'Additive QT prolongation; both are Possible Risk category; sedation and hypotension',
 'Moderate increase in arrhythmia risk; nortriptyline has narrower therapeutic window',
 'Use with caution; monitor nortriptyline levels; low doses of trazodone for sleep may be acceptable',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Trazodone and Nortriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Possible Risk; combined effect modest but requires monitoring', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Possible + Possible Risk', 'Geriatric Prescribing Guidelines',
 'B', '2024-01-15'),

-- Trazodone × Venlafaxine
('qt-trazodone-venlafaxine', 'trazodone', 'Trazodone', 'venlafaxine', 'Venlafaxine',
 'moderate', 'probable',
 'Additive QT prolongation (both Possible Risk); serotonin syndrome risk from dual serotonergic activity',
 'Moderate QT risk; serotonin syndrome concern; common combination for depression with insomnia',
 'Monitor for serotonin syndrome; ECG if risk factors; trazodone ≤100mg for sleep adjunct',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Trazodone and Venlafaxine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both Possible Risk; serotonergic interaction adds to cardiac concern', NULL, 'CYP3A4, CYP2D6',
 'LEXICOMP', 'CredibleMeds Possible + Possible Risk', 'SNRI Augmentation Guidelines',
 'B', '2024-01-15');

-- =====================================================================================
-- SECTION 5: SSRI × TCA COMBINATIONS (OTHER SSRIs)
-- =====================================================================================
-- SSRIs other than citalopram/escitalopram have lower intrinsic QT risk
-- Main concern is CYP2D6 inhibition increasing TCA levels

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    severity, evidence_level, mechanism, clinical_effect, management,
    onset, documentation_level, category, is_active,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES
-- Fluoxetine × Amitriptyline
('qt-fluoxetine-amitriptyline', 'fluoxetine', 'Fluoxetine', 'amitriptyline', 'Amitriptyline',
 'major', 'established',
 'Fluoxetine is potent CYP2D6 inhibitor increasing amitriptyline/nortriptyline levels 2-4 fold; amitriptyline causes QT prolongation',
 'TCA toxicity at therapeutic doses; QT prolongation from elevated TCA levels; serotonin syndrome risk',
 'AVOID or reduce amitriptyline dose by 50-75%; monitor TCA levels; ECG monitoring essential; watch for serotonin syndrome',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Fluoxetine (Prozac) FDA Label - CYP2D6 Inhibition', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'Fluoxetine Ki for CYP2D6 = 0.17μM (potent inhibitor); increases amitriptyline AUC 200-400%', NULL, 'CYP2D6',
 'LEXICOMP', 'Major PK interaction - CredibleMeds Known Risk (amitriptyline)', 'TDM Guidelines for TCAs with SSRIs',
 'A', '2024-01-15'),

-- Fluoxetine × Imipramine
('qt-fluoxetine-imipramine', 'fluoxetine', 'Fluoxetine', 'imipramine', 'Imipramine',
 'major', 'established',
 'Potent CYP2D6 inhibition by fluoxetine dramatically increases imipramine and desipramine levels',
 'Severe TCA toxicity risk; cardiac conduction abnormalities; QT prolongation from elevated levels',
 'AVOID: Extremely high interaction severity; if necessary reduce imipramine by 75%; mandatory TDM and ECG',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Fluoxetine and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'CYP2D6 inhibition increases imipramine + desipramine combined levels by 300-400%', NULL, 'CYP2D6',
 'LEXICOMP', 'Major PK interaction - CredibleMeds Known Risk (imipramine)', 'Clinical Pharmacokinetics of TCAs',
 'A', '2024-01-15'),

-- Fluoxetine × Desipramine
('qt-fluoxetine-desipramine', 'fluoxetine', 'Fluoxetine', 'desipramine', 'Desipramine',
 'major', 'established',
 'Fluoxetine potent CYP2D6 inhibition causes 4-5 fold increase in desipramine levels',
 'Desipramine toxicity; therapeutic doses become toxic; QT prolongation and cardiac conduction delays',
 'CONTRAINDICATED or reduce desipramine to 25% of usual dose; mandatory TDM; consider alternative SSRI',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Fluoxetine and Desipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'Desipramine AUC increases 400-500% with fluoxetine; well-documented case reports of toxicity', NULL, 'CYP2D6',
 'LEXICOMP', 'Major PK interaction - CredibleMeds Known Risk (desipramine)', 'Drug Interaction Principles',
 'A', '2024-01-15'),

-- Paroxetine × Amitriptyline
('qt-paroxetine-amitriptyline', 'paroxetine', 'Paroxetine', 'amitriptyline', 'Amitriptyline',
 'major', 'established',
 'Paroxetine is potent CYP2D6 inhibitor (Ki = 0.15μM); increases amitriptyline levels significantly',
 'TCA toxicity; QT prolongation from elevated amitriptyline; serotonin syndrome risk',
 'AVOID or substantial amitriptyline dose reduction (50-75%); TDM required; ECG monitoring',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Paroxetine (Paxil) FDA Label - CYP2D6 Inhibition', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'Paroxetine CYP2D6 Ki = 0.15μM (most potent SSRI inhibitor); increases TCA levels 200-300%', NULL, 'CYP2D6',
 'LEXICOMP', 'Major PK interaction - CredibleMeds Known Risk (amitriptyline)', 'SSRI-TCA Interaction Studies',
 'A', '2024-01-15'),

-- Paroxetine × Imipramine
('qt-paroxetine-imipramine', 'paroxetine', 'Paroxetine', 'imipramine', 'Imipramine',
 'major', 'established',
 'Potent CYP2D6 inhibition by paroxetine increases imipramine and desipramine metabolite substantially',
 'Combined parent/metabolite toxicity; cardiac conduction abnormalities; high QT prolongation risk',
 'AVOID: Extreme interaction potential; alternative antidepressants preferred; if unavoidable, reduce imipramine 75%',
 'delayed', 'excellent', 'qt_prolongation', true,
 'FDA_LABEL', 'Paroxetine and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'CYP2D6 inhibition by paroxetine among strongest of SSRIs', NULL, 'CYP2D6',
 'LEXICOMP', 'Major PK interaction - CredibleMeds Known Risk (imipramine)', 'Clinical Pharmacology',
 'A', '2024-01-15'),

-- Sertraline × Amitriptyline
('qt-sertraline-amitriptyline', 'sertraline', 'Sertraline', 'amitriptyline', 'Amitriptyline',
 'moderate', 'established',
 'Sertraline moderate CYP2D6 inhibitor; increases amitriptyline levels less than fluoxetine/paroxetine',
 'Modest TCA level increase; QT prolongation possible at higher combined exposures',
 'More acceptable than fluoxetine/paroxetine but still requires monitoring; reduce amitriptyline dose 25-50%',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Sertraline (Zoloft) FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'Sertraline CYP2D6 Ki = 2.8μM (moderate inhibitor); ~50% increase in TCA levels', NULL, 'CYP2D6',
 'LEXICOMP', 'Moderate PK interaction - CredibleMeds Known Risk (amitriptyline)', 'SSRI Interaction Profile Comparison',
 'B', '2024-01-15'),

-- Fluvoxamine × Amitriptyline
('qt-fluvoxamine-amitriptyline', 'fluvoxamine', 'Fluvoxamine', 'amitriptyline', 'Amitriptyline',
 'major', 'established',
 'Fluvoxamine potent CYP1A2 and moderate CYP2D6 inhibitor; affects alternative amitriptyline metabolism pathways',
 'Significant TCA level increase; QT prolongation from elevated amitriptyline',
 'AVOID or reduce amitriptyline substantially; TDM and ECG monitoring required',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Fluvoxamine (Luvox) FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'Fluvoxamine inhibits CYP1A2 and CYP2D6; affects multiple TCA clearance pathways', NULL, 'CYP1A2, CYP2D6',
 'LEXICOMP', 'Major PK interaction - CredibleMeds Known Risk (amitriptyline)', 'OCD Treatment Drug Interactions',
 'A', '2024-01-15');

-- =====================================================================================
-- SECTION 6: ADDITIONAL ANTIDEPRESSANT COMBINATIONS
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
-- Venlafaxine × Amitriptyline
('qt-venlafaxine-amitriptyline', 'venlafaxine', 'Venlafaxine', 'amitriptyline', 'Amitriptyline',
 'major', 'probable',
 'Additive QT prolongation (venlafaxine Possible Risk + amitriptyline Known Risk); serotonin syndrome risk',
 'Combined cardiac and serotonergic toxicity; no clinical rationale for combination',
 'AVOID: Overlapping mechanisms without therapeutic benefit; cross-taper if switching',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Venlafaxine (Effexor) and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Venlafaxine Possible Risk + Amitriptyline Known Risk; combined IKr blockade', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'Antidepressant Switching Guidelines',
 'B', '2024-01-15'),

-- Venlafaxine × Imipramine
('qt-venlafaxine-imipramine', 'venlafaxine', 'Venlafaxine', 'imipramine', 'Imipramine',
 'major', 'probable',
 'Additive QT prolongation; dual serotonergic mechanism; no therapeutic rationale',
 'High-risk combination with overlapping toxicities',
 'AVOID: No clinical indication; if switching, adequate washout required',
 'delayed', 'good', 'qt_prolongation', true,
 'FDA_LABEL', 'Venlafaxine and Imipramine FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Both drugs contribute to QT prolongation; serotonergic overlap compounds risk', NULL, 'CYP2D6',
 'LEXICOMP', 'CredibleMeds Possible + Known Risk', 'APA Guidelines',
 'B', '2024-01-15'),

-- Duloxetine × Amitriptyline
('qt-duloxetine-amitriptyline', 'duloxetine', 'Duloxetine', 'amitriptyline', 'Amitriptyline',
 'moderate', 'probable',
 'Duloxetine moderate CYP2D6 inhibitor increases amitriptyline levels; amitriptyline causes QT prolongation',
 'Elevated TCA levels with QT risk; combination sometimes used for neuropathic pain',
 'If combining: lowest effective doses; monitor for TCA toxicity; ECG monitoring advisable',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Duloxetine (Cymbalta) FDA Label', 'https://dailymed.nlm.nih.gov', 'US',
 'DRUGBANK', 'Duloxetine CYP2D6 Ki = 2.5μM; moderate inhibition increases TCA levels', NULL, 'CYP2D6',
 'LEXICOMP', 'Moderate PK interaction - CredibleMeds Known Risk (amitriptyline)', 'Neuropathic Pain Guidelines',
 'B', '2024-01-15'),

-- Mirtazapine × Amitriptyline
('qt-mirtazapine-amitriptyline', 'mirtazapine', 'Mirtazapine', 'amitriptyline', 'Amitriptyline',
 'moderate', 'possible',
 'Additive sedation and cardiac effects; mirtazapine has minimal QT risk but adds to overall burden',
 'Profound sedation; modest additive QT risk primarily from amitriptyline',
 'Can be combined with caution for treatment-resistant cases; warn about sedation; ECG monitoring',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Mirtazapine (Remeron) and Amitriptyline FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Mirtazapine Conditional Risk only; amitriptyline is primary QT concern', NULL, 'CYP3A4, CYP1A2',
 'LEXICOMP', 'CredibleMeds Conditional + Known Risk', 'TRD Augmentation Strategies',
 'C', '2024-01-15'),

-- Lithium × Citalopram (important for bipolar depression)
('qt-lithium-citalopram', 'lithium', 'Lithium', 'citalopram', 'Citalopram',
 'moderate', 'probable',
 'Lithium may prolong QT; citalopram has dose-dependent QT prolongation; common bipolar combination',
 'Additive QT risk; serotonin syndrome rare but possible; commonly used together',
 'Monitor ECG particularly in patients with cardiac history; limit citalopram to 20mg; monitor lithium levels',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Lithium and Citalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Lithium Conditional Risk; citalopram Known Risk; combination acceptable with monitoring', NULL, 'Renal (lithium)',
 'LEXICOMP', 'CredibleMeds Known + Conditional Risk', 'APA Bipolar Guidelines',
 'B', '2024-01-15'),

-- Lithium × Escitalopram
('qt-lithium-escitalopram', 'lithium', 'Lithium', 'escitalopram', 'Escitalopram',
 'moderate', 'probable',
 'Both drugs may affect cardiac conduction; common combination in bipolar depression',
 'Modest additive QT risk; generally well-tolerated combination',
 'Standard monitoring; ECG if risk factors; escitalopram typically ≤20mg',
 'delayed', 'fair', 'qt_prolongation', true,
 'FDA_LABEL', 'Lithium and Escitalopram FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
 'CREDIBLEMEDS', 'Escitalopram Known Risk + Lithium Conditional Risk', NULL, 'Renal (lithium)',
 'LEXICOMP', 'CredibleMeds Known + Conditional Risk', 'CANMAT Bipolar Guidelines',
 'B', '2024-01-15');

-- =====================================================================================
-- VERIFICATION QUERY
-- =====================================================================================
-- SELECT COUNT(*) as total_qt_antidepressant_ddis
-- FROM drug_interactions
-- WHERE interaction_id LIKE 'qt-%'
--   AND category = 'qt_prolongation'
--   AND (drug_a_name IN ('Amitriptyline', 'Imipramine', 'Desipramine', 'Nortriptyline', 'Clomipramine',
--                        'Doxepin', 'Citalopram', 'Escitalopram', 'Trazodone', 'Venlafaxine',
--                        'Fluoxetine', 'Paroxetine', 'Sertraline', 'Fluvoxamine', 'Duloxetine',
--                        'Mirtazapine', 'Bupropion', 'Lithium')
--    OR  drug_b_name IN ('Amitriptyline', 'Imipramine', 'Desipramine', 'Nortriptyline', 'Clomipramine',
--                        'Doxepin', 'Citalopram', 'Escitalopram', 'Trazodone', 'Venlafaxine',
--                        'Fluoxetine', 'Paroxetine', 'Sertraline', 'Fluvoxamine', 'Duloxetine',
--                        'Mirtazapine', 'Bupropion', 'Lithium'));
-- Expected: ~45 DDIs from this migration
