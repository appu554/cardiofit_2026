package com.cardiofit.flink.clients;

import com.cardiofit.flink.models.GraphData;
import com.cardiofit.flink.models.PatientSnapshot;
import org.neo4j.driver.*;
import org.neo4j.driver.async.AsyncSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * Async client for Neo4j graph database queries.
 *
 * This client provides non-blocking access to care network and patient relationship data
 * stored in Neo4j graph database.
 *
 * Graph Data Model:
 * - Patient nodes with relationships to Care Team, Risk Cohorts, Care Pathways
 * - Provider nodes (physicians, nurses, specialists)
 * - Cohort nodes (diabetes, CHF, high-risk, etc.)
 * - Pathway nodes (clinical care pathways)
 *
 * Query Patterns:
 * - MATCH (p:Patient {patientId: $id})-[:HAS_PROVIDER]->(provider)
 * - MATCH (p:Patient)-[:IN_COHORT]->(cohort)
 * - MATCH (p:Patient)-[:FOLLOWS_PATHWAY]->(pathway)
 */
public class Neo4jGraphClient implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Neo4jGraphClient.class);

    // Neo4j connection configuration
    private final String uri;
    private final String username;
    private final String password;

    // Transient fields (recreated in initialize())
    private transient Driver driver;

    // Configuration constants
    private static final int QUERY_TIMEOUT_MS = 500; // Async query timeout
    private static final int MAX_RETRY_TIME_MS = 1000;

    /**
     * Constructor with Neo4j connection details.
     *
     * @param uri Neo4j connection URI (e.g., "bolt://localhost:7687")
     * @param username Neo4j username
     * @param password Neo4j password
     */
    public Neo4jGraphClient(String uri, String username, String password) {
        this.uri = uri;
        this.username = username;
        this.password = password;

        LOG.info("Neo4jGraphClient initialized with URI: {}", uri);
    }

    /**
     * Initialize the Neo4j driver.
     *
     * Called in Flink operator's open() method.
     */
    public void initialize() {
        LOG.info("Initializing Neo4j driver for URI: {}", uri);

        Config config = Config.builder()
            .withMaxConnectionPoolSize(10)
            .withConnectionAcquisitionTimeout(QUERY_TIMEOUT_MS, TimeUnit.MILLISECONDS)
            .withMaxTransactionRetryTime(MAX_RETRY_TIME_MS, TimeUnit.MILLISECONDS)
            .withoutEncryption() // No encryption for development Neo4j 5.x
            .build();

        this.driver = GraphDatabase.driver(uri, AuthTokens.basic(username, password), config);

        // Verify connectivity
        try {
            driver.verifyConnectivity();
            LOG.info("Neo4j driver initialized and verified successfully");
        } catch (Exception e) {
            LOG.error("Failed to verify Neo4j connectivity", e);
            // Don't throw - allow Flink to start even if Neo4j is unavailable
            // Queries will fail gracefully and return empty data
        }
    }

    /**
     * Query patient graph data asynchronously.
     *
     * This retrieves:
     * - Care team members (providers associated with this patient)
     * - Risk cohorts (clinical risk groups patient belongs to)
     * - Care pathways (clinical pathways patient is following)
     * - Related patients (family members, research cohorts)
     *
     * @param patientId The patient identifier
     * @return CompletableFuture with GraphData or empty data on error
     */
    public CompletableFuture<GraphData> queryGraphAsync(String patientId) {
        if (driver == null) {
            LOG.warn("Neo4j driver not initialized - returning empty graph data");
            return CompletableFuture.completedFuture(new GraphData());
        }

        LOG.debug("Querying graph for patient: {}", patientId);

        CompletableFuture<GraphData> future = new CompletableFuture<>();

        // Execute async session
        AsyncSession session = driver.asyncSession();

        // Cypher query to get all patient relationships
        String query =
            "MATCH (p:Patient {patientId: $patientId}) " +
            "OPTIONAL MATCH (p)-[:HAS_PROVIDER]->(provider:Provider) " +
            "OPTIONAL MATCH (p)-[:IN_COHORT]->(cohort:Cohort) " +
            "OPTIONAL MATCH (p)-[:FOLLOWS_PATHWAY]->(pathway:Pathway) " +
            "OPTIONAL MATCH (p)-[:RELATED_TO]->(related:Patient) " +
            "RETURN " +
            "  collect(DISTINCT provider.providerId) AS careTeam, " +
            "  collect(DISTINCT cohort.name) AS riskCohorts, " +
            "  collect(DISTINCT pathway.name) AS carePathways, " +
            "  collect(DISTINCT related.patientId) AS relatedPatients";

        session.runAsync(query, Values.parameters("patientId", patientId))
            .thenCompose(cursor -> cursor.singleAsync())
            .thenAccept(record -> {
                if (record == null) {
                    LOG.warn("Neo4j query for patient {} returned null record", patientId);
                    future.complete(new GraphData());
                    session.closeAsync();
                    return;
                }

                GraphData graphData = new GraphData();

                // Parse care team
                List<String> careTeam = record.get("careTeam").asList(value -> value.asString());
                graphData.setCareTeam(careTeam);

                // Parse risk cohorts
                List<String> cohorts = record.get("riskCohorts").asList(value -> value.asString());
                graphData.setRiskCohorts(cohorts);

                // Parse care pathways
                List<String> pathways = record.get("carePathways").asList(value -> value.asString());
                graphData.setCarePathways(pathways);

                // Parse related patients
                List<String> related = record.get("relatedPatients").asList(value -> value.asString());
                graphData.setRelatedPatients(related);

                LOG.info("Neo4j lookup for patient {} successful: {} care team, {} cohorts",
                    patientId, careTeam.size(), cohorts.size());

                future.complete(graphData);
                session.closeAsync();
            })
            .exceptionally(throwable -> {
                LOG.error("Neo4j query failed for patient {}: {}", patientId, throwable.getMessage());
                future.complete(new GraphData());
                session.closeAsync();
                return null;
            });

        return future;
    }

    /**
     * Update care network graph when encounter closes.
     *
     * This creates/updates:
     * - Patient-Provider relationships based on care team
     * - Patient-Cohort relationships based on conditions/risk scores
     * - Updated timestamps for encounter activity
     *
     * @param snapshot The patient snapshot to persist to graph
     */
    public void updateCareNetwork(PatientSnapshot snapshot) {
        if (driver == null) {
            LOG.warn("Neo4j driver not initialized - skipping care network update");
            return;
        }

        LOG.info("Updating care network for patient: {}", snapshot.getPatientId());

        try (Session session = driver.session()) {
            // Merge patient node (create if not exists)
            String mergePatient =
                "MERGE (p:Patient {patientId: $patientId}) " +
                "SET p.lastName = $lastName, " +
                "    p.firstName = $firstName, " +
                "    p.lastUpdated = $timestamp";

            session.run(mergePatient, Values.parameters(
                "patientId", snapshot.getPatientId(),
                "lastName", snapshot.getLastName(),
                "firstName", snapshot.getFirstName(),
                "timestamp", System.currentTimeMillis()
            ));

            // Update care team relationships
            if (snapshot.getCareTeam() != null) {
                for (String providerId : snapshot.getCareTeam()) {
                    String mergeCareTeam =
                        "MATCH (p:Patient {patientId: $patientId}) " +
                        "MERGE (prov:Provider {providerId: $providerId}) " +
                        "MERGE (p)-[r:HAS_PROVIDER]->(prov) " +
                        "SET r.lastEncounter = $timestamp";

                    session.run(mergeCareTeam, Values.parameters(
                        "patientId", snapshot.getPatientId(),
                        "providerId", providerId,
                        "timestamp", System.currentTimeMillis()
                    ));
                }
            }

            // Update risk cohort relationships based on conditions
            if (snapshot.getActiveConditions() != null) {
                for (var condition : snapshot.getActiveConditions()) {
                    String conditionCode = condition.getCode();
                    // Map condition codes to cohort names (simplified)
                    String cohortName = mapConditionToCohort(conditionCode);

                    if (cohortName != null) {
                        String mergeCohort =
                            "MATCH (p:Patient {patientId: $patientId}) " +
                            "MERGE (c:Cohort {name: $cohortName}) " +
                            "MERGE (p)-[r:IN_COHORT]->(c) " +
                            "SET r.addedDate = coalesce(r.addedDate, $timestamp)";

                        session.run(mergeCohort, Values.parameters(
                            "patientId", snapshot.getPatientId(),
                            "cohortName", cohortName,
                            "timestamp", System.currentTimeMillis()
                        ));
                    }
                }
            }

            LOG.info("Successfully updated care network for patient: {}", snapshot.getPatientId());

        } catch (Exception e) {
            LOG.error("Error updating care network for patient {}", snapshot.getPatientId(), e);
            // Don't throw - allow process to continue even if graph update fails
        }
    }

    /**
     * Map condition code to risk cohort name.
     *
     * This is a simplified mapping. Production would use comprehensive
     * clinical knowledge base for cohort assignment.
     */
    private String mapConditionToCohort(String conditionCode) {
        if (conditionCode == null) return null;

        // Simplified ICD-10 to cohort mapping
        if (conditionCode.startsWith("I50")) return "CHF"; // Congestive Heart Failure
        if (conditionCode.startsWith("E10") || conditionCode.startsWith("E11")) return "Diabetes";
        if (conditionCode.startsWith("I21")) return "MI"; // Myocardial Infarction
        if (conditionCode.startsWith("J44")) return "COPD";
        if (conditionCode.startsWith("I10")) return "Hypertension";

        return null;
    }

    /**
     * Close the Neo4j driver and release resources.
     */
    public void close() {
        if (driver != null) {
            driver.close();
            LOG.info("Neo4j driver closed successfully");
        }
    }

    // Getters for configuration
    public String getUri() { return uri; }
    public String getUsername() { return username; }

    /**
     * Get the Neo4j driver for advanced queries
     * Used by AdvancedNeo4jQueries for Phase 2 features
     */
    public Driver getDriver() {
        return driver;
    }
}
