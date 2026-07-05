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

	TitleRu   string `json:"title_ru" validate:"required"`
	TitleEn   string `json:"title_en" validate:"required"`
	ExcerptRu string `json:"excerpt_ru"`
	ExcerptEn string `json:"excerpt_en"`

	ContentRu string `json:"content_ru"` // Markdown или HTML
	ContentEn string `json:"content_en"`

	CoverImage string   `json:"cover_image,omitempty"`
	GemSlugs   []string `json:"gem_slugs,omitempty"` // Связанные самоцветы
	Tags       []string `json:"tags,omitempty"`

	PublishedAt *time.Time `json:"published_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	IsPublished bool       `json:"is_published"`
	Author      string     `json:"author,omitempty"`
}
