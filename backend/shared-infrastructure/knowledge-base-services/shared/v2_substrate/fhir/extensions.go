// Package fhir provides AU FHIR R4 mappers translating v2 substrate types
// to and from HL7 AU Base v6.0.0 profiles. Vaidshala internal code uses
// v2_substrate/models throughout; this package only runs at integration
// boundaries (Layer 1B adapters in, regulatory reporting out).
//
// Reference IGs (procured under integration_specs/):
//   - HL7 AU Base v6.0.0:           hl7_au/base_ig_r4/
//   - MHR FHIR Gateway v5.0:        adha_fhir/mhr_gateway_ig_v1_4_0/
//   - Discharge Summary v1.7 (CDA): hospital_transitions/au_fhir_discharge_summary/
package fhir

// AU FHIR extension URIs. Sourced from HL7 AU Base IG v6.0.0; refresh
// quarterly when re-procured.
const (
	ExtIHI               = "http://hl7.org.au/fhir/StructureDefinition/ihi"
	ExtHPII              = "http://hl7.org.au/fhir/StructureDefinition/hpii"
	ExtIndigenousStatus  = "http://hl7.org.au/fhir/StructureDefinition/indigenous-status"
	ExtAHPRARegistration = "http://hl7.org.au/fhir/StructureDefinition/ahpra-registration"
	ExtAdmissionDate     = "http://hl7.org/fhir/StructureDefinition/patient-admission-date"
	// Vaidshala-internal extensions (URI namespace under our control, used
	// to round-trip Vaidshala-specific fields without colliding with HL7 AU)
	ExtCareIntensity      = "https://vaidshala.health/fhir/StructureDefinition/care-intensity"
	ExtSDMReference       = "https://vaidshala.health/fhir/StructureDefinition/substitute-decision-maker"
	ExtRoleQualifications = "https://vaidshala.health/fhir/StructureDefinition/role-qualifications"
	ExtRoleEvidenceURL    = "https://vaidshala.health/fhir/StructureDefinition/role-evidence-url"

	// Identifier system URIs.
	SystemIHI                = "http://ns.electronichealth.net.au/id/hi/ihi/1.0"
	SystemHPII               = "http://ns.electronichealth.net.au/id/hi/hpii/1.0"
	SystemAHPRARegistration  = "http://hl7.org.au/id/ahpra-registration"
	SystemRoleKindCodeSystem = "https://vaidshala.health/fhir/CodeSystem/role-kind"
)

// Vaidshala FHIR extension URIs for MedicineUse v2-distinguishing fields.
// AU FHIR MedicationRequest does not have native equivalents for Intent /
// Target / StopCriteria, so these are encoded as Vaidshala-namespaced
// extensions on the resource.
const (
	ExtMedicineIntent       = "https://vaidshala.health/fhir/StructureDefinition/medicine-intent"
	ExtMedicineTarget       = "https://vaidshala.health/fhir/StructureDefinition/medicine-target"
	ExtMedicineStopCriteria = "https://vaidshala.health/fhir/StructureDefinition/medicine-stop-criteria"
	ExtMedicineAMTCode      = "https://vaidshala.health/fhir/StructureDefinition/amt-code"
)

// Vaidshala FHIR extension URIs for Observation v2-distinguishing fields.
// AU FHIR Observation has no native equivalents for Vaidshala's kind
// discriminator, source provenance reference, or computed Delta — these
// are encoded as Vaidshala-namespaced extensions on the resource.
const (
	ExtObservationKind     = "https://vaidshala.health/fhir/StructureDefinition/observation-kind"
	ExtObservationDelta    = "https://vaidshala.health/fhir/StructureDefinition/observation-delta"
	ExtObservationSourceID = "https://vaidshala.health/fhir/StructureDefinition/observation-source-id"
)

// Vaidshala FHIR extension URIs for Event v2-distinguishing fields.
// Clinical / care-transitions / administrative events map to AU FHIR
// Encounter; system events map to AU FHIR Communication. Neither
// resource has native equivalents for Vaidshala's event_type discriminator,
// severity classification, structured description, regulatory reportable_under
// list, related-entity refs, or triggered_state_changes — all encoded as
// Vaidshala-namespaced extensions on the resource.
const (
	ExtEventType                  = "https://vaidshala.health/fhir/StructureDefinition/event-type"
	ExtEventSeverity              = "https://vaidshala.health/fhir/StructureDefinition/event-severity"
	ExtEventDescriptionStructured = "https://vaidshala.health/fhir/StructureDefinition/event-description-structured"
	ExtEventReportableUnder       = "https://vaidshala.health/fhir/StructureDefinition/event-reportable-under"
	ExtEventRelatedObservations   = "https://vaidshala.health/fhir/StructureDefinition/event-related-observations"
	ExtEventRelatedMedicationUses = "https://vaidshala.health/fhir/StructureDefinition/event-related-medication-uses"
	ExtEventTriggeredStateChanges = "https://vaidshala.health/fhir/StructureDefinition/event-triggered-state-changes"
	ExtEventReportedBy            = "https://vaidshala.health/fhir/StructureDefinition/event-reported-by"
	ExtEventWitnessedBy           = "https://vaidshala.health/fhir/StructureDefinition/event-witnessed-by"
)

// SystemRouteCode is the FHIR-style code system URI for Vaidshala route
// values (ORAL, IV, IM, etc.) when serialized as Coding entries.
const SystemRouteCode = "https://vaidshala.health/fhir/CodeSystem/route"

// Vaidshala FHIR extension URIs for EvidenceTrace v2-distinguishing fields.
// EvidenceTrace nodes route to FHIR Provenance (clinical state machines) or
// AuditEvent (system state machines: Authorisation, Consent). Neither
// resource has native equivalents for Vaidshala's state-machine
// discriminator, state-change-type tag, structured reasoning summary, or
// the role-in-decision qualifier on inputs — all encoded as Vaidshala-
// namespaced extensions on the routed resource.
const (
	ExtEvidenceTraceStateMachine     = "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-state-machine"
	ExtEvidenceTraceStateChangeType  = "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-state-change-type"
	ExtEvidenceTraceReasoningSummary = "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-reasoning-summary"
	ExtEvidenceTraceInputRole        = "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-input-role"
	ExtEvidenceTraceAuthorityBasis   = "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-authority-basis"
	ExtEvidenceTraceResidentRef      = "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-resident-ref"
	ExtEvidenceTraceOccurredAt       = "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-occurred-at"
)
