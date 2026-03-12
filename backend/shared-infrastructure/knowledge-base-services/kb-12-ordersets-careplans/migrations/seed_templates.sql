-- KB-12 Order Sets & Care Plans - Seed Data
-- Production-ready order set templates based on clinical guidelines
-- NO FALLBACKS - All templates must be loaded from this seed data

-- ============================================
-- ADMISSION ORDER SETS - CARDIAC
-- ============================================

INSERT INTO order_set_templates (
    template_id, name, description, version, category, subcategory, specialty,
    condition_code, condition_system, condition_display,
    sections, time_constraints, clinical_context, status, created_by, approved_by, approved_at
) VALUES
(
    'ADM-CHF-001',
    'CHF Exacerbation Admission Orders',
    'Comprehensive admission order set for acute decompensated heart failure (ADHF) based on ACC/AHA Heart Failure Guidelines',
    '2.0',
    'admission',
    'cardiac',
    'cardiology',
    '84114007',
    'http://snomed.info/sct',
    'Heart failure',
    '[
        {
            "section_id": "vitals",
            "name": "Vital Signs & Monitoring",
            "orders": [
                {"order_id": "VS-001", "name": "Vital signs q4h", "order_type": "nursing", "priority": "routine", "default_selected": true},
                {"order_id": "MON-001", "name": "Continuous telemetry monitoring", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-002", "name": "Daily weights", "order_type": "nursing", "priority": "routine", "default_selected": true},
                {"order_id": "MON-003", "name": "Strict I&O", "order_type": "nursing", "priority": "routine", "default_selected": true}
            ]
        },
        {
            "section_id": "labs",
            "name": "Laboratory Studies",
            "orders": [
                {"order_id": "LAB-001", "name": "BMP", "order_type": "lab", "priority": "stat", "default_selected": true, "loinc_code": "24320-4"},
                {"order_id": "LAB-002", "name": "BNP or NT-proBNP", "order_type": "lab", "priority": "stat", "default_selected": true, "loinc_code": "42637-9"},
                {"order_id": "LAB-003", "name": "Troponin", "order_type": "lab", "priority": "stat", "default_selected": true, "loinc_code": "6598-7"},
                {"order_id": "LAB-004", "name": "CBC with differential", "order_type": "lab", "priority": "routine", "default_selected": true},
                {"order_id": "LAB-005", "name": "Hepatic function panel", "order_type": "lab", "priority": "routine", "default_selected": true},
                {"order_id": "LAB-006", "name": "TSH", "order_type": "lab", "priority": "routine", "default_selected": true}
            ]
        },
        {
            "section_id": "medications",
            "name": "Medications",
            "orders": [
                {"order_id": "MED-001", "name": "Furosemide IV", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "4603", "dose": "40", "unit": "mg", "route": "IV", "frequency": "q12h"},
                {"order_id": "MED-002", "name": "Potassium chloride", "order_type": "medication", "priority": "routine", "default_selected": true, "rxnorm_code": "8591", "dose": "20", "unit": "mEq", "route": "PO", "frequency": "daily"},
                {"order_id": "MED-003", "name": "Continue home beta-blocker if stable", "order_type": "medication", "priority": "routine", "default_selected": false}
            ]
        },
        {
            "section_id": "imaging",
            "name": "Imaging Studies",
            "orders": [
                {"order_id": "IMG-001", "name": "Chest X-ray PA and lateral", "order_type": "imaging", "priority": "stat", "default_selected": true},
                {"order_id": "IMG-002", "name": "Echocardiogram if no recent study", "order_type": "imaging", "priority": "routine", "default_selected": true}
            ]
        }
    ]'::jsonb,
    '[]'::jsonb,
    '{"evidence_level": "A", "guideline_source": "ACC/AHA Heart Failure Guidelines 2022", "cms_measure": "HF-1"}'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
),
(
    'ADM-MI-001',
    'Acute MI/STEMI Admission Orders',
    'Time-critical admission order set for ST-elevation myocardial infarction based on ACC/AHA STEMI Guidelines. Door-to-balloon time <90 minutes required.',
    '2.0',
    'admission',
    'cardiac',
    'cardiology',
    '401303003',
    'http://snomed.info/sct',
    'Acute ST elevation myocardial infarction',
    '[
        {
            "section_id": "immediate",
            "name": "Immediate Interventions (STAT)",
            "orders": [
                {"order_id": "STAT-001", "name": "12-lead ECG", "order_type": "diagnostic", "priority": "stat", "default_selected": true, "time_critical": true, "deadline_minutes": 10},
                {"order_id": "STAT-002", "name": "Notify cath lab team", "order_type": "communication", "priority": "stat", "default_selected": true, "time_critical": true},
                {"order_id": "STAT-003", "name": "Aspirin 325mg chewable", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "1191", "dose": "325", "unit": "mg", "route": "PO"},
                {"order_id": "STAT-004", "name": "Ticagrelor 180mg loading dose", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "1116632", "dose": "180", "unit": "mg", "route": "PO"},
                {"order_id": "STAT-005", "name": "Heparin IV bolus", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "5224"}
            ]
        },
        {
            "section_id": "labs",
            "name": "Laboratory Studies",
            "orders": [
                {"order_id": "LAB-001", "name": "Troponin stat", "order_type": "lab", "priority": "stat", "default_selected": true, "loinc_code": "6598-7"},
                {"order_id": "LAB-002", "name": "BMP stat", "order_type": "lab", "priority": "stat", "default_selected": true},
                {"order_id": "LAB-003", "name": "CBC stat", "order_type": "lab", "priority": "stat", "default_selected": true},
                {"order_id": "LAB-004", "name": "PT/INR/PTT", "order_type": "lab", "priority": "stat", "default_selected": true},
                {"order_id": "LAB-005", "name": "Type and screen", "order_type": "lab", "priority": "stat", "default_selected": true}
            ]
        },
        {
            "section_id": "monitoring",
            "name": "Monitoring",
            "orders": [
                {"order_id": "MON-001", "name": "Continuous cardiac monitoring", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-002", "name": "Continuous pulse oximetry", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-003", "name": "Vital signs q15min x 4, then q1h", "order_type": "nursing", "priority": "stat", "default_selected": true}
            ]
        }
    ]'::jsonb,
    '[
        {"constraint_id": "TC-001", "action": "ECG acquisition", "deadline_minutes": 10, "severity": "critical", "metrics_code": "D2ECG"},
        {"constraint_id": "TC-002", "action": "Cath lab activation", "deadline_minutes": 30, "severity": "critical", "metrics_code": "D2ACTIVATION"},
        {"constraint_id": "TC-003", "action": "Door-to-balloon", "deadline_minutes": 90, "severity": "critical", "metrics_code": "D2B"}
    ]'::jsonb,
    '{"evidence_level": "A", "guideline_source": "ACC/AHA STEMI Guidelines 2021", "cms_measure": "AMI-8a", "time_critical": true}'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
);

-- ============================================
-- ADMISSION ORDER SETS - SEPSIS (SEP-1)
-- ============================================

INSERT INTO order_set_templates (
    template_id, name, description, version, category, subcategory, specialty,
    condition_code, condition_system, condition_display,
    sections, time_constraints, clinical_context, status, created_by, approved_by, approved_at
) VALUES
(
    'ADM-SEP-001',
    'Sepsis/Septic Shock (SEP-1 Bundle)',
    'Time-critical sepsis bundle based on CMS SEP-1 measure and Surviving Sepsis Campaign Guidelines. 3-hour and 6-hour bundle compliance required.',
    '3.0',
    'admission',
    'metabolic',
    'critical_care',
    '91302008',
    'http://snomed.info/sct',
    'Sepsis',
    '[
        {
            "section_id": "3hr_bundle",
            "name": "3-Hour Bundle (STAT)",
            "orders": [
                {"order_id": "SEP3-001", "name": "Serum lactate level", "order_type": "lab", "priority": "stat", "default_selected": true, "loinc_code": "2524-7", "time_critical": true, "deadline_minutes": 180},
                {"order_id": "SEP3-002", "name": "Blood cultures x2 (different sites) before antibiotics", "order_type": "lab", "priority": "stat", "default_selected": true, "time_critical": true, "deadline_minutes": 180},
                {"order_id": "SEP3-003", "name": "Broad-spectrum antibiotics", "order_type": "medication", "priority": "stat", "default_selected": true, "time_critical": true, "deadline_minutes": 180},
                {"order_id": "SEP3-004", "name": "IV fluid bolus 30mL/kg crystalloid if hypotensive or lactate >= 4", "order_type": "medication", "priority": "stat", "default_selected": true, "time_critical": true, "deadline_minutes": 180}
            ]
        },
        {
            "section_id": "6hr_bundle",
            "name": "6-Hour Bundle",
            "orders": [
                {"order_id": "SEP6-001", "name": "Vasopressors if hypotension persists after fluid resuscitation", "order_type": "medication", "priority": "stat", "default_selected": false, "time_critical": true, "deadline_minutes": 360},
                {"order_id": "SEP6-002", "name": "Repeat lactate if initial elevated", "order_type": "lab", "priority": "routine", "default_selected": true, "time_critical": true, "deadline_minutes": 360},
                {"order_id": "SEP6-003", "name": "Reassess volume status and tissue perfusion", "order_type": "assessment", "priority": "routine", "default_selected": true}
            ]
        },
        {
            "section_id": "antibiotics",
            "name": "Antibiotic Selection",
            "orders": [
                {"order_id": "ABX-001", "name": "Vancomycin IV", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "11124", "dose": "25", "unit": "mg/kg", "route": "IV", "frequency": "loading dose"},
                {"order_id": "ABX-002", "name": "Piperacillin-tazobactam IV", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "391876", "dose": "4.5", "unit": "g", "route": "IV", "frequency": "q6h"},
                {"order_id": "ABX-003", "name": "Consider antifungal coverage if risk factors", "order_type": "medication", "priority": "routine", "default_selected": false}
            ]
        },
        {
            "section_id": "monitoring",
            "name": "Monitoring & Assessment",
            "orders": [
                {"order_id": "MON-001", "name": "Continuous cardiac monitoring", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-002", "name": "Arterial line placement if vasopressors needed", "order_type": "procedure", "priority": "routine", "default_selected": false},
                {"order_id": "MON-003", "name": "Central venous catheter if vasopressors needed", "order_type": "procedure", "priority": "routine", "default_selected": false},
                {"order_id": "MON-004", "name": "Urine output q1h", "order_type": "nursing", "priority": "stat", "default_selected": true}
            ]
        }
    ]'::jsonb,
    '[
        {"constraint_id": "SEP-TC-001", "action": "Lactate measurement", "deadline_minutes": 180, "severity": "critical", "metrics_code": "SEP1-LACTATE"},
        {"constraint_id": "SEP-TC-002", "action": "Blood cultures obtained", "deadline_minutes": 180, "severity": "critical", "metrics_code": "SEP1-BCX"},
        {"constraint_id": "SEP-TC-003", "action": "Broad-spectrum antibiotics administered", "deadline_minutes": 180, "severity": "critical", "metrics_code": "SEP1-ABX"},
        {"constraint_id": "SEP-TC-004", "action": "Fluid resuscitation completed", "deadline_minutes": 180, "severity": "critical", "metrics_code": "SEP1-IVF"},
        {"constraint_id": "SEP-TC-005", "action": "Vasopressors if persistent hypotension", "deadline_minutes": 360, "severity": "warning", "metrics_code": "SEP1-VASO"},
        {"constraint_id": "SEP-TC-006", "action": "Repeat lactate if initially elevated", "deadline_minutes": 360, "severity": "warning", "metrics_code": "SEP1-RELACTATE"}
    ]'::jsonb,
    '{"evidence_level": "A", "guideline_source": "Surviving Sepsis Campaign 2021", "cms_measure": "SEP-1", "time_critical": true, "mortality_impact": "high"}'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
);

-- ============================================
-- ADMISSION ORDER SETS - STROKE
-- ============================================

INSERT INTO order_set_templates (
    template_id, name, description, version, category, subcategory, specialty,
    condition_code, condition_system, condition_display,
    sections, time_constraints, clinical_context, status, created_by, approved_by, approved_at
) VALUES
(
    'ADM-STROKE-001',
    'Acute Ischemic Stroke Orders',
    'Time-critical stroke order set based on AHA/ASA Stroke Guidelines. Door-to-needle time for tPA <60 minutes.',
    '2.0',
    'admission',
    'neuro',
    'neurology',
    '422504002',
    'http://snomed.info/sct',
    'Acute ischemic stroke',
    '[
        {
            "section_id": "immediate",
            "name": "Immediate Assessment (STAT)",
            "orders": [
                {"order_id": "STK-001", "name": "NIHSS assessment", "order_type": "assessment", "priority": "stat", "default_selected": true, "time_critical": true, "deadline_minutes": 15},
                {"order_id": "STK-002", "name": "CT head without contrast", "order_type": "imaging", "priority": "stat", "default_selected": true, "time_critical": true, "deadline_minutes": 25},
                {"order_id": "STK-003", "name": "Blood glucose POC", "order_type": "lab", "priority": "stat", "default_selected": true, "time_critical": true, "deadline_minutes": 15},
                {"order_id": "STK-004", "name": "Notify stroke team", "order_type": "communication", "priority": "stat", "default_selected": true}
            ]
        },
        {
            "section_id": "labs",
            "name": "Laboratory Studies",
            "orders": [
                {"order_id": "LAB-001", "name": "CBC", "order_type": "lab", "priority": "stat", "default_selected": true},
                {"order_id": "LAB-002", "name": "BMP", "order_type": "lab", "priority": "stat", "default_selected": true},
                {"order_id": "LAB-003", "name": "PT/INR/PTT", "order_type": "lab", "priority": "stat", "default_selected": true},
                {"order_id": "LAB-004", "name": "Troponin", "order_type": "lab", "priority": "stat", "default_selected": true}
            ]
        },
        {
            "section_id": "thrombolysis",
            "name": "Thrombolysis Consideration",
            "orders": [
                {"order_id": "TPA-001", "name": "tPA eligibility checklist", "order_type": "assessment", "priority": "stat", "default_selected": true},
                {"order_id": "TPA-002", "name": "Alteplase IV if eligible", "order_type": "medication", "priority": "stat", "default_selected": false, "rxnorm_code": "8410", "time_critical": true, "deadline_minutes": 60},
                {"order_id": "TPA-003", "name": "Consider thrombectomy if LVO suspected", "order_type": "procedure", "priority": "stat", "default_selected": false}
            ]
        },
        {
            "section_id": "monitoring",
            "name": "Monitoring",
            "orders": [
                {"order_id": "MON-001", "name": "Neuro checks q1h x 24h", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-002", "name": "BP monitoring q15min during/after tPA", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-003", "name": "NPO until swallow evaluation", "order_type": "nursing", "priority": "stat", "default_selected": true}
            ]
        }
    ]'::jsonb,
    '[
        {"constraint_id": "STK-TC-001", "action": "CT head completed", "deadline_minutes": 25, "severity": "critical", "metrics_code": "D2CT"},
        {"constraint_id": "STK-TC-002", "action": "Door-to-needle (tPA)", "deadline_minutes": 60, "severity": "critical", "metrics_code": "D2N"},
        {"constraint_id": "STK-TC-003", "action": "Neuro team evaluation", "deadline_minutes": 15, "severity": "critical", "metrics_code": "D2NEURO"}
    ]'::jsonb,
    '{"evidence_level": "A", "guideline_source": "AHA/ASA Stroke Guidelines 2019", "cms_measure": "STK-4", "time_critical": true}'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
);

-- ============================================
-- EMERGENCY PROTOCOLS
-- ============================================

INSERT INTO order_set_templates (
    template_id, name, description, version, category, subcategory, specialty,
    condition_code, condition_system, condition_display,
    sections, time_constraints, clinical_context, status, created_by, approved_by, approved_at
) VALUES
(
    'EMERG-CODE-001',
    'Code Blue - Cardiac Arrest Protocol',
    'ACLS-based cardiac arrest resuscitation protocol per AHA Guidelines',
    '2.0',
    'emergency',
    'resuscitation',
    'critical_care',
    '410429000',
    'http://snomed.info/sct',
    'Cardiac arrest',
    '[
        {
            "section_id": "immediate",
            "name": "Immediate Actions",
            "orders": [
                {"order_id": "CODE-001", "name": "Call code blue", "order_type": "communication", "priority": "stat", "default_selected": true},
                {"order_id": "CODE-002", "name": "Begin high-quality CPR", "order_type": "procedure", "priority": "stat", "default_selected": true, "time_critical": true},
                {"order_id": "CODE-003", "name": "Attach defibrillator/monitor", "order_type": "procedure", "priority": "stat", "default_selected": true},
                {"order_id": "CODE-004", "name": "Establish IV/IO access", "order_type": "procedure", "priority": "stat", "default_selected": true}
            ]
        },
        {
            "section_id": "medications",
            "name": "ACLS Medications",
            "orders": [
                {"order_id": "ACLS-001", "name": "Epinephrine 1mg IV/IO q3-5min", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "3992"},
                {"order_id": "ACLS-002", "name": "Amiodarone 300mg IV bolus (refractory VF/pVT)", "order_type": "medication", "priority": "stat", "default_selected": false, "rxnorm_code": "703"},
                {"order_id": "ACLS-003", "name": "Sodium bicarbonate 50mEq (if prolonged arrest)", "order_type": "medication", "priority": "stat", "default_selected": false, "rxnorm_code": "8814"},
                {"order_id": "ACLS-004", "name": "Calcium chloride 1g (if hyperkalemia)", "order_type": "medication", "priority": "stat", "default_selected": false}
            ]
        },
        {
            "section_id": "airway",
            "name": "Airway Management",
            "orders": [
                {"order_id": "AIR-001", "name": "Bag-valve-mask ventilation", "order_type": "procedure", "priority": "stat", "default_selected": true},
                {"order_id": "AIR-002", "name": "Advanced airway placement", "order_type": "procedure", "priority": "stat", "default_selected": false},
                {"order_id": "AIR-003", "name": "Capnography monitoring", "order_type": "monitoring", "priority": "stat", "default_selected": true}
            ]
        }
    ]'::jsonb,
    '[
        {"constraint_id": "CODE-TC-001", "action": "First defibrillation if shockable", "deadline_minutes": 2, "severity": "critical", "metrics_code": "TIME2SHOCK"},
        {"constraint_id": "CODE-TC-002", "action": "CPR fraction >80%", "deadline_minutes": 0, "severity": "critical", "metrics_code": "CPR_FRACTION"}
    ]'::jsonb,
    '{"evidence_level": "A", "guideline_source": "AHA ACLS Guidelines 2020", "time_critical": true, "mortality_impact": "critical"}'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
),
(
    'EMERG-ANAPH-001',
    'Anaphylaxis Emergency Protocol',
    'Anaphylaxis treatment protocol per WAO/AAAAI Guidelines',
    '2.0',
    'emergency',
    'allergy',
    'emergency_medicine',
    '39579001',
    'http://snomed.info/sct',
    'Anaphylaxis',
    '[
        {
            "section_id": "immediate",
            "name": "Immediate Treatment",
            "orders": [
                {"order_id": "ANAPH-001", "name": "Epinephrine 0.3-0.5mg IM (adult)", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "3992", "route": "IM", "time_critical": true},
                {"order_id": "ANAPH-002", "name": "Place patient supine with legs elevated", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "ANAPH-003", "name": "High-flow oxygen", "order_type": "respiratory", "priority": "stat", "default_selected": true},
                {"order_id": "ANAPH-004", "name": "Large-bore IV access x2", "order_type": "procedure", "priority": "stat", "default_selected": true}
            ]
        },
        {
            "section_id": "adjunct",
            "name": "Adjunct Medications",
            "orders": [
                {"order_id": "ADJ-001", "name": "Normal saline IV bolus 1-2L", "order_type": "medication", "priority": "stat", "default_selected": true},
                {"order_id": "ADJ-002", "name": "Diphenhydramine 50mg IV", "order_type": "medication", "priority": "stat", "default_selected": true, "rxnorm_code": "3498"},
                {"order_id": "ADJ-003", "name": "Methylprednisolone 125mg IV", "order_type": "medication", "priority": "routine", "default_selected": true, "rxnorm_code": "6902"},
                {"order_id": "ADJ-004", "name": "Albuterol nebulizer if bronchospasm", "order_type": "medication", "priority": "stat", "default_selected": false, "rxnorm_code": "435"}
            ]
        },
        {
            "section_id": "monitoring",
            "name": "Monitoring",
            "orders": [
                {"order_id": "MON-001", "name": "Continuous cardiac monitoring", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-002", "name": "Continuous pulse oximetry", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-003", "name": "BP q5min until stable", "order_type": "nursing", "priority": "stat", "default_selected": true},
                {"order_id": "MON-004", "name": "Observe minimum 4-6 hours for biphasic reaction", "order_type": "nursing", "priority": "routine", "default_selected": true}
            ]
        }
    ]'::jsonb,
    '[
        {"constraint_id": "ANAPH-TC-001", "action": "Epinephrine administered", "deadline_minutes": 5, "severity": "critical", "metrics_code": "TIME2EPI"}
    ]'::jsonb,
    '{"evidence_level": "A", "guideline_source": "WAO/AAAAI Anaphylaxis Guidelines 2020", "time_critical": true}'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
);

-- ============================================
-- CARE PLAN TEMPLATES
-- ============================================

INSERT INTO care_plan_templates (
    template_id, name, description, version, category, subcategory,
    condition_code, condition_system, condition_display,
    goals, activities, monitoring, duration, review_period, guidelines, status, created_by, approved_by, approved_at
) VALUES
(
    'CP-CHF-001',
    'Heart Failure Management Care Plan',
    'Comprehensive outpatient care plan for chronic heart failure management',
    '1.0',
    'chronic_disease',
    'cardiovascular',
    '84114007',
    'http://snomed.info/sct',
    'Heart failure',
    '[
        {"goal_id": "G001", "description": "Maintain NYHA functional class or improve by 1 class", "target": "NYHA I-II", "timeframe": "90 days"},
        {"goal_id": "G002", "description": "Reduce hospitalizations for HF exacerbation", "target": "0 hospitalizations", "timeframe": "180 days"},
        {"goal_id": "G003", "description": "Achieve optimal GDMT (ACEi/ARB/ARNI + BB + MRA)", "target": "100% GDMT", "timeframe": "90 days"},
        {"goal_id": "G004", "description": "Sodium restriction compliance", "target": "<2g/day", "timeframe": "ongoing"}
    ]'::jsonb,
    '[
        {"activity_id": "A001", "description": "Daily weight monitoring", "frequency": "daily", "responsible": "patient"},
        {"activity_id": "A002", "description": "Medication adherence check", "frequency": "weekly", "responsible": "care_team"},
        {"activity_id": "A003", "description": "Cardiology follow-up", "frequency": "every 3 months", "responsible": "cardiologist"},
        {"activity_id": "A004", "description": "BNP/NT-proBNP monitoring", "frequency": "every 3 months", "responsible": "lab"},
        {"activity_id": "A005", "description": "Diet and exercise counseling", "frequency": "monthly", "responsible": "dietitian"}
    ]'::jsonb,
    '[
        {"parameter": "Weight", "frequency": "daily", "target": "Within 3 lbs of dry weight", "alert_threshold": ">3 lb gain in 24h or >5 lbs in week"},
        {"parameter": "Blood pressure", "frequency": "daily", "target": "<130/80 mmHg", "alert_threshold": ">140/90 or <90/60"},
        {"parameter": "Heart rate", "frequency": "daily", "target": "60-100 bpm", "alert_threshold": "<50 or >120 bpm"},
        {"parameter": "Symptoms", "frequency": "daily", "target": "No worsening dyspnea/edema", "alert_threshold": "New or worsening symptoms"}
    ]'::jsonb,
    'ongoing',
    '3 months',
    '[{"guideline": "ACC/AHA Heart Failure Guidelines 2022", "evidence_level": "A"}]'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
),
(
    'CP-DM2-001',
    'Type 2 Diabetes Management Care Plan',
    'Comprehensive care plan for Type 2 Diabetes management per ADA Standards of Care',
    '1.0',
    'chronic_disease',
    'metabolic',
    '44054006',
    'http://snomed.info/sct',
    'Diabetes mellitus type 2',
    '[
        {"goal_id": "G001", "description": "Achieve glycemic control", "target": "HbA1c <7%", "timeframe": "90 days"},
        {"goal_id": "G002", "description": "Blood pressure control", "target": "<130/80 mmHg", "timeframe": "90 days"},
        {"goal_id": "G003", "description": "LDL cholesterol control", "target": "<100 mg/dL", "timeframe": "90 days"},
        {"goal_id": "G004", "description": "Complete annual diabetic exam", "target": "Eye, foot, kidney screening", "timeframe": "12 months"}
    ]'::jsonb,
    '[
        {"activity_id": "A001", "description": "Blood glucose self-monitoring", "frequency": "as prescribed", "responsible": "patient"},
        {"activity_id": "A002", "description": "HbA1c monitoring", "frequency": "every 3 months", "responsible": "lab"},
        {"activity_id": "A003", "description": "Lipid panel", "frequency": "annually", "responsible": "lab"},
        {"activity_id": "A004", "description": "Urine albumin/creatinine ratio", "frequency": "annually", "responsible": "lab"},
        {"activity_id": "A005", "description": "Dilated eye exam", "frequency": "annually", "responsible": "ophthalmology"},
        {"activity_id": "A006", "description": "Comprehensive foot exam", "frequency": "annually", "responsible": "primary_care"},
        {"activity_id": "A007", "description": "Diabetes self-management education", "frequency": "initial + annual", "responsible": "diabetes_educator"}
    ]'::jsonb,
    '[
        {"parameter": "Fasting glucose", "frequency": "daily or as prescribed", "target": "80-130 mg/dL", "alert_threshold": "<70 or >180 mg/dL"},
        {"parameter": "HbA1c", "frequency": "every 3 months", "target": "<7%", "alert_threshold": ">8%"},
        {"parameter": "Blood pressure", "frequency": "each visit", "target": "<130/80 mmHg", "alert_threshold": ">140/90 mmHg"},
        {"parameter": "Weight", "frequency": "each visit", "target": "BMI <25 or 5-7% weight loss", "alert_threshold": ">5% gain"}
    ]'::jsonb,
    'ongoing',
    '3 months',
    '[{"guideline": "ADA Standards of Medical Care in Diabetes 2024", "evidence_level": "A"}]'::jsonb,
    'active',
    'system',
    'clinical_committee',
    NOW()
);

-- ============================================
-- GRANT PERMISSIONS
-- ============================================

-- Grant all permissions to kb12_user
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO kb12_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kb12_user;

-- Verify seeded data
DO $$
DECLARE
    orderset_count INT;
    careplan_count INT;
BEGIN
    SELECT COUNT(*) INTO orderset_count FROM order_set_templates;
    SELECT COUNT(*) INTO careplan_count FROM care_plan_templates;

    RAISE NOTICE 'KB-12 Seed Data Loaded:';
    RAISE NOTICE '  - Order Set Templates: %', orderset_count;
    RAISE NOTICE '  - Care Plan Templates: %', careplan_count;

    IF orderset_count < 5 OR careplan_count < 2 THEN
        RAISE EXCEPTION 'Seed data incomplete. Expected at least 5 order sets and 2 care plans.';
    END IF;
END $$;
