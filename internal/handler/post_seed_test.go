package handler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/roslava/samotsvety-api/internal/domain"
)

func TestPrepareArticleV2SeedUsesPostRules(t *testing.T) {
	file := filepath.Join("..", "..", "seeds", "posts", "example.json")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	post, err := PrepareArticleV2Seed(context.Background(), data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if post.IsPublished || post.PublishedAt != nil || post.Type != domain.PostTypeGuide {
		t.Fatalf("invalid draft: %+v", post)
	}
	var root struct {
		Media struct {
			SchemaVersion int   `json:"schema_version"`
			Slides        []any `json:"slides"`
		} `json:"media"`
	}
	if err := json.Unmarshal(post.I18n, &root); err != nil {
		t.Fatal(err)
	}
	if root.Media.SchemaVersion != 2 || root.Media.Slides == nil || len(root.Media.Slides) != 0 {
		t.Fatalf("incorrect V2 media: %+v", root.Media)
	}
	for _, tc := range []struct{ name, body, path string }{
		{"legacy cover", `{"slug":"one-question","i18n":{"ru":{},"en":{}},"cover_image":"x"}`, "cover_image"},
		{"managed version", `{"slug":"one-question","i18n":{"ru":{},"en":{},"media":{"schema_version":2}}}`, "i18n.media.schema_version"},
		{"invalid translation", `{"slug":"one-question","i18n":{"ru":{},"en":[],"media":{"slides":[]}}}`, "i18n.en"},
		{"missing media", `{"slug":"one-question","i18n":{"ru":{},"en":{}} ,"is_published":true}`, "i18n.media"},
		{"published no verifier", fullPayload(), "media verification"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PrepareArticleV2Seed(context.Background(), []byte(tc.body), nil)
			if err == nil || !strings.Contains(err.Error(), tc.path) {
				t.Fatalf("got %v, want %q", err, tc.path)
			}
		})
	}
	post, err = PrepareArticleV2Seed(context.Background(), []byte(fullPayload()), verifiedSlides{})
	if err != nil || post.PublishedAt == nil {
		t.Fatalf("published seed: %+v %v", post, err)
	}
}
