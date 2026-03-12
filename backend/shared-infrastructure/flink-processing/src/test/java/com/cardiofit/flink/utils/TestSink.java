package com.cardiofit.flink.utils;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import java.util.concurrent.ConcurrentLinkedQueue;

/**
 * Shared test collector for test harness outputs.
 *
 * Thread-safe implementation using ConcurrentLinkedQueue for multi-threaded Flink tests.
 * Based on Flink test harness best practices.
 *
 * Usage:
 * <pre>
 *   TestSink.clear();  // Before test
 *   // In test harness, collect outputs manually:
 *   List<MyType> outputs = harness.extractOutputValues();
 *   TestSink.addAll(outputs);
 *   List<MyType> results = TestSink.getValues();
 * </pre>
 *
 * Note: This is a simplified collector for test harness usage.
 * Does not implement SinkFunction (deprecated in Flink 1.18+).
 */
public class TestSink<T> {

    // Thread-safe shared queue for all test outputs
    public static final ConcurrentLinkedQueue<Object> VALUES = new ConcurrentLinkedQueue<>();

    /**
     * Add a single value to the collector.
     * Thread-safe operation for parallel execution.
     */
    public static synchronized <T> void add(T value) {
        VALUES.add(value);
    }

    /**
     * Add multiple values to the collector.
     * Thread-safe operation for batch collection.
     */
    public static synchronized <T> void addAll(Collection<T> values) {
        VALUES.addAll(values);
    }

    /**
     * Clear all collected values.
     * Call before each test to ensure clean state.
     */
    public static void clear() {
        VALUES.clear();
    }

    /**
     * Get all collected values as a list.
     *
     * @param <T> The type of values
     * @return List of all collected values
     */
    @SuppressWarnings("unchecked")
    public static <T> List<T> getValues() {
        return new ArrayList<>((Collection<T>) VALUES);
    }

    /**
     * Get the last collected value.
     *
     * @param <T> The type of value
     * @return The last value, or null if empty
     */
    @SuppressWarnings("unchecked")
    public static <T> T getLastValue() {
        Object[] array = VALUES.toArray();
        return array.length > 0 ? (T) array[array.length - 1] : null;
    }

    /**
     * Get the number of collected values.
     *
     * @return Count of values
     */
    public static int size() {
        return VALUES.size();
    }

    /**
     * Check if any values have been collected.
     *
     * @return true if VALUES is not empty
     */
    public static boolean hasValues() {
        return !VALUES.isEmpty();
    }
}
