"""
KB-7 Terminology Service Client for V3 Pipeline L4 Validation.

This client replaces RxNav-in-a-Box for terminology validation, providing:
- RxNorm drug code validation
- SNOMED-CT clinical concept validation
- LOINC lab code validation
- ICD-10 diagnosis code validation (when available)
- Cross-system mapping capabilities
- FHIR-compliant validation operations

Usage:
    from kb7_client import KB7Client

    client = KB7Client(base_url="http://localhost:8092")
    result = client.validate_rxnorm("6809")  # Metformin
    if result.is_valid:
        print(f"Valid: {result.display_name}")
"""

import os
import httpx
from dataclasses import dataclass
from typing import Optional, List, Dict, Any
from enum import Enum

# PostgreSQL fallback for relationship queries when API endpoint unavailable
try:
    import psycopg2
    PSYCOPG2_AVAILABLE = True
except ImportError:
    PSYCOPG2_AVAILABLE = False


class TerminologySystem(Enum):
    """Supported terminology systems in KB-7."""
    RXNORM = "RxNorm"
    SNOMED = "SNOMED"
    LOINC = "LOINC"
    ICD10 = "ICD-10-CM"


@dataclass
class ValidationResult:
    """Result of a terminology code validation."""
    code: str
    system: str
    is_valid: bool
    display_name: Optional[str] = None
    fully_specified_name: Optional[str] = None
    term_type: Optional[str] = None
    active: bool = True
    synonyms: List[str] = None
    error: Optional[str] = None

    def __post_init__(self):
        if self.synonyms is None:
            self.synonyms = []


@dataclass
class MappingResult:
    """Result of a cross-terminology mapping lookup."""
    source_code: str
    source_system: str
    target_code: str
    target_system: str
    target_display: str
    equivalence: str  # equivalent, wider, narrower, inexact, unmatched
    confidence: float = 1.0


class KB7Client:
    """
    Client for KB-7 Terminology Service.

    Provides terminology validation and mapping operations for clinical
    decision support and guideline curation pipelines.
    """

    # System name normalization mapping
    SYSTEM_ALIASES = {
        "rxnorm": TerminologySystem.RXNORM,
        "rx": TerminologySystem.RXNORM,
        "snomed": TerminologySystem.SNOMED,
        "snomed-ct": TerminologySystem.SNOMED,
        "sct": TerminologySystem.SNOMED,
        "loinc": TerminologySystem.LOINC,
        "icd10": TerminologySystem.ICD10,
        "icd-10": TerminologySystem.ICD10,
        "icd10cm": TerminologySystem.ICD10,
        "icd-10-cm": TerminologySystem.ICD10,
    }

    # System URI mapping for FHIR compliance
    SYSTEM_URIS = {
        TerminologySystem.RXNORM: "http://www.nlm.nih.gov/research/umls/rxnorm",
        TerminologySystem.SNOMED: "http://snomed.info/sct",
        TerminologySystem.LOINC: "http://loinc.org",
        TerminologySystem.ICD10: "http://hl7.org/fhir/sid/icd-10-cm",
    }

    def __init__(
        self,
        base_url: str = None,
        timeout: float = 30.0,
        api_key: Optional[str] = None
    ):
        """
        Initialize KB-7 client.

        Args:
            base_url: KB-7 service URL (default: from KB7_URL env or localhost:8092)
            timeout: Request timeout in seconds
            api_key: Optional API key for authentication
        """
        self.base_url = base_url or os.environ.get("KB7_URL", "http://localhost:8092")
        self.base_url = self.base_url.rstrip("/")
        self.timeout = timeout
        self.api_key = api_key or os.environ.get("KB7_API_KEY")

        # Initialize HTTP client
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["X-API-Key"] = self.api_key

        self._client = httpx.Client(
            base_url=self.base_url,
            timeout=timeout,
            headers=headers
        )

        self._version_cache: Optional[Dict[str, str]] = None

        # PostgreSQL connection for direct relationship queries
        self._db_conn = None
        self._db_url = os.environ.get("KB7_DATABASE_URL", "postgresql://postgres:password@localhost:5437/kb_terminology")

    def close(self):
        """Close the HTTP client."""
        self._client.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

    def _normalize_system(self, system: str) -> TerminologySystem:
        """Normalize system name to TerminologySystem enum."""
        system_lower = system.lower().replace(" ", "").replace("_", "-")
        return self.SYSTEM_ALIASES.get(system_lower, TerminologySystem.RXNORM)

    # ─────────────────────────────────────────────────────────────────────────
    # Health Check
    # ─────────────────────────────────────────────────────────────────────────

    def health_check(self) -> bool:
        """
        Check if KB-7 service is healthy.

        Returns:
            True if service is healthy and responding
        """
        try:
            response = self._client.get("/health")
            if response.status_code == 200:
                data = response.json()
                return data.get("status") in ["healthy", "ok", "UP"]
            return False
        except Exception:
            return False

    def get_version(self) -> Dict[str, str]:
        """
        Get terminology system versions loaded in KB-7.

        Returns:
            Dictionary mapping system names to version strings
        """
        if self._version_cache:
            return self._version_cache

        try:
            response = self._client.get("/v1/terminology/systems")
            if response.status_code == 200:
                data = response.json()
                self._version_cache = {
                    sys.get("name", ""): sys.get("version", "unknown")
                    for sys in data.get("systems", [])
                }
                return self._version_cache
        except Exception:
            pass

        return {"RxNorm": "unknown", "SNOMED": "unknown", "LOINC": "unknown"}

    # ─────────────────────────────────────────────────────────────────────────
    # Individual Code Validation
    # ─────────────────────────────────────────────────────────────────────────

    def validate_code(
        self,
        code: str,
        system: str = "rxnorm"
    ) -> ValidationResult:
        """
        Validate a terminology code.

        Args:
            code: The code to validate
            system: Terminology system (rxnorm, snomed, loinc, icd10)

        Returns:
            ValidationResult with validation status and concept details
        """
        normalized_system = self._normalize_system(system)
        system_name = normalized_system.value

        try:
            # Try the concept lookup endpoint
            response = self._client.get(f"/v1/concepts/{system_name.lower()}/{code}")

            if response.status_code == 200:
                data = response.json()
                # Handle nested concept object in KB-7 API response
                concept = data.get("concept", data)
                return ValidationResult(
                    code=code,
                    system=system_name,
                    is_valid=True,
                    display_name=concept.get("display") or concept.get("preferredTerm"),
                    fully_specified_name=concept.get("definition") or concept.get("fullySpecifiedName"),
                    term_type=concept.get("termType"),
                    active=concept.get("status") == "active" if "status" in concept else concept.get("active", True),
                    synonyms=concept.get("synonyms", [])
                )
            elif response.status_code == 404:
                return ValidationResult(
                    code=code,
                    system=system_name,
                    is_valid=False,
                    error="Code not found"
                )
            else:
                return ValidationResult(
                    code=code,
                    system=system_name,
                    is_valid=False,
                    error=f"HTTP {response.status_code}: {response.text}"
                )

        except httpx.TimeoutException:
            return ValidationResult(
                code=code,
                system=system_name,
                is_valid=False,
                error="Request timeout"
            )
        except Exception as e:
            return ValidationResult(
                code=code,
                system=system_name,
                is_valid=False,
                error=str(e)
            )

    def validate_rxnorm(self, code: str) -> ValidationResult:
        """Validate an RxNorm drug code."""
        return self.validate_code(code, "rxnorm")

    def validate_snomed(self, code: str) -> ValidationResult:
        """Validate a SNOMED-CT clinical code."""
        return self.validate_code(code, "snomed")

    def validate_loinc(self, code: str) -> ValidationResult:
        """Validate a LOINC lab code."""
        return self.validate_code(code, "loinc")

    def validate_icd10(self, code: str) -> ValidationResult:
        """Validate an ICD-10-CM diagnosis code."""
        return self.validate_code(code, "icd10")

    # ─────────────────────────────────────────────────────────────────────────
    # Batch Validation
    # ─────────────────────────────────────────────────────────────────────────

    def batch_validate(
        self,
        codes: List[Dict[str, str]]
    ) -> List[ValidationResult]:
        """
        Validate multiple codes in a single request.

        Args:
            codes: List of dicts with 'code' and 'system' keys

        Returns:
            List of ValidationResult objects
        """
        results = []

        try:
            # Try batch endpoint first
            request_body = {
                "requests": [
                    {"code": c["code"], "system": self._normalize_system(c["system"]).value}
                    for c in codes
                ]
            }

            response = self._client.post("/v1/concepts/batch-lookup", json=request_body)

            if response.status_code == 200:
                data = response.json()
                for item in data.get("results", []):
                    results.append(ValidationResult(
                        code=item.get("code"),
                        system=item.get("system"),
                        is_valid=item.get("found", False),
                        display_name=item.get("preferredTerm"),
                        active=item.get("active", True)
                    ))
                return results

        except Exception:
            pass

        # Fallback to individual requests
        for code_info in codes:
            result = self.validate_code(code_info["code"], code_info["system"])
            results.append(result)

        return results

    # ─────────────────────────────────────────────────────────────────────────
    # Search
    # ─────────────────────────────────────────────────────────────────────────

    def search(
        self,
        query: str,
        system: Optional[str] = None,
        limit: int = 10,
        active_only: bool = True
    ) -> List[ValidationResult]:
        """
        Search for concepts by text query.

        Args:
            query: Search text (drug name, lab name, etc.)
            system: Optional system filter (rxnorm, snomed, loinc)
            limit: Maximum results to return
            active_only: Only return active concepts

        Returns:
            List of matching concepts as ValidationResult objects
        """
        params = {
            "q": query,
            "count": limit,
            "active": str(active_only).lower()
        }

        if system:
            params["system"] = self._normalize_system(system).value.lower()

        try:
            response = self._client.get("/v1/concepts", params=params)

            if response.status_code == 200:
                data = response.json()
                results = []
                for item in data.get("concepts", data.get("results", [])):
                    results.append(ValidationResult(
                        code=item.get("code"),
                        system=item.get("system"),
                        is_valid=True,
                        display_name=item.get("preferredTerm") or item.get("display"),
                        fully_specified_name=item.get("fullySpecifiedName"),
                        active=item.get("active", True),
                        synonyms=item.get("synonyms", [])
                    ))
                return results

        except Exception:
            pass

        return []

    # ─────────────────────────────────────────────────────────────────────────
    # Cross-System Mapping
    # ─────────────────────────────────────────────────────────────────────────

    def get_mappings(
        self,
        code: str,
        source_system: str,
        target_system: str
    ) -> List[MappingResult]:
        """
        Get cross-terminology mappings for a code.

        Args:
            code: Source code
            source_system: Source terminology system
            target_system: Target terminology system

        Returns:
            List of MappingResult objects
        """
        source_sys = self._normalize_system(source_system).value.lower()
        target_sys = self._normalize_system(target_system).value.lower()

        try:
            response = self._client.get(f"/v1/mappings/{source_sys}/{code}/{target_sys}")

            if response.status_code == 200:
                data = response.json()
                results = []
                for mapping in data.get("mappings", []):
                    results.append(MappingResult(
                        source_code=code,
                        source_system=source_system,
                        target_code=mapping.get("targetCode"),
                        target_system=target_system,
                        target_display=mapping.get("targetDisplay", ""),
                        equivalence=mapping.get("equivalence", "equivalent"),
                        confidence=mapping.get("confidence", 1.0)
                    ))
                return results

        except Exception:
            pass

        return []

    # ─────────────────────────────────────────────────────────────────────────
    # FHIR Operations (L4 Pipeline Compatibility)
    # ─────────────────────────────────────────────────────────────────────────

    def fhir_validate_code(
        self,
        code: str,
        system: str,
        display: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        FHIR $validate-code operation.

        Args:
            code: Code to validate
            system: System URI or short name
            display: Optional display to validate

        Returns:
            FHIR Parameters resource with validation result
        """
        normalized_system = self._normalize_system(system)
        system_uri = self.SYSTEM_URIS.get(normalized_system)

        params = {
            "code": code,
            "system": system_uri or system
        }
        if display:
            params["display"] = display

        try:
            response = self._client.post("/v1/concepts/validate", json=params)

            if response.status_code == 200:
                data = response.json()
                return {
                    "resourceType": "Parameters",
                    "parameter": [
                        {"name": "result", "valueBoolean": data.get("valid", False)},
                        {"name": "display", "valueString": data.get("display", "")}
                    ]
                }

        except Exception as e:
            return {
                "resourceType": "Parameters",
                "parameter": [
                    {"name": "result", "valueBoolean": False},
                    {"name": "message", "valueString": str(e)}
                ]
            }

        return {
            "resourceType": "Parameters",
            "parameter": [
                {"name": "result", "valueBoolean": False}
            ]
        }

    def fhir_lookup(self, code: str, system: str) -> Dict[str, Any]:
        """
        FHIR $lookup operation.

        Args:
            code: Code to look up
            system: System URI or short name

        Returns:
            FHIR Parameters resource with code details
        """
        result = self.validate_code(code, system)

        if result.is_valid:
            return {
                "resourceType": "Parameters",
                "parameter": [
                    {"name": "name", "valueString": result.system},
                    {"name": "display", "valueString": result.display_name},
                    {"name": "designation", "part": [
                        {"name": "value", "valueString": syn}
                        for syn in result.synonyms[:5]
                    ]} if result.synonyms else None
                ]
            }
        else:
            return {
                "resourceType": "OperationOutcome",
                "issue": [{
                    "severity": "error",
                    "code": "not-found",
                    "diagnostics": f"Code {code} not found in {system}"
                }]
            }

    # ─────────────────────────────────────────────────────────────────────────
    # Relationship Queries (L4 THREE-CHECK Pipeline)
    # ─────────────────────────────────────────────────────────────────────────

    def _get_db_connection(self):
        """Get or create PostgreSQL connection for direct relationship queries."""
        if not PSYCOPG2_AVAILABLE:
            return None

        if self._db_conn is None or self._db_conn.closed:
            try:
                self._db_conn = psycopg2.connect(self._db_url)
            except Exception:
                return None

        return self._db_conn

    def get_relationships(
        self,
        code: str,
        system: str = "rxnorm",
        relationship_type: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """
        Get relationships for a code from KB-7.

        Uses the concept_relationships table loaded with:
        - RxNorm: 1.6M relationships (isa, has_ingredient, tradename_of, etc.)
        - SNOMED: 617K IS-A hierarchy relationships
        - LOINC: 228K relationships

        Args:
            code: Source code to find relationships for
            system: Terminology system (rxnorm, snomed, loinc)
            relationship_type: Optional filter (isa, RB, RN, RO, etc.)

        Returns:
            List of relationship dicts with source_code, target_code, type
        """
        normalized_system = self._normalize_system(system).value
        params = {"system": normalized_system}
        if relationship_type:
            params["type"] = relationship_type

        # Try API endpoint first
        try:
            response = self._client.get(f"/v1/concepts/{normalized_system.lower()}/{code}/relationships", params=params)
            if response.status_code == 200:
                return response.json().get("relationships", [])
        except Exception:
            pass

        # Fallback to direct PostgreSQL query (uses loaded concept_relationships table)
        conn = self._get_db_connection()
        if conn:
            try:
                with conn.cursor() as cur:
                    sql = """
                        SELECT source_code, target_code, relationship_type, relationship_attr
                        FROM concept_relationships
                        WHERE source_code = %s AND source_vocab = %s
                    """
                    params_list = [code, normalized_system]

                    if relationship_type:
                        sql += " AND relationship_type = %s"
                        params_list.append(relationship_type)

                    sql += " LIMIT 100"
                    cur.execute(sql, params_list)

                    results = []
                    for row in cur.fetchall():
                        results.append({
                            "source_code": row[0],
                            "target_code": row[1],
                            "relationship_type": row[2],
                            "relationship_attr": row[3]
                        })
                    return results
            except Exception:
                pass

        return []

    def get_parents(self, code: str, system: str = "snomed") -> List[Dict[str, str]]:
        """
        Get parent concepts (IS-A relationships) for a code.

        Critical for subsumption testing in L4 THREE-CHECK pipeline.
        Example: CKD Stage 5 (433146000) → Chronic renal disease (709044004)

        Args:
            code: Child concept code
            system: Terminology system (snomed recommended for IS-A)

        Returns:
            List of parent concepts with code and display
        """
        relationships = self.get_relationships(code, system, "isa")
        parents = []
        for rel in relationships:
            parent_code = rel.get("target_code")
            if parent_code:
                # Look up parent display name (skip for performance if many)
                parent_display = None
                if len(relationships) <= 10:  # Only lookup if not too many
                    parent_result = self.validate_code(parent_code, system)
                    parent_display = parent_result.display_name if parent_result.is_valid else None
                parents.append({
                    "code": parent_code,
                    "display": parent_display
                })
        return parents

    def get_parents_direct(self, code: str, system: str = "snomed") -> List[str]:
        """
        Get parent codes directly from PostgreSQL without display lookup.

        Faster than get_parents() for subsumption testing.

        Args:
            code: Child concept code
            system: Terminology system

        Returns:
            List of parent codes (strings only)
        """
        normalized_system = self._normalize_system(system).value

        conn = self._get_db_connection()
        if conn:
            try:
                with conn.cursor() as cur:
                    cur.execute("""
                        SELECT target_code
                        FROM concept_relationships
                        WHERE source_code = %s
                          AND source_vocab = %s
                          AND relationship_type = 'isa'
                    """, [code, normalized_system])
                    return [row[0] for row in cur.fetchall()]
            except Exception:
                pass

        # Fallback to API-based lookup
        relationships = self.get_relationships(code, system, "isa")
        return [rel.get("target_code") for rel in relationships if rel.get("target_code")]

    def get_ingredients(self, rxcui: str, max_hops: int = 2) -> List[Dict[str, str]]:
        """
        Get active ingredients for an RxNorm clinical drug with automatic 2-hop traversal.

        RxNorm Hierarchy (requires 2 hops to reach base ingredient):
            Product (SCD) → Drug Component (SCDC) → Ingredient (IN)

        Example Path:
            861007 (metformin 500mg tablet)
               ↓ RO (constitutes)
            860974 (metformin hydrochloride 500 MG)
               ↓ RO (has_ingredient)
            6809 (metformin) ← BASE INGREDIENT

        This is critical for EHR-to-KB-1 matching:
            EHR prescribes product → need to find base ingredient in KB-1

        Args:
            rxcui: RxNorm concept ID (clinical drug or branded drug)
            max_hops: Maximum traversal depth (default 2 for full ingredient resolution)

        Returns:
            List of ingredient dicts with rxcui, display, relationship, hop_depth, and path
        """
        # Try optimized 2-hop PostgreSQL query first
        base_ingredients = self._get_base_ingredients_direct(rxcui)
        if base_ingredients:
            return base_ingredients

        # Fallback to iterative traversal
        return self._get_ingredients_iterative(rxcui, max_hops)

    def _get_base_ingredients_direct(self, rxcui: str) -> List[Dict[str, str]]:
        """
        Optimized 2-hop PostgreSQL query to find base ingredients.

        RxNorm Relationship Attributes (critical for correct traversal):
            - Hop 1: relationship_attr = 'constitutes' (Product → Drug Component)
            - Hop 2: relationship_attr = 'ingredient_of' (Drug Component → Base Ingredient)
            - Alternative: relationship_attr = 'precise_ingredient_of' (for salt forms)

        Example Path:
            861007 (metformin 500mg tablet)
               ↓ constitutes
            860974 (metformin hydrochloride 500 MG)
               ↓ ingredient_of
            6809 (metformin)

        This is much faster than iterative API calls for the common case.
        """
        conn = self._get_db_connection()
        if not conn:
            return []

        try:
            with conn.cursor() as cur:
                # 2-hop query: Product → SCDC (constitutes) → Ingredient (ingredient_of)
                cur.execute("""
                    WITH hop1 AS (
                        -- First hop: Product to Drug Component (SCDC)
                        -- Uses relationship_attr = 'constitutes' to avoid dose forms
                        SELECT
                            r1.source_code as product_code,
                            r1.target_code as scdc_code,
                            r1.relationship_attr as hop1_attr
                        FROM concept_relationships r1
                        WHERE r1.source_code = %s
                          AND r1.source_vocab = 'RxNorm'
                          AND r1.relationship_type = 'RO'
                          AND r1.relationship_attr = 'constitutes'
                    ),
                    hop2 AS (
                        -- Second hop: Drug Component to Base Ingredient
                        -- Uses relationship_attr IN ('ingredient_of', 'precise_ingredient_of')
                        SELECT
                            h1.product_code,
                            h1.scdc_code,
                            h1.hop1_attr,
                            r2.target_code as ingredient_code,
                            r2.relationship_attr as hop2_attr
                        FROM hop1 h1
                        JOIN concept_relationships r2
                            ON r2.source_code = h1.scdc_code
                           AND r2.source_vocab = 'RxNorm'
                           AND r2.relationship_type = 'RO'
                           AND r2.relationship_attr IN ('ingredient_of', 'precise_ingredient_of')
                    )
                    SELECT
                        h2.ingredient_code,
                        c.preferred_term,
                        h2.scdc_code,
                        cs.preferred_term as scdc_display,
                        h2.hop2_attr
                    FROM hop2 h2
                    LEFT JOIN concepts_rxnorm c ON c.code = h2.ingredient_code
                    LEFT JOIN concepts_rxnorm cs ON cs.code = h2.scdc_code
                """, [rxcui])

                results = []
                seen_ingredients = set()  # Deduplicate

                for row in cur.fetchall():
                    ingredient_code = row[0]
                    if ingredient_code and ingredient_code not in seen_ingredients:
                        seen_ingredients.add(ingredient_code)
                        hop2_attr = row[4] or "ingredient_of"
                        results.append({
                            "rxcui": ingredient_code,
                            "display": row[1],
                            "relationship": f"constitutes→{hop2_attr}",
                            "hop_depth": 2,
                            "ingredient_type": "base" if hop2_attr == "ingredient_of" else "precise",
                            "path": [
                                {"code": rxcui, "role": "product"},
                                {"code": row[2], "display": row[3], "role": "drug_component"},
                                {"code": ingredient_code, "display": row[1], "role": "base_ingredient"}
                            ]
                        })

                # Also check for direct 1-hop ingredients (some products link directly)
                if not results:
                    cur.execute("""
                        SELECT r.target_code, c.preferred_term, r.relationship_attr
                        FROM concept_relationships r
                        LEFT JOIN concepts_rxnorm c ON c.code = r.target_code
                        WHERE r.source_code = %s
                          AND r.source_vocab = 'RxNorm'
                          AND r.relationship_type = 'RO'
                          AND r.relationship_attr IN ('ingredient_of', 'precise_ingredient_of', 'has_ingredient')
                    """, [rxcui])

                    for row in cur.fetchall():
                        target_code = row[0]
                        if target_code and target_code not in seen_ingredients:
                            seen_ingredients.add(target_code)
                            rel_attr = row[2] or "ingredient_of"
                            results.append({
                                "rxcui": target_code,
                                "display": row[1],
                                "relationship": rel_attr,
                                "hop_depth": 1,
                                "ingredient_type": "base" if rel_attr == "ingredient_of" else "precise",
                                "path": [
                                    {"code": rxcui, "role": "product"},
                                    {"code": target_code, "display": row[1], "role": "ingredient"}
                                ]
                            })

                return results

        except Exception as e:
            # Log error but don't fail - fallback to iterative
            return []

    def _get_ingredients_iterative(self, rxcui: str, max_hops: int) -> List[Dict[str, str]]:
        """
        Iterative ingredient traversal using relationship API.

        Fallback when direct PostgreSQL query unavailable.
        Uses BFS to traverse specific RO relationship attributes:
            - Hop 1: 'constitutes' (Product → Drug Component)
            - Hop 2: 'ingredient_of' or 'precise_ingredient_of' (Drug Component → Ingredient)
        """
        ingredients = []
        seen = set()

        # Define valid relationship attributes per hop
        HOP1_ATTRS = {'constitutes'}  # Product → Drug Component
        HOP2_ATTRS = {'ingredient_of', 'precise_ingredient_of'}  # Drug Component → Ingredient

        # BFS traversal
        queue = [(rxcui, 0, [{"code": rxcui, "role": "product"}])]  # (code, depth, path)

        while queue:
            current_code, depth, path = queue.pop(0)

            if depth >= max_hops:
                continue

            relationships = self.get_relationships(current_code, "rxnorm", "RO")

            # Filter relationships by valid attributes for this hop
            valid_attrs = HOP1_ATTRS if depth == 0 else HOP2_ATTRS

            for rel in relationships:
                rel_attr = rel.get("relationship_attr", "")

                # Only follow relationships with correct attribute for this hop
                if rel_attr not in valid_attrs:
                    continue

                target = rel.get("target_code")
                if not target or target in seen:
                    continue

                seen.add(target)

                # Get display name for this target
                target_result = self.validate_rxnorm(target)
                target_display = target_result.display_name if target_result.is_valid else None

                # Determine role based on depth
                role = "drug_component" if depth == 0 else "base_ingredient"
                new_path = path + [{"code": target, "display": target_display, "role": role}]

                if depth == 0:
                    # Hop 1 complete, continue to hop 2 (look for ingredients)
                    queue.append((target, depth + 1, new_path))
                else:
                    # Hop 2 complete, this is a base ingredient
                    ing_type = "base" if rel_attr == "ingredient_of" else "precise"
                    ingredients.append({
                        "rxcui": target,
                        "display": target_display,
                        "relationship": f"constitutes→{rel_attr}",
                        "hop_depth": depth + 1,
                        "ingredient_type": ing_type,
                        "path": new_path
                    })

        return ingredients

    def get_base_ingredient(self, rxcui: str, prefer_generic: bool = True) -> Optional[Dict[str, str]]:
        """
        Get the primary base ingredient for a clinical drug.

        Convenience method that returns just the first/primary base ingredient.
        Useful for KB-1 matching where you need a single ingredient code.

        Priority order for selection:
            1. ingredient_type = "base" (true base ingredient like metformin)
            2. Generic over branded (no "[BrandName]" in display)
            3. ingredient_type = "precise" (salt form like metformin hydrochloride)

        Args:
            rxcui: RxNorm concept ID (clinical drug or branded drug)
            prefer_generic: If True, filter out branded ingredients (default True)

        Returns:
            Dict with rxcui and display, or None if not found
        """
        ingredients = self.get_ingredients(rxcui)
        if not ingredients:
            return None

        # Filter and sort for best match
        def sort_key(ing):
            # Priority 1: ingredient_type = "base" (True > False = 1 > 0)
            is_base = ing.get("ingredient_type") == "base"

            # Priority 2: Generic over branded (no brackets in name)
            display = ing.get("display") or ""
            is_generic = "[" not in display

            # Priority 3: Longer path = more specific (prefer 2-hop over 1-hop)
            hop_depth = ing.get("hop_depth", 1)

            # Return tuple for sorting (higher = better, so negate for desc)
            return (is_base, is_generic, hop_depth)

        # Filter out branded ingredients if prefer_generic
        if prefer_generic:
            generic_only = [
                ing for ing in ingredients
                if "[" not in (ing.get("display") or "")
            ]
            # Only use filtered list if it has results
            if generic_only:
                ingredients = generic_only

        # Sort by priority
        ingredients.sort(key=sort_key, reverse=True)

        ing = ingredients[0]
        return {
            "rxcui": ing["rxcui"],
            "display": ing.get("display"),
            "ingredient_type": ing.get("ingredient_type")
        }

    def subsumes(self, parent_code: str, child_code: str, system: str = "snomed") -> bool:
        """
        Test if parent_code subsumes child_code (child IS-A parent).

        This is the core of the THREE-CHECK pipeline's subsumption test.
        Walks up the IS-A hierarchy from child to find parent.

        Example: subsumes("709044004", "433146000", "snomed")
                 → True (Chronic renal disease subsumes CKD Stage 5)

        Args:
            parent_code: Potential ancestor concept
            child_code: Potential descendant concept
            system: Terminology system

        Returns:
            True if parent subsumes child (direct or transitive)
        """
        if parent_code == child_code:
            return True

        # BFS up the hierarchy (max 10 levels)
        visited = {child_code}
        queue = [child_code]
        max_depth = 10

        for _ in range(max_depth):
            if not queue:
                break

            current = queue.pop(0)
            # Use fast direct query instead of full lookup
            parent_codes = self.get_parents_direct(current, system)

            for p_code in parent_codes:
                if p_code == parent_code:
                    return True
                if p_code and p_code not in visited:
                    visited.add(p_code)
                    queue.append(p_code)

        return False

    def three_check_validate(
        self,
        code: str,
        system: str,
        value_set_codes: List[str]
    ) -> Dict[str, Any]:
        """
        THREE-CHECK validation pipeline (KB-7's full validation strategy).

        1. EXPANSION: Check if code is in expanded value set
        2. EXACT MATCH: Direct code lookup
        3. SUBSUMPTION: Check if code is subsumed by any value set member

        Args:
            code: Code to validate
            system: Terminology system
            value_set_codes: List of codes in the value set to check against

        Returns:
            Dict with validation result and match details
        """
        result = {
            "code": code,
            "system": system,
            "valid": False,
            "match_type": None,
            "matched_code": None
        }

        # CHECK 1: Exact match
        if code in value_set_codes:
            result["valid"] = True
            result["match_type"] = "EXACT_MATCH"
            result["matched_code"] = code
            return result

        # CHECK 2: Concept exists
        validation = self.validate_code(code, system)
        if not validation.is_valid:
            result["match_type"] = "INVALID_CODE"
            return result

        # CHECK 3: Subsumption - is this code subsumed by any value set member?
        for vs_code in value_set_codes:
            if self.subsumes(vs_code, code, system):
                result["valid"] = True
                result["match_type"] = "SUBSUMPTION"
                result["matched_code"] = vs_code
                result["display"] = validation.display_name
                return result

        result["match_type"] = "NO_MATCH"
        return result


# ─────────────────────────────────────────────────────────────────────────────
# Convenience function for L4 Pipeline
# ─────────────────────────────────────────────────────────────────────────────

def create_kb7_client(base_url: str = None) -> KB7Client:
    """
    Create a KB-7 client for L4 pipeline validation.

    Args:
        base_url: Optional KB-7 service URL

    Returns:
        Configured KB7Client instance
    """
    return KB7Client(base_url=base_url)


# ─────────────────────────────────────────────────────────────────────────────
# CLI for testing
# ─────────────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="KB-7 Terminology Client")
    parser.add_argument("--url", default="http://localhost:8092", help="KB-7 service URL")
    parser.add_argument("--code", help="Code to validate")
    parser.add_argument("--system", default="rxnorm", help="Terminology system")
    parser.add_argument("--search", help="Search query")
    parser.add_argument("--health", action="store_true", help="Health check only")
    args = parser.parse_args()

    client = KB7Client(base_url=args.url)

    print(f"KB-7 Client - {args.url}")
    print("=" * 50)

    if args.health or not (args.code or args.search):
        healthy = client.health_check()
        print(f"Health: {'✅ Healthy' if healthy else '❌ Unhealthy'}")

        if healthy:
            versions = client.get_version()
            print("\nLoaded Terminologies:")
            for sys, ver in versions.items():
                print(f"  {sys}: {ver}")

    if args.code:
        print(f"\nValidating {args.system.upper()} code: {args.code}")
        result = client.validate_code(args.code, args.system)

        if result.is_valid:
            print(f"  ✅ Valid")
            print(f"  Display: {result.display_name}")
            if result.fully_specified_name:
                print(f"  FSN: {result.fully_specified_name}")
            if result.synonyms:
                print(f"  Synonyms: {', '.join(result.synonyms[:3])}")
        else:
            print(f"  ❌ Invalid: {result.error}")

    if args.search:
        print(f"\nSearching: '{args.search}' in {args.system}")
        results = client.search(args.search, args.system, limit=5)

        for r in results:
            print(f"  [{r.code}] {r.display_name}")

    client.close()
