import requests
import json

# Get Auth0 token
auth_response = requests.post(
    "https://dev-hfw6wda5wtf8l13c.au.auth0.com/oauth/token",
    headers={"Content-Type": "application/json"},
    json={
        "grant_type": "client_credentials",
        "client_id": "qysYE1GswykrgR7OmfrSw475cBPjVRxl",
        "client_secret": "ernWK2y8VoAAMXpFJiBRydURRE-kU3DqXtfU29NYBTbEIGEkpyNFNTb4rQiZMZgk",
        "audience": "https://clinical-synthesis-hub-api"
    }
)

token_data = auth_response.json()
access_token = token_data.get("access_token")

if not access_token:
    print("Failed to get access token")
    print(token_data)
    exit(1)

print(f"Got access token: {access_token[:20]}...")

# Test FHIR service directly
fhir_response = requests.get(
    "http://localhost:8004/api/fhir/Patient",
    headers={"Authorization": f"Bearer {access_token}"}
)

print(f"FHIR service response status: {fhir_response.status_code}")
if fhir_response.status_code == 200:
    patients = fhir_response.json()
    print(f"Found {len(patients)} patients")
    if patients:
        print(f"First patient: {json.dumps(patients[0], indent=2)}")
else:
    print(f"FHIR service error: {fhir_response.text}")
