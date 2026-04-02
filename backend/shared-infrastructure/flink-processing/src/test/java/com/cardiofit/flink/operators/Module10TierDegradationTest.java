package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for data tier classification and degradation.
 */
class Module10TierDegradationTest {

    @Test
    void defaultTier_isSMBG() {
        MealCorrelationState state = new MealCorrelationState("P1");
        assertEquals(DataTier.TIER_3_SMBG, state.getDataTier());
    }

    @Test
    void dataTier_fromString_variants() {
        assertEquals(DataTier.TIER_1_CGM, DataTier.fromString("TIER_1_CGM"));
        assertEquals(DataTier.TIER_1_CGM, DataTier.fromString("CGM"));
        assertEquals(DataTier.TIER_2_HYBRID, DataTier.fromString("TIER_2_HYBRID"));
        assertEquals(DataTier.TIER_2_HYBRID, DataTier.fromString("HYBRID"));
        assertEquals(DataTier.TIER_3_SMBG, DataTier.fromString("TIER_3_SMBG"));
        assertEquals(DataTier.TIER_3_SMBG, DataTier.fromString(null));
        assertEquals(DataTier.TIER_3_SMBG, DataTier.fromString("UNKNOWN"));
    }

    @Test
    void tier1_supportsCurveClassification() {
        assertTrue(DataTier.TIER_1_CGM.supportsCurveClassification());
        assertFalse(DataTier.TIER_2_HYBRID.supportsCurveClassification());
        assertFalse(DataTier.TIER_3_SMBG.supportsCurveClassification());
    }

    @Test
    void tier1And2_supportFullIAUC() {
        assertTrue(DataTier.TIER_1_CGM.supportsFullIAUC());
        assertTrue(DataTier.TIER_2_HYBRID.supportsFullIAUC());
        assertFalse(DataTier.TIER_3_SMBG.supportsFullIAUC());
    }

    @Test
    void mealTimeCategory_fromTimestamp() {
        // 07:00 UTC → BREAKFAST
        long breakfast = java.time.ZonedDateTime.of(2025, 4, 2, 7, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.BREAKFAST, MealTimeCategory.fromTimestamp(breakfast));

        // 12:00 UTC → LUNCH
        long lunch = java.time.ZonedDateTime.of(2025, 4, 2, 12, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.LUNCH, MealTimeCategory.fromTimestamp(lunch));

        // 19:00 UTC → DINNER
        long dinner = java.time.ZonedDateTime.of(2025, 4, 2, 19, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.DINNER, MealTimeCategory.fromTimestamp(dinner));

        // 23:00 UTC → SNACK
        long snack = java.time.ZonedDateTime.of(2025, 4, 2, 23, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.SNACK, MealTimeCategory.fromTimestamp(snack));
    }

    @Test
    void saltSensitivityClass_fromBetaAndR2() {
        assertEquals(SaltSensitivityClass.HIGH,
            SaltSensitivityClass.fromBetaAndR2(0.01, 0.5, 40));
        assertEquals(SaltSensitivityClass.MODERATE,
            SaltSensitivityClass.fromBetaAndR2(0.003, 0.3, 35));
        assertEquals(SaltSensitivityClass.SALT_RESISTANT,
            SaltSensitivityClass.fromBetaAndR2(0.0005, 0.2, 30));
        assertEquals(SaltSensitivityClass.UNDETERMINED,
            SaltSensitivityClass.fromBetaAndR2(0.01, 0.5, 20)); // too few pairs
        assertEquals(SaltSensitivityClass.UNDETERMINED,
            SaltSensitivityClass.fromBetaAndR2(0.01, 0.05, 40)); // low R²
    }
}
