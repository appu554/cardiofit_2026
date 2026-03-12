package com.cardiofit.flink.knowledgebase;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.yaml.snakeyaml.Yaml;

import java.io.InputStream;
import java.io.Serializable;
import java.util.*;
import java.util.stream.Collectors;

/**
 * DrugInteractionAnalyzer - Self-contained drug-drug interaction checker
 *
 * Loads drug interaction knowledge base from YAML and checks for interactions
 * between protocol medications and patient's active medications.
 */
public class DrugInteractionAnalyzer implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(DrugInteractionAnalyzer.class);

    private final List<DrugInteraction> interactions;

    /**
     * DrugInteraction model matching YAML structure
     */
    public static class DrugInteraction implements Serializable {
        private static final long serialVersionUID = 1L;

        private String interactionId;
        private String drug1Name;
        private String drug2Name;
        private String severity;
        private String mechanism;
        private String clinicalEffect;
        private String onset;
        private String management;
        private List<String> evidenceReferences;
        private Boolean blackBoxWarning;

        // Getters and setters
        public String getInteractionId() { return interactionId; }
        public void setInteractionId(String interactionId) { this.interactionId = interactionId; }

        public String getDrug1Name() { return drug1Name; }
        public void setDrug1Name(String drug1Name) { this.drug1Name = drug1Name; }

        public String getDrug2Name() { return drug2Name; }
        public void setDrug2Name(String drug2Name) { this.drug2Name = drug2Name; }

        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }

        public String getMechanism() { return mechanism; }
        public void setMechanism(String mechanism) { this.mechanism = mechanism; }

        public String getClinicalEffect() { return clinicalEffect; }
        public void setClinicalEffect(String clinicalEffect) { this.clinicalEffect = clinicalEffect; }

        public String getOnset() { return onset; }
        public void setOnset(String onset) { this.onset = onset; }

        public String getManagement() { return management; }
        public void setManagement(String management) { this.management = management; }

        public List<String> getEvidenceReferences() { return evidenceReferences; }
        public void setEvidenceReferences(List<String> evidenceReferences) {
            this.evidenceReferences = evidenceReferences;
        }

        public Boolean getBlackBoxWarning() { return blackBoxWarning; }
        public void setBlackBoxWarning(Boolean blackBoxWarning) {
            this.blackBoxWarning = blackBoxWarning;
        }
    }

    /**
     * InteractionWarning result model
     */
    public static class InteractionWarning implements Serializable {
        private static final long serialVersionUID = 1L;

        private String protocolMedication;
        private String activeMedication;
        private String severity;
        private String clinicalEffect;
        private String management;
        private String onset;
        private Boolean blackBoxWarning;
        private List<String> evidencePMIDs;

        public InteractionWarning() {}

        public InteractionWarning(String protocolMed, String activeMed, DrugInteraction interaction) {
            this.protocolMedication = protocolMed;
            this.activeMedication = activeMed;
            this.severity = interaction.getSeverity();
            this.clinicalEffect = interaction.getClinicalEffect();
            this.management = interaction.getManagement();
            this.onset = interaction.getOnset();
            this.blackBoxWarning = interaction.getBlackBoxWarning();
            this.evidencePMIDs = interaction.getEvidenceReferences();
        }

        // Getters and setters
        public String getProtocolMedication() { return protocolMedication; }
        public void setProtocolMedication(String protocolMedication) {
            this.protocolMedication = protocolMedication;
        }

        public String getActiveMedication() { return activeMedication; }
        public void setActiveMedication(String activeMedication) {
            this.activeMedication = activeMedication;
        }

        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }

        public String getClinicalEffect() { return clinicalEffect; }
        public void setClinicalEffect(String clinicalEffect) {
            this.clinicalEffect = clinicalEffect;
        }

        public String getManagement() { return management; }
        public void setManagement(String management) { this.management = management; }

        public String getOnset() { return onset; }
        public void setOnset(String onset) { this.onset = onset; }

        public Boolean getBlackBoxWarning() { return blackBoxWarning; }
        public void setBlackBoxWarning(Boolean blackBoxWarning) {
            this.blackBoxWarning = blackBoxWarning;
        }

        public List<String> getEvidencePMIDs() { return evidencePMIDs; }
        public void setEvidencePMIDs(List<String> evidencePMIDs) {
            this.evidencePMIDs = evidencePMIDs;
        }
    }

    /**
     * Constructor - loads drug interactions from YAML knowledge base
     */
    public DrugInteractionAnalyzer() {
        this.interactions = loadInteractions();
        LOG.info("✅ DrugInteractionAnalyzer initialized with {} interactions", interactions.size());
    }

    /**
     * Load drug interactions from YAML file in resources
     */
    @SuppressWarnings("unchecked")
    private List<DrugInteraction> loadInteractions() {
        List<DrugInteraction> loadedInteractions = new ArrayList<>();

        try {
            InputStream inputStream = getClass().getClassLoader()
                .getResourceAsStream("knowledge-base/drug-interactions/major-interactions.yaml");

            if (inputStream == null) {
                LOG.error("❌ Could not find drug-interactions/major-interactions.yaml in resources");
                return loadedInteractions;
            }

            Yaml yaml = new Yaml();
            Map<String, Object> data = yaml.load(inputStream);

            List<Map<String, Object>> interactionsList =
                (List<Map<String, Object>>) data.get("interactions");

            if (interactionsList == null) {
                LOG.warn("No interactions found in YAML");
                return loadedInteractions;
            }

            for (Map<String, Object> interactionMap : interactionsList) {
                DrugInteraction interaction = new DrugInteraction();

                interaction.setInteractionId((String) interactionMap.get("interactionId"));
                interaction.setDrug1Name((String) interactionMap.get("drug1Name"));
                interaction.setDrug2Name((String) interactionMap.get("drug2Name"));
                interaction.setSeverity((String) interactionMap.get("severity"));
                interaction.setMechanism((String) interactionMap.get("mechanism"));
                interaction.setClinicalEffect((String) interactionMap.get("clinicalEffect"));
                interaction.setOnset((String) interactionMap.get("onset"));
                interaction.setManagement((String) interactionMap.get("management"));
                interaction.setBlackBoxWarning((Boolean) interactionMap.get("blackBoxWarning"));

                Object evidenceObj = interactionMap.get("evidenceReferences");
                if (evidenceObj instanceof List) {
                    interaction.setEvidenceReferences((List<String>) evidenceObj);
                }

                loadedInteractions.add(interaction);
            }

            LOG.info("✅ Loaded {} drug interactions from knowledge base", loadedInteractions.size());

        } catch (Exception e) {
            LOG.error("❌ Failed to load drug interactions: {}", e.getMessage(), e);
        }

        return loadedInteractions;
    }

    /**
     * Analyze interactions between protocol medications and patient's active medications
     *
     * @param protocolMedications List of medication names from protocol actions
     * @param activeMedications Map of patient's active medications (code -> medication)
     * @return List of interaction warnings
     */
    public List<InteractionWarning> analyzeInteractions(
            List<String> protocolMedications,
            Map<String, ?> activeMedications) {

        List<InteractionWarning> warnings = new ArrayList<>();

        if (protocolMedications == null || protocolMedications.isEmpty()) {
            LOG.debug("No protocol medications to analyze");
            return warnings;
        }

        if (activeMedications == null || activeMedications.isEmpty()) {
            LOG.debug("No active medications to analyze");
            return warnings;
        }

        LOG.info("🔍 Analyzing interactions: {} protocol meds × {} active meds",
            protocolMedications.size(), activeMedications.size());

        // Extract active medication names
        List<String> activeMedNames = activeMedications.values().stream()
            .map(med -> extractMedicationName(med))
            .filter(Objects::nonNull)
            .collect(Collectors.toList());

        LOG.debug("Active medications: {}", activeMedNames);

        // Check each protocol medication against each active medication
        for (String protocolMed : protocolMedications) {
            for (String activeMed : activeMedNames) {
                DrugInteraction interaction = findInteraction(protocolMed, activeMed);

                if (interaction != null) {
                    InteractionWarning warning = new InteractionWarning(
                        protocolMed, activeMed, interaction);
                    warnings.add(warning);

                    LOG.warn("⚠️ INTERACTION DETECTED: {} + {} → {} ({})",
                        protocolMed, activeMed, interaction.getSeverity(),
                        interaction.getClinicalEffect());
                }
            }
        }

        if (warnings.isEmpty()) {
            LOG.info("✅ No drug interactions detected");
        } else {
            LOG.warn("⚠️ Found {} drug-drug interactions requiring attention", warnings.size());
        }

        return warnings;
    }

    /**
     * Extract medication name from various medication object types
     */
    @SuppressWarnings("unchecked")
    private String extractMedicationName(Object medicationObj) {
        if (medicationObj instanceof String) {
            return (String) medicationObj;
        }

        if (medicationObj instanceof Map) {
            Map<String, Object> medMap = (Map<String, Object>) medicationObj;

            // Try different field names
            if (medMap.containsKey("name")) {
                return (String) medMap.get("name");
            }
            if (medMap.containsKey("medicationName")) {
                return (String) medMap.get("medicationName");
            }
            if (medMap.containsKey("display")) {
                return (String) medMap.get("display");
            }
        }

        return null;
    }

    /**
     * Find interaction between two medications (fuzzy matching)
     */
    private DrugInteraction findInteraction(String med1, String med2) {
        String med1Lower = med1.toLowerCase();
        String med2Lower = med2.toLowerCase();

        for (DrugInteraction interaction : interactions) {
            String drug1 = interaction.getDrug1Name().toLowerCase();
            String drug2 = interaction.getDrug2Name().toLowerCase();

            // Bidirectional matching with fuzzy name matching
            if ((matchesDrugName(med1Lower, drug1) && matchesDrugName(med2Lower, drug2)) ||
                (matchesDrugName(med1Lower, drug2) && matchesDrugName(med2Lower, drug1))) {
                return interaction;
            }
        }

        return null;
    }

    /**
     * Fuzzy medication name matching
     * Handles generic names, brand names, combinations
     */
    private boolean matchesDrugName(String queryName, String dbName) {
        // Exact match
        if (queryName.equals(dbName)) {
            return true;
        }

        // Contains match (e.g., "Piperacillin-Tazobactam" contains "piperacillin")
        if (queryName.contains(dbName) || dbName.contains(queryName)) {
            return true;
        }

        // Handle parenthetical alternatives: "NSAIDs (Ibuprofen, Naproxen)"
        if (dbName.contains("(") && dbName.contains(")")) {
            String mainName = dbName.substring(0, dbName.indexOf("(")).trim();
            String alternatives = dbName.substring(dbName.indexOf("(") + 1, dbName.indexOf(")"));

            if (queryName.contains(mainName.toLowerCase())) {
                return true;
            }

            String[] altNames = alternatives.split(",");
            for (String alt : altNames) {
                if (queryName.contains(alt.trim().toLowerCase())) {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * Get summary statistics for loaded interactions
     */
    public Map<String, Integer> getStatistics() {
        Map<String, Integer> stats = new HashMap<>();

        stats.put("total_interactions", interactions.size());
        stats.put("major_severity", (int) interactions.stream()
            .filter(i -> "MAJOR".equals(i.getSeverity()))
            .count());
        stats.put("moderate_severity", (int) interactions.stream()
            .filter(i -> "MODERATE".equals(i.getSeverity()))
            .count());
        stats.put("black_box_warnings", (int) interactions.stream()
            .filter(i -> Boolean.TRUE.equals(i.getBlackBoxWarning()))
            .count());

        return stats;
    }
}
