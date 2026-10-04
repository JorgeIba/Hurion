package presentation

import "github.com/JorgeIba/Hurion/app/services/service_model"

const DifficultyUnspecifiedLabel = "Not specified"

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
