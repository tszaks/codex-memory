package analysis

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tszaks/codex-memory/internal/db"
)

type StructuralLink struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

var goSymbolRegex = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)

func StructuralLinks(store *db.Store, targetPath string, limit int) ([]StructuralLink, error) {
	normalized, err := normalizeRepoPath(store.RepoRoot, targetPath)
	if err != nil {
		return nil, err
	}

	files, err := repoFiles(store.RepoRoot)
	if err != nil {
		return nil, err
	}
	targetAbs := filepath.Join(store.RepoRoot, filepath.FromSlash(normalized))
	targetContent, _ := osReadFile(targetAbs)

	out := make([]StructuralLink, 0)
	targetDir := filepath.ToSlash(filepath.Dir(normalized))
	targetName := filepath.Base(normalized)
	targetStem := fileStem(targetName)
	targetIsTest := isTestFile(targetName)

	for _, candidate := range files {
		if candidate == normalized {
			continue
		}

		candidateName := filepath.Base(candidate)
		candidateDir := filepath.ToSlash(filepath.Dir(candidate))
		candidateStem := fileStem(candidateName)
		candidateIsTest := isTestFile(candidateName)
		candidateAbs := filepath.Join(store.RepoRoot, filepath.FromSlash(candidate))
		candidateContent, _ := osReadFile(candidateAbs)

		switch {
		case targetStem != "" && candidateStem == targetStem && targetIsTest != candidateIsTest:
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "test-pair",
				Reason: "Shares the same source stem as the target file.",
			})
		case targetStem != "" && candidateStem == targetStem:
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "same-stem",
				Reason: "Shares the same file stem as the target file.",
			})
		case strings.HasSuffix(targetName, ".go") && strings.HasSuffix(candidateName, ".go") && referencesGoFile(targetContent, candidateStem):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "go-symbol",
				Reason: "Target file references a symbol that matches this Go file's stem.",
			})
		case strings.HasSuffix(candidateName, ".go") && strings.HasSuffix(targetName, ".go") && referencesGoFile(candidateContent, targetStem):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "go-dependent",
				Reason: "This Go file appears to reference the target file's symbol stem.",
			})
		case candidateDir == targetDir && candidateIsTest:
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "same-dir-test",
				Reason: "Test file in the same directory as the target file.",
			})
		case candidateDir == targetDir && filepath.Ext(candidateName) == filepath.Ext(targetName):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "same-dir",
				Reason: "File in the same directory with the same extension.",
			})
		}
	}

	return uniqueStructuralLinks(out, limit), nil
}

func SuggestedTestCommands(store *db.Store, targetPath string, limit int) ([]string, error) {
	normalized, err := normalizeRepoPath(store.RepoRoot, targetPath)
	if err != nil {
		return nil, err
	}

	tests, err := SuggestedTests(store, normalized, limit)
	if err != nil {
		return nil, err
	}

	commands := inferredTestCommands(store.RepoRoot, normalized, tests)

	return uniqueStrings(commands, limit), nil
}

func SuggestedTests(store *db.Store, targetPath string, limit int) ([]string, error) {
	links, err := StructuralLinks(store, targetPath, limit*3)
	if err != nil {
		return nil, err
	}

	tests := make([]string, 0, limit)
	for _, link := range links {
		if link.Kind != "test-pair" && link.Kind != "same-dir-test" {
			continue
		}
		if !isTestFile(filepath.Base(link.Path)) {
			continue
		}
		tests = append(tests, link.Path)
	}

	if len(tests) == 0 && isTestFile(filepath.Base(targetPath)) {
		tests = append(tests, filepath.ToSlash(filepath.Clean(targetPath)))
	}

	return uniqueStrings(tests, limit), nil
}

func BlastRadius(store *db.Store, targetPath string, limit int) ([]string, error) {
	normalized, err := normalizeRepoPath(store.RepoRoot, targetPath)
	if err != nil {
		return nil, err
	}

	neighbors, err := Neighbors(store, normalized, limit)
	if err != nil {
		return nil, err
	}
	links, err := StructuralLinks(store, normalized, limit)
	if err != nil {
		return nil, err
	}
	tests, err := SuggestedTests(store, normalized, limit)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, limit*3)
	for _, neighbor := range neighbors {
		out = append(out, neighbor.Path)
	}
	for _, link := range links {
		out = append(out, link.Path)
	}
	out = append(out, tests...)

	return uniqueStrings(out, limit), nil
}

func repoFiles(repoRoot string) ([]string, error) {
	out := make([]string, 0)
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			switch name {
			case ".git", ".codex-memory":
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(out)
	return out, nil
}

func fileStem(name string) string {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	stem = strings.TrimSuffix(stem, "_test")
	stem = strings.TrimSuffix(stem, ".test")
	return stem
}

func isTestFile(name string) bool {
	return strings.HasSuffix(name, "_test.go") ||
		strings.HasSuffix(name, "_spec.rb") ||
		strings.HasSuffix(name, ".test.js") ||
		strings.HasSuffix(name, ".test.ts") ||
		strings.HasSuffix(name, ".test.tsx") ||
		strings.HasSuffix(name, ".spec.js") ||
		strings.HasSuffix(name, ".spec.ts") ||
		strings.HasSuffix(name, ".spec.tsx") ||
		strings.HasSuffix(name, "_test.py") ||
		strings.HasPrefix(name, "test_")
}

func uniqueStructuralLinks(links []StructuralLink, limit int) []StructuralLink {
	seen := make(map[string]struct{})
	out := make([]StructuralLink, 0, len(links))
	for _, link := range links {
		key := link.Kind + "::" + link.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, link)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func uniqueStrings(values []string, limit int) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func referencesGoFile(content []byte, stem string) bool {
	if len(content) == 0 || stem == "" {
		return false
	}
	for _, match := range goSymbolRegex.FindAllStringSubmatch(string(content), -1) {
		if len(match) > 1 && strings.EqualFold(match[1], stem) {
			return true
		}
	}
	return false
}

func hasGoTests(paths []string) bool {
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			return true
		}
	}
	return false
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func inferredTestCommands(repoRoot, normalized string, tests []string) []string {
	commands := make([]string, 0, 3)

	if strings.HasSuffix(normalized, ".go") || hasGoTests(tests) {
		packageDir := filepath.ToSlash(filepath.Dir(normalized))
		if packageDir == "." {
			commands = append(commands, "go test .")
		} else {
			commands = append(commands, "go test ./"+packageDir)
		}
		commands = append(commands, "go test ./...")
		return commands
	}

	jsTest := firstMatchingPath(tests, func(path string) bool {
		return strings.HasSuffix(path, ".test.js") ||
			strings.HasSuffix(path, ".test.ts") ||
			strings.HasSuffix(path, ".test.tsx") ||
			strings.HasSuffix(path, ".spec.js") ||
			strings.HasSuffix(path, ".spec.ts") ||
			strings.HasSuffix(path, ".spec.tsx")
	})
	if jsTest != "" {
		packageManager := inferPackageManager(repoRoot)
		if packageManager != "" {
			commands = append(commands, packageManager+" test -- "+jsTest)
			commands = append(commands, packageManager+" test")
		}
		return commands
	}

	pyTest := firstMatchingPath(tests, func(path string) bool {
		return strings.HasSuffix(path, "_test.py") || strings.HasPrefix(filepath.Base(path), "test_")
	})
	if pyTest != "" {
		commands = append(commands, "pytest "+pyTest, "pytest")
		return commands
	}

	rubyTest := firstMatchingPath(tests, func(path string) bool {
		return strings.HasSuffix(path, "_spec.rb")
	})
	if rubyTest != "" {
		commands = append(commands, "bundle exec rspec "+rubyTest, "bundle exec rspec")
	}

	return commands
}

func firstMatchingPath(paths []string, predicate func(string) bool) string {
	for _, path := range paths {
		if predicate(path) {
			return path
		}
	}
	return ""
}

func inferPackageManager(repoRoot string) string {
	switch {
	case fileExists(filepath.Join(repoRoot, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(repoRoot, "yarn.lock")):
		return "yarn"
	case fileExists(filepath.Join(repoRoot, "bun.lock")), fileExists(filepath.Join(repoRoot, "bun.lockb")):
		return "bun"
	case fileExists(filepath.Join(repoRoot, "package-lock.json")), fileExists(filepath.Join(repoRoot, "package.json")):
		return "npm"
	default:
		return ""
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
