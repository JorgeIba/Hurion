package discord

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/bwmarrin/discordgo"
)

const maxFieldLength = 1024 - 1 // Discord limit

// FormatProblemsAsEmbeds transforms a list of problems grouped by topic into a slice of Discord embeds.
// It handles splitting into multiple embeds if the content exceeds Discord's field limits.
func FormatProblemsAsEmbeds(classCode string, groupedProblems []services.ProblemsByTopic) []*discordgo.MessageEmbed {
	var embeds []*discordgo.MessageEmbed
	title := fmt.Sprintf("List of problems for class %s", classCode)

	// Build values for the 3 columns
	idColumn := ""
	difficultyColumn := ""
	nameColumn := ""

	// Check if any column would exceed Discord's limit
	wouldExceedLimit := func(addID, addDiff, addName string) bool {
		return len(idColumn+addID) > maxFieldLength ||
			len(difficultyColumn+addDiff) > maxFieldLength ||
			len(nameColumn+addName) > maxFieldLength
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
	addProblem := func(problemID int64, difficultyEmoji, problemName, problemURL string) {
		idColumn += fmt.Sprintf("\n%d", problemID)
		difficultyColumn += fmt.Sprintf("\n%s", difficultyEmoji)
		nameColumn += fmt.Sprintf("\n[%s](%s)", problemName, problemURL)
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
			difficultyEmoji := GetDifficultyEmoji(problem.GetDifficulty())

			// Build what we want to add for problem
			pAddID := fmt.Sprintf("\n%d", problem.ID)
			pAddDiff := fmt.Sprintf("\n%s", difficultyEmoji)
			pAddName := fmt.Sprintf("\n[%s](%s)", problem.Name, problem.URL)

			// Check if adding this problem would exceed limit
			if wouldExceedLimit(pAddID, pAddDiff, pAddName) {
				flushPage()
				addTopicHeader(group.Topic, false)
			}
			addProblem(problem.ID, difficultyEmoji, problem.Name, problem.URL)
		}
	}

	// Flush the last page if there are remaining problems
	flushPage()

	return embeds
}

// GetDifficultyEmoji returns the emoji representation of a problem's difficulty.
func GetDifficultyEmoji(difficulty data.Difficulty) string {
	switch difficulty {
	case data.DifficultyEasy:
		return "🟩"
	case data.DifficultyMedium:
		return "🟨"
	case data.DifficultyHard:
		return "🟥"
	default:
		return "⬜"
	}
}
