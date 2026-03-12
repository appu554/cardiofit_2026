package com.cardiofit.stream.models;

public enum Severity {
    LOW("LOW", 1),
    MEDIUM("MEDIUM", 2),
    HIGH("HIGH", 3),
    CRITICAL("CRITICAL", 4),
    EMERGENCY("EMERGENCY", 5);

    private final String level;
    private final int priority;

    Severity(String level, int priority) {
        this.level = level;
        this.priority = priority;
    }

    public String getLevel() {
        return level;
    }

    public int getPriority() {
        return priority;
    }

    public boolean isHigherThan(Severity other) {
        return this.priority > other.priority;
    }
}