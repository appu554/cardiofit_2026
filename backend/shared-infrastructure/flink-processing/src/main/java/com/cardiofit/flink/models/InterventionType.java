package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashSet;
import java.util.Set;

public enum InterventionType implements Serializable {
    MEDICATION_ADD(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_REMOVE(28, "ABSENCE_IN_LOGS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_DOSE_INCREASE(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_DOSE_DECREASE(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_SWITCH(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    LIFESTYLE_ACTIVITY(14, "ACTIVITY_DATA",
            ClinicalDomain.GLUCOSE, ClinicalDomain.BLOOD_PRESSURE),
    LIFESTYLE_SLEEP(14, "SLEEP_DATA",
            ClinicalDomain.GLUCOSE_VARIABILITY, ClinicalDomain.BLOOD_PRESSURE),
    NUTRITION_FOOD_CHANGE(14, "MEAL_LOGS",
            ClinicalDomain.GLUCOSE),
    NUTRITION_PORTION_CHANGE(14, "MEAL_LOGS",
            ClinicalDomain.GLUCOSE, ClinicalDomain.WEIGHT),
    NUTRITION_TIMING_CHANGE(14, "MEAL_TIMESTAMPS",
            ClinicalDomain.GLUCOSE),
    NUTRITION_SODIUM_REDUCTION(14, "SODIUM_ESTIMATE",
            ClinicalDomain.BLOOD_PRESSURE),
    MONITORING_CHANGE(14, "DATA_DENSITY",
            ClinicalDomain.DATA_QUALITY),
    REFERRAL(28, "REFERRAL_ATTENDANCE",
            ClinicalDomain.FROM_REFERRAL_TYPE);

    private final int defaultWindowDays;
    private final String adherenceSource;
    private final Set<ClinicalDomain> staticDomains;
    private final boolean domainFromDetail;

    InterventionType(int defaultWindowDays, String adherenceSource,
                     ClinicalDomain... domains) {
        this.defaultWindowDays = defaultWindowDays;
        this.adherenceSource = adherenceSource;
        if (domains.length == 1 && (domains[0] == ClinicalDomain.FROM_DRUG_CLASS
                || domains[0] == ClinicalDomain.FROM_REFERRAL_TYPE)) {
            this.staticDomains = Collections.emptySet();
            this.domainFromDetail = true;
        } else {
            this.staticDomains = Collections.unmodifiableSet(
                    new HashSet<>(Arrays.asList(domains)));
            this.domainFromDetail = false;
        }
    }

    public int getDefaultWindowDays() { return defaultWindowDays; }
    public String getAdherenceSource() { return adherenceSource; }
    public boolean isMedication() { return name().startsWith("MEDICATION_"); }
    public boolean isLifestyle() { return name().startsWith("LIFESTYLE_"); }
    public boolean isNutrition() { return name().startsWith("NUTRITION_"); }

    public Set<ClinicalDomain> getDomains(String drugClass) {
        if (!domainFromDetail) {
            return staticDomains;
        }
        if (drugClass == null) {
            return Collections.emptySet();
        }
        return ClinicalDomain.fromDrugClass(drugClass);
    }

    public enum ClinicalDomain {
        GLUCOSE, BLOOD_PRESSURE, RENAL, WEIGHT, LIPIDS,
        GLUCOSE_VARIABILITY, DATA_QUALITY,
        FROM_DRUG_CLASS, FROM_REFERRAL_TYPE;

        public static Set<ClinicalDomain> fromDrugClass(String drugClass) {
            if (drugClass == null) return Collections.emptySet();
            switch (drugClass.toUpperCase()) {
                case "SGLT2I": return setOf(GLUCOSE, BLOOD_PRESSURE, RENAL);
                case "METFORMIN": return setOf(GLUCOSE);
                case "SULFONYLUREA": return setOf(GLUCOSE);
                case "DPP4I": return setOf(GLUCOSE);
                case "GLP1_RA": return setOf(GLUCOSE, WEIGHT);
                case "INSULIN": return setOf(GLUCOSE);
                case "ACEI": case "ARB": return setOf(BLOOD_PRESSURE, RENAL);
                case "CCB": return setOf(BLOOD_PRESSURE);
                case "THIAZIDE": return setOf(BLOOD_PRESSURE);
                case "FINERENONE": return setOf(RENAL, BLOOD_PRESSURE);
                case "STATIN": return setOf(LIPIDS);
                default: return Collections.emptySet();
            }
        }

        private static Set<ClinicalDomain> setOf(ClinicalDomain... domains) {
            return Collections.unmodifiableSet(
                    new HashSet<>(Arrays.asList(domains)));
        }
    }
}
