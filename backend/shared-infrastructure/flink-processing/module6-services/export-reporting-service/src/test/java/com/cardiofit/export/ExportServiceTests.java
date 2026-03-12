package com.cardiofit.export;

import com.cardiofit.export.model.Alert;
import com.cardiofit.export.model.PatientCurrentState;
import com.cardiofit.export.service.CsvExportService;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;

import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Basic tests for Export Service
 */
@SpringBootTest
public class ExportServiceTests {

    @Autowired
    private CsvExportService csvExportService;

    @Test
    public void testExportPatientsToCsv() throws IOException {
        // Create test data
        List<PatientCurrentState> patients = new ArrayList<>();

        PatientCurrentState patient = new PatientCurrentState();
        patient.setPatientId("PT-001");
        patient.setPatientName("John Doe");
        patient.setAge(45);
        patient.setGender("Male");
        patient.setRoom("ICU-101");
        patient.setDepartmentName("ICU");
        patient.setOverallRiskScore(85.5);
        patient.setRiskCategory("HIGH");
        patient.setActiveAlertCount(3);
        patient.setAdmissionTime(System.currentTimeMillis());
        patient.setLengthOfStay(24.5);

        patients.add(patient);

        // Export to CSV
        byte[] csvData = csvExportService.exportPatientsToCsv(patients);

        // Verify
        assertNotNull(csvData);
        assertTrue(csvData.length > 0);

        String csvContent = new String(csvData);
        assertTrue(csvContent.contains("Patient ID"));
        assertTrue(csvContent.contains("PT-001"));
        assertTrue(csvContent.contains("John Doe"));
    }

    @Test
    public void testExportAlertsToCsv() throws IOException {
        // Create test data
        List<Alert> alerts = new ArrayList<>();

        Alert alert = new Alert();
        alert.setAlertId("ALERT-001");
        alert.setPatientId("PT-001");
        alert.setPatientName("John Doe");
        alert.setAlertType("VITAL_ABNORMAL");
        alert.setSeverity("HIGH");
        alert.setMessage("Heart rate elevated");
        alert.setStatus("ACTIVE");
        alert.setDepartment("ICU");
        alert.setCreatedAt(System.currentTimeMillis());

        alerts.add(alert);

        // Export to CSV
        byte[] csvData = csvExportService.exportAlertsToCsv(alerts);

        // Verify
        assertNotNull(csvData);
        assertTrue(csvData.length > 0);

        String csvContent = new String(csvData);
        assertTrue(csvContent.contains("Alert ID"));
        assertTrue(csvContent.contains("ALERT-001"));
        assertTrue(csvContent.contains("Heart rate elevated"));
    }
}
