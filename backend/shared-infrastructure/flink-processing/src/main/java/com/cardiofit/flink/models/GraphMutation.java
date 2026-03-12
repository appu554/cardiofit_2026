package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Map;

/**
 * Graph Mutation model for Phase 3 supporting systems
 * Represents operations to be performed on the Neo4j clinical knowledge graph
 */
public class GraphMutation {

    @JsonProperty("mutation_id")
    private String mutationId;

    @JsonProperty("mutation_type")
    private String mutationType; // CREATE, MERGE, UPDATE, DELETE

    @JsonProperty("node_type")
    private String nodeType; // Patient, ClinicalEvent, Medication, etc.

    @JsonProperty("node_id")
    private String nodeId;

    @JsonProperty("relationship_type")
    private String relationshipType; // HAS_EVENT, TAKES_MEDICATION, etc.

    @JsonProperty("from_node_id")
    private String fromNodeId;

    @JsonProperty("to_node_id")
    private String toNodeId;

    @JsonProperty("properties")
    private Map<String, Object> properties;

    @JsonProperty("cypher_query")
    private String cypherQuery;

    @JsonProperty("timestamp")
    private Long timestamp;

    @JsonProperty("source_event_id")
    private String sourceEventId;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("priority")
    private Integer priority; // 1=highest, 5=lowest

    // Constructors
    public GraphMutation() {
        this.timestamp = System.currentTimeMillis();
        this.mutationId = java.util.UUID.randomUUID().toString();
    }

    public GraphMutation(String mutationType, String nodeType, String nodeId) {
        this();
        this.mutationType = mutationType;
        this.nodeType = nodeType;
        this.nodeId = nodeId;
    }

    // Getters and Setters
    public String getMutationId() { return mutationId; }
    public void setMutationId(String mutationId) { this.mutationId = mutationId; }

    public String getMutationType() { return mutationType; }
    public void setMutationType(String mutationType) { this.mutationType = mutationType; }

    public String getNodeType() { return nodeType; }
    public void setNodeType(String nodeType) { this.nodeType = nodeType; }

    public String getNodeId() { return nodeId; }
    public void setNodeId(String nodeId) { this.nodeId = nodeId; }

    public String getRelationshipType() { return relationshipType; }
    public void setRelationshipType(String relationshipType) { this.relationshipType = relationshipType; }

    public String getFromNodeId() { return fromNodeId; }
    public void setFromNodeId(String fromNodeId) { this.fromNodeId = fromNodeId; }

    public String getToNodeId() { return toNodeId; }
    public void setToNodeId(String toNodeId) { this.toNodeId = toNodeId; }

    public Map<String, Object> getProperties() { return properties; }
    public void setProperties(Map<String, Object> properties) { this.properties = properties; }

    public String getCypherQuery() { return cypherQuery; }
    public void setCypherQuery(String cypherQuery) { this.cypherQuery = cypherQuery; }

    public Long getTimestamp() { return timestamp; }
    public void setTimestamp(Long timestamp) { this.timestamp = timestamp; }

    public String getSourceEventId() { return sourceEventId; }
    public void setSourceEventId(String sourceEventId) { this.sourceEventId = sourceEventId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public Integer getPriority() { return priority; }
    public void setPriority(Integer priority) { this.priority = priority; }

    /**
     * Generate Cypher query based on mutation type
     */
    public String generateCypherQuery() {
        if (cypherQuery != null) {
            return cypherQuery;
        }

        StringBuilder cypher = new StringBuilder();

        switch (mutationType.toUpperCase()) {
            case "CREATE":
                if (relationshipType != null) {
                    // Create relationship
                    cypher.append("MATCH (a {id: '").append(fromNodeId).append("'}) ")
                          .append("MATCH (b {id: '").append(toNodeId).append("'}) ")
                          .append("CREATE (a)-[:").append(relationshipType);
                    if (properties != null && !properties.isEmpty()) {
                        cypher.append(" {").append(formatProperties()).append("}");
                    }
                    cypher.append("]->(b)");
                } else {
                    // Create node
                    cypher.append("CREATE (n:").append(nodeType).append(" {id: '").append(nodeId).append("'");
                    if (properties != null && !properties.isEmpty()) {
                        cypher.append(", ").append(formatProperties());
                    }
                    cypher.append("})");
                }
                break;

            case "MERGE":
                cypher.append("MERGE (n:").append(nodeType).append(" {id: '").append(nodeId).append("'})");
                if (properties != null && !properties.isEmpty()) {
                    cypher.append(" SET ").append(formatSetProperties("n"));
                }
                break;

            case "UPDATE":
                cypher.append("MATCH (n:").append(nodeType).append(" {id: '").append(nodeId).append("'}) ")
                      .append("SET ").append(formatSetProperties("n"));
                break;

            case "DELETE":
                cypher.append("MATCH (n:").append(nodeType).append(" {id: '").append(nodeId).append("'}) ")
                      .append("DETACH DELETE n");
                break;
        }

        this.cypherQuery = cypher.toString();
        return cypherQuery;
    }

    private String formatProperties() {
        if (properties == null || properties.isEmpty()) {
            return "";
        }

        StringBuilder sb = new StringBuilder();
        boolean first = true;
        for (Map.Entry<String, Object> entry : properties.entrySet()) {
            if (!first) sb.append(", ");
            sb.append(entry.getKey()).append(": ");
            if (entry.getValue() instanceof String) {
                sb.append("'").append(entry.getValue()).append("'");
            } else {
                sb.append(entry.getValue());
            }
            first = false;
        }
        return sb.toString();
    }

    private String formatSetProperties(String nodeVar) {
        if (properties == null || properties.isEmpty()) {
            return "";
        }

        StringBuilder sb = new StringBuilder();
        boolean first = true;
        for (Map.Entry<String, Object> entry : properties.entrySet()) {
            if (!first) sb.append(", ");
            sb.append(nodeVar).append(".").append(entry.getKey()).append(" = ");
            if (entry.getValue() instanceof String) {
                sb.append("'").append(entry.getValue()).append("'");
            } else {
                sb.append(entry.getValue());
            }
            first = false;
        }
        return sb.toString();
    }

    @Override
    public String toString() {
        return "GraphMutation{" +
                "mutationId='" + mutationId + '\'' +
                ", mutationType='" + mutationType + '\'' +
                ", nodeType='" + nodeType + '\'' +
                ", nodeId='" + nodeId + '\'' +
                ", timestamp=" + timestamp +
                '}';
    }
}