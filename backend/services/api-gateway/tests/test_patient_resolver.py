"""Tests for FHIR UUID → ABHA patient ID resolution."""
import pytest
from unittest.mock import AsyncMock, patch, MagicMock

from app.api.patient_resolver import resolve_patient_id, UUID_PATTERN


def test_uuid_pattern_matches_fhir_uuid():
    assert UUID_PATTERN.match("550e8400-e29b-41d4-a716-446655440000")


def test_uuid_pattern_rejects_abha_id():
    assert not UUID_PATTERN.match("91-1001-2001-3001")


def test_uuid_pattern_rejects_plain_string():
    assert not UUID_PATTERN.match("patient-abc-123")


@pytest.mark.anyio
async def test_non_uuid_returns_unchanged():
    """ABHA IDs pass through without any KB-20 call."""
    result = await resolve_patient_id("91-1001-2001-3001")
    assert result == "91-1001-2001-3001"


@pytest.mark.anyio
@patch("app.api.patient_resolver._cache_set", new_callable=AsyncMock)
@patch("app.api.patient_resolver._cache_get", new_callable=AsyncMock, return_value=None)
@patch("app.api.patient_resolver._call_kb20_resolve", new_callable=AsyncMock)
async def test_uuid_calls_kb20(mock_kb20, mock_cache_get, mock_cache_set):
    """FHIR UUID triggers KB-20 resolution."""
    mock_kb20.return_value = "91-1001-2001-3001"
    result = await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
    assert result == "91-1001-2001-3001"
    mock_kb20.assert_called_once_with("550e8400-e29b-41d4-a716-446655440000")


@pytest.mark.anyio
@patch("app.api.patient_resolver._call_kb20_resolve")
async def test_uuid_cached_on_second_call(mock_kb20):
    """Second call for same UUID uses cache, not KB-20."""
    mock_kb20.return_value = "91-1001-2001-3001"

    with patch("app.api.patient_resolver._cache_get", new_callable=AsyncMock) as mock_get, \
         patch("app.api.patient_resolver._cache_set", new_callable=AsyncMock) as mock_set:
        mock_get.return_value = None  # cache miss
        result1 = await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
        assert result1 == "91-1001-2001-3001"
        mock_set.assert_called_once()

        # Second call — cache hit
        mock_get.return_value = "91-1001-2001-3001"
        result2 = await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
        assert result2 == "91-1001-2001-3001"
        # KB-20 should NOT be called again
        assert mock_kb20.call_count == 1


@pytest.mark.anyio
@patch("app.api.patient_resolver._cache_get", new_callable=AsyncMock, return_value=None)
@patch("app.api.patient_resolver._call_kb20_resolve", new_callable=AsyncMock)
async def test_kb20_failure_raises_502(mock_kb20, mock_cache_get):
    """If KB-20 is unreachable, raise HTTP 502."""
    from fastapi import HTTPException
    mock_kb20.side_effect = HTTPException(status_code=502, detail="KB-20 service unavailable")
    with pytest.raises(HTTPException) as exc_info:
        await resolve_patient_id("550e8400-e29b-41d4-a716-446655440000")
    assert exc_info.value.status_code == 502
