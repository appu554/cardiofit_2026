# gRPC Context Architecture - Complete Implementation

## Overview

This document shows the complete gRPC architecture for connecting the Workflow Engine to the Context Service, providing high-performance, type-safe clinical context assembly.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                           CLINICAL WORKFLOW ENGINE                                  │
│                              (Port 8025)                                           │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                    Workflow Execution Service                               │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │              Context Service gRPC Client                           │   │   │
│  │  │                                                                     │   │   │
│  │  │  REAL gRPC CONNECTION:                                             │   │   │
│  │  │  • Server: localhost:50051                                         │   │   │
│  │  │  • Protocol: HTTP/2 + Protocol Buffers                            │   │   │
│  │  │  • Methods: GetContextByRecipe, GetContextFields,                  │   │   │
│  │  │             ValidateContextAvailability, StreamContextUpdates      │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              │ gRPC/HTTP2 + Protobuf
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                            CONTEXT SERVICE                                         │
│                         (HTTP: 8016, gRPC: 50051)                                 │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                      gRPC Server                                           │   │
│  │                   (Port 50051)                                             │   │
│  │                                                                             │   │
│  │  gRPC SERVICE METHODS:                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │  service ClinicalContextService {                                   │   │   │
│  │  │    rpc GetContextByRecipe(GetContextByRecipeRequest)               │   │   │
│  │  │        returns (ClinicalContextResponse);                          │   │   │
│  │  │    rpc GetContextFields(GetContextFieldsRequest)                   │   │   │
│  │  │        returns (ContextFieldsResponse);                            │   │   │
│  │  │    rpc ValidateContextAvailability(ValidateRequest)                │   │   │
│  │  │        returns (ContextAvailabilityResponse);                      │   │   │
│  │  │    rpc StreamContextUpdates(StreamRequest)                         │   │   │
│  │  │        returns (stream ContextUpdateEvent);                        │   │   │
│  │  │    rpc InvalidateContextCache(InvalidateRequest)                   │   │   │
│  │  │        returns (InvalidateResponse);                               │   │   │
│  │  │  }                                                                  │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                              │                                     │
│                                              ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                Context Assembly Service                                     │   │
│  │                                                                             │   │
│  │  REAL HTTP CONNECTIONS TO DATA SOURCES:                                   │   │
│  │  • Patient Service: HTTP GET localhost:8003                               │   │
│  │  • Medication Service: HTTP GET localhost:8009                            │   │
│  │  • FHIR Store: Google Cloud Healthcare API                                │   │
│  │  • Lab Service: HTTP GET localhost:8000                                   │   │
│  │  • CAE Service: HTTP GET localhost:8027                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              │ REAL HTTP CONNECTIONS
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              REAL DATA SOURCES                                     │
│                                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Patient Service │  │Medication Service│  │   FHIR Store    │  │  Lab Service    │ │
│  │ localhost:8003  │  │ localhost:8009   │  │ Google Cloud    │  │ localhost:8000  │ │
│  │                 │  │                  │  │ Healthcare API  │  │                 │ │
│  │ GET /api/       │  │ GET /api/        │  │ FHIR R4         │  │ GET /api/labs/  │ │
│  │ patients/{id}   │  │ medications/     │  │ REST API        │  │ patient/{id}    │ │
│  │                 │  │ patient/{id}     │  │                 │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                                     │
│  ┌─────────────────┐                                                               │
│  │  CAE Service    │                                                               │
│  │ localhost:8027  │                                                               │
│  │                 │                                                               │
│  │ GET /api/       │                                                               │
│  │ clinical-       │                                                               │
│  │ context/{id}    │                                                               │
│  └─────────────────┘                                                               │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## gRPC Protocol Buffer Definition

### Service Definition
```protobuf
syntax = "proto3";

package clinical_context;

import "google/protobuf/timestamp.proto";
import "google/protobuf/struct.proto";

service ClinicalContextService {
  // Primary method for workflow orchestration
  rpc GetContextByRecipe(GetContextByRecipeRequest) returns (ClinicalContextResponse);
  
  // Field-specific queries for domain services
  rpc GetContextFields(GetContextFieldsRequest) returns (ContextFieldsResponse);
  
  // Availability validation before workflow execution
  rpc ValidateContextAvailability(ValidateContextAvailabilityRequest) returns (ContextAvailabilityResponse);
  
  // Real-time context updates via streaming
  rpc StreamContextUpdates(StreamContextUpdatesRequest) returns (stream ContextUpdateEvent);
  
  // Cache management for real-time data
  rpc InvalidateContextCache(InvalidateContextCacheRequest) returns (InvalidateContextCacheResponse);
  
  // Service health and connectivity
  rpc GetServiceHealth(GetServiceHealthRequest) returns (ServiceHealthResponse);
}
```

### Key Message Types
```protobuf
message GetContextByRecipeRequest {
  string patient_id = 1;
  string recipe_id = 2;
  optional string provider_id = 3;
  optional string encounter_id = 4;
  bool force_refresh = 5;
}

message ClinicalContextResponse {
  string context_id = 1;
  string patient_id = 2;
  string recipe_used = 3;
  google.protobuf.Struct assembled_data = 4;
  double completeness_score = 5;
  google.protobuf.Struct source_metadata = 6;
  repeated SafetyFlag safety_flags = 7;
  ContextStatus status = 8;
  google.protobuf.Timestamp assembled_at = 9;
}

message SafetyFlag {
  SafetyFlagType flag_type = 1;
  SafetySeverity severity = 2;
  string message = 3;
  google.protobuf.Struct details = 4;
}

enum ContextStatus {
  CONTEXT_STATUS_UNKNOWN = 0;
  CONTEXT_STATUS_SUCCESS = 1;
  CONTEXT_STATUS_PARTIAL = 2;
  CONTEXT_STATUS_FAILED = 3;
  CONTEXT_STATUS_UNAVAILABLE = 4;
}
```

## Implementation Files

### 1. Protocol Buffer Definition
**File**: `backend/services/context-service/proto/clinical_context.proto`
- Complete gRPC service definition
- Message types for all operations
- Enums for status and safety flags
- Streaming support for real-time updates

### 2. gRPC Server (Context Service)
**File**: `backend/services/context-service/app/grpc/clinical_context_server.py`
- Implements `ClinicalContextServicer`
- Handles all gRPC method calls
- Integrates with Context Assembly Service
- Provides streaming capabilities

### 3. gRPC Client (Workflow Engine)
**File**: `backend/services/workflow-engine-service/app/services/context_service_grpc_client.py`
- High-performance gRPC client
- Async context manager for connection handling
- Type-safe method calls with protobuf
- Connection pooling and error handling

### 4. Integration with Workflow Engine
**File**: `backend/services/workflow-engine-service/app/services/workflow_execution_service.py`
- Uses gRPC client for context retrieval
- Integrates with Safety Framework
- Handles gRPC errors and fallbacks

## Connection Flow Example

### Medication Ordering Workflow with gRPC:

1. **Workflow Engine gRPC Request**:
   ```python
   async with context_service_grpc_client as grpc_client:
       clinical_context = await grpc_client.get_clinical_context_by_recipe(
           patient_id="905a60cb-8241-418f-b29b-5b020e851392",
           recipe_id="medication-prescribing-v1.0",
           provider_id="provider_123"
       )
   ```

2. **gRPC Call to Context Service**:
   ```
   gRPC Client → gRPC Server (localhost:50051)
   Method: GetContextByRecipe
   Protocol: HTTP/2 + Protocol Buffers
   Request: GetContextByRecipeRequest {
     patient_id: "905a60cb-8241-418f-b29b-5b020e851392"
     recipe_id: "medication-prescribing-v1.0"
     provider_id: "provider_123"
   }
   ```

3. **Context Service Data Assembly**:
   ```
   Context Service → Multiple Real HTTP Services:
   
   HTTP GET localhost:8003/api/patients/905a60cb-8241-418f-b29b-5b020e851392
   ↳ Patient demographics, allergies
   
   HTTP GET localhost:8009/api/medications/patient/905a60cb-8241-418f-b29b-5b020e851392
   ↳ Active medications
   
   HTTP GET localhost:8000/api/labs/patient/905a60cb-8241-418f-b29b-5b020e851392/recent
   ↳ Recent lab results
   
   HTTP GET localhost:8027/api/clinical-context/905a60cb-8241-418f-b29b-5b020e851392
   ↳ Clinical decision support data
   ```

4. **gRPC Response**:
   ```
   gRPC Server → gRPC Client → Workflow Engine
   
   ClinicalContextResponse {
     context_id: "ctx_12345"
     patient_id: "905a60cb-8241-418f-b29b-5b020e851392"
     recipe_used: "medication-prescribing-v1.0"
     assembled_data: { /* real clinical data from all sources */ }
     completeness_score: 0.95
     source_metadata: { /* connection details for each source */ }
     safety_flags: [ /* any clinical safety concerns */ ]
     status: CONTEXT_STATUS_SUCCESS
     assembled_at: "2024-01-20T10:30:00Z"
   }
   ```

## Performance Benefits of gRPC

### 1. **High Performance**
- **HTTP/2**: Multiplexing, header compression, binary protocol
- **Protocol Buffers**: Efficient binary serialization
- **Connection Reuse**: Persistent connections with multiplexing
- **Streaming**: Real-time updates without polling

### 2. **Type Safety**
- **Strong Typing**: Compile-time type checking
- **Code Generation**: Auto-generated client/server stubs
- **Schema Evolution**: Backward/forward compatibility
- **Validation**: Built-in message validation

### 3. **Developer Experience**
- **IDE Support**: Full IntelliSense and autocomplete
- **Documentation**: Self-documenting protocol definitions
- **Testing**: Built-in testing tools and mocking
- **Debugging**: Rich debugging and tracing support

## Deployment Configuration

### Docker Compose
```yaml
services:
  context-service:
    build: ./backend/services/context-service
    ports:
      - "8016:8000"    # HTTP/GraphQL (secondary)
      - "50051:50051"  # gRPC (primary)
    environment:
      - GRPC_PORT=50051
      - HTTP_PORT=8000
    healthcheck:
      test: ["CMD", "grpc_health_probe", "-addr=localhost:50051"]
      interval: 30s
      timeout: 10s
      retries: 3

  workflow-engine:
    build: ./backend/services/workflow-engine-service
    ports:
      - "8025:8000"
    environment:
      - CONTEXT_SERVICE_GRPC_URL=context-service:50051
    depends_on:
      - context-service
```

### Service Discovery
```python
# Environment-based configuration
CONTEXT_SERVICE_GRPC_URL = os.getenv(
    "CONTEXT_SERVICE_GRPC_URL", 
    "localhost:50051"
)

# Load balancing for production
CONTEXT_SERVICE_GRPC_ENDPOINTS = [
    "context-service-1:50051",
    "context-service-2:50051", 
    "context-service-3:50051"
]
```

## Monitoring and Observability

### gRPC Metrics
- **Request Rate**: Requests per second per method
- **Response Time**: P50, P95, P99 latencies
- **Error Rate**: gRPC status code distribution
- **Connection Health**: Active connections and failures

### Clinical Context Metrics
- **Context Assembly Time**: Time to gather all data sources
- **Data Source Latency**: Individual service response times
- **Completeness Score**: Average context completeness
- **Cache Hit Ratio**: Context cache effectiveness

### Alerts
- **gRPC Service Down**: Context Service unavailable
- **High Latency**: Response times > 200ms
- **Low Completeness**: Context completeness < 90%
- **Data Source Failures**: Individual service failures

This gRPC architecture provides high-performance, type-safe, and reliable clinical context assembly for the Clinical Workflow Engine while maintaining strict real data requirements and comprehensive safety mechanisms.
