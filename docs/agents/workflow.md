# Decisions, documentation and issues

- Write durable architecture decisions as `docs/adr/NNNN-slug.md` with Status / Context / Decision / Consequences.
- Record decisions on new dependencies, new abstractions, or public JSON/exit-code contract changes in an ADR in the same change.
- `CONTEXT.md` is the domain glossary. Use repository vocabulary instead of inventing synonymous concepts.
- Track future work in GitHub Issues via `gh`; triage vocabulary: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`.
- Prefer linking to existing contract/ADR files rather than copying their content into agent instructions.
