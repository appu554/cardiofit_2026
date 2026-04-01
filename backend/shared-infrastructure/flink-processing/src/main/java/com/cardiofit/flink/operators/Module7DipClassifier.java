package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import com.cardiofit.flink.models.DipClassification;
import java.util.List;

/**
 * Nocturnal dipping pattern classifier.
 * Dip ratio = 1 - (nocturnal_mean_SBP / daytime_mean_SBP)
 * Requires at least 3 days with both daytime AND nocturnal readings.
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7DipClassifier {

    private Module7DipClassifier() {}

    /** Immutable result record for dipping analysis. */
    public record DipResult(DipClassification classification, Double dipRatio,
                            Double daytimeMean, Double nocturnalMean, int validDays) {}

    public static DipResult classify(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.isEmpty()) {
            return new DipResult(DipClassification.INSUFFICIENT_DATA, null, null, null, 0);
        }

        double sumDaytime = 0;
        double sumNocturnal = 0;
        int daytimeDays = 0;
        int nocturnalDays = 0;

        for (DailyBPSummary s : summaries) {
            if (s.getDaytimeAvgSBP() != null && s.getDaytimeCount() > 0) {
                sumDaytime += s.getDaytimeAvgSBP();
                daytimeDays++;
            }
            if (s.getNocturnalAvgSBP() != null && s.getNocturnalCount() > 0) {
                sumNocturnal += s.getNocturnalAvgSBP();
                nocturnalDays++;
            }
        }

        if (daytimeDays < 3 || nocturnalDays < 3) {
            return new DipResult(DipClassification.INSUFFICIENT_DATA,
                null, null, null, Math.min(daytimeDays, nocturnalDays));
        }

        double daytimeMean = sumDaytime / daytimeDays;
        double nocturnalMean = sumNocturnal / nocturnalDays;

        if (daytimeMean < 1e-9) {
            return new DipResult(DipClassification.INSUFFICIENT_DATA,
                null, daytimeMean, nocturnalMean, Math.min(daytimeDays, nocturnalDays));
        }

        double dipRatio = 1.0 - (nocturnalMean / daytimeMean);
        DipClassification classification = DipClassification.fromDipRatio(dipRatio);

        return new DipResult(classification, dipRatio, daytimeMean, nocturnalMean,
            Math.min(daytimeDays, nocturnalDays));
    }
}
