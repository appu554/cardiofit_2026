"""Body and response transformers for gateway endpoint translation.

These bridge the gap between product-level URLs (Patient App, Doctor Dashboard)
and internal KB service contracts.
"""


def checkin_to_session(patient_id: str, body: dict) -> dict:
    """Transform Patient App checkin → KB-22 CreateSessionRequest.

    Maps symptom keywords to HPI node IDs. Defaults to P01 (Chest Pain)
    for unmapped symptoms so the flow is never blocked.
    """
    symptom = body.get("symptom", "")
    node_map = {
        "chest_pain": "P01_CHEST_PAIN",
        "breathlessness": "P02_DYSPNEA",
        "palpitations": "P03_PALPITATIONS",
    }
    return {
        "patient_id": patient_id,
        "node_id": node_map.get(symptom, "P01_CHEST_PAIN"),
    }


def extract_health_score(kb26_mri_response: dict) -> dict:
    """Extract simplified health score from KB-26 MRI response.

    KB-26 returns a rich MRI payload; the Patient App needs a simplified view.
    """
    data = kb26_mri_response.get("data", kb26_mri_response)
    return {
        "score": data.get("mri_score") or data.get("composite_score"),
        "trend": data.get("trend"),
        "components": data.get("decomposition", {}),
    }
