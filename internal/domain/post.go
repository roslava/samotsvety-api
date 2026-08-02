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

// BlockType — тип блока в композиции статьи
type BlockType string

const (
	BlockTypeHeading   BlockType = "heading"
	BlockTypeParagraph BlockType = "paragraph"
	BlockTypeImage     BlockType = "image"      // одиночная картинка, Layout: full | inset
	BlockTypeImagePair BlockType = "image_pair" // две картинки в 2 колонки
	BlockTypeQuote     BlockType = "quote"
)

// ImageLayout — вариант вёрстки одиночной картинки
type ImageLayout string

const (
	ImageLayoutFull  ImageLayout = "full"  // на всю ширину, выходит за пределы колонки текста
	ImageLayoutInset ImageLayout = "inset" // вписана в ширину колонки текста
)

// Post — статья/пост о камнях
type Post struct {
	ID   string   `json:"id" db:"id"`
	Slug string   `json:"slug" validate:"required,alphanumdash"`
	Type PostType `json:"type" validate:"required,oneof=blog guide history esoteric review"`

	I18n PostI18n `json:"i18n" validate:"required"`

	CoverImage    string         `json:"cover_image,omitempty"`
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty" validate:"dive"`
	GemSlugs      []string       `json:"gem_slugs,omitempty"` // Связанные самоцветы
	Tags          []string       `json:"tags,omitempty"`

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
	Content string `json:"content,omitempty"` // Markdown или HTML — устаревшее поле, для старых статей до перехода на блоки
}

// ContentBlock — один элемент композиции статьи. Порядок в массиве = порядок на странице.
// Медиа (URL картинок, layout) языконезависимы — одно и то же фото что в RU, что в EN версии.
// Языкозависим только текст: он лежит в I18n.
type ContentBlock struct {
	ID   string    `json:"id" validate:"required"`
	Type BlockType `json:"type" validate:"required,oneof=heading paragraph image image_pair quote"`

	// Для BlockTypeImage
	Layout   ImageLayout `json:"layout,omitempty" validate:"omitempty,oneof=full inset"`
	ImageURL string      `json:"image_url,omitempty"`

	// Для BlockTypeImagePair — ровно 2 URL
	ImageURLs []string `json:"image_urls,omitempty" validate:"omitempty,max=2"`

	I18n BlockI18n `json:"i18n"`
}

// BlockI18n — языкозависимый текст внутри блока
type BlockI18n struct {
	Ru BlockLangData `json:"ru"`
	En BlockLangData `json:"en"`
}

// BlockLangData — текстовые поля блока на одном языке.
// Какие поля используются — зависит от Type блока:
//
//	heading/paragraph/quote — Text
//	quote                   — плюс Attribution
//	image                   — Caption
//	image_pair              — Captions[0], Captions[1] (по числу ImageURLs)
type BlockLangData struct {
	Text        string   `json:"text,omitempty"`
	Attribution string   `json:"attribution,omitempty"`
	Caption     string   `json:"caption,omitempty"`
	Captions    []string `json:"captions,omitempty"`
}
