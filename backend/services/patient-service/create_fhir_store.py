"""
Script to create a Google Cloud Healthcare API dataset and FHIR store.

This script creates a dataset and FHIR store in Google Cloud Healthcare API
using the service account credentials provided.
"""

import os
import json
import argparse
import requests
from google.oauth2 import service_account

def create_dataset(project_id, location, dataset_id, credentials_path):
    """
    Create a dataset in Google Cloud Healthcare API.
    
    Args:
        project_id: Google Cloud project ID
        location: Google Cloud location (e.g., 'us-central1')
        dataset_id: Dataset ID to create
        credentials_path: Path to service account credentials JSON file
        
    Returns:
        bool: True if the dataset was created successfully, False otherwise
    """
    # Load credentials
    credentials = service_account.Credentials.from_service_account_file(
        credentials_path,
        scopes=['https://www.googleapis.com/auth/cloud-platform']
    )
    
    # Get token
    token = credentials.token
    if not token:
        credentials.refresh(None)
        token = credentials.token
    
    # Create dataset
    url = f"https://healthcare.googleapis.com/v1/projects/{project_id}/locations/{location}/datasets"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    data = {
        "name": f"projects/{project_id}/locations/{location}/datasets/{dataset_id}"
    }
    
    response = requests.post(url, headers=headers, json=data)
    
    if response.status_code == 200 or response.status_code == 201:
        print(f"Dataset '{dataset_id}' created successfully")
        return True
    elif response.status_code == 409:
        print(f"Dataset '{dataset_id}' already exists")
        return True
    else:
        print(f"Error creating dataset: {response.status_code} - {response.text}")
        return False

def create_fhir_store(project_id, location, dataset_id, fhir_store_id, credentials_path):
    """
    Create a FHIR store in Google Cloud Healthcare API.
    
    Args:
        project_id: Google Cloud project ID
        location: Google Cloud location (e.g., 'us-central1')
        dataset_id: Dataset ID
        fhir_store_id: FHIR store ID to create
        credentials_path: Path to service account credentials JSON file
        
    Returns:
        bool: True if the FHIR store was created successfully, False otherwise
    """
    # Load credentials
    credentials = service_account.Credentials.from_service_account_file(
        credentials_path,
        scopes=['https://www.googleapis.com/auth/cloud-platform']
    )
    
    # Get token
    token = credentials.token
    if not token:
        credentials.refresh(None)
        token = credentials.token
    
    # Create FHIR store
    url = f"https://healthcare.googleapis.com/v1/projects/{project_id}/locations/{location}/datasets/{dataset_id}/fhirStores"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    data = {
        "name": f"projects/{project_id}/locations/{location}/datasets/{dataset_id}/fhirStores/{fhir_store_id}",
        "version": "R4",
        "enableUpdateCreate": True,
        "disableReferentialIntegrity": False,
        "disableResourceVersioning": False,
        "enableHistoryImport": True
    }
    
    response = requests.post(url, headers=headers, json=data)
    
    if response.status_code == 200 or response.status_code == 201:
        print(f"FHIR store '{fhir_store_id}' created successfully")
        return True
    elif response.status_code == 409:
        print(f"FHIR store '{fhir_store_id}' already exists")
        return True
    else:
        print(f"Error creating FHIR store: {response.status_code} - {response.text}")
        return False

def main():
    """Main function."""
    parser = argparse.ArgumentParser(description='Create a Google Cloud Healthcare API dataset and FHIR store')
    parser.add_argument('--project-id', required=True, help='Google Cloud project ID')
    parser.add_argument('--location', default='us-central1', help='Google Cloud location')
    parser.add_argument('--dataset-id', default='clinical-synthesis-hub', help='Dataset ID')
    parser.add_argument('--fhir-store-id', default='fhir-store', help='FHIR store ID')
    parser.add_argument('--credentials-path', required=True, help='Path to service account credentials JSON file')
    
    args = parser.parse_args()
    
    # Create dataset
    if create_dataset(args.project_id, args.location, args.dataset_id, args.credentials_path):
        # Create FHIR store
        create_fhir_store(args.project_id, args.location, args.dataset_id, args.fhir_store_id, args.credentials_path)

if __name__ == '__main__':
    main()
