package script

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

func TestRegistries(t *testing.T) {
	t.Parallel()

	t.Run("UserRegistry - Register and GetNameByID", func(t *testing.T) {
		t.Parallel()
		reg := NewUserRegistry()
		reg.Register("student_maria", "4000")

		name, ok := reg.GetNameByID("4000")
		assert.True(t, ok)
		assert.Equal(t, "student_maria", name)

		_, ok = reg.GetNameByID("9999")
		assert.False(t, ok)
	})

	t.Run("UserRegistry - TranslateExecutor", func(t *testing.T) {
		t.Parallel()
		reg := NewUserRegistry()
		reg.Register("mentor_pedro", "2000")

		assert.Equal(t, "2000", reg.TranslateExecutor("mentor_pedro"))
		assert.Equal(t, "unregistered_user", reg.TranslateExecutor("unregistered_user"))
	})

	t.Run("UserRegistry - TranslateCommand with Space-Separated Tokens", func(t *testing.T) {
		t.Parallel()
		reg := NewUserRegistry()
		reg.Register("bob", "100")
		reg.Register("bobby", "200")

		cmd := "!ac @bobby and @bob"
		translated := reg.TranslateCommand(cmd)

		assert.Equal(t, "!ac <@200> and <@100>", translated)
	})

	t.Run("ChannelRegistry - Register and TranslateChannel", func(t *testing.T) {
		t.Parallel()
		reg := NewChannelRegistry()
		reg.Register("basicos", "123456")

		assert.Equal(t, "123456", reg.TranslateChannel("#basicos"))
		assert.Equal(t, "#unregistered_chan", reg.TranslateChannel("#unregistered_chan"))
		assert.Equal(t, "123456", reg.TranslateChannel("123456")) // No prefix, returns as-is
	})

	t.Run("ChannelRegistry - TranslateCommand", func(t *testing.T) {
		t.Parallel()
		reg := NewChannelRegistry()
		reg.Register("basicos", "123456")

		cmd := "!link_rooms advanced_go #basicos"
		translated := reg.TranslateCommand(cmd)

		assert.Equal(t, "!link_rooms advanced_go 123456", translated)
	})
}

// mockMessageSender captures printed/sent messages for testing
type mockMessageSender struct {
	sent []dispatcher.MessageResponse
}

func (m *mockMessageSender) Send(msg dispatcher.MessageResponse) {
	m.sent = append(m.sent, msg)
}

func TestRunScript_HappyPath(t *testing.T) {
	t.Parallel()

	// 1. Create a temporary script file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.jsonl")

	scriptContent := `# This is a comment
{"type": "setup_alias", "alias": "jorge", "user_id": "1000"}
{"type": "setup_alias", "alias": "luis", "user_id": "2000"}
{"type": "setup_roles", "user_id": "jorge", "roles": ["admin_role"]}
{"type": "setup_channel", "alias": "general", "channel_id": "c1"}
{"type": "command", "command": "!help", "executor_id": "jorge", "channel_id": "#general"}
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	require.NoError(t, err)

	// 2. Initialize in-memory database and services
	db, err := data.NewDB(&data.DBConfig{
		DBPath: ":memory:",
		DBType: data.Test,
	})
	require.NoError(t, err)
	defer db.CloseDB()

	fakeProvider := NewFakeDiscordProvider("admin_role", "mentor_role")
	fakeSender := &mockMessageSender{}
	svcs := services.NewServices(db)

	appBot := bot.NewBot(svcs, fakeProvider, fakeSender, bot.BotConfig{})
	runner := NewRunner(appBot, fakeProvider)

	// 3. Act - execute script
	err = runner.RunScript(scriptPath)
	require.NoError(t, err)

	// 4. Assert
	require.Len(t, fakeSender.sent, 1)
	assert.Contains(t, fakeSender.sent[0].Embed.Title, "Hurion - Available Commands")
}

func TestRunScript_ParseFailure(t *testing.T) {
	t.Parallel()

	// 1. Create a temporary script file with malformed JSON
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "invalid_script.jsonl")
	scriptContent := `{"type": "command", "command": "!help", "executor_id": "missing_alias"}
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	require.NoError(t, err)

	// 2. Initialize in-memory database and services
	db, err := data.NewDB(&data.DBConfig{
		DBPath: ":memory:",
		DBType: data.Test,
	})
	require.NoError(t, err)
	defer db.CloseDB()

	fakeProvider := NewFakeDiscordProvider("admin_role", "mentor_role")
	fakeSender := &mockMessageSender{}
	svcs := services.NewServices(db)
	appBot := bot.NewBot(svcs, fakeProvider, fakeSender, bot.BotConfig{})
	runner := NewRunner(appBot, fakeProvider)

	// 3. Act
	err = runner.RunScript(scriptPath)

	// 4. Assert - should fail fast with a descriptive error because "missing_alias" has no registered alias and 'executor_name' is missing
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor ID 'missing_alias' is not registered as an alias and 'executor_name' is missing")
}
