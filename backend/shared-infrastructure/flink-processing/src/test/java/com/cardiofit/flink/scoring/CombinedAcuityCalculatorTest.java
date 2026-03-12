package com.cardiofit.flink.scoring;

import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for CombinedAcuityCalculator
 *
 * Tests weighted combination formula: (0.7 × NEWS2) + (0.3 × Metabolic)
 */
public class CombinedAcuityCalculatorTest {

    @Test
    public void testLowAcuity() {
        // Low NEWS2 + Low metabolic = Low combined
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(1); // Low
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(0.0, "LOW");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 1) + (0.3 * 0) = 0.7
        assertEquals(0.7, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("LOW", combined.getAcuityLevel());
        assertTrue(combined.getMonitoringRecommendation().contains("Routine"));
    }

    @Test
    public void testMediumAcuity() {
        // Medium NEWS2 + Moderate metabolic = Medium combined
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(3); // Medium
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(2.0, "MODERATE");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 3) + (0.3 * 2) = 2.1 + 0.6 = 2.7
        assertEquals(2.7, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("MEDIUM", combined.getAcuityLevel());
        assertTrue(combined.getMonitoringRecommendation().contains("Increased monitoring"));
    }

    @Test
    public void testHighAcuity() {
        // High NEWS2 + High metabolic = High combined
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(7); // High
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(3.0, "HIGH");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 7) + (0.3 * 3) = 4.9 + 0.9 = 5.8
        assertEquals(5.8, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("HIGH", combined.getAcuityLevel());
        assertTrue(combined.getMonitoringRecommendation().contains("Urgent"));
    }

    @Test
    public void testCriticalAcuity() {
        // Critical NEWS2 + High metabolic = Critical combined
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(10); // Critical
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(4.0, "HIGH");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 10) + (0.3 * 4) = 7.0 + 1.2 = 8.2
        assertEquals(8.2, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("CRITICAL", combined.getAcuityLevel());
        assertTrue(combined.getMonitoringRecommendation().contains("Emergency"));
    }

    @Test
    public void testAcuteDominance() {
        // High NEWS2 with low metabolic - acute-driven
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(8);
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(1.0, "LOW");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 8) + (0.3 * 1) = 5.6 + 0.3 = 5.9
        assertEquals(5.9, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("HIGH", combined.getAcuityLevel());
        assertTrue(combined.getInterpretation().contains("Primarily driven by acute physiological"));
    }

    @Test
    public void testChronicDominance() {
        // Low NEWS2 with high metabolic - chronic-driven
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(2);
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(5.0, "HIGH");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 2) + (0.3 * 5) = 1.4 + 1.5 = 2.9
        assertEquals(2.9, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("MEDIUM", combined.getAcuityLevel());
        assertTrue(combined.getInterpretation().contains("Primarily driven by chronic metabolic"));
    }

    @Test
    public void testBalancedRisk() {
        // Moderate in both - balanced
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(4);
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(3.0, "HIGH");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 4) + (0.3 * 3) = 2.8 + 0.9 = 3.7
        assertEquals(3.7, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("MEDIUM", combined.getAcuityLevel());
        assertTrue(combined.getInterpretation().contains("Balanced acute and chronic"));
    }

    @Test
    public void testExactThresholds() {
        // Test exact threshold boundaries
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(10);
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(0.0, "LOW");

        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // (0.7 * 10) + (0.3 * 0) = 7.0 (exactly at CRITICAL threshold)
        assertEquals(7.0, combined.getCombinedAcuityScore(), 0.01);
        assertEquals("CRITICAL", combined.getAcuityLevel());
    }

    // Helper methods

    private NEWS2Calculator.NEWS2Score createNEWS2Score(int totalScore) {
        NEWS2Calculator.NEWS2Score score = new NEWS2Calculator.NEWS2Score();
        score.setTotalScore(totalScore);

        // Set risk level based on score
        if (totalScore >= 7) {
            score.setRiskLevel("HIGH");
        } else if (totalScore >= 5) {
            score.setRiskLevel("MEDIUM");
        } else {
            score.setRiskLevel("LOW");
        }

        return score;
    }

    private MetabolicAcuityCalculator.MetabolicAcuityScore createMetabolicScore(
            double score, String riskLevel) {

        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic =
            new MetabolicAcuityCalculator.MetabolicAcuityScore();

        metabolic.setScore(score);
        metabolic.setRiskLevel(riskLevel);
        metabolic.setComponentCount((int) score);
        metabolic.setPresentComponents(new ArrayList<>());

        return metabolic;
    }
}
