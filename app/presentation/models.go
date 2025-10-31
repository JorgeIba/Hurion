package presentation

import "github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"

// CellMetadata contains semantic information about a table cell
type CellMetadata struct {
	IsAC          bool
	IsProblemName bool
	IsUsername    bool
}

// RowMetadata contains semantic information for a row
type RowMetadata struct {
	IsTopicHeader bool
	Problem       *data.Problem  // nil for topic headers, contains problem reference for problem rows
	CellMetadata  []CellMetadata
}

// TableScore represents a formatted score table for display or export
type TableScore struct {
	Class       string
	Header      []string
	Table       [][]string
	RowMetadata []RowMetadata // Semantic metadata for each row
}
