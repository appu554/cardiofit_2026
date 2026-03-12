# Simple Patient Service Test Guide

This guide provides instructions for testing just the Patient Service through the complete flow:

```
API Gateway > Auth > Apollo Federation Gateway > Patient Service > Google Healthcare API
```

## Prerequisites

Ensure all required services are running:

1. **Auth Service**
2. **API Gateway**
3. **Apollo Federation Gateway** (with simplified configuration)
4. **Patient Service**

## Testing Steps

### 1. Start the Services

Start all services in the following order:

```bash
# 1. Start Auth Service
cd backend/services/auth-service
python main.py

# 2. Start API Gateway
cd backend/services/api-gateway
python main.py

# 3. Start Patient Service
cd backend/services/patient-service
python main.py

# 4. Start Simplified Apollo Federation Gateway
cd apollo-federation
npm run simple
```

### 2. Import the Postman Collection

1. Open Postman
2. Click the "Import" button
3. Select the file: `apollo-federation/postman/Patient-Service-Simple-Tests.postman_collection.json`

### 3. Set Up Environment Variables

1. Create a new environment in Postman
2. Add the following variables:
   - `api_gateway_url`: `http://localhost:8000`
   - `federation_gateway_url`: `http://localhost:4000`

### 4. Run the Tests

Run the tests in the following order:

#### Authentication
1. **Get Auth Token**: This will authenticate with the Auth Service and save the token

#### Patient Service Tests
1. **Get Patients**: This will retrieve a list of patients through the API Gateway
2. **Create Patient**: This will create a new patient and save the ID
3. **Get Patient by ID**: This will retrieve the patient you just created

#### Federation Gateway Tests
1. **Federation Health Check**: This will check the health of the Federation Gateway
2. **Get Patients (Direct Federation)**: This will retrieve patients directly from the Federation Gateway

## Troubleshooting

### Common Issues

1. **Federation Gateway Errors**:
   - If you see `MOCK_AUTH_TOKEN is not defined`, make sure you've updated the code as shown in the fixes
   - If you see `dataSource.process is not a function`, make sure you're using the simplified gateway configuration

2. **Patient Service Errors**:
   - Ensure the Patient Service is running and accessible at `http://localhost:8003`
   - Check that the federation endpoint is available at `http://localhost:8003/api/federation`

3. **Authentication Errors**:
   - Verify that the Auth Service is running and returning valid tokens
   - Check that the token is being properly forwarded through the system

### Logs to Check

- **Apollo Federation Gateway**: Look for errors related to service discovery or schema composition
- **Patient Service**: Check for errors related to request handling or Google Healthcare API interaction
- **API Gateway**: Verify that requests are being properly forwarded to the Federation Gateway

## Moving to Production

After successfully testing the Patient Service, you can:

1. Add other services to the Federation Gateway one by one
2. Test each service individually before adding the next one
3. Once all services are working, update the Federation Gateway to use `IntrospectAndCompose` for production

For production, you'll want to:

1. Ensure all services have proper error handling
2. Add monitoring and logging
3. Implement rate limiting and other security measures
4. Set up health checks for all services
