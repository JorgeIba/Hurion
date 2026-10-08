package mentor

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

const (
	maxBatchOperations = 100
	maxBatchInputBytes = 256 * 1024
)

type problemBatchEntry struct {
	line   int
	params service_model.CreateProblemParams
}

type acBatchEntry struct {
	line      int
	userID    string
	problemID int64
}

// A zero line identifies an error concerning the whole input.
type batchLineError struct {
	line    int
	message string
}

type batchTextLine struct {
	number int
	fields []string
}

// parseProblemBatch checks text and problem values without looking up classes,
// checking permissions, or writing. Any error prevents returning a usable batch.
func parseProblemBatch(input string) ([]problemBatchEntry, []batchLineError) {
	lines, issues := splitBatchLines(input)
	if len(issues) > 0 {
		return nil, issues
	}

	var entries []problemBatchEntry
	for _, line := range lines {
		fields := line.fields
		if len(fields) < 7 || fields[0] != "!add_problem" {
			issues = append(issues, batchLineError{line.number, "expected !add_problem <class> <url> <difficulty or -> <topic or -> <priority or -> <name>"})
			continue
		}

		params, err := service_model.NormalizeAndValidateCreateProblemParams(service_model.CreateProblemParams{
			ClassCode: fields[1],
			ProblemFields: service_model.ProblemFields{
				URL:        fields[2],
				Difficulty: optionalBatchField(fields[3]),
				Topic:      optionalBatchField(fields[4]),
				Priority:   optionalBatchField(fields[5]),
				Name:       strings.Join(fields[6:], " "),
			},
		})
		if err != nil {
			issues = append(issues, batchLineError{line.number, err.Error()})
			continue
		}
		entries = append(entries, problemBatchEntry{line.number, params})
	}
	if len(issues) > 0 {
		return nil, issues
	}
	return entries, nil
}

// parseACBatch reads explicit student ACs. Enrollment, problem existence, and
// duplicate checks belong to the later read-only preparation operation.
func parseACBatch(input string) ([]acBatchEntry, []batchLineError) {
	lines, issues := splitBatchLines(input)
	if len(issues) > 0 {
		return nil, issues
	}

	var entries []acBatchEntry
	for _, line := range lines {
		fields := line.fields
		if len(fields) != 3 || fields[0] != "!ac" {
			issues = append(issues, batchLineError{line.number, "expected !ac <user mention or ID> <problem ID>"})
			continue
		}

		userID, validUser := controllers_common.ParseUserID(fields[1])
		problemID, err := strconv.ParseInt(fields[2], 10, 64)
		if !validUser {
			issues = append(issues, batchLineError{line.number, "user must be a Discord user mention or numeric ID"})
		}
		if err != nil || problemID <= 0 {
			issues = append(issues, batchLineError{line.number, "problem ID must be a positive 64-bit integer"})
		}
		if validUser && err == nil && problemID > 0 {
			entries = append(entries, acBatchEntry{line.number, userID, problemID})
		}
	}
	if len(issues) > 0 {
		return nil, issues
	}
	return entries, nil
}

func splitBatchLines(input string) ([]batchTextLine, []batchLineError) {
	if len(input) > maxBatchInputBytes {
		return nil, []batchLineError{{message: fmt.Sprintf("batch input must be at most %d bytes", maxBatchInputBytes)}}
	}
	if !utf8.ValidString(input) {
		return nil, []batchLineError{{message: "batch input must be valid UTF-8 text"}}
	}

	var lines []batchTextLine
	for index, text := range strings.Split(input, "\n") {
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		if len(lines) == maxBatchOperations {
			return nil, []batchLineError{{line: index + 1, message: fmt.Sprintf("batch must contain at most %d operations", maxBatchOperations)}}
		}
		lines = append(lines, batchTextLine{index + 1, fields})
	}
	if len(lines) == 0 {
		return nil, []batchLineError{{message: "batch must contain at least one operation"}}
	}
	return lines, nil
}

func optionalBatchField(value string) *string {
	if value == "-" {
		return nil
	}
	return &value
}
