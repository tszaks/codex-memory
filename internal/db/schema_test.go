package db

import (
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
