<INSTRUCTIONS>
遵循奥卡姆剃刀和第一性原理，不写兜底代码，检查代码是否有过时/死代码。

每当对 CLI 命令、flags、JSON 输出结构、命令行为或帮助文本进行更改时，必须同步更新：
- Cobra 命令帮助：`Use` / `Short`，必要时补 `Long` / `Example`
- `COMMANDS.md`
- `docs/capabilities.md`
- 面向 agent 的行为说明：必要时更新 `docs/agent-guide.md`

不要只改实现而遗漏文档或帮助命令。
</INSTRUCTIONS>

## Agent skills

### Issue tracker

GitHub Issues via `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — `CONTEXT.md` + `docs/adr/` at repo root. See `docs/agents/domain.md`.
