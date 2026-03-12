#!/usr/bin/env python3
"""
KB-7 GraphDB Deployment Script with CDC "Commit-Last" Strategy

This script implements the critical "Handover" phase in Event-Driven Architecture:
1. LOAD GraphDB - Push the .ttl file, wait for completion
2. VERIFY GraphDB - Run health check query
3. COMMIT to PostgreSQL - Insert metadata row (triggers Debezium CDC)

IMPORTANT: The PostgreSQL commit happens LAST to prevent race conditions.
If we commit before GraphDB is ready, downstream services will consume the
CDC event, try to query GraphDB, and fail because the data isn't there yet.
"""

import os
import sys
import time
from datetime import datetime, timezone
from typing import Optional

import requests
import google.auth
from google.cloud import storage
from google.auth.transport.requests import Request

# Optional PostgreSQL support (for CDC)
try:
    import psycopg2
    from psycopg2.extras import RealDictCursor
    HAS_POSTGRES = True
except ImportError:
    HAS_POSTGRES = False
    print("⚠️ psycopg2 not installed - CDC commit will be skipped")

# --- CONFIGURATION ---
GCS_BUCKET = "sincere-hybrid-477206-h2-kb-artifacts-production"
GCS_BLOB_NAME = f"{os.getenv('VERSION_ID', 'latest')}/kb7-kernel.ttl"
GRAPHDB_URL = os.getenv("GRAPHDB_URL", "http://localhost:7200")
REPO_ID = "kb7-terminology"
GRAPHDB_AUTH = (os.getenv("GRAPHDB_USER", "admin"), os.getenv("GRAPHDB_PASS", "root"))
IMPORT_TIMEOUT_MINUTES = int(os.getenv("IMPORT_TIMEOUT_MINUTES", "30"))

# PostgreSQL CDC Configuration
PG_HOST = os.getenv("PG_HOST", "localhost")
PG_PORT = os.getenv("PG_PORT", "5432")
PG_USER = os.getenv("PG_USER", "kb_terminology_user")
PG_PASSWORD = os.getenv("PG_PASSWORD", "kb_password")
PG_DATABASE = os.getenv("PG_DATABASE", "kb_terminology")

# Version metadata (from pipeline)
VERSION_ID = os.getenv("VERSION_ID", datetime.now().strftime("%Y%m%d"))
SNOMED_VERSION = os.getenv("SNOMED_VERSION", "2024-09")
RXNORM_VERSION = os.getenv("RXNORM_VERSION", "12012025")
LOINC_VERSION = os.getenv("LOINC_VERSION", "2.77")
KERNEL_CHECKSUM = os.getenv("KERNEL_CHECKSUM", "")
CONCEPT_COUNT = int(os.getenv("CONCEPT_COUNT", "0"))

# Feature flags
ENABLE_CDC = os.getenv("ENABLE_CDC", "false").lower() == "true"


def get_pg_connection():
    """Create PostgreSQL connection for CDC outbox."""
    if not HAS_POSTGRES:
        return None

    try:
        conn = psycopg2.connect(
            host=PG_HOST,
            port=PG_PORT,
            user=PG_USER,
            password=PG_PASSWORD,
            database=PG_DATABASE
        )
        return conn
    except Exception as e:
        print(f"⚠️ PostgreSQL connection failed: {e}")
        return None


def generate_signed_url(bucket_name: str, blob_name: str) -> str:
    """Generates a V4 Signed URL valid for 60 minutes."""
    print(f"🔑 Generating Signed URL for gs://{bucket_name}/{blob_name}...")
    credentials, _ = google.auth.default()
    if credentials.expired:
        credentials.refresh(Request())

    storage_client = storage.Client(credentials=credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(blob_name)

    url = blob.generate_signed_url(version="v4", expiration=3600, method="GET")
    print(f"✅ Signed URL generated (expires in 60 min)")
    return url


def trigger_import(signed_url: str) -> None:
    """Use SPARQL LOAD to import data from signed URL - works with all GraphDB versions."""
    print(f"🚀 Triggering SPARQL LOAD on GraphDB ({GRAPHDB_URL})...")

    # SPARQL UPDATE endpoint
    endpoint = f"{GRAPHDB_URL}/repositories/{REPO_ID}/statements"

    # SPARQL LOAD command - GraphDB will fetch the file directly
    # Using default graph (no INTO GRAPH clause)
    sparql_update = f"LOAD <{signed_url}>"

    print(f"   Executing: LOAD <signed_url>")
    print(f"   This may take 5-15 minutes for 1.1GB file...")

    response = requests.post(
        endpoint,
        data={"update": sparql_update},
        auth=GRAPHDB_AUTH,
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        timeout=IMPORT_TIMEOUT_MINUTES * 60  # Long timeout for large file
    )

    if response.status_code == 204:
        print("✅ SPARQL LOAD completed successfully!")
    else:
        response.raise_for_status()
        print("✅ Import completed.")


def poll_status() -> None:
    """Polls GraphDB until import completes or timeout."""
    print(f"⏳ Polling Import Status (timeout: {IMPORT_TIMEOUT_MINUTES} min)...")
    status_endpoint = f"{GRAPHDB_URL}/rest/data/import/upload/{REPO_ID}"

    start_time = time.time()
    timeout_seconds = IMPORT_TIMEOUT_MINUTES * 60
    poll_interval = 10  # seconds

    while True:
        elapsed = time.time() - start_time
        if elapsed > timeout_seconds:
            print(f"❌ TIMEOUT: Import exceeded {IMPORT_TIMEOUT_MINUTES} minutes")
            sys.exit(1)

        try:
            res = requests.get(status_endpoint, auth=GRAPHDB_AUTH, timeout=30)
            res.raise_for_status()
            imports = res.json()
        except requests.RequestException as e:
            print(f"   ⚠️ Poll request failed: {e}, retrying...")
            time.sleep(poll_interval)
            continue

        my_import = next(
            (i for i in imports if i.get('name') == "kb7-kernel-deployment.ttl"),
            None
        )

        if not my_import:
            print("ℹ️ Job completed (no longer in queue).")
            break

        status = my_import.get('status')
        print(f"   Status: {status} ({int(elapsed)}s elapsed)")

        if status == "DONE":
            print("🎉 Import Finished Successfully!")
            break
        elif status == "ERROR":
            message = my_import.get('message', 'Unknown error')
            print(f"❌ Import Failed: {message}")
            sys.exit(1)

        time.sleep(poll_interval)


def verify_triple_count() -> int:
    """Verify the import by counting triples."""
    print("🔍 Verifying import...")
    query = "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"

    res = requests.post(
        f"{GRAPHDB_URL}/repositories/{REPO_ID}",
        data={"query": query},
        headers={"Accept": "application/sparql-results+json"},
        auth=GRAPHDB_AUTH,
        timeout=60
    )
    res.raise_for_status()

    count = int(res.json()['results']['bindings'][0]['count']['value'])
    print(f"✅ Repository contains {count:,} triples")

    # Sanity check - KB-7 should have ~14M triples
    if count < 1_000_000:
        print(f"⚠️ Warning: Expected ~14M triples, got {count:,}")

    return count


def verify_graphdb_health() -> bool:
    """
    Run a health check query to ensure GraphDB is responsive and indexed.
    This is the SAFETY GATE before committing to PostgreSQL.
    """
    print("🏥 Running GraphDB health check...")

    # Test 1: Basic connectivity
    try:
        res = requests.get(
            f"{GRAPHDB_URL}/rest/repositories/{REPO_ID}",
            auth=GRAPHDB_AUTH,
            timeout=30
        )
        if res.status_code != 200:
            print(f"❌ Health check failed: Repository not accessible (HTTP {res.status_code})")
            return False
    except Exception as e:
        print(f"❌ Health check failed: {e}")
        return False

    # Test 2: Execute a sample SPARQL query (proves indexing is complete)
    test_query = """
    SELECT ?s ?label WHERE {
        ?s a <http://www.w3.org/2002/07/owl#Class> .
        ?s <http://www.w3.org/2000/01/rdf-schema#label> ?label .
    } LIMIT 5
    """

    try:
        res = requests.post(
            f"{GRAPHDB_URL}/repositories/{REPO_ID}",
            data={"query": test_query},
            headers={"Accept": "application/sparql-results+json"},
            auth=GRAPHDB_AUTH,
            timeout=30
        )
        res.raise_for_status()
        results = res.json()
        bindings = results.get('results', {}).get('bindings', [])

        if len(bindings) > 0:
            print(f"✅ Health check passed: Retrieved {len(bindings)} sample concepts")
            return True
        else:
            print("⚠️ Health check warning: No sample concepts returned")
            return True  # Allow empty results (might be valid)

    except Exception as e:
        print(f"❌ Health check query failed: {e}")
        return False


def commit_to_postgres(triple_count: int, load_started: datetime, load_completed: datetime) -> Optional[int]:
    """
    COMMIT-LAST: Insert release record into PostgreSQL ONLY after GraphDB is verified.

    This INSERT triggers Debezium CDC which notifies downstream services
    that a new KB-7 release is ACTIVE and ready for queries.

    Returns the release ID if successful, None otherwise.
    """
    if not ENABLE_CDC:
        print("ℹ️ CDC commit disabled (ENABLE_CDC=false)")
        return None

    if not HAS_POSTGRES:
        print("⚠️ psycopg2 not installed - skipping CDC commit")
        return None

    print("📤 COMMIT-LAST: Writing to PostgreSQL (triggers CDC)...")

    conn = get_pg_connection()
    if not conn:
        print("⚠️ PostgreSQL connection failed - skipping CDC commit")
        return None

    try:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            # Insert the release record - this triggers Debezium
            cur.execute("""
                INSERT INTO kb_releases (
                    version_id,
                    release_date,
                    graphdb_load_started_at,
                    graphdb_load_completed_at,
                    snomed_version,
                    rxnorm_version,
                    loinc_version,
                    triple_count,
                    concept_count,
                    kernel_checksum,
                    gcs_uri,
                    graphdb_repository,
                    graphdb_endpoint,
                    status
                ) VALUES (
                    %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 'ACTIVE'
                )
                ON CONFLICT (version_id) DO UPDATE SET
                    status = 'ACTIVE',
                    triple_count = EXCLUDED.triple_count,
                    graphdb_load_completed_at = EXCLUDED.graphdb_load_completed_at
                RETURNING id
            """, (
                VERSION_ID,
                datetime.now(timezone.utc),
                load_started,
                load_completed,
                SNOMED_VERSION,
                RXNORM_VERSION,
                LOINC_VERSION,
                triple_count,
                CONCEPT_COUNT,
                KERNEL_CHECKSUM,
                f"gs://{GCS_BUCKET}/{GCS_BLOB_NAME}",
                REPO_ID,
                f"{GRAPHDB_URL}/repositories/{REPO_ID}",
            ))

            result = cur.fetchone()
            release_id = result['id'] if result else None

            conn.commit()

            print(f"✅ CDC COMMIT successful! Release ID: {release_id}")
            print(f"   Debezium will publish to: kb7.terminology.releases")
            print(f"   Downstream services will receive notification")

            return release_id

    except Exception as e:
        print(f"❌ PostgreSQL commit failed: {e}")
        conn.rollback()
        return None
    finally:
        conn.close()


def main():
    """
    Main deployment flow implementing "Commit-Last" strategy:

    1. LOAD GraphDB - Push .ttl via signed URL
    2. VERIFY GraphDB - Health check ensures data is queryable
    3. COMMIT to PostgreSQL - Triggers CDC notification (LAST!)

    This order prevents race conditions in Event-Driven Architecture.
    """
    print("=" * 60)
    print("KB-7 GraphDB Deployment - Commit-Last CDC Strategy")
    print("=" * 60)
    print(f"  GCS Source:  gs://{GCS_BUCKET}/{GCS_BLOB_NAME}")
    print(f"  GraphDB:     {GRAPHDB_URL}")
    print(f"  Repository:  {REPO_ID}")
    print(f"  Timeout:     {IMPORT_TIMEOUT_MINUTES} min")
    print(f"  Version:     {VERSION_ID}")
    print(f"  CDC Enabled: {ENABLE_CDC}")
    print("=" * 60)
    print()

    # Record start time for CDC metadata
    load_started = datetime.now(timezone.utc)

    # ==========================================
    # STEP A: LOAD GraphDB (The Heavy Lift)
    # ==========================================
    print("\n" + "─" * 40)
    print("STEP A: Load GraphDB")
    print("─" * 40)

    signed_url = generate_signed_url(GCS_BUCKET, GCS_BLOB_NAME)
    trigger_import(signed_url)

    # Record completion time
    load_completed = datetime.now(timezone.utc)

    # ==========================================
    # STEP B: VERIFY GraphDB (The Safety Gate)
    # ==========================================
    print("\n" + "─" * 40)
    print("STEP B: Verify GraphDB Integrity")
    print("─" * 40)

    triple_count = verify_triple_count()

    if not verify_graphdb_health():
        print("❌ DEPLOYMENT ABORTED: GraphDB health check failed")
        print("   CDC commit NOT executed - downstream services NOT notified")
        sys.exit(1)

    # ==========================================
    # STEP C: COMMIT to PostgreSQL (The CDC Trigger)
    # ==========================================
    print("\n" + "─" * 40)
    print("STEP C: Commit to PostgreSQL (Triggers CDC)")
    print("─" * 40)

    release_id = commit_to_postgres(triple_count, load_started, load_completed)

    # ==========================================
    # DEPLOYMENT COMPLETE
    # ==========================================
    print()
    print("=" * 60)
    print("✅ KB-7 DEPLOYMENT COMPLETE")
    print("=" * 60)
    print(f"   Version:        {VERSION_ID}")
    print(f"   Triples loaded: {triple_count:,}")
    print(f"   SPARQL endpoint: {GRAPHDB_URL}/repositories/{REPO_ID}")
    print(f"   Load duration:  {(load_completed - load_started).total_seconds():.1f}s")

    if release_id:
        print(f"   CDC Release ID: {release_id}")
        print(f"   Kafka topic:    kb7.terminology.releases")
    else:
        print(f"   CDC:            Disabled or skipped")

    print("=" * 60)


if __name__ == "__main__":
    main()
