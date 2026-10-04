package discord

import (
	"fmt"
	"unicode/utf16"

	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

const (
	maxFieldLength = 1024 // Discord's strict limit for field values

	// truncationSafetyMargin is a buffer subtracted from maxFieldLength during per-row truncation.
	// This accounts for extra characters like topic headers, newlines, and Discord's internal overhead
	// to ensure that even with multiple rows, the field value stays comfortably under the limit.
	truncationSafetyMargin = 20
)

// FormatProblemsAsEmbeds transforms a list of problems grouped by topic into a slice of Discord embeds.
// It handles splitting into multiple embeds if the content exceeds Discord's field limits.
func FormatProblemsAsEmbeds(classCode string, groupedProblems []presentation.ProblemsByTopic) []*discordgo.MessageEmbed {
	var embeds []*discordgo.MessageEmbed
	title := fmt.Sprintf("List of problems for class %s", classCode)

	// Build values for the 3 columns
	idColumn := ""
	difficultyColumn := ""
	nameColumn := ""

	// Check if any column would exceed Discord's limit
	wouldExceedLimit := func(addID, addDiff, addName string) bool {
		return utf16Count(idColumn+addID) > maxFieldLength ||
			utf16Count(difficultyColumn+addDiff) > maxFieldLength ||
			utf16Count(nameColumn+addName) > maxFieldLength
	}

	// Flush current page if not empty
	flushPage := func() {
		if idColumn != "" {
			embeds = append(embeds, &discordgo.MessageEmbed{
				Title: title,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "ID", Value: idColumn, Inline: true},
					{Name: "Difficulty", Value: difficultyColumn, Inline: true},
					{Name: "Name", Value: nameColumn, Inline: true},
				},
			})
			idColumn = ""
			difficultyColumn = ""
			nameColumn = ""
		}
	}

	// Add topic header to columns
	addTopicHeader := func(topic string, needsSeparator bool) {
		if needsSeparator {
			idColumn += "\n"
			difficultyColumn += "\n"
			nameColumn += "\n"
		}
		idColumn += fmt.Sprintf("**%s**", topic)
		difficultyColumn += "\u200b"
		nameColumn += "\u200b"
	}

	// Add problem data to columns
	addProblem := func(problemID string, difficultyEmoji string, problemName string) {
		idColumn += problemID
		difficultyColumn += difficultyEmoji
		nameColumn += problemName
	}

	for _, group := range groupedProblems {
		// Build what we want to add for topic header
		addID := fmt.Sprintf("**%s**", group.Topic)
		addDiff := "\u200b"
		addName := "\u200b"
		if idColumn != "" {
			addID = "\n" + addID
			addDiff = "\n" + addDiff
			addName = "\n" + addName
		}

		// Check if adding topic header would exceed limit
		if wouldExceedLimit(addID, addDiff, addName) {
			flushPage()
		}
		addTopicHeader(group.Topic, idColumn != "")

		// Add problems in this topic
		for _, problem := range group.Problems {
			difficultyEmoji := GetDifficultyEmoji(problem.Difficulty)

			// Build what we want to add for problem
			pAddID := fmt.Sprintf("\n%d", problem.ID)
			pAddDiff := fmt.Sprintf("\n%s", difficultyEmoji)
			pAddName := fmt.Sprintf("\n[%s](%s)", problem.Name, problem.URL)

			// Truncate to avoid exceeding max field length individually
			pAddName = truncateProblemName(problem.Name, pAddName)

			// Check if adding this problem would exceed limit
			if wouldExceedLimit(pAddID, pAddDiff, pAddName) {
				flushPage()
				addTopicHeader(group.Topic, false)
			}

			addProblem(pAddID, pAddDiff, pAddName)
		}
	}

	// Flush the last page if there are remaining problems
	flushPage()

	return embeds
}

// truncateProblemName ensures that the formatted problem row does not exceed the safe maximum field length.
// It prioritizes keeping the markdown link, but falls back to the plain name and finally a truncated name if needed.
func truncateProblemName(name, formattedRow string) string {
	safeMax := maxFieldLength - truncationSafetyMargin

	// Step 1: If the full link fits, we're good!
	if utf16Count(formattedRow) <= safeMax {
		return formattedRow
	}

	// Step 2: It's too long. Drop the URL and try with just the name.
	rowWithJustName := "\n" + name
	if utf16Count(rowWithJustName) <= safeMax {
		return rowWithJustName
	}

	// Step 3: Even the name alone is too long.
	// Account for "\n" (1) and "..." (3)
	maxNameChars := safeMax - 4
	if maxNameChars < 0 {
		maxNameChars = 0
	}

	runes := []rune(name)
	if len(runes) > maxNameChars {
		return "\n" + string(runes[:maxNameChars]) + "..."
	}

	return rowWithJustName
}

// GetDifficultyEmoji returns the emoji representation of a problem's difficulty.
func GetDifficultyEmoji(difficulty *string) string {
	if difficulty == nil {
		return "⬜"
	}
	switch *difficulty {
	case service_model.DifficultyEasy:
		return "🟩"
	case service_model.DifficultyMedium:
		return "🟨"
	case service_model.DifficultyHard:
		return "🟥"
	default:
		return "⬜"
	}
}

// utf16Count returns the number of UTF-16 code units in a string.
// Discord's character limits (like the 1024 char field limit) are measured in UTF-16 code units.
func utf16Count(s string) int {
	return len(utf16.Encode([]rune(s)))
}
