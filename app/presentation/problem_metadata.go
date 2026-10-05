package presentation

import "github.com/JorgeIba/Hurion/app/services/service_model"

const DifficultyUnspecifiedLabel = "Not specified"
const PriorityUnspecifiedLabel = "No priority assigned"

// DifficultyLabel formats a canonical code without changing the stored value.
// Unknown text is preserved so an editor can show it for correction.
func DifficultyLabel(code *string) string {
	if code == nil {
		return DifficultyUnspecifiedLabel
	}
	switch *code {
	case service_model.DifficultyEasy:
		return "Easy"
	case service_model.DifficultyMedium:
		return "Medium"
	case service_model.DifficultyHard:
		return "Hard"
	default:
		return *code
	}
}

// PriorityLabel formats a canonical code without changing the stored value.
// Unknown text is preserved so it remains distinguishable from an unassigned priority.
func PriorityLabel(code *string) string {
	if code == nil {
		return PriorityUnspecifiedLabel
	}
	switch *code {
	case service_model.PriorityEssential:
		return "Essential"
	case service_model.PriorityRecommended:
		return "Recommended"
	case service_model.PrioritySupplementary:
		return "Supplementary"
	case service_model.PriorityNotRequired:
		return "Not required"
	case service_model.PriorityDiscouraged:
		return "Discouraged"
	default:
		return *code
	}
}
