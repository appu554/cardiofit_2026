"""In-memory TTL cache for Auth Service /verify responses.

Eliminates redundant Auth Service calls by caching verified token claims.
JWT secrets stay in Auth Service (port 8001) — NOT duplicated here.
"""
import hashlib
import logging
import threading
import time
from collections import OrderedDict
from typing import Any, Optional

logger = logging.getLogger(__name__)


class AuthResponseCache:
    """Thread-safe LRU cache with TTL for Auth Service verify responses.

    Cache key: SHA-256 hash of the Bearer token (never stores raw tokens).
    Cache value: The user_info dict returned by Auth Service /verify.
    """

    def __init__(self, ttl_seconds: int = 60, max_size: int = 10000):
        self._ttl = ttl_seconds
        self._max_size = max_size
        self._cache: OrderedDict[str, tuple[float, dict]] = OrderedDict()
        self._lock = threading.Lock()

    @staticmethod
    def hash_token(token: str) -> str:
        """Hash a Bearer token for use as cache key. Never store raw tokens."""
        return hashlib.sha256(token.encode()).hexdigest()[:32]

    def get(self, token_hash: str) -> Optional[dict]:
        """Get cached user_info for a token hash. Returns None on miss or expiry."""
        with self._lock:
            entry = self._cache.get(token_hash)
            if entry is None:
                return None
            expires_at, user_info = entry
            if time.time() > expires_at:
                del self._cache[token_hash]
                return None
            # Move to end (most recently used)
            self._cache.move_to_end(token_hash)
            return user_info

    def put(self, token_hash: str, user_info: dict):
        """Cache a verified user_info dict."""
        with self._lock:
            expires_at = time.time() + self._ttl
            self._cache[token_hash] = (expires_at, user_info)
            self._cache.move_to_end(token_hash)
            # Evict LRU if over capacity
            while len(self._cache) > self._max_size:
                self._cache.popitem(last=False)

    def invalidate(self, token_hash: str):
        """Remove a specific token from cache (e.g., on logout)."""
        with self._lock:
            self._cache.pop(token_hash, None)

    def clear(self):
        """Clear all cached entries."""
        with self._lock:
            self._cache.clear()

    @property
    def size(self) -> int:
        return len(self._cache)
