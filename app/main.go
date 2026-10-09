package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/JorgeIba/Hurion/app/bootstrap"
	"github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/infrastructure/storage"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/script"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/workers"
	"github.com/bwmarrin/discordgo"
)

func main() {
	logger.Init()

	// 1. Parse CLI flags globally
	seedFlag := flag.Bool("seed", false, "Bootstrap the database from config.yaml")
	mockDiscordFlag := flag.Bool("mock-discord", false, "Use local mock providers instead of real Discord APIs")
	scriptFlag := flag.String("script", "", "Path to JSONL script file to run")
	exporterFlag := flag.Bool("exporter", false, "Enable automatic background scoreboard JSON exporting")
	flag.Parse()

	// 2. Shared Setup (Config, DB, Services)
	cfg, err := bootstrap.LoadConfig(bootstrap.DefaultConfigPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/db.sqlite"
	}

	db, err := data.NewDB(&data.DBConfig{
		DBPath: dbPath,
		DBType: data.Prod,
	})
	if err != nil {
		log.Fatalf("Error creating the DB: %v", err)
	}
	defer db.CloseDB()

	svcs := services.NewServices(db)
	token := os.Getenv("DISCORD_BOT_TOKEN")

	// Start background worker only if --exporter flag is set
	if *exporterFlag {
		const defaultExportsDir = "./data/exports"
		storeProvider := storage.NewStorageFromEnv(defaultExportsDir)
		exporterWorker := workers.NewScoreboardExporterWorker(svcs.Class, svcs.Score, storeProvider)
		exporterWorker.Start()
		defer exporterWorker.Stop()

		// Compile initial scoreboards on startup
		exporterWorker.Trigger()
		data.RegisterUpdateHookCallback(func(op int, dbName string, tableName string, rowid int64) {
			exporterWorker.Trigger()
		})
	}

	// 3. Hydrate the RunConfig
	runCfg, err := bootstrap.NewRunConfig(*mockDiscordFlag, *scriptFlag)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	logger.Log.Infof("Booting Discord Bot in mode: %s", runCfg.Mode)

	// 4. Optional Seeding
	if *seedFlag {
		reconciler := bootstrap.NewReconciler(svcs.Class)
		if err := reconciler.Reconcile(cfg); err != nil {
			log.Fatalf("Reconciliation failed: %v", err)
		}
	}

	// 5. Determine Provider (Offline Mock vs Online Real)
	var session *discordgo.Session
	var provider providers.DiscordProvider

	if runCfg.UsesMockProvider() {
		provider = script.NewFakeDiscordProvider(cfg.AdminRoleID, cfg.MentorRoleID)
	} else {
		if token == "" {
			log.Fatalf("DISCORD_BOT_TOKEN environment variable is required when not running offline")
		}
		session, err = discordgo.New("Bot " + token)
		if err != nil {
			log.Fatalf("Failed to create discord session: %v", err)
		}
		provider = providers.NewProviderDiscord(providers.NewDiscordGoSessionAdapter(session), cfg.AdminRoleID, cfg.MentorRoleID)
	}

	// 6. Determine Message Sender (Console vs Discord)
	var sender dispatcher.MessageSender
	if runCfg.UsesConsoleSender() {
		sender = dispatcher.NewConsoleMessageSender()
	} else {
		realSender := dispatcher.NewDispatcher(session, 5)
		realSender.Start()
		defer realSender.Stop()
		sender = realSender
	}

	// 7. Assemble the Bot
	var responder dispatcher.InteractionResponder
	if session != nil {
		responder = dispatcher.NewDiscordInteractionResponder(session)
	}
	botCfg := bot.BotConfig{
		TestMode:         cfg.TestMode,
		WhitelistUserIDs: cfg.WhitelistUserIDs,
		WhitelistRoleIDs: cfg.WhitelistRoleIDs,
		GuildID:          cfg.GuildID,
	}
	problemInteractionController := mentor.NewProblemInteractionController(svcs, provider, responder)
	acInteractionController := mentor.NewACInteractionController(svcs, provider, provider, responder)
	appBot := bot.NewBot(svcs, provider, sender, responder, botCfg)

	// 8. Execute Application Behavior
	switch runCfg.Mode {
	case bootstrap.ModeMockScript, bootstrap.ModeOnlineScript:
		guildID := cfg.GuildID
		if runCfg.Mode == bootstrap.ModeMockScript && guildID == "" {
			guildID = "guild_1"
		}
		if runCfg.Mode == bootstrap.ModeOnlineScript && guildID == "" {
			log.Fatalf("Configuration error: 'guild_id' must be configured in config.yaml for Online Script Mode.")
		}
		runner := script.NewRunner(appBot, provider, guildID)
		if err := runner.RunScript(runCfg.ScriptPath); err != nil {
			log.Fatalf("Script runner execution failed: %v", err)
		}
	case bootstrap.ModeOnlineDaemon:
		guildID := cfg.GuildID
		if guildID == "" {
			log.Fatalf("Configuration error: 'guild_id' must be configured to register guild slash commands.")
		}

		applicationUser, err := session.User("@me")
		if err != nil {
			log.Fatalf("Failed to identify Discord application before command registration: %v", err)
		}
		if applicationUser == nil || strings.TrimSpace(applicationUser.ID) == "" {
			log.Fatalf("Failed to identify Discord application before command registration: bot user ID is empty")
		}
		// Register the owned slash commands for the guild
		// It needs the associated application user ID to register the command correctly in the guild
		if err := bot.UpsertProblemSlashCommand(session, applicationUser.ID, guildID); err != nil {
			log.Fatalf("Failed to register guild slash commands: %v", err)
		}
		if err := bot.UpsertACSlashCommand(session, applicationUser.ID, guildID); err != nil {
			log.Fatalf("Failed to register guild slash commands: %v", err)
		}
		logger.Log.Infof("Upserted guild slash commands /problem and /ac in server %s", guildID)

		appBot.ConfigureSession(session)
		appBot.RegisterHandlers(session, problemInteractionController, acInteractionController)

		if err := session.Open(); err != nil {
			log.Fatalf("Error opening Discord session: %v", err)
		}
		defer session.Close()

		// Protect session shutdown
		defer func() {
			if r := recover(); r != nil {
				log.Printf("CRITICAL: Panic recovered in serve: %v", r)
			}
		}()

		logger.Log.Info("Bot daemon running. CTRL+C to exit.")

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		logger.Log.Info("Shutting down the bot...")
		logger.Log.Info("Bot shut down successfully.")
	}
}
