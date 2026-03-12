#!/usr/bin/env python3
"""
Setup personal Google authentication for BigQuery access.
This will open a browser window for you to authorize access.
"""

from google_auth_oauthlib import flow
from pathlib import Path
import json

# OAuth 2.0 configuration for installed applications
CLIENT_CONFIG = {
    "installed": {
        "client_id": "...",  # This would need to be created in GCP Console
        "project_id": "sincere-hybrid-477206-h2",
        "auth_uri": "https://accounts.google.com/o/oauth2/auth",
        "token_uri": "https://oauth2.googleapis.com/token",
        "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
        "redirect_uris": ["http://localhost"]
    }
}

# This approach requires OAuth client ID from GCP Console
# which is more complex than using gcloud...

print("❌ This approach requires OAuth client ID setup in GCP Console.")
print()
print("SIMPLER SOLUTION: Use service account with PhysioNet project linking")
print()
print("Since your personal account works in BigQuery console,")
print("the best path forward is:")
print()
print("1. Contact PhysioNet support at: https://physionet.org/about/contact/")
print("2. Request that your GCP project be added: sincere-hybrid-477206-h2")
print("3. Reference your approved email: onkarshahi@vaidshala.com")
print()
print("Or wait - PhysioNet might auto-approve project access shortly.")
print("Check again in 24 hours.")
