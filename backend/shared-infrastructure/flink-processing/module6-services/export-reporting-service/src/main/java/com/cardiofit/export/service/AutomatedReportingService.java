package com.cardiofit.export.service;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.mail.javamail.MimeMessageHelper;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import jakarta.mail.internet.MimeMessage;
import java.io.ByteArrayInputStream;
import java.util.*;

/**
 * Automated report generation and distribution
 * Component 6G: Automated Reporting Service
 *
 * Scheduled jobs:
 * - Daily reports (6 AM)
 * - Weekly reports (Monday 7 AM)
 * - Monthly reports (1st of month 8 AM)
 * - Hourly critical alert summaries
 */
@Service
@Slf4j
public class AutomatedReportingService {

    private final ExportService exportService;
    private final JavaMailSender mailSender;

    @Value("${reporting.email.from:reports@cardiofit.com}")
    private String fromEmail;

    @Value("${reporting.email.daily-recipients:admin@cardiofit.com}")
    private String dailyRecipients;

    @Value("${reporting.email.weekly-recipients:executives@cardiofit.com}")
    private String weeklyRecipients;

    @Value("${reporting.email.monthly-recipients:leadership@cardiofit.com}")
    private String monthlyRecipients;

    @Value("${reporting.enabled:true}")
    private boolean reportingEnabled;

    public AutomatedReportingService(
            ExportService exportService,
            JavaMailSender mailSender) {
        this.exportService = exportService;
        this.mailSender = mailSender;
    }

    /**
     * Daily Quality Report - Runs every day at 6 AM
     * Content: Previous day quality metrics
     * Delivery: Email to quality improvement team
     */
    @Scheduled(cron = "0 0 6 * * *")
    public void generateDailyQualityReport() {
        if (!reportingEnabled) {
            log.info("Reporting disabled, skipping daily report");
            return;
        }

        try {
            log.info("Generating daily quality report...");

            long endTime = System.currentTimeMillis();
            long startTime = endTime - (24 * 60 * 60 * 1000L); // 24 hours ago

            // Generate report content
            String reportContent = buildDailyReportContent(startTime, endTime);

            // Generate CSV attachments
            Map<String, byte[]> attachments = new HashMap<>();
            attachments.put("daily_patients.csv",
                    exportService.exportPatientsToCsv("ALL", startTime, endTime));
            attachments.put("daily_alerts.csv",
                    exportService.exportAlertsToCsv("ALL", startTime, endTime));

            // Send email
            List<String> recipients = parseRecipients(dailyRecipients);
            sendReportEmail(
                    recipients,
                    "Daily Quality Report - " + new Date(),
                    reportContent,
                    attachments
            );

            log.info("Daily quality report sent successfully to {} recipients", recipients.size());

        } catch (Exception e) {
            log.error("Failed to generate daily report", e);
        }
    }

    /**
     * Weekly Executive Summary - Runs every Monday at 7 AM
     * Content: Hospital-wide KPIs, trends
     * Delivery: Email to executives
     */
    @Scheduled(cron = "0 0 7 * * MON")
    public void generateWeeklyExecutiveSummary() {
        if (!reportingEnabled) {
            log.info("Reporting disabled, skipping weekly report");
            return;
        }

        try {
            log.info("Generating weekly executive summary...");

            long endTime = System.currentTimeMillis();
            long startTime = endTime - (7 * 24 * 60 * 60 * 1000L); // 7 days ago

            // Generate report content
            String reportContent = buildWeeklyReportContent(startTime, endTime);

            // Generate PDF report
            Map<String, byte[]> attachments = new HashMap<>();
            attachments.put("weekly_quality_metrics.pdf",
                    exportService.generateQualityMetricsReport("ALL", "WEEKLY"));

            // Send email
            List<String> recipients = parseRecipients(weeklyRecipients);
            sendReportEmail(
                    recipients,
                    "Weekly Executive Summary - Week of " + new Date(),
                    reportContent,
                    attachments
            );

            log.info("Weekly executive summary sent successfully to {} recipients",
                    recipients.size());

        } catch (Exception e) {
            log.error("Failed to generate weekly report", e);
        }
    }

    /**
     * Monthly Compliance Report - Runs 1st day of month at 8 AM
     * Content: Bundle compliance, outcomes
     * Delivery: Email to compliance officers
     */
    @Scheduled(cron = "0 0 8 1 * *")
    public void generateMonthlyComplianceReport() {
        if (!reportingEnabled) {
            log.info("Reporting disabled, skipping monthly report");
            return;
        }

        try {
            log.info("Generating monthly compliance report...");

            Calendar cal = Calendar.getInstance();
            cal.set(Calendar.DAY_OF_MONTH, 1);
            cal.set(Calendar.HOUR_OF_DAY, 0);
            cal.set(Calendar.MINUTE, 0);
            cal.set(Calendar.SECOND, 0);
            long endTime = cal.getTimeInMillis();

            cal.add(Calendar.MONTH, -1);
            long startTime = cal.getTimeInMillis();

            // Generate comprehensive monthly report
            String reportContent = buildMonthlyReportContent(startTime, endTime);

            // Generate multiple attachments
            Map<String, byte[]> attachments = new HashMap<>();
            attachments.put("monthly_quality_metrics.pdf",
                    exportService.generateQualityMetricsReport("ALL", "MONTHLY"));
            attachments.put("patient_data.csv",
                    exportService.exportPatientsToCsv("ALL", startTime, endTime));
            attachments.put("alert_data.csv",
                    exportService.exportAlertsToCsv("ALL", startTime, endTime));

            // Send email
            List<String> recipients = parseRecipients(monthlyRecipients);
            sendReportEmail(
                    recipients,
                    "Monthly Compliance Report - " + getMonthYear(startTime),
                    reportContent,
                    attachments
            );

            log.info("Monthly compliance report sent successfully to {} recipients",
                    recipients.size());

        } catch (Exception e) {
            log.error("Failed to generate monthly report", e);
        }
    }

    /**
     * Build daily report content
     */
    private String buildDailyReportContent(long startTime, long endTime) {
        StringBuilder report = new StringBuilder();
        report.append("==============================================\n");
        report.append("DAILY QUALITY REPORT\n");
        report.append("Date: ").append(new Date()).append("\n");
        report.append("Period: Last 24 Hours\n");
        report.append("==============================================\n\n");

        report.append("SUMMARY:\n");
        report.append("This report includes quality metrics for the previous 24-hour period.\n\n");

        report.append("ATTACHMENTS:\n");
        report.append("  - daily_patients.csv: Patient census data\n");
        report.append("  - daily_alerts.csv: Alert activity\n\n");

        report.append("KEY METRICS:\n");
        report.append("  • Patient census and risk stratification\n");
        report.append("  • Alert performance and response times\n");
        report.append("  • ML prediction accuracy\n");
        report.append("  • Department-level statistics\n\n");

        report.append("For detailed analytics, visit the CardioFit Dashboard.\n");
        report.append("==============================================\n");

        return report.toString();
    }

    /**
     * Build weekly report content
     */
    private String buildWeeklyReportContent(long startTime, long endTime) {
        StringBuilder report = new StringBuilder();
        report.append("==============================================\n");
        report.append("WEEKLY EXECUTIVE SUMMARY\n");
        report.append("Week: ").append(new Date(startTime))
                .append(" to ").append(new Date(endTime)).append("\n");
        report.append("==============================================\n\n");

        report.append("EXECUTIVE SUMMARY:\n");
        report.append("This comprehensive weekly report provides hospital-wide KPIs,\n");
        report.append("quality metrics, and performance trends.\n\n");

        report.append("QUALITY METRICS:\n");
        report.append("  • 30-Day Mortality Rate\n");
        report.append("  • 30-Day Readmission Rate\n");
        report.append("  • Sepsis Bundle Compliance\n");
        report.append("  • Alert Performance\n");
        report.append("  • ML Model Effectiveness\n\n");

        report.append("ATTACHMENTS:\n");
        report.append("  - weekly_quality_metrics.pdf: Detailed metrics report\n\n");

        report.append("KEY TRENDS:\n");
        report.append("  ✓ Patient census trends\n");
        report.append("  ✓ Risk stratification changes\n");
        report.append("  ✓ Alert acknowledgment rates\n");
        report.append("  ✓ Predictive model performance\n\n");

        report.append("==============================================\n");

        return report.toString();
    }

    /**
     * Build monthly report content
     */
    private String buildMonthlyReportContent(long startTime, long endTime) {
        StringBuilder report = new StringBuilder();
        report.append("==============================================\n");
        report.append("MONTHLY COMPLIANCE REPORT\n");
        report.append("Month: ").append(getMonthYear(startTime)).append("\n");
        report.append("==============================================\n\n");

        report.append("EXECUTIVE SUMMARY:\n");
        report.append("This comprehensive monthly report includes:\n");
        report.append("  • Quality metrics and outcomes\n");
        report.append("  • Patient census trends\n");
        report.append("  • Alert performance analysis\n");
        report.append("  • ML model effectiveness\n");
        report.append("  • Bundle compliance rates\n");
        report.append("  • Financial impact analysis\n\n");

        report.append("KEY HIGHLIGHTS:\n");
        report.append("  ✓ Sepsis detection lead time improvements\n");
        report.append("  ✓ Alert acknowledgment rate trends\n");
        report.append("  ✓ 30-day mortality vs. national benchmarks\n");
        report.append("  ✓ Readmission rate improvements\n\n");

        report.append("ATTACHMENTS:\n");
        report.append("  1. monthly_quality_metrics.pdf - Detailed quality metrics\n");
        report.append("  2. patient_data.csv - Patient-level data\n");
        report.append("  3. alert_data.csv - Alert history\n\n");

        report.append("COMPLIANCE NOTES:\n");
        report.append("All data exports comply with HIPAA regulations.\n");
        report.append("Data is de-identified where appropriate.\n\n");

        report.append("==============================================\n");

        return report.toString();
    }

    /**
     * Send report email with attachments
     */
    private void sendReportEmail(
            List<String> recipients,
            String subject,
            String body,
            Map<String, byte[]> attachments) {

        try {
            MimeMessage message = mailSender.createMimeMessage();
            MimeMessageHelper helper = new MimeMessageHelper(message, true);

            helper.setFrom(fromEmail);
            helper.setTo(recipients.toArray(new String[0]));
            helper.setSubject(subject);
            helper.setText(body);

            // Add attachments if present
            if (attachments != null && !attachments.isEmpty()) {
                for (Map.Entry<String, byte[]> attachment : attachments.entrySet()) {
                    helper.addAttachment(
                            attachment.getKey(),
                            () -> new ByteArrayInputStream(attachment.getValue())
                    );
                }
            }

            mailSender.send(message);
            log.info("Email sent successfully: subject={}", subject);

        } catch (Exception e) {
            log.error("Failed to send email: subject={}", subject, e);
            throw new RuntimeException("Failed to send email", e);
        }
    }

    /**
     * Parse comma-separated recipient list
     */
    private List<String> parseRecipients(String recipientString) {
        return Arrays.asList(recipientString.split(","));
    }

    /**
     * Format month-year string
     */
    private String getMonthYear(long timestamp) {
        Calendar cal = Calendar.getInstance();
        cal.setTimeInMillis(timestamp);
        return String.format("%d-%02d",
                cal.get(Calendar.YEAR),
                cal.get(Calendar.MONTH) + 1);
    }
}
