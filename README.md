# codex-memory

`codex-memory` is a local-first CLI for AI-powered coding workflows.

It gives an LLM fast repo context before, during, and after edits:

- what files are risky
- what else is likely to move
- what tests are most relevant
- what command to run for the fastest useful verification
- what the blast radius probably is
- what changed in the working tree right now

## Why It Matters

LLMs are good at writing code and bad at remembering repository context.

That leads to common mistakes:

- editing a risky file in isolation
- missing related files
- skipping the most useful tests
- handing work off without a clean summary

`codex-memory` exists to lower those surprises.

## Core Commands

```bash
codex-memory index
codex-memory explain <path>
codex-memory safe <path>
codex-memory plan <path>
codex-memory changed-now
codex-memory review [base-ref]
codex-memory handoff [base-ref]
```

Use `--json` with any command for agent-friendly output.

## Typical Agent Loop

```bash
codex-memory index
codex-memory explain path/to/file --json
codex-memory safe path/to/file --json
codex-memory plan path/to/file --json
codex-memory changed-now --json
codex-memory handoff origin/main --json
```

## Install

```bash
go install github.com/tszaks/codex-memory@latest
```

Or from source:

```bash
git clone https://github.com/tszaks/codex-memory.git
cd codex-memory
go test ./...
go run . --help
```

## What Each Command Does

- `explain`: best pre-edit briefing for a file
- `safe`: tells an agent how cautious it should be, with confidence
- `plan`: gives a lightweight edit plan plus likely test commands
- `changed-now`: shows the live working tree
- `review`: reviews branch diff plus working-tree changes with confidence
- `handoff`: generates a final summary before handoff

## Example

```bash
codex-memory explain src/auth/session.ts --json
codex-memory safe src/auth/session.ts --json
codex-memory handoff origin/main --json
```

## Development

```bash
go test ./...
go run . index
go run . explain README.md
go run . changed-now
go run . handoff HEAD~1
```

## Notes

- Local data lives in `.codex-memory/`
- If the repo has not been indexed yet, analysis commands will tell you to run `codex-memory index` first

## License

MIT
