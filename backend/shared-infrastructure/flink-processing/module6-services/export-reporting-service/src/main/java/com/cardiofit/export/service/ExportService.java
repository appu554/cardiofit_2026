package com.cardiofit.export.service;

import com.cardiofit.export.model.Alert;
import com.cardiofit.export.model.MlPrediction;
import com.cardiofit.export.model.PatientCurrentState;
import com.cardiofit.export.repository.AlertRepository;
import com.cardiofit.export.repository.MlPredictionRepository;
import com.cardiofit.export.repository.PatientRepository;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.util.List;

/**
 * Main service for data export operations
 * Component 6F: Data Export API
 */
@Service
@Slf4j
public class ExportService {

    private final PatientRepository patientRepository;
    private final AlertRepository alertRepository;
    private final MlPredictionRepository predictionRepository;
    private final CsvExportService csvExportService;
    private final FhirExportService fhirExportService;
    private final PdfReportService pdfReportService;
    private final ObjectMapper objectMapper;

    public ExportService(
            PatientRepository patientRepository,
            AlertRepository alertRepository,
            MlPredictionRepository predictionRepository,
            CsvExportService csvExportService,
            FhirExportService fhirExportService,
            PdfReportService pdfReportService,
            ObjectMapper objectMapper) {
        this.patientRepository = patientRepository;
        this.alertRepository = alertRepository;
        this.predictionRepository = predictionRepository;
        this.csvExportService = csvExportService;
        this.fhirExportService = fhirExportService;
        this.pdfReportService = pdfReportService;
        this.objectMapper = objectMapper;
    }

    /**
     * Export patient data to CSV
     */
    public byte[] exportPatientsToCsv(String departmentId, Long startTime, Long endTime)
            throws IOException {
        log.info("Exporting patients to CSV: dept={}, start={}, end={}",
                departmentId, startTime, endTime);

        List<PatientCurrentState> patients;
        if ("ALL".equals(departmentId)) {
            patients = patientRepository.findAllByTimeRange(startTime, endTime);
        } else {
            patients = patientRepository.findByDepartmentAndTimeRange(
                    departmentId, startTime, endTime);
        }

        log.info("Found {} patients to export", patients.size());
        return csvExportService.exportPatientsToCsv(patients);
    }

    /**
     * Export alert data to CSV
     */
    public byte[] exportAlertsToCsv(String departmentId, Long startTime, Long endTime)
            throws IOException {
        log.info("Exporting alerts to CSV: dept={}, start={}, end={}",
                departmentId, startTime, endTime);

        List<Alert> alerts;
        if ("ALL".equals(departmentId)) {
            alerts = alertRepository.findAllByTimeRange(startTime, endTime);
        } else {
            alerts = alertRepository.findByDepartmentAndTimeRange(
                    departmentId, startTime, endTime);
        }

        log.info("Found {} alerts to export", alerts.size());
        return csvExportService.exportAlertsToCsv(alerts);
    }

    /**
     * Export ML predictions to JSON
     */
    public String exportPredictionsToJson(
            String departmentId, String modelType, Long startTime, Long endTime) {
        log.info("Exporting predictions to JSON: dept={}, model={}, start={}, end={}",
                departmentId, modelType, startTime, endTime);

        List<MlPrediction> predictions = predictionRepository
                .findByDepartmentModelTypeAndTimeRange(
                        departmentId, modelType, startTime, endTime);

        try {
            log.info("Found {} predictions to export", predictions.size());
            return objectMapper.writerWithDefaultPrettyPrinter()
                    .writeValueAsString(predictions);
        } catch (Exception e) {
            log.error("Failed to convert predictions to JSON", e);
            throw new RuntimeException("Failed to convert to JSON", e);
        }
    }

    /**
     * Export patient data to HL7 FHIR format
     */
    public String exportPatientToFhir(String patientId) {
        log.info("Exporting patient to FHIR: patientId={}", patientId);

        PatientCurrentState patient = patientRepository.findById(patientId)
                .orElseThrow(() -> new RuntimeException("Patient not found: " + patientId));

        return fhirExportService.exportPatientToFhir(patient);
    }

    /**
     * Generate quality metrics report (PDF)
     */
    public byte[] generateQualityMetricsReport(String departmentId, String period)
            throws IOException {
        log.info("Generating quality metrics report: dept={}, period={}",
                departmentId, period);

        // Could fetch actual metrics from database here
        return pdfReportService.generateQualityMetricsReport(
                departmentId, period, null);
    }
}
