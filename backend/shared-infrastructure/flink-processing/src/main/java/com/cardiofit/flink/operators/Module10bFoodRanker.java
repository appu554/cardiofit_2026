package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.MealPatternSummary;
import com.cardiofit.flink.models.MealResponseRecord;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Food impact ranker for Module 10b.
 * Groups meal records by food_description, computes mean excursion and mean iAUC,
 * returns top-N foods ranked by mean excursion descending.
 */
public class Module10bFoodRanker {

    private Module10bFoodRanker() {}

    public static List<MealPatternSummary.FoodImpact> rank(
            List<MealResponseRecord> records, int topN) {
        if (records == null || records.isEmpty()) return Collections.emptyList();

        Map<String, List<MealResponseRecord>> byFood = new LinkedHashMap<>();
        for (MealResponseRecord r : records) {
            String food = extractFoodDescription(r);
            if (food == null || food.isBlank()) continue;
            byFood.computeIfAbsent(food, k -> new ArrayList<>()).add(r);
        }

        List<MealPatternSummary.FoodImpact> impacts = new ArrayList<>();
        for (Map.Entry<String, List<MealResponseRecord>> entry : byFood.entrySet()) {
            List<MealResponseRecord> meals = entry.getValue();
            double sumExcursion = 0, sumIAUC = 0;
            int countExcursion = 0, countIAUC = 0;

            for (MealResponseRecord m : meals) {
                if (m.getGlucoseExcursion() != null) {
                    sumExcursion += m.getGlucoseExcursion();
                    countExcursion++;
                }
                if (m.getIAUC() != null) {
                    sumIAUC += m.getIAUC();
                    countIAUC++;
                }
            }

            double meanExcursion = countExcursion > 0 ? sumExcursion / countExcursion : 0.0;
            double meanIAUC = countIAUC > 0 ? sumIAUC / countIAUC : 0.0;

            impacts.add(new MealPatternSummary.FoodImpact(
                entry.getKey(), meals.size(), meanExcursion, meanIAUC));
        }

        impacts.sort((a, b) -> Double.compare(b.meanExcursion, a.meanExcursion));
        return impacts.stream().limit(topN).collect(Collectors.toList());
    }

    private static String extractFoodDescription(MealResponseRecord record) {
        if (record.getMealPayload() == null) return null;
        Object desc = record.getMealPayload().get("food_description");
        return desc != null ? desc.toString().toLowerCase().trim() : null;
    }
}
