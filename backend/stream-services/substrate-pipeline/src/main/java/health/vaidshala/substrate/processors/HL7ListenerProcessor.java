package health.vaidshala.substrate.processors;

/**
 * Wave 3.3 SKELETON — interface stub for the HL7 v2.5 ORU^R01 listener.
 *
 * Production implementation accepts inbound HL7 messages from per-vendor
 * pathology providers (MLLP framing on a TCP socket, or HTTPS POST for
 * vendors using HL7-over-HTTP). The listener parses the ORU^R01 message
 * via the Go-side {@code shared/v2_substrate/ingestion.ParseORUR01}
 * (called via gRPC/HTTP from this Java service), then emits the resulting
 * {@code ParsedObservation} list onto the {@code raw_pathology_events}
 * Kafka topic for downstream identity matching + substrate write.
 *
 * Per-vendor adapters: registered server-side in Go via
 * {@code RegisterVendorAdapter(name, adapter)}. The Java listener passes
 * the vendor name (resolved from the source connection's
 * facility/vendor configuration) to the parser; vendor-specific quirks
 * are applied entirely on the Go side so this listener stays vendor-
 * agnostic.
 *
 * <p>TODO(wave-3.3-runtime):
 * <ul>
 *   <li>MLLP framing (start-of-block 0x0B, end-of-block 0x1C, CR 0x0D)</li>
 *   <li>HTTPS POST endpoint variant for vendor X</li>
 *   <li>Acknowledgement (ACK/NACK) per HL7 v2.5 spec</li>
 *   <li>Retry + DLQ routing</li>
 *   <li>Per-vendor authentication (mTLS for some, IP allowlist for others)</li>
 * </ul>
 *
 * <p>See also:
 * <ul>
 *   <li>docs/adr/2026-05-06-mhr-integration-strategy.md — overall
 *       integration strategy + deferral rationale</li>
 *   <li>shared/v2_substrate/ingestion/hl7_oru.go — Go-side parser</li>
 * </ul>
 */
public interface HL7ListenerProcessor {

    /**
     * @param vendorName the registered vendor identifier driving adapter
     *                   selection; passed through to the Go-side parser
     * @param rawHL7Message the raw HL7 v2.5 message bytes (segment-
     *                      delimited; CR or LF accepted)
     * @return the JSON-encoded {@code CDAPathologyResult} produced by
     *         the Go parser, or {@code null} when the message was
     *         malformed and routed to the DLQ
     */
    String process(String vendorName, byte[] rawHL7Message);
}
