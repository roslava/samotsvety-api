package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/roslava/samotsvety-api/internal/domain"
)

type rejectingMedia struct{}

func (rejectingMedia) VerifySlide(context.Context, string, domain.PostSlide) error {
	return errors.New("old bucket unavailable")
}
func (rejectingMedia) VerifyAvatar(context.Context, string, string) error {
	return errors.New("old bucket unavailable")
}

func TestCoverURLSequencesAndLegacyFallback(t *testing.T) {
	repo := newTestPostRepo()
	store := &articleMediaStore{objects: map[string][]byte{
		"articles/one-question/images/cover/01.webp": {1},
		"articles/one-question/images/cover/02.webp": {2},
	}}
	r := postRouter(t, repo, verifiedSlides{})
	r.GET("/api/v1/posts/:slug/cover-frames", NewMediaHandler(store, repo).ListArticleCover)
	created := postCall(r, "POST", "/api/v1/posts", fullPayload(), "test-key", "")
	assertStatus(t, created, http.StatusCreated)
	path := "/api/v1/posts/one-question/cover-frames"
	legacy := responseJSON(t, postCall(r, "GET", path, "", "", ""))
	if frames := legacy["frames"].([]any); len(frames) != 2 {
		t.Fatalf("legacy frames: %v", frames)
	}
	all := responseJSON(t, postCall(r, "GET", path+"?all=1", "", "test-key", ""))
	if all["managed"] != false || len(all["sequences"].(map[string]any)["default"].([]any)) != 2 {
		t.Fatalf("legacy sequences: %v", all)
	}

	base := "https://lithoteka-cdn.storage.yandexcloud.net/"
	first := base + "common_images/quartz.webp"
	second := "https://storage.yandexcloud.net/lithoteka-cdn/articles/quartz/images/cover/02.webp"
	russian := base + "entities/quartz/images/hero.webp"
	patch := map[string]any{"i18n": map[string]any{"media": map[string]any{"cover_frames": map[string]any{
		"default": []string{first, second}, "ru": []string{russian}, "en": []string{},
	}}}}
	encoded, _ := json.Marshal(patch)
	saved := postCall(r, "PUT", "/api/v1/posts/one-question", string(encoded), "test-key", created.Header().Get("ETag"))
	assertStatus(t, saved, http.StatusOK)
	if len(store.objects) != 2 {
		t.Fatal("saving references modified storage")
	}
	for _, tc := range []struct{ lang, want string }{{"", first}, {"ru", russian}, {"en", first}} {
		data := responseJSON(t, postCall(r, "GET", path+"?lang="+tc.lang, "", "", ""))
		frames := data["frames"].([]any)
		if frames[0].(map[string]any)["url"] != tc.want {
			t.Fatalf("%s: %v", tc.lang, frames)
		}
		if tc.lang == "" && frames[1].(map[string]any)["url"] != second {
			t.Fatalf("order changed: %v", frames)
		}
	}
	all = responseJSON(t, postCall(r, "GET", path+"?all=1", "", "test-key", ""))
	if all["managed"] != true || len(all["sequences"].(map[string]any)["en"].([]any)) != 0 {
		t.Fatalf("managed sequences: %v", all)
	}

	patch = map[string]any{"i18n": map[string]any{"media": map[string]any{"cover_frames": map[string]any{
		"default": []string{second}, "ru": []string{}, "en": []string{},
	}}}}
	encoded, _ = json.Marshal(patch)
	removed := postCall(r, "PUT", "/api/v1/posts/one-question", string(encoded), "test-key", saved.Header().Get("ETag"))
	assertStatus(t, removed, http.StatusOK)
	if len(store.objects) != 2 {
		t.Fatal("removing a reference deleted a file")
	}
	patch = map[string]any{"i18n": map[string]any{"media": map[string]any{"cover_frames": map[string]any{
		"default": []string{}, "ru": []string{}, "en": []string{},
	}}}}
	encoded, _ = json.Marshal(patch)
	empty := postCall(r, "PUT", "/api/v1/posts/one-question", string(encoded), "test-key", removed.Header().Get("ETag"))
	assertStatus(t, empty, http.StatusOK)
	read := responseJSON(t, postCall(r, "GET", path, "", "", ""))
	if len(read["frames"].([]any)) != 0 {
		t.Fatalf("old frames reappeared: %v", read)
	}
	reloaded := responseJSON(t, postCall(r, "GET", path+"?all=1", "", "test-key", ""))
	if reloaded["managed"] != true {
		t.Fatalf("empty managed list lost: %v", reloaded)
	}
}

func TestCoverURLValidation(t *testing.T) {
	for _, raw := range []string{"", "http://lithoteka-cdn.storage.yandexcloud.net/a.webp", "https://evil.example/a.webp", "https://storage.yandexcloud.net/other/a.webp", "https://lithoteka-cdn.storage.yandexcloud.net/../a.webp"} {
		if validLithotekaImageURL(raw) {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"https://lithoteka-cdn.storage.yandexcloud.net/common_images/a.webp", "https://storage.yandexcloud.net/lithoteka-cdn/entities/a/images/hero.webp"} {
		if !validLithotekaImageURL(raw) {
			t.Errorf("rejected %q", raw)
		}
	}
	if !validLithotekaImageURL("https://lithoteka-cdn.storage.yandexcloud.net/common_images/quartz%20example.jpg") {
		t.Fatal("rejected escaped JPEG URL")
	}
}

func TestArticleMarkdownImageURLSurvivesCoverUpdate(t *testing.T) {
	repo := newTestPostRepo()
	r := postRouter(t, repo, verifiedSlides{})
	image := "https://lithoteka-cdn.storage.yandexcloud.net/common_images/quartz.webp"
	created := postCall(r, "POST", "/api/v1/posts", `{"slug":"quartz","i18n":{"ru":{"content":"![Кварц](`+image+`)"},"en":{},"media":{"slides":[]}}}`, "test-key", "")
	assertStatus(t, created, http.StatusCreated)
	patch := `{"i18n":{"media":{"cover_frames":{"default":["` + image + `"],"ru":[],"en":[]}}}}`
	updated := postCall(r, "PUT", "/api/v1/posts/quartz", patch, "test-key", created.Header().Get("ETag"))
	assertStatus(t, updated, http.StatusOK)
	post := responseJSON(t, postCall(r, "GET", "/api/v1/posts/quartz", "", "test-key", ""))
	content := post["i18n"].(map[string]any)["ru"].(map[string]any)["content"]
	if content != "![Кварц]("+image+")" {
		t.Fatalf("content URL changed: %v", content)
	}
}

func TestUnchangedLegacyArticleURLSurvivesCoverUpdate(t *testing.T) {
	repo := newTestPostRepo()
	oldURL := "https://storage.yandexcloud.net/samotsvety-cdn/articles/old/slides/64ac5e2c-943d-4b5c-b942-a80deee45a23/old.webp"
	created := postCall(postRouter(t, repo, verifiedSlides{}), "POST", "/api/v1/posts", `{"slug":"old","i18n":{"ru":{},"en":{},"media":{"slides":[{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a23","url":"`+oldURL+`"}]}}}`, "test-key", "")
	assertStatus(t, created, http.StatusCreated)
	updated := postCall(postRouter(t, repo, rejectingMedia{}), "PUT", "/api/v1/posts/old", `{"i18n":{"media":{"cover_frames":{"default":[],"ru":[],"en":[]}}}}`, "test-key", created.Header().Get("ETag"))
	assertStatus(t, updated, http.StatusOK)
	post := responseJSON(t, updated)
	slides := post["i18n"].(map[string]any)["media"].(map[string]any)["slides"].([]any)
	if slides[0].(map[string]any)["url"] != oldURL {
		t.Fatal("legacy URL changed")
	}
}
