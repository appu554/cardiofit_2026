# Updated Patient Service Test Guide

This guide provides instructions for testing the Patient Service using the updated Postman collection.

## Schema Changes

We've updated the schema to match what's expected by the client:

1. Changed mutation arguments from `patientData` to `input`
2. Changed input types from `PatientInput` to `CreatePatientInput` and `UpdatePatientInput`
3. Changed mutation return types to include a wrapper with a `patient` field

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

### 2. Import the Updated Postman Collection

1. Open Postman
2. Click the "Import" button
3. Select the file: `apollo-federation/postman/Patient-Service-Updated-Tests.postman_collection.json`

### 3. Set Up Environment Variables

1. Create a new environment in Postman
2. Add the following variables:
   - `api_gateway_url`: `http://localhost:8005`
   - `federation_gateway_url`: `http://localhost:4000`

### 4. Run the Tests in Sequence

Run the tests in the following order:

#### Authentication
1. **Get Auth Token**: This will authenticate with the Auth Service and save the token

#### Patient Service Tests
1. **Get Patients**: This will retrieve a list of patients through the API Gateway
2. **Create Patient**: This will create a new patient and save the ID
3. **Get Patient by ID**: This will retrieve the patient you just created
4. **Update Patient**: This will update the patient you created

#### Federation Gateway Tests
1. **Federation Health Check**: This will check the health of the Federation Gateway
2. **Get Patients (Direct Federation)**: This will retrieve patients directly from the Federation Gateway

## Example Queries

### Create Patient

```graphql
mutation($input: CreatePatientInput!) {
  createPatient(input: $input) {
    patient {
      id
      name {
        family
        given
      }
      gender
      birthDate
    }
  }
}
```

### Update Patient

```graphql
mutation($id: String!, $input: UpdatePatientInput!) {
  updatePatient(id: $id, input: $input) {
    patient {
      id
      name {
        family
        given
      }
      gender
      birthDate
    }
  }
}
```

### Get Patient

```graphql
query($id: String!) {
  patient(id: $id) {
    id
    name {
      family
      given
    }
    gender
    birthDate
    telecom {
      system
      value
    }
    address {
      line
      city
      state
      postalCode
      country
    }
  }
}
```

## Troubleshooting

If you encounter any issues:

1. Check the logs of the Working Gateway for any errors
2. Verify that the API Gateway is forwarding requests correctly
3. Make sure the Patient Service is running and accessible
4. Ensure you're using the correct authentication token
