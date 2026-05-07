library ConsentStateHelpers_test version '1.0'

using FHIR version '4.0.1'

include ConsentStateHelpers version '1.0' called Consent

// TODO(wave-1-runtime): execute under HAPI Test Engine.

context Patient

define ResidentWithActiveConsent: 'fixture:resident-antipsy-consent-active'
define ResidentNoConsent: 'fixture:resident-antipsy-no-consent'
define ResidentConsentExpiring: 'fixture:resident-antipsy-consent-expiring-7d'

// 1. HasActiveConsentForClass true
define Test_HasActiveConsentForClass_True:
  Consent."HasActiveConsentForClass"(
    ResidentWithActiveConsent, 'Antipsychotic_deprescribe_review') = true

// 2. HasActiveConsentForClass false (no consent record)
define Test_HasActiveConsentForClass_False:
  Consent."HasActiveConsentForClass"(
    ResidentNoConsent, 'Antipsychotic_deprescribe_review') = false

// 3. ConsentExpiringWithin 14d true
define Test_ConsentExpiringWithin_14d:
  Consent."ConsentExpiringWithin"(
    ResidentConsentExpiring, 'Antipsychotic_deprescribe_review', 14) = true

// 4. NeedsConsentRefresh composition
define Test_NeedsConsentRefresh_True:
  Consent."NeedsConsentRefresh"(
    ResidentConsentExpiring, 'Antipsychotic_deprescribe_review') = true
