package formatter

import (
	"bytes"
	"image/color"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/fogleman/gg"
)

// GenerateTableImage generates a table image interpreting semantic metadata
func GenerateTableImage(tableScore services.TableScore) ([]byte, error) {
	const (
		fontSize      = 25
		rowHeight     = 35
		padding       = 40
		columnSpacing = 15
	)
	measureDC := gg.NewContext(1, 1) // Placeholder

	headers := tableScore.Header
	rows := tableScore.Table
	rowMetadata := tableScore.RowMetadata

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

	// Visual style colors
	headerBg := color.RGBA{150, 180, 230, 255}      // Softer blue for header
	topicHeaderBg := color.RGBA{180, 200, 240, 255} // Even lighter blue for topic headers
	acBg := color.RGBA{180, 231, 206, 255}          // Pastel green for AC cells (#B4E7CE)
	altBg := color.RGBA{245, 245, 245, 255}         // Alternating row background

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

		metadata := rowMetadata[ri]

		if metadata.IsTopicHeader {
			// Topic header row - full width lighter blue background
			dc.SetColor(topicHeaderBg)
			dc.DrawRectangle(float64(padding), y, float64(totalW-2*padding), rowHeight)
			dc.Fill()
			dc.SetRGB(0, 0, 0)
			for ci, cell := range row {
				dc.DrawStringAnchored(cell, x+colWidths[ci]/2, y+rowHeight/2, 0.5, 0.5)
				x += colWidths[ci]
			}
		} else {
			// Problem row - draw each cell with appropriate background
			cellX := float64(padding)
			for ci, cell := range row {
				// Determine cell background color
				var cellBg color.RGBA
				if ci < len(metadata.CellMetadata) && metadata.CellMetadata[ci].IsAC {
					cellBg = acBg // Green for AC
				} else if ri%2 == 0 {
					cellBg = altBg // Alternating gray
				} else {
					cellBg = color.RGBA{255, 255, 255, 255} // White
				}

				// Draw cell background
				dc.SetColor(cellBg)
				dc.DrawRectangle(cellX, y, colWidths[ci], rowHeight)
				dc.Fill()

				// Draw cell text (always black)
				dc.SetRGB(0, 0, 0)
				dc.DrawStringAnchored(cell, cellX+colWidths[ci]/2, y+rowHeight/2, 0.5, 0.5)

				cellX += colWidths[ci]
			}
		}
	}

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
