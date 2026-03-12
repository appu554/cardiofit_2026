#!/usr/bin/env python3
"""
Setup GitHub Repository Secrets for KB-7 Knowledge Factory
Uses GitHub API to encrypt and store secrets for GitHub Actions workflows
"""

import json
import base64
import requests
from nacl import encoding, public

# Configuration
GITHUB_TOKEN = "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"
REPO_OWNER = "onkarshahi-IND"
REPO_NAME = "knowledge-factory"
GITHUB_API = "https://api.github.com"

def get_public_key():
    """Get the repository's public key for secret encryption"""
    url = f"{GITHUB_API}/repos/{REPO_OWNER}/{REPO_NAME}/actions/secrets/public-key"
    headers = {
        "Authorization": f"token {GITHUB_TOKEN}",
        "Accept": "application/vnd.github.v3+json"
    }

    response = requests.get(url, headers=headers)
    response.raise_for_status()

    return response.json()

def encrypt_secret(public_key: str, secret_value: str) -> str:
    """Encrypt a secret using the repository's public key"""
    public_key_bytes = public.PublicKey(public_key.encode("utf-8"), encoding.Base64Encoder())
    sealed_box = public.SealedBox(public_key_bytes)
    encrypted = sealed_box.encrypt(secret_value.encode("utf-8"))
    return base64.b64encode(encrypted).decode("utf-8")

def create_or_update_secret(secret_name: str, secret_value: str, key_id: str, public_key: str):
    """Create or update a repository secret"""
    url = f"{GITHUB_API}/repos/{REPO_OWNER}/{REPO_NAME}/actions/secrets/{secret_name}"
    headers = {
        "Authorization": f"token {GITHUB_TOKEN}",
        "Accept": "application/vnd.github.v3+json"
    }

    encrypted_value = encrypt_secret(public_key, secret_value)

    data = {
        "encrypted_value": encrypted_value,
        "key_id": key_id
    }

    response = requests.put(url, headers=headers, json=data)
    response.raise_for_status()

    return response.status_code

def main():
    print("=" * 70)
    print("KB-7 Knowledge Factory - GitHub Secrets Setup")
    print("=" * 70)
    print()

    # Get public key
    print("📝 Step 1: Getting repository public key...")
    try:
        key_data = get_public_key()
        key_id = key_data['key_id']
        public_key = key_data['key']
        print(f"✅ Public key retrieved (Key ID: {key_id})")
        print()
    except Exception as e:
        print(f"❌ Failed to get public key: {e}")
        return 1

    # Read GCS service account key
    print("📝 Step 2: Reading GCS service account key...")
    try:
        with open('kb7-github-actions-key.json', 'r') as f:
            gcs_key_content = f.read()
        print("✅ GCS service account key loaded")
        print()
    except Exception as e:
        print(f"❌ Failed to read service account key: {e}")
        return 1

    # Define secrets
    secrets = {
        "GCS_SERVICE_ACCOUNT_KEY": gcs_key_content,
        "GRAPHDB_URL": "http://host.docker.internal:7200",  # Docker host reference
        "GRAPHDB_CREDENTIALS": json.dumps({
            "username": "admin",
            "password": ""  # GraphDB default has no password initially
        }),
        "GCS_BUCKET": "sincere-hybrid-477206-h2-kb-sources-production",
        "PROJECT_ID": "sincere-hybrid-477206-h2"
    }

    # Create/update each secret
    print("📝 Step 3: Creating/updating repository secrets...")
    print()

    for secret_name, secret_value in secrets.items():
        try:
            print(f"  ⏳ Setting {secret_name}...", end=" ")
            status = create_or_update_secret(secret_name, secret_value, key_id, public_key)
            if status in [201, 204]:
                print("✅")
            else:
                print(f"⚠️  Status: {status}")
        except Exception as e:
            print(f"❌ Error: {e}")
            return 1

    print()
    print("=" * 70)
    print("✅ All secrets configured successfully!")
    print("=" * 70)
    print()
    print("Configured secrets:")
    for secret_name in secrets.keys():
        print(f"  • {secret_name}")
    print()
    print("Next steps:")
    print("  1. Verify secrets in GitHub repository settings")
    print("  2. Test GitHub Actions workflow execution")
    print("  3. Monitor workflow logs for successful GCS access")
    print()

    return 0

if __name__ == "__main__":
    exit(main())
