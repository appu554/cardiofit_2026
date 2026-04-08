#!/bin/bash
# Download MIMIC-IV v3.1 tables needed for 55-feature Module5 training.
# Uses curl (pre-installed on macOS) instead of wget.
#
# Usage:
#   cd backend/shared-infrastructure/flink-processing
#   bash scripts/download_mimic_targeted.sh
#
# You will be prompted for your PhysioNet password once.
# Requires: curl, PhysioNet credentialed access (username: onkarshahi)

set -e

PHYSIONET_USER="onkarshahi"
BASE_URL="https://physionet.org/files/mimiciv/3.1"
OUTPUT_DIR="data/mimic_iv_raw"

# Tables needed for 55-feature extraction
TABLES=(
    "icu/chartevents.csv.gz"
    "icu/icustays.csv.gz"
    "hosp/labevents.csv.gz"
    "hosp/patients.csv.gz"
    "hosp/admissions.csv.gz"
    "hosp/prescriptions.csv.gz"
    "hosp/diagnoses_icd.csv.gz"
    "derived/sepsis3.csv.gz"
    "derived/sofa.csv.gz"
    "derived/first_day_vitalsign.csv.gz"
    "derived/first_day_lab.csv.gz"
)

echo "============================================================"
echo "MIMIC-IV v3.1 Targeted Download (curl)"
echo "Tables: ${#TABLES[@]}"
echo "Output: ${OUTPUT_DIR}/"
echo "============================================================"
echo ""
echo "Enter your PhysioNet password for user '${PHYSIONET_USER}':"
read -s PHYSIONET_PASS
echo ""

mkdir -p "${OUTPUT_DIR}/icu" "${OUTPUT_DIR}/hosp" "${OUTPUT_DIR}/derived"

DOWNLOADED=0
FAILED=0

for table in "${TABLES[@]}"; do
    OUTFILE="${OUTPUT_DIR}/${table}"
    echo ""
    echo "--- Downloading: ${table} ---"

    # Skip if already downloaded and non-empty
    if [ -s "${OUTFILE}" ]; then
        echo "  Already exists ($(du -h "${OUTFILE}" | cut -f1)), skipping"
        DOWNLOADED=$((DOWNLOADED + 1))
        continue
    fi

    if curl -# -f -L -C - \
        --user "${PHYSIONET_USER}:${PHYSIONET_PASS}" \
        -o "${OUTFILE}" \
        "${BASE_URL}/${table}" 2>&1; then
        DOWNLOADED=$((DOWNLOADED + 1))
        SIZE=$(du -h "${OUTFILE}" | cut -f1)
        echo "  OK (${SIZE})"
    else
        FAILED=$((FAILED + 1))
        # Remove partial/empty file on failure
        rm -f "${OUTFILE}"
        echo "  FAILED"
    fi
done

echo ""
echo "============================================================"
echo "Download complete: ${DOWNLOADED}/${#TABLES[@]} succeeded, ${FAILED} failed"
echo "============================================================"
echo ""

echo "Downloaded files:"
find "${OUTPUT_DIR}" -name "*.csv.gz" -exec ls -lh {} \;

echo ""
echo "Total size:"
du -sh "${OUTPUT_DIR}"

echo ""
echo "Next step:"
echo "  python scripts/extract_mimic_features_v3.py"
