# codex-memory

`codex-memory` is a local-first CLI for answering the context questions that slow down code changes:

- What files are risky here?
- What usually changes together?
- What recently changed in this area?
- What past commits explain why this exists?

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

## Why this exists

AI tools are good at editing code, but they are bad at remembering:

- hidden file coupling
- churn hotspots
- historical rationale
- which “simple” files keep causing regressions

`codex-memory` keeps that context local and queryable.

## Development

```bash
go test ./...
go run . index
go run . explain README.md
```
