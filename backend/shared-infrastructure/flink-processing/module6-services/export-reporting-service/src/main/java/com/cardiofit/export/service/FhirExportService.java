package com.cardiofit.export.service;

import ca.uhn.fhir.context.FhirContext;
import ca.uhn.fhir.parser.IParser;
import com.cardiofit.export.model.PatientCurrentState;
import org.hl7.fhir.r4.model.*;
import org.springframework.stereotype.Service;

import java.util.Calendar;
import java.util.Date;

/**
 * Service for FHIR format export
 * Converts patient data to HL7 FHIR R4 format
 */
@Service
public class FhirExportService {

    private final FhirContext fhirContext;
    private final IParser jsonParser;

    public FhirExportService() {
        this.fhirContext = FhirContext.forR4();
        this.jsonParser = fhirContext.newJsonParser().setPrettyPrint(true);
    }

    /**
     * Export patient data as FHIR Bundle
     */
    public String exportPatientToFhir(PatientCurrentState patient) {
        Bundle bundle = new Bundle();
        bundle.setType(Bundle.BundleType.COLLECTION);
        bundle.setId(patient.getPatientId());
        bundle.setTimestamp(new Date());

        // Add Patient resource
        Patient fhirPatient = createPatientResource(patient);
        bundle.addEntry()
                .setFullUrl("urn:uuid:" + patient.getPatientId())
                .setResource(fhirPatient);

        // Add Observation resource for risk score
        Observation riskObservation = createRiskScoreObservation(patient);
        bundle.addEntry()
                .setFullUrl("urn:uuid:risk-" + patient.getPatientId())
                .setResource(riskObservation);

        // Add Encounter resource
        Encounter encounter = createEncounter(patient);
        bundle.addEntry()
                .setFullUrl("urn:uuid:encounter-" + patient.getPatientId())
                .setResource(encounter);

        // Convert to JSON string
        return jsonParser.encodeResourceToString(bundle);
    }

    /**
     * Create FHIR Patient resource
     */
    private Patient createPatientResource(PatientCurrentState patient) {
        Patient fhirPatient = new Patient();
        fhirPatient.setId(patient.getPatientId());

        // Name
        HumanName name = new HumanName();
        name.setText(patient.getPatientName());
        fhirPatient.addName(name);

        // Gender
        if (patient.getGender() != null) {
            switch (patient.getGender().toLowerCase()) {
                case "male":
                    fhirPatient.setGender(Enumerations.AdministrativeGender.MALE);
                    break;
                case "female":
                    fhirPatient.setGender(Enumerations.AdministrativeGender.FEMALE);
                    break;
                default:
                    fhirPatient.setGender(Enumerations.AdministrativeGender.UNKNOWN);
            }
        }

        // Birth date (estimated from age)
        if (patient.getAge() != null) {
            Date birthDate = calculateBirthDate(patient.getAge());
            fhirPatient.setBirthDate(birthDate);
        }

        return fhirPatient;
    }

    /**
     * Create FHIR Observation for risk score
     */
    private Observation createRiskScoreObservation(PatientCurrentState patient) {
        Observation observation = new Observation();
        observation.setId("risk-" + patient.getPatientId());
        observation.setStatus(Observation.ObservationStatus.FINAL);

        // Category
        CodeableConcept category = new CodeableConcept();
        category.addCoding()
                .setSystem("http://terminology.hl7.org/CodeSystem/observation-category")
                .setCode("survey")
                .setDisplay("Survey");
        observation.addCategory(category);

        // Code
        CodeableConcept code = new CodeableConcept();
        code.setText("Overall Risk Score");
        observation.setCode(code);

        // Subject
        Reference subject = new Reference();
        subject.setReference("Patient/" + patient.getPatientId());
        observation.setSubject(subject);

        // Value
        if (patient.getOverallRiskScore() != null) {
            Quantity value = new Quantity();
            value.setValue(patient.getOverallRiskScore());
            value.setUnit("score");
            observation.setValue(value);
        }

        // Interpretation
        if (patient.getRiskCategory() != null) {
            CodeableConcept interpretation = new CodeableConcept();
            interpretation.setText(patient.getRiskCategory());
            observation.addInterpretation(interpretation);
        }

        return observation;
    }

    /**
     * Create FHIR Encounter resource
     */
    private Encounter createEncounter(PatientCurrentState patient) {
        Encounter encounter = new Encounter();
        encounter.setId("encounter-" + patient.getPatientId());
        encounter.setStatus(Encounter.EncounterStatus.INPROGRESS);

        // Class
        Coding classCode = new Coding();
        classCode.setSystem("http://terminology.hl7.org/CodeSystem/v3-ActCode");
        classCode.setCode("IMP");
        classCode.setDisplay("inpatient encounter");
        encounter.setClass_(classCode);

        // Subject
        Reference subject = new Reference();
        subject.setReference("Patient/" + patient.getPatientId());
        encounter.setSubject(subject);

        // Period (admission time)
        if (patient.getAdmissionTime() != null) {
            Period period = new Period();
            period.setStart(new Date(patient.getAdmissionTime()));
            encounter.setPeriod(period);
        }

        // Location
        if (patient.getRoom() != null) {
            Encounter.EncounterLocationComponent location = new Encounter.EncounterLocationComponent();
            Reference locationRef = new Reference();
            locationRef.setDisplay(patient.getRoom());
            location.setLocation(locationRef);
            encounter.addLocation(location);
        }

        return encounter;
    }

    /**
     * Calculate birth date from age
     */
    private Date calculateBirthDate(int age) {
        Calendar calendar = Calendar.getInstance();
        calendar.add(Calendar.YEAR, -age);
        calendar.set(Calendar.MONTH, Calendar.JANUARY);
        calendar.set(Calendar.DAY_OF_MONTH, 1);
        return calendar.getTime();
    }
}
