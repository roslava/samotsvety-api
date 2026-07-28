///home/akdev/projects/samotsvety-api/internal/handler/dto.go

package handler

import "github.com/roslava/samotsvety-api/internal/domain"

// CreateMineralRequest — DTO для создания минерала
// SafetyNotes больше не отдельное поле — он теперь внутри I18n.Ru/En.SafetyNotes,
// как и остальной языкозависимый текст (заполняется через поле i18n целиком).
type CreateMineralRequest struct {
	Slug            string                `json:"slug" validate:"required,alphanumdash"`
	Type            domain.EntityType     `json:"type" validate:"required,oneof=mineral rock gem_variety organic"`
	Scientific      domain.Scientific     `json:"scientific" validate:"required"`
	I18n            domain.I18n           `json:"i18n" validate:"required"`
	Localities      []domain.Locality     `json:"localities,omitempty"`
	MainImageURL    string                `json:"main_image_url,omitempty"`
	ThumbnailURL    string                `json:"thumbnail_url,omitempty"`
	Gallery         []domain.GalleryImage `json:"gallery,omitempty"`
	RelatedMinerals []string              `json:"related_minerals,omitempty"`
}

// UpdateMineralRequest — DTO для обновления (все поля опциональны + поддержка slug)
type UpdateMineralRequest struct {
	Slug            *string                `json:"slug,omitempty" validate:"omitempty,alphanumdash"`
	Type            *domain.EntityType     `json:"type,omitempty" validate:"omitempty,oneof=mineral rock gem_variety organic"`
	Scientific      *domain.Scientific     `json:"scientific,omitempty"`
	I18n            *domain.I18n           `json:"i18n,omitempty"`
	Localities      *[]domain.Locality     `json:"localities,omitempty"` // указатель — важно!
	MainImageURL    *string                `json:"main_image_url,omitempty"`
	ThumbnailURL    *string                `json:"thumbnail_url,omitempty"`
	Gallery         *[]domain.GalleryImage `json:"gallery,omitempty"`          // указатель
	RelatedMinerals *[]string              `json:"related_minerals,omitempty"` // указатель
}

// ListMineralsRequest — параметры списка с пагинацией и фильтрами
type ListMineralsRequest struct {
	Page        int    `form:"page" validate:"omitempty,min=1"`
	Limit       int    `form:"limit" validate:"omitempty,min=1,max=100"`
	Sort        string `form:"sort" validate:"omitempty,oneof=created_at name rarity hardness"`
	Order       string `form:"order" validate:"omitempty,oneof=asc desc"`
	Lang        string `form:"lang" validate:"omitempty,oneof=ru en"`
	View        string `form:"view" validate:"omitempty,oneof=normal esoteric"`
	RussianOnly bool   `form:"russian_only"`
}
