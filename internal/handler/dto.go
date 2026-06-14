package handler

import "github.com/roslava/samotsvety-api/internal/domain"

// CreateMineralRequest — DTO для создания минерала
type CreateMineralRequest struct {
	Slug            string                `json:"slug" validate:"required,alphanumdash"`
	Scientific      domain.Scientific     `json:"scientific" validate:"required"`
	I18n            domain.I18n           `json:"i18n" validate:"required"`
	Localities      []domain.Locality     `json:"localities,omitempty"`
	MainImageURL    string                `json:"main_image_url,omitempty"`
	Gallery         []domain.GalleryImage `json:"gallery,omitempty"`
	SafetyNotes     string                `json:"safety_notes,omitempty"`
	RelatedMinerals []string              `json:"related_minerals,omitempty"`
}

// UpdateMineralRequest — DTO для обновления минерала (все поля опциональны)
type UpdateMineralRequest struct {
	Scientific      *domain.Scientific    `json:"scientific,omitempty"`
	I18n            *domain.I18n          `json:"i18n,omitempty"`
	Localities      []domain.Locality     `json:"localities,omitempty"`
	MainImageURL    *string               `json:"main_image_url,omitempty"`
	Gallery         []domain.GalleryImage `json:"gallery,omitempty"`
	SafetyNotes     *string               `json:"safety_notes,omitempty"`
	RelatedMinerals []string              `json:"related_minerals,omitempty"`
}
