package health.vaidshala.substrate;

import org.apache.kafka.streams.StreamsBuilder;
import org.apache.kafka.streams.Topology;
import org.apache.kafka.streams.kstream.KStream;

/**
 * Wave 2.7 SKELETON.
 *
 * Builds the Layer 2 substrate Kafka Streams topology:
 *   raw_inbound_events
 *     → IdentityMatchingProcessor → identified_events
 *     → NormalisationProcessor    → normalised_events
 *     → SubstrateWriterProcessor  → substrate_updates
 *
 * See {@code shared/v2_substrate/streaming/topology.md} and the ADR
 * {@code docs/adr/2026-05-06-streaming-pipeline-choice.md}.
 *
 * Each {@code *Stream} method below is a placeholder; runtime wiring is
 * V1 work (search for {@code TODO(wave-2.7-runtime)}).
 */
public class SubstrateStreamApp {

    static final String TOPIC_RAW_INBOUND     = "raw_inbound_events";
    static final String TOPIC_IDENTIFIED      = "identified_events";
    static final String TOPIC_NORMALISED      = "normalised_events";
    static final String TOPIC_SUBSTRATE_UPDATES = "substrate_updates";

    public Topology buildTopology() {
        StreamsBuilder builder = new StreamsBuilder();

        identityMatchingStream(builder);
        normalisationStream(builder);
        substrateWriterStream(builder);

        return builder.build();
    }

    /**
     * raw_inbound_events → identified_events.
     * TODO(wave-2.7-runtime): wire IdentityMatchingProcessor.
     */
    KStream<String, String> identityMatchingStream(StreamsBuilder builder) {
        KStream<String, String> raw = builder.stream(TOPIC_RAW_INBOUND);
        // TODO(wave-2.7-runtime): raw.transformValues(IdentityMatchingProcessor::new).to(TOPIC_IDENTIFIED);
        return raw;
    }

    /**
     * identified_events → normalised_events.
     * TODO(wave-2.7-runtime): wire NormalisationProcessor.
     */
    KStream<String, String> normalisationStream(StreamsBuilder builder) {
        KStream<String, String> identified = builder.stream(TOPIC_IDENTIFIED);
        // TODO(wave-2.7-runtime): identified.transformValues(NormalisationProcessor::new).to(TOPIC_NORMALISED);
        return identified;
    }

    /**
     * normalised_events → substrate_updates (via kb-20 REST).
     * TODO(wave-2.7-runtime): wire SubstrateWriterProcessor — must call kb-20 REST,
     * not write to the substrate DB directly (preserves Go transactional ownership).
     */
    KStream<String, String> substrateWriterStream(StreamsBuilder builder) {
        KStream<String, String> normalised = builder.stream(TOPIC_NORMALISED);
        // TODO(wave-2.7-runtime): normalised.transformValues(SubstrateWriterProcessor::new).to(TOPIC_SUBSTRATE_UPDATES);
        return normalised;
    }
}
