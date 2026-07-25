package main

import (
	"context"
	"log/slog"

	"github.com/joho/godotenv"

	"github.com/roslava/samotsvety-api/internal/config"
	"github.com/roslava/samotsvety-api/internal/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using system environment")
	}

	cfg := config.LoadConfig()

	db, err := config.ConnectDB(context.Background(), cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return
	}
	defer db.Close()

	slog.Info("Starting seeding process...")

	if err := repository.SeedMinerals(db, "seeds/minerals"); err != nil {
		slog.Error("Seeding minerals failed", "error", err)
		return
	}

	if err := repository.SeedPosts(db, "seeds/posts"); err != nil {
		slog.Error("Seeding posts failed", "error", err)
		return
	}

	slog.Info("✅ Seeding completed successfully!")
}
