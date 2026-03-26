"""Tests for gateway body/response transformers."""
import pytest
from app.api.transforms import checkin_to_session, extract_health_score


class TestCheckinToSession:
    def test_known_symptom(self):
        result = checkin_to_session("91-1001", {"symptom": "chest_pain"})
        assert result == {
            "patient_id": "91-1001",
            "node_id": "P01_CHEST_PAIN",
        }

    def test_breathlessness(self):
        result = checkin_to_session("91-1001", {"symptom": "breathlessness"})
        assert result == {
            "patient_id": "91-1001",
            "node_id": "P02_DYSPNEA",
        }

    def test_palpitations(self):
        result = checkin_to_session("91-1001", {"symptom": "palpitations"})
        assert result == {
            "patient_id": "91-1001",
            "node_id": "P03_PALPITATIONS",
        }

    def test_unknown_symptom_defaults_to_p01(self):
        result = checkin_to_session("91-1001", {"symptom": "headache"})
        assert result["node_id"] == "P01_CHEST_PAIN"

    def test_missing_symptom_defaults_to_p01(self):
        result = checkin_to_session("91-1001", {})
        assert result["node_id"] == "P01_CHEST_PAIN"


class TestExtractHealthScore:
    def test_normal_response(self):
        kb26_resp = {
            "data": {
                "mri_score": 72.5,
                "trend": "improving",
                "decomposition": {"sbp": 0.3, "hba1c": 0.4},
            }
        }
        result = extract_health_score(kb26_resp)
        assert result["score"] == 72.5
        assert result["trend"] == "improving"
        assert result["components"]["sbp"] == 0.3

    def test_flat_response(self):
        """KB-26 may return data at top level."""
        kb26_resp = {
            "composite_score": 65.0,
            "trend": "stable",
            "decomposition": {},
        }
        result = extract_health_score(kb26_resp)
        assert result["score"] == 65.0

    def test_missing_fields(self):
        result = extract_health_score({})
        assert result["score"] is None
        assert result["trend"] is None
