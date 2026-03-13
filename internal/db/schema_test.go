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
