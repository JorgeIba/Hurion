package discord_test

import (
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestFormatCreateProblemErrorPreservesDetailsWithinLimitAndFallsBackAboveIt(t *testing.T) {
	t.Parallel()
	fallback := "Could not add the problem. Please try /problem add again."
	for _, test := range []struct {
		name         string
		content      string
		want         string
		recoverDraft bool
	}{
		{name: "short error keeps correction details", content: "Please correct the values: URL must be an absolute HTTP or HTTPS URL. Try /problem add class:basicos_2 again.", want: "Please correct the values: URL must be an absolute HTTP or HTTPS URL. Try /problem add class:basicos_2 again."},
		{name: "exactly 2000 ASCII units are preserved", content: strings.Repeat("a", 2000), want: strings.Repeat("a", 2000)},
		{name: "2001 ASCII units use the selected class in the fallback", content: strings.Repeat("a", 2001), want: fallback},
		{name: "exactly 2000 UTF-16 units are preserved despite larger UTF-8 length", content: strings.Repeat("🚀", 1000), want: strings.Repeat("🚀", 1000)},
		{name: "2001 UTF-16 units use the fallback", content: strings.Repeat("🚀", 1000) + "a", want: fallback},
		{name: "recoverable error gets retry instruction", content: "Invalid URL", recoverDraft: true, want: "Invalid URL. Try /problem add class:basicos_2 again."},
		{name: "long recoverable error keeps selected class", content: strings.Repeat("a", 2001), recoverDraft: true, want: "Could not add the problem. Please check the values and try /problem add class:basicos_2 again."},
		{name: "retry instruction counts toward limit", content: strings.Repeat("a", 2000), recoverDraft: true, want: "Could not add the problem. Please check the values and try /problem add class:basicos_2 again."},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			content := discord.FormatCreateProblemError(test.content, "basicos_2", test.recoverDraft)
			assert.Equal(t, test.want, content)
			assert.LessOrEqual(t, len(utf16.Encode([]rune(content))), 2000)
		})
	}
}

func TestFormatCreatedProblemReceiptPreservesIdentityWithinDiscordMessageLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		assertFunc func(*testing.T, string)
	}{
		{
			name:  "normal text is shown",
			input: "Two Sum",
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".\nName: \"Two Sum\"", content)
			},
		},
		{
			name:  "newlines stay inside their field",
			input: "Two\nSum",
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".\nName: \"Two\\nSum\"", content)
			},
		},
		{
			name:  "unicode name within the UTF-16 limit is shown even when its UTF-8 bytes exceed 2000",
			input: strings.Repeat("🚀", 600),
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".\nName: \""+strings.Repeat("🚀", 600)+"\"", content)
			},
		},
		{
			name:  "large unicode name is omitted from the receipt",
			input: strings.Repeat("🚀", 1000),
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".", content)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := discord.FormatCreatedProblemReceipt(42, "basicos_1", tt.input)
			tt.assertFunc(t, content)
			assert.LessOrEqual(t, len(utf16.Encode([]rune(content))), 2000)
		})
	}
}

func TestEditProblemReceiptDisplayLimits(t *testing.T) {
	receipt := discord.FormatUpdatedProblemReceipt(123, strings.Repeat("🚀", 2000), " Raw 🚀 ")
	assert.LessOrEqual(t, countUTF16ForTest(receipt), 2000)
	assert.Contains(t, receipt, "123")
	assert.LessOrEqual(t, countUTF16ForTest(discord.FormatEditProblemError(strings.Repeat("🚀", 2000), 123, "basicos", true)), 2000)
}

func TestEditProblemErrorPreservesDestinationWhenShortened(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		content   string
		classCode string
		want      string
	}{
		{"short error", "Invalid URL.", "avanzados", "Invalid URL. Try /problem edit problem:123 class:avanzados again."},
		{"long error", strings.Repeat("🚀", 2000), "avanzados", "class:avanzados"},
		{"class cannot fit", strings.Repeat("🚀", 2000), strings.Repeat("a", 2000), "same destination class code"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			content := discord.FormatEditProblemError(test.content, 123, test.classCode, true)
			assert.Contains(t, content, test.want)
			assert.LessOrEqual(t, countUTF16ForTest(content), 2000)
		})
	}
}

func TestEditProblemErrorWithoutRecoverableDraft(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Internal error.", discord.FormatEditProblemError("Internal error.", 123, "avanzados", false))
	content := discord.FormatEditProblemError(strings.Repeat("🚀", 2000), 123, "avanzados", false)
	assert.Equal(t, "Could not update problem `123`. Please try /problem edit again.", content)
	assert.LessOrEqual(t, countUTF16ForTest(content), 2000)
}

func TestJSONParseReceiptFitsDiscordLimit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, message string
		count         int
	}{
		{name: "many entries", message: "unknown field", count: 100},
		{name: "large Unicode error", message: strings.Repeat("🚀", 2000), count: 1},
		{name: "many large errors", message: strings.Repeat("🚀", 2000), count: 100},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			issues := make([]discord.JSONInputIssue, test.count)
			for index := range issues {
				issues[index] = discord.JSONInputIssue{Entry: index + 1, Message: test.message}
			}
			result := discord.FormatJSONParseResult(0, issues)
			assert.Contains(t, result, "Entry 1")
			assert.LessOrEqual(t, len(utf16.Encode([]rune(result))), 2000)
			if test.count > 1 {
				assert.Contains(t, result, "more issue(s)")
			}
		})
	}
}

func countUTF16ForTest(value string) int { return len(utf16.Encode([]rune(value))) }
