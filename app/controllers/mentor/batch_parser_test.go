package mentor

import (
	"strings"
	"testing"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestParseProblemBatch(t *testing.T) {
	t.Parallel()
	text := func(value string) *string { return &value }
	for _, test := range []struct {
		name  string
		input string
		want  []problemBatchEntry
	}{
		{
			name:  "normalizes metadata and retains multiword Unicode names",
			input: "!add_problem basicos https://example.com/problem HARD Árboles RECOMMENDED Árbol binario",
			want: []problemBatchEntry{{line: 1, params: service_model.CreateProblemParams{
				ClassCode: "basicos", ProblemFields: service_model.ProblemFields{
					URL: "https://example.com/problem", Name: "Árbol binario",
					Difficulty: text("hard"), Topic: text("Árboles"), Priority: text("recommended"),
				},
			}}},
		},
		{
			name:  "optional placeholders and CRLF keep original line numbers",
			input: "\r\n \t\r\n\t!add_problem basicos https://example.com/one - - - Two   Sum\r\n\r\n!add_problem avanzados https://example.com/two - DP not_required Otro problema\r\n",
			want: []problemBatchEntry{
				{line: 3, params: service_model.CreateProblemParams{
					ClassCode: "basicos", ProblemFields: service_model.ProblemFields{URL: "https://example.com/one", Name: "Two Sum"},
				}},
				{line: 5, params: service_model.CreateProblemParams{
					ClassCode: "avanzados", ProblemFields: service_model.ProblemFields{
						URL: "https://example.com/two", Name: "Otro problema", Topic: text("DP"), Priority: text("not_required"),
					},
				}},
			},
		},
		{
			name:  "identical problems are both preserved",
			input: "!add_problem basicos https://example.com/problem - - - Same name\n!add_problem basicos https://example.com/problem - - - Same name",
			want: []problemBatchEntry{
				{line: 1, params: service_model.CreateProblemParams{
					ClassCode: "basicos", ProblemFields: service_model.ProblemFields{URL: "https://example.com/problem", Name: "Same name"},
				}},
				{line: 2, params: service_model.CreateProblemParams{
					ClassCode: "basicos", ProblemFields: service_model.ProblemFields{URL: "https://example.com/problem", Name: "Same name"},
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, issues := parseProblemBatch(test.input)
			assert.Empty(t, issues)
			assert.Equal(t, test.want, entries)
		})
	}
}

func TestParseProblemBatchReportsAllInvalidLinesAndReturnsNoEntries(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		input    string
		lines    []int
		messages []string
	}{
		{
			name: "mixed operations cannot return valid subset",
			input: "!add_problem basicos https://example.com/one - - - Valid\n\n" +
				"!ac <@123> 42\n!update_problem 42 basicos https://example.com/one - - - Update",
			lines: []int{3, 4}, messages: []string{"expected !add_problem", "expected !add_problem"},
		},
		{
			name: "missing name and invalid values are both reported",
			input: "!add_problem basicos https://example.com/one - - -\n" +
				"!add_problem basicos javascript:alert(1) Extreme DP urgent Invalid\n" +
				"!add_problem basicos https://example.com/two - - - Valid",
			lines: []int{1, 2}, messages: []string{"expected !add_problem", "URL must be an absolute HTTP or HTTPS URL"},
		},
		{
			name:  "existing field length limit applies",
			input: "!add_problem basicos https://example.com/one - - - " + strings.Repeat("á", service_model.ProblemNameMaxLength+1),
			lines: []int{1}, messages: []string{"problem name must be at most 200 characters"},
		},
		{
			name:  "markdown fences and comments are not instructions",
			input: "```\n!add_problem basicos https://example.com/one - - - Valid\n```\n# comment",
			lines: []int{1, 3, 4}, messages: []string{"expected !add_problem", "expected !add_problem", "expected !add_problem"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, issues := parseProblemBatch(test.input)
			assert.Nil(t, entries)
			assertBatchIssues(t, issues, test.lines, test.messages)
		})
	}
}

func TestParseACBatch(t *testing.T) {
	t.Parallel()
	entries, issues := parseACBatch("\r\n!ac <@123> 42\r\n \t\r\n!ac <@!456> 43\r\n!ac 123 42\r\n")
	assert.Empty(t, issues)
	// Repeated ACs remain visible for the preparation service to report; the
	// parser must not silently deduplicate or hide their original line numbers.
	assert.Equal(t, []acBatchEntry{
		{line: 2, userID: "123", problemID: 42},
		{line: 4, userID: "456", problemID: 43},
		{line: 5, userID: "123", problemID: 42},
	}, entries)
}

func TestParseACBatchReportsAllInvalidLinesAndReturnsNoEntries(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		input    string
		lines    []int
		messages []string
	}{
		{
			name:  "explicit student and exact argument count required",
			input: "!ac 42\n!ac 123 42 extra\n!undo_ac 123 42\n!ac 123 42",
			lines: []int{1, 2, 3}, messages: []string{"expected !ac", "expected !ac", "expected !ac"},
		},
		{
			name:  "names role mentions and malformed mentions are rejected",
			input: "!ac @Ana 42\n!ac <@&123> 42\n!ac <@!abc> 42",
			lines: []int{1, 2, 3}, messages: []string{"user must be", "user must be", "user must be"},
		},
		{
			name:  "nonpositive malformed and overflowing problem IDs are rejected",
			input: "!ac 123 0\n!ac 123 -1\n!ac 123 abc\n!ac 123 9223372036854775808",
			lines: []int{1, 2, 3, 4}, messages: []string{"problem ID must be", "problem ID must be", "problem ID must be", "problem ID must be"},
		},
		{
			name:  "all errors on a line are retained",
			input: "!ac @Ana -1\n!ac 123 42",
			lines: []int{1, 1}, messages: []string{"user must be", "problem ID must be"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, issues := parseACBatch(test.input)
			assert.Nil(t, entries)
			assertBatchIssues(t, issues, test.lines, test.messages)
		})
	}
}

func TestBatchInputLimitsApplyToBothParsers(t *testing.T) {
	t.Parallel()
	for _, parser := range []struct {
		name  string
		valid string
		parse func(string) (int, []batchLineError)
	}{
		{name: "problems", valid: "!add_problem basicos https://example.com/one - - - Valid", parse: func(input string) (int, []batchLineError) {
			entries, issues := parseProblemBatch(input)
			return len(entries), issues
		}},
		{name: "ACs", valid: "!ac 123 42", parse: func(input string) (int, []batchLineError) {
			entries, issues := parseACBatch(input)
			return len(entries), issues
		}},
	} {
		t.Run(parser.name, func(t *testing.T) {
			t.Parallel()
			for _, test := range []struct {
				name    string
				input   string
				count   int
				line    int
				message string
			}{
				{name: "empty", message: "at least one operation"},
				{name: "only whitespace", input: "\r\n \t\n", message: "at least one operation"},
				{name: "invalid UTF-8", input: parser.valid + "\xff", message: "valid UTF-8"},
				{name: "byte limit is inclusive", input: parser.valid + strings.Repeat(" ", maxBatchInputBytes-len(parser.valid)), count: 1},
				{name: "too many bytes", input: parser.valid + strings.Repeat(" ", maxBatchInputBytes), message: "at most 262144 bytes"},
				{name: "100 operations with blank lines", input: strings.Repeat(parser.valid+"\n\n", maxBatchOperations), count: 100},
				{name: "101 operations", input: strings.Repeat(parser.valid+"\n\n", maxBatchOperations) + parser.valid, line: 201, message: "at most 100 operations"},
			} {
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()
					count, issues := parser.parse(test.input)
					assert.Equal(t, test.count, count)
					if test.message == "" {
						assert.Empty(t, issues)
					} else {
						assertBatchIssues(t, issues, []int{test.line}, []string{test.message})
					}
				})
			}
		})
	}
}

func assertBatchIssues(t *testing.T, issues []batchLineError, lines []int, messages []string) {
	t.Helper()
	if !assert.Len(t, issues, len(lines)) {
		return
	}
	for index, line := range lines {
		assert.Equal(t, line, issues[index].line)
		assert.Contains(t, issues[index].message, messages[index])
	}
}
