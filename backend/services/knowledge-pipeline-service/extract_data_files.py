#!/usr/bin/env python3
"""
Extract your downloaded ZIP files for pipeline processing
"""

import zipfile
import os
from pathlib import Path
import shutil

def extract_rxnorm():
    """Extract RxNorm ZIP file"""
    print("📋 Extracting RxNorm files...")
    
    rxnorm_dir = Path("data/rxnorm")
    zip_files = list(rxnorm_dir.glob("*.zip"))
    
    if not zip_files:
        print("   ❌ No RxNorm ZIP files found")
        return False
    
    zip_file = zip_files[0]
    print(f"   📦 Found: {zip_file.name}")
    
    # Extract to temporary directory
    extract_dir = rxnorm_dir / "extracted"
    extract_dir.mkdir(exist_ok=True)
    
    try:
        with zipfile.ZipFile(zip_file, 'r') as zip_ref:
            zip_ref.extractall(extract_dir)
        
        # Find RRF directory
        rrf_dir = None
        for root, dirs, files in os.walk(extract_dir):
            if any(f.endswith('.RRF') for f in files):
                rrf_dir = Path(root)
                break
        
        if rrf_dir:
            # Copy RRF files to main directory
            target_rrf_dir = rxnorm_dir / "rrf"
            target_rrf_dir.mkdir(exist_ok=True)
            
            rrf_files = ['RXNCONSO.RRF', 'RXNREL.RRF', 'RXNSAT.RRF', 'RXNCUI.RRF']
            copied_files = []
            
            for rrf_file in rrf_files:
                source_file = rrf_dir / rrf_file
                target_file = target_rrf_dir / rrf_file
                
                if source_file.exists():
                    shutil.copy2(source_file, target_file)
                    size_mb = target_file.stat().st_size / (1024*1024)
                    print(f"   ✅ Extracted: {rrf_file} ({size_mb:.1f} MB)")
                    copied_files.append(rrf_file)
                else:
                    print(f"   ⚠️  Not found: {rrf_file}")
            
            print(f"   📊 RxNorm: {len(copied_files)}/4 RRF files extracted")
            return len(copied_files) == 4
        else:
            print("   ❌ No RRF files found in ZIP")
            return False
            
    except Exception as e:
        print(f"   ❌ Extraction failed: {e}")
        return False

def extract_snomed():
    """Extract SNOMED CT ZIP file"""
    print("🩺 Extracting SNOMED CT files...")
    
    snomed_dir = Path("data/snomed")
    zip_files = list(snomed_dir.glob("*.zip"))
    
    if not zip_files:
        print("   ❌ No SNOMED CT ZIP files found")
        return False
    
    zip_file = zip_files[0]
    print(f"   📦 Found: {zip_file.name}")
    
    # Extract to temporary directory
    extract_dir = snomed_dir / "extracted"
    extract_dir.mkdir(exist_ok=True)
    
    try:
        with zipfile.ZipFile(zip_file, 'r') as zip_ref:
            zip_ref.extractall(extract_dir)
        
        # Find Snapshot directory
        snapshot_dir = None
        for root, dirs, files in os.walk(extract_dir):
            if "Snapshot" in root and any(f.endswith('.txt') for f in files):
                snapshot_dir = Path(root)
                break
        
        if snapshot_dir:
            # Copy snapshot files to main directory
            target_snapshot_dir = snomed_dir / "snapshot"
            target_snapshot_dir.mkdir(exist_ok=True)
            
            # Look for key SNOMED files
            snomed_files = [
                "sct2_Concept_Snapshot_INT",
                "sct2_Description_Snapshot-en_INT", 
                "sct2_Relationship_Snapshot_INT",
                "sct2_TextDefinition_Snapshot-en_INT"
            ]
            
            copied_files = []
            
            for pattern in snomed_files:
                # Find files matching pattern (they have dates in names)
                matching_files = list(snapshot_dir.glob(f"{pattern}*.txt"))
                
                if matching_files:
                    source_file = matching_files[0]
                    target_file = target_snapshot_dir / f"{pattern}.txt"
                    
                    shutil.copy2(source_file, target_file)
                    size_mb = target_file.stat().st_size / (1024*1024)
                    print(f"   ✅ Extracted: {source_file.name} ({size_mb:.1f} MB)")
                    copied_files.append(pattern)
                else:
                    print(f"   ⚠️  Not found: {pattern}*.txt")
            
            print(f"   📊 SNOMED CT: {len(copied_files)}/4 files extracted")
            return len(copied_files) >= 2  # At least concepts and descriptions
        else:
            print("   ❌ No Snapshot directory found in ZIP")
            return False
            
    except Exception as e:
        print(f"   ❌ Extraction failed: {e}")
        return False

def extract_loinc():
    """Extract LOINC ZIP file"""
    print("🧪 Extracting LOINC files...")
    
    loinc_dir = Path("data/loinc")
    zip_files = list(loinc_dir.glob("*.zip"))
    
    if not zip_files:
        print("   ❌ No LOINC ZIP files found")
        return False
    
    zip_file = zip_files[0]
    print(f"   📦 Found: {zip_file.name}")
    
    # Extract to temporary directory
    extract_dir = loinc_dir / "extracted"
    extract_dir.mkdir(exist_ok=True)
    
    try:
        with zipfile.ZipFile(zip_file, 'r') as zip_ref:
            zip_ref.extractall(extract_dir)
        
        # Find LOINC CSV files
        target_csv_dir = loinc_dir / "csv"
        target_csv_dir.mkdir(exist_ok=True)
        
        # Look for key LOINC files
        loinc_files = ["Loinc.csv", "Part.csv", "AnswerList.csv"]
        copied_files = []
        
        for loinc_file in loinc_files:
            # Search for file in extracted content
            found_files = list(extract_dir.rglob(loinc_file))
            
            if found_files:
                source_file = found_files[0]
                target_file = target_csv_dir / loinc_file
                
                shutil.copy2(source_file, target_file)
                size_mb = target_file.stat().st_size / (1024*1024)
                print(f"   ✅ Extracted: {loinc_file} ({size_mb:.1f} MB)")
                copied_files.append(loinc_file)
            else:
                print(f"   ⚠️  Not found: {loinc_file}")
        
        print(f"   📊 LOINC: {len(copied_files)}/3 files extracted")
        return len(copied_files) >= 1  # At least main LOINC table
        
    except Exception as e:
        print(f"   ❌ Extraction failed: {e}")
        return False

def main():
    print("🔧 Extracting your downloaded data files...")
    print("=" * 50)
    
    results = {}
    
    # Extract each source
    results['rxnorm'] = extract_rxnorm()
    print()
    
    results['snomed'] = extract_snomed()
    print()
    
    results['loinc'] = extract_loinc()
    print()
    
    # Summary
    print("=" * 50)
    successful = sum(results.values())
    total = len(results)
    
    print(f"📊 Extraction Summary: {successful}/{total} sources ready")
    
    for source, success in results.items():
        status = "✅ Ready" if success else "❌ Failed"
        print(f"   {source.upper()}: {status}")
    
    if successful == total:
        print("\n🎉 All files extracted successfully!")
        print("🚀 Next step: Run the pipeline")
        print("   Command: python start_pipeline.py --sources rxnorm snomed loinc")
    else:
        print(f"\n⚠️  {total - successful} sources failed extraction")
        print("   Check the error messages above")
        print("   You may need to re-download some files")

if __name__ == "__main__":
    main()
