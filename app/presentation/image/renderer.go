package image

import (
	"bytes"
	"image/color"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation"
	"github.com/fogleman/gg"
)

// GenerateScoreImage generates a table image from a TableScore interpreting semantic metadata
func GenerateScoreImage(tableScore presentation.TableScore) ([]byte, error) {
	measureDC := gg.NewContext(1, 1) // Placeholder

	headers := tableScore.Header
	rows := tableScore.Table
	rowMetadata := tableScore.RowMetadata

	// Calculate width and height
	numCols := len(headers)
	colWidths := make([]float64, numCols)
	for i, h := range headers {
		w, _ := measureDC.MeasureString(h)
		colWidths[i] = w + ColumnSpacing
	}
	for _, row := range rows {
		for i, cell := range row {
			w, _ := measureDC.MeasureString(cell)
			if w+ColumnSpacing > colWidths[i] {
				colWidths[i] = w + ColumnSpacing
			}
		}
	}

	totalW := Padding * 2
	for _, w := range colWidths {
		totalW += int(w)
	}
	totalH := Padding*2 + RowHeight*(len(rows)+1)

	// Canvas
	dc := gg.NewContext(totalW, totalH)
	dc.SetRGB(RGBWhite[0], RGBWhite[1], RGBWhite[2])
	dc.Clear()

	// Draw header
	x := float64(Padding)
	y := float64(Padding)
	dc.SetColor(ColorHeaderBg)
	dc.DrawRectangle(float64(Padding), y, float64(totalW-2*Padding), RowHeight)
	dc.Fill()
	dc.SetRGB(RGBBlack[0], RGBBlack[1], RGBBlack[2])
	for i, h := range headers {
		dc.DrawStringAnchored(h, x+colWidths[i]/2, y+RowHeight/2, 0.5, 0.5)
		x += colWidths[i]
	}

	// Draw rows
	for ri, row := range rows {
		y += RowHeight
		x = float64(Padding)

		metadata := rowMetadata[ri]

		if metadata.IsTopicHeader {
			// Topic header row - full width lighter blue background
			dc.SetColor(ColorTopicHeaderBg)
			dc.DrawRectangle(float64(Padding), y, float64(totalW-2*Padding), RowHeight)
			dc.Fill()
			dc.SetRGB(RGBBlack[0], RGBBlack[1], RGBBlack[2])
			for ci, cell := range row {
				dc.DrawStringAnchored(cell, x+colWidths[ci]/2, y+RowHeight/2, 0.5, 0.5)
				x += colWidths[ci]
			}
		} else {
			// Problem row - draw each cell with appropriate background
			cellX := float64(Padding)
			for ci, cell := range row {
				// Determine cell background color
				var cellBg color.RGBA
				if ci < len(metadata.CellMetadata) && metadata.CellMetadata[ci].IsAC {
					cellBg = ColorACBg // Green for AC
				} else if ri%2 == 0 {
					cellBg = ColorAltRowBg // Alternating gray
				} else {
					cellBg = ColorWhiteBg // White
				}

				// Draw cell background
				dc.SetColor(cellBg)
				dc.DrawRectangle(cellX, y, colWidths[ci], RowHeight)
				dc.Fill()

				// Draw cell text (always black)
				dc.SetRGB(RGBBlack[0], RGBBlack[1], RGBBlack[2])
				dc.DrawStringAnchored(cell, cellX+colWidths[ci]/2, y+RowHeight/2, 0.5, 0.5)

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
