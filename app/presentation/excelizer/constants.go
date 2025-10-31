package excelizer

// Column width constants
const (
	// MinColumnWidth is the minimum width for any column
	MinColumnWidth = 10.0

	// MaxColumnWidth is the maximum width to avoid extremely wide columns
	MaxColumnWidth = 90.0

	// ColumnWidthBuffer is the extra space added to column width for padding
	ColumnWidthBuffer = 5.0

	// ColumnWidthMultiplier is used to calculate width based on content length
	ColumnWidthMultiplier = 1.2
)

// Row and column positions
const (
	// HeaderRow is the row number for headers (1-indexed)
	HeaderRow = 1

	// FirstDataRow is the first row for actual data (1-indexed)
	FirstDataRow = 2

	// ScoresHeaderRow is the first row in scores sheet (user names)
	ScoresHeaderRow = 1

	// ScoresTotalsRow is the second row in scores sheet (totals)
	ScoresTotalsRow = 2

	// ScoresFirstDataRow is the first row for problem data in scores sheet
	ScoresFirstDataRow = 3

	// FirstColumn is the ASCII code for column A
	FirstColumn = 'A'

	// SecondColumn is the ASCII code for column B
	SecondColumn = 'B'
)

// Freeze panes configuration
const (
	// FreezeFirstRow freezes only the first row (headers)
	FreezeFirstRow = 1

	// FreezeFirstColumn freezes only the first column
	FreezeFirstColumn = 1

	// FreezeTwoRows freezes the first two rows
	FreezeTwoRows = 2

	// NoFreeze indicates no freezing on that axis
	NoFreeze = 0
)

// Cell references for freeze panes
const (
	// TopLeftCellOneRow is used when freezing first row only
	TopLeftCellOneRow = "A2"

	// TopLeftCellOneColumn is used when freezing first row and first column
	TopLeftCellOneColumn = "B2"

	// TopLeftCellTwoRows is used when freezing first two rows and first column
	TopLeftCellTwoRows = "B3"

	// ActivePaneBottom is the active pane when freezing rows
	ActivePaneBottom = "bottomLeft"

	// ActivePaneBottomRight is the active pane when freezing rows and columns
	ActivePaneBottomRight = "bottomRight"
)

// Sheet-specific constants
const (
	// ScoresSheetOffsetRows is the number of header/totals rows before data in scores sheet
	ScoresSheetOffsetRows = 2

	// ScoresSheetOffsetColumns is the offset for problem names column in scores sheet
	ScoresSheetOffsetColumns = 1
)

// Status indicators
const (
	// StatusAccepted is the marker for solved problems
	StatusAccepted = "AC"

	// TotalSolvedLabel is the label for the totals row in scores sheet
	TotalSolvedLabel = "Total Solved"
)
