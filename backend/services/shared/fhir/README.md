# Shared FHIR Router Module

This module provides a standardized FHIR router that can be used by all microservices to handle FHIR requests in a consistent way.

## Overview

The FHIR router module provides:

1. A factory function to create a standardized FHIR router for a specific resource type
2. A base service class that can be extended to implement FHIR operations
3. A mock service implementation for testing

## Usage

### Creating a FHIR Router

To create a FHIR router for a microservice, follow these steps:

1. Create a service class that extends `FHIRServiceBase` or use `MockFHIRService` for testing
2. Create a router configuration using `FHIRRouterConfig`
3. Create the router using the `create_fhir_router` factory function

```python
from fastapi import Depends
from services.shared.fhir import create_fhir_router, FHIRRouterConfig
from app.services.fhir_service import PatientFHIRService
from app.core.auth import get_token_payload

# Create the FHIR router configuration
config = FHIRRouterConfig(
    resource_type="Patient",
    service_class=PatientFHIRService,
    get_token_payload=get_token_payload,
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR Patient"]
)

# Create the router using the factory function
router = create_fhir_router(config)
```

### Implementing a FHIR Service

To implement a FHIR service for a specific resource type, extend the `FHIRServiceBase` class:

```python
from typing import Dict, List, Any, Optional
from services.shared.fhir.service import FHIRServiceBase

class PatientFHIRService(FHIRServiceBase):
    """FHIR service for Patient resources."""
    
    def __init__(self):
        """Initialize the Patient FHIR service."""
        super().__init__("Patient")
        # Initialize your database connection or other resources
    
    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """Create a new Patient resource."""
        # Implement your create logic here
        return resource
    
    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a Patient resource by ID."""
        # Implement your get logic here
        return {"resourceType": "Patient", "id": resource_id}
    
    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Update a Patient resource."""
        # Implement your update logic here
        return resource
    
    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """Delete a Patient resource."""
        # Implement your delete logic here
        return True
    
    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for Patient resources."""
        # Implement your search logic here
        return []
```

### Using the Mock FHIR Service

For testing or rapid prototyping, you can use the `MockFHIRService`:

```python
from services.shared.fhir import create_fhir_router, FHIRRouterConfig, MockFHIRService
from app.core.auth import get_token_payload

# Create the FHIR router configuration with the mock service
config = FHIRRouterConfig(
    resource_type="Patient",
    service_class=MockFHIRService,
    get_token_payload=get_token_payload,
    prefix="",
    tags=["FHIR Patient"]
)

# Create the router using the factory function
router = create_fhir_router(config)
```

## API Endpoints

The FHIR router creates the following endpoints for a resource type:

- `POST /{resource_type}` - Create a new resource
- `GET /{resource_type}/{id}` - Get a resource by ID
- `PUT /{resource_type}/{id}` - Update a resource
- `DELETE /{resource_type}/{id}` - Delete a resource
- `GET /{resource_type}` - Search for resources

## Authentication

The FHIR router automatically extracts the authentication token from the request and passes it to the service methods as an `auth_header` parameter. This allows the service to make authenticated requests to other services if needed.

## Error Handling

The FHIR router includes standardized error handling for all operations. If an operation fails, it returns an appropriate HTTP status code and error message.

## Logging

The FHIR router includes detailed logging for all operations, making it easier to debug issues in production.

## Customization

The FHIR router can be customized by extending the `FHIRServiceBase` class and implementing the required methods. You can also customize the router configuration using the `FHIRRouterConfig` class.
