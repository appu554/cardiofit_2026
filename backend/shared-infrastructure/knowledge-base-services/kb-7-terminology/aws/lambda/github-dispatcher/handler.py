"""
KB-7 Knowledge Factory - GitHub Dispatcher Lambda
Triggers GitHub Actions workflow via repository_dispatch event

Features:
- GitHub API v3 authentication with PAT
- repository_dispatch event creation
- Payload security (no secrets passed - GitHub uses its own Secrets store)
- Retry logic for API failures
"""

import os
import json
import boto3
import requests
from datetime import datetime
from botocore.exceptions import ClientError

# AWS clients
secrets_client = boto3.client('secretsmanager')
cloudwatch_client = boto3.client('cloudwatch')

# Environment variables
GITHUB_SECRET_NAME = os.environ['GITHUB_SECRET_NAME']
GITHUB_REPO_OWNER = os.environ['GITHUB_REPO_OWNER']
GITHUB_REPO_NAME = os.environ['GITHUB_REPO_NAME']
BUCKET_NAME = os.environ['BUCKET_NAME']
ENVIRONMENT = os.environ['ENVIRONMENT']


def lambda_handler(event, context):
    """
    Main Lambda handler for GitHub workflow dispatch

    Expected event structure:
    {
        "downloads": [
            {
                "snomed": {
                    "statusCode": 200,
                    "s3_key": "snomed-ct/20250131/SnomedCT_international_20250131.zip",
                    "checksum": "abc123..."
                }
            },
            {
                "rxnorm": {
                    "statusCode": 200,
                    "s3_key": "rxnorm/01042025/RxNorm_full_01042025.zip",
                    "checksum": "def456..."
                }
            },
            {
                "loinc": {
                    "statusCode": 200,
                    "s3_key": "loinc/2.77/LOINC_2.77.csv.zip",
                    "checksum": "ghi789..."
                }
            }
        ]
    }
    """
    try:
        print(f"[KB7-GITHUB] Dispatching Knowledge Factory workflow - Environment: {ENVIRONMENT}")

        # Get GitHub PAT from Secrets Manager
        credentials = get_credentials()
        github_token = credentials['token']
        api_endpoint = credentials.get('api_endpoint', 'https://api.github.com')

        # Extract download results from event
        downloads = event.get('downloads', [])
        snomed_data = downloads[0].get('snomed', {})
        rxnorm_data = downloads[1].get('rxnorm', {})
        loinc_data = downloads[2].get('loinc', {})

        # Construct dispatch payload
        # IMPORTANT: Do NOT pass secrets in payload - GitHub Actions uses GitHub Secrets
        payload = {
            'event_type': 'terminology_downloaded',
            'client_payload': {
                'trigger_source': 'aws-lambda',
                'environment': ENVIRONMENT,
                'timestamp': datetime.utcnow().isoformat(),
                's3_bucket': BUCKET_NAME,
                'sources': {
                    'snomed': {
                        's3_key': snomed_data.get('s3_key'),
                        'checksum': snomed_data.get('checksum')
                    },
                    'rxnorm': {
                        's3_key': rxnorm_data.get('s3_key'),
                        'checksum': rxnorm_data.get('checksum')
                    },
                    'loinc': {
                        's3_key': loinc_data.get('s3_key'),
                        'checksum': loinc_data.get('checksum')
                    }
                }
            }
        }

        # Dispatch GitHub workflow
        dispatch_url = f"{api_endpoint}/repos/{GITHUB_REPO_OWNER}/{GITHUB_REPO_NAME}/dispatches"
        response = dispatch_github_workflow(dispatch_url, github_token, payload)

        # Publish metrics
        publish_metrics(success=True)

        print(f"[KB7-GITHUB] Workflow dispatched successfully")

        return {
            'statusCode': 200,
            'body': json.dumps({
                'message': 'GitHub workflow dispatched successfully',
                'repository': f"{GITHUB_REPO_OWNER}/{GITHUB_REPO_NAME}",
                'event_type': 'terminology_downloaded',
                'dispatch_time': datetime.utcnow().isoformat()
            })
        }

    except Exception as e:
        print(f"[KB7-GITHUB] ERROR: {str(e)}")

        # Publish failure metric
        publish_metrics(success=False)

        return {
            'statusCode': 500,
            'body': json.dumps({
                'error': str(e),
                'message': 'GitHub workflow dispatch failed'
            })
        }


def get_credentials():
    """Retrieve GitHub PAT from Secrets Manager"""
    try:
        response = secrets_client.get_secret_value(SecretId=GITHUB_SECRET_NAME)
        return json.loads(response['SecretString'])
    except ClientError as e:
        raise Exception(f"Failed to retrieve GitHub credentials: {e}")


def dispatch_github_workflow(dispatch_url, token, payload):
    """
    Send repository_dispatch event to GitHub API

    GitHub API Documentation:
    https://docs.github.com/en/rest/repos/repos#create-a-repository-dispatch-event

    Args:
        dispatch_url: GitHub API dispatches endpoint
        token: GitHub Personal Access Token
        payload: Event payload (event_type + client_payload)

    Returns:
        requests.Response object
    """
    headers = {
        'Authorization': f'Bearer {token}',
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
        'Content-Type': 'application/json'
    }

    print(f"[KB7-GITHUB] Sending repository_dispatch to {dispatch_url}")
    print(f"[KB7-GITHUB] Event type: {payload['event_type']}")

    response = requests.post(
        dispatch_url,
        headers=headers,
        json=payload,
        timeout=30
    )

    # GitHub returns 204 No Content on success
    if response.status_code == 204:
        print(f"[KB7-GITHUB] Dispatch successful (204 No Content)")
        return response
    elif response.status_code == 404:
        raise Exception(f"Repository not found or PAT lacks permissions: {response.text}")
    elif response.status_code == 401:
        raise Exception(f"GitHub authentication failed: {response.text}")
    else:
        response.raise_for_status()

    return response


def publish_metrics(success):
    """Publish CloudWatch metrics for monitoring"""
    metric_name = 'GitHubDispatchSuccess' if success else 'GitHubDispatchFailures'

    cloudwatch_client.put_metric_data(
        Namespace='KB7/KnowledgeFactory',
        MetricData=[
            {
                'MetricName': metric_name,
                'Value': 1,
                'Unit': 'Count',
                'Timestamp': datetime.utcnow()
            }
        ]
    )
