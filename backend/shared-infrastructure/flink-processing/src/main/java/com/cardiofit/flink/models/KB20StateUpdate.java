package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class KB20StateUpdate implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum UpdateOperation {
        REPLACE,
        MERGE,
        APPEND,
        UPSERT
    }

    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("field_updates") private Map<String, Object> fieldUpdates;
    @JsonProperty("operation") private UpdateOperation operation;
    @JsonProperty("source_module") private String sourceModule;
    @JsonProperty("timestamp") private long timestamp;
    @JsonProperty("upsert_key") private String upsertKey;

    public KB20StateUpdate() {
        this.fieldUpdates = new HashMap<>();
    }

    public static Builder builder() { return new Builder(); }

    public String getPatientId() { return patientId; }
    public Map<String, Object> getFieldUpdates() { return fieldUpdates; }
    public UpdateOperation getOperation() { return operation; }
    public String getSourceModule() { return sourceModule; }
    public long getTimestamp() { return timestamp; }
    public String getUpsertKey() { return upsertKey; }

    public static class Builder {
        private final KB20StateUpdate u = new KB20StateUpdate();

        public Builder patientId(String id) { u.patientId = id; return this; }
        public Builder field(String name, Object value) { u.fieldUpdates.put(name, value); return this; }
        public Builder fieldUpdates(Map<String, Object> fields) { u.fieldUpdates.putAll(fields); return this; }
        public Builder operation(UpdateOperation op) { u.operation = op; return this; }
        public Builder sourceModule(String m) { u.sourceModule = m; return this; }
        public Builder timestamp(long t) { u.timestamp = t; return this; }
        public Builder upsertKey(String k) { u.upsertKey = k; return this; }
        public KB20StateUpdate build() { return u; }
    }
}
