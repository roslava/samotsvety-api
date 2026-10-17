package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestReadObjectUsesConfiguredBucketAndDistinguishesS3Errors(t *testing.T) {
	status := http.StatusOK
	client := s3.New(s3.Options{
		Region: "ru-central1", BaseEndpoint: aws.String("https://storage.yandexcloud.net"), UsePathStyle: true,
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
		HTTPClient: &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet || req.URL.Path != "/lithoteka-cdn/common_images/authors/editor.webp" {
				t.Errorf("unexpected S3 request: %s %s", req.Method, req.URL.Path)
			}
			body := "image"
			if status != http.StatusOK {
				code := "NoSuchKey"
				if status == http.StatusForbidden {
					code = "AccessDenied"
				}
				body = "<Error><Code>" + code + "</Code></Error>"
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/xml"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
		})},
	})
	store := &yandexS3Storage{client: client, bucket: "lithoteka-cdn"}
	key := "common_images/authors/editor.webp"
	data, err := store.ReadObject(context.Background(), key, 20)
	if err != nil || string(data) != "image" {
		t.Fatalf("read: %q %v", data, err)
	}
	status = http.StatusNotFound
	if _, err := store.ReadObject(context.Background(), key, 20); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("missing: %v", err)
	}
	status = http.StatusForbidden
	if _, err := store.ReadObject(context.Background(), key, 20); !errors.Is(err, ErrObjectAccessDenied) {
		t.Fatalf("access denied: %v", err)
	}
}

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

func TestLithotekaArticleKeyAliases(t *testing.T) {
	s := &yandexS3Storage{bucket: "lithoteka-cdn", publicURL: "https://storage.yandexcloud.net/lithoteka-cdn"}
	if key, err := s.ArticleKey("https://lithoteka-cdn.storage.yandexcloud.net/common_images/quartz%20example.webp"); err != nil || key != "common_images/quartz example.webp" {
		t.Fatalf("escaped file name: %q %v", key, err)
	}
	for _, raw := range []string{
		"https://storage.yandexcloud.net/lithoteka-cdn/common_images/quartz.webp",
		"https://lithoteka-cdn.storage.yandexcloud.net/common_images/quartz.webp",
	} {
		key, err := s.ArticleKey(raw)
		if err != nil || key != "common_images/quartz.webp" {
			t.Fatalf("%q: %q %v", raw, key, err)
		}
	}
	for _, raw := range []string{
		"https://other.storage.yandexcloud.net/common_images/quartz.webp",
		"https://storage.yandexcloud.net/other/common_images/quartz.webp",
		"https://lithoteka-cdn.storage.yandexcloud.net/../quartz.webp",
	} {
		if _, err := s.ArticleKey(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}
