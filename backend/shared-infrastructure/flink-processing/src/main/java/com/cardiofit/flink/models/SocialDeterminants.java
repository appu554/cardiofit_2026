package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Social Determinants of Health (SDOH) data model.
 *
 * Represents non-clinical factors that influence health outcomes,
 * used in patient context enrichment and risk stratification.
 */
public class SocialDeterminants implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("housingStatus")
    private String housingStatus; // stable, unstable, homeless

    @JsonProperty("employmentStatus")
    private String employmentStatus; // employed, unemployed, retired, disabled

    @JsonProperty("educationLevel")
    private String educationLevel; // less_than_high_school, high_school, some_college, college, graduate

    @JsonProperty("incomeLevel")
    private String incomeLevel; // below_poverty, low_income, middle_income, high_income

    @JsonProperty("insurance Coverage")
    private String insuranceCoverage; // none, medicaid, medicare, private, other

    @JsonProperty("foodSecurity")
    private String foodSecurity; // secure, at_risk, insecure

    @JsonProperty("foodInsecurity")
    private Boolean foodInsecurity; // True if patient has food insecurity

    @JsonProperty("transportationAccess")
    private Boolean transportationAccess;

    @JsonProperty("socialSupport")
    private String socialSupport; // strong, moderate, weak, none

    @JsonProperty("primaryLanguage")
    private String primaryLanguage;

    @JsonProperty("interpreterNeeded")
    private Boolean interpreterNeeded;

    @JsonProperty("healthLiteracy")
    private String healthLiteracy; // high, moderate, low

    @JsonProperty("substanceUse")
    private String substanceUse; // none, alcohol, tobacco, drugs, multiple

    @JsonProperty("mentalHealthSupport")
    private Boolean mentalHealthSupport;

    @JsonProperty("caregiverAvailable")
    private Boolean caregiverAvailable;

    @JsonProperty("zipCode")
    private String zipCode;

    @JsonProperty("ruralUrban")
    private String ruralUrban; // urban, suburban, rural, frontier

    // Default constructor
    public SocialDeterminants() {}

    // Getters and Setters
    public String getHousingStatus() {
        return housingStatus;
    }

    public void setHousingStatus(String housingStatus) {
        this.housingStatus = housingStatus;
    }

    public String getEmploymentStatus() {
        return employmentStatus;
    }

    public void setEmploymentStatus(String employmentStatus) {
        this.employmentStatus = employmentStatus;
    }

    public String getEducationLevel() {
        return educationLevel;
    }

    public void setEducationLevel(String educationLevel) {
        this.educationLevel = educationLevel;
    }

    public String getIncomeLevel() {
        return incomeLevel;
    }

    public void setIncomeLevel(String incomeLevel) {
        this.incomeLevel = incomeLevel;
    }

    public String getInsuranceCoverage() {
        return insuranceCoverage;
    }

    public void setInsuranceCoverage(String insuranceCoverage) {
        this.insuranceCoverage = insuranceCoverage;
    }

    public String getFoodSecurity() {
        return foodSecurity;
    }

    public void setFoodSecurity(String foodSecurity) {
        this.foodSecurity = foodSecurity;
    }

    public Boolean getTransportationAccess() {
        return transportationAccess;
    }

    public void setTransportationAccess(Boolean transportationAccess) {
        this.transportationAccess = transportationAccess;
    }

    public String getSocialSupport() {
        return socialSupport;
    }

    public void setSocialSupport(String socialSupport) {
        this.socialSupport = socialSupport;
    }

    public String getPrimaryLanguage() {
        return primaryLanguage;
    }

    public void setPrimaryLanguage(String primaryLanguage) {
        this.primaryLanguage = primaryLanguage;
    }

    public Boolean getInterpreterNeeded() {
        return interpreterNeeded;
    }

    public void setInterpreterNeeded(Boolean interpreterNeeded) {
        this.interpreterNeeded = interpreterNeeded;
    }

    public String getHealthLiteracy() {
        return healthLiteracy;
    }

    public void setHealthLiteracy(String healthLiteracy) {
        this.healthLiteracy = healthLiteracy;
    }

    public String getSubstanceUse() {
        return substanceUse;
    }

    public void setSubstanceUse(String substanceUse) {
        this.substanceUse = substanceUse;
    }

    public Boolean getMentalHealthSupport() {
        return mentalHealthSupport;
    }

    public void setMentalHealthSupport(Boolean mentalHealthSupport) {
        this.mentalHealthSupport = mentalHealthSupport;
    }

    public Boolean getCaregiverAvailable() {
        return caregiverAvailable;
    }

    public void setCaregiverAvailable(Boolean caregiverAvailable) {
        this.caregiverAvailable = caregiverAvailable;
    }

    public String getZipCode() {
        return zipCode;
    }

    public void setZipCode(String zipCode) {
        this.zipCode = zipCode;
    }

    public String getRuralUrban() {
        return ruralUrban;
    }

    public void setRuralUrban(String ruralUrban) {
        this.ruralUrban = ruralUrban;
    }

    public Boolean getFoodInsecurity() {
        return foodInsecurity;
    }

    public void setFoodInsecurity(Boolean foodInsecurity) {
        this.foodInsecurity = foodInsecurity;
    }

    // Alias method for compatibility
    public Boolean isFoodInsecurity() {
        return foodInsecurity;
    }

    @Override
    public String toString() {
        return "SocialDeterminants{" +
                "housingStatus='" + housingStatus + '\'' +
                ", employmentStatus='" + employmentStatus + '\'' +
                ", insuranceCoverage='" + insuranceCoverage + '\'' +
                ", foodSecurity='" + foodSecurity + '\'' +
                ", healthLiteracy='" + healthLiteracy + '\'' +
                '}';
    }
}
