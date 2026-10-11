# Testing

- Prefer repeatable, verifiable end-to-end CLI workflows for complex behavior, and define their expected artifacts before implementation.
- Create relevant regression or isolated tests before writing new behavior where possible. Use focused isolated tests when E2E cannot exercise a failure, after enumerating the failure paths.
- Do not add tests that merely mirror constants, test getters, or match arbitrary implementation strings.
- Treat coverage as evidence, not as a numerical target to game.
- `mise run check` must pass before push; it runs formatting, build, test, lint and conventions.
