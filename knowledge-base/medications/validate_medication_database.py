#!/usr/bin/env python3
"""
Medication Database Validation Script
Validates all medication YAML files for completeness and quality
"""

import os
import yaml
from pathlib import Path
from collections import defaultdict
from typing import Dict, List, Tuple

BASE_DIR = Path(__file__).parent

# Required fields for medication YAML validation
REQUIRED_FIELDS = [
    'medicationId',
    'genericName',
    'brandNames',
    'rxNormCode',
    'ndcCode',
    'atcCode',
    'classification',
    'adultDosing',
    'pediatricDosing',
    'geriatricDosing',
    'contraindications',
    'adverseEffects',
    'pregnancyLactation',
    'monitoring',
    'lastUpdated',
    'source',
    'version'
]

REQUIRED_CLASSIFICATION_FIELDS = [
    'therapeuticClass',
    'pharmacologicClass',
    'chemicalClass',
    'category',
    'subcategories',
    'highAlert',
    'blackBoxWarning'
]

REQUIRED_ADULT_DOSING_FIELDS = [
    'standard',
    'indicationBased',
    'renalAdjustment'
]

def validate_medication_file(filepath: Path) -> Tuple[bool, List[str]]:
    """Validate a single medication YAML file"""
    errors = []

    try:
        with open(filepath, 'r') as f:
            # Skip comment lines
            content = f.read()
            # Find first non-comment line
            yaml_start = 0
            for i, line in enumerate(content.split('\n')):
                if line.strip() and not line.strip().startswith('#'):
                    yaml_start = content.index(line)
                    break

            data = yaml.safe_load(content[yaml_start:])

        if not data:
            errors.append("Empty YAML file")
            return False, errors

        # Check required top-level fields
        for field in REQUIRED_FIELDS:
            if field not in data:
                errors.append(f"Missing required field: {field}")

        # Validate classification
        if 'classification' in data:
            classification = data['classification']
            for field in REQUIRED_CLASSIFICATION_FIELDS:
                if field not in classification:
                    errors.append(f"Missing classification field: {field}")

        # Validate adult dosing
        if 'adultDosing' in data:
            adult_dosing = data['adultDosing']
            for field in REQUIRED_ADULT_DOSING_FIELDS:
                if field not in adult_dosing:
                    errors.append(f"Missing adult dosing field: {field}")

        # Check for high-alert medications
        if 'classification' in data:
            if data['classification'].get('highAlert'):
                # Verify extra monitoring for high-alert drugs
                if 'monitoring' not in data or not data['monitoring']:
                    errors.append("High-alert medication missing monitoring parameters")

        # Check black box warnings
        if 'classification' in data:
            if data['classification'].get('blackBoxWarning'):
                if 'adverseEffects' not in data:
                    errors.append("Black box warning drug missing adverse effects section")

        return len(errors) == 0, errors

    except yaml.YAMLError as e:
        errors.append(f"YAML parsing error: {str(e)}")
        return False, errors
    except Exception as e:
        errors.append(f"Validation error: {str(e)}")
        return False, errors

def scan_medication_database() -> Dict:
    """Scan and validate entire medication database"""
    print("🏥 Medication Database Validation")
    print("=" * 70)
    print()

    all_medications = list(BASE_DIR.rglob("*.yaml"))
    # Exclude the generator scripts
    all_medications = [m for m in all_medications if m.name != 'validate_medication_database.yaml']

    print(f"📂 Found {len(all_medications)} medication files")
    print()

    results = {
        'total': len(all_medications),
        'valid': 0,
        'invalid': 0,
        'errors': [],
        'by_category': defaultdict(int),
        'high_alert': 0,
        'black_box': 0,
        'controlled_substances': 0
    }

    print("🔍 Validating medications...")
    print()

    for i, med_file in enumerate(sorted(all_medications), 1):
        is_valid, errors = validate_medication_file(med_file)

        # Get category from path
        parts = med_file.relative_to(BASE_DIR).parts
        category = parts[0] if len(parts) > 0 else 'unknown'

        if is_valid:
            results['valid'] += 1
            status = "✅"

            # Read file to check for high-alert and black box
            try:
                with open(med_file, 'r') as f:
                    content = f.read()
                    yaml_start = 0
                    for line in content.split('\n'):
                        if line.strip() and not line.strip().startswith('#'):
                            yaml_start = content.index(line)
                            break
                    data = yaml.safe_load(content[yaml_start:])

                    if data and 'classification' in data:
                        if data['classification'].get('highAlert'):
                            results['high_alert'] += 1
                        if data['classification'].get('blackBoxWarning'):
                            results['black_box'] += 1

                    # Check for controlled substance indicators in dosing info
                    if data and 'adultDosing' in data:
                        dose_str = str(data['adultDosing']).lower()
                        if 'schedule' in dose_str:
                            results['controlled_substances'] += 1

            except:
                pass

        else:
            results['invalid'] += 1
            status = "❌"
            results['errors'].append({
                'file': str(med_file.relative_to(BASE_DIR)),
                'errors': errors
            })

        results['by_category'][category] += 1

        # Print progress
        if results['invalid'] > 0 and not is_valid:
            print(f"{status} {i:3d}. {med_file.name:50s} {category:20s}")
            for error in errors:
                print(f"     ⚠️  {error}")
        elif i % 10 == 0:
            print(f"   ... validated {i}/{len(all_medications)} medications")

    return results

def print_summary(results: Dict):
    """Print validation summary"""
    print()
    print("=" * 70)
    print("📊 VALIDATION SUMMARY")
    print("=" * 70)
    print()

    # Overall stats
    pass_rate = (results['valid'] / results['total'] * 100) if results['total'] > 0 else 0
    print(f"📈 Overall Statistics:")
    print(f"   Total medications: {results['total']}")
    print(f"   Valid: {results['valid']} ✅")
    print(f"   Invalid: {results['invalid']} ❌")
    print(f"   Pass rate: {pass_rate:.1f}%")
    print()

    # Category breakdown
    print(f"📂 Medications by Category:")
    for category, count in sorted(results['by_category'].items()):
        print(f"   {category:25s}: {count:3d} medications")
    print()

    # Safety classifications
    print(f"⚠️  Safety Classifications:")
    print(f"   High-Alert medications (ISMP): {results['high_alert']}")
    print(f"   Black Box warnings (FDA): {results['black_box']}")
    print(f"   Controlled substances (DEA): {results['controlled_substances']}")
    print()

    # Errors
    if results['errors']:
        print(f"❌ VALIDATION ERRORS ({len(results['errors'])} files):")
        print()
        for error_info in results['errors']:
            print(f"   📄 {error_info['file']}")
            for error in error_info['errors']:
                print(f"      • {error}")
            print()

    # Final status
    if results['invalid'] == 0:
        print("✅ ALL MEDICATIONS VALIDATED SUCCESSFULLY!")
    else:
        print(f"⚠️  {results['invalid']} MEDICATIONS FAILED VALIDATION")

    print()

def generate_statistics_report(results: Dict):
    """Generate detailed statistics report"""
    report_path = BASE_DIR / "database_statistics.txt"

    with open(report_path, 'w') as f:
        f.write("MEDICATION DATABASE STATISTICS REPORT\n")
        f.write("=" * 70 + "\n")
        f.write(f"Generated: {str(Path.cwd())}\n")
        f.write("\n")

        f.write("OVERALL STATISTICS\n")
        f.write("-" * 70 + "\n")
        f.write(f"Total medications: {results['total']}\n")
        f.write(f"Valid medications: {results['valid']}\n")
        f.write(f"Invalid medications: {results['invalid']}\n")
        pass_rate = (results['valid'] / results['total'] * 100) if results['total'] > 0 else 0
        f.write(f"Validation pass rate: {pass_rate:.1f}%\n")
        f.write("\n")

        f.write("CATEGORY BREAKDOWN\n")
        f.write("-" * 70 + "\n")
        for category, count in sorted(results['by_category'].items()):
            f.write(f"{category:25s}: {count:3d} medications\n")
        f.write("\n")

        f.write("SAFETY CLASSIFICATIONS\n")
        f.write("-" * 70 + "\n")
        f.write(f"High-Alert medications (ISMP): {results['high_alert']}\n")
        f.write(f"Black Box warnings (FDA): {results['black_box']}\n")
        f.write(f"Controlled substances (DEA): {results['controlled_substances']}\n")
        f.write("\n")

        if results['errors']:
            f.write("VALIDATION ERRORS\n")
            f.write("-" * 70 + "\n")
            for error_info in results['errors']:
                f.write(f"\nFile: {error_info['file']}\n")
                for error in error_info['errors']:
                    f.write(f"  - {error}\n")

    print(f"📄 Detailed report saved to: {report_path.name}")

def main():
    """Main validation function"""
    results = scan_medication_database()
    print_summary(results)
    generate_statistics_report(results)

    # Return exit code
    return 0 if results['invalid'] == 0 else 1

if __name__ == "__main__":
    exit(main())
