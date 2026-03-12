package com.cardiofit.flink.neo4j;

import com.cardiofit.flink.models.PatientSnapshot;
import org.neo4j.driver.*;
import org.neo4j.driver.Record;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

/**
 * Advanced Neo4j Queries for Similar Patient Analysis and Cohort Analytics
 *
 * Implements complex Cypher queries for:
 * 1. Similar patient matching based on demographics, conditions, and cohort membership
 * 2. Cohort analytics with readmission rates and vital statistics
 * 3. Outcome prediction based on historical similar patient data
 *
 * Query Performance Target: <200ms (spec requirement)
 */
public class AdvancedNeo4jQueries {

    private static final Logger logger = LoggerFactory.getLogger(AdvancedNeo4jQueries.class);

    private final Driver driver;
    private final int timeoutSeconds;
    private static final int DEFAULT_TIMEOUT = 5; // 5 seconds

    public AdvancedNeo4jQueries(Driver driver) {
        this(driver, DEFAULT_TIMEOUT);
    }

    public AdvancedNeo4jQueries(Driver driver, int timeoutSeconds) {
        this.driver = driver;
        this.timeoutSeconds = timeoutSeconds;
    }

    /**
     * Find similar patients and their outcomes
     *
     * Uses Jaccard similarity on:
     * - Age (±5 years)
     * - Shared risk cohorts
     * - Shared medical conditions
     *
     * Spec: Lines 268-291 of MODULE2_ADVANCED_ENHANCEMENTS.md
     *
     * @param patientId Target patient ID
     * @param snapshot Patient demographics and conditions
     * @param limit Max number of similar patients to return
     * @return List of similar patients with outcomes
     */
    public List<SimilarPatient> findSimilarPatients(String patientId, PatientSnapshot snapshot, int limit) {
        List<SimilarPatient> similarPatients = new ArrayList<>();

        if (snapshot == null || snapshot.getActiveConditions() == null) {
            logger.warn("Cannot find similar patients - insufficient patient data for {}", patientId);
            return similarPatients;
        }

        // Cypher query from spec (lines 268-291)
        String query =
                "MATCH (p:Patient {patientId: $pid})-[:MEMBER_OF_COHORT]->(c:RiskCohort)" +
                "      <-[:MEMBER_OF_COHORT]-(similar:Patient) " +
                "WHERE abs(p.age - similar.age) <= 5 " +
                "  AND similar.patientId <> $pid " +
                "MATCH (p)-[:HAS_CONDITION]->(pc:Condition) " +
                "MATCH (similar)-[:HAS_CONDITION]->(sc:Condition) " +
                "WITH p, similar, " +
                "     collect(DISTINCT pc.code) AS pConditions, " +
                "     collect(DISTINCT sc.code) AS sConditions " +
                "WITH similar, " +
                "     size([x IN pConditions WHERE x IN sConditions]) * 2.0 / " +
                "     size(pConditions + sConditions) AS similarity " +
                "WHERE similarity > 0.7 " +
                "OPTIONAL MATCH (similar)-[:HAD_OUTCOME]->(outcome:ClinicalOutcome) " +
                "WHERE outcome.timestamp > datetime() - duration('P30D') " +
                "RETURN similar.patientId AS patientId, " +
                "       similarity, " +
                "       outcome.type AS outcome30Day, " +
                "       outcome.interventions AS keyInterventions, " +
                "       size([x IN pConditions WHERE x IN sConditions]) AS sharedConditions, " +
                "       abs(p.age - similar.age) AS ageDifference " +
                "ORDER BY similarity DESC " +
                "LIMIT $limit";

        try (Session session = driver.session()) {
            Result result = session.run(
                    query,
                    Map.of(
                            "pid", patientId,
                            "limit", limit
                    )
            );

            while (result.hasNext()) {
                Record record = result.next();
                SimilarPatient sp = new SimilarPatient();
                sp.setPatientId(record.get("patientId").asString());
                sp.setSimilarityScore(record.get("similarity").asDouble());

                // Outcome may be null
                if (!record.get("outcome30Day").isNull()) {
                    sp.setOutcome30Day(record.get("outcome30Day").asString());
                }

                // Interventions may be null
                if (!record.get("keyInterventions").isNull()) {
                    sp.setKeyInterventions(record.get("keyInterventions").asList(Value::asString));
                }

                sp.setSharedConditions(record.get("sharedConditions").asInt());
                sp.setAgeDifference(record.get("ageDifference").asInt());

                similarPatients.add(sp);
            }

            logger.info("Found {} similar patients for {}", similarPatients.size(), patientId);
            return similarPatients;

        } catch (Exception e) {
            logger.error("Error finding similar patients for {}: {}", patientId, e.getMessage(), e);
            return similarPatients;
        }
    }

    /**
     * Get cohort analytics and statistics
     *
     * Aggregates cohort-level metrics:
     * - Cohort size
     * - 30-day readmission rate
     * - Average vital signs
     *
     * Spec: Lines 293-306 of MODULE2_ADVANCED_ENHANCEMENTS.md
     *
     * @param patientId Patient ID (to identify their cohort)
     * @return Cohort insights or null if not found
     */
    public CohortInsights getCohortAnalytics(String patientId) {
        // Cypher query from spec (lines 293-306)
        String query =
                "MATCH (p:Patient {patientId: $pid})-[:MEMBER_OF_COHORT]->(c:RiskCohort) " +
                "MATCH (member:Patient)-[:MEMBER_OF_COHORT]->(c) " +
                "WITH c, count(distinct member) AS cohortSize, collect(member) AS members " +
                "OPTIONAL MATCH (member)-[:HAD_OUTCOME]->(outcome:ClinicalOutcome) " +
                "WHERE outcome.readmitted = true " +
                "  AND outcome.timestamp > datetime() - duration('P30D') " +
                "WITH c, cohortSize, members, count(outcome) AS readmissions " +
                "RETURN c.name AS cohortName, " +
                "       cohortSize, " +
                "       toFloat(readmissions) / cohortSize AS readmissionRate30Day, " +
                "       avg([m IN members | m.systolicBP]) AS avgSystolicBP, " +
                "       avg([m IN members | m.diastolicBP]) AS avgDiastolicBP, " +
                "       avg([m IN members | m.heartRate]) AS avgHeartRate, " +
                "       size([m IN members WHERE m.active = true]) AS activeMembers";

        try (Session session = driver.session()) {
            Result result = session.run(query, Map.of("pid", patientId));

            if (result.hasNext()) {
                Record record = result.next();
                CohortInsights insights = new CohortInsights();

                insights.setCohortName(record.get("cohortName").asString());
                insights.setCohortSize(record.get("cohortSize").asInt());
                insights.setReadmissionRate30Day(record.get("readmissionRate30Day").asDouble());

                // Vital averages (may be null)
                if (!record.get("avgSystolicBP").isNull()) {
                    insights.setAvgSystolicBP(record.get("avgSystolicBP").asDouble());
                }
                if (!record.get("avgDiastolicBP").isNull()) {
                    insights.setAvgDiastolicBP(record.get("avgDiastolicBP").asDouble());
                }
                if (!record.get("avgHeartRate").isNull()) {
                    insights.setAvgHeartRate(record.get("avgHeartRate").asDouble());
                }
                if (!record.get("activeMembers").isNull()) {
                    insights.setActiveMembers(record.get("activeMembers").asInt());
                }

                // Derive risk level from readmission rate
                double readmissionRate = insights.getReadmissionRate30Day();
                String riskLevel = readmissionRate >= 0.3 ? "HIGH" :
                                   readmissionRate >= 0.15 ? "MEDIUM" : "LOW";
                insights.setRiskLevel(riskLevel);

                logger.info("Cohort analytics for {}: {} members, {:.1f}% readmission rate",
                           insights.getCohortName(), insights.getCohortSize(),
                           readmissionRate * 100);

                return insights;
            }

            logger.warn("No cohort found for patient {}", patientId);
            return null;

        } catch (Exception e) {
            logger.error("Error getting cohort analytics for {}: {}", patientId, e.getMessage(), e);
            return null;
        }
    }

    /**
     * Find common successful interventions from similar patients
     *
     * Analyzes outcomes of similar patients to identify successful treatment patterns
     *
     * @param patientId Target patient ID
     * @param snapshot Patient data
     * @return Map of intervention to success count
     */
    public Map<String, Integer> findSuccessfulInterventions(String patientId, PatientSnapshot snapshot) {
        Map<String, Integer> interventionCounts = new HashMap<>();

        List<SimilarPatient> similarPatients = findSimilarPatients(patientId, snapshot, 10);

        for (SimilarPatient sp : similarPatients) {
            // Only count interventions from patients with positive outcomes
            if ("STABLE".equals(sp.getOutcome30Day()) || "IMPROVED".equals(sp.getOutcome30Day())) {
                if (sp.getKeyInterventions() != null) {
                    for (String intervention : sp.getKeyInterventions()) {
                        interventionCounts.merge(intervention, 1, Integer::sum);
                    }
                }
            }
        }

        return interventionCounts;
    }

    /**
     * Get patient risk trajectory over time
     *
     * Tracks changes in vital signs and risk scores over recent history
     *
     * @param patientId Patient ID
     * @param daysBack How many days of history to retrieve
     * @return List of historical vital measurements
     */
    public List<Map<String, Object>> getRiskTrajectory(String patientId, int daysBack) {
        List<Map<String, Object>> trajectory = new ArrayList<>();

        String query =
                "MATCH (p:Patient {patientId: $pid})-[:HAS_VITAL_MEASUREMENT]->(vm:VitalMeasurement) " +
                "WHERE vm.timestamp > datetime() - duration({days: $daysBack}) " +
                "RETURN vm.timestamp AS timestamp, " +
                "       vm.systolicBP AS systolicBP, " +
                "       vm.diastolicBP AS diastolicBP, " +
                "       vm.heartRate AS heartRate, " +
                "       vm.oxygenSaturation AS oxygenSaturation, " +
                "       vm.temperature AS temperature " +
                "ORDER BY vm.timestamp ASC";

        try (Session session = driver.session()) {
            Result result = session.run(query, Map.of("pid", patientId, "daysBack", daysBack));

            while (result.hasNext()) {
                Record record = result.next();
                Map<String, Object> measurement = new HashMap<>();

                measurement.put("timestamp", record.get("timestamp").asString());
                if (!record.get("systolicBP").isNull()) {
                    measurement.put("systolicBP", record.get("systolicBP").asInt());
                }
                if (!record.get("diastolicBP").isNull()) {
                    measurement.put("diastolicBP", record.get("diastolicBP").asInt());
                }
                if (!record.get("heartRate").isNull()) {
                    measurement.put("heartRate", record.get("heartRate").asInt());
                }
                if (!record.get("oxygenSaturation").isNull()) {
                    measurement.put("oxygenSaturation", record.get("oxygenSaturation").asInt());
                }
                if (!record.get("temperature").isNull()) {
                    measurement.put("temperature", record.get("temperature").asDouble());
                }

                trajectory.add(measurement);
            }

            logger.info("Retrieved {} vital measurements for patient {} over last {} days",
                       trajectory.size(), patientId, daysBack);
            return trajectory;

        } catch (Exception e) {
            logger.error("Error getting risk trajectory for {}: {}", patientId, e.getMessage(), e);
            return trajectory;
        }
    }

    /**
     * Close the driver connection
     */
    public void close() {
        if (driver != null) {
            driver.close();
        }
    }
}
