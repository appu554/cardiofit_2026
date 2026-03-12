"""
CQL Guideline Registry Validator - Integrity and Consistency Checks.

This module validates the CQL Guideline Registry to ensure:
1. Schema compliance: All required fields present
2. CQL file existence: Referenced CQL files exist in vaidshala
3. Line number accuracy: CQL defines exist at specified lines
4. Guideline consistency: No conflicting entries for same CQL define
5. KB coverage: All KB types properly documented

The registry is your institutional knowledge. Keeping it valid and consistent
ensures reliable bidirectional traceability between guidelines and CQL logic.

Usage:
    from registry_validator import RegistryValidator

    validator = RegistryValidator(registry_path, vaidshala_path)
    report = validator.validate_all()

    if not report.is_valid:
        for error in report.errors:
            print(f"ERROR: {error}")
"""

import re
import yaml
from pathlib import Path
from dataclasses import dataclass, field
from typing import Optional, Literal


@dataclass
class ValidationError:
    """A validation error in the registry."""
    error_type: Literal[
        "SCHEMA_ERROR",
        "FILE_NOT_FOUND",
        "DEFINE_NOT_FOUND",
        "LINE_MISMATCH",
        "DUPLICATE_ENTRY",
        "INVALID_KB_REFERENCE",
        "MISSING_GUIDELINE",
        "ORPHAN_CQL",
    ]
    message: str
    entry_index: Optional[int] = None
    cql_file: Optional[str] = None
    cql_define: Optional[str] = None
    severity: Literal["ERROR", "WARNING"] = "ERROR"

    def __str__(self) -> str:
        location = ""
        if self.entry_index is not None:
            location = f"[Entry {self.entry_index}] "
        if self.cql_file:
            location += f"({self.cql_file}"
            if self.cql_define:
                location += f"::{self.cql_define}"
            location += ") "
        return f"{self.severity}: {location}{self.message}"


@dataclass
class ValidationWarning:
    """A validation warning (non-blocking issue)."""
    warning_type: str
    message: str
    entry_index: Optional[int] = None

    def __str__(self) -> str:
        location = f"[Entry {self.entry_index}] " if self.entry_index else ""
        return f"WARNING: {location}{self.message}"


@dataclass
class ValidationReport:
    """Complete validation report for the registry."""
    registry_path: str
    registry_version: str
    total_entries: int
    errors: list[ValidationError] = field(default_factory=list)
    warnings: list[ValidationWarning] = field(default_factory=list)
    validated_cql_files: list[str] = field(default_factory=list)
    validated_defines: list[tuple[str, str]] = field(default_factory=list)

    @property
    def is_valid(self) -> bool:
        """Registry is valid if no errors (warnings are OK)."""
        return len(self.errors) == 0

    @property
    def error_count(self) -> int:
        return len(self.errors)

    @property
    def warning_count(self) -> int:
        return len(self.warnings)

    def to_dict(self) -> dict:
        return {
            "is_valid": self.is_valid,
            "registry_path": self.registry_path,
            "registry_version": self.registry_version,
            "total_entries": self.total_entries,
            "error_count": self.error_count,
            "warning_count": self.warning_count,
            "errors": [str(e) for e in self.errors],
            "warnings": [str(w) for w in self.warnings],
            "validated_cql_files": self.validated_cql_files,
            "validated_defines_count": len(self.validated_defines),
        }

    def print_summary(self) -> str:
        """Generate a human-readable summary."""
        lines = [
            f"Registry Validation Report",
            f"=" * 40,
            f"Registry: {self.registry_path}",
            f"Version: {self.registry_version}",
            f"Total Entries: {self.total_entries}",
            f"",
            f"Status: {'✅ VALID' if self.is_valid else '❌ INVALID'}",
            f"Errors: {self.error_count}",
            f"Warnings: {self.warning_count}",
        ]

        if self.errors:
            lines.append("")
            lines.append("ERRORS:")
            for error in self.errors:
                lines.append(f"  • {error}")

        if self.warnings:
            lines.append("")
            lines.append("WARNINGS:")
            for warning in self.warnings:
                lines.append(f"  • {warning}")

        lines.append("")
        lines.append(f"Validated CQL Files: {len(self.validated_cql_files)}")
        lines.append(f"Validated Defines: {len(self.validated_defines)}")

        return "\n".join(lines)


class RegistryValidator:
    """
    Validates the CQL Guideline Registry for integrity and consistency.

    Validation checks:
    1. Schema validation: Required fields present
    2. CQL file validation: Files exist in vaidshala
    3. Define validation: Defines exist at specified lines
    4. Consistency: No duplicate/conflicting entries
    5. KB reference validation: Valid KB types and fields
    """

    REQUIRED_ENTRY_FIELDS = ["cql_file", "cql_define", "guideline", "consumes"]
    REQUIRED_GUIDELINE_FIELDS = ["authority", "section"]
    REQUIRED_CONSUME_FIELDS = ["kb"]
    VALID_KB_TYPES = ["KB-1", "KB-4", "KB-5", "KB-16"]
    VALID_STATUSES = ["ACTIVE", "DEPRECATED", "PENDING_REVIEW"]

    def __init__(self, registry_path: Path, vaidshala_path: Path):
        """
        Initialize the validator.

        Args:
            registry_path: Path to cql_guideline_registry.yaml
            vaidshala_path: Path to vaidshala root directory
        """
        self.registry_path = Path(registry_path)
        self.vaidshala_path = Path(vaidshala_path)

        if not self.registry_path.exists():
            raise FileNotFoundError(f"Registry not found: {registry_path}")

        with open(self.registry_path) as f:
            self.registry = yaml.safe_load(f)

        # Cache for CQL file contents
        self._cql_cache: dict[str, list[str]] = {}

    def validate_all(self) -> ValidationReport:
        """
        Run all validation checks.

        Returns:
            ValidationReport with all errors and warnings
        """
        report = ValidationReport(
            registry_path=str(self.registry_path),
            registry_version=self.registry.get("registry_version", "unknown"),
            total_entries=len(self.registry.get("entries", [])),
        )

        # Run all validation checks
        self._validate_schema(report)
        self._validate_cql_files(report)
        self._validate_defines(report)
        self._validate_consistency(report)
        self._validate_kb_references(report)
        self._check_orphan_cql(report)

        return report

    def _validate_schema(self, report: ValidationReport) -> None:
        """Validate registry schema compliance."""
        entries = self.registry.get("entries", [])

        for i, entry in enumerate(entries):
            # Check required entry fields
            for field in self.REQUIRED_ENTRY_FIELDS:
                if field not in entry:
                    report.errors.append(ValidationError(
                        error_type="SCHEMA_ERROR",
                        message=f"Missing required field: {field}",
                        entry_index=i,
                        cql_file=entry.get("cql_file"),
                    ))

            # Check guideline subfields
            guideline = entry.get("guideline", {})
            for field in self.REQUIRED_GUIDELINE_FIELDS:
                if field not in guideline:
                    report.errors.append(ValidationError(
                        error_type="SCHEMA_ERROR",
                        message=f"Missing required guideline field: {field}",
                        entry_index=i,
                        cql_file=entry.get("cql_file"),
                        cql_define=entry.get("cql_define"),
                    ))

            # Check consume entries
            for consume in entry.get("consumes", []):
                for field in self.REQUIRED_CONSUME_FIELDS:
                    if field not in consume:
                        report.errors.append(ValidationError(
                            error_type="SCHEMA_ERROR",
                            message=f"Missing required consume field: {field}",
                            entry_index=i,
                            cql_file=entry.get("cql_file"),
                            cql_define=entry.get("cql_define"),
                        ))

            # Check status if present
            status = entry.get("status")
            if status and status not in self.VALID_STATUSES:
                report.warnings.append(ValidationWarning(
                    warning_type="INVALID_STATUS",
                    message=f"Unknown status '{status}', expected one of {self.VALID_STATUSES}",
                    entry_index=i,
                ))

    def _validate_cql_files(self, report: ValidationReport) -> None:
        """Validate that referenced CQL files exist."""
        entries = self.registry.get("entries", [])
        checked_files = set()

        # Find CQL directory
        cql_dir = self._find_cql_directory()

        for i, entry in enumerate(entries):
            cql_file = entry.get("cql_file")
            if not cql_file:
                continue

            if cql_file in checked_files:
                continue

            checked_files.add(cql_file)

            if cql_dir:
                cql_path = cql_dir / cql_file
                if cql_path.exists():
                    report.validated_cql_files.append(cql_file)
                    # Cache file contents
                    self._cql_cache[cql_file] = cql_path.read_text().split("\n")
                else:
                    report.errors.append(ValidationError(
                        error_type="FILE_NOT_FOUND",
                        message=f"CQL file not found: {cql_path}",
                        entry_index=i,
                        cql_file=cql_file,
                    ))
            else:
                report.warnings.append(ValidationWarning(
                    warning_type="CQL_DIR_NOT_FOUND",
                    message="Could not locate CQL directory in vaidshala",
                    entry_index=i,
                ))

    def _validate_defines(self, report: ValidationReport) -> None:
        """Validate that CQL defines exist at specified lines."""
        entries = self.registry.get("entries", [])

        for i, entry in enumerate(entries):
            cql_file = entry.get("cql_file")
            cql_define = entry.get("cql_define")
            cql_line = entry.get("cql_line")

            if not cql_file or not cql_define:
                continue

            if cql_file not in self._cql_cache:
                continue  # File validation already failed

            lines = self._cql_cache[cql_file]

            # Search for the define
            define_pattern = rf'define\s+"{re.escape(cql_define)}":'
            found_line = None

            for line_num, line in enumerate(lines, 1):
                if re.search(define_pattern, line):
                    found_line = line_num
                    break

            if found_line is None:
                report.errors.append(ValidationError(
                    error_type="DEFINE_NOT_FOUND",
                    message=f"Define '{cql_define}' not found in {cql_file}",
                    entry_index=i,
                    cql_file=cql_file,
                    cql_define=cql_define,
                ))
            else:
                report.validated_defines.append((cql_file, cql_define))

                # Check line number accuracy
                if cql_line and cql_line != found_line:
                    report.warnings.append(ValidationWarning(
                        warning_type="LINE_MISMATCH",
                        message=f"Define '{cql_define}' at line {found_line}, registry says {cql_line}",
                        entry_index=i,
                    ))

    def _validate_consistency(self, report: ValidationReport) -> None:
        """Check for duplicate or conflicting entries."""
        entries = self.registry.get("entries", [])
        seen = {}

        for i, entry in enumerate(entries):
            cql_file = entry.get("cql_file")
            cql_define = entry.get("cql_define")

            if not cql_file or not cql_define:
                continue

            key = (cql_file, cql_define)

            if key in seen:
                report.errors.append(ValidationError(
                    error_type="DUPLICATE_ENTRY",
                    message=f"Duplicate entry for {cql_file}::{cql_define} (also at entry {seen[key]})",
                    entry_index=i,
                    cql_file=cql_file,
                    cql_define=cql_define,
                ))
            else:
                seen[key] = i

    def _validate_kb_references(self, report: ValidationReport) -> None:
        """Validate KB references in consume entries."""
        entries = self.registry.get("entries", [])

        for i, entry in enumerate(entries):
            for consume in entry.get("consumes", []):
                kb = consume.get("kb")
                if kb and kb not in self.VALID_KB_TYPES:
                    report.errors.append(ValidationError(
                        error_type="INVALID_KB_REFERENCE",
                        message=f"Invalid KB type '{kb}', expected one of {self.VALID_KB_TYPES}",
                        entry_index=i,
                        cql_file=entry.get("cql_file"),
                        cql_define=entry.get("cql_define"),
                    ))

    def _check_orphan_cql(self, report: ValidationReport) -> None:
        """Check for CQL defines not in registry (backward gap detection)."""
        cql_dir = self._find_cql_directory()
        if not cql_dir:
            return

        # Get all registered defines
        registered = {
            (entry.get("cql_file"), entry.get("cql_define"))
            for entry in self.registry.get("entries", [])
        }

        # Scan all CQL files
        for cql_path in cql_dir.glob("*.cql"):
            content = cql_path.read_text()
            cql_file = cql_path.name

            for match in re.finditer(r'define\s+"([^"]+)":', content):
                define_name = match.group(1)
                if (cql_file, define_name) not in registered:
                    report.warnings.append(ValidationWarning(
                        warning_type="ORPHAN_CQL",
                        message=f"CQL define '{define_name}' in {cql_file} has no registry entry",
                    ))

    def _find_cql_directory(self) -> Optional[Path]:
        """Find the CQL directory in vaidshala."""
        possible_paths = [
            self.vaidshala_path / "clinical-knowledge-core" / "tier-4-guidelines" / "clinical",
            self.vaidshala_path / "tier-4-guidelines" / "clinical",
            self.vaidshala_path / "clinical",
        ]

        for path in possible_paths:
            if path.exists():
                return path

        return None

    def validate_entry(self, entry_index: int) -> list[ValidationError]:
        """Validate a single registry entry."""
        entries = self.registry.get("entries", [])

        if entry_index < 0 or entry_index >= len(entries):
            return [ValidationError(
                error_type="SCHEMA_ERROR",
                message=f"Entry index {entry_index} out of range",
            )]

        # Run validation on single entry
        report = ValidationReport(
            registry_path=str(self.registry_path),
            registry_version=self.registry.get("registry_version", "unknown"),
            total_entries=1,
        )

        # Temporarily replace entries for validation
        original_entries = self.registry["entries"]
        self.registry["entries"] = [entries[entry_index]]

        try:
            self._validate_schema(report)
            self._validate_cql_files(report)
            self._validate_defines(report)
            self._validate_kb_references(report)
        finally:
            self.registry["entries"] = original_entries

        return report.errors


def validate_registry(
    registry_path: Path,
    vaidshala_path: Path,
) -> ValidationReport:
    """
    Convenience function to validate a registry.

    Args:
        registry_path: Path to registry YAML
        vaidshala_path: Path to vaidshala root

    Returns:
        ValidationReport with all findings
    """
    validator = RegistryValidator(registry_path, vaidshala_path)
    return validator.validate_all()


# CLI interface
if __name__ == "__main__":
    import argparse
    import json

    parser = argparse.ArgumentParser(
        description="Validate CQL Guideline Registry"
    )
    parser.add_argument(
        "--registry", "-r",
        type=Path,
        required=True,
        help="Path to cql_guideline_registry.yaml"
    )
    parser.add_argument(
        "--vaidshala", "-v",
        type=Path,
        required=True,
        help="Path to vaidshala root directory"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON"
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Fail on warnings too"
    )

    args = parser.parse_args()

    report = validate_registry(args.registry, args.vaidshala)

    if args.json:
        print(json.dumps(report.to_dict(), indent=2))
    else:
        print(report.print_summary())

    # Exit code
    if not report.is_valid:
        exit(1)
    if args.strict and report.warnings:
        exit(1)
