package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/roslava/samotsvety-api/internal/config"
	"github.com/roslava/samotsvety-api/internal/handler"
	"github.com/roslava/samotsvety-api/internal/repository"
)

func main() {
	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using system environment")
	}

	// Конфигурация
	cfg := config.LoadConfig()

	// Подключение к PostgreSQL
	db, err := config.ConnectDB(context.Background(), cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Создаём репозиторий
	mineralRepo := repository.NewPostgresMineralRepository(db)

	// Настройка Gin
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Создаём роутер через handler
	router := handler.NewRouter(mineralRepo)

	// HTTP-сервер
	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// Запуск сервера
	go func() {
		slog.Info("Server starting", "port", cfg.AppPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}
