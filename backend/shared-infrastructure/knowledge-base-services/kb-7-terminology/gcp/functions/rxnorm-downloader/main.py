"""
RxNorm Downloader - GCP Cloud Run Job
Downloads RxNorm from NIH UMLS API and streams to Cloud Storage
"""

import os
import sys
import hashlib
import logging
import json
import zipfile
import io
from datetime import datetime
from google.cloud import storage
from google.cloud import secretmanager
import requests

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
SERVICE_NAME = "rxnorm"

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


def download_rxnorm():
    """
    Cloud Run Job to download RxNorm from NIH UMLS API

    Returns:
        JSON response with download status and metadata
    """

    try:
        # Get configuration
        project_id = os.environ['PROJECT_ID']
        source_bucket = os.environ['SOURCE_BUCKET']
        environment = os.environ['ENVIRONMENT']
        secret_name = os.environ['SECRET_NAME']

        logger.info(f"Starting RxNorm download for environment: {environment}")

        # Retrieve UMLS API key from Secret Manager
        secret_client = secretmanager.SecretManagerServiceClient()
        secret_path = f"projects/{project_id}/secrets/{secret_name}/versions/latest"

        logger.info(f"Retrieving UMLS API key from Secret Manager")
        api_key_response = secret_client.access_secret_version(name=secret_path)
        umls_api_key = api_key_response.payload.data.decode('UTF-8')

        # Get RxNorm current version metadata
        logger.info("Fetching RxNorm version metadata from NIH UMLS")
        version_url = "https://rxnav.nlm.nih.gov/REST/version.json"
        version_response = requests.get(version_url, timeout=30)
        version_response.raise_for_status()

        version_data = version_response.json()
        # RxNav returns version like "06-Oct-2025" but we'll use current for download
        version_display = version_data.get('version', datetime.now().strftime('%d-%b-%Y'))

        # For storage organization, convert to MMDDYYYY format
        # Parse "06-Oct-2025" to get date components
        try:
            date_obj = datetime.strptime(version_display, '%d-%b-%Y')
            version = date_obj.strftime('%m%d%Y')  # Convert to MMDDYYYY (10062025)
        except:
            version = datetime.now().strftime('%m%d%Y')

        logger.info(f"RxNorm version: {version_display} (storage: {version})")

        # UMLS download URL for RxNorm RRF files - use current for automation reliability
        # Reference: https://documentation.uts.nlm.nih.gov/automating-downloads.html
        download_url = "https://download.nlm.nih.gov/umls/kss/rxnorm/RxNorm_full_current.zip"

        # Initialize Cloud Storage
        storage_client = storage.Client()
        bucket = storage_client.bucket(source_bucket)
        blob_name = f"rxnorm/{version}/RxNorm_full_{version}.zip"
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
                'terminology': 'rxnorm',
                'timestamp': datetime.utcnow().isoformat()
            }

            # Write result to GCS for workflow coordination
            write_gcs_signal('success', 'File already exists (skipped download)', result)
            return result

        logger.info(f"Downloading RxNorm from: {download_url}")

        # Stream download with authentication using UTS proxy endpoint
        # UMLS requires using UTS proxy endpoint with API key
        # Format: https://uts-ws.nlm.nih.gov/download?url=[file_url]&apiKey=YOUR_API_KEY
        uts_proxy_url = f"https://uts-ws.nlm.nih.gov/download?url={download_url}&apiKey={umls_api_key}"

        hasher = hashlib.sha256()
        bytes_downloaded = 0
        chunk_size = 10 * 1024 * 1024  # 10MB chunks

        with requests.get(uts_proxy_url, stream=True, timeout=3600) as r:
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

                        # Log progress every 100MB
                        if bytes_downloaded % (100 * 1024 * 1024) < chunk_size:
                            mb_downloaded = bytes_downloaded / (1024**2)
                            logger.info(f"Downloaded: {mb_downloaded:.2f} MB")

        sha256_hash = hasher.hexdigest()

        # Set metadata
        blob.metadata = {
            'version': version,
            'source': 'NIH UMLS',
            'sha256': sha256_hash,
            'file_size_bytes': str(bytes_downloaded),
            'download_timestamp': datetime.utcnow().isoformat(),
            'environment': environment,
            'terminology': 'rxnorm'
        }
        blob.patch()

        logger.info(f"Download complete: {bytes_downloaded / (1024**2):.2f} MB")
        logger.info(f"SHA256: {sha256_hash}")
        logger.info(f"Successfully uploaded to: gs://{source_bucket}/{blob_name}")

        # Prepare success response
        result = {
            'status': 'success',
            'message': 'Download complete',
            'gcs_uri': f"gs://{source_bucket}/{blob_name}",
            'gcs_key': blob_name,
            'version': version,
            'sha256': sha256_hash,
            'file_size_bytes': bytes_downloaded,
            'terminology': 'rxnorm',
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
    logger.info("RxNorm Downloader - Cloud Run Job Starting")
    logger.info("=" * 80)

    result = download_rxnorm()

    # Print result as JSON for Cloud Run Jobs logging
    print(json.dumps(result, indent=2))

    # Exit with appropriate status code
    if result.get('status') == 'failed':
        logger.error("Job failed - exiting with error code 1")
        sys.exit(1)
    else:
        logger.info("Job completed successfully - exiting with code 0")
        sys.exit(0)
