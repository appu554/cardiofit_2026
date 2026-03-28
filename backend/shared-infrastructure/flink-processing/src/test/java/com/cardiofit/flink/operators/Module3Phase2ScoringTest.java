package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3Phase2ScoringTest {

    @Test
    void hypertensiveDiabetic_computesMHRI() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("HTN-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);
        MHRIScore mhri = (MHRIScore) result.getDetail("mhriScore");

        assertTrue(result.isActive());
        assertNotNull(mhri);
        assertNotNull(mhri.getComposite());
        assertTrue(mhri.getComposite() > 0);
        assertEquals("TIER_3_SMBG", mhri.getDataTier());
        assertNotNull(mhri.getHemodynamicComponent());
        assertNotNull(mhri.getGlycemicComponent());
        assertNotNull(mhri.getRenalComponent());
    }

    @Test
    void cgmPatient_usesTier1Weights() {
        EnrichedPatientContext patient = Module3TestBuilder.cgmPatient("CGM-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);
        MHRIScore mhri = (MHRIScore) result.getDetail("mhriScore");

        assertEquals("TIER_1_CGM", mhri.getDataTier());
        assertNotNull(mhri.getComposite());
    }

    @Test
    void extractsNEWS2andQSOFA() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SEP-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);

        assertEquals(9, result.getDetail("news2Score"));
        assertEquals(2, result.getDetail("qsofaScore"));
        assertEquals(8.5, result.getDetail("combinedAcuityScore"));
    }

    @Test
    void estimatesCKD_EPI_eGFR() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("CKD-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);

        // Patient has creatinine 1.4 mg/dL, age 58, male → eGFR should be ~55-60
        Double egfr = (Double) result.getDetail("estimatedGFR");
        assertNotNull(egfr);
        assertTrue(egfr > 40 && egfr < 70, "eGFR for Cr=1.4, age=58, male should be ~55-60, got " + egfr);
    }
}
