# Google Cloud Healthcare API Integration

This document provides instructions for setting up and using the Google Cloud Healthcare API integration with the Patient Service.

## Overview

The Patient Service now supports using Google Cloud Healthcare API as a backend for storing and retrieving FHIR resources. This integration replaces the MongoDB database with Google's fully managed, FHIR-compliant storage solution.

## Prerequisites

1. A Google Cloud Platform (GCP) account
2. A GCP project with the Healthcare API enabled
3. A service account with appropriate permissions
4. A FHIR dataset and store created in the Healthcare API

## Setup Instructions

### 1. Create a Google Cloud Project

If you don't already have a Google Cloud project:

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Click on "New Project"
3. Enter a project name and select a billing account
4. Click "Create"

### 2. Enable the Healthcare API

1. Go to the [API Library](https://console.cloud.google.com/apis/library)
2. Search for "Healthcare API"
3. Click on "Cloud Healthcare API"
4. Click "Enable"

### 3. Create a Service Account

1. Go to [IAM & Admin > Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts)
2. Click "Create Service Account"
3. Enter a name and description
4. Click "Create and Continue"
5. Add the following roles:
   - Healthcare FHIR Resource Editor
   - Healthcare FHIR Store Editor
6. Click "Done"
7. Click on the service account you just created
8. Go to the "Keys" tab
9. Click "Add Key" > "Create new key"
10. Select JSON and click "Create"
11. Save the JSON file securely

### 4. Create a Healthcare Dataset and FHIR Store

1. Go to the [Healthcare API](https://console.cloud.google.com/healthcare)
2. Click "Create Dataset"
3. Enter a name (e.g., "clinical-synthesis-hub")
4. Select a location (e.g., "us-central1")
5. Click "Create"
6. Click on the dataset you just created
7. Click "Create FHIR Store"
8. Enter a name (e.g., "fhir-store")
9. Select "R4" as the FHIR version
10. Enable "Enable FHIR History" if you want to track resource history
11. Click "Create"

### 5. Configure the Patient Service

1. Copy the `.env.example` file to `.env`
2. Update the following settings:
   ```
   USE_GOOGLE_HEALTHCARE_API=true
   GOOGLE_CLOUD_PROJECT_ID=your-project-id
   GOOGLE_CLOUD_LOCATION=us-central1
   GOOGLE_CLOUD_DATASET_ID=clinical-synthesis-hub
   GOOGLE_CLOUD_FHIR_STORE_ID=fhir-store
   GOOGLE_CLOUD_CREDENTIALS_PATH=/path/to/your/credentials.json
   ```
3. Replace the values with your actual Google Cloud settings
4. Place your service account credentials JSON file at the path specified in `GOOGLE_CLOUD_CREDENTIALS_PATH`

## Usage

Once configured, the Patient Service will automatically use Google Cloud Healthcare API for all FHIR operations. The API endpoints remain the same, so no changes are needed in how you interact with the service.

### API Endpoints

- `POST /api/fhir/Patient` - Create a new patient
- `GET /api/fhir/Patient/{id}` - Get a patient by ID
- `PUT /api/fhir/Patient/{id}` - Update a patient
- `DELETE /api/fhir/Patient/{id}` - Delete a patient
- `GET /api/fhir/Patient` - Search for patients

### GraphQL Endpoints

The GraphQL API also works with the Google Cloud Healthcare API backend:

- `/api/graphql` - GraphQL endpoint
- `/api/graphql/playground` - GraphQL playground

## Troubleshooting

### Authentication Issues

If you see authentication errors:

1. Check that your service account credentials file is correctly specified in `GOOGLE_CLOUD_CREDENTIALS_PATH`
2. Verify that the service account has the necessary permissions
3. Ensure that the Healthcare API is enabled in your Google Cloud project

### Resource Not Found

If resources are not found:

1. Check that the dataset and FHIR store names are correct in your configuration
2. Verify that the resources exist in the specified FHIR store
3. Check the logs for any errors during resource creation or retrieval

## Monitoring and Logging

Google Cloud provides comprehensive monitoring and logging for the Healthcare API:

1. Go to the [Logging](https://console.cloud.google.com/logs) page to view logs
2. Go to the [Monitoring](https://console.cloud.google.com/monitoring) page to set up alerts and dashboards

## Additional Resources

- [Google Cloud Healthcare API Documentation](https://cloud.google.com/healthcare/docs)
- [FHIR Resources Reference](https://cloud.google.com/healthcare/docs/reference/rest/v1/projects.locations.datasets.fhirStores.fhir)
- [Google Cloud Healthcare API Python Client](https://googleapis.dev/python/healthcare/latest/index.html)
