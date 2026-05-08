package au.vaidshala.cqlruntime;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * kb-cql-runtime entry point.
 *
 * <p>Plan 0.5 deliverable: stand up a HAPI FHIR R4 server with the
 * Clinical Reasoning module so the team's CQL rule definitions in
 * shared/cql-libraries/ can execute against the v2 substrate via
 * {@code Vaidshala.Substrate.*} external functions.
 *
 * <p>Plan 0.5 Task 1 of 8 ships only this scaffold: a Spring Boot app
 * that starts and serves /actuator/health. Subsequent tasks add the
 * HAPI server bean (Task 3 onward), the {@code SubstrateExternalFunctions}
 * binding (Task 3), CQL library loading (Task 4), and the
 * {@code $evaluate-rule} operation (Task 5).
 */
@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
