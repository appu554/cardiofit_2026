"""
Query Cache Layer for Neo4j

High-performance cache for Neo4j query results with TTL support,
cache statistics, and performance optimization for clinical queries.
"""

import asyncio
from typing import Dict, Any, Optional, Tuple
import hashlib
import json
import time
from dataclasses import dataclass
import logging

logger = logging.getLogger(__name__)

@dataclass
class CacheEntry:
    """Cache entry with data, timestamp, and TTL"""
    data: Any
    timestamp: float
    ttl: int  # Time to live in seconds

class Neo4jQueryCache:
    """High-performance cache for Neo4j query results"""
    
    def __init__(self, default_ttl: int = 300):  # 5 minutes default
        self.cache: Dict[str, CacheEntry] = {}
        self.default_ttl = default_ttl
        self.hit_count = 0
        self.miss_count = 0
        self.logger = logging.getLogger(__name__)
        
    def _generate_key(self, query: str, parameters: Dict[str, Any] = None) -> str:
        """Generate cache key from query and parameters"""
        cache_data = {
            'query': query,
            'parameters': parameters or {}
        }
        cache_string = json.dumps(cache_data, sort_keys=True)
        return hashlib.md5(cache_string.encode()).hexdigest()
    
    def _is_expired(self, entry: CacheEntry) -> bool:
        """Check if cache entry is expired"""
        return time.time() - entry.timestamp > entry.ttl
    
    async def get(self, query: str, parameters: Dict[str, Any] = None) -> Optional[Any]:
        """Get cached query result"""
        key = self._generate_key(query, parameters)
        
        if key in self.cache:
            entry = self.cache[key]
            if not self._is_expired(entry):
                self.hit_count += 1
                self.logger.debug(f"Cache hit for query: {query[:50]}...")
                return entry.data
            else:
                # Remove expired entry
                del self.cache[key]
                self.logger.debug(f"Cache entry expired for query: {query[:50]}...")
        
        self.miss_count += 1
        self.logger.debug(f"Cache miss for query: {query[:50]}...")
        return None
    
    async def set(self, query: str, parameters: Dict[str, Any], data: Any, ttl: int = None):
        """Cache query result"""
        key = self._generate_key(query, parameters)
        entry = CacheEntry(
            data=data,
            timestamp=time.time(),
            ttl=ttl or self.default_ttl
        )
        self.cache[key] = entry
        self.logger.debug(f"Cached result for query: {query[:50]}...")
    
    def get_stats(self) -> Dict[str, Any]:
        """Get cache statistics"""
        total_requests = self.hit_count + self.miss_count
        hit_rate = (self.hit_count / total_requests * 100) if total_requests > 0 else 0
        
        return {
            'hit_count': self.hit_count,
            'miss_count': self.miss_count,
            'hit_rate': f"{hit_rate:.1f}%",
            'cache_size': len(self.cache),
            'total_requests': total_requests
        }
    
    async def clear_expired(self):
        """Remove expired cache entries"""
        current_time = time.time()
        expired_keys = [
            key for key, entry in self.cache.items()
            if current_time - entry.timestamp > entry.ttl
        ]
        
        for key in expired_keys:
            del self.cache[key]
        
        if expired_keys:
            self.logger.info(f"Cleared {len(expired_keys)} expired cache entries")
    
    async def clear_all(self):
        """Clear all cache entries"""
        cache_size = len(self.cache)
        self.cache.clear()
        self.hit_count = 0
        self.miss_count = 0
        self.logger.info(f"Cleared all {cache_size} cache entries")
