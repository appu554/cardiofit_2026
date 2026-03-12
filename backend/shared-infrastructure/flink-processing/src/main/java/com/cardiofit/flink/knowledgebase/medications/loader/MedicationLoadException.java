package com.cardiofit.flink.knowledgebase.medications.loader;

/**
 * Exception thrown when medication database loading fails.
 *
 * Indicates errors during YAML parsing, validation, or file I/O operations.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-25
 */
public class MedicationLoadException extends RuntimeException {

    private static final long serialVersionUID = 1L;

    /**
     * Constructs a new medication load exception with the specified detail message.
     *
     * @param message the detail message explaining the load failure
     */
    public MedicationLoadException(String message) {
        super(message);
    }

    /**
     * Constructs a new medication load exception with the specified detail message and cause.
     *
     * @param message the detail message explaining the load failure
     * @param cause the cause of the exception (a throwable that triggered this exception)
     */
    public MedicationLoadException(String message, Throwable cause) {
        super(message, cause);
    }

    /**
     * Constructs a new medication load exception with the specified cause.
     *
     * @param cause the cause of the exception (a throwable that triggered this exception)
     */
    public MedicationLoadException(Throwable cause) {
        super(cause);
    }
}
