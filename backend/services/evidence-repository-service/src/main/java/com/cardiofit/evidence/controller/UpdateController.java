package com.cardiofit.evidence.controller;

import com.cardiofit.evidence.EvidenceUpdateService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

/**
 * Update Management REST API Controller
 *
 * Provides manual triggers for scheduled update tasks.
 * Useful for testing, debugging, and immediate update needs.
 *
 * Base path: /api/updates
 *
 * Endpoints:
 * - POST /api/updates/retraction-check     - Trigger retraction checking
 * - POST /api/updates/evidence-search      - Trigger evidence discovery
 * - POST /api/updates/verification         - Trigger citation verification
 * - GET  /api/updates/schedule             - Get scheduled task information
 */
@RestController
@RequestMapping("/api/updates")
@CrossOrigin(origins = "*")
public class UpdateController {

    private final EvidenceUpdateService updateService;

    @Autowired
    public UpdateController(EvidenceUpdateService updateService) {
        this.updateService = updateService;
    }

    /**
     * Trigger retraction check
     *
     * POST /api/updates/retraction-check
     *
     * Manually triggers daily retraction checking for all citations.
     * Normally runs automatically at 2 AM daily.
     *
     * WARNING: This may take several minutes for large citation repositories.
     * Rate limited to 10 requests/second to PubMed.
     *
     * @return Execution summary
     */
    @PostMapping("/retraction-check")
    public ResponseEntity<Object> triggerRetractionCheck() {
        String result = updateService.triggerRetractionCheck();

        return ResponseEntity.ok(Map.of(
                "task", "retraction-check",
                "status", "completed",
                "message", result,
                "note", "Check application logs for detailed results"
        ));
    }

    /**
     * Trigger evidence search
     *
     * POST /api/updates/evidence-search
     *
     * Manually triggers monthly evidence discovery.
     * Searches PubMed for new high-quality studies.
     * Normally runs automatically at 3 AM on 1st of month.
     *
     * WARNING: This may take several minutes and will add new citations to repository.
     *
     * @return Execution summary
     */
    @PostMapping("/evidence-search")
    public ResponseEntity<Object> triggerEvidenceSearch() {
        String result = updateService.triggerEvidenceSearch();

        return ResponseEntity.ok(Map.of(
                "task", "evidence-search",
                "status", "completed",
                "message", result,
                "note", "Check application logs for newly discovered citations"
        ));
    }

    /**
     * Trigger citation verification
     *
     * POST /api/updates/verification
     *
     * Manually triggers quarterly citation verification.
     * Re-fetches metadata for stale citations (>2 years old).
     * Normally runs automatically at 4 AM on 1st day of quarter.
     *
     * WARNING: This may take several minutes for large numbers of stale citations.
     *
     * @return Execution summary
     */
    @PostMapping("/verification")
    public ResponseEntity<Object> triggerVerification() {
        String result = updateService.triggerCitationVerification();

        return ResponseEntity.ok(Map.of(
                "task", "verification",
                "status", "completed",
                "message", result,
                "note", "Check application logs for verification results and changes detected"
        ));
    }

    /**
     * Get scheduled task information
     *
     * GET /api/updates/schedule
     *
     * Returns information about automated update schedules.
     *
     * @return Schedule information
     */
    @GetMapping("/schedule")
    public ResponseEntity<Object> getScheduleInfo() {
        return ResponseEntity.ok(Map.of(
                "scheduledTasks", Map.of(
                        "retractionCheck", Map.of(
                                "schedule", "Daily at 2:00 AM",
                                "cron", "0 0 2 * * *",
                                "description", "Checks all citations for retraction notices",
                                "manualTrigger", "/api/updates/retraction-check"
                        ),
                        "evidenceSearch", Map.of(
                                "schedule", "Monthly on 1st at 3:00 AM",
                                "cron", "0 0 3 1 * *",
                                "description", "Searches PubMed for new high-quality evidence",
                                "manualTrigger", "/api/updates/evidence-search"
                        ),
                        "verification", Map.of(
                                "schedule", "Quarterly on 1st at 4:00 AM",
                                "cron", "0 0 4 1 */3 *",
                                "description", "Verifies stale citations (>2 years) by re-fetching metadata",
                                "manualTrigger", "/api/updates/verification"
                        )
                ),
                "note", "All times in server timezone. Manual triggers available for testing/immediate needs."
        ));
    }
}
