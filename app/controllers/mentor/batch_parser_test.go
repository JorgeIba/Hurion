package mentor

import (
	"strings"
	"testing"

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
			name:  "preserves supplied spaces casing Unicode and escaped quotes",
			input: `[{"class":" basicos ","url":"https://example.com/problem","name":" Árbol \"binario\" ","difficulty":"HARD","topic":" Árboles binarios ","priority":"RECOMMENDED"}]`,
			want: []problemBatchEntry{{entry: 1, input: problemBatchInput{
				Class: " basicos ", URL: "https://example.com/problem", Name: ` Árbol "binario" `,
				Difficulty: text("HARD"), Topic: text(" Árboles binarios "), Priority: text("RECOMMENDED"),
			}}},
		},
		{
			name: "omitted null and blank optional fields and multiple classes",
			input: "[\r\n" + `{"class":"basicos","url":"https://example.com/one","name":"Two   Sum"},` + "\r\n\r\n" +
				`{"class":"avanzados","url":"https://example.com/two","name":"Otro problema","difficulty":null,"topic":" ","priority":"not_required"}]`,
			want: []problemBatchEntry{
				{entry: 1, input: problemBatchInput{Class: "basicos", URL: "https://example.com/one", Name: "Two   Sum"}},
				{entry: 2, input: problemBatchInput{Class: "avanzados", URL: "https://example.com/two", Name: "Otro problema", Topic: text(" "), Priority: text("not_required")}},
			},
		},
		{
			name:  "missing fields and invalid business values are left for preparation",
			input: `[{}, {"class":" ","url":"javascript:alert(1)","name":"","difficulty":"extreme","topic":"` + strings.Repeat("á", 101) + `","priority":"urgent"}]`,
			want: []problemBatchEntry{
				{entry: 1, input: problemBatchInput{}},
				{entry: 2, input: problemBatchInput{Class: " ", URL: "javascript:alert(1)", Difficulty: text("extreme"), Topic: text(strings.Repeat("á", 101)), Priority: text("urgent")}},
			},
		},
		{
			name:  "identical problems are both preserved",
			input: `[{"class":"basicos","url":"https://example.com/problem","name":"Same name"},{"class":"basicos","url":"https://example.com/problem","name":"Same name"}]`,
			want: []problemBatchEntry{
				{entry: 1, input: problemBatchInput{Class: "basicos", URL: "https://example.com/problem", Name: "Same name"}},
				{entry: 2, input: problemBatchInput{Class: "basicos", URL: "https://example.com/problem", Name: "Same name"}},
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

func TestParseProblemBatchReportsInvalidEntriesWithoutPartialResults(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		input    string
		entries  []int
		messages []string
	}{
		{
			name:    "mixed operations and unknown fields",
			input:   `[{"class":"basicos","url":"https://example.com/one","name":"Valid"},{"user_id":"123","problem_id":42},{"class":"basicos","url":"https://example.com/one","name":"Typo","dificulty":"easy"}]`,
			entries: []int{2, 3}, messages: []string{"unknown object member name", "unknown object member name"},
		},
		{
			name:    "optional metadata must be text",
			input:   `[{"class":"basicos","url":"https://example.com/one","name":"Invalid","topic":42}]`,
			entries: []int{1}, messages: []string{"JSON number into Go string"},
		},
		{
			name:    "duplicate fields and mismatched casing",
			input:   `[{"name":"Valid"},{"name":"First","name":"Second"},{"Name":"Typo"}]`,
			entries: []int{2, 3}, messages: []string{"duplicate object member name", "unknown object member name"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, issues := parseProblemBatch(test.input)
			assert.Nil(t, entries)
			assertBatchIssues(t, issues, test.entries, test.messages)
		})
	}
}

func TestParseACBatch(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		input string
		want  []acBatchEntry
	}{
		{
			name:  "preserves large IDs duplicates and original order",
			input: `[{"user_id":"123456789012345678","problem_id":42},{"user_id":"456","problem_id":9223372036854775807},{"user_id":"123456789012345678","problem_id":42}]`,
			want: []acBatchEntry{
				{entry: 1, input: acBatchInput{UserID: "123456789012345678", ProblemID: 42}},
				{entry: 2, input: acBatchInput{UserID: "456", ProblemID: 9223372036854775807}},
				{entry: 3, input: acBatchInput{UserID: "123456789012345678", ProblemID: 42}},
			},
		},
		{
			name:  "missing targets names mentions and nonpositive IDs await preparation",
			input: `[{},{"user_id":"Ana","problem_id":0},{"user_id":"<@123>","problem_id":-1},{"user_id":"<@!123>","problem_id":-9223372036854775808}]`,
			want: []acBatchEntry{
				{entry: 1, input: acBatchInput{}},
				{entry: 2, input: acBatchInput{UserID: "Ana", ProblemID: 0}},
				{entry: 3, input: acBatchInput{UserID: "<@123>", ProblemID: -1}},
				{entry: 4, input: acBatchInput{UserID: "<@!123>", ProblemID: -9223372036854775808}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, issues := parseACBatch(test.input)
			assert.Empty(t, issues)
			assert.Equal(t, test.want, entries)
		})
	}
}

func TestParseACBatchReportsInvalidEntriesWithoutPartialResults(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		input    string
		entries  []int
		messages []string
	}{
		{
			name:    "unknown fields cannot return a valid subset",
			input:   `[{"user_id":"123","problem_id":42},{"user_id":"123","problem_id":42,"command":"ac"}]`,
			entries: []int{2}, messages: []string{"unknown object member name"},
		},
		{
			name:    "numeric user IDs and malformed problem ID types are collected",
			input:   `[{"user_id":123,"problem_id":42},{"user_id":"123","problem_id":1.5},{"user_id":"123","problem_id":"42"},{"user_id":"123","problem_id":9223372036854775808}]`,
			entries: []int{1, 2, 3, 4}, messages: []string{"JSON number into Go string", "into Go int64", "into Go int64", "into Go int64"},
		},
		{
			name:    "duplicate fields and mismatched casing",
			input:   `[{"user_id":"123","problem_id":42},{"user_id":"123","user_id":"456"},{"User_id":"123"}]`,
			entries: []int{2, 3}, messages: []string{"duplicate object member name", "unknown object member name"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, issues := parseACBatch(test.input)
			assert.Nil(t, entries)
			assertBatchIssues(t, issues, test.entries, test.messages)
		})
	}
}

func TestBatchJSONContractAppliesToBothParsers(t *testing.T) {
	t.Parallel()
	for _, parser := range []struct {
		name  string
		valid string
		parse func(string) (int, []batchInputError)
	}{
		{name: "problems", valid: `{"class":"basicos","url":"https://example.com/one","name":"Valid"}`, parse: func(input string) (int, []batchInputError) {
			entries, issues := parseProblemBatch(input)
			return len(entries), issues
		}},
		{name: "ACs", valid: `{"user_id":"123","problem_id":42}`, parse: func(input string) (int, []batchInputError) {
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
				entry   int
				message string
			}{
				{name: "empty text", message: "expected one JSON array"},
				{name: "whitespace", input: "\r\n \t\n", message: "expected one JSON array"},
				{name: "empty array", input: "[]"},
				{name: "null document", input: "null", message: "not null"},
				{name: "object document", input: parser.valid, message: "expected one JSON array"},
				{name: "invalid UTF-8 inside string", input: "[" + strings.Replace(parser.valid, `":"`, "\":\"\xff", 1) + "]", entry: 1, message: "invalid UTF-8"},
				{name: "null entry", input: "[" + parser.valid + ",null]", entry: 2, message: "entry must be a JSON object"},
				{name: "array entry", input: "[" + parser.valid + ",[]]", entry: 2, message: "JSON array into Go"},
				{name: "string entry", input: "[" + parser.valid + `,"text"]`, entry: 2, message: "JSON string into Go"},
				{name: "second document", input: "[" + parser.valid + "] []", message: "expected one JSON array"},
				{name: "markdown fences", input: "```json\n[" + parser.valid + "]\n```", message: "expected one JSON array"},
				{name: "comments", input: "[/*comment*/" + parser.valid + "]", message: "expected one JSON array"},
				{name: "100 operations", input: "[" + strings.Repeat(parser.valid+",\r\n", 99) + parser.valid + "]", count: 100},
				{name: "101 operations", input: "[" + strings.Repeat(parser.valid+",", 100) + parser.valid + "]", count: 101},
			} {
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()
					count, issues := parser.parse(test.input)
					assert.Equal(t, test.count, count)
					if test.message == "" {
						assert.Empty(t, issues)
					} else {
						assertBatchIssues(t, issues, []int{test.entry}, []string{test.message})
					}
				})
			}
		})
	}
}

func TestBatchSyntaxErrorLocation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		input  string
		line   int
		column int
	}{
		{name: "CRLF and Unicode column", input: "[\r\n  {\"name\":\"á\",?}\r\n]", line: 2, column: 15},
		{name: "trailing comma", input: "[\n{},\n]", line: 2, column: 3},
		{name: "empty input", line: 1, column: 1},
		{name: "missing array closing bracket", input: "[{}", line: 1, column: 4},
		{name: "EOF after newline", input: "[{}\n", line: 2, column: 1},
		{name: "EOF after CRLF", input: "[{}\r\n", line: 2, column: 1},
		{name: "EOF after Unicode", input: "[\"á\"", line: 1, column: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			items, issues := parseBatchItems(test.input)
			assert.Nil(t, items)
			if assert.Len(t, issues, 1) {
				assert.Zero(t, issues[0].entry)
				assert.Equal(t, test.line, issues[0].line)
				assert.Equal(t, test.column, issues[0].column)
			}
		})
	}
}

func TestBatchParsersRejectUnpairedUnicodeEscapes(t *testing.T) {
	t.Parallel()
	for _, parser := range []struct {
		name  string
		field string
		parse func(string) (int, []batchInputError)
	}{
		{name: "problems", field: "name", parse: func(input string) (int, []batchInputError) {
			entries, issues := parseProblemBatch(input)
			return len(entries), issues
		}},
		{name: "ACs", field: "user_id", parse: func(input string) (int, []batchInputError) {
			entries, issues := parseACBatch(input)
			return len(entries), issues
		}},
	} {
		t.Run(parser.name, func(t *testing.T) {
			t.Parallel()
			for _, test := range []struct {
				name  string
				value string
			}{
				{name: "isolated high surrogate", value: `\ud800`},
				{name: "isolated low surrogate", value: `\udc00`},
				{name: "high surrogate before ordinary character", value: `\ud800A`},
				{name: "high surrogate before BMP escape", value: `\ud800\u0041`},
				{name: "two high surrogates", value: `\ud800\ud800`},
				{name: "reversed pair", value: `\udc00\ud800`},
				{name: "nonadjacent pair", value: `\ud800 \udc00`},
				{name: "escaped quote before surrogate", value: `\"\uD800`},
				{name: "literal backslash before actual surrogate", value: `\\\uD800`},
			} {
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()
					input := `[{}, {"` + parser.field + `":"` + test.value + `"}, {"` + parser.field + `":"\udc00"}]`
					count, issues := parser.parse(input)
					assert.Zero(t, count)
					assertBatchIssues(t, issues, []int{2, 3}, []string{"invalid surrogate pair", "invalid surrogate pair"})
				})
			}
			// Names also contain JSON strings and must not be silently repaired.
			count, issues := parser.parse(`[{"\ud800":"value"}]`)
			assert.Zero(t, count)
			assertBatchIssues(t, issues, []int{1}, []string{"invalid surrogate pair"})
		})
	}
}

func TestParseProblemBatchPreservesValidUnicodeEscapes(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "valid emoji pair", value: `\ud83d\ude00`, want: "😀"},
		{name: "mixed case and multiple pairs", value: `\uD83D\uDe00\ud83d\ude80`, want: "😀🚀"},
		{name: "surrogate range boundaries", value: `\ud800\udc00\udbff\udfff`, want: "\U00010000\U0010FFFF"},
		{name: "literal replacement character", value: "�", want: "�"},
		{name: "escaped replacement character", value: `\ufffd`, want: "�"},
		{name: "BMP range boundaries", value: `\ud7ff\ue000`, want: "\uD7FF\uE000"},
		{name: "literal Unicode escape text", value: `\\ud800`, want: `\ud800`},
		{name: "escaped backslashes and quotes", value: `\\\"\\ud800`, want: `\"\ud800`},
		{name: "ordinary escapes", value: `\u00e1\n\t\/`, want: "á\n\t/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entries, issues := parseProblemBatch(`[{"name":"` + test.value + `"}]`)
			assert.Empty(t, issues)
			if assert.Len(t, entries, 1) {
				assert.Equal(t, test.want, entries[0].input.Name)
			}
		})
	}
}

func assertBatchIssues(t *testing.T, issues []batchInputError, entries []int, messages []string) {
	t.Helper()
	if !assert.Len(t, issues, len(entries)) {
		return
	}
	for index, entry := range entries {
		assert.Equal(t, entry, issues[index].entry)
		assert.Contains(t, issues[index].message, messages[index])
	}
}
