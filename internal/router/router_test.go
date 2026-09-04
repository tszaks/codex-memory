package router

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouteChoosesLiveSessionsAndExplainsAlternatives(t *testing.T) {
	report, err := Route(context.Background(), Options{
		Task:      "find all running sessions on this computer",
		CWD:       t.TempDir(),
		Authority: AuthorityObserve,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Service != "session-awareness" || report.Action != "inspect-live-sessions" || !report.Allowed {
		t.Fatalf("unexpected route: %+v", report)
	}
	if !strings.Contains(report.Command, "sessions live") || len(report.Alternatives) < 2 || len(report.Caveats) == 0 {
		t.Fatalf("route must be actionable and honest about tradeoffs: %+v", report)
	}
}

func TestRouteNeverWidensAuthority(t *testing.T) {
	report, err := Route(context.Background(), Options{
		Task:      "implement a new API and verify it",
		CWD:       t.TempDir(),
		Authority: AuthorityObserve,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.RequiredAuthority != AuthorityEdit || report.Allowed || report.BlockedReason == "" {
		t.Fatalf("edit recommendation must remain blocked by observe ceiling: %+v", report)
	}

	report, err = Route(context.Background(), Options{
		Task:      "implement a new API and verify it",
		CWD:       report.CWD,
		Authority: AuthorityEdit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Allowed || report.AuthorityCeiling != AuthorityEdit {
		t.Fatalf("explicit edit ceiling should allow the recommendation: %+v", report)
	}
}

func TestRouteInspectsRepositoryState(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	path := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "tracked.txt")
	runGit(t, dir, "commit", "-m", "initial")
	if err := os.WriteFile(path, []byte("two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Route(context.Background(), Options{Task: "review the diff", CWD: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Repository.IsGit || !report.Repository.Dirty || report.Repository.Head == "" {
		t.Fatalf("repository state was not inspected: %+v", report.Repository)
	}
}

func TestRouteDistinguishesReviewFromReviewAndFix(t *testing.T) {
	dir := t.TempDir()
	review, err := Route(context.Background(), Options{Task: "review the current routing changes", CWD: dir})
	if err != nil {
		t.Fatal(err)
	}
	if review.Service != "repo-intelligence" || review.Action != "review-current-change" || review.DecisionConfidence != "high" {
		t.Fatalf("unexpected review route: %+v", review)
	}
	fix, err := Route(context.Background(), Options{Task: "review and fix the routing changes", CWD: dir, Authority: AuthorityEdit})
	if err != nil {
		t.Fatal(err)
	}
	if fix.Service != "workflow" || !fix.Allowed {
		t.Fatalf("review-and-fix should route to an editing workflow: %+v", fix)
	}
}

func TestRouteRejectsUnknownAuthority(t *testing.T) {
	_, err := Route(context.Background(), Options{Task: "inspect this", CWD: t.TempDir(), Authority: "unlimited"})
	if err == nil || !strings.Contains(err.Error(), "invalid authority") {
		t.Fatalf("expected invalid authority error, got %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
