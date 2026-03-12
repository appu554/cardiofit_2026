"""Mock Google API exceptions for local testing"""


class NotFound(Exception):
    """Resource not found"""
    pass


class ServiceUnavailable(Exception):
    """Service temporarily unavailable"""
    pass


class DeadlineExceeded(Exception):
    """Request deadline exceeded"""
    pass


class ResourceExhausted(Exception):
    """Resource quota exhausted"""
    pass


class InternalServerError(Exception):
    """Internal server error"""
    pass
