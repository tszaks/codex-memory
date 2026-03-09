package index

import (
	"testing"
)

func TestIndexerRun(t *testing.T) {
	repo := gitlogTestRepo(t)
	store, err := OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	indexer := New(store)
	result, err := indexer.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.CommitCount == 0 || result.FileCount == 0 {
		t.Fatalf("expected indexed data, got %+v", result)
	}
}

func gitlogTestRepo(t *testing.T) string {
	t.Helper()
	return gitlogTestRepoHelper(t)
}
