#!/usr/bin/env python
"""
Test the Auth Service.
This script tests if the Auth Service is running and can validate tokens.
"""

import requests
import sys
import json

def test_health():
    """Test the health endpoint."""
    try:
        response = requests.get("http://localhost:8001/api/health")
        if response.status_code == 200:
            print("✅ Health check passed")
            return True
        else:
            print(f"❌ Health check failed: {response.status_code} - {response.text}")
            return False
    except Exception as e:
        print(f"❌ Health check failed: {str(e)}")
        return False

def test_verify_token(token):
    """Test the token verification endpoint."""
    try:
        response = requests.post(
            "http://localhost:8001/api/auth/verify",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        print(f"Status code: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            print(f"Response: {json.dumps(result, indent=2)}")
            
            if result.get("valid", False):
                print("✅ Token verification passed")
                return True
            else:
                print(f"❌ Token verification failed: {result.get('error', 'Unknown error')}")
                return False
        else:
            print(f"❌ Token verification failed: {response.status_code} - {response.text}")
            return False
    except Exception as e:
        print(f"❌ Token verification failed: {str(e)}")
        return False

def main():
    """Main function."""
    print("Testing Auth Service...")
    
    # Test health endpoint
    if not test_health():
        print("Auth Service is not running or not accessible")
        sys.exit(1)
    
    # Get token from command line or prompt
    if len(sys.argv) > 1:
        token = sys.argv[1]
    else:
        token = input("Enter a JWT token to test: ")
    
    # Test token verification
    test_verify_token(token)

if __name__ == "__main__":
    main()
