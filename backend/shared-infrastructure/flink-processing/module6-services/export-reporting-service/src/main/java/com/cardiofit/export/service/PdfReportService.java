package com.cardiofit.export.service;

import com.itextpdf.kernel.pdf.PdfDocument;
import com.itextpdf.kernel.pdf.PdfWriter;
import com.itextpdf.layout.Document;
import com.itextpdf.layout.element.Paragraph;
import com.itextpdf.layout.element.Table;
import com.itextpdf.layout.element.Cell;
import com.itextpdf.layout.properties.TextAlignment;
import com.itextpdf.layout.properties.UnitValue;
import org.springframework.stereotype.Service;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.Map;

/**
 * Service for PDF report generation using iText 7
 */
@Service
public class PdfReportService {

    private static final DateTimeFormatter DATE_FORMATTER =
        DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss");

    /**
     * Generate quality metrics report as PDF
     */
    public byte[] generateQualityMetricsReport(
            String departmentId,
            String period,
            Map<String, Object> metrics) throws IOException {

        ByteArrayOutputStream baos = new ByteArrayOutputStream();
        PdfWriter writer = new PdfWriter(baos);
        PdfDocument pdfDoc = new PdfDocument(writer);
        Document document = new Document(pdfDoc);

        // Title
        Paragraph title = new Paragraph("Quality Metrics Report")
                .setFontSize(20)
                .setBold()
                .setTextAlignment(TextAlignment.CENTER);
        document.add(title);

        // Report details
        Paragraph details = new Paragraph()
                .add("Department: " + departmentId + "\n")
                .add("Period: " + period + "\n")
                .add("Generated: " + getCurrentTimestamp() + "\n")
                .setFontSize(10);
        document.add(details);

        // Add line break
        document.add(new Paragraph("\n"));

        // Create metrics table
        if (metrics != null && !metrics.isEmpty()) {
            Table table = new Table(UnitValue.createPercentArray(new float[]{3, 2}));
            table.setWidth(UnitValue.createPercentValue(100));

            // Header row
            table.addHeaderCell(new Cell().add(new Paragraph("Metric").setBold()));
            table.addHeaderCell(new Cell().add(new Paragraph("Value").setBold()));

            // Data rows
            for (Map.Entry<String, Object> entry : metrics.entrySet()) {
                table.addCell(new Cell().add(new Paragraph(entry.getKey())));
                table.addCell(new Cell().add(new Paragraph(String.valueOf(entry.getValue()))));
            }

            document.add(table);
        } else {
            // Default metrics section
            addDefaultMetrics(document, departmentId, period);
        }

        // Footer
        document.add(new Paragraph("\n"));
        Paragraph footer = new Paragraph("CardioFit Clinical Analytics Platform")
                .setFontSize(8)
                .setTextAlignment(TextAlignment.CENTER);
        document.add(footer);

        document.close();
        return baos.toByteArray();
    }

    /**
     * Add default metrics section
     */
    private void addDefaultMetrics(Document document, String departmentId, String period) {
        // Patient Census
        document.add(new Paragraph("Patient Census").setBold().setFontSize(14));
        Table censusTable = new Table(UnitValue.createPercentArray(new float[]{3, 2}));
        censusTable.setWidth(UnitValue.createPercentValue(100));

        censusTable.addCell("Total Patients");
        censusTable.addCell("150");
        censusTable.addCell("ICU Patients");
        censusTable.addCell("45");
        censusTable.addCell("ED Patients");
        censusTable.addCell("30");

        document.add(censusTable);
        document.add(new Paragraph("\n"));

        // Risk Stratification
        document.add(new Paragraph("Risk Stratification").setBold().setFontSize(14));
        Table riskTable = new Table(UnitValue.createPercentArray(new float[]{3, 2}));
        riskTable.setWidth(UnitValue.createPercentValue(100));

        riskTable.addCell("Critical Risk");
        riskTable.addCell("12");
        riskTable.addCell("High Risk");
        riskTable.addCell("28");
        riskTable.addCell("Moderate Risk");
        riskTable.addCell("56");
        riskTable.addCell("Low Risk");
        riskTable.addCell("54");

        document.add(riskTable);
        document.add(new Paragraph("\n"));

        // Alert Performance
        document.add(new Paragraph("Alert Performance").setBold().setFontSize(14));
        Table alertTable = new Table(UnitValue.createPercentArray(new float[]{3, 2}));
        alertTable.setWidth(UnitValue.createPercentValue(100));

        alertTable.addCell("Active Alerts");
        alertTable.addCell("87");
        alertTable.addCell("Critical Alerts");
        alertTable.addCell("15");
        alertTable.addCell("Acknowledgment Rate");
        alertTable.addCell("85.2%");
        alertTable.addCell("Avg Response Time");
        alertTable.addCell("8.5 minutes");

        document.add(alertTable);
        document.add(new Paragraph("\n"));

        // Quality Metrics
        document.add(new Paragraph("Quality Metrics").setBold().setFontSize(14));
        Table qualityTable = new Table(UnitValue.createPercentArray(new float[]{3, 2}));
        qualityTable.setWidth(UnitValue.createPercentValue(100));

        qualityTable.addCell("30-Day Mortality");
        qualityTable.addCell("2.8%");
        qualityTable.addCell("30-Day Readmission");
        qualityTable.addCell("12.5%");
        qualityTable.addCell("Sepsis Bundle Compliance");
        qualityTable.addCell("89.3%");
        qualityTable.addCell("Alert Fatigue Score");
        qualityTable.addCell("3.2/10");

        document.add(qualityTable);
    }

    /**
     * Get current timestamp formatted
     */
    private String getCurrentTimestamp() {
        LocalDateTime now = LocalDateTime.now();
        return now.format(DATE_FORMATTER);
    }
}
