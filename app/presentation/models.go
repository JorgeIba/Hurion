package presentation

// CellMetadata contains semantic information about a table cell
type CellMetadata struct {
	IsAC          bool
	IsProblemName bool
	IsUsername    bool
	IsTotal       bool // True if this cell is part of the totals row
}

const (
	DifficultyEasy   = "Easy"
	DifficultyMedium = "Medium"
	DifficultyHard   = "Hard"
)

// Problem represents a problem in the presentation layer
type Problem struct {
	ID         int64
	Name       string
	URL        string
	Difficulty string
	Topic      string
}

// RowMetadata contains semantic information for a row
type RowMetadata struct {
	IsTopicHeader bool
	IsTotalsRow   bool     // True if this is the totals row
	Problem       *Problem // Presentation-specific problem model
	CellMetadata  []CellMetadata
}

// TableScore represents a formatted score table for display or export
type TableScore struct {
	Class       string
	Header      []string
	Table       [][]string
	RowMetadata []RowMetadata // Semantic metadata for each row
}

// ProblemsByTopic is a presentation-layer model representing problems grouped by topic.
type ProblemsByTopic struct {
	Topic    string
	Problems []Problem
}
