package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/roslava/samotsvety-api/internal/config"
	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/handler"
	"github.com/roslava/samotsvety-api/internal/repository"
	"github.com/roslava/samotsvety-api/internal/storage"
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

	var verifier handler.SlideMediaVerifier
	if mediaStorage, err := storage.NewYandexS3Storage(context.Background(), cfg.Storage); err == nil {
		if articleStorage, ok := mediaStorage.(storage.ArticleMediaStorage); ok {
			verifier = handler.NewArticleMediaVerifier(articleStorage)
		}
	} else {
		slog.Warn("Article media verifier unavailable; published post seeds will be rejected")
	}
	report, err := repository.SeedPosts(context.Background(), db, "seeds/posts", func(ctx context.Context, data []byte) (*domain.Post, error) {
		return handler.PrepareArticleV2Seed(ctx, data, verifier)
	})
	slog.Info("Post seed summary", "inserted", report.Inserted, "skipped", report.Skipped, "invalid", report.Invalid)
	if err != nil {
		slog.Error("Seeding posts failed", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Seeding completed successfully!")
}
