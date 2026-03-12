"""
LOINC Downloader - GCP Cloud Run Job
Downloads LOINC from LOINC.org and streams to Cloud Storage
"""

import os
import sys
import hashlib
import logging
import json
from datetime import datetime
from google.cloud import storage
from google.cloud import secretmanager
import requests
from requests.auth import HTTPBasicAuth

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
SERVICE_NAME = "loinc"

def write_gcs_signal(status, message, data=None):
    """Writes the status file to GCS. This is the signal the workflow waits for."""
    try:
        bucket_name = os.environ.get('SOURCE_BUCKET')
        if not bucket_name:
            logger.warning("SOURCE_BUCKET env var not set, cannot write signal")
            return

        client = storage.Client()
        bucket = client.bucket(bucket_name)
        blob = bucket.blob(f"workflow-results/{SERVICE_NAME}-latest.json")
        
        payload = {
            "status": status,
            "message": message,
            "timestamp": datetime.utcnow().isoformat(),
            "data": data or {}
        }
        
        blob.upload_from_string(
            json.dumps(payload, indent=2),
            content_type='application/json'
        )
        logger.info(f"Signal file written to gs://{bucket_name}/workflow-results/{SERVICE_NAME}-latest.json: {status}")
    except Exception as e:
        logger.error(f"CRITICAL ERROR: Failed to write GCS signal: {str(e)}")


def download_loinc():
    """
    Cloud Run Job to download LOINC from LOINC.org

    Returns:
        JSON response with download status and metadata
    """

    try:
        # Get configuration
        project_id = os.environ['PROJECT_ID']
        source_bucket = os.environ['SOURCE_BUCKET']
        environment = os.environ['ENVIRONMENT']
        secret_name = os.environ['SECRET_NAME']

        logger.info(f"Starting LOINC download for environment: {environment}")

        # Retrieve LOINC credentials from Secret Manager
        secret_client = secretmanager.SecretManagerServiceClient()
        secret_path = f"projects/{project_id}/secrets/{secret_name}/versions/latest"

        logger.info(f"Retrieving LOINC credentials from Secret Manager")
        credentials_response = secret_client.access_secret_version(name=secret_path)
        credentials = json.loads(credentials_response.payload.data.decode('UTF-8'))

        loinc_username = credentials['username']
        loinc_password = credentials['password']

        # Get LOINC version metadata from official API
        api_url = "https://loinc.regenstrief.org/api/v1/Loinc"
        logger.info(f"Fetching LOINC version metadata from API")

        api_response = requests.get(
            api_url,
            auth=HTTPBasicAuth(loinc_username, loinc_password),
            timeout=30
        )
        api_response.raise_for_status()

        metadata = api_response.json()
        version = metadata['version']  # e.g., "2.81"
        download_url = metadata['downloadUrl']  # API-provided URL
        expected_md5 = metadata.get('downloadMD5Hash', '')
        release_date = metadata.get('releaseDate', '')

        logger.info(f"LOINC version: {version}")
        logger.info(f"Release date: {release_date}")
        logger.info(f"Download URL: {download_url}")
        logger.info(f"Expected MD5: {expected_md5}")

        # Initialize Cloud Storage
        storage_client = storage.Client()
        bucket = storage_client.bucket(source_bucket)
        blob_name = f"loinc/{version}/loinc-complete-{version}.zip"
        blob = bucket.blob(blob_name)

        # Check if file already exists
        if blob.exists():
            logger.info(f"File already exists: gs://{source_bucket}/{blob_name}")
            result = {
                'status': 'success',
                'message': 'File already exists (skipped download)',
                'gcs_uri': f"gs://{source_bucket}/{blob_name}",
                'gcs_key': blob_name,
                'version': version,
                'terminology': 'loinc',
                'timestamp': datetime.utcnow().isoformat()
            }

            # Write result to GCS for workflow coordination
            write_gcs_signal('success', 'File already exists (skipped download)', result)
            return result

        logger.info(f"Downloading LOINC from: {download_url}")

        # Stream download with authentication
        hasher = hashlib.md5()
        bytes_downloaded = 0
        chunk_size = 10 * 1024 * 1024  # 10MB chunks

        with requests.get(
            download_url,
            auth=HTTPBasicAuth(loinc_username, loinc_password),
            stream=True,
            timeout=1800
        ) as r:
            r.raise_for_status()

            # Get file size from headers
            file_size = int(r.headers.get('content-length', 0))
            logger.info(f"Expected file size: {file_size / (1024**2):.2f} MB")

            # Stream to Cloud Storage
            with blob.open("wb", chunk_size=chunk_size) as f:
                for chunk in r.iter_content(chunk_size=chunk_size):
                    if chunk:
                        f.write(chunk)
                        hasher.update(chunk)
                        bytes_downloaded += len(chunk)

                        # Log progress every 50MB
                        if bytes_downloaded % (50 * 1024 * 1024) < chunk_size:
                            mb_downloaded = bytes_downloaded / (1024**2)
                            logger.info(f"Downloaded: {mb_downloaded:.2f} MB")

        md5_hash = hasher.hexdigest()

        # Verify MD5 hash if provided by API
        if expected_md5:
            if md5_hash.lower() == expected_md5.lower():
                logger.info(f"✅ MD5 verification passed: {md5_hash}")
            else:
                logger.error(f"❌ MD5 mismatch! Expected: {expected_md5}, Got: {md5_hash}")
                raise ValueError(f"MD5 hash verification failed")
        else:
            logger.warning("No MD5 hash provided by API - skipping verification")

        # Set metadata
        blob.metadata = {
            'version': version,
            'release_date': release_date,
            'source': 'LOINC API',
            'md5': md5_hash,
            'file_size_bytes': str(bytes_downloaded),
            'download_timestamp': datetime.utcnow().isoformat(),
            'environment': environment,
            'terminology': 'loinc'
        }
        blob.patch()

        logger.info(f"Download complete: {bytes_downloaded / (1024**2):.2f} MB")
        logger.info(f"MD5: {md5_hash}")
        logger.info(f"Successfully uploaded to: gs://{source_bucket}/{blob_name}")

        # Prepare success response
        result = {
            'status': 'success',
            'message': 'Download complete',
            'gcs_uri': f"gs://{source_bucket}/{blob_name}",
            'gcs_key': blob_name,
            'version': version,
            'release_date': release_date,
            'md5': md5_hash,
            'file_size_bytes': bytes_downloaded,
            'terminology': 'loinc',
            'timestamp': datetime.utcnow().isoformat()
        }

        # Write result to GCS for workflow coordination
        write_gcs_signal('success', 'Download complete', result)
        return result

    except requests.exceptions.RequestException as e:
        logger.error(f"HTTP request failed: {str(e)}")
        error_msg = str(e)
        write_gcs_signal('failed', f"HTTP request failed: {error_msg}")
        return {
            'status': 'failed',
            'error': 'HTTP request failed',
            'error_type': 'RequestException',
            'details': error_msg
        }

    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}", exc_info=True)
        error_msg = str(e)
        write_gcs_signal('failed', f"Unexpected error: {error_msg}")
        return {
            'status': 'failed',
            'error': 'Unexpected error',
            'error_type': type(e).__name__,
            'details': error_msg
        }


if __name__ == "__main__":
    """
    Main entrypoint for Cloud Run Job execution
    """
    logger.info("=" * 80)
    logger.info("LOINC Downloader - Cloud Run Job Starting")
    logger.info("=" * 80)

    result = download_loinc()

    # Print result as JSON for Cloud Run Jobs logging
    print(json.dumps(result, indent=2))

    # Exit with appropriate status code
    if result.get('status') == 'failed':
        logger.error("Job failed - exiting with error code 1")
        sys.exit(1)
    else:
        logger.info("Job completed successfully - exiting with code 0")
        sys.exit(0)
