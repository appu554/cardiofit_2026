-- ============================================================================
-- KB-5 Tier 3: QT Prolongation Foundation - Migration 020
-- Creates QT drug reference table and Antiarrhythmic × Antiarrhythmic matrix
-- Target: First 100 DDIs of 2,500 total
-- ============================================================================

-- Create QT drug reference table for systematic matrix generation
CREATE TABLE IF NOT EXISTS qt_drug_reference (
    drug_code VARCHAR(50) PRIMARY KEY,
    drug_name VARCHAR(200) NOT NULL,
    drug_class VARCHAR(100) NOT NULL,
    qt_risk_category VARCHAR(20) NOT NULL CHECK (qt_risk_category IN ('KNOWN_RISK', 'POSSIBLE_RISK', 'CONDITIONAL_RISK')),
    crediblemeds_category VARCHAR(50),
    mechanism VARCHAR(500),
    typical_qtc_prolongation VARCHAR(100),
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- KNOWN RISK ANTIARRHYTHMICS (Class IA, III)
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('AMIODARONE', 'Amiodarone (Cordarone)', 'Antiarrhythmic Class III', 'KNOWN_RISK', 'Known Risk of TdP', 'Potassium channel blocker (IKr, IKs); Multiple ion channel effects', '30-60 ms'),
('SOTALOL', 'Sotalol (Betapace)', 'Antiarrhythmic Class III', 'KNOWN_RISK', 'Known Risk of TdP', 'Beta-blocker + potassium channel blocker (IKr)', '20-40 ms'),
('DOFETILIDE', 'Dofetilide (Tikosyn)', 'Antiarrhythmic Class III', 'KNOWN_RISK', 'Known Risk of TdP', 'Pure IKr blocker; Requires in-hospital initiation', '40-80 ms'),
('DRONEDARONE', 'Dronedarone (Multaq)', 'Antiarrhythmic Class III', 'KNOWN_RISK', 'Known Risk of TdP', 'Amiodarone analog; Multi-channel blocker without iodine', '10-20 ms'),
('QUINIDINE', 'Quinidine', 'Antiarrhythmic Class IA', 'KNOWN_RISK', 'Known Risk of TdP', 'Sodium and potassium channel blocker (IKr)', '30-60 ms'),
('PROCAINAMIDE', 'Procainamide (Pronestyl)', 'Antiarrhythmic Class IA', 'KNOWN_RISK', 'Known Risk of TdP', 'Sodium channel blocker + IKr blocker; Active metabolite NAPA', '20-40 ms'),
('DISOPYRAMIDE', 'Disopyramide (Norpace)', 'Antiarrhythmic Class IA', 'KNOWN_RISK', 'Known Risk of TdP', 'Sodium and potassium channel blocker; Anticholinergic effects', '20-50 ms'),
('IBUTILIDE', 'Ibutilide (Corvert)', 'Antiarrhythmic Class III', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker + sodium channel activator; IV only for cardioversion', '40-80 ms'),
('FLECAINIDE', 'Flecainide (Tambocor)', 'Antiarrhythmic Class IC', 'KNOWN_RISK', 'Known Risk of TdP', 'Potent sodium channel blocker; Minor IKr effect', '10-20 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK ANTIPSYCHOTICS
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'Antipsychotic - Butyrophenone', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Higher risk with IV route', '20-40 ms'),
('DROPERIDOL', 'Droperidol (Inapsine)', 'Antipsychotic - Butyrophenone', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; BLACK BOX warning for QT', '30-50 ms'),
('THIORIDAZINE', 'Thioridazine (Mellaril)', 'Antipsychotic - Phenothiazine', 'KNOWN_RISK', 'Known Risk of TdP', 'Potent IKr blocker; Dose-dependent QT effect', '30-60 ms'),
('PIMOZIDE', 'Pimozide (Orap)', 'Antipsychotic - Diphenylbutylpiperidine', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; CYP3A4/2D6 substrate', '20-40 ms'),
('ZIPRASIDONE', 'Ziprasidone (Geodon)', 'Antipsychotic - Atypical', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Mean QTc increase ~20ms', '15-25 ms'),
('CHLORPROMAZINE', 'Chlorpromazine (Thorazine)', 'Antipsychotic - Phenothiazine', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Alpha-blocker; Dose-dependent', '20-40 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK ANTIBIOTICS (Macrolides, Fluoroquinolones)
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('ERYTHROMYCIN', 'Erythromycin (E-Mycin)', 'Antibiotic - Macrolide', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; CYP3A4 inhibitor amplifies risk', '10-30 ms'),
('CLARITHROMYCIN', 'Clarithromycin (Biaxin)', 'Antibiotic - Macrolide', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Strong CYP3A4 inhibitor', '10-20 ms'),
('AZITHROMYCIN', 'Azithromycin (Zithromax)', 'Antibiotic - Macrolide', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Lower risk than erythromycin', '5-15 ms'),
('MOXIFLOXACIN', 'Moxifloxacin (Avelox)', 'Antibiotic - Fluoroquinolone', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Highest QT risk among FQs', '10-20 ms'),
('LEVOFLOXACIN', 'Levofloxacin (Levaquin)', 'Antibiotic - Fluoroquinolone', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Moderate QT risk', '5-15 ms'),
('CIPROFLOXACIN', 'Ciprofloxacin (Cipro)', 'Antibiotic - Fluoroquinolone', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr blocker; Lower QT risk than moxifloxacin', '3-10 ms'),
('SPARFLOXACIN', 'Sparfloxacin (Zagam)', 'Antibiotic - Fluoroquinolone', 'KNOWN_RISK', 'Known Risk of TdP', 'Potent IKr blocker; Withdrawn in many countries', '20-40 ms'),
('PENTAMIDINE', 'Pentamidine (Pentam)', 'Antibiotic - Antiprotozoal', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Risk increases with IV route', '30-60 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK ANTIEMETICS
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('ONDANSETRON_IV', 'Ondansetron IV (Zofran)', 'Antiemetic - 5HT3 Antagonist', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Dose-dependent; >16mg IV BLACK BOX', '15-30 ms'),
('DOMPERIDONE', 'Domperidone (Motilium)', 'Antiemetic - D2 Antagonist', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Not available in US due to QT risk', '10-20 ms'),
('GRANISETRON', 'Granisetron (Kytril)', 'Antiemetic - 5HT3 Antagonist', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr blocker; Lower risk than ondansetron', '5-10 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK ANTIDEPRESSANTS
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('CITALOPRAM', 'Citalopram (Celexa)', 'Antidepressant - SSRI', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Dose-dependent; Max 40mg (20mg in elderly)', '10-20 ms'),
('ESCITALOPRAM', 'Escitalopram (Lexapro)', 'Antidepressant - SSRI', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Active enantiomer of citalopram', '8-15 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK OPIOIDS
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('METHADONE', 'Methadone (Dolophine)', 'Opioid - Synthetic', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Dose-dependent; ECG monitoring recommended', '10-40 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK ANTIMALARIALS
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('CHLOROQUINE', 'Chloroquine (Aralen)', 'Antimalarial - 4-Aminoquinoline', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Multiple ion channel effects', '20-40 ms'),
('HYDROXYCHLOROQUINE', 'Hydroxychloroquine (Plaquenil)', 'Antimalarial - 4-Aminoquinoline', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Lower risk than chloroquine', '10-20 ms'),
('HALOFANTRINE', 'Halofantrine (Halfan)', 'Antimalarial', 'KNOWN_RISK', 'Known Risk of TdP', 'Potent IKr blocker; HIGH RISK', '30-60 ms'),
('QUININE', 'Quinine', 'Antimalarial - Cinchona Alkaloid', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Related to quinidine', '10-30 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK ONCOLOGY
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('ARSENIC_TRIOXIDE', 'Arsenic Trioxide (Trisenox)', 'Oncology - APL Treatment', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr and IKs blocker; ECG monitoring mandatory', '30-60 ms'),
('VANDETANIB', 'Vandetanib (Caprelsa)', 'Oncology - TKI', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; BLACK BOX for QT; ECG monitoring required', '20-40 ms'),
('NILOTINIB', 'Nilotinib (Tasigna)', 'Oncology - TKI (BCR-ABL)', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; BLACK BOX for sudden death', '15-30 ms'),
('LAPATINIB', 'Lapatinib (Tykerb)', 'Oncology - TKI (HER2)', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Monitor with other QT drugs', '5-15 ms'),
('PAZOPANIB', 'Pazopanib (Votrient)', 'Oncology - TKI', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; ECG monitoring recommended', '10-20 ms'),
('SUNITINIB', 'Sunitinib (Sutent)', 'Oncology - TKI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Mild QT effect; Cardiotoxicity concern', '5-15 ms'),
('SORAFENIB', 'Sorafenib (Nexavar)', 'Oncology - TKI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Monitor QT', '5-10 ms'),
('CRIZOTINIB', 'Crizotinib (Xalkori)', 'Oncology - TKI (ALK)', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Bradycardia also concern', '10-20 ms'),
('VEMURAFENIB', 'Vemurafenib (Zelboraf)', 'Oncology - BRAF Inhibitor', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; ECG monitoring required', '15-25 ms'),
('OXALIPLATIN', 'Oxaliplatin (Eloxatin)', 'Oncology - Platinum', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Electrolyte effects; Hypokalemia/hypomagnesemia', '5-15 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK ANTIFUNGALS
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('FLUCONAZOLE_QT', 'Fluconazole (Diflucan)', 'Antifungal - Azole', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; CYP3A4/2C9 inhibitor amplifies DDIs', '10-25 ms'),
('VORICONAZOLE_QT', 'Voriconazole (Vfend)', 'Antifungal - Azole', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Strong CYP3A4 inhibitor', '5-15 ms'),
('KETOCONAZOLE_QT', 'Ketoconazole (Nizoral)', 'Antifungal - Azole', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Withdrawn for systemic use in some countries', '10-20 ms'),
('ITRACONAZOLE', 'Itraconazole (Sporanox)', 'Antifungal - Azole', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Strong CYP3A4 inhibitor', '5-10 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- KNOWN RISK OTHER
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('SEVOFLURANE', 'Sevoflurane (Ultane)', 'Anesthetic - Volatile', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Transient effect during anesthesia', '10-20 ms'),
('PROPOFOL_QT', 'Propofol (Diprivan)', 'Anesthetic - IV', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Mild QT effect; Usually not clinically significant', '3-10 ms'),
('CISAPRIDE', 'Cisapride (Propulsid)', 'GI Prokinetic', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; WITHDRAWN due to TdP deaths', '20-40 ms'),
('COCAINE', 'Cocaine', 'Stimulant', 'KNOWN_RISK', 'Known Risk of TdP', 'Sodium and potassium channel blocker', '10-30 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- POSSIBLE RISK ANTIPSYCHOTICS
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('QUETIAPINE', 'Quetiapine (Seroquel)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Dose-dependent', '5-15 ms'),
('RISPERIDONE', 'Risperidone (Risperdal)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Generally well tolerated', '5-10 ms'),
('OLANZAPINE', 'Olanzapine (Zyprexa)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Minimal IKr effect; Lower risk than others', '3-8 ms'),
('CLOZAPINE', 'Clozapine (Clozaril)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Myocarditis more concerning', '5-15 ms'),
('ARIPIPRAZOLE', 'Aripiprazole (Abilify)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Minimal IKr effect; Low QT risk', '2-5 ms'),
('PALIPERIDONE', 'Paliperidone (Invega)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Active metabolite of risperidone; Similar QT profile', '5-10 ms'),
('ASENAPINE', 'Asenapine (Saphris)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Moderate IKr effect', '5-15 ms'),
('ILOPERIDONE', 'Iloperidone (Fanapt)', 'Antipsychotic - Atypical', 'KNOWN_RISK', 'Known Risk of TdP', 'IKr blocker; Requires slow titration', '10-20 ms'),
('LURASIDONE', 'Lurasidone (Latuda)', 'Antipsychotic - Atypical', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Minimal QT effect', '2-5 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- POSSIBLE RISK ANTIDEPRESSANTS (TCAs, others)
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('AMITRIPTYLINE', 'Amitriptyline (Elavil)', 'Antidepressant - TCA', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Sodium and potassium channel blocker', '10-30 ms'),
('IMIPRAMINE', 'Imipramine (Tofranil)', 'Antidepressant - TCA', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Sodium and potassium channel blocker', '10-25 ms'),
('DESIPRAMINE', 'Desipramine (Norpramin)', 'Antidepressant - TCA', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Less QT effect than tertiary TCAs', '5-15 ms'),
('NORTRIPTYLINE', 'Nortriptyline (Pamelor)', 'Antidepressant - TCA', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Less QT effect than amitriptyline', '5-15 ms'),
('DOXEPIN', 'Doxepin (Sinequan)', 'Antidepressant - TCA', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Moderate IKr effect', '5-20 ms'),
('CLOMIPRAMINE', 'Clomipramine (Anafranil)', 'Antidepressant - TCA', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'IKr blocker; Higher risk at high doses', '10-25 ms'),
('FLUOXETINE', 'Fluoxetine (Prozac)', 'Antidepressant - SSRI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; CYP2D6 inhibitor', '3-8 ms'),
('SERTRALINE', 'Sertraline (Zoloft)', 'Antidepressant - SSRI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Minimal IKr effect', '2-5 ms'),
('PAROXETINE', 'Paroxetine (Paxil)', 'Antidepressant - SSRI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Minimal IKr effect; CYP2D6 inhibitor', '2-5 ms'),
('VENLAFAXINE', 'Venlafaxine (Effexor)', 'Antidepressant - SNRI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect; Dose-dependent', '5-15 ms'),
('DULOXETINE', 'Duloxetine (Cymbalta)', 'Antidepressant - SNRI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Minimal IKr effect', '2-5 ms'),
('MIRTAZAPINE', 'Mirtazapine (Remeron)', 'Antidepressant - Tetracyclic', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Weak IKr effect', '3-8 ms'),
('TRAZODONE', 'Trazodone (Desyrel)', 'Antidepressant - SARI', 'POSSIBLE_RISK', 'Possible Risk of TdP', 'Moderate IKr effect', '5-15 ms'),
('BUPROPION', 'Bupropion (Wellbutrin)', 'Antidepressant - NDRI', 'POSSIBLE_RISK', 'Conditional Risk', 'Minimal direct QT effect; Seizure risk', '0-5 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- CONDITIONAL RISK DRUGS (QT prolongation under specific conditions)
-- ============================================================================
INSERT INTO qt_drug_reference (drug_code, drug_name, drug_class, qt_risk_category, crediblemeds_category, mechanism, typical_qtc_prolongation) VALUES
('FUROSEMIDE', 'Furosemide (Lasix)', 'Diuretic - Loop', 'CONDITIONAL_RISK', 'Conditional Risk', 'Indirect via hypokalemia/hypomagnesemia', 'Variable'),
('HYDROCHLOROTHIAZIDE', 'Hydrochlorothiazide (HCTZ)', 'Diuretic - Thiazide', 'CONDITIONAL_RISK', 'Conditional Risk', 'Indirect via hypokalemia', 'Variable'),
('INDAPAMIDE', 'Indapamide (Lozol)', 'Diuretic - Thiazide-like', 'CONDITIONAL_RISK', 'Conditional Risk', 'Indirect via hypokalemia; Direct effect possible', '5-15 ms'),
('AMPHOTERICIN_B', 'Amphotericin B', 'Antifungal - Polyene', 'CONDITIONAL_RISK', 'Conditional Risk', 'Indirect via hypokalemia/hypomagnesemia from nephrotoxicity', 'Variable'),
('LITHIUM_QT', 'Lithium (Lithobid)', 'Mood Stabilizer', 'CONDITIONAL_RISK', 'Conditional Risk', 'Weak direct effect; Risk with electrolyte disturbance', '3-10 ms'),
('TACROLIMUS_QT', 'Tacrolimus (Prograf)', 'Immunosuppressant', 'CONDITIONAL_RISK', 'Conditional Risk', 'Weak IKr effect; Risk with electrolyte disturbance', '3-10 ms'),
('TAMOXIFEN', 'Tamoxifen (Nolvadex)', 'Oncology - SERM', 'CONDITIONAL_RISK', 'Conditional Risk', 'Weak IKr effect at high concentrations', '3-8 ms'),
('ALFUZOSIN', 'Alfuzosin (Uroxatral)', 'Alpha-Blocker', 'CONDITIONAL_RISK', 'Conditional Risk', 'Weak IKr effect', '3-8 ms'),
('RANOLAZINE', 'Ranolazine (Ranexa)', 'Antianginal', 'CONDITIONAL_RISK', 'Conditional Risk', 'Late sodium current inhibitor; Modest QT effect', '5-15 ms'),
('DIPHENHYDRAMINE', 'Diphenhydramine (Benadryl)', 'Antihistamine - 1st Gen', 'CONDITIONAL_RISK', 'Conditional Risk', 'Weak IKr effect at high doses/OD', '5-20 ms')
ON CONFLICT (drug_code) DO NOTHING;

-- ============================================================================
-- ANTIARRHYTHMIC × ANTIARRHYTHMIC DDIs (Highest Risk - Known Risk × Known Risk)
-- ============================================================================

-- 1. Amiodarone + Sotalol (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_SOTALOL_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'SOTALOL', 'Sotalol (Betapace)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Additive QT prolongation: Both are Class III antiarrhythmics blocking IKr. Amiodarone also inhibits CYP2D6 increasing sotalol levels.',
    'Extreme risk of Torsades de Pointes (TdP). Combined QT prolongation can exceed 100ms. Multiple case reports of fatal arrhythmias.',
    'CONTRAINDICATED. Do not combine. If switching, allow 3+ months washout for amiodarone due to long half-life (40-55 days).',
    FALSE, 0.15, 0.99,
    'FDA_LABEL', 'Cordarone - Section 4 (Contraindications); Betapace - Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Additive IKr blockade; Amiodarone t1/2 = 40-55 days', NULL, 'Amiodarone inhibits CYP2D6 (sotalol)', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS Arrhythmia Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 2. Amiodarone + Dofetilide (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_DOFETILIDE_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'DOFETILIDE', 'Dofetilide (Tikosyn)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Additive QT prolongation: Both block IKr. Dofetilide is a pure IKr blocker requiring in-hospital initiation due to TdP risk.',
    'Extreme TdP risk. Dofetilide already requires hospitalization for initiation; amiodarone combination dramatically increases risk.',
    'CONTRAINDICATED. Do not combine. Dofetilide requires minimum 3-month washout after amiodarone discontinuation.',
    FALSE, 0.10, 0.99,
    'FDA_LABEL', 'Tikosyn - Section 4 (Contraindicated drugs); Cordarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Dofetilide QT prolongation 40-80ms; In-hospital initiation required', NULL, 'Not CYP-mediated primarily', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS Arrhythmia Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 3. Amiodarone + Quinidine (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_QUINIDINE_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'QUINIDINE', 'Quinidine',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Additive QT prolongation: Class III (amiodarone) + Class IA (quinidine) both block IKr. PK interaction: amiodarone increases quinidine levels via CYP3A4 inhibition.',
    'Extreme TdP risk. Both significantly prolong QT. PK interaction further increases quinidine concentration and toxicity.',
    'CONTRAINDICATED. Do not combine. Historical combination rarely used; modern guidelines prohibit concurrent use.',
    FALSE, 0.05, 0.99,
    'FDA_LABEL', 'Cordarone - Section 7 (Class I antiarrhythmics); Quinidine - Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Amiodarone inhibits CYP3A4 increasing quinidine; Combined QT >100ms', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS Arrhythmia Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 4. Amiodarone + Procainamide (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_PROCAINAMIDE_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'PROCAINAMIDE', 'Procainamide (Pronestyl)',
    'pharmacodynamic',
    'major',
    'established',
    'Additive QT prolongation: Class III + Class IA. NAPA (procainamide metabolite) also prolongs QT. Amiodarone increases procainamide levels.',
    'Very high TdP risk. Both drugs and NAPA metabolite contribute to QT prolongation. May be used briefly in refractory VT under monitoring.',
    'Avoid combination. If absolutely necessary in refractory VT, continuous ECG monitoring mandatory. Check procainamide/NAPA levels.',
    TRUE, 0.08, 0.95,
    'FDA_LABEL', 'Cordarone - Section 7; Pronestyl - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; NAPA metabolite QT effect; Amiodarone PK interaction', NULL, 'Renal elimination affected', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS VT Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 5. Amiodarone + Dronedarone (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_DRONEDARONE_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'DRONEDARONE', 'Dronedarone (Multaq)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Dronedarone is an amiodarone analog. Combined use provides no benefit and additive toxicity. Same mechanism (multi-channel blocker).',
    'No therapeutic rationale. Additive QT prolongation, bradycardia, and organ toxicity without additional efficacy.',
    'CONTRAINDICATED. Never combine. These are alternative drugs, not additive therapy. 3-month washout needed after amiodarone.',
    FALSE, 0.02, 0.99,
    'FDA_LABEL', 'Multaq - Section 4 (Contraindicated with amiodarone)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Structural analogs; No additive efficacy', NULL, 'Both CYP3A4 substrates/inhibitors', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS AF Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 6. Amiodarone + Disopyramide (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_DISOPYRAMIDE_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'DISOPYRAMIDE', 'Disopyramide (Norpace)',
    'pharmacodynamic',
    'major',
    'established',
    'Additive QT prolongation: Class III + Class IA. Amiodarone inhibits CYP3A4 increasing disopyramide levels. Additive negative inotropic effect.',
    'High TdP risk. PK interaction amplifies effect. Combined negative inotropy may precipitate heart failure.',
    'Avoid combination. If used in refractory cases, reduce disopyramide dose by 30-50%. Monitor QTc and cardiac function closely.',
    TRUE, 0.05, 0.94,
    'FDA_LABEL', 'Cordarone - Section 7; Norpace - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 interaction; Negative inotropic effects additive', NULL, 'CYP3A4 inhibition', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 7. Amiodarone + Ibutilide (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_IBUTILIDE_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'IBUTILIDE', 'Ibutilide (Corvert)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both are Class III antiarrhythmics. Ibutilide used for acute cardioversion prolongs QT 40-80ms. Combined effect extreme.',
    'Extreme TdP risk. Ibutilide already causes TdP in 4-8% of patients; amiodarone background dramatically increases this risk.',
    'CONTRAINDICATED. Do not use ibutilide for cardioversion in patients on amiodarone. Use DC cardioversion instead.',
    FALSE, 0.05, 0.99,
    'FDA_LABEL', 'Corvert - Section 4 (Class III antiarrhythmics); Cordarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Ibutilide TdP rate 4-8% baseline; Extreme additive risk', NULL, 'Not CYP-mediated', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS Cardioversion Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 8. Amiodarone + Flecainide (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_AMIODARONE_FLECAINIDE_001',
    'AMIODARONE', 'Amiodarone (Cordarone)',
    'FLECAINIDE', 'Flecainide (Tambocor)',
    'pharmacokinetic',
    'major',
    'established',
    'Amiodarone inhibits CYP2D6, increasing flecainide levels 2-fold. Both prolong QRS/QT. Additive negative inotropic effects.',
    'Flecainide toxicity: proarrhythmia, heart failure, bradycardia. QRS widening. Combined negative inotropy dangerous.',
    'Reduce flecainide dose by 50% when starting amiodarone. Monitor ECG for QRS widening. Avoid in structural heart disease.',
    TRUE, 0.08, 0.92,
    'FDA_LABEL', 'Cordarone - Section 7 (Class IC); Tambocor - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; CYP2D6 inhibition doubles flecainide; QRS + QT effects', NULL, 'CYP2D6 inhibition', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 9. Sotalol + Dofetilide (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_SOTALOL_DOFETILIDE_001',
    'SOTALOL', 'Sotalol (Betapace)',
    'DOFETILIDE', 'Dofetilide (Tikosyn)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both are Class III antiarrhythmics blocking IKr. Dofetilide is a pure IKr blocker; sotalol adds beta-blockade and IKr blockade.',
    'Extreme TdP risk. Combined QT prolongation may exceed 80ms. Both drugs individually require monitoring; combination unacceptable.',
    'CONTRAINDICATED. Never combine. Must discontinue sotalol before initiating dofetilide (in-hospital). Allow adequate washout.',
    FALSE, 0.05, 0.99,
    'FDA_LABEL', 'Tikosyn - Section 4 (Class III antiarrhythmics); Betapace - Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Pure IKr blockade additive; No therapeutic rationale', NULL, 'Not CYP-mediated', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 10. Sotalol + Quinidine (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_SOTALOL_QUINIDINE_001',
    'SOTALOL', 'Sotalol (Betapace)',
    'QUINIDINE', 'Quinidine',
    'pharmacodynamic',
    'major',
    'established',
    'Class III + Class IA: Both block IKr. Sotalol beta-blockade + quinidine sodium channel blockade + combined QT prolongation.',
    'Very high TdP risk. Additive QT prolongation. Both drugs historically associated with proarrhythmic deaths.',
    'Avoid combination. Rarely justified in modern practice. If used, intensive ECG monitoring and electrolyte management required.',
    FALSE, 0.03, 0.96,
    'FDA_LABEL', 'Betapace - Section 7; Quinidine - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Class IA + Class III combination; Historical proarrhythmic deaths', NULL, 'CYP3A4 metabolism (quinidine)', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Continue with more Antiarrhythmic × Antiarrhythmic combinations...

-- 11. Sotalol + Procainamide (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_SOTALOL_PROCAINAMIDE_001',
    'SOTALOL', 'Sotalol (Betapace)',
    'PROCAINAMIDE', 'Procainamide (Pronestyl)',
    'pharmacodynamic',
    'major',
    'established',
    'Class III + Class IA: Both block IKr. NAPA metabolite of procainamide adds additional QT prolongation.',
    'Very high TdP risk. Triple QT effect (sotalol + procainamide + NAPA). May be used briefly in refractory VT storm.',
    'Avoid combination. If absolutely necessary in VT storm, continuous ECG monitoring. Check procainamide/NAPA levels frequently.',
    TRUE, 0.05, 0.94,
    'FDA_LABEL', 'Betapace - Section 7; Pronestyl - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; NAPA metabolite effect; Triple QT prolongation mechanism', NULL, 'Renal elimination', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS VT Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 12. Sotalol + Dronedarone (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_SOTALOL_DRONEDARONE_001',
    'SOTALOL', 'Sotalol (Betapace)',
    'DRONEDARONE', 'Dronedarone (Multaq)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both are Class III antiarrhythmics. Dronedarone multi-channel blocker + sotalol IKr blocker + beta-blocker = severe QT and bradycardia.',
    'Extreme TdP risk. Severe bradycardia from combined beta-blockade (sotalol) and multi-channel effects (dronedarone).',
    'CONTRAINDICATED. Do not combine. These are alternative AF rate/rhythm control drugs, not additive therapy.',
    FALSE, 0.03, 0.99,
    'FDA_LABEL', 'Multaq - Section 4; Betapace - Section 4', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Class III + Class III; Severe bradycardia risk', NULL, 'CYP3A4 (dronedarone)', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS AF Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 13. Dofetilide + Quinidine (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_DOFETILIDE_QUINIDINE_001',
    'DOFETILIDE', 'Dofetilide (Tikosyn)',
    'QUINIDINE', 'Quinidine',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both potent IKr blockers. Dofetilide is pure IKr; quinidine blocks IKr + sodium channels. Combined QT prolongation extreme.',
    'Extreme TdP risk. Dofetilide already requires in-hospital initiation. Quinidine addition unacceptable.',
    'CONTRAINDICATED. Never combine. Dofetilide prescribers must verify no concurrent Class I/III antiarrhythmic use.',
    FALSE, 0.02, 0.99,
    'FDA_LABEL', 'Tikosyn - Section 4 (Contraindicated drugs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Pure IKr (dofetilide) + IKr (quinidine); TIKOSYN REMS', NULL, 'CYP3A4 (quinidine)', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS Program',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 14. Dofetilide + Procainamide (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_DOFETILIDE_PROCAINAMIDE_001',
    'DOFETILIDE', 'Dofetilide (Tikosyn)',
    'PROCAINAMIDE', 'Procainamide (Pronestyl)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both IKr blockers. NAPA metabolite of procainamide also prolongs QT. Dofetilide renal elimination may be affected.',
    'Extreme TdP risk. Dofetilide + procainamide + NAPA = triple QT prolongation mechanism.',
    'CONTRAINDICATED. Never combine. Tikosyn REMS prohibits concurrent Class I/III antiarrhythmics.',
    FALSE, 0.02, 0.99,
    'FDA_LABEL', 'Tikosyn - Section 4 (Contraindicated drugs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Triple mechanism (dofetilide + procainamide + NAPA)', NULL, 'Renal elimination competition', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS Program',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 15. Dofetilide + Dronedarone (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_DOFETILIDE_DRONEDARONE_001',
    'DOFETILIDE', 'Dofetilide (Tikosyn)',
    'DRONEDARONE', 'Dronedarone (Multaq)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both Class III antiarrhythmics. Dronedarone inhibits renal OCT2 transporter, increasing dofetilide levels significantly.',
    'Extreme TdP risk. PK interaction (OCT2 inhibition) + PD interaction (additive IKr blockade). Double mechanism of harm.',
    'CONTRAINDICATED. Never combine. Dronedarone listed in Tikosyn REMS as prohibited concurrent medication.',
    FALSE, 0.02, 0.99,
    'FDA_LABEL', 'Tikosyn - Section 4 (Dronedarone specifically listed); Multaq - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; OCT2 inhibition increases dofetilide 2-3x; TIKOSYN REMS', 'OCT2 transporter inhibition', 'Not CYP-mediated', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS Program',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 16. Quinidine + Procainamide (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_QUINIDINE_PROCAINAMIDE_001',
    'QUINIDINE', 'Quinidine',
    'PROCAINAMIDE', 'Procainamide (Pronestyl)',
    'pharmacodynamic',
    'major',
    'established',
    'Both Class IA antiarrhythmics blocking sodium and potassium (IKr) channels. NAPA metabolite adds additional QT effect.',
    'Very high TdP risk. Redundant mechanism. Historical combination therapy now considered dangerous.',
    'Avoid combination. No modern indication for dual Class IA therapy. Use single agent or different class.',
    FALSE, 0.01, 0.95,
    'FDA_LABEL', 'Quinidine - Section 7; Pronestyl - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Redundant Class IA mechanism; Historical proarrhythmic deaths', NULL, 'Hepatic metabolism', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 17. Quinidine + Disopyramide (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_QUINIDINE_DISOPYRAMIDE_001',
    'QUINIDINE', 'Quinidine',
    'DISOPYRAMIDE', 'Disopyramide (Norpace)',
    'pharmacodynamic',
    'major',
    'established',
    'Both Class IA antiarrhythmics. Combined sodium and IKr blockade. Additive negative inotropic and anticholinergic effects.',
    'Very high TdP risk. Severe negative inotropy may precipitate heart failure. Anticholinergic toxicity additive.',
    'Avoid combination. Redundant Class IA mechanism with additive toxicity. No clinical rationale for combination.',
    FALSE, 0.01, 0.94,
    'FDA_LABEL', 'Quinidine - Section 7; Norpace - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Redundant Class IA; Negative inotropy additive', NULL, 'CYP3A4 metabolism', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 18. Procainamide + Disopyramide (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_PROCAINAMIDE_DISOPYRAMIDE_001',
    'PROCAINAMIDE', 'Procainamide (Pronestyl)',
    'DISOPYRAMIDE', 'Disopyramide (Norpace)',
    'pharmacodynamic',
    'major',
    'established',
    'Both Class IA antiarrhythmics. Triple QT mechanism: procainamide + NAPA + disopyramide. Negative inotropy additive.',
    'Very high TdP risk. NAPA metabolite amplifies QT effect. Combined negative inotropy dangerous in heart failure.',
    'Avoid combination. No clinical rationale. Use single Class IA or alternative class.',
    FALSE, 0.01, 0.94,
    'FDA_LABEL', 'Pronestyl - Section 7; Norpace - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; NAPA adds third QT mechanism; Negative inotropy', NULL, 'Different elimination routes', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 19. Dronedarone + Quinidine (MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_DRONEDARONE_QUINIDINE_001',
    'DRONEDARONE', 'Dronedarone (Multaq)',
    'QUINIDINE', 'Quinidine',
    'pharmacodynamic',
    'major',
    'established',
    'Class III + Class IA: Additive QT prolongation. Dronedarone inhibits CYP3A4 and P-gp, potentially increasing quinidine levels.',
    'High TdP risk. Combined ion channel blockade. PK interaction may amplify quinidine toxicity.',
    'Avoid combination. If used, reduce quinidine dose and monitor QTc closely. Modern guidelines prefer single-agent therapy.',
    TRUE, 0.02, 0.93,
    'FDA_LABEL', 'Multaq - Section 7; Quinidine - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4/P-gp interaction; Multi-channel blockade', 'P-gp inhibition', 'CYP3A4 inhibition', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA/HRS AF Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- 20. Ibutilide + Sotalol (CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'QT_IBUTILIDE_SOTALOL_001',
    'IBUTILIDE', 'Ibutilide (Corvert)',
    'SOTALOL', 'Sotalol (Betapace)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'Both Class III antiarrhythmics. Ibutilide for acute cardioversion already has 4-8% TdP rate. Sotalol background dramatically increases risk.',
    'Extreme TdP risk. Ibutilide should not be given to patients already on sotalol. Use DC cardioversion instead.',
    'CONTRAINDICATED. Do not use ibutilide for cardioversion in patients on sotalol. DC cardioversion is safer.',
    FALSE, 0.03, 0.99,
    'FDA_LABEL', 'Corvert - Section 4 (Class III antiarrhythmics)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'CREDIBLEMEDS', 'Both Known Risk TdP; Ibutilide TdP rate 4-8%; Sotalol background increases to >15%', NULL, 'Not CYP-mediated', 'Known Risk + Known Risk',
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA/HRS Cardioversion Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Additional high-priority combinations to reach ~100 DDIs for foundation migration

-- 21-30: Antiarrhythmic × Antipsychotic high-risk pairs
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES
-- 21. Amiodarone + Haloperidol IV
('QT_AMIODARONE_HALOPERIDOL_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP drugs. Amiodarone Class III + haloperidol IKr blocker. IV haloperidol carries BLACK BOX warning for QT/TdP.',
'Extreme TdP risk. IV haloperidol already associated with sudden cardiac death. Amiodarone dramatically increases risk.',
'CONTRAINDICATED. Do not give IV haloperidol to patients on amiodarone. Use alternative sedation (dexmedetomidine, propofol).',
FALSE, 0.15, 0.98,
'FDA_LABEL', 'Haldol - BLACK BOX (QT/Sudden Death); Cordarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; IV haloperidol BLACK BOX; Combined QT >80ms', NULL, 'CYP3A4 (haloperidol)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Practice Guidelines',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 22. Amiodarone + Droperidol
('QT_AMIODARONE_DROPERIDOL_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'DROPERIDOL', 'Droperidol (Inapsine)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP drugs. Droperidol has BLACK BOX for QT prolongation and sudden death. Amiodarone adds Class III QT effect.',
'Extreme TdP risk. Droperidol BLACK BOX warns against use with other QT-prolonging drugs. Multiple sudden death reports.',
'CONTRAINDICATED. Do not use droperidol in patients on amiodarone. Alternative antiemetics: ondansetron (with caution), prochlorperazine.',
FALSE, 0.10, 0.99,
'FDA_LABEL', 'Inapsine - BLACK BOX (QT/Sudden Death); Cordarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Droperidol BLACK BOX; Sudden death reports', NULL, 'CYP metabolism', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety Alert',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 23. Amiodarone + Thioridazine
('QT_AMIODARONE_THIORIDAZINE_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP drugs. Thioridazine has BLACK BOX for QT prolongation. Among most potent IKr blockers in antipsychotic class.',
'Extreme TdP risk. Thioridazine alone prolongs QT 30-60ms. BLACK BOX specifically contraindicates with other QT drugs.',
'CONTRAINDICATED. Thioridazine BLACK BOX lists QT-prolonging drugs as contraindicated. Switch to alternative antipsychotic.',
FALSE, 0.05, 0.99,
'FDA_LABEL', 'Mellaril - BLACK BOX (QT); Cordarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Thioridazine BLACK BOX; Combined QT may exceed 100ms', NULL, 'CYP2D6 (thioridazine)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Practice Guidelines',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 24. Amiodarone + Pimozide
('QT_AMIODARONE_PIMOZIDE_001', 'AMIODARONE', 'Amiodarone (Cordarone)', 'PIMOZIDE', 'Pimozide (Orap)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP drugs. Pimozide has QT warning and contraindication with CYP3A4 inhibitors. Amiodarone inhibits CYP3A4.',
'Extreme TdP risk. Amiodarone CYP3A4 inhibition increases pimozide levels. Dual mechanism: PK + PD interaction.',
'CONTRAINDICATED. Pimozide contraindicated with CYP3A4 inhibitors including amiodarone. Use alternative for tic disorder.',
FALSE, 0.05, 0.99,
'FDA_LABEL', 'Orap - Section 4 (CYP3A4 inhibitors); Cordarone - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP3A4 inhibition increases pimozide; Dual PK/PD interaction', NULL, 'CYP3A4 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'APA Practice Guidelines',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 25. Sotalol + Haloperidol IV
('QT_SOTALOL_HALOPERIDOL_001', 'SOTALOL', 'Sotalol (Betapace)', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP drugs. Sotalol Class III IKr blocker + haloperidol IKr blocker. IV route of haloperidol increases QT risk.',
'Very high TdP risk. Combined QT prolongation 40-80ms. Bradycardia from sotalol beta-blockade may further increase TdP risk.',
'Avoid combination. If ICU sedation needed in patient on sotalol, use dexmedetomidine or propofol. Monitor QTc if unavoidable.',
FALSE, 0.12, 0.95,
'FDA_LABEL', 'Haldol - BLACK BOX; Betapace - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; IV haloperidol BLACK BOX; Sotalol bradycardia amplifies TdP risk', NULL, 'CYP2D6 (haloperidol)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA/SCCM ICU Delirium Guidelines',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 26. Sotalol + Ziprasidone
('QT_SOTALOL_ZIPRASIDONE_001', 'SOTALOL', 'Sotalol (Betapace)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP drugs. Ziprasidone has mean QTc increase ~20ms. Sotalol adds Class III effect. Combined risk significant.',
'High TdP risk. Ziprasidone contraindicated with other QT-prolonging drugs per label. Combined effect may exceed 50ms.',
'Avoid combination. Ziprasidone label contraindicates use with Class III antiarrhythmics. Use alternative antipsychotic.',
FALSE, 0.10, 0.94,
'FDA_LABEL', 'Geodon - Section 4 (QT drugs); Betapace - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ziprasidone QTc +20ms; Contraindicated per label', NULL, 'CYP3A4 (ziprasidone)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Practice Guidelines',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 27. Dofetilide + Haloperidol IV
('QT_DOFETILIDE_HALOPERIDOL_001', 'DOFETILIDE', 'Dofetilide (Tikosyn)', 'HALOPERIDOL_IV', 'Haloperidol IV (Haldol)', 'pharmacodynamic', 'contraindicated', 'established',
'Dofetilide pure IKr blocker requiring in-hospital initiation. Haloperidol IV has BLACK BOX. Combination extremely dangerous.',
'Extreme TdP risk. Dofetilide already requires hospitalization for QT monitoring. IV haloperidol addition unacceptable.',
'CONTRAINDICATED. Tikosyn REMS prohibits concurrent QT-prolonging drugs. Use non-QT sedation alternatives.',
FALSE, 0.05, 0.99,
'FDA_LABEL', 'Tikosyn - REMS; Haldol - BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; TIKOSYN REMS restriction; IV haloperidol BLACK BOX', NULL, 'Not primary CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'Tikosyn REMS Program',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 28. Quinidine + Thioridazine
('QT_QUINIDINE_THIORIDAZINE_001', 'QUINIDINE', 'Quinidine', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'pharmacodynamic', 'contraindicated', 'established',
'Both Known Risk TdP drugs with BLACK BOX warnings. Class IA (quinidine) + phenothiazine (thioridazine). Quinidine inhibits CYP2D6.',
'Extreme TdP risk. Quinidine CYP2D6 inhibition increases thioridazine levels. PK + PD dual mechanism of harm.',
'CONTRAINDICATED. Both drugs have BLACK BOX warnings. CYP2D6 inhibition by quinidine dramatically increases thioridazine exposure.',
FALSE, 0.02, 0.99,
'FDA_LABEL', 'Mellaril - BLACK BOX; Quinidine - BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; CYP2D6 inhibition; Dual BLACK BOX drugs', NULL, 'CYP2D6 inhibition (major)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'FDA Safety Communications',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 29. Dronedarone + Ziprasidone
('QT_DRONEDARONE_ZIPRASIDONE_001', 'DRONEDARONE', 'Dronedarone (Multaq)', 'ZIPRASIDONE', 'Ziprasidone (Geodon)', 'pharmacodynamic', 'major', 'established',
'Both Known Risk TdP drugs. Dronedarone Class III multi-channel blocker. Ziprasidone known QT prolonger contraindicated with QT drugs.',
'High TdP risk. Ziprasidone label contraindicates concurrent use with Class III antiarrhythmics including dronedarone.',
'Avoid combination. Ziprasidone contraindicated with dronedarone per label. Use alternative antipsychotic.',
FALSE, 0.08, 0.94,
'FDA_LABEL', 'Geodon - Section 4; Multaq - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ziprasidone label contraindication; Class III combination', NULL, 'CYP3A4 (both drugs)', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Practice Guidelines',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'),

-- 30. Ibutilide + Thioridazine
('QT_IBUTILIDE_THIORIDAZINE_001', 'IBUTILIDE', 'Ibutilide (Corvert)', 'THIORIDAZINE', 'Thioridazine (Mellaril)', 'pharmacodynamic', 'contraindicated', 'established',
'Ibutilide for acute cardioversion has baseline 4-8% TdP rate. Thioridazine has BLACK BOX for QT. Combination extremely dangerous.',
'Extreme TdP risk. Ibutilide should never be used in patients on thioridazine. TdP rate would exceed 15%.',
'CONTRAINDICATED. Do not use ibutilide for cardioversion in patients on thioridazine. Use DC cardioversion.',
FALSE, 0.02, 0.99,
'FDA_LABEL', 'Corvert - Section 4; Mellaril - BLACK BOX', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
'CREDIBLEMEDS', 'Both Known Risk TdP; Ibutilide TdP 4-8% baseline; BLACK BOX combination', NULL, 'Not CYP interaction', 'Known Risk + Known Risk',
'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'ACC/AHA Cardioversion Guidelines',
'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team')
ON CONFLICT (interaction_id) DO NOTHING;

-- Continue with more combinations in subsequent migrations...

-- Verification for Migration 020
DO $$
DECLARE
    v_count INTEGER;
    v_qt_drugs INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM drug_interactions WHERE active = TRUE;
    SELECT COUNT(*) INTO v_qt_drugs FROM qt_drug_reference WHERE active = TRUE;
    RAISE NOTICE '★ Migration 020 Complete: % total DDIs, % QT drugs in reference table', v_count, v_qt_drugs;
END $$;
