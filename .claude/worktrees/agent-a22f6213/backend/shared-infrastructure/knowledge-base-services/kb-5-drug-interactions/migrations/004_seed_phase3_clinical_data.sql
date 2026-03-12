-- KB-5 Phase 3 Seed Data: Clinical Safety Rules
-- Evidence-based clinical safety data for Drug-Disease, Allergy, and Duplicate Therapy
-- Version: 2025Q4.phase3

-- ============================================================================
-- DRUG-DISEASE CONTRAINDICATIONS
-- Common high-risk drug-disease interactions
-- ============================================================================

INSERT INTO ddi_drug_disease_rules (
  dataset_version, drug_code, drug_name, disease_code, disease_name,
  code_system, contraindication_type, severity, clinical_rationale,
  management_strategy, alternative_drugs, evidence, confidence
) VALUES
-- NSAIDs in Renal Disease
('2025Q4', 'RXCUI:5640', 'Ibuprofen', 'N18', 'Chronic kidney disease',
 'ICD10', 'relative', 'major',
 'NSAIDs reduce renal prostaglandins, causing decreased GFR and potential acute kidney injury in CKD patients',
 'Avoid in eGFR <30. If essential, use lowest effective dose for shortest duration. Monitor creatinine.',
 '["acetaminophen", "topical NSAIDs", "tramadol"]'::JSONB, 'A', 0.95),

('2025Q4', 'RXCUI:5640', 'Ibuprofen', 'I50', 'Heart failure',
 'ICD10', 'relative', 'major',
 'NSAIDs cause sodium and water retention, worsening heart failure and increasing hospitalization risk',
 'Avoid if possible. Acetaminophen preferred for pain. If NSAID essential, monitor weight and symptoms.',
 '["acetaminophen", "topical analgesics"]'::JSONB, 'A', 0.92),

-- Metformin in Renal Impairment
('2025Q4', 'RXCUI:6809', 'Metformin', 'N18.4', 'CKD Stage 4',
 'ICD10', 'relative', 'major',
 'Risk of lactic acidosis increases with declining renal function. Metformin accumulates when eGFR <30.',
 'Contraindicated if eGFR <30. Reduce dose if eGFR 30-45. Monitor lactate if used.',
 '["sulfonylureas", "DPP-4 inhibitors", "insulin"]'::JSONB, 'A', 0.90),

('2025Q4', 'RXCUI:6809', 'Metformin', 'N18.5', 'CKD Stage 5',
 'ICD10', 'absolute', 'contraindicated',
 'High risk of metformin accumulation and fatal lactic acidosis in end-stage renal disease.',
 'Absolutely contraindicated. Use alternative antidiabetic agents.',
 '["insulin", "sulfonylureas with renal dosing"]'::JSONB, 'A', 0.98),

-- ACE Inhibitors in Pregnancy
('2025Q4', 'RXCUI:29046', 'Lisinopril', 'Z33', 'Pregnancy',
 'ICD10', 'absolute', 'contraindicated',
 'ACE inhibitors cause fetal renal dysplasia, oligohydramnios, and neonatal renal failure. Teratogenic in 2nd/3rd trimester.',
 'Discontinue immediately upon pregnancy confirmation. Switch to methyldopa, labetalol, or nifedipine.',
 '["methyldopa", "labetalol", "nifedipine"]'::JSONB, 'A', 0.99),

-- Beta-blockers in Asthma
('2025Q4', 'RXCUI:6918', 'Metoprolol', 'J45', 'Asthma',
 'ICD10', 'relative', 'major',
 'Non-selective beta-blockers can trigger bronchospasm in asthmatic patients. Cardioselective agents safer but not without risk.',
 'Prefer cardioselective beta-blockers (metoprolol, bisoprolol) at lowest dose. Monitor respiratory symptoms.',
 '["CCBs (diltiazem, verapamil)", "cardioselective beta-blockers"]'::JSONB, 'A', 0.85),

('2025Q4', 'RXCUI:8787', 'Propranolol', 'J45', 'Asthma',
 'ICD10', 'absolute', 'contraindicated',
 'Non-selective beta-blockers block bronchial beta-2 receptors, causing potentially fatal bronchospasm.',
 'Contraindicated. Use cardioselective beta-blocker or alternative antihypertensive/anti-anginal.',
 '["metoprolol", "bisoprolol", "CCBs"]'::JSONB, 'A', 0.95),

-- Warfarin in Active Bleeding
('2025Q4', 'RXCUI:11289', 'Warfarin', 'K92.2', 'GI hemorrhage',
 'ICD10', 'absolute', 'contraindicated',
 'Anticoagulation in active bleeding is life-threatening. Risk of exsanguination.',
 'Withhold anticoagulation until bleeding resolved. Reverse with vitamin K and/or PCC if severe.',
 NULL, 'A', 0.99),

-- Statins in Active Liver Disease
('2025Q4', 'RXCUI:36567', 'Atorvastatin', 'K70', 'Alcoholic liver disease',
 'ICD10', 'absolute', 'contraindicated',
 'Statins are hepatotoxic. Active liver disease increases risk of severe hepatotoxicity.',
 'Contraindicated in active liver disease. May consider after LFTs normalize.',
 '["bile acid sequestrants", "ezetimibe"]'::JSONB, 'A', 0.92),

-- Opioids in Respiratory Depression
('2025Q4', 'RXCUI:7052', 'Morphine', 'J96', 'Respiratory failure',
 'ICD10', 'absolute', 'contraindicated',
 'Opioids depress respiratory drive. Life-threatening in patients with respiratory failure.',
 'Avoid opioids. If pain essential, use lowest dose with continuous monitoring and naloxone available.',
 '["non-opioid analgesics", "regional anesthesia"]'::JSONB, 'A', 0.95),

-- Fluoroquinolones in Myasthenia Gravis
('2025Q4', 'RXCUI:2551', 'Ciprofloxacin', 'G70.0', 'Myasthenia gravis',
 'ICD10', 'relative', 'major',
 'Fluoroquinolones can exacerbate myasthenic weakness and precipitate respiratory crisis.',
 'Avoid if possible. If essential, monitor closely for worsening weakness. Have neostigmine available.',
 '["azithromycin", "cephalosporins", "penicillins"]'::JSONB, 'B', 0.88),

-- Digoxin in Heart Block
('2025Q4', 'RXCUI:3407', 'Digoxin', 'I44.2', 'Complete heart block',
 'ICD10', 'absolute', 'contraindicated',
 'Digoxin slows AV conduction and can cause fatal asystole in complete heart block.',
 'Contraindicated without pacemaker. Consider rate control alternatives.',
 '["amiodarone with pacemaker", "ablation"]'::JSONB, 'A', 0.97),

-- SSRIs in Bleeding Disorders
('2025Q4', 'RXCUI:36437', 'Sertraline', 'D68', 'Coagulation defects',
 'ICD10', 'relative', 'moderate',
 'SSRIs inhibit platelet serotonin uptake, increasing bleeding risk in coagulopathic patients.',
 'Use with caution. Consider PPIs for GI protection. Monitor for bleeding.',
 '["mirtazapine", "bupropion"]'::JSONB, 'B', 0.78)

ON CONFLICT DO NOTHING;

-- ============================================================================
-- ALLERGY CROSS-REACTIVITY RULES
-- Evidence-based cross-reactivity patterns
-- ============================================================================

INSERT INTO ddi_allergy_rules (
  dataset_version, allergen_code, allergen_name, allergen_type,
  cross_reactive_drug, cross_reactive_drug_name, cross_reactivity_rate,
  severity, reaction_type, clinical_notes, alternatives, evidence, confidence
) VALUES
-- Penicillin Cross-Reactivity
('2025Q4', 'RXCUI:7984', 'Penicillin', 'class',
 'RXCUI:2231', 'Cephalexin', 2.0,
 'moderate', 'urticaria',
 'Cross-reactivity between penicillins and 1st-gen cephalosporins is ~2%. Lower with 3rd/4th-gen.',
 '["azithromycin", "fluoroquinolones", "3rd-gen cephalosporins"]'::JSONB, 'A', 0.85),

('2025Q4', 'RXCUI:7984', 'Penicillin', 'class',
 'RXCUI:2193', 'Cefazolin', 1.5,
 'moderate', 'rash',
 '1st-generation cephalosporins have ~1-2% cross-reactivity with penicillins.',
 '["azithromycin", "vancomycin", "fluoroquinolones"]'::JSONB, 'A', 0.85),

('2025Q4', 'RXCUI:7984', 'Penicillin', 'class',
 'RXCUI:25033', 'Ceftriaxone', 0.5,
 'minor', 'rash',
 '3rd-generation cephalosporins have very low (<1%) cross-reactivity with penicillins.',
 '["azithromycin", "fluoroquinolones"]'::JSONB, 'A', 0.88),

('2025Q4', 'RXCUI:7984', 'Penicillin', 'class',
 'RXCUI:18631', 'Aztreonam', 0.1,
 'minor', 'rash',
 'Aztreonam has minimal cross-reactivity except with ceftazidime due to shared side chain.',
 NULL, 'B', 0.80),

('2025Q4', 'RXCUI:7984', 'Penicillin', 'class',
 'RXCUI:1807', 'Carbapenem', 1.0,
 'moderate', 'urticaria',
 'Carbapenems have ~1% cross-reactivity with penicillins. Often safe in penicillin allergy.',
 '["aztreonam", "fluoroquinolones"]'::JSONB, 'A', 0.82),

-- Sulfonamide Cross-Reactivity
('2025Q4', 'RXCUI:10831', 'Sulfamethoxazole', 'class',
 'RXCUI:4603', 'Furosemide', 0.5,
 'minor', 'rash',
 'Sulfonamide antibiotics may cross-react with sulfonamide non-antibiotics, though evidence is weak.',
 '["bumetanide", "torsemide"]'::JSONB, 'C', 0.65),

('2025Q4', 'RXCUI:10831', 'Sulfamethoxazole', 'class',
 'RXCUI:4316', 'Thiazides', 0.3,
 'minor', 'rash',
 'Thiazides contain sulfonamide moiety but cross-reactivity is very rare.',
 '["loop diuretics", "potassium-sparing diuretics"]'::JSONB, 'C', 0.60),

-- Aspirin/NSAID Cross-Reactivity
('2025Q4', 'RXCUI:1191', 'Aspirin', 'drug',
 'RXCUI:5640', 'Ibuprofen', 20.0,
 'major', 'bronchospasm',
 'NSAID-exacerbated respiratory disease (AERD): 20-25% of aspirin-sensitive patients react to NSAIDs.',
 '["acetaminophen", "celecoxib (with caution)"]'::JSONB, 'A', 0.90),

('2025Q4', 'RXCUI:1191', 'Aspirin', 'drug',
 'RXCUI:7646', 'Naproxen', 22.0,
 'major', 'bronchospasm',
 'Cross-reactivity in AERD is COX-1 mediated. All COX-1 inhibitors carry risk.',
 '["acetaminophen", "celecoxib"]'::JSONB, 'A', 0.90),

-- Local Anesthetic Cross-Reactivity
('2025Q4', 'RXCUI:6387', 'Lidocaine', 'drug',
 'RXCUI:7647', 'Bupivacaine', 1.0,
 'minor', 'rash',
 'Amide local anesthetics rarely cross-react. True allergy is extremely rare.',
 '["ester local anesthetics"]'::JSONB, 'B', 0.75),

('2025Q4', 'RXCUI:8745', 'Procaine', 'drug',
 'RXCUI:1049', 'Benzocaine', 15.0,
 'moderate', 'urticaria',
 'Ester local anesthetics share PABA metabolite and have significant cross-reactivity.',
 '["amide local anesthetics (lidocaine, bupivacaine)"]'::JSONB, 'B', 0.82),

-- Latex Cross-Reactivity (Fruit)
('2025Q4', 'LATEX', 'Latex', 'ingredient',
 'RXCUI:BANANA', 'Banana (food)', 50.0,
 'moderate', 'anaphylaxis',
 'Latex-fruit syndrome: 30-50% of latex-allergic patients react to banana, avocado, kiwi, chestnut.',
 NULL, 'B', 0.78),

-- Contrast Media Cross-Reactivity
('2025Q4', 'CONTRAST:IODINATED', 'Iodinated contrast', 'drug',
 'RXCUI:GADOLINIUM', 'Gadolinium contrast', 1.0,
 'minor', 'urticaria',
 'Gadolinium reactions are rare and usually not related to iodinated contrast allergy.',
 NULL, 'B', 0.70),

-- ACE Inhibitor Angioedema
('2025Q4', 'RXCUI:29046', 'Lisinopril', 'class',
 'RXCUI:35208', 'Ramipril', 95.0,
 'major', 'angioedema',
 'ACE inhibitor angioedema is class effect. All ACEIs contraindicated after angioedema event.',
 '["ARBs (use with caution - 3-5% cross-reaction)", "CCBs", "direct renin inhibitors"]'::JSONB, 'A', 0.95)

ON CONFLICT DO NOTHING;

-- ============================================================================
-- THERAPEUTIC CLASS MAPPINGS (ATC Classification)
-- For duplicate therapy detection
-- ============================================================================

INSERT INTO ddi_therapeutic_classes (
  dataset_version, drug_code, drug_name, atc_code, atc_level,
  therapeutic_class, pharmacological_group, chemical_subgroup
) VALUES
-- C03 DIURETICS
('2025Q4', 'RXCUI:4603', 'Furosemide', 'C03CA01', 5, 'Sulfonamides, plain', 'High-ceiling diuretics', 'Loop diuretics'),
('2025Q4', 'RXCUI:1808', 'Bumetanide', 'C03CA02', 5, 'Sulfonamides, plain', 'High-ceiling diuretics', 'Loop diuretics'),
('2025Q4', 'RXCUI:4316', 'Hydrochlorothiazide', 'C03AA03', 5, 'Thiazides, plain', 'Low-ceiling diuretics', 'Thiazides'),
('2025Q4', 'RXCUI:2409', 'Chlorthalidone', 'C03BA04', 5, 'Sulfonamides, plain', 'Low-ceiling diuretics', 'Thiazide-like'),
('2025Q4', 'RXCUI:9997', 'Spironolactone', 'C03DA01', 5, 'Aldosterone antagonists', 'Potassium-sparing agents', 'Aldosterone antagonists'),

-- C07 BETA BLOCKING AGENTS
('2025Q4', 'RXCUI:6918', 'Metoprolol', 'C07AB02', 5, 'Metoprolol', 'Selective beta-blockers', 'Beta-blockers'),
('2025Q4', 'RXCUI:19484', 'Carvedilol', 'C07AG02', 5, 'Carvedilol', 'Alpha/beta-blockers', 'Beta-blockers'),
('2025Q4', 'RXCUI:8787', 'Propranolol', 'C07AA05', 5, 'Propranolol', 'Non-selective beta-blockers', 'Beta-blockers'),
('2025Q4', 'RXCUI:1202', 'Atenolol', 'C07AB03', 5, 'Atenolol', 'Selective beta-blockers', 'Beta-blockers'),

-- M01A ANTIINFLAMMATORY/ANTIRHEUMATIC NSAIDS
('2025Q4', 'RXCUI:5640', 'Ibuprofen', 'M01AE01', 5, 'Propionic acid derivatives', 'NSAIDs', 'NSAIDs'),
('2025Q4', 'RXCUI:7646', 'Naproxen', 'M01AE02', 5, 'Propionic acid derivatives', 'NSAIDs', 'NSAIDs'),
('2025Q4', 'RXCUI:3355', 'Diclofenac', 'M01AB05', 5, 'Acetic acid derivatives', 'NSAIDs', 'NSAIDs'),
('2025Q4', 'RXCUI:54552', 'Celecoxib', 'M01AH01', 5, 'Coxibs', 'NSAIDs', 'COX-2 selective'),
('2025Q4', 'RXCUI:6129', 'Meloxicam', 'M01AC06', 5, 'Oxicams', 'NSAIDs', 'Preferential COX-2'),

-- N05 PSYCHOLEPTICS (Benzodiazepines)
('2025Q4', 'RXCUI:6470', 'Lorazepam', 'N05BA06', 5, 'Lorazepam', 'Anxiolytics', 'Benzodiazepines'),
('2025Q4', 'RXCUI:596', 'Alprazolam', 'N05BA12', 5, 'Alprazolam', 'Anxiolytics', 'Benzodiazepines'),
('2025Q4', 'RXCUI:2598', 'Clonazepam', 'N03AE01', 5, 'Clonazepam', 'Antiepileptics', 'Benzodiazepines'),
('2025Q4', 'RXCUI:3322', 'Diazepam', 'N05BA01', 5, 'Diazepam', 'Anxiolytics', 'Benzodiazepines'),

-- N06A ANTIDEPRESSANTS
('2025Q4', 'RXCUI:36437', 'Sertraline', 'N06AB06', 5, 'Sertraline', 'SSRIs', 'SSRIs'),
('2025Q4', 'RXCUI:4493', 'Fluoxetine', 'N06AB03', 5, 'Fluoxetine', 'SSRIs', 'SSRIs'),
('2025Q4', 'RXCUI:2597', 'Citalopram', 'N06AB04', 5, 'Citalopram', 'SSRIs', 'SSRIs'),
('2025Q4', 'RXCUI:42355', 'Escitalopram', 'N06AB10', 5, 'Escitalopram', 'SSRIs', 'SSRIs'),
('2025Q4', 'RXCUI:321988', 'Duloxetine', 'N06AX21', 5, 'Duloxetine', 'SNRIs', 'SNRIs'),
('2025Q4', 'RXCUI:39786', 'Venlafaxine', 'N06AX16', 5, 'Venlafaxine', 'SNRIs', 'SNRIs'),

-- C09A ACE INHIBITORS
('2025Q4', 'RXCUI:29046', 'Lisinopril', 'C09AA03', 5, 'Lisinopril', 'ACE inhibitors', 'ACE inhibitors'),
('2025Q4', 'RXCUI:35208', 'Ramipril', 'C09AA05', 5, 'Ramipril', 'ACE inhibitors', 'ACE inhibitors'),
('2025Q4', 'RXCUI:1998', 'Benazepril', 'C09AA07', 5, 'Benazepril', 'ACE inhibitors', 'ACE inhibitors'),

-- C09C ANGIOTENSIN II RECEPTOR BLOCKERS
('2025Q4', 'RXCUI:52175', 'Losartan', 'C09CA01', 5, 'Losartan', 'ARBs', 'Angiotensin II antagonists'),
('2025Q4', 'RXCUI:69749', 'Valsartan', 'C09CA03', 5, 'Valsartan', 'ARBs', 'Angiotensin II antagonists'),

-- A10 DRUGS USED IN DIABETES
('2025Q4', 'RXCUI:6809', 'Metformin', 'A10BA02', 5, 'Metformin', 'Biguanides', 'Oral antidiabetics'),
('2025Q4', 'RXCUI:4815', 'Glipizide', 'A10BB07', 5, 'Glipizide', 'Sulfonylureas', 'Oral antidiabetics'),
('2025Q4', 'RXCUI:4821', 'Glyburide', 'A10BB01', 5, 'Glyburide', 'Sulfonylureas', 'Oral antidiabetics'),

-- C10 LIPID MODIFYING AGENTS
('2025Q4', 'RXCUI:36567', 'Atorvastatin', 'C10AA05', 5, 'Atorvastatin', 'HMG CoA reductase inhibitors', 'Statins'),
('2025Q4', 'RXCUI:42463', 'Rosuvastatin', 'C10AA07', 5, 'Rosuvastatin', 'HMG CoA reductase inhibitors', 'Statins'),
('2025Q4', 'RXCUI:6472', 'Simvastatin', 'C10AA01', 5, 'Simvastatin', 'HMG CoA reductase inhibitors', 'Statins'),
('2025Q4', 'RXCUI:8629', 'Pravastatin', 'C10AA03', 5, 'Pravastatin', 'HMG CoA reductase inhibitors', 'Statins')

ON CONFLICT DO NOTHING;

-- ============================================================================
-- DUPLICATE THERAPY RULES
-- ============================================================================

INSERT INTO ddi_duplicate_therapy_rules (
  dataset_version, atc_code, atc_level, therapeutic_class,
  severity, allow_combination, max_drugs_allowed, clinical_rationale, exceptions, evidence
) VALUES
-- Loop Diuretics
('2025Q4', 'C03CA', 4, 'High-ceiling diuretics (Loop diuretics)',
 'moderate', FALSE, 1,
 'Multiple loop diuretics increase electrolyte imbalance, dehydration, and ototoxicity risk.',
 '{"exception": "diuretic resistance requiring combination"}', 'B'),

-- Thiazide Diuretics
('2025Q4', 'C03AA', 4, 'Thiazides, plain',
 'moderate', FALSE, 1,
 'Duplicate thiazides provide no additional benefit but increase hypokalemia and metabolic risks.',
 NULL, 'B'),

-- Beta-Blockers
('2025Q4', 'C07', 2, 'Beta blocking agents',
 'major', FALSE, 1,
 'Multiple beta-blockers cause severe bradycardia, heart block, and hypotension.',
 NULL, 'A'),

-- NSAIDs
('2025Q4', 'M01A', 3, 'Antiinflammatory and antirheumatic products, non-steroids',
 'major', FALSE, 1,
 'Multiple NSAIDs dramatically increase GI bleeding, renal injury, and cardiovascular risk without added efficacy.',
 '{"exception": "low-dose aspirin for cardioprotection with another NSAID for pain"}', 'A'),

-- Benzodiazepines
('2025Q4', 'N05BA', 4, 'Benzodiazepine derivatives (anxiolytics)',
 'major', FALSE, 1,
 'Multiple benzodiazepines increase sedation, respiratory depression, falls, and cognitive impairment.',
 NULL, 'A'),

-- SSRIs
('2025Q4', 'N06AB', 4, 'Selective serotonin reuptake inhibitors',
 'major', FALSE, 1,
 'Duplicate SSRIs increase serotonin syndrome risk and adverse effects without therapeutic benefit.',
 NULL, 'A'),

-- SNRIs
('2025Q4', 'N06AX', 4, 'Other antidepressants (SNRIs)',
 'major', FALSE, 1,
 'Multiple SNRIs increase serotonin/norepinephrine toxicity risk.',
 NULL, 'A'),

-- ACE Inhibitors
('2025Q4', 'C09AA', 4, 'ACE inhibitors, plain',
 'major', FALSE, 1,
 'Dual ACE inhibitor therapy increases hyperkalemia and renal impairment without benefit.',
 NULL, 'A'),

-- ARBs
('2025Q4', 'C09CA', 4, 'Angiotensin II receptor blockers, plain',
 'major', FALSE, 1,
 'Multiple ARBs have no added benefit and increase adverse effects.',
 NULL, 'A'),

-- ACE + ARB (cross-class)
('2025Q4', 'C09A', 3, 'ACE inhibitors',
 'major', FALSE, 1,
 'ONTARGET trial showed ACE+ARB combination increases renal events without cardiovascular benefit.',
 '{"exception": "proteinuric CKD under specialist supervision"}', 'A'),

-- Sulfonylureas
('2025Q4', 'A10BB', 4, 'Sulfonylureas',
 'major', FALSE, 1,
 'Multiple sulfonylureas increase hypoglycemia risk without improved glycemic control.',
 NULL, 'A'),

-- Statins
('2025Q4', 'C10AA', 4, 'HMG CoA reductase inhibitors',
 'major', FALSE, 1,
 'Dual statin therapy increases myopathy and rhabdomyolysis risk.',
 NULL, 'A'),

-- Proton Pump Inhibitors
('2025Q4', 'A02BC', 4, 'Proton pump inhibitors',
 'moderate', FALSE, 1,
 'Multiple PPIs have no benefit and increase C. difficile, fracture, and hypomagnesemia risks.',
 NULL, 'B'),

-- Opioids
('2025Q4', 'N02A', 3, 'Opioids',
 'major', TRUE, 2,
 'Multiple opioids require careful dose conversion. Increases respiratory depression risk.',
 '{"exception": "long-acting + breakthrough pain management", "max_combined_mme": 90}', 'A')

ON CONFLICT DO NOTHING;

-- Update statistics
ANALYZE ddi_drug_disease_rules;
ANALYZE ddi_allergy_rules;
ANALYZE ddi_therapeutic_classes;
ANALYZE ddi_duplicate_therapy_rules;
