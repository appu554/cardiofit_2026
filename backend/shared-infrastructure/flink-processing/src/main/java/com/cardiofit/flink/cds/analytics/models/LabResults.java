package com.cardiofit.flink.cds.analytics.models;

import java.io.Serializable;
import java.time.LocalDateTime;

/**
 * Phase 8 - Laboratory results model for risk calculations
 *
 * Aggregates common lab values needed for predictive risk scoring.
 * Maps to FHIR Observation resources with LOINC codes.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class LabResults implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private LocalDateTime collectionTime;

    // Hematology
    private Double hemoglobin;        // g/dL (LOINC: 718-7)
    private Double hematocrit;        // % (LOINC: 4544-3)
    private Double wbc;               // K/uL (LOINC: 6690-2)
    private Double platelets;         // K/uL (LOINC: 777-3)

    // Chemistry
    private Double sodium;            // mEq/L (LOINC: 2951-2)
    private Double potassium;         // mEq/L (LOINC: 2823-3)
    private Double chloride;          // mEq/L (LOINC: 2075-0)
    private Double bicarbonate;       // mEq/L (LOINC: 1963-8)
    private Double glucose;           // mg/dL (LOINC: 2339-0)
    private Double calcium;           // mg/dL (LOINC: 17861-6)

    // Renal Function
    private Double creatinine;        // mg/dL (LOINC: 2160-0)
    private Double bun;               // mg/dL (LOINC: 3094-0)
    private Double gfr;               // mL/min/1.73m² (LOINC: 33914-3)

    // Liver Function
    private Double bilirubin;         // mg/dL (LOINC: 1975-2)
    private Double alt;               // U/L (LOINC: 1742-6)
    private Double ast;               // U/L (LOINC: 1920-8)
    private Double albumin;           // g/dL (LOINC: 1751-7)
    private Double alkalinePhosphatase; // U/L (LOINC: 6768-6)

    // Cardiac Markers
    private Double troponin;          // ng/mL (LOINC: 42719-5)
    private Double bnp;               // pg/mL (LOINC: 30934-4)
    private Double ntproBNP;          // pg/mL (LOINC: 33762-6)

    // Coagulation
    private Double inr;               // ratio (LOINC: 6301-6)
    private Double ptt;               // seconds (LOINC: 3173-2)
    private Double aptt;              // seconds (LOINC: 14979-9)

    // Blood Gases
    private Double arterialPH;        // pH units (LOINC: 2744-1)
    private Double paCO2;             // mmHg (LOINC: 2019-8)
    private Double paO2;              // mmHg (LOINC: 2703-7)
    private Double lactate;           // mmol/L (LOINC: 32623-1)

    // Metabolic
    private Double hbA1c;             // % (LOINC: 4548-4)
    private Double magnesium;         // mEq/L (LOINC: 2601-3)
    private Double phosphate;         // mg/dL (LOINC: 2777-1)

    // Constructors
    public LabResults() {
        this.collectionTime = LocalDateTime.now();
    }

    public LabResults(String patientId) {
        this();
        this.patientId = patientId;
    }

    // Getters and Setters
    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public LocalDateTime getCollectionTime() {
        return collectionTime;
    }

    public void setCollectionTime(LocalDateTime collectionTime) {
        this.collectionTime = collectionTime;
    }

    public Double getHemoglobin() {
        return hemoglobin;
    }

    public void setHemoglobin(Double hemoglobin) {
        this.hemoglobin = hemoglobin;
    }

    public Double getHematocrit() {
        return hematocrit;
    }

    public void setHematocrit(Double hematocrit) {
        this.hematocrit = hematocrit;
    }

    public Double getWBC() {
        return wbc;
    }

    public void setWBC(Double wbc) {
        this.wbc = wbc;
    }

    public Double getPlatelets() {
        return platelets;
    }

    public void setPlatelets(Double platelets) {
        this.platelets = platelets;
    }

    public Double getSodium() {
        return sodium;
    }

    public void setSodium(Double sodium) {
        this.sodium = sodium;
    }

    public Double getPotassium() {
        return potassium;
    }

    public void setPotassium(Double potassium) {
        this.potassium = potassium;
    }

    public Double getChloride() {
        return chloride;
    }

    public void setChloride(Double chloride) {
        this.chloride = chloride;
    }

    public Double getBicarbonate() {
        return bicarbonate;
    }

    public void setBicarbonate(Double bicarbonate) {
        this.bicarbonate = bicarbonate;
    }

    public Double getGlucose() {
        return glucose;
    }

    public void setGlucose(Double glucose) {
        this.glucose = glucose;
    }

    public Double getCalcium() {
        return calcium;
    }

    public void setCalcium(Double calcium) {
        this.calcium = calcium;
    }

    public Double getCreatinine() {
        return creatinine;
    }

    public void setCreatinine(Double creatinine) {
        this.creatinine = creatinine;
    }

    public Double getBUN() {
        return bun;
    }

    public void setBUN(Double bun) {
        this.bun = bun;
    }

    public Double getGFR() {
        return gfr;
    }

    public void setGFR(Double gfr) {
        this.gfr = gfr;
    }

    public Double getBilirubin() {
        return bilirubin;
    }

    public void setBilirubin(Double bilirubin) {
        this.bilirubin = bilirubin;
    }

    public Double getALT() {
        return alt;
    }

    public void setALT(Double alt) {
        this.alt = alt;
    }

    public Double getAST() {
        return ast;
    }

    public void setAST(Double ast) {
        this.ast = ast;
    }

    public Double getAlbumin() {
        return albumin;
    }

    public void setAlbumin(Double albumin) {
        this.albumin = albumin;
    }

    public Double getAlkalinePhosphatase() {
        return alkalinePhosphatase;
    }

    public void setAlkalinePhosphatase(Double alkalinePhosphatase) {
        this.alkalinePhosphatase = alkalinePhosphatase;
    }

    public Double getTroponin() {
        return troponin;
    }

    public void setTroponin(Double troponin) {
        this.troponin = troponin;
    }

    public Double getBNP() {
        return bnp;
    }

    public void setBNP(Double bnp) {
        this.bnp = bnp;
    }

    public Double getNtproBNP() {
        return ntproBNP;
    }

    public void setNtproBNP(Double ntproBNP) {
        this.ntproBNP = ntproBNP;
    }

    public Double getINR() {
        return inr;
    }

    public void setINR(Double inr) {
        this.inr = inr;
    }

    public Double getPTT() {
        return ptt;
    }

    public void setPTT(Double ptt) {
        this.ptt = ptt;
    }

    public Double getAPTT() {
        return aptt;
    }

    public void setAPTT(Double aptt) {
        this.aptt = aptt;
    }

    public Double getArterialPH() {
        return arterialPH;
    }

    public void setArterialPH(Double arterialPH) {
        this.arterialPH = arterialPH;
    }

    public Double getPaCO2() {
        return paCO2;
    }

    public void setPaCO2(Double paCO2) {
        this.paCO2 = paCO2;
    }

    public Double getPaO2() {
        return paO2;
    }

    public void setPaO2(Double paO2) {
        this.paO2 = paO2;
    }

    public Double getLactate() {
        return lactate;
    }

    public void setLactate(Double lactate) {
        this.lactate = lactate;
    }

    public Double getHbA1c() {
        return hbA1c;
    }

    public void setHbA1c(Double hbA1c) {
        this.hbA1c = hbA1c;
    }

    public Double getMagnesium() {
        return magnesium;
    }

    public void setMagnesium(Double magnesium) {
        this.magnesium = magnesium;
    }

    public Double getPhosphate() {
        return phosphate;
    }

    public void setPhosphate(Double phosphate) {
        this.phosphate = phosphate;
    }

    @Override
    public String toString() {
        return String.format("LabResults{patient='%s', collectionTime=%s, Hgb=%.1f, Cr=%.2f, Na=%.1f, K=%.1f}",
            patientId, collectionTime, hemoglobin, creatinine, sodium, potassium);
    }
}
