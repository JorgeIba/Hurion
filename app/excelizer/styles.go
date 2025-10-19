package excelizer

import "github.com/xuri/excelize/v2"

// Common color palette
const (
	ColorGreenDark   = "0D4F0D" // Dark green - headers
	ColorGreenMedium = "4CAF50" // Medium green - totals/highlights
	ColorGreenLight  = "81C784" // Light green - AC cells
	ColorGreenPastel = "C8E6C9" // Pastel green - alternating rows/first column
	ColorBlueLink    = "0563C1" // Blue for hyperlinks
	ColorWhite       = "FFFFFF"
	ColorBlack       = "000000"
)

var (
	// HeaderStyleDef is the standard header style (Dark green with white text)
	HeaderStyleDef = excelize.Style{
		Font: &excelize.Font{
			Bold:   false,
			Size:   11,
			Color:  ColorWhite,
			Family: "Arial",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{ColorGreenDark},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	// TotalsStyleDef is the totals style (Medium green with black text)
	TotalsStyleDef = excelize.Style{
		Font: &excelize.Font{
			Bold:   false,
			Size:   11,
			Color:  ColorBlack,
			Family: "Arial",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{ColorGreenMedium},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	// ACStyleDef is the AC cell style (Light green with black text)
	ACStyleDef = excelize.Style{
		Font: &excelize.Font{
			Bold:   false,
			Size:   11,
			Color:  ColorBlack,
			Family: "Arial",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{ColorGreenLight},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	// PastelRowStyleDef is a pastel row style (Pastel green with black text)
	PastelRowStyleDef = excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Color:  ColorBlack,
			Family: "Arial",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{ColorGreenPastel},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	// PlainCenteredStyleDef is a plain centered style (no fill, black text)
	PlainCenteredStyleDef = excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Color:  ColorBlack,
			Family: "Arial",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	// PlainLeftAlignedStyleDef is a plain left-aligned style (no fill, black text)
	PlainLeftAlignedStyleDef = excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Color:  ColorBlack,
			Family: "Arial",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	}

	// HyperlinkStyleDef is a hyperlink style (blue with underline, left-aligned)
	HyperlinkStyleDef = excelize.Style{
		Font: &excelize.Font{
			Size:      11,
			Color:     ColorBlueLink,
			Family:    "Arial",
			Underline: "single",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	}

	// PastelColumnStyleDef is a pastel green column style, centered
	PastelColumnStyleDef = excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Color:  ColorBlack,
			Family: "Arial",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{ColorGreenPastel},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}
)

func CreateHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&HeaderStyleDef)
}

func CreateTotalsStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&TotalsStyleDef)
}

func CreateACStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&ACStyleDef)
}

func CreatePastelRowStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&PastelRowStyleDef)
}

func CreatePlainCenteredStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&PlainCenteredStyleDef)
}

func CreatePlainLeftAlignedStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&PlainLeftAlignedStyleDef)
}

func CreateHyperlinkStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&HyperlinkStyleDef)
}

func CreatePastelColumnStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&PastelColumnStyleDef)
}
