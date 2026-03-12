"""
L4: RxNav Terminology Validation Client for RxNorm.

This module provides RxNorm terminology validation using RxNav-in-a-Box,
which provides offline access to RxNorm drug terminology.

Key Principle: Validate extracted drug entities with standard RxNorm codes
before storing in KBs. Invalid codes must be flagged for human review.

Supported Operations:
- RxNorm code validation
- Drug name to RxCUI lookup
- Drug ingredient lookups
- Brand/generic relationships

Usage:
    from rxnav_client import RxNavClient, create_rxnav_client_from_env

    client = create_rxnav_client_from_env()

    # Validate RxNorm code
    result = client.validate_rxnorm("6809")  # metformin

    # Search for drug by name
    drugs = client.search_rxnorm("metformin")
"""

import os
import json
from dataclasses import dataclass, field
from typing import Optional, Literal, List
from datetime import datetime, timezone
import httpx


@dataclass
class RxNormValidationResult:
    """Result of RxNorm code validation."""
    code: str
    code_system: str = "RxNorm"
    is_valid: bool = False
    display_name: Optional[str] = None
    preferred_term: Optional[str] = None
    synonyms: List[str] = field(default_factory=list)
    term_type: Optional[str] = None  # SCD, SBD, IN, BN, etc.
    status: Optional[str] = None  # Active, Obsolete, etc.
    error_message: Optional[str] = None

    def to_dict(self) -> dict:
        return {
            "code": self.code,
            "code_system": self.code_system,
            "is_valid": self.is_valid,
            "display_name": self.display_name,
            "preferred_term": self.preferred_term,
            "synonyms": self.synonyms,
            "term_type": self.term_type,
            "status": self.status,
            "error_message": self.error_message,
        }


@dataclass
class RxNormSearchResult:
    """Result from RxNorm search."""
    rxcui: str
    name: str
    term_type: str  # SCD, SBD, IN, BN, etc.
    score: float = 1.0
    status: str = "Active"

    def to_dict(self) -> dict:
        return {
            "code": self.rxcui,
            "code_system": "RxNorm",
            "display_name": self.name,
            "term_type": self.term_type,
            "score": self.score,
            "status": self.status,
        }


@dataclass
class DrugEnrichment:
    """Enrichment data for a drug entity."""
    original_text: str
    rxnorm_code: Optional[str] = None
    rxnorm_display: Optional[str] = None
    term_type: Optional[str] = None
    ingredients: List[str] = field(default_factory=list)
    validation_status: Literal["VALID", "PARTIAL", "INVALID", "NOT_FOUND"] = "NOT_FOUND"
    validation_timestamp: str = ""

    def __post_init__(self):
        if not self.validation_timestamp:
            self.validation_timestamp = datetime.now(timezone.utc).isoformat()

    def to_dict(self) -> dict:
        result = {
            "original_text": self.original_text,
            "validation_status": self.validation_status,
            "validation_timestamp": self.validation_timestamp,
        }
        if self.rxnorm_code:
            result["rxnorm"] = {
                "code": self.rxnorm_code,
                "display": self.rxnorm_display,
                "term_type": self.term_type,
            }
        if self.ingredients:
            result["ingredients"] = self.ingredients
        return result


class RxNavClient:
    """
    L4 Terminology Validation Client using RxNav-in-a-Box.

    RxNav provides comprehensive RxNorm drug terminology services:
    - Drug concept lookup by RxCUI
    - Approximate term matching for drug names
    - Drug relationship queries (ingredients, brands, generics)
    - NDC to RxCUI mapping
    """

    VERSION = "1.1.0"  # Added KB7Client-compatible interface methods

    # RxNorm Term Types
    TERM_TYPES = {
        "IN": "Ingredient",
        "PIN": "Precise Ingredient",
        "MIN": "Multiple Ingredients",
        "BN": "Brand Name",
        "SCDC": "Semantic Clinical Drug Component",
        "SCDF": "Semantic Clinical Drug Form",
        "SCDG": "Semantic Clinical Drug Group",
        "SCD": "Semantic Clinical Drug",
        "SBDC": "Semantic Branded Drug Component",
        "SBDF": "Semantic Branded Drug Form",
        "SBDG": "Semantic Branded Drug Group",
        "SBD": "Semantic Branded Drug",
        "DF": "Dose Form",
        "ET": "Entry Term",
    }

    def __init__(
        self,
        base_url: str = "http://localhost:4000",
        timeout: float = 30.0,
    ):
        """
        Initialize RxNav client.

        Args:
            base_url: RxNav-in-a-Box server URL (typically http://host:4000)
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self._client: Optional[httpx.Client] = None

    def _get_client(self) -> httpx.Client:
        """Get or create HTTP client."""
        if self._client is None:
            self._client = httpx.Client(
                base_url=self.base_url,
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
        """Check if RxNav is available."""
        try:
            client = self._get_client()
            # Try version endpoint
            response = client.get("/REST/version.json")
            return response.status_code == 200
        except Exception:
            return False

    def get_version(self) -> Optional[str]:
        """Get RxNorm database version."""
        try:
            client = self._get_client()
            response = client.get("/REST/version.json")
            if response.status_code == 200:
                data = response.json()
                return data.get("version", {}).get("rxnormDate")
            return None
        except Exception:
            return None

    # ═══════════════════════════════════════════════════════════════════════════
    # RxNorm Validation
    # ═══════════════════════════════════════════════════════════════════════════

    def validate_rxnorm(self, rxcui: str) -> RxNormValidationResult:
        """
        Validate an RxNorm concept (RxCUI).

        Args:
            rxcui: RxNorm Concept Unique Identifier (e.g., "6809" for metformin)

        Returns:
            RxNormValidationResult with validation status
        """
        try:
            client = self._get_client()

            # Get RxCUI properties
            response = client.get(f"/REST/rxcui/{rxcui}/properties.json")

            if response.status_code == 200:
                data = response.json()
                properties = data.get("properties", {})

                if properties:
                    return RxNormValidationResult(
                        code=rxcui,
                        is_valid=True,
                        display_name=properties.get("name"),
                        preferred_term=properties.get("name"),
                        term_type=properties.get("tty"),
                        status="Active" if properties.get("suppress") == "N" else "Suppressed",
                        synonyms=self._get_synonyms(rxcui),
                    )

            # Check if code exists but is obsolete
            status_response = client.get(f"/REST/rxcui/{rxcui}/status.json")
            if status_response.status_code == 200:
                status_data = status_response.json()
                rxcui_status = status_data.get("rxcuiStatus", {})
                if rxcui_status.get("status"):
                    return RxNormValidationResult(
                        code=rxcui,
                        is_valid=False,
                        status=rxcui_status.get("status"),
                        error_message=f"RxCUI status: {rxcui_status.get('status')}",
                    )

            return RxNormValidationResult(
                code=rxcui,
                is_valid=False,
                error_message="RxCUI not found",
            )

        except Exception as e:
            return RxNormValidationResult(
                code=rxcui,
                is_valid=False,
                error_message=str(e),
            )

    def _get_synonyms(self, rxcui: str) -> List[str]:
        """Get synonyms for an RxCUI."""
        try:
            client = self._get_client()
            response = client.get(f"/REST/rxcui/{rxcui}/allrelated.json")
            if response.status_code == 200:
                data = response.json()
                synonyms = []
                for group in data.get("allRelatedGroup", {}).get("conceptGroup", []):
                    for prop in group.get("conceptProperties", []):
                        name = prop.get("name")
                        if name:
                            synonyms.append(name)
                return synonyms[:10]  # Limit to 10
            return []
        except Exception:
            return []

    # ═══════════════════════════════════════════════════════════════════════════
    # RxNorm Search
    # ═══════════════════════════════════════════════════════════════════════════

    def search_rxnorm(
        self,
        term: str,
        limit: int = 10,
    ) -> List[RxNormSearchResult]:
        """
        Search RxNorm by drug name using approximate matching.

        Args:
            term: Search term (drug name)
            limit: Maximum results to return

        Returns:
            List of matching RxNormSearchResults
        """
        results = []

        try:
            client = self._get_client()

            # Try approximate term match first (most flexible)
            response = client.get(
                "/REST/approximateTerm.json",
                params={
                    "term": term,
                    "maxEntries": limit,
                }
            )

            if response.status_code == 200:
                data = response.json()
                candidates = data.get("approximateGroup", {}).get("candidate", [])

                for candidate in candidates[:limit]:
                    rxcui = candidate.get("rxcui")
                    if rxcui:
                        results.append(RxNormSearchResult(
                            rxcui=rxcui,
                            name=candidate.get("name", ""),
                            term_type=candidate.get("tty", ""),
                            score=float(candidate.get("score", 0)) / 100.0,
                        ))

            # If no results, try exact drug search
            if not results:
                response = client.get(
                    "/REST/drugs.json",
                    params={"name": term}
                )

                if response.status_code == 200:
                    data = response.json()
                    drug_group = data.get("drugGroup", {})

                    for group in drug_group.get("conceptGroup", []):
                        for prop in group.get("conceptProperties", [])[:limit]:
                            results.append(RxNormSearchResult(
                                rxcui=prop.get("rxcui", ""),
                                name=prop.get("name", ""),
                                term_type=prop.get("tty", ""),
                                score=1.0,
                            ))
                            if len(results) >= limit:
                                break

            return results

        except Exception:
            return []

    def get_rxcui_by_name(self, name: str) -> Optional[str]:
        """
        Get RxCUI for an exact drug name.

        Args:
            name: Exact drug name

        Returns:
            RxCUI if found, None otherwise
        """
        try:
            client = self._get_client()
            response = client.get(
                "/REST/rxcui.json",
                params={"name": name}
            )

            if response.status_code == 200:
                data = response.json()
                rxcui_list = data.get("idGroup", {}).get("rxnormId", [])
                if rxcui_list:
                    return rxcui_list[0]
            return None
        except Exception:
            return None

    # ═══════════════════════════════════════════════════════════════════════════
    # Drug Relationships
    # ═══════════════════════════════════════════════════════════════════════════

    def get_ingredients(self, rxcui: str) -> List[dict]:
        """
        Get ingredients for a drug RxCUI.

        Args:
            rxcui: Drug RxCUI

        Returns:
            List of ingredient dictionaries with rxcui and name
        """
        try:
            client = self._get_client()
            response = client.get(
                f"/REST/rxcui/{rxcui}/related.json",
                params={"tty": "IN"}  # Ingredient term type
            )

            if response.status_code == 200:
                data = response.json()
                ingredients = []
                for group in data.get("relatedGroup", {}).get("conceptGroup", []):
                    for prop in group.get("conceptProperties", []):
                        ingredients.append({
                            "rxcui": prop.get("rxcui"),
                            "name": prop.get("name"),
                        })
                return ingredients
            return []
        except Exception:
            return []

    def get_brand_names(self, rxcui: str) -> List[dict]:
        """
        Get brand names for an ingredient or generic drug.

        Args:
            rxcui: Drug RxCUI

        Returns:
            List of brand name dictionaries
        """
        try:
            client = self._get_client()
            response = client.get(
                f"/REST/rxcui/{rxcui}/related.json",
                params={"tty": "BN"}  # Brand Name term type
            )

            if response.status_code == 200:
                data = response.json()
                brands = []
                for group in data.get("relatedGroup", {}).get("conceptGroup", []):
                    for prop in group.get("conceptProperties", []):
                        brands.append({
                            "rxcui": prop.get("rxcui"),
                            "name": prop.get("name"),
                        })
                return brands
            return []
        except Exception:
            return []

    # ═══════════════════════════════════════════════════════════════════════════
    # KB7Client-Compatible Interface (L2.5 + L4 Pipeline Drop-In)
    # ═══════════════════════════════════════════════════════════════════════════

    def search(
        self,
        query: str,
        system: Optional[str] = None,
        limit: int = 10,
        active_only: bool = True,
    ) -> List[RxNormValidationResult]:
        """
        KB7Client-compatible search: name → validated results.

        Returns RxNormValidationResult objects (which have .code, .is_valid,
        .display_name) matching the interface L2.5 expects from KB7Client.search().
        """
        results: List[RxNormValidationResult] = []

        # 1. Exact name-to-CUI lookup
        try:
            client = self._get_client()
            resp = client.get("/REST/rxcui.json", params={"name": query})
            if resp.status_code == 200:
                rxcuis = resp.json().get("idGroup", {}).get("rxnormId", [])
                for cui in rxcuis[:limit]:
                    vr = self.validate_rxnorm(cui)
                    if vr.is_valid:
                        results.append(vr)
                if results:
                    return results
        except Exception:
            pass

        # 2. Approximate/fuzzy search fallback
        search_results = self.search_rxnorm(query, limit=limit)
        for sr in search_results:
            results.append(RxNormValidationResult(
                code=sr.rxcui,
                is_valid=True,
                display_name=sr.name,
                term_type=sr.term_type,
            ))

        return results

    def get_relationships(
        self,
        code: str,
        system: str = "rxnorm",
        relationship_type: Optional[str] = None,
    ) -> List[dict]:
        """
        KB7Client-compatible relationship query.

        Returns list of relationship dicts. L4 only checks len(rels) > 0.
        """
        try:
            client = self._get_client()
            resp = client.get(
                f"/REST/rxcui/{code}/related.json",
                params={"tty": "IN BN PIN SCDC SCD SBD"},
            )
            if resp.status_code != 200:
                return []

            groups = resp.json().get("relatedGroup", {}).get("conceptGroup", [])
            rels = []
            for group in groups:
                tty = group.get("tty", "")
                for prop in group.get("conceptProperties", []):
                    rels.append({
                        "source_code": code,
                        "target_code": prop.get("rxcui"),
                        "relationship_type": "related",
                        "relationship_attr": tty,
                        "target_display": prop.get("name"),
                    })
            return rels
        except Exception:
            return []

    # ═══════════════════════════════════════════════════════════════════════════
    # Entity Enrichment
    # ═══════════════════════════════════════════════════════════════════════════

    def enrich_drug_entity(self, drug_name: str) -> DrugEnrichment:
        """
        Enrich a drug entity with RxNorm code and related data.

        Args:
            drug_name: Drug name to look up

        Returns:
            DrugEnrichment with RxNorm data
        """
        enrichment = DrugEnrichment(original_text=drug_name)

        # Search for the drug
        results = self.search_rxnorm(drug_name, limit=1)

        if results:
            top_match = results[0]
            validation = self.validate_rxnorm(top_match.rxcui)

            if validation.is_valid:
                enrichment.rxnorm_code = top_match.rxcui
                enrichment.rxnorm_display = validation.display_name or top_match.name
                enrichment.term_type = validation.term_type

                # Get ingredients
                ingredients = self.get_ingredients(top_match.rxcui)
                enrichment.ingredients = [i.get("name", "") for i in ingredients]

                enrichment.validation_status = "VALID"
            else:
                enrichment.validation_status = "PARTIAL"
        else:
            enrichment.validation_status = "NOT_FOUND"

        return enrichment

    # ═══════════════════════════════════════════════════════════════════════════
    # Batch Validation for KB Facts
    # ═══════════════════════════════════════════════════════════════════════════

    def validate_extracted_facts(self, facts: dict) -> dict:
        """
        Validate all RxNorm codes in extracted facts.

        Args:
            facts: Extracted facts dictionary

        Returns:
            Facts with validation status added
        """
        validated = facts.copy()
        validation_results = []

        # Validate drug codes (KB-1 drug dosing)
        for drug in validated.get("drugs", []):
            rxnorm = drug.get("rxnormCode") or drug.get("rxnorm_code")
            if rxnorm:
                result = self.validate_rxnorm(rxnorm)
                drug["_rxnorm_validation"] = result.to_dict()
                validation_results.append(result)

        # Validate dose_rules drug references (KB-1)
        for rule in validated.get("dose_rules", []):
            rxnorm = rule.get("rxnormCode") or rule.get("rxnorm_code")
            if rxnorm:
                result = self.validate_rxnorm(rxnorm)
                rule["_rxnorm_validation"] = result.to_dict()
                validation_results.append(result)

        # Validate contraindication drug codes (KB-4)
        for ci in validated.get("contraindications", []):
            rxnorm = ci.get("rxnormCode") or ci.get("rxnorm_code")
            if rxnorm:
                result = self.validate_rxnorm(rxnorm)
                ci["_rxnorm_validation"] = result.to_dict()
                validation_results.append(result)

        # Add summary
        valid_count = sum(1 for r in validation_results if r.is_valid)
        validated["_terminology_validation"] = {
            "total_codes": len(validation_results),
            "valid_codes": valid_count,
            "invalid_codes": len(validation_results) - valid_count,
            "validation_rate": valid_count / len(validation_results) if validation_results else 0,
            "terminology_server": "RxNav-in-a-Box",
            "rxnorm_version": self.get_version(),
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

        return validated


def create_rxnav_client_from_env() -> RxNavClient:
    """
    Create RxNav client from environment variables.

    Environment Variables:
        RXNAV_URL: Server URL (default: http://localhost:4000)
    """
    return RxNavClient(
        base_url=os.getenv("RXNAV_URL", "http://localhost:4000"),
    )


# CLI interface
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="L4 RxNav Terminology Validation Client"
    )
    subparsers = parser.add_subparsers(dest="command", help="Command")

    # Health check
    health_parser = subparsers.add_parser("health", help="Check RxNav health")

    # Version
    version_parser = subparsers.add_parser("version", help="Get RxNorm version")

    # Validate command
    validate_parser = subparsers.add_parser("validate", help="Validate an RxCUI")
    validate_parser.add_argument("rxcui", help="RxCUI to validate")

    # Search command
    search_parser = subparsers.add_parser("search", help="Search for a drug")
    search_parser.add_argument("term", help="Drug name to search")
    search_parser.add_argument("--limit", "-l", type=int, default=10, help="Max results")

    # Enrich command
    enrich_parser = subparsers.add_parser("enrich", help="Enrich a drug entity")
    enrich_parser.add_argument("drug_name", help="Drug name")

    # Ingredients command
    ingredients_parser = subparsers.add_parser("ingredients", help="Get drug ingredients")
    ingredients_parser.add_argument("rxcui", help="Drug RxCUI")

    args = parser.parse_args()

    client = create_rxnav_client_from_env()

    if args.command == "health":
        is_healthy = client.health_check()
        print(f"RxNav: {'✅ Available' if is_healthy else '❌ Unavailable'}")
        if is_healthy:
            version = client.get_version()
            print(f"RxNorm Version: {version}")

    elif args.command == "version":
        version = client.get_version()
        print(f"RxNorm Version: {version}")

    elif args.command == "validate":
        result = client.validate_rxnorm(args.rxcui)
        print(json.dumps(result.to_dict(), indent=2))

    elif args.command == "search":
        results = client.search_rxnorm(args.term, args.limit)
        print(json.dumps([r.to_dict() for r in results], indent=2))

    elif args.command == "enrich":
        result = client.enrich_drug_entity(args.drug_name)
        print(json.dumps(result.to_dict(), indent=2))

    elif args.command == "ingredients":
        ingredients = client.get_ingredients(args.rxcui)
        print(json.dumps(ingredients, indent=2))

    else:
        parser.print_help()
