library ControllingHighBloodPressureFHIR version '0.1.000'

using QICore version '4.1.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include QICoreCommon version '2.1.000' called QICoreCommon
include SupplementalDataElements version '3.5.000' called SDE
include Status version '1.8.000' called Status
include AdvancedIllnessandFrailty version '1.16.000' called AIFrailLTCF
include AdultOutpatientEncounters version '4.11.000' called AdultOutpatientEncounters
include Hospice version '6.12.000' called Hospice
include PalliativeCare version '1.11.000' called PalliativeCare

codesystem "LOINC": 'http://loinc.org'

// KB-7 Terminology Service (localhost:8087) - uses exact PostgreSQL name column for lookup
valueset "Chronic Kidney Disease, Stage 5": 'http://localhost:8087/fhir/ValueSet/Chronic Kidney Disease, Stage 5'
valueset "Dialysis Services": 'http://localhost:8087/fhir/ValueSet/Dialysis Services'
// External CTS - not yet available in KB-7
valueset "Emergency Department Evaluation and Management Visit": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.101.12.1010'
valueset "Encounter Inpatient": 'http://localhost:8087/fhir/ValueSet/Encounter Inpatient'
valueset "End Stage Renal Disease": 'http://localhost:8087/fhir/ValueSet/ESRD'
valueset "ESRD Monthly Outpatient Services": 'http://localhost:8087/fhir/ValueSet/ESRD Monthly Outpatient Services'
valueset "Essential Hypertension": 'http://localhost:8087/fhir/ValueSet/Essential Hypertension'
valueset "Kidney Transplant": 'http://localhost:8087/fhir/ValueSet/Kidney Transplant'
valueset "Kidney Transplant Recipient": 'http://localhost:8087/fhir/ValueSet/Kidney Transplant Recipient'
valueset "Pregnancy": 'http://localhost:8087/fhir/ValueSet/Pregnancy'

code "Diastolic blood pressure": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "Systolic blood pressure": '8480-6' from "LOINC" display 'Systolic blood pressure'

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2025-01-01T00:00:00.000Z, @2025-12-31T23:59:59.999Z]

context Patient

define "Initial Population":
  AgeInYearsAt(date from 
    end of "Measurement Period"
  ) in Interval[18, 85]
    and exists "Essential Hypertension Diagnosis"
    and exists AdultOutpatientEncounters."Qualifying Encounters"

define "Essential Hypertension Diagnosis":
  ( [Condition: "Essential Hypertension"] ) Hypertension
    where Hypertension.prevalenceInterval ( ) overlaps Interval[start of "Measurement Period", start of "Measurement Period" + 6 months )

define "Denominator":
  "Initial Population"

define "Denominator Exclusions":
  Hospice."Has Hospice Services"
    or exists ( "Pregnancy or Renal Diagnosis" )
    or exists ( "End Stage Renal Disease Procedures" )
    or exists ( "End Stage Renal Disease Encounter" )
    or AIFrailLTCF."Is Age 66 to 80 with Advanced Illness and Frailty or Is Age 81 or Older with Frailty"
    or AIFrailLTCF."Is Age 66 or Older Living Long Term in a Nursing Home"
    or PalliativeCare."Has Palliative Care in the Measurement Period"

define "Pregnancy or Renal Diagnosis":
  ( [Condition: "Pregnancy"]
    union [Condition: "End Stage Renal Disease"]
    union [Condition: "Kidney Transplant Recipient"]
    union [Condition: "Chronic Kidney Disease, Stage 5"] ) PregnancyESRDDiagnosis
    where PregnancyESRDDiagnosis.prevalenceInterval ( ) overlaps "Measurement Period"

define "End Stage Renal Disease Procedures":
  ( ( [Procedure: "Kidney Transplant"]
      union [Procedure: "Dialysis Services"]
  ).isProcedurePerformed ( ) ) ESRDProcedure
    where ESRDProcedure.performed.toInterval ( ) ends on or before end of "Measurement Period"

define "End Stage Renal Disease Encounter":
  ( ( [Encounter: "ESRD Monthly Outpatient Services"] ).isEncounterPerformed ( ) ) ESRDEncounter
    where ESRDEncounter.period.toInterval ( ) starts on or before end of "Measurement Period"

define "Numerator":
  "Has Systolic Blood Pressure Less Than 140"
    and "Has Diastolic Blood Pressure Less Than 90"

define "Has Systolic Blood Pressure Less Than 140":
  "Lowest Systolic Reading on Most Recent Blood Pressure Day" < 140 'mm[Hg]'

define "Has Diastolic Blood Pressure Less Than 90":
  "Lowest Diastolic Reading on Most Recent Blood Pressure Day" < 90 'mm[Hg]'

define "Lowest Systolic Reading on Most Recent Blood Pressure Day":
  First("Qualifying Systolic Blood Pressure Reading" SBPReading
      where SBPReading.effective.latest() same day as "Most Recent Blood Pressure Day"
      return singleton from(SBPReading.component SBPComponent
          where SBPComponent.code ~ "Systolic blood pressure"
          return SBPComponent.value as Quantity
      )
      sort asc
  )

define "Lowest Diastolic Reading on Most Recent Blood Pressure Day":
  First("Qualifying Diastolic Blood Pressure Reading" DBPReading
      where DBPReading.effective.latest() same day as "Most Recent Blood Pressure Day"
      return singleton from(DBPReading.component DBPComponent
          where DBPComponent.code ~ "Diastolic blood pressure"
          return DBPComponent.value as Quantity
      )
      sort asc
  )

define "Qualifying Systolic Blood Pressure Reading":
  ( ( ["observation-bp"] ).isObservationBP ( ) ) BloodPressure
    where ( not ( ( BloodPressure.encounter.getEncounter ( ) ).class.code in { 'EMER', 'IMP', 'ACUTE', 'NONAC', 'PRENC', 'SS' } ) )
      and BloodPressure.effective.latest ( ) during day of "Measurement Period"

define "Qualifying Diastolic Blood Pressure Reading":
  ( ( ["observation-bp"] ).isObservationBP ( ) ) BloodPressure
    where ( not ( ( BloodPressure.encounter.getEncounter ( ) ).class.code in { 'EMER', 'IMP', 'ACUTE', 'NONAC', 'PRENC', 'SS' } ) )
      and BloodPressure.effective.latest ( ) during day of "Measurement Period"

define fluent function getEncounter(reference Reference):
  singleton from ( [Encounter] E
      where E.id = reference.reference.getId ( )
  )

define "Most Recent Blood Pressure Day":
  Last("Blood Pressure Days" BPDays
      sort asc
  )

define "Blood Pressure Days":
  ( "Qualifying Diastolic Blood Pressure Reading" DBPExam
      return date from DBPExam.effective.latest ( )
  )
    intersect ( "Qualifying Systolic Blood Pressure Reading" SBPExam
        return date from SBPExam.effective.latest ( )
    )

define "SDE Ethnicity":
  SDE."SDE Ethnicity"

define "SDE Payer":
  SDE."SDE Payer"

define "SDE Race":
  SDE."SDE Race"

define "SDE Sex":
  SDE."SDE Sex"