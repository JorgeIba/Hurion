package formatter

import (
	"bytes"

	"image/color"

	"github.com/fogleman/gg"
)

func GenerateTableImage(headers []string, rows [][]string) ([]byte, error) {
	const (
		fontSize      = 25
		rowHeight     = 35
		padding       = 40
		columnSpacing = 15
	)
	measureDC := gg.NewContext(1, 1) // Placeholder

	// Calculate width and height
	numCols := len(headers)
	colWidths := make([]float64, numCols)
	for i, h := range headers {
		w, _ := measureDC.MeasureString(h)
		colWidths[i] = w + columnSpacing
	}
	for _, row := range rows {
		for i, cell := range row {
			w, _ := measureDC.MeasureString(cell)
			if w+columnSpacing > colWidths[i] {
				colWidths[i] = w + columnSpacing
			}
		}
	}

	totalW := padding * 2
	for _, w := range colWidths {
		totalW += int(w)
	}
	totalH := padding*2 + rowHeight*(len(rows)+1)

	// Canvas
	dc := gg.NewContext(totalW, totalH)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Colors
	headerBg := color.RGBA{200, 200, 255, 255}
	altBg := color.RGBA{245, 245, 245, 255}

	// Draw header
	x := float64(padding)
	y := float64(padding)
	dc.SetColor(headerBg)
	dc.DrawRectangle(float64(padding), y, float64(totalW-2*padding), rowHeight)
	dc.Fill()
	dc.SetRGB(0, 0, 0)
	for i, h := range headers {
		dc.DrawStringAnchored(h, x+colWidths[i]/2, y+rowHeight/2, 0.5, 0.5)
		x += colWidths[i]
	}

	// Draw rows
	for ri, row := range rows {
		y += rowHeight
		x = float64(padding)
		if ri%2 == 0 {
			dc.SetColor(altBg)
			dc.DrawRectangle(float64(padding), y, float64(totalW-2*padding), rowHeight)
			dc.Fill()
		}
		dc.SetRGB(0, 0, 0)
		for ci, cell := range row {
			dc.DrawStringAnchored(cell, x+colWidths[ci]/2, y+rowHeight/2, 0.5, 0.5)
			x += colWidths[ci]
		}
	}

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
