package com.cardiofit.flink.test;

import com.cardiofit.flink.cdc.*;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * CDC Consumer Test Job
 *
 * Tests deserialization of CDC events from all KB services.
 * Consumes CDC topics and logs parsed events for verification.
 *
 * Usage:
 * flink run --class com.cardiofit.flink.test.CDCConsumerTest \
 *   target/flink-ehr-intelligence-1.0.0.jar
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
public class CDCConsumerTest {
    private static final Logger LOG = LoggerFactory.getLogger(CDCConsumerTest.class);

    private static final String KAFKA_BROKERS = "localhost:9092";

    public static void main(String[] args) throws Exception {
        final StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(1); // Single parallelism for testing

        LOG.info("=== CDC Consumer Test Started ===");
        LOG.info("Testing CDC event deserialization from all KB services...");

        // Test 1: KB3 Clinical Protocols
        testProtocolCDC(env);

        // Test 2: KB2 Clinical Phenotypes
        testPhenotypeCDC(env);

        // Test 3: KB1 Drug Rules
        testDrugRuleCDC(env);

        // Test 4: KB5 Drug Interactions
        testDrugInteractionCDC(env);

        // Test 5: KB6 Formulary Drugs
        testFormularyCDC(env);

        // Test 6: KB7 Terminology
        testTerminologyCDC(env);

        env.execute("CDC Consumer Test - All KB Services");
    }

    /**
     * Test KB3 Protocol CDC consumption
     */
    private static void testProtocolCDC(StreamExecutionEnvironment env) {
        LOG.info("Configuring KB3 Protocol CDC consumer...");

        KafkaSource<ProtocolCDCEvent> source = KafkaSource.<ProtocolCDCEvent>builder()
                .setBootstrapServers(KAFKA_BROKERS)
                .setTopics("kb3.clinical_protocols.changes")
                .setGroupId("cdc-test-kb3-protocols")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forProtocol())
                .build();

        DataStream<ProtocolCDCEvent> stream = env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "KB3 Protocol CDC Source"
        );

        stream.process(new ProcessFunction<ProtocolCDCEvent, String>() {
            @Override
            public void processElement(
                    ProtocolCDCEvent cdc,
                    Context ctx,
                    Collector<String> out
            ) throws Exception {
                LOG.info("=== KB3 PROTOCOL CDC EVENT ===");
                LOG.info("Operation: {}", cdc.getPayload().getOperation());
                LOG.info("Timestamp: {}", cdc.getPayload().getTimestampMs());

                if (cdc.getPayload().isDelete()) {
                    ProtocolCDCEvent.ProtocolData before = cdc.getPayload().getBefore();
                    LOG.info("DELETED Protocol: {} v{}", before.getProtocolId(), before.getVersion());
                } else {
                    ProtocolCDCEvent.ProtocolData after = cdc.getPayload().getAfter();
                    LOG.info("Protocol ID: {}", after.getProtocolId());
                    LOG.info("Name: {}", after.getName());
                    LOG.info("Category: {}", after.getCategory());
                    LOG.info("Version: {}", after.getVersion());
                    LOG.info("Specialty: {}", after.getSpecialty());
                }

                LOG.info("Source: {}.{}", cdc.getPayload().getSource().getDatabase(),
                        cdc.getPayload().getSource().getTable());
                LOG.info("=============================");

                out.collect("KB3-PROTOCOL-OK");
            }
        }).name("KB3 Protocol Processor");
    }

    /**
     * Test KB2 Phenotype CDC consumption
     */
    private static void testPhenotypeCDC(StreamExecutionEnvironment env) {
        LOG.info("Configuring KB2 Phenotype CDC consumer...");

        KafkaSource<ClinicalPhenotypeCDCEvent> source = KafkaSource.<ClinicalPhenotypeCDCEvent>builder()
                .setBootstrapServers(KAFKA_BROKERS)
                .setTopics("kb2.clinical_phenotypes.changes")
                .setGroupId("cdc-test-kb2-phenotypes")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forPhenotype())
                .build();

        DataStream<ClinicalPhenotypeCDCEvent> stream = env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "KB2 Phenotype CDC Source"
        );

        stream.process(new ProcessFunction<ClinicalPhenotypeCDCEvent, String>() {
            @Override
            public void processElement(
                    ClinicalPhenotypeCDCEvent cdc,
                    Context ctx,
                    Collector<String> out
            ) throws Exception {
                LOG.info("=== KB2 PHENOTYPE CDC EVENT ===");
                LOG.info("Operation: {}", cdc.getPayload().getOperation());

                if (!cdc.getPayload().isDelete()) {
                    ClinicalPhenotypeCDCEvent.PhenotypeData data = cdc.getPayload().getAfter();
                    LOG.info("Phenotype ID: {}", data.getPhenotypeId());
                    LOG.info("Name: {}", data.getName());
                    LOG.info("Priority: {}", data.getPriority());
                }
                LOG.info("===============================");

                out.collect("KB2-PHENOTYPE-OK");
            }
        }).name("KB2 Phenotype Processor");
    }

    /**
     * Test KB1 Drug Rule CDC consumption
     */
    private static void testDrugRuleCDC(StreamExecutionEnvironment env) {
        LOG.info("Configuring KB1 Drug Rule CDC consumer...");

        KafkaSource<DrugRuleCDCEvent> source = KafkaSource.<DrugRuleCDCEvent>builder()
                .setBootstrapServers(KAFKA_BROKERS)
                .setTopics("kb1.drug_rule_packs.changes", "kb1.dose_calculations.changes")
                .setGroupId("cdc-test-kb1-drug-rules")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forDrugRule())
                .build();

        DataStream<DrugRuleCDCEvent> stream = env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "KB1 Drug Rule CDC Source"
        );

        stream.process(new ProcessFunction<DrugRuleCDCEvent, String>() {
            @Override
            public void processElement(
                    DrugRuleCDCEvent cdc,
                    Context ctx,
                    Collector<String> out
            ) throws Exception {
                LOG.info("=== KB1 DRUG RULE CDC EVENT ===");
                LOG.info("Operation: {}", cdc.getPayload().getOperation());

                if (!cdc.getPayload().isDelete()) {
                    DrugRuleCDCEvent.DrugRuleData data = cdc.getPayload().getAfter();
                    LOG.info("Drug ID: {}", data.getDrugId());
                    LOG.info("Version: {}", data.getVersion());
                    LOG.info("Signature Valid: {}", data.getSignatureValid());
                }
                LOG.info("==============================");

                out.collect("KB1-DRUG-RULE-OK");
            }
        }).name("KB1 Drug Rule Processor");
    }

    /**
     * Test KB5 Drug Interaction CDC consumption
     */
    private static void testDrugInteractionCDC(StreamExecutionEnvironment env) {
        LOG.info("Configuring KB5 Drug Interaction CDC consumer...");

        KafkaSource<DrugInteractionCDCEvent> source = KafkaSource.<DrugInteractionCDCEvent>builder()
                .setBootstrapServers(KAFKA_BROKERS)
                .setTopics("kb5.drug_interactions.changes")
                .setGroupId("cdc-test-kb5-interactions")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forDrugInteraction())
                .build();

        DataStream<DrugInteractionCDCEvent> stream = env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "KB5 Interaction CDC Source"
        );

        stream.process(new ProcessFunction<DrugInteractionCDCEvent, String>() {
            @Override
            public void processElement(
                    DrugInteractionCDCEvent cdc,
                    Context ctx,
                    Collector<String> out
            ) throws Exception {
                LOG.info("=== KB5 DRUG INTERACTION CDC EVENT ===");
                LOG.info("Operation: {}", cdc.getPayload().getOperation());

                if (!cdc.getPayload().isDelete()) {
                    DrugInteractionCDCEvent.InteractionData data = cdc.getPayload().getAfter();
                    LOG.info("Interaction ID: {}", data.getInteractionId());
                    LOG.info("Drug A: {} | Drug B: {}", data.getDrugA(), data.getDrugB());
                    LOG.info("Severity: {}", data.getSeverity());
                }
                LOG.info("======================================");

                out.collect("KB5-INTERACTION-OK");
            }
        }).name("KB5 Interaction Processor");
    }

    /**
     * Test KB6 Formulary CDC consumption
     */
    private static void testFormularyCDC(StreamExecutionEnvironment env) {
        LOG.info("Configuring KB6 Formulary CDC consumer...");

        KafkaSource<FormularyDrugCDCEvent> source = KafkaSource.<FormularyDrugCDCEvent>builder()
                .setBootstrapServers(KAFKA_BROKERS)
                .setTopics("kb6.formulary_drugs.changes")
                .setGroupId("cdc-test-kb6-formulary")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forFormulary())
                .build();

        DataStream<FormularyDrugCDCEvent> stream = env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "KB6 Formulary CDC Source"
        );

        stream.process(new ProcessFunction<FormularyDrugCDCEvent, String>() {
            @Override
            public void processElement(
                    FormularyDrugCDCEvent cdc,
                    Context ctx,
                    Collector<String> out
            ) throws Exception {
                LOG.info("=== KB6 FORMULARY CDC EVENT ===");
                LOG.info("Operation: {}", cdc.getPayload().getOperation());

                if (!cdc.getPayload().isDelete()) {
                    FormularyDrugCDCEvent.FormularyData data = cdc.getPayload().getAfter();
                    LOG.info("Drug ID: {}", data.getDrugId());
                    LOG.info("Drug Name: {}", data.getDrugName());
                    LOG.info("Formulary Status: {}", data.getFormularyStatus());
                    LOG.info("Tier: {}", data.getTier());
                }
                LOG.info("===============================");

                out.collect("KB6-FORMULARY-OK");
            }
        }).name("KB6 Formulary Processor");
    }

    /**
     * Test KB7 Terminology CDC consumption
     */
    private static void testTerminologyCDC(StreamExecutionEnvironment env) {
        LOG.info("Configuring KB7 Terminology CDC consumer...");

        KafkaSource<TerminologyCDCEvent> source = KafkaSource.<TerminologyCDCEvent>builder()
                .setBootstrapServers(KAFKA_BROKERS)
                .setTopics("kb7.terminology.changes", "kb7.terminology_concepts.changes")
                .setGroupId("cdc-test-kb7-terminology")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forTerminology())
                .build();

        DataStream<TerminologyCDCEvent> stream = env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "KB7 Terminology CDC Source"
        );

        stream.process(new ProcessFunction<TerminologyCDCEvent, String>() {
            @Override
            public void processElement(
                    TerminologyCDCEvent cdc,
                    Context ctx,
                    Collector<String> out
            ) throws Exception {
                LOG.info("=== KB7 TERMINOLOGY CDC EVENT ===");
                LOG.info("Operation: {}", cdc.getPayload().getOperation());

                if (!cdc.getPayload().isDelete()) {
                    TerminologyCDCEvent.TerminologyData data = cdc.getPayload().getAfter();
                    LOG.info("Concept ID: {}", data.getConceptId());
                    LOG.info("Concept Code: {}", data.getConceptCode());
                    LOG.info("Display Name: {}", data.getDisplayName());
                    LOG.info("Code System: {}", data.getCodeSystem());
                }
                LOG.info("=================================");

                out.collect("KB7-TERMINOLOGY-OK");
            }
        }).name("KB7 Terminology Processor");
    }
}
