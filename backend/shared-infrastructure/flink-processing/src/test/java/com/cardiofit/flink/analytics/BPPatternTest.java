package com.cardiofit.flink.analytics;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class BPPatternTest {

    @Test
    void surge_normal_whenBelow20() {
        double surge = MorningSurgeDetector.computeSurge(135.0, 120.0);
        assertEquals(15.0, surge, 0.01);
        assertEquals("NORMAL", MorningSurgeDetector.classify(surge));
    }

    @Test
    void surge_exaggerated_whenAbove35() {
        double surge = MorningSurgeDetector.computeSurge(175.0, 135.0);
        assertEquals("EXAGGERATED", MorningSurgeDetector.classify(surge));
    }

    @Test
    void dipping_dipper_when10to20percent() {
        String classification = DippingPatternClassifier.classify(140.0, 125.0);
        assertEquals("DIPPER", classification);
    }

    @Test
    void dipping_nonDipper_when0to10percent() {
        assertEquals("NON_DIPPER", DippingPatternClassifier.classify(140.0, 133.0));
    }

    @Test
    void dipping_reverse_whenNightHigherThanDay() {
        assertEquals("REVERSE", DippingPatternClassifier.classify(130.0, 135.0));
    }
}
