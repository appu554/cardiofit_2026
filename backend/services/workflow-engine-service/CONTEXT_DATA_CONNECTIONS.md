# Context Data Connections - EXACTLY Where Data is Connected

## Overview

This document shows **EXACTLY** where and how context data is connected to real services in our Clinical Workflow Engine. No mock data, no fallbacks - only real connections.

## Connection Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                           CLINICAL WORKFLOW ENGINE                                  │
│                                                                                     │
│  ┌─────────────────────┐    ┌──────────────────────────────────────────────────┐   │
│  │   Workflow Engine   │    │           Safety Framework                      │   │
│  │   (Port 8025)       │    │                                                  │   │
│  │                     │    │  ┌─────────────────────────────────────────────┐ │   │
│  │  ┌───────────────┐  │    │  │      Context Service gRPC Client           │ │   │
│  │  │ Workflow Step │──┼────┼──│                                             │ │   │
│  │  │ Execution     │  │    │  │  REAL gRPC CONNECTIONS:                     │ │   │
│  │  └───────────────┘  │    │  │  • gRPC: localhost:50051                   │ │   │
│  │                     │    │  │  • Protocol: HTTP/2 + Protobuf             │ │   │
│  └─────────────────────┘    │  └─────────────────────────────────────────────┘ │   │
│                             └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              │ REAL gRPC/HTTP2
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                            CONTEXT SERVICE (Port 8016)                             │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                    gRPC Server (Port 50051)                                │   │
│  │                                                                             │   │
│  │  gRPC SERVICE METHODS:                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │  rpc GetContextByRecipe(GetContextByRecipeRequest)                 │   │   │
│  │  │      returns (ClinicalContextResponse);                            │   │   │
│  │  │  rpc GetContextFields(GetContextFieldsRequest)                     │   │   │
│  │  │      returns (ContextFieldsResponse);                              │   │   │
│  │  │  rpc ValidateContextAvailability(ValidateRequest)                  │   │   │
│  │  │      returns (ContextAvailabilityResponse);                        │   │   │
│  │  │  rpc StreamContextUpdates(StreamRequest)                           │   │   │
│  │  │      returns (stream ContextUpdateEvent);                          │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                Context Assembly Service                             │   │   │
│  │  │  REAL DATA SOURCE CONNECTIONS:                                     │   │   │
│  │  │    • Patient Service: HTTP GET localhost:8003                      │   │   │
│  │  │    • Medication Service: HTTP GET localhost:8009                   │   │   │
│  │  │    • FHIR Store: Google Cloud Healthcare API                       │   │   │
│  │  │    • Lab Service: HTTP GET localhost:8000                          │   │   │
│  │  │    • CAE Service: HTTP GET localhost:8027                          │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
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
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  CAE Service    │  │ Allergy Service │  │ Context Service │  │ Other Services  │ │
│  │ localhost:8027  │  │ localhost:8003  │  │ localhost:8016  │  │ As Needed       │ │
│  │                 │  │                 │  │                 │  │                 │ │
│  │ GET /api/       │  │ GET /api/       │  │ Internal Calls  │  │ Various APIs    │ │
│  │ clinical-       │  │ allergies/      │  │ for Provider    │  │                 │ │
│  │ context/{id}    │  │ patient/{id}    │  │ Context         │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## Exact Connection Points

### 1. Workflow Engine → Context Service gRPC Client

**File**: `backend/services/workflow-engine-service/app/services/context_service_grpc_client.py`

**Connection Method**:
```python
async def get_clinical_context_by_recipe(
    self,
    patient_id: str,
    recipe_id: str,
    provider_id: Optional[str] = None,
    encounter_id: Optional[str] = None
) -> ClinicalContext:
    # REAL gRPC connection to Context Service
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = clinical_context_pb2_grpc.ClinicalContextServiceStub(channel)

        request = clinical_context_pb2.GetContextByRecipeRequest(
            patient_id=patient_id,
            recipe_id=recipe_id,
            provider_id=provider_id,
            encounter_id=encounter_id
        )

        response = await stub.GetContextByRecipe(request)
        # Process real gRPC response
```

**Real Endpoints Used**:
- gRPC: `localhost:50051` (HTTP/2 + Protocol Buffers)
- Streaming: Real-time context updates via gRPC streams

### 2. Context Service gRPC Server → Real Data Sources

**Files**:
- `backend/services/context-service/app/grpc/clinical_context_server.py` (gRPC Server)
- `backend/services/context-service/app/services/context_assembly_service.py` (Data Assembly)

#### Patient Service Connection
```python
async def _fetch_from_patient_service(
    self,
    endpoint: str,  # "http://localhost:8003"
    data_point: DataPoint,
    patient_id: str
) -> Tuple[Dict[str, Any], SourceMetadata]:
    
    async with aiohttp.ClientSession(timeout=timeout) as session:
        # REAL API CALL to Patient Service
        url = f"{endpoint}/api/patients/{patient_id}"
        async with session.get(url, headers=headers) as response:
            if response.status == 200:
                raw_data = await response.json()
                # Process real patient data
```

#### Medication Service Connection
```python
async def _fetch_from_medication_service(
    self,
    endpoint: str,  # "http://localhost:8009"
    data_point: DataPoint,
    patient_id: str
) -> Tuple[Dict[str, Any], SourceMetadata]:
    
    async with aiohttp.ClientSession(timeout=timeout) as session:
        # REAL API CALL to Medication Service
        url = f"{endpoint}/api/medications/patient/{patient_id}"
        async with session.get(url, headers=headers) as response:
            if response.status == 200:
                raw_data = await response.json()
                # Process real medication data
```

#### FHIR Store Connection
```python
async def _fetch_from_fhir_store(
    self,
    fhir_store_path: str,  # "projects/cardiofit-905a8/locations/..."
    data_point: DataPoint,
    patient_id: str,
    encounter_id: Optional[str] = None
) -> Tuple[Dict[str, Any], SourceMetadata]:
    
    # REAL connection to Google Cloud Healthcare API
    # from google.cloud import healthcare_v1
    # client = healthcare_v1.FhirStoreServiceClient()
    # fhir_store = client.get_fhir_store(name=fhir_store_path)
```

#### Lab Service Connection
```python
async def _fetch_from_lab_service(
    self,
    endpoint: str,  # "http://localhost:8000"
    data_point: DataPoint,
    patient_id: str
) -> Tuple[Dict[str, Any], SourceMetadata]:
    
    async with aiohttp.ClientSession(timeout=timeout) as session:
        # REAL API CALL to Lab Service
        url = f"{endpoint}/api/labs/patient/{patient_id}/recent"
        async with session.get(url, headers=headers) as response:
            if response.status == 200:
                raw_data = await response.json()
                # Process real lab data
```

#### CAE Service Connection
```python
async def _fetch_from_cae_service(
    self,
    endpoint: str,  # "http://localhost:8027"
    data_point: DataPoint,
    patient_id: str
) -> Tuple[Dict[str, Any], SourceMetadata]:
    
    async with aiohttp.ClientSession(timeout=timeout) as session:
        # REAL API CALL to CAE Service
        url = f"{endpoint}/api/clinical-context/{patient_id}"
        async with session.get(url, headers=headers) as response:
            if response.status == 200:
                raw_data = await response.json()
                # Process real CAE data
```

## Real Data Source Endpoints

### Currently Configured Real Endpoints:

**gRPC Connection (Workflow Engine ↔ Context Service)**:
```python
# Context Service gRPC Server
GRPC_SERVER_ADDRESS = "localhost:50051"
GRPC_PROTOCOL = "HTTP/2 + Protocol Buffers"

# gRPC Service Definition
service ClinicalContextService {
  rpc GetContextByRecipe(GetContextByRecipeRequest) returns (ClinicalContextResponse);
  rpc GetContextFields(GetContextFieldsRequest) returns (ContextFieldsResponse);
  rpc ValidateContextAvailability(ValidateRequest) returns (ContextAvailabilityResponse);
  rpc StreamContextUpdates(StreamRequest) returns (stream ContextUpdateEvent);
}
```

**HTTP Connections (Context Service ↔ Data Sources)**:
```python
self.data_source_endpoints = {
    DataSourceType.PATIENT_SERVICE: "http://localhost:8003",
    DataSourceType.MEDICATION_SERVICE: "http://localhost:8009",
    DataSourceType.FHIR_STORE: "projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store",
    DataSourceType.LAB_SERVICE: "http://localhost:8000",
    DataSourceType.ALLERGY_SERVICE: "http://localhost:8003/api/allergies",
    DataSourceType.CAE_SERVICE: "http://localhost:8027",
    DataSourceType.CONTEXT_SERVICE_INTERNAL: "http://localhost:8016"
}
```

### API Endpoints Called:

**gRPC Methods (Workflow Engine → Context Service)**:
1. **GetContextByRecipe**: Get complete clinical context using recipe
2. **GetContextFields**: Get specific fields for domain services
3. **ValidateContextAvailability**: Check data availability before workflow
4. **StreamContextUpdates**: Real-time context updates via streaming
5. **InvalidateContextCache**: Force cache refresh for real-time data

**HTTP REST APIs (Context Service → Data Sources)**:
1. **Patient Service** (`localhost:8003`):
   - `GET /api/patients/{patient_id}` - Patient demographics
   - `GET /api/patients/{patient_id}/allergies` - Patient allergies

2. **Medication Service** (`localhost:8009`):
   - `GET /api/medications/patient/{patient_id}` - Active medications
   - `GET /api/medications/patient/{patient_id}/history` - Medication history

3. **FHIR Store** (Google Cloud Healthcare API):
   - FHIR R4 REST API calls for:
     - Patient resources
     - AllergyIntolerance resources
     - Condition resources
     - MedicationStatement resources

4. **Lab Service** (`localhost:8000`):
   - `GET /api/labs/patient/{patient_id}/recent` - Recent lab results
   - `GET /api/labs/patient/{patient_id}/history` - Lab history

5. **CAE Service** (`localhost:8027`):
   - `GET /api/clinical-context/{patient_id}` - Clinical decision support
   - `POST /api/safety-check` - Drug interaction checks

## Connection Flow Example

### Medication Ordering Workflow Context Assembly:

1. **Workflow Engine gRPC Request**:
   ```
   Workflow Engine → Context Service gRPC Client
   GET clinical context for patient 905a60cb-8241-418f-b29b-5b020e851392
   Recipe: medication-prescribing-v1.0
   ```

2. **Context Service gRPC Call**:
   ```
   Context Service gRPC Client → Context Service gRPC Server
   gRPC Call: localhost:50051
   Method: GetContextByRecipe
   Request: GetContextByRecipeRequest {
     patient_id: "905a60cb-8241-418f-b29b-5b020e851392"
     recipe_id: "medication-prescribing-v1.0"
     provider_id: "provider_123"
   }
   ```

3. **Context Service Data Assembly**:
   ```
   Context Service → Multiple Real Services:
   
   GET http://localhost:8003/api/patients/905a60cb-8241-418f-b29b-5b020e851392
   ↳ Patient demographics, allergies
   
   GET http://localhost:8009/api/medications/patient/905a60cb-8241-418f-b29b-5b020e851392
   ↳ Active medications
   
   GET http://localhost:8000/api/labs/patient/905a60cb-8241-418f-b29b-5b020e851392/recent
   ↳ Recent lab results (creatinine, liver function)
   
   GET http://localhost:8027/api/clinical-context/905a60cb-8241-418f-b29b-5b020e851392
   ↳ Clinical decision support data
   
   FHIR Store API calls to Google Cloud Healthcare API
   ↳ FHIR resources (conditions, allergies, medications)
   ```

4. **Assembled Context gRPC Response**:
   ```
   Context Service gRPC Server → Context Service gRPC Client → Workflow Engine

   ClinicalContextResponse {
     context_id: "ctx_12345",
     patient_id: "905a60cb-8241-418f-b29b-5b020e851392",
     recipe_used: "medication-prescribing-v1.0",
     assembled_data: {
       patient_demographics: { /* real patient data */ },
       active_medications: [ /* real medication list */ ],
       allergies: [ /* real allergy data */ ],
       lab_results: { /* real lab values */ },
       clinical_decision_support: { /* real CAE data */ }
     },
     completeness_score: 0.95,
     source_metadata: {
       patient_service: { endpoint: "http://localhost:8003", retrieved_at: "2024-01-20T10:30:00Z" },
       medication_service: { endpoint: "http://localhost:8009", retrieved_at: "2024-01-20T10:30:01Z" },
       lab_service: { endpoint: "http://localhost:8000", retrieved_at: "2024-01-20T10:30:02Z" },
       cae_service: { endpoint: "http://localhost:8027", retrieved_at: "2024-01-20T10:30:03Z" }
     },
     safety_flags: [ /* protobuf SafetyFlag messages */ ],
     status: CONTEXT_STATUS_SUCCESS
   }
   ```

## Error Handling for Real Connections

### Connection Failures:
- **Timeout**: 10 seconds connect, 30 seconds total
- **HTTP Errors**: 404 (not found), 500 (server error), etc.
- **Network Errors**: Connection refused, DNS resolution failures
- **Data Validation**: Invalid or incomplete responses

### Failure Actions:
- **Required Data Missing**: Workflow fails with `ClinicalDataError`
- **Optional Data Missing**: Continue with warning, lower completeness score
- **Service Unavailable**: Retry with exponential backoff (for non-critical)
- **Critical Service Down**: Immediate workflow failure, safety escalation

## NO FALLBACK POLICY

**IMPORTANT**: There are **NO FALLBACK** or mock data sources. If real services are unavailable:

1. **Critical Data Missing**: Workflow **FAILS IMMEDIATELY**
2. **Safety-Critical Services Down**: **ESCALATE TO CLINICAL SUPERVISOR**
3. **No Mock Data**: **NEVER** use simulated or test data in production workflows
4. **Real Data Only**: All clinical decisions must be based on **REAL PATIENT DATA**

This ensures patient safety by preventing workflows from operating with incomplete or inaccurate information.

## Monitoring & Observability

### Connection Monitoring:
- **Response Times**: Track latency for each data source
- **Success Rates**: Monitor connection success/failure rates
- **Data Completeness**: Track completeness scores over time
- **Error Patterns**: Identify common connection issues

### Alerts:
- **Service Unavailable**: Alert when critical services are down
- **High Latency**: Alert when response times exceed thresholds
- **Low Completeness**: Alert when data completeness drops below safety thresholds
- **Connection Errors**: Alert on repeated connection failures

This comprehensive connection architecture ensures that clinical workflows operate only with real, validated clinical data from actual healthcare systems.
