package health.vaidshala.substrate;

import org.apache.kafka.streams.KafkaStreams;
import org.apache.kafka.streams.StreamsConfig;
import org.apache.kafka.streams.Topology;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.InputStream;
import java.util.Properties;

/**
 * Wave 2.7 SKELETON entrypoint.
 *
 * Loads {@code application.properties}, builds the topology, and starts
 * the Kafka Streams app. Real config-loading (Confluent Cloud creds,
 * schema-registry URL, retry policy) is V1 work.
 */
public final class Main {

    private static final Logger LOG = LoggerFactory.getLogger(Main.class);

    private Main() {}

    public static void main(String[] args) throws IOException {
        Properties props = loadProperties();
        SubstrateStreamApp app = new SubstrateStreamApp();
        Topology topology = app.buildTopology();

        LOG.info("Starting substrate-pipeline (Wave 2.7 skeleton). Topology: {}", topology.describe());

        // TODO(wave-2.7-runtime): real lifecycle (state listener, uncaught
        // exception handler, graceful shutdown hook, Prometheus metrics binder).
        try (KafkaStreams streams = new KafkaStreams(topology, props)) {
            Runtime.getRuntime().addShutdownHook(new Thread(streams::close, "substrate-shutdown"));
            streams.start();
        }
    }

    private static Properties loadProperties() throws IOException {
        Properties props = new Properties();
        try (InputStream in = Main.class.getResourceAsStream("/application.properties")) {
            if (in == null) {
                throw new IOException("application.properties not found on classpath");
            }
            props.load(in);
        }
        // Fallbacks so the topology is well-formed even with the placeholder config.
        props.putIfAbsent(StreamsConfig.APPLICATION_ID_CONFIG, "substrate-pipeline");
        return props;
    }
}
