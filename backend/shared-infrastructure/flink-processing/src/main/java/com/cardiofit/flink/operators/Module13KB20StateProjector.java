package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.*;

public final class Module13KB20StateProjector {

    private Module13KB20StateProjector() {}

    /**
     * Map a CanonicalEvent from an upstream module to a list of KB20StateUpdate diffs.
     * Returns empty list if the event source is unrecognised.
     */
    public static List<KB20StateUpdate> project(CanonicalEvent event) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) return Collections.emptyList();

        String sourceModule = payload.get("source_module") != null
                ? payload.get("source_module").toString() : "";

        switch (sourceModule) {
            case "module7":
                return projectBPVariability(event, payload);
            case "module9":
                return projectEngagement(event, payload);
            case "module10b":
                return projectMealPatterns(event, payload);
            case "module11b":
                return projectFitnessPatterns(event, payload);
            case "module12":
                return projectInterventionWindow(event, payload);
            case "module12b":
                return projectInterventionDelta(event, payload);
            case "enriched":
                return projectLabResult(event, payload);
            default:
                return Collections.emptyList();
        }
    }

    private static List<KB20StateUpdate> projectBPVariability(CanonicalEvent event, Map<String, Object> payload) {
        List<KB20StateUpdate> updates = new ArrayList<>();
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module7", "bp_variability_arv", payload.get("arv"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module7", "bp_variability_classification", payload.get("variability_classification"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module7", "bp_mean_sbp", payload.get("mean_sbp"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module7", "bp_mean_dbp", payload.get("mean_dbp"));
        return updates;
    }

    private static List<KB20StateUpdate> projectEngagement(CanonicalEvent event, Map<String, Object> payload) {
        List<KB20StateUpdate> updates = new ArrayList<>();
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module9", "engagement_composite_score", payload.get("composite_score"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module9", "engagement_level", payload.get("engagement_level"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module9", "engagement_phenotype", payload.get("phenotype"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module9", "data_tier", payload.get("data_tier"));
        return updates;
    }

    private static List<KB20StateUpdate> projectMealPatterns(CanonicalEvent event, Map<String, Object> payload) {
        List<KB20StateUpdate> updates = new ArrayList<>();
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.MERGE, "module10b", "meal_mean_iauc", payload.get("mean_iauc"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.MERGE, "module10b", "meal_median_excursion", payload.get("median_excursion"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.MERGE, "module10b", "salt_sensitivity_class", payload.get("salt_sensitivity_class"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.MERGE, "module10b", "salt_beta", payload.get("salt_beta"));
        return updates;
    }

    private static List<KB20StateUpdate> projectFitnessPatterns(CanonicalEvent event, Map<String, Object> payload) {
        List<KB20StateUpdate> updates = new ArrayList<>();
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module11b", "estimated_vo2max", payload.get("estimated_vo2max"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module11b", "vo2max_trend", payload.get("vo2max_trend"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module11b", "total_met_minutes", payload.get("total_met_minutes"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "module11b", "mean_exercise_glucose_delta", payload.get("mean_exercise_glucose_delta"));
        return updates;
    }

    private static List<KB20StateUpdate> projectInterventionWindow(CanonicalEvent event, Map<String, Object> payload) {
        List<KB20StateUpdate> updates = new ArrayList<>();
        // Intervention windows use APPEND to add to active_interventions list
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12", "active_interventions.intervention_id", payload.get("intervention_id"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12", "active_interventions.window_status", payload.get("signal_type"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12", "active_interventions.intervention_type", payload.get("intervention_type"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12", "active_interventions.observation_start_ms", payload.get("observation_start_ms"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12", "active_interventions.observation_end_ms", payload.get("observation_end_ms"));
        return updates;
    }

    private static List<KB20StateUpdate> projectInterventionDelta(CanonicalEvent event, Map<String, Object> payload) {
        List<KB20StateUpdate> updates = new ArrayList<>();
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12b", "intervention_outcomes.intervention_id", payload.get("intervention_id"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12b", "intervention_outcomes.trajectory_attribution", payload.get("trajectory_attribution"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12b", "intervention_outcomes.adherence_score", payload.get("adherence_score"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12b", "intervention_outcomes.fbg_delta", payload.get("fbg_delta"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12b", "intervention_outcomes.sbp_delta", payload.get("sbp_delta"));
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.APPEND, "module12b", "intervention_outcomes.egfr_delta", payload.get("egfr_delta"));
        return updates;
    }

    private static List<KB20StateUpdate> projectLabResult(CanonicalEvent event, Map<String, Object> payload) {
        List<KB20StateUpdate> updates = new ArrayList<>();
        String labType = payload.get("lab_type") != null ? payload.get("lab_type").toString() : "";
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "enriched", "lab_type", labType);
        addIfPresent(updates, event, KB20StateUpdate.UpdateOperation.REPLACE, "enriched", "lab_value", payload.get("value"));
        return updates;
    }

    private static void addIfPresent(List<KB20StateUpdate> updates, CanonicalEvent event,
                                      KB20StateUpdate.UpdateOperation operation, String sourceModule,
                                      String fieldPath, Object value) {
        if (value == null) return;
        updates.add(KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(operation)
                .sourceModule(sourceModule)
                .fieldPath(fieldPath)
                .value(value)
                .updateTimestamp(event.getEventTime())
                .build());
    }
}
