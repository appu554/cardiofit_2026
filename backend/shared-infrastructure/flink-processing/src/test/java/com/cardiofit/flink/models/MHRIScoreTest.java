package com.cardiofit.flink.models;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class MHRIScoreTest {

    @Test
    void computeComposite_tier1_fullWeights() {
        MHRIScore score = new MHRIScore();
        score.setGlycemicComponent(70.0);
        score.setHemodynamicComponent(60.0);
        score.setRenalComponent(50.0);
        score.setMetabolicComponent(40.0);
        score.setEngagementComponent(80.0);
        score.setDataTier("TIER_1_CGM");

        score.computeComposite();

        // Tier 1 weights: glycemic=25, hemodynamic=25, renal=20, metabolic=15, engagement=15
        // (70*0.25) + (60*0.25) + (50*0.20) + (40*0.15) + (80*0.15) = 17.5 + 15 + 10 + 6 + 12 = 60.5
        assertEquals(60.5, score.getComposite(), 0.01);
        assertEquals("TIER_1_CGM", score.getDataTier());
        assertEquals("HIGH", score.getRiskCategory());
    }

    @Test
    void computeComposite_tier3_redistributedWeights() {
        MHRIScore score = new MHRIScore();
        score.setGlycemicComponent(70.0);
        score.setHemodynamicComponent(60.0);
        score.setRenalComponent(50.0);
        score.setMetabolicComponent(40.0);
        score.setEngagementComponent(80.0);
        score.setDataTier("TIER_3_SMBG");

        score.computeComposite();

        // Tier 3: glycemic weight reduced to 15, hemodynamic boosted to 30, renal 25, metabolic 15, engagement 15
        // (70*0.15) + (60*0.30) + (50*0.25) + (40*0.15) + (80*0.15) = 10.5 + 18 + 12.5 + 6 + 12 = 59.0
        assertEquals(59.0, score.getComposite(), 0.01);
    }

    @Test
    void riskCategory_thresholds() {
        MHRIScore low = new MHRIScore();
        low.setCompositeDirectly(25.0);
        assertEquals("LOW", low.getRiskCategory());

        MHRIScore moderate = new MHRIScore();
        moderate.setCompositeDirectly(50.0);
        assertEquals("MODERATE", moderate.getRiskCategory());

        MHRIScore high = new MHRIScore();
        high.setCompositeDirectly(75.0);
        assertEquals("HIGH", high.getRiskCategory());

        MHRIScore critical = new MHRIScore();
        critical.setCompositeDirectly(90.0);
        assertEquals("CRITICAL", critical.getRiskCategory());
    }

    @Test
    void nullDataTier_defaultsToTier3() {
        MHRIScore score = new MHRIScore();
        score.setGlycemicComponent(70.0);
        score.setHemodynamicComponent(60.0);
        score.setRenalComponent(50.0);
        score.setMetabolicComponent(40.0);
        score.setEngagementComponent(80.0);

        score.computeComposite();

        assertEquals(59.0, score.getComposite(), 0.01);
    }

    @Test
    void computeComposite_tier2Hybrid_interpolatedWeights() {
        MHRIScore score = new MHRIScore();
        score.setGlycemicComponent(70.0);
        score.setHemodynamicComponent(60.0);
        score.setRenalComponent(50.0);
        score.setMetabolicComponent(40.0);
        score.setEngagementComponent(80.0);
        score.setDataTier("TIER_2_FINGERSTICK");

        score.computeComposite();

        // Tier 2 weights: glycemic=0.20, hemodynamic=0.275, renal=0.225, metabolic=0.15, engagement=0.15
        // (70*0.20) + (60*0.275) + (50*0.225) + (40*0.15) + (80*0.15)
        // = 14.0 + 16.5 + 11.25 + 6.0 + 12.0 = 59.75
        assertEquals(59.75, score.getComposite(), 0.01);
    }
}
