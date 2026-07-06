// internal/handler/post_handler.go
package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
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
// @Success      200  {object}  gin.H
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

// CreatePost godoc
// @Summary      Создать новую статью
// @Description  Создаёт новую статью (только админ)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        post body handler.CreatePostRequest true "Данные статьи"
// @Success      201  {object}  domain.Post
// @Failure      400  {object}  handler.ErrorResponse
// @Failure      409  {object}  handler.ErrorResponse "Slug уже существует"
// @Router       /api/v1/posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	now := time.Now().UTC()

	post := &domain.Post{
		Slug:        req.Slug,
		Type:        req.Type,
		TitleRu:     req.TitleRu,
		TitleEn:     req.TitleEn,
		ExcerptRu:   req.ExcerptRu,
		ExcerptEn:   req.ExcerptEn,
		ContentRu:   req.ContentRu,
		ContentEn:   req.ContentEn,
		CoverImage:  req.CoverImage,
		GemSlugs:    req.GemSlugs,
		Tags:        req.Tags,
		PublishedAt: req.PublishedAt,
		UpdatedAt:   now,
		IsPublished: req.IsPublished,
		Author:      req.Author,
	}

	if err := h.repo.Create(c.Request.Context(), post); err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
			RespondWithError(c, http.StatusConflict, "Статья с таким slug уже существует")
			return
		}
		slog.Error("Failed to create post", "error", err, "slug", req.Slug)
		RespondInternalError(c, "Не удалось создать статью")
		return
	}

	c.JSON(http.StatusCreated, post)
}

type ListPostsRequest struct {
	Page      int    `form:"page" validate:"min=1"`
	Limit     int    `form:"limit" validate:"min=1,max=100"`
	Type      string `form:"type"`
	Tag       string `form:"tag"`
	GemSlug   string `form:"gem_slug"`
	Published bool   `form:"published"`
}

type CreatePostRequest struct {
	Slug        string          `json:"slug" validate:"required"`
	Type        domain.PostType `json:"type" validate:"required"`
	TitleRu     string          `json:"title_ru" validate:"required"`
	TitleEn     string          `json:"title_en" validate:"required"`
	ExcerptRu   string          `json:"excerpt_ru"`
	ExcerptEn   string          `json:"excerpt_en"`
	ContentRu   string          `json:"content_ru"`
	ContentEn   string          `json:"content_en"`
	CoverImage  string          `json:"cover_image"`
	GemSlugs    []string        `json:"gem_slugs"`
	Tags        []string        `json:"tags"`
	PublishedAt *time.Time      `json:"published_at"`
	IsPublished bool            `json:"is_published"`
	Author      string          `json:"author"`
}

// UpdatePost godoc
// @Summary      Обновить статью
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        slug  path    string  true  "Slug статьи"
// @Param        post body handler.UpdatePostRequest true "Данные для обновления"
// @Success      200  {object}  domain.Post
// @Failure      400  {object}  handler.ErrorResponse
// @Failure      404  {object}  handler.ErrorResponse
// @Router       /api/v1/posts/{slug} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) {
	slug := c.Param("slug")

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	// Получаем текущую статью
	post, err := h.repo.GetBySlug(c.Request.Context(), slug, "ru")
	if err != nil {
		RespondNotFound(c, "Статья не найдена")
		return
	}

	// Обновляем поля, если переданы
	if req.Slug != nil && *req.Slug != "" {
		post.Slug = *req.Slug
	}
	if req.Type != nil {
		post.Type = *req.Type
	}
	if req.TitleRu != nil {
		post.TitleRu = *req.TitleRu
	}
	if req.TitleEn != nil {
		post.TitleEn = *req.TitleEn
	}
	if req.ExcerptRu != nil {
		post.ExcerptRu = *req.ExcerptRu
	}
	if req.ExcerptEn != nil {
		post.ExcerptEn = *req.ExcerptEn
	}
	if req.ContentRu != nil {
		post.ContentRu = *req.ContentRu
	}
	if req.ContentEn != nil {
		post.ContentEn = *req.ContentEn
	}
	if req.CoverImage != nil {
		post.CoverImage = *req.CoverImage
	}
	if req.GemSlugs != nil {
		post.GemSlugs = *req.GemSlugs
	}
	if req.Tags != nil {
		post.Tags = *req.Tags
	}
	if req.PublishedAt != nil {
		post.PublishedAt = req.PublishedAt
	}
	if req.IsPublished != nil {
		post.IsPublished = *req.IsPublished
	}
	if req.Author != nil {
		post.Author = *req.Author
	}

	post.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(c.Request.Context(), slug, post); err != nil {
		slog.Error("Failed to update post", "error", err, "slug", slug)
		RespondInternalError(c, "Не удалось обновить статью")
		return
	}

	c.JSON(http.StatusOK, post)
}

// DeletePost godoc
// @Summary      Удалить статью
// @Tags         posts
// @Produce      json
// @Param        slug  path  string  true  "Slug статьи"
// @Success      204
// @Failure      404  {object}  handler.ErrorResponse
// @Router       /api/v1/posts/{slug} [delete]
func (h *PostHandler) DeletePost(c *gin.Context) {
	slug := c.Param("slug")

	if err := h.repo.Delete(c.Request.Context(), slug); err != nil {
		if err.Error() == "post not found" {
			RespondNotFound(c, "Статья не найдена")
			return
		}
		slog.Error("Failed to delete post", "error", err, "slug", slug)
		RespondInternalError(c, "Не удалось удалить статью")
		return
	}

	c.Status(http.StatusNoContent)
}

type UpdatePostRequest struct {
	Slug        *string          `json:"slug"`
	Type        *domain.PostType `json:"type"`
	TitleRu     *string          `json:"title_ru"`
	TitleEn     *string          `json:"title_en"`
	ExcerptRu   *string          `json:"excerpt_ru"`
	ExcerptEn   *string          `json:"excerpt_en"`
	ContentRu   *string          `json:"content_ru"`
	ContentEn   *string          `json:"content_en"`
	CoverImage  *string          `json:"cover_image"`
	GemSlugs    *[]string        `json:"gem_slugs"`
	Tags        *[]string        `json:"tags"`
	PublishedAt *time.Time       `json:"published_at"`
	IsPublished *bool            `json:"is_published"`
	Author      *string          `json:"author"`
}
