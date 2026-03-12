"""
CQL Gap Detector - Bidirectional gap detection between facts, CQL, and registry.

This module enables gap detection in THREE directions:

1. FORWARD GAP: Extracted fact has no CQL implementing it
   → Create backlog item for CQL authoring

2. BACKWARD GAP: CQL define exists but has no guideline source documented
   → Flag for provenance remediation

3. COVERAGE GAP: Guideline exists but not all fact types extracted
   → Queue additional extraction passes for missing KB types

Key Insight: The registry is institutional knowledge. Anyone can parse PDFs.
Knowing exactly which clinical logic depends on which guideline sentence —
that's the moat after CR-IR.

Usage:
    detector = CQLGapDetector(registry_path, vaidshala_path)
    forward_gaps = detector.detect_forward_gaps(extracted_facts)
    backward_gaps = detector.detect_backward_gaps()
    coverage_gaps = detector.detect_coverage_gaps("KDIGO 2022 §4.1")
"""

import re
import yaml
import json
from pathlib import Path
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class ForwardGap:
    """Extracted fact with no CQL implementing it."""
    gap_type: str = "FORWARD_GAP"
    drug_name: str = ""
    drug_rxnorm: str = ""
    guideline_authority: str = ""
    guideline_section: str = ""
    fact_type: str = ""  # dosing, safety, monitoring
    action: str = "Create CQL authoring backlog item"


@dataclass
class BackwardGap:
    """CQL define with no guideline source documented."""
    gap_type: str = "BACKWARD_GAP"
    cql_file: str = ""
    cql_define: str = ""
    cql_line: Optional[int] = None
    action: str = "Document guideline source or mark as institutional policy"


@dataclass
class CoverageGap:
    """Guideline section with incomplete KB extraction."""
    gap_type: str = "COVERAGE_GAP"
    guideline_authority: str = ""
    guideline_section: str = ""
    required_kb_types: list = field(default_factory=list)
    extracted_kb_types: list = field(default_factory=list)
    missing_kb_types: list = field(default_factory=list)
    action: str = "Queue additional extraction passes"


@dataclass
class GapReport:
    """Complete gap detection report."""
    forward_gaps: list[ForwardGap] = field(default_factory=list)
    backward_gaps: list[BackwardGap] = field(default_factory=list)
    coverage_gaps: list[CoverageGap] = field(default_factory=list)

    def to_dict(self) -> dict:
        """Convert report to dictionary for JSON serialization."""
        return {
            "forward_gaps": [
                {
                    "type": g.gap_type,
                    "drug": g.drug_name,
                    "rxnorm": g.drug_rxnorm,
                    "guideline": f"{g.guideline_authority} {g.guideline_section}",
                    "fact_type": g.fact_type,
                    "action": g.action,
                }
                for g in self.forward_gaps
            ],
            "backward_gaps": [
                {
                    "type": g.gap_type,
                    "cql_file": g.cql_file,
                    "cql_define": g.cql_define,
                    "cql_line": g.cql_line,
                    "action": g.action,
                }
                for g in self.backward_gaps
            ],
            "coverage_gaps": [
                {
                    "type": g.gap_type,
                    "guideline": f"{g.guideline_authority} {g.guideline_section}",
                    "required_kbs": g.required_kb_types,
                    "extracted_kbs": g.extracted_kb_types,
                    "missing_kbs": g.missing_kb_types,
                    "action": g.action,
                }
                for g in self.coverage_gaps
            ],
            "summary": {
                "total_forward_gaps": len(self.forward_gaps),
                "total_backward_gaps": len(self.backward_gaps),
                "total_coverage_gaps": len(self.coverage_gaps),
                "total_gaps": (
                    len(self.forward_gaps)
                    + len(self.backward_gaps)
                    + len(self.coverage_gaps)
                ),
            },
        }


class CQLGapDetector:
    """
    Detect gaps between extracted facts, CQL logic, and registry.

    This is the bidirectional traceability engine that ensures:
    - Every extracted fact has CQL implementing it
    - Every CQL define has documented guideline source
    - Every guideline has all KB types extracted
    """

    def __init__(self, registry_path: Path, vaidshala_path: Path):
        """
        Initialize the gap detector.

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

    def detect_forward_gaps(self, extracted_facts: dict) -> list[ForwardGap]:
        """
        Find extracted facts with no CQL implementing them.

        Forward Gap: You extracted a fact from KDIGO §4.3.2, but no CQL
        define in the registry claims to implement that section.

        Args:
            extracted_facts: Extracted facts from L3

        Returns:
            List of ForwardGap objects
        """
        gaps = []

        # Handle KB-1 dosing facts
        for drug in extracted_facts.get("drugs", []):
            governance = drug.get("governance", {})
            drug_name = drug.get("drugName", drug.get("drug_name", "Unknown"))
            drug_rxnorm = drug.get("rxnormCode", drug.get("rxnorm_code", ""))

            authority = governance.get(
                "sourceAuthority", governance.get("source_authority", "")
            )
            section = governance.get(
                "sourceSection", governance.get("source_section", "")
            )

            if authority and section:
                implementing_cql = self._find_cql_for_guideline(authority, section)
                if not implementing_cql:
                    gaps.append(
                        ForwardGap(
                            drug_name=drug_name,
                            drug_rxnorm=drug_rxnorm,
                            guideline_authority=authority,
                            guideline_section=section,
                            fact_type="dosing",
                        )
                    )

        # Handle KB-4 safety facts
        for ci in extracted_facts.get("contraindications", []):
            governance = ci.get("governance", {})
            drug_name = ci.get("drugName", ci.get("drug_name", "Unknown"))
            drug_rxnorm = ci.get("rxnormCode", ci.get("rxnorm_code", ""))

            authority = governance.get(
                "sourceAuthority", governance.get("source_authority", "")
            )
            section = governance.get(
                "sourceSection", governance.get("source_section", "")
            )

            if authority and section:
                implementing_cql = self._find_cql_for_guideline(authority, section)
                if not implementing_cql:
                    gaps.append(
                        ForwardGap(
                            drug_name=drug_name,
                            drug_rxnorm=drug_rxnorm,
                            guideline_authority=authority,
                            guideline_section=section,
                            fact_type="safety",
                        )
                    )

        # Handle KB-16 monitoring facts
        for req in extracted_facts.get(
            "labRequirements", extracted_facts.get("lab_requirements", [])
        ):
            governance = req.get("governance", {})
            drug_name = req.get("drugName", req.get("drug_name", "Unknown"))
            drug_rxnorm = req.get("rxnormCode", req.get("rxnorm_code", ""))

            authority = governance.get(
                "sourceAuthority", governance.get("source_authority", "")
            )
            section = governance.get(
                "sourceSection", governance.get("source_section", "")
            )

            if authority and section:
                implementing_cql = self._find_cql_for_guideline(authority, section)
                if not implementing_cql:
                    gaps.append(
                        ForwardGap(
                            drug_name=drug_name,
                            drug_rxnorm=drug_rxnorm,
                            guideline_authority=authority,
                            guideline_section=section,
                            fact_type="monitoring",
                        )
                    )

        return gaps

    def detect_backward_gaps(self) -> list[BackwardGap]:
        """
        Find CQL defines with no guideline source documented.

        Backward Gap: A CQL define "CustomHospitalRule" exists in vaidshala
        but has no registry entry documenting its source.

        Returns:
            List of BackwardGap objects
        """
        gaps = []

        # Get all CQL defines from vaidshala
        all_cql_defines = self._extract_all_cql_defines()

        # Get all CQL defines in registry
        registered_defines = {
            (e["cql_file"], e["cql_define"]): e.get("cql_line")
            for e in self.registry.get("entries", [])
        }

        for (cql_file, define_name), line_num in all_cql_defines.items():
            if (cql_file, define_name) not in registered_defines:
                gaps.append(
                    BackwardGap(
                        cql_file=cql_file,
                        cql_define=define_name,
                        cql_line=line_num,
                    )
                )

        return gaps

    def detect_coverage_gaps(
        self,
        guideline_section: str,
        extracted_kb_types: Optional[list[str]] = None,
    ) -> list[CoverageGap]:
        """
        Find KB types not yet extracted for a guideline.

        Coverage Gap: KDIGO §4.1.1 has been extracted for KB-1 dosing facts,
        but the registry shows CQL also expects KB-4 safety facts from this
        section, which haven't been extracted yet.

        Args:
            guideline_section: Section identifier (e.g., "4.1.1", "Recommendation 4.1.1")
            extracted_kb_types: List of KB types already extracted for this section

        Returns:
            List of CoverageGap objects
        """
        if extracted_kb_types is None:
            extracted_kb_types = []

        gaps = []

        # Find all KB types required by CQL for this guideline section
        required_kb_types = set()
        guideline_authority = ""

        for entry in self.registry.get("entries", []):
            entry_section = entry.get("guideline", {}).get("section", "")
            if guideline_section in entry_section or entry_section in guideline_section:
                guideline_authority = entry.get("guideline", {}).get("authority", "")
                for consume in entry.get("consumes", []):
                    required_kb_types.add(consume.get("kb", ""))

        if required_kb_types:
            missing = required_kb_types - set(extracted_kb_types)
            if missing:
                gaps.append(
                    CoverageGap(
                        guideline_authority=guideline_authority,
                        guideline_section=guideline_section,
                        required_kb_types=sorted(list(required_kb_types)),
                        extracted_kb_types=extracted_kb_types,
                        missing_kb_types=sorted(list(missing)),
                    )
                )

        return gaps

    def _find_cql_for_guideline(self, authority: str, section: str) -> list[dict]:
        """Find registry entries that implement a guideline section."""
        matches = []
        for entry in self.registry.get("entries", []):
            entry_guideline = entry.get("guideline", {})
            entry_authority = entry_guideline.get("authority", "")
            entry_section = entry_guideline.get("section", "")

            # Match authority and check if section is referenced
            if entry_authority == authority:
                if section in entry_section or entry_section in section:
                    matches.append(entry)

        return matches

    def _extract_all_cql_defines(self) -> dict[tuple[str, str], int]:
        """
        Extract all define statements from CQL files.

        Returns:
            Dict mapping (cql_file, define_name) to line number
        """
        defines = {}
        cql_dir = (
            self.vaidshala_path
            / "clinical-knowledge-core"
            / "tier-4-guidelines"
            / "clinical"
        )

        if not cql_dir.exists():
            # Try alternative paths
            alt_paths = [
                self.vaidshala_path / "tier-4-guidelines" / "clinical",
                self.vaidshala_path / "clinical",
            ]
            for alt in alt_paths:
                if alt.exists():
                    cql_dir = alt
                    break

        if cql_dir.exists():
            for cql_file in cql_dir.glob("*.cql"):
                content = cql_file.read_text()
                for i, line in enumerate(content.split("\n"), 1):
                    # Match: define "DefineName":
                    match = re.match(r'\s*define\s+"([^"]+)":', line)
                    if match:
                        defines[(cql_file.name, match.group(1))] = i

        return defines

    def generate_full_report(
        self,
        extracted_facts: Optional[dict] = None,
        guideline_sections: Optional[list[str]] = None,
    ) -> GapReport:
        """
        Generate a complete gap report.

        Args:
            extracted_facts: Optional extracted facts for forward gap detection
            guideline_sections: Optional list of sections for coverage gap detection

        Returns:
            Complete GapReport with all gap types
        """
        report = GapReport()

        # Forward gaps (if facts provided)
        if extracted_facts:
            report.forward_gaps = self.detect_forward_gaps(extracted_facts)

        # Backward gaps (always run)
        report.backward_gaps = self.detect_backward_gaps()

        # Coverage gaps (if sections provided)
        if guideline_sections:
            for section in guideline_sections:
                gaps = self.detect_coverage_gaps(section)
                report.coverage_gaps.extend(gaps)

        return report


# CLI interface
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="Detect gaps between extracted facts, CQL logic, and registry"
    )
    subparsers = parser.add_subparsers(dest="command", help="Gap detection command")

    # Forward gap command
    forward_parser = subparsers.add_parser(
        "forward", help="Detect forward gaps (facts without CQL)"
    )
    forward_parser.add_argument(
        "--registry", type=Path, required=True, help="Path to registry YAML"
    )
    forward_parser.add_argument(
        "--vaidshala", type=Path, required=True, help="Path to vaidshala"
    )
    forward_parser.add_argument(
        "--facts", type=Path, required=True, help="Path to extracted facts JSON"
    )

    # Backward gap command
    backward_parser = subparsers.add_parser(
        "backward", help="Detect backward gaps (CQL without documented source)"
    )
    backward_parser.add_argument(
        "--registry", type=Path, required=True, help="Path to registry YAML"
    )
    backward_parser.add_argument(
        "--vaidshala", type=Path, required=True, help="Path to vaidshala"
    )

    # Coverage gap command
    coverage_parser = subparsers.add_parser(
        "coverage", help="Detect coverage gaps (incomplete KB extraction)"
    )
    coverage_parser.add_argument(
        "--registry", type=Path, required=True, help="Path to registry YAML"
    )
    coverage_parser.add_argument(
        "--vaidshala", type=Path, required=True, help="Path to vaidshala"
    )
    coverage_parser.add_argument(
        "--guideline", type=str, required=True, help="Guideline section to check"
    )
    coverage_parser.add_argument(
        "--extracted", nargs="*", default=[], help="KB types already extracted"
    )

    # Full report command
    full_parser = subparsers.add_parser("full", help="Generate full gap report")
    full_parser.add_argument(
        "--registry", type=Path, required=True, help="Path to registry YAML"
    )
    full_parser.add_argument(
        "--vaidshala", type=Path, required=True, help="Path to vaidshala"
    )
    full_parser.add_argument(
        "--facts", type=Path, help="Path to extracted facts JSON"
    )

    args = parser.parse_args()

    if args.command == "forward":
        detector = CQLGapDetector(args.registry, args.vaidshala)
        facts = json.loads(args.facts.read_text())
        gaps = detector.detect_forward_gaps(facts)
        print(json.dumps([g.__dict__ for g in gaps], indent=2))

    elif args.command == "backward":
        detector = CQLGapDetector(args.registry, args.vaidshala)
        gaps = detector.detect_backward_gaps()
        print(json.dumps([g.__dict__ for g in gaps], indent=2))

    elif args.command == "coverage":
        detector = CQLGapDetector(args.registry, args.vaidshala)
        gaps = detector.detect_coverage_gaps(args.guideline, args.extracted)
        print(json.dumps([g.__dict__ for g in gaps], indent=2))

    elif args.command == "full":
        detector = CQLGapDetector(args.registry, args.vaidshala)
        facts = json.loads(args.facts.read_text()) if args.facts else None
        report = detector.generate_full_report(facts)
        print(json.dumps(report.to_dict(), indent=2))

    else:
        parser.print_help()
