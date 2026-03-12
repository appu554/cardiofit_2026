#!/usr/bin/env python3
"""
Generate 94 additional medications (6 already exist) to reach 100 total
Uses the existing generate_medications_bulk.py framework
"""

import sys
from pathlib import Path

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent))

from generate_medications_bulk import (
    generate_medication_yaml,
    save_medication_yaml,
    MEDICATION_DATABASE as EXISTING_MEDS
)

# Import the complete expansion database
from medication_expansion_complete import MEDICATION_EXPANSION_DATABASE

def main():
    """Generate all 94 new medications"""

    base_dir = Path(__file__).parent.parent / "medications"

    print(f"\n{'='*80}")
    print(f"🎯 MEDICATION DATABASE EXPANSION")
    print(f"{'='*80}\n")

    print(f"📊 Status:")
    print(f"   • Existing medications: {len(EXISTING_MEDS)}")
    print(f"   • New medications to generate: {len(MEDICATION_EXPANSION_DATABASE)}")
    print(f"   • Target total: 100 medications\n")

    # Generate all new medications
    created_files = []
    by_category = {}
    errors = []

    for drug_name, drug_data in MEDICATION_EXPANSION_DATABASE.items():
        try:
            print(f"   Generating: {drug_name}...", end=" ")
            file_path = save_medication_yaml(drug_name, drug_data, base_dir)
            created_files.append(file_path)

            category = drug_data["classification"]["category"]
            by_category[category] = by_category.get(category, 0) + 1
            print("✓")

        except Exception as e:
            print(f"✗")
            errors.append(f"{drug_name}: {str(e)}")

    # Print summary
    print(f"\n{'='*80}")
    print(f"📊 GENERATION SUMMARY")
    print(f"{'='*80}\n")

    print(f"✅ Successfully generated: {len(created_files)} medications")

    if errors:
        print(f"\n❌ Errors ({len(errors)}):")
        for error in errors:
            print(f"   • {error}")

    print(f"\n📋 New Medications by Category:")
    for category, count in sorted(by_category.items()):
        print(f"   • {category}: {count}")

    print(f"\n🎉 Total medications in database: {len(EXISTING_MEDS) + len(created_files)}")
    print(f"{'='*80}\n")

    return 0 if not errors else 1

if __name__ == "__main__":
    exit(main())
