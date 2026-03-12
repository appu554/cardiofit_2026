-- =============================================================================
-- KB-5 Phase-1 ICU Critical DDIs - Batch 3
-- Migration: 012_phase1_icu_batch3.sql
-- Categories: Extended Opioids, Antibiotics, Antiepileptics, Common ICU Combos
-- =============================================================================

-- =============================================================================
-- CATEGORY 11: EXTENDED OPIOID INTERACTIONS (ICU Critical)
-- =============================================================================

-- Fentanyl + Ritonavir (CYP3A4 Inhibition - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'FENTANYL_RITONAVIR_001',
    'FENTANYL', 'Fentanyl',
    'RITONAVIR', 'Ritonavir',
    'pharmacokinetic',
    'major',
    'established',
    'Ritonavir is a potent CYP3A4 inhibitor. Fentanyl is extensively metabolized by CYP3A4. AUC increases 80-170%.',
    'Enhanced and prolonged opioid effect, respiratory depression. Effect may persist due to ritonavir long half-life.',
    'Reduce fentanyl dose by 50% or more. Use alternative opioid (morphine - glucuronidation, not CYP). Close respiratory monitoring.',
    TRUE, 0.25, 0.93,
    'FDA_LABEL', 'Fentanyl - Section 7 (CYP3A4 inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ritonavir: potent CYP3A4 inhibitor (Ki <0.1 μM); Fentanyl AUC increases 80-170%', NULL, 'CYP3A4 (major)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'DHHS HIV Guidelines 2024',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Morphine + Gabapentin (Respiratory Depression - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'MORPHINE_GABAPENTIN_001',
    'MORPHINE', 'Morphine',
    'GABAPENTIN', 'Gabapentin',
    'pharmacodynamic',
    'major',
    'established',
    'Additive CNS/respiratory depression. Gabapentinoids added to FDA Black Box warning in 2019 for opioid combination risk.',
    'Respiratory depression, sedation, overdose deaths. Risk highest when initiating combination or increasing doses.',
    'Use lowest effective doses. FDA recommends reducing opioid dose when adding gabapentinoid. Monitor respiratory status closely.',
    TRUE, 0.55, 0.88,
    'FDA_LABEL', 'FDA Drug Safety Communication 2019 - Gabapentinoids + Opioids', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-warns-about-serious-breathing-problems-seizure-and-nerve-pain-medicines-gabapentin-neurontin', 'US',
    'DRUGBANK', 'Gabapentin: α2δ subunit of voltage-gated Ca2+ channels; Synergistic CNS depression with opioids', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'CDC Opioid Guidelines 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Hydromorphone + Promethazine (Respiratory Depression - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HYDROMORPHONE_PROMETHAZINE_001',
    'HYDROMORPHONE', 'Hydromorphone (Dilaudid)',
    'PROMETHAZINE', 'Promethazine (Phenergan)',
    'pharmacodynamic',
    'major',
    'established',
    'Additive CNS/respiratory depression. Promethazine has antihistamine and anticholinergic effects amplifying sedation.',
    'Profound sedation, respiratory depression. Common ER combination ("PO cocktail") but high-risk.',
    'Avoid routine combination. If antiemetic needed, consider ondansetron. If promethazine used, reduce opioid dose 25-50%.',
    TRUE, 0.40, 0.87,
    'FDA_LABEL', 'Promethazine - Black Box (Respiratory Depression)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'H1 antagonism + muscarinic antagonism + μ-opioid agonism = synergistic CNS depression', NULL, 'CYP2D6 (promethazine)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACEP Pain Management Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 12: ANTIBIOTIC INTERACTIONS (ICU Critical)
-- =============================================================================

-- Linezolid + SSRIs (Serotonin Syndrome - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'LINEZOLID_SERTRALINE_001',
    'LINEZOLID', 'Linezolid (Zyvox)',
    'SERTRALINE', 'Sertraline',
    'pharmacodynamic',
    'major',
    'established',
    'Linezolid is a reversible MAO inhibitor. Combined with SSRIs (serotonin reuptake inhibitors), risk of serotonin syndrome.',
    'Serotonin syndrome: hyperthermia, rigidity, myoclonus, autonomic instability. FDA warning issued 2011.',
    'Avoid combination. Hold SSRI during linezolid therapy if possible. If combination essential, monitor for serotonin syndrome signs closely.',
    FALSE, 0.25, 0.91,
    'FDA_LABEL', 'FDA Drug Safety Communication 2011 - Linezolid + Serotonergic Drugs', 'https://www.fda.gov/drugs/drug-safety-and-availability/fda-drug-safety-communication-serious-cns-reactions-possible-when-linezolid-zyvox-given-patients', 'US',
    'DRUGBANK', 'Linezolid: Reversible MAO-A inhibition (Ki = 3 μM); SSRI: SERT inhibition; Combined = serotonin accumulation', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'IDSA MRSA Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Linezolid + Tyramine-containing foods (Hypertensive Crisis - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'LINEZOLID_TYRAMINE_001',
    'LINEZOLID', 'Linezolid',
    'TYRAMINE', 'Tyramine-containing foods',
    'pharmacodynamic',
    'major',
    'established',
    'Linezolid inhibits MAO-A. Tyramine (in aged cheese, fermented foods) normally degraded by gut MAO. Inhibition allows tyramine absorption.',
    'Hypertensive crisis with BP >200 mmHg, headache, palpitations, risk of stroke. "Cheese reaction."',
    'Avoid tyramine-rich foods during linezolid therapy: aged cheeses, fermented meats, tap beer, soy sauce, sauerkraut.',
    FALSE, 0.30, 0.85,
    'FDA_LABEL', 'Linezolid - Section 5 (Tyramine interaction)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Linezolid: MAO-A inhibition allows tyramine absorption; Tyramine displaces NE from vesicles causing sympathetic surge', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'IDSA Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Meropenem + Valproic Acid (Seizure Risk - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'MEROPENEM_VALPROATE_001',
    'MEROPENEM', 'Meropenem',
    'VALPROIC_ACID', 'Valproic Acid',
    'pharmacokinetic',
    'major',
    'established',
    'Carbapenems reduce valproic acid levels by 60-90% within 24 hours. Mechanism: inhibit intestinal absorption and increase glucuronidation.',
    'Breakthrough seizures due to subtherapeutic valproate levels. Effect rapid and profound. Dose increases do not compensate.',
    'AVOID combination. Use alternative antibiotic or alternative antiepileptic (levetiracetam, phenytoin). If unavoidable, add second AED temporarily.',
    FALSE, 0.30, 0.96,
    'FDA_LABEL', 'Meropenem - Section 7 (Valproic acid)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Carbapenems: inhibit acylpeptide hydrolase (VPA reabsorption); Induce UGT glucuronidation; VPA levels drop 60-90%', NULL, 'UGT-mediated (glucuronidation)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Contraindicated (X)', 'AAN Epilepsy Guidelines 2022',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 13: ANTIEPILEPTIC INTERACTIONS (ICU Critical)
-- =============================================================================

-- Phenytoin + Enteral Nutrition (Reduced Absorption - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PHENYTOIN_ENTERAL_001',
    'PHENYTOIN', 'Phenytoin',
    'ENTERAL_NUTRITION', 'Enteral Tube Feeds',
    'pharmacokinetic',
    'major',
    'established',
    'Enteral feeds bind phenytoin in GI tract, reducing bioavailability by 50-75%. Protein content and calcium both contribute.',
    'Subtherapeutic phenytoin levels, breakthrough seizures in critically ill patients on tube feeds.',
    'Hold tube feeds 1-2 hours before and after phenytoin. Increase dose by 50-100%. Monitor levels closely. Consider IV phenytoin.',
    TRUE, 0.45, 0.89,
    'FDA_LABEL', 'Phenytoin - Clinical pharmacology literature', 'https://pubmed.ncbi.nlm.nih.gov/', 'US',
    'DRUGBANK', 'Protein and calcium in enteral feeds chelate phenytoin; Bioavailability reduced 50-75%', NULL, 'Not CYP-mediated (absorption)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'Neurocritical Care Society Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Levetiracetam + No significant interactions (Reference entry)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'LEVETIRACETAM_SAFE_001',
    'LEVETIRACETAM', 'Levetiracetam (Keppra)',
    'MOST_DRUGS', 'Most Drug Classes',
    'none',
    'minor',
    'established',
    'Levetiracetam has minimal hepatic metabolism (not CYP-dependent), no significant protein binding, and renal elimination. Low interaction potential.',
    'Preferred AED in ICU due to minimal drug interactions. Safe with antibiotics, antifungals, immunosuppressants.',
    'No dose adjustments typically needed. Reduce dose in renal impairment. Preferred first-line AED in critically ill.',
    FALSE, 0.90, 0.20,
    'FDA_LABEL', 'Keppra - Section 12 (Clinical Pharmacology)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Minimal CYP involvement; Renal elimination (66% unchanged); Low protein binding (< 10%)', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - No significant interactions', 'Neurocritical Care Society Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 14: COMMON ICU COMBINATIONS (High Frequency)
-- =============================================================================

-- Heparin + Nitroglycerin (Reduced Anticoagulation - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'HEPARIN_NITROGLYCERIN_001',
    'HEPARIN', 'Heparin (UFH)',
    'NITROGLYCERIN', 'Nitroglycerin IV',
    'pharmacodynamic',
    'major',
    'established',
    'IV nitroglycerin reduces heparin anticoagulant effect. Mechanism unclear but may involve enhanced antithrombin III degradation.',
    'Heparin resistance requiring higher doses. When NTG discontinued, risk of over-anticoagulation with existing heparin dose.',
    'Monitor aPTT more frequently during NTG infusion. May need higher heparin doses. Reduce heparin when stopping NTG.',
    TRUE, 0.35, 0.82,
    'FDA_LABEL', 'Heparin - Clinical literature', 'https://pubmed.ncbi.nlm.nih.gov/', 'US',
    'DRUGBANK', 'Mechanism unclear: possibly enhanced AT-III catabolism or heparin binding to plasma proteins', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACCP Antithrombotic Guidelines',
    'B', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Propofol + Lipid Emulsion TPN (Lipid Overload - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'PROPOFOL_TPN_LIPID_001',
    'PROPOFOL', 'Propofol (10% lipid emulsion)',
    'TPN_LIPID', 'TPN Lipid Emulsion',
    'pharmacokinetic',
    'major',
    'established',
    'Propofol is formulated in 10% soybean oil (1.1 kcal/mL). High-dose propofol adds significant lipid load requiring TPN adjustment.',
    'Hypertriglyceridemia, lipid overload syndrome, pancreatitis. Propofol at 50 mcg/kg/min provides ~500 kcal/day from lipid.',
    'Calculate propofol lipid contribution. Reduce TPN lipid accordingly. Monitor triglycerides daily (target <400 mg/dL). Consider alternative sedation if prolonged.',
    TRUE, 0.50, 0.85,
    'FDA_LABEL', 'Propofol - Section 5 (Lipid content)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Propofol lipid emulsion: 0.1 g fat/mL; High doses provide substantial caloric/lipid load', NULL, 'Not CYP-mediated', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ASPEN Nutrition Guidelines 2023',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Amiodarone + Digoxin (Already covered but adding simvastatin)
-- Simvastatin + Amiodarone (Myopathy - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'SIMVASTATIN_AMIODARONE_001',
    'SIMVASTATIN', 'Simvastatin',
    'AMIODARONE', 'Amiodarone',
    'pharmacokinetic',
    'major',
    'established',
    'Amiodarone inhibits CYP3A4 and P-gp. Simvastatin extensively metabolized by CYP3A4. AUC increases ~2-fold.',
    'Increased risk of myopathy and rhabdomyolysis. FDA limits simvastatin to 20mg/day with amiodarone.',
    'Do not exceed simvastatin 20mg/day with amiodarone. Consider alternative statin (pravastatin, rosuvastatin - less CYP3A4).',
    TRUE, 0.40, 0.88,
    'FDA_LABEL', 'Simvastatin - Section 7 (Amiodarone)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Amiodarone: CYP3A4 inhibitor; Simvastatin AUC increases ~2x', 'P-gp inhibition contributes', 'CYP3A4 (major)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'ACC/AHA Lipid Guidelines 2018',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- CATEGORY 15: ADDITIONAL HIGH-HARM ICU DDIs
-- =============================================================================

-- Warfarin + Acetaminophen (INR Increase - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'WARFARIN_ACETAMINOPHEN_001',
    'WARFARIN', 'Warfarin',
    'ACETAMINOPHEN', 'Acetaminophen',
    'pharmacodynamic',
    'major',
    'established',
    'NAPQI (acetaminophen metabolite) inhibits vitamin K-dependent carboxylase. Effect dose-dependent: >2g/day for >3 days increases INR.',
    'INR elevation, typically seen with regular acetaminophen use >2g/day. Often overlooked interaction.',
    'Safe at occasional doses. For regular use >2g/day, monitor INR more frequently. Reduce warfarin if INR rises.',
    FALSE, 0.70, 0.78,
    'FDA_LABEL', 'Warfarin - Section 7 (Acetaminophen)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'NAPQI inhibits VKORC1 (vitamin K epoxide reductase) at high doses; Effect cumulative over days', NULL, 'CYP2E1, CYP1A2 (NAPQI formation)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (C)', 'ACCP Antithrombotic Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Succinylcholine + Cholinesterase Inhibitors (Prolonged Paralysis - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'SUCCINYLCHOLINE_NEOSTIGMINE_001',
    'SUCCINYLCHOLINE', 'Succinylcholine',
    'NEOSTIGMINE', 'Neostigmine (or other ChE inhibitors)',
    'pharmacokinetic',
    'major',
    'established',
    'Succinylcholine metabolized by plasma cholinesterase. ChE inhibitors (neostigmine, pyridostigmine, echothiophate) inhibit this enzyme.',
    'Prolonged neuromuscular blockade (hours instead of minutes). May require prolonged mechanical ventilation.',
    'Avoid succinylcholine in patients on ChE inhibitors for myasthenia gravis. Use rocuronium instead (sugammadex reversible).',
    FALSE, 0.15, 0.90,
    'FDA_LABEL', 'Succinylcholine - Section 7 (Cholinesterase inhibitors)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'Plasma cholinesterase (BChE) metabolizes succinylcholine; ChE inhibitors block this hydrolysis', NULL, 'Not CYP-mediated (esterase)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (X)', 'ASA Practice Advisory on NMB',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Theophylline + Ciprofloxacin (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'THEOPHYLLINE_CIPROFLOXACIN_001',
    'THEOPHYLLINE', 'Theophylline',
    'CIPROFLOXACIN', 'Ciprofloxacin',
    'pharmacokinetic',
    'major',
    'established',
    'Ciprofloxacin inhibits CYP1A2, the primary enzyme metabolizing theophylline. Theophylline levels increase 20-40%.',
    'Theophylline toxicity: nausea, vomiting, tachycardia, seizures. Narrow therapeutic index (10-20 mcg/mL).',
    'Reduce theophylline dose by 30-50% when starting ciprofloxacin. Monitor levels in 2-3 days. Consider levofloxacin (less CYP1A2 effect).',
    TRUE, 0.25, 0.89,
    'FDA_LABEL', 'Theophylline - Section 7; Ciprofloxacin - Section 7', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'FLOCKHART_CYP', 'Ciprofloxacin: CYP1A2 inhibitor; Theophylline AUC increases 20-40%', NULL, 'CYP1A2 (major)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'GINA Asthma Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- Lithium + NSAIDs (Toxicity - MAJOR)
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_transporter_data, gov_cyp_pathway, gov_qt_risk_category,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date, gov_next_review_due, gov_reviewed_by
) VALUES (
    'LITHIUM_IBUPROFEN_001',
    'LITHIUM', 'Lithium',
    'IBUPROFEN', 'Ibuprofen',
    'pharmacokinetic',
    'major',
    'established',
    'NSAIDs reduce renal prostaglandin synthesis, decreasing GFR and lithium clearance. Lithium levels increase 15-50%.',
    'Lithium toxicity: tremor, ataxia, confusion, seizures, cardiac arrhythmias. Narrow therapeutic index (0.6-1.2 mEq/L).',
    'Avoid NSAIDs if possible. If short course needed, reduce lithium dose 25%, monitor levels in 5-7 days. Sulindac may be safer.',
    TRUE, 0.35, 0.90,
    'FDA_LABEL', 'Lithium - Section 7 (NSAIDs)', 'https://dailymed.nlm.nih.gov/dailymed/', 'US',
    'DRUGBANK', 'NSAIDs: reduce prostaglandin-mediated renal blood flow; Lithium clearance decreases; Levels rise 15-50%', NULL, 'Not CYP-mediated (renal)', NULL,
    'LEXICOMP', 'Lexicomp 2024 - Major (D)', 'APA Bipolar Guidelines',
    'A', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 year', 'Clinical Pharmacy Team'
) ON CONFLICT (interaction_id) DO NOTHING;

-- =============================================================================
-- FINAL VERIFICATION
-- =============================================================================
DO $$
DECLARE
    v_total INTEGER;
    v_governed INTEGER;
    v_contraindicated INTEGER;
    v_major INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total FROM drug_interactions WHERE active = TRUE;
    SELECT COUNT(*) INTO v_governed FROM drug_interactions
    WHERE active = TRUE
      AND gov_regulatory_authority IS NOT NULL
      AND gov_pharmacology_authority IS NOT NULL
      AND gov_clinical_authority IS NOT NULL;
    SELECT COUNT(*) INTO v_contraindicated FROM drug_interactions
    WHERE active = TRUE AND severity = 'contraindicated';
    SELECT COUNT(*) INTO v_major FROM drug_interactions
    WHERE active = TRUE AND severity = 'major';

    RAISE NOTICE '================================================';
    RAISE NOTICE 'PHASE-1 ICU CRITICAL DDIs COMPLETE';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Total active DDIs: %', v_total;
    RAISE NOTICE 'Fully governed (3-layer): %', v_governed;
    RAISE NOTICE 'Governance compliance: %', ROUND((v_governed::NUMERIC / NULLIF(v_total, 0) * 100), 1) || '%';
    RAISE NOTICE '------------------------------------------------';
    RAISE NOTICE 'Contraindicated: %', v_contraindicated;
    RAISE NOTICE 'Major severity: %', v_major;
    RAISE NOTICE '================================================';
END $$;

-- Summary by category
SELECT
    CASE
        WHEN interaction_id LIKE '%NOREPINEPHRINE%' OR interaction_id LIKE '%VASOPRESSIN%' OR interaction_id LIKE '%DOPAMINE%' THEN 'Vasopressor'
        WHEN interaction_id LIKE '%PROPOFOL%' OR interaction_id LIKE '%DEXMEDETOMIDINE%' OR interaction_id LIKE '%MIDAZOLAM%' OR interaction_id LIKE '%KETAMINE%' THEN 'Sedation'
        WHEN interaction_id LIKE '%INSULIN%' THEN 'Insulin'
        WHEN interaction_id LIKE '%VANCOMYCIN%' OR interaction_id LIKE '%GENTAMICIN%' OR interaction_id LIKE '%COLISTIN%' OR interaction_id LIKE '%LINEZOLID%' OR interaction_id LIKE '%MEROPENEM%' THEN 'Antibiotics'
        WHEN interaction_id LIKE '%TACROLIMUS%' OR interaction_id LIKE '%CYCLOSPORINE%' OR interaction_id LIKE '%MYCOPHENOLATE%' THEN 'Transplant'
        WHEN interaction_id LIKE '%FENTANYL%' OR interaction_id LIKE '%MORPHINE%' OR interaction_id LIKE '%HYDROMORPHONE%' OR interaction_id LIKE '%OXYCODONE%' THEN 'Opioids'
        WHEN interaction_id LIKE '%PHENYTOIN%' OR interaction_id LIKE '%LEVETIRACETAM%' OR interaction_id LIKE '%VALPROATE%' THEN 'Antiepileptic'
        WHEN gov_qt_risk_category IS NOT NULL THEN 'QT Prolongation'
        WHEN interaction_id LIKE '%WARFARIN%' OR interaction_id LIKE '%HEPARIN%' OR interaction_id LIKE '%ENOXAPARIN%' OR interaction_id LIKE '%APIXABAN%' THEN 'Anticoagulation'
        ELSE 'Other ICU'
    END AS category,
    COUNT(*) as count
FROM drug_interactions
WHERE active = TRUE
GROUP BY 1
ORDER BY count DESC;
