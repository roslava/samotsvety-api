package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/repository"
)

type v2RepositoryStub struct {
	entities     map[string]*domain.GemEntityV2
	deleteErr    error
	deletedSlug  string
	relatedItems []repository.RelatedGemEntityV2
	relatedErr   error
	relatedCalls int
	listCalls    int
	getErr       error
}

func newV2RepositoryStub(entity *domain.GemEntityV2) *v2RepositoryStub {
	return &v2RepositoryStub{entities: map[string]*domain.GemEntityV2{entity.Slug: entity}}
}
func (r *v2RepositoryStub) GetV2BySlug(_ context.Context, slug string) (*domain.GemEntityV2, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	entity, ok := r.entities[slug]
	if !ok {
		return nil, sql.ErrNoRows
	}
	copy := *entity
	return &copy, nil
}
func (r *v2RepositoryStub) ListV2(_ context.Context) ([]domain.GemEntityV2, error) {
	r.listCalls++
	result := make([]domain.GemEntityV2, 0, len(r.entities))
	for _, entity := range r.entities {
		result = append(result, *entity)
	}
	return result, nil
}
func (r *v2RepositoryStub) GetV2Related(_ context.Context, _ string) ([]repository.RelatedGemEntityV2, error) {
	r.relatedCalls++
	return r.relatedItems, r.relatedErr
}

func TestV2RelatedCompactResponse(t *testing.T) {
	repo := newV2RepositoryStub(&domain.GemEntityV2{Slug: "landscape-jasper"})
	repo.relatedItems = []repository.RelatedGemEntityV2{
		{Slug: "red-jasper", Type: domain.TypeRock, NameRU: "Яшма красная", NameEN: "Red Jasper", Direction: "reverse"},
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v2/gem-entities/landscape-jasper/related", nil)
	context.Params = gin.Params{{Key: "slug", Value: "landscape-jasper"}}
	NewGemEntityV2Handler(repo).Related(context)
	if recorder.Code != http.StatusOK || repo.relatedCalls != 1 || repo.listCalls != 0 {
		t.Fatalf("status=%d related calls=%d list calls=%d: %s", recorder.Code, repo.relatedCalls, repo.listCalls, recorder.Body.String())
	}
	var response struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 || response.Data[0]["name_ru"] != "Яшма красная" || response.Data[0]["name_en"] != "Red Jasper" || response.Data[0]["direction"] != "reverse" {
		t.Fatalf("unexpected related response: %s", recorder.Body.String())
	}
	if _, fullCard := response.Data[0]["scientific"]; fullCard {
		t.Fatalf("full card leaked: %s", recorder.Body.String())
	}
}

func TestV2RelatedMissingSourceAndQueryError(t *testing.T) {
	for _, tt := range []struct {
		repo         *v2RepositoryStub
		status       int
		wantError    string
		relatedCalls int
	}{
		{newV2RepositoryStub(&domain.GemEntityV2{Slug: "other"}), http.StatusNotFound, `"error":"not_found"`, 0},
		{&v2RepositoryStub{entities: map[string]*domain.GemEntityV2{}, getErr: errors.New("db unavailable")}, http.StatusInternalServerError, `"error":"internal_error"`, 0},
		{&v2RepositoryStub{entities: map[string]*domain.GemEntityV2{}, getErr: errors.New("index not found")}, http.StatusInternalServerError, `"error":"internal_error"`, 0},
		{&v2RepositoryStub{entities: map[string]*domain.GemEntityV2{"landscape-jasper": {Slug: "landscape-jasper"}}, relatedErr: errors.New("db unavailable")}, http.StatusInternalServerError, `"error":"internal_error"`, 1},
	} {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		context.Params = gin.Params{{Key: "slug", Value: "landscape-jasper"}}
		NewGemEntityV2Handler(tt.repo).Related(context)
		if recorder.Code != tt.status || tt.repo.listCalls != 0 || tt.repo.relatedCalls != tt.relatedCalls || !bytes.Contains(recorder.Body.Bytes(), []byte(tt.wantError)) {
			t.Fatalf("status=%d related calls=%d list calls=%d body=%s", recorder.Code, tt.repo.relatedCalls, tt.repo.listCalls, recorder.Body.String())
		}
	}
}

func TestV2RelatedEmptyList(t *testing.T) {
	repo := newV2RepositoryStub(&domain.GemEntityV2{Slug: "unlinked-gem"})
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	context.Params = gin.Params{{Key: "slug", Value: "unlinked-gem"}}
	NewGemEntityV2Handler(repo).Related(context)
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"data":[]`)) {
		t.Fatalf("empty relations must be an array: %d %s", recorder.Code, recorder.Body.String())
	}
}
func (r *v2RepositoryStub) CreateV2(_ context.Context, entity *domain.GemEntityV2) error {
	r.entities[entity.Slug] = entity
	return nil
}
func (r *v2RepositoryStub) UpdateV2(_ context.Context, oldSlug string, entity *domain.GemEntityV2) error {
	if _, ok := r.entities[oldSlug]; !ok {
		return errV2NotFound{}
	}
	delete(r.entities, oldSlug)
	r.entities[entity.Slug] = entity
	return nil
}
func (r *v2RepositoryStub) DeleteV2(_ context.Context, slug string) error {
	r.deletedSlug = slug
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.entities[slug]; !ok {
		return errV2NotFound{}
	}
	delete(r.entities, slug)
	return nil
}

type errV2NotFound struct{}

func (errV2NotFound) Error() string { return "not found" }

func validV2JSON() []byte {
	return []byte(`{"slug":"test-gem","type":"mineral","scientific":{"hardness":{"min":6,"max":7}},"i18n":{"ru":{"name":"Тест"},"en":{"name":"Test"}},"localities":[{"country_code":"RU"}],"related_entities":[],"sources":[]}`)
}

func TestV2CreateRejectsUnknownAndDuplicateNestedFields(t *testing.T) {
	for _, body := range [][]byte{
		bytes.Replace(validV2JSON(), []byte(`"name":"Тест"`), []byte(`"name":"Тест","legacy_name":"x"`), 1),
		bytes.Replace(validV2JSON(), []byte(`"max":7`), []byte(`"max":7,"max":8`), 1),
	} {
		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = request
		NewGemEntityV2Handler(newV2RepositoryStub(&domain.GemEntityV2{Slug: "existing"})).Create(context)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		var response map[string]any
		_ = json.Unmarshal(recorder.Body.Bytes(), &response)
		if response["error"] != "validation_failed" {
			t.Fatalf("unexpected response: %s", recorder.Body.String())
		}
	}
}

func TestV2CreateValidatesLocalityCoordinates(t *testing.T) {
	tests := []struct {
		name       string
		localities string
		wantStatus int
		wantPath   string
	}{
		{name: "existing locality without coordinates", localities: `[{"country_code":"MG","region_en":"Bongolava","locality_en":"Kambaba Jasper occurrence"}]`, wantStatus: http.StatusCreated},
		{name: "valid coordinates", localities: `[{"country_code":"MG","latitude":-16.4,"longitude":46.5,"coordinate_precision":"approximate"}]`, wantStatus: http.StatusCreated},
		{name: "latitude out of range", localities: `[{"latitude":90.1}]`, wantStatus: http.StatusBadRequest, wantPath: "localities[0].latitude"},
		{name: "longitude out of range", localities: `[{"longitude":-180.1}]`, wantStatus: http.StatusBadRequest, wantPath: "localities[0].longitude"},
		{name: "unknown precision", localities: `[{"coordinate_precision":"estimated"}]`, wantStatus: http.StatusBadRequest, wantPath: "localities[0].coordinate_precision"},
		{name: "unknown coordinate field remains rejected", localities: `[{"altitude":100}]`, wantStatus: http.StatusBadRequest, wantPath: "localities[0].altitude"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"slug":"test-gem","type":"mineral","scientific":{},"i18n":{"ru":{"name":"Тест"},"en":{"name":"Test"}},"localities":` + tt.localities + `,"related_entities":[],"sources":[]}`)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			NewGemEntityV2Handler(newV2RepositoryStub(&domain.GemEntityV2{Slug: "existing"})).Create(context)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if tt.wantPath != "" && !bytes.Contains(recorder.Body.Bytes(), []byte(`"path":"`+tt.wantPath+`"`)) {
				t.Fatalf("response path missing %q: %s", tt.wantPath, recorder.Body.String())
			}
		})
	}
}

func TestV2PatchMergesNestedObjectsAndRevalidates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var entity domain.GemEntityV2
	if err := json.Unmarshal(validV2JSON(), &entity); err != nil {
		t.Fatal(err)
	}
	repo := newV2RepositoryStub(&entity)
	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"scientific":{"hardness":{"max":8}}}`))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	context.Params = gin.Params{{Key: "slug", Value: "test-gem"}}
	NewGemEntityV2Handler(repo).Patch(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := repo.entities["test-gem"].Scientific.Hardness; got.Min != 6 || got.Max != 8 {
		t.Fatalf("hardness = %#v", got)
	}

	request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"scientific":{"hardness":{"min":9}}}`))
	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = request
	context.Params = gin.Params{{Key: "slug", Value: "test-gem"}}
	NewGemEntityV2Handler(repo).Patch(context)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"localities":[{"country_code":"MG","latitude":-16.4,"longitude":46.5,"coordinate_precision":"approximate"}]}`))
	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = request
	context.Params = gin.Params{{Key: "slug", Value: "test-gem"}}
	NewGemEntityV2Handler(repo).Patch(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("coordinate PATCH status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	locality := repo.entities["test-gem"].Localities[0]
	if locality.Latitude == nil || locality.Longitude == nil || *locality.Latitude != -16.4 || *locality.Longitude != 46.5 || locality.CoordinatePrecision != domain.CoordinatePrecisionApproximate {
		t.Fatalf("coordinate PATCH did not replace localities correctly: %#v", locality)
	}
}

func TestV2ListFiltersCountryCode(t *testing.T) {
	first := domain.GemEntityV2{Slug: "first", Localities: []domain.LocalityV2{{CountryCode: "RU"}}}
	second := domain.GemEntityV2{Slug: "second", Localities: []domain.LocalityV2{{CountryCode: "MG"}}}
	repo := newV2RepositoryStub(&first)
	repo.entities[second.Slug] = &second
	request := httptest.NewRequest(http.MethodGet, "/?country_code=MG", nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	NewGemEntityV2Handler(repo).List(context)
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"slug":"second"`)) || bytes.Contains(recorder.Body.Bytes(), []byte(`"slug":"first"`)) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestV2Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		repo       *v2RepositoryStub
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			repo:       newV2RepositoryStub(&domain.GemEntityV2{Slug: "test-gem"}),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found",
			repo:       newV2RepositoryStub(&domain.GemEntityV2{Slug: "other-gem"}),
			wantStatus: http.StatusNotFound,
			wantBody:   `"error":"not_found"`,
		},
		{
			name:       "repository error",
			repo:       &v2RepositoryStub{entities: map[string]*domain.GemEntityV2{}, deleteErr: errors.New("database unavailable")},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `"error":"internal_error"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
			context.Params = gin.Params{{Key: "slug", Value: "test-gem"}}

			NewGemEntityV2Handler(tt.repo).Delete(context)

			if context.Writer.Status() != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", context.Writer.Status(), tt.wantStatus, recorder.Body.String())
			}
			if tt.wantBody != "" && !bytes.Contains(recorder.Body.Bytes(), []byte(tt.wantBody)) {
				t.Fatalf("body = %q, want it to contain %q", recorder.Body.String(), tt.wantBody)
			}
			if tt.repo.deletedSlug != "test-gem" {
				t.Fatalf("DeleteV2 slug = %q, want test-gem", tt.repo.deletedSlug)
			}
		})
	}
}
