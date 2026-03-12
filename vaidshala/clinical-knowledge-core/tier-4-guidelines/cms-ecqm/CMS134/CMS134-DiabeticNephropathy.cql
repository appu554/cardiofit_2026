library KidneyHealthEvaluationFHIR version '0.1.000'

using QICore version '4.1.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include SupplementalDataElements version '3.5.000' called SDE
include CQMCommon version '2.2.000' called CQMCommon
include Hospice version '6.12.000' called Hospice
include PalliativeCare version '1.11.000' called PalliativeCare
include QICoreCommon version '2.1.000' called QICoreCommon

codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "ActCode": 'http://terminology.hl7.org/CodeSystem/v3-ActCode'
codesystem "ConditionClinicalStatusCodes": 'http://terminology.hl7.org/CodeSystem/condition-clinical'

// KB-7 Terminology Service (localhost:8087) - uses exact PostgreSQL name column for lookup
valueset "Acute Inpatient": 'http://localhost:8087/fhir/ValueSet/Acute Inpatient'
valueset "Annual Wellness Visit": 'http://localhost:8087/fhir/ValueSet/Annual Wellness Visit'
valueset "Chronic Kidney Disease, Stage 5": 'http://localhost:8087/fhir/ValueSet/Chronic Kidney Disease, Stage 5'
valueset "Diabetes": 'http://localhost:8087/fhir/ValueSet/Diabetes'
valueset "Encounter Inpatient": 'http://localhost:8087/fhir/ValueSet/Encounter Inpatient'
valueset "End Stage Renal Disease": 'http://localhost:8087/fhir/ValueSet/ESRD'
valueset "Estimated Glomerular Filtration Rate": 'http://localhost:8087/fhir/ValueSet/Estimated Glomerular Filtration Rate'
valueset "Home Healthcare Services": 'http://localhost:8087/fhir/ValueSet/Home Healthcare Services'
valueset "Hospice Care Ambulatory": 'http://localhost:8087/fhir/ValueSet/Hospice Care Ambulatory'
valueset "Office Visit": 'http://localhost:8087/fhir/ValueSet/Office Visit'
valueset "Outpatient Consultation": 'http://localhost:8087/fhir/ValueSet/Outpatient Consultation'
valueset "Palliative or Hospice Care": 'http://localhost:8087/fhir/ValueSet/Palliative or Hospice Care'
valueset "Preventive Care Services Established Office Visit, 18 and Up": 'http://localhost:8087/fhir/ValueSet/Preventive Care Services Established Office Visit, 18 and Up'
valueset "Preventive Care Services Initial Office Visit, 18 and Up": 'http://localhost:8087/fhir/ValueSet/Preventive Care Services Initial Office Visit, 18 and Up'
valueset "Telephone Visits": 'http://localhost:8087/fhir/ValueSet/Telephone Visits'
valueset "Urine Albumin Creatinine Ratio": 'http://localhost:8087/fhir/ValueSet/Urine Albumin Creatinine Ratio'

code "Discharge to healthcare facility for hospice care (procedure)": '428371000124100' from "SNOMEDCT" display 'Discharge to healthcare facility for hospice care (procedure)'
code "Discharge to home for hospice care (procedure)": '428361000124107' from "SNOMEDCT" display 'Discharge to home for hospice care (procedure)'
code "AMB": 'AMB' from "ActCode" display 'Ambulatory'
code "active": 'active' from "ConditionClinicalStatusCodes"

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2025-01-01T00:00:00.000Z, @2025-12-31T23:59:59.999Z]

context Patient

define "Initial Population":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[18, 85]
    and "Has Active Diabetes Overlaps Measurement Period"
    and "Has Outpatient Visit During Measurement Period"

define "Denominator":
  "Initial Population"

define "Denominator Exclusions":
  exists "Has CKD Stage 5 or ESRD Diagnosis Overlaps Measurement Period"
    or Hospice."Has Hospice Services"
    or PalliativeCare."Has Palliative Care in the Measurement Period"

define "Numerator":
  "Has Kidney Panel Performed During Measurement Period"

define "Has Kidney Panel Performed During Measurement Period":
  exists ( ["Observation": "Estimated Glomerular Filtration Rate"] eGFRTest
      where eGFRTest.effective.toInterval ( ) during day of "Measurement Period"
        and eGFRTest.value is not null
        and eGFRTest.status in { 'final', 'amended', 'corrected' }
  )
    and exists ( ["Observation": "Urine Albumin Creatinine Ratio"] uACRTest
        where uACRTest.effective.toInterval ( ) during day of "Measurement Period"
          and uACRTest.value is not null
          and uACRTest.status in { 'final', 'amended', 'corrected' }
    )

define "Has Outpatient Visit During Measurement Period":
  exists ( ( ["Encounter": "Annual Wellness Visit"]
      union ["Encounter": "Home Healthcare Services"]
      union ["Encounter": "Office Visit"]
      union ["Encounter": "Outpatient Consultation"]
      union ["Encounter": "Preventive Care Services Established Office Visit, 18 and Up"]
      union ["Encounter": "Preventive Care Services Initial Office Visit, 18 and Up"]
      union ["Encounter": "Telephone Visits"] ) ValidEncounter
      where ValidEncounter.period during "Measurement Period"
        and ValidEncounter.status = 'finished'
  )

define "Has Active Diabetes Overlaps Measurement Period":
  exists ( ["Condition": "Diabetes"] Diabetes
      where Diabetes.prevalenceInterval ( ) overlaps day of "Measurement Period"
        and Diabetes.clinicalStatus ~ "active"
  )

define "Has CKD Stage 5 or ESRD Diagnosis Overlaps Measurement Period":
  ( ["Condition": "Chronic Kidney Disease, Stage 5"]
    union ["Condition": "End Stage Renal Disease"] ) CKDOrESRD
    where CKDOrESRD.prevalenceInterval ( ) overlaps day of "Measurement Period"
      and CKDOrESRD.clinicalStatus ~ "active"

define "SDE Ethnicity":
  SDE."SDE Ethnicity"

define "SDE Payer":
  SDE."SDE Payer"

define "SDE Race":
  SDE."SDE Race"

define "SDE Sex":
  SDE."SDE Sex"