"""
GitHub Dispatcher AU - GCP Cloud Run Job
Triggers GitHub Actions workflow for Australian Knowledge Factory after successful downloads

Region: Australia (knowledge-factory-au repository)
Terminology: SNOMED CT-AU (32506021000036107) + AMT (900062011000036103, bundled)
"""

import os
import sys
import logging
import json
from datetime import datetime
from google.cloud import secretmanager
from google.cloud import storage
import requests

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def dispatch_github_workflow():
    """
    Cloud Run Job to dispatch GitHub Actions workflow for AU knowledge factory

    Differences from US dispatcher:
    - Uses SNOMED_AU_KEY (single key containing SNOMED-AU + AMT bundled)
    - No RxNorm (Australia uses AMT for medications)
    - Dispatches to knowledge-factory-au repository
    - Event type: terminology-update-au

    Returns:
        JSON response with dispatch status
    """

    try:
        # Get configuration
        project_id = os.environ['PROJECT_ID']
        # AU uses separate repository: knowledge-factory-au
        github_repo = os.environ.get('GITHUB_REPO', 'onkarshahi-IND/knowledge-factory-au')
        secret_name = os.environ['SECRET_NAME']
        environment = os.environ['ENVIRONMENT']

        logger.info(f"Starting AU GitHub workflow dispatch for repository: {github_repo}")
        logger.info(f"Region: Australia (SNOMED CT-AU + AMT)")

        # Parse AU-specific download key (SNOMED-AU includes AMT bundled)
        snomed_au_key = os.environ.get('SNOMED_AU_KEY', '')
        # LOINC is optional for AU (shared international standard)
        loinc_key = os.environ.get('LOINC_KEY', '')

        logger.info(f"Download results received:")
        logger.info(f"  SNOMED-AU (includes AMT): {snomed_au_key}")
        if loinc_key:
            logger.info(f"  LOINC: {loinc_key}")

        # Retrieve GitHub token from Secret Manager
        secret_client = secretmanager.SecretManagerServiceClient()
        secret_path = f"projects/{project_id}/secrets/{secret_name}/versions/latest"

        logger.info(f"Retrieving GitHub token from Secret Manager")
        token_response = secret_client.access_secret_version(name=secret_path)
        github_token = token_response.payload.data.decode('UTF-8').strip()

        # Prepare GitHub repository dispatch payload
        dispatch_url = f"https://api.github.com/repos/{github_repo}/dispatches"

        # Extract version from GCS key
        def extract_version(key):
            if not key or key == 'unknown':
                return 'unknown'
            parts = key.split('/')
            return parts[1] if len(parts) > 1 else 'unknown'

        snomed_au_version = extract_version(snomed_au_key)
        loinc_version = extract_version(loinc_key) if loinc_key else 'shared'

        # AU-specific dispatch payload
        dispatch_payload = {
            'event_type': 'terminology-update-au',
            'client_payload': {
                'trigger_source': 'gcp-cloud-workflow',
                'environment': environment,
                'region': 'au',
                'region_display': 'Australia',
                'timestamp': datetime.utcnow().isoformat(),
                # Flat structure for GitHub Actions workflow
                'snomed_au_key': snomed_au_key,
                'loinc_key': loinc_key if loinc_key else 'shared',
                'version': datetime.utcnow().strftime('%Y%m%d'),
                # Nested structure for detailed information
                'downloads': {
                    'snomed_au': {
                        'gcs_key': snomed_au_key,
                        'version': snomed_au_version,
                        'module_id': '32506021000036107',
                        'includes_amt': True
                    },
                    'amt': {
                        'status': 'bundled_with_snomed_au',
                        'module_id': '900062011000036103',
                        'note': 'AMT is included in SNOMED CT-AU RF2 package'
                    },
                    'loinc': {
                        'gcs_key': loinc_key if loinc_key else 'shared',
                        'version': loinc_version,
                        'note': 'International standard, can be shared with US pipeline'
                    }
                },
                # AU-specific metadata
                'terminology_source': 'Australian National Terminology Service (NTS)',
                'nts_api': 'api.healthterminologies.gov.au'
            }
        }

        # Send repository dispatch
        headers = {
            'Authorization': f"Bearer {github_token}",
            'Accept': 'application/vnd.github+json',
            'X-GitHub-Api-Version': '2022-11-28'
        }

        logger.info(f"Dispatching AU workflow to: {dispatch_url}")
        dispatch_response = requests.post(
            dispatch_url,
            headers=headers,
            json=dispatch_payload,
            timeout=30
        )
        dispatch_response.raise_for_status()

        logger.info(f"AU GitHub workflow dispatched successfully")
        logger.info(f"Response status: {dispatch_response.status_code}")

        # Send Slack notification (if configured)
        slack_webhook_secret = os.environ.get('SLACK_WEBHOOK_SECRET', '')
        if slack_webhook_secret:
            try:
                send_slack_notification(
                    project_id,
                    slack_webhook_secret,
                    snomed_au_version,
                    loinc_version,
                    github_repo
                )
            except Exception as slack_error:
                logger.warning(f"Failed to send Slack notification: {slack_error}")

        return {
            'status': 'success',
            'message': 'AU GitHub workflow dispatched',
            'repository': github_repo,
            'event_type': 'terminology-update-au',
            'region': 'au',
            'versions': {
                'snomed_au': snomed_au_version,
                'amt': 'bundled',
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
            'region': 'au',
            'details': str(e)
        }

    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}", exc_info=True)
        return {
            'status': 'failed',
            'error': 'Unexpected error',
            'error_type': type(e).__name__,
            'region': 'au',
            'details': str(e)
        }


def write_status_to_gcs(result):
    """Write dispatcher status to GCS for workflow polling"""
    try:
        bucket_name = os.environ.get('GCS_BUCKET_SOURCES', 'sincere-hybrid-477206-h2-kb-sources-production')
        status_file = 'workflow-results/github-dispatcher-au-latest.json'

        storage_client = storage.Client()
        bucket = storage_client.bucket(bucket_name)
        blob = bucket.blob(status_file)

        status_data = {
            'status': result.get('status', 'unknown'),
            'region': 'au',
            'repository': result.get('repository', ''),
            'event_type': result.get('event_type', 'terminology-update-au'),
            'versions': result.get('versions', {}),
            'timestamp': datetime.utcnow().isoformat(),
            'message': result.get('message', '')
        }

        blob.upload_from_string(
            json.dumps(status_data, indent=2),
            content_type='application/json'
        )

        logger.info(f"Status written to gs://{bucket_name}/{status_file}")

    except Exception as e:
        logger.warning(f"Failed to write status to GCS: {e}")


def send_slack_notification(project_id, slack_secret_name, snomed_au_version, loinc_version, repo):
    """Send Slack notification about AU workflow dispatch"""

    # Retrieve Slack webhook URL
    secret_client = secretmanager.SecretManagerServiceClient()
    secret_path = f"projects/{project_id}/secrets/{slack_secret_name}/versions/latest"

    webhook_response = secret_client.access_secret_version(name=secret_path)
    webhook_url = webhook_response.payload.data.decode('UTF-8')

    # Prepare AU-specific Slack message
    slack_message = {
        'text': 'KB-7 Knowledge Factory AU Pipeline Started',
        'blocks': [
            {
                'type': 'header',
                'text': {
                    'type': 'plain_text',
                    'text': '🦘 KB-7 Australian Terminology Update Pipeline Started'
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
                        'text': '*Region:*\nAustralia (AU)'
                    }
                ]
            },
            {
                'type': 'section',
                'fields': [
                    {
                        'type': 'mrkdwn',
                        'text': f'*SNOMED CT-AU:*\n{snomed_au_version}'
                    },
                    {
                        'type': 'mrkdwn',
                        'text': '*AMT:*\nbundled with SNOMED-AU'
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
                        'text': f"Trigger: GCP Cloud Workflow | Timestamp: {datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S UTC')}"
                    }
                ]
            }
        ]
    }

    # Send to Slack
    response = requests.post(webhook_url, json=slack_message, timeout=10)
    response.raise_for_status()

    logger.info("AU Slack notification sent successfully")


if __name__ == "__main__":
    """
    Main entrypoint for Cloud Run Job execution
    """
    logger.info("=" * 80)
    logger.info("GitHub Dispatcher AU - Cloud Run Job Starting")
    logger.info("Region: Australia | Terminologies: SNOMED CT-AU + AMT + LOINC")
    logger.info("=" * 80)

    result = dispatch_github_workflow()

    # Write status to GCS for workflow polling
    write_status_to_gcs(result)

    # Print result as JSON for Cloud Run Jobs logging
    print(json.dumps(result, indent=2))

    # Exit with appropriate status code
    if result.get('status') == 'failed':
        logger.error("Job failed - exiting with error code 1")
        sys.exit(1)
    else:
        logger.info("Job completed successfully - exiting with code 0")
        sys.exit(0)
