#!/usr/bin/env python3
"""
Quick validation script to check your downloaded files
"""

import os
from pathlib import Path

def check_files():
    base_dir = Path("data")
    
    print("🔍 Checking your downloaded files...")
    print("=" * 50)
    
    # Check RxNorm
    rxnorm_dir = base_dir / "rxnorm"
    rxnorm_files = list(rxnorm_dir.glob("*.zip"))
    
    print(f"📋 RxNorm Directory: {rxnorm_dir}")
    if rxnorm_files:
        for file in rxnorm_files:
            size_mb = file.stat().st_size / (1024*1024)
            print(f"   ✅ Found: {file.name} ({size_mb:.1f} MB)")
    else:
        print("   ❌ No RxNorm ZIP files found")
        print("   📝 Expected: RxNorm_full_current.zip or similar")
    
    print()
    
    # Check SNOMED CT
    snomed_dir = base_dir / "snomed"
    snomed_files = list(snomed_dir.glob("*.zip"))
    
    print(f"🩺 SNOMED CT Directory: {snomed_dir}")
    if snomed_files:
        for file in snomed_files:
            size_mb = file.stat().st_size / (1024*1024)
            print(f"   ✅ Found: {file.name} ({size_mb:.1f} MB)")
    else:
        print("   ❌ No SNOMED CT ZIP files found")
        print("   📝 Expected: SnomedCT_InternationalRF2_PRODUCTION_*.zip")
    
    print()
    
    # Check LOINC
    loinc_dir = base_dir / "loinc"
    loinc_files = list(loinc_dir.glob("*.zip"))
    
    print(f"🧪 LOINC Directory: {loinc_dir}")
    if loinc_files:
        for file in loinc_files:
            size_mb = file.stat().st_size / (1024*1024)
            print(f"   ✅ Found: {file.name} ({size_mb:.1f} MB)")
    else:
        print("   ❌ No LOINC ZIP files found")
        print("   📝 Expected: Loinc_*.zip")
    
    print()
    print("=" * 50)
    
    # Summary
    total_found = len(rxnorm_files) + len(snomed_files) + len(loinc_files)
    print(f"📊 Summary: {total_found}/3 data sources found")
    
    if total_found == 3:
        print("🎉 All your downloaded files are ready!")
        print("🚀 Next step: Run the pipeline with these sources")
        print("   Command: python start_pipeline.py --sources rxnorm snomed loinc")
    else:
        print("📋 Action needed:")
        print("   1. Copy your downloaded ZIP files to the directories above")
        print("   2. Keep the original ZIP file names")
        print("   3. Re-run this script to verify")

if __name__ == "__main__":
    check_files()
