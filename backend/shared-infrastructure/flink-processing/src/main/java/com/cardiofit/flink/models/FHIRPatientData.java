package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.JsonNode;
import java.io.Serializable;
import java.time.LocalDate;
import java.time.Period;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;

/**
 * Data transfer object for Google Healthcare FHIR API Patient resource responses.
 *
 * This class parses FHIR R4 Patient resources from Google Cloud Healthcare API
 * and extracts relevant fields for the PatientSnapshot.
 */
public class FHIRPatientData implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("id")
    private String id; // FHIR resource ID

    @JsonProperty("firstName")
    private String firstName;

    @JsonProperty("lastName")
    private String lastName;

    @JsonProperty("dateOfBirth")
    private String dateOfBirth; // ISO 8601 format (YYYY-MM-DD)

    @JsonProperty("gender")
    private String gender; // male, female, other, unknown

    @JsonProperty("age")
    private Integer age;

    @JsonProperty("mrn")
    private String mrn; // Medical Record Number from identifier

    @JsonProperty("allergies")
    private List<String> allergies = new ArrayList<>();

    public FHIRPatientData() {
    }

    /**
     * Parse FHIR Patient resource JSON from Google Healthcare API.
     *
     * Example FHIR Patient resource:
     * {
     *   "resourceType": "Patient",
     *   "id": "P12345",
     *   "identifier": [{"type": {"coding": [{"code": "MR"}]}, "value": "MRN123"}],
     *   "name": [{"family": "Doe", "given": ["John"]}],
     *   "gender": "male",
     *   "birthDate": "1980-01-15"
     * }
     */
    public static FHIRPatientData fromFHIRResource(JsonNode fhirPatient) {
        if (fhirPatient == null) {
            return null;
        }

        FHIRPatientData data = new FHIRPatientData();

        // Extract ID
        data.id = fhirPatient.has("id") ? fhirPatient.get("id").asText() : null;

        // Extract name (use first name entry)
        if (fhirPatient.has("name") && fhirPatient.get("name").isArray()) {
            JsonNode nameArray = fhirPatient.get("name");
            if (nameArray.size() > 0) {
                JsonNode name = nameArray.get(0);

                // Family name (last name)
                if (name.has("family")) {
                    data.lastName = name.get("family").asText();
                }

                // Given names (first name)
                if (name.has("given") && name.get("given").isArray()) {
                    JsonNode givenArray = name.get("given");
                    if (givenArray.size() > 0) {
                        data.firstName = givenArray.get(0).asText();
                    }
                }
            }
        }

        // Extract gender
        if (fhirPatient.has("gender")) {
            data.gender = fhirPatient.get("gender").asText();
        }

        // Extract birth date and calculate age
        if (fhirPatient.has("birthDate")) {
            data.dateOfBirth = fhirPatient.get("birthDate").asText();
            data.age = calculateAge(data.dateOfBirth);
        }

        // Extract MRN from identifiers
        if (fhirPatient.has("identifier") && fhirPatient.get("identifier").isArray()) {
            JsonNode identifiers = fhirPatient.get("identifier");
            for (JsonNode identifier : identifiers) {
                // Look for identifier with type = "MR" (Medical Record)
                if (identifier.has("type")) {
                    JsonNode type = identifier.get("type");
                    if (type.has("coding") && type.get("coding").isArray()) {
                        for (JsonNode coding : type.get("coding")) {
                            if (coding.has("code") && "MR".equals(coding.get("code").asText())) {
                                if (identifier.has("value")) {
                                    data.mrn = identifier.get("value").asText();
                                    break;
                                }
                            }
                        }
                    }
                }
            }
        }

        // TODO: Extract allergies from AllergyIntolerance resources
        // This would require additional FHIR API calls

        return data;
    }

    /**
     * Calculate age from birth date.
     */
    private static Integer calculateAge(String birthDateStr) {
        try {
            LocalDate birthDate = LocalDate.parse(birthDateStr, DateTimeFormatter.ISO_LOCAL_DATE);
            LocalDate now = LocalDate.now();
            return Period.between(birthDate, now).getYears();
        } catch (Exception e) {
            return null;
        }
    }

    // Getters and setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }

    public String getFirstName() { return firstName; }
    public void setFirstName(String firstName) { this.firstName = firstName; }

    public String getLastName() { return lastName; }
    public void setLastName(String lastName) { this.lastName = lastName; }

    public String getDateOfBirth() { return dateOfBirth; }
    public void setDateOfBirth(String dateOfBirth) {
        this.dateOfBirth = dateOfBirth;
        this.age = calculateAge(dateOfBirth);
    }

    public String getGender() { return gender; }
    public void setGender(String gender) { this.gender = gender; }

    public Integer getAge() { return age; }
    public void setAge(Integer age) { this.age = age; }

    public String getMrn() { return mrn; }
    public void setMrn(String mrn) { this.mrn = mrn; }

    public List<String> getAllergies() { return allergies; }
    public void setAllergies(List<String> allergies) { this.allergies = allergies; }

    @Override
    public String toString() {
        return "FHIRPatientData{" +
                "id='" + id + '\'' +
                ", name='" + firstName + " " + lastName + '\'' +
                ", gender='" + gender + '\'' +
                ", age=" + age +
                ", mrn='" + mrn + '\'' +
                '}';
    }
}
