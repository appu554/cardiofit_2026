"""
GitHub Dispatcher - GCP Cloud Run Job
Triggers GitHub Actions workflow after successful downloads
"""

import os
import sys
import logging
import json
from datetime import datetime
from google.cloud import secretmanager
import requests

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def dispatch_github_workflow():
    """
    Cloud Run Job to dispatch GitHub Actions workflow

    Returns:
        JSON response with dispatch status
    """

    try:
        # Get configuration
        project_id = os.environ['PROJECT_ID']
        github_repo = os.environ['GITHUB_REPO']
        secret_name = os.environ['SECRET_NAME']
        environment = os.environ['ENVIRONMENT']

        logger.info(f"Starting GitHub workflow dispatch for repository: {github_repo}")

        # Parse download keys from environment variables (set by workflow orchestrator)
        snomed_key = os.environ.get('SNOMED_KEY', '')
        rxnorm_key = os.environ.get('RXNORM_KEY', '')
        loinc_key = os.environ.get('LOINC_KEY', '')

        logger.info(f"Download results received:")
        logger.info(f"  SNOMED: {snomed_key}")
        logger.info(f"  RxNorm: {rxnorm_key}")
        logger.info(f"  LOINC: {loinc_key}")

        # Retrieve GitHub token from Secret Manager
        secret_client = secretmanager.SecretManagerServiceClient()
        secret_path = f"projects/{project_id}/secrets/{secret_name}/versions/latest"

        logger.info(f"Retrieving GitHub token from Secret Manager")
        token_response = secret_client.access_secret_version(name=secret_path)
        github_token = token_response.payload.data.decode('UTF-8').strip()

        # Prepare GitHub repository dispatch payload
        dispatch_url = f"https://api.github.com/repos/{github_repo}/dispatches"

        # Extract versions from GCS keys (handle case where keys might be 'unknown')
        def extract_version(key):
            if not key or key == 'unknown':
                return 'unknown'
            parts = key.split('/')
            return parts[1] if len(parts) > 1 else 'unknown'

        snomed_version = extract_version(snomed_key)
        rxnorm_version = extract_version(rxnorm_key)
        loinc_version = extract_version(loinc_key)

        dispatch_payload = {
            'event_type': 'terminology-update',
            'client_payload': {
                'trigger_source': 'gcp-cloud-workflow',
                'environment': environment,
                'timestamp': datetime.utcnow().isoformat(),
                # Flat structure for GitHub Actions workflow
                'snomed_key': snomed_key,
                'rxnorm_key': rxnorm_key,
                'loinc_key': loinc_key,
                'version': datetime.utcnow().strftime('%Y%m%d'),
                # Nested structure for detailed information
                'downloads': {
                    'snomed': {
                        'gcs_key': snomed_key,
                        'version': snomed_version
                    },
                    'rxnorm': {
                        'gcs_key': rxnorm_key,
                        'version': rxnorm_version
                    },
                    'loinc': {
                        'gcs_key': loinc_key,
                        'version': loinc_version
                    }
                }
            }
        }

        # Send repository dispatch
        headers = {
            'Authorization': f"Bearer {github_token}",
            'Accept': 'application/vnd.github+json',
            'X-GitHub-Api-Version': '2022-11-28'
        }

        logger.info(f"Dispatching workflow to: {dispatch_url}")
        dispatch_response = requests.post(
            dispatch_url,
            headers=headers,
            json=dispatch_payload,
            timeout=30
        )
        dispatch_response.raise_for_status()

        logger.info(f"GitHub workflow dispatched successfully")
        logger.info(f"Response status: {dispatch_response.status_code}")

        # Send Slack notification (if configured)
        slack_webhook_secret = os.environ.get('SLACK_WEBHOOK_SECRET', '')
        if slack_webhook_secret:
            try:
                send_slack_notification(
                    project_id,
                    slack_webhook_secret,
                    snomed_version,
                    rxnorm_version,
                    loinc_version,
                    github_repo
                )
            except Exception as slack_error:
                logger.warning(f"Failed to send Slack notification: {slack_error}")

        return {
            'status': 'success',
            'message': 'GitHub workflow dispatched',
            'repository': github_repo,
            'event_type': 'terminology-update',
            'versions': {
                'snomed': snomed_version,
                'rxnorm': rxnorm_version,
                'loinc': loinc_version
            },
            'timestamp': datetime.utcnow().isoformat()
        }

    except requests.exceptions.RequestException as e:
        logger.error(f"GitHub API request failed: {str(e)}")
        return {
            'status': 'failed',
            'error': 'GitHub API request failed',
            'error_type': 'RequestException',
            'details': str(e)
        }

    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}", exc_info=True)
        return {
            'status': 'failed',
            'error': 'Unexpected error',
            'error_type': type(e).__name__,
            'details': str(e)
        }


def send_slack_notification(project_id, slack_secret_name, snomed_version, rxnorm_version, loinc_version, repo):
    """Send Slack notification about workflow dispatch"""

    # Retrieve Slack webhook URL
    secret_client = secretmanager.SecretManagerServiceClient()
    secret_path = f"projects/{project_id}/secrets/{slack_secret_name}/versions/latest"

    webhook_response = secret_client.access_secret_version(name=secret_path)
    webhook_url = webhook_response.payload.data.decode('UTF-8')

    # Prepare Slack message
    slack_message = {
        'text': 'KB-7 Knowledge Factory Pipeline Started',
        'blocks': [
            {
                'type': 'header',
                'text': {
                    'type': 'plain_text',
                    'text': '🚀 KB-7 Terminology Update Pipeline Started'
                }
            },
            {
                'type': 'section',
                'fields': [
                    {
                        'type': 'mrkdwn',
                        'text': f'*Repository:*\n{repo}'
                    },
                    {
                        'type': 'mrkdwn',
                        'text': f'*Trigger:*\nGCP Cloud Workflow'
                    }
                ]
            },
            {
                'type': 'section',
                'fields': [
                    {
                        'type': 'mrkdwn',
                        'text': f'*SNOMED CT:*\n{snomed_version}'
                    },
                    {
                        'type': 'mrkdwn',
                        'text': f'*RxNorm:*\n{rxnorm_version}'
                    },
                    {
                        'type': 'mrkdwn',
                        'text': f'*LOINC:*\n{loinc_version}'
                    }
                ]
            },
            {
                'type': 'context',
                'elements': [
                    {
                        'type': 'mrkdwn',
                        'text': f"Timestamp: {datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S UTC')}"
                    }
                ]
            }
        ]
    }

    # Send to Slack
    response = requests.post(webhook_url, json=slack_message, timeout=10)
    response.raise_for_status()

    logger.info("Slack notification sent successfully")


if __name__ == "__main__":
    """
    Main entrypoint for Cloud Run Job execution
    """
    logger.info("=" * 80)
    logger.info("GitHub Dispatcher - Cloud Run Job Starting")
    logger.info("=" * 80)

    result = dispatch_github_workflow()

    # Print result as JSON for Cloud Run Jobs logging
    print(json.dumps(result, indent=2))

    # Exit with appropriate status code
    if result.get('status') == 'failed':
        logger.error("Job failed - exiting with error code 1")
        sys.exit(1)
    else:
        logger.info("Job completed successfully - exiting with code 0")
        sys.exit(0)
