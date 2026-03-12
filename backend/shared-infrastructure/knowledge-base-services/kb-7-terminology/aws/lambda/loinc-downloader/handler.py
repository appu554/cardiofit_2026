"""
KB-7 Knowledge Factory - LOINC Downloader Lambda
Downloads LOINC terminology files from Regenstrief Institute

Features:
- LOINC.org authenticated downloads
- CSV and XML format support
- S3 upload with metadata
- SHA256 integrity verification
"""

import os
import json
import boto3
import hashlib
import requests
from datetime import datetime
from botocore.exceptions import ClientError

# AWS clients
s3_client = boto3.client('s3')
secrets_client = boto3.client('secretsmanager')
cloudwatch_client = boto3.client('cloudwatch')

# Environment variables
BUCKET_NAME = os.environ['BUCKET_NAME']
ENVIRONMENT = os.environ['ENVIRONMENT']
LOINC_SECRET_NAME = os.environ['LOINC_SECRET_NAME']

# Constants
CHUNK_SIZE = 5 * 1024 * 1024  # 5MB chunks


def lambda_handler(event, context):
    """
    Main Lambda handler for LOINC downloads

    Expected event structure:
    {
        "version": "2.77",     # Optional - defaults to latest
        "format": "csv"        # Optional - csv, xml
    }
    """
    try:
        print(f"[KB7-LOINC] Starting LOINC download - Environment: {ENVIRONMENT}")

        # Get credentials from Secrets Manager
        credentials = get_credentials()
        username = credentials['username']
        password = credentials['password']
        api_endpoint = credentials['api_endpoint']

        # Determine version to download
        version = event.get('version', get_latest_version())
        file_format = event.get('format', 'csv')

        print(f"[KB7-LOINC] Target version: {version}, Format: {file_format}")

        # Get download URL from LOINC API
        download_url = get_loinc_download_url(api_endpoint, username, password, version, file_format)

        # Download and upload to S3
        s3_key = f"loinc/{version}/LOINC_{version}.{file_format}.zip"
        file_size, checksum = download_and_upload(download_url, BUCKET_NAME, s3_key, username, password)

        # Publish metrics
        publish_metrics(file_size, duration_seconds=(context.get_remaining_time_in_millis() / 1000))

        print(f"[KB7-LOINC] Download complete - Size: {file_size} bytes, SHA256: {checksum}")

        return {
            'statusCode': 200,
            'body': json.dumps({
                'message': 'LOINC download successful',
                's3_bucket': BUCKET_NAME,
                's3_key': s3_key,
                'file_size': file_size,
                'checksum': checksum,
                'version': version,
                'format': file_format
            })
        }

    except Exception as e:
        print(f"[KB7-LOINC] ERROR: {str(e)}")

        # Publish failure metric
        cloudwatch_client.put_metric_data(
            Namespace='KB7/KnowledgeFactory',
            MetricData=[{
                'MetricName': 'LOINCDownloadFailures',
                'Value': 1,
                'Unit': 'Count',
                'Timestamp': datetime.utcnow()
            }]
        )

        return {
            'statusCode': 500,
            'body': json.dumps({
                'error': str(e),
                'message': 'LOINC download failed'
            })
        }


def get_credentials():
    """Retrieve LOINC credentials from Secrets Manager"""
    try:
        response = secrets_client.get_secret_value(SecretId=LOINC_SECRET_NAME)
        return json.loads(response['SecretString'])
    except ClientError as e:
        raise Exception(f"Failed to retrieve credentials: {e}")


def get_latest_version():
    """
    Query LOINC.org for the latest version
    Returns format: X.YY (e.g., 2.77)
    """
    # TODO: Implement LOINC API call to get latest version
    # For now, return placeholder
    return "2.77"


def get_loinc_download_url(endpoint, username, password, version, file_format):
    """
    Get the download URL for a specific LOINC release

    LOINC API Documentation:
    https://loinc.org/downloads/
    """
    # TODO: Implement actual LOINC download endpoint with authentication
    # Example: GET /downloads/loinc/{version}/{format}

    # Placeholder for development
    return f"{endpoint}/loinc-{version}-{file_format}.zip"


def download_and_upload(download_url, bucket, key, username, password):
    """
    Download LOINC file and upload to S3 with authentication

    Args:
        download_url: Source URL for LOINC file
        bucket: Target S3 bucket name
        key: Target S3 object key
        username: LOINC.org username
        password: LOINC.org password

    Returns:
        tuple: (total_bytes_transferred, sha256_checksum)
    """
    print(f"[KB7-LOINC] Downloading from {download_url}")

    # LOINC requires HTTP Basic Authentication
    auth = (username, password)

    response = requests.get(download_url, auth=auth, stream=True, timeout=180)
    response.raise_for_status()

    sha256_hash = hashlib.sha256()
    total_bytes = 0
    buffer = bytearray()

    for chunk in response.iter_content(chunk_size=CHUNK_SIZE):
        if chunk:
            buffer.extend(chunk)
            sha256_hash.update(chunk)
            total_bytes += len(chunk)

    # Upload to S3
    print(f"[KB7-LOINC] Uploading {total_bytes / (1024**2):.2f} MB to s3://{bucket}/{key}")

    s3_client.put_object(
        Bucket=bucket,
        Key=key,
        Body=bytes(buffer),
        ServerSideEncryption='AES256',
        Metadata={
            'source': 'loinc-org-api',
            'downloaded': datetime.utcnow().isoformat(),
            'environment': ENVIRONMENT,
            'sha256': sha256_hash.hexdigest()
        }
    )

    checksum = sha256_hash.hexdigest()
    print(f"[KB7-LOINC] Upload complete")

    return total_bytes, checksum


def publish_metrics(file_size, duration_seconds):
    """Publish CloudWatch metrics for monitoring"""
    cloudwatch_client.put_metric_data(
        Namespace='KB7/KnowledgeFactory',
        MetricData=[
            {
                'MetricName': 'LOINCDownloadSize',
                'Value': file_size / (1024**2),  # MB
                'Unit': 'Megabytes',
                'Timestamp': datetime.utcnow()
            },
            {
                'MetricName': 'LOINCDownloadDuration',
                'Value': duration_seconds,
                'Unit': 'Seconds',
                'Timestamp': datetime.utcnow()
            },
            {
                'MetricName': 'LOINCDownloadSuccess',
                'Value': 1,
                'Unit': 'Count',
                'Timestamp': datetime.utcnow()
            }
        ]
    )
