import pytest
from app.middleware.circuit_breaker import ServiceCircuitBreaker


def test_breaker_starts_closed():
    cb = ServiceCircuitBreaker("test-svc", fail_max=3, reset_timeout=5)
    assert cb.is_available()


def test_breaker_opens_after_failures():
    cb = ServiceCircuitBreaker("test-svc", fail_max=3, reset_timeout=5)
    cb.record_failure()
    cb.record_failure()
    cb.record_failure()
    assert not cb.is_available()


def test_success_resets_count():
    cb = ServiceCircuitBreaker("test-svc", fail_max=3, reset_timeout=5)
    cb.record_failure()
    cb.record_failure()
    cb.record_success()
    assert cb.is_available()
