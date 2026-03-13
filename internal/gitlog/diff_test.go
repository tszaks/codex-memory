package gitlog

import (
	"path/filepath"
	"testing"
)

func TestChangedFilesBetween(t *testing.T) {
	repo := t.TempDir()
	run(t, repo, "git", "init", "-b", "main")
	run(t, repo, "git", "config", "user.name", "Test User")
	run(t, repo, "git", "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "feat: add main")

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(repo, "main_test.go"), "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {\n\tmain()\n}\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "test: add coverage")

	files, err := ChangedFilesBetween(repo, "HEAD~1", "HEAD")
	if err != nil {
		t.Fatalf("ChangedFilesBetween failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 changed files, got %#v", files)
	}
}
