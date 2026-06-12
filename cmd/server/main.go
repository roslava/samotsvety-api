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
)

func main() {
// Загружаем .env
if err := godotenv.Load(); err != nil {
slog.Warn("No .env file found, using system environment")
}

// Конфигурация
port := os.Getenv("APP_PORT")
if port == "" {
port = "8080"
}

// Настройка Gin
gin.SetMode(gin.DebugMode)
router := gin.Default()

// Healthcheck
router.GET("/health", func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{
"status":  "ok",
"service": "samotsvety-api",
"version": "0.1.0",
})
})

// Запуск сервера
srv := &http.Server{
Addr:    ":" + port,
Handler: router,
}

// Graceful shutdown
go func() {
slog.Info("Server starting", "port", port)
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
slog.Error("Server failed", "error", err)
}
}()

// Ожидание сигнала завершения
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

slog.Info("Shutting down server...")

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
slog.Error("Server forced to shutdown", "error", err)
}

slog.Info("Server exited")
}
