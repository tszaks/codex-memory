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

type VerificationPlan struct {
	Fast []string `json:"fast"`
	Safe []string `json:"safe"`
	Full []string `json:"full"`
}

var goSymbolRegex = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
var goImportRegex = regexp.MustCompile(`(?m)^\s*(?:"([^"]+)"|import\s+"([^"]+)")`)
var jsImportRegex = regexp.MustCompile(`(?m)(?:import|export)[^'"\n]*from\s+['"]([^'"]+)['"]|require\(\s*['"]([^'"]+)['"]\s*\)`)
var pyImportRegex = regexp.MustCompile(`(?m)^\s*(?:from\s+([.\w]+)\s+import|import\s+([.\w]+))`)

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
	goModulePath := readGoModulePath(store.RepoRoot)

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
		case referencesGoImport(normalized, targetContent, candidate, goModulePath):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "go-import",
				Reason: "Target file imports this Go package from the same repo.",
			})
		case strings.HasSuffix(candidateName, ".go") && strings.HasSuffix(targetName, ".go") && referencesGoFile(candidateContent, targetStem):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "go-dependent",
				Reason: "This Go file appears to reference the target file's symbol stem.",
			})
		case referencesGoImport(candidate, candidateContent, normalized, goModulePath):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "go-package-dependent",
				Reason: "This Go file imports the target package from the same repo.",
			})
		case referencesJSImport(normalized, targetContent, candidate):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "js-import",
				Reason: "Target file imports this JS/TS module with a relative path.",
			})
		case referencesJSImport(candidate, candidateContent, normalized):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "js-dependent",
				Reason: "This JS/TS file imports the target module with a relative path.",
			})
		case referencesPyImport(normalized, targetContent, candidate):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "py-import",
				Reason: "Target file imports this Python module with a local import path.",
			})
		case referencesPyImport(candidate, candidateContent, normalized):
			out = append(out, StructuralLink{
				Path:   candidate,
				Kind:   "py-dependent",
				Reason: "This Python file imports the target module with a local import path.",
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
	plan, err := SuggestedVerificationPlan(store, targetPath)
	if err != nil {
		return nil, err
	}
	commands := make([]string, 0, len(plan.Fast)+len(plan.Safe)+len(plan.Full))
	commands = append(commands, plan.Fast...)
	commands = append(commands, plan.Safe...)
	commands = append(commands, plan.Full...)
	return uniqueStrings(commands, limit), nil
}

func SuggestedVerificationPlan(store *db.Store, targetPath string) (VerificationPlan, error) {
	normalized, err := normalizeRepoPath(store.RepoRoot, targetPath)
	if err != nil {
		return VerificationPlan{}, err
	}

	tests, err := SuggestedTests(store, normalized, 8)
	if err != nil {
		return VerificationPlan{}, err
	}

	return inferredVerificationPlan(store.RepoRoot, normalized, tests), nil
}

func SuggestedTests(store *db.Store, targetPath string, limit int) ([]string, error) {
	links, err := StructuralLinks(store, targetPath, limit*3)
	if err != nil {
		return nil, err
	}

	tests := make([]string, 0, limit)
	for _, link := range links {
		if !isSuggestedTestLink(link) {
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

func referencesGoImport(sourcePath string, content []byte, candidatePath, modulePath string) bool {
	if len(content) == 0 || modulePath == "" || !strings.HasSuffix(sourcePath, ".go") || !strings.HasSuffix(candidatePath, ".go") {
		return false
	}
	candidateDir := filepath.ToSlash(filepath.Dir(candidatePath))
	if candidateDir == "." {
		candidateDir = ""
	}
	for _, match := range goImportRegex.FindAllStringSubmatch(string(content), -1) {
		if len(match) < 3 {
			continue
		}
		importPath := strings.TrimSpace(match[1])
		if importPath == "" {
			importPath = strings.TrimSpace(match[2])
		}
		if importPath == modulePath && candidateDir == "" {
			return true
		}
		if candidateDir != "" && importPath == modulePath+"/"+candidateDir {
			return true
		}
	}
	return false
}

func referencesJSImport(sourcePath string, content []byte, candidatePath string) bool {
	if len(content) == 0 || !isJSImportFile(sourcePath) || !isJSImportFile(candidatePath) {
		return false
	}

	sourceDir := filepath.ToSlash(filepath.Dir(sourcePath))
	for _, match := range jsImportRegex.FindAllStringSubmatch(string(content), -1) {
		spec := ""
		if len(match) > 1 && match[1] != "" {
			spec = match[1]
		} else if len(match) > 2 {
			spec = match[2]
		}
		if !strings.HasPrefix(spec, ".") {
			continue
		}
		for _, resolved := range resolveJSImportCandidates(sourceDir, spec) {
			if resolved == candidatePath {
				return true
			}
		}
	}
	return false
}

func referencesPyImport(sourcePath string, content []byte, candidatePath string) bool {
	if len(content) == 0 || !isPythonFile(sourcePath) || !isPythonFile(candidatePath) {
		return false
	}
	sourceDir := filepath.ToSlash(filepath.Dir(sourcePath))
	for _, match := range pyImportRegex.FindAllStringSubmatch(string(content), -1) {
		spec := ""
		if len(match) > 1 && match[1] != "" {
			spec = match[1]
		} else if len(match) > 2 {
			spec = match[2]
		}
		if spec == "" {
			continue
		}
		for _, resolved := range resolvePyImportCandidates(sourceDir, spec) {
			if resolved == candidatePath {
				return true
			}
		}
	}
	return false
}

func resolveJSImportCandidates(sourceDir, spec string) []string {
	base := filepath.ToSlash(filepath.Clean(filepath.Join(sourceDir, spec)))
	candidates := []string{
		base,
		base + ".js",
		base + ".jsx",
		base + ".ts",
		base + ".tsx",
		base + "/index.js",
		base + "/index.jsx",
		base + "/index.ts",
		base + "/index.tsx",
	}
	return uniqueStrings(candidates, 0)
}

func isJSImportFile(path string) bool {
	return strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".jsx") ||
		strings.HasSuffix(path, ".ts") ||
		strings.HasSuffix(path, ".tsx")
}

func isPythonFile(path string) bool {
	return strings.HasSuffix(path, ".py")
}

func isSuggestedTestLink(link StructuralLink) bool {
	switch link.Kind {
	case "test-pair", "same-dir-test", "js-dependent", "py-dependent", "go-package-dependent":
		return true
	default:
		return false
	}
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

func inferredVerificationPlan(repoRoot, normalized string, tests []string) VerificationPlan {
	plan := VerificationPlan{}

	if strings.HasSuffix(normalized, ".go") || hasGoTests(tests) {
		packageDir := filepath.ToSlash(filepath.Dir(normalized))
		if packageDir == "." {
			plan.Fast = append(plan.Fast, "go test .")
			plan.Safe = append(plan.Safe, "go test .")
		} else {
			packageCmd := "go test ./" + packageDir
			plan.Fast = append(plan.Fast, packageCmd)
			plan.Safe = append(plan.Safe, packageCmd)
		}
		plan.Full = append(plan.Full, "go test ./...")
		return normalizeVerificationPlan(plan)
	}

	if strings.HasSuffix(normalized, ".go") {
		plan.Full = append(plan.Full, "go test ./...")
		return normalizeVerificationPlan(plan)
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
			plan.Fast = append(plan.Fast, packageManager+" test -- "+jsTest)
			plan.Safe = append(plan.Safe, packageManager+" test")
			plan.Full = append(plan.Full, inferJSTieredFullCheck(repoRoot, packageManager)...)
		}
		return normalizeVerificationPlan(plan)
	}
	if isJSImportFile(normalized) {
		packageManager := inferPackageManager(repoRoot)
		if packageManager != "" {
			plan.Safe = append(plan.Safe, packageManager+" test")
			plan.Full = append(plan.Full, inferJSTieredFullCheck(repoRoot, packageManager)...)
		}
		return normalizeVerificationPlan(plan)
	}

	pyTest := firstMatchingPath(tests, func(path string) bool {
		return strings.HasSuffix(path, "_test.py") || strings.HasPrefix(filepath.Base(path), "test_")
	})
	if pyTest != "" {
		plan.Fast = append(plan.Fast, "pytest "+pyTest)
		plan.Safe = append(plan.Safe, inferPySafeCommand(pyTest))
		plan.Full = append(plan.Full, "pytest")
		return normalizeVerificationPlan(plan)
	}
	if isPythonFile(normalized) {
		plan.Safe = append(plan.Safe, "pytest")
		plan.Full = append(plan.Full, "pytest")
		return normalizeVerificationPlan(plan)
	}

	rubyTest := firstMatchingPath(tests, func(path string) bool {
		return strings.HasSuffix(path, "_spec.rb")
	})
	if rubyTest != "" {
		plan.Fast = append(plan.Fast, "bundle exec rspec "+rubyTest)
		plan.Safe = append(plan.Safe, inferRubySafeCommand(rubyTest))
		plan.Full = append(plan.Full, "bundle exec rspec")
		return normalizeVerificationPlan(plan)
	}
	if strings.HasSuffix(normalized, ".rb") {
		plan.Safe = append(plan.Safe, "bundle exec rspec")
		plan.Full = append(plan.Full, "bundle exec rspec")
		return normalizeVerificationPlan(plan)
	}

	return normalizeVerificationPlan(plan)
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

func inferJSTieredFullCheck(repoRoot, packageManager string) []string {
	commands := []string{packageManager + " test"}
	if hasPackageScript(repoRoot, "lint") {
		commands = append(commands, packageManager+" lint")
	}
	if hasPackageScript(repoRoot, "build") {
		commands = append(commands, packageManager+" build")
	}
	return commands
}

func inferPySafeCommand(testPath string) string {
	dir := filepath.ToSlash(filepath.Dir(testPath))
	if dir == "." {
		return "pytest"
	}
	return "pytest " + dir
}

func inferRubySafeCommand(testPath string) string {
	dir := filepath.ToSlash(filepath.Dir(testPath))
	if dir == "." {
		return "bundle exec rspec"
	}
	return "bundle exec rspec " + dir
}

func normalizeVerificationPlan(plan VerificationPlan) VerificationPlan {
	plan.Fast = uniqueStrings(plan.Fast, 0)
	plan.Safe = uniqueStrings(append(plan.Safe, plan.Fast...), 0)
	plan.Full = uniqueStrings(append(plan.Full, plan.Safe...), 0)
	return plan
}

func hasPackageScript(repoRoot, script string) bool {
	content, err := os.ReadFile(filepath.Join(repoRoot, "package.json"))
	if err != nil {
		return false
	}
	return strings.Contains(string(content), `"`+script+`"`)
}

func resolvePyImportCandidates(sourceDir, spec string) []string {
	spec = strings.TrimSpace(spec)
	candidates := make([]string, 0, 8)
	if strings.HasPrefix(spec, ".") {
		trimmed := strings.TrimLeft(spec, ".")
		parts := []string{}
		if trimmed != "" {
			parts = strings.Split(trimmed, ".")
		}
		up := len(spec) - len(trimmed)
		baseDir := sourceDir
		for i := 1; i < up; i++ {
			baseDir = filepath.ToSlash(filepath.Dir(baseDir))
		}
		candidateBase := baseDir
		if len(parts) > 0 {
			candidateBase = filepath.ToSlash(filepath.Join(baseDir, filepath.Join(parts...)))
		}
		candidates = append(candidates, candidateBase+".py", filepath.ToSlash(filepath.Join(candidateBase, "__init__.py")))
		return uniqueStrings(candidates, 0)
	}

	dotted := strings.ReplaceAll(spec, ".", "/")
	candidates = append(candidates, dotted+".py", filepath.ToSlash(filepath.Join(dotted, "__init__.py")))
	return uniqueStrings(candidates, 0)
}

func readGoModulePath(repoRoot string) string {
	content, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
