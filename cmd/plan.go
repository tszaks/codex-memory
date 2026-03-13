package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/output"
)

func runPlan(out io.Writer, args []string, jsonOutput bool) error {
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

	report, err := analysis.Plan(indexer.Store, target)
	if err != nil {
		return err
	}

	return output.Write(out, report, jsonOutput, func() string {
		lines := []string{
			fmt.Sprintf("Path: %s", report.Path),
			fmt.Sprintf("Goal: %s", report.Goal),
			"",
			"Files to inspect:",
		}
		for _, file := range report.FilesToInspect {
			lines = append(lines, "- "+file)
		}
		lines = append(lines, "", "Suggested steps:")
		for _, step := range report.Steps {
			lines = append(lines, "- "+step)
		}
		if len(report.TestsToRun) > 0 {
			lines = append(lines, "", "Tests to run:")
			for _, test := range report.TestsToRun {
				lines = append(lines, "- "+test)
			}
		}
		return strings.Join(lines, "\n")
	})
}
