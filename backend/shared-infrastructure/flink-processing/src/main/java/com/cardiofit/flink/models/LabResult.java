package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;

/**
 * Laboratory test result.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class LabResult implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("labCode")
    private String labCode; // LOINC code (e.g., "2524-7" for Lactate)

    @JsonProperty("labType")
    private String labType; // "WBC", "Hemoglobin", "Glucose", etc.

    @JsonProperty("value")
    private Double value;

    @JsonProperty("unit")
    private String unit; // "K/uL", "g/dL", "mg/dL", etc.

    @JsonProperty("referenceRangeLow")
    private Double referenceRangeLow;

    @JsonProperty("referenceRangeHigh")
    private Double referenceRangeHigh;

    @JsonProperty("abnormal")
    private boolean abnormal; // True if outside reference range

    @JsonProperty("abnormalFlag")
    private String abnormalFlag; // "H" (high), "L" (low), "N" (normal)

    public LabResult() {
        this.timestamp = System.currentTimeMillis();
    }

    public static LabResult fromPayload(Map<String, Object> payload) {
        LabResult lab = new LabResult();

        lab.labType = (String) payload.get("lab_type");
        lab.value = getDoubleValue(payload, "value");
        lab.unit = (String) payload.get("unit");
        lab.referenceRangeLow = getDoubleValue(payload, "reference_low");
        lab.referenceRangeHigh = getDoubleValue(payload, "reference_high");

        // Determine if abnormal
        if (lab.value != null && lab.referenceRangeLow != null && lab.referenceRangeHigh != null) {
            if (lab.value < lab.referenceRangeLow) {
                lab.abnormal = true;
                lab.abnormalFlag = "L";
            } else if (lab.value > lab.referenceRangeHigh) {
                lab.abnormal = true;
                lab.abnormalFlag = "H";
            } else {
                lab.abnormal = false;
                lab.abnormalFlag = "N";
            }
        }

        return lab;
    }

    private static Double getDoubleValue(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    // Getters and setters
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    public String getLabCode() { return labCode; }
    public void setLabCode(String labCode) { this.labCode = labCode; }

    public String getLabType() { return labType; }
    public void setLabType(String labType) { this.labType = labType; }

    public Double getValue() { return value; }
    public void setValue(Double value) { this.value = value; }

    public String getUnit() { return unit; }
    public void setUnit(String unit) { this.unit = unit; }

    public Double getReferenceRangeLow() { return referenceRangeLow; }
    public void setReferenceRangeLow(Double referenceRangeLow) {
        this.referenceRangeLow = referenceRangeLow;
    }

    public Double getReferenceRangeHigh() { return referenceRangeHigh; }
    public void setReferenceRangeHigh(Double referenceRangeHigh) {
        this.referenceRangeHigh = referenceRangeHigh;
    }

    public boolean isAbnormal() { return abnormal; }
    public void setAbnormal(boolean abnormal) { this.abnormal = abnormal; }

    public String getAbnormalFlag() { return abnormalFlag; }
    public void setAbnormalFlag(String abnormalFlag) { this.abnormalFlag = abnormalFlag; }

    @Override
    public String toString() {
        return "LabResult{" +
                "type='" + labType + '\'' +
                ", value=" + value +
                " " + unit +
                ", flag=" + abnormalFlag +
                '}';
    }
}
