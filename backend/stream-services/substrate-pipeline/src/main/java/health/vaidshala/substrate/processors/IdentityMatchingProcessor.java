package health.vaidshala.substrate.processors;

/**
 * Wave 2.7 SKELETON — interface stub.
 *
 * Calls kb-20 {@code POST /v2/identity/match} to resolve an inbound record
 * to a {@code resident_ref}. Emits to {@code identified_events} on
 * HIGH/MEDIUM confidence; routes LOW/NONE to {@code identity_review_queue}.
 *
 * TODO(wave-2.7-runtime): full HTTP client, retry policy, DLQ routing.
 */
public interface IdentityMatchingProcessor {

    /**
     * @param rawEventJson the raw inbound event payload
     * @return identified-event JSON, or {@code null} if the record was routed
     *         to the identity_review_queue and must not be forwarded
     */
    String process(String rawEventJson);
}
