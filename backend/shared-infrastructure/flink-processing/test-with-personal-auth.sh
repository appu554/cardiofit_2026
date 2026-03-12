#!/bin/bash

echo "🔍 Testing with your personal Google credentials..."
echo ""
echo "This will open a browser window for you to authenticate."
echo "After you authorize, the script will test MIMIC-IV access."
echo ""

python3 << 'EOF'
from google.cloud import bigquery

# This will use your personal Google account (opens browser)
client = bigquery.Client(project="sincere-hybrid-477206-h2")

try:
    query = "SELECT COUNT(*) as patient_count FROM `physionet-data.mimiciv_hosp.patients`"
    result = client.query(query).to_dataframe()
    count = result['patient_count'].iloc[0]
    
    print("=" * 60)
    print("✅ SUCCESS with Personal Credentials!")
    print("=" * 60)
    print(f"📊 MIMIC-IV Patients: {count:,}")
    print()
    print("This confirms MIMIC-IV access works with YOUR account.")
    print("Now you need to link your GCP project on PhysioNet")
    print("so service accounts can also access the data.")
    
except Exception as e:
    print(f"❌ Error: {e}")

EOF
