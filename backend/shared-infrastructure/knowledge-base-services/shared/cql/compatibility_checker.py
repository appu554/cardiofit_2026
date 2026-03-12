"""
CQL Compatibility Checker - Validate extracted facts against CQL expectations.

This module implements L5 of the 7-Layer Guideline Curation Pipeline.
It validates that extracted facts from L3 are compatible with the CQL
logic in vaidshala/tier-4-guidelines/.

Key Principle: CQL already exists. We validate compatibility, not generate CQL.

The checker uses the CQL Guideline Registry (cql_guideline_registry.yaml)
as the source of truth for expected thresholds. This is the manual mapping
approach for MVP phase.

For production, this can be upgraded to use ELM-based index for automated
threshold extraction from compiled CQL.

Usage:
    checker = CQLCompatibilityChecker(registry_path, vaidshala_path)
    result = checker.check_compatibility(extracted_facts, "T2DMGuidelines.cql")
"""

import yaml
import json
import re
from pathlib import Path
from typing import Optional, Any
from dataclasses import dataclass, field


@dataclass
class CompatibilityMatch:
    """A successful match between extracted fact and CQL expectation."""
    cql_define: str
    cql_line: int
    guideline_section: str
    extracted_value: Any
    expected_value: Any
    status: str = "ALIGNED"


@dataclass
class CompatibilityIssue:
    """An issue found during compatibility checking."""
    issue_type: str  # THRESHOLD_MISMATCH, FACT_NOT_FOUND, REGISTRY_MISSING
    cql_define: str
    cql_file: str
    cql_line: Optional[int]
    expected_value: Optional[Any]
    extracted_value: Optional[Any]
    message: str
    severity: str = "WARNING"  # WARNING, ERROR


@dataclass
class CompatibilityReport:
    """Complete compatibility check report."""
    compatible: bool
    cql_file: str
    registry_version: str
    matches: list[CompatibilityMatch] = field(default_factory=list)
    issues: list[CompatibilityIssue] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)

    def to_dict(self) -> dict:
        """Convert report to dictionary for JSON serialization."""
        return {
            "compatible": self.compatible,
            "cql_file": self.cql_file,
            "registry_version": self.registry_version,
            "matches": [
                {
                    "cql_define": m.cql_define,
                    "cql_line": m.cql_line,
                    "guideline": m.guideline_section,
                    "extracted_value": m.extracted_value,
                    "expected_value": m.expected_value,
                    "status": m.status,
                }
                for m in self.matches
            ],
            "issues": [
                {
                    "type": i.issue_type,
                    "cql_define": i.cql_define,
                    "cql_file": i.cql_file,
                    "cql_line": i.cql_line,
                    "expected": i.expected_value,
                    "extracted": i.extracted_value,
                    "message": i.message,
                    "severity": i.severity,
                }
                for i in self.issues
            ],
            "warnings": self.warnings,
            "summary": {
                "total_matches": len(self.matches),
                "total_issues": len(self.issues),
                "threshold_mismatches": len(
                    [i for i in self.issues if i.issue_type == "THRESHOLD_MISMATCH"]
                ),
                "facts_not_found": len(
                    [i for i in self.issues if i.issue_type == "FACT_NOT_FOUND"]
                ),
            },
        }


class CQLCompatibilityChecker:
    """
    L5: Validate extracted facts against CQL threshold expectations.

    Uses the CQL Guideline Registry (manual mapping) instead of regex parsing
    for reliability. The registry is human-curated from CQL file review.

    Attributes:
        registry: Loaded registry YAML
        vaidshala_path: Path to vaidshala CQL directory
    """

    def __init__(self, registry_path: Path, vaidshala_path: Path):
        """
        Initialize the compatibility checker.

        Args:
            registry_path: Path to cql_guideline_registry.yaml
            vaidshala_path: Path to vaidshala root directory
        """
        self.vaidshala_path = Path(vaidshala_path)
        self.registry_path = Path(registry_path)

        if not self.registry_path.exists():
            raise FileNotFoundError(f"Registry not found: {registry_path}")

        with open(self.registry_path) as f:
            self.registry = yaml.safe_load(f)

    def check_compatibility(
        self,
        extracted_facts: dict,
        target_cql: str,
    ) -> CompatibilityReport:
        """
        Check if extracted facts are compatible with CQL expectations.

        Args:
            extracted_facts: Extracted facts from L3 (KB1, KB4, or KB16 format)
            target_cql: CQL file name to check against (e.g., "T2DMGuidelines.cql")

        Returns:
            CompatibilityReport with matches, issues, and compatibility status
        """
        # Get registry entries for this CQL file
        cql_entries = [
            e for e in self.registry.get("entries", [])
            if e.get("cql_file") == target_cql
        ]

        if not cql_entries:
            return CompatibilityReport(
                compatible=False,
                cql_file=target_cql,
                registry_version=self.registry.get("registry_version", "unknown"),
                issues=[
                    CompatibilityIssue(
                        issue_type="REGISTRY_MISSING",
                        cql_define="*",
                        cql_file=target_cql,
                        cql_line=None,
                        expected_value=None,
                        extracted_value=None,
                        message=f"CQL file not in registry: {target_cql}. "
                        f"Add entries to cql_guideline_registry.yaml",
                        severity="ERROR",
                    )
                ],
            )

        matches = []
        issues = []
        warnings = []

        for entry in cql_entries:
            match_result = self._match_fact_to_entry(extracted_facts, entry)

            if match_result["found"]:
                if match_result["matches"]:
                    matches.append(
                        CompatibilityMatch(
                            cql_define=entry["cql_define"],
                            cql_line=entry.get("cql_line", 0),
                            guideline_section=entry["guideline"].get("section", ""),
                            extracted_value=match_result["extracted_value"],
                            expected_value=match_result["expected_value"],
                            status="ALIGNED",
                        )
                    )
                else:
                    issues.append(
                        CompatibilityIssue(
                            issue_type="THRESHOLD_MISMATCH",
                            cql_define=entry["cql_define"],
                            cql_file=target_cql,
                            cql_line=entry.get("cql_line"),
                            expected_value=match_result["expected_value"],
                            extracted_value=match_result["extracted_value"],
                            message=match_result["message"],
                            severity="ERROR",
                        )
                    )
            else:
                issues.append(
                    CompatibilityIssue(
                        issue_type="FACT_NOT_FOUND",
                        cql_define=entry["cql_define"],
                        cql_file=target_cql,
                        cql_line=entry.get("cql_line"),
                        expected_value=entry.get("expected_threshold", {}).get("value"),
                        extracted_value=None,
                        message=f"No extracted fact found for {entry['cql_define']}",
                        severity="WARNING",
                    )
                )

        # Determine compatibility (only threshold mismatches are errors)
        threshold_mismatches = [
            i for i in issues if i.issue_type == "THRESHOLD_MISMATCH"
        ]
        compatible = len(threshold_mismatches) == 0

        return CompatibilityReport(
            compatible=compatible,
            cql_file=target_cql,
            registry_version=self.registry.get("registry_version", "unknown"),
            matches=matches,
            issues=issues,
            warnings=warnings,
        )

    def _match_fact_to_entry(
        self,
        facts: dict,
        entry: dict,
    ) -> dict:
        """
        Match an extracted fact to a registry entry.

        Args:
            facts: Extracted facts dictionary
            entry: Registry entry to match against

        Returns:
            Dict with found, matches, extracted_value, expected_value, message
        """
        expected_threshold = entry.get("expected_threshold", {})
        consumes = entry.get("consumes", [])

        # Determine what KB field to look for
        for consume in consumes:
            kb = consume.get("kb")
            field = consume.get("field")
            filter_str = consume.get("filter", "")

            # Parse filter to extract drug identifier
            drug_rxnorm = self._extract_from_filter(filter_str, "rxnorm_code")
            drug_class = self._extract_from_filter(filter_str, "drug_class")

            if kb == "KB-1":
                result = self._match_kb1_fact(
                    facts, drug_rxnorm, drug_class, expected_threshold
                )
                if result["found"]:
                    return result

            elif kb == "KB-4":
                result = self._match_kb4_fact(
                    facts, drug_rxnorm, drug_class, expected_threshold
                )
                if result["found"]:
                    return result

            elif kb == "KB-16":
                result = self._match_kb16_fact(
                    facts, drug_rxnorm, expected_threshold
                )
                if result["found"]:
                    return result

        return {"found": False, "matches": False, "extracted_value": None, "message": ""}

    def _match_kb1_fact(
        self,
        facts: dict,
        drug_rxnorm: Optional[str],
        drug_class: Optional[str],
        expected: dict,
    ) -> dict:
        """Match against KB-1 dosing facts."""
        for drug in facts.get("drugs", []):
            # Match by RxNorm or drug class
            if drug_rxnorm and drug.get("rxnormCode") != drug_rxnorm:
                if drug.get("rxnorm_code") != drug_rxnorm:  # Try snake_case too
                    continue
            if drug_class and drug_class.lower() not in (drug.get("drugClass", "") or "").lower():
                if drug_class.lower() not in (drug.get("drug_class", "") or "").lower():
                    continue

            adjustments = drug.get("renalAdjustments", drug.get("renal_adjustments", []))
            for adj in adjustments:
                operator = expected.get("operator", "<")
                expected_value = expected.get("value")

                if operator == "<":
                    # Looking for contraindication threshold
                    if adj.get("contraindicated"):
                        egfr_max = adj.get("egfrMax", adj.get("egfr_max"))
                        if egfr_max is not None and expected_value is not None:
                            # Allow for floating point: 29.9 ≈ 30
                            matches = abs(float(egfr_max) - float(expected_value)) < 1.0
                            return {
                                "found": True,
                                "matches": matches,
                                "extracted_value": egfr_max,
                                "expected_value": expected_value,
                                "message": "" if matches else
                                f"Expected < {expected_value}, extracted threshold {egfr_max}",
                            }

                elif operator == "between":
                    # Looking for dose adjustment range
                    egfr_min = adj.get("egfrMin", adj.get("egfr_min"))
                    egfr_max = adj.get("egfrMax", adj.get("egfr_max"))
                    exp_min = expected.get("value_min")
                    exp_max = expected.get("value_max")

                    if all(v is not None for v in [egfr_min, egfr_max, exp_min, exp_max]):
                        min_match = abs(float(egfr_min) - float(exp_min)) < 1.0
                        max_match = abs(float(egfr_max) - float(exp_max)) < 1.0
                        matches = min_match and max_match
                        return {
                            "found": True,
                            "matches": matches,
                            "extracted_value": f"{egfr_min}-{egfr_max}",
                            "expected_value": f"{exp_min}-{exp_max}",
                            "message": "" if matches else
                            f"Expected range {exp_min}-{exp_max}, got {egfr_min}-{egfr_max}",
                        }

        return {"found": False, "matches": False, "extracted_value": None, "message": ""}

    def _match_kb4_fact(
        self,
        facts: dict,
        drug_rxnorm: Optional[str],
        drug_class: Optional[str],
        expected: dict,
    ) -> dict:
        """Match against KB-4 safety facts."""
        for ci in facts.get("contraindications", []):
            # Match by RxNorm or drug class
            if drug_rxnorm and ci.get("rxnormCode") != drug_rxnorm:
                if ci.get("rxnorm_code") != drug_rxnorm:
                    continue
            if drug_class and drug_class.lower() not in (ci.get("drugClass", "") or "").lower():
                if drug_class.lower() not in (ci.get("drug_class", "") or "").lower():
                    continue

            # Check lab-based threshold
            lab_threshold = ci.get("labThreshold", ci.get("lab_threshold"))
            expected_value = expected.get("value")

            if lab_threshold is not None and expected_value is not None:
                matches = abs(float(lab_threshold) - float(expected_value)) < 0.5
                return {
                    "found": True,
                    "matches": matches,
                    "extracted_value": lab_threshold,
                    "expected_value": expected_value,
                    "message": "" if matches else
                    f"Expected threshold {expected_value}, got {lab_threshold}",
                }

        return {"found": False, "matches": False, "extracted_value": None, "message": ""}

    def _match_kb16_fact(
        self,
        facts: dict,
        drug_rxnorm: Optional[str],
        expected: dict,
    ) -> dict:
        """Match against KB-16 lab monitoring facts."""
        for req in facts.get("labRequirements", facts.get("lab_requirements", [])):
            if drug_rxnorm and req.get("rxnormCode") != drug_rxnorm:
                if req.get("rxnorm_code") != drug_rxnorm:
                    continue

            for lab in req.get("labs", []):
                # Check critical value thresholds
                critical_high = lab.get("criticalHigh", lab.get("critical_high"))
                if critical_high:
                    critical_value = critical_high.get("value")
                    expected_value = expected.get("value")

                    if critical_value is not None and expected_value is not None:
                        matches = abs(float(critical_value) - float(expected_value)) < 0.5
                        return {
                            "found": True,
                            "matches": matches,
                            "extracted_value": critical_value,
                            "expected_value": expected_value,
                            "message": "" if matches else
                            f"Expected critical value {expected_value}, got {critical_value}",
                        }

        return {"found": False, "matches": False, "extracted_value": None, "message": ""}

    def _extract_from_filter(self, filter_str: str, key: str) -> Optional[str]:
        """Extract a value from a SQL-like filter string."""
        # Pattern: key = 'value' or key = "value"
        pattern = rf"{key}\s*=\s*['\"]([^'\"]+)['\"]"
        match = re.search(pattern, filter_str, re.IGNORECASE)
        if match:
            return match.group(1)
        return None

    def get_registry_coverage(self) -> dict:
        """
        Report which CQL files are covered by the registry.

        Returns:
            Dict with covered files, uncovered files, and coverage percentage
        """
        cql_dir = (
            self.vaidshala_path
            / "clinical-knowledge-core"
            / "tier-4-guidelines"
            / "clinical"
        )

        if not cql_dir.exists():
            return {
                "covered": [],
                "uncovered": [],
                "coverage_percent": 0,
                "error": f"CQL directory not found: {cql_dir}",
            }

        all_cql_files = {f.name for f in cql_dir.glob("*.cql")}
        covered = {e["cql_file"] for e in self.registry.get("entries", [])}

        return {
            "covered": sorted(list(covered)),
            "uncovered": sorted(list(all_cql_files - covered)),
            "coverage_percent": (
                len(covered) / len(all_cql_files) * 100 if all_cql_files else 0
            ),
            "total_cql_files": len(all_cql_files),
            "total_registry_entries": len(self.registry.get("entries", [])),
        }


# CLI interface
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="Check compatibility of extracted facts against CQL registry"
    )
    parser.add_argument(
        "--registry",
        type=Path,
        required=True,
        help="Path to cql_guideline_registry.yaml",
    )
    parser.add_argument(
        "--vaidshala",
        type=Path,
        required=True,
        help="Path to vaidshala root directory",
    )
    parser.add_argument(
        "--facts",
        type=Path,
        help="Path to extracted facts JSON file",
    )
    parser.add_argument(
        "--cql",
        type=str,
        help="CQL file name to check against",
    )
    parser.add_argument(
        "--coverage",
        action="store_true",
        help="Show registry coverage report",
    )

    args = parser.parse_args()

    checker = CQLCompatibilityChecker(args.registry, args.vaidshala)

    if args.coverage:
        coverage = checker.get_registry_coverage()
        print(json.dumps(coverage, indent=2))
    elif args.facts and args.cql:
        facts = json.loads(args.facts.read_text())
        report = checker.check_compatibility(facts, args.cql)
        print(json.dumps(report.to_dict(), indent=2))
    else:
        parser.print_help()
