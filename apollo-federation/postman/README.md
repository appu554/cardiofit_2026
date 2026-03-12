# Postman Collection for Testing Patient Service

This directory contains Postman collections for testing the Patient Service through the complete flow:

```
API Gateway > Auth > Apollo Federation Gateway > Microservices > Google Healthcare API
```

## Setup Instructions

1. Import the collection into Postman:
   - Open Postman
   - Click "Import" button
   - Select the `Patient-Service-Tests.postman_collection.json` file

2. Set up the environment variables:
   - Create a new environment in Postman
   - Add the following variables:
     - `api_gateway_url`: URL of the API Gateway (default: `http://localhost:8000`)
     - `federation_gateway_url`: URL of the Apollo Federation Gateway (default: `http://localhost:4000`)
     - `auth_token`: Will be automatically set after authentication
     - `patient_id`: Will be automatically set after creating a patient
     - `practitioner_id`: ID of a practitioner (default: `123456`)

## Running the Tests

The collection is organized into four folders:

### 1. Authentication

- **Get Auth Token**: Authenticates with the Auth Service and saves the token
- **Verify Token**: Verifies that the token is valid

Run these requests first to get a valid authentication token.

### 2. GraphQL Queries

- **Get Patients**: Retrieves a list of patients
- **Get Patient by ID**: Retrieves a specific patient by ID
- **Get Patients by Practitioner**: Retrieves patients associated with a specific practitioner

These requests test the ability to query patient data through GraphQL.

### 3. GraphQL Mutations

- **Create Patient**: Creates a new patient and saves the ID
- **Update Patient**: Updates an existing patient
- **Delete Patient**: Deletes a patient

These requests test the ability to modify patient data through GraphQL.

### 4. Direct Federation Tests

- **Get Federation Schema**: Retrieves the federation schema
- **Resolve Patient Reference**: Tests entity reference resolution
- **Federation Health Check**: Checks the health of the Federation Gateway

These requests test the Apollo Federation Gateway directly.

## Troubleshooting

### Common Issues

1. **Authentication Errors**:
   - Ensure the Auth Service is running
   - Check that the credentials in the "Get Auth Token" request are correct

2. **GraphQL Errors**:
   - Verify that all services are running
   - Check the logs of the Apollo Federation Gateway for schema errors
   - Ensure the patient ID in the requests is valid

3. **Federation Errors**:
   - Check that the Apollo Federation Gateway is running
   - Verify that the Patient Service has implemented the federation endpoint
   - Look for schema composition errors in the Federation Gateway logs

### Logs to Check

- API Gateway logs for authentication and header forwarding
- Apollo Federation Gateway logs for schema composition and request forwarding
- Patient Service logs for request handling and Google Healthcare API interaction
