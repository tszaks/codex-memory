# codex-memory

`codex-memory` is a local-first CLI for answering the context questions that slow down code changes:

- What files are risky here?
- What usually changes together?
- What recently changed in this area?
- What past commits explain why this exists?
- What should I check before I edit this file?

It indexes a repository's git history into a small SQLite database and exposes that memory through a few focused commands.

## Commands

```bash
codex-memory index
codex-memory explain <path>
codex-memory risk <path>
codex-memory neighbors <path>
codex-memory decisions <query>
```

Use `--json` after any command for machine-readable output.

`explain` is the best starting point. It now gives you:

- a risk summary
- a short pre-edit checklist
- recent commits touching the file
- files that commonly move with it
- likely rationale pulled from commit history

## Why this exists

AI tools are good at editing code, but they are bad at remembering:

- hidden file coupling
- churn hotspots
- historical rationale
- which “simple” files keep causing regressions

`codex-memory` keeps that context local and queryable.

## What v2 adds

- smarter risk scoring with author count and recency context
- actionable “why it matters” explanations
- a better `explain` command built for pre-edit decision making
- automatic migration for older local SQLite databases

## Development

```bash
go test ./...
go run . index
go run . explain README.md
```
