package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

// Minimal stub — just enough for CDSEvent to compile and test.
// Will be REPLACED in Task 2 with the full MHRI implementation.
@JsonInclude(JsonInclude.Include.NON_NULL)
public class MHRIScore implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("composite")
    private Double composite;

    public MHRIScore() {}

    public Double getComposite() { return composite; }
    public void setComposite(Double composite) { this.composite = composite; }
}
