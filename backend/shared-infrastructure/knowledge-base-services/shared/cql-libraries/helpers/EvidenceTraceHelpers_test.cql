library EvidenceTraceHelpers_test version '1.0'

using FHIR version '4.0.1'

include EvidenceTraceHelpers version '1.0' called Trace

// TODO(wave-1-runtime): execute under HAPI Test Engine.

context Patient

define ResidentWithFires: 'fixture:resident-ppi-fires-90d'
define ResidentNoFires: 'fixture:resident-no-rule-fires'
define TraceNodeRoot: 'fixture:trace-node-root'

// 1. RuleFiredFor positive
define Test_RuleFiredFor_Positive:
  Count(Trace."RuleFiredFor"(
    ResidentWithFires,
    'PPI_LONG_TERM_NO_INDICATION',
    Interval[Now() - 90 day, Now()])) > 0

// 2. RuleFiredFor negative
define Test_RuleFiredFor_Negative:
  Count(Trace."RuleFiredFor"(
    ResidentNoFires,
    'PPI_LONG_TERM_NO_INDICATION',
    Interval[Now() - 90 day, Now()])) = 0

// 3. LineageOf returns lineage list
define Test_LineageOf_NonEmpty:
  Count(Trace."LineageOf"(TraceNodeRoot, 2)) > 0
