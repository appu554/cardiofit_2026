library AuthorisationHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * AuthorisationHelpers
 * ====================
 *
 * Fan-outs to the Authorisation evaluator (port 8138, planned
 * Wave 3) and ScopeRule engine (port 8139, planned Wave 4).
 *
 * PRE-WAVE-3 DEFAULT POLICY
 * -------------------------
 * Until the Authorisation evaluator is deployed, helpers in this
 * file return `null`. Per the Wave 0 helper-surface spec
 * recommendation, `null` forces rules to handle the
 * "evaluator down" case explicitly. Wave 1 validator lints for
 * unhandled-null returns from these helpers.
 *
 * TODO(wave-1-runtime): wire to Authorisation evaluator under HAPI.
 *
 * Spec: docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md
 *       (File 4 — AuthorisationHelpers).
 */

// Latency budget for all helpers in this file: <60ms p95.

// RoleCanPrescribeFor — role × drug class lookup.
define function "RoleCanPrescribeFor"(roleRef String, drugClass String):
  Substrate.AuthRoleCanPrescribe(roleRef, drugClass)

// AuthorisationIsActive — Authorisation seam.
define function "AuthorisationIsActive"(
  authorisationRef String, asOf DateTime
):
  Substrate.AuthorisationActiveAt(authorisationRef, asOf)

// ScopeRulePermits — ScopeRule engine.
define function "ScopeRulePermits"(
  action String, jurisdiction String, asOf DateTime
):
  Substrate.ScopeRulePermits(action, jurisdiction, asOf)

// AvailablePrescriberForClass — credential cache lookup.
define function "AvailablePrescriberForClass"(
  facilityRef String, drugClass String
):
  Substrate.AvailablePrescribers(facilityRef, drugClass)

// HasAvailablePrescriberForClass — sugar over AvailablePrescriberForClass.
define function "HasAvailablePrescriberForClass"(
  facilityRef String, drugClass String
):
  Count(Substrate.AvailablePrescribers(facilityRef, drugClass)) > 0

// FallbackRouting — pure routing table; one of
// {'telehealth', 'ED_transfer', 'after_hours_GP'}. Latency: <30ms.
define function "FallbackRouting"(drugClass String, urgency String):
  Case
    when urgency = 'urgent' then 'ED_transfer'
    when drugClass = 'Insulin_dose_adjustment' then 'telehealth'
    else 'after_hours_GP'
  end

// FacilityHasReverseAgentFor — Authorisation evaluator inventory.
// Latency: <50ms p95.
define function "FacilityHasReverseAgentFor"(
  facilityRef String, drugClass String
):
  Substrate.FacilityHasReverseAgent(facilityRef, drugClass)
