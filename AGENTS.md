# monarchmoney-cli

Go/Cobra CLI for querying and safely automating Monarch Money financial data.

- Toolchain and checks: Go/Cobra with `mise`; run `mise run check` before every push.
- The canonical agent entry point is this `AGENTS.md`; do not add IDE-specific instruction copies.
- Read task-specific guides:
  - [Engineering and contract docs](docs/agents/engineering.md).
  - [Testing](docs/agents/testing.md).
  - [ADRs and issue workflow](docs/agents/workflow.md).
- [Mutation safety and Cobra help](docs/agents/cli-safety.md) when changing commands.
