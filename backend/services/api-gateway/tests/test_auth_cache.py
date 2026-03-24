import time
import pytest
from app.auth.auth_cache import AuthResponseCache


def test_cache_miss_returns_none():
    cache = AuthResponseCache(ttl_seconds=60, max_size=100)
    assert cache.get("unknown-token-hash") is None


def test_cache_stores_and_retrieves():
    cache = AuthResponseCache(ttl_seconds=60, max_size=100)
    user_info = {"id": "user-123", "email": "doc@cardiofit.in", "roles": ["physician"]}
    cache.put("token-hash-abc", user_info)
    assert cache.get("token-hash-abc") == user_info


def test_cache_expires_after_ttl():
    cache = AuthResponseCache(ttl_seconds=1, max_size=100)
    cache.put("token-hash-abc", {"id": "user-123"})
    time.sleep(1.1)
    assert cache.get("token-hash-abc") is None


def test_cache_evicts_lru_when_full():
    cache = AuthResponseCache(ttl_seconds=60, max_size=2)
    cache.put("a", {"id": "1"})
    cache.put("b", {"id": "2"})
    cache.put("c", {"id": "3"})  # should evict "a"
    assert cache.get("a") is None
    assert cache.get("b") is not None
    assert cache.get("c") is not None


def test_cache_invalidate():
    cache = AuthResponseCache(ttl_seconds=60, max_size=100)
    cache.put("token-hash-abc", {"id": "user-123"})
    cache.invalidate("token-hash-abc")
    assert cache.get("token-hash-abc") is None
