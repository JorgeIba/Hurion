package image

import "image/color"

// Layout constants for image generation
const (
	FontSize      = 25
	RowHeight     = 35
	Padding       = 40
	ColumnSpacing = 15
)

// Color palette for image rendering
var (
	// Background colors
	ColorHeaderBg      = color.RGBA{150, 180, 230, 255} // Softer blue for header
	ColorTotalsBg      = color.RGBA{180, 200, 240, 255} // Lighter blue for totals (was topic header)
	ColorTopicHeaderBg = color.RGBA{100, 181, 246, 255} // Pastel blue for topic headers (was totals)
	ColorACBg          = color.RGBA{180, 231, 206, 255} // Pastel green for AC cells (#B4E7CE)
	ColorAltRowBg      = color.RGBA{245, 245, 245, 255} // Alternating row background
	ColorWhiteBg       = color.RGBA{255, 255, 255, 255} // White background
	ColorCanvasBg      = color.RGBA{255, 255, 255, 255} // Canvas background (white)

	// Text colors
	ColorTextBlack = color.RGBA{0, 0, 0, 255} // Black text (default)
)

// RGB color values for convenience
var (
	RGBWhite = [3]float64{1, 1, 1} // White in RGB (0-1 scale)
	RGBBlack = [3]float64{0, 0, 0} // Black in RGB (0-1 scale)
)
