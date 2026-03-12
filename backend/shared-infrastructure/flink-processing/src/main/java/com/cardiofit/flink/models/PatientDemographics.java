package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Patient demographics for enriched events.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class PatientDemographics implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("firstName")
    private String firstName;

    @JsonProperty("lastName")
    private String lastName;

    @JsonProperty("dateOfBirth")
    private String dateOfBirth;

    @JsonProperty("gender")
    private String gender;

    @JsonProperty("age")
    private Integer age;

    @JsonProperty("mrn")
    private String mrn;

    @JsonProperty("weight")
    private Double weight; // Weight in kg

    @JsonProperty("sex")
    private String sex; // "M", "F", "MALE", "FEMALE"

    public PatientDemographics() {
    }

    public PatientDemographics(String firstName, String lastName, String dateOfBirth,
                               String gender, Integer age, String mrn) {
        this.firstName = firstName;
        this.lastName = lastName;
        this.dateOfBirth = dateOfBirth;
        this.gender = gender;
        this.age = age;
        this.mrn = mrn;
    }

    // Getters and setters
    public String getFirstName() { return firstName; }
    public void setFirstName(String firstName) { this.firstName = firstName; }

    public String getLastName() { return lastName; }
    public void setLastName(String lastName) { this.lastName = lastName; }

    public String getDateOfBirth() { return dateOfBirth; }
    public void setDateOfBirth(String dateOfBirth) { this.dateOfBirth = dateOfBirth; }

    public String getGender() { return gender; }
    public void setGender(String gender) { this.gender = gender; }

    public Integer getAge() { return age; }
    public void setAge(Integer age) { this.age = age; }

    public String getMrn() { return mrn; }
    public void setMrn(String mrn) { this.mrn = mrn; }

    public Double getWeight() { return weight; }
    public void setWeight(Double weight) { this.weight = weight; }

    public String getSex() { return sex; }
    public void setSex(String sex) { this.sex = sex; }

    @Override
    public String toString() {
        return "PatientDemographics{" +
                "name='" + firstName + " " + lastName + '\'' +
                ", age=" + age +
                ", gender='" + gender + '\'' +
                '}';
    }
}
