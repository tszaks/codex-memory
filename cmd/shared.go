package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/index"
)

func openIndexedStore(path string) (*index.Indexer, error) {
	store, err := index.OpenStore(path)
	if err != nil {
		return nil, err
	}
	return index.New(store), nil
}

func renderNeighbors(neighbors []analysis.Neighbor) string {
	if len(neighbors) == 0 {
		return "No related files found."
	}
	lines := make([]string, 0, len(neighbors)+1)
	lines = append(lines, "Related files:")
	for _, neighbor := range neighbors {
		lines = append(lines, fmt.Sprintf("- %s (%d co-changes)", neighbor.Path, neighbor.CochangeCount))
	}
	return strings.Join(lines, "\n")
}

func renderActionGuidance(guidance analysis.ActionGuidance) []string {
	lines := make([]string, 0)
	if len(guidance.InspectFirst) > 0 {
		lines = append(lines, "Inspect first:")
		for _, item := range guidance.InspectFirst {
			lines = append(lines, "- "+item)
		}
	}
	if len(guidance.RunNext) > 0 {
		lines = append(lines, "Run next:")
		for _, item := range guidance.RunNext {
			lines = append(lines, "- "+item)
		}
	}
	lines = append(lines, fmt.Sprintf("Safe to edit alone: %t", guidance.SafeToEditAlone))
	if len(guidance.AskForReviewIf) > 0 {
		lines = append(lines, "Ask for review if:")
		for _, item := range guidance.AskForReviewIf {
			lines = append(lines, "- "+item)
		}
	}
	if len(guidance.ConfidenceGaps) > 0 {
		lines = append(lines, "Confidence gaps:")
		for _, item := range guidance.ConfidenceGaps {
			lines = append(lines, "- "+item)
		}
	}
	if len(guidance.BoundaryWarnings) > 0 {
		lines = append(lines, "Boundary warnings:")
		for _, item := range guidance.BoundaryWarnings {
			lines = append(lines, fmt.Sprintf("- %s: %s", item.Label, item.Reason))
		}
	}
	return lines
}

func renderTaskScope(task analysis.TaskScopeReport) []string {
	if task.Goal == "" {
		return nil
	}
	lines := []string{fmt.Sprintf("Active task: %s", task.Goal)}
	if len(task.ScopePaths) > 0 {
		lines = append(lines, "Planned scope:")
		for _, scope := range task.ScopePaths {
			lines = append(lines, "- "+scope)
		}
	}
	if task.HasScopeDrift {
		lines = append(lines, "Scope drift:")
		for _, path := range task.OutOfScopeChanged {
			lines = append(lines, "- "+path)
		}
	}
	return lines
}

func writeError(out io.Writer, err error) error {
	_, writeErr := fmt.Fprintln(out, err)
	if writeErr != nil {
		return writeErr
	}
	return err
}
