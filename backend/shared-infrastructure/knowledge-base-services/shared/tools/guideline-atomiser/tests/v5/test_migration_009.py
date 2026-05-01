"""Tests for V5 migration 009: provenance_v5 JSONB column on l2_merged_spans.

These tests are file-based assertions on the SQL content — no GCP/PostgreSQL
connection is required. The migration is applied by push_to_kb0_gcp.py
(Task 11) at deploy time.
"""

from __future__ import annotations

from pathlib import Path

import pytest

# Migrations live under kb-0-governance-platform/migrations relative to repo root.
# We resolve from this test file's location upward to the worktree root.
_THIS = Path(__file__).resolve()
# .../guideline-atomiser/tests/v5/test_migration_009.py
# parents: [v5, tests, guideline-atomiser, tools, shared, knowledge-base-services,
#           shared-infrastructure, backend, <worktree-root>]
_WORKTREE_ROOT = _THIS.parents[8]
MIGRATIONS_DIR = (
    _WORKTREE_ROOT
    / "backend"
    / "shared-infrastructure"
    / "knowledge-base-services"
    / "kb-0-governance-platform"
    / "migrations"
)
MIGRATION_FILE = MIGRATIONS_DIR / "009_add_provenance_v5.sql"


@pytest.fixture(scope="module")
def migration_sql() -> str:
    assert MIGRATION_FILE.exists(), f"missing migration file: {MIGRATION_FILE}"
    return MIGRATION_FILE.read_text(encoding="utf-8")


def test_migration_file_exists() -> None:
    assert MIGRATION_FILE.exists(), (
        f"expected migration 009 at {MIGRATION_FILE}"
    )


def test_migration_sql_contains_alter_table(migration_sql: str) -> None:
    assert "ALTER TABLE l2_merged_spans" in migration_sql
    assert "ADD COLUMN IF NOT EXISTS provenance_v5 JSONB" in migration_sql


def test_migration_sql_contains_gin_index(migration_sql: str) -> None:
    assert "CREATE INDEX IF NOT EXISTS" in migration_sql
    assert "USING gin" in migration_sql


def test_migration_sql_is_idempotent(migration_sql: str) -> None:
    # Both DDL statements must be guarded so the migration is safe to re-apply.
    assert "ADD COLUMN IF NOT EXISTS" in migration_sql
    assert "CREATE INDEX IF NOT EXISTS" in migration_sql
