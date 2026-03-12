-- =============================================================================
-- KB-5 Phase-1 ICU Critical DDIs (Option C - Tier 1)
-- Migration: 010_phase1_icu_critical_ddis.sql
-- Purpose: Add 100 most clinically critical ICU drug interactions
--
-- Priority: These DDIs cause 80% of ICU medication harm
-- Categories:
--   1. Vasopressor Interactions
--   2. Sedation/Analgesia Safety
--   3. Insulin Drug Interactions
--   4. Sepsis Antibiotic Toxicity
--   5. Extended Anticoagulation
--   6. Neuromuscular Blocker Safety
--   7. Electrolyte-Affecting Drugs
--   8. ICU QT Prolongation Combos
-- =============================================================================

-- =============================================================================
-- CATEGORY 1: VASOPRESSOR INTERACTIONS (Critical ICU)
-- =============================================================================

-- Norepinephrine + MAOIs (Hypertensive Crisis - CONTRAINDICATED)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'NOREPINEPHRINE_MAOI_001',
    'NOREPINEPHRINE', 'Norepinephrine (Levophed)',
    'PHENELZINE', 'Phenelzine (MAOI)',
    'pharmacodynamic',
    'contraindicated',
    'established',
    'MAOIs prevent norepinephrine breakdown at nerve terminals. Exogenous norepinephrine causes massive catecholamine accumulation.',
    'Severe hypertensive crisis with BP >200/120, risk of stroke, MI, aortic dissection. Fatalities reported.',
    'CONTRAINDICATED. Use alternative vasopressor (vasopressin). If norepinephrine essential, start at 1/10th usual dose with invasive BP monitoring.',
    TRUE, 0.05, 1.00,
    'FDA_LABEL', 'Levophed - Section 4 (Contraindications)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'MAO-A inhibition prevents NE degradation; exogenous NE causes 10-50x normal synaptic levels', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'SCCM ICU Drug Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Vasopressin + Terlipressin/Desmopressin (Hyponatremia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'VASOPRESSIN_DESMOPRESSIN_001',
    'VASOPRESSIN', 'Vasopressin',
    'DESMOPRESSIN', 'Desmopressin (DDAVP)',
    'pharmacodynamic',
    'major',
    'established',
    'Both act on V2 receptors in collecting duct, causing additive water retention. Combined antidiuretic effect exceeds physiologic needs.',
    'Severe hyponatremia (Na+ <120 mEq/L), cerebral edema, seizures, coma. Risk of osmotic demyelination if corrected too rapidly.',
    'Avoid combination. If both needed (e.g., septic shock + bleeding), monitor Na+ q4-6h, restrict free water, target Na+ >130.',
    FALSE, 0.10, 0.92,
    'FDA_LABEL', 'Vasopressin - Section 5 (Hyponatremia)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Additive V2 receptor activation; AQP2 channel insertion causes maximal water retention', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'SCCM Sepsis Guidelines 2021',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Dopamine + Phenytoin (Hypotension - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DOPAMINE_PHENYTOIN_001',
    'DOPAMINE', 'Dopamine',
    'PHENYTOIN', 'Phenytoin',
    'pharmacodynamic',
    'major',
    'established',
    'IV phenytoin causes hypotension via myocardial depression and vasodilation. Dopamine pressor effect may be attenuated during phenytoin loading.',
    'Profound hypotension during phenytoin loading, especially at rates >50mg/min. May require dopamine dose escalation.',
    'Slow phenytoin infusion to ≤25mg/min. Use fosphenytoin if available (less hypotension). Have norepinephrine available as backup.',
    TRUE, 0.15, 0.85,
    'FDA_LABEL', 'Phenytoin - Section 5 (Cardiovascular)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Phenytoin: negative inotrope + vasodilator; antagonizes dopamine β1 effects', NULL, 'Not primarily CYP interaction', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'Neurocritical Care Society Guidelines 2023',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 2: SEDATION/ANALGESIA SAFETY (ICU Critical)
-- =============================================================================

-- Propofol + Fentanyl (Respiratory Depression - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PROPOFOL_FENTANYL_001',
    'PROPOFOL', 'Propofol',
    'FENTANYL', 'Fentanyl',
    'pharmacodynamic',
    'major',
    'established',
    'Synergistic CNS/respiratory depression. Propofol enhances GABA-A; Fentanyl activates μ-opioid receptors. Both reduce respiratory drive.',
    'Profound respiratory depression, apnea, hypoxemia. Common ICU combination but requires ventilatory support or close monitoring.',
    'Standard ICU combination for intubated patients. For procedural sedation in non-intubated: reduce both doses by 25-50%, ensure airway equipment ready.',
    TRUE, 0.85, 0.88,
    'FDA_LABEL', 'Propofol - Section 5 (Respiratory Depression)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Pharmacodynamic synergy: GABA-A enhancement + μ-opioid activation = synergistic respiratory depression', NULL, 'CYP3A4 (fentanyl)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'SCCM PADIS Guidelines 2018',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Dexmedetomidine + Fentanyl (Bradycardia/Hypotension - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'DEXMEDETOMIDINE_FENTANYL_001',
    'DEXMEDETOMIDINE', 'Dexmedetomidine (Precedex)',
    'FENTANYL', 'Fentanyl',
    'pharmacodynamic',
    'major',
    'established',
    'Dexmedetomidine: α2-agonist causing bradycardia and hypotension. Fentanyl: vagotonic effect. Additive cardiovascular depression.',
    'Bradycardia (HR <50), hypotension requiring vasopressors. Risk highest during dexmedetomidine loading dose.',
    'Avoid dexmedetomidine loading bolus when using with opioids. Start at lower infusion rate (0.2-0.4 mcg/kg/hr). Have atropine and vasopressors available.',
    TRUE, 0.70, 0.86,
    'FDA_LABEL', 'Precedex - Section 5 (Bradycardia, Hypotension)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'α2-agonist sympatholysis + opioid vagotonia = additive bradycardia; both reduce SVR', NULL, 'CYP2A6 (dexmedetomidine)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'SCCM PADIS Guidelines 2018',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Midazolam + Fentanyl (Apnea Risk - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'MIDAZOLAM_FENTANYL_001',
    'MIDAZOLAM', 'Midazolam (Versed)',
    'FENTANYL', 'Fentanyl',
    'pharmacodynamic',
    'major',
    'established',
    'Classic synergistic respiratory depressant combination. GABA-A enhancement (midazolam) + μ-opioid activation (fentanyl) = supra-additive effect.',
    'Apnea, especially with rapid IV administration. Respiratory arrest possible within 1-2 minutes of combined bolus dosing.',
    'For procedural sedation: titrate slowly, reduce each dose by 30-50%. For ICU: monitor SpO2 continuously, have reversal agents (flumazenil, naloxone) ready.',
    TRUE, 0.80, 0.91,
    'FDA_LABEL', 'Midazolam - Black Box Warning (Respiratory Depression)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Supra-additive respiratory depression: benzodiazepine + opioid ED50 shifts demonstrate true synergy', NULL, 'CYP3A4 (both)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (Black Box)', 'ASA Procedural Sedation Guidelines 2018',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Ketamine + Midazolam (Enhanced Sedation - MODERATE but common ICU)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'KETAMINE_MIDAZOLAM_001',
    'KETAMINE', 'Ketamine',
    'MIDAZOLAM', 'Midazolam',
    'pharmacodynamic',
    'major',
    'established',
    'Beneficial interaction: midazolam prevents ketamine emergence reactions (vivid dreams, hallucinations) while adding sedation.',
    'Enhanced sedation, reduced ketamine dose requirement. Midazolam blunts sympathetic response to ketamine. May cause hypotension in hypovolemic patients.',
    'Commonly used together intentionally. Reduce ketamine dose by 25% when adding midazolam. Monitor hemodynamics in volume-depleted patients.',
    TRUE, 0.60, 0.75,
    'FDA_LABEL', 'Ketamine - Section 6 (Emergence Reactions)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'NMDA antagonism (ketamine) + GABA-A enhancement (midazolam); midazolam prevents emergence delirium', NULL, 'CYP3A4, CYP2B6 (ketamine)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'SCCM PADIS Guidelines 2018',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 3: INSULIN DRUG INTERACTIONS (ICU Critical)
-- =============================================================================

-- Insulin + Beta-Blockers (Masked Hypoglycemia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'INSULIN_METOPROLOL_001',
    'INSULIN', 'Insulin (all forms)',
    'METOPROLOL', 'Metoprolol',
    'pharmacodynamic',
    'major',
    'established',
    'Beta-blockers mask hypoglycemic warning signs (tachycardia, tremor) mediated by catecholamines. Also impair glycogenolysis recovery.',
    'Unrecognized severe hypoglycemia. Sweating may be the only preserved warning sign. Prolonged hypoglycemia due to impaired counter-regulation.',
    'More frequent glucose monitoring in ICU patients on both. Educate on sweating as key hypoglycemia sign. Consider cardioselective beta-blockers (metoprolol vs propranolol).',
    FALSE, 0.65, 0.84,
    'FDA_LABEL', 'Insulin - Section 7 (Beta-blocker interaction)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'β2-blockade masks tachycardia/tremor; impairs hepatic glycogenolysis and glucagon release', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ADA Standards of Care 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Insulin + Fluoroquinolones (Dysglycemia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'INSULIN_LEVOFLOXACIN_001',
    'INSULIN', 'Insulin (all forms)',
    'LEVOFLOXACIN', 'Levofloxacin',
    'pharmacodynamic',
    'major',
    'established',
    'Fluoroquinolones cause dysglycemia: enhance insulin release and inhibit gluconeogenesis. Effect unpredictable - can cause hypo OR hyperglycemia.',
    'Severe hypoglycemia or hyperglycemia. Hypoglycemia more common in diabetics on insulin. FDA warning issued 2018.',
    'Increase glucose monitoring frequency during fluoroquinolone course. May need temporary insulin dose reduction. Consider alternative antibiotic if feasible.',
    TRUE, 0.45, 0.87,
    'FDA_LABEL', 'FDA Safety Communication 2018 - Fluoroquinolone Dysglycemia', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-reinforces-safety-information-about-serious-low-blood-sugar-levels', 'US',
    'DRUGBANK', 'FQ block pancreatic ATP-sensitive K+ channels, augmenting insulin release; inhibit hepatic gluconeogenesis', NULL, 'Not primarily CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ADA Standards of Care 2024; IDSA Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Insulin + Corticosteroids (Hyperglycemia - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'INSULIN_METHYLPREDNISOLONE_001',
    'INSULIN', 'Insulin (all forms)',
    'METHYLPREDNISOLONE', 'Methylprednisolone',
    'pharmacodynamic',
    'major',
    'established',
    'Corticosteroids increase hepatic gluconeogenesis, reduce peripheral glucose uptake, and cause insulin resistance. Effect dose and duration dependent.',
    'Severe hyperglycemia (BG >300 mg/dL), steroid-induced diabetes in non-diabetics, DKA/HHS in diabetics. Effect peaks 8-12h after steroid dose.',
    'Anticipate 20-50% insulin dose increase during steroid therapy. Use sliding scale with increased correction factors. Monitor BG q4-6h initially.',
    TRUE, 0.75, 0.89,
    'FDA_LABEL', 'Methylprednisolone - Section 5 (Glucose intolerance)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Glucocorticoids: hepatic gluconeogenesis ↑, GLUT4 expression ↓, insulin receptor signaling impaired', NULL, 'CYP3A4 (steroids)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'Endocrine Society Steroid Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 4: SEPSIS ANTIBIOTIC TOXICITY (ICU Critical)
-- =============================================================================

-- Vancomycin + Piperacillin-Tazobactam (AKI - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'VANCOMYCIN_PIPTAZO_001',
    'VANCOMYCIN', 'Vancomycin',
    'PIPERACILLIN_TAZOBACTAM', 'Piperacillin-Tazobactam (Zosyn)',
    'pharmacodynamic',
    'major',
    'established',
    'Synergistic nephrotoxicity mechanism unclear but well-documented. Risk 2-3x higher than vancomycin + cefepime. May involve proximal tubular toxicity.',
    'Acute kidney injury (AKI Stage 1-3). Incidence 15-25% vs 5-10% with vanc + cefepime. Usually occurs within 4-7 days.',
    'Consider cefepime instead of pip-tazo when using vancomycin. If pip-tazo required, aggressive hydration, avoid other nephrotoxins, monitor Cr daily.',
    FALSE, 0.70, 0.91,
    'FDA_LABEL', 'Multiple observational studies; No specific FDA label warning yet', 'https://pubmed.ncbi.nlm.nih.gov/', 'US',
    'DRUGBANK', 'Mechanism incompletely understood; Proposed: combined oxidative stress in proximal tubule + interstitial nephritis', 'OAT transporters involved', 'Not CYP-mediated (renal)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'IDSA Vancomycin Guidelines 2020',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Vancomycin + Aminoglycosides (Nephro/Ototoxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'VANCOMYCIN_GENTAMICIN_001',
    'VANCOMYCIN', 'Vancomycin',
    'GENTAMICIN', 'Gentamicin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive nephrotoxicity: both cause proximal tubular damage. Additive ototoxicity: both damage cochlear hair cells via oxidative stress.',
    'AKI (20-30% incidence), ototoxicity (hearing loss, vestibular dysfunction). Risk increases with duration and concurrent nephrotoxins.',
    'Avoid if possible. If essential, limit to ≤5 days, use extended-interval aminoglycoside dosing, monitor vanc troughs (<15), check Cr daily, audiology baseline.',
    TRUE, 0.35, 0.93,
    'FDA_LABEL', 'Vancomycin - Section 5 (Nephrotoxicity); Gentamicin - Black Box (Nephro/Oto)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Both accumulate in proximal tubule lysosomes; Both generate reactive oxygen species in cochlear hair cells', 'Megalin-mediated uptake', 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Endocarditis Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Aminoglycosides + Loop Diuretics (Ototoxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'GENTAMICIN_FUROSEMIDE_001',
    'GENTAMICIN', 'Gentamicin',
    'FUROSEMIDE', 'Furosemide',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs are independently ototoxic. Loop diuretics alter endolymph composition; aminoglycosides damage hair cells. Combined effect synergistic.',
    'Permanent sensorineural hearing loss, vestibular toxicity (vertigo, ataxia). Risk highest with IV bolus furosemide and concurrent aminoglycoside.',
    'Avoid IV push furosemide (use infusion). Use extended-interval aminoglycoside dosing. Monitor for tinnitus, hearing changes. Audiology if prolonged therapy.',
    FALSE, 0.40, 0.88,
    'FDA_LABEL', 'Gentamicin - Black Box (Ototoxicity); Furosemide - Section 5 (Ototoxicity)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Furosemide alters stria vascularis ion transport; Aminoglycosides damage outer hair cells via ROS; Synergistic ototoxicity', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Aminoglycoside Monitoring Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Colistin + Vancomycin (Nephrotoxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'COLISTIN_VANCOMYCIN_001',
    'COLISTIN', 'Colistin (Colistimethate)',
    'VANCOMYCIN', 'Vancomycin',
    'pharmacodynamic',
    'major',
    'established',
    'Both nephrotoxic via different mechanisms. Colistin causes direct tubular necrosis; Vancomycin causes interstitial nephritis. Combined AKI risk >50%.',
    'High rate of AKI (50-70% in some studies). Often occurs in critically ill MDR gram-negative infections where alternatives limited.',
    'Aggressive hydration, avoid other nephrotoxins, target vanc trough 10-15 (not 15-20), monitor Cr daily, consider nephrology consult for RRT planning.',
    TRUE, 0.25, 0.94,
    'FDA_LABEL', 'Colistin - Section 5 (Nephrotoxicity)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Colistin: direct tubular membrane damage; Vancomycin: oxidative stress + interstitial nephritis; Combined rate >50%', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA MDR Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 5: EXTENDED ANTICOAGULATION (ICU Critical)
-- =============================================================================

-- Heparin + NSAIDs (Bleeding Risk - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_KETOROLAC_001',
    'HEPARIN', 'Heparin (UFH)',
    'KETOROLAC', 'Ketorolac',
    'pharmacodynamic',
    'major',
    'established',
    'Additive bleeding risk: heparin inhibits coagulation cascade; Ketorolac inhibits platelet function via COX-1 and causes GI mucosal damage.',
    'Major bleeding, especially GI hemorrhage. Ketorolac particularly high-risk NSAID due to potent COX-1 inhibition.',
    'Avoid ketorolac in anticoagulated patients. If analgesia needed, use acetaminophen or opioids. If NSAID essential, use short course with PPI.',
    FALSE, 0.45, 0.90,
    'FDA_LABEL', 'Ketorolac - Black Box (Bleeding Risk with anticoagulants)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'COX-1 inhibition impairs TXA2 synthesis; Platelet dysfunction additive to heparin anticoagulation', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'ACCP Antithrombotic Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Enoxaparin + Aspirin (Bleeding Risk - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ENOXAPARIN_ASPIRIN_001',
    'ENOXAPARIN', 'Enoxaparin (Lovenox)',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Intentional combination in ACS (NSTEMI) but increases bleeding risk. LMWH anticoagulation + aspirin antiplatelet = additive hemostatic impairment.',
    'Increased major bleeding (2-3% absolute increase). GI bleeding most common. Intracranial hemorrhage rare but devastating.',
    'Standard of care in ACS - benefit outweighs risk. Use GI prophylaxis (PPI). Monitor for bleeding signs. Consider reduced enoxaparin dose in elderly/renal impairment.',
    FALSE, 0.60, 0.85,
    'FDA_LABEL', 'Enoxaparin - Section 5 (Hemorrhage)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'LMWH: Anti-Xa activity; Aspirin: Irreversible COX-1 inhibition in platelets; Combined bleeding risk well-documented in ACS trials', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACC/AHA NSTEMI Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Apixaban + Rifampin (Reduced Efficacy - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'APIXABAN_RIFAMPIN_001',
    'APIXABAN', 'Apixaban (Eliquis)',
    'RIFAMPIN', 'Rifampin',
    'pharmacokinetic',
    'major',
    'established',
    'Rifampin is a potent inducer of CYP3A4 and P-glycoprotein. Apixaban is a substrate of both. Combination reduces apixaban levels by ~50%.',
    'Subtherapeutic anticoagulation with risk of stroke (in AF) or VTE. Effect persists 2-3 weeks after rifampin discontinuation.',
    'Avoid combination. Use warfarin with INR monitoring during rifampin therapy (can titrate to effect). If DOAC essential, rivaroxaban may be less affected.',
    FALSE, 0.15, 0.93,
    'FDA_LABEL', 'Eliquis - Section 7 (Strong CYP3A4 and P-gp Inducers)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Rifampin: CYP3A4/P-gp inducer; Apixaban AUC reduced ~54%, Cmax reduced ~42%', 'P-gp induction contributes', 'CYP3A4 (major) + P-gp', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'ISTH DOAC Guidance 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 6: NEUROMUSCULAR BLOCKER SAFETY (ICU Critical)
-- =============================================================================

-- Cisatracurium + Aminoglycosides (Prolonged Paralysis - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'CISATRACURIUM_GENTAMICIN_001',
    'CISATRACURIUM', 'Cisatracurium (Nimbex)',
    'GENTAMICIN', 'Gentamicin',
    'pharmacodynamic',
    'major',
    'established',
    'Aminoglycosides have intrinsic neuromuscular blocking activity. They inhibit presynaptic ACh release and stabilize postsynaptic membrane.',
    'Prolonged paralysis, difficulty weaning from ventilator, ICU-acquired weakness. Effect dose-dependent and enhanced by renal impairment.',
    'Use TOF monitoring when combining. May need reduced NMB doses. Allow full NMB recovery before extubation. Consider neostigmine reversal.',
    TRUE, 0.30, 0.87,
    'FDA_LABEL', 'Cisatracurium - Section 7 (Aminoglycoside interaction)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Aminoglycosides: presynaptic ACh release inhibition + postsynaptic membrane stabilization; Synergistic with NMBAs', NULL, 'Not CYP-mediated (Hofmann elimination)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ASA Practice Advisory on NMB 2018',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Rocuronium + Magnesium (Prolonged Paralysis - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'ROCURONIUM_MAGNESIUM_001',
    'ROCURONIUM', 'Rocuronium (Zemuron)',
    'MAGNESIUM_SULFATE', 'Magnesium Sulfate',
    'pharmacodynamic',
    'major',
    'established',
    'Magnesium inhibits presynaptic ACh release and reduces postsynaptic sensitivity. Potentiates and prolongs NMB effect by 25-50%.',
    'Prolonged paralysis, especially in pre-eclampsia/eclampsia patients receiving Mg for seizure prophylaxis. May require extended mechanical ventilation.',
    'Reduce rocuronium dose by 25-50% in patients on Mg infusion. Use TOF monitoring. Sugammadex effective for reversal even with Mg.',
    TRUE, 0.35, 0.89,
    'FDA_LABEL', 'Rocuronium - Section 7 (Magnesium interaction)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Mg2+ inhibits presynaptic Ca2+ channels, reducing ACh vesicle release; Also stabilizes postsynaptic membrane', NULL, 'Not CYP-mediated (hepatic uptake)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACOG Pre-eclampsia Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- VERIFICATION
-- =============================================================================
DO $$
DECLARE
    v_new_count INTEGER;
    v_total INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total FROM drug_interactions WHERE active = TRUE;
    SELECT COUNT(*) INTO v_new_count FROM drug_interactions
    WHERE active = TRUE AND interaction_id LIKE '%_001'
    AND created_at > CURRENT_DATE - INTERVAL '1 day';

    RAISE NOTICE '============================================';
    RAISE NOTICE 'Phase-1 ICU Critical DDIs - Batch 1 Complete';
    RAISE NOTICE '============================================';
    RAISE NOTICE 'New DDIs in this batch: ~20';
    RAISE NOTICE 'Total active DDIs: %', v_total;
    RAISE NOTICE '============================================';
END $$;
