#!/usr/bin/env python3
"""
Vaidshala Phase 5: Phase 4 Integration Module

Connects Phase 4 Table Extractor output to Phase 5 Constrained Atomiser.

Workflow:
1. Load Phase 4 KB-15 JSON output
2. Identify recommendations needing LLM assistance (needs_llm_review=True)
3. Check AtomiserRegistry for existing CQL coverage
4. For gaps: Use ConstrainedAtomiser to extract structured data
5. Route to appropriate Knowledge Bases via KBRouter

Usage:
    python phase4_integration.py SSC-2021_kb15.json

    # Or programmatic:
    from phase4_integration import Phase4Integrator
    integrator = Phase4Integrator()
    results = integrator.process_extraction("SSC-2021_kb15.json")
"""

import json
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass, field

# Import Phase 5 components
from atomiser_registry import AtomiserRegistry
from constrained_atomiser import ConstrainedAtomiser, AtomiserConfig, AtomisedRecommendation
from kb_router import KBRouter, KnowledgeBase, RoutingResult


@dataclass
class IntegrationResult:
    """Result of Phase 4 → Phase 5 integration."""
    guideline_id: str
    total_recommendations: int
    deterministic_count: int  # Phase 4 fully extracted
    llm_assisted_count: int   # Phase 5 atomised
    skipped_count: int        # Existing CQL coverage
    routed_payloads: List[Dict]
    processing_stats: Dict
    errors: List[str]


class Phase4Integrator:
    """
    Integrates Phase 4 deterministic extraction with Phase 5 LLM atomiser.

    PHILOSOPHY: "Deterministic First, LLM as Last Resort"

    Flow:
    1. Phase 4 extracts ~85% deterministically (COR/LOE patterns)
    2. Phase 5 handles ~15% gaps (ambiguous text, novel formats)
    3. All LLM output is DRAFT status, requires SME review
    """

    def __init__(
        self,
        extractor_output_dir: str = None,
        cql_base_path: str = None,
        enable_llm: bool = False,  # Disabled by default for testing
        llm_config: AtomiserConfig = None
    ):
        """
        Initialize integrator.

        Args:
            extractor_output_dir: Directory containing Phase 4 KB-15 JSON files
            cql_base_path: Base path to existing CQL files for registry
            enable_llm: Whether to call actual LLM (disabled for testing)
            llm_config: Configuration for ConstrainedAtomiser
        """
        self.extractor_dir = Path(extractor_output_dir) if extractor_output_dir else \
                            Path(__file__).parent.parent / "extractors"
        self.cql_base_path = cql_base_path
        self.enable_llm = enable_llm

        # Initialize components
        self.registry = AtomiserRegistry(cql_base_path=cql_base_path)
        self.router = KBRouter()

        # Initialize atomiser only if LLM enabled
        if enable_llm:
            config = llm_config or AtomiserConfig(max_confidence=0.85)
            self.atomiser = ConstrainedAtomiser(config=config)
        else:
            self.atomiser = None

    def process_extraction(self, kb15_file: str) -> IntegrationResult:
        """
        Process a Phase 4 KB-15 extraction file.

        Args:
            kb15_file: Path to KB-15 JSON file (absolute or relative to extractor_dir)

        Returns:
            IntegrationResult with processing statistics and routed payloads
        """
        # Resolve file path
        if Path(kb15_file).is_absolute():
            file_path = Path(kb15_file)
        else:
            file_path = self.extractor_dir / kb15_file

        if not file_path.exists():
            return IntegrationResult(
                guideline_id="UNKNOWN",
                total_recommendations=0,
                deterministic_count=0,
                llm_assisted_count=0,
                skipped_count=0,
                routed_payloads=[],
                processing_stats={},
                errors=[f"File not found: {file_path}"]
            )

        # Load KB-15 data
        with open(file_path) as f:
            kb15_data = json.load(f)

        # Handle both list format (direct list) and dict format (with "recommendations" key)
        if isinstance(kb15_data, list):
            recommendations = kb15_data
            # Extract guideline ID from first recommendation or filename
            if recommendations and recommendations[0].get("recommendation_id"):
                guideline_id = recommendations[0]["recommendation_id"].rsplit("-", 1)[0]
            else:
                guideline_id = file_path.stem.replace("_kb15", "")
        else:
            guideline_id = kb15_data.get("guideline_id", file_path.stem.replace("_kb15", ""))
            recommendations = kb15_data.get("recommendations", [])

        # Process each recommendation
        routed_payloads = []
        deterministic_count = 0
        llm_assisted_count = 0
        skipped_count = 0
        errors = []

        for rec in recommendations:
            try:
                result = self._process_recommendation(rec, guideline_id)

                if result["action"] == "deterministic":
                    deterministic_count += 1
                elif result["action"] == "llm_assisted":
                    llm_assisted_count += 1
                elif result["action"] == "skipped":
                    skipped_count += 1

                if result.get("routed_payload"):
                    routed_payloads.append(result["routed_payload"])

            except Exception as e:
                errors.append(f"Error processing {rec.get('recommendation_id', 'unknown')}: {str(e)}")

        return IntegrationResult(
            guideline_id=guideline_id,
            total_recommendations=len(recommendations),
            deterministic_count=deterministic_count,
            llm_assisted_count=llm_assisted_count,
            skipped_count=skipped_count,
            routed_payloads=routed_payloads,
            processing_stats={
                "deterministic_pct": round(deterministic_count / max(len(recommendations), 1) * 100, 1),
                "llm_assisted_pct": round(llm_assisted_count / max(len(recommendations), 1) * 100, 1),
                "skipped_pct": round(skipped_count / max(len(recommendations), 1) * 100, 1),
            },
            errors=errors
        )

    def _process_recommendation(self, rec: Dict, guideline_id: str) -> Dict:
        """
        Process a single recommendation.

        Decision tree:
        1. Check if needs_llm_review is False → Use deterministic extraction
        2. Check registry for existing CQL → Skip if covered
        3. Otherwise → Use LLM atomiser (if enabled) or mock
        """
        rec_id = rec.get("recommendation_id", "unknown")
        rec_text = rec.get("recommendation", {}).get("text", "")
        needs_llm = rec.get("governance", {}).get("needs_llm_review", False)

        # Case 1: Deterministic extraction succeeded
        if not needs_llm:
            # Route the existing Phase 4 extraction
            payload = self._phase4_to_routable(rec, guideline_id)
            routing_result = self._route_payload(payload)

            return {
                "action": "deterministic",
                "recommendation_id": rec_id,
                "routed_payload": routing_result
            }

        # Case 2: Check if existing CQL covers this topic
        if self.registry.get_existing_cql(rec_text):
            return {
                "action": "skipped",
                "recommendation_id": rec_id,
                "reason": "Existing CQL coverage found",
                "routed_payload": None
            }

        # Case 3: Needs LLM assistance
        if self.enable_llm and self.atomiser:
            # Use ConstrainedAtomiser for extraction
            atomised = self.atomiser.atomise(
                text_chunk=rec_text,
                extraction_type="recommendation",
                guideline_source=guideline_id,
                existing_evidence=rec.get("evidence_envelope", {})
            )

            # Route the atomised result
            routing_result = self._route_atomised(atomised)

            return {
                "action": "llm_assisted",
                "recommendation_id": rec_id,
                "routed_payload": routing_result
            }
        else:
            # Mock LLM processing for testing
            return {
                "action": "llm_assisted",
                "recommendation_id": rec_id,
                "note": "LLM disabled - would atomise in production",
                "routed_payload": self._mock_atomised_routing(rec, guideline_id)
            }

    def _phase4_to_routable(self, rec: Dict, guideline_id: str) -> Dict:
        """Convert Phase 4 output to routable format."""
        evidence = rec.get("evidence_envelope", {})
        temporal = rec.get("temporal_constraints", [])

        return {
            "recommendation_id": rec.get("recommendation_id"),
            "original_text": rec.get("recommendation", {}).get("text", ""),
            "extraction": {
                "class_of_recommendation": evidence.get("class_of_recommendation"),
                "level_of_evidence": evidence.get("level_of_evidence"),
                "source_guideline": guideline_id,
                "temporal_constraints": temporal,
                "clinical_context": rec.get("recommendation", {}).get("clinical_context"),
            },
            "confidence": rec.get("confidence", 0.9),  # Phase 4 is high confidence
            "guideline_source": guideline_id,
            "governance_status": rec.get("governance", {}).get("status", "PENDING_REVIEW"),
            "requires_sme_review": rec.get("governance", {}).get("requires_sme_review", True),
            "extracted_at": rec.get("provenance", {}).get("extracted_at", datetime.utcnow().isoformat()),
        }

    def _route_payload(self, payload: Dict) -> Dict:
        """Route a payload using KBRouter."""
        # Create mock recommendation object for router
        mock_rec = type('MockRec', (), payload)()

        # Get routing result
        result = self.router.route(mock_rec)

        return {
            "kb_target": result.kb_target.value,
            "payload": result.payload,
            "validation": result.validation_result,
            "routing_confidence": result.routing_confidence,
        }

    def _route_atomised(self, atomised: AtomisedRecommendation) -> Dict:
        """Route an atomised recommendation."""
        result = self.router.route(atomised)

        return {
            "kb_target": result.kb_target.value,
            "payload": result.payload,
            "validation": result.validation_result,
            "routing_confidence": result.routing_confidence,
            "governance": {
                "status": "DRAFT",
                "requires_sme_review": True,
                "llm_confidence": atomised.confidence,
            }
        }

    def _mock_atomised_routing(self, rec: Dict, guideline_id: str) -> Dict:
        """Mock atomised routing for testing without LLM."""
        return {
            "kb_target": "KB-15",  # Default to evidence engine
            "payload": {
                "recommendation_id": rec.get("recommendation_id"),
                "original_text": rec.get("recommendation", {}).get("text", ""),
                "governance_status": "DRAFT",
                "requires_sme_review": True,
                "mock_extraction": True,
                "note": "LLM disabled - mock payload for testing",
            },
            "validation": {"valid": True, "errors": [], "warnings": ["Mock extraction - enable LLM for production"]},
            "routing_confidence": 0.5,
        }

    def save_results(self, result: IntegrationResult, output_path: str = None) -> str:
        """Save integration results to JSON."""
        if not output_path:
            output_path = self.extractor_dir / f"{result.guideline_id}_phase5_integrated.json"

        output_data = {
            "integration_timestamp": datetime.utcnow().isoformat(),
            "guideline_id": result.guideline_id,
            "statistics": {
                "total_recommendations": result.total_recommendations,
                "deterministic_count": result.deterministic_count,
                "llm_assisted_count": result.llm_assisted_count,
                "skipped_count": result.skipped_count,
                "percentages": result.processing_stats,
            },
            "routed_payloads": result.routed_payloads,
            "errors": result.errors,
        }

        with open(output_path, 'w') as f:
            json.dump(output_data, f, indent=2, default=str)

        return str(output_path)


def main():
    """Demo Phase 4 → Phase 5 integration."""
    print("=" * 70)
    print("Phase 5: Integration with Phase 4 Table Extractor")
    print("=" * 70)

    # Initialize integrator (LLM disabled for demo)
    integrator = Phase4Integrator(enable_llm=False)

    # Check for SSC-2021 extraction
    ssc_file = integrator.extractor_dir / "SSC-2021_kb15.json"

    if ssc_file.exists():
        print(f"\n📄 Processing: {ssc_file.name}")
        print("-" * 50)

        result = integrator.process_extraction(str(ssc_file))

        print(f"\n📊 Integration Results for {result.guideline_id}:")
        print(f"   Total recommendations: {result.total_recommendations}")
        print(f"   ✅ Deterministic (Phase 4): {result.deterministic_count} ({result.processing_stats.get('deterministic_pct', 0)}%)")
        print(f"   🤖 LLM Assisted (Phase 5): {result.llm_assisted_count} ({result.processing_stats.get('llm_assisted_pct', 0)}%)")
        print(f"   ⏭️  Skipped (Existing CQL): {result.skipped_count} ({result.processing_stats.get('skipped_pct', 0)}%)")

        if result.errors:
            print(f"\n   ⚠️ Errors: {len(result.errors)}")
            for err in result.errors[:5]:
                print(f"      - {err}")

        # Save results
        output_path = integrator.save_results(result)
        print(f"\n💾 Results saved to: {output_path}")

        # Show KB routing summary
        kb_counts = {}
        for payload in result.routed_payloads:
            kb = payload.get("kb_target", "UNKNOWN")
            kb_counts[kb] = kb_counts.get(kb, 0) + 1

        print(f"\n🎯 KB Routing Summary:")
        for kb, count in sorted(kb_counts.items()):
            print(f"   {kb}: {count} recommendations")

    else:
        print(f"\n⚠️ No Phase 4 extraction found at: {ssc_file}")
        print("   Run Phase 4 extractor first:")
        print("   cd ../extractors && python table_extractor.py pdfs/SSC-2021.pdf SSC-2021")

    print("\n" + "=" * 70)
    print("Philosophy: 'Deterministic First, LLM as Last Resort'")
    print("=" * 70)


if __name__ == "__main__":
    if len(sys.argv) > 1:
        # Process specific file
        integrator = Phase4Integrator(enable_llm=False)
        result = integrator.process_extraction(sys.argv[1])
        print(json.dumps({
            "guideline_id": result.guideline_id,
            "total": result.total_recommendations,
            "deterministic": result.deterministic_count,
            "llm_assisted": result.llm_assisted_count,
            "skipped": result.skipped_count,
        }, indent=2))
    else:
        main()
