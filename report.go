package main

import (
	"bytes"
	"fmt"
	"maps"
	"slices"

	"github.com/jung-kurt/gofpdf"
)

type PDFGenerator struct{}

func NewPDFGenerator() *PDFGenerator { return &PDFGenerator{} }

// GeneratePDF renders a table of link statuses, ordered by set ID and URL.
func (pg *PDFGenerator) GeneratePDF(sets map[int]*LinkSet) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Link Status Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(20, 10, "Set ID")
	pdf.Cell(80, 10, "URL")
	pdf.Cell(30, 10, "Status")
	pdf.Cell(40, 10, "Checked at")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 10)
	for _, id := range slices.Sorted(maps.Keys(sets)) {
		set := sets[id]
		for _, url := range slices.Sorted(maps.Keys(set.Links)) {
			status := set.Links[url]

			display := url
			if len(display) > 50 {
				display = display[:47] + "..."
			}
			pdf.Cell(20, 8, fmt.Sprintf("%d", id))
			pdf.Cell(80, 8, display)
			if status == StatusAvailable {
				pdf.SetTextColor(0, 128, 0)
			} else {
				pdf.SetTextColor(200, 0, 0)
			}
			pdf.Cell(30, 8, status)
			pdf.SetTextColor(0, 0, 0)
			pdf.Cell(40, 8, set.Timestamp.Format("2006-01-02 15:04:05"))
			pdf.Ln(8)
		}
		pdf.Ln(2)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("render pdf: %w", err)
	}
	return buf.Bytes(), nil
}
