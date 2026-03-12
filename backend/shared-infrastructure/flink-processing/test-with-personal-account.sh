#!/bin/bash

echo "🔍 Testing MIMIC-IV Access with Your Personal Google Account"
echo "=============================================================="
echo ""
echo "This will use YOUR Google credentials (onkarshahi@vaidshala.com)"
echo "instead of the service account credentials."
echo ""
echo "Press Ctrl+C to cancel, or Enter to continue..."
read

python3 << 'EOF'
from google.cloud import bigquery
import os

# Temporarily remove service account env var to force personal auth
if 'GOOGLE_APPLICATION_CREDENTIALS' in os.environ:
    del os.environ['GOOGLE_APPLICATION_CREDENTIALS']

print("🔐 Using your personal Google credentials...")
print("   This will open a browser for authentication if needed.")
print()

try:
    # This will use Application Default Credentials (your personal account)
    client = bigquery.Client(project="sincere-hybrid-477206-h2")

    print("✅ Client initialized with personal credentials")
    print()
    print("🔍 Testing query: Counting MIMIC-IV patients...")

    query = "SELECT COUNT(*) as patient_count FROM `physionet-data.mimiciv_hosp.patients`"
    result = client.query(query).to_dataframe()
    count = result['patient_count'].iloc[0]

    print()
    print("=" * 70)
    print("✅ SUCCESS! Personal Account Has Access")
    print("=" * 70)
    print(f"📊 MIMIC-IV Hospital Patients: {count:,}")
    print()
    print("Now testing ICU dataset...")

    query_icu = "SELECT COUNT(*) as stay_count FROM `physionet-data.mimiciv_icu.icustays`"
    result_icu = client.query(query_icu).to_dataframe()
    count_icu = result_icu['stay_count'].iloc[0]

    print(f"📊 MIMIC-IV ICU Stays: {count_icu:,}")
    print()
    print("=" * 70)
    print()
    print("✅ Your personal account works perfectly!")
    print()
    print("Options:")
    print("  1. Run training pipeline with personal account (works now)")
    print("  2. Link GCP project on PhysioNet (better for production)")
    print()
    print("To run training with personal credentials:")
    print("  1. Unset service account: unset GOOGLE_APPLICATION_CREDENTIALS")
    print("  2. Run pipeline: ./train-mimic-models.sh")
    print()

except Exception as e:
    print()
    print("=" * 70)
    print("❌ Error with Personal Credentials")
    print("=" * 70)
    print(f"Error: {e}")
    print()
    print("This might mean:")
    print("  - Need to authenticate: gcloud auth application-default login")
    print("  - Or PhysioNet access issue")

EOF
