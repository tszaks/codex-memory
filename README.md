# codex-memory

`codex-memory` is a local-first workflow tool for AI-powered coding.

It gives an LLM the missing repository context before it edits code:

- what files are risky
- what else is likely to move
- what tests are most relevant
- what recent commits matter
- what the blast radius probably looks like

The goal is simple: help agents make lower-surprise changes.

## Why This Exists

LLMs are strong at writing code and weak at remembering repository context.

They often miss:

- hidden file coupling
- recently unstable areas
- test files that should run after a change
- historical rationale buried in git
- when a "small edit" actually has a bigger blast radius

`codex-memory` turns git history and lightweight repo structure into a small local memory layer an agent can query before coding.

## Best Use Case

This tool is most useful right before an agent edits code.

The ideal loop looks like this:

```bash
codex-memory index
codex-memory explain path/to/file --json
codex-memory safe path/to/file --json
codex-memory plan path/to/file --json
```

That gives the model enough context to:

- decide whether the change is isolated or risky
- inspect the right neighboring files first
- choose the right tests to run
- avoid obvious regression traps

## Install

```bash
go install github.com/tszaks/codex-memory@latest
```

Or run it directly from source:

```bash
git clone https://github.com/tszaks/codex-memory.git
cd codex-memory
go test ./...
go run . --help
```

## Quick Start

### 1. Index the repo

Run this inside the repository you want to analyze:

```bash
codex-memory index
```

Example:

```text
Indexed 428 commits, 191 files, and 2634 co-change edges in /path/to/repo
```

### 2. Ask for a pre-edit briefing

```bash
codex-memory explain app/services/billing.rb
```

This is the best default command.

It returns:

- a risk summary
- a short edit checklist
- recent commits touching the file
- likely rationale from commit history
- suggested tests
- likely blast radius

### 3. Ask if the change is safe for an agent

```bash
codex-memory safe app/services/billing.rb
```

This gives an opinionated verdict such as:

- `safe_with_normal_review`
- `review_neighbors_first`
- `inspect_context_first`

It also gives the checks an agent should complete before and after editing.

### 4. Generate an execution plan

```bash
codex-memory plan app/services/billing.rb
```

This turns the file context into a lightweight agent plan:

- which files to inspect first
- what steps to follow
- which tests to run

### 5. Review what changed before handoff

```bash
codex-memory review origin/main
```

This reviews the files changed between `origin/main` and `HEAD`, then reports:

- risky changed files
- focused tests to run
- likely blast radius for each changed file

This is useful before asking an agent to finalize, open a PR, or hand work back to a human.

## Commands

### `codex-memory index`

Build or refresh the local memory database for the current repo.

```bash
codex-memory index
codex-memory index /path/to/repo
```

### `codex-memory explain <path>`

Get the highest-signal context before editing a file.

```bash
codex-memory explain src/auth/session.ts
```

### `codex-memory safe <path>`

Get an agent-oriented safety verdict plus required checks.

```bash
codex-memory safe src/auth/session.ts
```

### `codex-memory plan <path>`

Get a lightweight plan for how an agent should approach a change.

```bash
codex-memory plan src/auth/session.ts
```

### `codex-memory review [base-ref]`

Review changed files between a base ref and `HEAD`.

```bash
codex-memory review
codex-memory review HEAD~1
codex-memory review origin/main
```

### `codex-memory risk <path>`

Inspect the underlying file risk signals directly.

```bash
codex-memory risk src/auth/session.ts
```

### `codex-memory neighbors <path>`

See files that commonly change with the target file.

```bash
codex-memory neighbors src/auth/session.ts
```

### `codex-memory decisions <query>`

Search commit-derived rationale and decision notes.

```bash
codex-memory decisions "retry logic"
```

## JSON Output

Every command supports `--json`.

This is the intended mode for LLM workflows.

Examples:

```bash
codex-memory explain src/auth/session.ts --json
codex-memory safe src/auth/session.ts --json
codex-memory plan src/auth/session.ts --json
codex-memory review origin/main --json
```

## What The Tool Uses

`codex-memory` currently combines:

- git history
- co-change relationships
- recent touch patterns
- author count
- lightweight structural heuristics
- source-to-test pairing hints

It is intentionally small and local-first.

## What It Does Not Try To Be

This is not:

- a hosted code intelligence platform
- a full static-analysis engine
- an IDE extension
- an embeddings-first search product

It is a practical memory layer for AI coding workflows.

## Development

```bash
go test ./...
go run . index
go run . explain README.md
go run . safe README.md
go run . plan README.md
go run . review HEAD~1
```

## Notes

- Local databases live in `.codex-memory/`
- Schema updates migrate automatically
- If a repo has not been indexed yet, the analysis commands will tell you to run `codex-memory index` first

## License

MIT
