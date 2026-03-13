package analysis

import (
	"fmt"
	"strings"

	"github.com/tszaks/codex-memory/internal/db"
	"github.com/tszaks/codex-memory/internal/gitlog"
)

type SafeReport struct {
	Path           string     `json:"path"`
	Verdict        string     `json:"verdict"`
	Summary        string     `json:"summary"`
	RequiredChecks []string   `json:"required_checks"`
	SuggestedTests []string   `json:"suggested_tests"`
	BlastRadius    []string   `json:"blast_radius"`
	Risk           RiskReport `json:"risk"`
}

type PlanReport struct {
	Path           string     `json:"path"`
	Goal           string     `json:"goal"`
	Steps          []string   `json:"steps"`
	FilesToInspect []string   `json:"files_to_inspect"`
	TestsToRun     []string   `json:"tests_to_run"`
	Risk           RiskReport `json:"risk"`
}

type ReviewedFile struct {
	Path           string   `json:"path"`
	ChangeSource   string   `json:"change_source"`
	RiskLevel      string   `json:"risk_level"`
	TopReasons     []string `json:"top_reasons"`
	SuggestedTests []string `json:"suggested_tests"`
	BlastRadius    []string `json:"blast_radius"`
}

type ReviewReport struct {
	BaseRef       string         `json:"base_ref"`
	HeadRef       string         `json:"head_ref"`
	Summary       string         `json:"summary"`
	ChangedFiles  []ReviewedFile `json:"changed_files"`
	RequiredTests []string       `json:"required_tests"`
	Notes         []string       `json:"notes"`
}

type ChangedNowFile struct {
	Path              string   `json:"path"`
	WorkingTreeStatus string   `json:"working_tree_status"`
	RiskLevel         string   `json:"risk_level"`
	SuggestedTests    []string `json:"suggested_tests"`
	BlastRadius       []string `json:"blast_radius"`
}

type ChangedNowReport struct {
	Summary string           `json:"summary"`
	Files   []ChangedNowFile `json:"files"`
}

type HandoffReport struct {
	Summary     string           `json:"summary"`
	Review      ReviewReport     `json:"review"`
	ChangedNow  ChangedNowReport `json:"changed_now"`
	NextActions []string         `json:"next_actions"`
}

func Safe(store *db.Store, targetPath string) (SafeReport, error) {
	risk, err := Risk(store, targetPath)
	if err != nil {
		return SafeReport{}, err
	}
	tests, err := SuggestedTests(store, targetPath, 5)
	if err != nil {
		return SafeReport{}, err
	}
	blastRadius, err := BlastRadius(store, targetPath, 6)
	if err != nil {
		return SafeReport{}, err
	}

	verdict := "safe_with_normal_review"
	summary := "Looks reasonably safe for an agent to edit with a normal review pass."
	switch risk.Level {
	case "high":
		verdict = "inspect_context_first"
		summary = "High-risk edit. An agent should inspect neighbors and recent history before changing this file."
	case "medium":
		verdict = "review_neighbors_first"
		summary = "Medium-risk edit. An agent should inspect related files and run the suggested tests."
	}

	checks := []string{
		"Read the explain report before editing.",
	}
	if len(blastRadius) > 0 {
		checks = append(checks, fmt.Sprintf("Inspect likely impact files: %s.", strings.Join(blastRadius[:min(len(blastRadius), 3)], ", ")))
	}
	if len(tests) > 0 {
		checks = append(checks, fmt.Sprintf("Run focused tests after editing: %s.", strings.Join(tests, ", ")))
	}

	return SafeReport{
		Path:           risk.Path,
		Verdict:        verdict,
		Summary:        summary,
		RequiredChecks: checks,
		SuggestedTests: tests,
		BlastRadius:    blastRadius,
		Risk:           risk,
	}, nil
}

func Plan(store *db.Store, targetPath string) (PlanReport, error) {
	safe, err := Safe(store, targetPath)
	if err != nil {
		return PlanReport{}, err
	}

	filesToInspect := uniqueStrings(append([]string{safe.Path}, safe.BlastRadius...), 5)
	steps := []string{
		fmt.Sprintf("Read `codex-memory explain %s` and inspect the recent decisions.", safe.Path),
		"Open the highest-signal related files before editing.",
		"Make the minimal change needed for the task.",
		"Run the focused tests suggested by codex-memory.",
		"Re-run explain or review if the blast radius grew during the change.",
	}

	return PlanReport{
		Path:           safe.Path,
		Goal:           "Help an agent make a low-surprise change with the right context loaded first.",
		Steps:          steps,
		FilesToInspect: filesToInspect,
		TestsToRun:     safe.SuggestedTests,
		Risk:           safe.Risk,
	}, nil
}

func Review(store *db.Store, baseRef string) (ReviewReport, error) {
	if strings.TrimSpace(baseRef) == "" {
		baseRef = "HEAD~1"
	}

	changed, err := gitlog.ChangedFilesBetween(store.RepoRoot, baseRef, "HEAD")
	if err != nil {
		return ReviewReport{}, err
	}
	workingTree, err := gitlog.WorkingTreeChanges(store.RepoRoot)
	if err != nil {
		return ReviewReport{}, err
	}

	changeSources := make(map[string]string)
	for _, path := range changed {
		changeSources[path] = "committed"
	}
	for _, item := range workingTree {
		source := "working_tree"
		if item.Status == "??" {
			source = "untracked"
		}
		if _, ok := changeSources[item.Path]; ok {
			source = "committed+working_tree"
		}
		changeSources[item.Path] = source
	}
	changed = mapKeysSorted(changeSources)

	reviewed := make([]ReviewedFile, 0, len(changed))
	requiredTests := make([]string, 0)
	notes := make([]string, 0)
	highRiskCount := 0

	for _, path := range changed {
		risk, err := Risk(store, path)
		if err != nil {
			notes = append(notes, fmt.Sprintf("No indexed risk data for %s yet.", path))
			reviewed = append(reviewed, ReviewedFile{
				Path:           path,
				ChangeSource:   changeSources[path],
				RiskLevel:      "unknown",
				TopReasons:     []string{"This file is new or outside indexed history, so risk is unknown."},
				SuggestedTests: []string{},
				BlastRadius:    []string{},
			})
			continue
		}
		tests, err := SuggestedTests(store, path, 4)
		if err != nil {
			return ReviewReport{}, err
		}
		blastRadius, err := BlastRadius(store, path, 4)
		if err != nil {
			return ReviewReport{}, err
		}

		if risk.Level == "high" {
			highRiskCount++
		}

		requiredTests = append(requiredTests, tests...)
		reviewed = append(reviewed, ReviewedFile{
			Path:           path,
			ChangeSource:   changeSources[path],
			RiskLevel:      risk.Level,
			TopReasons:     risk.Reasons,
			SuggestedTests: tests,
			BlastRadius:    blastRadius,
		})
	}

	summary := fmt.Sprintf("Review %d changed files before handing this branch back to an agent.", len(changed))
	if highRiskCount > 0 {
		summary = fmt.Sprintf("Review %d changed files carefully. %d high-risk file(s) need extra attention.", len(changed), highRiskCount)
	}

	return ReviewReport{
		BaseRef:       baseRef,
		HeadRef:       "HEAD",
		Summary:       summary,
		ChangedFiles:  reviewed,
		RequiredTests: uniqueStrings(requiredTests, 10),
		Notes:         uniqueStrings(notes, 10),
	}, nil
}

func ChangedNow(store *db.Store) (ChangedNowReport, error) {
	workingTree, err := gitlog.WorkingTreeChanges(store.RepoRoot)
	if err != nil {
		return ChangedNowReport{}, err
	}

	files := make([]ChangedNowFile, 0, len(workingTree))
	for _, item := range workingTree {
		risk, err := Risk(store, item.Path)
		if err != nil {
			files = append(files, ChangedNowFile{
				Path:              item.Path,
				WorkingTreeStatus: item.Status,
				RiskLevel:         "unknown",
				SuggestedTests:    []string{},
				BlastRadius:       []string{},
			})
			continue
		}
		tests, err := SuggestedTests(store, item.Path, 4)
		if err != nil {
			return ChangedNowReport{}, err
		}
		blastRadius, err := BlastRadius(store, item.Path, 4)
		if err != nil {
			return ChangedNowReport{}, err
		}
		files = append(files, ChangedNowFile{
			Path:              item.Path,
			WorkingTreeStatus: item.Status,
			RiskLevel:         risk.Level,
			SuggestedTests:    tests,
			BlastRadius:       blastRadius,
		})
	}

	return ChangedNowReport{
		Summary: fmt.Sprintf("Working tree currently touches %d file(s).", len(files)),
		Files:   files,
	}, nil
}

func Handoff(store *db.Store, baseRef string) (HandoffReport, error) {
	review, err := Review(store, baseRef)
	if err != nil {
		return HandoffReport{}, err
	}
	changedNow, err := ChangedNow(store)
	if err != nil {
		return HandoffReport{}, err
	}

	nextActions := make([]string, 0, 3)
	if len(review.RequiredTests) > 0 {
		nextActions = append(nextActions, fmt.Sprintf("Run focused tests: %s.", strings.Join(review.RequiredTests, ", ")))
	}
	if len(changedNow.Files) > 0 {
		nextActions = append(nextActions, "Review unstaged and untracked files before handing work back.")
	}
	if len(review.ChangedFiles) > 0 {
		nextActions = append(nextActions, "Open the highest-risk changed files and scan their blast radius.")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "No extra handoff actions suggested.")
	}

	return HandoffReport{
		Summary:     "Use this report to hand work from an agent back to a human or another agent with less surprise.",
		Review:      review,
		ChangedNow:  changedNow,
		NextActions: nextActions,
	}, nil
}

func mapKeysSorted(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return uniqueStrings(keys, 0)
}
