package com.cardiofit.export.service;

import com.cardiofit.export.model.Alert;
import com.cardiofit.export.model.PatientCurrentState;
import com.opencsv.CSVWriter;
import org.springframework.stereotype.Service;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.OutputStreamWriter;
import java.util.List;

/**
 * Service for CSV data export
 */
@Service
public class CsvExportService {

    /**
     * Export patient data to CSV format
     */
    public byte[] exportPatientsToCsv(List<PatientCurrentState> patients) throws IOException {
        ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
        OutputStreamWriter writer = new OutputStreamWriter(outputStream);
        CSVWriter csvWriter = new CSVWriter(writer);

        // Write header
        String[] header = {
                "Patient ID", "Name", "Age", "Gender", "Room",
                "Department", "Risk Score", "Risk Category", "Active Alerts",
                "Admission Time", "Length of Stay (hours)"
        };
        csvWriter.writeNext(header);

        // Write data rows
        for (PatientCurrentState patient : patients) {
            String[] row = {
                    patient.getPatientId(),
                    patient.getPatientName(),
                    String.valueOf(patient.getAge()),
                    patient.getGender(),
                    patient.getRoom(),
                    patient.getDepartmentName(),
                    String.format("%.2f", patient.getOverallRiskScore()),
                    patient.getRiskCategory(),
                    String.valueOf(patient.getActiveAlertCount()),
                    String.valueOf(patient.getAdmissionTime()),
                    String.format("%.1f", patient.getLengthOfStay())
            };
            csvWriter.writeNext(row);
        }

        csvWriter.close();
        return outputStream.toByteArray();
    }

    /**
     * Export alert data to CSV format
     */
    public byte[] exportAlertsToCsv(List<Alert> alerts) throws IOException {
        ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
        OutputStreamWriter writer = new OutputStreamWriter(outputStream);
        CSVWriter csvWriter = new CSVWriter(writer);

        // Write header
        String[] header = {
                "Alert ID", "Patient ID", "Patient Name", "Alert Type",
                "Severity", "Message", "Status", "Department", "Created At",
                "Acknowledged At", "Acknowledged By"
        };
        csvWriter.writeNext(header);

        // Write data rows
        for (Alert alert : alerts) {
            String[] row = {
                    alert.getAlertId(),
                    alert.getPatientId(),
                    alert.getPatientName(),
                    alert.getAlertType(),
                    alert.getSeverity(),
                    alert.getMessage(),
                    alert.getStatus(),
                    alert.getDepartment(),
                    String.valueOf(alert.getCreatedAt()),
                    alert.getAcknowledgedAt() != null ? String.valueOf(alert.getAcknowledgedAt()) : "",
                    alert.getAcknowledgedBy() != null ? alert.getAcknowledgedBy() : ""
            };
            csvWriter.writeNext(row);
        }

        csvWriter.close();
        return outputStream.toByteArray();
    }
}
