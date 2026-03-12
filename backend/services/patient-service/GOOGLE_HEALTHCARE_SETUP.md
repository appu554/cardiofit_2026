# Google Healthcare API Manual Setup Guide

This guide provides step-by-step instructions for setting up the Google Healthcare API resources required for the Patient Service.

## Prerequisites

1. A Google Cloud Platform (GCP) account
2. Access to the Google Cloud Console
3. The Google Cloud SDK (optional, for command-line setup)

## Step 1: Enable the Healthcare API

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Select your project (`cardiofit-905a8`)
3. Navigate to "APIs & Services" > "Library"
4. Search for "Cloud Healthcare API"
5. Click on "Cloud Healthcare API" in the results
6. Click "Enable" if it's not already enabled

## Step 2: Create a Dataset

A dataset is a container for FHIR stores and other healthcare data stores.

### Option 1: Using the Google Cloud Console

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Navigate to "Healthcare" in the left sidebar
3. Click "Create Dataset"
4. Enter the following information:
   - Dataset ID: `clinical_synthesis_hub` (use underscores, not hyphens)
   - Location: `us-central1`
5. Click "Create"

### Option 2: Using the Google Cloud SDK (Command Line)

```bash
gcloud healthcare datasets create clinical_synthesis_hub --location=us-central1
```

## Step 3: Create a FHIR Store

A FHIR store is a repository for FHIR resources within a dataset.

### Option 1: Using the Google Cloud Console

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Navigate to "Healthcare" in the left sidebar
3. Click on the dataset you created (`clinical_synthesis_hub`)
4. Click "Create Data Store"
5. Select "FHIR" as the store type
6. Enter the following information:
   - FHIR store ID: `fhir_store` (use underscores, not hyphens)
   - FHIR version: `R4`
   - Enable FHIR history: Yes (recommended)
7. Click "Create"

### Option 2: Using the Google Cloud SDK (Command Line)

```bash
gcloud healthcare fhir-stores create fhir_store --dataset=clinical_synthesis_hub --location=us-central1 --version=R4
```

## Step 4: Set Permissions for the Service Account

The service account needs permissions to access and modify the FHIR store.

### Option 1: Using the Google Cloud Console

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Navigate to "IAM & Admin" > "IAM" in the left sidebar
3. Click the "+ Add" button
4. Enter your service account email: `healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com`
5. Add the following roles:
   - `Healthcare Dataset Administrator`
   - `Healthcare FHIR Store Administrator`
   - `Healthcare FHIR Resource Editor`
6. Click "Save"

### Option 2: Using the Google Cloud SDK (Command Line)

```bash
# Add Healthcare Dataset Administrator role
gcloud projects add-iam-policy-binding cardiofit-905a8 \
  --member="serviceAccount:healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com" \
  --role="roles/healthcare.datasetAdmin"

# Add Healthcare FHIR Store Administrator role
gcloud projects add-iam-policy-binding cardiofit-905a8 \
  --member="serviceAccount:healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com" \
  --role="roles/healthcare.fhirStoreAdmin"

# Add Healthcare FHIR Resource Editor role
gcloud projects add-iam-policy-binding cardiofit-905a8 \
  --member="serviceAccount:healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com" \
  --role="roles/healthcare.fhirResourceEditor"
```

## Step 5: Verify Setup

After completing the setup, you can verify that everything is working correctly:

### Option 1: Using the Google Cloud Console

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Navigate to "Healthcare" in the left sidebar
3. Click on your dataset (`clinical_synthesis_hub`)
4. Click on your FHIR store (`fhir_store`)
5. You should see the FHIR store details and be able to browse resources

### Option 2: Using the Google Cloud SDK (Command Line)

```bash
# List datasets
gcloud healthcare datasets list --location=us-central1

# List FHIR stores in the dataset
gcloud healthcare fhir-stores list --dataset=clinical_synthesis_hub --location=us-central1
```

## Step 6: Update Environment Variables

Make sure your environment variables are correctly set in the `.env` file:

```
USE_GOOGLE_HEALTHCARE_API=true
GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=us-central1
GOOGLE_CLOUD_DATASET_ID=clinical_synthesis_hub
GOOGLE_CLOUD_FHIR_STORE_ID=fhir_store
GOOGLE_CLOUD_CREDENTIALS_PATH=credentials/google-credentials.json
```

## Troubleshooting

### Common Issues

1. **Dataset or FHIR store creation fails with "malformed" error**:
   - Make sure you're using underscores (`_`) instead of hyphens (`-`) in the IDs
   - The pattern must match `^[\p{L}\p{N}_\-\.]{1,256}$`

2. **Permission denied errors**:
   - Make sure your service account has the necessary roles assigned
   - Check that the service account credentials file is correctly formatted and accessible

3. **"Dataset not found" error**:
   - Verify that the dataset exists in the specified location
   - Check that the dataset ID in your environment variables matches the actual dataset ID

4. **"FHIR store not found" error**:
   - Verify that the FHIR store exists in the specified dataset
   - Check that the FHIR store ID in your environment variables matches the actual FHIR store ID

### Getting Help

If you continue to experience issues, you can:

1. Check the [Google Cloud Healthcare API documentation](https://cloud.google.com/healthcare/docs)
2. Review the [Google Cloud Healthcare API troubleshooting guide](https://cloud.google.com/healthcare/docs/troubleshooting)
3. Contact Google Cloud Support
