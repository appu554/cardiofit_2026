package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Laboratory values container for clinical decision support.
 *
 * This class holds key lab results used by Module 2 for:
 * - Acute kidney injury detection (creatinine, BUN)
 * - Liver function monitoring (AST, ALT, bilirubin)
 * - Electrolyte imbalances (sodium, potassium, chloride)
 * - Hematology tracking (hemoglobin, WBC, platelets)
 * - Cardiac markers (troponin, BNP)
 */
public class LabValues implements Serializable {
    private static final long serialVersionUID = 1L;

    // ============================================================
    // RENAL FUNCTION
    // ============================================================

    @JsonProperty("creatinine")
    private Double creatinine; // mg/dL (normal: 0.6-1.2)

    @JsonProperty("bun")
    private Double bun; // Blood Urea Nitrogen, mg/dL (normal: 7-20)

    @JsonProperty("gfr")
    private Double gfr; // Glomerular Filtration Rate, mL/min/1.73m²

    // ============================================================
    // LIVER FUNCTION
    // ============================================================

    @JsonProperty("ast")
    private Double ast; // Aspartate Aminotransferase, U/L (normal: 10-40)

    @JsonProperty("alt")
    private Double alt; // Alanine Aminotransferase, U/L (normal: 7-56)

    @JsonProperty("bilirubin")
    private Double bilirubin; // Total bilirubin, mg/dL (normal: 0.1-1.2)

    @JsonProperty("albumin")
    private Double albumin; // g/dL (normal: 3.4-5.4)

    // ============================================================
    // ELECTROLYTES
    // ============================================================

    @JsonProperty("sodium")
    private Double sodium; // mEq/L (normal: 136-145)

    @JsonProperty("potassium")
    private Double potassium; // mEq/L (normal: 3.5-5.0)

    @JsonProperty("chloride")
    private Double chloride; // mEq/L (normal: 96-106)

    @JsonProperty("bicarbonate")
    private Double bicarbonate; // mEq/L (normal: 23-30)

    @JsonProperty("calcium")
    private Double calcium; // mg/dL (normal: 8.5-10.2)

    @JsonProperty("magnesium")
    private Double magnesium; // mg/dL (normal: 1.7-2.2)

    // ============================================================
    // HEMATOLOGY
    // ============================================================

    @JsonProperty("hemoglobin")
    private Double hemoglobin; // g/dL (normal: male 13.5-17.5, female 12-15.5)

    @JsonProperty("hematocrit")
    private Double hematocrit; // % (normal: male 38.8-50, female 34.9-44.5)

    @JsonProperty("wbc")
    private Double wbc; // White Blood Cell count, K/µL (normal: 4.5-11.0)

    @JsonProperty("platelets")
    private Double platelets; // K/µL (normal: 150-400)

    // ============================================================
    // CARDIAC MARKERS
    // ============================================================

    @JsonProperty("troponin")
    private Double troponin; // ng/mL (elevated: >0.04)

    @JsonProperty("bnp")
    private Double bnp; // Brain Natriuretic Peptide, pg/mL

    @JsonProperty("ckMb")
    private Double ckMb; // Creatine Kinase-MB, ng/mL

    // ============================================================
    // METABOLIC
    // ============================================================

    @JsonProperty("glucose")
    private Double glucose; // mg/dL (normal fasting: 70-100)

    @JsonProperty("lactate")
    private Double lactate; // mmol/L (normal: 0.5-2.2)

    @JsonProperty("hba1c")
    private Double hba1c; // % (normal: <5.7)

    // ============================================================
    // ADDITIONAL LABS (flexible storage)
    // ============================================================

    @JsonProperty("additionalLabs")
    private Map<String, Double> additionalLabs = new HashMap<>();

    // ============================================================
    // METADATA
    // ============================================================

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("isComplete")
    private boolean isComplete; // True if all critical labs are available

    // ============================================================
    // CONSTRUCTORS
    // ============================================================

    public LabValues() {
        this.timestamp = System.currentTimeMillis();
        this.isComplete = false;
    }

    /**
     * Create LabValues from LabHistory.
     * Extracts the most recent value for each lab type.
     */
    public static LabValues fromLabHistory(LabHistory labHistory) {
        LabValues values = new LabValues();

        if (labHistory == null || labHistory.isEmpty()) {
            return values;
        }

        // Process recent labs and extract values by type
        for (LabResult lab : labHistory.getAll()) {
            if (lab.getLabCode() == null || lab.getValue() == null) {
                continue; // Skip labs with null code or value
            }
            String labCode = lab.getLabCode().toLowerCase();
            Double value = lab.getValue();

            // Map lab codes to fields
            switch (labCode) {
                case "creatinine":
                case "crea":
                    values.creatinine = value;
                    break;
                case "bun":
                    values.bun = value;
                    break;
                case "gfr":
                    values.gfr = value;
                    break;
                case "ast":
                case "sgot":
                    values.ast = value;
                    break;
                case "alt":
                case "sgpt":
                    values.alt = value;
                    break;
                case "bilirubin":
                case "tbil":
                    values.bilirubin = value;
                    break;
                case "albumin":
                case "alb":
                    values.albumin = value;
                    break;
                case "sodium":
                case "na":
                    values.sodium = value;
                    break;
                case "potassium":
                case "k":
                    values.potassium = value;
                    break;
                case "chloride":
                case "cl":
                    values.chloride = value;
                    break;
                case "bicarbonate":
                case "hco3":
                    values.bicarbonate = value;
                    break;
                case "calcium":
                case "ca":
                    values.calcium = value;
                    break;
                case "magnesium":
                case "mg":
                    values.magnesium = value;
                    break;
                case "hemoglobin":
                case "hgb":
                case "hb":
                    values.hemoglobin = value;
                    break;
                case "hematocrit":
                case "hct":
                    values.hematocrit = value;
                    break;
                case "wbc":
                    values.wbc = value;
                    break;
                case "platelets":
                case "plt":
                    values.platelets = value;
                    break;
                case "troponin":
                case "tn":
                    values.troponin = value;
                    break;
                case "bnp":
                    values.bnp = value;
                    break;
                case "ckmb":
                case "ck-mb":
                    values.ckMb = value;
                    break;
                case "glucose":
                case "glu":
                    values.glucose = value;
                    break;
                case "lactate":
                case "lac":
                    values.lactate = value;
                    break;
                case "hba1c":
                case "a1c":
                    values.hba1c = value;
                    break;
                default:
                    // Store unknown labs in additional labs map
                    values.additionalLabs.put(labCode, value);
                    break;
            }
        }

        // Check if critical labs are complete
        values.isComplete = values.hasMinimumCriticalLabs();

        return values;
    }

    /**
     * Check if minimum critical labs are available for clinical decision support.
     */
    private boolean hasMinimumCriticalLabs() {
        // At minimum, need electrolytes and CBC
        return sodium != null && potassium != null &&
               hemoglobin != null && wbc != null;
    }

    // ============================================================
    // CLINICAL ASSESSMENT METHODS
    // ============================================================

    /**
     * Check for acute kidney injury based on creatinine.
     */
    public boolean hasAcuteKidneyInjury(Double baselineCreatinine) {
        if (creatinine == null || baselineCreatinine == null) {
            return false;
        }

        // AKI Stage 1: Creatinine increase ≥0.3 mg/dL or ≥1.5x baseline
        double increase = creatinine - baselineCreatinine;
        double ratio = creatinine / baselineCreatinine;

        return increase >= 0.3 || ratio >= 1.5;
    }

    /**
     * Check for electrolyte imbalance.
     */
    public boolean hasElectrolyteImbalance() {
        boolean hyponatremia = sodium != null && sodium < 135;
        boolean hypernatremia = sodium != null && sodium > 145;
        boolean hypokalemia = potassium != null && potassium < 3.5;
        boolean hyperkalemia = potassium != null && potassium > 5.5;

        return hyponatremia || hypernatremia || hypokalemia || hyperkalemia;
    }

    /**
     * Check for elevated cardiac markers.
     */
    public boolean hasElevatedCardiacMarkers() {
        boolean elevatedTroponin = troponin != null && troponin > 0.04;
        boolean elevatedBNP = bnp != null && bnp > 100;

        return elevatedTroponin || elevatedBNP;
    }

    // ============================================================
    // GETTERS AND SETTERS
    // ============================================================

    public Double getCreatinine() { return creatinine; }
    public void setCreatinine(Double creatinine) { this.creatinine = creatinine; }

    public Double getBun() { return bun; }
    public void setBun(Double bun) { this.bun = bun; }

    public Double getGfr() { return gfr; }
    public void setGfr(Double gfr) { this.gfr = gfr; }

    public Double getAst() { return ast; }
    public void setAst(Double ast) { this.ast = ast; }

    public Double getAlt() { return alt; }
    public void setAlt(Double alt) { this.alt = alt; }

    public Double getBilirubin() { return bilirubin; }
    public void setBilirubin(Double bilirubin) { this.bilirubin = bilirubin; }

    public Double getAlbumin() { return albumin; }
    public void setAlbumin(Double albumin) { this.albumin = albumin; }

    public Double getSodium() { return sodium; }
    public void setSodium(Double sodium) { this.sodium = sodium; }

    public Double getPotassium() { return potassium; }
    public void setPotassium(Double potassium) { this.potassium = potassium; }

    public Double getChloride() { return chloride; }
    public void setChloride(Double chloride) { this.chloride = chloride; }

    public Double getBicarbonate() { return bicarbonate; }
    public void setBicarbonate(Double bicarbonate) { this.bicarbonate = bicarbonate; }

    public Double getCalcium() { return calcium; }
    public void setCalcium(Double calcium) { this.calcium = calcium; }

    public Double getMagnesium() { return magnesium; }
    public void setMagnesium(Double magnesium) { this.magnesium = magnesium; }

    public Double getHemoglobin() { return hemoglobin; }
    public void setHemoglobin(Double hemoglobin) { this.hemoglobin = hemoglobin; }

    public Double getHematocrit() { return hematocrit; }
    public void setHematocrit(Double hematocrit) { this.hematocrit = hematocrit; }

    public Double getWbc() { return wbc; }
    public void setWbc(Double wbc) { this.wbc = wbc; }

    /** Alias for getWbc() - White Blood Cell Count */
    public Double getWbcCount() { return wbc; }

    public Double getPlatelets() { return platelets; }
    public void setPlatelets(Double platelets) { this.platelets = platelets; }

    public Double getTroponin() { return troponin; }
    public void setTroponin(Double troponin) { this.troponin = troponin; }

    public Double getBnp() { return bnp; }
    public void setBnp(Double bnp) { this.bnp = bnp; }

    public Double getCkMb() { return ckMb; }
    public void setCkMb(Double ckMb) { this.ckMb = ckMb; }

    public Double getGlucose() { return glucose; }
    public void setGlucose(Double glucose) { this.glucose = glucose; }

    public Double getLactate() { return lactate; }
    public void setLactate(Double lactate) { this.lactate = lactate; }

    public Double getHba1c() { return hba1c; }
    public void setHba1c(Double hba1c) { this.hba1c = hba1c; }

    public Map<String, Double> getAdditionalLabs() { return additionalLabs; }
    public void setAdditionalLabs(Map<String, Double> additionalLabs) {
        this.additionalLabs = additionalLabs;
    }

    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    public boolean isComplete() { return isComplete; }
    public void setComplete(boolean complete) { isComplete = complete; }

    @Override
    public String toString() {
        return "LabValues{" +
                "creatinine=" + creatinine +
                ", sodium=" + sodium +
                ", potassium=" + potassium +
                ", hemoglobin=" + hemoglobin +
                ", wbc=" + wbc +
                ", isComplete=" + isComplete +
                '}';
    }
}
