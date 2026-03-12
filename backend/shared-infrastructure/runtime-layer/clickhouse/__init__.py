"""
ClickHouse Integration for Runtime Layer

Consolidated ClickHouse components:
- analytics/: Pre-computed analytics and scoring
- runtime/: Real-time query management and caching
"""

from .analytics.multi_kb_analytics import *
from .runtime.manager import *

__version__ = "1.0.0"