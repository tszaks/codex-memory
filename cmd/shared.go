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

func writeError(out io.Writer, err error) error {
	_, writeErr := fmt.Fprintln(out, err)
	if writeErr != nil {
		return writeErr
	}
	return err
}
