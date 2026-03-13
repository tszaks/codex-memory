package cmd

import "testing"

func TestReviewArgsTreatSinglePathAsRepoPath(t *testing.T) {
	baseRef, repoPath := parseReviewArgs([]string{"/tmp/repo"})
	if baseRef != "HEAD~1" {
		t.Fatalf("expected default base ref, got %q", baseRef)
	}
	if repoPath != "/tmp/repo" {
		t.Fatalf("expected repo path, got %q", repoPath)
	}
}

func TestReviewArgsTreatSingleRefAsBaseRef(t *testing.T) {
	baseRef, repoPath := parseReviewArgs([]string{"origin/main"})
	if baseRef != "origin/main" {
		t.Fatalf("expected base ref, got %q", baseRef)
	}
	if repoPath != "." {
		t.Fatalf("expected default repo path, got %q", repoPath)
	}
}
