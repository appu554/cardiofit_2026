package com.cardiofit.flink.intelligence;

import com.cardiofit.flink.models.SimpleAlert;
import com.cardiofit.flink.models.AlertType;
import com.cardiofit.flink.models.AlertSeverity;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Alert Deduplication and Consolidation Engine
 *
 * Consolidates overlapping clinical alerts into parent-child hierarchies to reduce
 * cognitive load while preserving clinical reasoning chains.
 *
 * Example: Sepsis Detection Consolidation
 * ----------------------------------------
 * Before (3 alerts):
 * 1. "SEPSIS LIKELY - SIRS criteria with elevated lactate" (HIGH)
 * 2. "SIRS criteria met (3/4) - Consider sepsis workup" (HIGH)
 * 3. "SIRS CRITERIA MET (score 3/4) with infection markers" (WARNING)
 *
 * After (1 parent alert with 2 referenced children):
 * 1. "SEPSIS LIKELY - SIRS criteria with elevated lactate" (HIGH)
 *    - relatedAlerts: [alert_2_id, alert_3_id]
 *    - consolidatedFrom: ["SIRS criteria", "infection markers"]
 * 2. Alert #2 marked with suppressDisplay: true
 * 3. Alert #3 marked with suppressDisplay: true
 *
 * This preserves the full clinical reasoning chain while reducing duplicate display.
 */
public class AlertDeduplicator {
    private static final Logger LOG = LoggerFactory.getLogger(AlertDeduplicator.class);

    /**
     * Deduplicate and consolidate alerts using pattern-based rules
     *
     * @param alerts List of alerts to deduplicate
     * @param patientId Patient identifier for logging
     * @return Deduplicated list with parent-child relationships established
     */
    public static Set<SimpleAlert> deduplicateAlerts(Set<SimpleAlert> alerts, String patientId) {
        if (alerts == null || alerts.isEmpty()) {
            return alerts;
        }

        LOG.info("Deduplicating {} alerts for patient {}", alerts.size(), patientId);

        // Convert to list for processing
        List<SimpleAlert> alertList = new ArrayList<>(alerts);

        // Log all alert messages for debugging
        for (SimpleAlert alert : alertList) {
            LOG.info("  - [{}] {}: {}", alert.getAlertType(), alert.getSeverity(), alert.getMessage());
        }

        // Apply consolidation rules
        consolidateSepsisAlerts(alertList);
        consolidateRespiratoryAlerts(alertList);
        consolidateCardiacAlerts(alertList);

        // Convert back to set (excludes suppressed alerts from top-level display)
        Set<SimpleAlert> result = new LinkedHashSet<>(alertList);

        LOG.info("Alert deduplication for patient {}: {} -> {} alerts",
                patientId, alerts.size(), result.size());

        return result;
    }

    /**
     * Consolidate sepsis-related alerts into a single parent alert
     *
     * Pattern: If we have "SEPSIS LIKELY" alert, suppress child SIRS alerts
     */
    private static void consolidateSepsisAlerts(List<SimpleAlert> alerts) {
        LOG.debug("Checking for sepsis consolidation among {} alerts", alerts.size());

        // Find the parent sepsis alert (highest severity)
        SimpleAlert sepsisParent = alerts.stream()
                .filter(a -> a.getMessage() != null && a.getMessage().contains("SEPSIS LIKELY"))
                .findFirst()
                .orElse(null);

        if (sepsisParent == null) {
            LOG.debug("No SEPSIS LIKELY parent alert found for consolidation");
            return; // No sepsis parent to consolidate around
        }

        LOG.debug("Found SEPSIS parent alert: {}", sepsisParent.getAlertId());

        // Find child alerts to consolidate into sepsis parent
        // Include: SIRS alerts, fever (SIRS component), elevated lactate (sepsis diagnostic component)
        List<SimpleAlert> sepsisChildren = alerts.stream()
                .filter(a -> a.getMessage() != null &&
                        ((a.getMessage().contains("SIRS criteria met") ||
                          a.getMessage().contains("SIRS CRITERIA MET")) ||
                         (a.getMessage().contains("Lactate elevated") && a.getAlertType() == AlertType.LAB_ABNORMALITY) ||
                         (a.getMessage().contains("Fever") && a.getAlertType() == AlertType.VITAL_THRESHOLD_BREACH)))
                .filter(a -> a != sepsisParent) // Don't include parent in children
                .collect(Collectors.toList());

        if (sepsisChildren.isEmpty()) {
            return; // No children to consolidate
        }

        // Establish parent-child relationship
        List<String> childIds = sepsisChildren.stream()
                .map(SimpleAlert::getAlertId)
                .collect(Collectors.toList());

        List<String> evidenceSources = sepsisChildren.stream()
                .map(SimpleAlert::getMessage)
                .collect(Collectors.toList());

        // Update parent alert
        sepsisParent.setAlertHierarchy("parent");
        sepsisParent.setRelatedAlerts(childIds);
        sepsisParent.setConsolidatedFrom(evidenceSources);

        // Add evidence details to parent context
        Map<String, Object> parentContext = sepsisParent.getContext();
        if (parentContext == null) {
            parentContext = new HashMap<>();
            sepsisParent.setContext(parentContext);
        }
        parentContext.put("consolidatedAlerts", childIds.size());
        parentContext.put("evidenceChain", evidenceSources);

        // Mark children as suppressed
        for (SimpleAlert child : sepsisChildren) {
            child.setAlertHierarchy("child");
            child.setSuppressDisplay(true);
            List<String> parentList = new ArrayList<>();
            parentList.add(sepsisParent.getAlertId());
            child.setRelatedAlerts(parentList);

            LOG.info("Suppressed child alert: {} | Type: {} | Message: {}",
                    child.getAlertId(), child.getAlertType(), child.getMessage());
        }

        LOG.info("Consolidated {} sepsis-related alerts (SIRS + fever + lactate) into SEPSIS LIKELY parent", sepsisChildren.size());
    }

    /**
     * Consolidate respiratory alerts (hypoxia + tachypnea)
     *
     * Pattern: If both hypoxia and tachypnea present, consolidate under "RESPIRATORY_DISTRESS"
     */
    private static void consolidateRespiratoryAlerts(List<SimpleAlert> alerts) {
        // Find hypoxia and tachypnea alerts
        SimpleAlert hypoxia = alerts.stream()
                .filter(a -> a.getMessage() != null && a.getMessage().contains("Hypoxia detected"))
                .findFirst()
                .orElse(null);

        SimpleAlert tachypnea = alerts.stream()
                .filter(a -> a.getMessage() != null && a.getMessage().contains("Tachypnea detected"))
                .findFirst()
                .orElse(null);

        // Only consolidate if both present
        if (hypoxia == null || tachypnea == null) {
            return;
        }

        // Choose higher severity as parent (or hypoxia as default)
        SimpleAlert parent = hypoxia.getSeverity().ordinal() >= tachypnea.getSeverity().ordinal()
                ? hypoxia : tachypnea;
        SimpleAlert child = (parent == hypoxia) ? tachypnea : hypoxia;

        // Establish relationship
        List<String> childIds = Arrays.asList(child.getAlertId());
        parent.setAlertHierarchy("parent");
        parent.setRelatedAlerts(childIds);

        child.setAlertHierarchy("child");
        child.setSuppressDisplay(true);
        child.setRelatedAlerts(Arrays.asList(parent.getAlertId()));

        // Update parent message to reflect consolidation
        Map<String, Object> context = parent.getContext();
        if (context == null) {
            context = new HashMap<>();
            parent.setContext(context);
        }
        context.put("consolidatedRespiratory", true);

        LOG.debug("Consolidated respiratory alerts: {} (parent) + {} (child)",
                parent.getAlertId(), child.getAlertId());
    }

    /**
     * Consolidate cardiac alerts (placeholder for future enhancement)
     */
    private static void consolidateCardiacAlerts(List<SimpleAlert> alerts) {
        // Future: Consolidate multiple cardiac markers into ACS parent alert
        // Example: Troponin + CK-MB + chest pain → "Acute Coronary Syndrome Suspected"
    }
}
