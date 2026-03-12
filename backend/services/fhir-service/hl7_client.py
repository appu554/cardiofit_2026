import argparse
import requests
import json
import os
from pathlib import Path

def send_hl7_message(file_path, api_url, token=None):
    """
    Send an HL7 message to the API.
    
    Args:
        file_path: Path to the HL7 message file
        api_url: URL of the HL7 API endpoint
        token: Optional authentication token
    """
    # Load the HL7 message from file
    with open(file_path, "r") as f:
        message_str = f.read()
    
    # Set up the request payload
    payload = {
        "message": message_str
    }
    
    # Set up headers
    headers = {
        "Content-Type": "application/json"
    }
    
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
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
            print("\nMessage processed successfully!")
            return True
        else:
            print(f"Error: {response.status_code}")
            print(response.text)
            return False
    except Exception as e:
        print(f"Error making API request: {str(e)}")
        return False

def main():
    # Parse command line arguments
    parser = argparse.ArgumentParser(description="Send HL7 messages to the API")
    parser.add_argument("file", help="Path to the HL7 message file")
    parser.add_argument("--url", default="http://localhost:8004/api/hl7/process", help="URL of the HL7 API endpoint")
    parser.add_argument("--token", help="Authentication token")
    
    args = parser.parse_args()
    
    # Send the HL7 message
    send_hl7_message(args.file, args.url, args.token)

if __name__ == "__main__":
    main()
