package services

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestMapProblemToUpdateProblemParamsPreservesIdentityClassAndOptionalValues(t *testing.T) {
	t.Parallel()
	text := func(value string) *string { return &value }
	for _, test := range []struct {
		name    string
		problem service_model.Problem
		want    service_model.UpdateProblemParams
	}{
		{
			name: "populated problem keeps all values",
			problem: service_model.Problem{
				ID: 42, ClassID: 7, Name: "Two Sum", URL: "https://example.com/two-sum",
				Difficulty: text("Easy"), Topic: text("Arrays"),
			},
			want: service_model.UpdateProblemParams{
				ProblemID: 42, ClassCode: "basicos", Name: "Two Sum", URL: "https://example.com/two-sum",
				Difficulty: text("Easy"), Topic: text("Arrays"),
			},
		},
		{
			name: "different problem keeps its own values with both optional fields missing",
			problem: service_model.Problem{
				ID: 93, ClassID: 12, Name: "Reverse Linked List", URL: "https://example.com/reverse-list",
			},
			want: service_model.UpdateProblemParams{
				ProblemID: 93, ClassCode: "basicos", Name: "Reverse Linked List", URL: "https://example.com/reverse-list",
			},
		},
		{
			name: "missing difficulty stays nil while topic is preserved",
			problem: service_model.Problem{
				ID: 58, ClassID: 4, Name: "Search Insert Position", URL: "https://example.com/search-insert",
				Topic: text("Binary search"),
			},
			want: service_model.UpdateProblemParams{
				ProblemID: 58, ClassCode: "basicos", Name: "Search Insert Position", URL: "https://example.com/search-insert",
				Topic: text("Binary search"),
			},
		},
		{
			name: "missing topic stays nil while raw difficulty is copied without normalization",
			problem: service_model.Problem{
				ID: 81, ClassID: 9, Name: "Two Sum con números", URL: "https://example.com/two-sum-numeros",
				Difficulty: text(" medium "),
			},
			want: service_model.UpdateProblemParams{
				ProblemID: 81, ClassCode: "basicos", Name: "Two Sum con números", URL: "https://example.com/two-sum-numeros",
				Difficulty: text(" medium "),
			},
		},
		{
			name: "explicit empty optional values stay present instead of becoming nil",
			problem: service_model.Problem{
				ID: 106, ClassID: 3, Name: "Valid Parentheses", URL: "https://example.com/parentheses",
				Difficulty: text(""), Topic: text(""),
			},
			want: service_model.UpdateProblemParams{
				ProblemID: 106, ClassCode: "basicos", Name: "Valid Parentheses", URL: "https://example.com/parentheses",
				Difficulty: text(""), Topic: text(""),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, MapProblemToUpdateProblemParams(test.problem, "basicos"))
		})
	}
}
