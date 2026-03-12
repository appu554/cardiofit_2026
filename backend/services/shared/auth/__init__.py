# Authentication module for Clinical Synthesis Hub microservices

from .header_middleware import HeaderAuthMiddleware

__all__ = ["HeaderAuthMiddleware"]
