package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func indexRepoHelper(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	run(t, repo, "git", "init", "-b", "main")
	run(t, repo, "git", "config", "user.name", "Test User")
	run(t, repo, "git", "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "README.md"), "# test\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "docs: add readme")

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "key: value\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "feat: add app")

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "key: next\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "fix: update app logic")

	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(output))
	}
}
