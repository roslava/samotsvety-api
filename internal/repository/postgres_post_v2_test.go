package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/roslava/samotsvety-api/internal/domain"
)

func TestPostRevisionChangesAndKeepsUnknownJSON(t *testing.T) {
	post := &domain.Post{I18n: json.RawMessage(`{"ru":{"title":"A","extra":{"keep":true}},"en":{},"media":{"schema_version":2,"slides":[],"future":{"value":1}}}`)}
	if err := assignRevision(post); err != nil {
		t.Fatal(err)
	}
	first := post.Revision
	if first == "" || !isPostV2(post.I18n) {
		t.Fatalf("invalid initial revision or version: %s", post.I18n)
	}
	if err := assignRevision(post); err != nil {
		t.Fatal(err)
	}
	if first == post.Revision {
		t.Fatal("revision did not change")
	}
	var root map[string]any
	if err := json.Unmarshal(post.I18n, &root); err != nil {
		t.Fatal(err)
	}
	if root["ru"].(map[string]any)["extra"].(map[string]any)["keep"] != true ||
		root["media"].(map[string]any)["future"].(map[string]any)["value"] != float64(1) {
		t.Fatal("unknown JSON keys were lost")
	}
	if isPostV2(json.RawMessage(`{"ru":{},"en":{}}`)) {
		t.Fatal("old row identified as V2")
	}
}

// This test is intentionally opt-in: it writes only to a dedicated test DB and
// cleans up its row. It never runs against an implicitly configured database.
func TestPostgresV2CreateAndConcurrentUpdate(t *testing.T) {
	dsn := os.Getenv("SAMOTSVETY_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("SAMOTSVETY_ALLOW_DB_WRITES_FOR_TESTS") != "1" {
		t.Skip("dedicated writable PostgreSQL test database was not explicitly enabled")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skip("dedicated PostgreSQL test database is unavailable")
	}
	defer db.Close()
	ctx := context.Background()
	repo := NewPostgresPostRepository(db)
	unique, err := newPostRevision()
	if err != nil {
		t.Fatal(err)
	}
	post := &domain.Post{
		Slug: "article-v2-test-" + unique, Type: domain.PostTypeGuide,
		I18n:     json.RawMessage(`{"ru":{},"en":{},"media":{"schema_version":2,"slides":[]}}`),
		GemSlugs: []string{}, Tags: []string{},
	}
	if err := repo.Create(ctx, post); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM posts WHERE id=$1`, post.ID)
	if !regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`).MatchString(post.ID) {
		t.Fatalf("database did not return a UUID: %q", post.ID)
	}
	if _, err := repo.GetBySlug(ctx, post.Slug, false); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("public draft read: %v", err)
	}
	if _, err := repo.GetBySlug(ctx, post.Slug, true); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	var group sync.WaitGroup
	results := make(chan error, 2)
	for _, name := range []string{"First", "Second"} {
		group.Add(1)
		go func(name string) {
			defer group.Done()
			<-start
			_, err := repo.Update(ctx, post.Slug, post.Revision, func(current *domain.Post) error { current.Author = name; return nil })
			results <- err
		}(name)
	}
	close(start)
	group.Wait()
	close(results)
	var success, stale int
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPostPrecondition) {
			stale++
		} else {
			t.Errorf("unexpected update error: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("concurrent updates: success=%d stale=%d", success, stale)
	}
}
