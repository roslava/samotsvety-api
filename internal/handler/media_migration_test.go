package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/roslava/samotsvety-api/internal/storage"
)

func migrationUpload(t *testing.T, r *gin.Engine, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="image.png"`)
	header.Set("Content-Type", "image/png")
	f, err := w.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(pngImage(t, 2, 2)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func TestMigratedMediaPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &articleMediaStore{objects: map[string][]byte{}}
	r := gin.New()
	h := NewMediaHandler(s)
	r.POST("/api/v1/media", h.UploadMedia)
	r.GET("/api/v1/posts/:slug/cover-frames", h.ListArticleCover)
	base := map[string]string{"slug": "bruneau-jasper", "storage_key": "bruneau_jasper"}
	for _, kind := range []string{"hero", "thumbnail", "gallery", "gallery"} {
		fields := map[string]string{"slug": base["slug"], "storage_key": base["storage_key"], "kind": kind}
		resp := migrationUpload(t, r, fields)
		if resp.Code != 201 {
			t.Fatalf("%s: %d %s", kind, resp.Code, resp.Body.String())
		}
	}
	for _, path := range []string{"hero.webp", "thumbnail.webp", "gallery/bruneau_jasper01.webp", "gallery/bruneau_jasper02.webp"} {
		key := "entities/bruneau_jasper/images/" + path
		if _, ok := s.objects[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
	if len(s.objects) != 4 {
		t.Fatalf("unexpected objects: %v", s.objects)
	}
	for _, kind := range []string{"cover", "cover", "cover"} {
		resp := migrationUpload(t, r, map[string]string{"kind": kind, "slug": "example"})
		if resp.Code != 201 {
			t.Fatalf("cover: %d %s", resp.Code, resp.Body.String())
		}
	}
	if resp := migrationUpload(t, r, map[string]string{"kind": "cover", "slug": "example", "lang": "ru"}); resp.Code != 201 {
		t.Fatalf("localized cover: %d", resp.Code)
	}
	for _, tc := range []struct{ kind, extra, want string }{
		{"block_image", "", "articles/example/images/content/image-01.webp"},
		{"block_pair", "1", "articles/example/images/content/image-01-1-en.webp"},
	} {
		fields := map[string]string{"kind": tc.kind, "slug": "example", "block_index": "1"}
		if tc.extra != "" {
			fields["pair_index"] = tc.extra
			fields["lang"] = "en"
		}
		resp := migrationUpload(t, r, fields)
		if resp.Code != 201 {
			t.Fatalf("%s: %d %s", tc.kind, resp.Code, resp.Body.String())
		}
		if _, ok := s.objects[tc.want]; !ok {
			t.Errorf("missing %s", tc.want)
		}
	}
	for _, tc := range []struct {
		lang  string
		count int
	}{{"", 3}, {"ru", 1}, {"en", 3}} {
		path := "/api/v1/posts/example/cover-frames?lang=" + tc.lang
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, httptest.NewRequest("GET", path, nil))
		if resp.Code != 200 {
			t.Fatalf("list: %d %s", resp.Code, resp.Body.String())
		}
		var result struct {
			Frames []struct {
				Index int    `json:"index"`
				URL   string `json:"url"`
			} `json:"frames"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Frames) != tc.count {
			t.Fatalf("%s: %v", tc.lang, result.Frames)
		}
		for i, frame := range result.Frames {
			if frame.Index != i+1 {
				t.Fatalf("wrong order: %v", result.Frames)
			}
		}
	}
}

func TestCoverFramesOver99AndUnsafePaths(t *testing.T) {
	s := &articleMediaStore{objects: map[string][]byte{}}
	for i := 1; i <= 100; i++ {
		s.objects[fmt.Sprintf("articles/example/images/cover/%02d.webp", i)] = []byte("old")
	}
	s.objects["articles/example/images/cover/ru/01.webp"] = []byte("old")
	r := gin.New()
	h := NewMediaHandler(s)
	r.POST("/api/v1/media", h.UploadMedia)
	r.GET("/api/v1/posts/:slug/cover-frames", h.ListArticleCover)
	resp := migrationUpload(t, r, map[string]string{"kind": "cover", "slug": "example"})
	if resp.Code != 201 || !strings.Contains(resp.Body.String(), "101.webp") {
		t.Fatalf("next frame: %d %s", resp.Code, resp.Body.String())
	}
	if string(s.objects["articles/example/images/cover/100.webp"]) != "old" {
		t.Fatal("existing frame overwritten")
	}
	listed := httptest.NewRecorder()
	r.ServeHTTP(listed, httptest.NewRequest("GET", "/api/v1/posts/example/cover-frames", nil))
	var sequence struct {
		Frames []struct {
			Index int `json:"index"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &sequence); err != nil {
		t.Fatal(err)
	}
	if len(sequence.Frames) != 101 || sequence.Frames[99].Index != 100 || sequence.Frames[100].Index != 101 {
		t.Fatalf("numeric order after 99: %v", sequence.Frames)
	}
	s.listErr = fmt.Errorf("storage offline")
	failed := httptest.NewRecorder()
	r.ServeHTTP(failed, httptest.NewRequest("GET", "/api/v1/posts/example/cover-frames", nil))
	if failed.Code != 500 {
		t.Fatalf("storage error: %d", failed.Code)
	}
	for _, value := range []string{"../bad", "images/hero.webp", "entities/x", "/hero.webp"} {
		if _, err := storage.EntityImageKey("safe_key", value); err == nil {
			t.Errorf("accepted unsafe path %q", value)
		}
	}
	if resp := migrationUpload(t, r, map[string]string{"kind": "hero", "slug": "example", "storage_key": "../bad"}); resp.Code != 400 {
		t.Fatalf("unsafe storage key: %d", resp.Code)
	}
}
