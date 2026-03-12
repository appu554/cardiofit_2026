# Shared utilities for Clinical Synthesis Hub microservices

# Import key components for easy access
try:
    from .outbox_client import GlobalOutboxClient, publish_to_global_outbox
    __all__ = ['GlobalOutboxClient', 'publish_to_global_outbox']
except ImportError:
    # Graceful fallback if gRPC dependencies are not available
    __all__ = []