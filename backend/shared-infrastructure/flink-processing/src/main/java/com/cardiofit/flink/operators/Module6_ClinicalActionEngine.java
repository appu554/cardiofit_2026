package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.routing.NotificationRouter;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
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
 * Consumes unified ClinicalEvents (from CDS, Pattern, ML sources),
 * classifies into HALT/PAUSE/SOFT_FLAG/ROUTINE, deduplicates across modules,
 * manages alert lifecycle with SLA escalation timers, and distributes
 * output to multiple Kafka sinks via side-output tags.
 */
public class Module6_ClinicalActionEngine
        extends KeyedProcessFunction<String, ClinicalEvent, ClinicalAction> {

    private static final long serialVersionUID = 1L;
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

        LOG.info("Module6_ClinicalActionEngine initialized");
    }

    @Override
    public void processElement(ClinicalEvent event, Context ctx,
                                Collector<ClinicalAction> out) throws Exception {

        String patientId = event.getPatientId();
        if (patientId == null || patientId.isEmpty()) {
            LOG.warn("Dropping event with null patientId");
            return;
        }

        // 1. Classify
        ActionTier tier = Module6ActionClassifier.classify(event);
        if (tier == ActionTier.ROUTINE) return; // no action needed

        // 2. Initialize state
        PatientAlertState patState = alertState.value();
        if (patState == null) patState = new PatientAlertState(patientId);

        Module6CrossModuleDedup dedup = dedupState.value();
        if (dedup == null) dedup = new Module6CrossModuleDedup();

        // 3. Alert fatigue check
        if (AlertLifecycleManager.checkAlertFatigue(patState)) {
            alertState.update(patState);
            return; // suppress — fatigue threshold hit
        }

        // 4. Cross-module dedup
        String clinicalCategory = event.getClinicalCategory();
        if (!dedup.shouldEmit(patientId, tier, clinicalCategory, event.getEventTime())) {
            alertState.update(patState);
            dedupState.update(dedup);
            return; // suppressed by dedup
        }

        // 5. Create alert
        String sourceModule = switch (event.getSource()) {
            case CDS -> "MODULE_3_CDS";
            case PATTERN -> "MODULE_4_CEP";
            case ML_PREDICTION -> "MODULE_5_ML";
        };
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            patientId, tier, clinicalCategory, sourceModule);

        // 6. Register escalation timer if needed
        if (tier.requiresEscalation()) {
            ctx.timerService().registerProcessingTimeTimer(alert.getSlaDeadlineMs());
        }

        // 7. Store in active alerts
        patState.getActiveAlerts().put(clinicalCategory, alert);

        // 8. Emit main output
        out.collect(ClinicalAction.newAlert(alert));

        // 9. Emit notifications via side output
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
            ctx.output(NOTIFICATION_TAG, notif);
        }

        // 10. Emit audit record
        AuditRecord audit = AuditRecord.alertCreated(alert, event);
        ctx.output(AUDIT_TAG, audit);

        // 11. Update state
        alertState.update(patState);
        dedupState.update(dedup);

        LOG.info("Module6 {} alert: patient={}, category={}, source={}",
            tier, patientId, clinicalCategory, sourceModule);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<ClinicalAction> out) throws Exception {

        PatientAlertState patState = alertState.value();
        if (patState == null) return;

        // Find the alert whose SLA deadline matches this timer
        for (ClinicalAlert alert : patState.getActiveAlerts().values()) {
            if (alert.getState() == AlertState.ACTIVE
                    && alert.getSlaDeadlineMs() == timestamp) {

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
                        alert.setSlaDeadlineMs(nextEscalation);
                    }
                }
                break;
            }
        }
        alertState.update(patState);
    }
}
