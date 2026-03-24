"""Per-service circuit breaker with closed/open/half-open states."""
import logging
import time
from typing import Optional

logger = logging.getLogger(__name__)


class ServiceCircuitBreaker:
    """Simple circuit breaker per downstream service."""

    def __init__(self, service_name: str, fail_max: int = 5, reset_timeout: int = 30):
        self.service_name = service_name
        self.fail_max = fail_max
        self.reset_timeout = reset_timeout
        self._failures = 0
        self._last_failure: float = 0
        self._state = "closed"  # closed, open, half-open

    def is_available(self) -> bool:
        if self._state == "closed":
            return True
        if self._state == "open":
            if time.time() - self._last_failure > self.reset_timeout:
                self._state = "half-open"
                return True
            return False
        return True  # half-open allows one request through

    def record_success(self):
        self._failures = 0
        self._state = "closed"

    def record_failure(self):
        self._failures += 1
        self._last_failure = time.time()
        if self._failures >= self.fail_max:
            self._state = "open"
            logger.warning("Circuit OPEN for %s after %d failures", self.service_name, self._failures)

    @property
    def state(self) -> str:
        return self._state


# Registry of circuit breakers per service
_breakers: dict[str, ServiceCircuitBreaker] = {}


def get_breaker(service_name: str, fail_max: int = 5, reset_timeout: int = 30) -> ServiceCircuitBreaker:
    if service_name not in _breakers:
        _breakers[service_name] = ServiceCircuitBreaker(service_name, fail_max, reset_timeout)
    return _breakers[service_name]
