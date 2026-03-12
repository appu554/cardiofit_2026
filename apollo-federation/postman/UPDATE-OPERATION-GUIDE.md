# Testing the Update Operation

This guide provides detailed instructions for testing the update operation on patients.

## Prerequisites

Ensure all required services are running:

1. **Auth Service**
2. **API Gateway**
3. **Patient Service**
4. **Working Apollo Federation Gateway**

## Testing Steps

### 1. Create a Patient First

Before you can update a patient, you need to create one:

1. Run the "Get Auth Token" request to authenticate
2. Run the "Create Patient" request to create a new patient
   - This will save the patient ID to the `patient_id` variable

### 2. Update the Patient

Now you can update the patient:

1. Run the "Update Patient" request
   - This uses the `patient_id` variable from the previous step
   - The request updates the patient's name to include "Updated" in the given name

### 3. Verify the Update

To verify that the update was successful:

1. Run the "Get Patient by ID" request
   - This will retrieve the patient you just updated
   - Check that the patient's name now includes "Updated"

## Update Operation Details

The update operation uses a GraphQL mutation with the following format:

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

With variables:

```json
{
  "id": "{{patient_id}}",
  "input": {
    "name": [{
      "family": "Smith",
      "given": ["John", "Updated"]
    }],
    "gender": "male",
    "birthDate": "1980-01-01"
  }
}
```

## How It Works

1. The update mutation is sent to the API Gateway at `/api/graphql`
2. The API Gateway authenticates the request and forwards it to the Apollo Federation Gateway
3. The Apollo Federation Gateway processes the request and forwards it to the Patient Service
4. The Patient Service updates the patient in the Google Healthcare API
5. The response flows back through the chain to the client

## Troubleshooting

If you encounter issues with the update operation:

1. **Check Authentication**:
   - Make sure you have a valid authentication token
   - Verify that the token is being properly forwarded through the system

2. **Check Patient ID**:
   - Ensure the patient ID exists
   - Verify that the ID is correctly set in the `patient_id` variable

3. **Check Request Format**:
   - Make sure you're using the correct mutation format
   - Verify that the input fields match the expected schema

4. **Check Logs**:
   - Look at the Apollo Federation Gateway logs for any errors
   - Check the Patient Service logs for request handling issues
