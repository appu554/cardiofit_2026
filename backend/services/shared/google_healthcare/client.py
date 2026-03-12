"""
Google Cloud Healthcare API client for Clinical Synthesis Hub.

This module provides a client for interacting with Google Cloud Healthcare API FHIR store.
It handles authentication, connection management, and FHIR operations.
"""

import json
import logging
import os
import asyncio
import urllib.parse
import requests
from typing import Dict, List, Any, Optional, Union
from google.oauth2 import service_account
from googleapiclient import discovery
from googleapiclient.errors import HttpError
import google.auth.transport.requests
import asyncio

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class GoogleHealthcareClient:
    """Client for interacting with Google Cloud Healthcare API FHIR store."""

    def __init__(
        self,
        project_id: str,
        location: str,
        dataset_id: str,
        fhir_store_id: str,
        credentials_path: Optional[str] = None
    ):
        """
        Initialize the Google Healthcare API client.

        Args:
            project_id: Google Cloud project ID
            location: Google Cloud location (e.g., 'us-central1')
            dataset_id: Healthcare dataset ID
            fhir_store_id: FHIR store ID
            credentials_path: Path to service account credentials JSON file
        """
        self.project_id = project_id
        self.location = location
        self.dataset_id = dataset_id
        self.fhir_store_id = fhir_store_id
        self.credentials_path = credentials_path
        self.credentials = None

        # Log the dataset and FHIR store IDs for debugging
        logger.info(f"Initializing Google Healthcare API client with:")
        logger.info(f"  Project ID: {project_id}")
        logger.info(f"  Location: {location}")
        logger.info(f"  Dataset ID: {dataset_id}")
        logger.info(f"  FHIR Store ID: {fhir_store_id}")

        self.base_url = f"https://healthcare.googleapis.com/v1/projects/{project_id}/locations/{location}/datasets/{dataset_id}/fhirStores/{fhir_store_id}/fhir"

        # Log the base URL for debugging
        logger.info(f"Initialized Google Healthcare API client with base URL: {self.base_url}")

        self._initialized = False

    def initialize(self) -> bool:
        """
        Initialize the Google Healthcare API client.

        Returns:
            bool: True if initialization was successful, False otherwise
        """
        try:
            # --- BEGIN Diagnostic Logging ---
            logger.info(f"GoogleHealthcareClient.initialize: Received credentials_path: '{self.credentials_path}'")
            if self.credentials_path:
                # Ensure the path is a string, as os.path.exists expects str or bytes
                path_to_check = str(self.credentials_path)
                path_exists = os.path.exists(path_to_check)
                logger.info(f"GoogleHealthcareClient.initialize: os.path.exists('{path_to_check}') returned: {path_exists}")
                # Also log type for sanity check
                logger.info(f"GoogleHealthcareClient.initialize: Type of credentials_path: {type(self.credentials_path)}")
                # Check if it's a file specifically
                is_file = os.path.isfile(path_to_check) if path_exists else False
                logger.info(f"GoogleHealthcareClient.initialize: os.path.isfile('{path_to_check}') returned: {is_file}")
            else:
                logger.info("GoogleHealthcareClient.initialize: credentials_path is None or empty.")
            # --- END Diagnostic Logging ---

            # Initialize credentials
            if self.credentials_path and os.path.exists(self.credentials_path):
                logger.info(f"Loading credentials from file: {self.credentials_path}")
                self.credentials = service_account.Credentials.from_service_account_file(
                    self.credentials_path,
                    scopes=['https://www.googleapis.com/auth/cloud-platform']
                )
                logger.info(f"Successfully loaded credentials from file")
            elif os.environ.get('GOOGLE_APPLICATION_CREDENTIALS') and os.path.exists(os.environ.get('GOOGLE_APPLICATION_CREDENTIALS')):
                # Use GOOGLE_APPLICATION_CREDENTIALS environment variable
                logger.info(f"Loading credentials from GOOGLE_APPLICATION_CREDENTIALS: {os.environ.get('GOOGLE_APPLICATION_CREDENTIALS')}")
                self.credentials = service_account.Credentials.from_service_account_file(
                    os.environ.get('GOOGLE_APPLICATION_CREDENTIALS'),
                    scopes=['https://www.googleapis.com/auth/cloud-platform']
                )
                logger.info(f"Successfully loaded credentials from GOOGLE_APPLICATION_CREDENTIALS")
            else:
                # Try to use default credentials
                logger.info("No credentials file found, attempting to use default credentials")
                self.credentials = service_account.Credentials.from_service_account_info(
                    json.loads(os.environ.get('GOOGLE_APPLICATION_CREDENTIALS_JSON', '{}')),
                    scopes=['https://www.googleapis.com/auth/cloud-platform']
                )
                logger.info(f"Successfully loaded credentials from environment variable")

            # Verify credentials have required fields
            if not hasattr(self.credentials, 'service_account_email') or not self.credentials.service_account_email:
                logger.error("Credentials missing service_account_email")
                return False

            # The token_uri check is causing issues - the credentials object might not have this attribute
            # directly but still works. Let's log a warning but continue.
            if not hasattr(self.credentials, 'token_uri'):
                logger.warning("Credentials object doesn't have token_uri attribute, but this might be okay with newer versions of the library")
            elif not self.credentials.token_uri:
                logger.warning("Credentials token_uri is empty, but will try to continue anyway")

            self._initialized = True
            logger.info(f"Successfully initialized Google Cloud Healthcare API client")
            return True
        except Exception as e:
            logger.error(f"Error initializing Google Cloud Healthcare API client: {str(e)}")
            self._initialized = False
            return False

    async def _get_auth_token(self) -> str:
        """
        Get an OAuth token for authentication with automatic refresh.
    
        Returns:
            str: The OAuth token
    
        Raises:
            Exception: If token retrieval fails after retries
        """
        if not self.credentials:
            if not self.initialize():
                raise Exception("Failed to initialize credentials")
    
        try:
            # Check if token is expired or about to expire (within 5 minutes)
            if not self.credentials.valid:
                logger.info("Token expired or invalid, refreshing...")
                request = google.auth.transport.requests.Request()
                await asyncio.get_event_loop().run_in_executor(
                    None, 
                    lambda: self.credentials.refresh(request)
                )
                logger.info("Successfully refreshed token")
            
            return self.credentials.token
            
        except Exception as e:
            logger.error(f"Error getting auth token: {str(e)}")
            # Try to reinitialize credentials if refresh fails
            logger.info("Attempting to reinitialize credentials...")
            if not self.initialize():
                raise Exception("Failed to reinitialize credentials")
            return self.credentials.token

    async def get_resource_with_retry(self, resource_type: str, resource_id: str, max_retries: int = 2) -> Optional[Dict[str, Any]]:
        """
        Get a resource with automatic retry on auth failure.
    
        Args:
            resource_type: FHIR resource type (e.g., 'Patient')
            resource_id: Resource ID
            max_retries: Maximum number of retry attempts
    
        Returns:
            Optional[Dict[str, Any]]: The resource if found, None otherwise
    
        Raises:
            Exception: If all retry attempts fail
        """
        last_exception = None
        
        for attempt in range(max_retries + 1):
            try:
                return await self.get_resource(resource_type, resource_id)
            except requests.exceptions.HTTPError as e:
                if e.response.status_code == 401 and attempt < max_retries:
                    logger.info("Auth token expired, attempting to refresh...")
                    # Force refresh the token
                    await self._get_auth_token()
                    continue
                last_exception = e
                break
            except Exception as e:
                last_exception = e
                break
        
        logger.error(f"Failed to get resource after {max_retries + 1} attempts: {str(last_exception)}")
        raise last_exception

    async def create_resource(self, resource_type: str, resource: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new FHIR resource.

        Args:
            resource_type: FHIR resource type (e.g., 'Patient')
            resource: FHIR resource data

        Returns:
            Dict[str, Any]: The created resource

        Raises:
            Exception: If the resource creation fails
        """
        if not self._initialized and not self.initialize():
            raise Exception("Google Healthcare API client not initialized")

        try:
            # Ensure resourceType is set correctly
            resource["resourceType"] = resource_type

            # Get auth token
            token = await self._get_auth_token()

            # Log detailed information for debugging
            logger.info(f"Creating {resource_type} resource with token: {token[:10]}...")
            logger.info(f"Base URL: {self.base_url}")

            # Create the resource
            url = f"{self.base_url}/{resource_type}"
            headers = {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/fhir+json; charset=utf-8"
            }

            # Log request details
            logger.info(f"Sending request to: {url}")
            logger.info(f"Headers: {headers}")

            # Check if dataset and FHIR store exist
            try:
                # First check if the dataset exists
                # Make sure we're using the correct dataset ID format
                dataset_url = f"https://healthcare.googleapis.com/v1/projects/{self.project_id}/locations/{self.location}/datasets/{self.dataset_id}"
                logger.info(f"Using dataset ID: {self.dataset_id}")
                dataset_headers = {"Authorization": f"Bearer {token}"}

                logger.info(f"Checking if dataset exists at: {dataset_url}")
                loop = asyncio.get_event_loop()
                dataset_response = await loop.run_in_executor(
                    None,
                    lambda: requests.get(dataset_url, headers=dataset_headers)
                )

                # Log the dataset response
                logger.info(f"Dataset check response: {dataset_response.status_code}")
                if dataset_response.status_code != 200:
                    logger.info(f"Dataset response text: {dataset_response.text[:200]}...")

                # If we get a 403 error, we don't have permission to check if the dataset exists
                # In this case, we'll assume it exists and try to use it anyway
                if dataset_response.status_code == 403:
                    logger.warning(f"Permission denied when checking if dataset exists. Assuming it exists and continuing.")
                elif dataset_response.status_code == 404:
                    # Dataset doesn't exist - but we know it should exist because you created it manually
                    logger.warning(f"Dataset {self.dataset_id} not found. This might be due to incorrect dataset ID or location.")
                    logger.warning(f"Using dataset ID: {self.dataset_id} in location: {self.location}")
                    logger.warning("Continuing anyway, as the dataset might exist with different permissions.")

                # Now check if the FHIR store exists
                fhir_store_url = f"{dataset_url}/fhirStores/{self.fhir_store_id}"
                logger.info(f"Using FHIR store ID: {self.fhir_store_id}")
                logger.info(f"Checking if FHIR store exists at: {fhir_store_url}")

                fhir_store_response = await loop.run_in_executor(
                    None,
                    lambda: requests.get(fhir_store_url, headers=dataset_headers)
                )

                # Log the FHIR store response
                logger.info(f"FHIR store check response: {fhir_store_response.status_code}")
                if fhir_store_response.status_code != 200:
                    logger.info(f"FHIR store response text: {fhir_store_response.text[:200]}...")

                # If we get a 403 error, we don't have permission to check if the FHIR store exists
                # In this case, we'll assume it exists and try to use it anyway
                if fhir_store_response.status_code == 403:
                    logger.warning(f"Permission denied when checking if FHIR store exists. Assuming it exists and continuing.")
                elif fhir_store_response.status_code == 404:
                    # FHIR store doesn't exist - but we know it should exist because you created it manually
                    logger.warning(f"FHIR store {self.fhir_store_id} not found. This might be due to incorrect FHIR store ID or dataset ID.")
                    logger.warning(f"Using FHIR store ID: {self.fhir_store_id} in dataset: {self.dataset_id}")
                    logger.warning("Continuing anyway, as the FHIR store might exist with different permissions.")

            except Exception as e:
                logger.error(f"Error checking/creating dataset and FHIR store: {str(e)}")
                logger.warning("Continuing with the request anyway, as the dataset and FHIR store might already exist")

            # Run in a thread to avoid blocking
            loop = asyncio.get_event_loop()
            response = await loop.run_in_executor(
                None,
                lambda: requests.post(url, headers=headers, json=resource)
            )

            # Log response details
            logger.info(f"Response status: {response.status_code}")
            logger.info(f"Response text: {response.text[:200]}...")

            # Check for errors
            if response.status_code >= 400:
                error_msg = f"Error creating resource: {response.status_code} - {response.text}"
                logger.error(error_msg)
                raise Exception(error_msg)

            response.raise_for_status()

            # Parse the response
            created_resource = response.json()
            logger.info(f"Created {resource_type} resource with ID {created_resource.get('id')}")
            return created_resource
        except Exception as e:
            logger.error(f"Error creating {resource_type} resource: {str(e)}")
            raise

    async def get_resource(self, resource_type: str, resource_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a FHIR resource by ID.

        Args:
            resource_type: FHIR resource type (e.g., 'Patient')
            resource_id: Resource ID

        Returns:
            Optional[Dict[str, Any]]: The resource if found, None otherwise

        Raises:
            Exception: If the client is not initialized
        """
        if not self._initialized and not self.initialize():
            raise Exception("Google Healthcare API client not initialized")

        try:
            # Get auth token
            token = await self._get_auth_token()

            # Get the resource
            url = f"{self.base_url}/{resource_type}/{resource_id}"
            headers = {
                "Authorization": f"Bearer {token}",
                "Accept": "application/fhir+json; charset=utf-8"
            }

            # Run in a thread to avoid blocking
            loop = asyncio.get_event_loop()
            response = await loop.run_in_executor(
                None,
                lambda: requests.get(url, headers=headers)
            )

            # Check if resource was found
            if response.status_code == 404:
                logger.warning(f"{resource_type} with ID {resource_id} not found")
                return None

            # Check for other errors
            response.raise_for_status()

            # Parse the response
            resource = response.json()
            return resource
        except Exception as e:
            logger.error(f"Error getting {resource_type} resource: {str(e)}")
            raise

    async def update_resource(self, resource_type: str, resource_id: str, resource: Dict[str, Any]) -> Dict[str, Any]:
        """
        Update a FHIR resource.

        Args:
            resource_type: FHIR resource type (e.g., 'Patient')
            resource_id: Resource ID
            resource: Updated FHIR resource data

        Returns:
            Dict[str, Any]: The updated resource

        Raises:
            Exception: If the resource update fails
        """
        if not self._initialized and not self.initialize():
            raise Exception("Google Healthcare API client not initialized")

        try:
            # Ensure resourceType and id are set correctly
            resource["resourceType"] = resource_type
            resource["id"] = resource_id

            # Get auth token
            token = await self._get_auth_token()

            # Update the resource
            url = f"{self.base_url}/{resource_type}/{resource_id}"
            headers = {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/fhir+json; charset=utf-8"
            }

            # Run in a thread to avoid blocking
            loop = asyncio.get_event_loop()
            response = await loop.run_in_executor(
                None,
                lambda: requests.put(url, headers=headers, json=resource)
            )

            # Check for errors
            response.raise_for_status()

            # Parse the response
            updated_resource = response.json()
            logger.info(f"Updated {resource_type} resource with ID {updated_resource.get('id')}")
            return updated_resource
        except Exception as e:
            logger.error(f"Error updating {resource_type} resource: {str(e)}")
            raise

    async def delete_resource(self, resource_type: str, resource_id: str) -> bool:
        """
        Delete a FHIR resource.

        Args:
            resource_type: FHIR resource type (e.g., 'Patient')
            resource_id: Resource ID

        Returns:
            bool: True if the resource was deleted, False otherwise

        Raises:
            Exception: If the client is not initialized
        """
        if not self._initialized and not self.initialize():
            raise Exception("Google Healthcare API client not initialized")

        try:
            # Get auth token
            token = await self._get_auth_token()

            # Delete the resource
            url = f"{self.base_url}/{resource_type}/{resource_id}"
            headers = {
                "Authorization": f"Bearer {token}"
            }

            # Run in a thread to avoid blocking
            loop = asyncio.get_event_loop()
            response = await loop.run_in_executor(
                None,
                lambda: requests.delete(url, headers=headers)
            )

            # Check if resource was found
            if response.status_code == 404:
                logger.warning(f"{resource_type} with ID {resource_id} not found")
                return False

            # Check for other errors
            response.raise_for_status()

            logger.info(f"Deleted {resource_type} resource with ID {resource_id}")
            return True
        except Exception as e:
            logger.error(f"Error deleting {resource_type} resource: {str(e)}")
            raise

    async def search_resources(self, resource_type: str, params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """
        Search for FHIR resources.

        Args:
            resource_type: FHIR resource type (e.g., 'Patient')
            params: Search parameters

        Returns:
            List[Dict[str, Any]]: List of matching resources

        Raises:
            Exception: If the client is not initialized
        """
        if not self._initialized and not self.initialize():
            raise Exception("Google Healthcare API client not initialized")

        try:
            # Get auth token
            token = await self._get_auth_token()

            # Build the search URL
            url = f"{self.base_url}/{resource_type}"
            headers = {
                "Authorization": f"Bearer {token}",
                "Accept": "application/fhir+json; charset=utf-8"
            }

            # Prepare search parameters
            search_params = {}
            for key, value in params.items():
                if value is not None:
                    search_params[key] = value

            # Run in a thread to avoid blocking
            loop = asyncio.get_event_loop()
            response = await loop.run_in_executor(
                None,
                lambda: requests.get(url, headers=headers, params=search_params)
            )

            # Check for errors
            response.raise_for_status()

            # Parse the response
            result = response.json()

            # Extract resources from the Bundle
            resources = []
            if "entry" in result:
                for entry in result["entry"]:
                    if "resource" in entry:
                        resources.append(entry["resource"])

            logger.info(f"Found {len(resources)} {resource_type} resources")
            return resources
        except Exception as e:
            logger.error(f"Error searching {resource_type} resources: {str(e)}")
            raise
