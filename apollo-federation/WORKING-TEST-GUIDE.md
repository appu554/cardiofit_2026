# Working Gateway Test Guide

This guide provides instructions for testing the Patient Service using a working version of the Apollo Federation Gateway.

## Overview

Instead of using the full federation approach with `IntrospectAndCompose` which is causing errors, this guide uses a simplified gateway that:

1. Uses a static schema for the Patient service
2. Forwards requests directly to the Patient service
3. Maintains the same API contract as the federation approach

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

# 4. Start Working Apollo Federation Gateway
cd apollo-federation
npm run working
```

### 2. Verify the Gateway is Running

Open your browser and navigate to:
- http://localhost:4000/health - Should show a healthy status
- http://localhost:4000/sandbox - Should show the Apollo Sandbox interface

### 3. Test with Postman

You can use the same Postman collection we created earlier:

1. Import `apollo-federation/postman/Patient-Service-Simple-Tests.postman_collection.json`
2. Set up environment variables:
   - `api_gateway_url`: `http://localhost:8000`
   - `federation_gateway_url`: `http://localhost:4000`

3. Run the tests in the following order:
   - Authentication > Get Auth Token
   - Patient Service Tests > Get Patients
   - Patient Service Tests > Create Patient
   - Patient Service Tests > Get Patient by ID
   - Federation Gateway Tests > Get Patients (Direct Federation)

## How This Works

The working gateway uses a static schema approach instead of federation:

1. It defines the Patient schema directly in the gateway
2. It uses resolvers that forward requests to the Patient service
3. It maintains the same GraphQL API that would be available with federation

This approach allows you to test the flow:
```
API Gateway > Auth > Apollo Federation Gateway > Patient Service > Google Healthcare API
```

Without running into the federation-specific errors.

## Moving to Production

For production, you would want to:

1. Fix the federation issues by ensuring all services implement the federation spec correctly
2. Use the proper `IntrospectAndCompose` approach for schema composition
3. Add more services to the federation one by one

But for testing purposes, this working gateway provides a way to verify that the basic flow is functioning correctly.
