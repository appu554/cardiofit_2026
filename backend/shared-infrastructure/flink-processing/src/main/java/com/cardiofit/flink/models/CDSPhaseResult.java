package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class CDSPhaseResult implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("phaseName")
    private String phaseName;

    @JsonProperty("active")
    private boolean active;

    @JsonProperty("durationMs")
    private long durationMs;

    @JsonProperty("details")
    private Map<String, Object> details;

    public CDSPhaseResult() {
        this.details = new HashMap<>();
    }

    public CDSPhaseResult(String phaseName) {
        this();
        this.phaseName = phaseName;
    }

    public void addDetail(String key, Object value) {
        this.details.put(key, value);
    }

    public Object getDetail(String key) {
        return this.details.get(key);
    }

    // Getters and setters
    public String getPhaseName() { return phaseName; }
    public void setPhaseName(String phaseName) { this.phaseName = phaseName; }
    public boolean isActive() { return active; }
    public void setActive(boolean active) { this.active = active; }
    public long getDurationMs() { return durationMs; }
    public void setDurationMs(long durationMs) { this.durationMs = durationMs; }
    public Map<String, Object> getDetails() { return details; }
    public void setDetails(Map<String, Object> details) { this.details = details; }

    @Override
    public String toString() {
        return String.format("CDSPhaseResult{phase='%s', active=%s, details=%d}",
                phaseName, active, details.size());
    }
}
