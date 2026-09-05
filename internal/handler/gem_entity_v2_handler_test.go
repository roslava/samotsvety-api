package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
)

type v2RepositoryStub struct {
	entities map[string]*domain.GemEntityV2
}

func newV2RepositoryStub(entity *domain.GemEntityV2) *v2RepositoryStub {
	return &v2RepositoryStub{entities: map[string]*domain.GemEntityV2{entity.Slug: entity}}
}
func (r *v2RepositoryStub) GetV2BySlug(_ context.Context, slug string) (*domain.GemEntityV2, error) {
	entity, ok := r.entities[slug]
	if !ok {
		return nil, errV2NotFound{}
	}
	copy := *entity
	return &copy, nil
}
func (r *v2RepositoryStub) ListV2(_ context.Context) ([]domain.GemEntityV2, error) {
	result := make([]domain.GemEntityV2, 0, len(r.entities))
	for _, entity := range r.entities {
		result = append(result, *entity)
	}
	return result, nil
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
