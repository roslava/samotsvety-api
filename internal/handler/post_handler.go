package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/repository"
)

type PostHandler struct {
	repo          repository.PostRepository
	mediaVerifier SlideMediaVerifier
}

// A missing verifier fails closed when a supplied image must be checked.
func NewPostHandler(repo repository.PostRepository, verifier ...SlideMediaVerifier) *PostHandler {
	h := &PostHandler{repo: repo}
	if len(verifier) > 0 {
		h.mediaVerifier = verifier[0]
	}
	return h
}

func (h *PostHandler) adminRead(c *gin.Context) (bool, bool) {
	key := c.GetHeader("X-API-Key")
	if key == "" {
		return false, true
	}
	expected := os.Getenv("ADMIN_API_KEY")
	if expected == "" || key != expected {
		NewErrorResponse(c, http.StatusUnauthorized, "unauthorized", "Invalid API key")
		return false, false
	}
	return true, true
}

func respondPostError(c *gin.Context, err error) {
	var validation *PostValidationError
	switch {
	case errors.As(err, &validation):
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": validation.Error(), "fields": validation.Fields, "code": 400})
	case errors.Is(err, repository.ErrPostNotFound):
		NewErrorResponse(c, http.StatusNotFound, "not_found", "Article not found")
	case errors.Is(err, repository.ErrUnsupportedArticleFormat):
		NewErrorResponse(c, http.StatusConflict, "unsupported_article_format", "Article is not V2")
	case errors.Is(err, repository.ErrPostPrecondition):
		NewErrorResponse(c, http.StatusPreconditionFailed, "precondition_failed", "ETag is stale")
	case errors.Is(err, repository.ErrPostSlugConflict):
		NewErrorResponse(c, http.StatusConflict, "slug_already_exists", "Article slug already exists")
	case errors.Is(err, ErrMediaVerificationUnavailable):
		NewErrorResponse(c, http.StatusServiceUnavailable, "media_verification_unavailable", err.Error())
	default:
		slog.Error("Article operation failed", "error", err)
		RespondInternalError(c, "Article operation failed")
	}
}

func etag(revision string) string { return `"` + revision + `"` }
func requiredRevision(c *gin.Context) (string, bool) {
	value := strings.TrimSpace(c.GetHeader("If-Match"))
	if value == "" {
		NewErrorResponse(c, http.StatusPreconditionRequired, "precondition_required", "If-Match is required")
		return "", false
	}
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' || strings.HasPrefix(value, "W/") {
		NewErrorResponse(c, http.StatusPreconditionFailed, "precondition_failed", "ETag is stale")
		return "", false
	}
	return value[1 : len(value)-1], true
}

func decodePostPayload(c *gin.Context) (postJSONObject, bool) {
	var payload postJSONObject
	if err := c.ShouldBindJSON(&payload); err != nil || payload == nil {
		respondPostError(c, fieldError("body", "JSON object is required"))
		return nil, false
	}
	if err := checkNewPostPayloadFields(payload); err != nil {
		respondPostError(c, err)
		return nil, false
	}
	return payload, true
}

func checkNewPostPayloadFields(payload postJSONObject) error {
	for _, field := range []string{"id", "published_at", "updated_at", "cover_image", "content_blocks"} {
		if _, ok := payload[field]; ok {
			return fieldError(field, "field is not writable in Article V2")
		}
	}
	return nil
}

func applyPostFields(post *domain.Post, payload postJSONObject, create bool) error {
	for field, raw := range payload {
		switch field {
		case "slug":
			value, err := stringValue(raw, "slug")
			if err != nil {
				return err
			}
			post.Slug = value
		case "type":
			value, err := stringValue(raw, "type")
			if err != nil {
				return err
			}
			post.Type = domain.PostType(value)
		case "i18n":
			var err error
			if create {
				post.I18n, err = prepareNewI18n(raw)
			} else {
				post.I18n, err = mergePostI18n(post.I18n, raw)
			}
			if err != nil {
				return err
			}
		case "gem_slugs":
			value, err := stringArrayValue(raw, "gem_slugs")
			if err != nil {
				return err
			}
			post.GemSlugs = value
		case "tags":
			value, err := stringArrayValue(raw, "tags")
			if err != nil {
				return err
			}
			post.Tags = value
		case "author":
			value, err := stringValue(raw, "author")
			if err != nil {
				return err
			}
			post.Author = value
		case "is_published":
			value, err := boolValue(raw, "is_published")
			if err != nil {
				return err
			}
			post.IsPublished = value
		default:
			return fieldError(field, "unknown or unwritable field")
		}
	}
	return nil
}

type ListPostsRequest struct {
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
	Type      string `form:"type"`
	Tag       string `form:"tag"`
	GemSlug   string `form:"gem_slug"`
	Published bool   `form:"published"`
	Lang      string `form:"lang"`
}

func (h *PostHandler) ListPosts(c *gin.Context) {
	admin, ok := h.adminRead(c)
	if !ok {
		return
	}
	var req ListPostsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		RespondValidationError(c, err)
		return
	}
	if req.Page < 0 || req.Limit < 0 || req.Limit > 100 {
		respondPostError(c, fieldError("pagination", "invalid page or limit"))
		return
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}
	filters := repository.PostFilterParams{
		Page: req.Page, Limit: req.Limit, Type: req.Type, Tag: req.Tag,
		GemSlug: req.GemSlug, IsPublished: req.Published,
		IncludeDrafts: admin && !req.Published, Lang: req.Lang,
	}
	posts, total, err := h.repo.List(c.Request.Context(), filters)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts, "total": total, "page": req.Page, "limit": req.Limit})
}

func (h *PostHandler) GetPost(c *gin.Context) {
	admin, ok := h.adminRead(c)
	if !ok {
		return
	}
	post, err := h.repo.GetBySlug(c.Request.Context(), c.Param("slug"), admin)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.Header("ETag", etag(post.Revision))
	c.JSON(http.StatusOK, post)
}

// CreatePostRequest and UpdatePostRequest document the envelope; nested i18n is
// intentionally raw to preserve keys unknown to this version of the service.
type CreatePostRequest struct {
	Slug        string          `json:"slug"`
	Type        domain.PostType `json:"type,omitempty"`
	I18n        json.RawMessage `json:"i18n"`
	GemSlugs    []string        `json:"gem_slugs,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	IsPublished bool            `json:"is_published"`
	Author      string          `json:"author,omitempty"`
}
type UpdatePostRequest = CreatePostRequest

func (h *PostHandler) CreatePost(c *gin.Context) {
	payload, ok := decodePostPayload(c)
	if !ok {
		return
	}
	post, err := prepareNewPost(c.Request.Context(), payload, h.mediaVerifier)
	if err != nil {
		respondPostError(c, err)
		return
	}
	if err := h.repo.Create(c.Request.Context(), post); err != nil {
		respondPostError(c, err)
		return
	}
	c.Header("ETag", etag(post.Revision))
	c.JSON(http.StatusCreated, post)
}

func prepareNewPost(ctx context.Context, payload postJSONObject, verifier SlideMediaVerifier) (*domain.Post, error) {
	post := &domain.Post{Type: domain.PostTypeGuide, GemSlugs: []string{}, Tags: []string{}}
	if err := applyPostFields(post, payload, true); err != nil {
		return nil, err
	}
	if len(post.I18n) == 0 {
		return nil, fieldError("i18n", "object is required")
	}
	if post.IsPublished {
		now := time.Now().UTC()
		post.PublishedAt = &now
	}
	if err := validateArticle(ctx, post, verifier); err != nil {
		return nil, err
	}
	return post, nil
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	revision, ok := requiredRevision(c)
	if !ok {
		return
	}
	payload, ok := decodePostPayload(c)
	if !ok {
		return
	}
	post, err := h.repo.Update(c.Request.Context(), c.Param("slug"), revision, func(post *domain.Post) error {
		if err := applyPostFields(post, payload, false); err != nil {
			return err
		}
		if post.IsPublished && post.PublishedAt == nil {
			now := time.Now().UTC()
			post.PublishedAt = &now
		}
		return validateArticle(c.Request.Context(), post, h.mediaVerifier)
	})
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.Header("ETag", etag(post.Revision))
	c.JSON(http.StatusOK, post)
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	revision, ok := requiredRevision(c)
	if !ok {
		return
	}
	if err := h.repo.Delete(c.Request.Context(), c.Param("slug"), revision); err != nil {
		respondPostError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PostHandler) SearchPosts(c *gin.Context) {
	q := c.Query("q")
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			respondPostError(c, fieldError("limit", "must be between 1 and 100"))
			return
		}
		limit = value
	}
	posts, total, err := h.repo.Search(c.Request.Context(), q, c.DefaultQuery("lang", "ru"), limit, 0)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts, "total": total, "limit": limit})
}
