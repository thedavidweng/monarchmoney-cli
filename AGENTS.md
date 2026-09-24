# AGENTS.md

Instructions for every coding agent working in this repo. This is the only agent-instruction file — never create CLAUDE.md, .cursorrules, .windsurfrules, .clinerules, GEMINI.md, or any other per-tool variant (CI fails the build if one appears).

## Code principles

1. Self-explanatory code, no comments. Write a comment only for a constraint the code cannot express (a protocol quirk, a required ordering, a spec reference). Decision rationale goes in `docs/adr/`, conventions go here — never in code. CI enforces a comment budget of 5% of non-test lines.
2. Fewest lines that stay clear. No fallback code, no speculative features, no dead code, no abstraction with a single caller. Delete before you add.
3. Code and docs move together: a change to commands, flags, output, or behavior updates the cobra help, COMMANDS.md, and JSON_SCHEMA.md in the same change.
4. Help-text conventions: every safety-gated command's Short ends with ` (requires --confirm)`; required flags render a `required: ` prefix automatically from `MarkFlagRequired` via the `Execute()` walk in `internal/cli/root.go` (never hand-prefix); every mutation honors `--dry-run`/`--confirm`.

## Verification

`mise run check` (fmt + build + test + lint + conventions) must pass before every push. CI runs the same gates.

## Testing

- Never write unit tests after code.
- E2E is the sole testing mechanism by default: verify complex features end-to-end, each run producing a verifiable repeatable artifact.
- Isolated tests only when E2E cannot reach the failure: first enumerate all failure modes, then write code. No string-match, constant-mirror, or getter tests.
- Coverage is a side effect, never the goal; never add tests to hit a threshold.

## Decisions

Architectural decisions live in `docs/adr/NNNN-slug.md` (Status / Context / Decision / Consequences). A new dependency, a new abstraction, or a change to the public JSON/exit-code contract requires an ADR in the same change.

## Domain

`CONTEXT.md` at the repo root is the domain glossary.

## Issues

GitHub Issues via `gh`. Labels: needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix.
