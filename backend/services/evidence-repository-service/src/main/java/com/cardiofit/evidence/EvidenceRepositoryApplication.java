package com.cardiofit.evidence;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.web.client.RestTemplate;

/**
 * Evidence Repository Service - Main Application
 *
 * Spring Boot microservice for medical literature citation management.
 *
 * Features:
 * - PubMed E-utilities integration for automated citation fetching
 * - GRADE framework-based evidence quality assessment
 * - Multi-format citation rendering (AMA, Vancouver, APA, NLM, SHORT)
 * - Scheduled retraction checking and evidence updates
 * - REST API for citation CRUD operations
 *
 * Design: Phase 7 Evidence Repository (original specification)
 * Architecture: Spring Boot microservice (separate from Flink processing)
 *
 * API Documentation: http://localhost:8015/swagger-ui.html
 * Health Check: http://localhost:8015/actuator/health
 */
@SpringBootApplication
@EnableScheduling
public class EvidenceRepositoryApplication {

    public static void main(String[] args) {
        SpringApplication.run(EvidenceRepositoryApplication.class, args);
    }

    /**
     * Configure RestTemplate for HTTP requests
     *
     * Used by PubMedService for NCBI E-utilities API calls.
     */
    @Bean
    public RestTemplate restTemplate() {
        return new RestTemplate();
    }
}
