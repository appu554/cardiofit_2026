"""
Local NCTS package downloader (one-shot, dev use).

Mirrors the production GCP downloader at
gcp/functions/snomed-au-downloader/main.py but writes to the local
filesystem instead of GCS so we can parse downloaded packages locally
into KB-7 Postgres without going through GCP infra.

Reads credentials from environment:
    NCTS_CLIENT_ID
    NCTS_CLIENT_SECRET

Usage:
    # Default — SNOMED CT-AU RF2 SNAPSHOT (concepts/descriptions/relationships)
    python3 scripts/download_snomed_au_local.py

    # AMT (Australian Medicines Terminology) as TSV
    python3 scripts/download_snomed_au_local.py --package-type AMT_TSV

    # List all categories in the syndication feed
    python3 scripts/download_snomed_au_local.py --list

Supported categories (from NCTS syndication feed):
    SCT_RF2_SNAPSHOT  -> data/snomed/
    SCT_RF2_FULL      -> data/snomed/
    AMT_TSV           -> data/amt/
    AMT_CSV           -> data/amt/
"""

import argparse
import hashlib
import logging
import os
import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

import requests

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
log = logging.getLogger(__name__)

NTS_OAUTH_TOKEN_URL = "https://api.healthterminologies.gov.au/oauth2/token"
NTS_SYNDICATION_URL = "https://api.healthterminologies.gov.au/syndication/v1/syndication.xml"

# Map package-type CLI value -> output subdirectory under kb-7-terminology/
DEFAULT_OUTPUT_SUBDIR = {
    "SCT_RF2_SNAPSHOT": "data/snomed",
    "SCT_RF2_FULL":     "data/snomed",
    "AMT_TSV":          "data/amt",
    "AMT_CSV":          "data/amt",
}

REPO_KB7_DIR = Path(__file__).resolve().parent.parent

NS = {
    "atom": "http://www.w3.org/2005/Atom",
    "ncts": "http://ns.electronichealth.net.au/ncts/syndication/asf/extensions/1.0.0",
}


def get_oauth_token(client_id: str, client_secret: str) -> str:
    log.info("Requesting OAuth2 token from NCTS...")
    r = requests.post(
        NTS_OAUTH_TOKEN_URL,
        data={
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
        },
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "User-Agent": "KB7-Local-Downloader/1.0",
        },
        timeout=30,
    )
    r.raise_for_status()
    token = r.json().get("access_token")
    if not token:
        raise RuntimeError("OAuth2 response missing access_token")
    log.info("OAuth2 token obtained")
    return token


def fetch_syndication(token: str) -> bytes:
    log.info("Fetching syndication feed...")
    r = requests.get(
        NTS_SYNDICATION_URL,
        headers={
            "Accept": "application/atom+xml",
            "Authorization": f"Bearer {token}",
            "User-Agent": "KB7-Local-Downloader/1.0",
        },
        timeout=60,
    )
    r.raise_for_status()
    return r.content


def find_latest_for_category(xml_content: bytes, category: str) -> dict:
    root = ET.fromstring(xml_content)
    candidates = []
    for entry in root.findall(".//atom:entry", NS):
        cat = entry.find("atom:category", NS)
        cat_term = cat.get("term") if cat is not None else ""
        if cat_term != category:
            continue

        link = entry.find('atom:link[@rel="alternate"]', NS) or entry.find(
            "atom:link", NS
        )
        if link is None or not link.get("href"):
            continue

        url = link.get("href")
        size = int(link.get("length", "0") or "0")
        sha256 = link.get(
            f"{{{NS['ncts']}}}sha256Hash", ""
        )
        title_elem = entry.find("atom:title", NS)
        title = title_elem.text if title_elem is not None else ""

        version = ""
        m = re.search(r"/(\d{8})/", url)
        if m:
            version = m.group(1)

        candidates.append(
            {
                "title": title,
                "url": url,
                "size": size,
                "sha256": sha256,
                "version": version,
            }
        )

    if not candidates:
        raise RuntimeError(f"No {category} package found in syndication feed")

    candidates.sort(key=lambda p: p["version"], reverse=True)
    chosen = candidates[0]
    log.info(
        "Latest %s: %s v%s (%.1f MB)",
        category,
        chosen["title"],
        chosen["version"],
        chosen["size"] / (1024 * 1024),
    )
    return chosen


def list_all_categories(xml_content: bytes) -> None:
    root = ET.fromstring(xml_content)
    counts: dict[str, int] = {}
    latest_per_cat: dict[str, tuple[str, int]] = {}
    for entry in root.findall(".//atom:entry", NS):
        cat = entry.find("atom:category", NS)
        cat_term = cat.get("term") if cat is not None else ""
        if not cat_term:
            continue
        title_el = entry.find("atom:title", NS)
        title = (title_el.text or "") if title_el is not None else ""
        link = entry.find('atom:link[@rel="alternate"]', NS) or entry.find(
            "atom:link", NS
        )
        size = int(link.get("length", "0") or "0") if link is not None else 0
        counts[cat_term] = counts.get(cat_term, 0) + 1
        if cat_term not in latest_per_cat or title > latest_per_cat[cat_term][0]:
            latest_per_cat[cat_term] = (title, size)

    print(f"\nFound {len(counts)} unique categories:")
    print(f"{'CATEGORY':<32}  {'COUNT':>6}  {'LATEST TITLE':<60}  {'SIZE':>10}")
    for cat in sorted(counts):
        title, size = latest_per_cat[cat]
        print(
            f"{cat:<32}  {counts[cat]:>6}  {title[:60]:<60}  "
            f"{size/(1024*1024):>8.1f}MB"
        )


def download_to_disk(token: str, package: dict, output_dir: Path) -> Path:
    output_dir.mkdir(parents=True, exist_ok=True)

    filename = package["url"].rsplit("/", 1)[-1]
    if "?" in filename:
        filename = filename.split("?", 1)[0]
    out_path = output_dir / filename

    if out_path.exists() and package["sha256"]:
        existing = hashlib.sha256(out_path.read_bytes()).hexdigest()
        if existing.lower() == package["sha256"].lower():
            log.info("File already on disk with matching SHA256: %s", out_path)
            return out_path
        log.warning("Existing file SHA mismatch — redownloading")

    log.info("Downloading -> %s", out_path)
    hasher = hashlib.sha256()
    bytes_done = 0
    chunk = 10 * 1024 * 1024  # 10 MB

    with requests.get(
        package["url"],
        stream=True,
        headers={
            "Authorization": f"Bearer {token}",
            "User-Agent": "KB7-Local-Downloader/1.0",
        },
        timeout=3600,
    ) as r:
        r.raise_for_status()
        content_length = int(r.headers.get("content-length", package["size"] or 0))
        with out_path.open("wb") as f:
            for piece in r.iter_content(chunk_size=chunk):
                if not piece:
                    continue
                f.write(piece)
                hasher.update(piece)
                bytes_done += len(piece)
                if bytes_done % (50 * 1024 * 1024) < chunk:
                    pct = (
                        (bytes_done / content_length * 100) if content_length else 0
                    )
                    log.info("  %.1f MB (%.1f%%)", bytes_done / (1024 * 1024), pct)

    actual_sha = hasher.hexdigest()
    if package["sha256"] and actual_sha.lower() != package["sha256"].lower():
        out_path.unlink(missing_ok=True)
        raise RuntimeError(
            f"SHA256 mismatch. Expected {package['sha256']}, got {actual_sha}"
        )
    log.info("SHA256 verified: %s", actual_sha[:16] + "...")
    log.info("Done: %.1f MB written", bytes_done / (1024 * 1024))
    return out_path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--package-type",
        default="SCT_RF2_SNAPSHOT",
        help="NCTS syndication category (default: SCT_RF2_SNAPSHOT)",
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="List all categories in the syndication feed and exit",
    )
    args = parser.parse_args()

    client_id = os.environ.get("NCTS_CLIENT_ID")
    client_secret = os.environ.get("NCTS_CLIENT_SECRET")
    if not client_id or not client_secret:
        log.error(
            "Missing NCTS_CLIENT_ID or NCTS_CLIENT_SECRET env vars. "
            "Source .env.ncts.local before running."
        )
        return 2

    token = get_oauth_token(client_id, client_secret)
    feed = fetch_syndication(token)

    if args.list:
        list_all_categories(feed)
        return 0

    package = find_latest_for_category(feed, args.package_type)
    output_subdir = DEFAULT_OUTPUT_SUBDIR.get(args.package_type, "data/other")
    output_dir = REPO_KB7_DIR / output_subdir
    out_path = download_to_disk(token, package, output_dir)
    print(str(out_path))
    return 0


if __name__ == "__main__":
    sys.exit(main())
