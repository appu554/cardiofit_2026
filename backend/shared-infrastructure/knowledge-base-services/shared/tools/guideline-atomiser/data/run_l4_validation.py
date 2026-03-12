#!/usr/bin/env python3
"""
L4 Terminology Validation - Standalone Script
Validates RxNorm codes from L3 extraction against KB-7 Terminology Service.

Usage:
    python run_l4_validation.py
"""
import sys
import os
import json

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

print("=" * 70)
print("L4: TERMINOLOGY VALIDATION (KB-7 THREE-CHECK Pipeline)")
print("=" * 70)
print()

# Load L3 output
kb1_path = os.path.join(os.path.dirname(__file__), "output", "kb1_dosing_facts_targeted.json")

if not os.path.exists(kb1_path):
    print(f"ERROR: L3 output not found at {kb1_path}")
    print("Run the L1-L2-L3 pipeline first.")
    sys.exit(1)

with open(kb1_path) as f:
    kb1_data = json.load(f)

drugs = kb1_data.get("drugs", [])
print(f"Loaded {len(drugs)} drugs from L3 extraction")
print()

# Connect to KB-7
kb7_url = os.environ.get("KB7_URL", "http://localhost:8092")
print(f"Connecting to KB-7 at {kb7_url}...")

try:
    from kb7_client import KB7Client
    client = KB7Client(base_url=kb7_url)

    if client.health_check():
        print("   KB-7 connected and healthy")
    else:
        print("   KB-7 health check failed")
        sys.exit(1)
except Exception as e:
    print(f"ERROR: Failed to connect to KB-7: {e}")
    sys.exit(1)

print()
print("-" * 70)
print("L4 VALIDATION RESULTS")
print("-" * 70)
print()

# Track results
valid_codes = []
invalid_codes = []
mismatched_codes = []

for drug in drugs:
    rxnorm_code = drug.get("rxnormCode", "")
    drug_name = drug.get("drugName", "")

    print(f"Validating: {drug_name} (RxNorm: {rxnorm_code})")

    try:
        result = client.validate_rxnorm(rxnorm_code)

        if result.is_valid:
            # Check if display name matches expected drug
            display_lower = (result.display_name or "").lower()
            expected_lower = drug_name.lower()

            # Check for name mismatch (the hallucination case)
            if expected_lower not in display_lower and display_lower not in expected_lower:
                print(f"   MISMATCH DETECTED!")
                print(f"   Expected: {drug_name}")
                print(f"   KB-7 says: {result.display_name}")
                print(f"   Status: CODE VALID BUT WRONG DRUG")
                mismatched_codes.append({
                    "rxnorm_code": rxnorm_code,
                    "expected_drug": drug_name,
                    "actual_drug": result.display_name,
                    "issue": "LLM hallucinated wrong RxNorm code"
                })
            else:
                print(f"   VALID - {result.display_name}")
                valid_codes.append({
                    "rxnorm_code": rxnorm_code,
                    "drug_name": drug_name,
                    "display_name": result.display_name
                })
        else:
            print(f"   NOT FOUND in KB-7")
            invalid_codes.append({
                "rxnorm_code": rxnorm_code,
                "drug_name": drug_name,
                "error": result.error or "Code not found"
            })
    except Exception as e:
        print(f"   ERROR: {e}")
        invalid_codes.append({
            "rxnorm_code": rxnorm_code,
            "drug_name": drug_name,
            "error": str(e)
        })

    print()

client.close()

# Summary
print("=" * 70)
print("L4 VALIDATION SUMMARY")
print("=" * 70)
print()
print(f"Total drugs validated: {len(drugs)}")
print(f"   VALID codes:        {len(valid_codes)}")
print(f"   INVALID codes:      {len(invalid_codes)}")
print(f"   MISMATCHED codes:   {len(mismatched_codes)}")
print()

if mismatched_codes:
    print("CRITICAL - RxNorm Code Hallucinations Detected:")
    print("-" * 50)
    for m in mismatched_codes:
        print(f"   {m['expected_drug']}: Used {m['rxnorm_code']}")
        print(f"      But {m['rxnorm_code']} is actually: {m['actual_drug']}")
        print(f"      Issue: {m['issue']}")
        print()

if invalid_codes:
    print("INVALID Codes (not found in KB-7):")
    print("-" * 50)
    for inv in invalid_codes:
        print(f"   {inv['drug_name']}: {inv['rxnorm_code']} - {inv['error']}")
    print()

# Save validation report
report = {
    "validation_date": __import__("datetime").datetime.now().isoformat(),
    "kb7_url": kb7_url,
    "total_drugs": len(drugs),
    "valid_count": len(valid_codes),
    "invalid_count": len(invalid_codes),
    "mismatch_count": len(mismatched_codes),
    "valid_codes": valid_codes,
    "invalid_codes": invalid_codes,
    "mismatched_codes": mismatched_codes
}

report_path = os.path.join(os.path.dirname(__file__), "output", "l4_validation_report.json")
with open(report_path, "w") as f:
    json.dump(report, f, indent=2)

print(f"Validation report saved to: {report_path}")
print()

# Exit with error if mismatches found
if mismatched_codes:
    print("L4 VALIDATION FAILED - RxNorm code hallucinations require correction")
    sys.exit(1)
elif invalid_codes:
    print("L4 VALIDATION WARNING - Some codes not found in KB-7")
    sys.exit(0)
else:
    print("L4 VALIDATION PASSED - All codes validated successfully")
    sys.exit(0)
