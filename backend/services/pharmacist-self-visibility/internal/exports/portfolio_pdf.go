// Package exports provides generators for pharmacist-controlled export bundles.
//
// VisibilityClass: pharmacist-controlled — platform retains no submission record.
package exports

import (
	"fmt"
	"io"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// RenderPortfolioPDF renders an APC-aligned RPL evidence pack as a PDF document,
// writing the result to w.
//
// The generated PDF follows the five APC competency dimensions in a fixed order:
//
//  1. clinical_assessment
//  2. medication_review
//  3. communication
//  4. quality_use_of_medicines
//  5. professional_practice
//
// For each dimension, all matching items from pack.Items are listed with their
// title and annotation. Dimensions with no matching items are still rendered as
// section headings so the output is structurally consistent across packs.
//
// pharmacistName appears in the header block alongside the pack ID and generation
// date. The PDF is rendered once; the platform does not retain a copy.
//
// VisibilityClass: pharmacist-controlled — platform retains no submission record.
func RenderPortfolioPDF(pack RPLPack, pharmacistName string, w io.Writer) error {
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Pin the PDF creation-date metadata field to the zero time so that two
	// calls with identical inputs produce byte-identical output. Without this,
	// gofpdf embeds time.Now() in the metadata, making output non-deterministic.
	pdf.SetCreationDate(time.Time{})

	pdf.AddPage()

	// -----------------------------------------------------------------------
	// Header
	// -----------------------------------------------------------------------
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, "RPL Evidence Pack — APC-Aligned Submission")
	pdf.Ln(15)

	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 6, fmt.Sprintf("Pharmacist: %s", pharmacistName))
	pdf.Ln(8)
	pdf.Cell(0, 6, fmt.Sprintf("Pack ID: %s", pack.ID))
	pdf.Ln(8)
	pdf.Cell(0, 6, fmt.Sprintf("Generated: %s", pack.GeneratedAt.Format("2006-01-02")))
	pdf.Ln(12)

	// -----------------------------------------------------------------------
	// Five competency dimensions in fixed APC order
	// -----------------------------------------------------------------------
	dims := []string{
		"clinical_assessment",
		"medication_review",
		"communication",
		"quality_use_of_medicines",
		"professional_practice",
	}

	for _, d := range dims {
		pdf.SetFont("Helvetica", "B", 13)
		pdf.Cell(0, 8, "Competency: "+d)
		pdf.Ln(8)

		pdf.SetFont("Helvetica", "", 10)
		for _, item := range pack.Items {
			if item.Dimension != d {
				continue
			}
			pdf.MultiCell(0, 5,
				fmt.Sprintf("• %s\n  %s", item.Title, item.Annotation),
				"", "", false)
			pdf.Ln(2)
		}
		pdf.Ln(4)
	}

	return pdf.Output(w)
}
