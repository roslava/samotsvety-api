package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/middleware"
	"github.com/roslava/samotsvety-api/internal/repository"
)

func NewRouter(mineralRepo repository.MineralRepository, postRepo repository.PostRepository) *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	mineralHandler := NewMineralHandler(mineralRepo)
	postHandler := NewPostHandler(postRepo)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// === Минералы ===
		minerals := v1.Group("/minerals")
		{
			// Публичные методы (чтение)
			minerals.GET("", mineralHandler.ListMinerals)
			minerals.GET("/:slug", mineralHandler.GetMineral)

			// === ЗАЩИЩЁННЫЕ АДМИНСКИЕ МЕТОДЫ ===
			admin := minerals.Group("")
			admin.Use(middleware.APIKeyAuth())
			{
				admin.POST("", mineralHandler.CreateMineral)
				admin.PUT("/:slug", mineralHandler.UpdateMineral)
				admin.DELETE("/:slug", mineralHandler.DeleteMineral)
			}
		}

		// === Посты / Статьи ===
		posts := v1.Group("/posts")
		{
			// Публичные методы (чтение)
			posts.GET("", postHandler.ListPosts)
			posts.GET("/:slug", postHandler.GetPost)

			// === ЗАЩИЩЁННЫЕ АДМИНСКИЕ МЕТОДЫ ===
			admin := posts.Group("")
			admin.Use(middleware.APIKeyAuth())
			{
				admin.POST("", postHandler.CreatePost)
				admin.PUT("/:slug", postHandler.UpdatePost)
				admin.DELETE("/:slug", postHandler.DeletePost)
			}
		}

		// Поиск и фильтры — публичные
		v1.GET("/search", mineralHandler.SearchMinerals)
		v1.GET("/filters", mineralHandler.GetFilters)
	}

	// Healthcheck
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return router
}
