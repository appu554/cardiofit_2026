# Encounter Microservice GraphQL API Testing Guide

This guide will help you test the GraphQL API for the Encounter Microservice using Postman.

## 1. Setting Up Postman Collection

1. **Open Postman** and click on "Collections" in the sidebar
2. **Import the collection** by clicking "Import" and selecting the `graphql_postman_collection.json` file
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

## 3. Testing the GraphQL API

### Basic Queries

1. **Send the "Get All Encounters" request**
2. **Verify** that the response includes a list of encounters
3. **Send the "Get Encounters by Status" request**
4. **Verify** that the response includes only encounters with the specified status

### Creating an Encounter

1. **Send one of the "Create" requests** (e.g., "Create Hospital Inpatient Stay")
2. **Verify** that the response includes the created encounter details
3. **Copy the encounter ID** from the response (look for the `id` field)
4. **Set the encounter ID** in the collection variables:
   - Click on the collection name → "Variables" tab
   - Set `encounter_id` to the copied ID
   - Click "Save"

### Getting a Specific Encounter

1. **Send the "Get Encounter by ID" request**
2. **Verify** that the response includes the encounter details
3. **Check** that the encounter details match what you created

### Updating an Encounter

1. **Send the "Update Encounter" request**
2. **Verify** that the response includes the updated encounter details
3. **Check** that the encounter status is now `finished` and the period has an end date

### Getting Patient Encounters

1. **Send the "Get Patient Encounters" request**
2. **Verify** that the response includes a list of encounters for the patient
3. **Check** that the list includes the encounter you created

### Deleting an Encounter

1. **Send the "Delete Encounter" request**
2. **Verify** that the response is `true`
3. **Send the "Get Encounter by ID" request** to verify the encounter was deleted
4. **Verify** that the response is `null`

## 4. Testing Different Encounter Types

The collection includes example requests for different types of encounters:

1. **Hospital Inpatient Stay**: Use "Create Hospital Inpatient Stay" request
2. **Emergency Department Visit**: Use "Create Emergency Department Visit" request
3. **Outpatient Clinic Appointment**: Use "Create Outpatient Clinic Appointment" request
4. **Telehealth Session**: Use "Create Telehealth Session" request
5. **Home Health Visit**: Use "Create Home Health Visit" request

For each type:
1. **Send the corresponding create request**
2. **Verify** that the response includes the created encounter details
3. **Copy the encounter ID** and update the collection variable
4. **Send the "Get Encounter by ID" request** to verify the encounter was created correctly

## 5. GraphQL Schema Exploration

You can explore the GraphQL schema using GraphQL Playground:

1. **Open your browser** and navigate to `http://localhost:8020/api/graphql`
2. **Click on "Docs"** in the right sidebar to explore the schema
3. **Try different queries and mutations** directly in the playground

## 6. Troubleshooting

If you encounter any issues:

1. **Check the authorization token** is set correctly
2. **Verify the server is running** by sending a request to `http://localhost:8020/health`
3. **Check the GraphQL endpoint** is accessible at `http://localhost:8020/api/graphql`
4. **Look for error messages** in the response body
5. **Check the server logs** for more detailed error information
