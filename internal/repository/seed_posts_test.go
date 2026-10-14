package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/roslava/samotsvety-api/internal/domain"
)

type seedRecord struct {
	I18n        []byte
	Type        domain.PostType
	GemSlugs    []string
	Tags        []string
	PublishedAt *time.Time
	Published   bool
	Author      string
}
type fakeSeedExecer struct{ rows map[string]seedRecord }

func (f *fakeSeedExecer) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	if !strings.Contains(query, "ON CONFLICT (slug) DO NOTHING") || strings.Contains(query, "DO UPDATE") {
		return nil, fmt.Errorf("unsafe seed SQL")
	}
	slug := args[0].(string)
	if _, exists := f.rows[slug]; exists {
		return driver.RowsAffected(0), nil
	}
	var date *time.Time
	if args[5] != nil {
		date = args[5].(*time.Time)
	}
	f.rows[slug] = seedRecord{
		I18n: append([]byte(nil), args[2].([]byte)...), Type: args[1].(domain.PostType),
		GemSlugs: append([]string(nil), args[3].(pq.StringArray)...), Tags: append([]string(nil), args[4].(pq.StringArray)...),
		PublishedAt: date, Published: args[6].(bool), Author: args[7].(string),
	}
	return driver.RowsAffected(1), nil
}

func seedFixture(slug string) *domain.Post {
	return &domain.Post{Slug: slug, Type: domain.PostTypeGuide, Author: "Seed author",
		I18n:     json.RawMessage(`{"ru":{"title":"Вопрос","excerpt":"Лид","content":"Original **Markdown**"},"en":{"title":"Question","excerpt":"Lead","content":"Body"},"media":{"schema_version":2,"slides":[{"id":"64ac5e2c-943d-4b5c-b942-a80deee45a23","url":"https://cdn.example/media.webp","caption":{"ru":"Подпись","en":"Caption"},"alt":{"ru":"Фото","en":"Photo"}}],"author_avatar_url":"https://cdn.example/avatar.webp"}}`),
		GemSlugs: []string{"malachite"}, Tags: []string{"editorial"}}
}

func TestSeedPostsInsertOnlyPreservesEditsAndOldRows(t *testing.T) {
	ctx := context.Background()
	exec := &fakeSeedExecer{rows: map[string]seedRecord{}}
	post := seedFixture("question")
	first, err := insertPreparedSeedPosts(ctx, exec, []*domain.Post{post})
	if err != nil || first != (SeedPostsReport{Inserted: 1}) {
		t.Fatalf("first: %+v %v", first, err)
	}
	initial := exec.rows["question"]
	if !strings.Contains(string(initial.I18n), "Original **Markdown**") || !strings.Contains(string(initial.I18n), "author_avatar_url") || !strings.Contains(string(initial.I18n), `"caption"`) || !strings.Contains(string(initial.I18n), `"alt"`) || initial.Published {
		t.Fatalf("insertion lost fields: %+v", initial)
	}
	var edited map[string]any
	if err := json.Unmarshal(initial.I18n, &edited); err != nil {
		t.Fatal(err)
	}
	edited["ru"].(map[string]any)["content"] = "Edited **Markdown**"
	changed, _ := json.Marshal(edited)
	initial.I18n = changed
	initial.Published = true
	initial.Author = "Editor"
	initial.Tags = []string{"edited"}
	now := time.Now()
	initial.PublishedAt = &now
	exec.rows["question"] = initial
	second, err := insertPreparedSeedPosts(ctx, exec, []*domain.Post{seedFixture("question")})
	if err != nil || second != (SeedPostsReport{Skipped: 1}) {
		t.Fatalf("repeat: %+v %v", second, err)
	}
	if got := exec.rows["question"]; !reflect.DeepEqual(got, initial) {
		t.Fatalf("repeat overwrote edited row: %+v", got)
	}
	legacy := seedRecord{I18n: []byte(`{"ru":{"content":"old"}}`), Author: "Old author", Published: true}
	exec.rows["legacy"] = legacy
	mixed, err := insertPreparedSeedPosts(ctx, exec, []*domain.Post{seedFixture("legacy"), seedFixture("new")})
	if err != nil || mixed != (SeedPostsReport{Inserted: 1, Skipped: 1}) {
		t.Fatalf("mixed: %+v %v", mixed, err)
	}
	if !reflect.DeepEqual(exec.rows["legacy"], legacy) {
		t.Fatal("legacy row changed")
	}
}

func TestSeedPostsLoadReportsInvalidFilesBeforeInserts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"slug":"bad"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.json"), []byte(`{"slug":"good"}`), 0600); err != nil {
		t.Fatal(err)
	}
	prepare := func(_ context.Context, b []byte) (*domain.Post, error) {
		var value struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(b, &value); err != nil {
			return nil, err
		}
		if value.Slug == "bad" {
			return nil, fmt.Errorf("i18n.ru: object is required")
		}
		return seedFixture(value.Slug), nil
	}
	posts, report, err := loadSeedPosts(context.Background(), dir, prepare)
	if len(posts) != 0 || report.Invalid != 1 || err == nil || !strings.Contains(err.Error(), "bad.json") || !strings.Contains(err.Error(), "i18n.ru") {
		t.Fatalf("validation report: posts=%d report=%+v err=%v", len(posts), report, err)
	}
	if _, _, err := loadSeedPosts(context.Background(), dir, nil); err == nil {
		t.Fatal("nil preparer accepted")
	}
}
