package discord

// ProblemModalFields contains raw editable text, including invalid values a
// mentor may need to correct. It does not normalize or validate domain values.
type ProblemModalFields struct {
	Name       string
	URL        string
	Difficulty string
	Topic      string
}

// CreateProblemModal holds the editable values for a problem-creation form.
type CreateProblemModal struct {
	ClassCode string
	ProblemModalFields
}

// EditProblemModal holds the editable values for a problem-editing form.
// ProblemID is display metadata for the title, not an editable input. Saving
// must use the target ID from the controller's owned session.
type EditProblemModal struct {
	ProblemID int64
	ClassCode string
	ProblemModalFields
}
