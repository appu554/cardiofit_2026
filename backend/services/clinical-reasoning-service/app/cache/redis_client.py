"""
CAE Redis Client

This module implements Redis caching specifically for the Clinical Assertion Engine.
Separate from main microservices Redis - this is CAE's dedicated cache.
"""

import logging
import json
import asyncio
from typing import Dict, Any, Optional, List
from datetime import datetime, timedelta
import hashlib

logger = logging.getLogger(__name__)

# Try to import Redis - graceful fallback if not available
try:
    import redis.asyncio as redis
    REDIS_AVAILABLE = True
except ImportError:
    logger.warning("Redis not available - using in-memory cache fallback")
    REDIS_AVAILABLE = False

class CAERedisClient:
    """
    CAE-specific Redis client for clinical reasoning caching
    
    This is separate from the main microservices Redis and is dedicated
    to Clinical Assertion Engine performance optimization.
    """
    
    def __init__(
        self,
        host: str = "localhost",
        port: int = 6380,  # CAE-specific Redis port
        db: int = 0,
        password: Optional[str] = None
    ):
        self.host = host
        self.port = port
        self.db = db
        self.password = password
        self.redis_client = None
        self.fallback_cache = {}  # In-memory fallback
        self.cache_stats = {
            "hits": 0,
            "misses": 0,
            "sets": 0,
            "errors": 0
        }
        
        # Cache TTL settings (in seconds)
        self.ttl_settings = {
            "patient_context": 300,      # 5 minutes
            "drug_interactions": 3600,   # 1 hour
            "dosing_guidelines": 7200,   # 2 hours
            "contraindications": 3600,   # 1 hour
            "clinical_knowledge": 86400, # 24 hours
            "session_data": 1800,        # 30 minutes
        }
        
        logger.info(f"CAE Redis client initialized for {host}:{port}")
    
    async def connect(self) -> bool:
        """Connect to CAE Redis instance"""
        if not REDIS_AVAILABLE:
            logger.warning("Redis not available, using in-memory cache")
            return False
        
        try:
            self.redis_client = redis.Redis(
                host=self.host,
                port=self.port,
                db=self.db,
                password=self.password,
                decode_responses=True,
                socket_connect_timeout=5,
                socket_timeout=5
            )
            
            # Test connection
            await self.redis_client.ping()
            logger.info(f"✅ Connected to CAE Redis at {self.host}:{self.port}")
            return True
            
        except Exception as e:
            logger.warning(f"Failed to connect to CAE Redis: {e}")
            self.redis_client = None
            return False
    
    async def disconnect(self):
        """Disconnect from Redis"""
        if self.redis_client:
            await self.redis_client.close()
            logger.info("Disconnected from CAE Redis")
    
    def _generate_cache_key(self, category: str, identifier: str, **kwargs) -> str:
        """Generate consistent cache key"""
        key_parts = [f"cae:{category}:{identifier}"]
        
        # Add additional parameters to key
        if kwargs:
            sorted_params = sorted(kwargs.items())
            param_str = ":".join([f"{k}={v}" for k, v in sorted_params])
            key_parts.append(param_str)
        
        cache_key = ":".join(key_parts)
        
        # Hash long keys to prevent Redis key length issues
        if len(cache_key) > 200:
            cache_key = f"cae:{category}:{hashlib.md5(cache_key.encode()).hexdigest()}"
        
        return cache_key
    
    async def get_patient_context(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get cached patient context"""
        cache_key = self._generate_cache_key("patient_context", patient_id)
        return await self._get_json(cache_key, "patient_context")
    
    async def set_patient_context(self, patient_id: str, context: Dict[str, Any]) -> bool:
        """Cache patient context"""
        cache_key = self._generate_cache_key("patient_context", patient_id)
        return await self._set_json(cache_key, context, "patient_context")
    
    async def get_drug_interactions(
        self,
        medication_ids: List[str],
        patient_factors: Optional[Dict[str, Any]] = None
    ) -> Optional[List[Dict[str, Any]]]:
        """Get cached drug interactions"""
        # Sort medication IDs for consistent caching
        sorted_meds = sorted(medication_ids)
        med_key = ":".join(sorted_meds)
        
        cache_key = self._generate_cache_key(
            "drug_interactions",
            med_key,
            **(patient_factors or {})
        )
        return await self._get_json(cache_key, "drug_interactions")
    
    async def set_drug_interactions(
        self,
        medication_ids: List[str],
        interactions: List[Dict[str, Any]],
        patient_factors: Optional[Dict[str, Any]] = None
    ) -> bool:
        """Cache drug interactions"""
        sorted_meds = sorted(medication_ids)
        med_key = ":".join(sorted_meds)
        
        cache_key = self._generate_cache_key(
            "drug_interactions",
            med_key,
            **(patient_factors or {})
        )
        return await self._set_json(cache_key, interactions, "drug_interactions")
    
    async def get_dosing_recommendation(
        self,
        medication_id: str,
        patient_parameters: Dict[str, Any]
    ) -> Optional[Dict[str, Any]]:
        """Get cached dosing recommendation"""
        cache_key = self._generate_cache_key(
            "dosing_guidelines",
            medication_id,
            **patient_parameters
        )
        return await self._get_json(cache_key, "dosing_guidelines")
    
    async def set_dosing_recommendation(
        self,
        medication_id: str,
        patient_parameters: Dict[str, Any],
        recommendation: Dict[str, Any]
    ) -> bool:
        """Cache dosing recommendation"""
        cache_key = self._generate_cache_key(
            "dosing_guidelines",
            medication_id,
            **patient_parameters
        )
        return await self._set_json(cache_key, recommendation, "dosing_guidelines")
    
    async def get_contraindications(
        self,
        medication_ids: List[str],
        condition_ids: List[str],
        allergy_ids: List[str]
    ) -> Optional[List[Dict[str, Any]]]:
        """Get cached contraindications"""
        cache_key = self._generate_cache_key(
            "contraindications",
            f"meds:{':'.join(sorted(medication_ids))}",
            conditions=":".join(sorted(condition_ids)),
            allergies=":".join(sorted(allergy_ids))
        )
        return await self._get_json(cache_key, "contraindications")
    
    async def set_contraindications(
        self,
        medication_ids: List[str],
        condition_ids: List[str],
        allergy_ids: List[str],
        contraindications: List[Dict[str, Any]]
    ) -> bool:
        """Cache contraindications"""
        cache_key = self._generate_cache_key(
            "contraindications",
            f"meds:{':'.join(sorted(medication_ids))}",
            conditions=":".join(sorted(condition_ids)),
            allergies=":".join(sorted(allergy_ids))
        )
        return await self._set_json(cache_key, contraindications, "contraindications")
    
    async def get_clinical_knowledge(self, knowledge_type: str, identifier: str) -> Optional[Dict[str, Any]]:
        """Get cached clinical knowledge (drug databases, guidelines, etc.)"""
        cache_key = self._generate_cache_key("clinical_knowledge", f"{knowledge_type}:{identifier}")
        return await self._get_json(cache_key, "clinical_knowledge")
    
    async def set_clinical_knowledge(
        self,
        knowledge_type: str,
        identifier: str,
        knowledge_data: Dict[str, Any]
    ) -> bool:
        """Cache clinical knowledge"""
        cache_key = self._generate_cache_key("clinical_knowledge", f"{knowledge_type}:{identifier}")
        return await self._set_json(cache_key, knowledge_data, "clinical_knowledge")
    
    async def get_session_data(self, session_id: str) -> Optional[Dict[str, Any]]:
        """Get cached session data"""
        cache_key = self._generate_cache_key("session_data", session_id)
        return await self._get_json(cache_key, "session_data")
    
    async def set_session_data(self, session_id: str, session_data: Dict[str, Any]) -> bool:
        """Cache session data"""
        cache_key = self._generate_cache_key("session_data", session_id)
        return await self._set_json(cache_key, session_data, "session_data")
    
    async def _get_json(self, cache_key: str, category: str) -> Optional[Dict[str, Any]]:
        """Get JSON data from cache"""
        try:
            if self.redis_client:
                data = await self.redis_client.get(cache_key)
                if data:
                    self.cache_stats["hits"] += 1
                    return json.loads(data)
                else:
                    self.cache_stats["misses"] += 1
                    return None
            else:
                # Fallback to in-memory cache
                if cache_key in self.fallback_cache:
                    cached_item = self.fallback_cache[cache_key]
                    if datetime.now() < cached_item["expires"]:
                        self.cache_stats["hits"] += 1
                        return cached_item["data"]
                    else:
                        del self.fallback_cache[cache_key]
                
                self.cache_stats["misses"] += 1
                return None
                
        except Exception as e:
            logger.error(f"Cache get error for {cache_key}: {e}")
            self.cache_stats["errors"] += 1
            return None
    
    async def _set_json(self, cache_key: str, data: Dict[str, Any], category: str) -> bool:
        """Set JSON data in cache"""
        try:
            ttl = self.ttl_settings.get(category, 300)
            
            if self.redis_client:
                await self.redis_client.setex(
                    cache_key,
                    ttl,
                    json.dumps(data, default=str)
                )
                self.cache_stats["sets"] += 1
                return True
            else:
                # Fallback to in-memory cache
                expires = datetime.now() + timedelta(seconds=ttl)
                self.fallback_cache[cache_key] = {
                    "data": data,
                    "expires": expires
                }
                self.cache_stats["sets"] += 1
                return True
                
        except Exception as e:
            logger.error(f"Cache set error for {cache_key}: {e}")
            self.cache_stats["errors"] += 1
            return False
    
    async def invalidate_patient_cache(self, patient_id: str) -> bool:
        """Invalidate all cached data for a patient"""
        try:
            if self.redis_client:
                pattern = f"cae:*:{patient_id}*"
                keys = await self.redis_client.keys(pattern)
                if keys:
                    await self.redis_client.delete(*keys)
                    logger.info(f"Invalidated {len(keys)} cache entries for patient {patient_id}")
                return True
            else:
                # Fallback: remove from in-memory cache
                keys_to_remove = [k for k in self.fallback_cache.keys() if patient_id in k]
                for key in keys_to_remove:
                    del self.fallback_cache[key]
                logger.info(f"Invalidated {len(keys_to_remove)} cache entries for patient {patient_id}")
                return True
                
        except Exception as e:
            logger.error(f"Cache invalidation error for patient {patient_id}: {e}")
            return False
    
    async def get_cache_stats(self) -> Dict[str, Any]:
        """Get cache performance statistics"""
        total_requests = self.cache_stats["hits"] + self.cache_stats["misses"]
        hit_rate = (self.cache_stats["hits"] / total_requests * 100) if total_requests > 0 else 0
        
        stats = {
            "hits": self.cache_stats["hits"],
            "misses": self.cache_stats["misses"],
            "sets": self.cache_stats["sets"],
            "errors": self.cache_stats["errors"],
            "hit_rate_percent": round(hit_rate, 2),
            "total_requests": total_requests,
            "redis_connected": self.redis_client is not None,
            "fallback_cache_size": len(self.fallback_cache)
        }
        
        if self.redis_client:
            try:
                info = await self.redis_client.info()
                stats["redis_memory_used"] = info.get("used_memory_human", "unknown")
                stats["redis_connected_clients"] = info.get("connected_clients", 0)
            except Exception:
                pass
        
        return stats
    
    async def health_check(self) -> Dict[str, Any]:
        """Health check for CAE Redis"""
        health = {
            "status": "healthy",
            "redis_available": False,
            "fallback_active": False,
            "cache_stats": await self.get_cache_stats()
        }
        
        if self.redis_client:
            try:
                await self.redis_client.ping()
                health["redis_available"] = True
            except Exception as e:
                health["status"] = "degraded"
                health["redis_error"] = str(e)
                health["fallback_active"] = True
        else:
            health["status"] = "degraded"
            health["fallback_active"] = True
        
        return health
