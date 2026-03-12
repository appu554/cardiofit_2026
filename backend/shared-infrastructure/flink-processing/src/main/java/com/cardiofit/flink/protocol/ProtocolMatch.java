package com.cardiofit.flink.protocol;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Result of protocol matching
 */
public class ProtocolMatch implements Serializable {
    private static final long serialVersionUID = 1L;

    private String protocolId;
    private String protocolName;
    private double matchScore; // 0-100
    private int priority; // 1-10
    private List<String> recommendations;
    private long timestamp;

    public ProtocolMatch() {
        this.timestamp = System.currentTimeMillis();
        this.recommendations = new ArrayList<>();
    }

    public ProtocolMatch(String protocolId, String protocolName, double matchScore,
                        int priority, List<String> recommendations) {
        this.protocolId = protocolId;
        this.protocolName = protocolName;
        this.matchScore = matchScore;
        this.priority = priority;
        this.recommendations = recommendations != null ? recommendations : new ArrayList<>();
        this.timestamp = System.currentTimeMillis();
    }

    // Getters and setters
    public String getProtocolId() {
        return protocolId;
    }

    public void setProtocolId(String protocolId) {
        this.protocolId = protocolId;
    }

    public String getProtocolName() {
        return protocolName;
    }

    public void setProtocolName(String protocolName) {
        this.protocolName = protocolName;
    }

    public double getMatchScore() {
        return matchScore;
    }

    public void setMatchScore(double matchScore) {
        this.matchScore = matchScore;
    }

    public int getPriority() {
        return priority;
    }

    public void setPriority(int priority) {
        this.priority = priority;
    }

    public List<String> getRecommendations() {
        return recommendations;
    }

    public void setRecommendations(List<String> recommendations) {
        this.recommendations = recommendations;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

    @Override
    public String toString() {
        return "ProtocolMatch{" +
                "protocolId='" + protocolId + '\'' +
                ", protocolName='" + protocolName + '\'' +
                ", matchScore=" + String.format("%.2f", matchScore) +
                ", priority=" + priority +
                ", recommendations=" + recommendations.size() +
                '}';
    }
}