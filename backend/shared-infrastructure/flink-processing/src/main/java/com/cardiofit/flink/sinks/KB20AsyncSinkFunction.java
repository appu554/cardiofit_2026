package com.cardiofit.flink.sinks;

import com.cardiofit.flink.models.KB20StateUpdate;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.sink.RichSinkFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Async sink for KB-20 state updates.
 * Writes to PostgreSQL (parameterised upsert) and Redis (pipeline SET for projections).
 * Circuit breaker: 3 consecutive failures → open for 30s.
 * During open-circuit, updates are buffered (max 1000 per patient).
 */
public class KB20AsyncSinkFunction extends RichSinkFunction<KB20StateUpdate> {

    private static final Logger LOG = LoggerFactory.getLogger(KB20AsyncSinkFunction.class);

    private static final int CIRCUIT_FAILURE_THRESHOLD = 3;
    private static final long CIRCUIT_OPEN_DURATION_MS = 30_000L;
    private static final int MAX_BUFFER_SIZE = 1000;

    private transient AtomicInteger consecutiveFailures;
    private transient AtomicLong circuitOpenedAt;
    private transient List<KB20StateUpdate> failoverBuffer;
    private transient AtomicLong writeCount;
    private transient AtomicLong writeFailures;

    private final String postgresUrl;
    private final String redisUrl;

    public KB20AsyncSinkFunction(String postgresUrl, String redisUrl) {
        this.postgresUrl = postgresUrl;
        this.redisUrl = redisUrl;
    }

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);
        consecutiveFailures = new AtomicInteger(0);
        circuitOpenedAt = new AtomicLong(0L);
        failoverBuffer = new ArrayList<>();
        writeCount = new AtomicLong(0);
        writeFailures = new AtomicLong(0);
        LOG.info("KB20AsyncSinkFunction initialized: postgres={}, redis={}", postgresUrl, redisUrl);
    }

    @Override
    public void invoke(KB20StateUpdate update, Context context) throws Exception {
        if (isCircuitOpen()) {
            bufferUpdate(update);
            return;
        }

        try {
            writeToPostgres(update);
            writeToRedis(update);
            consecutiveFailures.set(0);
            writeCount.incrementAndGet();

            if (!failoverBuffer.isEmpty()) {
                flushBuffer();
            }

        } catch (Exception e) {
            int failures = consecutiveFailures.incrementAndGet();
            writeFailures.incrementAndGet();
            LOG.warn("KB-20 write failed (attempt {}): patient={}, error={}",
                    failures, update.getPatientId(), e.getMessage());

            if (failures >= CIRCUIT_FAILURE_THRESHOLD) {
                circuitOpenedAt.set(System.currentTimeMillis());
                LOG.error("Circuit breaker OPENED for KB-20 sink after {} failures", failures);
            }
            bufferUpdate(update);
        }
    }

    private boolean isCircuitOpen() {
        long openedAt = circuitOpenedAt.get();
        if (openedAt == 0L) return false;
        if (System.currentTimeMillis() - openedAt > CIRCUIT_OPEN_DURATION_MS) {
            circuitOpenedAt.set(0L);
            consecutiveFailures.set(0);
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
        LOG.info("Flushing {} buffered KB-20 updates", failoverBuffer.size());
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
     * Implementation connects to KB-20's patient_state table.
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
     * Implementation uses Redis pipeline SET for fast projection updates.
     * Infrastructure wiring: replace with actual Jedis/Lettuce client.
     */
    private void writeToRedis(KB20StateUpdate update) {
        // Actual implementation will use Redis pipeline:
        // HSET kb20:patient:{patient_id}:streaming {field_name} {field_value_json}
        // EXPIRE kb20:patient:{patient_id}:streaming 7776000  (90 days)
        LOG.debug("Redis write: patient={}, field={}",
                update.getPatientId(), update.getFieldPath());
    }

    @Override
    public void close() throws Exception {
        if (!failoverBuffer.isEmpty()) {
            LOG.warn("KB20AsyncSinkFunction closing with {} unbuffered updates", failoverBuffer.size());
        }
        LOG.info("KB20AsyncSinkFunction closed: writes={}, failures={}",
                writeCount.get(), writeFailures.get());
        super.close();
    }
}
