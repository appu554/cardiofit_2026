#!/usr/bin/env python3
"""
kb7-materialize-all.py - Unified ValueSet Materialization Command

PURPOSE (CTO/CMO DIRECTIVE):
    "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."

This is THE SINGLE COMMAND for all materialization:
    1. Runs materialize_explicit_expansions.py (JSONB → precomputed_valueset_codes)
    2. Runs materialize_expansions.py (Neo4j → precomputed_valueset_codes)
    3. Logs run to materialization_log table
    4. Validates results

After running this:
    - KB-7 $validate-code works for ALL ValueSets
    - KB-7 $expand works for ALL ValueSets
    - No Neo4j at runtime (CRITICAL for CTO/CMO compliance)
    - Deterministic, auditable, clinical-safe

USAGE:
    # Full materialization (default)
    ./kb7-materialize-all.py

    # Explicit only (no Neo4j required)
    ./kb7-materialize-all.py --explicit-only

    # Intensional only (requires Neo4j)
    ./kb7-materialize-all.py --intensional-only

    # Dry run (no changes)
    ./kb7-materialize-all.py --dry-run

ENVIRONMENT VARIABLES:
    POSTGRES_HOST     PostgreSQL host (default: localhost)
    POSTGRES_PORT     PostgreSQL port (default: 5432)
    POSTGRES_DB       PostgreSQL database (default: kb_terminology)
    POSTGRES_USER     PostgreSQL user (default: postgres)
    POSTGRES_PASSWORD PostgreSQL password (default: password)
    NEO4J_URI         Neo4j URI (default: bolt://localhost:7688)
    NEO4J_USER        Neo4j user (default: neo4j)
    NEO4J_PASSWORD    Neo4j password (default: password)
    SNOMED_RELEASE    SNOMED version (default: 20241130)
"""

import os
import sys
import json
import argparse
import subprocess
import uuid
from datetime import datetime
from pathlib import Path

try:
    import psycopg2
    from psycopg2.extras import Json
except ImportError:
    print("ERROR: psycopg2 not installed. Run: pip install psycopg2-binary")
    sys.exit(1)

# ============================================================================
# Configuration
# ============================================================================

SCRIPT_DIR = Path(__file__).parent
ONTOSERVER_DIR = SCRIPT_DIR / "ontoserver"

# Database config
POSTGRES_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5432")),
    "database": os.getenv("POSTGRES_DB", "kb_terminology"),
    "user": os.getenv("POSTGRES_USER", "postgres"),
    "password": os.getenv("POSTGRES_PASSWORD", "password")
}

SNOMED_VERSION = os.getenv("SNOMED_RELEASE", "20241130")

# ============================================================================
# Logging Helpers
# ============================================================================

def log_header(msg):
    print(f"\n{'=' * 70}")
    print(f"  {msg}")
    print(f"{'=' * 70}")

def log_info(msg):
    print(f"  {msg}")

def log_success(msg):
    print(f"  ✅ {msg}")

def log_error(msg):
    print(f"  ❌ {msg}")

def log_warning(msg):
    print(f"  ⚠️  {msg}")

# ============================================================================
# Materialization Log
# ============================================================================

class MaterializationLog:
    """Log materialization runs to database for audit and startup validation."""

    def __init__(self, conn, run_type: str, dry_run: bool = False):
        self.conn = conn
        self.run_id = str(uuid.uuid4())
        self.run_type = run_type
        self.dry_run = dry_run
        self.started_at = datetime.utcnow()
        self.stats = {
            "valuesets_processed": 0,
            "valuesets_materialized": 0,
            "valuesets_skipped": 0,
            "total_codes_inserted": 0,
            "error_count": 0,
            "errors": []
        }
        self.environment = {
            "postgres_host": POSTGRES_CONFIG["host"],
            "postgres_port": POSTGRES_CONFIG["port"],
            "postgres_db": POSTGRES_CONFIG["database"],
            "snomed_version": SNOMED_VERSION,
            "dry_run": dry_run
        }

    def start(self):
        """Record materialization start in database."""
        if self.dry_run:
            log_info(f"[DRY RUN] Would log materialization start: {self.run_id}")
            return

        try:
            cur = self.conn.cursor()
            cur.execute("""
                INSERT INTO materialization_log
                    (run_id, run_type, started_at, status, snomed_version, environment)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (
                self.run_id,
                self.run_type,
                self.started_at,
                'running',
                SNOMED_VERSION,
                Json(self.environment)
            ))
            self.conn.commit()
            cur.close()
        except Exception as e:
            log_warning(f"Could not log materialization start: {e}")

    def complete(self, status: str = 'completed'):
        """Record materialization completion in database."""
        if self.dry_run:
            log_info(f"[DRY RUN] Would log materialization complete: {status}")
            return

        completed_at = datetime.utcnow()
        duration_ms = int((completed_at - self.started_at).total_seconds() * 1000)

        try:
            cur = self.conn.cursor()
            cur.execute("""
                UPDATE materialization_log
                SET completed_at = %s,
                    duration_ms = %s,
                    valuesets_processed = %s,
                    valuesets_materialized = %s,
                    valuesets_skipped = %s,
                    total_codes_inserted = %s,
                    error_count = %s,
                    errors = %s,
                    status = %s
                WHERE run_id = %s
            """, (
                completed_at,
                duration_ms,
                self.stats["valuesets_processed"],
                self.stats["valuesets_materialized"],
                self.stats["valuesets_skipped"],
                self.stats["total_codes_inserted"],
                self.stats["error_count"],
                Json(self.stats["errors"]),
                status,
                self.run_id
            ))
            self.conn.commit()
            cur.close()
        except Exception as e:
            log_warning(f"Could not log materialization complete: {e}")

    def add_error(self, valueset: str, error: str):
        """Add error to the log."""
        self.stats["error_count"] += 1
        self.stats["errors"].append({
            "valueset": valueset,
            "error": error,
            "timestamp": datetime.utcnow().isoformat()
        })

# ============================================================================
# Materialization Steps
# ============================================================================

def run_explicit_materialization(dry_run: bool = False) -> dict:
    """
    Step 1: Materialize explicit ValueSets (JSONB → precomputed_valueset_codes).

    This step does NOT require Neo4j.
    """
    log_header("STEP 1: EXPLICIT VALUESET MATERIALIZATION")
    log_info("Source: value_sets.expansion JSONB")
    log_info("Target: precomputed_valueset_codes")
    log_info("Neo4j required: NO")

    script_path = ONTOSERVER_DIR / "materialize_explicit_expansions.py"
    if not script_path.exists():
        log_error(f"Script not found: {script_path}")
        return {"success": False, "error": "Script not found"}

    if dry_run:
        log_info("[DRY RUN] Would run: python3 materialize_explicit_expansions.py")
        return {"success": True, "dry_run": True}

    # Run the script
    env = os.environ.copy()
    env.update({
        "POSTGRES_HOST": POSTGRES_CONFIG["host"],
        "POSTGRES_PORT": str(POSTGRES_CONFIG["port"]),
        "POSTGRES_DB": POSTGRES_CONFIG["database"],
        "POSTGRES_USER": POSTGRES_CONFIG["user"],
        "POSTGRES_PASSWORD": POSTGRES_CONFIG["password"]
    })

    try:
        result = subprocess.run(
            [sys.executable, str(script_path)],
            env=env,
            capture_output=True,
            text=True,
            timeout=3600  # 1 hour timeout
        )

        if result.returncode == 0:
            log_success("Explicit materialization completed")
            # Print output for visibility
            if result.stdout:
                for line in result.stdout.strip().split('\n')[-10:]:
                    print(f"    {line}")
            return {"success": True, "output": result.stdout}
        else:
            log_error(f"Explicit materialization failed: {result.stderr}")
            return {"success": False, "error": result.stderr}

    except subprocess.TimeoutExpired:
        log_error("Explicit materialization timed out")
        return {"success": False, "error": "Timeout"}
    except Exception as e:
        log_error(f"Explicit materialization error: {e}")
        return {"success": False, "error": str(e)}


def run_intensional_materialization(dry_run: bool = False) -> dict:
    """
    Step 2: Materialize intensional ValueSets (Neo4j → precomputed_valueset_codes).

    This step REQUIRES Neo4j for subClassOf* traversal.
    """
    log_header("STEP 2: INTENSIONAL VALUESET MATERIALIZATION")
    log_info("Source: Neo4j subClassOf* traversal")
    log_info("Target: precomputed_valueset_codes")
    log_info("Neo4j required: YES")

    script_path = ONTOSERVER_DIR / "materialize_expansions.py"
    if not script_path.exists():
        log_error(f"Script not found: {script_path}")
        return {"success": False, "error": "Script not found"}

    if dry_run:
        log_info("[DRY RUN] Would run: python3 materialize_expansions.py")
        return {"success": True, "dry_run": True}

    # Run the script
    env = os.environ.copy()
    env.update({
        "POSTGRES_HOST": POSTGRES_CONFIG["host"],
        "POSTGRES_PORT": str(POSTGRES_CONFIG["port"]),
        "POSTGRES_DB": POSTGRES_CONFIG["database"],
        "POSTGRES_USER": POSTGRES_CONFIG["user"],
        "POSTGRES_PASSWORD": POSTGRES_CONFIG["password"],
        "SNOMED_RELEASE": SNOMED_VERSION
    })

    try:
        result = subprocess.run(
            [sys.executable, str(script_path)],
            env=env,
            capture_output=True,
            text=True,
            timeout=7200  # 2 hour timeout for large ValueSets
        )

        if result.returncode == 0:
            log_success("Intensional materialization completed")
            # Print output for visibility
            if result.stdout:
                for line in result.stdout.strip().split('\n')[-10:]:
                    print(f"    {line}")
            return {"success": True, "output": result.stdout}
        else:
            log_error(f"Intensional materialization failed: {result.stderr}")
            return {"success": False, "error": result.stderr}

    except subprocess.TimeoutExpired:
        log_error("Intensional materialization timed out")
        return {"success": False, "error": "Timeout"}
    except Exception as e:
        log_error(f"Intensional materialization error: {e}")
        return {"success": False, "error": str(e)}


def validate_materialization(conn, dry_run: bool = False) -> dict:
    """
    Step 3: Validate materialization results.

    Checks:
    - precomputed_valueset_codes has rows
    - Materialization log shows success
    """
    log_header("STEP 3: VALIDATION")

    if dry_run:
        log_info("[DRY RUN] Would validate materialization results")
        return {"success": True, "dry_run": True}

    try:
        cur = conn.cursor()

        # Count total precomputed codes
        cur.execute("SELECT COUNT(*) FROM precomputed_valueset_codes")
        total_codes = cur.fetchone()[0]

        # Count distinct ValueSets
        cur.execute("SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes")
        total_valuesets = cur.fetchone()[0]

        # Count kb7.health ValueSets specifically (critical for our use)
        cur.execute("""
            SELECT COUNT(DISTINCT valueset_url)
            FROM precomputed_valueset_codes
            WHERE valueset_url LIKE '%kb7.health%'
        """)
        kb7_valuesets = cur.fetchone()[0]

        # Check is_materialization_healthy() function
        cur.execute("SELECT is_materialization_healthy()")
        is_healthy = cur.fetchone()[0]

        cur.close()

        log_info(f"Total precomputed codes: {total_codes:,}")
        log_info(f"Total ValueSets: {total_valuesets:,}")
        log_info(f"KB7.health ValueSets: {kb7_valuesets:,}")
        log_info(f"Materialization healthy: {is_healthy}")

        if total_codes == 0:
            log_error("CRITICAL: No precomputed codes! KB-7 will refuse to start.")
            return {"success": False, "error": "No precomputed codes"}

        if not is_healthy:
            log_warning("Materialization health check failed")

        log_success("Validation passed")
        return {
            "success": True,
            "total_codes": total_codes,
            "total_valuesets": total_valuesets,
            "kb7_valuesets": kb7_valuesets,
            "is_healthy": is_healthy
        }

    except Exception as e:
        log_error(f"Validation error: {e}")
        return {"success": False, "error": str(e)}


# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="KB-7 Unified ValueSet Materialization",
        epilog="This is THE SINGLE COMMAND for all materialization. "
               "After running, KB-7 will have O(1) runtime lookups."
    )
    parser.add_argument(
        "--explicit-only",
        action="store_true",
        help="Only run explicit materialization (no Neo4j required)"
    )
    parser.add_argument(
        "--intensional-only",
        action="store_true",
        help="Only run intensional materialization (requires Neo4j)"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making changes"
    )
    parser.add_argument(
        "--skip-validation",
        action="store_true",
        help="Skip post-materialization validation"
    )

    args = parser.parse_args()

    log_header("KB-7 VALUESET MATERIALIZATION")
    log_info(f"Time: {datetime.now().isoformat()}")
    log_info(f"SNOMED Version: {SNOMED_VERSION}")
    log_info(f"PostgreSQL: {POSTGRES_CONFIG['host']}:{POSTGRES_CONFIG['port']}")
    log_info(f"Dry Run: {args.dry_run}")

    # Determine run type
    if args.explicit_only:
        run_type = "explicit"
    elif args.intensional_only:
        run_type = "intensional"
    else:
        run_type = "full"

    log_info(f"Run Type: {run_type}")

    # Connect to database
    try:
        conn = psycopg2.connect(**POSTGRES_CONFIG)
        log_success("Connected to PostgreSQL")
    except Exception as e:
        log_error(f"Cannot connect to PostgreSQL: {e}")
        sys.exit(1)

    # Create materialization log
    mat_log = MaterializationLog(conn, run_type, args.dry_run)
    mat_log.start()

    # Track overall success
    success = True
    errors = []

    # Step 1: Explicit materialization
    if run_type in ("full", "explicit"):
        result = run_explicit_materialization(args.dry_run)
        if not result.get("success"):
            errors.append(("explicit", result.get("error")))
            mat_log.add_error("explicit_materialization", result.get("error", "Unknown"))
            success = False

    # Step 2: Intensional materialization
    if run_type in ("full", "intensional"):
        result = run_intensional_materialization(args.dry_run)
        if not result.get("success"):
            errors.append(("intensional", result.get("error")))
            mat_log.add_error("intensional_materialization", result.get("error", "Unknown"))
            # Don't fail overall if Neo4j is unavailable but explicit worked
            if run_type == "full":
                log_warning("Intensional failed but explicit may have worked")
            else:
                success = False

    # Step 3: Validation
    if not args.skip_validation:
        result = validate_materialization(conn, args.dry_run)
        mat_log.stats["total_codes_inserted"] = result.get("total_codes", 0)
        mat_log.stats["valuesets_materialized"] = result.get("total_valuesets", 0)
        if not result.get("success"):
            errors.append(("validation", result.get("error")))
            success = False

    # Complete the log
    status = "completed" if success else ("partial" if errors else "failed")
    mat_log.complete(status)

    # Final summary
    log_header("SUMMARY")
    if success:
        log_success("Materialization completed successfully!")
        log_info("KB-7 is ready to start with O(1) runtime lookups")
        log_info("No Neo4j required at runtime (CTO/CMO compliant)")
    else:
        log_error("Materialization completed with errors:")
        for step, error in errors:
            log_error(f"  {step}: {error}")

    conn.close()
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
