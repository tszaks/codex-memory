# Agent-driven Pallium routing

## Problem

Pallium has broad execution capacity, but a user or steering agent must already
know which service to invoke. That manual translation makes the human the
router and leaves useful services idle.

The product boundary is explicit: agents choose tools and execution shape;
humans retain control of intent, constraints, and authority.

## Theory of constraints

System flow:

1. Understand the user's task and current authority.
2. Inspect repository and session state.
3. Select the narrowest suitable Pallium service.
4. Act through that service.
5. Verify the result and preserve useful state.

| Step | Capacity | Demand | Finding |
| --- | --- | --- | --- |
| Understand task | High in the steering agent | Every task | Not the current bottleneck |
| Inspect state | High across repo and session services | Needed before risky work | Underused unless manually invoked |
| Select service | Previously a static guide and command menu | Needed for every nontrivial task | Binding constraint |
| Execute | High across workflows, loops, teams, and verification | Only after routing | Starved by the selection constraint |
| Verify and preserve | Available in the kernel | Needed for trustworthy completion | Starved when the right service is never selected |

Binding-constraint hypothesis: Pallium's limiting factor is trustworthy
state-to-service routing at the moment an agent receives a task, not missing
execution primitives.

## Constraint intervention plan

### Exploit

- Route from natural-language task plus current repository state.
- Return one recommended service and command, the evidence used, why it fits,
  and why alternatives fit less well.
- Default authority to `observe` and return `allowed: false` when the selected
  action exceeds the supplied ceiling.
- Keep routing advisory and side-effect-free. The steering agent executes the
  returned command only when it remains inside user intent.

### Subordinate

- Make `pallium route` the first instruction in the adoption block and agent
  guide for nontrivial tasks.
- Keep service-specific commands as implementation details behind the routing
  decision, while preserving direct access for expert callers.
- Require structured results and explicit caveats so agents can inspect and
  override weak routes instead of blindly following them.

### Elevate

After the deterministic router has outcome data:

- Move service metadata into a capability registry so new tools become
  routable without duplicating decision logic.
- Add an MCP routing endpoint backed by the same contract.
- Record route, override, completion, verification, and authority-block
  outcomes; use them to improve policy without learning broader permission.
- Add provider and environment detectors when evidence shows missing coverage
  is the next constraint.

### Re-check

Measure these before replacing the deterministic policy:

- eligible tasks routed without a human naming a Pallium tool;
- accepted versus overridden recommendations by service;
- routes blocked by authority and any attempted ceiling escalation;
- verified completion rate and time-to-first-useful-action;
- unnecessary Pallium use on one-shot tasks.

The next constraint is likely route quality or adoption coverage. It should be
identified from these outcomes, not assumed in advance.
