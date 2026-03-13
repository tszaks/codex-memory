package index

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitlogTestRepoHelper(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	run(t, repo, "git", "init", "-b", "main")
	run(t, repo, "git", "config", "user.name", "Test User")
	run(t, repo, "git", "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "README.md"), "# test\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "docs: add readme")

	writeFile(t, filepath.Join(repo, "helper.go"), "package main\n\nfunc helper() string { return \"ok\" }\n")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { _ = helper() }\n")
	writeFile(t, filepath.Join(repo, "main_test.go"), "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "key: value\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "feat: add app")

	writeFile(t, filepath.Join(repo, "helper.go"), "package main\n\nfunc helper() string { return \"next\" }\n")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { println(helper()) }\n")
	writeFile(t, filepath.Join(repo, "main_test.go"), "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {\n\tmain()\n}\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "fix: update app")

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
