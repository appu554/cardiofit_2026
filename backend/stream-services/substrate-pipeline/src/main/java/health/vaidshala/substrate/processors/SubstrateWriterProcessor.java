package health.vaidshala.substrate.processors;

/**
 * Wave 2.7 SKELETON — interface stub.
 *
 * Writes the substrate by calling kb-20 REST endpoints
 * ({@code POST /v2/observations}, {@code POST /v2/medicine_uses},
 * {@code POST /v2/events}). MUST NOT write to the substrate DB directly —
 * Go owns the substrate transaction boundary; this processor is a thin proxy.
 *
 * TODO(wave-2.7-runtime): HTTP client with idempotency keys, retry/backoff,
 * DLQ on permanent failure, emit substrate_updates with kb-20 response payload.
 */
public interface SubstrateWriterProcessor {

    /**
     * @param normalisedEventJson event with AMT/SNOMED codes attached
     * @return substrate_updates JSON (kb-20 response with id + delta)
     */
    String process(String normalisedEventJson);
}
