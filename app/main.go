package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/bootstrap"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/bot"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/bwmarrin/discordgo"
)

func main() {
	logger.Init()

	// 1. Parse CLI flags
	seedFlag := flag.Bool("seed", false, "Bootstrap the database from config.yaml")
	flag.Parse()

	// 2. Load Configuration (Source of Truth)
	cfg, err := bootstrap.LoadConfig(bootstrap.DefaultConfigPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 3. Validate required environment variables
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatalf("DISCORD_BOT_TOKEN environment variable is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/db.sqlite"
	}

	// 4. Initialize database and services
	db, err := data.NewDB(
		&data.DBConfig{
			DBPath: dbPath,
			DBType: data.Prod,
		},
	)

	if err != nil {
		log.Fatalf("Error creating the DB: %v", err)
	}
	defer db.CloseDB()

	// Create shared services
	svcs := services.NewServices(db)

	// 5. Configuration Conciliation
	if *seedFlag {
		reconciler := bootstrap.NewReconciler(svcs.Class)
		if err := reconciler.Reconcile(cfg); err != nil {
			log.Fatalf("Failed to reconcile configuration: %v", err)
		}
	}

	stopBot, err := startBot(token, svcs, cfg.AdminRoleID, cfg.MentorRoleID)
	if err != nil {
		log.Fatalf("Error running the bot: %v", err)
	}
	defer stopBot()

	// Protect session open/shutdown path with a recover
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL: Panic recovered in main: %v", r)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	logger.Log.Info("Shutting down the bot...")
	logger.Log.Info("Bot shut down successfully.")
}

func startBot(token string, svcs *services.Services, adminRoleID, mentorRoleID string) (func(), error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	const bufferSize = 5
	msgDispatcher := dispatcher.NewDispatcher(dg, bufferSize)
	discordProvider := providers.NewProviderDiscord(
		providers.NewDiscordGoSessionAdapter(dg),
		adminRoleID,
		mentorRoleID,
	)

	appBot := bot.NewBot(svcs, discordProvider, msgDispatcher)
	appBot.ConfigureSession(dg)
	appBot.RegisterHandlers(dg)

	msgDispatcher.Start()
	if err := dg.Open(); err != nil {
		msgDispatcher.Stop()
		dg.Close()
		return nil, err
	}

	logger.Log.Info("Bot is running, CTRL+C to exit.")

	stop := func() {
		msgDispatcher.Stop()
		dg.Close()
	}
	return stop, nil
}
