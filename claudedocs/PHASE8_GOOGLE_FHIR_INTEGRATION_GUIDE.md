# Phase 8: Google Healthcare FHIR API Integration Guide

**Date**: October 27, 2025
**Purpose**: Correct SMART on FHIR implementation to use existing Google Healthcare API infrastructure
**Issue**: Generic FHIR implementation created, but system already uses Google Cloud Healthcare API

---

## Executive Summary

The SMART on FHIR implementation was created with generic "fhir.ehr.com" endpoints, but the CardioFit system **already has Google Healthcare FHIR API integration** established in Module 2 through `GoogleFHIRClient`. This document provides the corrected integration approach.

`★ Insight ─────────────────────────────────────────────────────────`
**Integration Context**: The system uses **Google Cloud Healthcare API** with service account OAuth2 authentication, not traditional SMART on FHIR OAuth2 flows. The existing GoogleFHIRClient already handles:
- Service account credentials (`google-credentials.json`)
- Automatic OAuth2 token refresh
- FHIR R4 API access to Google Healthcare FHIR store
- Circuit breaker and caching for resilience

**Correct Approach**: Adapt SMART on FHIR components to leverage GoogleFHIRClient for actual FHIR operations, while keeping OAuth2 models for future external EHR integration.
`─────────────────────────────────────────────────────────────────`

---

## Existing Google Healthcare API Setup

### GoogleFHIRClient Configuration

**Location**: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/clients/GoogleFHIRClient.java`

**Current Configuration**:
```java
GoogleFHIRClient client = new GoogleFHIRClient(
    "cardiofit-905a8",              // Project ID
    "asia-south1",                  // Location
    "clinical-synthesis-hub",       // Dataset ID
    "fhir-store",                   // FHIR Store ID
    "/path/to/google-credentials.json"  // Service account key
);
```

**FHIR Base URL**:
```
https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store/fhir
```

**Authentication**:
- Uses **Google Cloud service account** credentials
- OAuth2 token automatically obtained and refreshed
- Scope: `https://www.googleapis.com/auth/cloud-healthcare`
- No manual token management required

**Features**:
- Async operations with CompletableFuture
- Circuit breaker for resilience (50% failure threshold)
- Dual cache strategy (5-min fresh + 24-hour stale)
- 10-second request timeout
- Automatic retry on transient errors

---

## Required Changes to SMART on FHIR Implementation

### 1. Update FHIRExportService to Use GoogleFHIRClient

**Current Implementation** (Incorrect):
```java
public class FHIRExportService {
    private final String fhirBaseUrl;  // Generic endpoint
    private final RestTemplate restTemplate;

    public FHIRExportService(String fhirBaseUrl) {
        this.fhirBaseUrl = fhirBaseUrl;
        this.restTemplate = new RestTemplate();
    }
}
```

**Corrected Implementation**:
```java
public class FHIRExportService {
    private final GoogleFHIRClient googleFhirClient;

    /**
     * Constructor with Google Healthcare API client.
     * @param googleFhirClient Configured Google FHIR client
     */
    public FHIRExportService(GoogleFHIRClient googleFhirClient) {
        this.googleFhirClient = googleFhirClient;
    }

    /**
     * Export clinical recommendation to Google Healthcare FHIR API.
     * Uses service account authentication automatically.
     */
    public CompletableFuture<String> exportRecommendationToFHIR(
            ClinicalRecommendation recommendation) {

        // Build FHIR ServiceRequest JSON
        Map<String, Object> serviceRequest = buildServiceRequest(recommendation);

        // Use GoogleFHIRClient to create resource
        return googleFhirClient.createResourceAsync("ServiceRequest", serviceRequest)
            .thenApply(response -> {
                String resourceId = extractResourceId(response);
                LOG.info("Created ServiceRequest/{} for recommendation", resourceId);
                return resourceId;
            });
    }
}
```

### 2. Update SMART Authorization for Google Cloud

**Current Implementation** (Generic OAuth2):
```java
public class SMARTAuthorizationService {
    private final String authorizationEndpoint = "https://fhir.ehr.com/oauth/authorize";
    private final String tokenEndpoint = "https://fhir.ehr.com/oauth/token";
}
```

**Corrected for Google Cloud**:
```java
public class SMARTAuthorizationService {
    // Google Cloud OAuth2 endpoints
    private final String authorizationEndpoint =
        "https://accounts.google.com/o/oauth2/v2/auth";
    private final String tokenEndpoint =
        "https://oauth2.googleapis.com/token";

    // Google Healthcare API scope
    private static final String HEALTHCARE_SCOPE =
        "https://www.googleapis.com/auth/cloud-healthcare";

    /**
     * Get authorization URL for Google Cloud Healthcare API.
     *
     * Note: Service account authentication is recommended for server-to-server.
     * This OAuth2 flow is for user-facing applications requiring user consent.
     */
    public String getAuthorizationUrl(String clientId, String redirectUri) {
        return String.format(
            "%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&access_type=offline",
            authorizationEndpoint,
            clientId,
            URLEncoder.encode(redirectUri, StandardCharsets.UTF_8),
            URLEncoder.encode(HEALTHCARE_SCOPE, StandardCharsets.UTF_8)
        );
    }
}
```

### 3. Integration Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    CardioFit CDS System                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ├─── Module 2 (Existing) ───────┐
                              │                                │
                              ▼                                ▼
                   ┌──────────────────────┐      ┌─────────────────────┐
                   │  GoogleFHIRClient    │      │ Service Account     │
                   │  - OAuth2 auto       │◄─────│ Credentials         │
                   │  - Token refresh     │      │ google-creds.json   │
                   │  - Circuit breaker   │      └─────────────────────┘
                   └──────────────────────┘
                              │
                              ▼
              ┌──────────────────────────────────────┐
              │  Google Cloud Healthcare API         │
              │  - Project: cardiofit-905a8          │
              │  - Location: asia-south1             │
              │  - Dataset: clinical-synthesis-hub   │
              │  - FHIR Store: fhir-store            │
              └──────────────────────────────────────┘
                              │
                              ▼
                   ┌──────────────────────┐
                   │   FHIR R4 Resources  │
                   │   - Patient          │
                   │   - Observation      │
                   │   - Condition        │
                   │   - MedicationOrder  │
                   │   - ServiceRequest   │
                   │   - RiskAssessment   │
                   │   - DetectedIssue    │
                   └──────────────────────┘
```

---

## Corrected Usage Examples

### Example 1: Export Recommendation Using Google FHIR Client

**Incorrect** (generic FHIR):
```java
// Don't do this - uses generic endpoints
FHIRExportService exportService = new FHIRExportService("https://fhir.ehr.com/fhir");
SMARTToken token = authService.exchangeCodeForToken(...);
String serviceRequestId = exportService.exportRecommendationToFHIR(recommendation, token);
```

**Correct** (Google Healthcare API):
```java
// Use existing GoogleFHIRClient
GoogleFHIRClient googleClient = new GoogleFHIRClient(
    "cardiofit-905a8",
    "asia-south1",
    "clinical-synthesis-hub",
    "fhir-store",
    "credentials/google-credentials.json"
);
googleClient.initialize();

// Create export service with Google client
FHIRExportService exportService = new FHIRExportService(googleClient);

// Export without needing manual token management
CompletableFuture<String> exportFuture =
    exportService.exportRecommendationToFHIR(recommendation);

String serviceRequestId = exportFuture.join();
LOG.info("Created ServiceRequest: {}", serviceRequestId);
```

### Example 2: CDS Hooks with Google FHIR Integration

**Incorrect**:
```java
CdsHooksService cdsHooksService = new CdsHooksService(
    mockFhirClient,  // Generic mock
    observationMapper,
    qualityEvaluator
);
```

**Correct**:
```java
// Use real GoogleFHIRClient
GoogleFHIRClient googleClient = createGoogleFHIRClient();

CdsHooksService cdsHooksService = new CdsHooksService(
    googleClient,  // Real Google Healthcare API client
    new FHIRObservationMapper(googleClient),
    new FHIRQualityMeasureEvaluator(googleClient)
);
```

### Example 3: FHIR Population Health Queries

**Already Correct** (uses GoogleFHIRClient):
```java
FHIRPopulationHealthMapper mapper = new FHIRPopulationHealthMapper(
    googleFhirClient,
    new FHIRCohortBuilder(googleFhirClient, mapper),
    new FHIRObservationMapper(googleFhirClient),
    new FHIRQualityMeasureEvaluator(googleFhirClient)
);

// All FHIR queries use Google Healthcare API automatically
CompletableFuture<PatientCohort> cohort = cohortBuilder.buildDiabeticCohort();
```

---

## When to Use Each Authentication Method

### Google Service Account (Current - Recommended)

**Use For**:
- ✅ Server-to-server communication (Flink jobs)
- ✅ Batch processing and analytics
- ✅ Background CDS operations
- ✅ Population health queries
- ✅ Internal system integration

**How It Works**:
- Service account key file (`google-credentials.json`)
- Automatic OAuth2 token obtained by GoogleFHIRClient
- No user interaction required
- Token automatically refreshed

**Example**:
```java
// Automatic authentication
GoogleFHIRClient client = new GoogleFHIRClient(...);
client.initialize();  // Loads credentials, gets token
client.getPatientAsync("Patient/123");  // Uses OAuth2 token automatically
```

### SMART on FHIR OAuth2 (Future - External EHR Integration)

**Use For**:
- 🔮 Future: External EHR system integration
- 🔮 Future: Clinician-facing web apps (SMART apps)
- 🔮 Future: Third-party application access
- 🔮 Future: User-specific data access with consent

**How It Works**:
- User redirects to authorization page
- User grants consent for specific scopes
- Application receives authorization code
- Exchange code for access token
- Use token for API calls on behalf of user

**Example** (when integrating with external EHR):
```java
// Step 1: Redirect user to authorize
String authUrl = smartAuth.getAuthorizationUrl(clientId, redirectUri, scope, state);
response.sendRedirect(authUrl);

// Step 2: Handle callback
SMARTToken token = smartAuth.exchangeCodeForToken(code, clientId, clientSecret, redirectUri);

// Step 3: Use token for external EHR API (not Google Healthcare)
externalEhrClient.getPatient("Patient/123", token);
```

---

## Configuration Updates Required

### 1. FHIRExportService Constructor

**File**: `src/main/java/com/cardiofit/flink/cds/smart/FHIRExportService.java`

**Change**:
```java
// OLD:
public FHIRExportService(String fhirBaseUrl) {
    this.fhirBaseUrl = fhirBaseUrl;
    this.restTemplate = new RestTemplate();
}

// NEW:
public FHIRExportService(GoogleFHIRClient googleFhirClient) {
    this.googleFhirClient = googleFhirClient;
}
```

### 2. Export Methods

**Update all export methods**:
```java
// OLD:
public String exportRecommendationToFHIR(
        ClinicalRecommendation recommendation, SMARTToken token) {

    HttpHeaders headers = new HttpHeaders();
    headers.setBearerAuth(token.getAccessToken());

    HttpEntity<Map<String, Object>> request =
        new HttpEntity<>(serviceRequest, headers);

    ResponseEntity<Map> response = restTemplate.postForEntity(
        fhirBaseUrl + "/ServiceRequest",
        request,
        Map.class
    );

    return extractId(response.getBody());
}

// NEW:
public CompletableFuture<String> exportRecommendationToFHIR(
        ClinicalRecommendation recommendation) {

    Map<String, Object> serviceRequest = buildServiceRequest(recommendation);

    // GoogleFHIRClient handles authentication automatically
    return googleFhirClient.createResourceAsync("ServiceRequest", serviceRequest)
        .thenApply(this::extractResourceId);
}
```

### 3. Remove Manual Token Management

**Delete or mark as "Future Enhancement"**:
- Manual Bearer token headers (GoogleFHIRClient handles this)
- Token expiration checks for Google API (automatic in GoogleFHIRClient)
- Token refresh logic for Google API (automatic in GoogleFHIRClient)

**Keep for future external EHR integration**:
- SMARTToken model (useful for other EHR systems)
- SMARTAuthorizationService (for external systems)
- OAuth2 flow methods (for user-facing apps)

---

## Testing Updates

### Update Test Mocks

**Current Tests** (using generic mocks):
```java
@Mock
private RestTemplate mockRestTemplate;

@Test
void testExportRecommendation() {
    FHIRExportService service = new FHIRExportService("https://fhir.ehr.com/fhir");
    // ...
}
```

**Updated Tests** (using GoogleFHIRClient):
```java
@Mock
private GoogleFHIRClient mockGoogleClient;

@Test
void testExportRecommendation() {
    FHIRExportService service = new FHIRExportService(mockGoogleClient);

    when(mockGoogleClient.createResourceAsync(eq("ServiceRequest"), any()))
        .thenReturn(CompletableFuture.completedFuture(mockResponse));

    CompletableFuture<String> result = service.exportRecommendationToFHIR(recommendation);

    assertEquals("ServiceRequest/123", result.join());
    verify(mockGoogleClient).createResourceAsync(eq("ServiceRequest"), any());
}
```

---

## Documentation Updates

### Update README.md

**Section to Add**:

```markdown
## Google Healthcare API Integration

This system uses **Google Cloud Healthcare API** for FHIR data storage and access.

### Configuration

1. **Create service account** in Google Cloud Console
2. **Grant Healthcare FHIR User role** to service account
3. **Download credentials JSON** → save as `google-credentials.json`
4. **Update configuration**:

```java
GoogleFHIRClient client = new GoogleFHIRClient(
    "your-project-id",
    "your-location",
    "your-dataset-id",
    "your-fhir-store-id",
    "path/to/google-credentials.json"
);
```

### Authentication

Authentication is **automatic**:
- Service account credentials loaded on initialization
- OAuth2 token obtained and cached
- Token automatically refreshed before expiration
- No manual token management required

### FHIR Operations

All FHIR operations use Google Healthcare API:
- Read: `googleClient.getPatientAsync("Patient/123")`
- Create: `googleClient.createResourceAsync("ServiceRequest", data)`
- Search: `googleClient.searchConditionsAsync("Patient/123")`
- Update: `googleClient.updateResourceAsync("Patient/123", data)`
```

---

## Migration Checklist

- [ ] Update `FHIRExportService` constructor to accept `GoogleFHIRClient`
- [ ] Remove `fhirBaseUrl` field from `FHIRExportService`
- [ ] Remove `RestTemplate` from `FHIRExportService`
- [ ] Update all export methods to use `googleFhirClient.createResourceAsync()`
- [ ] Remove manual OAuth2 token headers
- [ ] Update test mocks to use `GoogleFHIRClient`
- [ ] Update `SMARTAuthorizationService` endpoints to Google Cloud URLs
- [ ] Add documentation note: "SMART OAuth2 for future external EHR integration"
- [ ] Update README.md with Google Healthcare API configuration
- [ ] Update QUICK_REFERENCE.md with correct endpoints
- [ ] Test all FHIR export operations with real Google Healthcare API

---

## Summary

**Current State**: SMART on FHIR implemented with generic endpoints

**Correct State**: Integrate with existing Google Healthcare FHIR API

**Key Changes**:
1. Use `GoogleFHIRClient` for all FHIR operations
2. Remove manual OAuth2 token management (automatic in GoogleFHIRClient)
3. Update endpoints to Google Cloud Healthcare API
4. Keep SMART OAuth2 models for future external EHR integration

**Benefits**:
- ✅ Leverages existing, tested Google FHIR integration
- ✅ Automatic authentication and token refresh
- ✅ Circuit breaker and caching for resilience
- ✅ Consistent with Module 2 architecture
- ✅ No duplicate FHIR client implementations

**Future Enhancement**: SMART OAuth2 components ready for external EHR integration when needed.
