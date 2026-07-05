// internal/handler/post_handler.go
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/repository"
)

type PostHandler struct {
	repo repository.PostRepository
}

func NewPostHandler(repo repository.PostRepository) *PostHandler {
	return &PostHandler{repo: repo}
}

// ListPosts godoc
// @Summary      Получить список статей
// @Description  Список постов с фильтрацией и пагинацией
// @Tags         posts
// @Produce      json
// @Param        page       query  int     false  "Страница"
// @Param        limit      query  int     false  "Лимит"
// @Param        type       query  string  false  "Тип (blog, guide...)"
// @Param        tag        query  string  false  "Тег"
// @Param        gem_slug   query  string  false  "Связанный камень"
// @Param        published  query  bool    false  "Только опубликованные"
// @Success      200  {object}  ListResponse
// @Router       /api/v1/posts [get]
func (h *PostHandler) ListPosts(c *gin.Context) {
	var req ListPostsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	filters := repository.PostFilterParams{
		Page:        req.Page,
		Limit:       req.Limit,
		Type:        req.Type,
		Tag:         req.Tag,
		GemSlug:     req.GemSlug,
		IsPublished: req.Published,
		Lang:        c.DefaultQuery("lang", "ru"),
	}

	posts, total, err := h.repo.List(c.Request.Context(), filters)
	if err != nil {
		slog.Error("Failed to list posts", "error", err)
		RespondInternalError(c, "Failed to fetch posts")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  posts,
		"total": total,
		"page":  filters.Page,
		"limit": filters.Limit,
	})
}

// GetPost godoc
// @Summary      Получить статью по slug
// @Tags         posts
// @Produce      json
// @Param        slug  path   string  true  "Slug статьи"
// @Param        lang  query  string  false "Язык"
// @Success      200  {object}  domain.Post
// @Router       /api/v1/posts/{slug} [get]
func (h *PostHandler) GetPost(c *gin.Context) {
	slug := c.Param("slug")
	lang := c.DefaultQuery("lang", "ru")

	post, err := h.repo.GetBySlug(c.Request.Context(), slug, lang)
	if err != nil {
		RespondNotFound(c, "Post not found")
		return
	}

	c.JSON(http.StatusOK, post)
}

// Остальные методы (Create, Update, Delete) — позже, с защитой API Key

type ListPostsRequest struct {
	Page      int    `form:"page" validate:"min=1"`
	Limit     int    `form:"limit" validate:"min=1,max=100"`
	Type      string `form:"type"`
	Tag       string `form:"tag"`
	GemSlug   string `form:"gem_slug"`
	Published bool   `form:"published"`
}
