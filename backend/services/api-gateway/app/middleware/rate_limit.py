from fastapi import Request, Response, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from typing import Callable, Dict, List, Optional, Any
import logging
import time
import asyncio
from collections import defaultdict

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

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
        
        # Check if client has exceeded rate limit
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
