# SchedulingService Federation Flow Testing

## Overview

This Postman collection tests the **complete architecture flow** for the SchedulingService:

```
API Gateway (8005) → Auth Service (8001) → Apollo Federation Gateway (4000) → SchedulingService (8014) → Google Healthcare API
```

## Architecture Flow Validation

### 1. **API Gateway Layer** (Port 8005)
- Entry point for all requests
- Handles routing and initial request processing
- Forwards GraphQL requests to Apollo Federation Gateway

### 2. **Authentication Service** (Port 8001)
- JWT token generation and validation
- User authentication and authorization
- RBAC (Role-Based Access Control) enforcement

### 3. **Apollo Federation Gateway** (Port 4000)
- GraphQL schema composition from multiple microservices
- Query planning and execution across services
- Type federation and entity resolution

### 4. **SchedulingService** (Port 8014)
- Core scheduling business logic
- FHIR-compliant appointment, schedule, and slot management
- Federation entity extensions (Patient, User)

### 5. **Google Healthcare API**
- FHIR data storage and retrieval
- Healthcare data compliance and security
- Persistent storage for all scheduling resources

## Collection Structure (35+ Requests)

### 1. Authentication Flow (2 requests)
- **Login Doctor** - Get JWT token from Auth Service
- **Verify Token** - Validate token via API Gateway

### 2. Service Health Checks (4 requests)
- **API Gateway Health** - Verify gateway is running
- **Apollo Federation Health** - Check GraphQL schema composition
- **SchedulingService Health** - Validate service availability
- **Federation Schema** - Verify SchedulingService federation integration

### 3. Complete Flow - Appointment Management (4 requests)
- **Create Appointment** - Full flow through all layers
- **Get Appointment** - Retrieve from Google Healthcare API
- **Search Appointments** - Query by patient ID
- **Update Appointment** - Modify appointment status

### 4. Complete Flow - Schedule Management (2 requests)
- **Create Schedule** - Provider schedule creation
- **Get Schedule** - Retrieve schedule details

### 5. Complete Flow - Slot Management (2 requests)
- **Create Slot** - Available time slot creation
- **Search Available Slots** - Find free appointment slots

### 6. Federation Cross-Service Queries (3 requests)
- **Patient with Appointments** - Cross-service data federation
- **User with Schedules** - Practitioner scheduling data
- **Complex Federation Query** - Multi-entity queries

### 7. Google Healthcare API Validation (2 requests)
- **Verify Appointment in Google Healthcare** - Confirm data persistence
- **FHIR Compliance Check** - Validate FHIR resource structure

### 8. Error Handling & Edge Cases (3 requests)
- **Authentication Failure** - Invalid token handling
- **Non-existent Resource** - 404 error scenarios
- **Invalid Data** - Validation error testing

### 9. Performance & Load Testing (2 requests)
- **Bulk Query** - Large dataset retrieval
- **Complex Federation Performance** - Multi-service query performance

## Variables Configuration

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `api_gateway_url` | API Gateway endpoint | `http://localhost:8005` |
| `auth_service_url` | Authentication service | `http://localhost:8001` |
| `apollo_gateway_url` | Apollo Federation Gateway | `http://localhost:4000` |
| `scheduling_service_url` | SchedulingService direct | `http://localhost:8014` |
| `auth_token` | JWT authentication token | Auto-set by login |
| `patient_id` | Test patient identifier | `patient-test-123` |
| `practitioner_id` | Test practitioner identifier | `practitioner-test-456` |
| `appointment_id` | Created appointment ID | Auto-set by create requests |
| `schedule_id` | Created schedule ID | Auto-set by create requests |
| `slot_id` | Created slot ID | Auto-set by create requests |

## Prerequisites

### Services Must Be Running:
1. **Auth Service** - `http://localhost:8001`
2. **Apollo Federation Gateway** - `http://localhost:4000`
3. **API Gateway** - `http://localhost:8005`
4. **SchedulingService** - `http://localhost:8014`
5. **All Federation Services** - patients, observations, medications, organizations, orders

### Google Healthcare API:
- Service account credentials configured
- FHIR store accessible: `projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store`

## Usage Instructions

### 1. Import Collection
1. Open Postman
2. Import `SchedulingService_Federation_Flow.json`
3. Verify all variables are set correctly

### 2. Run Authentication
1. Execute "Login Doctor (Get Auth Token)"
2. Verify token is automatically set in collection variables
3. Run "Verify Token via API Gateway" to confirm authentication

### 3. Health Checks
1. Run all requests in "Service Health Checks" folder
2. Ensure all services return 200 OK
3. Verify federation schema is properly composed

### 4. Complete Flow Testing
1. **Appointment Flow**: Create → Get → Search → Update
2. **Schedule Flow**: Create → Get
3. **Slot Flow**: Create → Search
4. Verify each step completes successfully

### 5. Federation Testing
1. Run cross-service queries
2. Verify data is properly federated across services
3. Test complex multi-entity queries

### 6. Validation & Error Testing
1. Verify Google Healthcare API integration
2. Test FHIR compliance
3. Validate error handling scenarios

## Expected Results

### ✅ Success Indicators:
- All health checks return 200 OK
- Authentication token is properly set and validated
- Appointments are created and stored in Google Healthcare API
- Federation queries return data from multiple services
- FHIR resources are properly structured and compliant

### ❌ Failure Scenarios:
- 401 Unauthorized - Authentication issues
- 404 Not Found - Service unavailable or resource missing
- 500 Internal Server Error - Backend service failures
- GraphQL errors - Schema or query issues

## Troubleshooting

### Common Issues:

1. **Authentication Failures**
   - Verify Auth Service is running on port 8001
   - Check user credentials in login request
   - Ensure token is properly set in variables

2. **Federation Errors**
   - Verify Apollo Gateway is running on port 4000
   - Check all microservices are available
   - Validate supergraph schema composition

3. **Google Healthcare API Issues**
   - Verify service account credentials
   - Check FHIR store path configuration
   - Ensure proper permissions are set

4. **Service Unavailable**
   - Check all required services are running
   - Verify port configurations
   - Review service logs for errors

## Performance Benchmarks

### Expected Response Times:
- **Authentication**: < 500ms
- **Simple Queries**: < 1000ms
- **Federation Queries**: < 2000ms
- **Complex Multi-Service**: < 3000ms

### Load Testing:
- Use bulk queries to test performance
- Monitor response times under load
- Verify federation gateway handles concurrent requests

## Security Testing

The collection includes tests for:
- JWT token validation
- Authorization failures
- Invalid data handling
- Cross-service security boundaries

## FHIR Compliance

All requests validate:
- Proper FHIR resource structure
- Required fields and data types
- Coding systems and terminologies
- Reference integrity across resources
