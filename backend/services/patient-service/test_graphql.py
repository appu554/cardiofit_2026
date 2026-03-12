"""
Test script for GraphQL integration with Google Cloud Healthcare API.

This script sends GraphQL queries to the API Gateway to test the
Patient service integration with Google Cloud Healthcare API.
"""

import requests
import json
import argparse
import sys

def test_create_patient(api_gateway_url, auth_token=None):
    """
    Test creating a patient via GraphQL.
    
    Args:
        api_gateway_url: URL of the API Gateway GraphQL endpoint
        auth_token: Optional authentication token
    """
    # GraphQL mutation to create a patient
    mutation = """
    mutation CreatePatient($input: PatientInput!) {
        createPatient(input: $input) {
            id
            resourceType
            name {
                family
                given
            }
            gender
            birthDate
            telecom {
                system
                value
                use
            }
            address {
                line
                city
                state
                postalCode
                country
            }
        }
    }
    """
    
    # Variables for the mutation
    variables = {
        "input": {
            "resourceType": "Patient",
            "name": [
                {
                    "family": "Smith",
                    "given": ["John"]
                }
            ],
            "gender": "male",
            "birthDate": "1970-01-01",
            "telecom": [
                {
                    "system": "phone",
                    "value": "555-123-4567",
                    "use": "home"
                },
                {
                    "system": "email",
                    "value": "john.smith@example.com",
                    "use": "work"
                }
            ],
            "address": [
                {
                    "line": ["123 Main St"],
                    "city": "Anytown",
                    "state": "CA",
                    "postalCode": "12345",
                    "country": "USA"
                }
            ]
        }
    }
    
    # Headers
    headers = {
        "Content-Type": "application/json"
    }
    
    if auth_token:
        headers["Authorization"] = f"Bearer {auth_token}"
    
    # Send the request
    response = requests.post(
        api_gateway_url,
        headers=headers,
        json={"query": mutation, "variables": variables}
    )
    
    # Print the response
    print("\n=== CREATE PATIENT RESPONSE ===")
    print(f"Status Code: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2))
        
        # Extract the patient ID for later use
        if "data" in result and "createPatient" in result["data"]:
            patient_id = result["data"]["createPatient"]["id"]
            print(f"\nCreated Patient ID: {patient_id}")
            return patient_id
        else:
            print("\nFailed to extract patient ID from response")
            return None
    except Exception as e:
        print(f"Error parsing response: {str(e)}")
        print(response.text)
        return None

def test_get_patient(api_gateway_url, patient_id, auth_token=None):
    """
    Test getting a patient via GraphQL.
    
    Args:
        api_gateway_url: URL of the API Gateway GraphQL endpoint
        patient_id: ID of the patient to retrieve
        auth_token: Optional authentication token
    """
    # GraphQL query to get a patient
    query = """
    query GetPatient($id: ID!) {
        patient(id: $id) {
            id
            resourceType
            name {
                family
                given
            }
            gender
            birthDate
            telecom {
                system
                value
                use
            }
            address {
                line
                city
                state
                postalCode
                country
            }
        }
    }
    """
    
    # Variables for the query
    variables = {
        "id": patient_id
    }
    
    # Headers
    headers = {
        "Content-Type": "application/json"
    }
    
    if auth_token:
        headers["Authorization"] = f"Bearer {auth_token}"
    
    # Send the request
    response = requests.post(
        api_gateway_url,
        headers=headers,
        json={"query": query, "variables": variables}
    )
    
    # Print the response
    print("\n=== GET PATIENT RESPONSE ===")
    print(f"Status Code: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2))
    except Exception as e:
        print(f"Error parsing response: {str(e)}")
        print(response.text)

def test_search_patients(api_gateway_url, auth_token=None):
    """
    Test searching for patients via GraphQL.
    
    Args:
        api_gateway_url: URL of the API Gateway GraphQL endpoint
        auth_token: Optional authentication token
    """
    # GraphQL query to search for patients
    query = """
    query SearchPatients($name: String) {
        patients(name: $name) {
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
    """
    
    # Variables for the query
    variables = {
        "name": "Smith"
    }
    
    # Headers
    headers = {
        "Content-Type": "application/json"
    }
    
    if auth_token:
        headers["Authorization"] = f"Bearer {auth_token}"
    
    # Send the request
    response = requests.post(
        api_gateway_url,
        headers=headers,
        json={"query": query, "variables": variables}
    )
    
    # Print the response
    print("\n=== SEARCH PATIENTS RESPONSE ===")
    print(f"Status Code: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2))
    except Exception as e:
        print(f"Error parsing response: {str(e)}")
        print(response.text)

def main():
    """Main function."""
    parser = argparse.ArgumentParser(description='Test GraphQL integration with Google Cloud Healthcare API')
    parser.add_argument('--api-gateway-url', default='http://localhost:8000/api/graphql', help='URL of the API Gateway GraphQL endpoint')
    parser.add_argument('--auth-token', help='Authentication token')
    
    args = parser.parse_args()
    
    print(f"Testing GraphQL integration with API Gateway at {args.api_gateway_url}")
    
    # Test creating a patient
    patient_id = test_create_patient(args.api_gateway_url, args.auth_token)
    
    if patient_id:
        # Test getting the patient
        test_get_patient(args.api_gateway_url, patient_id, args.auth_token)
    
    # Test searching for patients
    test_search_patients(args.api_gateway_url, args.auth_token)

if __name__ == '__main__':
    main()
