"""
SNOMED CT Downloader - GCP Cloud Run Job
Downloads SNOMED CT International Edition from UMLS and streams to Cloud Storage
"""

import os
import hashlib
import logging
import json
import sys
from datetime import datetime
from google.cloud import storage
from google.cloud import secretmanager
import requests

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Configuration
SERVICE_NAME = "snomed"

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


def download_snomed():
    """
    Cloud Run Job to download SNOMED CT International from UMLS
    Streams directly to Cloud Storage (no local disk needed)

    Returns:
        dict: Download status and metadata
    """

    try:
        # Get configuration from environment
        project_id = os.environ['PROJECT_ID']
        source_bucket = os.environ['SOURCE_BUCKET']
        environment = os.environ['ENVIRONMENT']
        secret_name = os.environ['SECRET_NAME']

        logger.info(f"Starting SNOMED CT download for environment: {environment}")

        # Retrieve UMLS API key from Secret Manager
        secret_client = secretmanager.SecretManagerServiceClient()
        secret_path = f"projects/{project_id}/secrets/{secret_name}/versions/latest"

        logger.info(f"Retrieving UMLS API key from Secret Manager: {secret_name}")
        api_key_response = secret_client.access_secret_version(name=secret_path)
        umls_api_key = api_key_response.payload.data.decode('UTF-8')

        # Use UMLS Release API to get current SNOMED CT International Edition
        # Reference: https://documentation.uts.nlm.nih.gov/automating-downloads.html#release-api
        logger.info("Fetching SNOMED CT release metadata from UMLS Release API")

        release_api_url = "https://uts-ws.nlm.nih.gov/releases"
        params = {
            'releaseType': 'snomed-ct-international-edition',
            'current': 'true',
            'apiKey': umls_api_key
        }

        release_response = requests.get(release_api_url, params=params, timeout=30)
        release_response.raise_for_status()

        release_data = release_response.json()

        # Extract download URL and version from API response
        # The API returns a list of releases, get the first (current) one
        if not release_data or len(release_data) == 0:
            raise ValueError("No SNOMED CT International Edition releases found")

        current_release = release_data[0]
        download_url = current_release.get('downloadUrl')
        version = current_release.get('releaseDate', datetime.now().strftime('%Y-%m-%d'))

        if not download_url:
            raise ValueError("No download URL found in release data")

        logger.info(f"SNOMED CT version: {version}")
        logger.info(f"Download URL: {download_url}")

        # Initialize Cloud Storage client
        storage_client = storage.Client()
        bucket = storage_client.bucket(source_bucket)

        # Extract filename from download URL and version for storage path
        filename = download_url.split('/')[-1]
        version_formatted = version.replace("-", "").replace(":", "").replace("T", "")[:8]  # Convert to YYYYMMDD
        blob_name = f"snomed-ct/{version_formatted}/{filename}"
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
                'version_formatted': version_formatted,
                'terminology': 'snomed',
                'edition': 'International',
                'timestamp': datetime.utcnow().isoformat()
            }

            # Write result to GCS for workflow coordination
            write_gcs_signal('success', 'File already exists (skipped download)', result)
            return result

        logger.info(f"Starting download to: gs://{source_bucket}/{blob_name}")

        # Stream download to Cloud Storage with hash calculation
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
            logger.info(f"Expected file size: {file_size / (1024**3):.2f} GB")

            # Use blob.open() for streaming upload
            with blob.open("wb", chunk_size=chunk_size) as f:
                for chunk in r.iter_content(chunk_size=chunk_size):
                    if chunk:
                        f.write(chunk)
                        hasher.update(chunk)
                        bytes_downloaded += len(chunk)

                        # Log progress every GB
                        if bytes_downloaded % (1024**3) < chunk_size:
                            gb_downloaded = bytes_downloaded / (1024**3)
                            logger.info(f"Downloaded: {gb_downloaded:.2f} GB")

        sha256_hash = hasher.hexdigest()

        # Set metadata on blob
        blob.metadata = {
            'version': version,
            'version_formatted': version_formatted,
            'source': 'UMLS API',
            'sha256': sha256_hash,
            'file_size_bytes': str(bytes_downloaded),
            'download_timestamp': datetime.utcnow().isoformat(),
            'environment': environment,
            'terminology': 'snomed',
            'edition': 'International'
        }
        blob.patch()

        logger.info(f"Download complete: {bytes_downloaded / (1024**3):.2f} GB")
        logger.info(f"SHA256: {sha256_hash}")
        logger.info(f"Successfully uploaded to: gs://{source_bucket}/{blob_name}")

        # Prepare success response
        result = {
            'status': 'success',
            'message': 'Download complete',
            'gcs_uri': f"gs://{source_bucket}/{blob_name}",
            'gcs_key': blob_name,
            'version': version,
            'version_formatted': version_formatted,
            'sha256': sha256_hash,
            'file_size_bytes': bytes_downloaded,
            'terminology': 'snomed',
            'edition': 'International',
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
    logger.info("SNOMED CT Downloader - Cloud Run Job Starting")
    logger.info("=" * 80)

    result = download_snomed()

    # Print result as JSON for Cloud Run Jobs logging
    print(json.dumps(result, indent=2))

    # Exit with appropriate status code
    if result.get('status') == 'failed':
        logger.error("Job failed - exiting with error code 1")
        sys.exit(1)
    else:
        logger.info("Job completed successfully - exiting with code 0")
        sys.exit(0)
