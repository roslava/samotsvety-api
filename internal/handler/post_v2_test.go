package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/middleware"
	"github.com/roslava/samotsvety-api/internal/repository"
)

type testPostRepo struct {
	mu          sync.Mutex
	posts       map[string]*domain.Post
	seq         int
	lastFilters repository.PostFilterParams
}

func newTestPostRepo() *testPostRepo { return &testPostRepo{posts: map[string]*domain.Post{}} }
func clonePost(post *domain.Post) *domain.Post {
	copy := *post
	copy.I18n = append(json.RawMessage(nil), post.I18n...)
	copy.GemSlugs = append([]string(nil), post.GemSlugs...)
	copy.Tags = append([]string(nil), post.Tags...)
	return &copy
}
func (r *testPostRepo) nextRevision(post *domain.Post) {
	r.seq++
	post.Revision = fmt.Sprintf("00000000-0000-4000-8000-%012d", r.seq)
	var root map[string]any
	_ = json.Unmarshal(post.I18n, &root)
	root["media"].(map[string]any)["revision"] = post.Revision
	post.I18n, _ = json.Marshal(root)
	post.UpdatedAt = time.Unix(int64(r.seq), 0).UTC()
}
func (r *testPostRepo) GetBySlug(_ context.Context, slug string, admin bool) (*domain.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	post := r.posts[slug]
	if post == nil || !testIsV2(post) || (!admin && !post.IsPublished) {
		return nil, repository.ErrPostNotFound
	}
	return clonePost(post), nil
}
func testIsV2(post *domain.Post) bool {
	var root struct {
		Media struct {
			SchemaVersion int `json:"schema_version"`
		} `json:"media"`
	}
	_ = json.Unmarshal(post.I18n, &root)
	return root.Media.SchemaVersion == 2
}
func (r *testPostRepo) List(_ context.Context, filters repository.PostFilterParams) ([]domain.Post, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastFilters = filters
	items := []domain.Post{}
	for _, post := range r.posts {
		if !testIsV2(post) || (!filters.IncludeDrafts && !post.IsPublished) {
			continue
		}
		if filters.Type != "" && string(post.Type) != filters.Type {
			continue
		}
		items = append(items, *clonePost(post))
	}
	return items, len(items), nil
}
func (r *testPostRepo) Search(_ context.Context, q, _ string, _, _ int) ([]domain.Post, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := []domain.Post{}
	for _, post := range r.posts {
		if !testIsV2(post) || !post.IsPublished {
			continue
		}
		if strings.Contains(strings.ToLower(string(post.I18n)), strings.ToLower(q)) {
			items = append(items, *clonePost(post))
		}
	}
	return items, len(items), nil
}
func (r *testPostRepo) Create(_ context.Context, post *domain.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.posts[post.Slug] != nil {
		return repository.ErrPostSlugConflict
	}
	r.seq++
	post.ID = fmt.Sprintf("11111111-1111-4111-8111-%012d", r.seq)
	r.nextRevision(post)
	r.posts[post.Slug] = clonePost(post)
	return nil
}
func (r *testPostRepo) Update(_ context.Context, slug, revision string, mutate func(*domain.Post) error) (*domain.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored := r.posts[slug]
	if stored == nil {
		return nil, repository.ErrPostNotFound
	}
	if !testIsV2(stored) {
		return nil, repository.ErrUnsupportedArticleFormat
	}
	if stored.Revision != revision {
		return nil, repository.ErrPostPrecondition
	}
	post := clonePost(stored)
	if err := mutate(post); err != nil {
		return nil, err
	}
	if post.Slug != slug && r.posts[post.Slug] != nil {
		return nil, repository.ErrPostSlugConflict
	}
	r.nextRevision(post)
	delete(r.posts, slug)
	r.posts[post.Slug] = clonePost(post)
	return post, nil
}
func (r *testPostRepo) Delete(_ context.Context, slug, revision string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	post := r.posts[slug]
	if post == nil {
		return repository.ErrPostNotFound
	}
	if !testIsV2(post) {
		return repository.ErrUnsupportedArticleFormat
	}
	if post.Revision != revision {
		return repository.ErrPostPrecondition
	}
	delete(r.posts, slug)
	return nil
}
func (r *testPostRepo) GetByGemSlug(context.Context, string, string, int) ([]domain.Post, error) {
	return nil, nil
}

type verifiedSlides struct{}

func (verifiedSlides) VerifySlide(context.Context, string, domain.PostSlide) error { return nil }
func (verifiedSlides) VerifyAvatar(context.Context, string, string) error          { return nil }

func postRouter(t *testing.T, repo repository.PostRepository, verifier SlideMediaVerifier) *gin.Engine {
	t.Helper()
	t.Setenv("ADMIN_API_KEY", "test-key")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewPostHandler(repo, verifier)
	posts := router.Group("/api/v1/posts")
	posts.GET("", h.ListPosts)
	posts.GET("/:slug", h.GetPost)
	admin := posts.Group("")
	admin.Use(middleware.APIKeyAuth())
	admin.POST("", h.CreatePost)
	admin.PUT("/:slug", h.UpdatePost)
	admin.DELETE("/:slug", h.DeletePost)
	router.GET("/api/v1/search/posts", h.SearchPosts)
	return router
}
func postCall(router http.Handler, method, path, body, key, tag string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		request.Header.Set("X-API-Key", key)
	}
	if tag != "" {
		request.Header.Set("If-Match", tag)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
func draftPayload() string {
	return `{"slug":"one-question","i18n":{"ru":{"title":"Вопрос?","excerpt":"","content":""},"en":{"title":"","excerpt":"","content":""},"media":{"slides":[]},"extra":{"keep":true}},"is_published":false}`
}
func fullPayload() string {
	return `{"slug":"one-question","type":"guide","i18n":{"ru":{"title":"Вопрос?","excerpt":"Лид","content":"**Текст**"},"en":{"title":"Question?","excerpt":"Lead","content":"**Body**"},"media":{"slides":[{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a23","url":"https://cdn.example/one.webp","caption":{"ru":"Подпись","en":"Caption"},"alt":{"ru":"Фото","en":"Photo"}},{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a24","url":"https://cdn.example/two.webp","caption":{"ru":"Вторая","en":"Second"},"alt":{"ru":"Второе фото","en":"Second photo"}}]}},"author":"Автор","is_published":true}`
}
func responseJSON(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response JSON: %v: %s", err, response.Body.String())
	}
	return body
}
func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status %d, want %d: %s", response.Code, want, response.Body.String())
	}
}

func TestArticleV2DraftAndRealID(t *testing.T) {
	repo := newTestPostRepo()
	router := postRouter(t, repo, nil)
	created := postCall(router, "POST", "/api/v1/posts", draftPayload(), "test-key", "")
	assertStatus(t, created, 201)
	body := responseJSON(t, created)
	if !postUUIDPattern.MatchString(body["id"].(string)) {
		t.Fatalf("invalid id: %v", body["id"])
	}
	if body["type"] != "guide" {
		t.Fatalf("default type: %v", body["type"])
	}
	media := body["i18n"].(map[string]any)["media"].(map[string]any)
	if media["schema_version"] != float64(2) || created.Header().Get("ETag") == "" {
		t.Fatal("missing schema version or ETag")
	}
	if _, leaked := media["revision"]; leaked {
		t.Fatal("internal revision leaked into response")
	}
	assertStatus(t, postCall(router, "GET", "/api/v1/posts/one-question", "", "", ""), 404)
	assertStatus(t, postCall(router, "GET", "/api/v1/posts/one-question", "", "test-key", ""), 200)
}

func TestArticleV2PublicationAndLocalizedSlides(t *testing.T) {
	repo := newTestPostRepo()
	router := postRouter(t, repo, verifiedSlides{})
	created := postCall(router, "POST", "/api/v1/posts", fullPayload(), "test-key", "")
	assertStatus(t, created, 201)
	body := responseJSON(t, created)
	if body["published_at"] == nil {
		t.Fatal("published_at missing")
	}
	slides := body["i18n"].(map[string]any)["media"].(map[string]any)["slides"].([]any)
	if slides[0].(map[string]any)["caption"].(map[string]any)["en"] != "Caption" || slides[0].(map[string]any)["alt"].(map[string]any)["ru"] != "Фото" {
		t.Fatal("localized slide text lost")
	}
	assertStatus(t, postCall(router, "GET", "/api/v1/posts/one-question", "", "", ""), 200)
	for _, query := range []string{"Вопрос", "Лид", "**Body**"} {
		search := postCall(router, "GET", "/api/v1/search/posts?q="+url.QueryEscape(query), "", "", "")
		assertStatus(t, search, 200)
		if responseJSON(t, search)["total"] != float64(1) {
			t.Fatalf("search missed %q", query)
		}
	}
	firstDate := body["published_at"]
	edited := postCall(router, "PUT", "/api/v1/posts/one-question", `{"i18n":{"en":{"title":"Revised question?"}}}`, "test-key", created.Header().Get("ETag"))
	assertStatus(t, edited, 200)
	if responseJSON(t, edited)["published_at"] != firstDate {
		t.Fatal("first publication date changed on edit")
	}
}

func TestArticleV2RejectsIncompletePublication(t *testing.T) {
	for _, tc := range []struct{ name, body, path string }{
		{"empty EN", strings.Replace(fullPayload(), `"content":"**Body**"`, `"content":""`, 1), "i18n.en.content"},
		{"no slides", strings.Replace(fullPayload(), `"slides":[{`, `"slides":[{`, 1), ""},
	} {
		if tc.name == "no slides" {
			tc.body = strings.Replace(fullPayload(), `"slides":[{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a23","url":"https://cdn.example/one.webp","caption":{"ru":"Подпись","en":"Caption"},"alt":{"ru":"Фото","en":"Photo"}},{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a24","url":"https://cdn.example/two.webp","caption":{"ru":"Вторая","en":"Second"},"alt":{"ru":"Второе фото","en":"Second photo"}}]`, `"slides":[]`, 1)
			tc.path = "i18n.media.slides"
		}
		t.Run(tc.name, func(t *testing.T) {
			router := postRouter(t, newTestPostRepo(), verifiedSlides{})
			response := postCall(router, "POST", "/api/v1/posts", tc.body, "test-key", "")
			assertStatus(t, response, 400)
			if responseJSON(t, response)["error"] != "validation_error" || !strings.Contains(response.Body.String(), tc.path) {
				t.Fatalf("missing path %s: %s", tc.path, response.Body.String())
			}
		})
	}
}

func TestArticleV2PartialUpdateReorderAndClear(t *testing.T) {
	repo := newTestPostRepo()
	router := postRouter(t, repo, verifiedSlides{})
	created := postCall(router, "POST", "/api/v1/posts", draftPayload(), "test-key", "")
	assertStatus(t, created, 201)
	tag := created.Header().Get("ETag")
	update := postCall(router, "PUT", "/api/v1/posts/one-question", `{"i18n":{"en":{"title":"Updated"}}}`, "test-key", tag)
	assertStatus(t, update, 200)
	body := responseJSON(t, update)
	root := body["i18n"].(map[string]any)
	if root["ru"].(map[string]any)["title"] != "Вопрос?" || root["extra"].(map[string]any)["keep"] != true {
		t.Fatal("partial update lost RU or unknown key")
	}
	if len(root["media"].(map[string]any)["slides"].([]any)) != 0 {
		t.Fatal("slides changed")
	}
	// Put two slides in a draft, then reorder and explicitly clear them.
	first := `{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a23","url":"https://cdn.example/one.webp"}`
	second := `{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a24","url":"https://cdn.example/two.webp"}`
	put := func(slides string, tag string) *httptest.ResponseRecorder {
		return postCall(router, "PUT", "/api/v1/posts/one-question", `{"i18n":{"media":{"slides":`+slides+`}}}`, "test-key", tag)
	}
	ordered := put("["+first+","+second+"]", update.Header().Get("ETag"))
	assertStatus(t, ordered, 200)
	reordered := put("["+second+","+first+"]", ordered.Header().Get("ETag"))
	assertStatus(t, reordered, 200)
	array := responseJSON(t, reordered)["i18n"].(map[string]any)["media"].(map[string]any)["slides"].([]any)
	if array[0].(map[string]any)["id"] != "64ac5e2c-943d-4b5c-b942-a80deee45a24" {
		t.Fatal("slide order did not change")
	}
	cleared := put("[]", reordered.Header().Get("ETag"))
	assertStatus(t, cleared, 200)
	if len(responseJSON(t, cleared)["i18n"].(map[string]any)["media"].(map[string]any)["slides"].([]any)) != 0 {
		t.Fatal("slides not cleared")
	}
}

func TestArticleV2ManagedVersionAndETag(t *testing.T) {
	repo := newTestPostRepo()
	router := postRouter(t, repo, verifiedSlides{})
	bad := strings.Replace(draftPayload(), `"media":{"slides":[]}`, `"media":{"schema_version":2,"slides":[]}`, 1)
	response := postCall(router, "POST", "/api/v1/posts", bad, "test-key", "")
	assertStatus(t, response, 400)
	created := postCall(router, "POST", "/api/v1/posts", draftPayload(), "test-key", "")
	assertStatus(t, created, 201)
	tag := created.Header().Get("ETag")
	assertStatus(t, postCall(router, "PUT", "/api/v1/posts/one-question", `{}`, "test-key", ""), 428)
	assertStatus(t, postCall(router, "DELETE", "/api/v1/posts/one-question", "", "test-key", ""), 428)
	assertStatus(t, postCall(router, "PUT", "/api/v1/posts/one-question", `{}`, "test-key", `"stale"`), 412)
	response = postCall(router, "PUT", "/api/v1/posts/one-question", `{"i18n":{"media":{"schema_version":2}}}`, "test-key", tag)
	assertStatus(t, response, 400)
	response = postCall(router, "PUT", "/api/v1/posts/one-question", `{"author":"Editor"}`, "test-key", tag)
	assertStatus(t, response, 200)
	if response.Header().Get("ETag") == tag {
		t.Fatal("ETag did not change")
	}
	assertStatus(t, postCall(router, "DELETE", "/api/v1/posts/one-question", "", "test-key", tag), 412)
}

func TestArticleV2ConcurrentEdits(t *testing.T) {
	repo := newTestPostRepo()
	router := postRouter(t, repo, verifiedSlides{})
	created := postCall(router, "POST", "/api/v1/posts", draftPayload(), "test-key", "")
	assertStatus(t, created, 201)
	tag := created.Header().Get("ETag")
	var group sync.WaitGroup
	codes := make(chan int, 2)
	for _, suffix := range []string{"23", "24"} {
		group.Add(1)
		go func(suffix string) {
			defer group.Done()
			body := `{"i18n":{"media":{"slides":[{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a` + suffix + `","url":"https://cdn.example/` + suffix + `.webp"}]}}}`
			codes <- postCall(router, "PUT", "/api/v1/posts/one-question", body, "test-key", tag).Code
		}(suffix)
	}
	group.Wait()
	close(codes)
	seen := map[int]int{}
	for code := range codes {
		seen[code]++
	}
	if seen[200] != 1 || seen[412] != 1 {
		t.Fatalf("expected one winner and one stale edit, got %v", seen)
	}
	final := postCall(router, "GET", "/api/v1/posts/one-question", "", "test-key", "")
	assertStatus(t, final, 200)
	if slides := responseJSON(t, final)["i18n"].(map[string]any)["media"].(map[string]any)["slides"].([]any); len(slides) != 1 {
		t.Fatalf("concurrent slide replacements were combined: %v", slides)
	}
}

func TestArticleV2PublicScopeOldRowsAndMediaGate(t *testing.T) {
	repo := newTestPostRepo()
	router := postRouter(t, repo, nil)
	created := postCall(router, "POST", "/api/v1/posts", draftPayload(), "test-key", "")
	assertStatus(t, created, 201)
	// Old row is deliberately retained but hidden and not mutable by the V2 API.
	repo.mu.Lock()
	repo.posts["old-article"] = &domain.Post{Slug: "old-article", I18n: json.RawMessage(`{"ru":{"title":"Old"}}`), Revision: "old"}
	repo.mu.Unlock()
	public := postCall(router, "GET", "/api/v1/posts?published=false&include_drafts=true", "", "", "")
	assertStatus(t, public, 200)
	if responseJSON(t, public)["total"] != float64(0) || repo.lastFilters.IncludeDrafts {
		t.Fatal("public list exposed a draft")
	}
	admin := postCall(router, "GET", "/api/v1/posts", "", "test-key", "")
	assertStatus(t, admin, 200)
	if responseJSON(t, admin)["total"] != float64(1) {
		t.Fatal("admin list does not include V2 draft")
	}
	assertStatus(t, postCall(router, "GET", "/api/v1/posts/old-article", "", "", ""), 404)
	assertStatus(t, postCall(router, "GET", "/api/v1/posts/old-article", "", "test-key", ""), 404)
	assertStatus(t, postCall(router, "PUT", "/api/v1/posts/old-article", `{}`, "test-key", `"old"`), 409)
	assertStatus(t, postCall(router, "DELETE", "/api/v1/posts/old-article", "", "test-key", `"old"`), 409)
	assertStatus(t, postCall(router, "GET", "/api/v1/search/posts?q=Old", "", "", ""), 200)
	if responseJSON(t, postCall(router, "GET", "/api/v1/search/posts?q=Old", "", "", ""))["total"] != float64(0) {
		t.Fatal("old article leaked through search")
	}
	// A syntactically valid URL cannot pass publication until a real verifier is wired.
	full := postCall(router, "POST", "/api/v1/posts", strings.Replace(fullPayload(), "one-question", "another-question", 1), "test-key", "")
	assertStatus(t, full, 503)
}

func TestArticleV2DeleteWithCurrentETag(t *testing.T) {
	repo := newTestPostRepo()
	router := postRouter(t, repo, verifiedSlides{})
	created := postCall(router, "POST", "/api/v1/posts", draftPayload(), "test-key", "")
	assertStatus(t, created, 201)
	deleted := postCall(router, "DELETE", "/api/v1/posts/one-question", "", "test-key", created.Header().Get("ETag"))
	assertStatus(t, deleted, 204)
	assertStatus(t, postCall(router, "GET", "/api/v1/posts/one-question", "", "test-key", ""), 404)
}
