package analysis

import (
	"testing"

	"github.com/tszaks/codex-memory/internal/index"
)

func TestSafe(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	report, err := Safe(store, "main.go")
	if err != nil {
		t.Fatalf("Safe failed: %v", err)
	}

	if report.Verdict == "" {
		t.Fatalf("expected safe verdict")
	}
	if len(report.RequiredChecks) == 0 {
		t.Fatalf("expected required checks")
	}
	if len(report.SuggestedTests) == 0 {
		t.Fatalf("expected suggested tests")
	}
}

func TestPlan(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	report, err := Plan(store, "main.go")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(report.Steps) == 0 {
		t.Fatalf("expected plan steps")
	}
	if len(report.FilesToInspect) == 0 {
		t.Fatalf("expected files to inspect")
	}
}

func TestReview(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	report, err := Review(store, "HEAD~1")
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	if len(report.ChangedFiles) == 0 {
		t.Fatalf("expected changed files in review report")
	}
	if len(report.RequiredTests) == 0 {
		t.Fatalf("expected review to suggest focused tests")
	}
}
