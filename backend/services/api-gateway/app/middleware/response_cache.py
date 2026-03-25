"""Read-through response cache for latency-sensitive reads.

Caches GET responses for KB-20 projections, KB-26 MRI, and other
read-heavy endpoints. TTL matches Redis cache in KB services (2min).
"""
import hashlib
import json
import logging
from typing import Optional

import redis.asyncio as aioredis

logger = logging.getLogger(__name__)

# Cacheable path prefixes and their TTLs (seconds)
CACHE_RULES: dict[str, int] = {
    "/api/v1/doctor/patients/": 120,      # 2min — matches KB-20 Redis TTL
    "/api/v1/patient/": 60,               # 1min — patient data
    "/api/v1/tenants/": 3600,             # 1hr — branding rarely changes
}


class ResponseCache:
    def __init__(self, redis_url: str):
        self.redis = aioredis.from_url(redis_url, decode_responses=True)

    def _cache_key(self, method: str, path: str, user_id: str) -> Optional[str]:
        """Generate cache key. Returns None if path is not cacheable."""
        if method != "GET":
            return None
        for prefix, _ in CACHE_RULES.items():
            if path.startswith(prefix):
                raw = f"{method}:{path}:{user_id}"
                return f"cache:{hashlib.sha256(raw.encode()).hexdigest()[:16]}"
        return None

    def _get_ttl(self, path: str) -> int:
        for prefix, ttl in CACHE_RULES.items():
            if path.startswith(prefix):
                return ttl
        return 60

    async def get(self, method: str, path: str, user_id: str) -> Optional[dict]:
        key = self._cache_key(method, path, user_id)
        if not key:
            return None
        try:
            data = await self.redis.get(key)
            if data:
                logger.debug("Cache HIT: %s", key)
                return json.loads(data)
        except Exception:
            pass
        return None

    async def set(self, method: str, path: str, user_id: str, response_data: dict):
        key = self._cache_key(method, path, user_id)
        if not key:
            return
        try:
            ttl = self._get_ttl(path)
            await self.redis.setex(key, ttl, json.dumps(response_data))
        except Exception:
            pass  # cache miss is not fatal
