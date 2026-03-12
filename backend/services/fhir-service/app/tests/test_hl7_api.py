import requests
import json
import os
from pathlib import Path

# Get the current directory
current_dir = Path(__file__).parent

def test_hl7_api():
    """Test the HL7 API endpoint"""
    # Load sample ADT message
    sample_file = current_dir / "sample_adt_message.hl7"
    with open(sample_file, "r") as f:
        message_str = f.read()
    
    # Set up the API endpoint URL
    api_url = "http://localhost:8004/api/hl7/process"
    
    # Set up the request payload
    payload = {
        "message": message_str
    }
    
    # Set up headers (you would need a valid token in a real scenario)
    headers = {
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_TOKEN_HERE"
    }
    
    print(f"Sending HL7 message to {api_url}")
    print(f"Message: {message_str[:100]}...")
    
    # Make the API request
    try:
        response = requests.post(api_url, json=payload, headers=headers)
        
        # Check the response
        if response.status_code == 200:
            result = response.json()
            print("API Response:")
            print(json.dumps(result, indent=2))
            print("\nTest completed successfully!")
        else:
            print(f"Error: {response.status_code}")
            print(response.text)
    except Exception as e:
        print(f"Error making API request: {str(e)}")

if __name__ == "__main__":
    # Run the test
    test_hl7_api()
