"""
Mock Google Cloud Healthcare API v1 for local testing
This allows the service to start without actual Google Cloud credentials
"""
import json
from typing import Any


class UpdateResourceRequest:
    """Mock update resource request"""
    def __init__(self, name, body):
        self.name = name
        self.body = body


class CreateResourceRequest:
    """Mock create resource request"""
    def __init__(self, parent, type_, body):
        self.parent = parent
        self.type_ = type_
        self.body = body


class Response:
    """Mock API response"""
    def __init__(self, data_dict):
        self.data = json.dumps(data_dict).encode('utf-8')


class FhirServiceClient:
    """Mock FHIR Service Client for local development"""

    def __init__(self, credentials=None):
        self.credentials = credentials
        print("⚠️  WARNING: Using MOCK Google Healthcare API (NOT PRODUCTION)")
        print("⚠️  Resources will NOT be persisted to actual FHIR store")

    def update_resource(self, request):
        """Mock update - always succeeds"""
        return Response({
            "resourceType": "OperationOutcome",
            "issue": [{
                "severity": "information",
                "code": "informational",
                "diagnostics": f"MOCK: Updated resource at {request.name}"
            }]
        })

    def create_resource(self, request):
        """Mock create - always succeeds"""
        return Response({
            "resourceType": "OperationOutcome",
            "issue": [{
                "severity": "information",
                "code": "informational",
                "diagnostics": f"MOCK: Created resource {request.type_} at {request.parent}"
            }]
        })
