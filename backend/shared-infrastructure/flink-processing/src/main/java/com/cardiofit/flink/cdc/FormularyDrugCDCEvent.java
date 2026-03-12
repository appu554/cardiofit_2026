package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

/**
 * CDC Event Model for KB6 Formulary Drugs
 *
 * Parses Debezium CDC events from kb6.formulary_drugs.changes topic.
 *
 * Formulary drugs define institution-specific medication availability,
 * restrictions, and tier preferences for medication selection.
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class FormularyDrugCDCEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("payload")
    private Payload payload;

    public FormularyDrugCDCEvent() {}

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
        private FormularyData before;

        @JsonProperty("after")
        private FormularyData after;

        @JsonProperty("source")
        private ProtocolCDCEvent.Source source;

        @JsonProperty("ts_ms")
        private Long timestampMs;

        public Payload() {}

        public String getOperation() { return operation; }
        public void setOperation(String operation) { this.operation = operation; }
        public FormularyData getBefore() { return before; }
        public void setBefore(FormularyData before) { this.before = before; }
        public FormularyData getAfter() { return after; }
        public void setAfter(FormularyData after) { this.after = after; }
        public ProtocolCDCEvent.Source getSource() { return source; }
        public void setSource(ProtocolCDCEvent.Source source) { this.source = source; }
        public Long getTimestampMs() { return timestampMs; }
        public void setTimestampMs(Long timestampMs) { this.timestampMs = timestampMs; }

        public boolean isCreate() { return "c".equals(operation) || "r".equals(operation); }
        public boolean isUpdate() { return "u".equals(operation); }
        public boolean isDelete() { return "d".equals(operation); }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class FormularyData implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("drug_id")
        private String drugId;

        @JsonProperty("drug_name")
        private String drugName;

        @JsonProperty("generic_name")
        private String genericName;

        @JsonProperty("formulary_status")
        private String formularyStatus; // PREFERRED, RESTRICTED, NON_FORMULARY

        @JsonProperty("tier")
        private Integer tier; // 1, 2, 3 (cost tiers)

        @JsonProperty("restrictions")
        private Object restrictions; // JSONB

        @JsonProperty("requires_prior_auth")
        private Boolean requiresPriorAuth;

        @JsonProperty("therapeutic_class")
        private String therapeuticClass;

        @JsonProperty("alternatives")
        private Object alternatives; // JSONB array of alternative drugs

        @JsonProperty("institution_id")
        private String institutionId;

        @JsonProperty("created_at")
        private String createdAt;

        @JsonProperty("updated_at")
        private String updatedAt;

        public FormularyData() {}

        public String getDrugId() { return drugId; }
        public void setDrugId(String drugId) { this.drugId = drugId; }
        public String getDrugName() { return drugName; }
        public void setDrugName(String drugName) { this.drugName = drugName; }
        public String getGenericName() { return genericName; }
        public void setGenericName(String genericName) { this.genericName = genericName; }
        public String getFormularyStatus() { return formularyStatus; }
        public void setFormularyStatus(String formularyStatus) { this.formularyStatus = formularyStatus; }
        public Integer getTier() { return tier; }
        public void setTier(Integer tier) { this.tier = tier; }
        public Object getRestrictions() { return restrictions; }
        public void setRestrictions(Object restrictions) { this.restrictions = restrictions; }
        public Boolean getRequiresPriorAuth() { return requiresPriorAuth; }
        public void setRequiresPriorAuth(Boolean requiresPriorAuth) { this.requiresPriorAuth = requiresPriorAuth; }
        public String getTherapeuticClass() { return therapeuticClass; }
        public void setTherapeuticClass(String therapeuticClass) { this.therapeuticClass = therapeuticClass; }
        public Object getAlternatives() { return alternatives; }
        public void setAlternatives(Object alternatives) { this.alternatives = alternatives; }
        public String getInstitutionId() { return institutionId; }
        public void setInstitutionId(String institutionId) { this.institutionId = institutionId; }
        public String getCreatedAt() { return createdAt; }
        public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
        public String getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(String updatedAt) { this.updatedAt = updatedAt; }

        @Override
        public String toString() {
            return "FormularyData{" +
                    "drugId='" + drugId + '\'' +
                    ", drugName='" + drugName + '\'' +
                    ", formularyStatus='" + formularyStatus + '\'' +
                    ", tier=" + tier +
                    '}';
        }
    }

    @Override
    public String toString() {
        if (payload == null) return "FormularyDrugCDCEvent{payload=null}";
        return "FormularyDrugCDCEvent{" +
                "op=" + payload.getOperation() +
                ", drugId=" + (payload.getAfter() != null ? payload.getAfter().getDrugId() :
                               payload.getBefore() != null ? payload.getBefore().getDrugId() : "null") +
                '}';
    }
}
