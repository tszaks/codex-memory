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
	writeFile(t, filepath.Join(repo, "go.mod"), "module example.com/testrepo\n\ngo 1.26.0\n")
	writeFile(t, filepath.Join(repo, "package.json"), "{\n  \"scripts\": {\n    \"test\": \"vitest run\",\n    \"test:unit\": \"vitest run\",\n    \"lint\": \"eslint .\",\n    \"typecheck\": \"tsc --noEmit\",\n    \"build\": \"vite build\"\n  }\n}\n")
	writeFile(t, filepath.Join(repo, "package-lock.json"), "{}\n")
	writeFile(t, filepath.Join(repo, "tsconfig.json"), "{\n  \"compilerOptions\": {\n    \"baseUrl\": \".\",\n    \"paths\": {\n      \"@/*\": [\"web/*\"]\n    }\n  }\n}\n")
	writeFile(t, filepath.Join(repo, "pyproject.toml"), "[tool.pytest.ini_options]\npythonpath = [\".\"]\n\n[tool.ruff]\nline-length = 100\n\n[tool.mypy]\npython_version = \"3.12\"\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "docs: add readme")

	writeFile(t, filepath.Join(repo, "helper.go"), "package main\n\nfunc helper() string { return \"ok\" }\n")
	writeFile(t, filepath.Join(repo, "internalpkg", "helper", "helper.go"), "package helper\n\nfunc Help() string { return \"ok\" }\n")
	writeFile(t, filepath.Join(repo, "cli", "app.go"), "package cli\n\nimport \"example.com/testrepo/internalpkg/helper\"\n\nfunc App() string { return helper.Help() }\n")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { _ = helper() }\n")
	writeFile(t, filepath.Join(repo, "main_test.go"), "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n")
	writeFile(t, filepath.Join(repo, "web", "session.ts"), "export function session() { return 'ok' }\n")
	writeFile(t, filepath.Join(repo, "web", "app.ts"), "import { session } from './session'\n\nexport function app() { return session() }\n")
	writeFile(t, filepath.Join(repo, "web", "alias_app.ts"), "import { session } from '@/session'\n\nexport function aliasApp() { return session() }\n")
	writeFile(t, filepath.Join(repo, "web", "session.test.ts"), "import { session } from './session'\n\ntest('session', () => { expect(session()).toBe('ok') })\n")
	writeFile(t, filepath.Join(repo, "pkg", "__init__.py"), "")
	writeFile(t, filepath.Join(repo, "pkg", "helper.py"), "def helper():\n    return 'ok'\n")
	writeFile(t, filepath.Join(repo, "pkg", "app.py"), "from .helper import helper\n\ndef app():\n    return helper()\n")
	writeFile(t, filepath.Join(repo, "pkg", "test_app.py"), "from .app import app\n\ndef test_app():\n    assert app() == 'ok'\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "key: value\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "feat: add app")

	run(t, repo, "git", "config", "user.name", "Pairing User")
	run(t, repo, "git", "config", "user.email", "pairing@example.com")
	writeFile(t, filepath.Join(repo, "helper.go"), "package main\n\nfunc helper() string { return \"next\" }\n")
	writeFile(t, filepath.Join(repo, "internalpkg", "helper", "helper.go"), "package helper\n\nfunc Help() string { return \"next\" }\n")
	writeFile(t, filepath.Join(repo, "cli", "app.go"), "package cli\n\nimport \"example.com/testrepo/internalpkg/helper\"\n\nfunc App() string { return helper.Help() + \"!\" }\n")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { println(helper()) }\n")
	writeFile(t, filepath.Join(repo, "main_test.go"), "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {\n\tmain()\n}\n")
	writeFile(t, filepath.Join(repo, "web", "session.ts"), "export function session() { return 'next' }\n")
	writeFile(t, filepath.Join(repo, "web", "app.ts"), "import { session } from './session'\n\nexport function app() { return session() + '!' }\n")
	writeFile(t, filepath.Join(repo, "pkg", "helper.py"), "def helper():\n    return 'next'\n")
	writeFile(t, filepath.Join(repo, "pkg", "app.py"), "from .helper import helper\n\ndef app():\n    return helper() + '!'\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "key: next\n")
	run(t, repo, "git", "add", ".")
	run(t, repo, "git", "commit", "-m", "fix: update app logic")

	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
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
