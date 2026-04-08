package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.InterventionType.ClinicalDomain;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;

public final class Module12ConcurrencyDetector {

    private static final long OVERLAP_THRESHOLD_MS = 7L * 86_400_000L;

    private Module12ConcurrencyDetector() {}

    public static Result detect(String newInterventionId,
                                 InterventionType newType,
                                 Map<String, Object> newDetail,
                                 long newStartMs, long newEndMs,
                                 Map<String, InterventionWindowState.InterventionWindow> activeWindows) {

        List<String> concurrentIds = new ArrayList<>();
        boolean sameDomain = false;

        String newDrugClass = extractDrugClass(newDetail);
        Set<ClinicalDomain> newDomains = newType.getDomains(newDrugClass);

        for (Map.Entry<String, InterventionWindowState.InterventionWindow> entry : activeWindows.entrySet()) {
            String existingId = entry.getKey();
            InterventionWindowState.InterventionWindow existing = entry.getValue();

            if (existingId.equals(newInterventionId)) continue;
            if (!"OBSERVING".equals(existing.status)) continue;

            long overlapStart = Math.max(newStartMs, existing.observationStartMs);
            long overlapEnd = Math.min(newEndMs, existing.observationEndMs);
            long overlapMs = overlapEnd - overlapStart;

            if (overlapMs >= OVERLAP_THRESHOLD_MS) {
                concurrentIds.add(existingId);

                String existingDrugClass = extractDrugClass(existing.interventionDetail);
                Set<ClinicalDomain> existingDomains =
                        existing.interventionType.getDomains(existingDrugClass);

                if (!Collections.disjoint(newDomains, existingDomains)) {
                    sameDomain = true;
                }
            }
        }

        return new Result(concurrentIds, sameDomain);
    }

    private static String extractDrugClass(Map<String, Object> detail) {
        if (detail == null) return null;
        Object dc = detail.get("drug_class");
        return dc != null ? dc.toString() : null;
    }

    public static class Result {
        private final List<String> concurrentIds;
        private final boolean sameDomainConcurrent;

        public Result(List<String> concurrentIds, boolean sameDomainConcurrent) {
            this.concurrentIds = concurrentIds;
            this.sameDomainConcurrent = sameDomainConcurrent;
        }

        public List<String> getConcurrentIds() { return concurrentIds; }
        public boolean isSameDomainConcurrent() { return sameDomainConcurrent; }
    }
}
