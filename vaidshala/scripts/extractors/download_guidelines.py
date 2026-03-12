#!/usr/bin/env python3
"""
Vaidshala Phase 4: Clinical Guideline PDF Downloader

Downloads clinical practice guideline PDFs for table extraction.
Most guidelines are open access - some may require journal subscription.

Usage:
    python download_guidelines.py --list              # Show all available guidelines
    python download_guidelines.py --download SSC-2021 # Download specific guideline
    python download_guidelines.py --download-all      # Download all open access
    python download_guidelines.py --status            # Check download status

Requirements:
    pip install requests
"""

import json
import os
import sys
import argparse
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional

# Optional: for actual downloading
try:
    import requests
    HAS_REQUESTS = True
except ImportError:
    HAS_REQUESTS = False


class GuidelineDownloader:
    """
    Download clinical practice guideline PDFs.

    Note: Many clinical guidelines are published in journals that may
    require institutional access. This tool handles open-access PDFs
    and provides manual download instructions for others.
    """

    def __init__(self, base_dir: str = None):
        self.base_dir = Path(base_dir) if base_dir else Path(__file__).parent
        self.pdf_dir = self.base_dir / "pdfs"
        self.manifest_path = self.base_dir / "pdf_sources.json"

        # Create PDF directory
        self.pdf_dir.mkdir(exist_ok=True)

        # Load manifest
        self.manifest = self._load_manifest()

    def _load_manifest(self) -> Dict:
        """Load PDF sources manifest."""
        if self.manifest_path.exists():
            with open(self.manifest_path) as f:
                return json.load(f)
        return {"priority_guidelines": [], "placeholder_guidelines": {}}

    def _save_manifest(self):
        """Save updated manifest."""
        with open(self.manifest_path, 'w') as f:
            json.dump(self.manifest, f, indent=2)

    def list_guidelines(self) -> None:
        """Display all available guidelines."""
        print("\n" + "=" * 70)
        print("PRIORITY GUIDELINES (5) - For Phase 4 Extraction")
        print("=" * 70)

        for g in self.manifest.get("priority_guidelines", []):
            status = self._get_status(g["id"])
            status_icon = "✅" if status == "DOWNLOADED" else "⏳"
            print(f"\n{status_icon} [{g['id']}]")
            print(f"   Title: {g['title'][:60]}...")
            print(f"   Org: {g['organization']} ({g['year']})")
            print(f"   DOI: {g['doi']}")
            print(f"   Open Access: {'Yes' if g.get('open_access') else 'Subscription Required'}")
            print(f"   Pages: {g.get('pages', 'Unknown')}")
            print(f"   Existing CQL: {g.get('existing_cql', 'None')}")

        print("\n" + "=" * 70)
        print("PLACEHOLDER GUIDELINES - For Future Import")
        print("=" * 70)

        for region, guidelines in self.manifest.get("placeholder_guidelines", {}).items():
            print(f"\n📁 {region.upper()}/")
            for g in guidelines:
                print(f"   - [{g['id']}] {g['title'][:50]}...")

    def _get_status(self, guideline_id: str) -> str:
        """Check if PDF already downloaded."""
        pdf_patterns = [
            f"{guideline_id}.pdf",
            f"{guideline_id.lower()}.pdf",
            f"{guideline_id.replace('-', '_')}.pdf"
        ]

        for pattern in pdf_patterns:
            if (self.pdf_dir / pattern).exists():
                return "DOWNLOADED"

        return "PENDING"

    def download(self, guideline_id: str) -> bool:
        """
        Download a specific guideline PDF.

        Note: Direct PDF downloads often don't work due to journal paywalls.
        This provides instructions for manual download when needed.
        """
        guideline = None
        for g in self.manifest.get("priority_guidelines", []):
            if g["id"] == guideline_id:
                guideline = g
                break

        if not guideline:
            print(f"❌ Unknown guideline: {guideline_id}")
            print("   Use --list to see available guidelines")
            return False

        print(f"\n📥 DOWNLOAD INSTRUCTIONS: {guideline['id']}")
        print("=" * 60)
        print(f"Title: {guideline['title']}")
        print(f"DOI: https://doi.org/{guideline['doi']}")
        print(f"Direct URL: {guideline['download_url']}")
        print()

        if guideline.get("open_access"):
            print("✅ This guideline is OPEN ACCESS")
            print()
            print("MANUAL DOWNLOAD STEPS:")
            print(f"1. Open: {guideline['download_url']}")
            print("2. Click 'Download PDF' or 'Full Text PDF'")
            print(f"3. Save as: {self.pdf_dir}/{guideline_id}.pdf")
            print()

            # Try automated download for truly open access
            if HAS_REQUESTS and self._try_automated_download(guideline):
                return True

        else:
            print("⚠️  This guideline may require SUBSCRIPTION ACCESS")
            print()
            print("OPTIONS:")
            print("1. Access via institutional login (university/hospital)")
            print("2. Use interlibrary loan service")
            print(f"3. Check PubMed Central: https://www.ncbi.nlm.nih.gov/pmc/")

        print()
        print(f"After downloading, save to: {self.pdf_dir}/{guideline_id}.pdf")

        return False

    def _try_automated_download(self, guideline: Dict) -> bool:
        """Attempt automated PDF download (works for some open access)."""
        if not HAS_REQUESTS:
            print("Note: Install 'requests' for automated download attempts")
            return False

        # Known patterns for direct PDF links
        pdf_patterns = {
            "springer": lambda url: url.replace("/article/", "/content/pdf/") + ".pdf",
            "ahajournals": lambda url: url.replace("/doi/", "/doi/pdf/"),
            "chestnet": lambda url: url.replace("/fulltext", "/pdf"),
        }

        url = guideline["download_url"]

        # Try to construct PDF URL
        for site, pattern_fn in pdf_patterns.items():
            if site in url:
                try:
                    pdf_url = pattern_fn(url)
                    print(f"Attempting download from: {pdf_url[:60]}...")

                    response = requests.get(
                        pdf_url,
                        headers={"User-Agent": "Vaidshala-CQL-Importer/1.0"},
                        timeout=30,
                        stream=True
                    )

                    if response.status_code == 200 and "pdf" in response.headers.get("content-type", ""):
                        pdf_path = self.pdf_dir / f"{guideline['id']}.pdf"
                        with open(pdf_path, 'wb') as f:
                            for chunk in response.iter_content(chunk_size=8192):
                                f.write(chunk)

                        print(f"✅ Downloaded successfully: {pdf_path}")

                        # Update manifest
                        for g in self.manifest["priority_guidelines"]:
                            if g["id"] == guideline["id"]:
                                g["status"] = "DOWNLOADED"
                                g["downloaded_at"] = datetime.utcnow().isoformat()
                        self._save_manifest()

                        return True

                except Exception as e:
                    print(f"Download attempt failed: {e}")

        return False

    def check_status(self) -> None:
        """Show download status for all priority guidelines."""
        print("\n" + "=" * 60)
        print("PDF DOWNLOAD STATUS")
        print("=" * 60)

        downloaded = 0
        pending = 0

        for g in self.manifest.get("priority_guidelines", []):
            status = self._get_status(g["id"])

            if status == "DOWNLOADED":
                print(f"✅ {g['id']}: Downloaded")
                downloaded += 1
            else:
                print(f"⏳ {g['id']}: Pending")
                pending += 1

        print()
        print(f"Downloaded: {downloaded}/5")
        print(f"Pending: {pending}/5")
        print()

        if pending > 0:
            print("To download pending PDFs:")
            print("  python download_guidelines.py --download <GUIDELINE-ID>")

    def verify_pdfs(self) -> Dict[str, bool]:
        """Verify all downloaded PDFs are valid."""
        results = {}

        for g in self.manifest.get("priority_guidelines", []):
            pdf_path = self.pdf_dir / f"{g['id']}.pdf"

            if pdf_path.exists():
                # Basic validation: check file size and PDF header
                size = pdf_path.stat().st_size
                with open(pdf_path, 'rb') as f:
                    header = f.read(8)

                is_valid = size > 10000 and header.startswith(b'%PDF')
                results[g['id']] = is_valid

                if is_valid:
                    print(f"✅ {g['id']}: Valid PDF ({size // 1024} KB)")
                else:
                    print(f"❌ {g['id']}: Invalid or corrupted")
            else:
                results[g['id']] = False
                print(f"⏳ {g['id']}: Not downloaded")

        return results


def main():
    parser = argparse.ArgumentParser(
        description="Download clinical guideline PDFs for Phase 4 extraction"
    )
    parser.add_argument(
        "--list", action="store_true",
        help="List all available guidelines"
    )
    parser.add_argument(
        "--download", type=str, metavar="ID",
        help="Download specific guideline by ID"
    )
    parser.add_argument(
        "--download-all", action="store_true",
        help="Attempt to download all open-access guidelines"
    )
    parser.add_argument(
        "--status", action="store_true",
        help="Check download status"
    )
    parser.add_argument(
        "--verify", action="store_true",
        help="Verify downloaded PDFs"
    )

    args = parser.parse_args()

    downloader = GuidelineDownloader()

    if args.list:
        downloader.list_guidelines()
    elif args.download:
        downloader.download(args.download)
    elif args.download_all:
        for g in downloader.manifest.get("priority_guidelines", []):
            if g.get("open_access"):
                downloader.download(g["id"])
                print()
    elif args.status:
        downloader.check_status()
    elif args.verify:
        downloader.verify_pdfs()
    else:
        parser.print_help()
        print("\nExample:")
        print("  python download_guidelines.py --list")
        print("  python download_guidelines.py --download SSC-2021")


if __name__ == "__main__":
    main()
