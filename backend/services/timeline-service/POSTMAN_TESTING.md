# Testing the Timeline Service with Postman

This document provides instructions for testing the Timeline Service API using Postman.

## Setup

1. Install [Postman](https://www.postman.com/downloads/) if you haven't already.
2. Import the Postman collection:
   - Open Postman
   - Click "Import" in the top left
   - Select the `timeline_service_postman_collection.json` file
3. Import the Postman environment:
   - Click "Import" in the top left
   - Select the `timeline_service_postman_environment.json` file
4. Select the "Timeline Service - Local" environment from the dropdown in the top right

## Available Tests

The collection includes the following requests:

1. **Get Patient Timeline**: Retrieves a complete timeline for a patient
2. **Get Patient Timeline with Query Filters**: Retrieves a filtered timeline using query parameters
3. **Filter Patient Timeline (POST)**: Filters a timeline using a JSON body
4. **Filter Timeline - Observations Only**: Shows only observation events
5. **Filter Timeline - Conditions Only**: Shows only condition events
6. **Filter Timeline - Medications Only**: Shows only medication events
7. **Filter Timeline - Encounters Only**: Shows only encounter events
8. **Filter Timeline - Documents Only**: Shows only document events
9. **Health Check**: Verifies the service is running

## Running the Tests

1. Make sure the Timeline Service is running on port 8010
2. Select a request from the collection
3. Click the "Send" button
4. View the response in the bottom panel

## Modifying the Tests

You can modify the tests by:

1. Changing the patient ID in the URL
2. Modifying the filter parameters in the request body
3. Changing the authorization token in the request headers

## Environment Variables

The environment includes the following variables:

- `baseUrl`: The base URL of the Timeline Service (default: http://localhost:8010)
- `token`: The authorization token to use (default: test_token)
- `patientId`: The patient ID to use in requests (default: 123)

You can change these values by editing the environment variables in Postman.

## Troubleshooting

If you encounter issues:

1. Make sure the Timeline Service is running
2. Check that you're using the correct environment
3. Verify that the authorization token is being sent correctly
4. Check the response status code and body for error messages
