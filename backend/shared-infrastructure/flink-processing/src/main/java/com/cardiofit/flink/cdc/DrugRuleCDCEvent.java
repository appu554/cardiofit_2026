package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

/**
 * CDC Event Model for KB1 Drug Rules & Dose Calculations
 *
 * Parses Debezium CDC events from:
 * - kb1.drug_rule_packs.changes
 * - kb1.dose_calculations.changes
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class DrugRuleCDCEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("payload")
    private Payload payload;

    public DrugRuleCDCEvent() {}

    public Payload getPayload() {
        return payload;
    }

    public void setPayload(Payload payload) {
        this.payload = payload;
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Payload implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("op")
        private String operation;

        @JsonProperty("before")
        private DrugRuleData before;

        @JsonProperty("after")
        private DrugRuleData after;

        @JsonProperty("source")
        private ProtocolCDCEvent.Source source;

        @JsonProperty("ts_ms")
        private Long timestampMs;

        public Payload() {}

        public String getOperation() { return operation; }
        public void setOperation(String operation) { this.operation = operation; }
        public DrugRuleData getBefore() { return before; }
        public void setBefore(DrugRuleData before) { this.before = before; }
        public DrugRuleData getAfter() { return after; }
        public void setAfter(DrugRuleData after) { this.after = after; }
        public ProtocolCDCEvent.Source getSource() { return source; }
        public void setSource(ProtocolCDCEvent.Source source) { this.source = source; }
        public Long getTimestampMs() { return timestampMs; }
        public void setTimestampMs(Long timestampMs) { this.timestampMs = timestampMs; }

        public boolean isCreate() { return "c".equals(operation) || "r".equals(operation); }
        public boolean isUpdate() { return "u".equals(operation); }
        public boolean isDelete() { return "d".equals(operation); }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class DrugRuleData implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("drug_id")
        private String drugId;

        @JsonProperty("version")
        private String version;

        @JsonProperty("content_sha")
        private String contentSha;

        @JsonProperty("signed_by")
        private String signedBy;

        @JsonProperty("signature_valid")
        private Boolean signatureValid;

        @JsonProperty("regions")
        private Object regions; // JSONB array

        @JsonProperty("content")
        private Object content; // JSONB rule pack content

        @JsonProperty("calculation_rules")
        private Object calculationRules; // JSONB for dose calculations

        @JsonProperty("min_dose")
        private Double minDose;

        @JsonProperty("max_dose")
        private Double maxDose;

        @JsonProperty("unit")
        private String unit;

        @JsonProperty("created_at")
        private String createdAt;

        @JsonProperty("updated_at")
        private String updatedAt;

        public DrugRuleData() {}

        public String getDrugId() { return drugId; }
        public void setDrugId(String drugId) { this.drugId = drugId; }
        public String getVersion() { return version; }
        public void setVersion(String version) { this.version = version; }
        public String getContentSha() { return contentSha; }
        public void setContentSha(String contentSha) { this.contentSha = contentSha; }
        public String getSignedBy() { return signedBy; }
        public void setSignedBy(String signedBy) { this.signedBy = signedBy; }
        public Boolean getSignatureValid() { return signatureValid; }
        public void setSignatureValid(Boolean signatureValid) { this.signatureValid = signatureValid; }
        public Object getRegions() { return regions; }
        public void setRegions(Object regions) { this.regions = regions; }
        public Object getContent() { return content; }
        public void setContent(Object content) { this.content = content; }
        public Object getCalculationRules() { return calculationRules; }
        public void setCalculationRules(Object calculationRules) { this.calculationRules = calculationRules; }
        public Double getMinDose() { return minDose; }
        public void setMinDose(Double minDose) { this.minDose = minDose; }
        public Double getMaxDose() { return maxDose; }
        public void setMaxDose(Double maxDose) { this.maxDose = maxDose; }
        public String getUnit() { return unit; }
        public void setUnit(String unit) { this.unit = unit; }
        public String getCreatedAt() { return createdAt; }
        public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
        public String getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(String updatedAt) { this.updatedAt = updatedAt; }

        @Override
        public String toString() {
            return "DrugRuleData{" +
                    "drugId='" + drugId + '\'' +
                    ", version='" + version + '\'' +
                    ", signatureValid=" + signatureValid +
                    '}';
        }
    }

    @Override
    public String toString() {
        if (payload == null) return "DrugRuleCDCEvent{payload=null}";
        return "DrugRuleCDCEvent{" +
                "op=" + payload.getOperation() +
                ", drugId=" + (payload.getAfter() != null ? payload.getAfter().getDrugId() :
                               payload.getBefore() != null ? payload.getBefore().getDrugId() : "null") +
                '}';
    }
}
