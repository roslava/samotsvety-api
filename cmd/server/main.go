package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/roslava/samotsvety-api/docs" // Swagger docs

	"github.com/roslava/samotsvety-api/internal/config"
	"github.com/roslava/samotsvety-api/internal/handler"
	"github.com/roslava/samotsvety-api/internal/repository"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using system environment")
	}

	// Конфигурация
	cfg := config.LoadConfig()

	// Логгер
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Samotsvety API...", "env", cfg.AppEnv, "port", cfg.AppPort)

	// Подключение к БД
	db, err := config.ConnectDB(context.Background(), cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Репозиторий
	mineralRepo := repository.NewPostgresMineralRepository(db)

	// Роутер
	router := handler.NewRouter(mineralRepo)

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	slog.Info("Swagger documentation available at http://localhost:" + cfg.AppPort + "/swagger/index.html")

	// HTTP сервер
	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// Запуск сервера
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
		}
	}()

	slog.Info("Server started", "port", cfg.AppPort)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}
