package handler

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/repository"
	"net/http"
	"net/http/httptest"
	"testing"
)

type catalogStub struct {
	*v2RepositoryStub
	query repository.V2CatalogQuery
	calls int
}

func (s *catalogStub) QueryV2(_ context.Context, q repository.V2CatalogQuery) (*repository.V2CatalogPage, error) {
	s.calls++
	s.query = q
	return &repository.V2CatalogPage{Data: []domain.GemEntityV2{}, Total: 71, Page: q.Page, Limit: q.Limit, BaseColors: []string{"blue"}}, nil
}
func TestV2CatalogPaginationDoesNotLoadFullCatalog(t *testing.T) {
	repo := &catalogStub{v2RepositoryStub: &v2RepositoryStub{}}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/api/v2/gem-entities?page=2&limit=20&lang=en&q=blue&rarity=rare&base_color=blue&country_code=RU&letter=O", nil)
	NewGemEntityV2Handler(repo).List(c)
	if rec.Code != http.StatusOK || repo.calls != 1 || repo.listCalls != 0 {
		t.Fatalf("status=%d pageCalls=%d listCalls=%d", rec.Code, repo.calls, repo.listCalls)
	}
	if repo.query.Page != 2 || repo.query.Lang != "en" || repo.query.Query != "blue" || repo.query.CountryCode != "RU" || repo.query.Letter != "O" || repo.query.Rarity != "rare" || repo.query.BaseColor != "blue" {
		t.Fatalf("query: %+v", repo.query)
	}
	var response repository.V2CatalogPage
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Total != 71 || response.Page != 2 || response.Limit != 20 || response.Data == nil {
		t.Fatalf("response: %+v", response)
	}
}
func TestV2CatalogRejectsInvalidPaginationBeforeRepositoryCall(t *testing.T) {
	for _, query := range []string{"page=0", "page=abc", "limit=101", "page=1&lang=de", "page=1&country_code=ru", "page=1&letter=AB"} {
		repo := &catalogStub{v2RepositoryStub: &v2RepositoryStub{}}
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("GET", "/?"+query, nil)
		NewGemEntityV2Handler(repo).List(c)
		if rec.Code != http.StatusBadRequest || repo.calls != 0 || repo.listCalls != 0 {
			t.Fatalf("%s: status=%d calls=%d/%d", query, rec.Code, repo.calls, repo.listCalls)
		}
	}
}
