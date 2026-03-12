"""
SNOMED CT-AU Downloader - GCP Cloud Run Job
Downloads SNOMED CT-AU (Australian Edition) + AMT from Australian NTS
Source: https://api.healthterminologies.gov.au/syndication/v1/syndication.xml

Module IDs:
- SNOMED CT-AU: 32506021000036107
- AMT: 900062011000036103

Authentication:
Uses OAuth2 client credentials flow (client_id + client_secret from NCTS Portal)
"""

import os
import hashlib
import logging
import json
import sys
import xml.etree.ElementTree as ET
from datetime import datetime
from google.cloud import storage
from google.cloud import secretmanager
import requests

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Configuration
SERVICE_NAME = "snomed-au"
REGION = "au"

# Australian NTS Syndication Feed and OAuth2 endpoints
NTS_SYNDICATION_URL = "https://api.healthterminologies.gov.au/syndication/v1/syndication.xml"
NTS_API_BASE = "https://api.healthterminologies.gov.au"
NTS_OAUTH_TOKEN_URL = "https://api.healthterminologies.gov.au/oauth2/token"

# Package identifiers in syndication feed
SNOMED_AU_RF2_SNAPSHOT = "SNOMEDCT-AU RF2 SNAPSHOT"
SNOMED_AU_RF2_FULL = "SNOMEDCT-AU RF2 FULL"
AMT_CSV = "AMT CSV"
AMT_TSV = "AMT TSV"
FHIR_R4_BUNDLE = "SNOMED CT-AU FHIR R4"


def write_gcs_signal(status, message, data=None):
    """Writes the status file to GCS for workflow coordination."""
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
            "region": REGION,
            "data": data or {}
        }

        blob.upload_from_string(
            json.dumps(payload, indent=2),
            content_type='application/json'
        )
        logger.info(f"Signal file written: {status}")
    except Exception as e:
        logger.error(f"Failed to write GCS signal: {str(e)}")


def get_nts_credentials():
    """
    Retrieves NTS OAuth2 credentials from Secret Manager.
    Secret should contain JSON: {"client_id": "...", "client_secret": "..."}
    (Legacy format with username/password also supported for backwards compatibility)
    """
    try:
        project_id = os.environ.get('PROJECT_ID')
        secret_name = os.environ.get('NTS_SECRET_NAME', 'kb7-nts-australia-credentials')

        if not project_id:
            logger.warning("PROJECT_ID not set, trying without auth")
            return None, None

        secret_client = secretmanager.SecretManagerServiceClient()
        secret_path = f"projects/{project_id}/secrets/{secret_name}/versions/latest"

        logger.info(f"Retrieving NTS credentials from: {secret_name}")
        response = secret_client.access_secret_version(name=secret_path)
        secret_data = json.loads(response.payload.data.decode('UTF-8'))

        # Support both OAuth2 (client_id/client_secret) and legacy (username/password) formats
        client_id = secret_data.get('client_id') or secret_data.get('username')
        client_secret = secret_data.get('client_secret') or secret_data.get('password')

        return client_id, client_secret
    except Exception as e:
        logger.warning(f"Could not retrieve NTS credentials: {e}")
        logger.info("Attempting download without authentication...")
        return None, None


def get_oauth2_token(client_id, client_secret):
    """
    Obtains an OAuth2 access token using client credentials flow.
    Australian NTS uses OAuth2 for API authentication.
    """
    try:
        logger.info("Requesting OAuth2 access token from NTS...")

        token_data = {
            'grant_type': 'client_credentials',
            'client_id': client_id,
            'client_secret': client_secret
        }

        headers = {
            'Content-Type': 'application/x-www-form-urlencoded',
            'User-Agent': 'KB7-Terminology-Service/1.0'
        }

        response = requests.post(
            NTS_OAUTH_TOKEN_URL,
            data=token_data,
            headers=headers,
            timeout=30
        )
        response.raise_for_status()

        token_response = response.json()
        access_token = token_response.get('access_token')

        if access_token:
            token_type = token_response.get('token_type', 'Bearer')
            expires_in = token_response.get('expires_in', 'unknown')
            logger.info(f"OAuth2 token obtained successfully (expires in {expires_in}s)")
            return access_token, token_type
        else:
            logger.error("No access_token in OAuth2 response")
            return None, None

    except requests.exceptions.RequestException as e:
        logger.error(f"OAuth2 token request failed: {e}")
        return None, None
    except Exception as e:
        logger.error(f"Unexpected error getting OAuth2 token: {e}")
        return None, None


def parse_syndication_feed(xml_content):
    """
    Parses the NTS Atom syndication feed to extract package information.
    Returns dict of packages with their download URLs, versions, and checksums.
    """
    packages = {}

    # Parse XML with namespace handling
    root = ET.fromstring(xml_content)

    # Atom namespace - note ncts attributes on link element
    ns = {
        'atom': 'http://www.w3.org/2005/Atom',
        'ncts': 'http://ns.electronichealth.net.au/ncts/syndication/asf/extensions/1.0.0'
    }

    # Find all entry elements
    for entry in root.findall('.//atom:entry', ns):
        try:
            title_elem = entry.find('atom:title', ns)
            if title_elem is None:
                continue
            title = title_elem.text or ''

            # Get category (package type) - e.g., "SCT_RF2_SNAPSHOT"
            category_elem = entry.find('atom:category', ns)
            category = category_elem.get('term') if category_elem is not None else ''

            # Get download link - NTS uses link[@rel="alternate"] with href attribute
            # Format: <link rel="alternate" type="application/zip" href="..." length="..." ncts:sha256Hash="..." />
            link_elem = entry.find('atom:link[@rel="alternate"]', ns)
            if link_elem is None:
                # Try without namespace prefix for the attribute
                link_elem = entry.find('atom:link', ns)
            if link_elem is None:
                continue

            download_url = link_elem.get('href', '')
            content_type = link_elem.get('type', '')

            # Get file size from link length attribute
            size_str = link_elem.get('length', '0')
            size = int(size_str) if size_str.isdigit() else 0

            # Get SHA256 hash from link attribute (with ncts namespace)
            sha256 = link_elem.get('{http://ns.electronichealth.net.au/ncts/syndication/asf/extensions/1.0.0}sha256Hash', '')

            # Get updated date
            updated_elem = entry.find('atom:updated', ns)
            updated = updated_elem.text if updated_elem is not None else ''

            # Extract version from URL or title
            # URL format: .../20250630/...
            # Title format: "SNOMED CT-AU 30 June 2025 (RF2 SNAPSHOT)"
            version = ''
            if download_url:
                # Try to extract date from URL path
                import re
                date_match = re.search(r'/(\d{8})/', download_url)
                if date_match:
                    version = date_match.group(1)

            if not version and updated:
                version = updated[:10].replace('-', '')

            # Determine package key based on category term (most reliable)
            # Categories: SCT_RF2_SNAPSHOT, SCT_RF2_FULL, SCT_RF2_ALL, AMT_CSV, AMT_TSV, FHIR_Bundle
            package_key = None

            if category == 'SCT_RF2_SNAPSHOT':
                package_key = SNOMED_AU_RF2_SNAPSHOT
            elif category == 'SCT_RF2_FULL':
                package_key = SNOMED_AU_RF2_FULL
            elif category == 'AMT_CSV':
                package_key = AMT_CSV
            elif category == 'AMT_TSV':
                package_key = AMT_TSV
            elif 'FHIR' in category.upper() and 'R4' in title.upper():
                package_key = FHIR_R4_BUNDLE
            # Fallback to title-based detection if category doesn't match
            elif 'RF2' in title.upper() and 'SNAPSHOT' in title.upper():
                package_key = SNOMED_AU_RF2_SNAPSHOT
            elif 'RF2' in title.upper() and 'FULL' in title.upper():
                package_key = SNOMED_AU_RF2_FULL

            if package_key and download_url:
                # Only keep the latest version of each package
                if package_key not in packages or version > packages[package_key].get('version', ''):
                    packages[package_key] = {
                        'title': title,
                        'download_url': download_url,
                        'version': version,
                        'size': size,
                        'sha256': sha256,
                        'content_type': content_type,
                        'category': category
                    }
                    logger.info(f"Found package: {package_key} v{version} ({size/(1024*1024):.1f}MB)")

        except Exception as e:
            logger.warning(f"Error parsing entry: {e}")
            continue

    return packages


def download_snomed_au():
    """
    Downloads SNOMED CT-AU RF2 SNAPSHOT from Australian NTS.
    Streams directly to Cloud Storage with checksum verification.
    """
    try:
        # Get configuration
        project_id = os.environ.get('PROJECT_ID')
        source_bucket = os.environ.get('SOURCE_BUCKET')
        environment = os.environ.get('ENVIRONMENT', 'production')

        # Package to download (can be overridden via env var)
        package_type = os.environ.get('PACKAGE_TYPE', SNOMED_AU_RF2_SNAPSHOT)

        logger.info("=" * 80)
        logger.info(f"SNOMED CT-AU Downloader - Region: {REGION}")
        logger.info(f"Environment: {environment}")
        logger.info(f"Target Package: {package_type}")
        logger.info("=" * 80)

        # Get NTS OAuth2 credentials and token
        client_id, client_secret = get_nts_credentials()
        access_token = None

        if client_id and client_secret:
            access_token, token_type = get_oauth2_token(client_id, client_secret)
            if not access_token:
                logger.warning("Failed to obtain OAuth2 token, attempting without auth")

        # Fetch syndication feed
        logger.info(f"Fetching syndication feed from: {NTS_SYNDICATION_URL}")

        headers = {
            'Accept': 'application/atom+xml',
            'User-Agent': 'KB7-Terminology-Service/1.0'
        }

        # Add Bearer token if available
        if access_token:
            headers['Authorization'] = f'Bearer {access_token}'

        feed_response = requests.get(
            NTS_SYNDICATION_URL,
            headers=headers,
            timeout=60
        )
        feed_response.raise_for_status()

        logger.info("Syndication feed retrieved successfully")

        # Parse feed to find packages
        packages = parse_syndication_feed(feed_response.content)

        logger.info(f"Found {len(packages)} packages in syndication feed:")
        for pkg_name, pkg_info in packages.items():
            logger.info(f"  - {pkg_name}: v{pkg_info['version']} ({pkg_info['size'] / (1024*1024):.1f} MB)")

        # Get target package
        if package_type not in packages:
            raise ValueError(f"Package '{package_type}' not found in syndication feed. Available: {list(packages.keys())}")

        package = packages[package_type]
        download_url = package['download_url']
        version = package['version']
        expected_sha256 = package['sha256']
        expected_size = package['size']

        logger.info(f"\nDownloading: {package['title']}")
        logger.info(f"Version: {version}")
        logger.info(f"URL: {download_url}")
        logger.info(f"Expected Size: {expected_size / (1024*1024):.1f} MB")
        logger.info(f"Expected SHA256: {expected_sha256[:16]}...")

        # Initialize Cloud Storage
        storage_client = storage.Client()
        bucket = storage_client.bucket(source_bucket)

        # Determine blob path with region prefix
        filename = download_url.split('/')[-1]
        if not filename.endswith('.zip'):
            filename = f"SNOMED_CT_AU_RF2_SNAPSHOT_{version}.zip"

        blob_name = f"{REGION}/snomed-ct-au/{version}/{filename}"
        blob = bucket.blob(blob_name)

        # Check if already exists
        if blob.exists():
            existing_metadata = blob.metadata or {}
            if existing_metadata.get('sha256') == expected_sha256:
                logger.info(f"File already exists with matching checksum: gs://{source_bucket}/{blob_name}")
                result = {
                    'status': 'success',
                    'message': 'File already exists (skipped download)',
                    'gcs_uri': f"gs://{source_bucket}/{blob_name}",
                    'gcs_key': blob_name,
                    'version': version,
                    'region': REGION,
                    'terminology': 'snomed-ct-au',
                    'module_id': '32506021000036107',
                    'timestamp': datetime.utcnow().isoformat()
                }
                write_gcs_signal('success', 'File already exists', result)
                return result

        # Download with streaming to GCS
        logger.info(f"Starting download to: gs://{source_bucket}/{blob_name}")

        hasher = hashlib.sha256()
        bytes_downloaded = 0
        chunk_size = 10 * 1024 * 1024  # 10MB chunks

        download_headers = {
            'User-Agent': 'KB7-Terminology-Service/1.0'
        }

        # Add Bearer token for download authentication
        if access_token:
            download_headers['Authorization'] = f'Bearer {access_token}'

        with requests.get(download_url, stream=True, headers=download_headers, timeout=3600) as r:
            r.raise_for_status()

            content_length = int(r.headers.get('content-length', expected_size))
            logger.info(f"Content-Length: {content_length / (1024*1024):.1f} MB")

            with blob.open("wb", chunk_size=chunk_size) as f:
                for chunk in r.iter_content(chunk_size=chunk_size):
                    if chunk:
                        f.write(chunk)
                        hasher.update(chunk)
                        bytes_downloaded += len(chunk)

                        # Log progress every 50MB
                        if bytes_downloaded % (50 * 1024 * 1024) < chunk_size:
                            mb_downloaded = bytes_downloaded / (1024 * 1024)
                            pct = (bytes_downloaded / content_length * 100) if content_length else 0
                            logger.info(f"Progress: {mb_downloaded:.1f} MB ({pct:.1f}%)")

        calculated_sha256 = hasher.hexdigest()

        # Verify checksum if available
        if expected_sha256:
            if calculated_sha256.lower() != expected_sha256.lower():
                logger.error(f"Checksum mismatch! Expected: {expected_sha256}, Got: {calculated_sha256}")
                # Delete corrupted file
                blob.delete()
                raise ValueError("SHA256 checksum verification failed")
            logger.info(f"SHA256 verified: {calculated_sha256[:16]}...")

        # Set metadata on blob
        blob.metadata = {
            'version': version,
            'source': 'Australian NTS',
            'sha256': calculated_sha256,
            'file_size_bytes': str(bytes_downloaded),
            'download_timestamp': datetime.utcnow().isoformat(),
            'environment': environment,
            'region': REGION,
            'terminology': 'snomed-ct-au',
            'module_id': '32506021000036107',
            'includes_amt': 'true',
            'amt_module_id': '900062011000036103'
        }
        blob.patch()

        logger.info(f"\nDownload complete: {bytes_downloaded / (1024*1024):.1f} MB")
        logger.info(f"Uploaded to: gs://{source_bucket}/{blob_name}")

        # Prepare success response
        result = {
            'status': 'success',
            'message': 'Download complete',
            'gcs_uri': f"gs://{source_bucket}/{blob_name}",
            'gcs_key': blob_name,
            'version': version,
            'sha256': calculated_sha256,
            'file_size_bytes': bytes_downloaded,
            'region': REGION,
            'terminology': 'snomed-ct-au',
            'module_id': '32506021000036107',
            'includes_amt': True,
            'amt_module_id': '900062011000036103',
            'timestamp': datetime.utcnow().isoformat()
        }

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
            'region': REGION,
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
            'region': REGION,
            'details': error_msg
        }


if __name__ == "__main__":
    """Main entrypoint for Cloud Run Job execution"""
    logger.info("=" * 80)
    logger.info("SNOMED CT-AU Downloader - Cloud Run Job Starting")
    logger.info("Australian National Terminology Service (NTS)")
    logger.info("=" * 80)

    result = download_snomed_au()

    # Print result as JSON for logging
    print(json.dumps(result, indent=2))

    # Exit with appropriate status code
    if result.get('status') == 'failed':
        logger.error("Job failed - exiting with error code 1")
        sys.exit(1)
    else:
        logger.info("Job completed successfully - exiting with code 0")
        sys.exit(0)
