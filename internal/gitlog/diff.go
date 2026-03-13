package gitlog

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func ChangedFilesBetween(repoRoot, baseRef, headRef string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "diff", "--name-only", baseRef, headRef)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to diff changed files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, filepath.ToSlash(line))
	}
	return out, nil
}
