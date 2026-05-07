library MedicationHelpers_test version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds

// TODO(wave-1-runtime): execute under HAPI Test Engine when the
// Substrate adapter library lands. Until then this file is a static
// fixture-binding spec; the validator inspects it for define-coverage
// rather than evaluating the CQL.

context Patient

// Fixtures (resident IDs are placeholders matching the JSON fixture
// loader convention; the runtime will resolve to real Patient
// resources).
define ResidentWithLongPpi: 'fixture:resident-ppi-60d'
define ResidentNoActiveMeds: 'fixture:resident-no-meds'
define ResidentWithOpioids: 'fixture:resident-opioids-30mg-mEdd'

// 1. HasActiveAtcPrefix positive
define Test_HasActiveAtcPrefix_PpiPositive:
  Meds."HasActiveAtcPrefix"(ResidentWithLongPpi, 'A02BC') = true

// 2. HasActiveAtcPrefix negative
define Test_HasActiveAtcPrefix_NoMatch:
  Meds."HasActiveAtcPrefix"(ResidentNoActiveMeds, 'A02BC') = false

// 3. IsPrescribingOmission discriminator
define Test_IsPrescribingOmission_StartV3:
  Meds."IsPrescribingOmission"('START_V3') = true

// 4. IsPrescribingOmission negative
define Test_IsPrescribingOmission_StoppV3:
  Meds."IsPrescribingOmission"('STOPP_V3') = false

// 5. MorphineEquivalentMgPerDay
define Test_MorphineEquivalentMgPerDay_30mg:
  Meds."MorphineEquivalentMgPerDay"(ResidentWithOpioids) = 30
