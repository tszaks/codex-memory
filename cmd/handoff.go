package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/output"
)

func runHandoff(out io.Writer, args []string, jsonOutput bool) error {
	baseRef, repoPath := parseReviewArgs(args)
	indexer, err := openIndexedStore(repoPath)
	if err != nil {
		return err
	}
	defer indexer.Store.Close()

	report, err := analysis.Handoff(indexer.Store, baseRef)
	if err != nil {
		return err
	}

	return output.Write(out, report, jsonOutput, func() string {
		lines := []string{
			fmt.Sprintf("Summary: %s", report.Summary),
			fmt.Sprintf("Review: %s", report.Review.Summary),
			fmt.Sprintf("Changed now: %s", report.ChangedNow.Summary),
		}
		if len(report.NextActions) > 0 {
			lines = append(lines, "", "Next actions:")
			for _, action := range report.NextActions {
				lines = append(lines, "- "+action)
			}
		}
		return strings.Join(lines, "\n")
	})
}
