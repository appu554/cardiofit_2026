"""
FHIR Terminology Endpoints for KB7 Hybrid Architecture

This module provides FastAPI endpoints for FHIR R4 terminology operations
integrated with the hybrid query router architecture. Supports both JSON and XML
FHIR formats with proper error handling and performance optimization.
"""

import json
import logging
from typing import Dict, List, Any, Optional, Union
from datetime import datetime
from fastapi import APIRouter, Depends, HTTPException, Query, Body, Header, Response, Request
from fastapi.responses import JSONResponse, PlainTextResponse
from pydantic import BaseModel, validator
import xml.etree.ElementTree as ET
from xml.dom import minidom

from .terminology_service import FHIRTerminologyService
from .google_fhir_service import GoogleFHIRHybridService, create_hybrid_service, OperationResult
from .google_config import load_google_fhir_config, GoogleFHIRConfig
from .models import (
    CodeSystemLookupRequest, ValueSetExpandRequest, ConceptMapTranslateRequest,
    ValidateCodeRequest, OperationOutcome, Parameters
)

logger = logging.getLogger(__name__)

# Initialize router
router = APIRouter(prefix="/fhir", tags=["FHIR Terminology"])

# Global service instances
terminology_service: Optional[FHIRTerminologyService] = None
google_hybrid_service: Optional[GoogleFHIRHybridService] = None


async def get_terminology_service() -> FHIRTerminologyService:
    """Dependency to get the terminology service instance"""
    global terminology_service
    if terminology_service is None:
        terminology_service = FHIRTerminologyService()
        await terminology_service.__aenter__()
    return terminology_service


async def get_google_hybrid_service() -> GoogleFHIRHybridService:
    """Dependency to get the Google FHIR hybrid service instance"""
    global google_hybrid_service
    if google_hybrid_service is None:
        try:
            google_config = load_google_fhir_config()
            google_hybrid_service = await create_hybrid_service(
                google_config=google_config,
                query_router_url="http://localhost:8087",  # KB7 query router
                redis_url="redis://localhost:6379",
                fallback_strategy="best_effort",
                enable_sync=True
            )
            logger.info("Google FHIR hybrid service initialized successfully")
        except Exception as e:
            logger.warning(f"Failed to initialize Google FHIR service: {e}")
            # Fallback to local service only
            return await get_terminology_service()
    return google_hybrid_service


def get_accept_format(accept: Optional[str] = Header(None)) -> str:
    """Determine response format from Accept header"""
    if accept:
        if "application/fhir+xml" in accept or "application/xml" in accept:
            return "xml"
        elif "application/fhir+json" in accept or "application/json" in accept:
            return "json"
    return "json"  # Default to JSON


def create_fhir_response(data: Dict[str, Any], format_type: str = "json") -> Response:
    """Create properly formatted FHIR response"""
    if format_type == "xml":
        xml_content = dict_to_fhir_xml(data)
        return Response(
            content=xml_content,
            media_type="application/fhir+xml; charset=utf-8",
            headers={
                "Cache-Control": "no-cache",
                "X-Content-Type-Options": "nosniff"
            }
        )
    else:
        return JSONResponse(
            content=data,
            media_type="application/fhir+json; charset=utf-8",
            headers={
                "Cache-Control": "no-cache",
                "X-Content-Type-Options": "nosniff"
            }
        )


def dict_to_fhir_xml(data: Dict[str, Any]) -> str:
    """Convert FHIR dictionary to XML format"""
    resource_type = data.get('resourceType', 'Resource')
    root = ET.Element(resource_type, xmlns="http://hl7.org/fhir")

    def add_element(parent, key, value):
        if value is None:
            return

        if isinstance(value, dict):
            if key == 'resourceType':
                return  # Skip resourceType in XML
            elem = ET.SubElement(parent, key)
            for sub_key, sub_value in value.items():
                add_element(elem, sub_key, sub_value)

        elif isinstance(value, list):
            for item in value:
                if isinstance(item, dict):
                    elem = ET.SubElement(parent, key)
                    for sub_key, sub_value in item.items():
                        add_element(elem, sub_key, sub_value)
                else:
                    elem = ET.SubElement(parent, key)
                    elem.set('value', str(item))

        else:
            elem = ET.SubElement(parent, key)
            elem.set('value', str(value))

    for key, value in data.items():
        add_element(root, key, value)

    # Pretty print XML
    rough_string = ET.tostring(root, 'unicode')
    reparsed = minidom.parseString(rough_string)
    return reparsed.toprettyxml(indent="  ")


class FHIRErrorHandler:
    """Utility class for creating FHIR-compliant error responses"""

    @staticmethod
    def create_operation_outcome(
        severity: str,
        code: str,
        diagnostics: str,
        http_status: int = 400
    ) -> Dict[str, Any]:
        """Create FHIR OperationOutcome for errors"""
        return {
            "resourceType": "OperationOutcome",
            "issue": [{
                "severity": severity,
                "code": code,
                "diagnostics": diagnostics
            }]
        }

    @staticmethod
    def handle_service_error(error: Exception) -> HTTPException:
        """Convert service errors to HTTP exceptions with FHIR OperationOutcome"""
        if "timeout" in str(error).lower():
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "timeout", "Service request timed out", 408
            )
            raise HTTPException(status_code=408, detail=outcome)

        elif "unavailable" in str(error).lower():
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "exception", "Service temporarily unavailable", 503
            )
            raise HTTPException(status_code=503, detail=outcome)

        else:
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "exception", f"Internal error: {str(error)}", 500
            )
            raise HTTPException(status_code=500, detail=outcome)


# Terminology Capability Statement

@router.get("/metadata", response_model=Dict[str, Any])
async def get_terminology_capabilities(
    service: FHIRTerminologyService = Depends(get_terminology_service),
    format_type: str = Depends(get_accept_format)
):
    """
    Get TerminologyCapabilities resource describing service capabilities.

    This endpoint provides metadata about the terminology service including
    supported operations, code systems, and performance characteristics.
    """
    try:
        capabilities = await service.get_terminology_capabilities()
        return create_fhir_response(capabilities, format_type)

    except Exception as e:
        logger.error(f"Error getting capabilities: {e}")
        FHIRErrorHandler.handle_service_error(e)


# CodeSystem Operations

@router.get("/CodeSystem/$lookup", response_model=Dict[str, Any])
@router.post("/CodeSystem/$lookup", response_model=Dict[str, Any])
async def lookup_code_system_concept(
    request: Request,
    system: Optional[str] = Query(None, description="Code system URI"),
    code: Optional[str] = Query(None, description="Code to lookup"),
    version: Optional[str] = Query(None, description="Code system version"),
    displayLanguage: Optional[str] = Query(None, description="Language for display text"),
    property: Optional[List[str]] = Query(None, description="Properties to include"),
    prefer_source: Optional[str] = Query(None, description="Preferred source: 'google' or 'local'"),
    # POST body parameters
    parameters: Optional[Dict[str, Any]] = Body(None),
    format_type: str = Depends(get_accept_format)
):
    """
    CodeSystem $lookup operation.

    Looks up a code in a code system and returns details about the concept.
    Routes to PostgreSQL for fast exact lookups.
    Target response time: <10ms for 95% of requests.

    Parameters can be provided as query parameters (GET) or in request body (POST).
    """
    try:
        # Handle POST with Parameters resource
        if request.method == "POST" and parameters:
            if parameters.get('resourceType') == 'Parameters':
                param_dict = {}
                for param in parameters.get('parameter', []):
                    name = param.get('name')
                    if name == 'system':
                        system = param.get('valueUri')
                    elif name == 'code':
                        code = param.get('valueCode') or param.get('valueString')
                    elif name == 'version':
                        version = param.get('valueString')
                    elif name == 'displayLanguage':
                        displayLanguage = param.get('valueCode')
                    elif name == 'property':
                        if property is None:
                            property = []
                        property.append(param.get('valueCode'))

        # Validate required parameters
        if not system or not code:
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "required", "System and code parameters are required"
            )
            raise HTTPException(status_code=400, detail=outcome)

        # Try Google hybrid service first, fallback to local if needed
        try:
            hybrid_service = await get_google_hybrid_service()
            if isinstance(hybrid_service, GoogleFHIRHybridService):
                # Use Google hybrid service
                lookup_request = CodeSystemLookupRequest(
                    system_url=system,
                    code=code,
                    version=version,
                    display_language=displayLanguage
                )

                operation_result = await hybrid_service.lookup_code(
                    request=lookup_request,
                    prefer_source=prefer_source
                )

                if operation_result.success:
                    # Add metadata to response
                    result = operation_result.data
                    if isinstance(result, dict):
                        result['_metadata'] = {
                            'source': operation_result.source,
                            'latency_ms': operation_result.latency_ms,
                            'cached': operation_result.cached,
                            'fallback_used': operation_result.fallback_used
                        }
                    return create_fhir_response(result, format_type)
                else:
                    # Google hybrid service failed, try local fallback
                    logger.warning(f"Google hybrid lookup failed: {operation_result.error}")

            # Fallback to local service
            local_service = await get_terminology_service()
            result = await local_service.lookup_code_system_concept(
                system=system,
                code=code,
                version=version,
                properties=property,
                display_language=displayLanguage
            )

        except Exception as e:
            logger.warning(f"Hybrid service error, using local fallback: {e}")
            # Final fallback to local service
            local_service = await get_terminology_service()
            result = await local_service.lookup_code_system_concept(
                system=system,
                code=code,
                version=version,
                properties=property,
                display_language=displayLanguage
            )

        # Check for errors in result
        if result.get('resourceType') == 'OperationOutcome':
            for issue in result.get('issue', []):
                if issue.get('severity') == 'error':
                    raise HTTPException(status_code=404, detail=result)

        return create_fhir_response(result, format_type)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"CodeSystem lookup error: {e}")
        FHIRErrorHandler.handle_service_error(e)


# ValueSet Operations

@router.get("/ValueSet/$expand", response_model=Dict[str, Any])
@router.post("/ValueSet/$expand", response_model=Dict[str, Any])
async def expand_value_set(
    request: Request,
    url: Optional[str] = Query(None, description="ValueSet canonical URL"),
    valueSetVersion: Optional[str] = Query(None, description="ValueSet version"),
    context: Optional[str] = Query(None, description="Context for expansion"),
    filter: Optional[str] = Query(None, description="Text filter"),
    count: Optional[int] = Query(50, description="Maximum concepts to return"),
    offset: Optional[int] = Query(0, description="Pagination offset"),
    includeDefinition: Optional[bool] = Query(False, description="Include definitions"),
    includeDesignation: Optional[bool] = Query(False, description="Include designations"),
    activeOnly: Optional[bool] = Query(True, description="Only active concepts"),
    displayLanguage: Optional[str] = Query(None, description="Language for display"),
    # POST body parameters
    parameters: Optional[Dict[str, Any]] = Body(None),
    valueSet: Optional[Dict[str, Any]] = Body(None),
    service: FHIRTerminologyService = Depends(get_terminology_service),
    format_type: str = Depends(get_accept_format)
):
    """
    ValueSet $expand operation.

    Expands a ValueSet to return the list of concepts it contains.
    Routes to GraphDB for semantic expansion with subsumption reasoning.
    Target response time: <50ms for 95% of requests.

    Supports both inline ValueSet (POST) and ValueSet URL (GET/POST).
    """
    try:
        # Handle POST with Parameters resource
        if request.method == "POST" and parameters:
            if parameters.get('resourceType') == 'Parameters':
                for param in parameters.get('parameter', []):
                    name = param.get('name')
                    if name == 'url':
                        url = param.get('valueUri')
                    elif name == 'valueSet':
                        valueSet = param.get('resource')
                    elif name == 'filter':
                        filter = param.get('valueString')
                    elif name == 'count':
                        count = param.get('valueInteger')
                    elif name == 'offset':
                        offset = param.get('valueInteger')
                    elif name == 'includeDefinition':
                        includeDefinition = param.get('valueBoolean')

        # Validate parameters
        if not url and not valueSet:
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "required", "Either url or valueSet parameter is required"
            )
            raise HTTPException(status_code=400, detail=outcome)

        # Perform expansion
        result = await service.expand_value_set(
            url=url,
            value_set=valueSet,
            value_set_version=valueSetVersion,
            filter_text=filter,
            count=count,
            offset=offset,
            include_definition=includeDefinition,
            include_designation=includeDesignation,
            active_only=activeOnly,
            display_language=displayLanguage
        )

        return create_fhir_response(result, format_type)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"ValueSet expansion error: {e}")
        FHIRErrorHandler.handle_service_error(e)


# ConceptMap Operations

@router.get("/ConceptMap/$translate", response_model=Dict[str, Any])
@router.post("/ConceptMap/$translate", response_model=Dict[str, Any])
async def translate_concept_map(
    request: Request,
    url: Optional[str] = Query(None, description="ConceptMap URL"),
    conceptMapVersion: Optional[str] = Query(None, description="ConceptMap version"),
    code: Optional[str] = Query(None, description="Code to translate"),
    system: Optional[str] = Query(None, description="Source system"),
    version: Optional[str] = Query(None, description="Source system version"),
    source: Optional[str] = Query(None, description="Source ValueSet"),
    target: Optional[str] = Query(None, description="Target ValueSet"),
    targetsystem: Optional[str] = Query(None, description="Target system"),
    reverse: Optional[bool] = Query(False, description="Reverse translation"),
    # POST body parameters
    parameters: Optional[Dict[str, Any]] = Body(None),
    service: FHIRTerminologyService = Depends(get_terminology_service),
    format_type: str = Depends(get_accept_format)
):
    """
    ConceptMap $translate operation.

    Translates a code from one terminology to another using concept maps.
    Routes to PostgreSQL for fast mapping lookups.
    Target response time: <15ms for 95% of requests.

    Supports both direct code translation and reverse mapping.
    """
    try:
        # Handle POST with Parameters resource
        if request.method == "POST" and parameters:
            if parameters.get('resourceType') == 'Parameters':
                for param in parameters.get('parameter', []):
                    name = param.get('name')
                    if name == 'url':
                        url = param.get('valueUri')
                    elif name == 'code':
                        code = param.get('valueCode')
                    elif name == 'system':
                        system = param.get('valueUri')
                    elif name == 'target':
                        target = param.get('valueUri')
                    elif name == 'targetsystem':
                        targetsystem = param.get('valueUri')
                    elif name == 'reverse':
                        reverse = param.get('valueBoolean')

        # Validate required parameters
        if not code or not system:
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "required", "Code and system parameters are required"
            )
            raise HTTPException(status_code=400, detail=outcome)

        if not targetsystem and not target:
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "required", "Target system or target ValueSet is required"
            )
            raise HTTPException(status_code=400, detail=outcome)

        # Use targetsystem if provided, otherwise extract from target
        target_system = targetsystem or target

        # Perform translation
        result = await service.translate_concept_map(
            source_system=system,
            source_code=code,
            target_system=target_system,
            concept_map_url=url,
            concept_map_version=conceptMapVersion,
            reverse=reverse
        )

        return create_fhir_response(result, format_type)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"ConceptMap translation error: {e}")
        FHIRErrorHandler.handle_service_error(e)


# Validation Operations

@router.get("/$validate-code", response_model=Dict[str, Any])
@router.post("/$validate-code", response_model=Dict[str, Any])
async def validate_code(
    request: Request,
    url: Optional[str] = Query(None, description="ValueSet URL for validation context"),
    context: Optional[str] = Query(None, description="Validation context"),
    valueSetVersion: Optional[str] = Query(None, description="ValueSet version"),
    code: Optional[str] = Query(None, description="Code to validate"),
    system: Optional[str] = Query(None, description="Code system"),
    version: Optional[str] = Query(None, description="System version"),
    display: Optional[str] = Query(None, description="Display text to validate"),
    date: Optional[str] = Query(None, description="Validation date"),
    abstract: Optional[bool] = Query(None, description="Allow abstract codes"),
    displayLanguage: Optional[str] = Query(None, description="Language for display"),
    # POST body parameters
    parameters: Optional[Dict[str, Any]] = Body(None),
    service: FHIRTerminologyService = Depends(get_terminology_service),
    format_type: str = Depends(get_accept_format)
):
    """
    Terminology $validate-code operation.

    Validates that a code is valid in a given context (code system or value set).
    Uses hybrid routing: simple validation to PostgreSQL, ValueSet validation to GraphDB.
    Target response time: <25ms for 95% of requests.

    Supports validation against code systems or value sets.
    """
    try:
        # Handle POST with Parameters resource
        if request.method == "POST" and parameters:
            if parameters.get('resourceType') == 'Parameters':
                for param in parameters.get('parameter', []):
                    name = param.get('name')
                    if name == 'url':
                        url = param.get('valueUri')
                    elif name == 'code':
                        code = param.get('valueCode')
                    elif name == 'system':
                        system = param.get('valueUri')
                    elif name == 'display':
                        display = param.get('valueString')
                    elif name == 'abstract':
                        abstract = param.get('valueBoolean')

        # Validate required parameters
        if not code:
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "required", "Code parameter is required"
            )
            raise HTTPException(status_code=400, detail=outcome)

        if not system and not url:
            outcome = FHIRErrorHandler.create_operation_outcome(
                "error", "required", "Either system or url parameter is required"
            )
            raise HTTPException(status_code=400, detail=outcome)

        # Parse date if provided
        validation_date = None
        if date:
            try:
                validation_date = datetime.fromisoformat(date.replace('Z', '+00:00'))
            except ValueError:
                outcome = FHIRErrorHandler.create_operation_outcome(
                    "error", "invalid", "Invalid date format"
                )
                raise HTTPException(status_code=400, detail=outcome)

        # Perform validation
        result = await service.validate_code(
            system=system,
            code=code,
            display=display,
            value_set_url=url,
            date=validation_date,
            abstract=abstract
        )

        return create_fhir_response(result, format_type)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Code validation error: {e}")
        FHIRErrorHandler.handle_service_error(e)


# Search Operations

@router.get("/CodeSystem", response_model=Dict[str, Any])
async def search_code_systems(
    url: Optional[str] = Query(None, description="CodeSystem URL"),
    name: Optional[str] = Query(None, description="CodeSystem name"),
    title: Optional[str] = Query(None, description="CodeSystem title"),
    version: Optional[str] = Query(None, description="CodeSystem version"),
    status: Optional[str] = Query(None, description="Publication status"),
    _count: Optional[int] = Query(20, description="Number of results"),
    _offset: Optional[int] = Query(0, description="Pagination offset"),
    service: FHIRTerminologyService = Depends(get_terminology_service),
    format_type: str = Depends(get_accept_format)
):
    """
    Search for CodeSystem resources.

    Returns a Bundle of CodeSystem resources matching the search criteria.
    This is a basic implementation - full FHIR search would require more parameters.
    """
    try:
        # For now, return capabilities as an example
        # In a full implementation, this would search actual CodeSystem resources
        bundle = {
            "resourceType": "Bundle",
            "id": "codesystem-search",
            "type": "searchset",
            "total": 3,
            "entry": [
                {
                    "resource": {
                        "resourceType": "CodeSystem",
                        "id": "snomed-ct",
                        "url": "http://snomed.info/sct",
                        "name": "SNOMED_CT",
                        "title": "SNOMED CT",
                        "status": "active",
                        "version": "20250701"
                    }
                },
                {
                    "resource": {
                        "resourceType": "CodeSystem",
                        "id": "loinc",
                        "url": "http://loinc.org",
                        "name": "LOINC",
                        "title": "Logical Observation Identifiers Names and Codes",
                        "status": "active",
                        "version": "2.76"
                    }
                },
                {
                    "resource": {
                        "resourceType": "CodeSystem",
                        "id": "rxnorm",
                        "url": "http://www.nlm.nih.gov/research/umls/rxnorm",
                        "name": "RxNorm",
                        "title": "RxNorm",
                        "status": "active",
                        "version": "20250101"
                    }
                }
            ]
        }

        return create_fhir_response(bundle, format_type)

    except Exception as e:
        logger.error(f"CodeSystem search error: {e}")
        FHIRErrorHandler.handle_service_error(e)


@router.get("/ValueSet", response_model=Dict[str, Any])
async def search_value_sets(
    url: Optional[str] = Query(None, description="ValueSet URL"),
    name: Optional[str] = Query(None, description="ValueSet name"),
    title: Optional[str] = Query(None, description="ValueSet title"),
    version: Optional[str] = Query(None, description="ValueSet version"),
    status: Optional[str] = Query(None, description="Publication status"),
    _count: Optional[int] = Query(20, description="Number of results"),
    _offset: Optional[int] = Query(0, description="Pagination offset"),
    service: FHIRTerminologyService = Depends(get_terminology_service),
    format_type: str = Depends(get_accept_format)
):
    """
    Search for ValueSet resources.

    Returns a Bundle of ValueSet resources matching the search criteria.
    This is a basic implementation - full FHIR search would require more parameters.
    """
    try:
        # Basic bundle response
        bundle = {
            "resourceType": "Bundle",
            "id": "valueset-search",
            "type": "searchset",
            "total": 0,
            "entry": []
        }

        return create_fhir_response(bundle, format_type)

    except Exception as e:
        logger.error(f"ValueSet search error: {e}")
        FHIRErrorHandler.handle_service_error(e)


# General terminology search

@router.get("/terminology/search", response_model=Dict[str, Any])
async def search_terminology(
    q: str = Query(..., description="Search query text"),
    system: Optional[str] = Query(None, description="Limit to specific system"),
    _count: Optional[int] = Query(20, description="Maximum results"),
    _offset: Optional[int] = Query(0, description="Pagination offset"),
    includeAbstract: Optional[bool] = Query(False, description="Include abstract concepts"),
    activeOnly: Optional[bool] = Query(True, description="Only active concepts"),
    displayLanguage: Optional[str] = Query(None, description="Language for results"),
    service: FHIRTerminologyService = Depends(get_terminology_service),
    format_type: str = Depends(get_accept_format)
):
    """
    Free-text search across terminology concepts.

    Routes to PostgreSQL for full-text search capabilities.
    Target response time: <50ms for 95% of requests.

    Returns a Bundle of matching concepts from various terminology systems.
    """
    try:
        result = await service.search_terminology(
            query=q,
            system=system,
            count=_count,
            offset=_offset,
            include_abstract=includeAbstract,
            active_only=activeOnly,
            display_language=displayLanguage
        )

        return create_fhir_response(result, format_type)

    except Exception as e:
        logger.error(f"Terminology search error: {e}")
        FHIRErrorHandler.handle_service_error(e)


# Monitoring and metrics endpoints

@router.get("/terminology/health", response_model=Dict[str, Any])
async def health_check():
    """
    Health check endpoint for terminology service.

    Returns comprehensive health status including performance metrics,
    circuit breaker status, and backend service availability.
    """
    try:
        # Check Google hybrid service first
        health_status = {
            "timestamp": datetime.now().isoformat(),
            "services": {}
        }

        # Google FHIR Hybrid Service health
        try:
            hybrid_service = await get_google_hybrid_service()
            if isinstance(hybrid_service, GoogleFHIRHybridService):
                google_health = await hybrid_service.health_check()
                health_status["services"]["google_fhir_hybrid"] = google_health
            else:
                health_status["services"]["google_fhir_hybrid"] = {
                    "status": "unavailable",
                    "message": "Google FHIR service not initialized"
                }
        except Exception as e:
            health_status["services"]["google_fhir_hybrid"] = {
                "status": "error",
                "error": str(e)
            }

        # Local terminology service health
        try:
            local_service = await get_terminology_service()
            local_health = await local_service.health_check()
            health_status["services"]["local_terminology"] = local_health
        except Exception as e:
            health_status["services"]["local_terminology"] = {
                "status": "error",
                "error": str(e)
            }

        # Overall health determination
        healthy_services = sum(1 for svc in health_status["services"].values()
                             if svc.get("status") == "healthy")
        total_services = len(health_status["services"])

        health_status["healthy"] = healthy_services > 0  # At least one service healthy
        health_status["overall_status"] = "healthy" if healthy_services == total_services else (
            "degraded" if healthy_services > 0 else "unhealthy"
        )
        health_status["healthy_services"] = f"{healthy_services}/{total_services}"

        status_code = 200 if health_status["healthy"] else 503
        return JSONResponse(content=health_status, status_code=status_code)

    except Exception as e:
        logger.error(f"Health check error: {e}")
        return JSONResponse(
            content={
                "healthy": False,
                "overall_status": "error",
                "timestamp": datetime.now().isoformat(),
                "error": str(e)
            },
            status_code=503
        )


@router.get("/terminology/metrics", response_model=Dict[str, Any])
async def get_performance_metrics(
    service: FHIRTerminologyService = Depends(get_terminology_service)
):
    """
    Get detailed performance metrics for the terminology service.

    Includes response times, cache hit rates, backend utilization,
    and circuit breaker status.
    """
    try:
        metrics = await service.get_performance_metrics()
        return JSONResponse(content=metrics)

    except Exception as e:
        logger.error(f"Metrics error: {e}")
        return JSONResponse(
            content={"error": str(e)},
            status_code=500
        )


# Utility endpoints

@router.get("/terminology/google-stats", response_model=Dict[str, Any])
async def get_google_fhir_statistics():
    """
    Get comprehensive statistics for Google FHIR hybrid service.

    Returns performance metrics, operation counts, cache statistics,
    and service configuration details.
    """
    try:
        # Try to get hybrid service statistics
        try:
            hybrid_service = await get_google_hybrid_service()
            if isinstance(hybrid_service, GoogleFHIRHybridService):
                stats = await hybrid_service.get_statistics()
                return JSONResponse(content=stats, status_code=200)
            else:
                return JSONResponse(
                    content={
                        "status": "unavailable",
                        "message": "Google FHIR hybrid service not initialized",
                        "timestamp": datetime.now().isoformat()
                    },
                    status_code=503
                )
        except Exception as e:
            return JSONResponse(
                content={
                    "status": "error",
                    "error": str(e),
                    "timestamp": datetime.now().isoformat()
                },
                status_code=500
            )

    except Exception as e:
        logger.error(f"Google FHIR statistics error: {e}")
        return JSONResponse(
            content={
                "status": "error",
                "error": str(e),
                "timestamp": datetime.now().isoformat()
            },
            status_code=500
        )


@router.get("/terminology/formats", response_model=Dict[str, Any])
async def get_supported_formats():
    """
    Get information about supported FHIR formats and content types.
    """
    return JSONResponse(content={
        "supported_formats": [
            {
                "format": "JSON",
                "mime_types": ["application/fhir+json", "application/json"],
                "default": True
            },
            {
                "format": "XML",
                "mime_types": ["application/fhir+xml", "application/xml"],
                "default": False
            }
        ],
        "charset": "utf-8"
    })


# Startup event handler
@router.on_event("startup")
async def startup_event():
    """Initialize terminology services on startup"""
    global terminology_service, google_hybrid_service

    # Initialize local terminology service
    if terminology_service is None:
        terminology_service = FHIRTerminologyService(
            query_router_url="http://localhost:8087",
            redis_url="redis://localhost:6379",
            enable_caching=True,
            performance_target_ms=200,
            fallback_enabled=True
        )
        await terminology_service.__aenter__()
        logger.info("FHIR Terminology Service initialized successfully")

    # Initialize Google hybrid service
    if google_hybrid_service is None:
        try:
            google_config = load_google_fhir_config()
            google_hybrid_service = await create_hybrid_service(
                google_config=google_config,
                query_router_url="http://localhost:8087",
                redis_url="redis://localhost:6379",
                fallback_strategy="best_effort",
                enable_sync=True
            )
            logger.info("Google FHIR Hybrid Service initialized successfully")
        except Exception as e:
            logger.warning(f"Failed to initialize Google FHIR hybrid service: {e}")
            logger.info("Continuing with local terminology service only")


# Shutdown event handler
@router.on_event("shutdown")
async def shutdown_event():
    """Clean up terminology services on shutdown"""
    global terminology_service, google_hybrid_service

    # Cleanup Google hybrid service
    if google_hybrid_service is not None:
        try:
            await google_hybrid_service.__aexit__(None, None, None)
            google_hybrid_service = None
            logger.info("Google FHIR Hybrid Service shut down successfully")
        except Exception as e:
            logger.error(f"Error shutting down Google FHIR service: {e}")

    # Cleanup local terminology service
    if terminology_service is not None:
        try:
            await terminology_service.__aexit__(None, None, None)
            terminology_service = None
            logger.info("FHIR Terminology Service shut down successfully")
        except Exception as e:
            logger.error(f"Error shutting down terminology service: {e}")