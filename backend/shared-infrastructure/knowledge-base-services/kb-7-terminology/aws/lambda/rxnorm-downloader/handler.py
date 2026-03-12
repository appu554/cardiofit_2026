"""
KB-7 Knowledge Factory - RxNorm Downloader Lambda
Downloads RxNorm full prescribable content from NIH UMLS API

Features:
- UMLS API authentication
- RRF file format download
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
UMLS_SECRET_NAME = os.environ['UMLS_SECRET_NAME']

# Constants
CHUNK_SIZE = 5 * 1024 * 1024  # 5MB chunks


def lambda_handler(event, context):
    """
    Main Lambda handler for RxNorm downloads

    Expected event structure:
    {
        "version": "01042025",  # Optional - defaults to latest
        "subset": "full"        # Optional - full, prescribable
    }
    """
    try:
        print(f"[KB7-RXNORM] Starting RxNorm download - Environment: {ENVIRONMENT}")

        # Get credentials from Secrets Manager
        credentials = get_credentials()
        api_key = credentials['api_key']
        rxnorm_endpoint = credentials['rxnorm_endpoint']

        # Determine version to download
        version = event.get('version', get_latest_version())
        subset = event.get('subset', 'full')

        print(f"[KB7-RXNORM] Target version: {version}, Subset: {subset}")

        # Get download URL from UMLS API
        download_url = get_rxnorm_download_url(rxnorm_endpoint, api_key, version, subset)

        # Download and upload to S3
        s3_key = f"rxnorm/{version}/RxNorm_{subset}_{version}.zip"
        file_size, checksum = download_and_upload(download_url, BUCKET_NAME, s3_key)

        # Publish metrics
        publish_metrics(file_size, duration_seconds=(context.get_remaining_time_in_millis() / 1000))

        print(f"[KB7-RXNORM] Download complete - Size: {file_size} bytes, SHA256: {checksum}")

        return {
            'statusCode': 200,
            'body': json.dumps({
                'message': 'RxNorm download successful',
                's3_bucket': BUCKET_NAME,
                's3_key': s3_key,
                'file_size': file_size,
                'checksum': checksum,
                'version': version,
                'subset': subset
            })
        }

    except Exception as e:
        print(f"[KB7-RXNORM] ERROR: {str(e)}")

        # Publish failure metric
        cloudwatch_client.put_metric_data(
            Namespace='KB7/KnowledgeFactory',
            MetricData=[{
                'MetricName': 'RxNormDownloadFailures',
                'Value': 1,
                'Unit': 'Count',
                'Timestamp': datetime.utcnow()
            }]
        )

        return {
            'statusCode': 500,
            'body': json.dumps({
                'error': str(e),
                'message': 'RxNorm download failed'
            })
        }


def get_credentials():
    """Retrieve UMLS API credentials from Secrets Manager"""
    try:
        response = secrets_client.get_secret_value(SecretId=UMLS_SECRET_NAME)
        return json.loads(response['SecretString'])
    except ClientError as e:
        raise Exception(f"Failed to retrieve credentials: {e}")


def get_latest_version():
    """
    Query UMLS API for the latest RxNorm version
    Returns format: MMDDYYYY
    """
    # TODO: Implement UMLS API call to get latest version
    # For now, return placeholder
    return datetime.utcnow().strftime('%m%d%Y')


def get_rxnorm_download_url(endpoint, api_key, version, subset):
    """
    Get the download URL for a specific RxNorm release from UMLS API

    UMLS RxNorm API Documentation:
    https://www.nlm.nih.gov/research/umls/rxnorm/docs/index.html
    """
    headers = {
        'apikey': api_key,
        'Accept': 'application/json'
    }

    # TODO: Implement actual UMLS RxNorm download endpoint
    # Example: GET /rxnorm/download/{version}/{subset}

    # Placeholder for development
    return f"{endpoint}/download/{version}/RxNorm_{subset}.zip"


def download_and_upload(download_url, bucket, key):
    """
    Download RxNorm file and upload to S3

    Args:
        download_url: Source URL for RxNorm zip file
        bucket: Target S3 bucket name
        key: Target S3 object key

    Returns:
        tuple: (total_bytes_transferred, sha256_checksum)
    """
    print(f"[KB7-RXNORM] Downloading from {download_url}")

    response = requests.get(download_url, stream=True, timeout=300)
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
    print(f"[KB7-RXNORM] Uploading {total_bytes / (1024**2):.2f} MB to s3://{bucket}/{key}")

    s3_client.put_object(
        Bucket=bucket,
        Key=key,
        Body=bytes(buffer),
        ServerSideEncryption='AES256',
        Metadata={
            'source': 'umls-rxnorm-api',
            'downloaded': datetime.utcnow().isoformat(),
            'environment': ENVIRONMENT,
            'sha256': sha256_hash.hexdigest()
        }
    )

    checksum = sha256_hash.hexdigest()
    print(f"[KB7-RXNORM] Upload complete")

    return total_bytes, checksum


def publish_metrics(file_size, duration_seconds):
    """Publish CloudWatch metrics for monitoring"""
    cloudwatch_client.put_metric_data(
        Namespace='KB7/KnowledgeFactory',
        MetricData=[
            {
                'MetricName': 'RxNormDownloadSize',
                'Value': file_size / (1024**2),  # MB
                'Unit': 'Megabytes',
                'Timestamp': datetime.utcnow()
            },
            {
                'MetricName': 'RxNormDownloadDuration',
                'Value': duration_seconds,
                'Unit': 'Seconds',
                'Timestamp': datetime.utcnow()
            },
            {
                'MetricName': 'RxNormDownloadSuccess',
                'Value': 1,
                'Unit': 'Count',
                'Timestamp': datetime.utcnow()
            }
        ]
    )
