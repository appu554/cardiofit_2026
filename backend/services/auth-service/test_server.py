import requests
import sys
import time

def test_server():
    """Test if the server is running and responding to requests."""
    print("Testing server...")

    # Test the test endpoint
    try:
        response = requests.get("http://127.0.0.1:8000/api/auth/test")
        print(f"Test endpoint response: {response.status_code} - {response.text}")
        if response.status_code == 200:
            print("Server is working!")
            return True
    except Exception as e:
        print(f"Error testing server: {str(e)}")

    print("Server is not responding.")
    return False

def main():
    """Main function."""
    # Wait for the server to start
    print("Waiting for server to start...")
    for i in range(5):
        if test_server():
            break
        print(f"Retrying in 2 seconds... ({i+1}/5)")
        time.sleep(2)

    # Get a client token
    print("\nGetting client token...")
    try:
        response = requests.post(
            "https://dev-hfw6wda5wtf8l13c.au.auth0.com/oauth/token",
            json={
                "grant_type": "client_credentials",
                "client_id": "qysYE1GswykrgR7OmfrSw475cBPjVRxl",
                "client_secret": "ernWK2y8VoAAMXpFJiBRydURRE-kU3DqXtfU29NYBTbEIGEkpyNFNTb4rQiZMZgk",
                "audience": "https://clinical-synthesis-hub-api"
            }
        )

        if response.status_code != 200:
            print(f"Error getting token: {response.status_code} - {response.text}")
            return

        token = response.json()["access_token"]
        print(f"Token obtained: {token[:20]}...")

        # Create a user
        print("\nCreating user...")
        user_response = requests.post(
            "http://127.0.0.1:8000/api/auth/users",
            headers={"Authorization": f"Bearer {token}"},
            json={
                "email": "test.user5@example.com",  # Use a different email each time
                "password": "StrongP@ssw0rd",
                "full_name": "Test User 5",
                "role": "doctor",
                "permissions": ["read:patients", "write:notes"],
                "metadata": {"department": "Cardiology"}
            }
        )

        print(f"Create user response: {user_response.status_code}")
        print(f"Response headers: {user_response.headers}")
        print(f"Response body: {user_response.text}")

    except Exception as e:
        print(f"Error: {str(e)}")

if __name__ == "__main__":
    main()
