## Doc sync

When modifying CLI commands, flags, JSON output structure, command behavior, or help text, update all three surfaces:

1. **Cobra help** — `Use` / `Short`, plus `Long` / `Example` where warranted
2. **`docs/capabilities.md`** — capability matrix
3. **`docs/agent-guide.md`** — agent-facing behavior docs (when behavior changes affect agents)

Don't ship an implementation change without the matching doc updates.

## Agent skills

### Issue tracker

GitHub Issues via `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — `CONTEXT.md` + `docs/adr/` at repo root. See `docs/agents/domain.md`.
