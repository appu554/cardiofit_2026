// PRN Escalation Velocity — CAPE-canonical CQL definition
// Source: docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md lines 271–290
// This file is NOT executed by any Go runtime here. It exists so the clinical
// informatics lead can review the canonical specification independent of the
// Go implementation in compute.go. The Go implementation is the runtime
// authority; the CQL is the human-readable spec.
//
// Phase 1 instantiations (CAPE Guidelines lines 569–571):
//   - PRN_benzodiazepine_escalation_velocity
//   - PRN_antipsychotic_escalation_velocity
//   - PRN_analgesic_escalation_velocity

define "PRN Escalation Velocity":
  let recent_30d = count of PRN administrations of class X in last 30 days
  let baseline_90d = average of PRN administrations of class X per 30 days,
                     prior 90 days
  let velocity_ratio = recent_30d / baseline_90d
  in
    case
      when velocity_ratio > 4.0 then 5  // 400%+ increase
      when velocity_ratio > 2.5 then 4  // 250%+ increase
      when velocity_ratio > 1.5 then 3  // 150%+ increase
      when velocity_ratio > 1.0 then 2  // any increase
      else 1
    end
