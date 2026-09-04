package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/tszaks/pallium/internal/output"
	"github.com/tszaks/pallium/internal/router"
)

func runRoute(out io.Writer, args []string, jsonOutput bool) error {
	if hasHelpArg(args) {
		printRouteHelp(out)
		return nil
	}
	fs := newSessionFlagSet("route")
	cwd := fs.String("cwd", ".", "")
	authority := fs.String("authority", router.AuthorityObserve, "")
	if err := parseSessionFlags(fs, args, map[string]struct{}{"cwd": {}, "authority": {}}, nil); err != nil {
		return err
	}
	task := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if task == "" {
		return fmt.Errorf("usage: pallium route <task> [--cwd path] [--authority observe|execute|edit|external] [--json]")
	}
	report, err := router.Route(context.Background(), router.Options{Task: task, CWD: *cwd, Authority: *authority})
	if err != nil {
		return err
	}
	return output.Write(out, report, jsonOutput, func() string {
		lines := []string{
			"Recommended: " + report.Service + " / " + report.Action,
			"Why: " + report.Why,
			"Decision confidence: " + report.DecisionConfidence,
			fmt.Sprintf("Authority: requires %s; ceiling %s; allowed=%t", report.RequiredAuthority, report.AuthorityCeiling, report.Allowed),
		}
		if report.Command != "" {
			lines = append(lines, "Command: "+report.Command)
		}
		if report.BlockedReason != "" {
			lines = append(lines, "Blocked: "+report.BlockedReason)
		}
		if len(report.Signals) > 0 {
			lines = append(lines, "Signals: "+strings.Join(report.Signals, "; "))
		}
		if len(report.Alternatives) > 0 {
			lines = append(lines, "", "Alternatives considered:")
			for _, alternative := range report.Alternatives {
				line := "- " + alternative.Service + ": " + alternative.WhyNot
				if alternative.Command != "" {
					line += " (" + alternative.Command + ")"
				}
				lines = append(lines, line)
			}
		}
		return strings.Join(lines, "\n")
	})
}

func printRouteHelp(out io.Writer) {
	fmt.Fprintln(out, `pallium route

Choose and explain the best Pallium service for a task. The authority ceiling
is supplied by the caller and defaults to read-only observation; routing never
widens it or executes the returned command.

Usage:
  pallium route <task> [--cwd path] [--authority observe|execute|edit|external] [--json]`)
}
