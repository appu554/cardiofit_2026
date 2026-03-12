package com.cardiofit.export.controller;

import com.cardiofit.export.service.ExportService;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.io.IOException;

/**
 * REST API for data export
 * Component 6F: Data Export API
 *
 * Supports CSV, JSON, PDF, and HL7 FHIR formats
 */
@RestController
@RequestMapping("/api/export")
@Slf4j
public class DataExportController {

    private final ExportService exportService;

    public DataExportController(ExportService exportService) {
        this.exportService = exportService;
    }

    /**
     * Export patient data to CSV
     *
     * GET /api/export/patients/csv?departmentId=ICU&startTime=1234567890&endTime=1234567890
     */
    @GetMapping("/patients/csv")
    public ResponseEntity<byte[]> exportPatientsCsv(
            @RequestParam String departmentId,
            @RequestParam Long startTime,
            @RequestParam Long endTime) throws IOException {

        log.info("Export patients CSV request: dept={}, start={}, end={}",
                departmentId, startTime, endTime);

        byte[] csvData = exportService.exportPatientsToCsv(departmentId, startTime, endTime);

        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.parseMediaType("text/csv"));
        headers.setContentDispositionFormData("attachment",
                "patients_" + departmentId + "_" + System.currentTimeMillis() + ".csv");

        return ResponseEntity.ok()
                .headers(headers)
                .body(csvData);
    }

    /**
     * Export alert data to CSV
     *
     * GET /api/export/alerts/csv?departmentId=ICU&startTime=1234567890&endTime=1234567890
     */
    @GetMapping("/alerts/csv")
    public ResponseEntity<byte[]> exportAlertsCsv(
            @RequestParam String departmentId,
            @RequestParam Long startTime,
            @RequestParam Long endTime) throws IOException {

        log.info("Export alerts CSV request: dept={}, start={}, end={}",
                departmentId, startTime, endTime);

        byte[] csvData = exportService.exportAlertsToCsv(departmentId, startTime, endTime);

        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.parseMediaType("text/csv"));
        headers.setContentDispositionFormData("attachment",
                "alerts_" + departmentId + "_" + System.currentTimeMillis() + ".csv");

        return ResponseEntity.ok()
                .headers(headers)
                .body(csvData);
    }

    /**
     * Export ML predictions to JSON
     *
     * GET /api/export/predictions/json?departmentId=ICU&modelType=SEPSIS&startTime=1234567890&endTime=1234567890
     */
    @GetMapping("/predictions/json")
    public ResponseEntity<String> exportPredictionsJson(
            @RequestParam String departmentId,
            @RequestParam String modelType,
            @RequestParam Long startTime,
            @RequestParam Long endTime) {

        log.info("Export predictions JSON request: dept={}, model={}, start={}, end={}",
                departmentId, modelType, startTime, endTime);

        String jsonData = exportService.exportPredictionsToJson(
                departmentId, modelType, startTime, endTime
        );

        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_JSON);
        headers.setContentDispositionFormData("attachment",
                "predictions_" + modelType + "_" + System.currentTimeMillis() + ".json");

        return ResponseEntity.ok()
                .headers(headers)
                .body(jsonData);
    }

    /**
     * Export patient data in HL7 FHIR format
     *
     * GET /api/export/patients/fhir?patientId=PT-12345
     */
    @GetMapping("/patients/fhir")
    public ResponseEntity<String> exportPatientsFhir(
            @RequestParam String patientId) {

        log.info("Export patient FHIR request: patientId={}", patientId);

        String fhirBundle = exportService.exportPatientToFhir(patientId);

        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_JSON);
        headers.setContentDispositionFormData("attachment",
                "patient_" + patientId + "_fhir.json");

        return ResponseEntity.ok()
                .headers(headers)
                .body(fhirBundle);
    }

    /**
     * Generate quality metrics report (PDF)
     *
     * GET /api/export/reports/quality-metrics?departmentId=ICU&period=MONTHLY
     */
    @GetMapping("/reports/quality-metrics")
    public ResponseEntity<byte[]> generateQualityMetricsReport(
            @RequestParam String departmentId,
            @RequestParam String period) throws IOException {

        log.info("Generate quality metrics report: dept={}, period={}",
                departmentId, period);

        byte[] pdfData = exportService.generateQualityMetricsReport(departmentId, period);

        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_PDF);
        headers.setContentDispositionFormData("attachment",
                "quality_metrics_" + departmentId + "_" + period + ".pdf");

        return ResponseEntity.ok()
                .headers(headers)
                .body(pdfData);
    }

    /**
     * Health check endpoint
     */
    @GetMapping("/health")
    public ResponseEntity<String> health() {
        return ResponseEntity.ok("Export & Reporting Service is running");
    }
}
