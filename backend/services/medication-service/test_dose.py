#!/usr/bin/env python3
import asyncio
import httpx
import json

async def test_dose_calculation():
    test_request = {
        "patient_id": "test-patient-123",
        "medication_code": "vancomycin",
        "indication": "pneumonia",
        "calculation_type": "weight_based",
        "patient_context": {
            "weight_kg": 70,
            "age_years": 45,
            "creatinine_clearance": 80
        },
        "dosing_parameters": {
            "dose_per_kg": 15
        }
    }
    
    async with httpx.AsyncClient() as client:
        response = await client.post(
            "http://localhost:8009/api/dose-calculation/calculate",
            json=test_request
        )
        data = response.json()
        print(f"Status: {response.status_code}")
        print(f"Response: {json.dumps(data, indent=2)}")

if __name__ == "__main__":
    asyncio.run(test_dose_calculation())
