package storage

import "testing"

func TestArticleKeyCanonicalURL(t *testing.T) {
	s := &yandexS3Storage{publicURL: "https://cdn.example/media"}
	good := "https://cdn.example/media/articles/a/slides/64ac5e2c-943d-4b5c-b942-a80deee45a23/0123456789abcdef0123456789abcdef.webp"
	if got, err := s.ArticleKey(good); err != nil || got != "articles/a/slides/64ac5e2c-943d-4b5c-b942-a80deee45a23/0123456789abcdef0123456789abcdef.webp" {
		t.Fatalf("key %q: %v", got, err)
	}
	for _, raw := range []string{
		"https://evil.example/media/articles/a/slides/x.webp",
		"https://cdn.example/media2/articles/a/slides/x.webp",
		"https://cdn.example/media/articles/a/%2e%2e/x.webp",
		"https://cdn.example/media/articles/a/../x.webp",
		"https://cdn.example/media/articles/a//x.webp",
		"https://cdn.example/media/articles/a/x.webp?override=1",
		"https://cdn.example/media/articles/a/x.webp#fragment",
		"https://user@cdn.example/media/articles/a/x.webp",
		"http://cdn.example/media/articles/a/x.webp",
		"https://cdn.example/media/articles/a\\x.webp",
	} {
		if key, err := s.ArticleKey(raw); err == nil {
			t.Errorf("accepted %q as %q", raw, key)
		}
	}
}
