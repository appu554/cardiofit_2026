package com.cardiofit.flink.serialization;

import org.apache.flink.api.common.serialization.SerializationSchema;

import java.lang.reflect.Method;
import java.nio.charset.StandardCharsets;

/**
 * PIPE-8: Generic Kafka key serializer that extracts patientId from any event type.
 *
 * Uses reflection to call {@code getPatientId()} on the event object, producing
 * a UTF-8 encoded byte[] key.  This ensures all patient-keyed Kafka topics
 * co-partition on the same key, enabling downstream consumers to maintain
 * per-patient state locality without re-shuffling.
 *
 * Falls back to null key (round-robin) if the event has no patientId or the
 * method is absent — preserving backward compatibility.
 *
 * @param <T> Event type that exposes {@code getPatientId()}
 */
public class PatientIdKeySerializer<T> implements SerializationSchema<T> {

    private static final long serialVersionUID = 1L;

    @Override
    public byte[] serialize(T element) {
        if (element == null) return null;

        try {
            Method m = element.getClass().getMethod("getPatientId");
            Object id = m.invoke(element);
            if (id instanceof String && !((String) id).isEmpty()) {
                return ((String) id).getBytes(StandardCharsets.UTF_8);
            }
        } catch (NoSuchMethodException e) {
            // Event type lacks getPatientId — fall through to null key
        } catch (Exception e) {
            // Reflection failure — log would be noisy in hot path; null key is safe
        }
        return null;
    }
}
