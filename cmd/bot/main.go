package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"github.com/VolkHackVH/detective-bot.git/internal/config"
	"github.com/VolkHackVH/detective-bot.git/internal/database"
	"github.com/VolkHackVH/detective-bot.git/internal/evidence"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	migrationContext, cancelMigration := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelMigration()

	if err := database.Migrate(migrationContext, db); err != nil {
		log.Fatalf("Error applying database schema: %v", err)
	}

	evidenceRepository := evidence.NewRepository(db)

	evidenceHandler := evidence.NewHandler(
		evidenceRepository,
		cfg.Evidence.IntakeChannelID,
		cfg.Evidence.ReviewChannelID,
		cfg.Evidence.ReviewerRoleIDs,
	)

	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	session.Identify.Intents =
		discordgo.IntentsGuilds |
			discordgo.IntentsGuildMessages |
			discordgo.IntentsMessageContent

	session.AddHandler(onReady)
	session.AddHandler(evidenceHandler.OnMessageCreate)
	session.AddHandler(evidenceHandler.OnInteraction)

	if err := session.Open(); err != nil {
		log.Fatalf("Error connecting Discord: %v", err)
	}
	defer session.Close()

	slog.Info(
		"Bot is running",
		"guild_id", cfg.Discord.GuildID,
		"intake_channel_id", cfg.Evidence.IntakeChannelID,
		"review_channel_id", cfg.Evidence.ReviewChannelID,
	)

	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-stop

	slog.Info("Bot is shutting down")
}

func onReady(
	_ *discordgo.Session,
	event *discordgo.Ready,
) {
	slog.Info(
		"Discord authorization successful",
		"user_id", event.User.ID,
		"username", event.User.Username,
	)
}
