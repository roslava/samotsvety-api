// internal/domain/post.go
package domain

import "time"

// PostType — тип статьи
type PostType string

const (
	PostTypeBlog     PostType = "blog"
	PostTypeGuide    PostType = "guide"
	PostTypeHistory  PostType = "history"
	PostTypeEsoteric PostType = "esoteric"
	PostTypeReview   PostType = "review"
)

// Post — статья/пост о камнях
type Post struct {
	ID   string   `json:"id" db:"id"`
	Slug string   `json:"slug" validate:"required,alphanumdash"`
	Type PostType `json:"type" validate:"required,oneof=blog guide history esoteric review"`

	I18n PostI18n `json:"i18n" validate:"required"`

	CoverImage string   `json:"cover_image,omitempty"`
	GemSlugs   []string `json:"gem_slugs,omitempty"` // Связанные самоцветы
	Tags       []string `json:"tags,omitempty"`

	PublishedAt *time.Time `json:"published_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	IsPublished bool       `json:"is_published"`
	Author      string     `json:"author,omitempty"`
}

// PostI18n — переводимый контент статьи, тот же паттерн, что и I18n у GemEntity
type PostI18n struct {
	Ru PostLangData `json:"ru"`
	En PostLangData `json:"en"`
}

// PostLangData — контент статьи на одном языке
type PostLangData struct {
	Title   string `json:"title" validate:"required"`
	Excerpt string `json:"excerpt,omitempty"`
	Content string `json:"content,omitempty"` // Markdown или HTML
}
