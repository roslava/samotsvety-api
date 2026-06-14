package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/repository"
)

func NewRouter(repo repository.MineralRepository) *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORS()) // ← Используем существующий middleware

	mineralHandler := NewMineralHandler(repo)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Минералы
		minerals := v1.Group("/minerals")
		{
			minerals.GET("", mineralHandler.ListMinerals)
			minerals.GET("/:slug", mineralHandler.GetMineral)
			minerals.POST("", mineralHandler.CreateMineral)
			minerals.PUT("/:slug", mineralHandler.UpdateMineral)
			minerals.DELETE("/:slug", mineralHandler.DeleteMineral)
		}

		// Поиск
		v1.GET("/search", mineralHandler.SearchMinerals)

		// Фильтры
		v1.GET("/filters", mineralHandler.GetFilters)
	}

	// Healthcheck
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return router
}
