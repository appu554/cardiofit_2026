"""
IHACPA ICD-10-AM / ACHI downloader (one-shot, dev use).

Status: SKELETON. The real IHACPA distribution API is not publicly
documented — they primarily distribute via authenticated portal download
(login -> click -> ZIP). This script implements the most likely API
shape based on the existing Go IHACPAConfig struct in
internal/regional/icd10am/loader.go:

    BaseURL        e.g., https://api.ihacpa.gov.au or portal URL
    InstitutionID  institutional account identifier
    AccessKey      API key / bearer token
    CertificatePath  optional client cert for mTLS
    PrivateKeyPath   optional private key for mTLS

Two auth patterns are attempted in order:
  1. Bearer token in Authorization header
  2. X-Institution-ID + X-Access-Key headers

If neither works, the most likely production path is:
  (a) Log in to https://www.ihacpa.gov.au manually,
  (b) Download the ICD-10-AM/ACHI ZIP files,
  (c) Extract into data/icd10am/<edition>/,
  (d) Run scripts/load_icd10am.py with --tabular / --index paths.

That manual flow works *today* without any IHACPA API integration,
which is why the loader (load_icd10am.py) is parametric on file paths.

Reads credentials from environment:
    IHACPA_BASE_URL
    IHACPA_INSTITUTION_ID
    IHACPA_ACCESS_KEY
    IHACPA_CERT_PATH      (optional, for mTLS)
    IHACPA_KEY_PATH       (optional, for mTLS)

Output:
    data/icd10am/<edition>/icd10am_tabular.xml
    data/icd10am/<edition>/icd10am_index.csv
    data/icd10am/<edition>/achi_tabular.xml
    data/icd10am/<edition>/achi_index.csv

Usage:
    python3 scripts/download_icd10am_local.py --edition 12th
    python3 scripts/download_icd10am_local.py --probe  # print auth diag without downloading
"""

from __future__ import annotations

import argparse
import logging
import os
import sys
from pathlib import Path

import requests

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
log = logging.getLogger(__name__)

REPO_KB7 = Path(__file__).resolve().parent.parent

# Standard package names IHACPA distributes as part of an edition.
# The actual download URL pattern depends on the IHACPA API spec —
# fill in EXPECTED_URL_TEMPLATES once their docs are obtained.
EXPECTED_PACKAGES = [
    "icd10am_tabular.xml",
    "icd10am_index.csv",
    "achi_tabular.xml",
    "achi_index.csv",
]


def auth_session(institution_id: str, access_key: str,
                 cert_path: str | None, key_path: str | None) -> requests.Session:
    """Build a requests.Session with IHACPA-style auth.

    Tries Bearer first; the caller can attach X-Institution-ID /
    X-Access-Key headers per request if Bearer doesn't work.
    """
    s = requests.Session()
    s.headers.update({
        "User-Agent": "KB7-IHACPA-Downloader/1.0",
        "Accept": "application/xml,application/octet-stream,*/*",
        "Authorization": f"Bearer {access_key}",
        "X-Institution-ID": institution_id,
        "X-Access-Key": access_key,
    })
    if cert_path and key_path:
        s.cert = (cert_path, key_path)
    return s


def probe_auth(session: requests.Session, base_url: str) -> None:
    """Print a diagnostic auth check against base_url. Read-only."""
    log.info("Probing IHACPA endpoint: %s", base_url)
    try:
        r = session.get(base_url, timeout=15)
        log.info("  status=%d  content-type=%s  bytes=%d",
                 r.status_code, r.headers.get("content-type", ""), len(r.content))
        if 200 <= r.status_code < 300:
            log.info("  auth APPEARS to work; ready to download specific packages")
        elif r.status_code in (401, 403):
            log.warning(
                "  auth REJECTED. Check IHACPA_INSTITUTION_ID and IHACPA_ACCESS_KEY. "
                "If correct, the actual auth header shape may differ from "
                "Bearer/X-API-Key — consult IHACPA API documentation."
            )
        else:
            log.warning("  unexpected status; not necessarily an auth failure")
    except requests.RequestException as e:
        log.error("  request failed: %s", e)


def download_package(session: requests.Session, base_url: str, package_name: str,
                     edition: str, out_dir: Path) -> Path:
    """Download one IHACPA package. URL pattern is a best guess —
    adapt once IHACPA documentation is obtained.
    """
    out_path = out_dir / package_name
    url = f"{base_url.rstrip('/')}/distribution/{edition}/{package_name}"
    log.info("Fetching %s -> %s", url, out_path)
    with session.get(url, stream=True, timeout=600) as r:
        if r.status_code == 404:
            raise RuntimeError(
                f"IHACPA returned 404 for {url}. "
                f"URL pattern is speculative; correct it in this script "
                f"once the IHACPA API spec is confirmed. "
                f"As a workaround, download {package_name} manually from the "
                f"IHACPA portal and place at {out_path}."
            )
        r.raise_for_status()
        out_path.parent.mkdir(parents=True, exist_ok=True)
        bytes_done = 0
        with out_path.open("wb") as f:
            for piece in r.iter_content(chunk_size=10 * 1024 * 1024):
                if piece:
                    f.write(piece)
                    bytes_done += len(piece)
        log.info("  done: %.1f MB written", bytes_done / (1024 * 1024))
    return out_path


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--edition", default="12th",
                   help='Edition label, e.g., "12th"')
    p.add_argument("--probe", action="store_true",
                   help="Probe auth only, do not download")
    args = p.parse_args()

    base_url = os.environ.get("IHACPA_BASE_URL")
    institution_id = os.environ.get("IHACPA_INSTITUTION_ID")
    access_key = os.environ.get("IHACPA_ACCESS_KEY")
    cert_path = os.environ.get("IHACPA_CERT_PATH")
    key_path = os.environ.get("IHACPA_KEY_PATH")

    if not (base_url and institution_id and access_key):
        log.error(
            "Missing IHACPA credentials. Set the following env vars "
            "(via .env.ihacpa.local + `set -a; source ...; set +a`):"
        )
        log.error("  IHACPA_BASE_URL")
        log.error("  IHACPA_INSTITUTION_ID")
        log.error("  IHACPA_ACCESS_KEY")
        log.error(
            "Alternative: download files from https://www.ihacpa.gov.au "
            "manually, place them under data/icd10am/<edition>/, and run "
            "scripts/load_icd10am.py with --tabular / --index paths."
        )
        return 2

    session = auth_session(institution_id, access_key, cert_path, key_path)

    if args.probe:
        probe_auth(session, base_url)
        return 0

    out_dir = REPO_KB7 / "data" / "icd10am" / args.edition
    out_dir.mkdir(parents=True, exist_ok=True)

    failures = []
    for pkg in EXPECTED_PACKAGES:
        try:
            download_package(session, base_url, pkg, args.edition, out_dir)
        except Exception as e:
            log.error("  failed: %s — %s", pkg, e)
            failures.append(pkg)

    if failures:
        log.error("Failed packages: %s", failures)
        log.error(
            "If failures are HTTP 404, the URL template in this script needs "
            "adjustment to match IHACPA's actual API. As a workaround, download "
            "the failed packages manually from the IHACPA portal and run "
            "scripts/load_icd10am.py."
        )
        return 1

    log.info("All packages downloaded -> %s", out_dir)
    return 0


if __name__ == "__main__":
    sys.exit(main())
