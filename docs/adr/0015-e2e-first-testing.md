# 0015: E2E-First Testing

Status: Accepted

Context: Unit tests mirrored constants, getters, help text, and JSON envelopes via string-contains assertions. Coverage thresholds drove test creation rather than failure modes. E2E verified only help output and produced no repeatable artifact.

Decision: E2E is the default testing mechanism. Each E2E run writes verifiable repeatable artifacts (version.json, doctor.json, commands.json) with stable-field equality across runs. Isolated tests exist only for failures E2E cannot reach (file errors, network faults, retries, migrations, permission gates, pure-math edges), asserting exact error codes and DB or journal artifacts instead of string matching. Coverage is informational. codecov patch target lowered 80 to 50, project threshold widened 2 to 5, trivial packages excluded.

Consequences: Deleted 37 garbage test files and functions. CLI suite runtime dropped 45s to 8s. New commands update requiredCommands and TestAllCommandsInHelp enforces registration. New failure modes add isolated tests with exact codes before code.
