import requests

# Configuration
BASE_URL = "http://localhost:8008/api"
TEST_TOKEN = "test_token"  # Any token will work with our placeholder auth

# Headers
headers = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {TEST_TOKEN}"
}

def test_create_medication():
    """Test creating a medication"""
    payload = {
        "code": {
            "coding": [
                {
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "code": "1049502",
                    "display": "Acetaminophen 325 MG Oral Tablet"
                }
            ],
            "text": "Acetaminophen 325 MG Oral Tablet"
        }
    }

    print(f"Sending POST request to {BASE_URL}/medications")
    print(f"Headers: {headers}")
    print(f"Payload: {payload}")
    
    response = requests.post(f"{BASE_URL}/medications", headers=headers, json=payload)
    print(f"Response status code: {response.status_code}")
    print(f"Response content: {response.text}")
    
    if response.status_code == 201:
        data = response.json()
        print(f"✅ Created medication with ID: {data.get('id')}")
        return data.get('id')
    else:
        print(f"❌ Failed to create medication: {response.text}")
        return None

if __name__ == "__main__":
    print("Starting simple medication service test...")
    medication_id = test_create_medication()
    if medication_id:
        print(f"Test passed! Created medication with ID: {medication_id}")
    else:
        print("Test failed!")
