library AuthorisationHelpers_test version '1.0'

using FHIR version '4.0.1'

include AuthorisationHelpers version '1.0' called Auth

// TODO(wave-1-runtime): execute under HAPI Test Engine.

context Patient

define FacilityWithCardioRenal: 'fixture:facility-cardio-renal-prescriber'
define FacilityNoCardioRenal: 'fixture:facility-no-cardio-renal'

// 1. HasAvailablePrescriberForClass true
define Test_HasAvailablePrescriberForClass_True:
  Auth."HasAvailablePrescriberForClass"(
    FacilityWithCardioRenal, 'Cardio_renal_prescriber') = true

// 2. HasAvailablePrescriberForClass false
define Test_HasAvailablePrescriberForClass_False:
  Auth."HasAvailablePrescriberForClass"(
    FacilityNoCardioRenal, 'Cardio_renal_prescriber') = false

// 3. FallbackRouting urgent → ED_transfer
define Test_FallbackRouting_Urgent_ED:
  Auth."FallbackRouting"('Insulin_dose_adjustment', 'urgent') = 'ED_transfer'

// 4. FallbackRouting non-urgent insulin → telehealth
define Test_FallbackRouting_Insulin_NonUrgent:
  Auth."FallbackRouting"('Insulin_dose_adjustment', 'routine') = 'telehealth'
