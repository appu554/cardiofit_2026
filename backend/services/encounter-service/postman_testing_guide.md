# Encounter Microservice REST API Testing Guide with Postman

This guide will help you test the Encounter Microservice REST API using Postman, including authentication.

## 1. Setting Up Postman Collection

1. **Open Postman** and click on "Collections" in the sidebar
2. **Import the collection** by clicking "Import" and selecting the `postman_collection.json` file
3. **Set collection variables** by clicking on the collection name → "Variables" tab:
   - `base_url`: `http://localhost:8020`
   - `auth_token`: Leave empty for now (we'll get this from Auth0)
   - `encounter_id`: Leave empty for now (we'll get this from the API)

## 2. Getting an Auth Token

For testing purposes, you can use a development token:

1. **Use a dummy token** for development: `dummy_token`
2. **Set the token** in the collection variables:
   - Click on the collection name → "Variables" tab
   - Set `auth_token` to `dummy_token`
   - Click "Save"

For production testing with Auth0:

1. **Get a token from Auth0**:
   - Go to your Auth0 dashboard
   - Navigate to "Applications" → Your API Application → "Test"
   - Copy the access token
2. **Set the token** in the collection variables

## 3. Testing the API

### Health Check

1. **Send the "Health Check" request**
2. **Verify** that the response is `200 OK` with a message indicating the service is healthy

### Creating an Encounter

1. **Send the "Create Encounter" request**
2. **Verify** that the response is `201 Created` with the encounter details
3. **Copy the encounter ID** from the response (look for the `id` field)
4. **Set the encounter ID** in the collection variables:
   - Click on the collection name → "Variables" tab
   - Set `encounter_id` to the copied ID
   - Click "Save"

### Getting an Encounter

1. **Send the "Get Encounter" request**
2. **Verify** that the response is `200 OK` with the encounter details
3. **Check** that the encounter details match what you created

### Updating an Encounter

1. **Send the "Update Encounter" request**
2. **Verify** that the response is `200 OK` with the updated encounter details
3. **Check** that the encounter status is now `finished` and the period has an end date

### Searching for Encounters

1. **Send the "Search Encounters" request**
2. **Verify** that the response is `200 OK` with a list of encounters
3. **Check** that the list includes the encounter you created

### Getting Patient Encounters

1. **Send the "Get Patient Encounters" request**
2. **Verify** that the response is `200 OK` with a list of encounters for the patient
3. **Check** that the list includes the encounter you created

### Processing HL7 Messages

1. **Send the "Process HL7 Message" request**
2. **Verify** that the response is `200 OK` with a success message
3. **Check** that the response includes details about the processed message and created encounter

### Processing ADT Messages

1. **Send the "Process ADT Message" request**
2. **Verify** that the response is `200 OK` with a success message
3. **Check** that the response includes details about the processed ADT message and created encounter

## 4. Testing Error Handling

### Invalid Encounter ID

1. **Change the encounter ID** in the collection variables to an invalid ID (e.g., `invalid_id`)
2. **Send the "Get Encounter" request**
3. **Verify** that the response is `404 Not Found` with an error message

### Invalid HL7 Message

1. **Modify the HL7 message** in the "Process HL7 Message" request to be invalid (e.g., remove the MSH segment)
2. **Send the request**
3. **Verify** that the response is `400 Bad Request` with an error message

## 5. Cleanup

After testing, you may want to delete the encounters you created:

1. **Set the encounter ID** in the collection variables to the ID of the encounter you want to delete
2. **Send the "Delete Encounter" request**
3. **Verify** that the response is `204 No Content`
4. **Send the "Get Encounter" request** to verify the encounter was deleted
5. **Verify** that the response is `404 Not Found`
