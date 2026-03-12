package com.cardiofit.flink.cds.analytics.models;

import java.io.Serializable;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

/**
 * Phase 8 - Patient context model for risk calculations
 *
 * Contains patient demographics, medical history, and current clinical context
 * needed for predictive analytics.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class PatientContext implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core Identification
    private String patientId;
    private String mrn;  // Medical Record Number

    // Demographics
    private LocalDate dateOfBirth;
    private String gender;          // "M", "F", "O", "U"
    private String ethnicity;
    private String race;

    // Medical History
    private List<String> activeConditions;      // ICD-10 codes or descriptions
    private List<String> pastMedicalHistory;
    private List<String> surgicalHistory;
    private List<String> familyHistory;
    private List<String> allergies;

    // Current Medications
    private List<String> currentMedications;
    private List<String> recentMedications;     // Last 30 days

    // Admission Context
    private String admissionType;               // "ELECTIVE", "URGENT", "EMERGENT"
    private String admissionDiagnosis;
    private LocalDate admissionDate;
    private String admittingService;            // "CARDIOLOGY", "MEDICINE", "SURGERY", etc.
    private String currentLocation;             // "ICU", "STEP_DOWN", "FLOOR"

    // Clinical Status
    private String acuityLevel;                 // "CRITICAL", "ACUTE", "STABLE"
    private boolean isICUPatient;
    private boolean isMechanicallyVentilated;
    private boolean hasActiveSepsis;
    private boolean hasAcuteKidneyInjury;

    // Social History
    private boolean isSmoker;
    private boolean hasAlcoholAbuse;
    private boolean hasSubstanceAbuse;

    // Functional Status
    private String functionalStatus;            // "INDEPENDENT", "ASSISTED", "DEPENDENT"
    private String codeStatus;                  // "FULL_CODE", "DNR", "DNI", "DNR/DNI"

    // Constructors
    public PatientContext() {
        this.activeConditions = new ArrayList<>();
        this.pastMedicalHistory = new ArrayList<>();
        this.surgicalHistory = new ArrayList<>();
        this.familyHistory = new ArrayList<>();
        this.allergies = new ArrayList<>();
        this.currentMedications = new ArrayList<>();
        this.recentMedications = new ArrayList<>();
    }

    public PatientContext(String patientId) {
        this();
        this.patientId = patientId;
    }

    /**
     * Calculate patient age in years
     */
    public int getAgeYears() {
        if (dateOfBirth == null) return 0;
        return LocalDate.now().getYear() - dateOfBirth.getYear();
    }

    /**
     * Check if patient has a specific condition
     */
    public boolean hasCondition(String condition) {
        if (activeConditions == null) return false;
        return activeConditions.stream()
            .anyMatch(c -> c.toLowerCase().contains(condition.toLowerCase()));
    }

    /**
     * Check if patient is on a specific medication
     */
    public boolean isOnMedication(String medication) {
        if (currentMedications == null) return false;
        return currentMedications.stream()
            .anyMatch(m -> m.toLowerCase().contains(medication.toLowerCase()));
    }

    // Getters and Setters
    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getMrn() {
        return mrn;
    }

    public void setMrn(String mrn) {
        this.mrn = mrn;
    }

    public LocalDate getDateOfBirth() {
        return dateOfBirth;
    }

    public void setDateOfBirth(LocalDate dateOfBirth) {
        this.dateOfBirth = dateOfBirth;
    }

    public String getGender() {
        return gender;
    }

    public void setGender(String gender) {
        this.gender = gender;
    }

    public String getEthnicity() {
        return ethnicity;
    }

    public void setEthnicity(String ethnicity) {
        this.ethnicity = ethnicity;
    }

    public String getRace() {
        return race;
    }

    public void setRace(String race) {
        this.race = race;
    }

    public List<String> getActiveConditions() {
        return activeConditions;
    }

    public void setActiveConditions(List<String> activeConditions) {
        this.activeConditions = activeConditions;
    }

    public void addCondition(String condition) {
        if (this.activeConditions == null) {
            this.activeConditions = new ArrayList<>();
        }
        this.activeConditions.add(condition);
    }

    public List<String> getPastMedicalHistory() {
        return pastMedicalHistory;
    }

    public void setPastMedicalHistory(List<String> pastMedicalHistory) {
        this.pastMedicalHistory = pastMedicalHistory;
    }

    public List<String> getSurgicalHistory() {
        return surgicalHistory;
    }

    public void setSurgicalHistory(List<String> surgicalHistory) {
        this.surgicalHistory = surgicalHistory;
    }

    public List<String> getFamilyHistory() {
        return familyHistory;
    }

    public void setFamilyHistory(List<String> familyHistory) {
        this.familyHistory = familyHistory;
    }

    public List<String> getAllergies() {
        return allergies;
    }

    public void setAllergies(List<String> allergies) {
        this.allergies = allergies;
    }

    public List<String> getCurrentMedications() {
        return currentMedications;
    }

    public void setCurrentMedications(List<String> currentMedications) {
        this.currentMedications = currentMedications;
    }

    public void addMedication(String medication) {
        if (this.currentMedications == null) {
            this.currentMedications = new ArrayList<>();
        }
        this.currentMedications.add(medication);
    }

    public List<String> getRecentMedications() {
        return recentMedications;
    }

    public void setRecentMedications(List<String> recentMedications) {
        this.recentMedications = recentMedications;
    }

    public String getAdmissionType() {
        return admissionType;
    }

    public void setAdmissionType(String admissionType) {
        this.admissionType = admissionType;
    }

    public String getAdmissionDiagnosis() {
        return admissionDiagnosis;
    }

    public void setAdmissionDiagnosis(String admissionDiagnosis) {
        this.admissionDiagnosis = admissionDiagnosis;
    }

    public LocalDate getAdmissionDate() {
        return admissionDate;
    }

    public void setAdmissionDate(LocalDate admissionDate) {
        this.admissionDate = admissionDate;
    }

    public String getAdmittingService() {
        return admittingService;
    }

    public void setAdmittingService(String admittingService) {
        this.admittingService = admittingService;
    }

    public String getCurrentLocation() {
        return currentLocation;
    }

    public void setCurrentLocation(String currentLocation) {
        this.currentLocation = currentLocation;
    }

    public String getAcuityLevel() {
        return acuityLevel;
    }

    public void setAcuityLevel(String acuityLevel) {
        this.acuityLevel = acuityLevel;
    }

    public boolean isICUPatient() {
        return isICUPatient;
    }

    public void setICUPatient(boolean ICUPatient) {
        isICUPatient = ICUPatient;
    }

    public boolean isMechanicallyVentilated() {
        return isMechanicallyVentilated;
    }

    public void setMechanicallyVentilated(boolean mechanicallyVentilated) {
        isMechanicallyVentilated = mechanicallyVentilated;
    }

    public boolean isHasActiveSepsis() {
        return hasActiveSepsis;
    }

    public void setHasActiveSepsis(boolean hasActiveSepsis) {
        this.hasActiveSepsis = hasActiveSepsis;
    }

    public boolean isHasAcuteKidneyInjury() {
        return hasAcuteKidneyInjury;
    }

    public void setHasAcuteKidneyInjury(boolean hasAcuteKidneyInjury) {
        this.hasAcuteKidneyInjury = hasAcuteKidneyInjury;
    }

    public boolean isSmoker() {
        return isSmoker;
    }

    public void setSmoker(boolean smoker) {
        isSmoker = smoker;
    }

    public boolean isHasAlcoholAbuse() {
        return hasAlcoholAbuse;
    }

    public void setHasAlcoholAbuse(boolean hasAlcoholAbuse) {
        this.hasAlcoholAbuse = hasAlcoholAbuse;
    }

    public boolean isHasSubstanceAbuse() {
        return hasSubstanceAbuse;
    }

    public void setHasSubstanceAbuse(boolean hasSubstanceAbuse) {
        this.hasSubstanceAbuse = hasSubstanceAbuse;
    }

    public String getFunctionalStatus() {
        return functionalStatus;
    }

    public void setFunctionalStatus(String functionalStatus) {
        this.functionalStatus = functionalStatus;
    }

    public String getCodeStatus() {
        return codeStatus;
    }

    public void setCodeStatus(String codeStatus) {
        this.codeStatus = codeStatus;
    }

    @Override
    public String toString() {
        return String.format("PatientContext{id='%s', age=%d, gender='%s', ICU=%s, conditions=%d}",
            patientId, getAgeYears(), gender, isICUPatient, activeConditions.size());
    }
}
