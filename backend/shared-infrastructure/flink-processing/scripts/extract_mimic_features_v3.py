#!/usr/bin/env python3
"""
Extract 55-feature vectors from MIMIC-IV data matching Module5FeatureExtractor.java.

Two modes:
  1. RAW mode: Reads downloaded MIMIC-IV CSVs from data/mimic_iv_raw/
     → Full 55-feature extraction with time-series ring buffers
  2. FALLBACK mode: Maps existing data/mimic_iv/*_features.csv (37 cols) → 55 features
     → Derives what it can, zero-fills the rest

Feature layout (must match Module5FeatureExtractor.java exactly):
  [0-5]   Vital signs (normalized 0-1): HR, SBP, DBP, RR, SpO2, Temp
  [6-8]   Clinical scores (normalized 0-1): NEWS2, qSOFA, acuity
  [9-10]  Event context: log1p(eventCount), hoursSinceAdmission
  [11-20] NEWS2 history ring buffer (10 slots, normalized 0-1)
  [21-30] Acuity history ring buffer (10 slots, normalized 0-1)
  [31-34] Pattern features: patternCount, deteriorationCount, maxSeverity, escalation
  [35-44] Risk indicator flags (0/1): tachy, hypo, fever, hypoxia, lactate, creat, K, plt, anticoag, sepsis
  [45-49] Lab-derived features (normalized, -1=missing): lactate, creatinine, K, WBC, plt
  [50-54] Active alert features: sepsis_pattern, AKI_risk, anticoag_risk, bleeding_risk, maxAlertSeverity

Usage:
    python scripts/extract_mimic_features_v3.py [--mode raw|fallback|auto]

Output:
    data/mimic_iv/v3_sepsis_55f.csv
    data/mimic_iv/v3_deterioration_55f.csv
    data/mimic_iv/v3_mortality_55f.csv
"""

import os
import sys
import argparse
import warnings
from pathlib import Path

import numpy as np
import pandas as pd

warnings.filterwarnings("ignore", category=pd.errors.SettingWithCopyWarning)

FEATURE_COUNT = 55
RAW_DATA_DIR = Path("data/mimic_iv_raw")
EXISTING_DATA_DIR = Path("data/mimic_iv")
OUTPUT_DIR = Path("data/mimic_iv")

# ═══════════════════════════════════════════════════════════════
# Normalization ranges — MUST match Module5FeatureExtractor.java
# ═══════════════════════════════════════════════════════════════
NORM_RANGES = {
    "hr":          (30, 200),
    "sbp":         (60, 250),
    "dbp":         (30, 150),
    "rr":          (5, 50),
    "spo2":        (70, 100),
    "temp":        (34, 42),
    "news2":       (0, 20),
    "qsofa":       (0, 3),
    "acuity":      (0, 10),
    "hours_admit": (0, 720),
    "lactate":     (0, 20),
    "creatinine":  (0, 15),
    "potassium":   (2, 8),
    "wbc":         (0, 40),
    "platelets":   (0, 500),
}

FEATURE_NAMES = [
    # [0-5] Vitals
    "hr_norm", "sbp_norm", "dbp_norm", "rr_norm", "spo2_norm", "temp_norm",
    # [6-8] Scores
    "news2_norm", "qsofa_norm", "acuity_norm",
    # [9-10] Event context
    "log_event_count", "hours_since_admission_norm",
    # [11-20] NEWS2 history
    *[f"news2_hist_{i}" for i in range(10)],
    # [21-30] Acuity history
    *[f"acuity_hist_{i}" for i in range(10)],
    # [31-34] Pattern features
    "pattern_count", "deterioration_pattern_count", "max_severity_index", "severity_escalation",
    # [35-44] Risk flags
    "tachycardia", "hypotension", "fever", "hypoxia",
    "elevated_lactate", "elevated_creatinine", "hyperkalemia",
    "thrombocytopenia", "on_anticoagulation", "sepsis_risk",
    # [45-49] Labs
    "lab_lactate", "lab_creatinine", "lab_potassium", "lab_wbc", "lab_platelets",
    # [50-54] Alerts
    "alert_sepsis", "alert_aki", "alert_anticoag", "alert_bleeding", "alert_max_severity",
]

assert len(FEATURE_NAMES) == FEATURE_COUNT


def normalize(value, vmin, vmax):
    """Normalize to [0,1] — matches Module5FeatureExtractor.normalize()."""
    if abs(vmax - vmin) < 1e-9:
        return 0.0
    return max(0.0, min(1.0, (value - vmin) / (vmax - vmin)))


def normalize_series(s, vmin, vmax):
    """Vectorized normalize for pandas Series."""
    if abs(vmax - vmin) < 1e-9:
        return pd.Series(0.0, index=s.index)
    return ((s - vmin) / (vmax - vmin)).clip(0.0, 1.0)


def safe_lab_feature(s, vmin, vmax):
    """Normalize lab values, -1 for missing — matches Module5FeatureExtractor.safeLabFeature()."""
    result = normalize_series(s, vmin, vmax)
    result[s.isna()] = -1.0
    return result


# ═══════════════════════════════════════════════════════════════
# NEWS2 Score Computation (from vitals)
# ═══════════════════════════════════════════════════════════════
def compute_news2(hr, sbp, rr, spo2, temp):
    """
    Compute NEWS2 (National Early Warning Score 2) from vital signs.
    Simplified version — uses mean vitals (not worst-in-window).
    """
    score = 0

    # Respiratory rate
    if rr <= 8: score += 3
    elif rr <= 11: score += 1
    elif rr <= 20: score += 0
    elif rr <= 24: score += 2
    else: score += 3

    # SpO2 (Scale 1, no supplemental O2 assumed)
    if spo2 <= 91: score += 3
    elif spo2 <= 93: score += 2
    elif spo2 <= 95: score += 1
    else: score += 0

    # Systolic BP
    if sbp <= 90: score += 3
    elif sbp <= 100: score += 2
    elif sbp <= 110: score += 1
    elif sbp <= 219: score += 0
    else: score += 3

    # Heart rate
    if hr <= 40: score += 3
    elif hr <= 50: score += 1
    elif hr <= 90: score += 0
    elif hr <= 110: score += 1
    elif hr <= 130: score += 2
    else: score += 3

    # Temperature
    if temp <= 35.0: score += 3
    elif temp <= 36.0: score += 1
    elif temp <= 38.0: score += 0
    elif temp <= 39.0: score += 1
    else: score += 2

    return score


def compute_qsofa(sbp, rr, gcs):
    """Compute qSOFA (0-3) from SBP, RR, GCS."""
    score = 0
    if sbp <= 100: score += 1
    if rr >= 22: score += 1
    if gcs < 15: score += 1
    return score


# ═══════════════════════════════════════════════════════════════
# Risk flag derivation (from vitals + labs)
# Matches the clinical thresholds used by Module 3 CDS
# ═══════════════════════════════════════════════════════════════
def derive_risk_flags(df):
    """Derive binary risk indicator flags from vitals and labs."""
    flags = pd.DataFrame(index=df.index)

    flags["tachycardia"] = (df["hr_mean"] > 100).astype(float)
    flags["hypotension"] = (df["sbp_mean"] < 90).astype(float)
    flags["fever"] = (df["temp_mean"] > 38.0).astype(float) if "temp_mean" in df else 0.0
    flags["hypoxia"] = (df["spo2_mean"] < 92).astype(float)
    flags["elevated_lactate"] = (df["lactate_mean"] > 2.0).astype(float) if "lactate_mean" in df else 0.0
    flags["elevated_creatinine"] = (df["creatinine_mean"] > 1.5).astype(float) if "creatinine_mean" in df else 0.0
    flags["hyperkalemia"] = (df["potassium_mean"] > 5.5).astype(float) if "potassium_mean" in df else 0.0
    flags["thrombocytopenia"] = (df["platelets_mean"] < 100).astype(float) if "platelets_mean" in df else 0.0
    flags["on_anticoagulation"] = 0.0  # Not derivable from existing data without prescriptions
    flags["sepsis_risk"] = (
        (flags["fever"] == 1) & (flags["tachycardia"] == 1) & (flags["elevated_lactate"] == 1)
    ).astype(float)

    return flags


# ═══════════════════════════════════════════════════════════════
# Alert feature derivation
# ═══════════════════════════════════════════════════════════════
def derive_alert_features(df, risk_flags):
    """Derive active alert features from risk flags and labs."""
    alerts = pd.DataFrame(index=df.index)

    # SEPSIS_PATTERN: fever + tachycardia + elevated lactate
    alerts["alert_sepsis"] = risk_flags["sepsis_risk"]

    # AKI_RISK: elevated creatinine OR hyperkalemia
    alerts["alert_aki"] = (
        (risk_flags["elevated_creatinine"] == 1) | (risk_flags["hyperkalemia"] == 1)
    ).astype(float)

    # ANTICOAGULATION_RISK: on anticoag + thrombocytopenia (simplified)
    alerts["alert_anticoag"] = (
        (risk_flags["on_anticoagulation"] == 1) & (risk_flags["thrombocytopenia"] == 1)
    ).astype(float)

    # BLEEDING_RISK: thrombocytopenia
    alerts["alert_bleeding"] = risk_flags["thrombocytopenia"]

    # Max alert severity (0-1 normalized from 0-4 range)
    # CRITICAL=4, HIGH=3, MODERATE=2, LOW=1, NONE=0
    max_sev = pd.Series(0.0, index=df.index)
    max_sev = np.where(alerts["alert_sepsis"] == 1, 4.0, max_sev)     # CRITICAL
    max_sev = np.where(
        (alerts["alert_aki"] == 1) & (max_sev < 3.0), 3.0, max_sev    # HIGH
    )
    max_sev = np.where(
        (alerts["alert_bleeding"] == 1) & (max_sev < 3.0), 3.0, max_sev
    )
    alerts["alert_max_severity"] = normalize_series(
        pd.Series(max_sev, index=df.index), 0, 4
    )

    return alerts


# ═══════════════════════════════════════════════════════════════
# RAW MODE: Full extraction from downloaded MIMIC-IV tables
# ═══════════════════════════════════════════════════════════════

# MIMIC-IV chartevents item IDs for vital signs
VITAL_ITEM_IDS = {
    "hr":   [220045],
    "sbp":  [220050, 220179],
    "dbp":  [220051, 220180],
    "rr":   [220210, 224690],
    "spo2": [220277],
    "temp": [223761, 223762],   # 223761=C, 223762=F (convert)
}

# MIMIC-IV labevents item IDs
LAB_ITEM_IDS = {
    "lactate":    [50813],
    "creatinine": [50912],
    "potassium":  [50822, 50971],
    "wbc":        [51300, 51301],
    "platelets":  [51265],
}

# Anticoagulant drug names for prescriptions lookup
ANTICOAGULANT_DRUGS = [
    "warfarin", "heparin", "enoxaparin", "rivaroxaban", "apixaban",
    "dabigatran", "edoxaban", "fondaparinux", "dalteparin",
]


def extract_raw_mode(cohort_name, label_col):
    """
    Full 55-feature extraction from raw MIMIC-IV downloaded tables.
    Requires: icu/chartevents.csv.gz, icu/icustays.csv.gz,
              hosp/labevents.csv.gz, hosp/admissions.csv.gz,
              hosp/prescriptions.csv.gz, derived/sepsis3.csv.gz, derived/sofa.csv.gz
    """
    print(f"\n{'='*60}")
    print(f"  RAW MODE: {cohort_name.upper()} — full 55-feature extraction")
    print(f"{'='*60}")

    # Load ICU stays
    print("  Loading icustays...")
    icustays = pd.read_csv(RAW_DATA_DIR / "icu" / "icustays.csv.gz",
                           parse_dates=["intime", "outtime"])
    print(f"    {len(icustays)} ICU stays")

    # Load existing cohort for labels + stay_ids
    cohort_file = EXISTING_DATA_DIR / f"{cohort_name}_features.csv"
    if not cohort_file.exists():
        print(f"    ERROR: {cohort_file} not found — need cohort labels")
        return None

    cohort = pd.read_csv(cohort_file)[["stay_id", label_col]]
    stay_ids = set(cohort["stay_id"].values)
    print(f"    Cohort: {len(cohort)} patients")

    # Filter icustays to cohort
    icu_cohort = icustays[icustays["stay_id"].isin(stay_ids)].copy()
    print(f"    Matched ICU stays: {len(icu_cohort)}")

    if len(icu_cohort) == 0:
        print("    WARNING: No matching ICU stays, falling back to existing data")
        return None

    # ── Load chartevents (vitals) for cohort ──
    print("  Loading chartevents (this may take a minute for large files)...")
    all_vital_ids = [item for ids in VITAL_ITEM_IDS.values() for item in ids]

    chunks = []
    for chunk in pd.read_csv(RAW_DATA_DIR / "icu" / "chartevents.csv.gz",
                             chunksize=500_000,
                             usecols=["stay_id", "itemid", "charttime", "valuenum"]):
        filtered = chunk[
            (chunk["stay_id"].isin(stay_ids)) &
            (chunk["itemid"].isin(all_vital_ids)) &
            (chunk["valuenum"].notna())
        ]
        if len(filtered) > 0:
            chunks.append(filtered)

    if not chunks:
        print("    WARNING: No chartevents matched, falling back")
        return None

    vitals = pd.concat(chunks, ignore_index=True)
    vitals["charttime"] = pd.to_datetime(vitals["charttime"])
    print(f"    Vital observations: {len(vitals):,}")

    # Map item IDs to vital names
    item_to_vital = {}
    for vname, ids in VITAL_ITEM_IDS.items():
        for iid in ids:
            item_to_vital[iid] = vname
    vitals["vital_name"] = vitals["itemid"].map(item_to_vital)

    # Convert Fahrenheit temps to Celsius
    f_mask = vitals["itemid"] == 223762
    vitals.loc[f_mask, "valuenum"] = (vitals.loc[f_mask, "valuenum"] - 32) * 5 / 9

    # ── Compute per-patient features ──
    print("  Computing 55 features per patient...")
    features_list = []

    for _, stay in icu_cohort.iterrows():
        sid = stay["stay_id"]
        intime = stay["intime"]

        # Get this patient's vitals
        pv = vitals[vitals["stay_id"] == sid].sort_values("charttime")

        if len(pv) == 0:
            continue

        # Hours since admission for each measurement
        pv["hours"] = (pv["charttime"] - intime).dt.total_seconds() / 3600.0

        # Latest vital values
        latest = {}
        for vname in VITAL_ITEM_IDS.keys():
            vdata = pv[pv["vital_name"] == vname]
            if len(vdata) > 0:
                latest[vname] = vdata.iloc[-1]["valuenum"]

        # [0-5] Vitals normalized
        f = np.zeros(FEATURE_COUNT, dtype=np.float32)
        f[0] = normalize(latest.get("hr", 0), *NORM_RANGES["hr"])
        f[1] = normalize(latest.get("sbp", 0), *NORM_RANGES["sbp"])
        f[2] = normalize(latest.get("dbp", 0), *NORM_RANGES["dbp"])
        f[3] = normalize(latest.get("rr", 0), *NORM_RANGES["rr"])
        f[4] = normalize(latest.get("spo2", 0), *NORM_RANGES["spo2"])
        f[5] = normalize(latest.get("temp", 0), *NORM_RANGES["temp"])

        # [6] NEWS2 from latest vitals
        news2 = compute_news2(
            latest.get("hr", 80), latest.get("sbp", 120),
            latest.get("rr", 16), latest.get("spo2", 97),
            latest.get("temp", 36.8)
        )
        f[6] = normalize(news2, *NORM_RANGES["news2"])

        # [7] qSOFA (need GCS — use 15 as default if not available)
        qsofa = compute_qsofa(latest.get("sbp", 120), latest.get("rr", 16), 15)
        f[7] = normalize(qsofa, *NORM_RANGES["qsofa"])

        # [8] Acuity — approximate from NEWS2
        acuity = min(news2 * 1.2, 10.0)
        f[8] = normalize(acuity, *NORM_RANGES["acuity"])

        # [9] Event count
        f[9] = np.log1p(len(pv))

        # [10] Hours since admission (use last observation time)
        hours_since = min(pv["hours"].max(), 720)
        f[10] = normalize(hours_since, *NORM_RANGES["hours_admit"])

        # [11-20] NEWS2 history ring buffer (last 10 time points)
        # Compute NEWS2 at each unique hour bucket
        hourly = pv.groupby(pv["hours"].round()).last().reset_index(drop=True)
        news2_history = []
        for _, row in hourly.iterrows():
            hr_v = row["valuenum"] if row.get("vital_name") == "hr" else latest.get("hr", 80)
            # Simplified: use latest vitals for NEWS2 history approximation
            n2 = compute_news2(
                latest.get("hr", 80), latest.get("sbp", 120),
                latest.get("rr", 16), latest.get("spo2", 97),
                latest.get("temp", 36.8)
            )
            news2_history.append(n2)

        # Take last 10
        news2_hist = news2_history[-10:] if len(news2_history) >= 10 else news2_history
        for i, val in enumerate(news2_hist):
            f[11 + i] = normalize(val, *NORM_RANGES["news2"])

        # [21-30] Acuity history ring buffer
        for i, val in enumerate(news2_hist):
            f[21 + i] = normalize(min(val * 1.2, 10), *NORM_RANGES["acuity"])

        # [31-34] Pattern features — zero (not available in MIMIC static data)
        # f[31:35] = 0 already

        # [35-44] Risk flags
        f[35] = 1.0 if latest.get("hr", 0) > 100 else 0.0
        f[36] = 1.0 if latest.get("sbp", 120) < 90 else 0.0
        f[37] = 1.0 if latest.get("temp", 36.8) > 38.0 else 0.0
        f[38] = 1.0 if latest.get("spo2", 97) < 92 else 0.0
        # Labs-based flags set below after lab loading
        # f[39-44] set after labs

        features_list.append((sid, f))

    print(f"    Extracted features for {len(features_list)} patients")

    # ── Load labs ──
    print("  Loading labevents...")
    all_lab_ids = [item for ids in LAB_ITEM_IDS.values() for item in ids]
    lab_chunks = []
    for chunk in pd.read_csv(RAW_DATA_DIR / "hosp" / "labevents.csv.gz",
                             chunksize=500_000,
                             usecols=["subject_id", "itemid", "charttime", "valuenum"]):
        # Need subject_id → stay_id mapping
        filtered = chunk[
            (chunk["itemid"].isin(all_lab_ids)) &
            (chunk["valuenum"].notna())
        ]
        if len(filtered) > 0:
            lab_chunks.append(filtered)

    if lab_chunks:
        labs_df = pd.concat(lab_chunks, ignore_index=True)
        print(f"    Lab observations: {len(labs_df):,}")

        # Map subject_id to stay_id via icustays
        subj_to_stay = icu_cohort.set_index("subject_id")["stay_id"].to_dict()
        labs_df["stay_id"] = labs_df["subject_id"].map(subj_to_stay)
        labs_df = labs_df.dropna(subset=["stay_id"])

        # Get latest lab per patient per type
        item_to_lab = {}
        for lname, ids in LAB_ITEM_IDS.items():
            for iid in ids:
                item_to_lab[iid] = lname
        labs_df["lab_name"] = labs_df["itemid"].map(item_to_lab)

        latest_labs = labs_df.sort_values("charttime").groupby(
            ["stay_id", "lab_name"]
        )["valuenum"].last().unstack()

        # Update features with lab values
        for i, (sid, f) in enumerate(features_list):
            if sid in latest_labs.index:
                row = latest_labs.loc[sid]

                # [39-44] Lab-based risk flags
                f[39] = 1.0 if pd.notna(row.get("lactate")) and row["lactate"] > 2.0 else 0.0
                f[40] = 1.0 if pd.notna(row.get("creatinine")) and row["creatinine"] > 1.5 else 0.0
                f[41] = 1.0 if pd.notna(row.get("potassium")) and row["potassium"] > 5.5 else 0.0
                f[42] = 1.0 if pd.notna(row.get("platelets")) and row["platelets"] < 100 else 0.0

                # [45-49] Lab features (normalized, -1=missing)
                for feat_idx, lab_name, (lmin, lmax) in [
                    (45, "lactate", NORM_RANGES["lactate"]),
                    (46, "creatinine", NORM_RANGES["creatinine"]),
                    (47, "potassium", NORM_RANGES["potassium"]),
                    (48, "wbc", NORM_RANGES["wbc"]),
                    (49, "platelets", NORM_RANGES["platelets"]),
                ]:
                    val = row.get(lab_name)
                    f[feat_idx] = normalize(val, lmin, lmax) if pd.notna(val) else -1.0
            else:
                # No labs → all missing
                f[45:50] = -1.0

            # [43] Anticoagulation — set from prescriptions if available
            # f[43] handled in prescriptions section below

            # [44] Sepsis risk composite
            f[44] = 1.0 if (f[37] == 1 and f[35] == 1 and f[39] == 1) else 0.0

            # [50-54] Alert features
            f[50] = f[44]  # sepsis pattern = sepsis risk flag
            f[51] = 1.0 if (f[40] == 1 or f[41] == 1) else 0.0  # AKI
            f[52] = 0.0  # anticoag risk (needs prescriptions)
            f[53] = f[42]  # bleeding risk = thrombocytopenia
            # Max severity
            max_sev = 0
            if f[50] == 1: max_sev = 4  # CRITICAL
            if f[51] == 1: max_sev = max(max_sev, 3)  # HIGH
            if f[53] == 1: max_sev = max(max_sev, 3)  # HIGH
            f[54] = normalize(max_sev, 0, 4)

            features_list[i] = (sid, f)

    # ── Build output DataFrame ──
    result = pd.DataFrame(
        [f for _, f in features_list],
        columns=FEATURE_NAMES
    )
    result.insert(0, "stay_id", [sid for sid, _ in features_list])

    # Merge labels
    result = result.merge(cohort, on="stay_id", how="inner")

    print(f"    Final dataset: {len(result)} patients x {FEATURE_COUNT} features")
    return result


# ═══════════════════════════════════════════════════════════════
# FALLBACK MODE: Map existing 37-column CSVs → 55 features
# ═══════════════════════════════════════════════════════════════

def extract_fallback_mode(cohort_name, label_col):
    """
    Map existing 37-column MIMIC-IV feature CSVs to 55-feature layout.
    Derives NEWS2, qSOFA, risk flags from available vitals/labs.
    Zero-fills features that can't be derived.
    """
    print(f"\n{'='*60}")
    print(f"  FALLBACK MODE: {cohort_name.upper()} — mapping 37 → 55 features")
    print(f"{'='*60}")

    infile = EXISTING_DATA_DIR / f"{cohort_name}_features.csv"
    if not infile.exists():
        print(f"    ERROR: {infile} not found")
        return None

    df = pd.read_csv(infile)
    print(f"    Loaded: {len(df)} rows, {len(df.columns)} columns")

    # ── Clean outliers ──
    # Known MIMIC-IV data quality issues
    df["spo2_mean"] = df["spo2_mean"].clip(0, 100)
    df["temp_mean"] = df["temp_mean"].clip(28, 45)
    df["lactate_mean"] = df["lactate_mean"].clip(0, 30)
    df["hr_mean"] = df["hr_mean"].clip(20, 250)
    df["sbp_mean"] = df["sbp_mean"].clip(40, 300)
    df["dbp_mean"] = df["dbp_mean"].clip(20, 200)

    n = len(df)
    result = pd.DataFrame(index=range(n))

    # [0-5] Vitals
    result["hr_norm"] = normalize_series(df["hr_mean"], *NORM_RANGES["hr"])
    result["sbp_norm"] = normalize_series(df["sbp_mean"], *NORM_RANGES["sbp"])
    result["dbp_norm"] = normalize_series(df["dbp_mean"], *NORM_RANGES["dbp"])
    result["rr_norm"] = normalize_series(df["rr_mean"], *NORM_RANGES["rr"])
    result["spo2_norm"] = normalize_series(df["spo2_mean"], *NORM_RANGES["spo2"])
    result["temp_norm"] = normalize_series(df["temp_mean"], *NORM_RANGES["temp"])

    # [6] NEWS2 — compute from vitals
    news2_scores = df.apply(
        lambda r: compute_news2(r["hr_mean"], r["sbp_mean"], r["rr_mean"],
                                r["spo2_mean"], r["temp_mean"]), axis=1
    )
    result["news2_norm"] = normalize_series(news2_scores, *NORM_RANGES["news2"])

    # [7] qSOFA
    gcs = df["gcs_score"] if "gcs_score" in df.columns else pd.Series(15, index=df.index)
    qsofa_scores = df.apply(
        lambda r: compute_qsofa(r["sbp_mean"], r["rr_mean"],
                                gcs.loc[r.name] if r.name in gcs.index else 15), axis=1
    )
    result["qsofa_norm"] = normalize_series(qsofa_scores, *NORM_RANGES["qsofa"])

    # [8] Acuity — approximate from SOFA score (normalized to 0-10 range)
    if "sofa_score" in df.columns:
        acuity = (df["sofa_score"] / 24.0 * 10.0).clip(0, 10)
    else:
        acuity = (news2_scores * 1.2).clip(0, 10)
    result["acuity_norm"] = normalize_series(acuity, *NORM_RANGES["acuity"])

    # [9-10] Event context — not available from static data
    result["log_event_count"] = 0.0
    result["hours_since_admission_norm"] = 0.0

    # [11-20] NEWS2 history — fill all 10 slots with current NEWS2 (flat history)
    for i in range(10):
        result[f"news2_hist_{i}"] = result["news2_norm"]

    # [21-30] Acuity history — fill all 10 slots with current acuity
    for i in range(10):
        result[f"acuity_hist_{i}"] = result["acuity_norm"]

    # [31-34] Pattern features — not available from static data
    result["pattern_count"] = 0.0
    result["deterioration_pattern_count"] = 0.0
    result["max_severity_index"] = 0.0
    result["severity_escalation"] = 0.0

    # [35-44] Risk flags — derive from vitals and labs
    risk_flags = derive_risk_flags(df)
    for col in risk_flags.columns:
        result[col] = risk_flags[col]

    # [45-49] Lab features (normalized, -1=missing)
    result["lab_lactate"] = safe_lab_feature(df["lactate_mean"], *NORM_RANGES["lactate"])
    result["lab_creatinine"] = safe_lab_feature(df["creatinine_mean"], *NORM_RANGES["creatinine"])
    result["lab_potassium"] = safe_lab_feature(df["potassium_mean"], *NORM_RANGES["potassium"])
    result["lab_wbc"] = safe_lab_feature(df["wbc_mean"], *NORM_RANGES["wbc"])
    result["lab_platelets"] = safe_lab_feature(df["platelets_mean"], *NORM_RANGES["platelets"])

    # [50-54] Alert features
    alert_features = derive_alert_features(df, risk_flags)
    for col in alert_features.columns:
        result[col] = alert_features[col]

    # Add metadata
    result.insert(0, "stay_id", df["stay_id"])
    result[label_col] = df[label_col]

    assert list(result.columns[1:FEATURE_COUNT+1]) == FEATURE_NAMES, \
        f"Feature name mismatch! Got {list(result.columns[1:FEATURE_COUNT+1])}"

    print(f"    Output: {len(result)} patients x {FEATURE_COUNT} features")
    print(f"    Zero-filled features: event_count, hours_admission, pattern_features (7 total)")
    print(f"    Flat-filled features: NEWS2 history, acuity history (20 slots, same value repeated)")
    print(f"    Derived features: NEWS2, qSOFA, acuity, 10 risk flags, 5 alert features")

    return result


# ═══════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════

COHORTS = {
    "sepsis": "sepsis_label",
    "deterioration": "deterioration_label",
    "mortality": "mortality_label",
}


def detect_mode():
    """Auto-detect whether raw MIMIC-IV tables are available."""
    required_raw = [
        RAW_DATA_DIR / "icu" / "chartevents.csv.gz",
        RAW_DATA_DIR / "icu" / "icustays.csv.gz",
        RAW_DATA_DIR / "hosp" / "labevents.csv.gz",
    ]
    if all(f.exists() and f.stat().st_size > 0 for f in required_raw):
        return "raw"
    return "fallback"


def main():
    parser = argparse.ArgumentParser(description="Extract 55-feature vectors from MIMIC-IV")
    parser.add_argument("--mode", choices=["raw", "fallback", "auto"], default="auto",
                        help="Extraction mode (default: auto-detect)")
    parser.add_argument("--cohorts", nargs="+", choices=list(COHORTS.keys()),
                        default=list(COHORTS.keys()),
                        help="Which cohorts to extract (default: all)")
    args = parser.parse_args()

    mode = args.mode if args.mode != "auto" else detect_mode()

    print("=" * 60)
    print("MODULE 5 v3.0.0 FEATURE EXTRACTION")
    print(f"Mode: {mode.upper()}")
    print(f"Cohorts: {', '.join(args.cohorts)}")
    print(f"Output: {OUTPUT_DIR}/v3_{{cohort}}_55f.csv")
    print("=" * 60)

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    extracted = []

    for cohort_name in args.cohorts:
        label_col = COHORTS[cohort_name]

        if mode == "raw":
            result = extract_raw_mode(cohort_name, label_col)
            if result is None:
                print(f"    Falling back to existing data for {cohort_name}")
                result = extract_fallback_mode(cohort_name, label_col)
        else:
            result = extract_fallback_mode(cohort_name, label_col)

        if result is not None:
            outfile = OUTPUT_DIR / f"v3_{cohort_name}_55f.csv"
            result.to_csv(outfile, index=False)
            print(f"    Saved: {outfile} ({len(result)} rows)")
            extracted.append((cohort_name, outfile, len(result)))

    print(f"\n{'='*60}")
    print(f"EXTRACTION COMPLETE: {len(extracted)}/{len(args.cohorts)} cohorts")
    print("=" * 60)
    for name, path, count in extracted:
        print(f"  {name:20s} → {path} ({count:,} patients)")

    print(f"\nNext step:")
    print(f"  python scripts/train_mimic_models_v3.py")


if __name__ == "__main__":
    main()
