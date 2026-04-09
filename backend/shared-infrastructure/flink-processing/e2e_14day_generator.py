#!/usr/bin/env python3
"""
E2E 14-Day Clinical Dataset Generator
======================================

Generates a comprehensive real-time test dataset for the Flink clinical
pipeline (Modules 7-13) with 3 patient profiles designed to exercise
every module feature including edge cases.

Patient A — Rajesh Kumar (The Deteriorator)
  Multi-domain worsening: HTN escalation, renal decline, engagement
  collapse, dual concurrent interventions. Tests M7 escalation, M8
  safety alerts (triple whammy, DKA), M9 engagement collapse, M12
  concurrent windows, M12b intervention delta, M13 cross-domain
  amplification.

Patient B — Priya Sharma (The Improver)
  Steady improvement after ARB initiation. Regular meals + exercise.
  Tests M7 improvement trajectory, M10/M10b meal patterns, M11/M11b
  fitness patterns, M12 successful intervention, M8 negative case
  (should produce ZERO alerts), M13 stable/improving detection.

Patient C — Amit Patel (Edge Case Specialist)
  Masked HTN (normal in clinic, high at home), nocturnal non-dipper,
  acute surge events, ACEi + hyperkalemia. Tests M7 masked_htn,
  dip_classification, acute_surge_flag, white_coat_suspected. Tests
  M8 potassium alert with ACEi.

Usage:
    python3 e2e_14day_generator.py --start-date 2026-04-09
    python3 e2e_14day_generator.py --start-date 2026-04-09 --day 3
    python3 e2e_14day_generator.py --start-date 2026-04-09 --output-dir ./e2e_days/

Output:
    Produces per-day JSON files with events organized by Kafka topic,
    plus an assertions file for each day.
"""

import json
import uuid
import os
import argparse
from datetime import datetime, timedelta, timezone
from typing import List, Dict, Any, Optional, Tuple

# ═══════════════════════════════════════════════════════════════════════
#  TIME CONSTANTS (IST → UTC offsets from midnight UTC)
# ═══════════════════════════════════════════════════════════════════════

# IST = UTC + 5:30. So 06:30 IST = 01:00 UTC, etc.
T_MORNING_BP     = timedelta(hours=1, minutes=0)     # 06:30 IST
T_MED_AM         = timedelta(hours=2, minutes=30)    # 08:00 IST
T_FBG_LAB        = timedelta(hours=3, minutes=0)     # 08:30 IST
T_CLINIC         = timedelta(hours=5, minutes=0)     # 10:30 IST
T_APP_SESSION_AM = timedelta(hours=5, minutes=30)    # 11:00 IST
T_LUNCH          = timedelta(hours=7, minutes=30)    # 13:00 IST
T_CGM_POST30     = timedelta(hours=8, minutes=0)     # 13:30 IST
T_CGM_POST60     = timedelta(hours=8, minutes=30)    # 14:00 IST
T_CGM_POST120    = timedelta(hours=9, minutes=30)    # 15:00 IST
T_CGM_POST180    = timedelta(hours=10, minutes=30)   # 16:00 IST
T_ACTIVITY       = timedelta(hours=11, minutes=30)   # 17:00 IST
T_ACT_HR_5       = timedelta(hours=11, minutes=35)   # 17:05 IST
T_ACT_HR_15      = timedelta(hours=11, minutes=45)   # 17:15 IST
T_ACT_HR_30      = timedelta(hours=12, minutes=0)    # 17:30 IST
T_ACT_HR_60      = timedelta(hours=12, minutes=30)   # 18:00 IST
T_ACT_BP_REC     = timedelta(hours=12, minutes=35)   # 18:05 IST
T_ACT_CGM_REC    = timedelta(hours=13, minutes=0)    # 18:30 IST
T_EVENING_BP     = timedelta(hours=13, minutes=30)   # 19:00 IST
T_MED_PM         = timedelta(hours=14, minutes=30)   # 20:00 IST
T_APP_SESSION_PM = timedelta(hours=15, minutes=0)    # 20:30 IST
T_NIGHT_BP       = timedelta(hours=17, minutes=0)    # 22:30 IST
T_STEP_COUNT     = timedelta(hours=16, minutes=30)   # 22:00 IST
T_GOAL_CHECK     = timedelta(hours=15, minutes=30)   # 21:00 IST

# Kafka topics
TOPIC_VITALS       = "ingestion.vitals"
TOPIC_ENRICHED     = "enriched-patient-events-v1"
TOPIC_INTERVENTION = "clinical.intervention-events"


# ═══════════════════════════════════════════════════════════════════════
#  PATIENT PROFILES
# ═══════════════════════════════════════════════════════════════════════

def patient_rajesh(run_id: str) -> Dict:
    return {
        "patient_id": f"e2e-rajesh-14d-{run_id}",
        "name": "Rajesh Kumar",
        "age": 58, "sex": "M",
        "conditions": ["T2DM (12yr)", "HTN Stage 2", "CKD Stage 3b (eGFR 42)"],
        "medications": [
            {"drug_name": "Metformin",           "drug_class": "BIGUANIDE", "dose_mg": 1000, "frequency": "BD"},
            {"drug_name": "Amlodipine",          "drug_class": "CCB",       "dose_mg": 10,   "frequency": "OD"},
            {"drug_name": "Telmisartan",         "drug_class": "ARB",       "dose_mg": 80,   "frequency": "OD"},
            {"drug_name": "Dapagliflozin",       "drug_class": "SGLT2I",    "dose_mg": 10,   "frequency": "OD"},
            {"drug_name": "Hydrochlorothiazide", "drug_class": "THIAZIDE",  "dose_mg": 12.5, "frequency": "OD"},
        ],
        "role": "DETERIORATOR",
    }

def patient_priya(run_id: str) -> Dict:
    return {
        "patient_id": f"e2e-priya-14d-{run_id}",
        "name": "Priya Sharma",
        "age": 45, "sex": "F",
        "conditions": ["HTN Stage 1", "Pre-diabetes (HbA1c 6.3)"],
        "medications": [
            {"drug_name": "Amlodipine", "drug_class": "CCB", "dose_mg": 5, "frequency": "OD"},
        ],
        "role": "IMPROVER",
    }

def patient_amit(run_id: str) -> Dict:
    return {
        "patient_id": f"e2e-amit-14d-{run_id}",
        "name": "Amit Patel",
        "age": 52, "sex": "M",
        "conditions": ["HTN newly diagnosed", "T2DM (2yr)"],
        "medications": [
            {"drug_name": "Enalapril",  "drug_class": "ACEI",     "dose_mg": 10,  "frequency": "OD"},
            {"drug_name": "Metformin",  "drug_class": "BIGUANIDE","dose_mg": 500, "frequency": "BD"},
            {"drug_name": "Hydrochlorothiazide", "drug_class": "THIAZIDE", "dose_mg": 25, "frequency": "OD"},
        ],
        "role": "EDGE_CASE",
    }


# ═══════════════════════════════════════════════════════════════════════
#  BP TRAJECTORIES — exact values per day per patient
# ═══════════════════════════════════════════════════════════════════════

# Each entry: (sbp, dbp, hr, time_context, source)
# Additional entries per day can include NIGHT, CLINIC, second MORNING, etc.

RAJESH_BP = {
    #          Morning home             Evening home             Optional extras
    1:  [( 160, 98, 80, "MORNING","HOME_CUFF"), (144, 88, 76, "EVENING","HOME_CUFF")],
    2:  [( 162,100, 80, "MORNING","HOME_CUFF"), (140, 84, 76, "EVENING","HOME_CUFF")],
    3:  [( 158, 96, 79, "MORNING","HOME_CUFF"), (142, 86, 76, "EVENING","HOME_CUFF"),
         (148, 90, 75, "NIGHT",  "HOME_CUFF")],
    4:  [( 164,100, 80, "MORNING","HOME_CUFF"), (146, 90, 77, "EVENING","HOME_CUFF")],
    5:  [( 170,104, 82, "MORNING","CLINIC"),    # Clinic visit — white coat test
         (155, 94, 78, "MORNING","HOME_CUFF"),  # 2h later at home — gap detection
         (148, 92, 77, "EVENING","HOME_CUFF")],
    6:  [( 165,100, 80, "MORNING","HOME_CUFF"), (148, 92, 77, "EVENING","HOME_CUFF")],
    7:  [( 170,104, 82, "MORNING","HOME_CUFF"), (146, 90, 77, "EVENING","HOME_CUFF"),
         (152, 94, 76, "NIGHT",  "HOME_CUFF")],
    8:  [( 172,104, 82, "MORNING","HOME_CUFF"), (150, 94, 78, "EVENING","HOME_CUFF")],
    9:  [( 176,106, 83, "MORNING","HOME_CUFF"), (155, 96, 79, "EVENING","HOME_CUFF"),
         (160,100, 78, "NIGHT",  "HOME_CUFF")],   # 3rd NIGHT → dip classification now possible
    10: [( 178,108, 83, "MORNING","HOME_CUFF")],  # NO evening — engagement declining
    11: [( 182,110, 84, "MORNING","HOME_CUFF"), (160,100, 80, "EVENING","HOME_CUFF")],
    12: [( 180,108, 84, "MORNING","HOME_CUFF"),
         (168,102, 80, "NIGHT",  "HOME_CUFF")],   # 4th NIGHT reading — further dip data
    13: [(164,100, 80, "EVENING","HOME_CUFF")],    # NO morning, single reluctant evening
    14: [( 174,104, 82, "MORNING","HOME_CUFF")],   # Single morning, day 14
}

PRIYA_BP = {
    1:  [( 148, 92, 78, "MORNING","HOME_CUFF"), (138, 86, 74, "EVENING","HOME_CUFF")],
    2:  [( 146, 90, 77, "MORNING","HOME_CUFF"), (136, 84, 73, "EVENING","HOME_CUFF")],
    3:  [( 144, 88, 76, "MORNING","HOME_CUFF"), (134, 82, 73, "EVENING","HOME_CUFF"),
         (128, 78, 70, "NIGHT",  "HOME_CUFF")],
    4:  [( 142, 88, 76, "MORNING","HOME_CUFF"), (132, 80, 72, "EVENING","HOME_CUFF")],
    5:  [( 140, 86, 75, "MORNING","HOME_CUFF"), (130, 80, 72, "EVENING","HOME_CUFF")],
    6:  [( 138, 84, 74, "MORNING","HOME_CUFF"), (128, 78, 71, "EVENING","HOME_CUFF"),
         (122, 76, 68, "NIGHT",  "HOME_CUFF")],
    7:  [( 136, 84, 74, "MORNING","HOME_CUFF"), (126, 78, 71, "EVENING","HOME_CUFF")],
    8:  [( 135, 82, 73, "MORNING","HOME_CUFF"), (125, 78, 70, "EVENING","HOME_CUFF")],
    9:  [( 134, 82, 73, "MORNING","HOME_CUFF"), (124, 76, 70, "EVENING","HOME_CUFF"),
         (118, 74, 68, "NIGHT",  "HOME_CUFF")],
    10: [( 132, 80, 72, "MORNING","HOME_CUFF"), (122, 76, 70, "EVENING","HOME_CUFF")],
    11: [( 130, 80, 72, "MORNING","HOME_CUFF"), (120, 76, 69, "EVENING","HOME_CUFF")],
    12: [( 128, 78, 71, "MORNING","HOME_CUFF"), (118, 74, 68, "EVENING","HOME_CUFF"),
         (114, 72, 66, "NIGHT",  "HOME_CUFF")],
    13: [( 130, 80, 72, "MORNING","HOME_CUFF"), (120, 76, 69, "EVENING","HOME_CUFF")],
    14: [( 128, 78, 71, "MORNING","HOME_CUFF"), (118, 74, 68, "EVENING","HOME_CUFF")],
}

AMIT_BP = {
    1:  [( 132, 82, 72, "MORNING","HOME_CUFF"), (128, 78, 70, "EVENING","HOME_CUFF")],
    2:  [( 134, 84, 73, "MORNING","HOME_CUFF"), (130, 80, 71, "EVENING","HOME_CUFF")],
    3:  [( 136, 86, 74, "MORNING","HOME_CUFF"), (132, 82, 72, "EVENING","HOME_CUFF"),
         (134, 84, 72, "NIGHT",  "HOME_CUFF")],  # NON-DIPPER: night ≈ morning
    4:  [( 138, 88, 75, "MORNING","HOME_CUFF"), (134, 84, 73, "EVENING","HOME_CUFF")],
    5:  [( 124, 76, 70, "MORNING","CLINIC"),    # Clinic looks GOOD → masked HTN
         (144, 92, 76, "MORNING","HOME_CUFF"),  # Home 3h later → HIGH
         (140, 88, 75, "EVENING","HOME_CUFF")],
    6:  [( 145, 92, 77, "MORNING","HOME_CUFF"), (140, 88, 75, "EVENING","HOME_CUFF"),
         (142, 90, 76, "NIGHT",  "HOME_CUFF")],  # NON-DIPPER confirmed
    7:  [( 148, 94, 78, "MORNING","HOME_CUFF"),
         (182,108, 88, "MORNING","HOME_CUFF"),  # 2h later — ACUTE SURGE after stress (delta=34, >30 threshold)
         (136, 84, 73, "EVENING","HOME_CUFF")],
    8:  [( 150, 96, 78, "MORNING","HOME_CUFF"), (142, 90, 76, "EVENING","HOME_CUFF")],
    9:  [( 152, 96, 79, "MORNING","HOME_CUFF"), (144, 90, 76, "EVENING","HOME_CUFF"),
         (148, 92, 77, "NIGHT",  "HOME_CUFF")],  # NON-DIPPER still
    10: [( 122, 76, 69, "MORNING","CLINIC"),    # 2nd clinic visit — still looks GOOD → masked HTN confirmed
         (148, 94, 78, "MORNING","HOME_CUFF"), (140, 88, 75, "EVENING","HOME_CUFF")],
    11: [( 146, 92, 77, "MORNING","HOME_CUFF"), (138, 86, 74, "EVENING","HOME_CUFF")],
    12: [( 144, 90, 76, "MORNING","HOME_CUFF"), (136, 84, 73, "EVENING","HOME_CUFF"),
         (140, 88, 75, "NIGHT",  "HOME_CUFF")],
    13: [( 142, 88, 76, "MORNING","HOME_CUFF"), (134, 82, 72, "EVENING","HOME_CUFF")],
    14: [( 140, 86, 75, "MORNING","HOME_CUFF"), (132, 80, 71, "EVENING","HOME_CUFF")],
}


# ═══════════════════════════════════════════════════════════════════════
#  ENRICHED EVENT SCHEDULES
# ═══════════════════════════════════════════════════════════════════════

# Rajesh engagement signals schedule:
#   Days 1-5: full engagement (app session, meal log, med taken, goal, step count, CGM)
#   Days 6-7: declining (shorter sessions, fewer signals)
#   Days 8-9: sparse (med taken only some days)
#   Days 10-14: near-zero (occasional single signal)

# Rajesh lab schedule:
RAJESH_LABS = {
    1:  [("EGFR", 52.0, "mL/min/1.73m2"), ("FBG", 142.0, "mg/dL")],
    3:  [("FBG", 145.0, "mg/dL")],
    5:  [("EGFR", 50.0, "mL/min/1.73m2"), ("FBG", 152.0, "mg/dL"),
         ("HBA1C", 8.2, "%"), ("POTASSIUM", 4.6, "mmol/L"), ("ACR", 45.0, "mg/g")],
    7:  [("EGFR", 48.0, "mL/min/1.73m2"), ("FBG", 158.0, "mg/dL")],
    9:  [("FBG", 168.0, "mg/dL")],
    11: [("FBG", 175.0, "mg/dL")],
    14: [("EGFR", 45.0, "mL/min/1.73m2"), ("FBG", 180.0, "mg/dL"),
         ("POTASSIUM", 4.9, "mmol/L")],
}

# Rajesh symptom schedule (triggers M8 rules):
RAJESH_SYMPTOMS = {
    4: [("NAUSEA", "mild", "today")],           # Mild nausea → CID_04 with SGLT2i
    5: [("NAUSEA", "moderate", "2_days_ago")],   # Worsening → HALT
    8: [("FATIGUE", "moderate", "few_days")],
    11:[("DIZZINESS", "moderate", "today"), ("NAUSEA", "mild", "ongoing")],
}

# Rajesh meal schedule (triggers M10):
# (meal_type, carb_g, protein_g, fat_g, sodium_mg)
RAJESH_MEALS = {
    1:  [("lunch",  55, 22, 15, 780)],
    2:  [("lunch",  60, 25, 18, 850)],
    3:  [("lunch",  50, 20, 14, 720)],
    4:  [("lunch",  65, 28, 20, 950)],
    5:  [("lunch",  70, 30, 22, 1020)],
    6:  [("lunch",  55, 22, 16, 880)],
    7:  [("lunch",  60, 24, 18, 900)],
    8:  [("lunch",  75, 30, 25, 1100)],   # Larger, unhealthier meal
    # Days 9-14: no meal logs (engagement collapse)
}

# Rajesh post-meal CGM readings (glucose values at +30, +60, +120, +180 min)
RAJESH_CGM_POSTMEAL = {
    1:  [155, 185, 210, 190],
    2:  [160, 192, 215, 195],
    3:  [150, 178, 202, 185],
    4:  [165, 198, 225, 205],
    5:  [170, 205, 232, 210],
    6:  [158, 190, 218, 198],
    7:  [162, 195, 222, 202],
    8:  [175, 215, 248, 225],   # Worst day — high carb meal
}

# Rajesh fasting CGM (single morning reading)
RAJESH_CGM_FASTING = {
    1: 142, 2: 148, 3: 145, 4: 150, 5: 155,
    6: 158, 7: 162, 8: 168,
    # Days 9-14: no CGM (stopped wearing device)
}

# Rajesh activity schedule:
# (exercise_type, duration_min)
RAJESH_ACTIVITIES = {
    2:  ("BRISK_WALKING", 25),
    4:  ("BRISK_WALKING", 20),
    6:  ("BRISK_WALKING", 15),   # Declining duration
    # Days 7+: no activity
}

# Rajesh post-activity HR readings (at +5, +15, +30, +60 min)
RAJESH_ACT_HR = {
    2:  [105, 98, 88, 80],
    4:  [108, 100, 90, 82],
    6:  [112, 104, 94, 85],   # Slower recovery
}

# Rajesh engagement signals per day:
#   (app_session_sec, goal_completed, goal_total, med_taken, steps)
RAJESH_ENGAGEMENT = {
    1:  (180, 3, 5, True, 4500),
    2:  (180, 4, 5, True, 5200),
    3:  (120, 3, 5, True, 3800),
    4:  (150, 3, 5, True, 4000),
    5:  (120, 3, 5, True, 3500),
    6:  (90,  2, 5, True, 2800),
    7:  (60,  2, 5, True, 2200),
    8:  (30,  1, 5, True, 1500),
    9:  (0,   0, 5, True, 800),     # No app session, still takes meds
    10: (0,   0, 5, False, 500),    # Stopped taking meds
    11: (0,   0, 5, False, 400),
    12: (0,   0, 5, False, 0),      # Zero activity
    13: (0,   0, 5, False, 0),
    14: (0,   0, 5, False, 0),
}

# Rajesh intervention schedule:
RAJESH_INTERVENTIONS = {
    5: {
        "intervention_id_suffix": "amlodipine-uptitrate",
        "intervention_type": "MEDICATION_DOSE_INCREASE",
        "drug_name": "Amlodipine", "drug_class": "CCB",
        "old_dose_mg": 10, "new_dose_mg": 15,
        "reason": "Uncontrolled Stage 2 HTN despite 3-drug therapy, SBP avg >155",
        "observation_window_days": 14,
        "originating_card_id_suffix": "bp-escalation",
    },
    7: {
        "intervention_id_suffix": "metformin-uptitrate",
        "intervention_type": "MEDICATION_DOSE_INCREASE",
        "drug_name": "Metformin", "drug_class": "METFORMIN",
        "old_dose_mg": 1000, "new_dose_mg": 1500,
        "reason": "FBG 158, HbA1c 8.2 — inadequate glycemic control",
        "observation_window_days": 21,
        "originating_card_id_suffix": "fbg-escalation",
    },
}


# ── Priya enriched schedules ──

PRIYA_LABS = {
    1:  [("FBG", 118.0, "mg/dL")],
    5:  [("FBG", 112.0, "mg/dL"), ("EGFR", 88.0, "mL/min/1.73m2")],
    10: [("FBG", 108.0, "mg/dL")],
    14: [("FBG", 105.0, "mg/dL"), ("EGFR", 87.0, "mL/min/1.73m2"),
         ("HBA1C", 6.1, "%")],
}

# Priya meals — consistent, healthy
PRIYA_MEALS = {d: [("lunch", 45, 28, 12, 650)] for d in range(1, 15)}

PRIYA_CGM_POSTMEAL = {
    d: [110 + d, 135 - d, 155 - d*2, 120 - d] for d in range(1, 15)
}

PRIYA_CGM_FASTING = {d: 118 - d for d in range(1, 15)}

# Priya activities — regular, every other day
PRIYA_ACTIVITIES = {d: ("BRISK_WALKING" if d % 3 != 0 else "YOGA", 30) for d in range(1, 15)}

PRIYA_ACT_HR = {d: [108 - d//2, 96 - d//2, 84 - d//3, 76] for d in range(1, 15)}

PRIYA_ENGAGEMENT = {d: (300, 4, 5, True, 5500 + d*100) for d in range(1, 15)}

PRIYA_INTERVENTIONS = {
    1: {
        "intervention_id_suffix": "telmisartan-initiate",
        "intervention_type": "MEDICATION_ADD",
        "drug_name": "Telmisartan", "drug_class": "ARB",
        "old_dose_mg": 0, "new_dose_mg": 40,
        "reason": "Stage 1 HTN uncontrolled on CCB monotherapy, adding ARB",
        "observation_window_days": 12,   # 12-day window so CLOSE fires on day 13
        "originating_card_id_suffix": "htn-add-arb",
    },
}


# ── Amit enriched schedules ──

AMIT_LABS = {
    1:  [("EGFR", 68.0, "mL/min/1.73m2"), ("FBG", 130.0, "mg/dL"),
         ("POTASSIUM", 4.4, "mmol/L")],
    5:  [("EGFR", 65.0, "mL/min/1.73m2"), ("FBG", 128.0, "mg/dL"),
         ("POTASSIUM", 5.2, "mmol/L"),    # ← ELEVATED with ACEi! Triggers M8
         ("CREATININE", 1.3, "mg/dL")],
    10: [("POTASSIUM", 5.0, "mmol/L"), ("FBG", 125.0, "mg/dL")],
    14: [("EGFR", 64.0, "mL/min/1.73m2"), ("POTASSIUM", 4.8, "mmol/L"),
         ("FBG", 122.0, "mg/dL")],
}

AMIT_ENGAGEMENT = {d: (120, 3, 5, True, 3500) for d in range(1, 15)}

# Amit has no interventions, no symptoms, minimal meals/activities
AMIT_MEALS = {d: [("lunch", 50, 25, 15, 700)] for d in [1, 3, 5, 7, 9, 11, 13]}
AMIT_CGM_POSTMEAL = {d: [125, 148, 165, 140] for d in [1, 3, 5, 7, 9, 11, 13]}
AMIT_CGM_FASTING = {d: 130 - d//2 for d in range(1, 15)}
AMIT_ACTIVITIES = {d: ("BRISK_WALKING", 20) for d in [2, 4, 8, 12]}
AMIT_ACT_HR = {d: [100, 90, 82, 76] for d in [2, 4, 8, 12]}


# ═══════════════════════════════════════════════════════════════════════
#  EVENT BUILDERS
# ═══════════════════════════════════════════════════════════════════════

def ts_ms(base_date: datetime, offset: timedelta, extra_ms: int = 0) -> int:
    """Return epoch milliseconds for base_date + offset + extra."""
    dt = base_date + offset + timedelta(milliseconds=extra_ms)
    return int(dt.timestamp() * 1000)

def make_bp(pid: str, corr_id: str, ts: int, sbp: int, dbp: int, hr: int,
            context: str, source: str) -> Dict:
    return {
        "patient_id": pid,
        "systolic": sbp, "diastolic": dbp, "heart_rate": hr,
        "timestamp": ts,
        "time_context": context,
        "source": source,
        "position": "SEATED",
        "device_type": "oscillometric_cuff",
        "correlation_id": corr_id,
    }

def make_enriched(pid: str, corr_id: str, ts: int, event_type: str,
                  payload: Dict) -> Dict:
    return {
        "eventId": str(uuid.uuid4()),
        "patientId": pid,
        "eventType": event_type,
        "timestamp": ts,
        "sourceSystem": "flink-e2e-14day",
        "correlationId": corr_id,
        "payload": payload,
        "enrichmentData": {},
        "enrichmentVersion": "1.0",
    }

def make_medication_order(pid, corr_id, ts, drug_name, drug_class, dose_mg, freq):
    return make_enriched(pid, corr_id, ts, "MEDICATION_ORDERED", {
        "drug_name": drug_name, "drug_class": drug_class,
        "dose_mg": dose_mg, "route": "oral",
        "frequency": freq, "status": "active",
    })

def make_lab(pid, corr_id, ts, lab_type, value, unit):
    test_names = {
        "EGFR": "eGFR", "FBG": "Fasting Blood Glucose", "HBA1C": "HbA1c",
        "POTASSIUM": "Serum Potassium", "ACR": "Albumin:Creatinine Ratio",
        "CREATININE": "Serum Creatinine",
    }
    return make_enriched(pid, corr_id, ts, "LAB_RESULT", {
        "lab_type": lab_type, "value": value, "unit": unit,
        "testName": test_names.get(lab_type, lab_type),
        "results": {lab_type.lower(): value},
    })

def make_vital_sign(pid, corr_id, ts, sbp, dbp, hr=None, weight=None, rhr=None):
    p = {"systolic_bp": sbp, "diastolic_bp": dbp}
    if hr: p["heart_rate"] = hr
    if weight: p["weight_kg"] = weight
    if rhr: p["resting_heart_rate"] = rhr
    return make_enriched(pid, corr_id, ts, "VITAL_SIGN", p)

def make_symptom(pid, corr_id, ts, symptom_type, severity, onset):
    return make_enriched(pid, corr_id, ts, "PATIENT_REPORTED", {
        "symptom_type": symptom_type, "severity": severity,
        "onset": onset, "status": "active",
    })

def make_meal(pid, corr_id, ts, meal_type, carbs, protein, fat, sodium):
    return make_enriched(pid, corr_id, ts, "PATIENT_REPORTED", {
        "report_type": "MEAL_LOG",
        "meal_type": meal_type,
        "carb_grams": carbs, "protein_grams": protein,
        "fat_grams": fat, "sodium_mg": sodium,
        "protein_flag": protein > 25,
        "data_tier": "TIER_1_CGM",
    })

def make_cgm(pid, corr_id, ts, glucose_value):
    return make_enriched(pid, corr_id, ts, "DEVICE_READING", {
        "glucose_value": glucose_value,
        "source": "CGM", "data_tier": "TIER_1_CGM",
    })

def make_activity(pid, corr_id, ts, exercise_type, duration_min, age, sex):
    return make_enriched(pid, corr_id, ts, "PATIENT_REPORTED", {
        "report_type": "ACTIVITY_LOG",
        "exercise_type": exercise_type,
        "duration_minutes": duration_min,
        "patient_age": age, "patient_sex": sex,
        "data_tier": "TIER_1_CGM",
    })

def make_hr_reading(pid, corr_id, ts, hr, activity_state="RESTING"):
    return make_enriched(pid, corr_id, ts, "DEVICE_READING", {
        "heart_rate": hr, "source": "WEARABLE",
        "activity_state": activity_state,
        "data_tier": "TIER_1_CGM",
    })

def make_step_count(pid, corr_id, ts, steps):
    return make_enriched(pid, corr_id, ts, "DEVICE_READING", {
        "step_count": steps, "source": "WEARABLE",
        "data_tier": "TIER_1_CGM",
    })

def make_app_session(pid, corr_id, ts, duration_sec):
    return make_enriched(pid, corr_id, ts, "PATIENT_REPORTED", {
        "report_type": "APP_SESSION",
        "session_duration_sec": duration_sec,
        "data_tier": "TIER_1_CGM",
    })

def make_goal(pid, corr_id, ts, completed, total):
    return make_enriched(pid, corr_id, ts, "PATIENT_REPORTED", {
        "report_type": "GOAL_COMPLETED",
        "fields_completed": completed, "total_fields": total,
        "data_tier": "TIER_1_CGM",
    })

def make_med_taken(pid, corr_id, ts, drug_name, scheduled_ts):
    return make_enriched(pid, corr_id, ts, "MEDICATION_EVENT", {
        "drug_name": drug_name, "action": "TAKEN",
        "scheduled_time": scheduled_ts, "actual_time": ts,
        "data_tier": "TIER_1_CGM",
    })

def make_intervention(pid, corr_id, ts, run_id, spec):
    intv_id = f"intv-{run_id}-{spec['intervention_id_suffix']}"
    card_id = f"card-{run_id}-{spec['originating_card_id_suffix']}"
    return make_enriched(pid, corr_id, ts, "INTERVENTION_APPROVED", {
        "event_type": "INTERVENTION_APPROVED",
        "intervention_id": intv_id,
        "intervention_type": spec["intervention_type"],
        "intervention_detail": {
            "drug_name": spec["drug_name"], "drug_class": spec["drug_class"],
            "old_dose_mg": spec["old_dose_mg"], "new_dose_mg": spec["new_dose_mg"],
            "reason": spec["reason"],
        },
        "observation_window_days": spec["observation_window_days"],
        "originating_card_id": card_id,
        "physician_action": "APPROVE_WITH_MONITORING",
    })


# ═══════════════════════════════════════════════════════════════════════
#  DAY GENERATOR — builds all events for one patient for one day
# ═══════════════════════════════════════════════════════════════════════

def generate_patient_day(day: int, patient: Dict, base_date: datetime,
                         run_id: str,
                         bp_traj: Dict, labs: Dict, symptoms: Dict,
                         meals: Dict, cgm_postmeal: Dict, cgm_fasting: Dict,
                         activities: Dict, act_hr: Dict,
                         engagement: Dict, interventions: Dict) -> Dict:
    """Returns {"vitals": [...], "enriched": [...], "interventions": [...]}"""

    pid = patient["patient_id"]
    corr_id = f"e2e-14day-{run_id}-{pid.split('-')[1]}"  # e2e-14day-xxx-rajesh
    day_base = base_date + timedelta(days=day - 1)

    vitals = []
    enriched = []
    intv_events = []

    # ── BP Readings ──
    bp_list = bp_traj.get(day, [])
    for idx, (sbp, dbp, hr, ctx, src) in enumerate(bp_list):
        if ctx == "MORNING" and src == "CLINIC":
            offset = T_CLINIC
        elif ctx == "MORNING" and src == "HOME_CUFF" and idx > 0:
            # Second morning reading within 45 min of first — required for
            # Module7 acute surge detection (isAcuteSurge needs < 1 hour gap)
            offset = T_MORNING_BP + timedelta(minutes=45)
        elif ctx == "MORNING":
            offset = T_MORNING_BP
        elif ctx == "EVENING":
            offset = T_EVENING_BP
        elif ctx == "NIGHT":
            offset = T_NIGHT_BP
        else:
            offset = T_MORNING_BP + timedelta(minutes=idx * 30)

        ts = ts_ms(day_base, offset, extra_ms=idx * 100)
        vitals.append(make_bp(pid, corr_id, ts, sbp, dbp, hr, ctx, src))

        # Also emit as VITAL_SIGN enriched event for M8/M13
        enriched.append(make_vital_sign(pid, corr_id, ts + 50, sbp, dbp, hr))

    # ── Medication Orders (day 1 only — initial state setup) ──
    if day == 1:
        for idx, med in enumerate(patient["medications"]):
            ts = ts_ms(day_base, timedelta(minutes=idx), extra_ms=0)
            enriched.append(make_medication_order(
                pid, corr_id, ts,
                med["drug_name"], med["drug_class"], med["dose_mg"], med["frequency"]))

    # ── Labs ──
    for lab_type, value, unit in labs.get(day, []):
        ts = ts_ms(day_base, T_FBG_LAB, extra_ms=labs.get(day, []).index((lab_type, value, unit)) * 500)
        enriched.append(make_lab(pid, corr_id, ts, lab_type, value, unit))

    # ── Symptoms ──
    for symptom_type, severity, onset in symptoms.get(day, []):
        ts = ts_ms(day_base, T_APP_SESSION_AM, extra_ms=500)
        enriched.append(make_symptom(pid, corr_id, ts, symptom_type, severity, onset))

    # ── Meal Log → triggers M10 session window ──
    for meal_type, carbs, protein, fat, sodium in meals.get(day, []):
        ts = ts_ms(day_base, T_LUNCH)
        enriched.append(make_meal(pid, corr_id, ts, meal_type, carbs, protein, fat, sodium))

    # ── Post-meal CGM → collected by M10 within 3h window ──
    cgm_offsets = [T_CGM_POST30, T_CGM_POST60, T_CGM_POST120, T_CGM_POST180]
    for glucose_list in [cgm_postmeal.get(day, [])]:
        for idx, glucose in enumerate(glucose_list):
            if idx < len(cgm_offsets):
                ts = ts_ms(day_base, cgm_offsets[idx])
                enriched.append(make_cgm(pid, corr_id, ts, glucose))

    # ── Fasting CGM ──
    if day in cgm_fasting:
        ts = ts_ms(day_base, T_FBG_LAB, extra_ms=200)
        enriched.append(make_cgm(pid, corr_id, ts, cgm_fasting[day]))

    # ── Activity → triggers M11 session window ──
    if day in activities:
        ex_type, duration = activities[day]
        ts = ts_ms(day_base, T_ACTIVITY)
        enriched.append(make_activity(pid, corr_id, ts, ex_type, duration,
                                       patient["age"], patient["sex"]))
        # Post-activity wearable HR readings
        hr_offsets = [T_ACT_HR_5, T_ACT_HR_15, T_ACT_HR_30, T_ACT_HR_60]
        for idx, hr_val in enumerate(act_hr.get(day, [])):
            if idx < len(hr_offsets):
                hr_ts = ts_ms(day_base, hr_offsets[idx])
                enriched.append(make_hr_reading(pid, corr_id, hr_ts, hr_val,
                                                 "ACTIVE" if idx == 0 else "RECOVERY"))
        # Post-activity BP (recovery)
        if bp_list:  # Use last BP as reference
            last_sbp = bp_list[-1][0]
            rec_ts = ts_ms(day_base, T_ACT_BP_REC)
            enriched.append(make_vital_sign(pid, corr_id, rec_ts,
                                             last_sbp + 8, bp_list[-1][1] + 4, 88))
        # Post-activity CGM
        if day in cgm_fasting:
            rec_cgm_ts = ts_ms(day_base, T_ACT_CGM_REC)
            enriched.append(make_cgm(pid, corr_id, rec_cgm_ts, cgm_fasting[day] - 12))

    # ── Engagement signals ──
    if day in engagement:
        app_sec, goal_done, goal_total, med_taken, steps = engagement[day]

        if app_sec > 0:
            ts = ts_ms(day_base, T_APP_SESSION_PM)
            enriched.append(make_app_session(pid, corr_id, ts, app_sec))

        if goal_done > 0:
            ts = ts_ms(day_base, T_GOAL_CHECK)
            enriched.append(make_goal(pid, corr_id, ts, goal_done, goal_total))

        if med_taken and patient["medications"]:
            drug = patient["medications"][0]["drug_name"]
            sched_ts = ts_ms(day_base, T_MED_AM)
            actual_ts = ts_ms(day_base, T_MED_AM, extra_ms=300)
            enriched.append(make_med_taken(pid, corr_id, actual_ts, drug, sched_ts))
            # Evening dose for BD drugs
            if patient["medications"][0].get("frequency") == "BD":
                sched_ts2 = ts_ms(day_base, T_MED_PM)
                actual_ts2 = ts_ms(day_base, T_MED_PM, extra_ms=200)
                enriched.append(make_med_taken(pid, corr_id, actual_ts2, drug, sched_ts2))

        if steps > 0:
            ts = ts_ms(day_base, T_STEP_COUNT)
            enriched.append(make_step_count(pid, corr_id, ts, steps))

    # ── Resting HR (daily, if engaging) ──
    if day in engagement and engagement[day][0] > 0:
        rhr = 78 if patient["role"] == "DETERIORATOR" else 72
        ts = ts_ms(day_base, T_APP_SESSION_AM, extra_ms=100)
        enriched.append(make_hr_reading(pid, corr_id, ts, rhr, "RESTING"))

    # ── Interventions ──
    if day in interventions:
        spec = interventions[day]
        ts = ts_ms(day_base, T_CLINIC, extra_ms=30_000)
        event = make_intervention(pid, corr_id, ts, run_id, spec)
        intv_events.append(event)
        enriched.append(event)  # Also goes to enriched topic

    return {"vitals": vitals, "enriched": enriched, "interventions": intv_events}


# ═══════════════════════════════════════════════════════════════════════
#  ASSERTION GENERATOR — what to expect after each day's injection
# ═══════════════════════════════════════════════════════════════════════

def generate_assertions(day: int, cumulative_bp_counts: Dict,
                         patient_profiles: List[Dict]) -> Dict:
    """Generate expected assertions for a given day."""
    a = {
        "day": day,
        "date_offset": f"START + {day - 1} days",
        "modules": {},
    }

    # ── M7 ──
    m7 = {"check_at": "immediately after injection"}
    total_bp = sum(cumulative_bp_counts.values())
    m7["expected_total_outputs_cumulative"] = total_bp
    m7["checks"] = []

    if day >= 3:
        m7["checks"].append("variability_classification_7d should not be INSUFFICIENT_DATA for patients with 4+ readings")
    if day >= 5:
        m7["checks"].append("Rajesh: clinic_home_gap_sbp should be non-null (day 5 clinic visit)")
        m7["checks"].append("Amit: masked_htn_suspected should be TRUE (day 5 clinic 126 vs home 142)")
    if day >= 7:
        m7["checks"].append("Rajesh: morning_surge_7d_avg should be computable")
        m7["checks"].append("Priya: variability_classification should be NORMAL or LOW (improving)")
    if day >= 11:
        m7["checks"].append("Rajesh: crisis_flag TRUE for SBP >= 180")
        m7["checks"].append("Rajesh: variability_classification_7d should be HIGH")
    if day >= 3:
        m7["checks"].append("All patients with NIGHT readings: dip_ratio should be non-null")
        m7["checks"].append("Amit: dip_classification should show NON_DIPPER pattern")

    a["modules"]["M7"] = m7

    # ── M8 ──
    m8 = {"check_at": "immediately after injection"}
    m8["checks"] = []
    if day == 1:
        m8["checks"].append("Rajesh: CID_01 (triple whammy: ARB+SGLT2i+Diuretic) should fire on med orders")
    if day >= 4:
        m8["checks"].append("Rajesh: CID_04 (euglycemic DKA: SGLT2i + nausea) should fire")
    if day >= 5:
        m8["checks"].append("Rajesh: CID_03 (Metformin + eGFR 50) should fire or update severity")
        m8["checks"].append("Amit: Hyperkalemia alert (K=5.2 + ACEi) should fire")
    if day >= 14:
        m8["checks"].append("Rajesh: eGFR 45 should trigger critical Metformin threshold alert")
    m8["checks"].append("Priya: ZERO alerts expected (no triple whammy, no symptoms, no eGFR concern)")
    m8["checks"].append("Verify deduplication: unique suppressionKeys == unique alerts")
    a["modules"]["M8"] = m8

    # ── M9 ──
    m9 = {"check_at": "after 23:59 UTC (05:29 IST next morning)"}
    m9["checks"] = []
    if day >= 1:
        m9["checks"].append("Should fire daily at 23:59 UTC with engagement composite")
    if day >= 7:
        m9["checks"].append("Rajesh: engagement score should be declining (compare day 1 vs day 7)")
    if day >= 10:
        m9["checks"].append("Rajesh: engagement composite should be < 0.30 (RED)")
        m9["checks"].append("Rajesh: MEASUREMENT_AVOIDANT phenotype should be detected")
    m9["checks"].append("Priya: engagement should remain > 0.70 (GREEN)")
    a["modules"]["M9"] = m9

    # ── M10 ──
    m10 = {"check_at": f"~3h05m after lunch injection (meal event timestamp + 3h05m)"}
    m10["checks"] = []
    if day <= 8:
        m10["checks"].append("Rajesh: meal response should fire with post-prandial CGM curve")
        m10["checks"].append("Priya: meal response should fire with post-prandial CGM curve")
    if day >= 9:
        m10["checks"].append("Rajesh: NO meal response (no meal logs after day 8)")
    a["modules"]["M10"] = m10

    # ── M10b ──
    m10b = {"check_at": "Monday 00:00 UTC"}
    m10b["checks"] = []
    if day >= 7:
        m10b["checks"].append("Weekly meal pattern aggregation should fire on first Monday after M10 output exists")
        m10b["checks"].append("Priya: consistent pattern, Rajesh: worsening sodium trend")
    a["modules"]["M10b"] = m10b

    # ── M11 ──
    m11 = {"check_at": f"~2h05m after activity injection (activity_end + 2h05m)"}
    m11["checks"] = []
    if day in [2, 4, 6]:
        m11["checks"].append("Rajesh: activity response with HR recovery curve")
    m11["checks"].append("Priya: activity response on active days")
    if day >= 7:
        m11["checks"].append("Rajesh: NO activity response (no activities after day 6)")
    a["modules"]["M11"] = m11

    # ── M11b ──
    m11b = {"check_at": "Monday 00:00 UTC"}
    m11b["checks"] = []
    if day >= 7:
        m11b["checks"].append("Weekly fitness pattern should fire on first Monday after M11 output exists")
    a["modules"]["M11b"] = m11b

    # ── M12 ──
    m12 = {"check_at": "immediately after intervention injection"}
    m12["checks"] = []
    if day == 5:
        m12["checks"].append("Rajesh: WINDOW_OPENED for Amlodipine uptitration (14-day window)")
    if day == 7:
        m12["checks"].append("Rajesh: WINDOW_OPENED for Metformin uptitration (21-day window)")
        m12["checks"].append("Rajesh: concurrent_intervention_count should be 1 for Metformin (Amlodipine is concurrent)")
        m12["checks"].append("CHECK: Does Amlodipine window retroactively update concurrent count?")
    if day == 1:
        m12["checks"].append("Priya: WINDOW_OPENED for Telmisartan addition (12-day window)")
    if day == 12:
        m12["checks"].append("Rajesh: Amlodipine MIDPOINT timer should fire (day 5 + 7 = day 12)")
    if day == 13:
        m12["checks"].append("Priya: Telmisartan WINDOW_CLOSED should fire (day 1 + 12 + 1d grace = day 14)")
    a["modules"]["M12"] = m12

    # ── M12b ──
    m12b = {"check_at": "after M12 emits WINDOW_CLOSED"}
    m12b["checks"] = []
    if day >= 13:
        m12b["checks"].append("Priya: intervention delta should fire after Telmisartan window closes")
        m12b["checks"].append("Priya: delta should show BP improvement (148/92 → ~128/78)")
    a["modules"]["M12b"] = m12b

    # ── M13 ──
    m13 = {"check_at": "after upstream module outputs arrive (may have timer-based snapshot rotation)"}
    m13["checks"] = []
    if day >= 1:
        m13["checks"].append("CKM_RISK_ESCALATION or state change event should fire after sufficient data")
    if day >= 7:
        m13["checks"].append("Rajesh: RENAL velocity > 0 (eGFR declining 52→50→48)")
        m13["checks"].append("Rajesh: CARDIOVASCULAR velocity > 0 (if ARV bug is fixed)")
        m13["checks"].append("Rajesh: data_completeness should improve from day 1")
    if day >= 10:
        m13["checks"].append("Rajesh: composite_classification should be DETERIORATING")
        m13["checks"].append("Rajesh: cross_domain_amplification should be TRUE")
        m13["checks"].append("Rajesh: domains_deteriorating >= 2")
    if day >= 14:
        m13["checks"].append("Priya: composite should show STABLE or IMPROVING")
        m13["checks"].append("Rajesh: confidence_score should be > 0.5 by now")
    a["modules"]["M13"] = m13

    return a


# ═══════════════════════════════════════════════════════════════════════
#  MAIN GENERATOR
# ═══════════════════════════════════════════════════════════════════════

def generate_full_dataset(start_date_str: str, specific_day: int = None,
                           output_dir: str = ".") -> None:
    start_date = datetime.strptime(start_date_str, "%Y-%m-%d").replace(
        tzinfo=timezone.utc)
    run_id = str(int(start_date.timestamp()))

    patients = {
        "rajesh": patient_rajesh(run_id),
        "priya":  patient_priya(run_id),
        "amit":   patient_amit(run_id),
    }

    patient_configs = {
        "rajesh": {
            "bp": RAJESH_BP, "labs": RAJESH_LABS, "symptoms": RAJESH_SYMPTOMS,
            "meals": RAJESH_MEALS, "cgm_postmeal": RAJESH_CGM_POSTMEAL,
            "cgm_fasting": RAJESH_CGM_FASTING, "activities": RAJESH_ACTIVITIES,
            "act_hr": RAJESH_ACT_HR, "engagement": RAJESH_ENGAGEMENT,
            "interventions": RAJESH_INTERVENTIONS,
        },
        "priya": {
            "bp": PRIYA_BP, "labs": PRIYA_LABS, "symptoms": {},
            "meals": PRIYA_MEALS, "cgm_postmeal": PRIYA_CGM_POSTMEAL,
            "cgm_fasting": PRIYA_CGM_FASTING, "activities": PRIYA_ACTIVITIES,
            "act_hr": PRIYA_ACT_HR, "engagement": PRIYA_ENGAGEMENT,
            "interventions": PRIYA_INTERVENTIONS,
        },
        "amit": {
            "bp": AMIT_BP, "labs": AMIT_LABS, "symptoms": {},
            "meals": AMIT_MEALS, "cgm_postmeal": AMIT_CGM_POSTMEAL,
            "cgm_fasting": AMIT_CGM_FASTING, "activities": AMIT_ACTIVITIES,
            "act_hr": AMIT_ACT_HR, "engagement": AMIT_ENGAGEMENT,
            "interventions": {},
        },
    }

    os.makedirs(output_dir, exist_ok=True)

    # Track cumulative BP counts for assertions
    cumulative_bp = {"rajesh": 0, "priya": 0, "amit": 0}

    days_to_generate = range(1, 15) if specific_day is None else [specific_day]

    # ── Metadata ──
    metadata = {
        "test_run": f"e2e-14day-{run_id}",
        "start_date": start_date_str,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "patients": {name: p for name, p in patients.items()},
        "pipeline": "M7→M8→M9→M10→M10b→M11→M11b→M12→M12b→M13",
        "notes": [
            "Inject each day's events on the actual calendar day",
            "Check assertions after injection + timer windows",
            "M9 fires at 23:59 UTC daily — check next morning",
            "M10 fires ~3h05m after meal event — check same afternoon",
            "M11 fires ~2h05m after activity end — check same evening",
            "M10b/M11b fire Monday 00:00 UTC — check Tuesday",
            "M12 midpoint/close timers fire based on observation_window_days",
        ],
    }

    all_days = []

    for day in days_to_generate:
        day_date = start_date + timedelta(days=day - 1)
        day_output = {
            "day": day,
            "date": day_date.strftime("%Y-%m-%d"),
            "date_ist": (day_date + timedelta(hours=5, minutes=30)).strftime("%Y-%m-%d"),
            "inject": {
                TOPIC_VITALS: [],
                TOPIC_ENRICHED: [],
                TOPIC_INTERVENTION: [],
            },
            "event_count": 0,
        }

        for name, profile in patients.items():
            cfg = patient_configs[name]
            result = generate_patient_day(
                day=day, patient=profile, base_date=start_date,
                run_id=run_id,
                bp_traj=cfg["bp"], labs=cfg["labs"], symptoms=cfg["symptoms"],
                meals=cfg["meals"], cgm_postmeal=cfg["cgm_postmeal"],
                cgm_fasting=cfg["cgm_fasting"], activities=cfg["activities"],
                act_hr=cfg["act_hr"], engagement=cfg["engagement"],
                interventions=cfg["interventions"],
            )

            day_output["inject"][TOPIC_VITALS].extend(result["vitals"])
            day_output["inject"][TOPIC_ENRICHED].extend(result["enriched"])
            day_output["inject"][TOPIC_INTERVENTION].extend(result["interventions"])
            cumulative_bp[name] += len(result["vitals"])

        day_output["event_count"] = sum(len(v) for v in day_output["inject"].values())
        day_output["assertions"] = generate_assertions(day, cumulative_bp, list(patients.values()))

        all_days.append(day_output)

        # Write per-day file
        day_file = os.path.join(output_dir, f"day{day:02d}.json")
        with open(day_file, "w") as f:
            json.dump(day_output, f, indent=2)
        print(f"  Day {day:2d} ({day_output['date']}): {day_output['event_count']:3d} events → {day_file}")

    # Write consolidated file
    consolidated = {"metadata": metadata, "days": all_days}
    consolidated_file = os.path.join(output_dir, "e2e_14day_full.json")
    with open(consolidated_file, "w") as f:
        json.dump(consolidated, f, indent=2)

    # Write summary
    total_events = sum(d["event_count"] for d in all_days)
    total_vitals = sum(len(d["inject"][TOPIC_VITALS]) for d in all_days)
    total_enriched = sum(len(d["inject"][TOPIC_ENRICHED]) for d in all_days)
    total_interventions = sum(len(d["inject"][TOPIC_INTERVENTION]) for d in all_days)

    print(f"\n{'═' * 60}")
    print(f"  GENERATED: {total_events} total events over {len(all_days)} days")
    print(f"  {TOPIC_VITALS}: {total_vitals} BP readings")
    print(f"  {TOPIC_ENRICHED}: {total_enriched} enriched events")
    print(f"  {TOPIC_INTERVENTION}: {total_interventions} intervention events")
    print(f"  Consolidated: {consolidated_file}")
    print(f"{'═' * 60}")

    # Write injection helper script
    write_injector_script(output_dir, start_date_str)


def write_injector_script(output_dir: str, start_date: str):
    """Write a bash helper script for daily injection."""
    script = f"""#!/bin/bash
# E2E 14-Day Daily Injector
# Usage: ./inject_day.sh <day_number> <kafka_bootstrap_servers>
#
# Example: ./inject_day.sh 1 localhost:9092

DAY=$1
BOOTSTRAP=${{2:-localhost:9092}}
DIR="{output_dir}"

if [ -z "$DAY" ]; then
    echo "Usage: $0 <day_number> [kafka_bootstrap_servers]"
    exit 1
fi

FILE="$DIR/day$(printf '%02d' $DAY).json"
if [ ! -f "$FILE" ]; then
    echo "ERROR: $FILE not found"
    exit 1
fi

echo "═══════════════════════════════════════════════════════"
echo "  Injecting Day $DAY events from $FILE"
echo "  Kafka: $BOOTSTRAP"
echo "═══════════════════════════════════════════════════════"

# Extract and inject vitals
echo "[1/3] Injecting BP readings to ingestion.vitals..."
python3 -c "
import json, subprocess, sys
with open('$FILE') as f:
    data = json.load(f)
for event in data['inject']['ingestion.vitals']:
    key = event['patient_id']
    value = json.dumps(event)
    print(f'{{key}}|{{value}}')
" | kafka-console-producer.sh \\
    --bootstrap-server $BOOTSTRAP \\
    --topic ingestion.vitals \\
    --property parse.key=true \\
    --property key.separator='|'

# Extract and inject enriched events
echo "[2/3] Injecting enriched events to enriched-patient-events-v1..."
python3 -c "
import json
with open('$FILE') as f:
    data = json.load(f)
for event in data['inject']['enriched-patient-events-v1']:
    key = event['patientId']
    value = json.dumps(event)
    print(f'{{key}}|{{value}}')
" | kafka-console-producer.sh \\
    --bootstrap-server $BOOTSTRAP \\
    --topic enriched-patient-events-v1 \\
    --property parse.key=true \\
    --property key.separator='|'

# Extract and inject interventions
echo "[3/3] Injecting interventions to clinical.intervention-events..."
python3 -c "
import json
with open('$FILE') as f:
    data = json.load(f)
for event in data['inject']['clinical.intervention-events']:
    key = event['patientId']
    value = json.dumps(event)
    print(f'{{key}}|{{value}}')
" | kafka-console-producer.sh \\
    --bootstrap-server $BOOTSTRAP \\
    --topic clinical.intervention-events \\
    --property parse.key=true \\
    --property key.separator='|'

EVENT_COUNT=$(python3 -c "import json; d=json.load(open('$FILE')); print(sum(len(v) for v in d['inject'].values()))")
echo ""
echo "✓ Injected $EVENT_COUNT events for Day $DAY"
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  ASSERTIONS TO CHECK (see playbook for full list):"
echo "═══════════════════════════════════════════════════════"
python3 -c "
import json
with open('$FILE') as f:
    data = json.load(f)
for module, info in data['assertions']['modules'].items():
    checks = info.get('checks', [])
    if checks:
        check_at = info.get('check_at', 'immediately')
        print(f'  {{module}} (check: {{check_at}}):')
        for c in checks:
            print(f'    □ {{c}}')
        print()
"
"""
    script_path = os.path.join(output_dir, "inject_day.sh")
    with open(script_path, "w") as f:
        f.write(script)
    os.chmod(script_path, 0o755)
    print(f"  Injector script: {script_path}")


# ═══════════════════════════════════════════════════════════════════════
#  CLI
# ═══════════════════════════════════════════════════════════════════════

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="E2E 14-Day Clinical Dataset Generator")
    parser.add_argument("--start-date", default="2026-04-09",
                        help="Start date YYYY-MM-DD (default: 2026-04-09)")
    parser.add_argument("--day", type=int, default=None,
                        help="Generate only a specific day (1-14)")
    parser.add_argument("--output-dir", default="./e2e_14day_dataset",
                        help="Output directory (default: ./e2e_14day_dataset)")
    args = parser.parse_args()

    print("╔══════════════════════════════════════════════════════════════╗")
    print("║     E2E 14-Day Clinical Dataset Generator                   ║")
    print("║     3 Patients × 14 Days × All Modules (M7-M13)            ║")
    print("╚══════════════════════════════════════════════════════════════╝\n")
    print(f"  Start date:  {args.start_date}")
    print(f"  Output dir:  {args.output_dir}")
    if args.day:
        print(f"  Day:         {args.day}")
    print()

    generate_full_dataset(args.start_date, args.day, args.output_dir)
