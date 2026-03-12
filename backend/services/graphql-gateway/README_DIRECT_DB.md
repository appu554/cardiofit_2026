# Direct Database Access for GraphQL Gateway

This implementation provides direct access to your existing MongoDB Atlas database without going through the FHIR service layer. This is useful for testing and development purposes.

## Files

- `direct_db_server.py`: The main server file that connects directly to MongoDB
- `app/db/mongodb.py`: MongoDB connection and helper functions

## Setup

1. Make sure you have the required dependencies:
   ```
   pip install motor pymongo fastapi uvicorn strawberry-graphql python-dotenv
   ```

2. Your existing `.env` file in the `services/graphql-gateway` directory already contains the MongoDB connection string.

3. Start the server:
   ```
   python direct_db_server.py
   ```

4. The GraphQL endpoint will be available at:
   ```
   http://localhost:8006/graphql
   ```

## Using with Postman

1. Create a new POST request to `http://localhost:8006/graphql`
2. Set the Content-Type header to `application/json`
3. In the request body, add a GraphQL query:
   ```json
   {
     "query": "query { searchPatients { id name { family given } gender birthDate } }"
   }
   ```

4. Send the request and you should see the mock patients in the response

## Available Queries

- `searchPatients`: Get all patients or filter by name, gender, or birthDate
- `patient(id: "...")`: Get a specific patient by ID
- `countPatients`: Get the total number of patients

## Available Mutations

- `createPatient(patientData: {...})`: Create a new patient
- `updatePatient(id: "...", patientData: {...})`: Update an existing patient
- `deletePatient(id: "...")`: Delete a patient

## Example Queries

### Search Patients
```graphql
query {
  searchPatients {
    id
    name {
      family
      given
    }
    gender
    birthDate
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

### Get Patient by ID
```graphql
query {
  patient(id: "your-patient-id") {
    id
    name {
      family
      given
    }
    gender
    birthDate
  }
}
```

### Create Patient
```graphql
mutation {
  createPatient(patientData: {
    identifier: [{ system: "http://example.org/fhir/ids", value: "12346" }],
    name: [{ family: "Doe", given: ["Jane"] }],
    gender: "female",
    birthDate: "1980-01-01",
    address: [{
      line: ["456 Oak Ave"],
      city: "Somewhere",
      state: "NY",
      postalCode: "67890",
      country: "USA"
    }]
  }) {
    id
    name {
      family
      given
    }
    gender
    birthDate
  }
}
```
