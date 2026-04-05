package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class Module13CKMRiskComputerTest {

    @Test
    void metabolicVelocity_decliningFBG_returnsImproving() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.previous().fbg = 140.0;
        state.previous().hba1c = 7.5;
        state.current().fbg = 120.0;
        state.current().hba1c = 7.0;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.getDomainVelocity(CKMRiskDomain.METABOLIC) < 0,
                "Metabolic velocity should be negative (improving) when FBG dropped 20 mg/dL");
    }

    @Test
    void renalVelocity_rapidEGFRDecline_returnsDeteriorating() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.previous().egfr = 65.0;
        state.previous().uacr = 30.0;
        state.current().egfr = 55.0;
        state.current().uacr = 120.0;
        state.current().arv = 14.0;
        state.current().meanSBP = 148.0;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.getDomainVelocity(CKMRiskDomain.RENAL) > 0.4,
                "Renal velocity should exceed deteriorating threshold for 10-point eGFR drop in 7 days");
    }

    @Test
    void cardiovascularVelocity_worseningARVAndSurge_returnsDeteriorating() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.previous().arv = 9.0;
        state.previous().morningSurgeMagnitude = 18.0;
        state.previous().engagementScore = 0.8;
        state.current().arv = 18.0;
        state.current().morningSurgeMagnitude = 45.0;
        state.current().engagementScore = 0.3;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.getDomainVelocity(CKMRiskDomain.CARDIOVASCULAR) > 0.4,
                "CV velocity should exceed deteriorating threshold");
    }

    @Test
    void composite_worstDomainWins() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.previous().egfr = 65.0; state.current().egfr = 50.0;
        state.previous().uacr = 30.0; state.current().uacr = 180.0;
        state.previous().fbg = 130.0; state.current().fbg = 128.0;
        state.previous().arv = 10.0; state.current().arv = 11.0;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                result.getCompositeClassification(),
                "Composite should be DETERIORATING when any domain exceeds threshold");
    }

    @Test
    void composite_crossDomainAmplification_when2DomainsWorsening() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.previous().egfr = 65.0; state.current().egfr = 52.0;
        state.previous().uacr = 30.0; state.current().uacr = 150.0;
        state.previous().fbg = 130.0; state.current().fbg = 175.0;
        state.previous().hba1c = 7.2; state.current().hba1c = 8.5;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.isCrossDomainAmplification(),
                "Should flag cross-domain amplification");
        assertEquals(1.5, result.getAmplificationFactor(), 0.01,
                "Amplification factor should be 1.5x");
        assertTrue(result.getDomainsDeteriorating() >= 2);
    }

    @Test
    void composite_allDomainsImproving_returnsImproving() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.previous().fbg = 145.0; state.current().fbg = 115.0;
        state.previous().hba1c = 7.8; state.current().hba1c = 6.9;
        state.previous().egfr = 58.0; state.current().egfr = 64.0;
        state.previous().uacr = 80.0; state.current().uacr = 35.0;
        state.previous().arv = 14.0; state.current().arv = 8.0;
        state.previous().engagementScore = 0.5; state.current().engagementScore = 0.9;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.IMPROVING,
                result.getCompositeClassification());
        assertFalse(result.isCrossDomainAmplification());
    }

    @Test
    void compute_noPreviousSnapshot_returnsUnknown() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN,
                result.getCompositeClassification(),
                "Cannot compute velocity without previous snapshot");
    }
}
