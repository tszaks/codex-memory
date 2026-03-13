package analysis

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tszaks/codex-memory/internal/db"
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
	if len(report.TestCommands) == 0 {
		t.Fatalf("expected safe test commands")
	}
	if len(report.Verification.Fast) == 0 {
		t.Fatalf("expected safe verification plan")
	}
	if report.Confidence.Level == "" {
		t.Fatalf("expected safe confidence")
	}
	if len(report.ActionGuidance.RunNext) == 0 {
		t.Fatalf("expected safe action guidance")
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
	if len(report.TestCommands) == 0 {
		t.Fatalf("expected plan test commands")
	}
	if len(report.Verification.Fast) == 0 {
		t.Fatalf("expected plan verification plan")
	}
	if len(report.ActionGuidance.InspectFirst) == 0 {
		t.Fatalf("expected plan action guidance")
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
	if len(report.TestCommands) == 0 {
		t.Fatalf("expected review test commands")
	}
	if len(report.Verification.Fast) == 0 {
		t.Fatalf("expected review verification plan")
	}
	if len(report.ActionGuidance.RunNext) == 0 {
		t.Fatalf("expected review action guidance")
	}
}

func TestReviewIncludesWorkingTreeChanges(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { println(\"changed\") }\n")

	report, err := Review(store, "HEAD~1")
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	found := false
	for _, file := range report.ChangedFiles {
		if file.Path == "main.go" {
			found = true
			if file.ChangeSource == "" {
				t.Fatalf("expected working tree change source for main.go")
			}
		}
	}
	if !found {
		t.Fatalf("expected working tree change to appear in review report")
	}
}

func TestChangedNow(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { println(\"changed\") }\n")

	report, err := ChangedNow(store)
	if err != nil {
		t.Fatalf("ChangedNow failed: %v", err)
	}

	if len(report.Files) == 0 {
		t.Fatalf("expected changed-now files")
	}
}

func TestHandoff(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { println(\"changed\") }\n")

	report, err := Handoff(store, "HEAD~1")
	if err != nil {
		t.Fatalf("Handoff failed: %v", err)
	}

	if report.Summary == "" {
		t.Fatalf("expected handoff summary")
	}
	if len(report.NextActions) == 0 {
		t.Fatalf("expected handoff next actions")
	}
}

func TestReviewDetectsTaskScopeDrift(t *testing.T) {
	repo := indexRepo(t)
	store, err := index.OpenStore(repo)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	if _, err := index.New(store).Run(); err != nil {
		t.Fatalf("index run failed: %v", err)
	}

	if err := store.SaveActiveTask(db.ActiveTask{
		Goal:       "Adjust main entrypoint only",
		ScopePaths: []string{"main.go"},
		StartedAt:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveActiveTask failed: %v", err)
	}

	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() { println(\"changed\") }\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "key: drifted\n")

	report, err := Review(store, "HEAD~1")
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	if !report.Task.HasScopeDrift {
		t.Fatalf("expected task scope drift, got %#v", report.Task)
	}
	if len(report.Task.OutOfScopeChanged) == 0 {
		t.Fatalf("expected out-of-scope changes, got %#v", report.Task)
	}
}
