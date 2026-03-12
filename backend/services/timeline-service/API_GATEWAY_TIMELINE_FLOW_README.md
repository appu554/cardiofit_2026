# Testing the API Gateway > Auth > FHIR > Timeline Flow

This document provides instructions for testing the API Gateway > Auth > FHIR > Timeline flow to ensure that the architecture is correctly implemented.

## Architecture Flow

The correct architecture flow is:

```
API Gateway > Auth > FHIR > Timeline Service
```

## Prerequisites

Before testing, ensure that the following services are running:

1. **API Gateway Service** (port 8005)
2. **Auth Service** (port 8001)
3. **FHIR Service** (port 8014)
4. **Timeline Service** (port 8012)
5. **Other Microservices** (Observation, Condition, Medication, Encounter, etc.)

## Running the Services

### 1. Start the Timeline Service

```bash
cd backend/services/timeline-service
./run_timeline_service.bat  # On Windows
# OR
python run_service.py  # Alternative method
```

### 2. Start the FHIR Service

```bash
cd backend/services/fhir-service
./run_fhir_service.bat  # On Windows
# OR
python run_service.py  # Alternative method
```

### 3. Start the API Gateway

```bash
cd backend/services/api-gateway
./run_api_gateway.bat  # On Windows
# OR
python run_service.py  # Alternative method
```

## Testing with Postman

1. Import the `api_gateway_timeline_flow_postman_collection.json` file into Postman.
2. Set the environment variables:
   - `apiGatewayUrl`: http://localhost:8005
   - `fhirServiceUrl`: http://localhost:8014
   - `timelineServiceUrl`: http://localhost:8012
   - `patientId`: test-patient-1
   - `token`: test_token

### Test 1: Direct Timeline Service

1. Send the "Get Patient Timeline (Direct)" request from the "Direct Timeline Service" folder.
2. Verify that you receive a 200 OK response with a timeline for the patient.
3. Check the Timeline service logs to see that it received the request directly.

### Test 2: FHIR Service

1. Send the "Get Patient Timeline via FHIR" request from the "FHIR Service" folder.
2. Verify that you receive a 200 OK response with a timeline for the patient.
3. Check the FHIR service logs to see that it received the request and forwarded it to the Timeline service.
4. Check the Timeline service logs to see that it received the request from the FHIR service.

### Test 3: API Gateway > FHIR > Timeline Flow (FHIR Path)

1. Send the "Get Patient Timeline via API Gateway (FHIR Path)" request from the "API Gateway" folder.
2. Verify that you receive a 200 OK response with a timeline for the patient.
3. Check the logs in the following order to trace the request flow:
   - API Gateway logs: Should show the request being received and forwarded to the FHIR service
   - FHIR Service logs: Should show the request being received and forwarded to the Timeline service
   - Timeline Service logs: Should show the request being received and processed

### Test 4: API Gateway > FHIR > Timeline Flow (Timeline Path)

1. Send the "Get Patient Timeline via API Gateway (Timeline Path)" request from the "API Gateway" folder.
2. Verify that you receive a 200 OK response with a timeline for the patient.
3. Check the logs in the following order to trace the request flow:
   - API Gateway logs: Should show the request being received, converted to a FHIR path, and forwarded to the FHIR service
   - FHIR Service logs: Should show the request being received and forwarded to the Timeline service
   - Timeline Service logs: Should show the request being received and processed

## Expected Results

### Direct Timeline Service Request

```
GET http://localhost:8012/api/timeline/patients/test-patient-1
Authorization: Bearer test_token
```

### FHIR Service Request

```
GET http://localhost:8014/api/fhir/Patient/test-patient-1/timeline
Authorization: Bearer test_token
```

### API Gateway > FHIR > Timeline Flow (FHIR Path)

```
GET http://localhost:8005/api/fhir/Patient/test-patient-1/timeline
Authorization: Bearer test_token
```

### API Gateway > FHIR > Timeline Flow (Timeline Path)

```
GET http://localhost:8005/api/timeline/patient/test-patient-1
Authorization: Bearer test_token
```

## Troubleshooting

If you encounter issues with the API Gateway > FHIR > Timeline flow:

1. **Check Service Availability**: Ensure all services are running.
2. **Check Logs**: Look for error messages in each service's logs.
3. **Check Configuration**: Verify that the FHIR service is correctly configured to route to the Timeline service.
4. **Check Authentication**: Ensure the token is being properly forwarded through the chain.

## Common Issues and Solutions

1. **404 Not Found**: The API Gateway is not correctly routing to the FHIR service or the FHIR service is not correctly routing to the Timeline service.
   - Solution: Check the API Gateway's proxy.py file to ensure it's correctly routing timeline requests to the FHIR service.
   - Verify that the FHIR service's integration.py file has the correct URL for the Timeline service.
   - Verify that the Timeline service has a /api/fhir/Patient/{id}/timeline endpoint.

2. **401 Unauthorized**: The auth token is not being properly forwarded.
   - Solution: Check the HeaderAuthMiddleware in each service to ensure it's extracting and forwarding the token correctly.

3. **500 Internal Server Error**: There's an error in one of the services.
   - Solution: Check the logs of each service to identify the error.
   - Verify that all services are using the correct ports.

## Port Configuration

Ensure that the services are running on the correct ports:

- API Gateway: 8005
- Auth Service: 8001
- FHIR Service: 8014
- Timeline Service: 8012
- Patient Service: 8003
- Observation Service: 8007
- Condition Service: 8010
- Medication Service: 8009
- Encounter Service: 8011
- Lab Service: 8000
