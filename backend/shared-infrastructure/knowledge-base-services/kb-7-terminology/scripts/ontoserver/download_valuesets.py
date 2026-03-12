#!/usr/bin/env python3
"""
Download ALL ValueSets from CSIRO Public Ontoserver
Target: ~23,710 ValueSets from https://r4.ontoserver.csiro.au/fhir/

Usage:
    python download_valuesets.py                    # Download all
    python download_valuesets.py --resume           # Resume from last position
    python download_valuesets.py --filter sepsis    # Download only matching name
    python download_valuesets.py --expand           # Also expand each ValueSet
"""

import os
import sys
import json
import time
import argparse
import requests
from datetime import datetime
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Optional, Dict, List, Any

# ============================================================================
# Configuration
# ============================================================================

ONTOSERVER_BASE_URL = "https://r4.ontoserver.csiro.au/fhir"
OUTPUT_DIR = Path(__file__).parent.parent.parent / "data" / "ontoserver-valuesets"
PROGRESS_FILE = OUTPUT_DIR / "_progress.json"
SUMMARY_FILE = OUTPUT_DIR / "_summary.json"

# API Settings
PAGE_SIZE = 100  # ValueSets per page
REQUEST_TIMEOUT = 60  # seconds
RETRY_ATTEMPTS = 3
RETRY_DELAY = 5  # seconds between retries
CONCURRENT_DOWNLOADS = 5  # parallel downloads for expansion

# ============================================================================
# Progress Tracking
# ============================================================================

class ProgressTracker:
    """Track download progress for resume capability"""

    def __init__(self, progress_file: Path):
        self.progress_file = progress_file
        self.data = self._load()

    def _load(self) -> Dict:
        if self.progress_file.exists():
            with open(self.progress_file, 'r') as f:
                return json.load(f)
        return {
            "started_at": datetime.now().isoformat(),
            "last_page": 0,
            "total_downloaded": 0,
            "total_expanded": 0,
            "failed": [],
            "downloaded_ids": []
        }

    def save(self):
        with open(self.progress_file, 'w') as f:
            json.dump(self.data, f, indent=2)

    def mark_downloaded(self, vs_id: str):
        if vs_id not in self.data["downloaded_ids"]:
            self.data["downloaded_ids"].append(vs_id)
            self.data["total_downloaded"] += 1

    def mark_expanded(self):
        self.data["total_expanded"] += 1

    def mark_failed(self, vs_id: str, error: str):
        self.data["failed"].append({"id": vs_id, "error": error})

    def is_downloaded(self, vs_id: str) -> bool:
        return vs_id in self.data["downloaded_ids"]

    def set_page(self, page: int):
        self.data["last_page"] = page

# ============================================================================
# Ontoserver API Client
# ============================================================================

class OntoserverClient:
    """FHIR API client for CSIRO Ontoserver"""

    def __init__(self, base_url: str = ONTOSERVER_BASE_URL):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.session.headers.update({
            "Accept": "application/fhir+json",
            "User-Agent": "KB7-ValueSet-Downloader/1.0"
        })

    def _request(self, method: str, url: str, **kwargs) -> Optional[Dict]:
        """Make HTTP request with retry logic"""
        for attempt in range(RETRY_ATTEMPTS):
            try:
                response = self.session.request(
                    method, url,
                    timeout=REQUEST_TIMEOUT,
                    **kwargs
                )
                response.raise_for_status()
                return response.json()
            except requests.exceptions.RequestException as e:
                if attempt < RETRY_ATTEMPTS - 1:
                    print(f"  Retry {attempt + 1}/{RETRY_ATTEMPTS}: {e}")
                    time.sleep(RETRY_DELAY)
                else:
                    raise
        return None

    def search_valuesets(self, offset: int = 0, count: int = PAGE_SIZE,
                         name_filter: Optional[str] = None) -> Dict:
        """Search ValueSets with pagination"""
        params = {
            "_count": count,
            "_offset": offset,
            "_summary": "true"  # Get summary first, then full details
        }
        if name_filter:
            params["name:contains"] = name_filter

        url = f"{self.base_url}/ValueSet"
        return self._request("GET", url, params=params)

    def get_valueset(self, vs_id: str) -> Optional[Dict]:
        """Get full ValueSet by ID"""
        url = f"{self.base_url}/ValueSet/{vs_id}"
        return self._request("GET", url)

    def expand_valueset(self, vs_id: str) -> Optional[Dict]:
        """Expand ValueSet to get all codes"""
        url = f"{self.base_url}/ValueSet/{vs_id}/$expand"
        try:
            return self._request("GET", url)
        except Exception as e:
            # Some ValueSets can't be expanded - that's OK
            print(f"  Warning: Cannot expand {vs_id}: {e}")
            return None

    def get_total_count(self, name_filter: Optional[str] = None) -> int:
        """Get total number of ValueSets"""
        result = self.search_valuesets(offset=0, count=1, name_filter=name_filter)
        return result.get("total", 0) if result else 0

# ============================================================================
# ValueSet Downloader
# ============================================================================

class ValueSetDownloader:
    """Download and save ValueSets from Ontoserver"""

    def __init__(self, output_dir: Path, expand: bool = False):
        self.output_dir = output_dir
        self.expand = expand
        self.client = OntoserverClient()
        self.progress = ProgressTracker(PROGRESS_FILE)

        # Create output directories
        self.output_dir.mkdir(parents=True, exist_ok=True)
        (self.output_dir / "definitions").mkdir(exist_ok=True)
        (self.output_dir / "expansions").mkdir(exist_ok=True)

    def save_valueset(self, valueset: Dict, expanded: Optional[Dict] = None):
        """Save ValueSet definition and optional expansion"""
        vs_id = valueset.get("id", "unknown")

        # Save definition
        def_file = self.output_dir / "definitions" / f"{vs_id}.json"
        with open(def_file, 'w') as f:
            json.dump(valueset, f, indent=2)

        # Save expansion if available
        if expanded:
            exp_file = self.output_dir / "expansions" / f"{vs_id}_expanded.json"
            with open(exp_file, 'w') as f:
                json.dump(expanded, f, indent=2)

    def download_page(self, offset: int, name_filter: Optional[str] = None) -> List[Dict]:
        """Download a page of ValueSets"""
        result = self.client.search_valuesets(offset=offset, name_filter=name_filter)
        if not result:
            return []

        entries = result.get("entry", [])
        valuesets = []

        for entry in entries:
            vs = entry.get("resource", {})
            vs_id = vs.get("id")

            if not vs_id:
                continue

            # Skip if already downloaded (resume mode)
            if self.progress.is_downloaded(vs_id):
                continue

            valuesets.append(vs)

        return valuesets

    def download_full_valueset(self, vs_summary: Dict) -> Optional[Dict]:
        """Download full ValueSet details and optionally expand"""
        vs_id = vs_summary.get("id")
        if not vs_id:
            return None

        try:
            # Get full definition
            full_vs = self.client.get_valueset(vs_id)
            if not full_vs:
                return None

            # Optionally expand
            expanded = None
            if self.expand:
                expanded = self.client.expand_valueset(vs_id)
                if expanded:
                    self.progress.mark_expanded()

            # Save to disk
            self.save_valueset(full_vs, expanded)
            self.progress.mark_downloaded(vs_id)

            return full_vs

        except Exception as e:
            self.progress.mark_failed(vs_id, str(e))
            return None

    def download_all(self, name_filter: Optional[str] = None, resume: bool = False):
        """Download all ValueSets with progress tracking"""

        # Get total count
        total = self.client.get_total_count(name_filter)
        print(f"\n{'='*60}")
        print(f"  ONTOSERVER VALUESET DOWNLOADER")
        print(f"{'='*60}")
        print(f"  Source: {ONTOSERVER_BASE_URL}")
        print(f"  Total ValueSets: {total:,}")
        print(f"  Filter: {name_filter or 'None'}")
        print(f"  Expand: {self.expand}")
        print(f"  Output: {self.output_dir}")
        print(f"{'='*60}\n")

        if total == 0:
            print("No ValueSets found!")
            return

        # Calculate starting point
        start_offset = 0
        if resume and self.progress.data["last_page"] > 0:
            start_offset = self.progress.data["last_page"] * PAGE_SIZE
            print(f"Resuming from offset {start_offset} (page {self.progress.data['last_page']})")

        # Download in pages
        downloaded = self.progress.data["total_downloaded"]

        for offset in range(start_offset, total, PAGE_SIZE):
            page_num = offset // PAGE_SIZE + 1
            total_pages = (total + PAGE_SIZE - 1) // PAGE_SIZE

            print(f"\n[Page {page_num}/{total_pages}] Fetching offset {offset}...")

            # Get page of ValueSets
            valuesets = self.download_page(offset, name_filter)

            if not valuesets:
                print(f"  No new ValueSets on this page")
                self.progress.set_page(page_num)
                self.progress.save()
                continue

            # Download full details (with optional parallel expansion)
            if self.expand and len(valuesets) > 1:
                # Parallel download for expansion
                with ThreadPoolExecutor(max_workers=CONCURRENT_DOWNLOADS) as executor:
                    futures = {
                        executor.submit(self.download_full_valueset, vs): vs
                        for vs in valuesets
                    }
                    for future in as_completed(futures):
                        vs = futures[future]
                        try:
                            result = future.result()
                            if result:
                                downloaded += 1
                                print(f"  [{downloaded:,}/{total:,}] {vs.get('id', 'unknown')}")
                        except Exception as e:
                            print(f"  ERROR: {vs.get('id', 'unknown')}: {e}")
            else:
                # Sequential download
                for vs in valuesets:
                    result = self.download_full_valueset(vs)
                    if result:
                        downloaded += 1
                        print(f"  [{downloaded:,}/{total:,}] {vs.get('id', 'unknown')}")

            # Save progress after each page
            self.progress.set_page(page_num)
            self.progress.save()

            # Rate limiting - be nice to the server
            time.sleep(0.5)

        # Final summary
        self.save_summary(total)
        print(f"\n{'='*60}")
        print(f"  DOWNLOAD COMPLETE!")
        print(f"{'='*60}")
        print(f"  Total Downloaded: {self.progress.data['total_downloaded']:,}")
        print(f"  Total Expanded: {self.progress.data['total_expanded']:,}")
        print(f"  Failed: {len(self.progress.data['failed'])}")
        print(f"  Output Directory: {self.output_dir}")
        print(f"{'='*60}\n")

    def save_summary(self, total: int):
        """Save download summary"""
        summary = {
            "download_date": datetime.now().isoformat(),
            "source": ONTOSERVER_BASE_URL,
            "total_available": total,
            "total_downloaded": self.progress.data["total_downloaded"],
            "total_expanded": self.progress.data["total_expanded"],
            "failed_count": len(self.progress.data["failed"]),
            "failed_ids": [f["id"] for f in self.progress.data["failed"]]
        }
        with open(SUMMARY_FILE, 'w') as f:
            json.dump(summary, f, indent=2)

# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Download ValueSets from CSIRO Ontoserver"
    )
    parser.add_argument(
        "--filter", "-f",
        help="Filter ValueSets by name (e.g., 'sepsis', 'diabetes')"
    )
    parser.add_argument(
        "--expand", "-e",
        action="store_true",
        help="Also download expanded codes for each ValueSet"
    )
    parser.add_argument(
        "--resume", "-r",
        action="store_true",
        help="Resume from last download position"
    )
    parser.add_argument(
        "--output", "-o",
        type=Path,
        default=OUTPUT_DIR,
        help=f"Output directory (default: {OUTPUT_DIR})"
    )

    args = parser.parse_args()

    # Create downloader and run
    downloader = ValueSetDownloader(
        output_dir=args.output,
        expand=args.expand
    )

    try:
        downloader.download_all(
            name_filter=args.filter,
            resume=args.resume
        )
    except KeyboardInterrupt:
        print("\n\nDownload interrupted. Use --resume to continue later.")
        downloader.progress.save()
        sys.exit(1)
    except Exception as e:
        print(f"\nERROR: {e}")
        downloader.progress.save()
        sys.exit(1)

if __name__ == "__main__":
    main()
