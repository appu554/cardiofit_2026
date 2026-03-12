"""
Tests for the GraphQL API of the Patient Service.

This module provides tests for the GraphQL API of the Patient Service.
"""

import pytest
from fastapi.testclient import TestClient
import json
from app.main import app

client = TestClient(app)

def test_graphql_patient_query():
    """Test the patient query."""
    # Define the GraphQL query
    query = """
    query {
        patient(id: "test-patient-id") {
            id
            resourceType
            name {
                family
                given
                use
            }
            gender
            birthDate
            active
        }
    }
    """
    
    # Execute the query
    response = client.post(
        "/api/graphql",
        json={"query": query}
    )
    
    # Check the response
    assert response.status_code == 200
    data = response.json()
    assert "data" in data
    assert "patient" in data["data"]
    assert data["data"]["patient"] is not None
    assert data["data"]["patient"]["id"] == "test-patient-id"
    assert data["data"]["patient"]["resourceType"] == "Patient"

def test_graphql_patients_query():
    """Test the patients query."""
    # Define the GraphQL query
    query = """
    query {
        patients(page: 1, count: 10) {
            items {
                id
                resourceType
                name {
                    family
                    given
                    use
                }
                gender
                birthDate
                active
            }
            total
            page
            count
        }
    }
    """
    
    # Execute the query
    response = client.post(
        "/api/graphql",
        json={"query": query}
    )
    
    # Check the response
    assert response.status_code == 200
    data = response.json()
    assert "data" in data
    assert "patients" in data["data"]
    assert data["data"]["patients"] is not None
    assert "items" in data["data"]["patients"]
    assert "total" in data["data"]["patients"]
    assert "page" in data["data"]["patients"]
    assert "count" in data["data"]["patients"]
    assert data["data"]["patients"]["page"] == 1
    assert data["data"]["patients"]["count"] == 10

def test_graphql_create_patient_mutation():
    """Test the createPatient mutation."""
    # Define the GraphQL mutation
    mutation = """
    mutation {
        createPatient(input: {
            name: [{
                family: "Test",
                given: ["Patient"]
            }],
            gender: "male",
            birthDate: "1970-01-01"
        }) {
            patient {
                id
                resourceType
                name {
                    family
                    given
                }
                gender
                birthDate
            }
        }
    }
    """
    
    # Execute the mutation
    response = client.post(
        "/api/graphql",
        json={"query": mutation}
    )
    
    # Check the response
    assert response.status_code == 200
    data = response.json()
    assert "data" in data
    assert "createPatient" in data["data"]
    assert data["data"]["createPatient"] is not None
    assert "patient" in data["data"]["createPatient"]
    assert data["data"]["createPatient"]["patient"] is not None
    assert "id" in data["data"]["createPatient"]["patient"]
    assert data["data"]["createPatient"]["patient"]["resourceType"] == "Patient"
    assert data["data"]["createPatient"]["patient"]["gender"] == "male"
    assert data["data"]["createPatient"]["patient"]["birthDate"] == "1970-01-01"
    assert len(data["data"]["createPatient"]["patient"]["name"]) == 1
    assert data["data"]["createPatient"]["patient"]["name"][0]["family"] == "Test"
    assert data["data"]["createPatient"]["patient"]["name"][0]["given"] == ["Patient"]
