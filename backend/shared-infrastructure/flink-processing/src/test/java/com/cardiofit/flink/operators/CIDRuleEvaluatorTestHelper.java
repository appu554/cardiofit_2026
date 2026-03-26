package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ComorbidityAlert;
import java.util.*;

public class CIDRuleEvaluatorTestHelper {

    // ==================== HALT RULES ====================

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID01(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasRASi = meds.contains("ACEI") || meds.contains("ARB");
        boolean hasSGLT2i = meds.contains("SGLT2I");
        boolean hasDiuretic = meds.contains("THIAZIDE") || meds.contains("LOOP_DIURETIC");
        if (!hasRASi || !hasSGLT2i || !hasDiuretic) return alerts;

        Double currentEGFR = (Double) state.get("currentEGFR");
        Double prevEGFR = (Double) state.get("previousEGFR");
        boolean egfrDrop = currentEGFR != null && prevEGFR != null &&
                           prevEGFR > 0 && (prevEGFR - currentEGFR) / prevEGFR > 0.20;
        if (egfrDrop) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-01", "Triple Whammy AKI",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Triple whammy AKI risk. eGFR: %.0f (prev: %.0f).", currentEGFR, prevEGFR),
                "Pause SGLT2i and diuretic. Urgent eGFR + creatinine within 48 hours."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID02(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasRASi = meds.contains("ACEI") || meds.contains("ARB");
        boolean hasFinerenone = meds.contains("FINERENONE");
        Double kPlus = (Double) state.get("currentK");
        Double prevK = (Double) state.get("previousK");
        if (hasRASi && hasFinerenone && kPlus != null && kPlus > 5.3 &&
            prevK != null && kPlus > prevK) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-02", "Hyperkalemia Cascade",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Hyperkalemia cascade. K+ %.1f (rising from %.1f) on RASi + finerenone.", kPlus, prevK),
                "Hold finerenone immediately. Recheck K+ in 48-72 hours."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID03(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasHypoRisk = meds.contains("INSULIN") || meds.contains("SULFONYLUREA");
        boolean hasBetaBlocker = meds.contains("BETA_BLOCKER");
        Double glucose = (Double) state.get("currentGlucose");
        Boolean symptomPresent = (Boolean) state.getOrDefault("symptomReportPresent", false);
        if (hasHypoRisk && hasBetaBlocker && glucose != null && glucose < 60 && !symptomPresent) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-03", "Hypoglycemia Masking",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Hypoglycemia masking. Glucose %.0f on insulin/SU + beta-blocker.", glucose),
                "Check glucose immediately. Consider reducing beta-blocker or switching to cardioselective agent."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID04(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasSGLT2i = meds.contains("SGLT2I");
        Boolean nauseaSignal = (Boolean) state.getOrDefault("nauseaVomitingSignal", false);
        if (hasSGLT2i && nauseaSignal) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-04", "Euglycemic DKA Risk",
                ComorbidityAlert.AlertSeverity.HALT,
                "HALT: Euglycemic DKA risk. Patient on SGLT2i with nausea/vomiting. Glucose may appear normal.",
                "Hold SGLT2i immediately. Check blood ketones urgently."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID05(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasSGLT2i = meds.contains("SGLT2I");
        int antihtnCount = 0;
        for (String cls : new String[]{"ACEI", "ARB", "CCB", "THIAZIDE", "LOOP_DIURETIC", "BETA_BLOCKER", "MRA"}) {
            if (meds.contains(cls)) antihtnCount++;
        }
        Double sbp = (Double) state.get("currentSBP");
        if (hasSGLT2i && antihtnCount >= 3 && sbp != null && sbp < 95) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-05", "Severe Hypotension Risk",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Severe hypotension risk. SBP %.0f on %d antihypertensives + SGLT2i.", sbp, antihtnCount),
                "Hold SGLT2i and review antihypertensive doses. Target SBP >100 before resuming."));
        }
        return alerts;
    }

    // ==================== PAUSE RULES ====================

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID06(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasThiazide = meds.contains("THIAZIDE");
        boolean hasLoop = meds.contains("LOOP_DIURETIC");
        Double sodium = (Double) state.get("currentNa");
        Double prevSodium = (Double) state.get("previousNa");
        if (hasThiazide && hasLoop && sodium != null && sodium < 130 &&
            prevSodium != null && sodium < prevSodium) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-06", "Severe Hyponatremia",
                ComorbidityAlert.AlertSeverity.PAUSE,
                String.format("PAUSE: Hyponatremia risk. Na+ %.0f (falling from %.0f) on thiazide + loop diuretic.", sodium, prevSodium),
                "Review diuretic combination. Check Na+ in 48h. Consider stopping one diuretic."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID07(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasGLP1RA = meds.contains("GLP1RA");
        boolean hasSU = meds.contains("SULFONYLUREA");
        Double fbg = (Double) state.get("currentFBG");
        if (hasGLP1RA && hasSU && fbg != null && fbg < 70) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-07", "Recurrent Hypoglycemia",
                ComorbidityAlert.AlertSeverity.PAUSE,
                String.format("PAUSE: Recurrent hypo risk. FBG %.0f on GLP-1RA + sulfonylurea.", fbg),
                "Reduce SU dose by 50%. Monitor FBG daily for 1 week."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID08(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasThiazide = meds.contains("THIAZIDE");
        boolean hasSGLT2i = meds.contains("SGLT2I");
        Double sodium = (Double) state.get("currentNa");
        Double prevSodium = (Double) state.get("previousNa");
        boolean dehydrationSignal = sodium != null && prevSodium != null && sodium > 145 && sodium > prevSodium;
        if (hasThiazide && hasSGLT2i && dehydrationSignal) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-08", "Volume Depletion Risk",
                ComorbidityAlert.AlertSeverity.PAUSE,
                "PAUSE: Volume depletion risk. Thiazide + SGLT2i with rising sodium (dehydration marker).",
                "Advise increased fluid intake. Consider holding thiazide in hot weather."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID09(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasBeta = meds.contains("BETA_BLOCKER");
        boolean hasGLP1RA = meds.contains("GLP1RA");
        if (hasBeta && hasGLP1RA) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-09", "Heart Rate Masking",
                ComorbidityAlert.AlertSeverity.PAUSE,
                "PAUSE: Heart rate masking. Beta-blocker + GLP-1RA — tachycardia response blunted.",
                "Monitor resting heart rate. Consider heart rate-neutral alternatives if symptomatic."));
        }
        return alerts;
    }

    // ==================== SOFT_FLAG RULES ====================

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID11(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasMetformin = meds.contains("METFORMIN");
        Double egfr = (Double) state.get("currentEGFR");
        if (hasMetformin && egfr != null && egfr >= 30 && egfr < 45) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-11", "Metformin Dose Cap",
                ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                String.format("INFO: eGFR %.0f — metformin should be capped at 1000mg/day.", egfr),
                "Verify metformin dose ≤1000mg/day. Recheck eGFR in 3 months."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID12(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasStatin = meds.contains("STATIN");
        boolean hasFibrate = meds.contains("FIBRATE");
        if (hasStatin && hasFibrate) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-12", "Statin-Fibrate Myopathy Risk",
                ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                "INFO: Statin + fibrate combination — monitor for myalgia/myopathy.",
                "Check CK if patient reports muscle pain. Prefer fenofibrate over gemfibrozil."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID13(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasRASi = meds.contains("ACEI") || meds.contains("ARB");
        boolean hasSGLT2i = meds.contains("SGLT2I");
        Double egfr = (Double) state.get("currentEGFR");
        Double prevEGFR = (Double) state.get("previousEGFR");
        if (hasRASi && hasSGLT2i && egfr != null && prevEGFR != null) {
            double dropPct = (prevEGFR - egfr) / prevEGFR;
            if (dropPct > 0.10 && dropPct <= 0.20) {
                alerts.add(new ComorbidityAlert("test-patient", "CID-13", "Expected eGFR Dip",
                    ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                    String.format("INFO: eGFR dropped %.0f%% on RASi + SGLT2i — expected hemodynamic dip.", dropPct * 100),
                    "Continue medications. Recheck eGFR in 4 weeks. Only stop if drop >20%."));
            }
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID14(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasAntiplatelet = meds.contains("ASPIRIN") || meds.contains("CLOPIDOGREL");
        boolean hasAnticoagulant = meds.contains("WARFARIN") || meds.contains("NOAC");
        if (hasAntiplatelet && hasAnticoagulant) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-14", "Triple Antithrombotic Risk",
                ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                "INFO: Antiplatelet + anticoagulant — elevated bleeding risk.",
                "Review need for dual therapy. Consider PPI for GI protection."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID15(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasNSAID = meds.contains("NSAID");
        boolean hasRASi = meds.contains("ACEI") || meds.contains("ARB");
        if (hasNSAID && hasRASi) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-15", "NSAID-RASi Renal Risk",
                ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                "INFO: NSAID + RASi combination — increased renal impairment risk.",
                "Avoid chronic NSAID use. Prefer paracetamol. Monitor eGFR."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID16(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasBeta = meds.contains("BETA_BLOCKER");
        boolean hasCCB = meds.contains("CCB");
        if (hasBeta && hasCCB) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-16", "Bradycardia Risk",
                ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                "INFO: Beta-blocker + CCB — verify CCB type. Non-DHP CCBs cause bradycardia.",
                "If verapamil/diltiazem: monitor heart rate closely. Prefer amlodipine."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID17(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasSGLT2i = meds.contains("SGLT2I");
        boolean hasInsulin = meds.contains("INSULIN");
        Integer mealSkips = (Integer) state.get("mealSkips24h");
        boolean possibleFasting = mealSkips != null && mealSkips >= 3;
        if ((hasSGLT2i || hasInsulin) && possibleFasting) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-17", "Fasting Period Drug Risk",
                ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                "INFO: Possible fasting period detected with SGLT2i/insulin — DKA and hypo risk elevated.",
                "Review Ramadan/fasting guidelines. Adjust insulin timing. Consider holding SGLT2i during fasts."));
        }
        return alerts;
    }
}
