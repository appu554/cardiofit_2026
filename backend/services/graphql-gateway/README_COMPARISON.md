# GraphQL Gateway Implementation Comparison

This project provides two different implementations of the GraphQL gateway:

1. **FHIR Service Layer (standalone_server.py)**: Uses a layered architecture with a FHIR microservice
2. **Direct Database Access (direct_db_server.py)**: Connects directly to MongoDB

## Standalone Server with FHIR Layer

The `standalone_server.py` implementation:

- Connects to the FHIR microservice running on port 8004
- Makes HTTP requests to the FHIR service for all database operations
- Follows a more standardized architecture with separation of concerns
- Requires both services to be running

### Running the Standalone Server

```bash
python standalone_server.py
```

The GraphQL endpoint will be available at: http://localhost:8006/graphql

## Direct Database Server

The `direct_db_server.py` implementation:

- Connects directly to MongoDB Atlas
- Performs database operations using MongoDB drivers
- Simplifies the architecture by eliminating the FHIR service layer
- Only requires the GraphQL gateway service to be running

### Running the Direct Database Server

```bash
python direct_db_server.py
```

The GraphQL endpoint will be available at: http://localhost:8006/graphql

## Key Differences

### Architecture
- **Standalone Server**: GraphQL → FHIR service → MongoDB
- **Direct DB Server**: GraphQL → MongoDB

### Performance
- **Standalone Server**: More network hops, higher latency
- **Direct DB Server**: Fewer network hops, lower latency

### Maintenance
- **Standalone Server**: Better separation of concerns, more standardized
- **Direct DB Server**: Simpler architecture, fewer moving parts

### Error Handling
- Both implementations now have robust error handling for missing or malformed data
- Both implementations provide default values for required fields

## Using with Postman

1. Create a new POST request to `http://localhost:8006/graphql`
2. Set the Content-Type header to `application/json`
3. Add the Authorization header with your Auth0 token:
   ```
   Authorization: Bearer your-auth0-token
   ```
4. In the request body, add a GraphQL query:
   ```json
   {
     "query": "query { searchPatients { id name { family given } gender birthDate } }"
   }
   ```

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
    birthDate: "1980-01-01"
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
