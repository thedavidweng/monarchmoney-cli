# Go coding rules

- Use the repository's existing Cobra command and output patterns; don't introduce speculative features, dead helpers or unnecessary fallback branches.
- Keep changed code and matching CLI help/reference docs synchronized: changes to commands, flags, output or behavior update Cobra help, `COMMANDS.md` and `JSON_SCHEMA.md` in the same commit.
- The `mise run conventions` gate permits at most 5% comment lines in non-test `cmd/` and `internal/`, excluding `//go:` and `//nolint`. Use names/types for ordinary explanation; record durable reasons in `docs/adr/`.
- Do not create `CLAUDE.md`, `.cursorrules`, `.windsurfrules`, `.clinerules` or `GEMINI.md`; the conventions gate enforces one canonical root `AGENTS.md`.
