package au.vaidshala.cqlruntime.operation;

/**
 * Typed result of a {@code $evaluate-rule} operation. POJOs are simpler
 * for Spring REST mapping than constructing FHIR Parameters in Java —
 * the JSON wire format remains compatible.
 *
 * <p>Plan 0.5 Task 5 ships a stub-shaped result. The eventual CQL engine
 * integration (deferred to a follow-up task) will populate the
 * {@code decision} and {@code reasoningSummary} fields with rule-firing
 * outcomes.
 */
public class EvaluateRuleResult {

    private String ruleId;
    private String residentId;
    private boolean libraryFound;
    /** Status string. One of: "library_found_engine_pending" | "library_not_found" | "missing_parameter". */
    private String status;
    private String message;

    public EvaluateRuleResult() {}

    public EvaluateRuleResult(String ruleId, String residentId,
                              boolean libraryFound, String status, String message) {
        this.ruleId = ruleId;
        this.residentId = residentId;
        this.libraryFound = libraryFound;
        this.status = status;
        this.message = message;
    }

    public String getRuleId() { return ruleId; }
    public void setRuleId(String v) { this.ruleId = v; }

    public String getResidentId() { return residentId; }
    public void setResidentId(String v) { this.residentId = v; }

    public boolean isLibraryFound() { return libraryFound; }
    public void setLibraryFound(boolean v) { this.libraryFound = v; }

    public String getStatus() { return status; }
    public void setStatus(String v) { this.status = v; }

    public String getMessage() { return message; }
    public void setMessage(String v) { this.message = v; }
}
