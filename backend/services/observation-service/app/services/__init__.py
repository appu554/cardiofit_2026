# Import the observation service
# Using lazy imports to prevent circular imports
# Re-export the async function
from .observation_service import get_observation_service

__all__ = ["get_observation_service"]
