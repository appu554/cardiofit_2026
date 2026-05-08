library MonitoringHelpers_test version '1.0'

using FHIR version '4.0.1'

include MonitoringHelpers version '1.0' called Mon

// TODO(plan-0.5-followup): wired runtime requires (a) cqf-fhir-cr engine
// integration in $evaluate-rule (Task 5 followup) and (b) registration of
// unregistered Substrate.* symbols (see Plan-0.5-Task-8 audit).

context Patient

define ResidentWarfarinPlan: 'fixture:resident-warfarin-inr-plan'
define ResidentNoPlans: 'fixture:resident-no-monitoring-plans'
define WarfarinPlanRef: 'fixture:plan-warfarin-inr'

// 1. MonitoringPlanIsActive true
define Test_MonitoringPlanIsActive_True:
  Mon."MonitoringPlanIsActive"(WarfarinPlanRef, Now()) = true

// 2. HasOpenPlanOfType warfarin true
define Test_HasOpenPlanOfType_Warfarin:
  Mon."HasOpenPlanOfType"(ResidentWarfarinPlan, 'warfarin_inr') = true

// 3. HasOpenPlanOfType none
define Test_HasOpenPlanOfType_None:
  Mon."HasOpenPlanOfType"(ResidentNoPlans, 'warfarin_inr') = false

// 4. ObservationOverdueBy >24h
define Test_ObservationOverdueBy_24h:
  Mon."ObservationOverdueBy"(WarfarinPlanRef, 'INR', Now()) > 24
