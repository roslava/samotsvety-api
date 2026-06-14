package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/roslava/samotsvety-api/internal/config"
	"github.com/roslava/samotsvety-api/internal/domain"
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
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("Starting seeding process...")

	if err := seedMinerals(db); err != nil {
		slog.Error("Seeding failed", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Seeding completed successfully!")
}

func seedMinerals(db *repository.DB) error { // ← corrected if needed
	seedDir := "seeds/minerals"
	files, err := filepath.Glob(filepath.Join(seedDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to read seed directory: %w", err)
	}

	if len(files) == 0 {
		slog.Warn("No JSON files found in seeds/minerals/")
		return nil
	}

	slog.Info("Found seed files", "count", len(files))

	repo := repository.NewPostgresMineralRepository(db)

	for _, file := range files {
		slug := strings.TrimSuffix(filepath.Base(file), ".json")

		data, err := os.ReadFile(file)
		if err != nil {
			slog.Error("Failed to read file", "file", file, "error", err)
			continue
		}

		var mineral domain.Mineral
		if err := json.Unmarshal(data, &mineral); err != nil {
			slog.Error("Failed to unmarshal mineral", "file", file, "error", err)
			continue
		}

		// Create (skip if already exists)
		err = repo.Create(context.Background(), &mineral)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") ||
				strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "conflict") {
				slog.Info("⏭️  Already exists, skipped", "slug", slug)
				continue
			}
			slog.Error("Failed to seed mineral", "slug", slug, "error", err)
			continue
		}

		slog.Info("✅ Seeded", "slug", slug)
	}

	return nil
}
