package com.cardiofit.flink.config;

import org.apache.flink.contrib.streaming.state.EmbeddedRocksDBStateBackend;
import org.apache.flink.state.rocksdb.PredefinedOptions;
import org.apache.flink.contrib.streaming.state.RocksDBConfigurableOptions;
import org.apache.flink.configuration.ConfigOption;
import org.apache.flink.configuration.ConfigOptions;
import org.apache.flink.configuration.Configuration;
import org.rocksdb.*;

import java.util.ArrayList;
import java.util.Collection;

/**
 * Production-ready RocksDB configuration for healthcare data processing
 * Based on CardioFit platform requirements and clinical data patterns
 */
public class RocksDBStateConfig {

    // Configuration options for different healthcare data types
    public static final ConfigOption<String> PATIENT_STATE_TTL = ConfigOptions
        .key("state.backend.rocksdb.patient.ttl")
        .stringType()
        .defaultValue("365d")
        .withDescription("TTL for patient demographic and condition state");

    public static final ConfigOption<String> VITAL_SIGNS_TTL = ConfigOptions
        .key("state.backend.rocksdb.vitals.ttl")
        .stringType()
        .defaultValue("24h")
        .withDescription("TTL for vital signs and monitoring data");

    public static final ConfigOption<String> MEDICATION_STATE_TTL = ConfigOptions
        .key("state.backend.rocksdb.medications.ttl")
        .stringType()
        .defaultValue("30d")
        .withDescription("TTL for active medication state");

    public static final ConfigOption<String> LAB_RESULTS_TTL = ConfigOptions
        .key("state.backend.rocksdb.labs.ttl")
        .stringType()
        .defaultValue("90d")
        .withDescription("TTL for laboratory results and trends");

    /**
     * Create optimized RocksDB state backend for healthcare data processing
     * Configured for high-throughput patient data with memory efficiency
     */
    public static EmbeddedRocksDBStateBackend createOptimizedStateBackend() throws Exception {
        EmbeddedRocksDBStateBackend stateBackend = new EmbeddedRocksDBStateBackend(true);

        // Set predefined options for high-throughput workloads
        stateBackend.setPredefinedOptions(PredefinedOptions.SPINNING_DISK_OPTIMIZED_HIGH_MEM);

        // Configure RocksDB options for healthcare data patterns
        Configuration rocksConfig = new Configuration();

        // Memory management for patient data processing
        rocksConfig.setString("state.backend.rocksdb.memory.write-buffer-ratio", "0.5");
        rocksConfig.setString("state.backend.rocksdb.memory.high-prio-pool-ratio", "0.1");

        // Configure for high-throughput healthcare workloads
        rocksConfig.setString("state.backend.rocksdb.write-batch-size", "2048");
        rocksConfig.setString("state.backend.rocksdb.block.cache-size", "512MB");
        rocksConfig.setString("state.backend.rocksdb.block.blocksize", "32KB");
        rocksConfig.setString("state.backend.rocksdb.compaction.level.target-file-size-base", "256MB");
        rocksConfig.setString("state.backend.rocksdb.compaction.level.max-size-level-base", "1GB");

        stateBackend.configure(rocksConfig, RocksDBStateConfig.class.getClassLoader());

        // Enable incremental checkpointing for large patient state
        // Note: incremental checkpointing is enabled by default in Flink 1.17

        return stateBackend;
    }

    /**
     * Create RocksDB configuration for development environment
     * Lighter resource usage for local testing
     */
    public static EmbeddedRocksDBStateBackend createDevelopmentStateBackend() throws Exception {
        EmbeddedRocksDBStateBackend stateBackend = new EmbeddedRocksDBStateBackend(true);

        stateBackend.setPredefinedOptions(PredefinedOptions.DEFAULT);

        // Configure RocksDB for development environment
        Configuration devRocksConfig = new Configuration();

        // Lighter configuration for development
        devRocksConfig.setString("state.backend.rocksdb.memory.write-buffer-ratio", "0.3");
        devRocksConfig.setString("state.backend.rocksdb.block.cache-size", "128MB");
        devRocksConfig.setString("state.backend.rocksdb.block.blocksize", "16KB");
        devRocksConfig.setString("state.backend.rocksdb.write-batch-size", "1024");

        stateBackend.configure(devRocksConfig, RocksDBStateConfig.class.getClassLoader());

        // Incremental checkpointing is enabled by default in Flink 1.17
        return stateBackend;
    }

    /**
     * Configure environment-specific state backend
     */
    public static EmbeddedRocksDBStateBackend createStateBackend(Configuration config) throws Exception {
        String environment = config.getString("deployment.environment", "development");

        if ("production".equalsIgnoreCase(environment)) {
            return createOptimizedStateBackend();
        } else {
            return createDevelopmentStateBackend();
        }
    }

    /**
     * Clinical data state TTL configurations
     */
    public static class ClinicalStateTTL {
        // Patient demographic and administrative data - long retention
        public static final long PATIENT_DEMOGRAPHICS_TTL_DAYS = 365 * 5; // 5 years
        public static final long ADMISSION_HISTORY_TTL_DAYS = 365 * 2; // 2 years

        // Clinical data - medium retention
        public static final long ACTIVE_CONDITIONS_TTL_DAYS = 365; // 1 year
        public static final long MEDICATION_HISTORY_TTL_DAYS = 90; // 3 months
        public static final long PROCEDURE_HISTORY_TTL_DAYS = 180; // 6 months

        // Monitoring data - short retention
        public static final long VITAL_SIGNS_TTL_HOURS = 24; // 24 hours
        public static final long DEVICE_DATA_TTL_HOURS = 12; // 12 hours
        public static final long ALERT_STATE_TTL_HOURS = 48; // 48 hours

        // Laboratory data - regulatory retention
        public static final long LAB_RESULTS_TTL_DAYS = 90; // 90 days
        public static final long PATHOLOGY_TTL_DAYS = 365; // 1 year

        // Calculated scores and derived data
        public static final long RISK_SCORES_TTL_DAYS = 30; // 30 days
        public static final long PREDICTION_CACHE_TTL_HOURS = 6; // 6 hours
        public static final long TREND_ANALYSIS_TTL_DAYS = 7; // 7 days
    }

    /**
     * Memory management configurations for different node types
     */
    public static class MemoryConfig {
        // TaskManager memory allocation for different modules
        public static final double PATIENT_STATE_MEMORY_FRACTION = 0.6; // 60% for patient state
        public static final double BROADCAST_STATE_MEMORY_FRACTION = 0.2; // 20% for clinical knowledge
        public static final double WINDOW_STATE_MEMORY_FRACTION = 0.15; // 15% for windowed analytics
        public static final double BUFFER_MEMORY_FRACTION = 0.05; // 5% for buffers

        // RocksDB cache sizing based on available memory
        public static long calculateBlockCacheSize(long availableMemory) {
            return (long) (availableMemory * 0.3); // 30% of available memory
        }

        public static long calculateWriteBufferSize(long availableMemory) {
            return (long) (availableMemory * 0.1); // 10% of available memory
        }
    }

    /**
     * Performance monitoring configuration
     */
    public static class MonitoringConfig {
        // Compaction performance thresholds
        public static final long MAX_COMPACTION_TIME_MS = 30000; // 30 seconds
        public static final long MAX_FLUSH_TIME_MS = 5000; // 5 seconds

        // State size monitoring
        public static final long MAX_PATIENT_STATE_SIZE_MB = 100; // 100MB per patient
        public static final long ALERT_STATE_SIZE_THRESHOLD_MB = 1024; // 1GB total state alert

        // Performance metrics collection interval
        public static final long METRICS_COLLECTION_INTERVAL_MS = 30000; // 30 seconds
    }
}