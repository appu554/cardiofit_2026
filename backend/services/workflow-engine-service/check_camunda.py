import requests
import json
import sys
from datetime import datetime, timedelta

# Camunda Cloud credentials
CLIENT_ID = "zKn-MPzkpJzosRlJsL9ivKwZRvkX07D2"
CLIENT_SECRET = "nIG2l8I1pAM~Pa0LTHBM0X_sIVfoeTBticVodbM5Z1CF9bSz6KOHMsMzwANhoNCv"
CLUSTER_ID = "fe2ef9e5-11f2-4fe4-ba72-87b440bfe879"
REGION = "syd-1"

# API Endpoints
AUTH_URL = "https://login.cloud.camunda.io/oauth/token"
OPERATE_API = f"https://{REGION}.operate.camunda.io/{CLUSTER_ID}"
TASKLIST_API = f"https://{REGION}.tasklist.camunda.io/{CLUSTER_ID}"

class CamundaCloudClient:
    def __init__(self):
        self.token = None
        self.token_expiry = None

    def get_access_token(self):
        """Get OAuth2 token from Camunda Cloud"""
        try:
            # First, get the token using client credentials
            token_url = "https://login.cloud.camunda.io/oauth/token"
            
            # Encode client credentials for Basic Auth
            import base64
            credentials = f"{CLIENT_ID}:{CLIENT_SECRET}"
            encoded_credentials = base64.b64encode(credentials.encode()).decode()
            
            headers = {
                'Content-Type': 'application/x-www-form-urlencoded',
                'Authorization': f'Basic {encoded_credentials}'
            }
            
            # Prepare the form data
            data = {
                'grant_type': 'client_credentials',
                'audience': 'api.cloud.camunda.io'
            }
            
            print(f"🔑 Requesting token with client_id: {CLIENT_ID[:4]}...")
            print(f"🔑 Token URL: {token_url}")
            
            # Make the request with Basic Auth
            response = requests.post(
                token_url,
                headers=headers,
                data=data
            )
            
            print(f"🔑 Response status: {response.status_code}")
            
            if response.status_code != 200:
                print(f"❌ Failed to get access token: {response.status_code} - {response.text}")
                sys.exit(1)
                
            token_data = response.json()
            self.token = token_data.get("access_token")
            
            if not self.token:
                print("❌ No access token in response")
                print(f"Response: {token_data}")
                sys.exit(1)
                
            self.token_expiry = datetime.now() + timedelta(seconds=token_data.get("expires_in", 3600) - 60)
            print("✅ Successfully obtained access token")
            return self.token
            
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get access token: {str(e)}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"Response status: {e.response.status_code}")
                print(f"Response text: {e.response.text}")
            sys.exit(1)

    def ensure_token_valid(self):
        """Ensure we have a valid token"""
        if not self.token or datetime.now() >= self.token_expiry:
            self.get_access_token()

    def make_request(self, method, endpoint, **kwargs):
        """Make an authenticated request to Camunda Cloud"""
        self.ensure_token_valid()
        headers = {
            "Authorization": f"Bearer {self.token}",
            "Content-Type": "application/json"
        }
        if 'headers' in kwargs:
            headers.update(kwargs['headers'])
            del kwargs['headers']
            
        url = f"{OPERATE_API}/{endpoint.lstrip('/')}"
        
        try:
            response = requests.request(method, url, headers=headers, **kwargs)
            response.raise_for_status()
            return response.json() if response.text else {}
        except requests.exceptions.RequestException as e:
            print(f"❌ API Request failed: {str(e)}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"Response: {e.response.text}")
            return None

def check_workflow_instances(client):
    """Check workflow instances in Camunda"""
    print("\n🔍 Checking workflow instances...")
    
    # Get process instances
    now = datetime.utcnow().isoformat() + "Z"  # Current time in ISO format
    one_hour_ago = (datetime.utcnow() - timedelta(hours=1)).isoformat() + "Z"
    
    # Get recent process instances
    response = client.make_request(
        "POST",
        "/v1/process-instances/search",
        json={
            "filter": {
                "startDateAfter": one_hour_ago,
                "startDateBefore": now
            },
            "size": 10,
            "sort": [{"field": "startDate", "order": "DESC"}]
        }
    )
    
    if response and 'items' in response:
        print(f"✅ Found {len(response['items'])} recent workflow instances:")
        for instance in response['items']:
            print(f"   - ID: {instance.get('id')}")
            print(f"     Process: {instance.get('bpmnProcessId')}")
            print(f"     State: {instance.get('state')}")
            print(f"     Start Time: {instance.get('startDate')}\n")
    else:
        print("ℹ️ No recent workflow instances found.")

def check_operate_health(client):
    """Check Operate API health"""
    print("\n🏥 Checking Camunda Operate health...")
    try:
        response = client.make_request("GET", "/actuator/health")
        if response and response.get("status") == "UP":
            print("✅ Camunda Operate is healthy")
            print(f"   Status: {response.get('status')}")
            print(f"   Version: {response.get('version')}")
            return True
        else:
            print("❌ Camunda Operate is not healthy")
            print(f"   Response: {response}")
            return False
    except Exception as e:
        print(f"❌ Error checking Operate health: {str(e)}")
        return False

if __name__ == "__main__":
    print("🔍 Checking Camunda Cloud connection...")
    client = CamundaCloudClient()
    
    # First verify we can get a token
    token = client.get_access_token()
    if not token:
        print("❌ Failed to obtain access token. Please check your credentials.")
        sys.exit(1)
        
    print("🔑 Successfully obtained access token")
    
    # Check Operate health
    if check_operate_health(client):
        # Check workflow instances if Operate is healthy
        check_workflow_instances(client)
    else:
        print("\n⚠️  Cannot check workflow instances due to Operate health issues")
    token = get_access_token()
    
