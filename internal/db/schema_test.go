package db

import (
	"errors"
	"os"
	"testing"
)

func TestSchemaInitializes(t *testing.T) {
	repo := t.TempDir()
	store, err := OpenPath(repo, t.TempDir()+"/test.sqlite")
	if err != nil {
		t.Fatalf("OpenPath failed: %v", err)
	}
	defer store.Close()

	tables := []string{"repos", "files", "commits", "file_commits", "cochange_edges", "decision_notes"}
	for _, table := range tables {
		row := store.DB().QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table)
		var name string
		if err := row.Scan(&name); err != nil {
			t.Fatalf("expected table %s to exist: %v", table, err)
		}
	}
}

func TestSchemaMigratesExistingFilesTable(t *testing.T) {
	repo := t.TempDir()
	dbPath := t.TempDir() + "/test.sqlite"

	store, err := OpenPath(repo, dbPath)
	if err != nil {
		t.Fatalf("OpenPath failed: %v", err)
	}

	if _, err := store.DB().Exec(`DROP TABLE files`); err != nil {
		t.Fatalf("drop files table: %v", err)
	}
	if _, err := store.DB().Exec(`
CREATE TABLE files (
  repo_id INTEGER NOT NULL,
  path TEXT NOT NULL,
  extension TEXT NOT NULL,
  churn_score INTEGER NOT NULL DEFAULT 0,
  recent_touch_count INTEGER NOT NULL DEFAULT 0,
  exists_on_disk INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (repo_id, path)
)`); err != nil {
		t.Fatalf("create legacy files table: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sqlite db to exist: %v", err)
	}

	migrated, err := OpenPath(repo, dbPath)
	if err != nil {
		t.Fatalf("re-open migrated db: %v", err)
	}
	defer migrated.Close()

	columns := []string{"author_count", "last_touched_at"}
	for _, column := range columns {
		row := migrated.DB().QueryRow(`SELECT 1 FROM pragma_table_info('files') WHERE name = ?`, column)
		var found int
		if err := row.Scan(&found); err != nil {
			t.Fatalf("expected files.%s to exist after migration: %v", column, err)
		}
	}
}

func TestSchemaMigrationBackfillsNewFileSignals(t *testing.T) {
	repo := t.TempDir()
	dbPath := t.TempDir() + "/test.sqlite"

	store, err := OpenPath(repo, dbPath)
	if err != nil {
		t.Fatalf("OpenPath failed: %v", err)
	}

	if _, err := store.DB().Exec(`DROP TABLE files`); err != nil {
		t.Fatalf("drop files table: %v", err)
	}
	if _, err := store.DB().Exec(`
CREATE TABLE files (
  repo_id INTEGER NOT NULL,
  path TEXT NOT NULL,
  extension TEXT NOT NULL,
  churn_score INTEGER NOT NULL DEFAULT 0,
  recent_touch_count INTEGER NOT NULL DEFAULT 0,
  exists_on_disk INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (repo_id, path)
)`); err != nil {
		t.Fatalf("create legacy files table: %v", err)
	}

	if _, err := store.DB().Exec(`INSERT INTO repos (id, root, branch, last_indexed_commit, indexed_at) VALUES (1, ?, 'main', 'abc123', '2026-03-13T12:00:00Z')`, repo); err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO files (repo_id, path, extension, churn_score, recent_touch_count, exists_on_disk) VALUES (1, 'main.go', 'go', 4, 2, 1)`); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO commits (repo_id, sha, author_name, author_email, committed_at, subject, body) VALUES
		(1, 'a1', 'One', 'one@example.com', '2026-03-10T12:00:00Z', 'first', ''),
		(1, 'b2', 'Two', 'two@example.com', '2026-03-12T12:00:00Z', 'second', '')`); err != nil {
		t.Fatalf("insert commits: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO file_commits (repo_id, file_path, commit_sha, committed_at) VALUES
		(1, 'main.go', 'a1', '2026-03-10T12:00:00Z'),
		(1, 'main.go', 'b2', '2026-03-12T12:00:00Z')`); err != nil {
		t.Fatalf("insert file commits: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	migrated, err := OpenPath(repo, dbPath)
	if err != nil {
		t.Fatalf("re-open migrated db: %v", err)
	}
	defer migrated.Close()

	row := migrated.DB().QueryRow(`SELECT author_count, last_touched_at FROM files WHERE repo_id = 1 AND path = 'main.go'`)
	var authorCount int
	var lastTouchedAt string
	if err := row.Scan(&authorCount, &lastTouchedAt); err != nil {
		t.Fatalf("read migrated file stats: %v", err)
	}

	if authorCount != 2 {
		t.Fatalf("expected author_count=2 after backfill, got %d", authorCount)
	}
	if lastTouchedAt != "2026-03-12T12:00:00Z" {
		t.Fatalf("expected last_touched_at to backfill from newest file commit, got %q", lastTouchedAt)
	}
}

func TestRepoReturnsHelpfulErrorWhenNotIndexedYet(t *testing.T) {
	repo := t.TempDir()
	store, err := OpenPath(repo, t.TempDir()+"/test.sqlite")
	if err != nil {
		t.Fatalf("OpenPath failed: %v", err)
	}
	defer store.Close()

	_, err = store.Repo()
	if !errors.Is(err, ErrRepoNotIndexed) {
		t.Fatalf("expected ErrRepoNotIndexed, got %v", err)
	}
}
