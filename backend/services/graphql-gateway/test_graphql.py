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

# Print the raw response
print("\nRaw FHIR response:")
print(fhir_response.text)

print(f"FHIR service response status: {fhir_response.status_code}")
if fhir_response.status_code == 200:
    patients = fhir_response.json()
    print(f"Found {len(patients)} patients")
    if patients:
        print(f"First patient: {json.dumps(patients[0], indent=2)}")
        # Print all keys in the patient object
        print(f"Patient keys: {list(patients[0].keys())}")
else:
    print(f"FHIR service error: {fhir_response.text}")

# Create a GraphQL query to test
graphql_query = """
query {
  searchPatients {
    id
    name {
      family
      given
    }
    gender
    birthDate
  }
}
"""

# Now let's try the GraphQL endpoint directly
graphql_response = requests.post(
    "http://localhost:8006/graphql",
    headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer {access_token}"
    },
    json={"query": graphql_query}
)

print(f"\nGraphQL response status: {graphql_response.status_code}")
if graphql_response.status_code == 200:
    graphql_data = graphql_response.json()
    print(f"GraphQL response: {json.dumps(graphql_data, indent=2)}")

    # Print the data part of the response
    if graphql_data.get("data") and graphql_data["data"].get("searchPatients"):
        patients = graphql_data["data"]["searchPatients"]
        print(f"\nFound {len(patients)} patients in GraphQL response")
        for i, patient in enumerate(patients):
            print(f"\nPatient {i+1}:")
            print(f"ID: {patient.get('id')}")
            print(f"Name: {patient.get('name')}")
    else:
        print("\nNo patients found in GraphQL response")
else:
    print(f"GraphQL error: {graphql_response.text}")



# Simulate GraphQL query processing

# This would be used if the GraphQL server was running
# graphql_response = requests.post(
#     "http://localhost:8005/graphql",
#     headers={
#         "Content-Type": "application/json",
#         "Authorization": f"Bearer {access_token}"
#     },
#     json={"query": graphql_query}
# )
#
# print(f"GraphQL response status: {graphql_response.status_code}")
# if graphql_response.status_code == 200:
#     graphql_data = graphql_response.json()
#     print(f"GraphQL response: {json.dumps(graphql_data, indent=2)}")
# else:
#     print(f"GraphQL error: {graphql_response.text}")

# Since the GraphQL server is not running, let's simulate what it would do
# by directly calling the FHIR service and transforming the data
print("\nSimulating GraphQL query:")
print("Query: searchPatients")

# This is what the GraphQL resolver would do
patients_data = fhir_response.json()
graphql_patients = []
for patient in patients_data:
    try:
        graphql_patient = {
            "id": patient.get("id", ""),
            "name": patient.get("name", []),
            "gender": patient.get("gender"),
            "birthDate": patient.get("birthDate")
        }
        graphql_patients.append(graphql_patient)
    except Exception as e:
        print(f"Error processing patient: {e}")
        print(f"Patient data: {json.dumps(patient, indent=2)}")


simulated_response = {
    "data": {
        "searchPatients": graphql_patients
    }
}

print(f"Simulated GraphQL response: {json.dumps(simulated_response, indent=2)}")
