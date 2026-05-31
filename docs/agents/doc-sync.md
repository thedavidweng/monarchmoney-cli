# Doc Sync Rule

When modifying CLI commands, flags, JSON output structure, command behavior, or help text, update all four surfaces:

1. **Cobra help** — `Use` / `Short`, plus `Long` / `Example` where warranted
2. **`COMMANDS.md`** — command reference
3. **`docs/capabilities.md`** — capability matrix
4. **`docs/agent-guide.md`** — agent-facing behavior docs (when behavior changes affect agents)

Don't ship an implementation change without the matching doc updates.
