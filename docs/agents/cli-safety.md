# Monarch CLI help and mutation gates

- Every safety-gated command's Cobra `Short` help must end with ` (requires --confirm)`.
- Required-flag help receives `required: ` automatically through `MarkFlagRequired` and the `Execute()` walk in `internal/cli/root.go`; do not hand-prefix descriptions.
- Every mutation honors `--dry-run` and `--confirm` and preserves stable JSON/exit-code contracts.
- Document changes to these contracts in `COMMANDS.md`, `JSON_SCHEMA.md`, Cobra help and, when required, an ADR.
