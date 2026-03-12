# SMART Authorization Quick Reference

## 30-Second Setup

```java
// 1. Create authorization service
SMARTAuthorizationService authService = new SMARTAuthorizationService(
    "https://fhir.ehr.com/oauth/authorize",
    "https://fhir.ehr.com/oauth/token",
    "https://fhir.ehr.com/oauth/introspect"
);

// 2. Generate auth URL
String authUrl = authService.getAuthorizationUrl(
    "client-id", "redirect-uri", "patient/*.read patient/*.write", "state-123"
);

// 3. Exchange code for token
SMARTToken token = authService.exchangeCodeForToken(
    "auth-code", "client-id", "client-secret", "redirect-uri"
);

// 4. Use token for FHIR operations
FHIRExportService exportService = new FHIRExportService("https://fhir.ehr.com/fhir");
String serviceRequestId = exportService.exportRecommendationToFHIR(recommendation, token);
```

## Common Tasks

### Get Authorization URL
```java
String authUrl = authService.getAuthorizationUrl(clientId, redirectUri, scope, state);
// User opens in browser → authorizes → redirects with code
```

### Exchange Code for Token
```java
SMARTToken token = authService.exchangeCodeForToken(code, clientId, clientSecret, redirectUri);
```

### Refresh Expired Token
```java
if (token.isExpired() || token.expiresWithin(300)) {
    token = authService.refreshToken(token.getRefreshToken(), clientId, clientSecret);
}
```

### Check Token Validity
```java
if (token.isExpired()) {
    // Token expired, need to refresh
}

if (token.expiresWithin(300)) {
    // Token expires in 5 minutes, refresh proactively
}
```

### Validate Scopes
```java
List<String> required = Arrays.asList("patient/*.read", "patient/*.write");
if (!authService.validateScopes(token, required)) {
    // Token missing required scopes
}
```

### Use Cached Token
```java
SMARTToken token = authService.getCachedToken(patientId);
if (token == null) {
    // Not cached, need to authorize
}
```

### Export to FHIR
```java
// Export recommendation
String id = exportService.exportRecommendationToFHIR(recommendation, token);

// Export risk score
String id = exportService.exportRiskScoreToFHIR(riskScore, patientId, token);

// Export care gap
String id = exportService.exportCareGapToFHIR(careGap, token);
```

## SMART Scopes

| Scope | Description |
|-------|-------------|
| `patient/*.read` | Read all patient data |
| `patient/*.write` | Write all patient data |
| `launch/patient` | Patient context at launch |
| `openid fhirUser` | User identity |
| `offline_access` | Refresh tokens |

## FHIR Resource Mappings

| CardioFit | FHIR Resource | Method |
|-----------|---------------|--------|
| ClinicalRecommendation | ServiceRequest | `exportRecommendationToFHIR()` |
| RiskScore | RiskAssessment | `exportRiskScoreToFHIR()` |
| CareGap | DetectedIssue | `exportCareGapToFHIR()` |

## Configuration

```properties
smart.auth.endpoint=https://fhir.ehr.com/oauth/authorize
smart.token.endpoint=https://fhir.ehr.com/oauth/token
smart.client.id=your-client-id
smart.client.secret=${SMART_CLIENT_SECRET}
smart.redirect.uri=https://your-app.com/callback
fhir.base.url=https://fhir.ehr.com/fhir
```

## Error Handling

```java
try {
    SMARTToken token = authService.exchangeCodeForToken(...);
} catch (IOException e) {
    // Network error or invalid response
    // Common errors: invalid_grant, invalid_client, invalid_request
}

try {
    String id = exportService.exportRecommendationToFHIR(recommendation, token);
} catch (IOException e) {
    // Token expired, network error, or invalid resource
}
```

## Security Checklist

- ✅ Use HTTPS for all OAuth2 endpoints
- ✅ Validate state parameter to prevent CSRF
- ✅ Store client secret in environment variables
- ✅ Check token expiration before use
- ✅ Never log access tokens or refresh tokens
- ✅ Use minimum required scopes
- ✅ Refresh tokens proactively (5 min before expiry)
- ✅ Clear token cache on logout

## Testing

```bash
# Run all SMART tests
mvn test -Dtest=SMART*

# Run specific test
mvn test -Dtest=SMARTAuthorizationServiceTest
```

## Quick Debugging

### Token Not Working?
```java
System.out.println("Token expired: " + token.isExpired());
System.out.println("Token scopes: " + token.getScope());
System.out.println("Seconds until expiry: " + token.getSecondsUntilExpiration());
```

### FHIR Export Failing?
```java
// Check token validity
if (token.isExpired()) {
    LOG.error("Token expired!");
}

// Check write permissions
if (!token.hasScope("patient/*.write")) {
    LOG.error("Token missing write permission");
}

// Verify FHIR base URL
LOG.info("FHIR URL: " + exportService.getFhirBaseUrl());
```

## Complete Example

```java
public class SMARTWorkflow {
    public static void main(String[] args) throws Exception {
        // 1. Setup services
        SMARTAuthorizationService authService = new SMARTAuthorizationService(
            "https://fhir.ehr.com/oauth/authorize",
            "https://fhir.ehr.com/oauth/token",
            null
        );

        FHIRExportService exportService = new FHIRExportService(
            "https://fhir.ehr.com/fhir"
        );

        // 2. Generate authorization URL
        String authUrl = authService.getAuthorizationUrl(
            "cardiofit-app",
            "https://cardiofit.health/callback",
            "patient/*.read patient/*.write launch/patient offline_access",
            UUID.randomUUID().toString()
        );

        System.out.println("Open: " + authUrl);

        // 3. User authorizes → get code from redirect
        String code = "..."; // From callback URL

        // 4. Exchange code for token
        SMARTToken token = authService.exchangeCodeForToken(
            code,
            "cardiofit-app",
            System.getenv("SMART_CLIENT_SECRET"),
            "https://cardiofit.health/callback"
        );

        System.out.println("Access token obtained for patient: " + token.getPatientId());

        // 5. Use token for FHIR operations
        ClinicalRecommendation rec = ClinicalRecommendation.builder()
            .patientId(token.getPatientId())
            .protocolName("Sepsis Protocol")
            .priority("CRITICAL")
            .build();

        String serviceRequestId = exportService.exportRecommendationToFHIR(rec, token);
        System.out.println("Created ServiceRequest: " + serviceRequestId);

        // 6. Check if token needs refresh
        if (token.expiresWithin(300)) {
            token = authService.refreshToken(
                token.getRefreshToken(),
                "cardiofit-app",
                System.getenv("SMART_CLIENT_SECRET")
            );
            System.out.println("Token refreshed");
        }
    }
}
```

## More Information

- Full documentation: `README.md`
- Test examples: See test files in `src/test/java/.../smart/`
- SMART spec: http://hl7.org/fhir/smart-app-launch/
