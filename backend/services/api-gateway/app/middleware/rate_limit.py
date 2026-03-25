from fastapi import Request, Response, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from typing import Callable, Dict, List, Optional, Any
import logging
import time
import asyncio
from collections import defaultdict

import redis.asyncio as aioredis
from app.config import settings  # module-level singleton

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class RedisRateLimiter:
    """Sliding window rate limiter backed by Redis sorted sets.
    Falls back to in-memory limiter if Redis is unavailable.
    """

    def __init__(self, redis_url: str, max_requests: int = 100, window_seconds: int = 60):
        self.redis = aioredis.from_url(redis_url, decode_responses=True)
        self.max_requests = max_requests
        self.window_seconds = window_seconds

    async def is_allowed(self, key: str) -> bool:
        try:
            now = time.time()
            window_start = now - self.window_seconds
            redis_key = f"rl:{key}"

            pipe = self.redis.pipeline()
            pipe.zremrangebyscore(redis_key, 0, window_start)
            pipe.zcard(redis_key)
            pipe.zadd(redis_key, {str(now): now})
            pipe.expire(redis_key, self.window_seconds + 1)
            results = await pipe.execute()

            count = results[1]
            return count < self.max_requests
        except Exception:
            # Redis down — allow (fail-open for availability)
            return True


class RateLimitMiddleware(BaseHTTPMiddleware):
    """
    Middleware for rate limiting requests.
    
    This middleware limits the number of requests a client can make within a specified time window.
    """
    
    def __init__(
        self, 
        app,
        requests_limit: int = 100,
        window_size: int = 60,  # seconds
        exclude_paths: Optional[list] = None
    ):
        super().__init__(app)
        self.requests_limit = requests_limit
        self.window_size = window_size
        self.exclude_paths = exclude_paths or ["/docs", "/openapi.json", "/redoc", "/health"]
        self.requests = defaultdict(list)  # client_ip -> list of request timestamps
        self.lock = asyncio.Lock()
        logger.info(f"Initialized RateLimitMiddleware with limit: {requests_limit} requests per {window_size} seconds")

        self._redis_limiter = None
        if settings.REDIS_RATE_LIMIT_ENABLED and settings.REDIS_URL:
            self._redis_limiter = RedisRateLimiter(
                settings.REDIS_URL,
                max_requests=self.requests_limit,
                window_seconds=self.window_size,
            )
            logger.info("Redis rate limiter enabled: %s", settings.REDIS_URL)

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process the request, check rate limits, and pass it to the next middleware.
        
        Args:
            request: The incoming request
            call_next: The next middleware to call
            
        Returns:
            The response from the next middleware
        """
        # Skip rate limiting for excluded paths
        path = request.url.path
        if any(path.startswith(excluded) for excluded in self.exclude_paths):
            return await call_next(request)
            
        # Skip rate limiting for OPTIONS requests (CORS preflight)
        if request.method == "OPTIONS":
            return await call_next(request)
        
        # Get client IP
        client_ip = request.client.host

        # Redis rate limiting (preferred when configured)
        if self._redis_limiter:
            allowed = await self._redis_limiter.is_allowed(client_ip)
            if not allowed:
                logger.warning("Redis rate limit exceeded for IP: %s", client_ip)
                return JSONResponse(
                    status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                    content={"detail": "Rate limit exceeded. Please try again later."},
                    headers={"X-RateLimit-Limit": str(self.requests_limit)},
                )
            return await call_next(request)

        # Fallback: in-memory rate limiting
        current_time = time.time()

        async with self.lock:
            if client_ip in self.requests:
                # Remove old requests
                self.requests[client_ip] = [t for t in self.requests[client_ip] if current_time - t < self.window_size]
                
                # Check if limit exceeded
                if len(self.requests[client_ip]) >= self.requests_limit:
                    logger.warning(f"Rate limit exceeded for IP: {client_ip}")
                    return JSONResponse(
                        status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                        content={"detail": "Rate limit exceeded. Please try again later."}
                    )
                
                # Add current request
                self.requests[client_ip].append(current_time)
            else:
                # First request from this IP
                self.requests[client_ip] = [current_time]
        
        # Process the request
        return await call_next(request)
