package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/repository"
)

// GemEntityV2Handler serves only the canonical V2 contract. It intentionally
// has no dependency on the V1 handler or DTOs.
type GemEntityV2Handler struct {
	repo repository.GemEntityV2Repository
}

var countryCodeQueryPattern = regexp.MustCompile(`^[A-Z]{2}$`)

func NewGemEntityV2Handler(repo repository.GemEntityV2Repository) *GemEntityV2Handler {
	return &GemEntityV2Handler{repo: repo}
}

func (h *GemEntityV2Handler) List(c *gin.Context) {
	entities, err := h.repo.ListV2(c.Request.Context())
	if err != nil {
		RespondInternalError(c, "Failed to fetch gem entities")
		return
	}
	countryCode := c.Query("country_code")
	if countryCode != "" {
		if !countryCodeQueryPattern.MatchString(countryCode) {
			respondV2Error(c, &v2PayloadError{Path: "country_code", Message: "must be two uppercase ASCII letters"})
			return
		}
		entities = filterV2ByCountry(entities, countryCode)
	}
	if entities == nil {
		entities = []domain.GemEntityV2{}
	}
	c.JSON(http.StatusOK, gin.H{"data": entities, "total": len(entities)})
}

func filterV2ByCountry(entities []domain.GemEntityV2, code string) []domain.GemEntityV2 {
	result := make([]domain.GemEntityV2, 0)
	for _, entity := range entities {
		for _, locality := range entity.Localities {
			if locality.CountryCode == code {
				result = append(result, entity)
				break
			}
		}
	}
	return result
}

func (h *GemEntityV2Handler) Get(c *gin.Context) {
	entity, err := h.repo.GetV2BySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		RespondNotFound(c, "Gem entity not found")
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *GemEntityV2Handler) Create(c *gin.Context) {
	entity, err := decodeV2Entity(c.Request.Body)
	if err != nil {
		respondV2Error(c, err)
		return
	}
	if err := entity.Validate(); err != nil {
		respondV2Error(c, err)
		return
	}
	if err := h.repo.CreateV2(c.Request.Context(), entity); err != nil {
		if isConflict(err) {
			RespondWithError(c, http.StatusConflict, "Gem entity with this slug already exists")
			return
		}
		respondRepositoryV2Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, entity)
}

// Replace implements PUT as full replacement; every required V2 field must be
// present in the request, rather than inheriting fields from the old entity.
func (h *GemEntityV2Handler) Replace(c *gin.Context) {
	entity, err := decodeV2Entity(c.Request.Body)
	if err != nil {
		respondV2Error(c, err)
		return
	}
	if err := entity.Validate(); err != nil {
		respondV2Error(c, err)
		return
	}
	if err := h.repo.UpdateV2(c.Request.Context(), c.Param("slug"), entity); err != nil {
		respondRepositoryV2Error(c, err)
		return
	}
	h.respondUpdated(c, entity)
}

// Patch uses RFC 7396 merge-patch semantics for objects. Arrays are replaced,
// and null deletes an optional field before the fully merged entity is decoded
// and atomically validated.
func (h *GemEntityV2Handler) Patch(c *gin.Context) {
	patch, err := decodeV2Patch(c.Request.Body)
	if err != nil {
		respondV2Error(c, err)
		return
	}
	oldSlug := c.Param("slug")
	current, err := h.repo.GetV2BySlug(c.Request.Context(), oldSlug)
	if err != nil {
		RespondNotFound(c, "Gem entity not found")
		return
	}
	currentJSON, err := json.Marshal(current)
	if err != nil {
		RespondInternalError(c, "Failed to prepare gem entity")
		return
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(currentJSON, &document); err != nil {
		RespondInternalError(c, "Failed to prepare gem entity")
		return
	}
	// Timestamps are response-only and are therefore excluded from the input
	// document before it goes through the same strict canonical decoder.
	delete(document, "created_at")
	delete(document, "updated_at")
	merged := mergePatch(document, patch)
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		RespondInternalError(c, "Failed to apply patch")
		return
	}
	entity, err := decodeV2Entity(bytes.NewReader(mergedJSON))
	if err != nil {
		respondV2Error(c, err)
		return
	}
	if err := entity.Validate(); err != nil {
		respondV2Error(c, err)
		return
	}
	if err := h.repo.UpdateV2(c.Request.Context(), oldSlug, entity); err != nil {
		respondRepositoryV2Error(c, err)
		return
	}
	h.respondUpdated(c, entity)
}

func (h *GemEntityV2Handler) respondUpdated(c *gin.Context, fallback *domain.GemEntityV2) {
	// PostgreSQL owns updated_at (NOW()); return the persisted representation so
	// the response never advertises the stale pre-update timestamp.
	updated, err := h.repo.GetV2BySlug(c.Request.Context(), fallback.Slug)
	if err != nil {
		c.JSON(http.StatusOK, fallback)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func mergePatch(target, patch map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(target)+len(patch))
	for key, value := range target {
		result[key] = value
	}
	for key, value := range patch {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			delete(result, key)
			continue
		}
		var patchObject map[string]json.RawMessage
		var targetObject map[string]json.RawMessage
		if json.Unmarshal(value, &patchObject) == nil && json.Unmarshal(result[key], &targetObject) == nil {
			encoded, _ := json.Marshal(mergePatch(targetObject, patchObject))
			result[key] = encoded
			continue
		}
		result[key] = value
	}
	return result
}

type v2PayloadError struct{ Path, Message string }

func (e *v2PayloadError) Error() string { return e.Path + ": " + e.Message }

func decodeV2Entity(body io.Reader) (*domain.GemEntityV2, error) {
	raw, err := readAndValidateV2JSON(body, false)
	if err != nil {
		return nil, err
	}
	var entity domain.GemEntityV2
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&entity); err != nil {
		return nil, &v2PayloadError{Path: "$", Message: err.Error()}
	}
	return &entity, nil
}

func decodeV2Patch(body io.Reader) (map[string]json.RawMessage, error) {
	raw, err := readAndValidateV2JSON(body, true)
	if err != nil {
		return nil, err
	}
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(raw, &patch); err != nil {
		return nil, &v2PayloadError{Path: "$", Message: "must be a JSON object"}
	}
	return patch, nil
}

func readAndValidateV2JSON(body io.Reader, patch bool) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(body, 2<<20))
	if err != nil {
		return nil, &v2PayloadError{Path: "$", Message: "unable to read request body"}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, &v2PayloadError{Path: "$", Message: "request body is required"}
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, &v2PayloadError{Path: "$", Message: "invalid JSON: " + err.Error()}
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, &v2PayloadError{Path: "$", Message: "request body must contain one JSON value"}
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, &v2PayloadError{Path: "$", Message: "must be a JSON object"}
	}
	if err := validateV2JSONStructure(raw, patch); err != nil {
		return nil, err
	}
	return raw, nil
}

// validateV2JSONStructure is intentionally performed before decoding: the
// standard JSON decoder silently accepts duplicate object keys.
func validateV2JSONStructure(raw []byte, patch bool) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return validateV2Value(decoder, "$", v2Schema, patch)
}

type v2SchemaNode map[string]v2SchemaNode

var leaf = v2SchemaNode{}
var v2Schema = v2SchemaNode{
	"slug": leaf, "type": leaf, "scientific": {"chemical_formula": leaf, "hardness": {"min": leaf, "max": leaf}, "specific_gravity": {"min": leaf, "max": leaf}, "rarity": leaf, "base_color": leaf, "mineral_class": leaf, "silicate_subclass": leaf, "mineral_family": leaf, "crystal_system": leaf, "crystal_habit": leaf, "streak": leaf, "transparency": leaf, "luster": leaf, "tenacity": leaf, "fracture": leaf, "cleavage_degree": leaf, "cleavage_direction": leaf, "cleavage_type": leaf, "phenomena": leaf, "ima_status": leaf, "rock_type": leaf},
	"i18n":             {"ru": localizedSchema(), "en": localizedSchema()},
	"localities":       {"country_code": leaf, "country_ru": leaf, "country_en": leaf, "region_ru": leaf, "region_en": leaf, "locality_ru": leaf, "locality_en": leaf, "description_ru": leaf, "description_en": leaf, "famous": leaf},
	"images":           {"storage_key": leaf, "hero": {"path": leaf}, "thumbnail": {"path": leaf}, "gallery": {"path": leaf, "type": leaf, "caption": {"ru": leaf, "en": leaf}}},
	"related_entities": leaf, "sources": {"title": leaf, "url": leaf, "author": leaf, "publisher": leaf},
}

func localizedSchema() v2SchemaNode {
	return v2SchemaNode{"name": leaf, "synonyms": leaf, "color": leaf, "color_description": leaf, "lore": leaf, "identification_tips": leaf, "safety_notes": leaf, "scientific_notes": {"hardness": leaf, "composition": leaf}, "esoteric": {"metaphysical_properties": leaf, "chakras": leaf, "zodiac": leaf, "healing_interpretation": leaf, "energy_notes": leaf, "ritual_uses": leaf}}
}

func validateV2Value(decoder *json.Decoder, path string, schema v2SchemaNode, patch bool) error {
	token, err := decoder.Token()
	if err != nil {
		return &v2PayloadError{Path: path, Message: "invalid JSON"}
	}
	delim, object := token.(json.Delim)
	if !object || delim != '{' {
		return &v2PayloadError{Path: path, Message: "must be a JSON object"}
	}
	seen := map[string]bool{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return &v2PayloadError{Path: path, Message: "invalid JSON object"}
		}
		key := keyToken.(string)
		keyPath := key
		if path != "$" {
			keyPath = path + "." + key
		}
		if seen[key] {
			return &v2PayloadError{Path: keyPath, Message: "duplicate field"}
		}
		seen[key] = true
		child, known := schema[key]
		if !known {
			return &v2PayloadError{Path: keyPath, Message: "unknown field"}
		}
		if len(child) == 0 {
			if err := skipJSONValue(decoder); err != nil {
				return err
			}
			continue
		}
		if err := validateKnownContainer(decoder, keyPath, child, patch); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

func validateKnownContainer(decoder *json.Decoder, path string, schema v2SchemaNode, patch bool) error {
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return &v2PayloadError{Path: path, Message: "invalid JSON value"}
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return validateV2JSONStructureAt(raw, path, schema, patch)
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return &v2PayloadError{Path: path, Message: "must be an array"}
		}
		for i, value := range values {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				continue
			}
			if err := validateV2JSONStructureAt(value, fmt.Sprintf("%s[%d]", path, i), schema, patch); err != nil {
				return err
			}
		}
		return nil
	}
	return &v2PayloadError{Path: path, Message: "must be an object or array"}
}
func validateV2JSONStructureAt(raw []byte, path string, schema v2SchemaNode, patch bool) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	return validateV2Value(d, path, schema, patch)
}
func skipJSONValue(decoder *json.Decoder) error {
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return &v2PayloadError{Path: "$", Message: "invalid JSON value"}
	}
	return nil
}

func respondV2Error(c *gin.Context, err error) {
	var validation *domain.ValidationError
	var payload *v2PayloadError
	path, message := "$", err.Error()
	if errors.As(err, &validation) {
		path, message = validation.Path, validation.Message
	}
	if errors.As(err, &payload) {
		path, message = payload.Path, payload.Message
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": message, "path": path, "code": http.StatusBadRequest})
}
func respondRepositoryV2Error(c *gin.Context, err error) {
	if strings.Contains(err.Error(), "not found") {
		RespondNotFound(c, "Gem entity not found")
		return
	}
	if isConflict(err) {
		RespondWithError(c, http.StatusConflict, "Gem entity with this slug already exists")
		return
	}
	RespondInternalError(c, "Failed to save gem entity")
}
func isConflict(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(err.Error(), "already exists")
}
