package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.List;

/**
 * BP control status classification and white-coat/masked HTN detection.
 *
 * BP control uses 7-day average SBP and DBP against home BP thresholds
 * (5 mmHg lower than clinic thresholds per JSH 2025).
 *
 * White-coat detection: clinic_avg_SBP - home_avg_SBP > 15 mmHg.
 * Masked HTN detection: home_avg_SBP - clinic_avg_SBP > 15 mmHg.
 * Both require >= 2 clinic readings and >= 5 home readings in 30 days.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7BPControlClassifier {

    private Module7BPControlClassifier() {}

    private static final double WHITE_COAT_THRESHOLD = 15.0; // mmHg

    public record WhiteCoatResult(boolean whiteCoatSuspect, boolean maskedHtnSuspect,
                                   Double clinicHomeDelta) {}

    /**
     * Classify BP control status from 7-day daily summaries.
     */
    public static BPControlStatus classifyControl(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.isEmpty()) return BPControlStatus.ELEVATED; // conservative

        double avgSBP = Module7ARVComputer.computeMeanSBP(summaries);
        double avgDBP = Module7ARVComputer.computeMeanDBP(summaries);

        return BPControlStatus.fromAverages(avgSBP, avgDBP);
    }

    /**
     * Detect white-coat and masked hypertension from clinic vs home readings.
     * Requires >= 2 clinic readings and >= 5 home readings.
     */
    public static WhiteCoatResult detectWhiteCoatMasked(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.isEmpty()) {
            return new WhiteCoatResult(false, false, null);
        }

        double sumClinic = 0;
        int clinicReadings = 0;
        double sumHome = 0;
        int homeReadings = 0;

        for (DailyBPSummary s : summaries) {
            if (s.getClinicAvgSBP() != null && s.getClinicCount() > 0) {
                sumClinic += s.getClinicAvgSBP() * s.getClinicCount();
                clinicReadings += s.getClinicCount();
            }
            if (s.getHomeAvgSBP() != null && s.getHomeCount() > 0) {
                sumHome += s.getHomeAvgSBP() * s.getHomeCount();
                homeReadings += s.getHomeCount();
            }
        }

        if (clinicReadings < 2 || homeReadings < 5) {
            return new WhiteCoatResult(false, false, null);
        }

        double clinicMean = sumClinic / clinicReadings;
        double homeMean = sumHome / homeReadings;
        double delta = clinicMean - homeMean;

        boolean whiteCoat = delta > WHITE_COAT_THRESHOLD;
        boolean masked = delta < -WHITE_COAT_THRESHOLD;

        return new WhiteCoatResult(whiteCoat, masked, delta);
    }
}
