package bot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
)

const (
	InstructionTypeSetupAlias   = "setup_alias"
	InstructionTypeSetupRoles   = "setup_roles"
	InstructionTypeSetupChannel = "setup_channel"
	InstructionTypeCommand      = "command"
)

type ScriptInstruction struct {
	Type         string   `json:"type"` // "setup_alias", "setup_roles", "setup_channel", or "command"
	Alias        string   `json:"alias"`
	UserID       string   `json:"user_id"`
	Command      string   `json:"command"`
	ExecutorID   string   `json:"executor_id"`
	ExecutorName string   `json:"executor_name"`
	ChannelID    string   `json:"channel_id"`
	Roles        []string `json:"roles"`
}

type UserRegistry struct {
	aliases map[string]string
	names   map[string]string
}

func NewUserRegistry() *UserRegistry {
	return &UserRegistry{
		aliases: make(map[string]string),
		names:   make(map[string]string),
	}
}

func (ur *UserRegistry) Register(alias, id string) {
	ur.aliases[alias] = id
	ur.names[id] = alias
}

// TODO(Optimization): If script execution performance becomes an issue (e.g. processing large files
// with thousands of commands), we can optimize this by doing a single pre-processing pass over the
// JSONL file. We would resolve all @aliases in one pass and write the fully-translated commands
// to a temporary JSONL file, then execute that file directly to avoid loop replacements on the fly.
func (ur *UserRegistry) TranslateCommand(cmd string) string {
	words := strings.Fields(cmd)
	for i, word := range words {
		if alias, ok := strings.CutPrefix(word, "@"); ok {
			if id, exists := ur.aliases[alias]; exists {
				words[i] = "<@" + id + ">"
			}
		}
	}
	return strings.Join(words, " ")
}

func (ur *UserRegistry) TranslateExecutor(alias string) string {
	if id, ok := ur.aliases[alias]; ok {
		return id
	}
	return alias
}

func (ur *UserRegistry) GetNameByID(id string) (string, bool) {
	name, ok := ur.names[id]
	return name, ok
}

type ChannelRegistry struct {
	channels map[string]string
}

func NewChannelRegistry() *ChannelRegistry {
	return &ChannelRegistry{
		channels: make(map[string]string),
	}
}

func (cr *ChannelRegistry) Register(alias, id string) {
	cr.channels[alias] = id
}

func (cr *ChannelRegistry) TranslateChannel(alias string) string {
	if aliasClean, ok := strings.CutPrefix(alias, "#"); ok {
		if id, exists := cr.channels[aliasClean]; exists {
			return id
		}
	}
	return alias
}

func (cr *ChannelRegistry) TranslateCommand(cmd string) string {
	words := strings.Fields(cmd)
	for i, word := range words {
		if alias, ok := strings.CutPrefix(word, "#"); ok {
			if id, exists := cr.channels[alias]; exists {
				words[i] = id
			}
		}
	}
	return strings.Join(words, " ")
}

type MockRoleSetter interface {
	SetUserRoles(userID string, roles []string)
}

// RunScript reads commands from a JSONL file and executes them sequentially.
func (bot *Bot) RunScript(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open script file %s: %w", filepath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	userAliases := NewUserRegistry()
	channelRegistry := NewChannelRegistry()
	messageIDCounter := 1

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		var inst ScriptInstruction
		if err := json.Unmarshal([]byte(line), &inst); err != nil {
			return fmt.Errorf("line %d: failed to parse JSON: %w. Line: %s", lineNumber, err, line)
		}

		switch inst.Type {
		case InstructionTypeSetupAlias:
			if inst.Alias == "" || inst.UserID == "" {
				return fmt.Errorf("line %d: setup_alias requires non-empty alias and user_id", lineNumber)
			}

			userAliases.Register(inst.Alias, inst.UserID)
			logger.Log.Infof("Registered script alias: %s -> ID %s", inst.Alias, inst.UserID)

		case InstructionTypeSetupRoles:
			if inst.UserID == "" {
				return fmt.Errorf("line %d: setup_roles requires non-empty user_id", lineNumber)
			}

			realUserID := userAliases.TranslateExecutor(inst.UserID)
			if roleSetter, ok := bot.discordProvider.(MockRoleSetter); ok {
				roleSetter.SetUserRoles(realUserID, inst.Roles)
				logger.Log.Infof("Registered mock roles for %s: %v", realUserID, inst.Roles)
			}

		case InstructionTypeSetupChannel:
			if inst.Alias == "" || inst.ChannelID == "" {
				return fmt.Errorf("line %d: setup_channel requires non-empty alias and channel_id", lineNumber)
			}

			channelRegistry.Register(inst.Alias, inst.ChannelID)
			logger.Log.Infof("Registered script channel alias: #%s -> ID %s", inst.Alias, inst.ChannelID)

		case InstructionTypeCommand:
			// Translate executor and command text
			inst.ExecutorID = userAliases.TranslateExecutor(inst.ExecutorID)
			inst.Command = userAliases.TranslateCommand(inst.Command)
			inst.Command = channelRegistry.TranslateCommand(inst.Command)

			// Translate channel ID alias (e.g. "#basicos" -> "123456")
			inst.ChannelID = channelRegistry.TranslateChannel(inst.ChannelID)

			// Try to find the alias name if ExecutorName is missing
			if inst.ExecutorName == "" {
				name, ok := userAliases.GetNameByID(inst.ExecutorID)
				if !ok {
					return fmt.Errorf("line %d: executor ID '%s' is not registered as an alias and 'executor_name' is missing", lineNumber, inst.ExecutorID)
				}
				inst.ExecutorName = name
			}

			bot.executeScriptCommand(inst, messageIDCounter)
			messageIDCounter++

		default:
			return fmt.Errorf("line %d: unknown instruction type '%s'", lineNumber, inst.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading script file: %w", err)
	}

	logger.Log.Info("Script execution completed.")
	return nil
}

// executeScriptCommand builds a mocked context and routes the command.
func (bot *Bot) executeScriptCommand(cmdData ScriptInstruction, msgID int) {
	if cmdData.Command == "" {
		return
	}

	// Create a mock discordgo Message (always default GuildID to "guild_1")
	author := &discordgo.User{
		ID:       cmdData.ExecutorID,
		Username: cmdData.ExecutorName,
	}

	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        fmt.Sprintf("mock_msg_%d", msgID),
			ChannelID: cmdData.ChannelID,
			GuildID:   "guild_1",
			Content:   cmdData.Command,
			Author:    author,
			Member: &discordgo.Member{
				User: author,
				Nick: cmdData.ExecutorName,
			},
		},
	}

	// Route it directly through the primary bot pipeline (tests the real handlers)
	bot.HandleMessageEvent(mockMessage)
}
