#!/usr/bin/env python3
"""
Vaidshala Phase 4: Batch Extraction Runner

Processes multiple guideline PDFs and consolidates output.

Usage:
    python batch_runner.py --input-dir /path/to/pdfs --output-dir /path/to/output
    python batch_runner.py --manifest manifest.json
"""

import argparse
import json
import sys
from pathlib import Path
from datetime import datetime
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, asdict

from table_extractor import (
    GuidelineTableExtractor,
    KB15Formatter,
    KB3TemporalFormatter,
    RecommendationRow
)


@dataclass
class ExtractionResult:
    """Result of extracting a single guideline."""
    guideline_id: str
    pdf_path: str
    success: bool
    recommendation_count: int
    temporal_constraint_count: int
    needs_llm_count: int
    llm_exposure_pct: float
    error_message: Optional[str] = None
    processing_time_seconds: float = 0.0


@dataclass
class BatchResult:
    """Consolidated batch extraction result."""
    run_id: str
    timestamp: str
    total_guidelines: int
    successful: int
    failed: int
    total_recommendations: int
    total_temporal_constraints: int
    overall_llm_exposure_pct: float
    individual_results: List[Dict]


class BatchRunner:
    """
    Run extraction on multiple guideline PDFs.
    """

    def __init__(self, output_dir: str):
        """
        Initialize batch runner.

        Args:
            output_dir: Directory for output files
        """
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)

        self.kb15_dir = self.output_dir / "kb15"
        self.kb3_dir = self.output_dir / "kb3"
        self.kb15_dir.mkdir(exist_ok=True)
        self.kb3_dir.mkdir(exist_ok=True)

        self.results: List[ExtractionResult] = []

    def run_from_manifest(self, manifest_path: str) -> BatchResult:
        """
        Run extraction based on manifest file.

        Manifest format:
        {
            "guidelines": [
                {
                    "id": "ACC-AHA-HF-2022",
                    "pdf_path": "/path/to/hf_guideline.pdf",
                    "metadata": {
                        "title": "2022 AHA/ACC/HFSA Guideline...",
                        "organization": "ACC/AHA/HFSA",
                        "year": "2022",
                        "doi": "10.1016/j.jacc.2021.12.012"
                    }
                }
            ]
        }
        """
        with open(manifest_path) as f:
            manifest = json.load(f)

        for entry in manifest.get("guidelines", []):
            result = self._process_single(
                pdf_path=entry["pdf_path"],
                guideline_id=entry["id"],
                metadata=entry.get("metadata", {})
            )
            self.results.append(result)

        return self._compile_batch_result()

    def run_from_directory(self, input_dir: str) -> BatchResult:
        """
        Run extraction on all PDFs in a directory.

        Args:
            input_dir: Directory containing PDF files
        """
        input_path = Path(input_dir)

        for pdf_file in input_path.glob("*.pdf"):
            guideline_id = pdf_file.stem
            result = self._process_single(
                pdf_path=str(pdf_file),
                guideline_id=guideline_id,
                metadata={"title": guideline_id}
            )
            self.results.append(result)

        return self._compile_batch_result()

    def _process_single(
        self,
        pdf_path: str,
        guideline_id: str,
        metadata: Dict[str, str]
    ) -> ExtractionResult:
        """
        Process a single guideline PDF.
        """
        import time
        start_time = time.time()

        print(f"\nProcessing: {guideline_id}")
        print(f"  PDF: {pdf_path}")

        try:
            # Extract recommendations
            extractor = GuidelineTableExtractor(guideline_id)
            recommendations = extractor.extract_from_pdf(pdf_path)

            # Get stats
            stats = extractor.get_stats()
            total = len(recommendations)
            needs_llm = sum(1 for r in recommendations if r.needs_llm_review)
            llm_pct = (needs_llm / total * 100) if total > 0 else 0

            # Format for KB-15
            kb15_formatter = KB15Formatter(metadata)
            kb15_output = kb15_formatter.format(recommendations)

            # Save KB-15 output
            kb15_file = self.kb15_dir / f"{guideline_id}_kb15.json"
            with open(kb15_file, 'w') as f:
                json.dump(kb15_output, f, indent=2)

            # Format for KB-3
            kb3_formatter = KB3TemporalFormatter()
            kb3_output = kb3_formatter.format(recommendations)

            # Save KB-3 output
            kb3_file = self.kb3_dir / f"{guideline_id}_kb3.json"
            with open(kb3_file, 'w') as f:
                json.dump(kb3_output, f, indent=2)

            elapsed = time.time() - start_time

            print(f"  Recommendations: {total}")
            print(f"  Temporal constraints: {len(kb3_output)}")
            print(f"  LLM exposure: {llm_pct:.1f}%")
            print(f"  Time: {elapsed:.2f}s")

            return ExtractionResult(
                guideline_id=guideline_id,
                pdf_path=pdf_path,
                success=True,
                recommendation_count=total,
                temporal_constraint_count=len(kb3_output),
                needs_llm_count=needs_llm,
                llm_exposure_pct=llm_pct,
                processing_time_seconds=elapsed
            )

        except Exception as e:
            elapsed = time.time() - start_time
            print(f"  ERROR: {str(e)}")

            return ExtractionResult(
                guideline_id=guideline_id,
                pdf_path=pdf_path,
                success=False,
                recommendation_count=0,
                temporal_constraint_count=0,
                needs_llm_count=0,
                llm_exposure_pct=0,
                error_message=str(e),
                processing_time_seconds=elapsed
            )

    def _compile_batch_result(self) -> BatchResult:
        """Compile batch results."""
        successful = [r for r in self.results if r.success]
        failed = [r for r in self.results if not r.success]

        total_recs = sum(r.recommendation_count for r in successful)
        total_temporal = sum(r.temporal_constraint_count for r in successful)
        total_llm = sum(r.needs_llm_count for r in successful)

        overall_llm_pct = (total_llm / total_recs * 100) if total_recs > 0 else 0

        batch_result = BatchResult(
            run_id=f"batch-{datetime.now().strftime('%Y%m%d-%H%M%S')}",
            timestamp=datetime.utcnow().isoformat() + "Z",
            total_guidelines=len(self.results),
            successful=len(successful),
            failed=len(failed),
            total_recommendations=total_recs,
            total_temporal_constraints=total_temporal,
            overall_llm_exposure_pct=overall_llm_pct,
            individual_results=[asdict(r) for r in self.results]
        )

        # Save batch summary
        summary_file = self.output_dir / "batch_summary.json"
        with open(summary_file, 'w') as f:
            json.dump(asdict(batch_result), f, indent=2)

        print("\n" + "=" * 60)
        print("BATCH EXTRACTION COMPLETE")
        print("=" * 60)
        print(f"  Total guidelines: {batch_result.total_guidelines}")
        print(f"  Successful: {batch_result.successful}")
        print(f"  Failed: {batch_result.failed}")
        print(f"  Total recommendations: {batch_result.total_recommendations}")
        print(f"  Total temporal constraints: {batch_result.total_temporal_constraints}")
        print(f"  Overall LLM exposure: {batch_result.overall_llm_exposure_pct:.1f}%")
        print(f"\nOutput saved to: {self.output_dir}")

        return batch_result


def create_sample_manifest(output_path: str):
    """Create a sample manifest file."""
    sample = {
        "description": "Sample extraction manifest for Vaidshala Phase 4",
        "created": datetime.utcnow().isoformat() + "Z",
        "guidelines": [
            {
                "id": "ACC-AHA-HF-2022",
                "pdf_path": "/path/to/2022_hf_guideline.pdf",
                "metadata": {
                    "title": "2022 AHA/ACC/HFSA Guideline for the Management of Heart Failure",
                    "organization": "ACC/AHA/HFSA",
                    "year": "2022",
                    "doi": "10.1016/j.jacc.2021.12.012"
                }
            },
            {
                "id": "SSC-2021",
                "pdf_path": "/path/to/ssc_2021.pdf",
                "metadata": {
                    "title": "Surviving Sepsis Campaign: International Guidelines 2021",
                    "organization": "SCCM/ESICM",
                    "year": "2021",
                    "doi": "10.1007/s00134-021-06506-y"
                }
            },
            {
                "id": "ADA-2024",
                "pdf_path": "/path/to/ada_standards_2024.pdf",
                "metadata": {
                    "title": "Standards of Care in Diabetes—2024",
                    "organization": "American Diabetes Association",
                    "year": "2024",
                    "doi": "10.2337/dc24-SINT"
                }
            }
        ]
    }

    with open(output_path, 'w') as f:
        json.dump(sample, f, indent=2)

    print(f"Sample manifest created: {output_path}")


def main():
    parser = argparse.ArgumentParser(
        description="Vaidshala Phase 4: Batch Guideline Extraction"
    )

    parser.add_argument(
        "--manifest",
        help="Path to manifest JSON file"
    )
    parser.add_argument(
        "--input-dir",
        help="Directory containing PDF files"
    )
    parser.add_argument(
        "--output-dir",
        default="./extraction_output",
        help="Output directory for results"
    )
    parser.add_argument(
        "--create-sample-manifest",
        help="Create a sample manifest file at specified path"
    )

    args = parser.parse_args()

    if args.create_sample_manifest:
        create_sample_manifest(args.create_sample_manifest)
        return

    if not args.manifest and not args.input_dir:
        print("Error: Please provide --manifest or --input-dir")
        parser.print_help()
        sys.exit(1)

    runner = BatchRunner(args.output_dir)

    if args.manifest:
        runner.run_from_manifest(args.manifest)
    else:
        runner.run_from_directory(args.input_dir)


if __name__ == "__main__":
    main()
