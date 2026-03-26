package com.cardiofit.flink;

import com.cardiofit.flink.operators.*;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.CheckpointingMode;
import org.apache.flink.contrib.streaming.state.EmbeddedRocksDBStateBackend;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;

/**
 * Main entry point for CardioFit Flink EHR Intelligence Engine
 * Complete 6-module pipeline orchestrator for hybrid topic architecture
 */
public class FlinkJobOrchestrator {
    private static final Logger LOG = LoggerFactory.getLogger(FlinkJobOrchestrator.class);

    public static void main(String[] args) throws Exception {
        LOG.info("Starting CardioFit EHR Intelligence Engine - Complete Pipeline");

        // Parse command line arguments
        // Default to comprehensive-cds (Module 3 with all 8 phases integrated)
        String jobType = args.length > 0 ? args[0] : "comprehensive-cds";
        String environmentMode = args.length > 1 ? args[1] : "production";

        // Initialize Flink execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure environment for healthcare workloads
        configureEnvironment(env, environmentMode);

        // Launch complete pipeline based on job type
        switch (jobType.toLowerCase()) {
            case "full-pipeline":
                launchFullPipeline(env);
                break;
            case "ingestion-only":
                Module1_Ingestion.createIngestionPipeline(env);
                break;
            case "context-assembly":
                Module2_Enhanced.createEnhancedPipeline(env);
                break;
            case "comprehensive-cds":
                // Module 3: Comprehensive CDS with all 8 phases integrated
                Module3_ComprehensiveCDS.createComprehensiveCDSPipeline(env);
                break;
            case "semantic-mesh":
                // Module 3: Basic semantic mesh (legacy)
                Module3_SemanticMesh.createSemanticMeshPipeline(env);
                break;
            case "pattern-detection":
                Module4_PatternDetection.createPatternDetectionPipeline(env);
                break;
            case "ml-inference":
                Module5_MLInference.createMLInferencePipeline(env);
                break;
            case "egress-routing":
                Module6_EgressRouting.createEgressRoutingPipeline(env);
                break;
            case "module1b-canonicalizer":
            case "ingestion-canonicalizer":
                // Module 1b: Canonicalizes ingestion service outbox events
                // Consumes all 9 ingestion.* topics → enriched-patient-events-v1
                Module1b_IngestionCanonicalizer.createIngestionPipeline(env);
                break;
            case "comorbidity-interaction":
            case "module8":
                Module8_ComorbidityInteraction.createComorbidityPipeline(env);
                break;
            default:
                LOG.warn("Unknown job type: {}. Defaulting to comprehensive CDS.", jobType);
                Module3_ComprehensiveCDS.createComprehensiveCDSPipeline(env);
        }

        // Execute the complete pipeline
        String jobName = String.format("CardioFit EHR Intelligence - %s (%s)",
                                      jobType, environmentMode);
        LOG.info("Executing job: {}", jobName);
        env.execute(jobName);
    }

    /**
     * Configure Flink environment for healthcare data processing
     */
    private static void configureEnvironment(StreamExecutionEnvironment env, String environmentMode) {
        LOG.info("Configuring Flink environment for mode: {}", environmentMode);

        // Set parallelism based on environment
        // Reduced from 8 to 2 for initial deployment to avoid RPC coordination overhead
        int parallelism = "production".equals(environmentMode) ? 2 : 2;
        env.setParallelism(parallelism);

        // Configure checkpointing for exactly-once processing
        env.enableCheckpointing(30000); // 30 second checkpoints
        env.getCheckpointConfig().setCheckpointingMode(CheckpointingMode.EXACTLY_ONCE);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);
        env.getCheckpointConfig().setCheckpointTimeout(600000); // 10 minutes
        env.getCheckpointConfig().setTolerableCheckpointFailureNumber(3);

        // Configure state backend for large state (Flink 2.x compatible)
        try {
            // For Flink 2.x, state backend configuration is different
            // Using default state backend for compatibility
            LOG.info("Using default state backend (compatible with Flink 2.x)");
        } catch (Exception e) {
            LOG.warn("Failed to configure state backend, using default: {}", e.getMessage());
        }

        // Configure restart strategy
        env.getConfig().setAutoWatermarkInterval(1000);

        // Configure for healthcare compliance
        env.getConfig().setGlobalJobParameters(KafkaConfigLoader.getGlobalParameters());

        LOG.info("Environment configured: parallelism={}, checkpointing=30s", parallelism);
    }

    /**
     * Launch the complete 6-module EHR Intelligence pipeline
     */
    private static void launchFullPipeline(StreamExecutionEnvironment env) {
        LOG.info("Launching complete EHR Intelligence pipeline with all 7 modules (1, 1b, 2-6)");

        try {
            // Module 1: Ingestion & Gateway (traditional EHR sources)
            LOG.info("Initializing Module 1: Ingestion & Gateway");
            Module1_Ingestion.createIngestionPipeline(env);

            // Module 1b: Ingestion Canonicalizer (outbox events from ingestion service)
            LOG.info("Initializing Module 1b: Ingestion Canonicalizer");
            Module1b_IngestionCanonicalizer.createIngestionPipeline(env);

            // Module 2: Enhanced Context Assembly
            LOG.info("Initializing Module 2: Enhanced Context Assembly");
            Module2_Enhanced.createEnhancedPipeline(env);

            // Module 3: Semantic Mesh
            LOG.info("Initializing Module 3: Semantic Mesh");
            Module3_SemanticMesh.createSemanticMeshPipeline(env);

            // Module 4: Pattern Detection
            LOG.info("Initializing Module 4: Pattern Detection");
            Module4_PatternDetection.createPatternDetectionPipeline(env);

            // Module 5: ML Inference
            LOG.info("Initializing Module 5: ML Inference");
            Module5_MLInference.createMLInferencePipeline(env);

            // Module 6: Egress Routing
            LOG.info("Initializing Module 6: Egress Routing");
            Module6_EgressRouting.createEgressRoutingPipeline(env);

            // Module 8: Comorbidity Interaction Detector
            LOG.info("Initializing Module 8: Comorbidity Interaction Detector");
            Module8_ComorbidityInteraction.createComorbidityPipeline(env);

            LOG.info("All 8 modules initialized successfully - Complete EHR Intelligence Pipeline Ready");

        } catch (Exception e) {
            LOG.error("Failed to initialize complete pipeline", e);
            throw new RuntimeException("Pipeline initialization failed", e);
        }
    }
}