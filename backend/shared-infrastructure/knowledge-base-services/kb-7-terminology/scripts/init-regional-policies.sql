-- Regional Policy Rules for Australian Healthcare
-- KB-7 Terminology Service Phase 2 Implementation
-- Australian-specific clinical policies and governance rules

-- =====================================================
-- AUSTRALIAN REGIONAL POLICY RULES
-- =====================================================

-- Insert Australian-specific policy rule set
INSERT INTO policy_rule_sets (
    rule_set_id, rule_set_name, rule_set_version, description,
    rule_set_type, applicable_domains, applicable_resources,
    evaluation_order, created_by
) VALUES (
    'australian-healthcare-policies',
    'Australian Healthcare Policy Rules',
    '1.0.0',
    'Policy rules specific to Australian healthcare requirements including PBS, TGA, and ACCC compliance',
    'compliance',
    '["medication", "allergy", "diagnosis", "clinical-coding"]',
    '["terminology", "mapping", "api", "amt", "icd10am"]',
    'priority',
    'system'
) ON CONFLICT (rule_set_id) DO NOTHING;

-- AMT (Australian Medicines Terminology) Policy Rules
INSERT INTO policy_rules (
    rule_id, rule_name, rule_version, rule_type, rule_category,
    priority, rule_description, rule_expression, trigger_events,
    resource_types, clinical_domains, action_type, action_parameters,
    escalation_rules, created_by
) VALUES
(
    'amt-medication-validation',
    'AMT Medication Validation Rule',
    '1.0.0',
    'validation',
    'compliance',
    5,
    'Validates that all medication terminologies comply with AMT standards and PBS requirements',
    '{"and": [{"in": [{"var": "terminology_system"}, ["amt", "pbs"]]}, {"or": [{"==": [{"var": "change_type"}, "create"]}, {"==": [{"var": "change_type"}, "update"]}, {"==": [{"var": "change_type"}, "map"]}]}]}',
    '["terminology_change", "mapping_change"]',
    '["terminology", "mapping"]',
    '["medication"]',
    'require_review',
    '{"required_reviewers": ["pharmacy-lead", "amt-specialist"], "tga_compliance_check": true, "pbs_subsidy_verification": true}',
    '{"escalate_to": "medical_director", "if_conditions": {"safety_score": ">= 80", "pbs_impact": true}}',
    'system'
),
(
    'pbs-subsidy-validation',
    'PBS Subsidy Validation Rule',
    '1.0.0',
    'validation',
    'compliance',
    10,
    'Ensures PBS subsidy information is accurate and complies with Department of Health guidelines',
    '{"and": [{"==": [{"var": "pbs_listed"}, true]}, {"!=": [{"var": "subsidy_code"}, null]}]}',
    '["terminology_change", "pbs_update"]',
    '["terminology", "pbs-data"]',
    '["medication"]',
    'require_review',
    '{"required_reviewers": ["pbs-specialist"], "department_health_verification": true, "subsidy_calculation_check": true}',
    '{"escalate_to": "pbs_authority", "if_conditions": {"subsidy_change": "> 20%"}}',
    'system'
),
(
    'tga-artg-compliance',
    'TGA ARTG Compliance Rule',
    '1.0.0',
    'blocking',
    'compliance',
    3,
    'Blocks medication entries that do not have valid TGA ARTG registration',
    '{"and": [{"==": [{"var": "medication_type"}, "prescription"]}, {"or": [{"==": [{"var": "artg_number"}, null]}, {"==": [{"var": "artg_status"}, "cancelled"]}]}]}',
    '["terminology_change", "medication_registration"]',
    '["terminology", "registration"]',
    '["medication"]',
    'block',
    '{"block_reason": "Invalid or missing TGA ARTG registration", "required_documentation": ["artg_certificate", "tga_approval"]}',
    '{"immediate_escalation": true, "notify": ["tga-liaison", "medical_director"]}',
    'system'
),
(
    'amt-snomed-consistency',
    'AMT-SNOMED Consistency Rule',
    '1.0.0',
    'validation',
    'quality',
    20,
    'Ensures consistency between AMT concepts and their SNOMED CT parent concepts',
    '{"and": [{"==": [{"var": "terminology_system"}, "amt"]}, {"!=": [{"var": "snomed_parent"}, null]}]}',
    '["terminology_change", "mapping_change"]',
    '["terminology", "mapping"]',
    '["medication"]',
    'warn',
    '{"consistency_check": true, "parent_validation": true, "hierarchy_verification": true}',
    '{"escalate_if": {"consistency_score": "< 0.8"}}',
    'system'
),
(
    'controlled-substance-au',
    'Australian Controlled Substance Rule',
    '1.0.0',
    'blocking',
    'safety',
    1,
    'Requires special authorization for Australian controlled substances (S8, S9) under TGA scheduling',
    '{"and": [{"in": [{"var": "tga_schedule"}, ["S8", "S9"]]}, {"==": [{"var": "controlled_substance"}, true]}]}',
    '["terminology_change", "scheduling_update"]',
    '["terminology", "scheduling"]',
    '["medication"]',
    'require_review',
    '{"required_reviewers": ["controlled-substance-specialist", "medical_director"], "tga_schedule_verification": true, "state_permit_check": true}',
    '{"immediate_escalation": true, "notify": ["controlled_substance_authority", "tga_liaison"]}',
    'system'
),
(
    'indigenous-health-coding',
    'Indigenous Health Coding Rule',
    '1.0.0',
    'validation',
    'compliance',
    15,
    'Ensures proper coding for Indigenous Australian health services according to AIHW guidelines',
    '{"and": [{"==": [{"var": "indigenous_specific"}, true]}, {"in": [{"var": "service_type"}, ["indigenous_health", "aboriginal_health", "torres_strait_health"]]}]}',
    '["terminology_change", "service_coding"]',
    '["terminology", "service-classification"]',
    '["diagnosis", "procedure"]',
    'require_review',
    '{"required_reviewers": ["indigenous-health-specialist"], "aihw_compliance_check": true, "cultural_appropriateness_review": true}',
    '{"escalate_to": "indigenous_health_authority"}',
    'system'
);

-- ICD-10-AM Policy Rules
INSERT INTO policy_rules (
    rule_id, rule_name, rule_version, rule_type, rule_category,
    priority, rule_description, rule_expression, trigger_events,
    resource_types, clinical_domains, action_type, action_parameters,
    escalation_rules, created_by
) VALUES
(
    'icd10am-drg-validation',
    'ICD-10-AM DRG Validation Rule',
    '1.0.0',
    'validation',
    'compliance',
    10,
    'Validates ICD-10-AM codes used for DRG (Diagnosis Related Group) assignment in Australian hospitals',
    '{"and": [{"==": [{"var": "terminology_system"}, "icd10am"]}, {"==": [{"var": "drg_relevant"}, true]}]}',
    '["terminology_change", "drg_update"]',
    '["terminology", "classification"]',
    '["diagnosis"]',
    'require_review',
    '{"required_reviewers": ["clinical-coder", "drg-specialist"], "ihacpa_compliance": true, "casemix_impact_assessment": true}',
    '{"escalate_to": "ihacpa_liaison", "if_conditions": {"drg_weight_change": "> 0.1"}}',
    'system'
),
(
    'australian-procedure-codes',
    'Australian Procedure Code Validation',
    '1.0.0',
    'validation',
    'compliance',
    12,
    'Validates Australian-specific procedure codes against ACHI (Australian Classification of Health Interventions)',
    '{"and": [{"==": [{"var": "code_system"}, "achi"]}, {"!=": [{"var": "procedure_code"}, null]}]}',
    '["terminology_change", "procedure_update"]',
    '["terminology", "procedure-classification"]',
    '["procedure"]',
    'require_review',
    '{"required_reviewers": ["procedure-specialist", "clinical-coder"], "achi_compliance": true, "mbs_alignment_check": true}',
    '{"escalate_to": "achi_authority"}',
    'system'
),
(
    'private-health-insurance-codes',
    'Private Health Insurance Code Validation',
    '1.0.0',
    'validation',
    'compliance',
    18,
    'Validates codes used for private health insurance claims against PHIAC guidelines',
    '{"and": [{"==": [{"var": "insurance_relevant"}, true]}, {"in": [{"var": "code_usage"}, ["billing", "claiming", "rebate"]]}]}',
    '["terminology_change", "insurance_update"]',
    '["terminology", "billing"]',
    '["procedure", "diagnosis"]',
    'warn',
    '{"phiac_compliance": true, "rebate_calculation_check": true, "fund_recognition_verification": true}',
    '{"escalate_if": {"rebate_impact": "> 500"}}',
    'system'
),
(
    'mental-health-coding-au',
    'Australian Mental Health Coding Rule',
    '1.0.0',
    'validation',
    'safety',
    8,
    'Special validation for mental health terminologies under Australian mental health legislation',
    '{"and": [{"==": [{"var": "clinical_domain"}, "mental_health"]}, {"==": [{"var": "australian_context"}, true]}]}',
    '["terminology_change", "mental_health_update"]',
    '["terminology", "classification"]',
    '["diagnosis", "procedure"]',
    'require_review',
    '{"required_reviewers": ["mental-health-specialist", "clinical-lead"], "legislation_compliance": true, "privacy_impact_assessment": true}',
    '{"escalate_to": "mental_health_authority", "if_conditions": {"involuntary_treatment": true}}',
    'system'
);

-- Regional Compliance Policy Rules
INSERT INTO policy_rules (
    rule_id, rule_name, rule_version, rule_type, rule_category,
    priority, rule_description, rule_expression, trigger_events,
    resource_types, clinical_domains, action_type, action_parameters,
    escalation_rules, created_by
) VALUES
(
    'australian-privacy-compliance',
    'Australian Privacy Compliance Rule',
    '1.0.0',
    'blocking',
    'compliance',
    2,
    'Ensures compliance with Australian Privacy Principles under Privacy Act 1988',
    '{"and": [{"==": [{"var": "contains_personal_info"}, true]}, {"!=": [{"var": "privacy_classification"}, "de-identified"]}]}',
    '["terminology_change", "data_handling"]',
    '["terminology", "data", "api"]',
    '["all"]',
    'require_review',
    '{"required_reviewers": ["privacy-officer", "legal-specialist"], "privacy_impact_assessment": true, "app_compliance": true}',
    '{"immediate_escalation": true, "notify": ["privacy_commissioner", "legal_department"]}',
    'system'
),
(
    'therapeutic-goods-classification',
    'Therapeutic Goods Classification Rule',
    '1.0.0',
    'validation',
    'compliance',
    6,
    'Validates therapeutic goods classification according to TGA guidelines',
    '{"and": [{"==": [{"var": "product_type"}, "therapeutic_good"]}, {"!=": [{"var": "tga_classification"}, null]}]}',
    '["terminology_change", "product_classification"]',
    '["terminology", "classification"]',
    '["medication", "device"]',
    'require_review',
    '{"required_reviewers": ["tga-specialist"], "classification_accuracy": true, "regulatory_compliance": true}',
    '{"escalate_to": "tga_liaison"}',
    'system'
),
(
    'aged-care-coding-compliance',
    'Aged Care Coding Compliance Rule',
    '1.0.0',
    'validation',
    'compliance',
    14,
    'Ensures aged care service coding complies with Australian aged care standards',
    '{"and": [{"==": [{"var": "service_setting"}, "aged_care"]}, {"!=": [{"var": "acfi_relevant"}, false]}]}',
    '["terminology_change", "aged_care_update"]',
    '["terminology", "service-classification"]',
    '["diagnosis", "procedure", "assessment"]',
    'require_review',
    '{"required_reviewers": ["aged-care-specialist"], "acfi_compliance": true, "quality_standards_check": true}',
    '{"escalate_to": "aged_care_authority"}',
    'system'
),
(
    'public-health-reporting-au',
    'Australian Public Health Reporting Rule',
    '1.0.0',
    'validation',
    'compliance',
    11,
    'Validates terminologies used for public health reporting to Australian health authorities',
    '{"and": [{"==": [{"var": "reporting_required"}, true]}, {"in": [{"var": "reporting_authority"}, ["aihw", "health_department", "cdc_au"]]}]}',
    '["terminology_change", "reporting_update"]',
    '["terminology", "reporting"]',
    '["diagnosis", "procedure", "population_health"]',
    'require_review',
    '{"required_reviewers": ["public-health-specialist"], "reporting_standards_compliance": true, "data_quality_check": true}',
    '{"escalate_to": "public_health_authority"}',
    'system'
);

-- Link Australian rules to the Australian rule set
INSERT INTO policy_rule_set_rules (rule_set_id, rule_id, execution_order, is_mandatory, rule_parameters)
SELECT
    rs.id,
    r.id,
    ROW_NUMBER() OVER (ORDER BY r.priority)
FROM policy_rule_sets rs
CROSS JOIN policy_rules r
WHERE rs.rule_set_id = 'australian-healthcare-policies'
  AND r.rule_id IN (
    'amt-medication-validation',
    'pbs-subsidy-validation',
    'tga-artg-compliance',
    'amt-snomed-consistency',
    'controlled-substance-au',
    'indigenous-health-coding',
    'icd10am-drg-validation',
    'australian-procedure-codes',
    'private-health-insurance-codes',
    'mental-health-coding-au',
    'australian-privacy-compliance',
    'therapeutic-goods-classification',
    'aged-care-coding-compliance',
    'public-health-reporting-au'
)
ON CONFLICT (rule_set_id, rule_id) DO NOTHING;

-- =====================================================
-- AUSTRALIAN CLINICAL SAFETY RULES
-- =====================================================

-- Australian-specific clinical safety rules with local evidence and regulatory requirements
INSERT INTO clinical_safety_rules (
    safety_rule_id, rule_name, clinical_domain, safety_category,
    risk_level, rule_description, clinical_logic, trigger_conditions,
    patient_impact_level, required_reviewer_level, evidence_level,
    clinical_references, regulatory_requirements, created_by
) VALUES
(
    'amt-high-risk-medication-au',
    'Australian High-Risk Medication Safety Rule',
    'medication',
    'drug_interaction',
    'critical',
    'Identifies high-risk medications under Australian TGA guidelines requiring enhanced monitoring',
    '{"and": [{"in": [{"var": "tga_schedule"}, ["S4", "S8", "S9"]]}, {"==": [{"var": "high_risk_medication"}, true]}, {"in": [{"var": "risk_category"}, ["black_box", "rems", "high_alert"]]}]}',
    '{"medication_schedules": ["S4", "S8", "S9"], "risk_categories": ["black_box", "rems", "high_alert"], "requires_monitoring": true}',
    'severe',
    'medical_director',
    'systematic_review',
    '{"tga_guidelines": "TGA Guidelines for High-Risk Medicines", "nps_medicinewise": "NPS MedicineWise High-Risk Medicine Guidelines", "safetyandquality_gov_au": "Australian Commission on Safety and Quality in Health Care"}',
    '{"tga_compliance": true, "pbs_authority_required": true, "state_permit_requirements": true}',
    'system'
),
(
    'aboriginal-torres-strait-medication-safety',
    'Aboriginal and Torres Strait Islander Medication Safety Rule',
    'medication',
    'cultural_safety',
    'high',
    'Ensures culturally safe medication practices for Aboriginal and Torres Strait Islander patients',
    '{"and": [{"==": [{"var": "patient_indigenous_status"}, true]}, {"!=": [{"var": "cultural_considerations"}, null]}]}',
    '{"indigenous_patient": true, "cultural_safety_required": true, "community_consultation": true}',
    'significant',
    'clinical_lead',
    'expert_opinion',
    '{"aihw_guidelines": "AIHW Indigenous Health Guidelines", "racgp_guidelines": "RACGP Aboriginal and Torres Strait Islander Health", "natsihwa": "National Aboriginal and Torres Strait Islander Health Worker Association Guidelines"}',
    '{"cultural_safety_standards": true, "community_consultation_required": true}',
    'system'
),
(
    'pregnancy-medication-tga-category',
    'TGA Pregnancy Category Safety Rule',
    'medication',
    'pregnancy',
    'critical',
    'Validates pregnancy safety categories according to TGA guidelines for Australian medications',
    '{"and": [{"==": [{"var": "patient_pregnant"}, true]}, {"in": [{"var": "tga_pregnancy_category"}, ["D", "X"]]}, {"==": [{"var": "pregnancy_risk_assessment"}, "required"}}]}',
    '{"pregnancy_categories": ["D", "X"], "teratogenic_risk": "high", "tga_classification": true}',
    'severe',
    'medical_director',
    'systematic_review',
    '{"tga_pregnancy_guidelines": "TGA Pregnancy Classification System", "ranzcog_guidelines": "RANZCOG Medication in Pregnancy Guidelines", "motherisk": "Motherisk Australia Guidelines"}',
    '{"tga_pregnancy_classification": true, "teratogenicity_assessment": true}',
    'system'
),
(
    'paediatric-dosing-australia',
    'Australian Paediatric Dosing Safety Rule',
    'medication',
    'dosing',
    'high',
    'Ensures safe paediatric dosing according to Australian paediatric guidelines and PBS regulations',
    '{"and": [{"<=": [{"var": "patient_age_years"}, 18]}, {"==": [{"var": "weight_based_dosing"}, true]}, {"!=": [{"var": "paediatric_indication"}, null]}]}',
    '{"patient_age": "≤18 years", "weight_based_dosing": true, "paediatric_formulation": true}',
    'significant',
    'clinical_lead',
    'rct',
    '{"racp_guidelines": "Royal Australasian College of Physicians Paediatric Guidelines", "apls_guidelines": "Australian Paediatric Life Support", "pbs_paediatric": "PBS Paediatric Dosing Guidelines"}',
    '{"pbs_paediatric_authority": true, "weight_verification": true}',
    'system'
),
(
    'mental-health-medication-au',
    'Australian Mental Health Medication Safety Rule',
    'medication',
    'mental_health',
    'high',
    'Special safety measures for mental health medications under Australian mental health legislation',
    '{"and": [{"==": [{"var": "medication_class"}, "psychiatric"]}, {"==": [{"var": "mental_health_act_relevant"}, true]}, {"in": [{"var": "patient_status"}, ["involuntary", "community_treatment_order"]]}]}',
    '{"psychiatric_medication": true, "mental_health_act": true, "involuntary_treatment": true}',
    'significant',
    'medical_director',
    'systematic_review',
    '{"ranzcp_guidelines": "Royal Australian and New Zealand College of Psychiatrists Guidelines", "mental_health_act": "Australian Mental Health Acts", "orygen_guidelines": "Orygen Youth Mental Health Guidelines"}',
    '{"mental_health_act_compliance": true, "capacity_assessment": true, "tribunal_notification": true}',
    'system'
),
(
    'aged-care-medication-management',
    'Aged Care Medication Management Safety Rule',
    'medication',
    'geriatric',
    'moderate',
    'Ensures safe medication management in Australian aged care facilities under aged care standards',
    '{"and": [{">=": [{"var": "patient_age_years"}, 65]}, {"==": [{"var": "aged_care_resident"}, true]}, {">=": [{"var": "medication_count"}, 5]}]}',
    '{"aged_care_setting": true, "polypharmacy": true, "elderly_patient": true}',
    'moderate',
    'clinical_lead',
    'cohort_study',
    '{"aged_care_standards": "Australian Aged Care Quality Standards", "beers_criteria_australia": "Australian Adapted Beers Criteria", "stopp_start_australia": "STOPP/START Criteria for Australia"}',
    '{"aged_care_quality_standards": true, "medication_review_requirements": true}',
    'system'
);

-- =====================================================
-- REGIONAL CLINICAL REVIEWERS
-- =====================================================

-- Australian-specific clinical reviewer roles
INSERT INTO clinical_reviewers (
    reviewer_id, full_name, email, role, authorization_level,
    review_domains, max_risk_level, specializations, created_by
) VALUES
(
    'tga-specialist',
    'Dr. Margaret Chen',
    'tga.specialist@cardiofit.health',
    'regulatory_specialist',
    'expert',
    '["medication", "regulatory", "tga-compliance"]',
    'critical',
    '{"areas": ["therapeutic_goods", "tga_registration", "artg_compliance", "scheduling"], "certifications": ["TGA_Regulatory_Affairs", "RAPS_Certification"]}',
    'system'
),
(
    'pbs-specialist',
    'Dr. James Wilson',
    'pbs.specialist@cardiofit.health',
    'pharmaceutical_economist',
    'expert',
    '["medication", "economics", "pbs-subsidy"]',
    'high',
    '{"areas": ["pharmaceutical_economics", "pbs_listing", "subsidy_analysis", "health_technology_assessment"], "certifications": ["PBAC_Experience", "HTA_Certification"]}',
    'system'
),
(
    'amt-specialist',
    'Dr. Rachel Thompson',
    'amt.specialist@cardiofit.health',
    'terminology_specialist',
    'expert',
    '["medication", "terminology", "amt-mapping"]',
    'high',
    '{"areas": ["amt_maintenance", "snomed_ct_au", "terminology_mapping", "medication_coding"], "certifications": ["NCTS_Certification", "SNOMED_International"]}',
    'system'
),
(
    'indigenous-health-specialist',
    'Dr. David Namatjira',
    'indigenous.health@cardiofit.health',
    'indigenous_health_specialist',
    'expert',
    '["indigenous_health", "cultural_safety", "community_health"]',
    'high',
    '{"areas": ["aboriginal_health", "torres_strait_health", "cultural_competency", "community_engagement"], "certifications": ["Indigenous_Health_Specialization", "Cultural_Safety_Training"]}',
    'system'
),
(
    'controlled-substance-specialist',
    'Dr. Andrew Roberts',
    'controlled.substances@cardiofit.health',
    'controlled_substance_specialist',
    'director',
    '["medication", "controlled_substances", "regulatory"]',
    'critical',
    '{"areas": ["s8_medications", "addiction_medicine", "regulatory_compliance", "permit_requirements"], "certifications": ["Addiction_Medicine", "Controlled_Substances_Authority"]}',
    'system'
),
(
    'drg-specialist',
    'Dr. Catherine Lee',
    'drg.specialist@cardiofit.health',
    'drg_specialist',
    'expert',
    '["diagnosis", "coding", "casemix"]',
    'high',
    '{"areas": ["drg_assignment", "casemix_analysis", "clinical_coding", "ihacpa_guidelines"], "certifications": ["Clinical_Coding", "DRG_Specialist", "IHACPA_Training"]}',
    'system'
),
(
    'mental-health-specialist',
    'Dr. Sarah Kim',
    'mental.health@cardiofit.health',
    'mental_health_specialist',
    'expert',
    '["mental_health", "psychiatry", "legislation"]',
    'critical',
    '{"areas": ["psychiatric_medication", "mental_health_act", "capacity_assessment", "involuntary_treatment"], "certifications": ["RANZCP_Fellowship", "Mental_Health_Law"]}',
    'system'
),
(
    'aged-care-specialist',
    'Dr. Helen Brown',
    'aged.care@cardiofit.health',
    'aged_care_specialist',
    'expert',
    '["aged_care", "geriatrics", "medication_management"]',
    'high',
    '{"areas": ["aged_care_standards", "geriatric_medicine", "polypharmacy", "dementia_care"], "certifications": ["Geriatric_Medicine", "Aged_Care_Quality"]}',
    'system'
),
(
    'public-health-specialist',
    'Dr. Michael O\'Connor',
    'public.health@cardiofit.health',
    'public_health_specialist',
    'expert',
    '["public_health", "epidemiology", "reporting"]',
    'moderate',
    '{"areas": ["public_health_reporting", "epidemiology", "health_surveillance", "communicable_diseases"], "certifications": ["Public_Health_Medicine", "Epidemiology"]}',
    'system'
),
(
    'privacy-officer',
    'Ms. Jennifer Walsh',
    'privacy.officer@cardiofit.health',
    'privacy_officer',
    'expert',
    '["privacy", "data_protection", "compliance"]',
    'critical',
    '{"areas": ["privacy_law", "data_protection", "app_compliance", "health_information_management"], "certifications": ["Privacy_Law", "Health_Information_Management", "IAPP_Certification"]}',
    'system'
)
ON CONFLICT (reviewer_id) DO NOTHING;

-- =====================================================
-- REGIONAL COMPLIANCE TEMPLATES
-- =====================================================

-- Create view for Australian compliance monitoring
CREATE OR REPLACE VIEW v_australian_compliance_dashboard AS
SELECT
    pe.evaluation_timestamp::date as evaluation_date,
    pr.rule_category,
    pr.rule_name,
    pe.evaluation_result,
    COUNT(*) as evaluation_count,
    COUNT(*) FILTER (WHERE pe.evaluation_result = 'block') as blocked_count,
    COUNT(*) FILTER (WHERE pe.evaluation_result = 'require_review') as review_required_count,
    COUNT(*) FILTER (WHERE pe.escalation_triggered = true) as escalation_count,
    ARRAY_AGG(DISTINCT pe.actor_id) FILTER (WHERE pe.evaluation_result != 'allow') as affected_users
FROM policy_evaluations pe
JOIN policy_rules pr ON pe.rule_id = pr.id
JOIN policy_rule_set_rules prsr ON pr.id = prsr.rule_id
JOIN policy_rule_sets prs ON prsr.rule_set_id = prs.id
WHERE prs.rule_set_id = 'australian-healthcare-policies'
  AND pe.evaluation_timestamp >= NOW() - INTERVAL '30 days'
GROUP BY
    pe.evaluation_timestamp::date,
    pr.rule_category,
    pr.rule_name,
    pe.evaluation_result
ORDER BY evaluation_date DESC, evaluation_count DESC;

-- Create view for TGA compliance tracking
CREATE OR REPLACE VIEW v_tga_compliance_tracking AS
SELECT
    av.event_timestamp::date as compliance_date,
    av.resource_type,
    COUNT(*) as total_events,
    COUNT(*) FILTER (WHERE av.compliance_flags->>'tga_compliant' = 'true') as compliant_events,
    COUNT(*) FILTER (WHERE av.compliance_flags->>'tga_compliant' = 'false') as non_compliant_events,
    COUNT(*) FILTER (WHERE av.patient_safety_flag = true) as safety_flagged_events,
    ROUND(
        (COUNT(*) FILTER (WHERE av.compliance_flags->>'tga_compliant' = 'true') * 100.0) /
        NULLIF(COUNT(*), 0), 2
    ) as compliance_percentage
FROM audit_events av
WHERE av.clinical_domain = 'medication'
  AND av.event_timestamp >= NOW() - INTERVAL '30 days'
  AND av.compliance_flags ? 'tga_compliant'
GROUP BY av.event_timestamp::date, av.resource_type
ORDER BY compliance_date DESC;

-- Create view for PBS impact monitoring
CREATE OR REPLACE VIEW v_pbs_impact_monitoring AS
SELECT
    tc.change_timestamp::date as change_date,
    tc.terminology_system,
    tc.concept_id,
    tc.concept_name,
    tc.change_type,
    tc.metadata->>'pbs_impact' as pbs_impact,
    tc.metadata->>'subsidy_change' as subsidy_change,
    tc.approval_status,
    cr.full_name as approver_name
FROM terminology_changes tc
LEFT JOIN clinical_reviewers cr ON tc.approved_by = cr.reviewer_id
WHERE tc.terminology_system IN ('amt', 'pbs')
  AND tc.change_timestamp >= NOW() - INTERVAL '30 days'
  AND tc.metadata ? 'pbs_impact'
ORDER BY change_date DESC, tc.change_timestamp DESC;

-- Regional reporting function
CREATE OR REPLACE FUNCTION generate_australian_compliance_report(
    p_start_date DATE DEFAULT NOW() - INTERVAL '30 days',
    p_end_date DATE DEFAULT NOW()
)
RETURNS TABLE (
    report_section TEXT,
    metric_name TEXT,
    metric_value NUMERIC,
    metric_description TEXT,
    compliance_status TEXT
) AS $$
BEGIN
    -- TGA Compliance Metrics
    RETURN QUERY
    SELECT
        'TGA Compliance'::TEXT as report_section,
        'Overall Compliance Rate'::TEXT as metric_name,
        ROUND(
            (COUNT(*) FILTER (WHERE av.compliance_flags->>'tga_compliant' = 'true') * 100.0) /
            NULLIF(COUNT(*), 0), 2
        ) as metric_value,
        'Percentage of medication events that meet TGA compliance requirements'::TEXT as metric_description,
        CASE
            WHEN ROUND(
                (COUNT(*) FILTER (WHERE av.compliance_flags->>'tga_compliant' = 'true') * 100.0) /
                NULLIF(COUNT(*), 0), 2
            ) >= 95 THEN 'EXCELLENT'
            WHEN ROUND(
                (COUNT(*) FILTER (WHERE av.compliance_flags->>'tga_compliant' = 'true') * 100.0) /
                NULLIF(COUNT(*), 0), 2
            ) >= 90 THEN 'GOOD'
            WHEN ROUND(
                (COUNT(*) FILTER (WHERE av.compliance_flags->>'tga_compliant' = 'true') * 100.0) /
                NULLIF(COUNT(*), 0), 2
            ) >= 80 THEN 'ACCEPTABLE'
            ELSE 'NEEDS_IMPROVEMENT'
        END as compliance_status
    FROM audit_events av
    WHERE av.clinical_domain = 'medication'
      AND av.event_timestamp::date BETWEEN p_start_date AND p_end_date
      AND av.compliance_flags ? 'tga_compliant';

    -- PBS Compliance Metrics
    RETURN QUERY
    SELECT
        'PBS Compliance'::TEXT as report_section,
        'PBS Subsidy Accuracy'::TEXT as metric_name,
        ROUND(
            (COUNT(*) FILTER (WHERE tc.metadata->>'pbs_verified' = 'true') * 100.0) /
            NULLIF(COUNT(*), 0), 2
        ) as metric_value,
        'Percentage of PBS-related changes that have verified subsidy information'::TEXT as metric_description,
        CASE
            WHEN ROUND(
                (COUNT(*) FILTER (WHERE tc.metadata->>'pbs_verified' = 'true') * 100.0) /
                NULLIF(COUNT(*), 0), 2
            ) >= 98 THEN 'EXCELLENT'
            WHEN ROUND(
                (COUNT(*) FILTER (WHERE tc.metadata->>'pbs_verified' = 'true') * 100.0) /
                NULLIF(COUNT(*), 0), 2
            ) >= 95 THEN 'GOOD'
            ELSE 'NEEDS_IMPROVEMENT'
        END as compliance_status
    FROM terminology_changes tc
    WHERE tc.terminology_system IN ('amt', 'pbs')
      AND tc.change_timestamp::date BETWEEN p_start_date AND p_end_date
      AND tc.metadata ? 'pbs_relevant';

    -- Clinical Review Response Time
    RETURN QUERY
    SELECT
        'Clinical Governance'::TEXT as report_section,
        'Average Review Response Time (Hours)'::TEXT as metric_name,
        ROUND(
            AVG(EXTRACT(EPOCH FROM (aus.clinical_review_timestamp - aus.initiated_at)) / 3600)::NUMERIC, 2
        ) as metric_value,
        'Average time from clinical review request to completion for Australian-specific cases'::TEXT as metric_description,
        CASE
            WHEN AVG(EXTRACT(EPOCH FROM (aus.clinical_review_timestamp - aus.initiated_at)) / 3600) <= 24 THEN 'EXCELLENT'
            WHEN AVG(EXTRACT(EPOCH FROM (aus.clinical_review_timestamp - aus.initiated_at)) / 3600) <= 48 THEN 'GOOD'
            WHEN AVG(EXTRACT(EPOCH FROM (aus.clinical_review_timestamp - aus.initiated_at)) / 3600) <= 72 THEN 'ACCEPTABLE'
            ELSE 'NEEDS_IMPROVEMENT'
        END as compliance_status
    FROM audit_sessions aus
    WHERE aus.clinical_review_status = 'approved'
      AND aus.clinical_reviewer IN (
          SELECT reviewer_id FROM clinical_reviewers
          WHERE specializations ? 'areas'
          AND specializations->'areas' @> '["tga_registration"]'::jsonb
      )
      AND aus.initiated_at::date BETWEEN p_start_date AND p_end_date;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions for Australian compliance monitoring
GRANT SELECT ON v_australian_compliance_dashboard TO kb_readonly_user;
GRANT SELECT ON v_tga_compliance_tracking TO kb_readonly_user;
GRANT SELECT ON v_pbs_impact_monitoring TO kb_readonly_user;
GRANT EXECUTE ON FUNCTION generate_australian_compliance_report TO kb_audit_user;