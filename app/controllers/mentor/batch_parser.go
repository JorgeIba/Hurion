package mentor

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxBatchOperations = 100

type problemBatchEntry struct {
	entry int
	input problemBatchInput
}

type acBatchEntry struct {
	entry int
	input acBatchInput
}

// Entry numbers are one-based. Zero identifies an input-wide error; line and
// column locate JSON syntax errors independently of entry formatting.
type batchInputError struct {
	entry   int
	line    int
	column  int
	message string
}

type problemBatchInput struct {
	Class      string  `json:"class"`
	Name       string  `json:"name"`
	URL        string  `json:"url"`
	Difficulty *string `json:"difficulty"`
	Topic      *string `json:"topic"`
	Priority   *string `json:"priority"`
}

type acBatchInput struct {
	UserID    string `json:"user_id"`
	ProblemID int64  `json:"problem_id"`
}

// parseProblemBatch decodes JSON without normalizing or validating field values.
// Successful entries still require preparation before preview or execution.
func parseProblemBatch(input string) ([]problemBatchEntry, []batchInputError) {
	items, issues := parseBatchItems(input)
	if len(issues) > 0 {
		return nil, issues
	}

	var entries []problemBatchEntry
	for index, item := range items {
		var value problemBatchInput

		if err := decodeBatchItem(item, &value); err != nil {
			issues = append(issues, batchInputError{entry: index + 1, message: err.Error()})
			continue
		}

		entries = append(entries, problemBatchEntry{entry: index + 1, input: value})
	}

	if len(issues) > 0 {
		return nil, issues
	}
	return entries, nil
}

// parseACBatch preserves supplied targets, including missing or invalid values.
// ID validity, enrollment, existence, and duplicates are checked by preparation.
func parseACBatch(input string) ([]acBatchEntry, []batchInputError) {
	items, issues := parseBatchItems(input)
	if len(issues) > 0 {
		return nil, issues
	}

	var entries []acBatchEntry
	for index, item := range items {
		var value acBatchInput

		if err := decodeBatchItem(item, &value); err != nil {
			issues = append(issues, batchInputError{entry: index + 1, message: err.Error()})
			continue
		}

		entries = append(entries, acBatchEntry{entry: index + 1, input: value})
	}

	if len(issues) > 0 {
		return nil, issues
	}
	return entries, nil
}

func parseBatchItems(input string) ([]jsontext.Value, []batchInputError) {
	// Defer Unicode and duplicate-field checks to strict entry decoding so both
	// parsers can collect errors for multiple entries.
	var items []jsontext.Value
	if err := jsonv2.Unmarshal([]byte(input), &items, jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true)); err != nil {
		issue := batchInputError{message: "expected one JSON array: " + err.Error()}
		if syntax, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
			offset := max(0, min(len(input), int(syntax.ByteOffset)))
			before := input[:offset]
			issue.line = strings.Count(before, "\n") + 1
			issue.column = utf8.RuneCountInString(before[strings.LastIndex(before, "\n")+1:]) + 1
		}

		return nil, []batchInputError{issue}
	}

	if items == nil {
		return nil, []batchInputError{{message: "expected one JSON array, not null"}}
	}

	if len(items) == 0 {
		return nil, []batchInputError{{message: "batch must contain at least one operation"}}
	}

	if len(items) > maxBatchOperations {
		return nil, []batchInputError{{message: fmt.Sprintf("batch must contain at most %d operations", maxBatchOperations)}}
	}
	return items, nil
}

func decodeBatchItem(item jsontext.Value, value any) error {
	if string(item) == "null" {
		return errors.New("entry must be a JSON object, not null")
	}
	return jsonv2.Unmarshal(item, value, jsonv2.RejectUnknownMembers(true))
}
