package au.vaidshala.cqlruntime.operation;

import au.vaidshala.cqlruntime.external.SubstrateExternalFunctions;
import au.vaidshala.cqlruntime.loader.CqlLibraryRegistry;

import org.hl7.fhir.r4.model.Library;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Optional;

/**
 * Exposes the kb-cql-runtime {@code $evaluate-rule} operation.
 *
 * <p>FHIR convention is to mount operations at
 * {@code POST /Library/{id}/$evaluate-rule}, but HAPI's RestfulServer
 * registration adds enough configuration weight that Plan 0.5 ships
 * the operation as a Spring REST controller using the same path
 * shape. The JSON wire format is unchanged; future tasks can swap to
 * a HAPI ResourceProvider with no contract change.
 *
 * <p>Plan 0.5 Task 5 ships the operation contract + library lookup.
 * The actual CQL engine evaluation is deferred to a follow-up task —
 * when that arrives, this controller will:
 * <ol>
 *   <li>Load the Library from the registry (already done here)</li>
 *   <li>Build a CQL engine context with SubstrateExternalFunctions
 *       registered as external function providers (TODO)</li>
 *   <li>Call engine.evaluate(libraryName, "Recommendation", residentId)</li>
 *   <li>Map the engine result to EvaluateRuleResult.decision</li>
 * </ol>
 *
 * <p>Plan 0.5 Task 5 of 8.
 */
@RestController
public class EvaluateRuleController {

    private static final Logger log = LoggerFactory.getLogger(EvaluateRuleController.class);

    private final CqlLibraryRegistry libraryRegistry;
    private final SubstrateExternalFunctions substrateFunctions;

    public EvaluateRuleController(CqlLibraryRegistry libraryRegistry,
                                  SubstrateExternalFunctions substrateFunctions) {
        this.libraryRegistry = libraryRegistry;
        this.substrateFunctions = substrateFunctions;
    }

    /**
     * POST /Library/{ruleId}/$evaluate-rule?residentId=X
     *
     * <p>Looks up the named CQL library in the registry. If found,
     * returns a result indicating the library was located and engine
     * evaluation is pending integration. If not found, returns 404.
     */
    @PostMapping("/Library/{ruleId}/$evaluate-rule")
    public ResponseEntity<EvaluateRuleResult> evaluate(
            @PathVariable("ruleId") String ruleId,
            @RequestParam(value = "residentId", required = false) String residentId) {

        if (ruleId == null || ruleId.isBlank()) {
            return ResponseEntity.badRequest().body(new EvaluateRuleResult(
                ruleId, residentId, false, "missing_parameter",
                "ruleId path variable is required"));
        }
        if (residentId == null || residentId.isBlank()) {
            return ResponseEntity.badRequest().body(new EvaluateRuleResult(
                ruleId, residentId, false, "missing_parameter",
                "residentId query parameter is required"));
        }

        Optional<Library> lib = libraryRegistry.get(ruleId);
        if (lib.isEmpty()) {
            log.info("$evaluate-rule: library {} not found", ruleId);
            return ResponseEntity.status(HttpStatus.NOT_FOUND).body(new EvaluateRuleResult(
                ruleId, residentId, false, "library_not_found",
                "no CQL library named '" + ruleId + "' is registered"));
        }

        // TODO(plan-0.5-followup): integrate cqf-fhir-cr CQL engine.
        // The library is loaded; the substrate functions are wired
        // (substrateFunctions field); the engine is the missing piece.
        // For now, return a stub result indicating the contract works.
        log.info("$evaluate-rule: library {} found for resident {}; engine evaluation pending",
            ruleId, residentId);
        return ResponseEntity.ok(new EvaluateRuleResult(
            ruleId, residentId, true, "library_found_engine_pending",
            "library located in registry; CQL engine integration deferred to follow-up task"));
    }

    /** Health check helper; mounted to confirm the controller is alive. */
    @GetMapping("/Library/{ruleId}")
    public ResponseEntity<EvaluateRuleResult> head(
            @PathVariable("ruleId") String ruleId) {
        Optional<Library> lib = libraryRegistry.get(ruleId);
        if (lib.isEmpty()) {
            return ResponseEntity.status(HttpStatus.NOT_FOUND).body(new EvaluateRuleResult(
                ruleId, null, false, "library_not_found",
                "no CQL library named '" + ruleId + "' is registered"));
        }
        return ResponseEntity.ok(new EvaluateRuleResult(
            ruleId, null, true, "library_loaded", "library is registered"));
    }
}
