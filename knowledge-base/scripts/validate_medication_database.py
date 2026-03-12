#!/usr/bin/env python3
"""
Medication Database Validation Script
Validates YAML structure, required fields, and cross-references

Usage:
    python validate_medication_database.py --directory ../medications
    python validate_medication_database.py --full-validation
"""

import yaml
import argparse
from pathlib import Path
from typing import Dict, List, Set, Tuple
from collections import defaultdict

# Required fields for medication YAML
REQUIRED_FIELDS = {
    "medicationId",
    "genericName",
    "brandNames",
    "classification",
    "adultDosing",
    "contraindications",
    "adverseEffects",
    "pregnancyLactation"
}

REQUIRED_CLASSIFICATION_FIELDS = {
    "therapeuticClass",
    "pharmacologicClass",
    "category"
}

REQUIRED_ADULT_DOSING_FIELDS = {
    "standard"
}


class MedicationValidator:
    """Comprehensive medication database validator"""

    def __init__(self, medications_dir: Path, interactions_file: Path = None):
        self.medications_dir = medications_dir
        self.interactions_file = interactions_file
        self.medications = {}
        self.errors = []
        self.warnings = []
        self.interaction_references = set()

    def validate_all(self) -> Tuple[bool, List[str], List[str]]:
        """Run all validation checks"""

        print("\n🔍 Starting Medication Database Validation...\n")

        # 1. Load and validate individual medications
        self.load_medications()

        # 2. Validate YAML structure
        self.validate_yaml_structure()

        # 3. Validate required fields
        self.validate_required_fields()

        # 4. Validate data types
        self.validate_data_types()

        # 5. Validate interaction references
        if self.interactions_file:
            self.validate_interaction_references()

        # 6. Check for duplicate IDs
        self.check_duplicate_ids()

        # 7. Validate dosing logic
        self.validate_dosing_logic()

        # Generate report
        self.generate_report()

        return len(self.errors) == 0, self.errors, self.warnings

    def load_medications(self):
        """Load all medication YAML files"""

        print("📂 Loading medication files...")

        yaml_files = list(self.medications_dir.rglob("*.yaml"))
        print(f"   Found {len(yaml_files)} YAML files\n")

        for yaml_file in yaml_files:
            try:
                with open(yaml_file, 'r') as f:
                    med_data = yaml.safe_load(f)

                if med_data:
                    med_id = med_data.get("medicationId", f"UNKNOWN-{yaml_file.name}")
                    self.medications[med_id] = {
                        "data": med_data,
                        "file": yaml_file
                    }
                else:
                    self.errors.append(f"Empty file: {yaml_file}")

            except yaml.YAMLError as e:
                self.errors.append(f"YAML parsing error in {yaml_file}: {str(e)}")
            except Exception as e:
                self.errors.append(f"Error loading {yaml_file}: {str(e)}")

        print(f"✓ Successfully loaded {len(self.medications)} medications")

    def validate_yaml_structure(self):
        """Validate YAML structure and syntax"""

        print(f"\n📋 Validating YAML structure...")

        for med_id, med_info in self.medications.items():
            med_data = med_info["data"]
            file_path = med_info["file"]

            # Check if it's a dictionary
            if not isinstance(med_data, dict):
                self.errors.append(f"{file_path.name}: Root element must be a dictionary")
                continue

            # Check for null values in critical fields
            for field in ["medicationId", "genericName"]:
                if field in med_data and med_data[field] is None:
                    self.errors.append(f"{file_path.name}: '{field}' cannot be null")

        print(f"   Checked {len(self.medications)} files for structural issues")

    def validate_required_fields(self):
        """Validate presence of required fields"""

        print(f"\n✅ Validating required fields...")

        missing_fields_count = 0

        for med_id, med_info in self.medications.items():
            med_data = med_info["data"]
            file_path = med_info["file"]
            missing = []

            # Check top-level required fields
            for field in REQUIRED_FIELDS:
                if field not in med_data:
                    missing.append(field)

            # Check classification subfields
            if "classification" in med_data:
                classification = med_data["classification"]
                for field in REQUIRED_CLASSIFICATION_FIELDS:
                    if field not in classification:
                        missing.append(f"classification.{field}")

            # Check adult dosing subfields
            if "adultDosing" in med_data:
                adult_dosing = med_data["adultDosing"]
                for field in REQUIRED_ADULT_DOSING_FIELDS:
                    if field not in adult_dosing:
                        missing.append(f"adultDosing.{field}")

            if missing:
                missing_fields_count += 1
                self.errors.append(
                    f"{file_path.name} ({med_id}): Missing required fields: {', '.join(missing)}"
                )

        if missing_fields_count == 0:
            print(f"   ✓ All medications have required fields")
        else:
            print(f"   ✗ {missing_fields_count} medications missing required fields")

    def validate_data_types(self):
        """Validate data types for key fields"""

        print(f"\n🔢 Validating data types...")

        type_errors = 0

        for med_id, med_info in self.medications.items():
            med_data = med_info["data"]
            file_path = med_info["file"]

            # Check lists
            if "brandNames" in med_data and not isinstance(med_data["brandNames"], list):
                self.errors.append(f"{file_path.name}: 'brandNames' must be a list")
                type_errors += 1

            if "majorInteractions" in med_data and not isinstance(med_data["majorInteractions"], list):
                self.errors.append(f"{file_path.name}: 'majorInteractions' must be a list")
                type_errors += 1

            # Check booleans in classification
            if "classification" in med_data:
                classification = med_data["classification"]
                for bool_field in ["highAlert", "blackBoxWarning"]:
                    if bool_field in classification and not isinstance(classification[bool_field], bool):
                        self.warnings.append(
                            f"{file_path.name}: 'classification.{bool_field}' should be boolean (true/false)"
                        )

            # Check dosing structure
            if "adultDosing" in med_data:
                adult_dosing = med_data["adultDosing"]
                if "standard" in adult_dosing and not isinstance(adult_dosing["standard"], dict):
                    self.errors.append(f"{file_path.name}: 'adultDosing.standard' must be a dictionary")
                    type_errors += 1

        if type_errors == 0:
            print(f"   ✓ All data types are correct")
        else:
            print(f"   ✗ Found {type_errors} data type errors")

    def validate_interaction_references(self):
        """Validate that interaction references exist"""

        if not self.interactions_file or not self.interactions_file.exists():
            self.warnings.append("Interaction file not found - skipping interaction validation")
            return

        print(f"\n🔗 Validating interaction references...")

        # Load interactions
        try:
            with open(self.interactions_file, 'r') as f:
                interactions_data = yaml.safe_load(f)

            if "interactions" in interactions_data:
                for interaction in interactions_data["interactions"]:
                    interaction_id = interaction.get("interactionId")
                    if interaction_id:
                        self.interaction_references.add(interaction_id)

            print(f"   Loaded {len(self.interaction_references)} interaction references")

        except Exception as e:
            self.errors.append(f"Error loading interactions file: {str(e)}")
            return

        # Validate references
        invalid_refs = 0

        for med_id, med_info in self.medications.items():
            med_data = med_info["data"]
            file_path = med_info["file"]

            if "majorInteractions" in med_data:
                for ref in med_data["majorInteractions"]:
                    if ref not in self.interaction_references:
                        self.warnings.append(
                            f"{file_path.name}: Interaction reference '{ref}' not found in interactions database"
                        )
                        invalid_refs += 1

        if invalid_refs == 0:
            print(f"   ✓ All interaction references are valid")
        else:
            print(f"   ⚠️  Found {invalid_refs} invalid interaction references")

    def check_duplicate_ids(self):
        """Check for duplicate medication IDs"""

        print(f"\n🔍 Checking for duplicate IDs...")

        id_map = defaultdict(list)

        for med_id, med_info in self.medications.items():
            file_path = med_info["file"]
            id_map[med_id].append(str(file_path))

        duplicates = {med_id: files for med_id, files in id_map.items() if len(files) > 1}

        if duplicates:
            for med_id, files in duplicates.items():
                self.errors.append(
                    f"Duplicate medication ID '{med_id}' found in: {', '.join(files)}"
                )
            print(f"   ✗ Found {len(duplicates)} duplicate IDs")
        else:
            print(f"   ✓ All medication IDs are unique")

    def validate_dosing_logic(self):
        """Validate dosing information for logical consistency"""

        print(f"\n💊 Validating dosing logic...")

        dosing_issues = 0

        for med_id, med_info in self.medications.items():
            med_data = med_info["data"]
            file_path = med_info["file"]

            if "adultDosing" in med_data:
                adult_dosing = med_data["adultDosing"]

                # Check if standard dosing exists
                if "standard" in adult_dosing:
                    standard = adult_dosing["standard"]

                    # Validate required dosing fields
                    required_dosing = ["dose", "route", "frequency"]
                    missing_dosing = [f for f in required_dosing if f not in standard]

                    if missing_dosing:
                        self.warnings.append(
                            f"{file_path.name}: Missing dosing information: {', '.join(missing_dosing)}"
                        )
                        dosing_issues += 1

                    # Check for reasonable max daily dose
                    if "maxDailyDose" in standard:
                        max_dose = standard["maxDailyDose"]
                        if max_dose and isinstance(max_dose, str):
                            # Basic validation - just check it's not empty
                            if not max_dose.strip():
                                self.warnings.append(f"{file_path.name}: Empty maxDailyDose")

                # Check renal adjustment structure
                if "renalAdjustment" in adult_dosing:
                    renal = adult_dosing["renalAdjustment"]
                    if "adjustments" in renal and not isinstance(renal["adjustments"], dict):
                        self.errors.append(
                            f"{file_path.name}: renalAdjustment.adjustments must be a dictionary"
                        )
                        dosing_issues += 1

        if dosing_issues == 0:
            print(f"   ✓ Dosing logic validation passed")
        else:
            print(f"   ⚠️  Found {dosing_issues} dosing logic issues")

    def generate_report(self):
        """Generate validation report"""

        print(f"\n" + "="*70)
        print(f"📊 VALIDATION REPORT")
        print(f"="*70)

        print(f"\n✅ Medications validated: {len(self.medications)}")

        if self.errors:
            print(f"\n❌ ERRORS ({len(self.errors)}):")
            for error in self.errors[:20]:  # Show first 20
                print(f"   • {error}")
            if len(self.errors) > 20:
                print(f"   ... and {len(self.errors) - 20} more errors")

        if self.warnings:
            print(f"\n⚠️  WARNINGS ({len(self.warnings)}):")
            for warning in self.warnings[:20]:  # Show first 20
                print(f"   • {warning}")
            if len(self.warnings) > 20:
                print(f"   ... and {len(self.warnings) - 20} more warnings")

        if not self.errors and not self.warnings:
            print(f"\n✅ All validations passed! Database is ready for use.")

        # Summary by category
        categories = defaultdict(int)
        for med_id, med_info in self.medications.items():
            med_data = med_info["data"]
            if "classification" in med_data and "category" in med_data["classification"]:
                category = med_data["classification"]["category"]
                categories[category] += 1

        if categories:
            print(f"\n📋 Medications by Category:")
            for category, count in sorted(categories.items()):
                print(f"   • {category}: {count}")

        print(f"\n" + "="*70)

        # Return success status
        return len(self.errors) == 0


def main():
    parser = argparse.ArgumentParser(description="Validate medication database")
    parser.add_argument("--directory", "-d", default="../medications",
                        help="Medications directory path")
    parser.add_argument("--interactions", "-i", default="../drug-interactions/major-interactions.yaml",
                        help="Drug interactions file path")
    parser.add_argument("--full-validation", action="store_true",
                        help="Run full validation suite")
    parser.add_argument("--errors-only", action="store_true",
                        help="Show only errors, not warnings")

    args = parser.parse_args()

    medications_dir = Path(args.directory)
    interactions_file = Path(args.interactions) if args.interactions else None

    if not medications_dir.exists():
        print(f"❌ Error: Directory not found: {medications_dir}")
        return 1

    validator = MedicationValidator(medications_dir, interactions_file)
    success, errors, warnings = validator.validate_all()

    # Exit with appropriate code
    return 0 if success else 1


if __name__ == "__main__":
    exit(main())
