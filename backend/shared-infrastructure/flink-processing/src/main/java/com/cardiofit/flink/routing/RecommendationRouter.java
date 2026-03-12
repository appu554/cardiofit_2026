package com.cardiofit.flink.routing;

import com.cardiofit.flink.models.ClinicalRecommendation;
import org.apache.flink.api.common.functions.RichMapFunction;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;

/**
 * RecommendationRouter - Routes clinical recommendations to appropriate output channels
 * based on urgency level using Flink side outputs.
 *
 * <p>Routing Strategy:
 * <ul>
 *   <li><b>CRITICAL</b>: Route to clinical-recommendations-critical topic
 *       (Multi-channel: SMS, Pager, Push, Email, EHR, Dashboard with audio/visual alerts)</li>
 *   <li><b>HIGH</b>: Route to clinical-recommendations-high topic
 *       (Push notifications, Email, EHR, Dashboard with highlighting)</li>
 *   <li><b>MEDIUM</b>: Route to clinical-recommendations-medium topic
 *       (Email, EHR, Dashboard without highlighting)</li>
 *   <li><b>LOW/ROUTINE</b>: Route to clinical-recommendations-routine topic (main output)
 *       (EHR integration, Dashboard silent notification)</li>
 * </ul>
 *
 * <p>Implementation Pattern:
 * Uses Flink's {@link OutputTag} pattern to create side outputs for CRITICAL, HIGH, and MEDIUM
 * priorities, while routing LOW/ROUTINE recommendations to the main output stream.
 *
 * <p>Integration:
 * This router should be applied after {@link com.cardiofit.flink.processors.ClinicalRecommendationProcessor}
 * and before Kafka sink operators.
 *
 * <p>Example Usage:
 * <pre>{@code
 * // Define output tags
 * OutputTag<ClinicalRecommendation> criticalTag = new OutputTag<>("critical"){};
 * OutputTag<ClinicalRecommendation> highTag = new OutputTag<>("high"){};
 * OutputTag<ClinicalRecommendation> mediumTag = new OutputTag<>("medium"){};
 *
 * // Apply router
 * SingleOutputStreamOperator<ClinicalRecommendation> routed = recommendations
 *     .process(new RecommendationRouter(criticalTag, highTag, mediumTag));
 *
 * // Extract side outputs
 * DataStream<ClinicalRecommendation> criticalStream = routed.getSideOutput(criticalTag);
 * DataStream<ClinicalRecommendation> highStream = routed.getSideOutput(highTag);
 * DataStream<ClinicalRecommendation> mediumStream = routed.getSideOutput(mediumTag);
 * DataStream<ClinicalRecommendation> routineStream = routed; // main output
 * }</pre>
 *
 * <p>Downstream Processing:
 * Each output stream should be connected to:
 * <ul>
 *   <li>Kafka sink for the corresponding topic</li>
 *   <li>Notification service sink (for CRITICAL and HIGH priorities)</li>
 *   <li>EMR integration sink (all priorities)</li>
 *   <li>Dashboard WebSocket sink (all priorities with appropriate flags)</li>
 *   <li>Analytics data warehouse sink (all recommendations)</li>
 * </ul>
 *
 * @see com.cardiofit.flink.models.ClinicalRecommendation
 * @see com.cardiofit.flink.processors.ClinicalRecommendationProcessor
 * @author Module 3 Clinical Recommendation Engine
 * @version 1.0
 */
public class RecommendationRouter
    extends ProcessFunction<ClinicalRecommendation, ClinicalRecommendation>
    implements Serializable {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(RecommendationRouter.class);

    // Output tags for side outputs (urgency-based routing)
    private final OutputTag<ClinicalRecommendation> criticalTag;
    private final OutputTag<ClinicalRecommendation> highTag;
    private final OutputTag<ClinicalRecommendation> mediumTag;

    // Routing statistics (for monitoring)
    private transient long criticalCount = 0;
    private transient long highCount = 0;
    private transient long mediumCount = 0;
    private transient long routineCount = 0;

    /**
     * Constructor with output tags for routing
     *
     * @param criticalTag Output tag for CRITICAL priority recommendations
     * @param highTag Output tag for HIGH priority recommendations
     * @param mediumTag Output tag for MEDIUM priority recommendations
     */
    public RecommendationRouter(
            OutputTag<ClinicalRecommendation> criticalTag,
            OutputTag<ClinicalRecommendation> highTag,
            OutputTag<ClinicalRecommendation> mediumTag) {
        this.criticalTag = criticalTag;
        this.highTag = highTag;
        this.mediumTag = mediumTag;
    }

    /**
     * Process element and route to appropriate side output based on urgency
     *
     * @param recommendation Clinical recommendation to route
     * @param ctx Process function context for side outputs
     * @param out Main output collector (for ROUTINE recommendations)
     * @throws Exception if routing fails
     */
    @Override
    public void processElement(
            ClinicalRecommendation recommendation,
            Context ctx,
            Collector<ClinicalRecommendation> out) throws Exception {

        try {
            String urgency = recommendation.getPriority();
            String patientId = recommendation.getPatientId();

            if (urgency == null) {
                LOG.warn("Recommendation {} has null urgency, defaulting to ROUTINE",
                    recommendation.getRecommendationId());
                urgency = "ROUTINE";
            }

            // Route based on urgency level
            switch (urgency.toUpperCase()) {
                case "CRITICAL":
                    routeCritical(recommendation, ctx);
                    criticalCount++;
                    break;

                case "HIGH":
                    routeHigh(recommendation, ctx);
                    highCount++;
                    break;

                case "MEDIUM":
                    routeMedium(recommendation, ctx);
                    mediumCount++;
                    break;

                case "LOW":
                case "ROUTINE":
                    routeRoutine(recommendation, out);
                    routineCount++;
                    break;

                default:
                    LOG.warn("Unknown urgency level '{}' for patient {}, defaulting to MEDIUM",
                        urgency, patientId);
                    routeMedium(recommendation, ctx);
                    mediumCount++;
                    break;
            }

            // Log routing statistics periodically (every 100 recommendations)
            long totalCount = criticalCount + highCount + mediumCount + routineCount;
            if (totalCount % 100 == 0) {
                LOG.info("Routing statistics - CRITICAL: {}, HIGH: {}, MEDIUM: {}, ROUTINE: {} (Total: {})",
                    criticalCount, highCount, mediumCount, routineCount, totalCount);
            }

        } catch (Exception e) {
            LOG.error("Error routing recommendation {} for patient {}: {}",
                recommendation.getRecommendationId(),
                recommendation.getPatientId(),
                e.getMessage(), e);

            // Fail-safe: route to ROUTINE on error to ensure recommendation is not lost
            out.collect(recommendation);
        }
    }

    /**
     * Route CRITICAL priority recommendation to side output
     *
     * <p>CRITICAL recommendations require:
     * <ul>
     *   <li>Multi-channel notifications (SMS, Pager, Push, Email)</li>
     *   <li>EMR integration with STAT priority and interruptive flag</li>
     *   <li>Dashboard with audio/visual alerts and highlighting</li>
     *   <li>Audit logging for critical actions</li>
     *   <li>Real-time escalation to on-call physicians</li>
     * </ul>
     *
     * @param recommendation CRITICAL priority recommendation
     * @param ctx Process function context for side output
     */
    private void routeCritical(ClinicalRecommendation recommendation, Context ctx) {
        ctx.output(criticalTag, recommendation);

        LOG.info("CRITICAL recommendation {} routed for patient {} - Protocol: {}, Actions: {}",
            recommendation.getRecommendationId(),
            recommendation.getPatientId(),
            recommendation.getProtocolId(),
            recommendation.getActions() != null ? recommendation.getActions().size() : 0);

        // Log for downstream multi-channel notification service
        LOG.debug("CRITICAL routing metadata - Patient: {}, Priority: {}, Timestamp: {}",
            recommendation.getPatientId(),
            recommendation.getPriority(),
            recommendation.getTimestamp());
    }

    /**
     * Route HIGH priority recommendation to side output
     *
     * <p>HIGH recommendations require:
     * <ul>
     *   <li>Push notifications and email alerts</li>
     *   <li>EMR integration with URGENT priority (non-interruptive)</li>
     *   <li>Dashboard with visual highlighting (no audio)</li>
     *   <li>Escalation if not acknowledged within 15 minutes</li>
     * </ul>
     *
     * @param recommendation HIGH priority recommendation
     * @param ctx Process function context for side output
     */
    private void routeHigh(ClinicalRecommendation recommendation, Context ctx) {
        ctx.output(highTag, recommendation);

        LOG.info("HIGH recommendation {} routed for patient {} - Protocol: {}",
            recommendation.getRecommendationId(),
            recommendation.getPatientId(),
            recommendation.getProtocolId());

        LOG.debug("HIGH routing metadata - Patient: {}, Actions: {}, Contraindications: {}",
            recommendation.getPatientId(),
            recommendation.getActions() != null ? recommendation.getActions().size() : 0,
            recommendation.getContraindicationsChecked() != null ? recommendation.getContraindicationsChecked().size() : 0);
    }

    /**
     * Route MEDIUM priority recommendation to side output
     *
     * <p>MEDIUM recommendations require:
     * <ul>
     *   <li>Email notifications (non-urgent)</li>
     *   <li>EMR integration with ROUTINE priority</li>
     *   <li>Dashboard notification without highlighting or audio</li>
     *   <li>Review within 2 hours (best effort)</li>
     * </ul>
     *
     * @param recommendation MEDIUM priority recommendation
     * @param ctx Process function context for side output
     */
    private void routeMedium(ClinicalRecommendation recommendation, Context ctx) {
        ctx.output(mediumTag, recommendation);

        LOG.debug("MEDIUM recommendation {} routed for patient {} - Protocol: {}",
            recommendation.getRecommendationId(),
            recommendation.getPatientId(),
            recommendation.getProtocolId());
    }

    /**
     * Route LOW/ROUTINE priority recommendation to main output stream
     *
     * <p>ROUTINE recommendations require:
     * <ul>
     *   <li>EMR integration with ROUTINE priority</li>
     *   <li>Dashboard silent notification</li>
     *   <li>Review during normal workflow</li>
     *   <li>No active notifications sent to clinicians</li>
     * </ul>
     *
     * @param recommendation LOW/ROUTINE priority recommendation
     * @param out Main output collector
     */
    private void routeRoutine(ClinicalRecommendation recommendation, Collector<ClinicalRecommendation> out) {
        out.collect(recommendation);

        LOG.debug("ROUTINE recommendation {} routed for patient {} - Protocol: {}",
            recommendation.getRecommendationId(),
            recommendation.getPatientId(),
            recommendation.getProtocolId());
    }

    /**
     * Get routing statistics summary
     *
     * @return Routing statistics as formatted string
     */
    public String getRoutingStatistics() {
        long total = criticalCount + highCount + mediumCount + routineCount;
        if (total == 0) {
            return "No recommendations routed yet";
        }

        return String.format(
            "Routing Statistics - CRITICAL: %d (%.1f%%), HIGH: %d (%.1f%%), MEDIUM: %d (%.1f%%), ROUTINE: %d (%.1f%%), Total: %d",
            criticalCount, (criticalCount * 100.0 / total),
            highCount, (highCount * 100.0 / total),
            mediumCount, (mediumCount * 100.0 / total),
            routineCount, (routineCount * 100.0 / total),
            total
        );
    }

    /**
     * Reset routing statistics (useful for testing)
     */
    public void resetStatistics() {
        criticalCount = 0;
        highCount = 0;
        mediumCount = 0;
        routineCount = 0;
    }
}
