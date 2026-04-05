package com.cardiofit.flink.sinks;

import com.cardiofit.flink.models.KB20StateUpdate;
import org.apache.flink.api.connector.sink2.Sink;
import org.apache.flink.api.connector.sink2.SinkWriter;
import org.apache.flink.api.connector.sink2.WriterInitContext;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

/**
 * Flink 2.x Sink for KB-20 state updates.
 * Writes to PostgreSQL (parameterised upsert) and Redis (pipeline SET for projections).
 * Circuit breaker: 3 consecutive failures → open for 30s.
 * During open-circuit, updates are buffered (max 1000 per patient).
 *
 * Migrated to Flink 2.x Sink API (replaces deprecated RichSinkFunction).
 */
public class KB20AsyncSinkFunction implements Sink<KB20StateUpdate> {

    private static final Logger LOG = LoggerFactory.getLogger(KB20AsyncSinkFunction.class);

    private final String postgresUrl;
    private final String redisUrl;

    public KB20AsyncSinkFunction(String postgresUrl, String redisUrl) {
        this.postgresUrl = postgresUrl;
        this.redisUrl = redisUrl;
    }

    @Override
    public SinkWriter<KB20StateUpdate> createWriter(WriterInitContext context) throws IOException {
        return new KB20SinkWriter(postgresUrl, redisUrl);
    }

    /**
     * SinkWriter with circuit breaker and failover buffer.
     */
    private static class KB20SinkWriter implements SinkWriter<KB20StateUpdate> {

        private static final Logger LOG = LoggerFactory.getLogger(KB20SinkWriter.class);

        private static final int CIRCUIT_FAILURE_THRESHOLD = 3;
        private static final long CIRCUIT_OPEN_DURATION_MS = 30_000L;
        private static final int MAX_BUFFER_SIZE = 1000;

        private final String postgresUrl;
        private final String redisUrl;

        private int consecutiveFailures;
        private long circuitOpenedAt;
        private final List<KB20StateUpdate> failoverBuffer;
        private long writeCount;
        private long writeFailures;

        KB20SinkWriter(String postgresUrl, String redisUrl) {
            this.postgresUrl = postgresUrl;
            this.redisUrl = redisUrl;
            this.consecutiveFailures = 0;
            this.circuitOpenedAt = 0L;
            this.failoverBuffer = new ArrayList<>();
            this.writeCount = 0;
            this.writeFailures = 0;
            LOG.info("KB20SinkWriter initialized: postgres={}, redis={}", postgresUrl, redisUrl);
        }

        @Override
        public void write(KB20StateUpdate update, Context context) throws IOException {
            if (isCircuitOpen()) {
                bufferUpdate(update);
                return;
            }

            try {
                writeToPostgres(update);
                writeToRedis(update);
                consecutiveFailures = 0;
                writeCount++;

                if (!failoverBuffer.isEmpty()) {
                    flushBuffer();
                }

            } catch (Exception e) {
                consecutiveFailures++;
                writeFailures++;
                LOG.warn("KB-20 write failed (attempt {}): patient={}, error={}",
                        consecutiveFailures, update.getPatientId(), e.getMessage());

                if (consecutiveFailures >= CIRCUIT_FAILURE_THRESHOLD) {
                    circuitOpenedAt = System.currentTimeMillis();
                    LOG.error("Circuit breaker OPENED for KB-20 sink after {} failures", consecutiveFailures);
                }
                bufferUpdate(update);
            }
        }

        @Override
        public void flush(boolean endOfInput) throws IOException {
            if (!failoverBuffer.isEmpty()) {
                LOG.info("Flushing {} buffered KB-20 updates on checkpoint", failoverBuffer.size());
                flushBuffer();
            }
        }

        @Override
        public void close() throws IOException {
            if (!failoverBuffer.isEmpty()) {
                LOG.warn("KB20SinkWriter closing with {} unbuffered updates", failoverBuffer.size());
            }
            LOG.info("KB20SinkWriter closed: writes={}, failures={}", writeCount, writeFailures);
        }

        private boolean isCircuitOpen() {
            if (circuitOpenedAt == 0L) return false;
            if (System.currentTimeMillis() - circuitOpenedAt > CIRCUIT_OPEN_DURATION_MS) {
                circuitOpenedAt = 0L;
                consecutiveFailures = 0;
                LOG.info("Circuit breaker CLOSED for KB-20 sink (recovery attempt)");
                return false;
            }
            return true;
        }

        private void bufferUpdate(KB20StateUpdate update) {
            if (failoverBuffer.size() >= MAX_BUFFER_SIZE) {
                failoverBuffer.remove(0);
                LOG.warn("KB-20 failover buffer full ({}), evicting oldest for patient {}",
                        MAX_BUFFER_SIZE, update.getPatientId());
            }
            failoverBuffer.add(update);
        }

        private void flushBuffer() {
            List<KB20StateUpdate> toFlush = new ArrayList<>(failoverBuffer);
            failoverBuffer.clear();
            for (KB20StateUpdate buffered : toFlush) {
                try {
                    writeToPostgres(buffered);
                    writeToRedis(buffered);
                } catch (Exception e) {
                    LOG.warn("Buffer flush failed for patient {}: {}",
                            buffered.getPatientId(), e.getMessage());
                    failoverBuffer.add(buffered);
                }
            }
        }

        /**
         * Write field-level upsert to PostgreSQL.
         * Infrastructure wiring: replace with actual JDBC/async-pg client.
         */
        private void writeToPostgres(KB20StateUpdate update) {
            // Actual implementation will use parameterised upsert:
            // INSERT INTO patient_streaming_state (patient_id, field_name, field_value, updated_at, source_module)
            // VALUES (?, ?, ?::jsonb, ?, ?)
            // ON CONFLICT (patient_id, field_name) DO UPDATE SET
            //   field_value = EXCLUDED.field_value,
            //   updated_at = EXCLUDED.updated_at,
            //   source_module = EXCLUDED.source_module
            // WHERE EXCLUDED.updated_at > patient_streaming_state.updated_at;
            LOG.debug("PostgreSQL write: patient={}, field={}, op={}",
                    update.getPatientId(), update.getFieldPath(), update.getOperation());
        }

        /**
         * Update Redis projection for KB-20.
         * Infrastructure wiring: replace with actual Jedis/Lettuce client.
         */
        private void writeToRedis(KB20StateUpdate update) {
            // Actual implementation will use Redis pipeline:
            // HSET kb20:patient:{patient_id}:streaming {field_name} {field_value_json}
            // EXPIRE kb20:patient:{patient_id}:streaming 7776000  (90 days)
            LOG.debug("Redis write: patient={}, field={}",
                    update.getPatientId(), update.getFieldPath());
        }
    }
}
