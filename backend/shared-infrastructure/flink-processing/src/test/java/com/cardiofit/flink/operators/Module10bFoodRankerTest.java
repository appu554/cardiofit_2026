package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.HashMap;
import static org.junit.jupiter.api.Assertions.*;

class Module10bFoodRankerTest {

    @Test
    void ranksTopFoodsByMeanExcursion() {
        List<MealResponseRecord> records = new ArrayList<>();
        records.add(mealWith("rice", 60.0, 8000.0));
        records.add(mealWith("rice", 55.0, 7500.0));
        records.add(mealWith("salad", 15.0, 2000.0));
        records.add(mealWith("bread", 45.0, 6000.0));
        records.add(mealWith("bread", 50.0, 6500.0));

        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(records, 5);

        assertEquals(3, ranked.size());
        assertEquals("rice", ranked.get(0).foodDescription);
        assertEquals(2, ranked.get(0).mealCount);
        assertEquals(57.5, ranked.get(0).meanExcursion, 0.1);
        assertEquals("bread", ranked.get(1).foodDescription);
    }

    @Test
    void limitOutput_topN() {
        List<MealResponseRecord> records = new ArrayList<>();
        for (int i = 0; i < 20; i++) {
            records.add(mealWith("food_" + i, 10.0 + i * 5, 1000.0 + i * 500));
        }

        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(records, 3);

        assertEquals(3, ranked.size());
    }

    @Test
    void emptyRecords_returnsEmpty() {
        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(new ArrayList<>(), 5);
        assertTrue(ranked.isEmpty());
    }

    @Test
    void skipsRecordsWithNoFoodDescription() {
        List<MealResponseRecord> records = new ArrayList<>();
        records.add(mealWith(null, 50.0, 7000.0));
        records.add(mealWith("rice", 60.0, 8000.0));

        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(records, 5);

        assertEquals(1, ranked.size());
        assertEquals("rice", ranked.get(0).foodDescription);
    }

    private MealResponseRecord mealWith(String food, double excursion, double iauc) {
        Map<String, Object> payload = new HashMap<>();
        if (food != null) payload.put("food_description", food);

        return MealResponseRecord.builder()
            .recordId("test-" + System.nanoTime())
            .patientId("P1")
            .mealEventId("meal-" + System.nanoTime())
            .mealTimestamp(System.currentTimeMillis())
            .glucoseExcursion(excursion)
            .iAUC(iauc)
            .mealPayload(payload)
            .build();
    }
}
