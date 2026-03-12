-- ═══════════════════════════════════════════════════════════════════════════
-- MIMIC-IV BigQuery Export Queries
-- Run these queries in BigQuery console and export results to CSV
-- ═══════════════════════════════════════════════════════════════════════════

-- Instructions:
-- 1. Copy each query below
-- 2. Paste into BigQuery console (https://console.cloud.google.com/bigquery)
-- 3. Click "RUN"
-- 4. Click "SAVE RESULTS" → "CSV (local file)"
-- 5. Save with the specified filename


-- ═══════════════════════════════════════════════════════════════════════════
-- QUERY 1: Sepsis Cohort (10,000 patients)
-- Export as: sepsis_cohort_with_features.csv
-- ═══════════════════════════════════════════════════════════════════════════

WITH sepsis_cases AS (
  SELECT
    s.subject_id,
    s.hadm_id,
    s.stay_id,
    s.sepsis3 as sepsis_label,
    s.sofa_score,
    i.intime as icu_intime,
    i.outtime as icu_outtime,
    p.anchor_age as age,
    p.gender,
    a.admission_type,
    a.hospital_expire_flag as died_in_hospital
  FROM `physionet-data.mimiciv_derived.sepsis3` s
  INNER JOIN `physionet-data.mimiciv_icu.icustays` i
    ON s.stay_id = i.stay_id
  INNER JOIN `physionet-data.mimiciv_hosp.patients` p
    ON s.subject_id = p.subject_id
  INNER JOIN `physionet-data.mimiciv_hosp.admissions` a
    ON s.hadm_id = a.hadm_id
  WHERE p.anchor_age >= 18
    AND s.sepsis3 = 1
  LIMIT 10000
),
vitals AS (
  SELECT
    stay_id,
    AVG(heart_rate) as hr_mean,
    MIN(heart_rate) as hr_min,
    MAX(heart_rate) as hr_max,
    STDDEV(heart_rate) as hr_std,
    AVG(resp_rate) as rr_mean,
    MIN(resp_rate) as rr_min,
    MAX(resp_rate) as rr_max,
    AVG(temperature) as temp_mean,
    MAX(temperature) as temp_max,
    AVG(sbp) as sbp_mean,
    MIN(sbp) as sbp_min,
    AVG(dbp) as dbp_mean,
    AVG(mbp) as map_mean,
    MIN(mbp) as map_min,
    AVG(spo2) as spo2_mean,
    MIN(spo2) as spo2_min
  FROM `physionet-data.mimiciv_derived.first_day_vitalsign`
  WHERE stay_id IN (SELECT stay_id FROM sepsis_cases)
  GROUP BY stay_id
),
labs AS (
  SELECT
    stay_id,
    MAX(lactate) as lactate_max,
    MAX(creatinine) as creatinine_max,
    MIN(platelet) as platelet_min,
    MAX(wbc) as wbc_max,
    MIN(hematocrit) as hct_min,
    MAX(bilirubin_total) as bili_max
  FROM `physionet-data.mimiciv_derived.first_day_lab`
  WHERE stay_id IN (SELECT stay_id FROM sepsis_cases)
  GROUP BY stay_id
)
SELECT
  c.*,
  v.* EXCEPT(stay_id),
  l.* EXCEPT(stay_id)
FROM sepsis_cases c
LEFT JOIN vitals v ON c.stay_id = v.stay_id
LEFT JOIN labs l ON c.stay_id = l.stay_id;


-- ═══════════════════════════════════════════════════════════════════════════
-- QUERY 2: Clinical Deterioration Cohort (8,000 patients)
-- Export as: deterioration_cohort_with_features.csv
-- ═══════════════════════════════════════════════════════════════════════════

WITH deterioration_cases AS (
  SELECT
    i.subject_id,
    i.hadm_id,
    i.stay_id,
    CASE
      WHEN a.hospital_expire_flag = 1 THEN 1
      WHEN MAX(s.sofa_24hours) - MIN(s.sofa_24hours) >= 2 THEN 1
      ELSE 0
    END as deterioration_label,
    p.anchor_age as age,
    p.gender,
    a.admission_type,
    a.hospital_expire_flag as died_in_hospital,
    MAX(s.sofa_24hours) as max_sofa
  FROM `physionet-data.mimiciv_icu.icustays` i
  INNER JOIN `physionet-data.mimiciv_hosp.patients` p
    ON i.subject_id = p.subject_id
  INNER JOIN `physionet-data.mimiciv_hosp.admissions` a
    ON i.hadm_id = a.hadm_id
  LEFT JOIN `physionet-data.mimiciv_derived.sofa` s
    ON i.stay_id = s.stay_id
  WHERE p.anchor_age >= 18
  GROUP BY
    i.subject_id, i.hadm_id, i.stay_id,
    a.hospital_expire_flag, p.anchor_age,
    p.gender, a.admission_type
  HAVING deterioration_label = 1
  LIMIT 8000
),
vitals AS (
  SELECT
    stay_id,
    AVG(heart_rate) as hr_mean,
    MIN(heart_rate) as hr_min,
    MAX(heart_rate) as hr_max,
    STDDEV(heart_rate) as hr_std,
    AVG(resp_rate) as rr_mean,
    MIN(resp_rate) as rr_min,
    MAX(resp_rate) as rr_max,
    AVG(temperature) as temp_mean,
    MAX(temperature) as temp_max,
    AVG(sbp) as sbp_mean,
    MIN(sbp) as sbp_min,
    AVG(dbp) as dbp_mean,
    AVG(mbp) as map_mean,
    MIN(mbp) as map_min,
    AVG(spo2) as spo2_mean,
    MIN(spo2) as spo2_min
  FROM `physionet-data.mimiciv_derived.first_day_vitalsign`
  WHERE stay_id IN (SELECT stay_id FROM deterioration_cases)
  GROUP BY stay_id
),
labs AS (
  SELECT
    stay_id,
    MAX(lactate) as lactate_max,
    MAX(creatinine) as creatinine_max,
    MIN(platelet) as platelet_min,
    MAX(wbc) as wbc_max,
    MIN(hematocrit) as hct_min,
    MAX(bilirubin_total) as bili_max
  FROM `physionet-data.mimiciv_derived.first_day_lab`
  WHERE stay_id IN (SELECT stay_id FROM deterioration_cases)
  GROUP BY stay_id
)
SELECT
  c.*,
  v.* EXCEPT(stay_id),
  l.* EXCEPT(stay_id)
FROM deterioration_cases c
LEFT JOIN vitals v ON c.stay_id = v.stay_id
LEFT JOIN labs l ON c.stay_id = l.stay_id;


-- ═══════════════════════════════════════════════════════════════════════════
-- QUERY 3: Mortality Cohort (5,000 patients)
-- Export as: mortality_cohort_with_features.csv
-- ═══════════════════════════════════════════════════════════════════════════

WITH mortality_cases AS (
  SELECT
    i.subject_id,
    i.hadm_id,
    i.stay_id,
    a.hospital_expire_flag as mortality_label,
    p.anchor_age as age,
    p.gender,
    a.admission_type,
    a.hospital_expire_flag as died_in_hospital
  FROM `physionet-data.mimiciv_icu.icustays` i
  INNER JOIN `physionet-data.mimiciv_hosp.patients` p
    ON i.subject_id = p.subject_id
  INNER JOIN `physionet-data.mimiciv_hosp.admissions` a
    ON i.hadm_id = a.hadm_id
  WHERE p.anchor_age >= 18
    AND a.hospital_expire_flag = 1
  LIMIT 5000
),
vitals AS (
  SELECT
    stay_id,
    AVG(heart_rate) as hr_mean,
    MIN(heart_rate) as hr_min,
    MAX(heart_rate) as hr_max,
    STDDEV(heart_rate) as hr_std,
    AVG(resp_rate) as rr_mean,
    MIN(resp_rate) as rr_min,
    MAX(resp_rate) as rr_max,
    AVG(temperature) as temp_mean,
    MAX(temperature) as temp_max,
    AVG(sbp) as sbp_mean,
    MIN(sbp) as sbp_min,
    AVG(dbp) as dbp_mean,
    AVG(mbp) as map_mean,
    MIN(mbp) as map_min,
    AVG(spo2) as spo2_mean,
    MIN(spo2) as spo2_min
  FROM `physionet-data.mimiciv_derived.first_day_vitalsign`
  WHERE stay_id IN (SELECT stay_id FROM mortality_cases)
  GROUP BY stay_id
),
labs AS (
  SELECT
    stay_id,
    MAX(lactate) as lactate_max,
    MAX(creatinine) as creatinine_max,
    MIN(platelet) as platelet_min,
    MAX(wbc) as wbc_max,
    MIN(hematocrit) as hct_min,
    MAX(bilirubin_total) as bili_max
  FROM `physionet-data.mimiciv_derived.first_day_lab`
  WHERE stay_id IN (SELECT stay_id FROM mortality_cases)
  GROUP BY stay_id
)
SELECT
  c.*,
  v.* EXCEPT(stay_id),
  l.* EXCEPT(stay_id)
FROM mortality_cases c
LEFT JOIN vitals v ON c.stay_id = v.stay_id
LEFT JOIN labs l ON c.stay_id = l.stay_id;


-- ═══════════════════════════════════════════════════════════════════════════
-- QUERY 4: 30-Day Readmission Cohort (10,000 patients)
-- Export as: readmission_cohort_with_features.csv
-- ═══════════════════════════════════════════════════════════════════════════

WITH readmissions AS (
  SELECT
    i1.subject_id,
    i1.hadm_id,
    i1.stay_id,
    CASE
      WHEN COUNT(i2.stay_id) > 0 THEN 1
      ELSE 0
    END as readmission_label,
    p.anchor_age as age,
    p.gender,
    a1.admission_type,
    a1.hospital_expire_flag as died_in_hospital
  FROM `physionet-data.mimiciv_icu.icustays` i1
  INNER JOIN `physionet-data.mimiciv_hosp.patients` p
    ON i1.subject_id = p.subject_id
  INNER JOIN `physionet-data.mimiciv_hosp.admissions` a1
    ON i1.hadm_id = a1.hadm_id
  LEFT JOIN `physionet-data.mimiciv_icu.icustays` i2
    ON i1.subject_id = i2.subject_id
    AND i2.intime > i1.outtime
    AND DATE_DIFF(DATE(i2.intime), DATE(i1.outtime), DAY) <= 30
  WHERE p.anchor_age >= 18
  GROUP BY
    i1.subject_id, i1.hadm_id, i1.stay_id,
    p.anchor_age, p.gender,
    a1.admission_type, a1.hospital_expire_flag
  HAVING readmission_label = 1
  LIMIT 10000
),
vitals AS (
  SELECT
    stay_id,
    AVG(heart_rate) as hr_mean,
    MIN(heart_rate) as hr_min,
    MAX(heart_rate) as hr_max,
    STDDEV(heart_rate) as hr_std,
    AVG(resp_rate) as rr_mean,
    MIN(resp_rate) as rr_min,
    MAX(resp_rate) as rr_max,
    AVG(temperature) as temp_mean,
    MAX(temperature) as temp_max,
    AVG(sbp) as sbp_mean,
    MIN(sbp) as sbp_min,
    AVG(dbp) as dbp_mean,
    AVG(mbp) as map_mean,
    MIN(mbp) as map_min,
    AVG(spo2) as spo2_mean,
    MIN(spo2) as spo2_min
  FROM `physionet-data.mimiciv_derived.first_day_vitalsign`
  WHERE stay_id IN (SELECT stay_id FROM readmissions)
  GROUP BY stay_id
),
labs AS (
  SELECT
    stay_id,
    MAX(lactate) as lactate_max,
    MAX(creatinine) as creatinine_max,
    MIN(platelet) as platelet_min,
    MAX(wbc) as wbc_max,
    MIN(hematocrit) as hct_min,
    MAX(bilirubin_total) as bili_max
  FROM `physionet-data.mimiciv_derived.first_day_lab`
  WHERE stay_id IN (SELECT stay_id FROM readmissions)
  GROUP BY stay_id
)
SELECT
  r.*,
  v.* EXCEPT(stay_id),
  l.* EXCEPT(stay_id)
FROM readmissions r
LEFT JOIN vitals v ON r.stay_id = v.stay_id
LEFT JOIN labs l ON r.stay_id = l.stay_id;


-- ═══════════════════════════════════════════════════════════════════════════
-- END OF QUERIES
-- ═══════════════════════════════════════════════════════════════════════════

-- After exporting all 4 CSV files, place them in:
-- /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/data/mimic_iv/
--
-- Then run: python3 scripts/train_from_csv.py
