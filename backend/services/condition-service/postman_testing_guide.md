# Condition Microservice REST API Testing Guide with Postman

This guide will help you test the Condition Microservice REST API using Postman, including authentication.

## 1. Setting Up Postman Collection

1. **Open Postman** and click on "Collections" in the sidebar
2. **Create a new collection** by clicking the "+" button
3. **Name it** "Condition Service API Tests"
4. **Add collection variables** by clicking on the collection name → "Variables" tab:
   - `base_url`: `http://localhost:8019/api`
   - `auth_service_url`: `http://localhost:8001/api`
   - `auth_token`: Leave empty for now (we'll get this from Auth0)

## 2. Setting Up Authentication

### Option 1: Using a Real Auth Token (Recommended for Production)

1. **Create a new request** in the collection:
   - Method: `POST`
   - URL: `{{auth_service_url}}/auth/token`
   - Body (raw JSON):
     ```json
     {
       "username": "your_username",
       "password": "your_password"
     }
     ```
2. **Send the request** and copy the `access_token` from the response
3. **Set the collection variable** `auth_token` to the copied token

### Option 2: Using Any Token (For Development/Testing)

1. **Set the collection variable** `auth_token` to any string, e.g., `"test-token"`

### Setting Up the Authorization Header

1. **Add an Authorization header** to the collection:
   - Click on the collection name → "Authorization" tab
   - Type: "Bearer Token"
   - Token: `{{auth_token}}`

> **Note:** The condition service is configured to accept any token in development mode, so you can use any string as the token for testing purposes.

## 3. Testing Condition Endpoints

### Create a Condition

1. **Create a new request**:
   - Method: `POST`
   - URL: `{{base_url}}/conditions`
   - Body (raw JSON):
     ```json
     {
       "clinicalStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
             "code": "active",
             "display": "Active"
           }
         ]
       },
       "verificationStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
             "code": "confirmed",
             "display": "Confirmed"
           }
         ]
       },
       "category": [
         {
           "coding": [
             {
               "system": "http://terminology.hl7.org/CodeSystem/condition-category",
               "code": "problem-list-item",
               "display": "Problem List Item"
             }
           ]
         }
       ],
       "code": {
         "coding": [
           {
             "system": "http://snomed.info/sct",
             "code": "73211009",
             "display": "Diabetes mellitus"
           }
         ],
         "text": "Diabetes mellitus"
       },
       "subject": {
         "reference": "Patient/123"
       },
       "onsetDateTime": "2023-01-15",
       "recordedDate": "2023-01-15"
     }
     ```
2. **Send the request** and verify that a new condition is created
3. **Save the condition ID** from the response for later use

### Create a Problem List Item

1. **Create a new request**:
   - Method: `POST`
   - URL: `{{base_url}}/conditions/problems`
   - Body (raw JSON):
     ```json
     {
       "clinicalStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
             "code": "active",
             "display": "Active"
           }
         ]
       },
       "verificationStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
             "code": "confirmed",
             "display": "Confirmed"
           }
         ]
       },
       "category": [],
       "code": {
         "coding": [
           {
             "system": "http://snomed.info/sct",
             "code": "44054006",
             "display": "Diabetes mellitus type 2"
           }
         ],
         "text": "Type 2 Diabetes"
       },
       "subject": {
         "reference": "Patient/123"
       },
       "onsetDateTime": "2023-01-15",
       "recordedDate": "2023-01-15"
     }
     ```
2. **Send the request** and verify that a new problem list item is created
3. **Save the condition ID** from the response for later use

### Create a Diagnosis

1. **Create a new request**:
   - Method: `POST`
   - URL: `{{base_url}}/conditions/diagnoses`
   - Body (raw JSON):
     ```json
     {
       "clinicalStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
             "code": "active",
             "display": "Active"
           }
         ]
       },
       "verificationStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
             "code": "confirmed",
             "display": "Confirmed"
           }
         ]
       },
       "category": [],
       "code": {
         "coding": [
           {
             "system": "http://snomed.info/sct",
             "code": "386661006",
             "display": "Fever"
           }
         ],
         "text": "Fever"
       },
       "subject": {
         "reference": "Patient/123"
       },
       "onsetDateTime": "2023-06-15",
       "recordedDate": "2023-06-15"
     }
     ```
2. **Send the request** and verify that a new diagnosis is created
3. **Save the condition ID** from the response for later use

### Get a Condition by ID

1. **Create a new request**:
   - Method: `GET`
   - URL: `{{base_url}}/conditions/{condition_id}` (replace `{condition_id}` with the ID from the previous step)
2. **Send the request** and verify that the condition details are returned

### Update a Condition

1. **Create a new request**:
   - Method: `PUT`
   - URL: `{{base_url}}/conditions/{condition_id}` (replace `{condition_id}` with the ID from the previous step)
   - Body (raw JSON):
     ```json
     {
       "clinicalStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
             "code": "resolved",
             "display": "Resolved"
           }
         ]
       },
       "verificationStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
             "code": "confirmed",
             "display": "Confirmed"
           }
         ]
       },
       "abatementDateTime": "2023-06-15"
     }
     ```
2. **Send the request** and verify that the condition is updated

### Search for Conditions

1. **Create a new request**:
   - Method: `GET`
   - URL: `{{base_url}}/conditions`
   - Query Parameters:
     - `clinical_status`: `active` (optional)
     - `verification_status`: `confirmed` (optional)
     - `category`: `problem-list-item` (optional)
2. **Send the request** and verify that matching conditions are returned

### Create a Health Concern

1. **Create a new request**:
   - Method: `POST`
   - URL: `{{base_url}}/conditions/health-concerns`
   - Body (raw JSON):
     ```json
     {
       "clinicalStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
             "code": "active",
             "display": "Active"
           }
         ]
       },
       "verificationStatus": {
         "coding": [
           {
             "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
             "code": "confirmed",
             "display": "Confirmed"
           }
         ]
       },
       "category": [],
       "code": {
         "coding": [
           {
             "system": "http://snomed.info/sct",
             "code": "267022002",
             "display": "Tiredness"
           }
         ],
         "text": "Feeling tired"
       },
       "subject": {
         "reference": "Patient/123"
       },
       "onsetDateTime": "2023-06-10",
       "recordedDate": "2023-06-15"
     }
     ```
2. **Send the request** and verify that a new health concern is created
3. **Save the condition ID** from the response for later use

### Get Conditions for a Patient

1. **Create a new request**:
   - Method: `GET`
   - URL: `{{base_url}}/conditions/patient/{patient_id}` (replace `{patient_id}` with a valid patient ID, e.g., `123`)
   - Query Parameters:
     - `clinical_status`: `active` (optional)
     - `verification_status`: `confirmed` (optional)
     - `category`: `problem-list-item` (optional)
2. **Send the request** and verify that the patient's conditions are returned

### Get Problem List Items for a Patient

1. **Create a new request**:
   - Method: `GET`
   - URL: `{{base_url}}/conditions/patient/{patient_id}/problems` (replace `{patient_id}` with a valid patient ID, e.g., `123`)
   - Query Parameters:
     - `clinical_status`: `active` (optional)
     - `verification_status`: `confirmed` (optional)
2. **Send the request** and verify that the patient's problem list items are returned

### Get Diagnoses for a Patient

1. **Create a new request**:
   - Method: `GET`
   - URL: `{{base_url}}/conditions/patient/{patient_id}/diagnoses` (replace `{patient_id}` with a valid patient ID, e.g., `123`)
   - Query Parameters:
     - `clinical_status`: `active` (optional)
     - `verification_status`: `confirmed` (optional)
2. **Send the request** and verify that the patient's diagnoses are returned

### Get Health Concerns for a Patient

1. **Create a new request**:
   - Method: `GET`
   - URL: `{{base_url}}/conditions/patient/{patient_id}/health-concerns` (replace `{patient_id}` with a valid patient ID, e.g., `123`)
   - Query Parameters:
     - `clinical_status`: `active` (optional)
     - `verification_status`: `confirmed` (optional)
2. **Send the request** and verify that the patient's health concerns are returned

### Delete a Condition

1. **Create a new request**:
   - Method: `DELETE`
   - URL: `{{base_url}}/conditions/{condition_id}` (replace `{condition_id}` with the ID from the previous step)
2. **Send the request** and verify that the condition is deleted

## 4. Troubleshooting

### Authentication Issues

If you encounter authentication issues:

1. **Check if your token is valid** by inspecting it at [jwt.io](https://jwt.io/)
2. **Get a new token** if the current one has expired
3. **Verify that the Authorization header** is correctly set in the request

### Connection Issues

If you can't connect to the condition service:

1. **Check if the condition service is running** at `http://localhost:8019`
2. **Verify the request URL** is correct
3. **Check the request headers** - make sure the Authorization header is set correctly
4. **Check the request body** - make sure the JSON is valid
5. **Check the condition service logs** for any errors

### FHIR Server Issues

If you're getting errors related to the FHIR server:

1. **Check if the FHIR server is running** at `http://localhost:8004`
2. **Verify the FHIR server URL** in the condition service configuration
3. **Check the FHIR server logs** for any errors
