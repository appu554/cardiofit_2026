package health.vaidshala.substrate.processors;

/**
 * Wave 2.7 SKELETON — interface stub.
 *
 * Normalises medication codes to AMT and indication free-text to SNOMED-CT-AU
 * by calling kb-7-terminology. Adds the resolved codes to the event payload.
 *
 * TODO(wave-2.7-runtime): kb-7 client + local cache + retry policy.
 */
public interface NormalisationProcessor {

    /**
     * @param identifiedEventJson event with resident_ref attached
     * @return normalised-event JSON with AMT/SNOMED codes appended
     */
    String process(String identifiedEventJson);
}
