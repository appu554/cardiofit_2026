# Google FHIR Integration Setup

The medication-service-v2 now integrates with Google Cloud Healthcare API FHIR store for persistent, FHIR-compliant data storage.

## Configuration

The service uses the same Google FHIR store as the patient-service:

- **Project ID**: `cardiofit-905a8`
- **Location**: `asia-south1`
- **Dataset ID**: `clinical-synthesis-hub`
- **FHIR Store ID**: `fhir-store`

## Credentials Setup

### Option 1: Service Account Key File

1. Copy the Google Cloud service account key file to:
   ```
   credentials/google-credentials.json
   ```

2. Ensure the credentials directory is in `.gitignore` to avoid committing secrets

### Option 2: Environment Variables

Set one of the following environment variables:

```bash
# Path to service account key file
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Or JSON content directly (not recommended for production)
export GOOGLE_APPLICATION_CREDENTIALS_JSON='{"type":"service_account",...}'
```

### Option 3: Default Credentials (GCP Environment)

When running in Google Cloud Platform (GCE, GKE, Cloud Run, etc.), the service will automatically use the default service account.

## Environment Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key Google FHIR settings in `.env`:
```bash
USE_GOOGLE_HEALTHCARE_API=true
GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=asia-south1
GOOGLE_CLOUD_DATASET_ID=clinical-synthesis-hub
GOOGLE_CLOUD_FHIR_STORE_ID=fhir-store
GOOGLE_CLOUD_CREDENTIALS_PATH=credentials/google-credentials.json
```

## Required Permissions

The service account needs the following Google Cloud IAM roles:

- `roles/healthcare.fhirResourceEditor` - Full CRUD operations on FHIR resources
- `roles/healthcare.fhirStoreViewer` - Read access to FHIR store metadata

## Testing the Integration

1. Start the service:
   ```bash
   go run cmd/server/main.go
   ```

2. Check the logs for successful FHIR client initialization:
   ```
   INFO  Successfully initialized Google FHIR client
   ```

3. Test GraphQL endpoints:
   - Playground: http://localhost:8005/graphql
   - Federation: http://localhost:8005/federation

## FHIR Operations

The service supports full CRUD operations for:

- **Medication** resources: Drug definitions, formulations, ingredients
- **MedicationRequest** resources: Prescriptions, orders, administration instructions

All operations are FHIR R4 compliant and stored in the Google Healthcare API FHIR store.

## Troubleshooting

### Authentication Issues
- Check credentials file path and permissions
- Verify service account has required roles
- Ensure project ID and location are correct

### API Errors
- Verify FHIR store exists in the specified project/location
- Check network connectivity to `healthcare.googleapis.com`
- Review service logs for detailed error messages

### Fallback Behavior
If Google FHIR initialization fails, the service will:
- Log the error
- Continue to start (without FHIR functionality)
- Allow other service operations to work normally