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

	// Difficulty colors - Pastel versions
	ColorDifficultyEasy   = "A5D6A7" // Pastel green for Easy
	ColorDifficultyMedium = "FFF59D" // Pastel yellow for Medium
	ColorDifficultyHard   = "EF9A9A" // Pastel red for Hard
	ColorDifficultyNone   = "E0E0E0" // Gray for Not specified
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

	// TotalsStyleDef is the totals style (Pastel green with black text)
	TotalsStyleDef = excelize.Style{
		Font: &excelize.Font{
			Bold:   false,
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

	// ACStyleDef is the AC cell style (Pastel green with black text - same as totals)
	ACStyleDef = excelize.Style{
		Font: &excelize.Font{
			Bold:   false,
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

	// Creates a style for topic header rows (medium green background, centered, no bold)
	TopicHeaderStyleDef = excelize.Style{
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

func CreateTopicHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&TopicHeaderStyleDef)
}

// Creates a style for difficulty cells based on the difficulty level
func CreateDifficultyStyle(f *excelize.File, difficulty string) (int, error) {
	var color string
	switch difficulty {
	case "Easy":
		color = ColorDifficultyEasy
	case "Medium":
		color = ColorDifficultyMedium
	case "Hard":
		color = ColorDifficultyHard
	default:
		color = ColorDifficultyNone
	}

	return f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{color},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
}
