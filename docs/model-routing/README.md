# Model and reasoning routing: research and evaluation

Research date: September 4, 2026 (America/New_York). Local model-cache timestamp: September 5, 2026 UTC. Source checkout: `b9866188e97d0e6a5f6b91b307e0a8663cab92f4`.

## Decision

Proceed with a bounded experiment. Auto should select **provider, model, and reasoning effort together**, constrained by installed/authenticated providers, supported settings, user policy, task requirements, and budget. Explicit user selections win. Auto is an eventual default, not enabled by this research.

Deliverables:

- [Evidence catalog](catalog.md): primary sources, prices, capabilities, conflicts, and provisional candidates.
- [Evaluation protocol](evaluation.md): configurations, scoring, accounting, isolation, and promotion criteria.
- [Task specifications](tasks.jsonl): 24 source-grounded task designs, each with an acceptance contract. These are specifications, not executable fixtures or measured results.

## What the checkout supports

`internal/workflow/provider.go` resolves a provider and dispatches Codex, built-in Claude, or a configured wrapper. `AgentOptions.Model` in `internal/workflow/runtime.go` and the two built-in provider adapters support model overrides. `providers/gemini.sh` offers a Gemini wrapper. The generic wrapper contract supplies `PALLIUM_WORKFLOW_MODEL`.

There is **no explicit reasoning-effort field in AgentOptions, CheckOptions, or GateOptions**. No reasoning-effort reference was found in `internal/workflow` or `cmd`. Inherited CLI configuration is not a reproducible per-task choice. Adding, validating, forwarding, and persisting effort is a prerequisite to controlled trials, not implemented here.

Codex CLI 0.153.0 and Claude Code 2.1.223 are installed. `gemini` was not on PATH. The local Codex model cache advertises the four initial OpenAI candidates and supported effort levels; this is discovery evidence, not a successful worker invocation or proof of account billing terms. No authentication files were read and no paid worker was launched.

The built-in ordinary Codex worker writes final output but does not write the usage file consumed by the common runtime. Cost estimates already stored by Pallium must not be mistaken for metered cost. Accurate token/usage capture is another trial prerequisite.

## Lab boundaries

Separate three things: a model's origin, a host's built-in delegation options, and providers reachable through Pallium. A restricted host may offer only same-lab delegation. Pallium can dispatch a different provider through its external CLI when tools, authentication, and policy permit. Code support is not proof that a particular account can execute that model.

The first experiment is OpenAI-only to match the requested Astra/Luna comparison and reduce harness differences. Claude and Gemini remain cataloged expansion candidates. A future allowed-provider list must be explicit; missing access never triggers a silent cross-provider fallback. Provider-native session IDs are not transferable. New-provider workers need a bounded context handoff, and its cost belongs in the comparison.

## Next implementation slice

1. Add explicit effort support across worker/check/gate/team options, provider adapters, persisted execution records, and replay/cache identities. Validate provider-specific values; unsupported values fail clearly.
2. Capture requested and effective provider/model/effort, usage, service tier, CLI version, and all retry costs. Keep unknown costs null.
3. Materialize the task fixtures and private graders described in the protocol, then freeze hashes and the calibration/holdout split.
4. Run a budgeted pilot only after its execution budget is set. Research collection is complete for this first catalog; model trials and implementation have not started.

This is a research-backed shortlist, not an exhaustive model ranking. No evidence here establishes that Luna xhigh is superior to Astra high on Pallium tasks.
