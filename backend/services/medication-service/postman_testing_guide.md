# Medication Microservice REST API Testing Guide with Postman

This guide will help you test the Medication Microservice REST API using Postman, including authentication.

## 1. Setting Up Postman Collection

1. **Open Postman** and click on "Collections" in the sidebar
2. **Create a new collection** by clicking the "+" button
3. **Name it** "Medication Service API Tests"
4. **Add collection variables** by clicking on the collection name → "Variables" tab:
   - `base_url`: `http://localhost:8008/api`
   - `auth_service_url`: `http://localhost:8001/api`
   - `auth_token`: Leave empty for now (we'll get this from Auth0)

## 2. Getting an Authentication Token

### Option 1: Using the Auth Service Client Token Endpoint

1. **Create a new request** in the collection
2. **Name it** "Get Auth Token (Client Credentials)"
3. **Set method** to POST
4. **Set URL** to `{{auth_service_url}}/auth/client-token`
5. **Save the request**
6. **Add a test script** to automatically set the token as a collection variable:
```javascript
if (pm.response.code === 200) {
    var jsonData = pm.response.json();
    pm.collectionVariables.set("auth_token", jsonData.access_token);
    console.log("Auth token set successfully");
}
```

### Option 2: Using Username/Password Authentication

1. **Create a new request** in the collection
2. **Name it** "Get Auth Token (Password)"
3. **Set method** to POST
4. **Set URL** to `{{auth_service_url}}/auth/token`
5. **Add query parameters**:
   - `username`: Your Auth0 username (e.g., `test@example.com`)
   - `password`: Your Auth0 password
6. **Save the request**
7. **Add a test script** to automatically set the token as a collection variable:
```javascript
if (pm.response.code === 200) {
    var jsonData = pm.response.json();
    pm.collectionVariables.set("auth_token", jsonData.access_token);
    console.log("Auth token set successfully");
}
```

### Option 3: Using Auth0 Login Page

1. Open `http://localhost:8001/auth0-login.html` in your browser
2. Log in with your Auth0 credentials
3. Copy the token from the page
4. In Postman, go to your collection variables and paste the token as the value for `auth_token`

## 3. Health Check Test

1. **Create a new request** in the collection
2. **Name it** "Health Check"
3. **Set method** to GET
4. **Set URL** to `http://localhost:8008/health`
5. **Save the request**

## 4. Medication Endpoints

### Create Medication

1. **Create a new request** in the collection
2. **Name it** "Create Medication"
3. **Set method** to POST
4. **Set URL** to `{{base_url}}/medications`
5. **Headers**:
   - Content-Type: application/json
   - Authorization: Bearer {{auth_token}}
6. **Body** (raw, JSON):
```json
{
  "code": {
    "coding": [
      {
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "code": "1049502",
        "display": "Acetaminophen 325 MG Oral Tablet"
      }
    ],
    "text": "Acetaminophen 325 MG Oral Tablet"
  }
}
```
7. **Save the request**
8. **Add a test script** to store the medication ID:
```javascript
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.collectionVariables.set("medication_id", jsonData.id);
    console.log("Medication ID set: " + jsonData.id);
}
```

### Get Medication

1. **Create a new request** in the collection
2. **Name it** "Get Medication"
3. **Set method** to GET
4. **Set URL** to `{{base_url}}/medications/{{medication_id}}`
5. **Headers**:
   - Authorization: Bearer {{auth_token}}
6. **Save the request**

### Update Medication

1. **Create a new request** in the collection
2. **Name it** "Update Medication"
3. **Set method** to PUT
4. **Set URL** to `{{base_url}}/medications/{{medication_id}}`
5. **Headers**:
   - Content-Type: application/json
   - Authorization: Bearer {{auth_token}}
6. **Body** (raw, JSON):
```json
{
  "status": "active",
  "code": {
    "coding": [
      {
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "code": "1049502",
        "display": "Acetaminophen 325 MG Oral Tablet [Updated]"
      }
    ],
    "text": "Acetaminophen 325 MG Oral Tablet [Updated]"
  }
}
```
7. **Save the request**

### Search Medications

1. **Create a new request** in the collection
2. **Name it** "Search Medications"
3. **Set method** to GET
4. **Set URL** to `{{base_url}}/medications`
5. **Query Parameters**:
   - name: Acetaminophen
   - _count: 10
   - _page: 1
6. **Headers**:
   - Authorization: Bearer {{auth_token}}
7. **Save the request**

### Delete Medication

1. **Create a new request** in the collection
2. **Name it** "Delete Medication"
3. **Set method** to DELETE
4. **Set URL** to `{{base_url}}/medications/{{medication_id}}`
5. **Headers**:
   - Authorization: Bearer {{auth_token}}
6. **Save the request**

## 5. Medication Request Endpoints

### Create Medication Request

1. **Create a new request** in the collection
2. **Name it** "Create Medication Request"
3. **Set method** to POST
4. **Set URL** to `{{base_url}}/medication-requests`
5. **Headers**:
   - Content-Type: application/json
   - Authorization: Bearer {{auth_token}}
6. **Body** (raw, JSON):
```json
{
  "status": "active",
  "intent": "order",
  "medicationCodeableConcept": {
    "coding": [
      {
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "code": "1049502",
        "display": "Acetaminophen 325 MG Oral Tablet"
      }
    ],
    "text": "Acetaminophen 325 MG Oral Tablet"
  },
  "subject": {
    "reference": "Patient/123"
  },
  "authoredOn": "2023-06-15T08:00:00",
  "dosageInstruction": [
    {
      "text": "Take 1 tablet by mouth every 4-6 hours as needed for pain",
      "timing": {
        "code": {
          "text": "Every 4-6 hours as needed"
        }
      }
    }
  ]
}
```
7. **Save the request**
8. **Add a test script** to store the medication request ID:
```javascript
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.collectionVariables.set("medication_request_id", jsonData.id);
    console.log("Medication Request ID set: " + jsonData.id);
}
```

### Get Medication Request

1. **Create a new request** in the collection
2. **Name it** "Get Medication Request"
3. **Set method** to GET
4. **Set URL** to `{{base_url}}/medication-requests/{{medication_request_id}}`
5. **Headers**:
   - Authorization: Bearer {{auth_token}}
6. **Save the request**

### Get Patient Medication Requests

1. **Create a new request** in the collection
2. **Name it** "Get Patient Medication Requests"
3. **Set method** to GET
4. **Set URL** to `{{base_url}}/medication-requests/patient/123`
5. **Headers**:
   - Authorization: Bearer {{auth_token}}
6. **Save the request**

## 6. Medication Administration Endpoints

### Create Medication Administration

1. **Create a new request** in the collection
2. **Name it** "Create Medication Administration"
3. **Set method** to POST
4. **Set URL** to `{{base_url}}/medication-administrations`
5. **Headers**:
   - Content-Type: application/json
   - Authorization: Bearer {{auth_token}}
6. **Body** (raw, JSON):
```json
{
  "status": "completed",
  "medicationCodeableConcept": {
    "coding": [
      {
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "code": "1049502",
        "display": "Acetaminophen 325 MG Oral Tablet"
      }
    ],
    "text": "Acetaminophen 325 MG Oral Tablet"
  },
  "subject": {
    "reference": "Patient/123"
  },
  "effectiveDateTime": "2023-06-15T10:00:00",
  "performer": [
    {
      "actor": {
        "reference": "Practitioner/456",
        "display": "Dr. Jane Smith"
      }
    }
  ],
  "request": {
    "reference": "MedicationRequest/{{medication_request_id}}"
  },
  "dosage": {
    "text": "1 tablet",
    "dose": {
      "value": 1,
      "unit": "tablet"
    }
  }
}
```
7. **Save the request**
8. **Add a test script** to store the medication administration ID:
```javascript
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.collectionVariables.set("medication_administration_id", jsonData.id);
    console.log("Medication Administration ID set: " + jsonData.id);
}
```

### Get Patient Medication Administrations

1. **Create a new request** in the collection
2. **Name it** "Get Patient Medication Administrations"
3. **Set method** to GET
4. **Set URL** to `{{base_url}}/medication-administrations/patient/123`
5. **Headers**:
   - Authorization: Bearer {{auth_token}}
6. **Save the request**

## 7. Medication Statement Endpoints

### Create Medication Statement

1. **Create a new request** in the collection
2. **Name it** "Create Medication Statement"
3. **Set method** to POST
4. **Set URL** to `{{base_url}}/medication-statements`
5. **Headers**:
   - Content-Type: application/json
   - Authorization: Bearer {{auth_token}}
6. **Body** (raw, JSON):
```json
{
  "status": "active",
  "medicationCodeableConcept": {
    "coding": [
      {
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "code": "1049502",
        "display": "Acetaminophen 325 MG Oral Tablet"
      }
    ],
    "text": "Acetaminophen 325 MG Oral Tablet"
  },
  "subject": {
    "reference": "Patient/123"
  },
  "effectiveDateTime": "2023-06-15T10:00:00",
  "dateAsserted": "2023-06-15T10:30:00",
  "informationSource": {
    "reference": "Patient/123"
  },
  "dosage": [
    {
      "text": "Take 1 tablet by mouth every 4-6 hours as needed for pain"
    }
  ]
}
```
7. **Save the request**
8. **Add a test script** to store the medication statement ID:
```javascript
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.collectionVariables.set("medication_statement_id", jsonData.id);
    console.log("Medication Statement ID set: " + jsonData.id);
}
```

### Get Patient Medication Statements

1. **Create a new request** in the collection
2. **Name it** "Get Patient Medication Statements"
3. **Set method** to GET
4. **Set URL** to `{{base_url}}/medication-statements/patient/123`
5. **Headers**:
   - Authorization: Bearer {{auth_token}}
6. **Save the request**

## 8. HL7 Message Processing

### Process HL7 RDE Message

1. **Create a new request** in the collection
2. **Name it** "Process HL7 RDE Message"
3. **Set method** to POST
4. **Set URL** to `{{base_url}}/hl7/rde`
5. **Headers**:
   - Content-Type: application/json
   - Authorization: Bearer {{auth_token}}
6. **Body** (raw, JSON):
```json
{
  "message": "MSH|^~\\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20230615080000||RDE^O11|MSGID123|P|2.5.1|\nPID|||123^^^MRN||DOE^JOHN||19700101|M||\nORC|NW|ORDER123||||||20230615080000|||DOCTOR^JOHN^A|\nRXE||1049502^Acetaminophen 325 MG Oral Tablet^RXNORM|1|TAB|Q4-6H PRN||||\nRXR|PO||\nTQ1|||Q4-6H PRN|||20230615080000|||"
}
```
7. **Save the request**

### Process HL7 RAS Message

1. **Create a new request** in the collection
2. **Name it** "Process HL7 RAS Message"
3. **Set method** to POST
4. **Set URL** to `{{base_url}}/hl7/ras`
5. **Headers**:
   - Content-Type: application/json
   - Authorization: Bearer {{auth_token}}
6. **Body** (raw, JSON):
```json
{
  "message": "MSH|^~\\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20230615100000||RAS^O17|MSGID456|P|2.5.1|\nPID|||123^^^MRN||DOE^JOHN||19700101|M||\nORC|NW|ORDER123||||||20230615080000|||DOCTOR^JOHN^A|\nRXA|0|1|20230615100000||1049502^Acetaminophen 325 MG Oral Tablet^RXNORM|1|TAB||NURSE^JANE^B||\nRXR|PO||"
}
```
7. **Save the request**

## 9. Running the Tests

To run the tests in the correct order:

1. First, run the "Get Auth Token" request to get a valid token
2. Run the "Health Check" to ensure the service is up
3. Run the Medication endpoints in this order:
   - Create Medication
   - Get Medication
   - Update Medication
   - Search Medications
4. Run the MedicationRequest endpoints:
   - Create Medication Request
   - Get Medication Request
   - Get Patient Medication Requests
5. Run the MedicationAdministration endpoints:
   - Create Medication Administration
   - Get Patient Medication Administrations
6. Run the MedicationStatement endpoints:
   - Create Medication Statement
   - Get Patient Medication Statements
7. Run the HL7 message processing endpoints:
   - Process HL7 RDE Message
   - Process HL7 RAS Message
8. Finally, run the Delete Medication request if needed

## 10. Troubleshooting

### Authentication Issues

If you encounter authentication issues:

1. **Check if the auth service is running** at `http://localhost:8001`
2. **Verify your Auth0 credentials** are correct
3. **Check the token expiration** - tokens typically expire after 24 hours
4. **Try getting a new token** using one of the methods described above
5. **Check the auth service logs** for any errors

### API Request Issues

If your API requests are failing:

1. **Check if the medication service is running** at `http://localhost:8008`
2. **Verify the request URL** is correct
3. **Check the request headers** - make sure the Authorization header is set correctly
4. **Check the request body** - make sure the JSON is valid
5. **Check the medication service logs** for any errors

### FHIR Server Issues

If you're getting errors related to the FHIR server:

1. **Check if the FHIR server is running** at `http://localhost:8004`
2. **Verify the FHIR server URL** in the medication service configuration
3. **Check the FHIR server logs** for any errors
