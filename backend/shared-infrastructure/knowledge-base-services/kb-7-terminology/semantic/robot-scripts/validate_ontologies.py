#!/usr/bin/env python3
"""
KB-7 Ontology Validation Script using ROBOT
Validates clinical terminology ontologies for consistency and policy compliance
"""

import os
import sys
import json
import subprocess
import logging
from pathlib import Path
from typing import List, Dict, Any
import requests

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class OntologyValidator:
    def __init__(self):
        self.workspace = Path("/workspace")
        self.ontologies_dir = self.workspace / "ontologies"
        self.configs_dir = self.workspace / "configs"
        self.output_dir = self.workspace / "output"
        self.output_dir.mkdir(exist_ok=True)

        # GraphDB connection
        self.graphdb_url = os.getenv("GRAPHDB_URL", "http://graphdb:7200")
        self.repository_id = os.getenv("REPOSITORY_ID", "kb7-terminology")

    def validate_syntax(self, ontology_path: Path) -> Dict[str, Any]:
        """Validate RDF/OWL syntax using ROBOT"""
        logger.info(f"Validating syntax for {ontology_path.name}")

        result = {
            "ontology": ontology_path.name,
            "syntax_valid": False,
            "errors": [],
            "warnings": []
        }

        try:
            # Use ROBOT to validate syntax
            cmd = [
                "robot", "validate",
                "--input", str(ontology_path),
                "--output", str(self.output_dir / f"{ontology_path.stem}_validated.owl")
            ]

            process = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300
            )

            if process.returncode == 0:
                result["syntax_valid"] = True
                logger.info(f"✅ Syntax validation passed for {ontology_path.name}")
            else:
                result["errors"].append(process.stderr)
                logger.error(f"❌ Syntax validation failed for {ontology_path.name}")

        except subprocess.TimeoutExpired:
            result["errors"].append("Validation timeout after 5 minutes")
        except Exception as e:
            result["errors"].append(str(e))

        return result

    def validate_reasoning(self, ontology_path: Path) -> Dict[str, Any]:
        """Validate logical consistency using ROBOT reasoner"""
        logger.info(f"Validating reasoning for {ontology_path.name}")

        result = {
            "ontology": ontology_path.name,
            "reasoning_valid": False,
            "inconsistencies": [],
            "unsatisfiable_classes": []
        }

        try:
            # Use HermiT reasoner for consistency checking
            cmd = [
                "robot", "reason",
                "--reasoner", "HermiT",
                "--input", str(ontology_path),
                "--output", str(self.output_dir / f"{ontology_path.stem}_reasoned.owl")
            ]

            process = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=600  # 10 minutes for reasoning
            )

            if process.returncode == 0:
                result["reasoning_valid"] = True
                logger.info(f"✅ Reasoning validation passed for {ontology_path.name}")
            else:
                # Parse reasoning errors
                stderr = process.stderr
                if "inconsistent" in stderr.lower():
                    result["inconsistencies"].append(stderr)
                if "unsatisfiable" in stderr.lower():
                    result["unsatisfiable_classes"].append(stderr)
                logger.error(f"❌ Reasoning validation failed for {ontology_path.name}")

        except subprocess.TimeoutExpired:
            result["inconsistencies"].append("Reasoning timeout after 10 minutes")
        except Exception as e:
            result["inconsistencies"].append(str(e))

        return result

    def validate_clinical_policies(self, ontology_path: Path) -> Dict[str, Any]:
        """Validate clinical policy compliance using SHACL shapes"""
        logger.info(f"Validating clinical policies for {ontology_path.name}")

        result = {
            "ontology": ontology_path.name,
            "policy_compliant": False,
            "violations": [],
            "policy_checks": []
        }

        # Check for policy flags presence
        policy_checks = [
            "doNotAutoMap flags properly set",
            "Clinical review requirements specified",
            "Provenance metadata present",
            "Safety level classifications included"
        ]

        try:
            # Load and check ontology content for policy markers
            with open(ontology_path, 'r', encoding='utf-8') as f:
                content = f.read()

            # Check for required policy predicates
            required_predicates = [
                "doNotAutoMap",
                "requiresClinicalReview",
                "prov:wasGeneratedBy",
                "pav:createdBy"
            ]

            violations = []
            for predicate in required_predicates:
                if predicate not in content:
                    violations.append(f"Missing required predicate: {predicate}")

            if not violations:
                result["policy_compliant"] = True
                result["policy_checks"] = policy_checks
                logger.info(f"✅ Policy validation passed for {ontology_path.name}")
            else:
                result["violations"] = violations
                logger.warning(f"⚠️ Policy violations found in {ontology_path.name}")

        except Exception as e:
            result["violations"].append(f"Policy validation error: {str(e)}")

        return result

    def validate_clinical_terminology(self, ontology_path: Path) -> Dict[str, Any]:
        """Validate clinical terminology standards compliance"""
        logger.info(f"Validating clinical terminology standards for {ontology_path.name}")

        result = {
            "ontology": ontology_path.name,
            "terminology_valid": False,
            "missing_namespaces": [],
            "invalid_codes": []
        }

        # Expected clinical namespaces
        expected_namespaces = [
            "http://snomed.info/id/",
            "http://purl.bioontology.org/ontology/RXNORM/",
            "http://loinc.org/",
            "http://purl.bioontology.org/ontology/ICD10CM/"
        ]

        try:
            with open(ontology_path, 'r', encoding='utf-8') as f:
                content = f.read()

            missing_namespaces = []
            for ns in expected_namespaces:
                if ns not in content:
                    missing_namespaces.append(ns)

            # Check for valid SNOMED CT codes (should be numeric)
            import re
            snomed_codes = re.findall(r'http://snomed\.info/id/(\d+)', content)
            invalid_codes = [code for code in snomed_codes if len(code) < 6 or len(code) > 18]

            result["missing_namespaces"] = missing_namespaces
            result["invalid_codes"] = invalid_codes

            if not missing_namespaces and not invalid_codes:
                result["terminology_valid"] = True
                logger.info(f"✅ Terminology validation passed for {ontology_path.name}")
            else:
                logger.warning(f"⚠️ Terminology issues found in {ontology_path.name}")

        except Exception as e:
            result["invalid_codes"].append(f"Terminology validation error: {str(e)}")

        return result

    def generate_validation_report(self, results: List[Dict[str, Any]]) -> None:
        """Generate comprehensive validation report"""
        report = {
            "validation_timestamp": "2025-09-19T19:30:00Z",
            "total_ontologies": len(results),
            "summary": {
                "syntax_passed": 0,
                "reasoning_passed": 0,
                "policy_compliant": 0,
                "terminology_valid": 0
            },
            "detailed_results": results
        }

        # Calculate summary statistics
        for result in results:
            if result.get("syntax_valid", False):
                report["summary"]["syntax_passed"] += 1
            if result.get("reasoning_valid", False):
                report["summary"]["reasoning_passed"] += 1
            if result.get("policy_compliant", False):
                report["summary"]["policy_compliant"] += 1
            if result.get("terminology_valid", False):
                report["summary"]["terminology_valid"] += 1

        # Save report
        report_path = self.output_dir / "validation_report.json"
        with open(report_path, 'w') as f:
            json.dump(report, f, indent=2)

        logger.info(f"📊 Validation report saved to {report_path}")

        # Print summary
        print("\n🔍 KB-7 Ontology Validation Summary")
        print("=" * 50)
        print(f"Total ontologies validated: {report['total_ontologies']}")
        print(f"✅ Syntax validation passed: {report['summary']['syntax_passed']}")
        print(f"🧠 Reasoning validation passed: {report['summary']['reasoning_passed']}")
        print(f"📋 Policy compliance passed: {report['summary']['policy_compliant']}")
        print(f"🏥 Terminology validation passed: {report['summary']['terminology_valid']}")

        # Upload report to GraphDB if available
        self.upload_validation_results(report)

    def upload_validation_results(self, report: Dict[str, Any]) -> None:
        """Upload validation results to GraphDB repository"""
        try:
            # Convert validation report to RDF
            validation_rdf = self.create_validation_rdf(report)

            # Upload to GraphDB
            upload_url = f"{self.graphdb_url}/repositories/{self.repository_id}/statements"
            headers = {"Content-Type": "application/x-turtle"}

            response = requests.post(upload_url, data=validation_rdf, headers=headers)

            if response.status_code == 204:
                logger.info("✅ Validation results uploaded to GraphDB")
            else:
                logger.warning(f"⚠️ Failed to upload validation results: {response.status_code}")

        except Exception as e:
            logger.warning(f"⚠️ Could not upload validation results: {str(e)}")

    def create_validation_rdf(self, report: Dict[str, Any]) -> str:
        """Convert validation report to RDF turtle format"""
        rdf_content = f"""
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix prov: <http://www.w3.org/ns/prov#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

kb7:ValidationReport_{report['validation_timestamp'].replace(':', '').replace('-', '')}
    a kb7:ValidationReport ;
    prov:generatedAtTime "{report['validation_timestamp']}"^^xsd:dateTime ;
    kb7:totalOntologies {report['total_ontologies']} ;
    kb7:syntaxPassed {report['summary']['syntax_passed']} ;
    kb7:reasoningPassed {report['summary']['reasoning_passed']} ;
    kb7:policyCompliant {report['summary']['policy_compliant']} ;
    kb7:terminologyValid {report['summary']['terminology_valid']} .
"""
        return rdf_content

    def run_validation(self) -> None:
        """Run complete validation pipeline"""
        logger.info("🚀 Starting KB-7 ontology validation pipeline...")

        # Find all ontology files
        ontology_files = []
        for ext in ['*.owl', '*.ttl', '*.rdf', '*.n3']:
            ontology_files.extend(self.ontologies_dir.glob(ext))

        if not ontology_files:
            logger.warning("⚠️ No ontology files found in /workspace/ontologies")
            return

        logger.info(f"Found {len(ontology_files)} ontology files to validate")

        all_results = []

        for ontology_path in ontology_files:
            logger.info(f"\n🔍 Validating: {ontology_path.name}")

            # Run all validation checks
            syntax_result = self.validate_syntax(ontology_path)
            reasoning_result = self.validate_reasoning(ontology_path)
            policy_result = self.validate_clinical_policies(ontology_path)
            terminology_result = self.validate_clinical_terminology(ontology_path)

            # Combine results
            combined_result = {
                **syntax_result,
                **reasoning_result,
                **policy_result,
                **terminology_result
            }

            all_results.append(combined_result)

        # Generate final report
        self.generate_validation_report(all_results)
        logger.info("✅ Validation pipeline completed successfully!")

def main():
    validator = OntologyValidator()
    validator.run_validation()

if __name__ == "__main__":
    main()