"""
TGA Product Information (PI) scraper — Wave 2 / Tier 3 item #12-14.

Downloads Australian Product Information PDFs from the TGA eBS PICMI
repository. Per v1.0 Layer 1 spec, this is the most engineering-heavy
piece of Layer 1 because TGA has no clean API and serves PDFs behind
a JS license-acceptance gate.

How TGA eBS PICMI works (reverse-engineered 2026-04-30):

  1. Browse:  GET .../picmirepository.nsf/PICMI?OpenForm&Seq=1&t=PI&q=<letter>
              returns HTML with <table> rows containing trade name +
              `<a href="pdf?OpenAgent&id=CP-YYYY-PI-NNNNN-V">PI</a>`
              + active ingredient(s).

  2. PDF gate: GET .../pdf?OpenAgent&id=<DOC_ID>
              returns disclaimer HTML with hidden field
              `<input name="Remote_Addr" value="X.X.X.X" id="remoteaddr">`
              + JS function IAccept() that:
                - computes cookie value: <UTC YYYYMMDD><RemoteAddr without dots>
                - sets cookie PICMIIAccept=<value>
                - reloads same URL with &d=<value> appended

  3. Download: GET same URL with &d=<value> + Cookie: PICMIIAccept=<value>
              returns the actual PDF (typically 30-150 pages, 50-500 KB).

Two-stage usage:
    # Stage A — discover what's available (full PI catalog ~7,000 docs)
    python3 scripts/tga_pi_scraper.py discover

    # Stage B — download specific drugs (use a watchlist YAML)
    python3 scripts/tga_pi_scraper.py download --watchlist data/top_racf_drugs.yaml
"""

from __future__ import annotations

import argparse
import json
import logging
import re
import sys
import time
from datetime import datetime, timezone
from html import unescape
from pathlib import Path
from urllib.error import URLError, HTTPError
from urllib.request import Request, urlopen

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)-7s %(message)s")
log = logging.getLogger(__name__)

KB3 = Path(__file__).resolve().parent.parent
TGA_DIR = KB3 / "knowledge" / "au" / "tga_pi"
DOCS_DIR = TGA_DIR / "docs"
CACHE_DIR = TGA_DIR / "cache"
INDEX_JSON = TGA_DIR / "tga_pi_index.json"
CMI_INDEX_JSON = TGA_DIR / "tga_cmi_index.json"
CMI_DOCS_DIR = TGA_DIR / "cmi_docs"


def index_path_for(doc_type: str) -> Path:
    return CMI_INDEX_JSON if doc_type == "CMI" else INDEX_JSON


def docs_dir_for(doc_type: str) -> Path:
    return CMI_DOCS_DIR if doc_type == "CMI" else DOCS_DIR

PICMI_BASE = "https://www.ebs.tga.gov.au/ebs/picmi/picmirepository.nsf"
USER_AGENT = "Mozilla/5.0 (research; Layer1-AU-Aged-Care)"

# eBS Lotus Notes uses category letters 0-9, A-Z for browsing
CATEGORY_LETTERS = list("0123456789") + list("ABCDEFGHIJKLMNOPQRSTUVWXYZ")


def fetch(url: str, cookie: str | None = None, timeout: int = 30) -> bytes:
    headers = {"User-Agent": USER_AGENT}
    if cookie:
        headers["Cookie"] = cookie
    req = Request(url, headers=headers)
    try:
        with urlopen(req, timeout=timeout) as resp:
            return resp.read()
    except (URLError, HTTPError, TimeoutError) as e:
        log.error("fetch failed: %s — %s", url, e)
        return b""


# ─── Stage A: Discovery ─────────────────────────────────────────────

# Both PI and CMI rows match this shape; the label inside <a>...</a> distinguishes.
ROW_RE_TEMPLATE = (
    r"<tr>\s*<td>([^<]*)</td>\s*"            # trade name
    r"<td>\s*<a[^>]*href='pdf\?OpenAgent&id=([^']+)'[^>]*>{label}</a>\s*</td>\s*"
    r"<td>([^<]*)</td>\s*</tr>"
)
PI_ROW_RE = re.compile(ROW_RE_TEMPLATE.format(label="PI"), re.IGNORECASE)
CMI_ROW_RE = re.compile(ROW_RE_TEMPLATE.format(label="CMI"), re.IGNORECASE)


def discover_letter(letter: str, doc_type: str = "PI") -> list[dict]:
    """Fetch the listing for one trade-name category letter.

    URL pattern (from displayCategory() JS):
        PICMI?OpenForm&t=<TYPE>&k=<LETTER>&r=/
    where <TYPE> is 'PI' (Product Information) or 'CMI' (Consumer Medicine
    Information). Returns ALL documents for that letter (no pagination
    needed within a letter).
    """
    url = f"{PICMI_BASE}/PICMI?OpenForm&t={doc_type}&k={letter}&r=/"
    body = fetch(url)
    if not body:
        return []
    text = body.decode("utf-8", errors="replace")
    row_re = CMI_ROW_RE if doc_type == "CMI" else PI_ROW_RE
    id_field = "cmi_id" if doc_type == "CMI" else "pi_id"
    rows = []
    for m in row_re.finditer(text):
        trade_name = unescape(m.group(1)).strip()
        doc_id = unescape(m.group(2)).strip()
        ingredients = unescape(m.group(3)).strip()
        rows.append({
            "trade_name": trade_name,
            id_field: doc_id,
            "doc_type": doc_type,
            "active_ingredients": [a.strip() for a in re.split(r";", ingredients) if a.strip()],
            "category_letter": letter,
        })
    # Capture the doc-count header for sanity
    m = re.search(r"(\d+)\s+Documents available", text)
    docs_available = int(m.group(1)) if m else None
    log.info("  k=%s: %d rows parsed (header says %s available)",
             letter, len(rows), docs_available)
    return rows


def discover_all(letters: list[str] | None = None, sleep_s: float = 0.4,
                 doc_type: str = "PI") -> dict:
    """Crawl every category letter (0-9, A-Z), build full PI or CMI index."""
    letters = letters or CATEGORY_LETTERS
    id_field = "cmi_id" if doc_type == "CMI" else "pi_id"
    all_rows: list[dict] = []
    for letter in letters:
        rows = discover_letter(letter, doc_type=doc_type)
        all_rows.extend(rows)
        time.sleep(sleep_s)

    # Dedupe by document ID (different letters may surface same doc)
    by_id: dict[str, dict] = {}
    for r in all_rows:
        doc_id = r[id_field]
        if doc_id not in by_id:
            by_id[doc_id] = r
        else:
            existing = by_id[doc_id]
            if r["trade_name"] != existing["trade_name"] and \
               r["trade_name"] not in existing.get("alt_trade_names", []):
                existing.setdefault("alt_trade_names", []).append(r["trade_name"])

    count_key = f"total_unique_{doc_type.lower()}s"
    index = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "doc_type": doc_type,
        count_key: len(by_id),
        "total_rows_seen": len(all_rows),
        "letters_crawled": letters,
        "documents": list(by_id.values()),
    }
    return index


# ─── Stage B: Download a single PI ──────────────────────────────────

REMOTE_ADDR_RE = re.compile(
    r'<input[^>]*name="Remote_Addr"[^>]*value="([^"]+)"', re.IGNORECASE,
)


def get_disclaimer_remote_ip(pi_id: str) -> str | None:
    """First-stage GET: returns the disclaimer HTML, extract Remote_Addr field."""
    url = f"{PICMI_BASE}/pdf?OpenAgent&id={pi_id}"
    body = fetch(url).decode("utf-8", errors="replace")
    m = REMOTE_ADDR_RE.search(body)
    return m.group(1) if m else None


def build_disclaimer_cookie(remote_ip: str) -> str:
    """Reproduce the JS IAccept cookie value computation."""
    utc_date = datetime.now(timezone.utc).strftime("%Y%m%d")
    ip_compact = remote_ip.replace(".", "")
    return f"{utc_date}{ip_compact}"


def download_pi_pdf(pi_id: str, dest: Path, force: bool = False) -> bool:
    """Two-step download: get disclaimer cookie, then fetch the actual PDF."""
    if dest.exists() and not force:
        log.info("  %s already at %s (skipping; use --force to redownload)",
                 pi_id, dest.name)
        return True

    remote_ip = get_disclaimer_remote_ip(pi_id)
    if not remote_ip:
        log.error("  %s: could not extract Remote_Addr from disclaimer", pi_id)
        return False

    cookie_val = build_disclaimer_cookie(remote_ip)
    url = f"{PICMI_BASE}/pdf?OpenAgent&id={pi_id}&d={cookie_val}"
    cookie_header = f"PICMIIAccept={cookie_val}"

    body = fetch(url, cookie=cookie_header, timeout=60)
    if not body:
        log.error("  %s: download returned empty body", pi_id)
        return False
    if not body.startswith(b"%PDF"):
        log.error("  %s: response is not a PDF (got %d bytes starting %r...)",
                  pi_id, len(body), body[:8])
        return False

    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_bytes(body)
    log.info("  %s -> %s (%.1f KB)", pi_id, dest.name, len(body) / 1024)
    return True


# ─── CLI ────────────────────────────────────────────────────────────

def cmd_discover(args) -> int:
    letters = list(args.letters) if args.letters else None
    doc_type = args.doc_type
    index = discover_all(letters=letters, sleep_s=args.sleep, doc_type=doc_type)
    target = index_path_for(doc_type)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(index, indent=2))
    count_key = f"total_unique_{doc_type.lower()}s"
    log.info("=" * 70)
    log.info("DISCOVERY COMPLETE (%s)", doc_type)
    log.info("  Unique %s documents: %d", doc_type, index.get(count_key, 0))
    log.info("  Total rows seen:     %d", index["total_rows_seen"])
    log.info("  Index: %s", target)
    return 0


def cmd_download(args) -> int:
    doc_type = args.doc_type
    id_field = "cmi_id" if doc_type == "CMI" else "pi_id"
    index_file = index_path_for(doc_type)
    docs_dir = docs_dir_for(doc_type)

    if not index_file.exists():
        log.error("Index missing for %s — run `discover --doc-type %s` first",
                  doc_type, doc_type)
        return 1
    index = json.loads(index_file.read_text())

    explicit_id = args.cmi_id if doc_type == "CMI" else args.pi_id
    if explicit_id:
        targets = [d for d in index["documents"] if d.get(id_field) == explicit_id]
        if not targets:
            log.error("%s ID not found in index: %s", doc_type, explicit_id)
            return 1
    elif args.watchlist:
        wl_path = Path(args.watchlist)
        if not wl_path.exists():
            log.error("Watchlist not found: %s", wl_path)
            return 1
        wanted = set()
        for line in wl_path.read_text().splitlines():
            line = line.split("#", 1)[0].strip().lower()
            if line:
                wanted.add(line)
        log.info("Watchlist: %d INN names", len(wanted))
        targets = []
        for d in index["documents"]:
            ingredients_lower = " ".join(d["active_ingredients"]).lower()
            for w in wanted:
                if w in ingredients_lower:
                    targets.append(d)
                    break
        log.info("Matched: %d %s documents from %d watchlist INNs",
                 len(targets), doc_type, len(wanted))
    else:
        log.error("Provide --pi-id/--cmi-id <ID> or --watchlist <path>")
        return 1

    ok = 0
    fail = 0
    for d in targets:
        doc_id = d[id_field]
        safe_name = re.sub(r"[^\w\-]+", "_", d["trade_name"])[:60]
        dest = docs_dir / f"{doc_id}__{safe_name}.pdf"
        if download_pi_pdf(doc_id, dest, force=args.force):
            ok += 1
        else:
            fail += 1
        time.sleep(args.sleep)

    log.info("=" * 70)
    log.info("DOWNLOAD COMPLETE (%s): %d ok, %d failed", doc_type, ok, fail)
    log.info("  Storage: %s", docs_dir)
    return 0 if fail == 0 else 2


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)

    p_disc = sub.add_parser("discover", help="Crawl all category letters, build index")
    p_disc.add_argument("--doc-type", choices=["PI", "CMI"], default="PI",
                        help="PI=Product Information (clinician-facing), "
                             "CMI=Consumer Medicine Information (patient-facing)")
    p_disc.add_argument("--letters", nargs="*",
                        help="Subset of category letters (default: 0-9 + A-Z)")
    p_disc.add_argument("--sleep", type=float, default=0.4,
                        help="Seconds between requests (default 0.4)")
    p_disc.set_defaults(func=cmd_discover)

    p_dl = sub.add_parser("download", help="Download PI/CMI PDFs (by ID or watchlist)")
    p_dl.add_argument("--doc-type", choices=["PI", "CMI"], default="PI",
                      help="Picks which index file + storage dir to use "
                           "(tga_pi_index.json + docs/ vs tga_cmi_index.json + cmi_docs/)")
    p_dl.add_argument("--pi-id", help="Specific PI ID to download")
    p_dl.add_argument("--cmi-id", help="Specific CMI ID to download")
    p_dl.add_argument("--watchlist", help="File with one INN per line")
    p_dl.add_argument("--force", action="store_true",
                      help="Redownload even if file exists")
    p_dl.add_argument("--sleep", type=float, default=0.5)
    p_dl.set_defaults(func=cmd_download)

    args = p.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
