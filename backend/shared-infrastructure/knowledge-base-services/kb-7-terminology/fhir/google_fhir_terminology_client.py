"""
Google FHIR Terminology Client for KB7 Hybrid Architecture

This module provides a Python client for integrating with Google Cloud Healthcare API
for FHIR terminology operations, complementing the hybrid PostgreSQL/GraphDB architecture.
"""

import json
import asyncio
import logging
from typing import Dict, List, Any, Optional, Union, Tuple
from datetime import datetime, timedelta
import aiohttp
import aioredis
from google.cloud import healthcare_v1
from google.oauth2 import service_account
from google.auth import default
import google.auth.transport.requests
from urllib.parse import urlencode, quote

from .google_config import GoogleFHIRConfig, load_google_fhir_config
from .models import (
    CodeSystemLookupRequest, CodeSystemLookupResponse,
    ValueSetExpandRequest, ConceptMapTranslateRequest,
    ValidateCodeRequest, ValidateCodeResponse,
    OperationOutcome, Parameters
)

logger = logging.getLogger(__name__)


class GoogleFHIRTerminologyError(Exception):
    """Base exception for Google FHIR Terminology operations"""
    pass


class GoogleFHIRAuthenticationError(GoogleFHIRTerminologyError):
    """Authentication-related errors"""
    pass


class GoogleFHIRResourceNotFoundError(GoogleFHIRTerminologyError):
    """Resource not found errors"""
    pass


class GoogleFHIROperationError(GoogleFHIRTerminologyError):
    """FHIR operation execution errors"""
    pass


class GoogleFHIRTerminologyClient:
    """
    Client for Google FHIR Healthcare API terminology operations.

    This client provides FHIR R4-compliant terminology operations using Google Cloud
    Healthcare API, integrated with the KB7 hybrid query router architecture.
    """

    def __init__(self, config: Optional[GoogleFHIRConfig] = None,
                 redis_client: Optional[aioredis.Redis] = None):
        """
        Initialize Google FHIR Terminology Client.

        Args:
            config: GoogleFHIRConfig instance, loads from env if None
            redis_client: Optional Redis client for caching
        """
        self.config = config or load_google_fhir_config()
        self.redis_client = redis_client
        self._healthcare_client = None
        self._credentials = None
        self._session = None
        self._auth_token = None
        self._token_expiry = None

        # Performance tracking
        self._request_count = 0
        self._error_count = 0
        self._cache_hits = 0
        self._cache_misses = 0

        logger.info(
            "Initialized Google FHIR Terminology Client",
            extra={
                "project_id": self.config.project_id,
                "dataset_id": self.config.dataset_id,
                "fhir_store_id": self.config.fhir_store_id,
                "location": self.config.location
            }
        )

    async def __aenter__(self):
        """Async context manager entry."""
        await self._initialize_clients()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        await self._cleanup()

    async def _initialize_clients(self):
        """Initialize Google Healthcare client and authentication."""
        try:
            # Initialize credentials
            await self._initialize_credentials()

            # Initialize HTTP session
            self._session = aiohttp.ClientSession(
                timeout=aiohttp.ClientTimeout(total=self.config.timeout),
                headers={'User-Agent': 'KB7-Terminology-Service/1.0'}
            )

            # Initialize Healthcare client
            self._healthcare_client = healthcare_v1.FhirServiceAsyncClient(
                credentials=self._credentials
            )

            logger.info("Google FHIR clients initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize Google FHIR clients: {e}")
            raise GoogleFHIRAuthenticationError(f"Client initialization failed: {e}")

    async def _initialize_credentials(self):
        """Initialize Google Cloud credentials."""
        try:
            if self.config.credentials_path:
                # Load from service account file
                self._credentials = service_account.Credentials.from_service_account_file(
                    self.config.credentials_path,
                    scopes=['https://www.googleapis.com/auth/cloud-healthcare']
                )
            elif self.config.credentials_json:
                # Load from JSON string
                cred_info = json.loads(self.config.credentials_json)
                self._credentials = service_account.Credentials.from_service_account_info(
                    cred_info,
                    scopes=['https://www.googleapis.com/auth/cloud-healthcare']
                )
            else:
                # Use application default credentials
                self._credentials, _ = default(
                    scopes=['https://www.googleapis.com/auth/cloud-healthcare']
                )

            # Refresh credentials if needed
            if not self._credentials.valid:
                request = google.auth.transport.requests.Request()
                self._credentials.refresh(request)

            logger.info("Google Cloud credentials initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize credentials: {e}")
            raise GoogleFHIRAuthenticationError(f"Credential initialization failed: {e}")

    async def _cleanup(self):
        """Cleanup resources."""
        if self._session:
            await self._session.close()

    async def _get_auth_header(self) -> Dict[str, str]:
        """Get authorization header with fresh token."""
        if not self._credentials:
            await self._initialize_credentials()

        # Check if token needs refresh
        if not self._credentials.valid or (
            self._token_expiry and datetime.now() >= self._token_expiry
        ):
            request = google.auth.transport.requests.Request()
            self._credentials.refresh(request)
            self._token_expiry = datetime.now() + timedelta(minutes=50)  # Refresh 10 min early

        return {"Authorization": f"Bearer {self._credentials.token}"}

    async def _make_request(self, method: str, url: str,
                          params: Optional[Dict] = None,
                          data: Optional[Dict] = None,
                          use_cache: bool = True) -> Dict[str, Any]:
        """
        Make authenticated request to Google FHIR API.

        Args:
            method: HTTP method
            url: Request URL
            params: Query parameters
            data: Request body data
            use_cache: Whether to use Redis cache

        Returns:
            Dict: Response data

        Raises:
            GoogleFHIRTerminologyError: On request failures
        """
        self._request_count += 1
        cache_key = None

        # Try cache for GET requests
        if method == "GET" and use_cache and self.redis_client:
            cache_key = f"gfhir:{hash(f'{url}:{params}')}"
            try:
                cached = await self.redis_client.get(cache_key)
                if cached:
                    self._cache_hits += 1
                    return json.loads(cached)
            except Exception as e:
                logger.warning(f"Cache read error: {e}")

        self._cache_misses += 1

        try:
            headers = await self._get_auth_header()
            headers.update({
                'Content-Type': 'application/fhir+json',
                'Accept': 'application/fhir+json'
            })

            kwargs = {
                'headers': headers,
                'params': params
            }

            if data:
                kwargs['json'] = data

            async with self._session.request(method, url, **kwargs) as response:
                response_text = await response.text()

                if response.status >= 400:
                    self._error_count += 1
                    error_data = {}
                    try:
                        error_data = json.loads(response_text)
                    except json.JSONDecodeError:
                        pass

                    if response.status == 404:
                        raise GoogleFHIRResourceNotFoundError(
                            f"Resource not found: {error_data.get('message', response_text)}"
                        )
                    elif response.status in (401, 403):
                        raise GoogleFHIRAuthenticationError(
                            f"Authentication failed: {error_data.get('message', response_text)}"
                        )
                    else:
                        raise GoogleFHIROperationError(
                            f"Request failed ({response.status}): {error_data.get('message', response_text)}"
                        )

                # Parse response
                try:
                    result = json.loads(response_text)
                except json.JSONDecodeError:
                    raise GoogleFHIROperationError(f"Invalid JSON response: {response_text}")

                # Cache successful GET responses
                if method == "GET" and use_cache and cache_key and self.redis_client:
                    try:
                        await self.redis_client.setex(
                            cache_key,
                            self.config.cache_ttl,
                            json.dumps(result)
                        )
                    except Exception as e:
                        logger.warning(f"Cache write error: {e}")

                return result

        except GoogleFHIRTerminologyError:
            raise
        except Exception as e:
            self._error_count += 1
            logger.error(f"Request failed: {e}")
            raise GoogleFHIROperationError(f"Request execution failed: {e}")

    async def health_check(self) -> Dict[str, Any]:
        """
        Perform health check against Google FHIR store.

        Returns:
            Dict: Health check results
        """
        try:
            # Try to get FHIR store capabilities
            url = f"{self.config.base_url}/metadata"
            start_time = datetime.now()

            response = await self._make_request("GET", url, use_cache=False)

            latency = (datetime.now() - start_time).total_seconds()

            return {
                "status": "healthy",
                "fhir_store": self.config.fhir_store_name,
                "fhir_version": response.get("fhirVersion", "unknown"),
                "software": response.get("software", {}),
                "latency_seconds": latency,
                "request_count": self._request_count,
                "error_count": self._error_count,
                "cache_hit_ratio": self._cache_hits / max(1, self._cache_hits + self._cache_misses),
                "timestamp": datetime.now().isoformat()
            }

        except Exception as e:
            return {
                "status": "unhealthy",
                "error": str(e),
                "fhir_store": self.config.fhir_store_name,
                "timestamp": datetime.now().isoformat()
            }

    async def lookup_code(self, request: CodeSystemLookupRequest) -> CodeSystemLookupResponse:
        """
        Perform CodeSystem $lookup operation.

        Args:
            request: CodeSystem lookup request

        Returns:
            CodeSystemLookupResponse: Lookup results
        """
        # Build operation URL
        if request.system_url:
            # Find CodeSystem by URL first
            search_params = {"url": request.system_url}
            search_url = f"{self.config.base_url}/CodeSystem"

            search_response = await self._make_request("GET", search_url, params=search_params)

            if not search_response.get("entry"):
                raise GoogleFHIRResourceNotFoundError(f"CodeSystem not found: {request.system_url}")

            code_system_id = search_response["entry"][0]["resource"]["id"]
            operation_url = f"{self.config.base_url}/CodeSystem/{code_system_id}/$lookup"
        else:
            # Type-level operation
            operation_url = f"{self.config.base_url}/CodeSystem/$lookup"

        # Build operation parameters
        params = {
            "code": request.code
        }
        if request.system_url:
            params["system"] = request.system_url
        if request.version:
            params["version"] = request.version
        if request.display_language:
            params["displayLanguage"] = request.display_language

        try:
            response = await self._make_request("GET", operation_url, params=params)

            # Convert response to our model
            parameters = response.get("parameter", [])
            param_dict = {p["name"]: p.get("valueString") or p.get("valueBoolean") for p in parameters}

            return CodeSystemLookupResponse(
                name=param_dict.get("name"),
                display=param_dict.get("display"),
                version=param_dict.get("version"),
                designation=param_dict.get("designation", []),
                property=param_dict.get("property", [])
            )

        except GoogleFHIRTerminologyError:
            raise
        except Exception as e:
            raise GoogleFHIROperationError(f"CodeSystem lookup failed: {e}")

    async def expand_valueset(self, request: ValueSetExpandRequest) -> Dict[str, Any]:
        """
        Perform ValueSet $expand operation.

        Args:
            request: ValueSet expansion request

        Returns:
            Dict: Expanded ValueSet
        """
        # Build operation URL
        if request.url:
            # Find ValueSet by URL first
            search_params = {"url": request.url}
            search_url = f"{self.config.base_url}/ValueSet"

            search_response = await self._make_request("GET", search_url, params=search_params)

            if not search_response.get("entry"):
                raise GoogleFHIRResourceNotFoundError(f"ValueSet not found: {request.url}")

            valueset_id = search_response["entry"][0]["resource"]["id"]
            operation_url = f"{self.config.base_url}/ValueSet/{valueset_id}/$expand"
        else:
            # Type-level operation
            operation_url = f"{self.config.base_url}/ValueSet/$expand"

        # Build operation parameters
        params = {}
        if request.url:
            params["url"] = request.url
        if request.filter:
            params["filter"] = request.filter
        if request.count:
            params["count"] = str(request.count)
        if request.offset:
            params["offset"] = str(request.offset)
        if request.include_designations:
            params["includeDesignations"] = "true"

        try:
            response = await self._make_request("GET", operation_url, params=params)
            return response

        except GoogleFHIRTerminologyError:
            raise
        except Exception as e:
            raise GoogleFHIROperationError(f"ValueSet expansion failed: {e}")

    async def translate_concept(self, request: ConceptMapTranslateRequest) -> Dict[str, Any]:
        """
        Perform ConceptMap $translate operation.

        Args:
            request: ConceptMap translation request

        Returns:
            Dict: Translation results
        """
        # Build operation URL
        if request.url:
            # Find ConceptMap by URL first
            search_params = {"url": request.url}
            search_url = f"{self.config.base_url}/ConceptMap"

            search_response = await self._make_request("GET", search_url, params=search_params)

            if not search_response.get("entry"):
                raise GoogleFHIRResourceNotFoundError(f"ConceptMap not found: {request.url}")

            conceptmap_id = search_response["entry"][0]["resource"]["id"]
            operation_url = f"{self.config.base_url}/ConceptMap/{conceptmap_id}/$translate"
        else:
            # Type-level operation
            operation_url = f"{self.config.base_url}/ConceptMap/$translate"

        # Build operation parameters
        params = {
            "code": request.code,
            "system": request.system
        }
        if request.url:
            params["url"] = request.url
        if request.target_system:
            params["targetsystem"] = request.target_system
        if request.version:
            params["version"] = request.version

        try:
            response = await self._make_request("GET", operation_url, params=params)
            return response

        except GoogleFHIRTerminologyError:
            raise
        except Exception as e:
            raise GoogleFHIROperationError(f"ConceptMap translation failed: {e}")

    async def validate_code(self, request: ValidateCodeRequest) -> ValidateCodeResponse:
        """
        Perform $validate-code operation.

        Args:
            request: Code validation request

        Returns:
            ValidateCodeResponse: Validation results
        """
        # Build operation URL based on resource type
        if request.valueset_url:
            # Find ValueSet by URL
            search_params = {"url": request.valueset_url}
            search_url = f"{self.config.base_url}/ValueSet"

            search_response = await self._make_request("GET", search_url, params=search_params)

            if not search_response.get("entry"):
                raise GoogleFHIRResourceNotFoundError(f"ValueSet not found: {request.valueset_url}")

            valueset_id = search_response["entry"][0]["resource"]["id"]
            operation_url = f"{self.config.base_url}/ValueSet/{valueset_id}/$validate-code"
        elif request.codesystem_url:
            # Find CodeSystem by URL
            search_params = {"url": request.codesystem_url}
            search_url = f"{self.config.base_url}/CodeSystem"

            search_response = await self._make_request("GET", search_url, params=search_params)

            if not search_response.get("entry"):
                raise GoogleFHIRResourceNotFoundError(f"CodeSystem not found: {request.codesystem_url}")

            codesystem_id = search_response["entry"][0]["resource"]["id"]
            operation_url = f"{self.config.base_url}/CodeSystem/{codesystem_id}/$validate-code"
        else:
            # Type-level operation
            operation_url = f"{self.config.base_url}/ValueSet/$validate-code"

        # Build operation parameters
        params = {
            "code": request.code
        }
        if request.system:
            params["system"] = request.system
        if request.valueset_url:
            params["url"] = request.valueset_url
        if request.display:
            params["display"] = request.display

        try:
            response = await self._make_request("GET", operation_url, params=params)

            # Extract result from Parameters resource
            parameters = response.get("parameter", [])
            param_dict = {p["name"]: p.get("valueBoolean") or p.get("valueString") for p in parameters}

            return ValidateCodeResponse(
                result=param_dict.get("result", False),
                message=param_dict.get("message"),
                display=param_dict.get("display")
            )

        except GoogleFHIRTerminologyError:
            raise
        except Exception as e:
            raise GoogleFHIROperationError(f"Code validation failed: {e}")

    async def get_statistics(self) -> Dict[str, Any]:
        """
        Get client performance statistics.

        Returns:
            Dict: Performance metrics
        """
        cache_hit_ratio = 0.0
        if self._cache_hits + self._cache_misses > 0:
            cache_hit_ratio = self._cache_hits / (self._cache_hits + self._cache_misses)

        return {
            "request_count": self._request_count,
            "error_count": self._error_count,
            "success_rate": 1.0 - (self._error_count / max(1, self._request_count)),
            "cache_hits": self._cache_hits,
            "cache_misses": self._cache_misses,
            "cache_hit_ratio": cache_hit_ratio,
            "config": {
                "project_id": self.config.project_id,
                "dataset_id": self.config.dataset_id,
                "fhir_store_id": self.config.fhir_store_id,
                "location": self.config.location,
                "base_url": self.config.base_url
            }
        }


async def create_google_fhir_client(config: Optional[GoogleFHIRConfig] = None,
                                  redis_url: Optional[str] = None) -> GoogleFHIRTerminologyClient:
    """
    Factory function to create and initialize Google FHIR client.

    Args:
        config: Optional GoogleFHIRConfig instance
        redis_url: Optional Redis URL for caching

    Returns:
        GoogleFHIRTerminologyClient: Initialized client
    """
    redis_client = None
    if redis_url:
        try:
            redis_client = await aioredis.from_url(redis_url)
        except Exception as e:
            logger.warning(f"Failed to connect to Redis: {e}")

    client = GoogleFHIRTerminologyClient(config=config, redis_client=redis_client)
    await client._initialize_clients()

    return client