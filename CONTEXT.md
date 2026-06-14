# monarchmoney-cli 领域术语表

## 核心守则

**monarchmoney-cli 是 Monarch Money 网页版的替代品**，额外提供自动化便利和 agent 友好。能参照 Monarch Money API 做法的自行决定，只询问 UX 相关的决策。

## 核心概念

**monarchmoney-cli** — Monarch Money 的本地 CLI 工具，用于查询、管理和自动化个人财务数据。仓库名 `monarchmoney-cli`，安装后的二进制文件是 `monarch`。

## 用户

**个人理财用户** — 需要从终端管理 Monarch Money 账户的人：查看账户余额、搜索交易、分析支出、管理预算。可能有多个账户，通过 profile 管理。

**Agent** — 自动化代理（CI、脚本、cron 任务）。需要确定性行为：稳定 JSON 输出、可预测退出码、安全门控。典型场景：定期同步数据、异常检测、自动分类交易。

## 命令设计决策

**安全门控** — 三级安全机制：
- `--read-only` / `MONARCH_READONLY=1` — 全局开关，阻止所有远程修改
- `--dry-run` — 操作级，只预览不执行
- `--confirm` — 操作级，确认高风险操作（删除账户、删除交易）

**审计日志** — 每个远程写操作都记录到 `~/.monarchmoney-cli/audit/` 的 JSONL 文件。`monarch audit cleanup --older-than N` 清理旧日志（默认 30 天）。

**输出格式** — 三级输出：
- 默认 — 简洁关键信息（表格形式）
- `--full` — 完整信息
- `--json` — 自动化/Agent 用

**缓存** — 本地 SQLite 缓存，用于加速查询和离线访问。大多数读操作优先使用缓存，`cache sync` 从 Monarch 拉取最新数据。

**分析命令** — `monarch analyze` 提供确定性分析（异常检测、订阅汇总、商户比较、预算消耗率），不嵌入 AI 行为。

**Feature Parity** — 与 monarch-mcp-server 完全对齐，所有 MCP 工具都有对应的 CLI 命令。
