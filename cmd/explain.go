package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/output"
)

func runExplain(out io.Writer, args []string, jsonOutput bool) error {
	target, err := requireArg(args, "path")
	if err != nil {
		return err
	}
	repoPath := optionalRepoArg(args, 1)
	indexer, err := openIndexedStore(repoPath)
	if err != nil {
		return err
	}
	defer indexer.Store.Close()

	report, err := analysis.Explain(indexer.Store, target)
	if err != nil {
		return err
	}

	return output.Write(out, report, jsonOutput, func() string {
		lines := []string{
			fmt.Sprintf("Path: %s", report.Path),
			fmt.Sprintf("Risk: %s (%d)", report.Risk.Level, report.Risk.Score),
			fmt.Sprintf("Summary: %s", report.Summary),
			"",
			"Before you edit:",
		}
		for _, item := range report.EditChecklist {
			lines = append(lines, "- "+item)
		}
		lines = append(lines,
			"",
			"Recent commits:",
		)
		for _, commit := range report.RecentCommits {
			lines = append(lines, fmt.Sprintf("- %s %s", commit.SHA[:8], commit.Subject))
		}
		lines = append(lines, "", "Why it matters:")
		for _, reason := range report.Risk.Reasons {
			lines = append(lines, "- "+reason)
		}
		lines = append(lines, "", renderNeighbors(report.Neighbors))
		if len(report.Decisions) > 0 {
			lines = append(lines, "", "Decision notes:")
			for _, decision := range report.Decisions {
				lines = append(lines, fmt.Sprintf("- %s", decision.Title))
			}
		}
		return strings.Join(lines, "\n")
	})
}
