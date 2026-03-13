package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tszaks/codex-memory/internal/index"
)

func TestStructuralLinksIncludeGoDependencies(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	links, err := StructuralLinks(store, "main.go", 10)
	if err != nil {
		t.Fatalf("StructuralLinks failed: %v", err)
	}

	found := false
	for _, link := range links {
		if link.Path == "helper.go" && link.Kind == "go-symbol" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected go dependency link to helper.go, got %#v", links)
	}
}

func TestSuggestedTestCommandsForGoFile(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	commands, err := SuggestedTestCommands(store, "main.go", 5)
	if err != nil {
		t.Fatalf("SuggestedTestCommands failed: %v", err)
	}

	if len(commands) < 2 || commands[0] != "go test ." || commands[1] != "go test ./..." {
		t.Fatalf("expected focused and broad go test commands, got %#v", commands)
	}
}

func TestStructuralLinksIncludeJSImports(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	links, err := StructuralLinks(store, "web/app.ts", 10)
	if err != nil {
		t.Fatalf("StructuralLinks failed: %v", err)
	}

	found := false
	for _, link := range links {
		if link.Path == "web/session.ts" && link.Kind == "js-import" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected js import link to web/session.ts, got %#v", links)
	}
}

func TestRiskInfersContextForNewFile(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "feature.go"), []byte("package main\n\nfunc feature() string { return helper() }\n"), 0o644); err != nil {
		t.Fatalf("write feature.go failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "feature_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestFeature(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatalf("write feature_test.go failed: %v", err)
	}

	report, err := Risk(store, "feature.go")
	if err != nil {
		t.Fatalf("Risk failed for new file: %v", err)
	}

	if report.Level == "unknown" {
		t.Fatalf("expected inferred risk level for new file, got %#v", report)
	}
	if len(report.Reasons) == 0 {
		t.Fatalf("expected inferred reasons for new file, got %#v", report)
	}

	commands, err := SuggestedTestCommands(store, "feature.go", 5)
	if err != nil {
		t.Fatalf("SuggestedTestCommands failed for new file: %v", err)
	}
	if len(commands) < 2 || commands[0] != "go test ." {
		t.Fatalf("expected focused go commands for new file, got %#v", commands)
	}
}
