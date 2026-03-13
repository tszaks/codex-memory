# codex-memory

`codex-memory` is a local-first CLI that turns a repository's git history into quick, practical context.

It helps you answer questions like:

- Is this file risky to touch?
- What other files usually move with it?
- What changed here recently?
- What past commits might explain why this exists?
- What should I check before I edit this file?

Instead of digging through `git log`, guessing file coupling, or relying on vague memory, you can ask the repo directly.

## Why Use It

Code changes rarely fail because the edit was hard.

They fail because the hidden context was missing:

- the file was touched three times last week
- a "simple" config file quietly fans out into five other files
- the logic was added for a very specific reason six weeks ago
- a risky file looks isolated until you see what usually changes with it

`codex-memory` keeps that context local, fast, and inspectable.

## What It Does

`codex-memory` indexes your repository history into a small SQLite database inside `.codex-memory/`, then exposes that history through a few focused commands.

Current commands:

- `codex-memory index`
- `codex-memory explain <path>`
- `codex-memory risk <path>`
- `codex-memory neighbors <path>`
- `codex-memory decisions <query>`

Use `--json` after any command if you want machine-readable output for agents, scripts, or other tooling.

## Install

If you want the CLI on your path:

```bash
go install github.com/tszaks/codex-memory@latest
```

If you're working from the repo directly:

```bash
git clone https://github.com/tszaks/codex-memory.git
cd codex-memory
go test ./...
go run . --help
```

## Quick Start

### 1. Index a repository

Run this once inside the repo you want to analyze:

```bash
codex-memory index
```

Example output:

```text
Indexed 428 commits, 191 files, and 2634 co-change edges in /path/to/repo
```

### 2. Start with `explain`

This is the best default command when you're about to edit a file.

```bash
codex-memory explain app/services/billing.rb
```

It gives you:

- a risk summary
- a short pre-edit checklist
- recent commits touching the file
- files that commonly move with it
- likely rationale pulled from commit history

### 3. Check risk directly

```bash
codex-memory risk app/services/billing.rb
```

This shows:

- overall risk score and level
- churn and recent touch count
- number of related files
- author count
- last-touched timestamp
- plain-English reasons the file may deserve extra care

### 4. Find related files

```bash
codex-memory neighbors app/services/billing.rb
```

Use this before making a "small" change. It helps reveal files that commonly change alongside your target.

### 5. Search for likely rationale

```bash
codex-memory decisions "retry logic"
```

This searches stored decision notes derived from commit history so you can find likely explanation trails faster.

## Example Workflow

Before changing a file:

```bash
codex-memory index
codex-memory explain path/to/file
codex-memory neighbors path/to/file
```

If you want structured output for another tool:

```bash
codex-memory explain path/to/file --json
```

## Why `explain` Matters

Most tools stop at raw history. `codex-memory explain` tries to answer the more useful question:

> "What should I know before I touch this file?"

That makes it useful for:

- developers making changes in unfamiliar areas
- AI coding tools that need local codebase context
- quick pre-edit sanity checks
- spotting files that are more coupled or volatile than they look

## Local-First by Design

`codex-memory` is intentionally simple:

- no remote backend
- no daemon
- no embeddings requirement
- no IDE lock-in
- no hidden service dependency

Your repository history stays local. The database is transparent. The output is easy to inspect.

## Current Scope

Today, `codex-memory` is built around git-history-based context:

- file churn
- recent touches
- co-change relationships
- commit-derived rationale
- actionable pre-edit summaries

It is not trying to be a full code intelligence platform. The goal is a small tool that makes edit decisions smarter.

## Development

Run the full test suite:

```bash
go test ./...
```

Run against the repo locally:

```bash
go run . index
go run . explain README.md
go run . risk README.md
```

## Notes

- Existing local databases are migrated automatically when new file-level metadata is added.
- If a repo has not been indexed yet, `explain`, `risk`, `neighbors`, and `decisions` will tell you to run `codex-memory index` first.

## License

MIT
