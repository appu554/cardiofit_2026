#!/usr/bin/env python3
"""
Citation YAML Validation Script
Validates citation files for completeness and correctness.
"""

import yaml
from pathlib import Path
from typing import Dict, List
from collections import defaultdict


def validate_citation(file_path: Path) -> tuple[bool, List[str]]:
    """Validate a single citation YAML file"""
    errors = []

    try:
        with open(file_path, 'r') as f:
            data = yaml.safe_load(f)

        # Required fields
        required_fields = [
            'pmid', 'doi', 'title', 'authors', 'journal',
            'publicationYear', 'volume', 'issue', 'pages',
            'studyType', 'evidenceQuality', 'abstract', 'pubmedUrl'
        ]

        for field in required_fields:
            if field not in data:
                errors.append(f"Missing required field: {field}")

        # Validate study type
        valid_study_types = ['RCT', 'META_ANALYSIS', 'SYSTEMATIC_REVIEW', 'GUIDELINE', 'COHORT', 'OBSERVATIONAL']
        if 'studyType' in data and data['studyType'] not in valid_study_types:
            errors.append(f"Invalid studyType: {data['studyType']}")

        # Validate evidence quality
        valid_qualities = ['HIGH', 'MODERATE', 'LOW', 'VERY_LOW']
        if 'evidenceQuality' in data and data['evidenceQuality'] not in valid_qualities:
            errors.append(f"Invalid evidenceQuality: {data['evidenceQuality']}")

        # Validate PubMed URL format
        if 'pubmedUrl' in data:
            expected_url = f"https://pubmed.ncbi.nlm.nih.gov/{data.get('pmid', '')}"
            if data['pubmedUrl'] != expected_url:
                errors.append(f"PubMed URL mismatch: {data['pubmedUrl']} vs {expected_url}")

        # Validate authors list
        if 'authors' in data:
            if not isinstance(data['authors'], list) or len(data['authors']) == 0:
                errors.append("Authors must be a non-empty list")

        return len(errors) == 0, errors

    except Exception as e:
        return False, [f"YAML parsing error: {str(e)}"]


def main():
    """Validate all citation files"""
    citations_dir = Path("/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/resources/knowledge-base/evidence/citations")

    print("=" * 80)
    print("CITATION YAML VALIDATION")
    print("=" * 80)

    citation_files = list(citations_dir.glob("pmid-*.yaml"))
    total_files = len(citation_files)
    valid_files = 0
    invalid_files = 0

    # Statistics
    stats = {
        'study_types': defaultdict(int),
        'evidence_quality': defaultdict(int),
        'journals': defaultdict(int),
        'years': defaultdict(int)
    }

    print(f"\n[Validation] Processing {total_files} citation files...")
    print()

    # Validate each file
    for file_path in sorted(citation_files):
        is_valid, errors = validate_citation(file_path)

        if is_valid:
            valid_files += 1

            # Collect statistics
            with open(file_path, 'r') as f:
                data = yaml.safe_load(f)
                stats['study_types'][data.get('studyType', 'UNKNOWN')] += 1
                stats['evidence_quality'][data.get('evidenceQuality', 'UNKNOWN')] += 1
                stats['journals'][data.get('journal', 'UNKNOWN')] += 1
                stats['years'][data.get('publicationYear', 'UNKNOWN')] += 1
        else:
            invalid_files += 1
            print(f"INVALID: {file_path.name}")
            for error in errors:
                print(f"  - {error}")
            print()

    # Summary
    print("\n" + "=" * 80)
    print("VALIDATION SUMMARY")
    print("=" * 80)
    print(f"Total files: {total_files}")
    print(f"Valid files: {valid_files} ({valid_files/total_files*100:.1f}%)")
    print(f"Invalid files: {invalid_files}")

    if valid_files > 0:
        # Study type statistics
        print(f"\n[Statistics] Study Type Distribution:")
        for study_type, count in sorted(stats['study_types'].items(), key=lambda x: -x[1]):
            print(f"  {study_type}: {count} ({count/valid_files*100:.1f}%)")

        # Evidence quality statistics
        print(f"\n[Statistics] Evidence Quality Distribution:")
        for quality, count in sorted(stats['evidence_quality'].items(), key=lambda x: -x[1]):
            print(f"  {quality}: {count} ({count/valid_files*100:.1f}%)")

        # Top journals
        print(f"\n[Statistics] Top 10 Journals:")
        top_journals = sorted(stats['journals'].items(), key=lambda x: -x[1])[:10]
        for journal, count in top_journals:
            print(f"  {journal}: {count}")

        # Year range
        years = [y for y in stats['years'].keys() if y != 'UNKNOWN']
        if years:
            print(f"\n[Statistics] Publication Year Range:")
            print(f"  Earliest: {min(years)}")
            print(f"  Latest: {max(years)}")
            print(f"  Median: {sorted(years)[len(years)//2]}")

    print("\n" + "=" * 80)
    print("VALIDATION COMPLETE")
    print("=" * 80)

    return 0 if invalid_files == 0 else 1


if __name__ == "__main__":
    exit(main())
