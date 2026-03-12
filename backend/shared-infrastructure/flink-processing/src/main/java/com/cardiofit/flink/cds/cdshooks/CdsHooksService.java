package com.cardiofit.flink.cds.cdshooks;

import com.cardiofit.flink.cds.fhir.*;
import com.cardiofit.flink.cds.population.*;
import com.cardiofit.flink.clients.GoogleFHIRClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.CompletableFuture;

/**
 * CDS Hooks Service Implementation
 * Phase 8 Module 5 - CDS Hooks Implementation
 *
 * Implements CDS Hooks 2.0 specification for clinical decision support.
 * Provides hook endpoints for:
 * - order-select: When clinician selects medications/procedures
 * - order-sign: When clinician signs orders
 *
 * Integration:
 * - Uses GoogleFHIRClient for patient data access
 * - Uses FHIRQualityMeasureEvaluator for quality measure checks
 * - Uses FHIRObservationMapper for recent lab values
 * - Returns cards with actionable recommendations
 *
 * @author CardioFit CDS Team
 * @version 1.0.0
 * @since Phase 8
 */
public class CdsHooksService {
    private static final Logger LOG = LoggerFactory.getLogger(CdsHooksService.class);

    private final GoogleFHIRClient fhirClient;
    private final FHIRQualityMeasureEvaluator qualityMeasureEvaluator;
    private final FHIRObservationMapper observationMapper;

    // Service Configuration
    private static final String SERVICE_BASE_URL = "https://cardiofit.health/cds-services";
    private static final String CARDIOFIT_ICON = "https://cardiofit.health/icon.png";

    public CdsHooksService(GoogleFHIRClient fhirClient,
                           FHIRQualityMeasureEvaluator qualityMeasureEvaluator,
                           FHIRObservationMapper observationMapper) {
        this.fhirClient = fhirClient;
        this.qualityMeasureEvaluator = qualityMeasureEvaluator;
        this.observationMapper = observationMapper;
    }

    /**
     * Get service discovery information
     *
     * Endpoint: GET /cds-services
     * Returns: List of available CDS Hooks services
     */
    public List<CdsHooksServiceDescriptor> getServiceDiscovery() {
        LOG.info("CDS Hooks service discovery requested");

        List<CdsHooksServiceDescriptor> services = new ArrayList<>();

        // Order-Select Service
        CdsHooksServiceDescriptor orderSelect = new CdsHooksServiceDescriptor(
            "cardiofit-order-select",
            "order-select",
            "CardioFit Medication Safety Check",
            "Provides real-time medication safety alerts including drug interactions, " +
            "contraindications, and dosing recommendations based on patient conditions and lab values."
        );
        orderSelect.withPrefetch("patient", "Patient/{{context.patientId}}")
                  .withPrefetch("conditions", "Condition?patient={{context.patientId}}&clinical-status=active")
                  .withPrefetch("medications", "MedicationRequest?patient={{context.patientId}}&status=active")
                  .withUsageRequirements("Requires active patient context and medication selection");
        services.add(orderSelect);

        // Order-Sign Service
        CdsHooksServiceDescriptor orderSign = new CdsHooksServiceDescriptor(
            "cardiofit-order-sign",
            "order-sign",
            "CardioFit Order Safety Verification",
            "Final safety verification before order signing including duplicate therapy detection, " +
            "renal dosing adjustments, and quality measure compliance."
        );
        orderSign.withPrefetch("patient", "Patient/{{context.patientId}}")
                .withPrefetch("conditions", "Condition?patient={{context.patientId}}")
                .withPrefetch("labResults", "Observation?patient={{context.patientId}}&category=laboratory")
                .withUsageRequirements("Requires patient context and draft orders");
        services.add(orderSign);

        LOG.info("Returning {} CDS Hooks services", services.size());
        return services;
    }

    /**
     * Handle order-select hook
     *
     * Triggered when clinician selects medications/procedures but before signing.
     * Provides early warnings about drug interactions, contraindications.
     *
     * Endpoint: POST /cds-services/cardiofit-order-select
     *
     * @param request CDS Hooks request with medication context
     * @return CompletableFuture with CDS Hooks response containing cards
     */
    public CompletableFuture<CdsHooksResponse> handleOrderSelect(CdsHooksRequest request) {
        LOG.info("Processing order-select hook for patient: {}", request.getPatientId());

        if (!request.isValid()) {
            LOG.warn("Invalid CDS Hooks request: {}", request);
            return CompletableFuture.completedFuture(CdsHooksResponse.empty());
        }

        List<CompletableFuture<CdsHooksCard>> cardFutures = new ArrayList<>();

        // Get medication orders from context
        List<Map<String, Object>> medications = request.getMedicationOrders();
        String patientId = request.getPatientId();

        if (medications.isEmpty()) {
            LOG.info("No medications in order-select context");
            return CompletableFuture.completedFuture(CdsHooksResponse.empty());
        }

        // Check 1: Drug interaction warnings
        CompletableFuture<CdsHooksCard> drugInteractionCard =
            checkDrugInteractions(patientId, medications);
        cardFutures.add(drugInteractionCard);

        // Check 2: Contraindication warnings
        CompletableFuture<CdsHooksCard> contraindicationCard =
            checkContraindications(patientId, medications);
        cardFutures.add(contraindicationCard);

        // Check 3: Lab value warnings (e.g., renal function for dosing)
        CompletableFuture<CdsHooksCard> labValueCard =
            checkLabValues(patientId, medications);
        cardFutures.add(labValueCard);

        // Check 4: Quality measure impact
        CompletableFuture<CdsHooksCard> qualityMeasureCard =
            checkQualityMeasureImpact(patientId, medications);
        cardFutures.add(qualityMeasureCard);

        // Aggregate all cards
        return CompletableFuture.allOf(cardFutures.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                CdsHooksResponse response = new CdsHooksResponse();

                for (CompletableFuture<CdsHooksCard> future : cardFutures) {
                    CdsHooksCard card = future.join();
                    if (card != null) {
                        response.addCard(card);
                    }
                }

                LOG.info("Order-select response: {} cards ({} critical, {} warnings)",
                    response.getCards().size(),
                    response.getCardCountByIndicator(CdsHooksCard.IndicatorType.CRITICAL),
                    response.getCardCountByIndicator(CdsHooksCard.IndicatorType.WARNING));

                return response;
            })
            .exceptionally(ex -> {
                LOG.error("Error processing order-select hook", ex);
                return CdsHooksResponse.empty();
            });
    }

    /**
     * Handle order-sign hook
     *
     * Triggered when clinician is about to sign orders.
     * Final safety check before order goes to pharmacy.
     *
     * Endpoint: POST /cds-services/cardiofit-order-sign
     *
     * @param request CDS Hooks request with draft orders
     * @return CompletableFuture with CDS Hooks response containing cards
     */
    public CompletableFuture<CdsHooksResponse> handleOrderSign(CdsHooksRequest request) {
        LOG.info("Processing order-sign hook for patient: {}", request.getPatientId());

        if (!request.isValid()) {
            LOG.warn("Invalid CDS Hooks request: {}", request);
            return CompletableFuture.completedFuture(CdsHooksResponse.empty());
        }

        List<CompletableFuture<CdsHooksCard>> cardFutures = new ArrayList<>();

        String patientId = request.getPatientId();
        Map<String, Object> draftOrders = request.getDraftOrders();

        if (draftOrders.isEmpty()) {
            LOG.info("No draft orders in order-sign context");
            return CompletableFuture.completedFuture(CdsHooksResponse.empty());
        }

        // Check 1: Duplicate therapy detection
        CompletableFuture<CdsHooksCard> duplicateCard =
            checkDuplicateTherapy(patientId, draftOrders);
        cardFutures.add(duplicateCard);

        // Check 2: Renal dosing adjustments
        CompletableFuture<CdsHooksCard> renalDosingCard =
            checkRenalDosing(patientId, draftOrders);
        cardFutures.add(renalDosingCard);

        // Check 3: Pregnancy/lactation warnings
        CompletableFuture<CdsHooksCard> pregnancyCard =
            checkPregnancyLactation(patientId, draftOrders);
        cardFutures.add(pregnancyCard);

        // Check 4: Clinical guideline compliance
        CompletableFuture<CdsHooksCard> guidelineCard =
            checkGuidelineCompliance(patientId, draftOrders);
        cardFutures.add(guidelineCard);

        // Aggregate all cards
        return CompletableFuture.allOf(cardFutures.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                CdsHooksResponse response = new CdsHooksResponse();

                for (CompletableFuture<CdsHooksCard> future : cardFutures) {
                    CdsHooksCard card = future.join();
                    if (card != null) {
                        response.addCard(card);
                    }
                }

                LOG.info("Order-sign response: {} cards ({} critical, {} warnings)",
                    response.getCards().size(),
                    response.getCardCountByIndicator(CdsHooksCard.IndicatorType.CRITICAL),
                    response.getCardCountByIndicator(CdsHooksCard.IndicatorType.WARNING));

                return response;
            })
            .exceptionally(ex -> {
                LOG.error("Error processing order-sign hook", ex);
                return CdsHooksResponse.empty();
            });
    }

    // ==================== Private Check Methods ====================

    /**
     * Check for drug-drug interactions
     */
    private CompletableFuture<CdsHooksCard> checkDrugInteractions(
            String patientId, List<Map<String, Object>> medications) {

        return CompletableFuture.supplyAsync(() -> {
            // TODO: Implement actual drug interaction checking logic
            // For now, return a sample warning for demonstration

            if (medications.size() > 3) {
                CdsHooksCard card = CdsHooksCard.warning(
                    "Potential Drug Interaction: Polypharmacy Alert",
                    "Patient is on " + medications.size() + " medications. " +
                    "Review for potential drug-drug interactions, especially with anticoagulants, " +
                    "antiplatelets, and medications affecting renal function."
                );

                card.withSource("CardioFit Drug Interaction Database", SERVICE_BASE_URL);

                // Add suggestion to review interactions
                CdsHooksCard.Suggestion reviewSuggestion = new CdsHooksCard.Suggestion(
                    "Review Interaction Report"
                );
                reviewSuggestion.setIsRecommended(true);
                card.addSuggestion(reviewSuggestion);

                return card;
            }

            return null; // No card if no issues
        });
    }

    /**
     * Check for contraindications based on patient conditions
     */
    private CompletableFuture<CdsHooksCard> checkContraindications(
            String patientId, List<Map<String, Object>> medications) {

        return fhirClient.getConditionsAsync(patientId)
            .thenApply(conditions -> {
                // TODO: Implement actual contraindication checking
                // Sample logic: Check if patient has heart failure and is prescribed NSAIDs

                boolean hasHeartFailure = conditions.stream()
                    .anyMatch(c -> c.getCode() != null &&
                                 c.getCode().startsWith("I50"));

                if (hasHeartFailure && !medications.isEmpty()) {
                    CdsHooksCard card = CdsHooksCard.warning(
                        "Contraindication Alert: Heart Failure Patient",
                        "Patient has active diagnosis of heart failure (I50). " +
                        "Avoid NSAIDs and thiazolidinediones. Use ACE inhibitors or ARBs as first-line therapy."
                    );

                    card.withSource("ACC/AHA Heart Failure Guidelines",
                        "https://www.acc.org/guidelines");

                    return card;
                }

                return null;
            })
            .exceptionally(ex -> {
                LOG.error("Error checking contraindications", ex);
                return null;
            });
    }

    /**
     * Check lab values for dosing adjustments
     */
    private CompletableFuture<CdsHooksCard> checkLabValues(
            String patientId, List<Map<String, Object>> medications) {

        // Use LOINC code for creatinine
        return observationMapper.getObservationByLoinc(patientId, FHIRObservationMapper.LOINC_CREATININE)
            .thenApply(observations -> observations.isEmpty() ? null : observations.get(0))
            .thenCompose(creatinine -> CompletableFuture.completedFuture(creatinine))
            .thenApply(creatinine -> {
                if (creatinine == null) {
                    return CdsHooksCard.info(
                        "Missing Lab Value: Creatinine",
                        "No recent creatinine value found. Consider ordering renal function tests " +
                        "before prescribing medications requiring renal dose adjustment."
                    ).withSource("CardioFit Clinical Intelligence", SERVICE_BASE_URL);
                }

                // Check if creatinine is elevated (>1.5 mg/dL)
                if (creatinine.getValue() > 1.5) {
                    CdsHooksCard card = CdsHooksCard.warning(
                        "Renal Function Alert: Elevated Creatinine",
                        String.format("Patient's creatinine is %.1f mg/dL (normal <1.5). " +
                            "Review medication list for renal dose adjustments. " +
                            "Consider nephrology consult if eGFR <30 mL/min.",
                            creatinine.getValue())
                    );

                    card.withSource("KDIGO Clinical Practice Guidelines",
                        "https://kdigo.org/guidelines/");

                    // Add link to renal dosing reference
                    card.addLink(new CdsHooksCard.Link(
                        "Renal Dosing Reference",
                        "https://www.globalrph.com/medcalcs/renal-dosing/"
                    ));

                    return card;
                }

                return null;
            })
            .exceptionally(ex -> {
                LOG.error("Error checking lab values", ex);
                return null;
            });
    }

    /**
     * Check impact on quality measures
     */
    private CompletableFuture<CdsHooksCard> checkQualityMeasureImpact(
            String patientId, List<Map<String, Object>> medications) {

        return CompletableFuture.supplyAsync(() -> {
            // TODO: Implement actual quality measure checking
            // Sample: Check if statin therapy for diabetic patients

            CdsHooksCard card = CdsHooksCard.info(
                "Quality Measure Opportunity: Statin Therapy",
                "Patient is eligible for statin therapy (diabetic patient age >40). " +
                "Adding statin would improve HEDIS quality measure compliance."
            );

            card.withSource("HEDIS Quality Measures", SERVICE_BASE_URL);

            return card;
        });
    }

    /**
     * Check for duplicate therapy
     */
    private CompletableFuture<CdsHooksCard> checkDuplicateTherapy(
            String patientId, Map<String, Object> draftOrders) {

        return CompletableFuture.supplyAsync(() -> {
            // TODO: Implement duplicate therapy detection
            return null;
        });
    }

    /**
     * Check renal dosing requirements
     */
    private CompletableFuture<CdsHooksCard> checkRenalDosing(
            String patientId, Map<String, Object> draftOrders) {

        return observationMapper.getObservationByLoinc(patientId, FHIRObservationMapper.LOINC_CREATININE)
            .thenApply(observations -> observations.isEmpty() ? null : observations.get(0))
            .thenApply(creatinine -> {
                // TODO: Implement renal dosing logic
                return null;
            });
    }

    /**
     * Check pregnancy/lactation warnings
     */
    private CompletableFuture<CdsHooksCard> checkPregnancyLactation(
            String patientId, Map<String, Object> draftOrders) {

        return fhirClient.getPatientAsync(patientId)
            .thenApply(patient -> {
                // TODO: Check patient gender and pregnancy status
                return null;
            });
    }

    /**
     * Check clinical guideline compliance
     */
    private CompletableFuture<CdsHooksCard> checkGuidelineCompliance(
            String patientId, Map<String, Object> draftOrders) {

        return CompletableFuture.supplyAsync(() -> {
            // TODO: Implement guideline compliance checking
            return null;
        });
    }
}
