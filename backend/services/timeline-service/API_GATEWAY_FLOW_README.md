# Testing the API Gateway > Auth > FHIR > Timeline Flow

This document provides instructions for testing the API Gateway > Auth > FHIR > Timeline flow to ensure that the architecture is correctly implemented.

## Architecture Flow

The correct architecture flow is:

```
API Gateway > Auth > FHIR > Microservices
```

For the Timeline service, the flow should be:

```
API Gateway > Auth > FHIR > Timeline Service
```

## Prerequisites

Before testing, ensure that the following services are running:

1. **API Gateway Service** (port 8005)
2. **Auth Service** (port 8001)
3. **FHIR Service** (port 8004)
4. **Timeline Service** (port 8012)
5. **Other Microservices** (Observation, Condition, Medication, Encounter, etc.)

## Running the Timeline Service

To run the Timeline service:

```bash
cd backend/services/timeline-service
./run_timeline_service.bat  # On Windows
# OR
python run_service.py  # Alternative method
```

## Testing with Postman

1. Import the `timeline_service_postman_collection.json` file into Postman.
2. Import the `timeline_service_postman_environment.json` file into Postman.
3. Select the "Timeline Service - Local" environment.

### Test 1: Direct Timeline Service

1. Send the "Get Patient Timeline" request from the "Direct Timeline Service" folder.
2. Verify that you receive a 200 OK response with a timeline for the patient.
3. Check the Timeline service logs to see that it received the request directly.

### Test 2: API Gateway > FHIR > Timeline Flow

1. Send the "Get Patient Timeline via API Gateway" request from the "API Gateway > FHIR > Timeline Flow" folder.
2. Verify that you receive a 200 OK response with a timeline for the patient.
3. Check the logs in the following order to trace the request flow:
   - API Gateway logs: Should show the request being received and forwarded to the FHIR service
   - FHIR Service logs: Should show the request being received and forwarded to the Timeline service
   - Timeline Service logs: Should show the request being received and processed

## Expected Results

### Direct Timeline Service Request

```
GET http://localhost:8012/api/timeline/patients/123
Authorization: Bearer test_token
```

### API Gateway > FHIR > Timeline Flow Request

```
GET http://localhost:8005/api/fhir/Patient/123/timeline
Authorization: Bearer test_token
```

## Troubleshooting

If you encounter issues with the API Gateway > FHIR > Timeline flow:

1. **Check Service Availability**: Ensure all services are running.
2. **Check Logs**: Look for error messages in each service's logs.
3. **Check Configuration**: Verify that the FHIR service is correctly configured to route to the Timeline service.
4. **Check Authentication**: Ensure the token is being properly forwarded through the chain.

## Common Issues

1. **401 Unauthorized**: The auth token is not being properly forwarded.
2. **404 Not Found**: The FHIR service is not correctly routing to the Timeline service.
3. **500 Internal Server Error**: There's an error in one of the services.

## Debugging Tips

- Use the `/health` endpoint on each service to check if they're running correctly.
- Check the FHIR service's `integration.py` file to ensure it has the correct URL for the Timeline service.
- Verify that the API Gateway is correctly routing `/api/fhir/*` requests to the FHIR service.
