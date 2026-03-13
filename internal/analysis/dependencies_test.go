package analysis

import (
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

	if len(commands) == 0 || commands[0] != "go test ./..." {
		t.Fatalf("expected go test command, got %#v", commands)
	}
}
