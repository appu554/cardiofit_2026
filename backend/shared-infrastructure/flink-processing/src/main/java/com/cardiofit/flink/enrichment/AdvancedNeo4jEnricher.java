package com.cardiofit.flink.enrichment;

import org.apache.flink.configuration.Configuration;
import org.neo4j.driver.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * Advanced Neo4j Enricher with Similar Patient Queries and Predictive Analytics
 */
public class AdvancedNeo4jEnricher implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(AdvancedNeo4jEnricher.class);

    private transient Driver driver;
    private final String uri;
    private final String username;
    private final String password;
    private final long queryTimeout = 5000; // 5 seconds

    public AdvancedNeo4jEnricher(String uri, String username, String password) {
        this.uri = uri;
        this.username = username;
        this.password = password;
    }

    public void open(Configuration config) {
        this.driver = GraphDatabase.driver(uri, AuthTokens.basic(username, password),
            Config.builder()
                .withConnectionTimeout(10, TimeUnit.SECONDS)
                .withMaxConnectionPoolSize(50)
                .build());
        LOG.info("Advanced Neo4j connection established");
    }

    public void close() {
        if (driver != null) {
            driver.close();
        }
    }

    /**
     * Find similar patients based on clinical profile
     */
    public CompletableFuture<List<SimilarPatient>> findSimilarPatients(
            String patientId, Map<String, Object> patientProfile) {

        return CompletableFuture.supplyAsync(() -> {
            List<SimilarPatient> similarPatients = new ArrayList<>();

            try (Session session = driver.session()) {
                // Extract key features from profile
                List<String> diagnoses = (List<String>) patientProfile.getOrDefault("diagnoses", new ArrayList<>());
                Integer age = (Integer) patientProfile.get("age");
                String gender = (String) patientProfile.get("gender");
                List<String> medications = (List<String>) patientProfile.getOrDefault("medications", new ArrayList<>());

                // Build similarity query
                String query = buildSimilarityQuery();

                Map<String, Object> params = new HashMap<>();
                params.put("patientId", patientId);
                params.put("diagnoses", diagnoses);
                params.put("age", age != null ? age : 0);
                params.put("ageRange", 5); // +/- 5 years
                params.put("gender", gender);
                params.put("medications", medications);
                params.put("limit", 10);

                Result result = session.run(query, params);

                while (result.hasNext()) {
                    org.neo4j.driver.Record record = result.next();
                    SimilarPatient similar = new SimilarPatient();
                    similar.setPatientId(record.get("similarId").asString());
                    similar.setSimilarityScore(record.get("similarity").asDouble());
                    similar.setSharedDiagnoses(record.get("sharedDiagnoses").asList(Value::asString));
                    similar.setSharedMedications(record.get("sharedMedications").asList(Value::asString));
                    similar.setOutcomes(extractOutcomes(record));
                    similarPatients.add(similar);
                }

                LOG.debug("Found {} similar patients for {}", similarPatients.size(), patientId);

            } catch (Exception e) {
                LOG.error("Error finding similar patients", e);
            }

            return similarPatients;
        });
    }

    /**
     * Get cohort statistics for patients with similar conditions
     */
    public CompletableFuture<CohortStatistics> getCohortStatistics(
            List<String> diagnoses, Map<String, Object> demographics) {

        return CompletableFuture.supplyAsync(() -> {
            CohortStatistics stats = new CohortStatistics();

            try (Session session = driver.session()) {
                String query = buildCohortStatisticsQuery();

                Map<String, Object> params = new HashMap<>();
                params.put("diagnoses", diagnoses);
                params.put("minAge", demographics.getOrDefault("minAge", 0));
                params.put("maxAge", demographics.getOrDefault("maxAge", 120));
                params.put("gender", demographics.get("gender"));

                Result result = session.run(query, params);

                if (result.hasNext()) {
                    org.neo4j.driver.Record record = result.next();
                    stats.setCohortSize(record.get("cohortSize").asInt());
                    stats.setAverageAge(record.get("avgAge").asDouble());
                    stats.setMortalityRate(record.get("mortalityRate").asDouble());
                    stats.setReadmissionRate(record.get("readmissionRate").asDouble());
                    stats.setAverageLengthOfStay(record.get("avgLOS").asDouble());
                    stats.setCommonMedications(record.get("commonMeds").asList(Value::asString));
                    stats.setCommonComplications(record.get("complications").asList(Value::asString));
                }

                LOG.debug("Retrieved cohort statistics for {} patients", stats.getCohortSize());

            } catch (Exception e) {
                LOG.error("Error getting cohort statistics", e);
            }

            return stats;
        });
    }

    /**
     * Predict patient trajectory based on similar patient outcomes
     */
    public CompletableFuture<PatientTrajectory> predictPatientTrajectory(
            String patientId, List<SimilarPatient> similarPatients) {

        return CompletableFuture.supplyAsync(() -> {
            PatientTrajectory trajectory = new PatientTrajectory();
            trajectory.setPatientId(patientId);

            try (Session session = driver.session()) {
                String query = buildTrajectoryPredictionQuery();

                // Collect similar patient IDs
                List<String> similarIds = new ArrayList<>();
                for (SimilarPatient sp : similarPatients) {
                    similarIds.add(sp.getPatientId());
                }

                Map<String, Object> params = new HashMap<>();
                params.put("patientId", patientId);
                params.put("similarIds", similarIds);
                params.put("timeWindow", 90); // 90 days prediction window

                Result result = session.run(query, params);

                if (result.hasNext()) {
                    org.neo4j.driver.Record record = result.next();

                    // Risk predictions
                    Map<String, Double> risks = new HashMap<>();
                    risks.put("readmission30Day", record.get("readmissionRisk").asDouble(0.0));
                    risks.put("complication", record.get("complicationRisk").asDouble(0.0));
                    risks.put("deterioration", record.get("deteriorationRisk").asDouble(0.0));
                    trajectory.setPredictedRisks(risks);

                    // Likely next events
                    List<String> nextEvents = record.get("likelyEvents").asList(Value::asString);
                    trajectory.setLikelyNextEvents(nextEvents);

                    // Recommended interventions
                    List<String> interventions = record.get("interventions").asList(Value::asString);
                    trajectory.setRecommendedInterventions(interventions);
                }

                LOG.debug("Predicted trajectory for patient {}", patientId);

            } catch (Exception e) {
                LOG.error("Error predicting patient trajectory", e);
            }

            return trajectory;
        });
    }

    /**
     * Get treatment patterns from successful similar patient cases
     */
    public CompletableFuture<List<TreatmentPattern>> getSuccessfulTreatmentPatterns(
            String patientId, List<String> diagnoses) {

        return CompletableFuture.supplyAsync(() -> {
            List<TreatmentPattern> patterns = new ArrayList<>();

            try (Session session = driver.session()) {
                String query = buildTreatmentPatternQuery();

                Map<String, Object> params = new HashMap<>();
                params.put("patientId", patientId);
                params.put("diagnoses", diagnoses);
                params.put("successThreshold", 0.8); // 80% success rate
                params.put("minPatients", 5); // Minimum 5 patients for pattern

                Result result = session.run(query, params);

                while (result.hasNext()) {
                    org.neo4j.driver.Record record = result.next();
                    TreatmentPattern pattern = new TreatmentPattern();
                    pattern.setPatternId(record.get("patternId").asString());
                    pattern.setDescription(record.get("description").asString());
                    pattern.setMedications(record.get("medications").asList(Value::asString));
                    pattern.setProcedures(record.get("procedures").asList(Value::asString));
                    pattern.setSuccessRate(record.get("successRate").asDouble());
                    pattern.setPatientCount(record.get("patientCount").asInt());
                    pattern.setAverageTimeToImprovement(record.get("avgTimeToImprovement").asInt());
                    patterns.add(pattern);
                }

                LOG.debug("Found {} successful treatment patterns", patterns.size());

            } catch (Exception e) {
                LOG.error("Error getting treatment patterns", e);
            }

            return patterns;
        });
    }

    /**
     * Build similarity query
     */
    private String buildSimilarityQuery() {
        return "MATCH (p:Patient {id: $patientId}) " +
               "MATCH (similar:Patient) " +
               "WHERE similar.id <> p.id " +
               "AND abs(similar.age - $age) <= $ageRange " +
               "AND ($gender IS NULL OR similar.gender = $gender) " +
               "WITH p, similar " +

               // Match shared diagnoses
               "OPTIONAL MATCH (p)-[:HAS_CONDITION]->(d:Diagnosis)<-[:HAS_CONDITION]-(similar) " +
               "WHERE d.code IN $diagnoses " +
               "WITH p, similar, collect(DISTINCT d.code) as sharedDiagnoses " +

               // Match shared medications
               "OPTIONAL MATCH (p)-[:TAKES_MEDICATION]->(m:Medication)<-[:TAKES_MEDICATION]-(similar) " +
               "WHERE m.code IN $medications " +
               "WITH p, similar, sharedDiagnoses, collect(DISTINCT m.code) as sharedMedications " +

               // Calculate similarity score
               "WITH similar.id as similarId, " +
               "     (size(sharedDiagnoses) * 0.4 + " +
               "      size(sharedMedications) * 0.3 + " +
               "      (1.0 / (1.0 + abs(similar.age - $age))) * 0.3) as similarity, " +
               "     sharedDiagnoses, sharedMedications, similar " +

               // Get outcomes
               "OPTIONAL MATCH (similar)-[:HAD_OUTCOME]->(outcome:Outcome) " +
               "WITH similarId, similarity, sharedDiagnoses, sharedMedications, " +
               "     collect({type: outcome.type, value: outcome.value, date: outcome.date}) as outcomes " +

               "WHERE similarity > 0.3 " +
               "RETURN similarId, similarity, sharedDiagnoses, sharedMedications, outcomes " +
               "ORDER BY similarity DESC " +
               "LIMIT $limit";
    }

    /**
     * Build cohort statistics query
     */
    private String buildCohortStatisticsQuery() {
        return "MATCH (p:Patient) " +
               "WHERE p.age >= $minAge AND p.age <= $maxAge " +
               "AND ($gender IS NULL OR p.gender = $gender) " +
               "MATCH (p)-[:HAS_CONDITION]->(d:Diagnosis) " +
               "WHERE d.code IN $diagnoses " +
               "WITH p " +

               // Basic statistics
               "WITH count(DISTINCT p) as cohortSize, " +
               "     avg(p.age) as avgAge, " +
               "     collect(DISTINCT p) as patients " +

               // Mortality rate
               "UNWIND patients as patient " +
               "OPTIONAL MATCH (patient)-[:HAD_OUTCOME]->(death:Outcome {type: 'DEATH'}) " +
               "WITH cohortSize, avgAge, patients, " +
               "     count(death) * 1.0 / cohortSize as mortalityRate " +

               // Readmission rate
               "UNWIND patients as patient " +
               "OPTIONAL MATCH (patient)-[:HAD_OUTCOME]->(readmit:Outcome {type: 'READMISSION_30DAY'}) " +
               "WITH cohortSize, avgAge, mortalityRate, patients, " +
               "     count(readmit) * 1.0 / cohortSize as readmissionRate " +

               // Average length of stay
               "UNWIND patients as patient " +
               "OPTIONAL MATCH (patient)-[:HAD_ENCOUNTER]->(enc:Encounter) " +
               "WITH cohortSize, avgAge, mortalityRate, readmissionRate, " +
               "     avg(enc.lengthOfStay) as avgLOS, patients " +

               // Common medications
               "UNWIND patients as patient " +
               "OPTIONAL MATCH (patient)-[:TAKES_MEDICATION]->(med:Medication) " +
               "WITH cohortSize, avgAge, mortalityRate, readmissionRate, avgLOS, " +
               "     collect(med.name) as allMeds, patients " +

               // Common complications
               "UNWIND patients as patient " +
               "OPTIONAL MATCH (patient)-[:HAD_COMPLICATION]->(comp:Complication) " +
               "WITH cohortSize, avgAge, mortalityRate, readmissionRate, avgLOS, " +
               "     [m IN allMeds | m][0..5] as commonMeds, " +
               "     collect(comp.name) as allComps " +

               "RETURN cohortSize, avgAge, mortalityRate, readmissionRate, avgLOS, " +
               "       commonMeds, [c IN allComps | c][0..5] as complications";
    }

    /**
     * Build trajectory prediction query
     */
    private String buildTrajectoryPredictionQuery() {
        return "MATCH (p:Patient {id: $patientId}) " +
               "MATCH (similar:Patient) WHERE similar.id IN $similarIds " +

               // Get outcomes within time window
               "MATCH (similar)-[:HAD_OUTCOME]->(outcome:Outcome) " +
               "WHERE outcome.daysFromDiagnosis <= $timeWindow " +
               "WITH p, collect({patient: similar.id, outcome: outcome}) as outcomes " +

               // Calculate risk scores
               "WITH p, " +
               "     size([o IN outcomes WHERE o.outcome.type = 'READMISSION_30DAY']) * 1.0 / size(outcomes) as readmissionRisk, " +
               "     size([o IN outcomes WHERE o.outcome.type = 'COMPLICATION']) * 1.0 / size(outcomes) as complicationRisk, " +
               "     size([o IN outcomes WHERE o.outcome.type = 'DETERIORATION']) * 1.0 / size(outcomes) as deteriorationRisk, " +
               "     outcomes " +

               // Get likely next events
               "UNWIND outcomes as o " +
               "WITH p, readmissionRisk, complicationRisk, deteriorationRisk, " +
               "     collect(DISTINCT o.outcome.nextEvent) as likelyEvents " +

               // Get recommended interventions
               "MATCH (intervention:Intervention)-[:PREVENTS]->(outcome:Outcome) " +
               "WHERE outcome.type IN ['READMISSION_30DAY', 'COMPLICATION', 'DETERIORATION'] " +
               "AND (readmissionRisk > 0.3 OR complicationRisk > 0.3 OR deteriorationRisk > 0.3) " +
               "WITH p, readmissionRisk, complicationRisk, deteriorationRisk, likelyEvents, " +
               "     collect(DISTINCT intervention.name) as interventions " +

               "RETURN readmissionRisk, complicationRisk, deteriorationRisk, " +
               "       likelyEvents[0..5], interventions[0..5] as interventions";
    }

    /**
     * Build treatment pattern query
     */
    private String buildTreatmentPatternQuery() {
        return "MATCH (p:Patient)-[:HAS_CONDITION]->(d:Diagnosis) " +
               "WHERE d.code IN $diagnoses " +
               "MATCH (p)-[:RECEIVED_TREATMENT]->(t:Treatment) " +
               "MATCH (p)-[:HAD_OUTCOME]->(outcome:Outcome {type: 'IMPROVEMENT'}) " +
               "WHERE outcome.daysFromTreatment < 90 " +

               // Group treatments into patterns
               "WITH t.medications as medications, t.procedures as procedures, " +
               "     count(DISTINCT p) as patientCount, " +
               "     avg(outcome.daysFromTreatment) as avgTimeToImprovement, " +
               "     count(outcome) * 1.0 / count(DISTINCT p) as successRate " +

               "WHERE patientCount >= $minPatients " +
               "AND successRate >= $successThreshold " +

               "RETURN toString(id(t)) as patternId, " +
               "       'Treatment with ' + head(medications) as description, " +
               "       medications, procedures, successRate, patientCount, " +
               "       toInteger(avgTimeToImprovement) as avgTimeToImprovement " +
               "ORDER BY successRate DESC, patientCount DESC " +
               "LIMIT 10";
    }

    /**
     * Extract outcomes from record
     */
    private Map<String, Object> extractOutcomes(org.neo4j.driver.Record record) {
        Map<String, Object> outcomes = new HashMap<>();
        try {
            List<Map<String, Object>> outcomeList = record.get("outcomes")
                .asList(v -> v.asMap());

            for (Map<String, Object> outcome : outcomeList) {
                String type = (String) outcome.get("type");
                Object value = outcome.get("value");
                outcomes.put(type, value);
            }
        } catch (Exception e) {
            LOG.warn("Error extracting outcomes: {}", e.getMessage());
        }
        return outcomes;
    }

    // Supporting classes

    public static class SimilarPatient implements Serializable {
        private String patientId;
        private double similarityScore;
        private List<String> sharedDiagnoses;
        private List<String> sharedMedications;
        private Map<String, Object> outcomes;

        // Getters and setters
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public double getSimilarityScore() { return similarityScore; }
        public void setSimilarityScore(double similarityScore) { this.similarityScore = similarityScore; }

        public List<String> getSharedDiagnoses() { return sharedDiagnoses; }
        public void setSharedDiagnoses(List<String> sharedDiagnoses) { this.sharedDiagnoses = sharedDiagnoses; }

        public List<String> getSharedMedications() { return sharedMedications; }
        public void setSharedMedications(List<String> sharedMedications) { this.sharedMedications = sharedMedications; }

        public Map<String, Object> getOutcomes() { return outcomes; }
        public void setOutcomes(Map<String, Object> outcomes) { this.outcomes = outcomes; }
    }

    public static class CohortStatistics implements Serializable {
        private int cohortSize;
        private double averageAge;
        private double mortalityRate;
        private double readmissionRate;
        private double averageLengthOfStay;
        private List<String> commonMedications;
        private List<String> commonComplications;

        // Getters and setters
        public int getCohortSize() { return cohortSize; }
        public void setCohortSize(int cohortSize) { this.cohortSize = cohortSize; }

        public double getAverageAge() { return averageAge; }
        public void setAverageAge(double averageAge) { this.averageAge = averageAge; }

        public double getMortalityRate() { return mortalityRate; }
        public void setMortalityRate(double mortalityRate) { this.mortalityRate = mortalityRate; }

        public double getReadmissionRate() { return readmissionRate; }
        public void setReadmissionRate(double readmissionRate) { this.readmissionRate = readmissionRate; }

        public double getAverageLengthOfStay() { return averageLengthOfStay; }
        public void setAverageLengthOfStay(double averageLengthOfStay) { this.averageLengthOfStay = averageLengthOfStay; }

        public List<String> getCommonMedications() { return commonMedications; }
        public void setCommonMedications(List<String> commonMedications) { this.commonMedications = commonMedications; }

        public List<String> getCommonComplications() { return commonComplications; }
        public void setCommonComplications(List<String> commonComplications) { this.commonComplications = commonComplications; }
    }

    public static class PatientTrajectory implements Serializable {
        private String patientId;
        private Map<String, Double> predictedRisks;
        private List<String> likelyNextEvents;
        private List<String> recommendedInterventions;

        // Getters and setters
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public Map<String, Double> getPredictedRisks() { return predictedRisks; }
        public void setPredictedRisks(Map<String, Double> predictedRisks) { this.predictedRisks = predictedRisks; }

        public List<String> getLikelyNextEvents() { return likelyNextEvents; }
        public void setLikelyNextEvents(List<String> likelyNextEvents) { this.likelyNextEvents = likelyNextEvents; }

        public List<String> getRecommendedInterventions() { return recommendedInterventions; }
        public void setRecommendedInterventions(List<String> recommendedInterventions) { this.recommendedInterventions = recommendedInterventions; }
    }

    public static class TreatmentPattern implements Serializable {
        private String patternId;
        private String description;
        private List<String> medications;
        private List<String> procedures;
        private double successRate;
        private int patientCount;
        private int averageTimeToImprovement;

        // Getters and setters
        public String getPatternId() { return patternId; }
        public void setPatternId(String patternId) { this.patternId = patternId; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public List<String> getMedications() { return medications; }
        public void setMedications(List<String> medications) { this.medications = medications; }

        public List<String> getProcedures() { return procedures; }
        public void setProcedures(List<String> procedures) { this.procedures = procedures; }

        public double getSuccessRate() { return successRate; }
        public void setSuccessRate(double successRate) { this.successRate = successRate; }

        public int getPatientCount() { return patientCount; }
        public void setPatientCount(int patientCount) { this.patientCount = patientCount; }

        public int getAverageTimeToImprovement() { return averageTimeToImprovement; }
        public void setAverageTimeToImprovement(int averageTimeToImprovement) { this.averageTimeToImprovement = averageTimeToImprovement; }
    }
}