package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

@JsonIgnoreProperties(ignoreUnknown = true)
public class KB20StateUpdate implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum UpdateOperation {
        REPLACE,
        MERGE,
        APPEND,
        INCREMENT
    }

    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("operation") private UpdateOperation operation;
    @JsonProperty("field_path") private String fieldPath;
    @JsonProperty("value") private Object value;
    @JsonProperty("previous_value") private Object previousValue;
    @JsonProperty("source_module") private String sourceModule;
    @JsonProperty("update_timestamp") private long updateTimestamp;

    public KB20StateUpdate() {}

    public static Builder builder() { return new Builder(); }

    public String getPatientId() { return patientId; }
    public UpdateOperation getOperation() { return operation; }
    public String getFieldPath() { return fieldPath; }
    public Object getValue() { return value; }
    public Object getPreviousValue() { return previousValue; }
    public String getSourceModule() { return sourceModule; }
    public long getUpdateTimestamp() { return updateTimestamp; }

    public static class Builder {
        private final KB20StateUpdate u = new KB20StateUpdate();

        public Builder patientId(String id) { u.patientId = id; return this; }
        public Builder operation(UpdateOperation op) { u.operation = op; return this; }
        public Builder fieldPath(String path) { u.fieldPath = path; return this; }
        public Builder value(Object val) { u.value = val; return this; }
        public Builder previousValue(Object prev) { u.previousValue = prev; return this; }
        public Builder sourceModule(String m) { u.sourceModule = m; return this; }
        public Builder updateTimestamp(long t) { u.updateTimestamp = t; return this; }
        public KB20StateUpdate build() { return u; }
    }
}
