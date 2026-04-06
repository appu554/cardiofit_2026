package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.routing.NotificationRouter;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.streaming.api.functions.co.KeyedCoProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.List;
import java.util.UUID;

/**
 * Module 6: Clinical Action Engine — main operator.
 *
 * Dual-input KeyedCoProcessFunction:
 *   Input 1 (processElement1): ClinicalEvent from Modules 3/4/5
 *   Input 2 (processElement2): AlertAcknowledgment from physicians via alert-acknowledgments.v1
 *
 * Classifies events into HALT/PAUSE/SOFT_FLAG/ROUTINE, deduplicates across modules,
 * manages alert lifecycle with SLA escalation timers, processes physician acknowledgments
 * to cancel pending escalations, and distributes output to multiple Kafka sinks.
 */
public class Module6_ClinicalActionEngine
        extends KeyedCoProcessFunction<String, ClinicalEvent, AlertAcknowledgment, ClinicalAction> {

    private static final long serialVersionUID = 2L;
    private static final Logger LOG = LoggerFactory.getLogger(Module6_ClinicalActionEngine.class);

    // ── Side-output tags ──
    public static final OutputTag<NotificationRequest> NOTIFICATION_TAG =
        new OutputTag<>("notifications", TypeInformation.of(NotificationRequest.class));
    public static final OutputTag<AuditRecord> AUDIT_TAG =
        new OutputTag<>("audit", TypeInformation.of(AuditRecord.class));
    public static final OutputTag<FhirWriteRequest> FHIR_TAG =
        new OutputTag<>("fhir-writeback", TypeInformation.of(FhirWriteRequest.class));

    // ── State ──
    private transient ValueState<PatientAlertState> alertState;
    private transient ValueState<Module6CrossModuleDedup> dedupState;

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        // Patient alert state with 7-day TTL
        ValueStateDescriptor<PatientAlertState> alertDescriptor =
            new ValueStateDescriptor<>("patient-alert-state", PatientAlertState.class);
        StateTtlConfig ttlConfig = StateTtlConfig
            .newBuilder(Duration.ofDays(7))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        alertDescriptor.enableTimeToLive(ttlConfig);
        alertState = getRuntimeContext().getState(alertDescriptor);

        // Dedup state with 7-day TTL
        ValueStateDescriptor<Module6CrossModuleDedup> dedupDescriptor =
            new ValueStateDescriptor<>("dedup-state", Module6CrossModuleDedup.class);
        dedupDescriptor.enableTimeToLive(ttlConfig);
        dedupState = getRuntimeContext().getState(dedupDescriptor);

        LOG.info("Module6_ClinicalActionEngine initialized (dual-input: ClinicalEvent + AlertAcknowledgment)");
    }

    // ═══════════════════════════════════════════════════════════════════════════
    // Input 1: ClinicalEvent (from Modules 3/4/5)
    // ═══════════════════════════════════════════════════════════════════════════

    @Override
    public void processElement1(ClinicalEvent event, Context ctx,
                                Collector<ClinicalAction> out) throws Exception {

        String patientId = event.getPatientId();
        if (patientId == null || patientId.isEmpty()) {
            LOG.warn("Dropping event with null patientId");
            return;
        }

        // 1. Classify
        ActionTier tier = Module6ActionClassifier.classify(event);

        // 2. Initialize state
        PatientAlertState patState = alertState.value();
        if (patState == null) patState = new PatientAlertState(patientId);

        Module6CrossModuleDedup dedup = dedupState.value();
        if (dedup == null) dedup = new Module6CrossModuleDedup();

        // 2b. Prune expired dedup entries to bound memory growth
        dedup.pruneExpired(event.getEventTime());

        String clinicalCategory = event.getClinicalCategory();
        String sourceModule = switch (event.getSource()) {
            case CDS -> "MODULE_3_CDS";
            case PATTERN -> "MODULE_4_CEP";
            case ML_PREDICTION -> "MODULE_5_ML";
        };

        // 3. ROUTINE: emit audit record only (no alert, no notification)
        if (tier == ActionTier.ROUTINE) {
            AuditRecord routineAudit = new AuditRecord();
            routineAudit.setAuditId(UUID.randomUUID().toString());
            routineAudit.setEventType("ROUTINE_PROCESSED");
            routineAudit.setPatientId(patientId);
            routineAudit.setSourceModule(sourceModule);
            routineAudit.setTier(ActionTier.ROUTINE);
            routineAudit.setClinicalCategory(clinicalCategory);
            ctx.output(AUDIT_TAG, routineAudit);
            return;
        }

        // 4. Alert fatigue check (HALT always bypasses)
        if (AlertLifecycleManager.checkAlertFatigue(patState, tier)) {
            alertState.update(patState);
            return; // suppress — fatigue threshold hit
        }

        // 5. Cross-module dedup (keyed by category only)
        if (!dedup.shouldEmit(patientId, tier, clinicalCategory, event.getEventTime())) {
            alertState.update(patState);
            dedupState.update(dedup);
            return; // suppressed by dedup
        }

        // 6. Create alert (using event time, not wall clock)
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            patientId, tier, clinicalCategory, sourceModule, event.getEventTime());

        // 7. Register escalation timer and record mapping for lookup
        if (tier.requiresEscalation()) {
            long deadline = alert.getSlaDeadlineMs();
            ctx.timerService().registerProcessingTimeTimer(deadline);
            patState.registerTimerMapping(deadline, alert.getAlertId());
        }

        // 8. Store in active alerts
        patState.getActiveAlerts().put(alert.getAlertId(), alert);

        // 9. Emit main output
        out.collect(ClinicalAction.newAlert(alert));

        // 10. Emit notifications via side output
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        for (NotificationRequest.Channel channel : channels) {
            NotificationRequest notif = new NotificationRequest();
            notif.setNotificationId(UUID.randomUUID().toString());
            notif.setAlertId(alert.getAlertId());
            notif.setPatientId(patientId);
            notif.setChannel(channel);
            notif.setTier(tier);
            notif.setTitle(tier + ": " + clinicalCategory);
            notif.setRequiresAcknowledgment(tier == ActionTier.HALT);
            // PIPE-4: Deterministic idempotency key for dedup on Flink restart
            notif.setIdempotencyKey(NotificationRequest.computeIdempotencyKey(
                    patientId, alert.getAlertId(), channel));
            ctx.output(NOTIFICATION_TAG, notif);
        }

        // 11. Emit audit record
        AuditRecord audit = AuditRecord.alertCreated(alert, event);
        ctx.output(AUDIT_TAG, audit);

        // 12. Update state
        alertState.update(patState);
        dedupState.update(dedup);

        LOG.info("Module6 {} alert: patient={}, category={}, source={}",
            tier, patientId, clinicalCategory, sourceModule);
    }

    // ═══════════════════════════════════════════════════════════════════════════
    // Input 2: AlertAcknowledgment (from physicians)
    // ═══════════════════════════════════════════════════════════════════════════

    @Override
    public void processElement2(AlertAcknowledgment ack, Context ctx,
                                Collector<ClinicalAction> out) throws Exception {

        String alertId = ack.getAlertId();
        if (alertId == null || alertId.isEmpty()) {
            LOG.warn("Dropping acknowledgment with null alertId");
            return;
        }

        PatientAlertState patState = alertState.value();
        if (patState == null) {
            LOG.warn("Acknowledgment for unknown patient state, alertId={}", alertId);
            return;
        }

        ClinicalAlert alert = patState.getActiveAlerts().get(alertId);
        if (alert == null) {
            LOG.warn("Acknowledgment for unknown alertId={}", alertId);
            return;
        }

        // Determine target state from acknowledgment action
        AlertState targetState = switch (ack.getAction()) {
            case ACKNOWLEDGE -> AlertState.ACKNOWLEDGED;
            case ACTION_TAKEN -> AlertState.ACTIONED;
            case DISMISS -> alert.getState() == AlertState.ACTIVE
                ? AlertState.AUTO_RESOLVED : AlertState.RESOLVED;
        };

        // Validate state transition
        if (!alert.getState().canTransitionTo(targetState)) {
            LOG.warn("Invalid state transition for alertId={}: {} → {}",
                alertId, alert.getState(), targetState);
            return;
        }

        // Apply state transition
        alert.setState(targetState);
        alert.setAcknowledgedBy(ack.getPractitionerId());

        if (targetState == AlertState.ACKNOWLEDGED) {
            alert.setAcknowledgedAt(ack.getTimestamp());
        } else if (targetState == AlertState.ACTIONED) {
            alert.setActionedAt(ack.getTimestamp());
            alert.setActionDescription(ack.getActionDescription());
        } else if (targetState == AlertState.RESOLVED) {
            alert.setResolvedAt(ack.getTimestamp());
        }

        // Remove from active alerts if terminal state
        if (targetState.isTerminal()) {
            patState.getActiveAlerts().remove(alertId);
        }

        // Emit audit record for the acknowledgment
        AuditRecord audit = new AuditRecord();
        audit.setAuditId(UUID.randomUUID().toString());
        audit.setEventType("ALERT_" + targetState.name());
        audit.setEventDescription("Physician " + ack.getPractitionerId()
            + " " + ack.getAction().name().toLowerCase() + " alert " + alertId);
        audit.setPatientId(ack.getPatientId());
        audit.setAlertId(alertId);
        audit.setTier(alert.getTier());
        audit.setClinicalCategory(alert.getClinicalCategory());
        ctx.output(AUDIT_TAG, audit);

        // Emit ClinicalAction for downstream consumers
        ClinicalAction action = new ClinicalAction();
        action.setActionId(UUID.randomUUID().toString());
        action.setActionTypeStr("ACKNOWLEDGMENT");
        action.setAlert(alert);
        action.setTimestamp(ack.getTimestamp());
        out.collect(action);

        alertState.update(patState);

        LOG.info("Module6 acknowledgment: alertId={}, action={}, by={}",
            alertId, ack.getAction(), ack.getPractitionerId());
    }

    // ═══════════════════════════════════════════════════════════════════════════
    // Timer: SLA escalation
    // ═══════════════════════════════════════════════════════════════════════════

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<ClinicalAction> out) throws Exception {

        PatientAlertState patState = alertState.value();
        if (patState == null) return;

        // Lookup alert by timer→alertId mapping
        String alertId = patState.popTimerMapping(timestamp);
        if (alertId == null) return;

        ClinicalAlert alert = patState.getActiveAlerts().get(alertId);
        if (alert == null || alert.getState() != AlertState.ACTIVE) return;

        AlertLifecycleManager.escalate(alert);
        out.collect(ClinicalAction.escalation(alert, alert.getAssignedTo()));

        // Emit audit for escalation
        AuditRecord audit = new AuditRecord();
        audit.setAuditId(UUID.randomUUID().toString());
        audit.setEventType("ALERT_ESCALATED");
        audit.setPatientId(alert.getPatientId());
        audit.setAlertId(alert.getAlertId());
        audit.setTier(alert.getTier());
        audit.setClinicalCategory(alert.getClinicalCategory());
        ctx.output(AUDIT_TAG, audit);

        // Register next escalation if under max level
        if (alert.getEscalationLevel() < 3 && alert.getTier().requiresEscalation()) {
            long nextEscalation = switch (alert.getTier()) {
                case HALT -> timestamp + (90 * 60 * 1000L);     // +90 min
                case PAUSE -> timestamp + (48 * 60 * 60 * 1000L); // +48 hr
                default -> -1L;
            };
            if (nextEscalation > 0) {
                ctx.timerService().registerProcessingTimeTimer(nextEscalation);
                patState.registerTimerMapping(nextEscalation, alertId);
                alert.setSlaDeadlineMs(nextEscalation);
            }
        }

        alertState.update(patState);
    }
}
