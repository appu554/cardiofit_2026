package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonAlias;
import java.io.Serializable;

/**
 * Laboratory test result payload for GenericEvent wrapper.
 *
 * Uses LOINC (Logical Observation Identifiers Names and Codes) for standardized
 * lab test identification across different hospital systems.
 *
 * Common LOINC Codes for Cardiofit Platform:
 * - Cardiac Markers:
 *   - 10839-9: Troponin I (ng/mL), Critical: >0.04
 *   - 42757-5: BNP (pg/mL), Elevated: >400
 *   - 13969-1: CK-MB (U/L), Elevated: >25
 *
 * - Metabolic Panel:
 *   - 2160-0: Creatinine (mg/dL), Elevated: >1.3
 *   - 2823-3: Potassium (mEq/L), Range: 3.5-5.5
 *   - 2951-2: Sodium (mEq/L), Range: 135-145
 *   - 2345-7: Glucose (mg/dL), Range: 70-180
 *   - 2524-7: Lactate (mmol/L), Elevated: >2.0
 *
 * - Hematology:
 *   - 6690-2: WBC (K/uL), Range: 4-11
 *   - 777-3: Platelets (K/uL), Range: 150-400
 *   - 718-7: Hemoglobin (g/dL), Range: 12-16
 *
 * - Coagulation:
 *   - 6301-6: INR, Range: 0.8-1.2
 *   - 5902-2: PT (seconds), Range: 11-13.5
 */
public class LabPayload implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * LOINC code for standardized lab test identification
     * Example: "10839-9" for Troponin I
     */
    @JsonProperty("loincCode")
    @JsonAlias({"loinccode", "loinc_code", "LOINCCODE"})
    private String loincCode;

    /**
     * Human-readable lab test name
     * Example: "Troponin I, High Sensitivity"
     */
    @JsonProperty("labName")
    @JsonAlias({"labname", "lab_name", "LABNAME"})
    private String labName;

    /**
     * Lab test category for grouping
     * Examples: "CARDIAC", "METABOLIC", "HEMATOLOGY", "COAGULATION"
     */
    @JsonProperty("category")
    private String category;

    /**
     * Numeric result value
     */
    @JsonProperty("value")
    private Double value;

    /**
     * Unit of measurement
     * Examples: "ng/mL", "mg/dL", "mEq/L", "K/uL"
     */
    @JsonProperty("unit")
    private String unit;

    /**
     * Reference range - low bound
     */
    @JsonProperty("referenceRangeLow")
    @JsonAlias({"referencerangelow", "reference_range_low"})
    private Double referenceRangeLow;

    /**
     * Reference range - high bound
     */
    @JsonProperty("referenceRangeHigh")
    @JsonAlias({"referencerangehigh", "reference_range_high"})
    private Double referenceRangeHigh;

    /**
     * Abnormal flag from lab system
     * Values: "N" (normal), "L" (low), "H" (high), "LL" (critically low), "HH" (critically high)
     */
    @JsonProperty("abnormalFlag")
    @JsonAlias({"abnormalflag", "abnormal_flag"})
    private String abnormalFlag;

    /**
     * Calculated: is result outside reference range?
     */
    @JsonProperty("abnormal")
    private boolean abnormal;

    /**
     * Lab specimen details
     */
    @JsonProperty("specimenType")
    @JsonAlias({"specimentype", "specimen_type"})
    private String specimenType; // "blood", "serum", "plasma", "urine"

    @JsonProperty("collectionTime")
    @JsonAlias({"collectiontime", "collection_time"})
    private Long collectionTime;

    @JsonProperty("resultTime")
    @JsonAlias({"resulttime", "result_time"})
    private Long resultTime;

    /**
     * Lab system metadata
     */
    @JsonProperty("labSystemId")
    @JsonAlias({"labsystemid", "lab_system_id"})
    private String labSystemId; // "epic_labs", "cerner_millennium", "meditech_labs"

    @JsonProperty("orderId")
    @JsonAlias({"orderid", "order_id"})
    private String orderId;

    @JsonProperty("resultStatus")
    @JsonAlias({"resultstatus", "result_status"})
    private String resultStatus; // "final", "preliminary", "corrected", "canceled"

    /**
     * Clinical interpretation notes (optional)
     */
    @JsonProperty("interpretation")
    private String interpretation;

    @JsonProperty("performingLab")
    @JsonAlias({"performinglab", "performing_lab"})
    private String performingLab;

    public LabPayload() {
        this.resultTime = System.currentTimeMillis();
        this.abnormal = false;
        this.resultStatus = "final";
    }

    /**
     * Constructor with essential fields
     */
    public LabPayload(String loincCode, String labName, Double value, String unit) {
        this();
        this.loincCode = loincCode;
        this.labName = labName;
        this.value = value;
        this.unit = unit;
    }

    /**
     * Calculate abnormal status based on reference ranges
     */
    public void calculateAbnormalStatus() {
        if (value != null && referenceRangeLow != null && referenceRangeHigh != null) {
            if (value < referenceRangeLow) {
                abnormal = true;
                if (abnormalFlag == null) abnormalFlag = "L";
            } else if (value > referenceRangeHigh) {
                abnormal = true;
                if (abnormalFlag == null) abnormalFlag = "H";
            } else {
                abnormal = false;
                if (abnormalFlag == null) abnormalFlag = "N";
            }
        }
    }

    // Getters and setters

    public String getLoincCode() {
        return loincCode;
    }

    public void setLoincCode(String loincCode) {
        this.loincCode = loincCode;
    }

    public String getLabName() {
        return labName;
    }

    public void setLabName(String labName) {
        this.labName = labName;
    }

    public String getCategory() {
        return category;
    }

    public void setCategory(String category) {
        this.category = category;
    }

    public Double getValue() {
        return value;
    }

    public void setValue(Double value) {
        this.value = value;
    }

    public String getUnit() {
        return unit;
    }

    public void setUnit(String unit) {
        this.unit = unit;
    }

    public Double getReferenceRangeLow() {
        return referenceRangeLow;
    }

    public void setReferenceRangeLow(Double referenceRangeLow) {
        this.referenceRangeLow = referenceRangeLow;
    }

    public Double getReferenceRangeHigh() {
        return referenceRangeHigh;
    }

    public void setReferenceRangeHigh(Double referenceRangeHigh) {
        this.referenceRangeHigh = referenceRangeHigh;
    }

    public String getAbnormalFlag() {
        return abnormalFlag;
    }

    public void setAbnormalFlag(String abnormalFlag) {
        this.abnormalFlag = abnormalFlag;
    }

    public boolean isAbnormal() {
        return abnormal;
    }

    public void setAbnormal(boolean abnormal) {
        this.abnormal = abnormal;
    }

    public String getSpecimenType() {
        return specimenType;
    }

    public void setSpecimenType(String specimenType) {
        this.specimenType = specimenType;
    }

    public Long getCollectionTime() {
        return collectionTime;
    }

    public void setCollectionTime(Long collectionTime) {
        this.collectionTime = collectionTime;
    }

    public Long getResultTime() {
        return resultTime;
    }

    public void setResultTime(Long resultTime) {
        this.resultTime = resultTime;
    }

    public String getLabSystemId() {
        return labSystemId;
    }

    public void setLabSystemId(String labSystemId) {
        this.labSystemId = labSystemId;
    }

    public String getOrderId() {
        return orderId;
    }

    public void setOrderId(String orderId) {
        this.orderId = orderId;
    }

    public String getResultStatus() {
        return resultStatus;
    }

    public void setResultStatus(String resultStatus) {
        this.resultStatus = resultStatus;
    }

    public String getInterpretation() {
        return interpretation;
    }

    public void setInterpretation(String interpretation) {
        this.interpretation = interpretation;
    }

    public String getPerformingLab() {
        return performingLab;
    }

    public void setPerformingLab(String performingLab) {
        this.performingLab = performingLab;
    }

    /**
     * Convert to legacy LabResult format for backward compatibility
     */
    public LabResult toLabResult() {
        LabResult lab = new LabResult();
        lab.setLabCode(loincCode); // Store LOINC code for risk indicator lookups
        lab.setLabType(loincCode != null ? loincCode : labName);
        lab.setValue(value);
        lab.setUnit(unit);
        lab.setReferenceRangeLow(referenceRangeLow);
        lab.setReferenceRangeHigh(referenceRangeHigh);
        lab.setAbnormal(abnormal);
        lab.setAbnormalFlag(abnormalFlag);
        if (resultTime != null) {
            lab.setTimestamp(resultTime);
        }
        return lab;
    }

    @Override
    public String toString() {
        return "LabPayload{" +
                "lab='" + (labName != null ? labName : loincCode) + '\'' +
                ", value=" + value + " " + unit +
                ", flag=" + abnormalFlag +
                ", category=" + category +
                ", status=" + resultStatus +
                '}';
    }
}
