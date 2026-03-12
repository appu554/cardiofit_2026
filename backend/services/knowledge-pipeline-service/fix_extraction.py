#!/usr/bin/env python3
"""
Fix extraction based on actual file structure
"""

import shutil
from pathlib import Path

def fix_rxnorm_extraction():
    """Copy RxNorm RRF files to correct location"""
    print("📋 Fixing RxNorm extraction...")
    
    # Source: extracted/rrf/
    source_dir = Path("data/rxnorm/extracted/rrf")
    target_dir = Path("data/rxnorm/rrf")
    target_dir.mkdir(exist_ok=True)
    
    required_files = ['RXNCONSO.RRF', 'RXNREL.RRF', 'RXNSAT.RRF', 'RXNCUI.RRF']
    copied_files = []
    
    for rrf_file in required_files:
        source_file = source_dir / rrf_file
        target_file = target_dir / rrf_file
        
        if source_file.exists():
            shutil.copy2(source_file, target_file)
            size_mb = target_file.stat().st_size / (1024*1024)
            print(f"   ✅ Copied: {rrf_file} ({size_mb:.1f} MB)")
            copied_files.append(rrf_file)
        else:
            print(f"   ❌ Missing: {rrf_file}")
    
    print(f"   📊 RxNorm: {len(copied_files)}/4 RRF files ready")
    return len(copied_files) >= 3  # Need at least 3 core files

def fix_snomed_extraction():
    """Copy SNOMED CT files to correct location"""
    print("🩺 Fixing SNOMED CT extraction...")
    
    # Source: extracted/SnomedCT_InternationalRF2_PRODUCTION_20250701T120000Z/Snapshot/Terminology/
    source_dir = Path("data/snomed/extracted/SnomedCT_InternationalRF2_PRODUCTION_20250701T120000Z/Snapshot/Terminology")
    target_dir = Path("data/snomed/snapshot")
    target_dir.mkdir(exist_ok=True)
    
    # Map actual files to expected names
    file_mapping = {
        'sct2_Concept_Snapshot_INT_20250701.txt': 'sct2_Concept_Snapshot_INT.txt',
        'sct2_Description_Snapshot-en_INT_20250701.txt': 'sct2_Description_Snapshot-en_INT.txt',
        'sct2_Relationship_Snapshot_INT_20250701.txt': 'sct2_Relationship_Snapshot_INT.txt',
        'sct2_TextDefinition_Snapshot-en_INT_20250701.txt': 'sct2_TextDefinition_Snapshot-en_INT.txt'
    }
    
    copied_files = []
    
    for source_name, target_name in file_mapping.items():
        source_file = source_dir / source_name
        target_file = target_dir / target_name
        
        if source_file.exists():
            shutil.copy2(source_file, target_file)
            size_mb = target_file.stat().st_size / (1024*1024)
            print(f"   ✅ Copied: {target_name} ({size_mb:.1f} MB)")
            copied_files.append(target_name)
        else:
            print(f"   ❌ Missing: {source_name}")
    
    print(f"   📊 SNOMED CT: {len(copied_files)}/4 files ready")
    return len(copied_files) >= 2  # Need at least concepts and descriptions

def extract_snomed_loinc_extension():
    """Extract SNOMED CT LOINC Extension files"""
    print("🧪 Extracting SNOMED CT LOINC Extension...")

    # Source: extracted/SnomedCT_LOINCExtension_PRODUCTION_LO1010000_20250321T120000Z/Snapshot/Terminology/
    source_dir = Path("data/loinc/extracted/SnomedCT_LOINCExtension_PRODUCTION_LO1010000_20250321T120000Z/Snapshot/Terminology")
    target_dir = Path("data/loinc/snapshot")
    target_dir.mkdir(exist_ok=True)

    if not source_dir.exists():
        print("   ❌ SNOMED CT LOINC Extension directory not found")
        return False

    # Look for LOINC extension files
    extension_files = list(source_dir.glob("*.txt"))

    if not extension_files:
        print("   ❌ No extension files found")
        return False

    print("   📄 Found SNOMED CT LOINC Extension files:")
    copied_files = []

    for ext_file in extension_files:
        target_file = target_dir / ext_file.name
        shutil.copy2(ext_file, target_file)
        size_mb = target_file.stat().st_size / (1024*1024)
        print(f"   ✅ Copied: {ext_file.name} ({size_mb:.1f} MB)")
        copied_files.append(ext_file.name)

    # Also check for Refset directory (important for LOINC mappings)
    refset_source = Path("data/loinc/extracted/SnomedCT_LOINCExtension_PRODUCTION_LO1010000_20250321T120000Z/Snapshot/Refset")
    if refset_source.exists():
        refset_target = Path("data/loinc/refset")
        refset_target.mkdir(exist_ok=True)

        refset_files = list(refset_source.glob("*.txt"))
        for refset_file in refset_files:
            target_file = refset_target / refset_file.name
            shutil.copy2(refset_file, target_file)
            size_mb = target_file.stat().st_size / (1024*1024)
            print(f"   ✅ Copied refset: {refset_file.name} ({size_mb:.1f} MB)")
            copied_files.append(refset_file.name)

    print(f"   📊 SNOMED CT LOINC Extension: {len(copied_files)} files ready")
    print("   ℹ️  This provides LOINC-to-SNOMED CT mappings")
    return len(copied_files) > 0

def main():
    print("🔧 Fixing file extraction based on actual structure...")
    print("=" * 60)
    
    results = {}
    
    # Fix RxNorm
    results['rxnorm'] = fix_rxnorm_extraction()
    print()
    
    # Fix SNOMED CT
    results['snomed'] = fix_snomed_extraction()
    print()
    
    # Extract SNOMED CT LOINC Extension
    results['loinc'] = extract_snomed_loinc_extension()
    print()
    
    # Summary
    print("=" * 60)
    successful = sum(results.values())
    total = len(results)
    
    print(f"📊 Extraction Fix Summary: {successful}/{total} sources ready")
    
    for source, success in results.items():
        status = "✅ Ready" if success else "❌ Not Ready"
        print(f"   {source.upper()}: {status}")
    
    if successful >= 2:  # At least 2 sources working
        print(f"\n🎉 {successful} sources are ready!")
        
        ready_sources = [source for source, success in results.items() if success]
        sources_str = " ".join(ready_sources)
        
        print(f"🚀 You can run the pipeline with: {sources_str}")
        print(f"   Command: python start_pipeline.py --sources {sources_str}")
        
        if not results['loinc']:
            print("\n📝 For LOINC:")
            print("   1. Go to https://loinc.org/downloads/")
            print("   2. Create account and accept license")
            print("   3. Download 'LOINC Table File (CSV)'")
            print("   4. Save as: data/loinc/Loinc_current.zip")
    else:
        print(f"\n⚠️  Only {successful} sources ready")
        print("   Check the error messages above")

if __name__ == "__main__":
    main()
