package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module6TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6ActionClassifierTest {

    @Test
    void clinicalEvent_fromCDSEvent_extractsNews2Score() {
        CDSEvent cds = new CDSEvent();
        PatientContextState state = new PatientContextState("P001");
        state.setNews2Score(13);
        state.setQsofaScore(2);
        cds.setPatientId("P001");
        cds.setPatientState(state);

        ClinicalEvent event = ClinicalEvent.fromCDS(cds);

        assertEquals("P001", event.getPatientId());
        assertEquals(13, event.getNews2Score());
        assertEquals(2, event.getQsofaScore());
        assertEquals(ClinicalEvent.Source.CDS, event.getSource());
    }

    // ══ HALT conditions ══

    @Test
    void sepsisPatient_news2Above10_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.sepsisHaltEvent("P-SEPSIS");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "NEWS2=13 must produce HALT");
    }

    @Test
    void hyperkalemiaPatient_news2Normal_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.hyperkalemiaHaltEvent("P-HYPER-K");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "K+ 6.5 with CRITICAL alert must produce HALT even with low NEWS2");
    }

    @Test
    void akiStage3_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.akiStage3HaltEvent("P-AKI");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "AKI Stage 3 must produce HALT");
    }

    @Test
    void sepsisMLPrediction_above060_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.sepsisHighRiskPrediction("P-ML-SEPSIS", 0.72);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "Sepsis calibrated score 0.72 must produce HALT");
    }

    @Test
    void sepsisMLPrediction_exactlyAt060_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.sepsisHighRiskPrediction("P-ML-BOUNDARY", 0.60);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "Sepsis calibrated score exactly 0.60 must produce HALT (epsilon)");
    }

    @Test
    void criticalDeteriorationPattern_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.criticalDeteriorationPatternEvent("P-DET-CRIT");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "CLINICAL_DETERIORATION CRITICAL must produce HALT");
    }

    // ══ PAUSE conditions ══

    @Test
    void moderateDeterioration_news2Is7_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.moderateDeteriorationPauseEvent("P-MOD");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "NEWS2=7 must produce PAUSE");
    }

    @Test
    void sepsisMLPrediction_above035_below060_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.sepsisHighRiskPrediction("P-ML-MOD", 0.45);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "Sepsis calibrated score 0.45 must produce PAUSE");
    }

    @Test
    void deteriorationPrediction_above045_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.deteriorationPrediction("P-DET-ML", 0.50);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "Deterioration calibrated score 0.50 must produce PAUSE");
    }

    @Test
    void highSeverityPattern_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.highSeverityPatternEvent("P-PAT-HIGH", "VITAL_SIGNS_TREND");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "HIGH severity pattern must produce PAUSE");
    }

    // ══ SOFT_FLAG conditions ══

    @Test
    void mildElevation_news2Is5_classifiesSoftFlag() {
        ClinicalEvent event = Module6TestBuilder.mildElevationSoftFlagEvent("P-MILD");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.SOFT_FLAG, tier, "NEWS2=5 must produce SOFT_FLAG");
    }

    @Test
    void moderatePattern_classifiesSoftFlag() {
        ClinicalEvent event = Module6TestBuilder.moderatePatternEvent("P-PAT-MOD", "TREND_ANALYSIS");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.SOFT_FLAG, tier, "MODERATE severity pattern must produce SOFT_FLAG");
    }

    @Test
    void anyPredictionAbove025_classifiesSoftFlag() {
        ClinicalEvent event = Module6TestBuilder.deteriorationPrediction("P-DET-LOW", 0.30);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.SOFT_FLAG, tier, "Deterioration calibrated score 0.30 must produce SOFT_FLAG");
    }

    // ══ ROUTINE conditions ══

    @Test
    void stablePatient_classifiesRoutine() {
        ClinicalEvent event = Module6TestBuilder.stableRoutineEvent("P-STABLE");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.ROUTINE, tier, "Stable patient must produce ROUTINE");
    }

    @Test
    void lowRiskPrediction_classifiesRoutine() {
        ClinicalEvent event = Module6TestBuilder.lowRiskPrediction("P-LOW-ML");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.ROUTINE, tier, "Low risk prediction (0.15) must produce ROUTINE");
    }
}
