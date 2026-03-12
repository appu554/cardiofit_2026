package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Medication update payload for GenericEvent wrapper.
 *
 * Uses RxNorm codes for standardized medication identification.
 * Supports medication administration, dose changes, and discontinuation events.
 *
 * Common RxNorm Codes for Cardiofit Platform:
 * - Antihypertensives:
 *   - 314076: Lisinopril (ACE Inhibitor)
 *   - 83367: Losartan (ARB)
 *   - 6918: Metoprolol (Beta Blocker)
 *   - 17767: Amlodipine (Calcium Channel Blocker)
 *   - 83367: Telmisartan (ARB)
 *
 * - Anticoagulants:
 *   - 11289: Warfarin
 *   - 1114195: Apixaban
 *   - 1037045: Dabigatran
 *
 * - Diuretics:
 *   - 4603: Furosemide (Loop Diuretic)
 *   - 8163: Hydrochlorothiazide (Thiazide)
 *   - 9997: Spironolactone (K-sparing)
 *
 * - Antidiabetic:
 *   - 6809: Metformin
 *   - 274783: Insulin Glargine
 *
 * - Cardiac:
 *   - 3616: Digoxin
 *   - 1191: Amiodarone
 */
public class MedicationPayload implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * RxNorm code for standardized medication identification
     * Example: "83367" for Telmisartan
     */
    @JsonProperty("rxNormCode")
    private String rxNormCode;

    /**
     * Human-readable medication name
     * Example: "Telmisartan 40 mg Tablet"
     */
    @JsonProperty("medicationName")
    private String medicationName;

    /**
     * Generic (non-proprietary) name
     * Example: "Telmisartan" (generic for "Micardis")
     */
    @JsonProperty("genericName")
    private String genericName;

    /**
     * Brand name if applicable
     */
    @JsonProperty("brandName")
    private String brandName;

    /**
     * Therapeutic class for interaction checking
     * Examples: "ACE_INHIBITOR", "ARB", "BETA_BLOCKER", "CCB", "DIURETIC_LOOP", "DIURETIC_K_SPARING"
     */
    @JsonProperty("therapeuticClass")
    private String therapeuticClass;

    /**
     * Drug class categories (can be multiple)
     */
    @JsonProperty("drugClasses")
    private List<String> drugClasses;

    /**
     * Dosage information
     */
    @JsonProperty("dose")
    private Double dose;

    @JsonProperty("doseUnit")
    private String doseUnit; // "mg", "mcg", "units", "mL"

    @JsonProperty("route")
    private String route; // "oral", "IV", "IM", "SC", "topical"

    @JsonProperty("frequency")
    private String frequency; // "daily", "BID", "TID", "QID", "Q4H", "PRN"

    /**
     * Administration details
     */
    @JsonProperty("administrationTime")
    private Long administrationTime;

    @JsonProperty("administeredBy")
    private String administeredBy; // Nurse ID or automated system

    @JsonProperty("administrationStatus")
    private String administrationStatus; // "administered", "scheduled", "held", "refused", "discontinued"

    /**
     * Medication lifecycle
     */
    @JsonProperty("orderTime")
    private Long orderTime;

    @JsonProperty("startTime")
    private Long startTime;

    @JsonProperty("stopTime")
    private Long stopTime;

    @JsonProperty("orderingProvider")
    private String orderingProvider;

    /**
     * Clinical context
     */
    @JsonProperty("indication")
    private String indication; // Why medication is prescribed

    @JsonProperty("priority")
    private String priority; // "stat", "urgent", "routine"

    /**
     * Safety checks
     */
    @JsonProperty("allergiesChecked")
    private Boolean allergiesChecked;

    @JsonProperty("interactionsChecked")
    private Boolean interactionsChecked;

    @JsonProperty("renalAdjusted")
    private Boolean renalAdjusted; // Dose adjusted for kidney function

    /**
     * Pharmacy system metadata
     */
    @JsonProperty("pharmacySystemId")
    private String pharmacySystemId; // "epic_pharmacy", "pyxis", "omnicell"

    @JsonProperty("orderId")
    private String orderId;

    @JsonProperty("dispensedQuantity")
    private Double dispensedQuantity;

    /**
     * Event type for medication updates
     */
    @JsonProperty("eventType")
    private String eventType; // "new_order", "dose_change", "administration", "discontinuation"

    public MedicationPayload() {
        this.drugClasses = new ArrayList<>();
        this.administrationTime = System.currentTimeMillis();
        this.administrationStatus = "scheduled";
    }

    /**
     * Constructor with essential fields
     */
    public MedicationPayload(String rxNormCode, String medicationName, Double dose, String doseUnit) {
        this();
        this.rxNormCode = rxNormCode;
        this.medicationName = medicationName;
        this.dose = dose;
        this.doseUnit = doseUnit;
    }

    // Getters and setters

    public String getRxNormCode() {
        return rxNormCode;
    }

    public void setRxNormCode(String rxNormCode) {
        this.rxNormCode = rxNormCode;
    }

    public String getMedicationName() {
        return medicationName;
    }

    public void setMedicationName(String medicationName) {
        this.medicationName = medicationName;
    }

    public String getGenericName() {
        return genericName;
    }

    public void setGenericName(String genericName) {
        this.genericName = genericName;
    }

    public String getBrandName() {
        return brandName;
    }

    public void setBrandName(String brandName) {
        this.brandName = brandName;
    }

    public String getTherapeuticClass() {
        return therapeuticClass;
    }

    public void setTherapeuticClass(String therapeuticClass) {
        this.therapeuticClass = therapeuticClass;
    }

    public List<String> getDrugClasses() {
        return drugClasses;
    }

    public void setDrugClasses(List<String> drugClasses) {
        this.drugClasses = drugClasses;
    }

    public Double getDose() {
        return dose;
    }

    public void setDose(Double dose) {
        this.dose = dose;
    }

    public String getDoseUnit() {
        return doseUnit;
    }

    public void setDoseUnit(String doseUnit) {
        this.doseUnit = doseUnit;
    }

    public String getRoute() {
        return route;
    }

    public void setRoute(String route) {
        this.route = route;
    }

    public String getFrequency() {
        return frequency;
    }

    public void setFrequency(String frequency) {
        this.frequency = frequency;
    }

    public Long getAdministrationTime() {
        return administrationTime;
    }

    public void setAdministrationTime(Long administrationTime) {
        this.administrationTime = administrationTime;
    }

    public String getAdministeredBy() {
        return administeredBy;
    }

    public void setAdministeredBy(String administeredBy) {
        this.administeredBy = administeredBy;
    }

    public String getAdministrationStatus() {
        return administrationStatus;
    }

    public void setAdministrationStatus(String administrationStatus) {
        this.administrationStatus = administrationStatus;
    }

    public Long getOrderTime() {
        return orderTime;
    }

    public void setOrderTime(Long orderTime) {
        this.orderTime = orderTime;
    }

    public Long getStartTime() {
        return startTime;
    }

    public void setStartTime(Long startTime) {
        this.startTime = startTime;
    }

    public Long getStopTime() {
        return stopTime;
    }

    public void setStopTime(Long stopTime) {
        this.stopTime = stopTime;
    }

    public String getOrderingProvider() {
        return orderingProvider;
    }

    public void setOrderingProvider(String orderingProvider) {
        this.orderingProvider = orderingProvider;
    }

    public String getIndication() {
        return indication;
    }

    public void setIndication(String indication) {
        this.indication = indication;
    }

    public String getPriority() {
        return priority;
    }

    public void setPriority(String priority) {
        this.priority = priority;
    }

    public Boolean getAllergiesChecked() {
        return allergiesChecked;
    }

    public void setAllergiesChecked(Boolean allergiesChecked) {
        this.allergiesChecked = allergiesChecked;
    }

    public Boolean getInteractionsChecked() {
        return interactionsChecked;
    }

    public void setInteractionsChecked(Boolean interactionsChecked) {
        this.interactionsChecked = interactionsChecked;
    }

    public Boolean getRenalAdjusted() {
        return renalAdjusted;
    }

    public void setRenalAdjusted(Boolean renalAdjusted) {
        this.renalAdjusted = renalAdjusted;
    }

    public String getPharmacySystemId() {
        return pharmacySystemId;
    }

    public void setPharmacySystemId(String pharmacySystemId) {
        this.pharmacySystemId = pharmacySystemId;
    }

    public String getOrderId() {
        return orderId;
    }

    public void setOrderId(String orderId) {
        this.orderId = orderId;
    }

    public Double getDispensedQuantity() {
        return dispensedQuantity;
    }

    public void setDispensedQuantity(Double dispensedQuantity) {
        this.dispensedQuantity = dispensedQuantity;
    }

    public String getEventType() {
        return eventType;
    }

    public void setEventType(String eventType) {
        this.eventType = eventType;
    }

    /**
     * Check if medication is an antihypertensive
     */
    public boolean isAntihypertensive() {
        if (therapeuticClass == null) return false;
        String tcLower = therapeuticClass.toLowerCase();
        return tcLower.contains("ace") || tcLower.contains("arb") ||
               tcLower.contains("beta") || tcLower.contains("ccb") ||
               tcLower.contains("diuretic");
    }

    /**
     * Check if medication is a diuretic
     */
    public boolean isDiuretic() {
        if (therapeuticClass == null) return false;
        return therapeuticClass.toLowerCase().contains("diuretic");
    }

    /**
     * Check if medication is potassium-sparing diuretic
     */
    public boolean isKSparingDiuretic() {
        if (therapeuticClass == null) return false;
        String tcLower = therapeuticClass.toLowerCase();
        return tcLower.contains("k_sparing") || tcLower.contains("potassium_sparing");
    }

    /**
     * Convert to legacy Medication format for backward compatibility
     */
    public Medication toMedication() {
        Medication med = new Medication();

        // Map name fields - use generic name if available, otherwise medication name
        med.setName(genericName != null ? genericName : medicationName);

        // Map RxNorm code
        med.setCode(rxNormCode);

        // Map dosage as string "dose unit" (e.g., "40 mg")
        if (dose != null && doseUnit != null) {
            med.setDosage(dose + " " + doseUnit);
        } else if (dose != null) {
            med.setDosage(String.valueOf(dose));
        }

        // Map route
        med.setRoute(route);

        // Map frequency
        med.setFrequency(frequency);

        // Map status based on administration status
        if ("discontinued".equals(eventType) || "held".equals(administrationStatus) ||
            "refused".equals(administrationStatus)) {
            med.setStatus("stopped");
        } else if ("administered".equals(administrationStatus)) {
            med.setStatus("active");
        } else {
            med.setStatus("active"); // Default to active
        }

        // Map start time
        if (startTime != null) {
            med.setStartDate(startTime);
        } else if (orderTime != null) {
            med.setStartDate(orderTime);
        } else if (administrationTime != null) {
            med.setStartDate(administrationTime);
        }

        // Map display name - use full medication name
        med.setDisplay(medicationName);

        return med;
    }

    @Override
    public String toString() {
        return "MedicationPayload{" +
                "name='" + medicationName + '\'' +
                ", dose=" + dose + " " + doseUnit +
                ", route=" + route +
                ", frequency=" + frequency +
                ", status=" + administrationStatus +
                ", class=" + therapeuticClass +
                '}';
    }
}
