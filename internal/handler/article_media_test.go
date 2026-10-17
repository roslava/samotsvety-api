package handler

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/imaging"
	"github.com/roslava/samotsvety-api/internal/middleware"
	"github.com/roslava/samotsvety-api/internal/storage"
)

type articleMediaStore struct {
	objects map[string][]byte
	readErr error
	listErr error
}

func (s *articleMediaStore) Upload(ctx context.Context, folder, filename, contentType string, data io.Reader, size int64) (string, error) {
	return "", errors.New("unused")
}
func (s *articleMediaStore) UploadAt(_ context.Context, key, _ string, data io.Reader, _ int64) (string, error) {
	if s.objects == nil {
		s.objects = map[string][]byte{}
	}
	b, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	s.objects[key] = b
	return "https://cdn.example/" + key, nil
}
func (s *articleMediaStore) UploadUnique(ctx context.Context, key, contentType string, data io.Reader, size int64) (string, error) {
	if _, exists := s.objects[key]; exists {
		return "", storage.ErrObjectExists
	}
	return s.UploadAt(ctx, key, contentType, data, size)
}
func (s *articleMediaStore) PublicURL(key string) string { return "https://cdn.example/" + key }
func (s *articleMediaStore) ListKeys(_ context.Context, prefix string) ([]string, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var keys []string
	for key := range s.objects {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}
func (s *articleMediaStore) ArticleKey(raw string) (string, error) {
	if !strings.HasPrefix(raw, "https://cdn.example/") || strings.ContainsAny(raw, "%?\\#") {
		return "", storage.ErrInvalidArticleURL
	}
	return strings.TrimPrefix(raw, "https://cdn.example/"), nil
}
func (s *articleMediaStore) ReadObject(_ context.Context, key string, _ int64) ([]byte, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	b, ok := s.objects[key]
	if !ok {
		return nil, storage.ErrObjectNotFound
	}
	return b, nil
}

func pngImage(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func uploadArticleMedia(t *testing.T, router *gin.Engine, kind string, img []byte, slug, slideID string, authorized bool) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("kind", kind)
	_ = w.WriteField("slug", slug)
	_ = w.WriteField("slide_id", slideID)
	f, err := w.CreateFormFile("file", "image.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write(img)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if authorized {
		req.Header.Set("X-API-Key", "test-key")
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestArticleMediaUploadAndVerification(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-key")
	gin.SetMode(gin.TestMode)
	s := &articleMediaStore{}
	r := gin.New()
	r.POST("/api/v1/media", middleware.APIKeyAuth(), NewMediaHandler(s).UploadMedia)
	id := "64ac5e2c-943d-4b5c-b942-a80deee45a23"
	good := pngImage(t, 160, 90)
	if resp := uploadArticleMedia(t, r, "article_slide", good, "one-question", id, false); resp.Code != 401 {
		t.Fatalf("missing API key: %d", resp.Code)
	}
	first := uploadArticleMedia(t, r, "article_slide", good, "one-question", id, true)
	if first.Code != 201 {
		t.Fatalf("upload: %d %s", first.Code, first.Body.String())
	}
	second := uploadArticleMedia(t, r, "article_slide", good, "one-question", id, true)
	if second.Code != 201 || second.Body.String() == first.Body.String() || len(s.objects) != 2 {
		t.Fatalf("immutable keys: %d %s", second.Code, second.Body.String())
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	v := NewArticleMediaVerifier(s)
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: result.URL}); err != nil {
		t.Fatal(err)
	}
	key, _ := s.ArticleKey(result.URL)
	goodObject := s.objects[key]
	badRatio, err := imaging.ToWebP(context.Background(), pngImage(t, 120, 90), webpQuality)
	if err != nil {
		t.Fatal(err)
	}
	s.objects[key] = badRatio
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: result.URL}); err != nil {
		t.Fatalf("4:3 slide: %v", err)
	}
	s.objects[key] = []byte("not a WebP")
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: result.URL}); !errors.Is(err, ErrInvalidArticleMedia) {
		t.Fatalf("invalid WebP: %v", err)
	}
	s.objects[key] = goodObject
	// A pre-existing image in the configured CDN can be selected without uploading it.
	cdnKey := "illustrations/existing.webp"
	s.objects[cdnKey] = goodObject
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: "https://cdn.example/" + cdnKey, Source: "cdn"}); err != nil {
		t.Fatalf("existing CDN image: %v", err)
	}
	s.objects[cdnKey] = badRatio
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: "https://cdn.example/" + cdnKey, Source: "cdn"}); err != nil {
		t.Fatalf("4:3 CDN image: %v", err)
	}
	s.objects[cdnKey] = goodObject
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: "https://evil.example/" + cdnKey, Source: "cdn"}); !errors.Is(err, ErrInvalidArticleMediaURL) {
		t.Fatalf("external CDN URL: %v", err)
	}
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: "https://cdn.example/" + cdnKey}); !errors.Is(err, ErrInvalidArticleMedia) {
		t.Fatalf("unmarked CDN image: %v", err)
	}
	for _, tc := range []struct {
		name, url, slug, id string
		readErr             error
	}{
		{"external", "https://evil.example/" + strings.TrimPrefix(result.URL, "https://cdn.example/"), "one-question", id, nil},
		{"slug", result.URL, "another", id, nil},
		{"uuid", result.URL, "one-question", "64ac5e2c-943d-4b5c-b942-a80deee45a24", nil},
		{"missing", strings.Replace(result.URL, ".webp", "f.webp", 1), "one-question", id, nil},
		{"storage down", result.URL, "one-question", id, errors.New("s3 unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.readErr = tc.readErr
			err := v.VerifySlide(context.Background(), tc.slug, domain.PostSlide{ID: tc.id, URL: tc.url})
			if tc.readErr != nil {
				if !errors.Is(err, ErrMediaVerificationUnavailable) {
					t.Fatalf("got %v", err)
				}
			} else if !errors.Is(err, ErrInvalidArticleMedia) && !errors.Is(err, ErrInvalidArticleMediaURL) {
				t.Fatalf("got %v", err)
			}
		})
	}
	s.readErr = nil
	for _, dims := range [][2]int{{120, 90}, {90, 160}} {
		resp := uploadArticleMedia(t, r, "article_slide", pngImage(t, dims[0], dims[1]), "one-question", id, true)
		if resp.Code != 201 {
			t.Fatalf("ratio %v: %d %s", dims, resp.Code, resp.Body.String())
		}
		var uploaded struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &uploaded); err != nil {
			t.Fatal(err)
		}
		if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: uploaded.URL}); err != nil {
			t.Fatalf("uploaded ratio %v: %v", dims, err)
		}
	}
	if resp := uploadArticleMedia(t, r, "article_slide", []byte("not an image"), "one-question", id, true); resp.Code != 400 {
		t.Fatalf("invalid data: %d", resp.Code)
	}
	if resp := uploadArticleMedia(t, r, "article_slide", good, "../bad", id, true); resp.Code != 400 {
		t.Fatalf("bad slug: %d", resp.Code)
	}
	if resp := uploadArticleMedia(t, r, "article_slide", good, "one-question", "../bad", true); resp.Code != 400 {
		t.Fatalf("bad UUID: %d", resp.Code)
	}
	avatar := uploadArticleMedia(t, r, "author_avatar", pngImage(t, 80, 120), "one-question", "", true)
	if avatar.Code != 201 {
		t.Fatalf("avatar: %d %s", avatar.Code, avatar.Body.String())
	}
	var avatarResult struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(avatar.Body.Bytes(), &avatarResult); err != nil {
		t.Fatal(err)
	}
	if err := v.VerifyAvatar(context.Background(), "one-question", avatarResult.URL); err != nil {
		t.Fatal(err)
	}
	if err := v.VerifyAvatar(context.Background(), "another", avatarResult.URL); err != nil {
		t.Fatalf("avatar after article rename: %v", err)
	}
	if err := v.VerifyAvatar(context.Background(), "another", strings.Replace(avatarResult.URL, "/authors/", "/slides/", 1)); !errors.Is(err, ErrArticleMediaMissing) {
		t.Fatalf("invalid avatar path: %v", err)
	}
	for _, cdnKey := range []string{
		"common_images/authors/editor.webp",
		"entities/bruneau_jasper/images/hero.webp",
		"articles/other/images/cover/01.webp",
		"articles/other/images/content/photo.webp",
	} {
		s.objects[cdnKey] = goodObject
		if err := v.VerifyAvatar(context.Background(), "one-question", s.PublicURL(cdnKey)); err != nil {
			t.Fatalf("existing avatar %s: %v", cdnKey, err)
		}
		if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: s.PublicURL(cdnKey), Source: "cdn"}); err != nil {
			t.Fatalf("existing slide %s: %v", cdnKey, err)
		}
	}
	if err := v.VerifyAvatar(context.Background(), "one-question", "https://evil.example/common_images/authors/editor.webp"); !errors.Is(err, ErrInvalidArticleMediaURL) {
		t.Fatalf("foreign URL: %v", err)
	}
	if err := v.VerifyAvatar(context.Background(), "one-question", "https://cdn.example/common_images/missing.webp"); !errors.Is(err, ErrArticleMediaMissing) {
		t.Fatalf("missing object: %v", err)
	}
	if err := v.VerifySlide(context.Background(), "one-question", domain.PostSlide{ID: id, URL: "https://cdn.example/common_images/missing.webp", Source: "cdn"}); !errors.Is(err, ErrArticleMediaMissing) {
		t.Fatalf("missing slide: %v", err)
	}
	s.readErr = storage.ErrObjectAccessDenied
	if err := v.VerifyAvatar(context.Background(), "one-question", s.PublicURL("common_images/authors/editor.webp")); !errors.Is(err, ErrMediaVerificationUnavailable) || !errors.Is(err, storage.ErrObjectAccessDenied) {
		t.Fatalf("access denied: %v", err)
	}
	s.readErr = nil
	for _, data := range s.objects {
		if _, _, err := imaging.WebPDimensions(data); err != nil {
			t.Fatalf("stored invalid webp: %v", err)
		}
	}
}

func TestArticleAvatarFromExistingCDNObject(t *testing.T) {
	key := "common_images/authors/editor.png"
	s := &articleMediaStore{objects: map[string][]byte{key: pngImage(t, 80, 120)}}
	r := postRouter(t, newTestPostRepo(), NewArticleMediaVerifier(s))
	create := func(rawURL string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{
			"slug": "one-question",
			"i18n": map[string]any{"ru": map[string]any{}, "en": map[string]any{}, "media": map[string]any{
				"slides": []any{}, "author_avatar_url": rawURL,
			}},
		})
		return postCall(r, http.MethodPost, "/api/v1/posts", string(body), "test-key", "")
	}
	missing := create(s.PublicURL("common_images/authors/missing.png"))
	if missing.Code != http.StatusBadRequest || !strings.Contains(missing.Body.String(), "media object not found") {
		t.Fatalf("missing: %d %s", missing.Code, missing.Body.String())
	}
	foreign := create("https://evil.example/" + key)
	if foreign.Code != http.StatusBadRequest || !strings.Contains(foreign.Body.String(), "invalid Lithoteka CDN URL") {
		t.Fatalf("foreign: %d %s", foreign.Code, foreign.Body.String())
	}
	s.readErr = storage.ErrObjectAccessDenied
	denied := create(s.PublicURL(key))
	if denied.Code != http.StatusServiceUnavailable || !strings.Contains(denied.Body.String(), "media_verification_unavailable") {
		t.Fatalf("access denied: %d %s", denied.Code, denied.Body.String())
	}
	s.readErr = nil
	created := create(s.PublicURL(key))
	assertStatus(t, created, http.StatusCreated)
	updated := postCall(r, http.MethodPut, "/api/v1/posts/one-question", `{"i18n":{"ru":{"title":"Кварц"}}}`, "test-key", created.Header().Get("ETag"))
	assertStatus(t, updated, http.StatusOK)
}

func TestPublishedArticleUsesStoredSlide(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-key")
	s := &articleMediaStore{}
	r := gin.New()
	r.POST("/api/v1/media", middleware.APIKeyAuth(), NewMediaHandler(s).UploadMedia)
	id := "64ac5e2c-943d-4b5c-b942-a80deee45a23"
	upload := uploadArticleMedia(t, r, "article_slide", pngImage(t, 160, 90), "one-question", id, true)
	if upload.Code != 201 {
		t.Fatalf("upload: %d %s", upload.Code, upload.Body.String())
	}
	var media struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(upload.Body.Bytes(), &media); err != nil {
		t.Fatal(err)
	}
	avatar := uploadArticleMedia(t, r, "author_avatar", pngImage(t, 80, 120), "one-question", "", true)
	if avatar.Code != 201 {
		t.Fatalf("avatar upload: %d %s", avatar.Code, avatar.Body.String())
	}
	var avatarMedia struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(avatar.Body.Bytes(), &avatarMedia); err != nil {
		t.Fatal(err)
	}
	post := map[string]any{
		"slug": "one-question", "author": "Автор", "is_published": true,
		"i18n": map[string]any{
			"ru":    map[string]string{"title": "Вопрос", "excerpt": "Лид", "content": "**Ответ**"},
			"en":    map[string]string{"title": "Question", "excerpt": "Lead", "content": "**Answer**"},
			"media": map[string]any{"slides": []any{map[string]any{"id": id, "url": media.URL, "alt": map[string]string{"ru": "Фото", "en": "Photo"}}}, "author_avatar_url": avatarMedia.URL},
		},
	}
	encoded, _ := json.Marshal(post)
	posts := postRouter(t, newTestPostRepo(), NewArticleMediaVerifier(s))
	created := postCall(posts, "POST", "/api/v1/posts", string(encoded), "test-key", "")
	assertStatus(t, created, 201)
	key, _ := s.ArticleKey(media.URL)
	stored := s.objects[key]
	s.objects[key] = []byte("not a WebP")
	update := postCall(posts, "PUT", "/api/v1/posts/one-question", `{"i18n":{"en":{"title":"Updated"}}}`, "test-key", created.Header().Get("ETag"))
	if update.Code != 400 {
		t.Fatalf("published update must reverify all slides: %d %s", update.Code, update.Body.String())
	}
	s.objects[key] = stored
	rename := postCall(posts, "PUT", "/api/v1/posts/one-question", `{"slug":"renamed-question","i18n":{"media":{"slides":[{"id":"`+id+`","url":"`+media.URL+`","source":"cdn","alt":{"ru":"Фото","en":"Photo"}}]}}}`, "test-key", created.Header().Get("ETag"))
	if rename.Code != 200 {
		t.Fatalf("rename with existing media: %d %s", rename.Code, rename.Body.String())
	}
	post["slug"] = "another"
	encoded, _ = json.Marshal(post)
	bad := postCall(posts, "POST", "/api/v1/posts", string(encoded), "test-key", "")
	if bad.Code != 400 || !strings.Contains(bad.Body.String(), "validation_error") {
		t.Fatalf("wrong slug: %d %s", bad.Code, bad.Body.String())
	}
	s.readErr = errors.New("storage unavailable")
	post["slug"] = "different"
	post["i18n"].(map[string]any)["media"].(map[string]any)["slides"].([]any)[0].(map[string]any)["url"] = strings.Replace(media.URL, "one-question", "different", 1)
	encoded, _ = json.Marshal(post)
	bad = postCall(posts, "POST", "/api/v1/posts", string(encoded), "test-key", "")
	if bad.Code != 503 {
		t.Fatalf("storage unavailable: %d %s", bad.Code, bad.Body.String())
	}
}

func TestArticleMediaFormatsOrientationSizeAndLegacyModes(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-key")
	gin.SetMode(gin.TestMode)
	s := &articleMediaStore{}
	r := gin.New()
	r.POST("/api/v1/media", middleware.APIKeyAuth(), NewMediaHandler(s).UploadMedia)
	id := "64ac5e2c-943d-4b5c-b942-a80deee45a23"
	var jpegData bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 90, 160))
	if err := jpeg.Encode(&jpegData, img, nil); err != nil {
		t.Fatal(err)
	}
	// EXIF orientation 6 turns a 90x160 pixel matrix into 160x90.
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	segment := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(exif)+2))
	oriented := append(append(append([]byte{}, jpegData.Bytes()[:2]...), segment...), exif...)
	oriented = append(oriented, jpegData.Bytes()[2:]...)
	resp := uploadArticleMedia(t, r, "article_slide", oriented, "one-question", id, true)
	if resp.Code != 201 {
		t.Fatalf("oriented JPEG: %d %s", resp.Code, resp.Body.String())
	}
	for _, data := range s.objects {
		w, h, err := imaging.WebPDimensions(data)
		if err != nil || w != 160 || h != 90 {
			t.Fatalf("oriented output: %d x %d, %v", w, h, err)
		}
	}
	var gifData bytes.Buffer
	if err := gif.Encode(&gifData, image.NewPaletted(image.Rect(0, 0, 160, 90), color.Palette{color.Black}), nil); err != nil {
		t.Fatal(err)
	}
	resp = uploadArticleMedia(t, r, "article_slide", gifData.Bytes(), "one-question", id, true)
	if resp.Code != 201 {
		t.Fatalf("GIF: %d %s", resp.Code, resp.Body.String())
	}
	var webp []byte
	for _, data := range s.objects {
		webp = data
		break
	}
	resp = uploadArticleMedia(t, r, "article_slide", webp, "one-question", id, true)
	if resp.Code != 201 {
		t.Fatalf("WebP: %d %s", resp.Code, resp.Body.String())
	}
	truncated := pngImage(t, 160, 90)
	storedBeforeInvalid := len(s.objects)
	resp = uploadArticleMedia(t, r, "article_slide", truncated[:len(truncated)-12], "one-question", id, true)
	if resp.Code != 400 || len(s.objects) != storedBeforeInvalid {
		t.Fatalf("truncated PNG: %d %s", resp.Code, resp.Body.String())
	}
	resp = uploadArticleMedia(t, r, "article_slide", bytes.Repeat([]byte("x"), maxUploadSize+1), "one-question", id, true)
	if resp.Code != 400 || len(s.objects) != storedBeforeInvalid {
		t.Fatalf("oversize: %d %s", resp.Code, resp.Body.String())
	}
	for _, kind := range []string{"hero", "cover", "block_image", "block_pair"} {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		_ = w.WriteField("kind", kind)
		_ = w.WriteField("slug", "one-question")
		_ = w.WriteField("block_index", "1")
		_ = w.WriteField("pair_index", "1")
		f, _ := w.CreatePart(textproto.MIMEHeader{"Content-Disposition": []string{`form-data; name="file"; filename="image.png"`}, "Content-Type": []string{"image/png"}})
		_, _ = f.Write(pngImage(t, 160, 90))
		_ = w.Close()
		req := httptest.NewRequest("POST", "/api/v1/media", &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("X-API-Key", "test-key")
		out := httptest.NewRecorder()
		r.ServeHTTP(out, req)
		if out.Code != 201 {
			t.Fatalf("legacy %s: %d %s", kind, out.Code, out.Body.String())
		}
	}
}
