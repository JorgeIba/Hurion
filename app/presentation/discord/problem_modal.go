package discord

// ProblemModalFields contains raw editable text, including invalid values a
// mentor may need to correct. It does not normalize or validate domain values.
type ProblemModalFields struct {
	Name       string
	URL        string
	Difficulty string
	Priority   string
	Topic      string
}

// CreateProblemModal holds editable values and the class selected before opening the form.
type CreateProblemModal struct {
	ClassCode string
	ProblemModalFields
}

// EditProblemModal holds editable values and the destination class selected before opening the form.
// ProblemID identifies the problem being edited and is encoded in the modal custom ID.
type EditProblemModal struct {
	ProblemID int64
	ClassCode string
	ProblemModalFields
}
