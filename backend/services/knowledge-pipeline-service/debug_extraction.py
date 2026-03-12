#!/usr/bin/env python3
"""
Debug extraction script to explore ZIP file contents
"""

import zipfile
import os
from pathlib import Path

def explore_zip_contents(zip_path, max_files=20):
    """Explore what's inside a ZIP file"""
    print(f"🔍 Exploring: {zip_path.name}")
    
    try:
        with zipfile.ZipFile(zip_path, 'r') as zip_ref:
            file_list = zip_ref.namelist()
            
            print(f"   📊 Total files: {len(file_list)}")
            
            # Show directory structure
            directories = set()
            for file_path in file_list:
                parts = Path(file_path).parts
                for i in range(1, len(parts)):
                    directories.add('/'.join(parts[:i]))
            
            print("   📁 Directory structure:")
            for directory in sorted(directories)[:10]:  # Show first 10 directories
                print(f"      {directory}/")
            
            if len(directories) > 10:
                print(f"      ... and {len(directories) - 10} more directories")
            
            # Show some files
            print(f"   📄 Sample files (first {max_files}):")
            for file_path in file_list[:max_files]:
                if not file_path.endswith('/'):  # Skip directories
                    size = zip_ref.getinfo(file_path).file_size
                    size_mb = size / (1024*1024)
                    print(f"      {file_path} ({size_mb:.1f} MB)")
            
            if len(file_list) > max_files:
                print(f"      ... and {len(file_list) - max_files} more files")
            
            return file_list
    
    except Exception as e:
        print(f"   ❌ Error exploring ZIP: {e}")
        return []

def find_target_files(file_list, patterns):
    """Find files matching specific patterns"""
    matches = {}
    
    for pattern in patterns:
        matches[pattern] = []
        for file_path in file_list:
            if pattern.lower() in file_path.lower():
                matches[pattern].append(file_path)
    
    return matches

def main():
    print("🔍 Debugging ZIP file extraction...")
    print("=" * 60)
    
    # Check RxNorm
    print("\n📋 RxNorm Analysis:")
    rxnorm_dir = Path("data/rxnorm")
    rxnorm_zips = list(rxnorm_dir.glob("*.zip"))
    
    if rxnorm_zips:
        file_list = explore_zip_contents(rxnorm_zips[0])
        
        # Look for RRF files
        rrf_patterns = ['RXNCONSO.RRF', 'RXNREL.RRF', 'RXNSAT.RRF', 'RXNCUI.RRF']
        matches = find_target_files(file_list, rrf_patterns)
        
        print("   🎯 RRF file matches:")
        for pattern, found_files in matches.items():
            if found_files:
                print(f"      ✅ {pattern}: {found_files[0]}")
            else:
                print(f"      ❌ {pattern}: Not found")
    
    # Check SNOMED CT
    print("\n🩺 SNOMED CT Analysis:")
    snomed_dir = Path("data/snomed")
    snomed_zips = list(snomed_dir.glob("*.zip"))
    
    if snomed_zips:
        file_list = explore_zip_contents(snomed_zips[0])
        
        # Look for snapshot files
        snapshot_patterns = ['sct2_Concept_Snapshot', 'sct2_Description_Snapshot', 
                           'sct2_Relationship_Snapshot', 'sct2_TextDefinition_Snapshot']
        matches = find_target_files(file_list, snapshot_patterns)
        
        print("   🎯 Snapshot file matches:")
        for pattern, found_files in matches.items():
            if found_files:
                print(f"      ✅ {pattern}: {found_files[0]}")
            else:
                print(f"      ❌ {pattern}: Not found")
    
    # Check LOINC
    print("\n🧪 LOINC Analysis:")
    loinc_dir = Path("data/loinc")
    loinc_zips = list(loinc_dir.glob("*.zip"))
    
    if loinc_zips:
        file_list = explore_zip_contents(loinc_zips[0])
        
        # Look for CSV files
        csv_patterns = ['Loinc.csv', 'Part.csv', 'AnswerList.csv']
        matches = find_target_files(file_list, csv_patterns)
        
        print("   🎯 CSV file matches:")
        for pattern, found_files in matches.items():
            if found_files:
                print(f"      ✅ {pattern}: {found_files[0]}")
            else:
                print(f"      ❌ {pattern}: Not found")
        
        # Also look for any CSV files
        csv_files = [f for f in file_list if f.lower().endswith('.csv')]
        if csv_files:
            print("   📄 All CSV files found:")
            for csv_file in csv_files[:10]:  # Show first 10
                print(f"      {csv_file}")
    
    print("\n" + "=" * 60)
    print("🎯 Next Steps:")
    print("1. Review the file paths shown above")
    print("2. I'll create a custom extraction script based on your actual file structure")
    print("3. Run the custom extraction script")

if __name__ == "__main__":
    main()
