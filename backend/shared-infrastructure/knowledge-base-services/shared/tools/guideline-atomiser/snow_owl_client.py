"""
L4: Snow Owl Terminology Validation Client.

This module provides terminology validation and enrichment using Snow Owl,
supporting RxNorm, LOINC, and SNOMED-CT code systems.

Key Principle: Validate and enrich extracted entities with standard codes
before storing in KBs. Invalid codes must be flagged for human review.

Supported Code Systems:
- RxNorm: Drug codes (KB-1, KB-4)
- LOINC: Lab test codes (KB-16)
- SNOMED-CT: Clinical conditions (KB-4)
- ICD-10: Diagnosis codes (KB-4)

Usage:
    from snow_owl_client import SnowOwlClient, create_client_from_env

    client = create_client_from_env()

    # Validate RxNorm code
    result = client.validate_rxnorm("6809")  # metformin

    # Search for drug by name
    drugs = client.search_rxnorm("metformin")

    # Validate LOINC code
    result = client.validate_loinc("33914-3")  # eGFR
"""

import os
import json
from dataclasses import dataclass, field
from typing import Optional, Literal
from datetime import datetime, timezone
import httpx


@dataclass
class CodeValidationResult:
    """Result of code validation."""
    code: str
    code_system: str
    is_valid: bool
    display_name: Optional[str] = None
    preferred_term: Optional[str] = None
    synonyms: list[str] = field(default_factory=list)
    status: Optional[str] = None  # active, inactive, deprecated
    error_message: Optional[str] = None

    def to_dict(self) -> dict:
        return {
            "code": self.code,
            "code_system": self.code_system,
            "is_valid": self.is_valid,
            "display_name": self.display_name,
            "preferred_term": self.preferred_term,
            "synonyms": self.synonyms,
            "status": self.status,
            "error_message": self.error_message,
        }


@dataclass
class SearchResult:
    """Result from terminology search."""
    code: str
    code_system: str
    display_name: str
    score: float = 1.0
    status: str = "active"

    def to_dict(self) -> dict:
        return {
            "code": self.code,
            "code_system": self.code_system,
            "display_name": self.display_name,
            "score": self.score,
            "status": self.status,
        }


@dataclass
class TerminologyEnrichment:
    """Enrichment data for an entity."""
    original_text: str
    rxnorm_code: Optional[str] = None
    rxnorm_display: Optional[str] = None
    loinc_code: Optional[str] = None
    loinc_display: Optional[str] = None
    snomed_code: Optional[str] = None
    snomed_display: Optional[str] = None
    icd10_code: Optional[str] = None
    icd10_display: Optional[str] = None
    validation_status: Literal["VALID", "PARTIAL", "INVALID", "NOT_FOUND"] = "NOT_FOUND"
    validation_timestamp: str = ""

    def __post_init__(self):
        if not self.validation_timestamp:
            self.validation_timestamp = datetime.now(timezone.utc).isoformat()

    def to_dict(self) -> dict:
        result = {"original_text": self.original_text}
        if self.rxnorm_code:
            result["rxnorm"] = {"code": self.rxnorm_code, "display": self.rxnorm_display}
        if self.loinc_code:
            result["loinc"] = {"code": self.loinc_code, "display": self.loinc_display}
        if self.snomed_code:
            result["snomed"] = {"code": self.snomed_code, "display": self.snomed_display}
        if self.icd10_code:
            result["icd10"] = {"code": self.icd10_code, "display": self.icd10_display}
        result["validation_status"] = self.validation_status
        result["validation_timestamp"] = self.validation_timestamp
        return result


class SnowOwlClient:
    """
    L4 Terminology Validation Client using Snow Owl.

    Snow Owl is an open-source terminology server that provides:
    - FHIR R4 compliant terminology services
    - Support for SNOMED-CT, LOINC, ICD-10, RxNorm
    - Concept validation and lookup
    - Semantic search capabilities
    """

    VERSION = "1.0.0"

    # Code system URIs
    RXNORM_URI = "http://www.nlm.nih.gov/research/umls/rxnorm"
    LOINC_URI = "http://loinc.org"
    SNOMED_URI = "http://snomed.info/sct"
    ICD10_URI = "http://hl7.org/fhir/sid/icd-10-cm"

    # Snow Owl code system IDs
    CODE_SYSTEM_IDS = {
        "rxnorm": "SNOMEDCT/900000000000207008",  # RxNorm extension in SNOMED
        "loinc": "LOINC",
        "snomed": "SNOMEDCT",
        "icd10": "ICD-10-CM",
    }

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        username: str = "snowowl",
        password: str = "snowowl",
        timeout: float = 30.0,
    ):
        """
        Initialize Snow Owl client.

        Args:
            base_url: Snow Owl server URL
            username: Authentication username
            password: Authentication password
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip("/")
        self.auth = (username, password)
        self.timeout = timeout
        self._client: Optional[httpx.Client] = None

    def _get_client(self) -> httpx.Client:
        """Get or create HTTP client."""
        if self._client is None:
            self._client = httpx.Client(
                base_url=self.base_url,
                auth=self.auth,
                timeout=self.timeout,
                headers={"Accept": "application/json"},
            )
        return self._client

    def close(self):
        """Close the HTTP client."""
        if self._client:
            self._client.close()
            self._client = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

    # ═══════════════════════════════════════════════════════════════════════════
    # Health Check
    # ═══════════════════════════════════════════════════════════════════════════

    def health_check(self) -> bool:
        """Check if Snow Owl is available."""
        try:
            client = self._get_client()
            response = client.get("/snowowl/admin/info")
            return response.status_code == 200
        except Exception:
            return False

    # ═══════════════════════════════════════════════════════════════════════════
    # RxNorm Validation (KB-1, KB-4)
    # ═══════════════════════════════════════════════════════════════════════════

    def validate_rxnorm(self, code: str) -> CodeValidationResult:
        """
        Validate an RxNorm code.

        Args:
            code: RxNorm concept code (e.g., "6809" for metformin)

        Returns:
            CodeValidationResult with validation status
        """
        try:
            client = self._get_client()
            # Try FHIR CodeSystem lookup
            response = client.get(
                f"/snowowl/fhir/CodeSystem/$lookup",
                params={
                    "system": self.RXNORM_URI,
                    "code": code,
                }
            )

            if response.status_code == 200:
                data = response.json()
                return CodeValidationResult(
                    code=code,
                    code_system="RxNorm",
                    is_valid=True,
                    display_name=self._extract_display(data),
                    preferred_term=self._extract_designation(data, "preferred"),
                    synonyms=self._extract_designations(data, "synonym"),
                    status="active",
                )
            else:
                return CodeValidationResult(
                    code=code,
                    code_system="RxNorm",
                    is_valid=False,
                    error_message=f"Code not found: {response.status_code}",
                )
        except Exception as e:
            return CodeValidationResult(
                code=code,
                code_system="RxNorm",
                is_valid=False,
                error_message=str(e),
            )

    def search_rxnorm(
        self,
        term: str,
        limit: int = 10,
    ) -> list[SearchResult]:
        """
        Search RxNorm by term.

        Args:
            term: Search term (drug name)
            limit: Maximum results to return

        Returns:
            List of matching SearchResults
        """
        try:
            client = self._get_client()
            response = client.get(
                f"/snowowl/fhir/CodeSystem/$find-matches",
                params={
                    "system": self.RXNORM_URI,
                    "exact": "false",
                    "property": f"display:{term}",
                    "_count": limit,
                }
            )

            if response.status_code == 200:
                data = response.json()
                return self._parse_search_results(data, "RxNorm")
            return []
        except Exception:
            return []

    # ═══════════════════════════════════════════════════════════════════════════
    # LOINC Validation (KB-16)
    # ═══════════════════════════════════════════════════════════════════════════

    def validate_loinc(self, code: str) -> CodeValidationResult:
        """
        Validate a LOINC code.

        Args:
            code: LOINC code (e.g., "33914-3" for eGFR)

        Returns:
            CodeValidationResult with validation status
        """
        try:
            client = self._get_client()
            response = client.get(
                f"/snowowl/fhir/CodeSystem/$lookup",
                params={
                    "system": self.LOINC_URI,
                    "code": code,
                }
            )

            if response.status_code == 200:
                data = response.json()
                return CodeValidationResult(
                    code=code,
                    code_system="LOINC",
                    is_valid=True,
                    display_name=self._extract_display(data),
                    preferred_term=self._extract_designation(data, "preferred"),
                    status="active",
                )
            else:
                return CodeValidationResult(
                    code=code,
                    code_system="LOINC",
                    is_valid=False,
                    error_message=f"Code not found: {response.status_code}",
                )
        except Exception as e:
            return CodeValidationResult(
                code=code,
                code_system="LOINC",
                is_valid=False,
                error_message=str(e),
            )

    def search_loinc(
        self,
        term: str,
        limit: int = 10,
    ) -> list[SearchResult]:
        """
        Search LOINC by term.

        Args:
            term: Search term (lab test name)
            limit: Maximum results

        Returns:
            List of matching SearchResults
        """
        try:
            client = self._get_client()
            response = client.get(
                f"/snowowl/fhir/CodeSystem/$find-matches",
                params={
                    "system": self.LOINC_URI,
                    "exact": "false",
                    "property": f"display:{term}",
                    "_count": limit,
                }
            )

            if response.status_code == 200:
                data = response.json()
                return self._parse_search_results(data, "LOINC")
            return []
        except Exception:
            return []

    # ═══════════════════════════════════════════════════════════════════════════
    # SNOMED-CT Validation (KB-4)
    # ═══════════════════════════════════════════════════════════════════════════

    def validate_snomed(self, code: str) -> CodeValidationResult:
        """
        Validate a SNOMED-CT code.

        Args:
            code: SNOMED-CT concept ID

        Returns:
            CodeValidationResult with validation status
        """
        try:
            client = self._get_client()
            response = client.get(
                f"/snowowl/fhir/CodeSystem/$lookup",
                params={
                    "system": self.SNOMED_URI,
                    "code": code,
                }
            )

            if response.status_code == 200:
                data = response.json()
                return CodeValidationResult(
                    code=code,
                    code_system="SNOMED-CT",
                    is_valid=True,
                    display_name=self._extract_display(data),
                    preferred_term=self._extract_designation(data, "preferred"),
                    synonyms=self._extract_designations(data, "synonym"),
                    status="active",
                )
            else:
                return CodeValidationResult(
                    code=code,
                    code_system="SNOMED-CT",
                    is_valid=False,
                    error_message=f"Code not found: {response.status_code}",
                )
        except Exception as e:
            return CodeValidationResult(
                code=code,
                code_system="SNOMED-CT",
                is_valid=False,
                error_message=str(e),
            )

    def search_snomed(
        self,
        term: str,
        limit: int = 10,
    ) -> list[SearchResult]:
        """
        Search SNOMED-CT by term.

        Args:
            term: Search term (clinical condition)
            limit: Maximum results

        Returns:
            List of matching SearchResults
        """
        try:
            client = self._get_client()
            response = client.get(
                f"/snowowl/fhir/CodeSystem/$find-matches",
                params={
                    "system": self.SNOMED_URI,
                    "exact": "false",
                    "property": f"display:{term}",
                    "_count": limit,
                }
            )

            if response.status_code == 200:
                data = response.json()
                return self._parse_search_results(data, "SNOMED-CT")
            return []
        except Exception:
            return []

    # ═══════════════════════════════════════════════════════════════════════════
    # Entity Enrichment
    # ═══════════════════════════════════════════════════════════════════════════

    def enrich_drug_entity(self, drug_name: str) -> TerminologyEnrichment:
        """
        Enrich a drug entity with RxNorm code.

        Args:
            drug_name: Drug name to look up

        Returns:
            TerminologyEnrichment with RxNorm data
        """
        enrichment = TerminologyEnrichment(original_text=drug_name)

        results = self.search_rxnorm(drug_name, limit=1)
        if results:
            top_match = results[0]
            validation = self.validate_rxnorm(top_match.code)
            if validation.is_valid:
                enrichment.rxnorm_code = top_match.code
                enrichment.rxnorm_display = validation.display_name or top_match.display_name
                enrichment.validation_status = "VALID"
            else:
                enrichment.validation_status = "PARTIAL"
        else:
            enrichment.validation_status = "NOT_FOUND"

        return enrichment

    def enrich_lab_entity(self, lab_name: str) -> TerminologyEnrichment:
        """
        Enrich a lab test entity with LOINC code.

        Args:
            lab_name: Lab test name to look up

        Returns:
            TerminologyEnrichment with LOINC data
        """
        enrichment = TerminologyEnrichment(original_text=lab_name)

        results = self.search_loinc(lab_name, limit=1)
        if results:
            top_match = results[0]
            validation = self.validate_loinc(top_match.code)
            if validation.is_valid:
                enrichment.loinc_code = top_match.code
                enrichment.loinc_display = validation.display_name or top_match.display_name
                enrichment.validation_status = "VALID"
            else:
                enrichment.validation_status = "PARTIAL"
        else:
            enrichment.validation_status = "NOT_FOUND"

        return enrichment

    def enrich_condition_entity(self, condition_name: str) -> TerminologyEnrichment:
        """
        Enrich a condition entity with SNOMED-CT code.

        Args:
            condition_name: Condition name to look up

        Returns:
            TerminologyEnrichment with SNOMED-CT data
        """
        enrichment = TerminologyEnrichment(original_text=condition_name)

        results = self.search_snomed(condition_name, limit=1)
        if results:
            top_match = results[0]
            validation = self.validate_snomed(top_match.code)
            if validation.is_valid:
                enrichment.snomed_code = top_match.code
                enrichment.snomed_display = validation.display_name or top_match.display_name
                enrichment.validation_status = "VALID"
            else:
                enrichment.validation_status = "PARTIAL"
        else:
            enrichment.validation_status = "NOT_FOUND"

        return enrichment

    # ═══════════════════════════════════════════════════════════════════════════
    # Batch Validation
    # ═══════════════════════════════════════════════════════════════════════════

    def validate_extracted_facts(
        self,
        facts: dict,
    ) -> dict:
        """
        Validate all terminology codes in extracted facts.

        Args:
            facts: Extracted facts dictionary

        Returns:
            Facts with validation status added
        """
        validated = facts.copy()
        validation_results = []

        # Validate drug codes (KB-1)
        for drug in validated.get("drugs", []):
            rxnorm = drug.get("rxnormCode") or drug.get("rxnorm_code")
            if rxnorm:
                result = self.validate_rxnorm(rxnorm)
                drug["_rxnorm_validation"] = result.to_dict()
                validation_results.append(result)

        # Validate contraindication codes (KB-4)
        for ci in validated.get("contraindications", []):
            rxnorm = ci.get("rxnormCode") or ci.get("rxnorm_code")
            if rxnorm:
                result = self.validate_rxnorm(rxnorm)
                ci["_rxnorm_validation"] = result.to_dict()
                validation_results.append(result)

            for snomed in ci.get("snomedCodes") or ci.get("snomed_codes") or []:
                result = self.validate_snomed(snomed)
                ci.setdefault("_snomed_validation", []).append(result.to_dict())
                validation_results.append(result)

        # Validate lab codes (KB-16)
        for req in validated.get("labRequirements") or validated.get("lab_requirements") or []:
            for lab in req.get("labs", []):
                loinc = lab.get("loincCode") or lab.get("loinc_code")
                if loinc:
                    result = self.validate_loinc(loinc)
                    lab["_loinc_validation"] = result.to_dict()
                    validation_results.append(result)

        # Add summary
        valid_count = sum(1 for r in validation_results if r.is_valid)
        validated["_terminology_validation"] = {
            "total_codes": len(validation_results),
            "valid_codes": valid_count,
            "invalid_codes": len(validation_results) - valid_count,
            "validation_rate": valid_count / len(validation_results) if validation_results else 0,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

        return validated

    # ═══════════════════════════════════════════════════════════════════════════
    # Helper Methods
    # ═══════════════════════════════════════════════════════════════════════════

    def _extract_display(self, data: dict) -> Optional[str]:
        """Extract display name from FHIR response."""
        for param in data.get("parameter", []):
            if param.get("name") == "display":
                return param.get("valueString")
        return None

    def _extract_designation(self, data: dict, use: str) -> Optional[str]:
        """Extract designation by use type."""
        for param in data.get("parameter", []):
            if param.get("name") == "designation":
                parts = param.get("part", [])
                for part in parts:
                    if part.get("name") == "use" and use in str(part.get("valueCoding", {})):
                        for p in parts:
                            if p.get("name") == "value":
                                return p.get("valueString")
        return None

    def _extract_designations(self, data: dict, use: str) -> list[str]:
        """Extract all designations of a type."""
        results = []
        for param in data.get("parameter", []):
            if param.get("name") == "designation":
                parts = param.get("part", [])
                is_match = False
                value = None
                for part in parts:
                    if part.get("name") == "use" and use in str(part.get("valueCoding", {})):
                        is_match = True
                    if part.get("name") == "value":
                        value = part.get("valueString")
                if is_match and value:
                    results.append(value)
        return results

    def _parse_search_results(self, data: dict, code_system: str) -> list[SearchResult]:
        """Parse FHIR search results."""
        results = []
        for param in data.get("parameter", []):
            if param.get("name") == "match":
                parts = param.get("part", [])
                code = None
                display = None
                score = 1.0
                for part in parts:
                    if part.get("name") == "code":
                        code = part.get("valueCode")
                    if part.get("name") == "display":
                        display = part.get("valueString")
                    if part.get("name") == "score":
                        score = float(part.get("valueDecimal", 1.0))
                if code:
                    results.append(SearchResult(
                        code=code,
                        code_system=code_system,
                        display_name=display or code,
                        score=score,
                    ))
        return results


def create_client_from_env() -> SnowOwlClient:
    """
    Create Snow Owl client from environment variables.

    Environment Variables:
        SNOW_OWL_URL: Server URL (default: http://localhost:8080)
        SNOW_OWL_USER: Username (default: snowowl)
        SNOW_OWL_PASSWORD: Password (default: snowowl)
    """
    return SnowOwlClient(
        base_url=os.getenv("SNOW_OWL_URL", "http://localhost:8080"),
        username=os.getenv("SNOW_OWL_USER", "snowowl"),
        password=os.getenv("SNOW_OWL_PASSWORD", "snowowl"),
    )


# CLI interface
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="L4 Snow Owl Terminology Validation Client"
    )
    subparsers = parser.add_subparsers(dest="command", help="Command")

    # Health check
    health_parser = subparsers.add_parser("health", help="Check Snow Owl health")

    # Validate command
    validate_parser = subparsers.add_parser("validate", help="Validate a code")
    validate_parser.add_argument("code", help="Code to validate")
    validate_parser.add_argument(
        "--system", "-s",
        choices=["rxnorm", "loinc", "snomed"],
        required=True,
        help="Code system"
    )

    # Search command
    search_parser = subparsers.add_parser("search", help="Search for a term")
    search_parser.add_argument("term", help="Term to search")
    search_parser.add_argument(
        "--system", "-s",
        choices=["rxnorm", "loinc", "snomed"],
        required=True,
        help="Code system"
    )
    search_parser.add_argument("--limit", "-l", type=int, default=10, help="Max results")

    # Enrich command
    enrich_parser = subparsers.add_parser("enrich", help="Enrich an entity")
    enrich_parser.add_argument("text", help="Entity text")
    enrich_parser.add_argument(
        "--type", "-t",
        choices=["drug", "lab", "condition"],
        required=True,
        help="Entity type"
    )

    args = parser.parse_args()

    client = create_client_from_env()

    if args.command == "health":
        is_healthy = client.health_check()
        print(f"Snow Owl: {'✅ Available' if is_healthy else '❌ Unavailable'}")

    elif args.command == "validate":
        if args.system == "rxnorm":
            result = client.validate_rxnorm(args.code)
        elif args.system == "loinc":
            result = client.validate_loinc(args.code)
        elif args.system == "snomed":
            result = client.validate_snomed(args.code)
        print(json.dumps(result.to_dict(), indent=2))

    elif args.command == "search":
        if args.system == "rxnorm":
            results = client.search_rxnorm(args.term, args.limit)
        elif args.system == "loinc":
            results = client.search_loinc(args.term, args.limit)
        elif args.system == "snomed":
            results = client.search_snomed(args.term, args.limit)
        print(json.dumps([r.to_dict() for r in results], indent=2))

    elif args.command == "enrich":
        if args.type == "drug":
            result = client.enrich_drug_entity(args.text)
        elif args.type == "lab":
            result = client.enrich_lab_entity(args.text)
        elif args.type == "condition":
            result = client.enrich_condition_entity(args.text)
        print(json.dumps(result.to_dict(), indent=2))

    else:
        parser.print_help()
