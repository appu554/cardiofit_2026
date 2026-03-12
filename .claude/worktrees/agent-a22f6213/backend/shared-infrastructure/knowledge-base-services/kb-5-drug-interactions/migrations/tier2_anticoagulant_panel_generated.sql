-- =====================================================================================
-- TIER 2: ANTICOAGULANT PANEL - BLEEDING RISK DDIS
-- Generated: 2026-01-16T07:02:46.648280
-- DDI Count: 210
-- Generator: KB-5 DDI Matrix Generator Pipeline
-- =====================================================================================

BEGIN;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-rivaroxaban-warfarin', 'warfarin', 'Warfarin', 'rivaroxaban', 'Rivaroxaban',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + DOAC (Rivaroxaban)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Rivaroxaban FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + DOAC (Rivaroxaban)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-warfarin', 'warfarin', 'Warfarin', 'apixaban', 'Apixaban',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + DOAC (Apixaban)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Apixaban FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + DOAC (Apixaban)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-warfarin', 'warfarin', 'Warfarin', 'edoxaban', 'Edoxaban',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + DOAC (Edoxaban)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Edoxaban FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + DOAC (Edoxaban)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-warfarin', 'warfarin', 'Warfarin', 'dabigatran', 'Dabigatran',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + DOAC (Dabigatran)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Dabigatran FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + DOAC (Dabigatran)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-warfarin', 'warfarin', 'Warfarin', 'heparin', 'Heparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + Heparin (Heparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Heparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + Heparin (Heparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-warfarin', 'warfarin', 'Warfarin', 'enoxaparin', 'Enoxaparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + LMWH (Enoxaparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Enoxaparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + LMWH (Enoxaparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-warfarin', 'warfarin', 'Warfarin', 'fondaparinux', 'Fondaparinux',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: VKA (Warfarin) + Factor Xa (Fondaparinux)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Warfarin and Fondaparinux FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + Factor Xa (Fondaparinux)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-warfarin', 'warfarin', 'Warfarin', 'aspirin', 'Aspirin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-warfarin', 'warfarin', 'Warfarin', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-prasugrel-warfarin', 'warfarin', 'Warfarin', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ticagrelor-warfarin', 'warfarin', 'Warfarin', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-warfarin', 'warfarin', 'Warfarin', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: VKA (Warfarin) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Warfarin and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-warfarin', 'warfarin', 'Warfarin', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: VKA (Warfarin) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Warfarin and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-warfarin', 'warfarin', 'Warfarin', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-naproxen-warfarin', 'warfarin', 'Warfarin', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-warfarin', 'warfarin', 'Warfarin', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-warfarin', 'warfarin', 'Warfarin', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: VKA (Warfarin) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Warfarin and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-meloxicam-warfarin', 'warfarin', 'Warfarin', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ketorolac-warfarin', 'warfarin', 'Warfarin', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-indomethacin-warfarin', 'warfarin', 'Warfarin', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: VKA (Warfarin) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Warfarin and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: VKA (Warfarin) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'apixaban', 'Apixaban',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + DOAC (Apixaban)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Apixaban FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + DOAC (Apixaban)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'edoxaban', 'Edoxaban',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + DOAC (Edoxaban)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Edoxaban FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + DOAC (Edoxaban)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'dabigatran', 'Dabigatran',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + DOAC (Dabigatran)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Dabigatran FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + DOAC (Dabigatran)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'heparin', 'Heparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + Heparin (Heparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Heparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + Heparin (Heparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'enoxaparin', 'Enoxaparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + LMWH (Enoxaparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Enoxaparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + LMWH (Enoxaparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'fondaparinux', 'Fondaparinux',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Rivaroxaban) + Factor Xa (Fondaparinux)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Fondaparinux FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + Factor Xa (Fondaparinux)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'aspirin', 'Aspirin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-prasugrel-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-rivaroxaban-ticagrelor', 'rivaroxaban', 'Rivaroxaban', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Rivaroxaban) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Rivaroxaban) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-naproxen-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Rivaroxaban) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-meloxicam-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ketorolac-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-indomethacin-rivaroxaban', 'rivaroxaban', 'Rivaroxaban', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Rivaroxaban and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Rivaroxaban) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-edoxaban', 'apixaban', 'Apixaban', 'edoxaban', 'Edoxaban',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + DOAC (Edoxaban)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Edoxaban FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + DOAC (Edoxaban)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-dabigatran', 'apixaban', 'Apixaban', 'dabigatran', 'Dabigatran',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + DOAC (Dabigatran)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Dabigatran FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + DOAC (Dabigatran)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-heparin', 'apixaban', 'Apixaban', 'heparin', 'Heparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + Heparin (Heparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Heparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + Heparin (Heparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-enoxaparin', 'apixaban', 'Apixaban', 'enoxaparin', 'Enoxaparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + LMWH (Enoxaparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Enoxaparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + LMWH (Enoxaparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-fondaparinux', 'apixaban', 'Apixaban', 'fondaparinux', 'Fondaparinux',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Apixaban) + Factor Xa (Fondaparinux)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Apixaban and Fondaparinux FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + Factor Xa (Fondaparinux)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-aspirin', 'apixaban', 'Apixaban', 'aspirin', 'Aspirin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-clopidogrel', 'apixaban', 'Apixaban', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-prasugrel', 'apixaban', 'Apixaban', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-ticagrelor', 'apixaban', 'Apixaban', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-dipyridamole', 'apixaban', 'Apixaban', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Apixaban) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Apixaban and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-cilostazol', 'apixaban', 'Apixaban', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Apixaban) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Apixaban and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-ibuprofen', 'apixaban', 'Apixaban', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-naproxen', 'apixaban', 'Apixaban', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-diclofenac', 'apixaban', 'Apixaban', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-celecoxib', 'apixaban', 'Apixaban', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Apixaban) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Apixaban and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-meloxicam', 'apixaban', 'Apixaban', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-ketorolac', 'apixaban', 'Apixaban', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-apixaban-indomethacin', 'apixaban', 'Apixaban', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Apixaban) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Apixaban and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Apixaban) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-edoxaban', 'edoxaban', 'Edoxaban', 'dabigatran', 'Dabigatran',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + DOAC (Dabigatran)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Dabigatran FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + DOAC (Dabigatran)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-heparin', 'edoxaban', 'Edoxaban', 'heparin', 'Heparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + Heparin (Heparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Heparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + Heparin (Heparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-enoxaparin', 'edoxaban', 'Edoxaban', 'enoxaparin', 'Enoxaparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + LMWH (Enoxaparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Enoxaparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + LMWH (Enoxaparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-fondaparinux', 'edoxaban', 'Edoxaban', 'fondaparinux', 'Fondaparinux',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Edoxaban) + Factor Xa (Fondaparinux)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Edoxaban and Fondaparinux FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + Factor Xa (Fondaparinux)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-edoxaban', 'edoxaban', 'Edoxaban', 'aspirin', 'Aspirin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-edoxaban', 'edoxaban', 'Edoxaban', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-prasugrel', 'edoxaban', 'Edoxaban', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-ticagrelor', 'edoxaban', 'Edoxaban', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-edoxaban', 'edoxaban', 'Edoxaban', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Edoxaban) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Edoxaban and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-edoxaban', 'edoxaban', 'Edoxaban', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Edoxaban) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Edoxaban and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-ibuprofen', 'edoxaban', 'Edoxaban', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-naproxen', 'edoxaban', 'Edoxaban', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-edoxaban', 'edoxaban', 'Edoxaban', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-edoxaban', 'edoxaban', 'Edoxaban', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Edoxaban) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Edoxaban and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-meloxicam', 'edoxaban', 'Edoxaban', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-ketorolac', 'edoxaban', 'Edoxaban', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-edoxaban-indomethacin', 'edoxaban', 'Edoxaban', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Edoxaban and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Edoxaban) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-heparin', 'dabigatran', 'Dabigatran', 'heparin', 'Heparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + Heparin (Heparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Heparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + Heparin (Heparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-enoxaparin', 'dabigatran', 'Dabigatran', 'enoxaparin', 'Enoxaparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + LMWH (Enoxaparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Enoxaparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + LMWH (Enoxaparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-fondaparinux', 'dabigatran', 'Dabigatran', 'fondaparinux', 'Fondaparinux',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Dabigatran) + Factor Xa (Fondaparinux)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Dabigatran and Fondaparinux FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + Factor Xa (Fondaparinux)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-dabigatran', 'dabigatran', 'Dabigatran', 'aspirin', 'Aspirin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-dabigatran', 'dabigatran', 'Dabigatran', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-prasugrel', 'dabigatran', 'Dabigatran', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-ticagrelor', 'dabigatran', 'Dabigatran', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-dipyridamole', 'dabigatran', 'Dabigatran', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Dabigatran) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Dabigatran and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-dabigatran', 'dabigatran', 'Dabigatran', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Dabigatran) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Dabigatran and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-ibuprofen', 'dabigatran', 'Dabigatran', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-naproxen', 'dabigatran', 'Dabigatran', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-diclofenac', 'dabigatran', 'Dabigatran', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-dabigatran', 'dabigatran', 'Dabigatran', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: DOAC (Dabigatran) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Dabigatran and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-meloxicam', 'dabigatran', 'Dabigatran', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-ketorolac', 'dabigatran', 'Dabigatran', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dabigatran-indomethacin', 'dabigatran', 'Dabigatran', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dabigatran and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: DOAC (Dabigatran) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-heparin', 'heparin', 'Heparin', 'enoxaparin', 'Enoxaparin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + LMWH (Enoxaparin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Enoxaparin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + LMWH (Enoxaparin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-heparin', 'heparin', 'Heparin', 'fondaparinux', 'Fondaparinux',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Heparin (Heparin) + Factor Xa (Fondaparinux)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Heparin and Fondaparinux FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + Factor Xa (Fondaparinux)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-heparin', 'heparin', 'Heparin', 'aspirin', 'Aspirin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-heparin', 'heparin', 'Heparin', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-prasugrel', 'heparin', 'Heparin', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-ticagrelor', 'heparin', 'Heparin', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-heparin', 'heparin', 'Heparin', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Heparin (Heparin) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Heparin and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-heparin', 'heparin', 'Heparin', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Heparin (Heparin) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Heparin and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-ibuprofen', 'heparin', 'Heparin', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-naproxen', 'heparin', 'Heparin', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-heparin', 'heparin', 'Heparin', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-heparin', 'heparin', 'Heparin', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Heparin (Heparin) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Heparin and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-meloxicam', 'heparin', 'Heparin', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-ketorolac', 'heparin', 'Heparin', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-heparin-indomethacin', 'heparin', 'Heparin', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Heparin (Heparin) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Heparin and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Heparin (Heparin) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-fondaparinux', 'enoxaparin', 'Enoxaparin', 'fondaparinux', 'Fondaparinux',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: LMWH (Enoxaparin) + Factor Xa (Fondaparinux)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Enoxaparin and Fondaparinux FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + Factor Xa (Fondaparinux)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-enoxaparin', 'enoxaparin', 'Enoxaparin', 'aspirin', 'Aspirin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-enoxaparin', 'enoxaparin', 'Enoxaparin', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-prasugrel', 'enoxaparin', 'Enoxaparin', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-ticagrelor', 'enoxaparin', 'Enoxaparin', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-enoxaparin', 'enoxaparin', 'Enoxaparin', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: LMWH (Enoxaparin) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Enoxaparin and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-enoxaparin', 'enoxaparin', 'Enoxaparin', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: LMWH (Enoxaparin) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Enoxaparin and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-ibuprofen', 'enoxaparin', 'Enoxaparin', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-naproxen', 'enoxaparin', 'Enoxaparin', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-enoxaparin', 'enoxaparin', 'Enoxaparin', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-enoxaparin', 'enoxaparin', 'Enoxaparin', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: LMWH (Enoxaparin) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Enoxaparin and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-meloxicam', 'enoxaparin', 'Enoxaparin', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-ketorolac', 'enoxaparin', 'Enoxaparin', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-enoxaparin-indomethacin', 'enoxaparin', 'Enoxaparin', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Enoxaparin and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: LMWH (Enoxaparin) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-fondaparinux', 'fondaparinux', 'Fondaparinux', 'aspirin', 'Aspirin',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + COX inhibitor (Aspirin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Fondaparinux and Aspirin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + COX inhibitor (Aspirin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-fondaparinux', 'fondaparinux', 'Fondaparinux', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Fondaparinux and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-prasugrel', 'fondaparinux', 'Fondaparinux', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Fondaparinux and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-ticagrelor', 'fondaparinux', 'Fondaparinux', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Fondaparinux and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-fondaparinux', 'fondaparinux', 'Fondaparinux', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Fondaparinux and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-fondaparinux', 'fondaparinux', 'Fondaparinux', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Fondaparinux and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-ibuprofen', 'fondaparinux', 'Fondaparinux', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Fondaparinux and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-naproxen', 'fondaparinux', 'Fondaparinux', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Fondaparinux and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-fondaparinux', 'fondaparinux', 'Fondaparinux', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Fondaparinux and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-fondaparinux', 'fondaparinux', 'Fondaparinux', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Fondaparinux and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-meloxicam', 'fondaparinux', 'Fondaparinux', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Fondaparinux and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-ketorolac', 'fondaparinux', 'Fondaparinux', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Fondaparinux and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-fondaparinux-indomethacin', 'fondaparinux', 'Fondaparinux', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Fondaparinux and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: Factor Xa (Fondaparinux) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-clopidogrel', 'aspirin', 'Aspirin', 'clopidogrel', 'Clopidogrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + P2Y12 (Clopidogrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Clopidogrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + P2Y12 (Clopidogrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-prasugrel', 'aspirin', 'Aspirin', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-ticagrelor', 'aspirin', 'Aspirin', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-dipyridamole', 'aspirin', 'Aspirin', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: COX inhibitor (Aspirin) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Aspirin and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-cilostazol', 'aspirin', 'Aspirin', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: COX inhibitor (Aspirin) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Aspirin and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-ibuprofen', 'aspirin', 'Aspirin', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-naproxen', 'aspirin', 'Aspirin', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-diclofenac', 'aspirin', 'Aspirin', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-celecoxib', 'aspirin', 'Aspirin', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: COX inhibitor (Aspirin) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Aspirin and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-meloxicam', 'aspirin', 'Aspirin', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-ketorolac', 'aspirin', 'Aspirin', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-aspirin-indomethacin', 'aspirin', 'Aspirin', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Aspirin and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX inhibitor (Aspirin) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-prasugrel', 'clopidogrel', 'Clopidogrel', 'prasugrel', 'Prasugrel',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + P2Y12 (Prasugrel)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Prasugrel FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + P2Y12 (Prasugrel)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-ticagrelor', 'clopidogrel', 'Clopidogrel', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-dipyridamole', 'clopidogrel', 'Clopidogrel', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Clopidogrel and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-clopidogrel', 'clopidogrel', 'Clopidogrel', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Clopidogrel and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-ibuprofen', 'clopidogrel', 'Clopidogrel', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-naproxen', 'clopidogrel', 'Clopidogrel', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-diclofenac', 'clopidogrel', 'Clopidogrel', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-clopidogrel', 'clopidogrel', 'Clopidogrel', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Clopidogrel and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-meloxicam', 'clopidogrel', 'Clopidogrel', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-ketorolac', 'clopidogrel', 'Clopidogrel', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-clopidogrel-indomethacin', 'clopidogrel', 'Clopidogrel', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Clopidogrel and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Clopidogrel) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-prasugrel-ticagrelor', 'prasugrel', 'Prasugrel', 'ticagrelor', 'Ticagrelor',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Prasugrel) + P2Y12 (Ticagrelor)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Prasugrel and Ticagrelor FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + P2Y12 (Ticagrelor)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-prasugrel', 'prasugrel', 'Prasugrel', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Prasugrel) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Prasugrel and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-prasugrel', 'prasugrel', 'Prasugrel', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Prasugrel) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Prasugrel and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-prasugrel', 'prasugrel', 'Prasugrel', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Prasugrel and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-naproxen-prasugrel', 'prasugrel', 'Prasugrel', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Prasugrel and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-prasugrel', 'prasugrel', 'Prasugrel', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Prasugrel and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-prasugrel', 'prasugrel', 'Prasugrel', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Prasugrel) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Prasugrel and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-meloxicam-prasugrel', 'prasugrel', 'Prasugrel', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Prasugrel and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ketorolac-prasugrel', 'prasugrel', 'Prasugrel', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Prasugrel and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-indomethacin-prasugrel', 'prasugrel', 'Prasugrel', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Prasugrel and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Prasugrel) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-ticagrelor', 'ticagrelor', 'Ticagrelor', 'dipyridamole', 'Dipyridamole',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + PDE inhibitor (Dipyridamole)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Ticagrelor and Dipyridamole FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + PDE inhibitor (Dipyridamole)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-ticagrelor', 'ticagrelor', 'Ticagrelor', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Ticagrelor and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-ticagrelor', 'ticagrelor', 'Ticagrelor', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ticagrelor and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-naproxen-ticagrelor', 'ticagrelor', 'Ticagrelor', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ticagrelor and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-ticagrelor', 'ticagrelor', 'Ticagrelor', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ticagrelor and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-ticagrelor', 'ticagrelor', 'Ticagrelor', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Ticagrelor and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-meloxicam-ticagrelor', 'ticagrelor', 'Ticagrelor', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ticagrelor and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ketorolac-ticagrelor', 'ticagrelor', 'Ticagrelor', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ticagrelor and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-indomethacin-ticagrelor', 'ticagrelor', 'Ticagrelor', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ticagrelor and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: P2Y12 (Ticagrelor) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-dipyridamole', 'dipyridamole', 'Dipyridamole', 'cilostazol', 'Cilostazol',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + PDE3 inhibitor (Cilostazol)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Dipyridamole and Cilostazol FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + PDE3 inhibitor (Cilostazol)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-ibuprofen', 'dipyridamole', 'Dipyridamole', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dipyridamole and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-naproxen', 'dipyridamole', 'Dipyridamole', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dipyridamole and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-dipyridamole', 'dipyridamole', 'Dipyridamole', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dipyridamole and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-dipyridamole', 'dipyridamole', 'Dipyridamole', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Dipyridamole and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-meloxicam', 'dipyridamole', 'Dipyridamole', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dipyridamole and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-ketorolac', 'dipyridamole', 'Dipyridamole', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dipyridamole and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-dipyridamole-indomethacin', 'dipyridamole', 'Dipyridamole', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Dipyridamole and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE inhibitor (Dipyridamole) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-ibuprofen', 'cilostazol', 'Cilostazol', 'ibuprofen', 'Ibuprofen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Ibuprofen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Cilostazol and Ibuprofen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Ibuprofen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-naproxen', 'cilostazol', 'Cilostazol', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Cilostazol and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-diclofenac', 'cilostazol', 'Cilostazol', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Cilostazol and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-cilostazol', 'cilostazol', 'Cilostazol', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'moderate', 'probable',
    'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'B', 0.85,
    'FDA_LABEL', 'Cilostazol and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'B', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-meloxicam', 'cilostazol', 'Cilostazol', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Cilostazol and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-ketorolac', 'cilostazol', 'Cilostazol', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Cilostazol and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-cilostazol-indomethacin', 'cilostazol', 'Cilostazol', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Cilostazol and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: PDE3 inhibitor (Cilostazol) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-naproxen', 'ibuprofen', 'Ibuprofen', 'naproxen', 'Naproxen',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Naproxen)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ibuprofen and Naproxen FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Naproxen)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-ibuprofen', 'ibuprofen', 'Ibuprofen', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ibuprofen and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-ibuprofen', 'ibuprofen', 'Ibuprofen', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Ibuprofen) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ibuprofen and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Ibuprofen) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-meloxicam', 'ibuprofen', 'Ibuprofen', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ibuprofen and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-ketorolac', 'ibuprofen', 'Ibuprofen', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ibuprofen and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ibuprofen-indomethacin', 'ibuprofen', 'Ibuprofen', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ibuprofen and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Ibuprofen) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-naproxen', 'naproxen', 'Naproxen', 'diclofenac', 'Diclofenac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Naproxen) + NSAID (Diclofenac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Naproxen and Diclofenac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Naproxen) + NSAID (Diclofenac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-naproxen', 'naproxen', 'Naproxen', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Naproxen) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Naproxen and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Naproxen) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-meloxicam-naproxen', 'naproxen', 'Naproxen', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Naproxen) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Naproxen and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Naproxen) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ketorolac-naproxen', 'naproxen', 'Naproxen', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Naproxen) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Naproxen and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Naproxen) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-indomethacin-naproxen', 'naproxen', 'Naproxen', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Naproxen) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Naproxen and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Naproxen) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-diclofenac', 'diclofenac', 'Diclofenac', 'celecoxib', 'Celecoxib',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Diclofenac) + COX-2 (Celecoxib)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Diclofenac and Celecoxib FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Diclofenac) + COX-2 (Celecoxib)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-meloxicam', 'diclofenac', 'Diclofenac', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Diclofenac) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Diclofenac and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Diclofenac) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-ketorolac', 'diclofenac', 'Diclofenac', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Diclofenac) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Diclofenac and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Diclofenac) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-diclofenac-indomethacin', 'diclofenac', 'Diclofenac', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Diclofenac) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Diclofenac and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Diclofenac) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-meloxicam', 'celecoxib', 'Celecoxib', 'meloxicam', 'Meloxicam',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX-2 (Celecoxib) + NSAID (Meloxicam)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Celecoxib and Meloxicam FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX-2 (Celecoxib) + NSAID (Meloxicam)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-ketorolac', 'celecoxib', 'Celecoxib', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX-2 (Celecoxib) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Celecoxib and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX-2 (Celecoxib) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-celecoxib-indomethacin', 'celecoxib', 'Celecoxib', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: COX-2 (Celecoxib) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Celecoxib and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: COX-2 (Celecoxib) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-ketorolac-meloxicam', 'meloxicam', 'Meloxicam', 'ketorolac', 'Ketorolac',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Meloxicam) + NSAID (Ketorolac)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Meloxicam and Ketorolac FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Meloxicam) + NSAID (Ketorolac)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-indomethacin-meloxicam', 'meloxicam', 'Meloxicam', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Meloxicam) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Meloxicam and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Meloxicam) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, time_to_onset, documentation_level, active,
    dataset_version, evidence, confidence,
    gov_regulatory_authority, gov_regulatory_document, gov_regulatory_url, gov_regulatory_jurisdiction,
    gov_pharmacology_authority, gov_mechanism_evidence, gov_cyp_pathway,
    gov_clinical_authority, gov_severity_source, gov_management_source,
    gov_evidence_grade, gov_last_reviewed_date
) VALUES (
    'bleeding_risk-indomethacin-ketorolac', 'ketorolac', 'Ketorolac', 'indomethacin', 'Indomethacin',
    'bleeding_risk', 'major', 'established',
    'Additive bleeding risk: NSAID (Ketorolac) + NSAID (Indomethacin)',
    'Increased risk of bleeding including GI hemorrhage, intracranial bleeding, and surgical bleeding complications',
    'Avoid combination if possible; if necessary, use lowest effective doses, monitor for bleeding, consider GI prophylaxis',
    'delayed', 'good', true,
    '2025Q3.matrix_gen', 'A', 0.85,
    'FDA_LABEL', 'Ketorolac and Indomethacin FDA Labels', 'https://dailymed.nlm.nih.gov', 'US',
    'DRUGBANK', 'Additive bleeding risk: NSAID (Ketorolac) + NSAID (Indomethacin)', NULL,
    'LEXICOMP', 'DRUGBANK Classification', 'Clinical Guidelines',
    'A', CURRENT_TIMESTAMP
) ON CONFLICT (interaction_id) DO NOTHING;

COMMIT;

-- Verification: SELECT COUNT(*) FROM drug_interactions WHERE category LIKE '%anticoagulant%';