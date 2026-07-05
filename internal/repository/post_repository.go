// internal/repository/post_repository.go
package repository

import (
	"context"

	"github.com/roslava/samotsvety-api/internal/domain"
)

// PostRepository определяет операции со статьями
type PostRepository interface {
	GetBySlug(ctx context.Context, slug, lang string) (*domain.Post, error)
	List(ctx context.Context, filters PostFilterParams) ([]domain.Post, int, error)
	Search(ctx context.Context, query, lang string, limit, offset int) ([]domain.Post, int, error)

	// Admin CRUD
	Create(ctx context.Context, post *domain.Post) error
	Update(ctx context.Context, slug string, post *domain.Post) error
	Delete(ctx context.Context, slug string) error

	// Связанные статьи
	GetByGemSlug(ctx context.Context, gemSlug string, lang string, limit int) ([]domain.Post, error)
}

// PostFilterParams параметры фильтрации статей
type PostFilterParams struct {
	Lang        string `form:"lang"`
	Type        string `form:"type"`
	Tag         string `form:"tag"`
	GemSlug     string `form:"gem_slug"`
	IsPublished bool   `form:"published"`
	Limit       int    `form:"limit"`
	Page        int    `form:"page"`
	Sort        string `form:"sort"`
	Order       string `form:"order"`
}
