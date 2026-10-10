package mentor

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestBatchInputValidationRejectsMissingNullAndEmptyRequiredJSONValues(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, input string
		problems    bool
	}{
		{name: "missing problem values", input: `[{}]`, problems: true},
		{name: "null problem values", input: `[{"class":null,"name":null,"url":null}]`, problems: true},
		{name: "empty problem values", input: `[{"class":"","name":"","url":""}]`, problems: true},
		{name: "missing AC values", input: `[{}]`},
		{name: "null AC values", input: `[{"user_id":null,"problem_id":null}]`},
		{name: "empty AC values", input: `[{"user_id":"","problem_id":0}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.problems {
				entries, parseIssues := parseProblemBatchInput(documentForTest(t, test.input))
				assert.Empty(t, parseIssues)
				issues := entries.validate(validateProblemInput)
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 1, issues[0].entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issues[0].err)
				}
			} else {
				entries, parseIssues := parseACBatchInput(documentForTest(t, test.input))
				assert.Empty(t, parseIssues)
				issues := entries.validate(validateACInput)
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 1, issues[0].entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issues[0].err)
				}
			}
		})
	}
}

func TestACInputValidationRejectsMalformedTargetsBeforeDiscord(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		input       string
		wantEntries []int
	}{
		{name: "non-numeric student", input: `[{"user_id":"Ana","problem_id":42}]`, wantEntries: []int{1}},
		{name: "overflowing student ID", input: `[{"user_id":"18446744073709551616","problem_id":42}]`, wantEntries: []int{1}},
		{name: "negative problem", input: `[{"user_id":"123","problem_id":-1}]`, wantEntries: []int{1}},
		{name: "valid target followed by invalid target", input: `[{"user_id":"123","problem_id":42},{"user_id":"Ana","problem_id":42}]`, wantEntries: []int{2}},
		{name: "all invalid entries are reported", input: `[{"user_id":"Ana","problem_id":42},{"user_id":"123","problem_id":0}]`, wantEntries: []int{1, 2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, parseIssues := parseACBatchInput(documentForTest(t, test.input))
			assert.Empty(t, parseIssues)
			issues := entries.validate(validateACInput)
			if assert.Len(t, issues, len(test.wantEntries)) {
				for index, issue := range issues {
					assert.Equal(t, test.wantEntries[index], issue.entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issue.err)
				}
			}
		})
	}
}

func TestBatchInputSizePrecedesEntryValidation(t *testing.T) {
	for _, size := range []int{0, service_model.MaxBatchOperations + 1} {
		input := make(batchInput[acInput], size)
		issues := input.validate(func(acInput) error { t.Fatal("size must fail before validating entries"); return nil })
		if assert.Len(t, issues, 1) {
			assert.Zero(t, issues[0].entry)
			assert.IsType(t, &serviceerr.ValidationError{}, issues[0].err)
		}
	}
}
