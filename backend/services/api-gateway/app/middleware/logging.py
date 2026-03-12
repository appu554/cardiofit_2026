from fastapi import Request, Response
from starlette.middleware.base import BaseHTTPMiddleware
from typing import Callable, Optional
import logging
import time
import uuid
import json

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class RequestLoggingMiddleware(BaseHTTPMiddleware):
    """
    Middleware for logging request and response details.
    
    This middleware logs information about incoming requests and outgoing responses,
    including timing information, status codes, and user information when available.
    """
    
    def __init__(
        self, 
        app,
        log_request_body: bool = False,
        log_response_body: bool = False
    ):
        super().__init__(app)
        self.log_request_body = log_request_body
        self.log_response_body = log_response_body
        logger.info(f"Initialized RequestLoggingMiddleware")
        
    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process the request, log details, and pass it to the next middleware.
        
        Args:
            request: The incoming request
            call_next: The next middleware to call
            
        Returns:
            The response from the next middleware
        """
        # Generate a unique request ID
        request_id = str(uuid.uuid4())
        
        # Add request ID to the request state
        request.state.request_id = request_id
        
        # Get the start time
        start_time = time.time()
        
        # Extract user information if available
        user_id = None
        user_role = None
        if hasattr(request.state, 'user'):
            user_id = request.state.user.get('id', None)
            user_role = request.state.user_role if hasattr(request.state, 'user_role') else None
        
        # Log the request
        log_data = {
            "request_id": request_id,
            "method": request.method,
            "path": request.url.path,
            "query_params": str(request.query_params),
            "client_ip": request.client.host,
            "user_id": user_id,
            "user_role": user_role,
            "timestamp": time.strftime("%Y-%m-%d %H:%M:%S", time.localtime())
        }
        
        # Log request headers if in debug mode
        if logger.level <= logging.DEBUG:
            log_data["headers"] = dict(request.headers)
        
        # Log request body if enabled
        if self.log_request_body:
            try:
                body = await request.body()
                if body:
                    try:
                        # Try to parse as JSON
                        body_json = json.loads(body)
                        log_data["body"] = body_json
                    except json.JSONDecodeError:
                        # If not JSON, log as string (truncated if too long)
                        body_str = body.decode('utf-8', errors='replace')
                        if len(body_str) > 1000:
                            body_str = body_str[:1000] + "... [truncated]"
                        log_data["body"] = body_str
            except Exception as e:
                log_data["body_error"] = str(e)
        
        logger.info(f"Request: {json.dumps(log_data)}")
        
        # Process the request
        try:
            response = await call_next(request)
            
            # Calculate request processing time
            process_time = time.time() - start_time
            
            # Log the response
            response_log = {
                "request_id": request_id,
                "status_code": response.status_code,
                "process_time_ms": round(process_time * 1000, 2),
                "timestamp": time.strftime("%Y-%m-%d %H:%M:%S", time.localtime())
            }
            
            # Log response headers if in debug mode
            if logger.level <= logging.DEBUG:
                response_log["headers"] = dict(response.headers)
            
            # Log response body if enabled
            if self.log_response_body and hasattr(response, "body"):
                try:
                    body = response.body
                    if body:
                        try:
                            # Try to parse as JSON
                            body_json = json.loads(body)
                            response_log["body"] = body_json
                        except json.JSONDecodeError:
                            # If not JSON, log as string (truncated if too long)
                            body_str = body.decode('utf-8', errors='replace')
                            if len(body_str) > 1000:
                                body_str = body_str[:1000] + "... [truncated]"
                            response_log["body"] = body_str
                except Exception as e:
                    response_log["body_error"] = str(e)
            
            logger.info(f"Response: {json.dumps(response_log)}")
            
            # Add the request ID to the response headers
            response.headers["X-Request-ID"] = request_id
            
            return response
        except Exception as e:
            # Log the exception
            logger.error(f"Exception during request processing: {str(e)}", exc_info=True)
            
            # Calculate request processing time
            process_time = time.time() - start_time
            
            # Log the error
            error_log = {
                "request_id": request_id,
                "error": str(e),
                "process_time_ms": round(process_time * 1000, 2),
                "timestamp": time.strftime("%Y-%m-%d %H:%M:%S", time.localtime())
            }
            
            logger.error(f"Error: {json.dumps(error_log)}")
            
            # Re-raise the exception
            raise
