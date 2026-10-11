package jsonexport

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// JSONClassSummary represents the public class metadata exported for clients and AI agents.
type JSONClassSummary struct {
	ClassCode string `json:"classCode"`
	Name      string `json:"name"`
}

// FormatClasses converts domain classes into a deterministic, sorted JSON array.
func FormatClasses(classes []service_model.Class) ([]byte, error) {
	summaries := make([]JSONClassSummary, 0, len(classes))
	for _, c := range classes {
		summaries = append(summaries, JSONClassSummary{
			ClassCode: c.ClassCode,
			Name:      c.Name,
		})
	}

	slices.SortFunc(summaries, func(a, b JSONClassSummary) int {
		return strings.Compare(a.ClassCode, b.ClassCode)
	})

	return json.MarshalIndent(summaries, "", "  ")
}
