library ClinicalStateHelpers_test version '1.0'

using FHIR version '4.0.1'

include ClinicalStateHelpers version '1.0' called State

// TODO(wave-1-runtime): execute under HAPI Test Engine.

context Patient

define ResidentBaselineKPlus: 'fixture:resident-baseline-k-4.5'
define ResidentTrajectoryUp: 'fixture:resident-k-trajectory-up-0.9'
define ResidentPalliative: 'fixture:resident-care-intensity-palliative'
define ResidentWithGiBleed: 'fixture:resident-active-concern-gi-bleed'

// 1. BaselineFor potassium
define Test_BaselineFor_KPlus_4_5:
  State."BaselineFor"(ResidentBaselineKPlus, 'potassium') = 4.5

// 2. DeltaFromBaseline > 0.8 over 7 days
define Test_DeltaFromBaseline_KUp_0_9:
  State."DeltaFromBaseline"(ResidentTrajectoryUp, 'potassium', 7) >= 0.8

// 3. IsPalliative true
define Test_IsPalliative_True:
  State."IsPalliative"(ResidentPalliative) = true

// 4. HasActiveConcernType GI_BLEED_WATCH_14D
define Test_HasActiveConcernType_GI:
  State."HasActiveConcernType"(ResidentWithGiBleed, 'GI_BLEED_WATCH_14D') = true
