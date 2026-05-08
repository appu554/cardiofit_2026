package au.vaidshala.cqlruntime.config;

import ca.uhn.fhir.context.FhirContext;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Spring configuration for HAPI FHIR R4 server beans.
 *
 * <p>Plan 0.5 Task 5 ships only the FhirContext bean — that's enough for
 * the {@code $evaluate-rule} operation provider to construct Parameters
 * responses. Full HAPI server registration (RestfulServer with
 * ResourceProviders) is deferred to a follow-up if/when kb-cql-runtime
 * needs to expose REST endpoints for FHIR resource CRUD; for Plan 0.5,
 * the {@link au.vaidshala.cqlruntime.operation.EvaluateRuleController}
 * Spring controller is the operation entry point.
 *
 * <p>Plan 0.5 Task 5 of 8.
 */
@Configuration
public class HapiFhirConfig {

    @Bean
    public FhirContext fhirContext() {
        return FhirContext.forR4();
    }
}
