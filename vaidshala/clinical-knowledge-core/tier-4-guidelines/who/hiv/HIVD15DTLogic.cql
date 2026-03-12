/**
Library: HIV.D15.DT Logic

@DecisionID: HIV.D15.DT
@BusinessRule: Determine WHO clinical staging of HIV disease in adults, adolescents and children
@Trigger: HIV.D15 Determine clinical stage of HIV
@HitPolicy: Rule Order
@Description: Determine WHO clinical staging of HIV disease in adults, adolescents and children

Data Concepts:
 * HIV.D.DE289: WHO HIV clinical stage condition or symptom
 * HIV.D.DE934: WHO HIV clinical stage condition or symptom
 * HIV.A.DE17: Age | Calculated age (number of years) of the client based on date of birth

Consolidated guidelines on HIV prevention, testing, treatment, service delivery and monitoring: recommendations for a public health approach (2021) Chapter 5: Advanced HIV Disease. Figure 5.1: Algorithm for providing a package of care for people with advanced HIV disease.
*/

library HIVD15DTLogic

using FHIR version '4.0.1'

include HIVCommon version '0.0.1' called HIC
include HIVConcepts called Concepts
include HIVEncounterElements called Elements
include FHIRHelpers version '4.0.1'

include WHOCommon called WCom

parameter Today default Today()

context Patient

/*
 @input: "WHO HIV clinical stage condition or symptom"
 */
define "WHO HIV clinical stage condition or symptom":
  if Elements."WHO HIV clinical stage condition or symptom D.DE289 Observation" is not null then
    Elements."WHO HIV clinical stage condition or symptom D.DE289 Observation"
  else if Elements."WHO HIV clinical stage condition or symptom D.DE934 Observation" is not null then
    Elements."WHO HIV clinical stage condition or symptom D.DE934 Observation"
  else
    null

/*
 @input: "Age"
 */
define "Age 15 or older":
  AgeInYearsAt(Today) >= 15

/*
 @input: "Age"
 */
define "Age under 15":
  AgeInYearsAt(Today) < 15

/*
 @input: "Age"
 */
define "Age over 6 weeks":
  AgeInWeeksAt(Today) >= 6

/*
 @input: "Age"
 */
define "Age over 1 month":
  AgeInMonthsAt(Today) > 1

/*
 @output: "HIV clinical stage 1"
 */
define "WHO HIV clinical stage 1":
  exists(
    "WHO HIV clinical stage condition or symptom" O
      where
        O.value ~ Concepts."Asymptomatic" or
        O.value ~ Concepts."Persistent generalized lymphadenopathy"
  )

/*
 @output: "HIV clinical stage 2"
 */
define "WHO HIV clinical stage 2":
  exists(
    "WHO HIV clinical stage condition or symptom" O
      where
        O.value ~ Concepts."Herpes zoster" or
        O.value ~ Concepts."Recurrent oral ulceration" or
        O.value ~ Concepts."Papular pruritic eruption" or
        O.value ~ Concepts."Fungal nail infections" or
        ("Age 15 or older" and (
          O.value ~ Concepts."Moderate unexplained weight loss" or
          O.value ~ Concepts."Recurrent respiratory tract infections" or
          O.value ~ Concepts."Angular cheilitis" or
          O.value ~ Concepts."Seborrhoeic dermatitis"
        )) or ("Age under 15" and (
          O.value ~ Concepts."Unexplained persistent hepatosplenomegaly" or
          O.value ~ Concepts."Linear gingival erythema" or
          O.value ~ Concepts."Extensive wart virus infection" or
          O.value ~ Concepts."Extensive molluscum contagiosum" or
          O.value ~ Concepts."Unexplained persistent parotid enlargement"
        ))
  )

/*
 @output: "HIV clinical stage 3"
 */
define "WHO HIV clinical stage 3":
  exists(
    "WHO HIV clinical stage condition or symptom" O
      where
        O.value ~ Concepts."Unexplained persistent fever above 37.5 C, intermittent or constant, for longer than one 1 month" or
        O.value ~ Concepts."Oral hairy leukoplakia" or
        O.value ~ Concepts."Pulmonary TB - HIV.D.DE312" or
        O.value ~ Concepts."Pulmonary TB - HIV.D.DE935" or
        O.value ~ Concepts."Acute necrotizing ulcerative gingivitis" or
        O.value ~ Concepts."Acute necrotizing ulcerative periodontitis" or
        O.value ~ Concepts."Unexplained anaemia <8 g/dL" or
        O.value ~ Concepts."Neutropaenia <0.5 x 10^9/L" or
        O.value ~ Concepts."Chronic thrombocytopaenia <50 x 10^9/L" or
        ("Age over 6 weeks" and O.value ~ Concepts."Persistent oral candidiasis") or
        ("Age 15 or older" and (
          O.value ~ Concepts."Severe bacterial infections such as pneumonia, empyema, pyomyositis, bone or joint infection, meningitis, bacteraemia" or
          O.value ~ Concepts."Unexplained severe weight loss in adults"
        ))
  )

/*
 @output: "HIV clinical stage 4"
 */
define "WHO HIV clinical stage 4":
  exists(
    "WHO HIV clinical stage condition or symptom" O
      where
        O.value ~ Concepts."Pneumocystis jirovecii pneumonia" or
        O.value ~ Concepts."Oesophageal candidiasis or candidiasis of trachea, bronchi or lungs" or
        O.value ~ Concepts."Extrapulmonary TB - HIV.D.DE338" or
        O.value ~ Concepts."Extrapulmonary TB - HIV.D.DE937" or
        O.value ~ Concepts."Kaposi sarcoma" or
        O.value ~ Concepts."Cytomegalovirus infection retinitis or infection of other organs" or
        O.value ~ Concepts."Central nervous system toxoplasmosis" or
        O.value ~ Concepts."HIV encephalopathy" or
        O.value ~ Concepts."Extrapulmonary cryptococcosis, including meningitis" or
        O.value ~ Concepts."Disseminated nontuberculous mycobacterial infection" or
        O.value ~ Concepts."Progressive multifocal leukoencephalopathy" or
        O.value ~ Concepts."Chronic isosporiasis" or
        O.value ~ Concepts."Cerebral lymphoma" or
        O.value ~ Concepts."B-cell non-Hodgkin lymphoma" or
        O.value ~ Concepts."HIV-associated nephropathy or cardiomyopathy" or
        ("Age over 1 month" and (
          O.value ~ Concepts."Cytomegalovirus infection retinitis or infection of other organs" or
          O.value ~ Concepts."Central nervous system toxoplasmosis"
        )) or
        ("Age 15 or older" and (
          O.value ~ Concepts."Chronic herpes simplex infection orolabial, genital or anorectal of more than 1 month in duration or visceral at any site" or
          O.value ~ Concepts."Disseminated mycosis extrapulmonary histoplasmosis, coccidioidomycosis" or
          O.value ~ Concepts."HIV wasting syndrome" or
          O.value ~ Concepts."Recurrent severe bacterial pneumonia" or
          O.value ~ Concepts."Chronic cryptosporidiosis" or
          O.value ~ Concepts."Recurrent septicaemia including nontyphoidal Salmonella" or
          O.value ~ Concepts."Invasive cervical carcinoma" or
          O.value ~ Concepts."Atypical disseminated leishmaniasis" 
        )) or
        ("Age under 15" and (
          O.value ~ Concepts."Chronic herpes simplex infection orolabial or cutaneous of more than 1 month duration or visceral at any site" or
          O.value ~ Concepts."Disseminated endemic mycosis extrapulmonary histoplasmosis, coccidioidomycosis, penicilliosis" or
          O.value ~ Concepts."Unexplained severe wasting not responding to standard therapy" or
          O.value ~ Concepts."Unexplained stunting not responding to standard therapy" or
          O.value ~ Concepts."Unexplained severe malnutrition not responding to standard therapy" or
          O.value ~ Concepts."Recurrent severe bacterial infections such as empyema, pyomyositis, bone or joint infection, meningitis, but excluding pneumonia" or
          O.value ~ Concepts."Chronic cryptosporidiosis with diarrhoea"
        ))
  )

/*
 @dynamicValue: Guidance
 */
define "Guidance":
  Combine({ 
    if "WHO HIV clinical stage 1" 
      then 'WHO HIV clinical stage 1'
      else '',
    if "WHO HIV clinical stage 2"
      then 'WHO HIV clinical stage 2'
      else '',
    if "WHO HIV clinical stage 3"
      then 'WHO HIV clinical stage 3'
      else '',
    if "WHO HIV clinical stage 3"
      then 'WHO HIV clinical stage 4'
      else ''
  })
