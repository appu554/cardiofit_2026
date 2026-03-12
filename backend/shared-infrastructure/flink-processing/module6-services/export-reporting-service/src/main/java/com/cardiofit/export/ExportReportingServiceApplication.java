package com.cardiofit.export;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

/**
 * Main application class for Export & Reporting Service
 *
 * Component 6F: Data Export API
 * Component 6G: Automated Reporting Service
 *
 * Provides REST endpoints for CSV, JSON, PDF, and FHIR exports
 * Automated scheduled reports (daily, weekly, monthly)
 */
@SpringBootApplication
@EnableScheduling
public class ExportReportingServiceApplication {

    public static void main(String[] args) {
        SpringApplication.run(ExportReportingServiceApplication.class, args);
    }
}
